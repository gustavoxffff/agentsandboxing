package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestInitCmdWritesProfileFile(t *testing.T) {
	dir := t.TempDir()
	if code := initCmd([]string{"-profile", "none", "-C", dir}); code != 0 {
		t.Fatalf("initCmd returned %d, want 0", code)
	}
	data, err := os.ReadFile(filepath.Join(dir, "asb.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `profile = "none"`) {
		t.Fatalf("asb.toml does not declare the profile:\n%s", data)
	}

	// An existing file is kept unless --force is given.
	if code := initCmd([]string{"-profile", "none", "-C", dir}); code != 2 {
		t.Fatalf("repeated initCmd returned %d, want 2", code)
	}
	// Invalid input is rejected before anything is written.
	if code := initCmd([]string{"-C", dir}); code != 2 {
		t.Fatalf("initCmd without --profile returned %d, want 2", code)
	}
	if code := initCmd([]string{"-profile", "bogus", "-C", t.TempDir()}); code != 2 {
		t.Fatalf("initCmd with an invalid profile returned %d, want 2", code)
	}
	if code := initCmd([]string{"-profile", "none", "-allow", "example.com", "-C", t.TempDir()}); code != 2 {
		t.Fatalf("initCmd with --allow outside allowlist returned %d, want 2", code)
	}
}

func TestDoctorCmdReportsHealthyHost(t *testing.T) {
	// Preflight must succeed, so provide a bwrap that satisfies the probe.
	bin := t.TempDir()
	bwrap := filepath.Join(bin, "bwrap")
	if err := os.WriteFile(bwrap, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if code := doctorCmd([]string{"-C", t.TempDir()}); code != 0 {
		t.Fatalf("doctorCmd returned %d, want 0 on a healthy host", code)
	}
}

func TestProxyCmdParsesFlagsAndKeepsReadyFDIdle(t *testing.T) {
	// Keep SIGTERM harmless for this process: the reaper below must not kill
	// the test binary before proxyCmd installs its own handler.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)

	saved, dupErr := syscall.Dup(1)
	canCheckFD1 := dupErr == nil
	if !canCheckFD1 {
		t.Logf("descriptor 1 cannot be duplicated (%v): the readiness check is skipped", dupErr)
	}

	stop := make(chan struct{})
	reaperDone := make(chan struct{})
	go func() {
		defer close(reaperDone)
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
			}
		}
	}()
	defer func() {
		close(stop)
		<-reaperDone
		signal.Stop(sigCh)
		if canCheckFD1 {
			_ = syscall.Dup3(saved, 1, 0)
			_ = syscall.Close(saved)
		}
	}()

	socket := filepath.Join(t.TempDir(), "proxy.sock")
	done := make(chan int, 1)
	go func() { done <- proxyCmd([]string{"-socket", socket}) }()

	// A successful flag parse must reach the listen call; a parse failure
	// returns before the socket exists.
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(socket); err == nil {
			break
		}
		select {
		case code := <-done:
			t.Fatalf("proxyCmd returned %d before creating the socket; flag parsing must succeed", code)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("proxyCmd did not create %s", socket)
		}
		time.Sleep(2 * time.Millisecond)
	}

	// proxyCmd serves until it is signalled.
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("proxyCmd returned %d on SIGTERM, want 0", code)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("proxyCmd did not stop on SIGTERM")
	}

	// Without -ready-fd the readiness byte must never touch descriptor 1.
	if !canCheckFD1 {
		return
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(1, &st); err != nil {
		t.Fatalf("proxyCmd closed descriptor 1: the default -ready-fd must stay negative (%v)", err)
	}
}

func TestExecCmdReadsEnvAndRemovesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := os.WriteFile(path, []byte("GITHUB_TOKEN=ghp_secret\x00PATH=/usr/bin\x00"), 0o600); err != nil {
		t.Fatal(err)
	}

	old := execProcess
	defer func() { execProcess = old }()
	var argv0 string
	var argv, envv []string
	execProcess = func(name string, args, env []string) error {
		argv0 = name
		argv = append([]string(nil), args...)
		envv = append([]string(nil), env...)
		return nil
	}

	if code := execCmd([]string{"--env-file", path, "--", "/bin/true", "arg"}); code != 0 {
		t.Fatalf("execCmd returned %d, want 0", code)
	}
	if argv0 != "/bin/true" || !reflect.DeepEqual(argv, []string{"/bin/true", "arg"}) {
		t.Fatalf("unexpected exec target: %q %v", argv0, argv)
	}
	joined := strings.Join(envv, "\n")
	if !strings.Contains(joined, "GITHUB_TOKEN=ghp_secret") {
		t.Fatalf("env file value missing from exec environment:\n%s", joined)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("env file was not removed before exec: %v", err)
	}
}

func TestExecCmdRejectsMissingInput(t *testing.T) {
	if code := execCmd([]string{"--", "true"}); code != 2 {
		t.Fatalf("execCmd without --env-file returned %d, want 2", code)
	}
	path := filepath.Join(t.TempDir(), "env")
	if code := execCmd([]string{"--env-file", path}); code != 2 {
		t.Fatalf("execCmd without a command returned %d, want 2", code)
	}
	if code := execCmd([]string{"--env-file", path, "--", "/bin/true"}); code != 1 {
		t.Fatalf("execCmd with a missing env file returned %d, want 1", code)
	}
}

func TestExecCmdRemovesFileWithReservedVariable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := os.WriteFile(path, []byte("HOME=/evil\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := execCmd([]string{"--env-file", path, "--", "/bin/true"}); code != 1 {
		t.Fatalf("execCmd with a reserved variable returned %d, want 1", code)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("env file was not removed after a read error: %v", err)
	}
}

func TestBridgeCmdReportsCommandProblems(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "proxy.sock")
	if code := bridgeCmd([]string{"-socket", socket, "--", "asb-command-that-does-not-exist"}); code != 1 {
		t.Fatalf("bridgeCmd with an unstartable command returned %d, want 1", code)
	}
	if code := bridgeCmd([]string{"-socket", socket}); code != 2 {
		t.Fatalf("bridgeCmd without a command returned %d, want 2", code)
	}
	if code := bridgeCmd([]string{"--", "true"}); code != 2 {
		t.Fatalf("bridgeCmd without --socket returned %d, want 2", code)
	}
}
