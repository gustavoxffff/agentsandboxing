package run

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
	"github.com/gustavoxffff/agentsandboxing/internal/sandbox"
)

func TestBuildSandboxEnvKeepsHostValuesOut(t *testing.T) {
	cfg := config.Resolved{
		EnvAllow: []string{"PATH", "HOME", "HTTP_PROXY", "GITHUB_TOKEN"},
		FromFileVars: map[string]string{
			"API_KEY": "secret-value",
		},
	}
	env, err := buildSandboxEnv(cfg, func(name string) (string, bool) {
		values := map[string]string{
			"PATH": "/usr/bin", "HOME": "/host/home",
			"HTTP_PROXY": "http://evil", "GITHUB_TOKEN": "ghp_x",
		}
		v, ok := values[name]
		return v, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(env, "\n")
	for _, want := range []string{"PATH=/usr/bin", "GITHUB_TOKEN=ghp_x", "API_KEY=secret-value"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	for _, forbidden := range []string{"HOME=", "HTTP_PROXY=", "evil"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("host value %q leaked into the env file:\n%s", forbidden, joined)
		}
	}
}

func TestBuildToolEnvSetsHomeAndXDG(t *testing.T) {
	layout := sandbox.Layout{
		Home:             "/home/user",
		InsideConfigHome: "/home/user/.local/state/asb/sandboxes/x/opencode/config",
		InsideDataHome:   "/home/user/.local/state/asb/sandboxes/x/opencode/data",
		InsideStateHome:  "/home/user/.local/state/asb/sandboxes/x/opencode/state",
		InsideCacheHome:  "/home/user/.local/state/asb/sandboxes/x/opencode/cache",
		InsideRuntimeDir: "/run/asb/runtime",
	}
	joined := strings.Join(buildToolEnv(layout), "\n")
	for _, want := range []string{
		"HOME=/home/user",
		"TMPDIR=/tmp",
		"XDG_CONFIG_HOME=/home/user/.local/state/asb/sandboxes/x/opencode/config",
		"XDG_RUNTIME_DIR=/run/asb/runtime",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	for _, forbidden := range []string{"PATH=", "GITHUB_TOKEN"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("non-tool value %q in --setenv env:\n%s", forbidden, joined)
		}
	}
}

func TestPrintConfigDoesNotLeakValues(t *testing.T) {
	cfg := config.Resolved{
		ProjectDir:   "/proj",
		Profile:      config.ProfileAllowlist,
		NetworkAllow: []string{"github.com"},
		EnvAllow:     []string{"PATH"},
		EnvFromFile:  ".asb.env",
		FromFileVars: map[string]string{"GITHUB_TOKEN": "ghp_supersecret"},
	}
	layout := sandbox.Layout{
		ID:               "proj-1",
		HostOpencodeRoot: "/state/opencode",
		InsideRoot:       "/home/user/.local/state/asb/sandboxes/proj-1",
		ProxySocket:      "/run/asb/proxy.sock",
	}
	var out bytes.Buffer
	PrintConfig(&out, cfg, layout, []string{"opencode", "--standalone"})
	text := out.String()
	if strings.Contains(text, "ghp_supersecret") {
		t.Fatalf("secret value printed:\n%s", text)
	}
	for _, want := range []string{"network profile: allowlist", "github.com", "GITHUB_TOKEN", "opencode --standalone"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

// TestExecuteDoesNotWriteGitconfigWhenNoMatchingKeys verifies that when
// [git] mode is not off but no keys match the host .gitconfig, no .gitconfig
// file is written to the sandbox state dir. The host .gitconfig is never
// mounted; the generated file exists only when there are keys to copy).
func TestExecuteDoesNotWriteGitconfigWhenNoMatchingKeys(t *testing.T) {
	project := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	// Host .gitconfig has keys that don't match minimal mode.
	home := os.Getenv("HOME")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[core]\n\teditor = vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := "version = 1\n\n[network]\nprofile = \"none\"\n\n[git]\nmode = \"minimal\"\n"
	if err := os.WriteFile(filepath.Join(project, "asb.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "bwrap"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout, stderr bytes.Buffer
	code, err := Execute(Options{
		ProjectDir: project,
		Command:    []string{"true"},
		Stdin:      strings.NewReader(""),
		Stdout:     &stdout,
		Stderr:     &stderr,
		SelfPath:   "/usr/bin/asb",
		LookupEnv: func(name string) (string, bool) {
			if name == "PATH" {
				return "/usr/bin", true
			}
			return "", false
		},
	})
	if err != nil || code != 0 {
		t.Fatalf("Execute = (%d, %v)\nstderr:\n%s", code, err, stderr.String())
	}

	// Resolve the layout to find HostGitConfig and verify no file was written.
	resolved, err := config.Resolve(project, config.Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	layout, err := sandbox.NewLayout(resolved.ProjectDir, resolved.SandboxID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(layout.HostGitConfig); !os.IsNotExist(err) {
		t.Fatalf(".gitconfig was written despite no matching keys: %v", err)
	}
}

// TestPrintConfigBranches asserts every conditional branch of PrintConfig so
// that negating any of them changes the output (mutation gate).
func TestPrintConfigBranches(t *testing.T) {
	layout := sandbox.Layout{
		ID:               "proj-1",
		HostOpencodeRoot: "/state/opencode",
		InsideRoot:       "/home/user/.local/state/asb/sandboxes/proj-1",
		ProxySocket:      "/run/asb/proxy.sock",
	}

	// Set global/project paths and an allowlist profile: the positive branches
	// must print the values and the proxy lines.
	full := config.Resolved{
		ProjectDir:  "/proj",
		GlobalPath:  "/home/user/.config/asb/config.toml",
		ProjectPath: "/proj/asb.toml",
		Profile:     config.ProfileAllowlist,
	}
	var out bytes.Buffer
	PrintConfig(&out, full, layout, []string{"opencode"})
	text := out.String()
	for _, want := range []string{
		"global config: /home/user/.config/asb/config.toml",
		"project config: /proj/asb.toml",
		"proxy (socket): /run/asb/proxy.sock",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}

	// Empty paths and a non-allowlist profile: the negative branches must
	// print "(none)" and must not print the proxy lines.
	empty := config.Resolved{
		ProjectDir: "/proj",
		Profile:    config.ProfileHost,
	}
	out.Reset()
	PrintConfig(&out, empty, layout, []string{"opencode"})
	text = out.String()
	for _, want := range []string{"global config: (none)", "project config: (none)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "proxy (socket)") {
		t.Fatalf("proxy lines printed for a non-allowlist profile:\n%s", text)
	}
}
