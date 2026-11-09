package run

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecuteKeepsSecretsOutOfBwrapArgv is the regression test for RFC-0001
// §5.4: values from env.allow must never reach the bwrap argv.
func TestExecuteKeepsSecretsOutOfBwrapArgv(t *testing.T) {
	project := t.TempDir()
	cfg := "version = 1\n\n[network]\nprofile = \"none\"\n\n[env]\nallow = [\"PATH\", \"GITHUB_TOKEN\"]\n"
	if err := os.WriteFile(filepath.Join(project, "asb.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	bin := t.TempDir()
	argvFile := filepath.Join(t.TempDir(), "argv")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argvFile + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "bwrap"), []byte(script), 0o755); err != nil {
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
			switch name {
			case "PATH":
				return "/usr/bin", true
			case "GITHUB_TOKEN":
				return "ghp_secret_value", true
			}
			return "", false
		},
	})
	if err != nil || code != 0 {
		t.Fatalf("Execute = (%d, %v)\nstderr:\n%s", code, err, stderr.String())
	}

	data, err := os.ReadFile(argvFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "ghp_secret_value") {
		t.Fatalf("secret value reached the bwrap argv:\n%s", data)
	}
	if strings.Contains(string(data), "GITHUB_TOKEN") {
		t.Fatalf("secret variable name reached the bwrap argv:\n%s", data)
	}
}
