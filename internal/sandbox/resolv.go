package sandbox

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gustavoxffff/agentsandboxing/internal/config"
)

// ResolvContent generates the sandbox /etc/resolv.conf.
//
// host: reuses the host nameservers (needed for tools that resolve DNS
// directly). none/allowlist: points to loopback, where there is no listener;
// direct DNS fails and, in allowlist, resolution happens in the proxy.
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

// ResolvFile materializes the content in an open temporary file. The caller
// passes the file as fd 3 to bwrap (--file 3 DEST).
func ResolvFile(content string) (*os.File, error) {
	f, err := os.CreateTemp("", "asb-resolv-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary resolv.conf: %w", err)
	}
	// The fd stays valid after the unlink.
	_ = os.Remove(f.Name())
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return nil, fmt.Errorf("write temporary resolv.conf: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return nil, fmt.Errorf("rewind temporary resolv.conf: %w", err)
	}
	return f, nil
}
