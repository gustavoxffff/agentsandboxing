// Package config loads and resolves the asb declarative configuration.
//
// Precedence: CLI flag > project asb.toml > global config > no default.
// Lists replace, they do not merge.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Profile is the v1 network profile.
type Profile string

const (
	ProfileHost      Profile = "host"
	ProfileNone      Profile = "none"
	ProfileAllowlist Profile = "allowlist"
)

// BaseEnvAllow is the closed base allowlist of v1 (RFC-0001 §5.4). HOME and
// the proxy variables are set by the tool, not inherited.
var BaseEnvAllow = []string{
	"PATH", "TERM", "LANG", "LC_ALL", "TZ", "USER", "LOGNAME", "SHELL",
}

// reservedEnv is never inherited from the host, even when declared in [env]
// allow or in from_file: HOME and the proxy channel are set by the tool.
var reservedEnv = map[string]bool{
	"HOME": true, "TMPDIR": true,
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "ALL_PROXY": true, "NO_PROXY": true,
	"http_proxy": true, "https_proxy": true, "all_proxy": true, "no_proxy": true,
	"NODE_USE_ENV_PROXY": true,
}

var envNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidEnvName reports whether name is a valid environment variable name.
func ValidEnvName(name string) bool { return envNameRE.MatchString(name) }

// ReservedEnv reports whether name is set by the tool and must never come from
// the host environment or the env file.
func ReservedEnv(name string) bool { return reservedEnv[name] }

// ErrNoProfile indicates that no profile was chosen at any level.
var ErrNoProfile = errors.New("no network profile configured")

// Layer is a configuration layer that has already been validated.
type Layer struct {
	Path         string
	Profile      *string
	NetworkAllow *[]string
	EnvAllow     *[]string
	EnvFromFile  *string
}

// Overrides are the CLI flags. Nil means "not set".
type Overrides struct {
	Profile      *string
	NetworkAllow *[]string
	EnvAllow     *[]string
	SandboxID    *string
}

// Resolved is the effective configuration after precedence.
type Resolved struct {
	ProjectDir   string
	GlobalPath   string
	ProjectPath  string
	Profile      Profile
	NetworkAllow []string
	EnvAllow     []string
	EnvFromFile  string
	FromFileVars map[string]string
	SandboxID    string
}

type rawFile struct {
	Version int        `toml:"version"`
	Network rawNetwork `toml:"network"`
	Env     rawEnv     `toml:"env"`
}

type rawNetwork struct {
	Profile string   `toml:"profile"`
	Allow   []string `toml:"allow"`
}

type rawEnv struct {
	Allow    []string `toml:"allow"`
	FromFile string   `toml:"from_file"`
}

// LoadFile loads a layer. Returns os.ErrNotExist if the file does not exist.
func LoadFile(path string) (*Layer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var raw rawFile
	md, err := toml.Decode(string(data), &raw)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid TOML: %w", path, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, k := range undecoded {
			keys = append(keys, k.String())
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("%s: unknown keys: %s", path, strings.Join(keys, ", "))
	}
	if !md.IsDefined("version") {
		return nil, fmt.Errorf("%s: missing required field: version = 1", path)
	}
	if raw.Version != 1 {
		return nil, fmt.Errorf("%s: unsupported config version: %d (expected 1)", path, raw.Version)
	}

	layer := &Layer{Path: path}
	if md.IsDefined("network", "profile") {
		p := strings.TrimSpace(raw.Network.Profile)
		if p == "" {
			return nil, fmt.Errorf("%s: network.profile cannot be empty", path)
		}
		layer.Profile = &p
	}
	if md.IsDefined("network", "allow") {
		list := copyList(raw.Network.Allow)
		layer.NetworkAllow = &list
	}
	if md.IsDefined("env", "allow") {
		list := copyList(raw.Env.Allow)
		layer.EnvAllow = &list
	}
	if md.IsDefined("env", "from_file") {
		f := strings.TrimSpace(raw.Env.FromFile)
		if f == "" {
			return nil, fmt.Errorf("%s: env.from_file cannot be empty", path)
		}
		layer.EnvFromFile = &f
	}
	return layer, nil
}

// GlobalPath returns the global config path.
func GlobalPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("discover config directory: %w", err)
	}
	return filepath.Join(base, "asb", "config.toml"), nil
}

