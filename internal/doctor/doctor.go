package doctor

import (
	"fmt"
	"os/exec"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/gitinit"
)

type Check struct{ Name, Status, Detail string }

func Run(cfg config.Config, cfgPath string) []Check {
	var out []Check
	if cfgPath == "" {
		out = append(out, Check{"config", "ok", "built-in defaults"})
	} else {
		out = append(out, Check{"config", "ok", cfgPath})
	}
	if _, err := exec.LookPath("git"); err != nil {
		out = append(out, Check{"git", "warn", "git not found"})
	} else {
		cur, exists, err := gitinit.Current(true)
		switch {
		case err != nil:
			out = append(out, Check{"git", "warn", err.Error()})
		case !exists:
			out = append(out, Check{"git", "warn", "pretty.nopii-v1 not configured (run: nopii init git)"})
		case cur != gitinit.Expected():
			out = append(out, Check{"git", "warn", "pretty.nopii-v1 differs from expected value"})
		default:
			out = append(out, Check{"git", "ok", "pretty.nopii-v1 configured"})
		}
	}
	if v := cfg.Key.Env; v != "" {
		out = append(out, Check{"key", "info", fmt.Sprintf("expected via %s (value not displayed)", v)})
	}
	return out
}
