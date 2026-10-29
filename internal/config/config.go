// Package config carrega e resolve a configuração declarativa do asb.
//
// Precedência: flag de CLI > asb.toml do projeto > config global > nenhum
// default. Listas substituem, não fazem merge.
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

// Profile é o perfil de rede da v1.
type Profile string

const (
	ProfileHost      Profile = "host"
	ProfileNone      Profile = "none"
	ProfileAllowlist Profile = "allowlist"
)

// BaseEnvAllow é a allowlist base fechada da v1 (RFC-0001 §5.4). HOME e as
// variáveis de proxy são definidas pela ferramenta, não herdadas.
var BaseEnvAllow = []string{
	"PATH", "TERM", "LANG", "LC_ALL", "TZ", "USER", "LOGNAME", "SHELL",
}

// reservedEnv nunca é herdado do host, mesmo que declarado em [env] allow ou
// em from_file: HOME e o canal de proxy são definidos pela ferramenta.
var reservedEnv = map[string]bool{
	"HOME": true, "TMPDIR": true,
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "ALL_PROXY": true, "NO_PROXY": true,
	"http_proxy": true, "https_proxy": true, "all_proxy": true, "no_proxy": true,
	"NODE_USE_ENV_PROXY": true,
}

var envNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ErrNoProfile indica que nenhum perfil foi escolhido em nenhum nível.
var ErrNoProfile = errors.New("nenhum perfil de rede configurado")

// Layer é uma camada de configuração já validada.
type Layer struct {
	Path         string
	Profile      *string
	NetworkAllow *[]string
	EnvAllow     *[]string
	EnvFromFile  *string
}

// Overrides são as flags de CLI. Nil significa "não informado".
type Overrides struct {
	Profile      *string
	NetworkAllow *[]string
	EnvAllow     *[]string
	SandboxID    *string
}

// Resolved é a configuração efetiva depois da precedência.
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

// LoadFile carrega uma camada. Retorna os.ErrNotExist se o arquivo não existir.
func LoadFile(path string) (*Layer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("ler %s: %w", path, err)
	}

	var raw rawFile
	md, err := toml.Decode(string(data), &raw)
	if err != nil {
		return nil, fmt.Errorf("%s: TOML inválido: %w", path, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, k := range undecoded {
			keys = append(keys, k.String())
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("%s: chaves desconhecidas: %s", path, strings.Join(keys, ", "))
	}
	if !md.IsDefined("version") {
		return nil, fmt.Errorf("%s: campo obrigatório ausente: version = 1", path)
	}
	if raw.Version != 1 {
		return nil, fmt.Errorf("%s: versão de config não suportada: %d (esperado 1)", path, raw.Version)
	}

	layer := &Layer{Path: path}
	if md.IsDefined("network", "profile") {
		p := strings.TrimSpace(raw.Network.Profile)
		if p == "" {
			return nil, fmt.Errorf("%s: network.profile não pode ser vazio", path)
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
			return nil, fmt.Errorf("%s: env.from_file não pode ser vazio", path)
		}
		layer.EnvFromFile = &f
	}
	return layer, nil
}

// GlobalPath devolve o caminho da config global.
func GlobalPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("descobrir diretório de config: %w", err)
	}
	return filepath.Join(base, "asb", "config.toml"), nil
}

