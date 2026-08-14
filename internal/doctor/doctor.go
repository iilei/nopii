// Package doctor validates local nopii setup and configuration health.
package doctor

import (
	"fmt"
	"os/exec"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/gitinit"
)

const (
	checkConfig = "config"
	checkGit    = "git"
	checkKey    = "key"
	statusOK    = "ok"
	statusWarn  = "warn"
	statusInfo  = "info"
)

type Check struct{ Name, Status, Detail string }

func Run(cfg *config.Config, cfgPath string) []Check {
	var out []Check
	if cfgPath == "" {
		out = append(out, Check{checkConfig, statusOK, "built-in defaults"})
	} else {
		out = append(out, Check{checkConfig, statusOK, cfgPath})
	}
	if _, err := exec.LookPath("git"); err != nil {
		out = append(out, Check{checkGit, statusWarn, "git not found"})
	} else {
		cur, exists, err := gitinit.Current(true)
		switch {
		case err != nil:
			out = append(out, Check{checkGit, statusWarn, err.Error()})
		case !exists:
			out = append(out, Check{checkGit, statusWarn, "pretty.nopii-v1 not configured (run: nopii init git)"})
		case cur != gitinit.Expected():
			out = append(out, Check{checkGit, statusWarn, "pretty.nopii-v1 differs from expected value"})
		default:
			out = append(out, Check{checkGit, statusOK, "pretty.nopii-v1 configured"})
		}
	}
	if cfg != nil {
		if v := cfg.Key.Env; v != "" {
			out = append(out, Check{checkKey, statusInfo, fmt.Sprintf("expected via %s (value not displayed)", v)})
		}
	}
	return out
}
