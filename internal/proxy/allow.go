package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ErrDeniedHost indica destino fora da allowlist.
var ErrDeniedHost = errors.New("host fora da allowlist")

// ErrDeniedIP indica que o destino resolveu para IP privado, link-local,
// loopback, metadata ou outro intervalo bloqueado (proteção contra rebinding).
var ErrDeniedIP = errors.New("destino resolveu para IP bloqueado")

// ErrNoIPv4 indica host sem registro A; o proxy é IPv4-only e falha fechado.
var ErrNoIPv4 = errors.New("host sem endereço IPv4")

// ErrIPLiteral indica tentativa de usar IP direto na allowlist.
var ErrIPLiteral = errors.New("IP direto não é permitido")

// Allowed informa se host casa com a lista. Entradas exatas casam o próprio
// host; entradas "*.example.com" casam subdomínios, não o apex.
func Allowed(host string, allow []string) bool {
	host = normalizeHost(host)
	if host == "" {
		return false
	}
	for _, entry := range allow {
		entry = normalizeHost(entry)
		if entry == "" {
			continue
		}
		if strings.HasPrefix(entry, "*.") {
			suffix := entry[1:] // ".example.com"
			if strings.HasSuffix(host, suffix) && len(host) > len(suffix) {
				return true
			}
			continue
		}
		if host == entry {
			return true
		}
	}
	return false
}

func normalizeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

// Resolver resolve nomes e aplica a política de IP. A interface permite
// injetar DNS nos testes.
type Resolver interface {
	Resolve(ctx context.Context, host string) ([]net.IP, error)
}

// LookupFunc é a assinatura de net.Resolver.LookupIPAddr.
type LookupFunc func(ctx context.Context, host string) ([]net.IPAddr, error)

// DNSResolver resolve apenas IPv4 e falha fechado se qualquer resposta for
// privada. O IP resolvido é o mesmo que será discado (sem re-resolução),
// fechando a janela de DNS rebinding.
type DNSResolver struct {
	Lookup LookupFunc
	// Timeout limita a resolução; zero usa 10s.
	Timeout time.Duration
}

func (r DNSResolver) Resolve(ctx context.Context, host string) ([]net.IP, error) {
	if net.ParseIP(host) != nil {
		return nil, ErrIPLiteral
	}
	lookup := r.Lookup
	if lookup == nil {
		lookup = net.DefaultResolver.LookupIPAddr
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addrs, err := lookup(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolver %s: %w", host, err)
	}
	var v4 []net.IP
	for _, addr := range addrs {
		if ip4 := addr.IP.To4(); ip4 != nil {
			v4 = append(v4, ip4)
		}
	}
	if len(v4) == 0 {
		return nil, ErrNoIPv4
	}
	for _, ip := range v4 {
		if Denied(ip) {
			return nil, fmt.Errorf("%w: %s -> %s", ErrDeniedIP, host, ip)
		}
	}
	return v4, nil
}

var deniedNets = mustParseCIDRs(
	"0.0.0.0/8",
	"100.64.0.0/10",   // CGNAT
	"169.254.0.0/16",  // link-local / metadata
	"192.0.0.0/24",    // IETF protocol assignments
	"192.0.2.0/24",    // TEST-NET-1
	"198.18.0.0/15",   // benchmarking
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"240.0.0.0/4",     // reservado
)

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		out = append(out, n)
	}
	return out
}

// Denied decide se um IP não pode ser destino. IPv6 é sempre negado na v1.
func Denied(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return true
	}
	if ip4.IsPrivate() || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() ||
		ip4.IsLinkLocalMulticast() || ip4.IsMulticast() || ip4.IsUnspecified() {
		return true
	}
	for _, n := range deniedNets {
		if n.Contains(ip4) {
			return true
		}
	}
	return false
}