// Resolve carrega global + projeto, aplica overrides e valida o resultado.
func Resolve(projectDir string, cli Overrides) (Resolved, error) {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolver caminho do projeto: %w", err)
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolver caminho do projeto: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Resolved{}, fmt.Errorf("projeto %s: %w", abs, err)
	}
	if !info.IsDir() {
		return Resolved{}, fmt.Errorf("projeto %s não é um diretório", abs)
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

	// Default fechado: sem escolha em nenhum nível, falha.
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
		return res, fmt.Errorf("network.allow só é válido com network.profile = \"allowlist\" (perfil atual: %s)", res.Profile)
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

// copyList preserva a diferença entre nil (não definido) e vazio (definido
// como lista vazia), necessária para a regra de substituição.
func copyList(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// ValidateProfile valida um perfil de rede.
func ValidateProfile(p Profile) error {
	return validateProfile(p)
}

func validateProfile(p Profile) error {
	switch p {
	case ProfileHost, ProfileNone, ProfileAllowlist:
		return nil
	default:
		return fmt.Errorf("network.profile inválido: %q (use host, none ou allowlist)", p)
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

// NormalizeHost valida e normaliza uma entrada de allowlist. Aceita nome
// exato (github.com) ou wildcard de subdomínio (*.example.com). IP direto não
// é aceito: a lista é por nome de destino.
func NormalizeHost(entry string) (string, error) {
	h := strings.ToLower(strings.TrimSpace(entry))
	h = strings.TrimSuffix(h, ".")
	if h == "" {
		return "", errors.New("network.allow: entrada vazia")
	}
	wildcard := strings.HasPrefix(h, "*.")
	body := h
	if wildcard {
		body = strings.TrimPrefix(h, "*.")
	}
	if body == "" {
		return "", fmt.Errorf("network.allow: entrada inválida %q", entry)
	}
	if net.ParseIP(body) != nil {
		return "", fmt.Errorf("network.allow: IP direto não é permitido (%q); use o nome do destino", entry)
	}
	if strings.ContainsAny(body, "/:@[] \t") {
		return "", fmt.Errorf("network.allow: entrada inválida %q (use apenas nome de host)", entry)
	}
	if len(body) > 253 {
		return "", fmt.Errorf("network.allow: hostname longo demais %q", entry)
	}
	for _, label := range strings.Split(body, ".") {
		if label == "" || len(label) > 63 {
			return "", fmt.Errorf("network.allow: hostname inválido %q", entry)
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("network.allow: hostname inválido %q", entry)
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return "", fmt.Errorf("network.allow: hostname inválido %q", entry)
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
			return nil, fmt.Errorf("env.allow: nome de variável inválido %q", name)
		}
		if reservedEnv[name] {
			// HOME e proxy são definidos pela ferramenta; ignorar aqui evita
			// herdar um canal que a política não controla.
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

// BuildEnv monta o ambiente efetivo do sandbox. Valores nunca são logados.
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
		return nil, fmt.Errorf("env.from_file: %s precisa estar dentro do projeto", configured)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("env.from_file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("env.from_file: %s não é um arquivo regular", configured)
	}
	if info.Mode().Perm() != 0o600 {
		return nil, fmt.Errorf("env.from_file: %s precisa ter permissão 0600 (atual %04o)", configured, info.Mode().Perm())
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
			return nil, fmt.Errorf("env.from_file: %s:%d: esperado VAR=valor", configured, lineNo)
		}
		name = strings.TrimSpace(name)
		if !envNameRE.MatchString(name) {
			return nil, fmt.Errorf("env.from_file: %s:%d: nome de variável inválido %q", configured, lineNo, name)
		}
		if reservedEnv[name] {
			return nil, fmt.Errorf("env.from_file: %s:%d: %s é reservada e definida pela ferramenta", configured, lineNo, name)
		}
		if _, dup := vars[name]; dup {
			return nil, fmt.Errorf("env.from_file: %s:%d: variável duplicada %s", configured, lineNo, name)
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
		return fmt.Errorf("env.from_file: git não encontrado; não é possível verificar se %s está no .gitignore", rel)
	}
	cmd := exec.Command("git", "-C", projectDir, "check-ignore", "-q", "--", rel)
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return fmt.Errorf("env.from_file: %s não está no .gitignore; adicione antes de usar segredos", rel)
	}
	return fmt.Errorf("env.from_file: não foi possível verificar .gitignore para %s (o projeto é um repositório git?): %w", rel, err)
}

// ValidateSandboxID impede path traversal e nomes ambíguos.
func ValidateSandboxID(id string) error {
	if id == "" {
		return errors.New("sandbox id vazio")
	}
	if len(id) > 128 {
		return fmt.Errorf("sandbox id longo demais (máx. 128): %q", id)
	}
	if id == "." || id == ".." || strings.HasPrefix(id, ".") {
		return fmt.Errorf("sandbox id inválido: %q", id)
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
			return fmt.Errorf("sandbox id inválido: %q (use letras, números, . _ -)", id)
		}
	}
	return nil
}
