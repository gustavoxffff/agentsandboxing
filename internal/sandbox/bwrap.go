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

// Options describes a bwrap execution.
type Options struct {
	ProjectDir string
	Profile    config.Profile
	Layout     Layout
	Env        []string // tool-defined values only (HOME, XDG, TMPDIR); never secrets
	EnvFile    string   // env file path inside the sandbox, read by `asb exec`
	Command    []string // command and arguments; the path is resolved here
	SelfPath   string   // asb binary, mounted read-only so `asb exec` can run
	ResolvDest string   // real target of /etc/resolv.conf on the host
}

// toolEnvNames is the complete set of variables the tool may pass through
// bwrap --setenv. Proxy variables are added later by the bridge.
var toolEnvNames = map[string]bool{
	"HOME":            true,
	"TMPDIR":          true,
	"XDG_CONFIG_HOME": true,
	"XDG_DATA_HOME":   true,
	"XDG_STATE_HOME":  true,
	"XDG_CACHE_HOME":  true,
	"XDG_RUNTIME_DIR": true,
}

// BuildArgs builds the complete bwrap arguments, ending in the command.
func BuildArgs(o Options) ([]string, error) {
	if len(o.Command) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	if o.Layout.Home == "" {
		return nil, fmt.Errorf("layout without HOME")
	}
	if o.EnvFile == "" {
		return nil, fmt.Errorf("missing env file path")
	}
	if o.SelfPath == "" {
		return nil, fmt.Errorf("missing asb binary path")
	}
	project, err := filepath.Abs(o.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("resolve project: %w", err)
	}

	args := []string{
		// Read-only base system.
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/etc", "/etc",
	}
	args = append(args, baseLinks()...)
	args = append(args,
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--tmpfs", "/run",

		// Ephemeral HOME.
		"--tmpfs", o.Layout.Home,

		// Project read/write.
		"--bind", project, project,

		// Per-sandbox OpenCode state (never the host global state).
		"--bind", o.Layout.HostRoot, o.Layout.InsideRoot,
	)

	// Git without SSH: only .gitconfig, if it exists.
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

	// `asb exec` reads the env file and removes it before the command runs.
	args = append(args, "--ro-bind", o.SelfPath, o.Layout.InsideBinary)
	if o.Profile == config.ProfileAllowlist {
		args = append(args, "--ro-bind", o.Layout.ProxyDir, o.Layout.InsideProxyDir)
		command = append([]string{
			o.Layout.InsideBinary, "bridge",
			"--socket", o.Layout.InsideProxySock,
			"--",
		}, command...)
	}
	command = append([]string{
		o.Layout.InsideBinary, "exec",
		"--env-file", o.EnvFile,
		"--",
	}, command...)

	// resolv.conf: mount over the real target of the symlink (WSL and systemd-resolved).
	if o.ResolvDest != "" {
		if dir := filepath.Dir(o.ResolvDest); needsDir(dir, o.Layout.Home) {
			args = append(args, "--dir", dir)
		}
		args = append(args, "--file", "3", o.ResolvDest)
	}

	// Process/namespace isolation.
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

	// Closed environment: --setenv carries only the variables the tool owns.
	// Values from env.allow/from_file travel in the env file (RFC-0001 §5.4).
	args = append(args, "--clearenv")
	env := append([]string(nil), o.Env...)
	sort.Strings(env)
	for _, kv := range env {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid environment variable: %q", kv)
		}
		if !toolEnvNames[name] {
			return nil, fmt.Errorf("--setenv is restricted to tool variables: %s", name)
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
			return "", fmt.Errorf("resolve command: %w", err)
		}
		if !fileExists(abs) {
			return "", fmt.Errorf("command not found: %s", name)
		}
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		return abs, nil
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("command not found on the host: %s", name)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve command: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return abs, nil
}

// extraCommandMount returns the path that must be remounted so that the
// command exists inside the sandbox. HOME and /tmp are tmpfs, and the base
// profile only mounts /usr, /etc, /bin, /sbin, /lib and /lib64.
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

// ResolvDestination returns the real target of /etc/resolv.conf on the host.
func ResolvDestination() string {
	if resolved, err := filepath.EvalSymlinks("/etc/resolv.conf"); err == nil {
		return resolved
	}
	return "/etc/resolv.conf"
}
