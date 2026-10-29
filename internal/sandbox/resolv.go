package sandbox

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
)

// ResolvContent gera o /etc/resolv.conf do sandbox.
//
// host: reaproveita os nameservers do host (necessário para ferramentas que
// resolvem DNS diretamente). none/allowlist: aponta para loopback, onde não há
// listener; DNS direto falha e, no allowlist, a resolução acontece no proxy.
func ResolvContent(profile config.Profile) string {
	if profile == config.ProfileHost {
		if content := hostNameservers(); content != "" {
			return content + "\noptions edns0\n"
		}
		return "nameserver 1.1.1.1\noptions edns0\n"
	}
	return "nameserver 127.0.0.1\noptions edns0\n"
}

func hostNameservers() string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return ""
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "nameserver") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// ResolvFile materializa o conteúdo em um arquivo temporário aberto. O
// chamador passa o arquivo como fd 3 para o bwrap (--file 3 DEST).
func ResolvFile(content string) (*os.File, error) {
	f, err := os.CreateTemp("", "asb-resolv-*")
	if err != nil {
		return nil, fmt.Errorf("criar resolv.conf temporário: %w", err)
	}
	// O fd continua válido depois do unlink.
	_ = os.Remove(f.Name())
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return nil, fmt.Errorf("escrever resolv.conf temporário: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return nil, fmt.Errorf("rebobinar resolv.conf temporário: %w", err)
	}
	return f, nil
}
