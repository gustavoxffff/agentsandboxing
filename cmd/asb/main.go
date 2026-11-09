// Command asb: agent sandbox with a declarative network policy.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gustavoxffff/agentsandboxing/internal/bridge"
	"github.com/gustavoxffff/agentsandboxing/internal/config"
	"github.com/gustavoxffff/agentsandboxing/internal/proxy"
	"github.com/gustavoxffff/agentsandboxing/internal/run"
	"github.com/gustavoxffff/agentsandboxing/internal/sandbox"
)

const version = "0.1.0-dev"

// execProcess is swapped in tests; syscall.Exec replaces the process.
var execProcess = syscall.Exec

func main() {
	os.Exit(cli(os.Args[1:]))
}

func cli(args []string) int {
	if len(args) == 0 {
		usage(os.Stderr)
		return 2
	}
	switch args[0] {
	case "run":
		return runCmd(args[1:], false)
	case "print-config":
		return runCmd(args[1:], true)
	case "exec":
		return execCmd(args[1:])
	case "init":
		return initCmd(args[1:])
	case "doctor":
		return doctorCmd(args[1:])
	case "proxy":
		return proxyCmd(args[1:])
	case "bridge":
		return bridgeCmd(args[1:])
	case "version", "--version", "-v":
		fmt.Printf("asb %s\n", version)
		return 0
	case "help", "--help", "-h":
		usage(os.Stdout)
		return 0
	default:
		if strings.HasPrefix(args[0], "-") {
			// Allows `asb --print-config` and `asb --profile none -- cmd`.
			return runCmd(args, false)
		}
		fmt.Fprintf(os.Stderr, "asb: unknown command: %s\n\n", args[0])
		usage(os.Stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `asb — agent sandbox with declarative network policy

Usage:
  asb run [flags] [--] [command...]
  asb print-config [flags]
  asb exec --env-file FILE -- command...
  asb init --profile host|none|allowlist [flags]
  asb doctor
  asb version

Flags for run:
  -C, --project DIR     project root (default: current directory)
      --profile P       host | none | allowlist (overrides asb.toml)
      --allow HOST      allowed destination; repeatable; replaces the list
      --env-allow VAR   allowed environment variable; repeatable; replaces the list
      --sandbox-id ID   overrides the id derived from the project
      --print-config    prints the resolved config and does not run

Precedence: flag > project asb.toml > global config > no default.
Without a configured profile, asb fails asking for an explicit choice.
`)
}

type listFlag struct {
	values []string
	set    bool
}

func (l *listFlag) String() string { return strings.Join(l.values, ",") }

func (l *listFlag) Set(value string) error {
	l.set = true
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			l.values = append(l.values, item)
		}
	}
	return nil
}

