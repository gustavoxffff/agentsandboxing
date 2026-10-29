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
		t.Fatalf("esperava ErrNoProfile, veio %v", err)
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
		t.Fatalf("perfil = %q, esperava host", cfg.Profile)
	}
	if got := strings.Join(cfg.EnvAllow, ","); got != "LANG" {
		t.Fatalf("env.allow = %q, esperava LANG (substitui, não merge)", got)
	}
	if cfg.ProjectPath == "" || cfg.GlobalPath == "" {
		t.Fatalf("caminhos de config não preenchidos: %+v", cfg)
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
		t.Fatalf("perfil = %q", cfg.Profile)
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
		t.Fatalf("env.allow explícito vazio deveria ser preservado, veio %#v", cfg.EnvAllow)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"sem versão", "[network]\nprofile = \"host\"\n", "version = 1"},
		{"versão errada", "version = 2\n[network]\nprofile = \"host\"\n", "não suportada"},
		{"chave desconhecida", "version = 1\n[network]\nprofile = \"host\"\nfoo = 1\n", "desconhecidas"},
		{"perfil inválido", "version = 1\n[network]\nprofile = \"internet\"\n", "inválido"},
		{"allow sem allowlist", "version = 1\n[network]\nprofile = \"host\"\nallow = [\"github.com\"]\n", "só é válido"},
		{"ip na allowlist", "version = 1\n[network]\nprofile = \"allowlist\"\nallow = [\"1.1.1.1\"]\n", "IP direto"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			write(t, filepath.Join(project, "asb.toml"), tt.content)
			_, err := Resolve(project, Overrides{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("erro = %v, esperava conter %q", err, tt.want)
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
			t.Fatalf("%q => %q, esperava %q", in, got, want)
		}
	}
	for _, in := range []string{"", "1.1.1.1", "http://github.com", "github.com/path", "bad host", "*.", "a..b"} {
		if _, err := NormalizeHost(in); err == nil {
			t.Fatalf("%q deveria falhar", in)
		}
	}
}

func TestEnvFile(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git ausente")
	}
	project := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runGit(t, project, "init", "-q")
	write(t, filepath.Join(project, ".gitignore"), ".asb.env\n")
	envPath := filepath.Join(project, ".asb.env")
	writeMode(t, envPath, "FOO=bar\n# comentário\nBAZ=com espaço\n", 0o600)
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
	if cfg.FromFileVars["FOO"] != "bar" || cfg.FromFileVars["BAZ"] != "com espaço" {
		t.Fatalf("variáveis do from_file: %#v", cfg.FromFileVars)
	}

	if err := os.Chmod(envPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(project, Overrides{}); err == nil || !strings.Contains(err.Error(), "0600") {
		t.Fatalf("esperava erro de permissão 0600, veio %v", err)
	}

	if err := os.Chmod(envPath, 0o600); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(project, ".gitignore"), "outro\n")
	if _, err := Resolve(project, Overrides{}); err == nil || !strings.Contains(err.Error(), ".gitignore") {
		t.Fatalf("esperava erro de .gitignore, veio %v", err)
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
		t.Fatalf("env esperado não encontrado: %v", env)
	}
	if strings.Contains(joined, "HOME=") || strings.Contains(joined, "HTTP_PROXY=") {
		t.Fatalf("HOME/proxy não deveriam ser herdados: %v", env)
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
			t.Fatalf("%q deveria falhar", id)
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
