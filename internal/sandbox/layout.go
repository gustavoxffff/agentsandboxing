// Package sandbox calcula o layout por sandbox e monta os argumentos do bwrap.
package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Layout descreve os caminhos de estado e de runtime de um sandbox.
//
// O estado persistente fica no host em
// $XDG_STATE_HOME/asb/sandboxes/<id>/opencode/{config,data,state,cache}.
// Dentro do sandbox o mesmo diretório é montado em
// $HOME/.local/state/asb/sandboxes/<id>, e as variáveis XDG apontam para os
// subdiretórios. Nada do estado global do OpenCode do host é montado.
type Layout struct {
	ID string

	Home string

	// Host
	HostRoot         string
	HostOpencodeRoot string
	ProxyDir         string
	ProxySocket      string

	// Sandbox (caminhos absolutos dentro do bwrap)
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

// NewLayout calcula o layout. sandboxID vazio usa o id derivado do projeto.
func NewLayout(projectDir, sandboxID string) (Layout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Layout{}, fmt.Errorf("descobrir HOME: %w", err)
	}
	home = filepath.Clean(home)
	if !filepath.IsAbs(home) || home == "/" || home == "." {
		return Layout{}, fmt.Errorf("HOME inválido: %q", home)
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
		// sun_path do Linux tem ~108 bytes; cai para /tmp se o XDG for longo.
		proxyDir = filepath.Join(os.TempDir(), fmt.Sprintf("asb-%d", os.Getuid()), shortHash(id))
		proxySocket = filepath.Join(proxyDir, "proxy.sock")
	}
	if len(proxySocket) > 100 {
		return Layout{}, fmt.Errorf("caminho do socket do proxy longo demais: %s", proxySocket)
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

// DefaultID devolve <basename-do-projeto>-<sha256(caminho)[0:8]>.
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

// CreateDirs cria o estado persistente e o diretório privado do socket.
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
			return fmt.Errorf("criar estado do sandbox: %w", err)
		}
		// O estado pode conter credenciais; não confie no umask ou em modo
		// preexistente.
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("ajustar permissões do estado: %w", err)
		}
	}
	if err := os.MkdirAll(l.ProxyDir, 0o700); err != nil {
		return fmt.Errorf("criar diretório do proxy: %w", err)
	}
	// Garante 0700 mesmo se o diretório já existia com outro modo.
	if err := os.Chmod(l.ProxyDir, 0o700); err != nil {
		return fmt.Errorf("ajustar permissões do proxy: %w", err)
	}
	return nil
}

// RemoveStaleSocket remove um socket de uma execução anterior.
func (l Layout) RemoveStaleSocket() error {
	if err := os.Remove(l.ProxySocket); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remover socket antigo: %w", err)
	}
	return nil
}
