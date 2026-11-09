package sandbox

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWriteEnvFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := WriteEnvFile(path, []string{"GITHUB_TOKEN=ghp_secret", "PATH=/usr/bin"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("env file mode = %04o, want 0600", info.Mode().Perm())
	}
	got, err := ReadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"GITHUB_TOKEN=ghp_secret", "PATH=/usr/bin"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadEnvFile = %v, want %v", got, want)
	}
}

func TestWriteEnvFileReportsStaleRemovalFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "child"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := WriteEnvFile(path, []string{"PATH=/usr/bin"})
	if err == nil || !strings.Contains(err.Error(), "remove stale env file") {
		t.Fatalf("WriteEnvFile = %v, want a stale-removal error", err)
	}
}

func TestWriteEnvFileRejectsReservedVariable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := WriteEnvFile(path, []string{"HOME=/evil"}); err == nil {
		t.Fatal("WriteEnvFile accepted HOME")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("env file was created for a rejected entry: %v", err)
	}
}

func TestWriteEnvFileRejectsInvalidEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := WriteEnvFile(path, []string{"1BAD=value"}); err == nil {
		t.Fatal("WriteEnvFile accepted an invalid name")
	}
	if err := WriteEnvFile(path, []string{"NOVALUE"}); err == nil {
		t.Fatal("WriteEnvFile accepted an entry without =")
	}
	if err := WriteEnvFile("", []string{"PATH=/usr/bin"}); err == nil {
		t.Fatal("WriteEnvFile accepted an empty path")
	}
}

func TestReadEnvFileRejectsReservedVariable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := os.WriteFile(path, []byte("HOME=/evil\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadEnvFile(path); err == nil {
		t.Fatal("ReadEnvFile accepted HOME")
	}
}

func TestReadEnvFileRejectsInvalidEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	if err := os.WriteFile(path, []byte("BAD NAME=value\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadEnvFile(path); err == nil {
		t.Fatal("ReadEnvFile accepted an invalid name")
	}
}

func TestReadEnvFileMissing(t *testing.T) {
	if _, err := ReadEnvFile(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("ReadEnvFile accepted a missing file")
	}
}

func TestMergeEnvExtraWinsAndSorts(t *testing.T) {
	got := MergeEnv([]string{"B=1", "A=1"}, []string{"A=2", "C=3"})
	want := []string{"A=2", "B=1", "C=3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeEnv = %v, want %v", got, want)
	}
	if strings.Contains(strings.Join(got, "\n"), "\x00") {
		t.Fatal("MergeEnv produced NUL bytes")
	}
}
