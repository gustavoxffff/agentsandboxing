package run

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
	"github.com/gustavoxffff/agentsandboxing/internal/sandbox"
)

// PrintConfig imprime a configuração resolvida para auditoria. Nunca imprime
// valores de segredos, apenas nomes de variáveis.
func PrintConfig(w io.Writer, cfg config.Resolved, layout sandbox.Layout, command []string) {
	fmt.Fprintln(w, "asb: configuração resolvida")
	fmt.Fprintf(w, "  projeto: %s\n", cfg.ProjectDir)
	if cfg.GlobalPath != "" {
		fmt.Fprintf(w, "  config global: %s\n", cfg.GlobalPath)
	} else {
		fmt.Fprintln(w, "  config global: (nenhuma)")
	}
	if cfg.ProjectPath != "" {
		fmt.Fprintf(w, "  config do projeto: %s\n", cfg.ProjectPath)
	} else {
		fmt.Fprintln(w, "  config do projeto: (nenhuma)")
	}
	fmt.Fprintf(w, "  perfil de rede: %s\n", cfg.Profile)
	fmt.Fprintf(w, "  garantia: %s\n", guarantee(cfg.Profile))
	fmt.Fprintf(w, "  rede.allow: %s\n", joinOrNone(cfg.NetworkAllow))
	fmt.Fprintf(w, "  env.allow: %s\n", joinOrNone(cfg.EnvAllow))
	if cfg.EnvFromFile != "" {
		names := make([]string, 0, len(cfg.FromFileVars))
		for name := range cfg.FromFileVars {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintf(w, "  env.from_file: %s (variáveis: %s)\n", cfg.EnvFromFile, joinOrNone(names))
	}
	fmt.Fprintf(w, "  sandbox id: %s\n", layout.ID)
	fmt.Fprintf(w, "  estado (host): %s\n", layout.HostOpencodeRoot)
	fmt.Fprintf(w, "  estado (sandbox): %s\n", layout.InsideRoot)
	if cfg.Profile == config.ProfileAllowlist {
		fmt.Fprintf(w, "  proxy (socket): %s\n", layout.ProxySocket)
		fmt.Fprintln(w, "  proxy: bridge TCP→Unix dentro do sandbox; egress direto bloqueado")
	}
	fmt.Fprintf(w, "  comando: %s\n", strings.Join(command, " "))
}

func guarantee(p config.Profile) string {
	switch p {
	case config.ProfileHost:
		return "rede do host (sem isolamento de rede)"
	case config.ProfileNone:
		return "apenas loopback; nenhuma conexão externa"
	case config.ProfileAllowlist:
		return "só destinos da allowlist, via proxy; egress direto bloqueado"
	default:
		return "desconhecida"
	}
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "(nenhum)"
	}
	return strings.Join(items, ", ")
}
