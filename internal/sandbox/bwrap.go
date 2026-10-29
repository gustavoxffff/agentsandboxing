package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
)

// Options descreve uma execução do bwrap.
type Options struct {
	ProjectDir string
	Profile    config.Profile
	Layout     Layout
	Env        []string // KEY=VALUE já filtrado (sem proxy; o bridge define no allowlist)
	Command    []string // comando e argumentos; o caminho é resolvido aqui
	SelfPath   string   // binário asb, montado read-only no allowlist
	ResolvDest string   // alvo real de /etc/resolv.conf no host
}

// BuildArgs monta os argumentos completos do bwrap, terminando no comando.
func BuildArgs(o Options) ([]string, error) {
	if len(o.Command) == 0 {
		return nil, fmt.Errorf("comando vazio")
	}
	if o.Layout.Home == "" {
		return nil, fmt.Errorf("layout sem HOME")
	}
	project, err := filepath.Abs(o.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("resolver projeto: %w", err)
	}

	args := []string{
		// Sistema base somente leitura.
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/etc", "/etc",
	}
	args = append(args, baseLinks()...)
	args = append(args,
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--tmpfs", "/run",

		// HOME efêmero.
		"--tmpfs", o.Layout.Home,

		// Projeto em leitura/escrita.
		"--bind", project, project,

		// Estado do OpenCode por sandbox (nunca o estado global do host).
		"--bind", o.Layout.HostRoot, o.Layout.InsideRoot,
	)

	// Git sem SSH: só o .gitconfig, se existir.
	if gitconfig := filepath.Join(o.Layout.Home, ".gitconfig"); fileExists(gitconfig) {
		args = append(args, "--ro-bind", gitconfig, gitconfig)
	}

	commandPath, err := resolveCommand(o.Command[0])
	if err != nil {
		return nil, err
	}
	command := append([]string(nil), o.Command...)
	command[0] = commandPath
	if mount, ok := extraCommandMount(commandPath, project, o.Layout); ok {
		args = append(args, "--ro-bind", mount, mount)
	}

	if o.Profile == config.ProfileAllowlist {
		if o.SelfPath == "" {
			return nil, fmt.Errorf("perfil allowlist exige o caminho do binário asb")
		}
		args = append(args,
			"--ro-bind", o.SelfPath, o.Layout.InsideBinary,
			"--ro-bind", o.Layout.ProxyDir, o.Layout.InsideProxyDir,
		)
		command = append([]string{
			o.Layout.InsideBinary, "bridge",
			"--socket", o.Layout.InsideProxySock,
			"--",
		}, command...)
	}

	// resolv.conf: monta sobre o alvo real do symlink (WSL e systemd-resolved).
	if o.ResolvDest != "" {
		if dir := filepath.Dir(o.ResolvDest); needsDir(dir, o.Layout.Home) {
			args = append(args, "--dir", dir)
		}
		args = append(args, "--file", "3", o.ResolvDest)
	}

	// Isolamento de processos/namespace.
	args = append(args,
		"--unshare-pid",
		"--unshare-ipc",
		"--unshare-uts",
		"--unshare-cgroup",
		"--die-with-parent",
		"--new-session",
		"--cap-drop", "ALL",
	)
	if o.Profile == config.ProfileNone || o.Profile == config.ProfileAllowlist {
		args = append(args, "--unshare-net")
	}

	// Ambiente fechado: nada do host entra sem estar na allowlist.
	args = append(args, "--clearenv")
	env := append([]string(nil), o.Env...)
	sort.Strings(env)
	for _, kv := range env {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("variável de ambiente inválida: %q", kv)
		}
		args = append(args, "--setenv", name, value)
	}

	args = append(args, "--chdir", project)
	args = append(args, command...)
	return args, nil
}

func baseLinks() []string {
	var out []string
	for _, path := range []string{"/bin", "/sbin", "/lib", "/lib64"} {
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err == nil {
				out = append(out, "--symlink", target, path)
			}
			continue
		}
		out = append(out, "--ro-bind", path, path)
	}
	return out
}

func resolveCommand(name string) (string, error) {
	if strings.ContainsRune(name, filepath.Separator) {
		abs, err := filepath.Abs(name)
		if err != nil {
			return "", fmt.Errorf("resolver comando: %w", err)
		}
		if !fileExists(abs) {
			return "", fmt.Errorf("comando não encontrado: %s", name)
		}
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		return abs, nil
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("comando não encontrado no host: %s", name)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolver comando: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return abs, nil
}

// extraCommandMount devolve o caminho que precisa ser remontado para que o
// comando exista dentro do sandbox. O HOME e /tmp são tmpfs, e o perfil base
// só monta /usr, /etc, /bin, /sbin, /lib e /lib64.
func extraCommandMount(commandPath, project string, layout Layout) (string, bool) {
	if underDir(commandPath, project) || underDir(commandPath, layout.HostRoot) {
		return "", false
	}
	for _, dir := range []string{"/usr", "/etc", "/bin", "/sbin", "/lib", "/lib64"} {
		if underDir(commandPath, dir) {
			return "", false
		}
	}
	for _, dir := range []string{"/nix/store", "/opt", "/snap"} {
		if underDir(commandPath, dir) {
			return dir, true
		}
	}
	return commandPath, true
}

func underDir(path, dir string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func needsDir(dir, home string) bool {
	clean := filepath.Clean(dir)
	switch clean {
	case "/", "/etc", "/run", "/tmp", "/proc", "/dev", home:
		return false
	}
	if strings.HasPrefix(clean, "/etc/") {
		return false
	}
	return true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ResolvDestination devolve o alvo real de /etc/resolv.conf no host.
func ResolvDestination() string {
	if resolved, err := filepath.EvalSymlinks("/etc/resolv.conf"); err == nil {
		return resolved
	}
	return "/etc/resolv.conf"
}
