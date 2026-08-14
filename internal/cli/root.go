package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/doctor"
	"github.com/iilei/nopii/internal/gitinit"
	"github.com/iilei/nopii/internal/pseudonym"
	"github.com/iilei/nopii/internal/recognizer"
	"github.com/iilei/nopii/internal/stream"
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
		case "init":
			return runInit(args[1:], out)
		case "doctor":
			return runDoctor(args[1:], out, errOut)
		case "config":
			return runConfig(args[1:], out, errOut)
		case "version":
			fmt.Fprintln(out, version)
			return nil
		case "help", "--help", "-h":
			printHelp(out)
			return nil
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
	key, _, err := config.ResolveKey(cfg, o.keyEnv, o.keyFile, nil)
	if err != nil {
		return err
	}
	gen := pseudonym.New(key, cfg.Scope, cfg.Output.TokenLength)
	rec := recognizer.New(cfg, gen)
	return stream.New(gen, rec).Process(in, out)
}

func filterFlags(errOut io.Writer) (*flag.FlagSet, *options) {
	o := &options{}
	fs := flag.NewFlagSet("nopii", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&o.configFile, "config", "", "explicit config file")
	fs.StringVar(&o.scope, "scope", "", "pseudonymization scope")
	fs.StringVar(&o.keyEnv, "key-env", "", "read key from environment variable")
	fs.StringVar(&o.keyFile, "key-file", "", "read key from file")
	return fs, o
}

func runInit(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "git" {
		return fmt.Errorf("usage: nopii init git [--local] [--force] [--remove]")
	}
	fs := flag.NewFlagSet("nopii init git", flag.ContinueOnError)
	local := fs.Bool("local", false, "repository-local Git config")
	force := fs.Bool("force", false, "replace differing value")
	remove := fs.Bool("remove", false, "remove integration")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	global := !*local
	if *remove {
		if err := gitinit.Remove(global); err != nil {
			return err
		}
		fmt.Fprintln(out, "removed "+gitinit.Key)
		return nil
	}
	status, err := gitinit.Install(global, *force)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s %s\n", status, gitinit.Key)
	return nil
}

func commonConfigFlags(name string, args []string, errOut io.Writer) (config.Config, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)
	path := fs.String("config", "", "explicit config file")
	if err := fs.Parse(args); err != nil {
		return config.Config{}, "", err
	}
	if fs.NArg() != 0 {
		return config.Config{}, "", fmt.Errorf("unexpected arguments")
	}
	return config.Load(*path)
}
func runDoctor(args []string, out, errOut io.Writer) error {
	cfg, path, err := commonConfigFlags("nopii doctor", args, errOut)
	if err != nil {
		return err
	}
	for _, c := range doctor.Run(cfg, path) {
		fmt.Fprintf(out, "%-10s %-5s %s\n", c.Name, c.Status, c.Detail)
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
	fmt.Fprintf(out, "config = %q\nversion = %d\nscope = %q\nkey_env = %q\nkey_file = %q\ntoken_length = %d\n", path, cfg.Version, cfg.Scope, cfg.Key.Env, cfg.Key.File, cfg.Output.TokenLength)
	return nil
}
func printHelp(w io.Writer) {
	fmt.Fprint(w, `nopii - deterministic PII pseudonymization for streams

Usage:
  command | nopii [flags]
  nopii init git [--local] [--force] [--remove]
  nopii doctor [--config path]
  nopii config [--config path]
  nopii version

Filter flags:
  --config PATH
  --scope NAME
  --key-env NAME
  --key-file PATH
`)
}
