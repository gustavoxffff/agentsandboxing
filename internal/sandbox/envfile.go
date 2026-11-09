package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
)

// WriteEnvFile writes KEY=VALUE entries as NUL-separated records to path with
// mode 0600. The file is consumed by `asb exec`, which removes it before
// running the command, so values never reach the bwrap argv (RFC-0001 §5.4).
func WriteEnvFile(path string, env []string) error {
	if path == "" {
		return errors.New("env file path is empty")
	}
	entries := append([]string(nil), env...)
	sort.Strings(entries)

	var data strings.Builder
	for _, kv := range entries {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || !config.ValidEnvName(name) {
			return fmt.Errorf("invalid environment name: %q", name)
		}
		if config.ReservedEnv(name) {
			return fmt.Errorf("reserved environment variable: %s", name)
		}
		if strings.ContainsRune(value, 0) {
			return fmt.Errorf("environment variable %s contains NUL", name)
		}
		data.WriteString(kv)
		data.WriteByte(0)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create env file directory: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale env file: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create env file: %w", err)
	}
	if _, err := f.WriteString(data.String()); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write env file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close env file: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("adjust env file permissions: %w", err)
	}
	return nil
}

// ReadEnvFile reads a NUL-separated env file and rejects invalid or reserved
// names. It never includes a value in an error message.
func ReadEnvFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}
	parts := strings.Split(string(data), "\x00")
	out := make([]string, 0, len(parts))
	for _, entry := range parts {
		if entry == "" {
			continue
		}
		name, _, ok := strings.Cut(entry, "=")
		if !ok || !config.ValidEnvName(name) {
			return nil, errors.New("invalid entry in env file")
		}
		if config.ReservedEnv(name) {
			return nil, fmt.Errorf("reserved environment variable: %s", name)
		}
		out = append(out, entry)
	}
	return out, nil
}

// MergeEnv returns base with extra entries replacing variables of the same
// name. The result is sorted for deterministic exec environments.
func MergeEnv(base, extra []string) []string {
	values := make(map[string]string, len(base)+len(extra))
	for _, kv := range base {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || name == "" {
			continue
		}
		values[name] = value
	}
	for _, kv := range extra {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || name == "" {
			continue
		}
		values[name] = value
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, name+"="+values[name])
	}
	return out
}
