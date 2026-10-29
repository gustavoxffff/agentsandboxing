// Comando asb: sandbox de agentes com política de rede declarativa.
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
			// Permite `asb --print-config` e `asb --profile none -- cmd`.
			return runCmd(args, false)
		}
		fmt.Fprintf(os.Stderr, "asb: comando desconhecido: %s\n\n", args[0])
		usage(os.Stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `asb — sandbox de agentes com política de rede

Uso:
  asb run [flags] [--] [comando...]
  asb print-config [flags]
  asb init --profile host|none|allowlist [flags]
  asb doctor
  asb version

Flags de run:
  -C, --project DIR     raiz do projeto (default: diretório atual)
      --profile P       host | none | allowlist (sobrescreve asb.toml)
      --allow HOST      destino permitido; repetível; substitui a lista
      --env-allow VAR   variável de ambiente permitida; repetível; substitui
      --sandbox-id ID   sobrescreve o id derivado do projeto
      --print-config    imprime a config resolvida e não executa

Precedência: flag > asb.toml do projeto > config global > nenhum default.
Sem perfil configurado o asb falha pedindo uma escolha explícita.
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
	fs.StringVar(&project, "C", ".", "raiz do projeto")
	fs.StringVar(&project, "project", ".", "raiz do projeto")
	fs.StringVar(&profile, "profile", "", "perfil de rede")
	fs.Var(&allow, "allow", "destino permitido (repetível)")
	fs.Var(&envAllow, "env-allow", "variável permitida (repetível)")
	fs.StringVar(&sandboxID, "sandbox-id", "", "id do sandbox")
	fs.BoolVar(&printCfg, "print-config", false, "imprime a config e sai")
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
			fmt.Fprintln(os.Stderr, "asb: nenhum perfil de rede configurado.")
			fmt.Fprintln(os.Stderr, "Escolha explicitamente em asb.toml:")
			fmt.Fprintln(os.Stderr, "  [network]")
			fmt.Fprintln(os.Stderr, "  profile = \"host\" | \"none\" | \"allowlist\"")
			fmt.Fprintln(os.Stderr, "ou use --profile. Para criar o arquivo: asb init --profile none")
			return 2
		}
		fmt.Fprintf(os.Stderr, "asb: %v\n", err)
		return code
	}
	return code
}

func initCmd(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var project, profile string
	var allow listFlag
	var force bool
	fs.StringVar(&project, "C", ".", "raiz do projeto")
	fs.StringVar(&project, "project", ".", "raiz do projeto")
	fs.StringVar(&profile, "profile", "", "perfil de rede (obrigatório)")
	fs.Var(&allow, "allow", "destino permitido (repetível)")
	fs.BoolVar(&force, "force", false, "sobrescrever asb.toml existente")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if profile == "" {
		fmt.Fprintln(os.Stderr, "asb init: --profile é obrigatório (host, none ou allowlist)")
		return 2
	}
	if err := config.ValidateProfile(config.Profile(profile)); err != nil {
		fmt.Fprintf(os.Stderr, "asb init: %v\n", err)
		return 2
	}
	if profile != string(config.ProfileAllowlist) && allow.set {
		fmt.Fprintln(os.Stderr, "asb init: --allow só é válido com --profile allowlist")
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
		fmt.Fprintf(os.Stderr, "asb init: %s já existe (use --force para sobrescrever)\n", path)
		return 2
	}

	var b strings.Builder
	b.WriteString("# Configuração do asb (versionada). Precedência: CLI > projeto > global.\n")
	b.WriteString("version = 1\n\n[network]\n")
	fmt.Fprintf(&b, "profile = %q\n", profile)
	if len(normalized) > 0 {
		b.WriteString("allow = [\n")
		for _, h := range normalized {
			fmt.Fprintf(&b, "  %q,\n", h)
		}
		b.WriteString("]\n")
	}
	b.WriteString("\n# Variáveis herdadas do host. HOME e proxy são definidos pela ferramenta.\n")
	b.WriteString("# [env]\n# allow = [\"PATH\", \"TERM\", \"LANG\"]\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "asb init: %v\n", err)
		return 1
	}
	fmt.Printf("asb: criado %s (perfil %s)\n", path, profile)
	return 0
}

func doctorCmd(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var project string
	fs.StringVar(&project, "C", ".", "raiz do projeto")
	fs.StringVar(&project, "project", ".", "raiz do projeto")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ok := true
	checks, err := sandbox.Preflight()
	for _, check := range checks {
		status := "ok"
		if !check.OK {
			status = "FALHA"
			ok = false
		}
		fmt.Printf("[%-5s] %-16s %s\n", status, check.Name, check.Detail)
	}
	if err != nil {
		fmt.Printf("[FALHA] %v\n", err)
		ok = false
	}
	for _, name := range []string{"opencode", "git"} {
		if path, err := exec.LookPath(name); err == nil {
			fmt.Printf("[ok   ] %-16s %s\n", name, path)
		} else {
			fmt.Printf("[aviso] %-16s não encontrado\n", name)
		}
	}
	if global, err := config.GlobalPath(); err == nil {
		fmt.Printf("[info ] %-16s %s\n", "config global", global)
	}
	resolv := sandbox.ResolvDestination()
	if data, err := os.ReadFile(resolv); err == nil && strings.Contains(string(data), "nameserver") {
		fmt.Printf("[ok   ] %-16s %s\n", "resolv.conf", resolv)
	} else {
		fmt.Printf("[aviso] %-16s sem nameserver legível em %s (WSL? veja docs/asb.md)\n", "resolv.conf", resolv)
	}
	abs, _ := filepath.Abs(project)
	if _, err := os.Stat(filepath.Join(abs, "asb.toml")); err == nil {
		fmt.Printf("[info ] %-16s %s\n", "asb.toml", filepath.Join(abs, "asb.toml"))
	} else {
		fmt.Printf("[aviso] %-16s ausente em %s\n", "asb.toml", abs)
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
	fs.StringVar(&socket, "socket", "", "caminho do socket Unix")
	fs.Var(&allow, "allow", "host permitido (repetível)")
	fs.IntVar(&readyFD, "ready-fd", -1, "fd para sinalizar readiness")
	fs.BoolVar(&verbose, "verbose", false, "registrar conexões permitidas")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if socket == "" {
		fmt.Fprintln(os.Stderr, "asb proxy: --socket é obrigatório")
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
	fs.StringVar(&socket, "socket", "", "caminho do socket Unix do proxy")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if socket == "" {
		fmt.Fprintln(os.Stderr, "asb bridge: --socket é obrigatório")
		return 2
	}
	command := fs.Args()
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "asb bridge: comando ausente (use -- comando...)")
		return 2
	}
	code, err := bridge.Run(socket, command, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "asb bridge: %v\n", err)
		return 1
	}
	return code
}
