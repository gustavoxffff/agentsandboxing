package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ErrDeniedHost indicates a destination outside the allowlist.
var ErrDeniedHost = errors.New("host outside the allowlist")

// ErrDeniedIP indicates that the destination resolved to a private IP,
// link-local, loopback, metadata or another blocked range (protection against
// rebinding).
var ErrDeniedIP = errors.New("destination resolved to a blocked IP")

// ErrNoIPv4 indicates a host with no A record; the proxy is IPv4-only and
// fails closed.
var ErrNoIPv4 = errors.New("host without an IPv4 address")

// ErrIPLiteral indicates an attempt to use a direct IP in the allowlist.
var ErrIPLiteral = errors.New("direct IP is not allowed")

// Allowed reports whether host matches the list. Exact entries match the host
// itself; "*.example.com" entries match subdomains, not the apex.
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

// Resolver resolves names and applies the IP policy. The interface allows
// injecting DNS in tests.
type Resolver interface {
	Resolve(ctx context.Context, host string) ([]net.IP, error)
}

// LookupFunc is the signature of net.Resolver.LookupIPAddr.
type LookupFunc func(ctx context.Context, host string) ([]net.IPAddr, error)

// DNSResolver resolves only IPv4 and fails closed if any response is private.
// The resolved IP is the same one that will be dialed (no re-resolution),
// closing the DNS rebinding window.
type DNSResolver struct {
	Lookup LookupFunc
	// Timeout limits resolution; zero uses 10s.
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
	"240.0.0.0/4",     // reserved
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

// Denied decides whether an IP cannot be a destination. IPv6 is always denied
// in v1.
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