func runCmd(args []string, forcePrint bool) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var project, profile, sandboxID string
	var allow, envAllow listFlag
	var printCfg bool
	fs.StringVar(&project, "C", ".", "project root")
	fs.StringVar(&project, "project", ".", "project root")
	fs.StringVar(&profile, "profile", "", "network profile")
	fs.Var(&allow, "allow", "allowed destination (repeatable)")
	fs.Var(&envAllow, "env-allow", "allowed variable (repeatable)")
	fs.StringVar(&sandboxID, "sandbox-id", "", "sandbox id")
	fs.BoolVar(&printCfg, "print-config", false, "prints the config and exits")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	overrides := config.Overrides{}
	if set["profile"] {
		overrides.Profile = &profile
	}
	if allow.set {
		overrides.NetworkAllow = &allow.values
	}
	if envAllow.set {
		overrides.EnvAllow = &envAllow.values
	}
	if set["sandbox-id"] {
		overrides.SandboxID = &sandboxID
	}

	code, err := run.Execute(run.Options{
		ProjectDir: project,
		Overrides:  overrides,
		PrintOnly:  forcePrint || printCfg,
		Command:    fs.Args(),
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
	if err != nil {
		if errors.Is(err, config.ErrNoProfile) {
			fmt.Fprintln(os.Stderr, "asb: no network profile configured.")
			fmt.Fprintln(os.Stderr, "Choose explicitly in asb.toml:")
			fmt.Fprintln(os.Stderr, "  [network]")
			fmt.Fprintln(os.Stderr, "  profile = \"host\" | \"none\" | \"allowlist\"")
			fmt.Fprintln(os.Stderr, "or use --profile. To create the file: asb init --profile none")
			return 2
		}
		fmt.Fprintf(os.Stderr, "asb: %v\n", err)
		return code
	}
	return code
}

// execCmd is the in-sandbox shim: it reads the env file, removes it, and
// replaces itself with the command. Values never reach the bwrap argv.
func execCmd(args []string) int {
	fs := flag.NewFlagSet("exec", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var envFile string
	fs.StringVar(&envFile, "env-file", "", "environment file read before exec")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	command := fs.Args()
	if envFile == "" {
		fmt.Fprintln(os.Stderr, "asb exec: --env-file is required")
		return 2
	}
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "asb exec: missing command (use -- command...)")
		return 2
	}

	entries, err := sandbox.ReadEnvFile(envFile)
	if err != nil {
		_ = os.Remove(envFile)
		fmt.Fprintf(os.Stderr, "asb exec: %v\n", err)
		return 1
	}
	if err := os.Remove(envFile); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "asb exec: remove env file: %v\n", err)
		return 1
	}
	if err := execProcess(command[0], command, sandbox.MergeEnv(os.Environ(), entries)); err != nil {
		fmt.Fprintf(os.Stderr, "asb exec: %v\n", err)
		return 1
	}
	return 0
}

func initCmd(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var project, profile string
	var allow listFlag
	var force bool
	fs.StringVar(&project, "C", ".", "project root")
	fs.StringVar(&project, "project", ".", "project root")
	fs.StringVar(&profile, "profile", "", "network profile (required)")
	fs.Var(&allow, "allow", "allowed destination (repeatable)")
	fs.BoolVar(&force, "force", false, "overwrite an existing asb.toml")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if profile == "" {
		fmt.Fprintln(os.Stderr, "asb init: --profile is required (host, none or allowlist)")
		return 2
	}
	if err := config.ValidateProfile(config.Profile(profile)); err != nil {
		fmt.Fprintf(os.Stderr, "asb init: %v\n", err)
		return 2
	}
	if profile != string(config.ProfileAllowlist) && allow.set {
		fmt.Fprintln(os.Stderr, "asb init: --allow is only valid with --profile allowlist")
		return 2
	}
	normalized := make([]string, 0, len(allow.values))
	for _, entry := range allow.values {
		h, err := config.NormalizeHost(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "asb init: %v\n", err)
			return 2
		}
		normalized = append(normalized, h)
	}

	abs, err := filepath.Abs(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "asb init: %v\n", err)
		return 2
	}
	path := filepath.Join(abs, "asb.toml")
	if _, err := os.Stat(path); err == nil && !force {
		fmt.Fprintf(os.Stderr, "asb init: %s already exists (use --force to overwrite)\n", path)
		return 2
	}

	var b strings.Builder
	b.WriteString("# asb configuration (versioned). Precedence: CLI > project > global.\n")
	b.WriteString("version = 1\n\n[network]\n")
	fmt.Fprintf(&b, "profile = %q\n", profile)
	if len(normalized) > 0 {
		b.WriteString("allow = [\n")
		for _, h := range normalized {
			fmt.Fprintf(&b, "  %q,\n", h)
		}
		b.WriteString("]\n")
	}
	b.WriteString("\n# Variables inherited from the host. HOME and proxy are set by the tool.\n")
	b.WriteString("# [env]\n# allow = [\"PATH\", \"TERM\", \"LANG\"]\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "asb init: %v\n", err)
		return 1
	}
	fmt.Printf("asb: created %s (profile %s)\n", path, profile)
	return 0
}

