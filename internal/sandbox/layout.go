// Package sandbox computes the per-sandbox layout and builds the bwrap arguments.
package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Layout describes the state and runtime paths of a sandbox.
//
// Persistent state lives on the host at
// $XDG_STATE_HOME/asb/sandboxes/<id>/opencode/{config,data,state,cache}.
// Inside the sandbox the same directory is mounted at
// $HOME/.local/state/asb/sandboxes/<id>, and the XDG variables point to the
// subdirectories. Nothing from the host global OpenCode state is mounted.
type Layout struct {
	ID string

	Home string

	// Host
	HostRoot         string
	HostOpencodeRoot string
	ProxyDir         string
	ProxySocket      string

	// Sandbox (absolute paths inside bwrap)
	InsideRoot       string
	InsideConfigHome string
	InsideDataHome   string
	InsideStateHome  string
	InsideCacheHome  string
	InsideRuntimeDir string
	InsideProxyDir   string
	InsideProxySock  string
	InsideBinary     string
}

// NewLayout computes the layout. An empty sandboxID uses the id derived from
// the project.
func NewLayout(projectDir, sandboxID string) (Layout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Layout{}, fmt.Errorf("discover HOME: %w", err)
	}
	home = filepath.Clean(home)
	if !filepath.IsAbs(home) || home == "/" || home == "." {
		return Layout{}, fmt.Errorf("invalid HOME: %q", home)
	}

	id := sandboxID
	if id == "" {
		id = DefaultID(projectDir)
	}

	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	hostRoot := filepath.Join(stateHome, "asb", "sandboxes", id)
	hostOpencode := filepath.Join(hostRoot, "opencode")

	insideRoot := filepath.Join(home, ".local", "state", "asb", "sandboxes", id)
	insideOpencode := filepath.Join(insideRoot, "opencode")

	runtimeBase := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeBase == "" {
		runtimeBase = os.TempDir()
	}
	proxyDir := filepath.Join(runtimeBase, "asb", shortHash(id))
	proxySocket := filepath.Join(proxyDir, "proxy.sock")
	if len(proxySocket) > 100 {
		// Linux sun_path is ~108 bytes; fall back to /tmp if XDG is long.
		proxyDir = filepath.Join(os.TempDir(), fmt.Sprintf("asb-%d", os.Getuid()), shortHash(id))
		proxySocket = filepath.Join(proxyDir, "proxy.sock")
	}
	if len(proxySocket) > 100 {
		return Layout{}, fmt.Errorf("proxy socket path too long: %s", proxySocket)
	}

	return Layout{
		ID:               id,
		Home:             home,
		HostRoot:         hostRoot,
		HostOpencodeRoot: hostOpencode,
		ProxyDir:         proxyDir,
		ProxySocket:      proxySocket,
		InsideRoot:       insideRoot,
		InsideConfigHome: filepath.Join(insideOpencode, "config"),
		InsideDataHome:   filepath.Join(insideOpencode, "data"),
		InsideStateHome:  filepath.Join(insideOpencode, "state"),
		InsideCacheHome:  filepath.Join(insideOpencode, "cache"),
		InsideRuntimeDir: "/run/asb/runtime",
		InsideProxyDir:   "/run/asb/proxy",
		InsideProxySock:  "/run/asb/proxy/proxy.sock",
		InsideBinary:     "/run/asb/bin/asb",
	}, nil
}

// DefaultID returns <project-basename>-<sha256(path)[0:8]>.
func DefaultID(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		abs = projectDir
	}
	base := sanitize(filepath.Base(filepath.Clean(abs)))
	sum := sha256.Sum256([]byte(abs))
	return base + "-" + hex.EncodeToString(sum[:])[:8]
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-.")
	if out == "" {
		out = "sandbox"
	}
	if len(out) > 48 {
		out = out[:48]
	}
	return out
}

// CreateDirs creates the persistent state and the private socket directory.
func (l Layout) CreateDirs() error {
	for _, dir := range []string{
		l.HostRoot,
		l.HostOpencodeRoot,
		filepath.Join(l.HostOpencodeRoot, "config"),
		filepath.Join(l.HostOpencodeRoot, "data"),
		filepath.Join(l.HostOpencodeRoot, "state"),
		filepath.Join(l.HostOpencodeRoot, "cache"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create sandbox state: %w", err)
		}
		// The state may contain credentials; do not trust umask or a
		// preexisting mode.
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("adjust state permissions: %w", err)
		}
	}
	if err := os.MkdirAll(l.ProxyDir, 0o700); err != nil {
		return fmt.Errorf("create proxy directory: %w", err)
	}
	// Ensures 0700 even if the directory already existed with another mode.
	if err := os.Chmod(l.ProxyDir, 0o700); err != nil {
		return fmt.Errorf("adjust proxy permissions: %w", err)
	}
	return nil
}

// RemoveStaleSocket removes a socket from a previous run.
func (l Layout) RemoveStaleSocket() error {
	if err := os.Remove(l.ProxySocket); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	return nil
}
