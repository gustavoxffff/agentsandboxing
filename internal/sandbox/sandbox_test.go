package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
)

func testLayout(t *testing.T) Layout {
	t.Helper()
	home := t.TempDir()
	root := filepath.Join(home, ".local", "state", "asb", "sandboxes", "proj-12345678")
	opencode := filepath.Join(root, "opencode")
	return Layout{
		ID:               "proj-12345678",
		Home:             home,
		HostRoot:         root,
		HostOpencodeRoot: opencode,
		ProxyDir:         filepath.Join(home, "runtime"),
		ProxySocket:      filepath.Join(home, "runtime", "proxy.sock"),
		InsideRoot:       root,
		InsideConfigHome: filepath.Join(opencode, "config"),
		InsideDataHome:   filepath.Join(opencode, "data"),
		InsideStateHome:  filepath.Join(opencode, "state"),
		InsideCacheHome:  filepath.Join(opencode, "cache"),
		InsideRuntimeDir: "/run/asb/runtime",
		InsideProxyDir:   "/run/asb/proxy",
		InsideProxySock:  "/run/asb/proxy/proxy.sock",
		InsideBinary:     "/run/asb/bin/asb",
	}
}

func build(t *testing.T, profile config.Profile, command ...string) (Layout, []string) {
	t.Helper()
	layout := testLayout(t)
	if command == nil {
		command = []string{"true"}
	}
	args, err := BuildArgs(Options{
		ProjectDir: t.TempDir(),
		Profile:    profile,
		Layout:     layout,
		Env: []string{
			"HOME=" + layout.Home,
			"PATH=/usr/bin",
			"XDG_CONFIG_HOME=" + layout.InsideConfigHome,
		},
		Command:    command,
		SelfPath:   "/usr/bin/asb",
		ResolvDest: "/etc/resolv.conf",
	})
	if err != nil {
		t.Fatal(err)
	}
	return layout, args
}

func contains(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func containsPair(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func containsSetenv(args []string, name, value string) bool {
	for i := 0; i+2 < len(args); i++ {
		if args[i] == "--setenv" && args[i+1] == name && args[i+2] == value {
			return true
		}
	}
	return false
}

func TestBuildArgsNone(t *testing.T) {
	layout, args := build(t, config.ProfileNone)
	for _, want := range []string{"--unshare-net", "--clearenv", "--unshare-pid", "--die-with-parent", "--new-session"} {
		if !contains(args, want) {
			t.Fatalf("faltou %s em %v", want, args)
		}
	}
	if !containsSetenv(args, "XDG_CONFIG_HOME", layout.InsideConfigHome) {
		t.Fatalf("XDG_CONFIG_HOME não aponta para o sandbox: %v", args)
	}
	if contains(args, "bridge") {
		t.Fatalf("perfil none não deveria usar bridge: %v", args)
	}
}

func TestBuildArgsHost(t *testing.T) {
	_, args := build(t, config.ProfileHost)
	if contains(args, "--unshare-net") {
		t.Fatalf("perfil host não deveria isolar rede: %v", args)
	}
	if !contains(args, "--clearenv") {
		t.Fatalf("perfil host deveria limpar ambiente: %v", args)
	}
}

func TestBuildArgsAllowlist(t *testing.T) {
	layout, args := build(t, config.ProfileAllowlist)
	for _, want := range []string{"--unshare-net", "bridge", "--socket", "/run/asb/proxy/proxy.sock"} {
		if !contains(args, want) {
			t.Fatalf("faltou %s em %v", want, args)
		}
	}
	if !containsPair(args, "--ro-bind", "/usr/bin/asb") {
		t.Fatalf("binário asb não montado read-only: %v", args)
	}
	if !containsPair(args, "--ro-bind", layout.ProxyDir) {
		t.Fatalf("diretório do proxy não montado read-only: %v", args)
	}
}

func TestBuildArgsNeverMountsGlobalOpencodeState(t *testing.T) {
	_, args := build(t, config.ProfileHost)
	home, _ := os.UserHomeDir()
	forbidden := []string{
		filepath.Join(home, ".config", "opencode"),
		filepath.Join(home, ".local", "share", "opencode"),
		filepath.Join(home, ".local", "state", "opencode"),
		filepath.Join(home, ".cache", "opencode"),
		filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
	}
	joined := strings.Join(args, "\n")
	for _, path := range forbidden {
		if strings.Contains(joined, path) {
			t.Fatalf("estado global montado: %s\n%s", path, joined)
		}
	}
}

func TestDefaultID(t *testing.T) {
	a := DefaultID("/tmp/proj")
	b := DefaultID("/tmp/proj")
	if a != b {
		t.Fatalf("id não determinístico: %s != %s", a, b)
	}
	if !strings.HasPrefix(a, "proj-") || len(a) != len("proj-")+8 {
		t.Fatalf("id inesperado: %s", a)
	}
	if got := DefaultID("/tmp/meu projeto!"); !strings.HasPrefix(got, "meu-projeto-") {
		t.Fatalf("sanitização inesperada: %s", got)
	}
}

func TestNewLayoutUsesXDGStateHome(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	project := t.TempDir()
	layout, err := NewLayout(project, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(state, "asb", "sandboxes", layout.ID)
	if layout.HostRoot != want {
		t.Fatalf("HostRoot = %s, esperava %s", layout.HostRoot, want)
	}
	if !strings.HasPrefix(layout.InsideRoot, layout.Home) {
		t.Fatalf("InsideRoot fora do HOME: %s", layout.InsideRoot)
	}
	if len(layout.ProxySocket) > 100 {
		t.Fatalf("socket longo demais: %s", layout.ProxySocket)
	}
}

func TestCreateDirs(t *testing.T) {
	layout := testLayout(t)
	if err := layout.CreateDirs(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(layout.ProxyDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("permissão do diretório do proxy = %04o", info.Mode().Perm())
	}
	for _, dir := range []string{"config", "data", "state", "cache"} {
		if info, err := os.Stat(filepath.Join(layout.HostOpencodeRoot, dir)); err != nil || !info.IsDir() {
			t.Fatalf("diretório %s não criado: %v", dir, err)
		}
	}
}

func TestResolvContent(t *testing.T) {
	if got := ResolvContent(config.ProfileNone); !strings.Contains(got, "nameserver 127.0.0.1") {
		t.Fatalf("none: %q", got)
	}
	if got := ResolvContent(config.ProfileAllowlist); !strings.Contains(got, "nameserver 127.0.0.1") {
		t.Fatalf("allowlist: %q", got)
	}
	if got := ResolvContent(config.ProfileHost); !strings.Contains(got, "nameserver") {
		t.Fatalf("host: %q", got)
	}
}