// Resolve loads global + project, applies overrides and validates the result.
func Resolve(projectDir string, cli Overrides) (Resolved, error) {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve project path: %w", err)
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve project path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Resolved{}, fmt.Errorf("project %s: %w", abs, err)
	}
	if !info.IsDir() {
		return Resolved{}, fmt.Errorf("project %s is not a directory", abs)
	}

	res := Resolved{
		ProjectDir:   abs,
		FromFileVars: map[string]string{},
	}

	globalPath, err := GlobalPath()
	if err != nil {
		return Resolved{}, err
	}
	globalLayer, err := LoadFile(globalPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Resolved{}, err
	}
	if globalLayer != nil {
		res.GlobalPath = globalPath
		applyLayer(&res, globalLayer)
	}

	projectPath := filepath.Join(abs, "asb.toml")
	projectLayer, err := LoadFile(projectPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Resolved{}, err
	}
	if projectLayer != nil {
		res.ProjectPath = projectPath
		applyLayer(&res, projectLayer)
	}

	// Closed default: with no choice at any level, fail.
	if cli.Profile != nil {
		res.Profile = Profile(strings.TrimSpace(*cli.Profile))
	}
	if res.Profile == "" {
		return res, ErrNoProfile
	}
	if cli.NetworkAllow != nil {
		res.NetworkAllow = copyList(*cli.NetworkAllow)
	}
	if cli.EnvAllow != nil {
		res.EnvAllow = copyList(*cli.EnvAllow)
	}
	if cli.SandboxID != nil {
		res.SandboxID = strings.TrimSpace(*cli.SandboxID)
	}
	if res.EnvAllow == nil {
		res.EnvAllow = append([]string(nil), BaseEnvAllow...)
	}

	if err := validateProfile(res.Profile); err != nil {
		return res, err
	}
	res.NetworkAllow, err = normalizeAllow(res.NetworkAllow)
	if err != nil {
		return res, err
	}
	if res.Profile != ProfileAllowlist && len(res.NetworkAllow) > 0 {
		return res, fmt.Errorf("network.allow is only valid with network.profile = \"allowlist\" (current profile: %s)", res.Profile)
	}
	res.EnvAllow, err = normalizeEnvAllow(res.EnvAllow)
	if err != nil {
		return res, err
	}
	if res.EnvFromFile != "" {
		vars, err := loadEnvFile(res.ProjectDir, res.EnvFromFile)
		if err != nil {
			return res, err
		}
		res.FromFileVars = vars
	}
	if res.SandboxID != "" {
		if err := ValidateSandboxID(res.SandboxID); err != nil {
			return res, err
		}
	}
	return res, nil
}

func applyLayer(res *Resolved, layer *Layer) {
	if layer.Profile != nil {
		res.Profile = Profile(strings.TrimSpace(*layer.Profile))
	}
	if layer.NetworkAllow != nil {
		res.NetworkAllow = copyList(*layer.NetworkAllow)
	}
	if layer.EnvAllow != nil {
		res.EnvAllow = copyList(*layer.EnvAllow)
	}
	if layer.EnvFromFile != nil {
		res.EnvFromFile = *layer.EnvFromFile
	}
}

// copyList preserves the difference between nil (not set) and empty (set as
// an empty list), which the replacement rule requires.
func copyList(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// ValidateProfile validates a network profile.
func ValidateProfile(p Profile) error {
	return validateProfile(p)
}

func validateProfile(p Profile) error {
	switch p {
	case ProfileHost, ProfileNone, ProfileAllowlist:
		return nil
	default:
		return fmt.Errorf("invalid network.profile: %q (use host, none or allowlist)", p)
	}
}

func normalizeAllow(in []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, entry := range in {
		host, err := NormalizeHost(entry)
		if err != nil {
			return nil, err
		}
		if !seen[host] {
			seen[host] = true
			out = append(out, host)
		}
	}
	sort.Strings(out)
	return out, nil
}

// NormalizeHost validates and normalizes an allowlist entry. It accepts an
// exact name (github.com) or a subdomain wildcard (*.example.com). A direct IP
// is not accepted: the list is by destination name.
func NormalizeHost(entry string) (string, error) {
	h := strings.ToLower(strings.TrimSpace(entry))
	h = strings.TrimSuffix(h, ".")
	if h == "" {
		return "", errors.New("network.allow: empty entry")
	}
	wildcard := strings.HasPrefix(h, "*.")
	body := h
	if wildcard {
		body = strings.TrimPrefix(h, "*.")
	}
	if body == "" {
		return "", fmt.Errorf("network.allow: invalid entry %q", entry)
	}
	if net.ParseIP(body) != nil {
		return "", fmt.Errorf("network.allow: direct IP is not allowed (%q); use the destination name", entry)
	}
	if strings.ContainsAny(body, "/:@[] \t") {
		return "", fmt.Errorf("network.allow: invalid entry %q (use only a host name)", entry)
	}
	if len(body) > 253 {
		return "", fmt.Errorf("network.allow: hostname too long %q", entry)
	}
	for _, label := range strings.Split(body, ".") {
		if label == "" || len(label) > 63 {
			return "", fmt.Errorf("network.allow: invalid hostname %q", entry)
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("network.allow: invalid hostname %q", entry)
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return "", fmt.Errorf("network.allow: invalid hostname %q", entry)
			}
		}
	}
	return h, nil
}

