// Package gitinit manages the Git configuration that enables the nopii pretty format.
package gitinit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/iilei/nopii/internal/stream"
)

const (
	Key           = "pretty.nopii-v1"
	gitConfig     = "config"
	showSignature = "log.showSignature"
)

func Expected() string { return stream.GitPrettyV1 }

// Current reports the configured value, whether it is set, and any error.
// Unnamed returns are intentional to satisfy nonamedreturns; see the
// gocritic exception below.
//
//nolint:gocritic // unnamedResult conflicts with nonamedreturns for this signature
func Current(global bool) (string, bool, error) {
	args := []string{gitConfig}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, "--get", Key)
	cmd := exec.CommandContext(context.Background(), "git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(out.String()), true, nil
}

func Install(global, force bool) (string, error) {
	cur, exists, err := Current(global)
	if err != nil {
		return "", err
	}
	if exists && cur == Expected() {
		return warnIfShowSignature(global, "already configured")
	}
	if exists && !force {
		return "", fmt.Errorf("%s exists with a different value; use --force to replace it", Key)
	}
	args := []string{gitConfig}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, Key, Expected())
	cmd := exec.CommandContext(context.Background(), "git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git config: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return warnIfShowSignature(global, "configured")
}

// warnIfShowSignature appends a warning to status when log.showSignature is
// enabled in the relevant git config scope. nopii scrubs annotation lines
// through its recognizers regardless of signing backend (GPG, SSH, X.509),
// but signer names and key fingerprints are not covered by built-in patterns.
func warnIfShowSignature(global bool, status string) (string, error) {
	args := []string{gitConfig}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, "--get", showSignature)
	out, err := exec.CommandContext(context.Background(), "git", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return status, nil
		}
		return status, err
	}
	if strings.TrimSpace(string(out)) == "true" {
		return status + "\nwarn: log.showSignature is enabled; gpg annotation lines will be" +
			" passed through recognizers only — signer names and key fingerprints are not pseudonymized", nil
	}
	return status, nil
}

func Remove(global bool) error {
	args := []string{gitConfig}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, "--unset", Key)
	cmd := exec.CommandContext(context.Background(), "git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil
		}
		return fmt.Errorf("git config: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
