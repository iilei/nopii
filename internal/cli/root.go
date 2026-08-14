// Package cli wires flags, config loading, and subcommands for the nopii CLI.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/configinit"
	"github.com/iilei/nopii/internal/doctor"
	"github.com/iilei/nopii/internal/gitinit"
	"github.com/iilei/nopii/internal/pseudonym"
	"github.com/iilei/nopii/internal/recognizer"
	"github.com/iilei/nopii/internal/stream"
)

const (
	cmdInit    = "init"
	cmdDoctor  = "doctor"
	cmdConfig  = "config"
	cmdVersion = "version"
)

var version = "dev"

type options struct{ configFile, scope, keyEnv, keyFile string }

func Execute() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "nopii:", err)
		os.Exit(1)
	}
}

func run(args []string, in io.Reader, out, errOut io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case cmdInit:
			return runInit(args[1:], out)
		case cmdDoctor:
			return runDoctor(args[1:], out, errOut)
		case cmdConfig:
			return runConfig(args[1:], out, errOut)
		case cmdVersion:
			_, err := fmt.Fprintln(out, version)
			return err
		case "help", "--help", "-h":
			return printHelp(out)
		}
	}
	fs, o := filterFlags(errOut)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	cfg, _, err := config.Load(o.configFile)
	if err != nil {
		return err
	}
	if o.scope != "" {
		cfg.Scope = o.scope
	}
	key, _, err := config.ResolveKey(&cfg, o.keyEnv, o.keyFile, nil)
	if err != nil {
		return err
	}
	gen := pseudonym.New(key, cfg.Scope, cfg.Output.TokenLength)
	rec := recognizer.New(&cfg, gen)
	return stream.New(gen, rec, cfg.Git.DateClamp).Process(in, out)
}

func filterFlags(errOut io.Writer) (*flag.FlagSet, *options) {
	o := &options{}
	fs := flag.NewFlagSet("nopii", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&o.configFile, cmdConfig, "", "explicit config file")
	fs.StringVar(&o.scope, "scope", "", "pseudonymization scope")
	fs.StringVar(&o.keyEnv, "key-env", "", "read key from environment variable")
	fs.StringVar(&o.keyFile, "key-file", "", "read key from file")
	return fs, o
}

func runInit(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: nopii init git [--local] [--force] [--remove]\n       nopii init config [--force]")
	}
	switch args[0] {
	case "git":
		return runInitGit(args[1:], out)
	case cmdConfig:
		return runInitConfig(args[1:], out)
	default:
		return fmt.Errorf("unknown init target %q; expected git or config", args[0])
	}
}

func runInitGit(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("nopii init git", flag.ContinueOnError)
	local := fs.Bool("local", false, "repository-local Git config")
	force := fs.Bool("force", false, "replace differing value")
	remove := fs.Bool("remove", false, "remove integration")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	global := !*local
	if *remove {
		if err := gitinit.Remove(global); err != nil {
			return err
		}
		_, err := fmt.Fprintln(out, "removed "+gitinit.Key)
		return err
	}
	status, err := gitinit.Install(global, *force)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s %s\n", status, gitinit.Key)
	return err
}

func runInitConfig(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("nopii init config", flag.ContinueOnError)
	force := fs.Bool("force", false, "overwrite existing config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	path, err := configinit.Create(dir, *force)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "created %s\n", path)
	return err
}

func commonConfigFlags(name string, args []string, errOut io.Writer) (config.Config, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)
	path := fs.String(cmdConfig, "", "explicit config file")
	if err := fs.Parse(args); err != nil {
		return config.Config{}, "", err
	}
	if fs.NArg() != 0 {
		return config.Config{}, "", errors.New("unexpected arguments")
	}
	return config.Load(*path)
}

func runDoctor(args []string, out, errOut io.Writer) error {
	cfg, path, err := commonConfigFlags("nopii doctor", args, errOut)
	if err != nil {
		return err
	}
	for _, c := range doctor.Run(&cfg, path) {
		if _, err := fmt.Fprintf(out, "%-10s %-5s %s\n", c.Name, c.Status, c.Detail); err != nil {
			return err
		}
	}
	return nil
}

func runConfig(args []string, out, errOut io.Writer) error {
	cfg, path, err := commonConfigFlags("nopii config", args, errOut)
	if err != nil {
		return err
	}
	if path == "" {
		path = "<built-in>"
	}
	_, err = fmt.Fprintf(
		out,
		"config = %q\nversion = %d\nscope = %q\nkey_env = %q\nkey_file = %q\ntoken_length = %d\ngit.date_clamp.enabled = %v\ngit.date_clamp.granularity_seconds = %d\n",
		path,
		cfg.Version,
		cfg.Scope,
		cfg.Key.Env,
		cfg.Key.File,
		cfg.Output.TokenLength,
		cfg.Git.DateClamp.Enabled,
		cfg.Git.DateClamp.GranularitySeconds,
	)
	return err
}

func printHelp(w io.Writer) error {
	_, err := fmt.Fprint(w, `nopii - deterministic PII pseudonymization for streams

Usage:
  command | nopii [flags]
  nopii init git [--local] [--force] [--remove]
  nopii init config [--force]
  nopii doctor [--config path]
  nopii config [--config path]
  nopii version

Filter flags:
  --config PATH
  --scope NAME
  --key-env NAME
  --key-file PATH
`)
	return err
}
