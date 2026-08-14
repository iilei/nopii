package gitinit

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/iilei/nopii/internal/stream"
)

const Key = "pretty.nopii-v1"

func Expected() string { return stream.GitPrettyV1 }

func Current(global bool) (string, bool, error) {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, "--get", Key)
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		ee := &exec.ExitError{}
		if errors.As(err, &ee) {
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
		return "already configured", nil
	}
	if exists && !force {
		return "", fmt.Errorf("%s exists with a different value; use --force to replace it", Key)
	}
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, Key, Expected())
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("git config: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return "configured", nil
}

func Remove(global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, "--unset", Key)
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		ee := &exec.ExitError{}
		if errors.As(err, &ee) {
			return nil
		}
		return fmt.Errorf("git config: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
