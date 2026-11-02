// Package run orchestrates config, preflight, proxy and bwrap.
package run

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
	"github.com/gustavoxffff/agentsandboxing/internal/sandbox"
)

// Options are the parameters of an asb execution.
type Options struct {
	ProjectDir string
	Overrides  config.Overrides
	PrintOnly  bool
	Command    []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	LookupEnv func(string) (string, bool)
	SelfPath  string
}

// Execute resolves, prints, validates and runs the sandbox. It returns the
// command exit code. ErrNoProfile is returned without executing anything.
func Execute(o Options) (int, error) {
	started := time.Now()
	if o.Stdin == nil {
		o.Stdin = os.Stdin
	}
	if o.Stdout == nil {
		o.Stdout = os.Stdout
	}
	if o.Stderr == nil {
		o.Stderr = os.Stderr
	}
	if o.LookupEnv == nil {
		o.LookupEnv = os.LookupEnv
	}
	project := o.ProjectDir
	if project == "" {
		project = "."
	}

	cfg, err := config.Resolve(project, o.Overrides)
	if err != nil {
		return 2, err
	}
	layout, err := sandbox.NewLayout(cfg.ProjectDir, cfg.SandboxID)
	if err != nil {
		return 2, err
	}

	command := append([]string(nil), o.Command...)
	if len(command) == 0 {
		command = []string{"opencode", "--standalone"}
	}
	// --standalone is required in v1: the server that runs tools must start
	// inside the sandbox, never in the host shared service.
	if filepath.Base(command[0]) == "opencode" && !hasArg(command[1:], "--standalone") {
		command = append([]string{command[0], "--standalone"}, command[1:]...)
	}

	configOut := o.Stderr
	if o.PrintOnly {
		configOut = o.Stdout
	}
	PrintConfig(configOut, cfg, layout, command)
	if o.PrintOnly {
		return 0, nil
	}

	// The resolved config has already been printed; preflight failures happen
	// later.
	if err := sandbox.RequireBwrap(); err != nil {
		return 1, err
	}
	if err := layout.CreateDirs(); err != nil {
		return 1, err
	}

	env, err := buildEnv(cfg, layout, o.LookupEnv)
	if err != nil {
		return 1, err
	}

	var proxyCmd *exec.Cmd
	if cfg.Profile == config.ProfileAllowlist {
		proxyCmd, err = startProxy(o, layout, cfg.NetworkAllow)
		if err != nil {
			return 1, err
		}
		defer stopProxy(proxyCmd)
	}

	resolv, err := sandbox.ResolvFile(sandbox.ResolvContent(cfg.Profile))
	if err != nil {
		return 1, err
	}
	defer resolv.Close()

	selfPath := o.SelfPath
	if selfPath == "" {
		selfPath, err = os.Executable()
		if err != nil {
			return 1, fmt.Errorf("discover asb binary: %w", err)
		}
	}
	if resolved, err := filepath.EvalSymlinks(selfPath); err == nil {
		selfPath = resolved
	}

	args, err := sandbox.BuildArgs(sandbox.Options{
		ProjectDir: cfg.ProjectDir,
		Profile:    cfg.Profile,
		Layout:     layout,
		Env:        env,
		Command:    command,
		SelfPath:   selfPath,
		ResolvDest: sandbox.ResolvDestination(),
	})
	if err != nil {
		return 1, err
	}

	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		return 1, fmt.Errorf("bwrap not found in PATH: %w", err)
	}

	cmd := exec.Command(bwrapPath, args...)
	cmd.Stdin = o.Stdin
	cmd.Stdout = o.Stdout
	cmd.Stderr = o.Stderr
	cmd.ExtraFiles = []*os.File{resolv}

	fmt.Fprintf(o.Stderr, "asb: startup %.1fms\n", float64(time.Since(started))/float64(time.Millisecond))

	if err := runWithSignals(cmd); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("run bwrap: %w", err)
	}
	return 0, nil
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func buildEnv(cfg config.Resolved, layout sandbox.Layout, lookup func(string) (string, bool)) ([]string, error) {
	values := map[string]string{}
	for _, kv := range cfg.BuildEnv(lookup) {
		name, value, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid environment: %q", kv)
		}
		values[name] = value
	}
	// HOME and XDG are set by the tool, not inherited.
	tool := map[string]string{
		"HOME":            layout.Home,
		"TMPDIR":          "/tmp",
		"XDG_CONFIG_HOME": layout.InsideConfigHome,
		"XDG_DATA_HOME":   layout.InsideDataHome,
		"XDG_STATE_HOME":  layout.InsideStateHome,
		"XDG_CACHE_HOME":  layout.InsideCacheHome,
		"XDG_RUNTIME_DIR": layout.InsideRuntimeDir,
	}
	for name, value := range tool {
		values[name] = value
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	env := make([]string, 0, len(names))
	for _, name := range names {
		env = append(env, name+"="+values[name])
	}
	return env, nil
}

func startProxy(o Options, layout sandbox.Layout, allow []string) (*exec.Cmd, error) {
	selfPath := o.SelfPath
	var err error
	if selfPath == "" {
		selfPath, err = os.Executable()
		if err != nil {
			return nil, fmt.Errorf("discover asb binary: %w", err)
		}
	}
	if resolved, err := filepath.EvalSymlinks(selfPath); err == nil {
		selfPath = resolved
	}
	if err := layout.RemoveStaleSocket(); err != nil {
		return nil, err
	}

	args := []string{"proxy", "--socket", layout.ProxySocket, "--ready-fd", "3"}
	if os.Getenv("ASB_PROXY_LOG") != "" {
		args = append(args, "--verbose")
	}
	for _, host := range allow {
		args = append(args, "--allow", host)
	}
	cmd := exec.Command(selfPath, args...)
	cmd.Stdout = o.Stderr
	cmd.Stderr = o.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}

	readyR, readyW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("proxy readiness pipe: %w", err)
	}
	cmd.ExtraFiles = []*os.File{readyW}
	if err := cmd.Start(); err != nil {
		readyR.Close()
		readyW.Close()
		return nil, fmt.Errorf("start proxy: %w", err)
	}
	readyW.Close()

	_ = readyR.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 1)
	n, err := readyR.Read(buf)
	readyR.Close()
	if err != nil || n != 1 || buf[0] != 'R' {
		stopProxy(cmd)
		return nil, fmt.Errorf("proxy did not become ready; failing closed")
	}
	return cmd, nil
}

func stopProxy(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}

func runWithSignals(cmd *exec.Cmd) error {
	sigCh := make(chan os.Signal, 4)
	done := make(chan struct{})
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigCh)
	go func() {
		for {
			select {
			case sig := <-sigCh:
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()
	err := cmd.Run()
	close(done)
	return err
}
