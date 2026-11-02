package config

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveNoProfile(t *testing.T) {
	project := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, err := Resolve(project, Overrides{})
	if !errors.Is(err, ErrNoProfile) {
		t.Fatalf("expected ErrNoProfile, got %v", err)
	}
}

func TestPrecedenceAndListsReplace(t *testing.T) {
	project := t.TempDir()
	globalDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", globalDir)

	write(t, filepath.Join(globalDir, "asb", "config.toml"), `
version = 1
[network]
profile = "none"
[env]
allow = ["PATH", "TERM"]
`)
	write(t, filepath.Join(project, "asb.toml"), `
version = 1
[network]
profile = "host"
[env]
allow = ["LANG"]
`)

	cfg, err := Resolve(project, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != ProfileHost {
		t.Fatalf("profile = %q, expected host", cfg.Profile)
	}
	if got := strings.Join(cfg.EnvAllow, ","); got != "LANG" {
		t.Fatalf("env.allow = %q, expected LANG (replace, not merge)", got)
	}
	if cfg.ProjectPath == "" || cfg.GlobalPath == "" {
		t.Fatalf("config paths not populated: %+v", cfg)
	}
}

func TestCLIOverrides(t *testing.T) {
	project := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(project, "asb.toml"), `
version = 1
[network]
profile = "host"
`)
	profile := "none"
	env := []string{"SHELL"}
	id := "custom-1"
	cfg, err := Resolve(project, Overrides{Profile: &profile, EnvAllow: &env, SandboxID: &id})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != ProfileNone {
		t.Fatalf("profile = %q", cfg.Profile)
	}
	if got := strings.Join(cfg.EnvAllow, ","); got != "SHELL" {
		t.Fatalf("env.allow = %q", got)
	}
	if cfg.SandboxID != "custom-1" {
		t.Fatalf("sandbox id = %q", cfg.SandboxID)
	}
}

func TestExplicitEmptyEnvAllowIsPreserved(t *testing.T) {
	project := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(project, "asb.toml"), `
version = 1
[network]
profile = "none"
[env]
allow = []
`)
	cfg, err := Resolve(project, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EnvAllow == nil || len(cfg.EnvAllow) != 0 {
		t.Fatalf("explicit empty env.allow should be preserved, got %#v", cfg.EnvAllow)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"missing version", "[network]\nprofile = \"host\"\n", "version = 1"},
		{"wrong version", "version = 2\n[network]\nprofile = \"host\"\n", "unsupported"},
		{"unknown key", "version = 1\n[network]\nprofile = \"host\"\nfoo = 1\n", "unknown keys"},
		{"invalid profile", "version = 1\n[network]\nprofile = \"internet\"\n", "invalid"},
		{"allow without allowlist", "version = 1\n[network]\nprofile = \"host\"\nallow = [\"github.com\"]\n", "only valid"},
		{"ip in allowlist", "version = 1\n[network]\nprofile = \"allowlist\"\nallow = [\"1.1.1.1\"]\n", "direct IP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			write(t, filepath.Join(project, "asb.toml"), tt.content)
			_, err := Resolve(project, Overrides{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, expected to contain %q", err, tt.want)
			}
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	valid := map[string]string{
		"github.com":           "github.com",
		"GitHub.COM.":          "github.com",
		"*.example.com":        "*.example.com",
		" registry.npmjs.org ": "registry.npmjs.org",
	}
	for in, want := range valid {
		got, err := NormalizeHost(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != want {
			t.Fatalf("%q => %q, expected %q", in, got, want)
		}
	}
	for _, in := range []string{"", "1.1.1.1", "http://github.com", "github.com/path", "bad host", "*.", "a..b"} {
		if _, err := NormalizeHost(in); err == nil {
			t.Fatalf("%q should fail", in)
		}
	}
}

func TestEnvFile(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	project := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runGit(t, project, "init", "-q")
	write(t, filepath.Join(project, ".gitignore"), ".asb.env\n")
	envPath := filepath.Join(project, ".asb.env")
	writeMode(t, envPath, "FOO=bar\n# comment\nBAZ=with space\n", 0o600)
	write(t, filepath.Join(project, "asb.toml"), `
version = 1
[network]
profile = "none"
[env]
allow = ["PATH"]
from_file = ".asb.env"
`)

	cfg, err := Resolve(project, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FromFileVars["FOO"] != "bar" || cfg.FromFileVars["BAZ"] != "with space" {
		t.Fatalf("from_file variables: %#v", cfg.FromFileVars)
	}

	if err := os.Chmod(envPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(project, Overrides{}); err == nil || !strings.Contains(err.Error(), "0600") {
		t.Fatalf("expected permission 0600 error, got %v", err)
	}

	if err := os.Chmod(envPath, 0o600); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(project, ".gitignore"), "other\n")
	if _, err := Resolve(project, Overrides{}); err == nil || !strings.Contains(err.Error(), ".gitignore") {
		t.Fatalf("expected .gitignore error, got %v", err)
	}
}

func TestBuildEnvExcludesReserved(t *testing.T) {
	project := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(project, "asb.toml"), `
version = 1
[network]
profile = "host"
[env]
allow = ["PATH", "HOME", "HTTP_PROXY", "GITHUB_TOKEN"]
`)
	cfg, err := Resolve(project, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	env := cfg.BuildEnv(func(name string) (string, bool) {
		values := map[string]string{
			"PATH": "/usr/bin", "HOME": "/home/user",
			"HTTP_PROXY": "http://evil", "GITHUB_TOKEN": "secret",
		}
		v, ok := values[name]
		return v, ok
	})
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "PATH=/usr/bin") || !strings.Contains(joined, "GITHUB_TOKEN=secret") {
		t.Fatalf("expected env entry not found: %v", env)
	}
	if strings.Contains(joined, "HOME=") || strings.Contains(joined, "HTTP_PROXY=") {
		t.Fatalf("HOME/proxy should not be inherited: %v", env)
	}
}

func TestValidateSandboxID(t *testing.T) {
	for _, id := range []string{"ok", "repo-abc12345", "a.b_c-d"} {
		if err := ValidateSandboxID(id); err != nil {
			t.Fatalf("%q: %v", id, err)
		}
	}
	for _, id := range []string{"", ".", "..", "../x", "a/b", ".hidden", "a b"} {
		if err := ValidateSandboxID(id); err == nil {
			t.Fatalf("%q should fail", id)
		}
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMode(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}