func doctorCmd(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var project string
	fs.StringVar(&project, "C", ".", "project root")
	fs.StringVar(&project, "project", ".", "project root")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ok := true
	checks, err := sandbox.Preflight()
	for _, check := range checks {
		status := "ok"
		if !check.OK {
			status = "FAIL"
			ok = false
		}
		fmt.Printf("[%-5s] %-16s %s\n", status, check.Name, check.Detail)
	}
	if err != nil {
		fmt.Printf("[FAIL ] %v\n", err)
		ok = false
	}
	for _, name := range []string{"opencode", "git"} {
		if path, err := exec.LookPath(name); err == nil {
			fmt.Printf("[ok   ] %-16s %s\n", name, path)
		} else {
			fmt.Printf("[warn ] %-16s not found\n", name)
		}
	}
	if global, err := config.GlobalPath(); err == nil {
		fmt.Printf("[info ] %-16s %s\n", "global config", global)
	}
	resolv := sandbox.ResolvDestination()
	if data, err := os.ReadFile(resolv); err == nil && strings.Contains(string(data), "nameserver") {
		fmt.Printf("[ok   ] %-16s %s\n", "resolv.conf", resolv)
	} else {
		fmt.Printf("[warn ] %-16s no readable nameserver in %s (WSL? see docs/asb.md)\n", "resolv.conf", resolv)
	}
	abs, _ := filepath.Abs(project)
	if _, err := os.Stat(filepath.Join(abs, "asb.toml")); err == nil {
		fmt.Printf("[info ] %-16s %s\n", "asb.toml", filepath.Join(abs, "asb.toml"))
	} else {
		fmt.Printf("[warn ] %-16s missing in %s\n", "asb.toml", abs)
	}
	if !ok {
		return 1
	}
	return 0
}

func proxyCmd(args []string) int {
	fs := flag.NewFlagSet("proxy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var socket string
	var allow listFlag
	var readyFD int
	var verbose bool
	fs.StringVar(&socket, "socket", "", "Unix socket path")
	fs.Var(&allow, "allow", "allowed host (repeatable)")
	fs.IntVar(&readyFD, "ready-fd", -1, "fd used to signal readiness")
	fs.BoolVar(&verbose, "verbose", false, "log allowed connections")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if socket == "" {
		fmt.Fprintln(os.Stderr, "asb proxy: --socket is required")
		return 2
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "asb proxy: %v\n", err)
		return 1
	}
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "asb proxy: %v\n", err)
		return 1
	}
	l, err := net.Listen("unix", socket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "asb proxy: %v\n", err)
		return 1
	}
	defer os.Remove(socket)
	defer l.Close()
	if err := os.Chmod(socket, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "asb proxy: %v\n", err)
		return 1
	}

	logger := log.New(os.Stderr, "asb proxy: ", log.LstdFlags)
	p := &proxy.Proxy{Allow: allow.values, Logger: logger, Verbose: verbose}

	if readyFD >= 0 {
		f := os.NewFile(uintptr(readyFD), "asb-ready")
		if f != nil {
			_, _ = f.Write([]byte("R"))
			_ = f.Close()
		}
	}

	errCh := make(chan error, 1)
	go func() { errCh <- p.Serve(l) }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	select {
	case <-sigCh:
		_ = l.Close()
		<-errCh
		return 0
	case err := <-errCh:
		if err != nil {
			fmt.Fprintf(os.Stderr, "asb proxy: %v\n", err)
			return 1
		}
		return 0
	}
}

func bridgeCmd(args []string) int {
	fs := flag.NewFlagSet("bridge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var socket string
	fs.StringVar(&socket, "socket", "", "proxy Unix socket path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if socket == "" {
		fmt.Fprintln(os.Stderr, "asb bridge: --socket is required")
		return 2
	}
	command := fs.Args()
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "asb bridge: missing command (use -- command...)")
		return 2
	}
	code, err := bridge.Run(socket, command, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "asb bridge: %v\n", err)
		return 1
	}
	return code
}