func normalizeEnvAllow(in []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, name := range in {
		name = strings.TrimSpace(name)
		if !envNameRE.MatchString(name) {
			return nil, fmt.Errorf("env.allow: invalid variable name %q", name)
		}
		if reservedEnv[name] {
			// HOME and proxy are set by the tool; skipping them here avoids
			// inheriting a channel the policy does not control.
			continue
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

// BuildEnv builds the effective sandbox environment. Values are never logged.
func (r Resolved) BuildEnv(hostEnv func(string) (string, bool)) []string {
	env := make(map[string]string)
	for _, name := range r.EnvAllow {
		if reservedEnv[name] {
			continue
		}
		if v, ok := hostEnv(name); ok {
			env[name] = v
		}
	}
	for name, v := range r.FromFileVars {
		if reservedEnv[name] {
			continue
		}
		env[name] = v
	}
	names := make([]string, 0, len(env))
	for name := range env {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, name+"="+env[name])
	}
	return out
}

func loadEnvFile(projectDir, configured string) (map[string]string, error) {
	path := configured
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectDir, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("env.from_file: %w", err)
	}
	rel, err := filepath.Rel(projectDir, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("env.from_file: %s must be inside the project", configured)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("env.from_file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("env.from_file: %s is not a regular file", configured)
	}
	if info.Mode().Perm() != 0o600 {
		return nil, fmt.Errorf("env.from_file: %s must have mode 0600 (current %04o)", configured, info.Mode().Perm())
	}
	if err := requireGitIgnored(projectDir, rel); err != nil {
		return nil, err
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("env.from_file: %w", err)
	}
	defer f.Close()

	vars := map[string]string{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSuffix(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("env.from_file: %s:%d: expected VAR=value", configured, lineNo)
		}
		name = strings.TrimSpace(name)
		if !envNameRE.MatchString(name) {
			return nil, fmt.Errorf("env.from_file: %s:%d: invalid variable name %q", configured, lineNo, name)
		}
		if reservedEnv[name] {
			return nil, fmt.Errorf("env.from_file: %s:%d: %s is reserved and set by the tool", configured, lineNo, name)
		}
		if _, dup := vars[name]; dup {
			return nil, fmt.Errorf("env.from_file: %s:%d: duplicate variable %s", configured, lineNo, name)
		}
		vars[name] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("env.from_file: %w", err)
	}
	return vars, nil
}

func requireGitIgnored(projectDir, rel string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("env.from_file: git not found; cannot verify whether %s is in .gitignore", rel)
	}
	cmd := exec.Command("git", "-C", projectDir, "check-ignore", "-q", "--", rel)
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return fmt.Errorf("env.from_file: %s is not in .gitignore; add it before using secrets", rel)
	}
	return fmt.Errorf("env.from_file: could not check .gitignore for %s (is the project a git repository?): %w", rel, err)
}

// ValidateSandboxID prevents path traversal and ambiguous names.
func ValidateSandboxID(id string) error {
	if id == "" {
		return errors.New("empty sandbox id")
	}
	if len(id) > 128 {
		return fmt.Errorf("sandbox id too long (max. 128): %q", id)
	}
	if id == "." || id == ".." || strings.HasPrefix(id, ".") {
		return fmt.Errorf("invalid sandbox id: %q", id)
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
			return fmt.Errorf("invalid sandbox id: %q (use letters, numbers, . _ -)", id)
		}
	}
	return nil
}
