package run

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
	"github.com/gustavoxffff/agentsandboxing/internal/sandbox"
)

// PrintConfig prints the resolved configuration for auditing. It never prints
// secret values, only variable names.
func PrintConfig(w io.Writer, cfg config.Resolved, layout sandbox.Layout, command []string) {
	fmt.Fprintln(w, "asb: resolved configuration")
	fmt.Fprintf(w, "  project: %s\n", cfg.ProjectDir)
	if cfg.GlobalPath != "" {
		fmt.Fprintf(w, "  global config: %s\n", cfg.GlobalPath)
	} else {
		fmt.Fprintln(w, "  global config: (none)")
	}
	if cfg.ProjectPath != "" {
		fmt.Fprintf(w, "  project config: %s\n", cfg.ProjectPath)
	} else {
		fmt.Fprintln(w, "  project config: (none)")
	}
	fmt.Fprintf(w, "  network profile: %s\n", cfg.Profile)
	fmt.Fprintf(w, "  guarantee: %s\n", guarantee(cfg.Profile))
	fmt.Fprintf(w, "  network.allow: %s\n", joinOrNone(cfg.NetworkAllow))
	fmt.Fprintf(w, "  env.allow: %s\n", joinOrNone(cfg.EnvAllow))
	if cfg.EnvFromFile != "" {
		names := make([]string, 0, len(cfg.FromFileVars))
		for name := range cfg.FromFileVars {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintf(w, "  env.from_file: %s (variables: %s)\n", cfg.EnvFromFile, joinOrNone(names))
	}
	fmt.Fprintf(w, "  sandbox id: %s\n", layout.ID)
	fmt.Fprintf(w, "  state (host): %s\n", layout.HostOpencodeRoot)
	fmt.Fprintf(w, "  state (sandbox): %s\n", layout.InsideRoot)
	if cfg.Profile == config.ProfileAllowlist {
		fmt.Fprintf(w, "  proxy (socket): %s\n", layout.ProxySocket)
		fmt.Fprintln(w, "  proxy: TCP→Unix bridge inside the sandbox; direct egress blocked")
	}
	fmt.Fprintf(w, "  command: %s\n", strings.Join(command, " "))
}

func guarantee(p config.Profile) string {
	switch p {
	case config.ProfileHost:
		return "host network (no network isolation)"
	case config.ProfileNone:
		return "loopback only; no external connection"
	case config.ProfileAllowlist:
		return "allowlist destinations only, via proxy; direct egress blocked"
	default:
		return "unknown"
	}
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
