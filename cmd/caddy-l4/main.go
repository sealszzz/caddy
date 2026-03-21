package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	caddy "github.com/caddyserver/caddy/v2"

	_ "github.com/caddyserver/caddy/v2/modules/logging"
	_ "github.com/mholt/caddy-l4"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildTime    = "unknown"
)

func main() {
	os.Exit(realMain(os.Args[1:]))
}

func realMain(args []string) int {
	if len(args) == 0 {
		usage(os.Stderr)
		return 2
	}

	switch args[0] {
	case "run":
		return runCmd(args[1:])
	case "validate":
		return validateCmd(args[1:])
	case "version", "--version", "-version", "-v":
		fmt.Printf("caddy-l4 %s\ncommit: %s\nbuild_time: %s\n", buildVersion, buildCommit, buildTime)
		return 0
	case "help", "--help", "-h":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", args[0])
		usage(os.Stderr)
		return 2
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  caddy-l4 run --config /path/to/config.json")
	fmt.Fprintln(w, "  caddy-l4 validate --config /path/to/config.json")
	fmt.Fprintln(w, "  caddy-l4 version")
}

func parseConfigFlag(name string, args []string) (string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	config := fs.String("config", "", "path to JSON config file")
	fs.StringVar(config, "c", "", "path to JSON config file")

	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() != 0 {
		return "", fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if *config == "" {
		return "", errors.New("--config is required")
	}
	return *config, nil
}

func readConfig(path string) ([]byte, *caddy.Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var cfg caddy.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, nil, err
	}

	return raw, &cfg, nil
}

func validateCmd(args []string) int {
	configPath, err := parseConfigFlag("validate", args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate: %v\n", err)
		return 2
	}

	_, cfg, err := readConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate: read config: %v\n", err)
		return 1
	}

	if err := caddy.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "validate: invalid config: %v\n", err)
		return 1
	}

	fmt.Printf("valid: %s\n", configPath)
	return 0
}

func runCmd(args []string) int {
	configPath, err := parseConfigFlag("run", args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		return 2
	}

	raw, _, err := readConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: read config: %v\n", err)
		return 1
	}

	if err := caddy.Load(raw, true); err != nil {
		fmt.Fprintf(os.Stderr, "run: load config: %v\n", err)
		return 1
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	sig := <-sigCh
	fmt.Fprintf(os.Stderr, "received signal: %s, stopping\n", sig)

	if err := caddy.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "run: stop: %v\n", err)
		return 1
	}

	return 0
}
