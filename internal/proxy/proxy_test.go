package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAllowed(t *testing.T) {
	allow := []string{"github.com", "*.example.com"}
	cases := map[string]bool{
		"github.com":         true,
		"api.github.com":     false,
		"example.com":        false,
		"a.example.com":      true,
		"a.b.example.com":    true,
		"notexample.com":     false,
		"evil-example.com":   false,
		"GITHUB.COM.":        true,
		"registry.npmjs.org": false,
	}
	for host, want := range cases {
		if got := Allowed(host, allow); got != want {
			t.Errorf("Allowed(%q) = %v, esperava %v", host, got, want)
		}
	}
}

func TestDenied(t *testing.T) {
	denied := []string{
		"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1",
		"169.254.169.254", "100.64.0.1", "0.0.0.0", "224.0.0.1",
		"198.18.0.1", "::1", "2001:db8::1",
	}
	for _, s := range denied {
		if !Denied(net.ParseIP(s)) {
			t.Errorf("%s deveria ser negado", s)
		}
	}
	for _, s := range []string{"1.1.1.1", "8.8.8.8", "140.82.112.3"} {
		if Denied(net.ParseIP(s)) {
			t.Errorf("%s deveria ser permitido", s)
		}
	}
}

type staticResolver struct {
	ips []net.IP
	err error
}

func (r staticResolver) Resolve(ctx context.Context, host string) ([]net.IP, error) {
	return r.ips, r.err
}

func TestDNSResolver(t *testing.T) {
	mk := func(ips ...string) DNSResolver {
		return DNSResolver{Lookup: func(ctx context.Context, host string) ([]net.IPAddr, error) {
			var out []net.IPAddr
			for _, s := range ips {
				out = append(out, net.IPAddr{IP: net.ParseIP(s)})
			}
			return out, nil
		}}
	}

	got, err := mk("1.1.1.1", "2606:4700::1").Resolve(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].String() != "1.1.1.1" {
		t.Fatalf("IPv4-only falhou: %v", got)
	}

	if _, err := mk("2606:4700::1").Resolve(context.Background(), "v6.test"); !errors.Is(err, ErrNoIPv4) {
		t.Fatalf("esperava ErrNoIPv4, veio %v", err)
	}
	if _, err := mk("1.1.1.1", "127.0.0.1").Resolve(context.Background(), "rebind.test"); !errors.Is(err, ErrDeniedIP) {
		t.Fatalf("esperava ErrDeniedIP para rebinding, veio %v", err)
	}
	if _, err := mk("169.254.169.254").Resolve(context.Background(), "metadata.test"); !errors.Is(err, ErrDeniedIP) {
		t.Fatalf("esperava ErrDeniedIP para metadata, veio %v", err)
	}
	if _, err := mk("1.1.1.1").Resolve(context.Background(), "1.1.1.1"); !errors.Is(err, ErrIPLiteral) {
		t.Fatalf("esperava ErrIPLiteral, veio %v", err)
	}
}

func startProxy(t *testing.T, allow []string, resolver Resolver) (addr string, stop func()) {
	t.Helper()
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := &Proxy{Allow: allow, Resolver: resolver}
	done := make(chan struct{})
	go func() {
		_ = p.Serve(l)
		close(done)
	}()
	return l.Addr().String(), func() {
		_ = l.Close()
		<-done
	}
}

func echoServer(t *testing.T) (port string, stop func()) {
	t.Helper()
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	_, p, _ := net.SplitHostPort(l.Addr().String())
	return p, func() { _ = l.Close() }
}

func TestConnectTunnel(t *testing.T) {
	upstreamPort, stopEcho := echoServer(t)
	defer stopEcho()
	proxyAddr, stopProxy := startProxy(t, []string{"allowed.test"}, staticResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}})
	defer stopProxy()

	conn, err := net.Dial("tcp4", proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	fmt.Fprintf(conn, "CONNECT allowed.test:%s HTTP/1.1\r\nHost: allowed.test:%s\r\n\r\n", upstreamPort, upstreamPort)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if _, err := io.WriteString(conn, "ping"); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(br, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "ping" {
		t.Fatalf("echo = %q", buf)
	}
}

func TestConnectDenied(t *testing.T) {
	proxyAddr, stop := startProxy(t, []string{"allowed.test"}, staticResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}})
	defer stop()

	conn, err := net.Dial("tcp4", proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprint(conn, "CONNECT denied.test:443 HTTP/1.1\r\nHost: denied.test:443\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, esperava 403", resp.StatusCode)
	}
}

func TestIPLiteralDenied(t *testing.T) {
	proxyAddr, stop := startProxy(t, []string{"1.1.1.1"}, staticResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}})
	defer stop()

	conn, err := net.Dial("tcp4", proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprint(conn, "CONNECT 1.1.1.1:443 HTTP/1.1\r\nHost: 1.1.1.1:443\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, esperava 403", resp.StatusCode)
	}
}

func TestHTTPForward(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, "world")
	}))
	defer upstream.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	proxyAddr, stop := startProxy(t, []string{"allowed.test"}, staticResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}})
	defer stop()

	conn, err := net.Dial("tcp4", proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET http://allowed.test:%s/hello HTTP/1.1\r\nHost: allowed.test:%s\r\nConnection: close\r\n\r\n", port, port)
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "world" {
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}
}
