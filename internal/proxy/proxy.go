// Package proxy implements the v1 CONNECT proxy: allowlist by hostname,
// DNS resolved on the host, IPv4 only, no MITM and no fallback to the network.
package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	headerTimeout = 15 * time.Second
	dialTimeout   = 10 * time.Second
	maxHeaderSize = 64 << 10
)

// Proxy is the HTTP/CONNECT proxy. It only talks to allowed destinations.
type Proxy struct {
	Allow    []string
	Resolver Resolver
	Dialer   *net.Dialer
	Logger   *log.Logger
	// Verbose also logs allowed connections; denials are always logged.
	Verbose bool
}

// Serve accepts connections until the listener is closed.
func (p *Proxy) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		go p.handle(conn)
	}
}

func (p *Proxy) handle(client net.Conn) {
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(headerTimeout))

	br := bufio.NewReaderSize(client, maxHeaderSize)
	req, err := http.ReadRequest(br)
	if err != nil {
		p.writeError(client, http.StatusBadRequest, "invalid request")
		return
	}
	defer req.Body.Close()

	host, port, err := requestTarget(req)
	if err != nil {
		p.deny(client, "-", err.Error())
		return
	}
	host = normalizeHost(host)
	if net.ParseIP(host) != nil {
		p.deny(client, host, ErrIPLiteral.Error())
		return
	}
	if !Allowed(host, p.Allow) {
		p.deny(client, host, ErrDeniedHost.Error())
		return
	}

	resolver := p.Resolver
	if resolver == nil {
		resolver = DNSResolver{}
	}
	resolveCtx, cancelResolve := context.WithTimeout(context.Background(), dialTimeout)
	ips, err := resolver.Resolve(resolveCtx, host)
	cancelResolve()
	if err != nil {
		p.deny(client, host, err.Error())
		return
	}

	dialCtx, cancelDial := context.WithTimeout(context.Background(), dialTimeout)
	upstream, err := p.dialAny(dialCtx, ips, port)
	cancelDial()
	if err != nil {
		p.deny(client, host, "failed to connect: "+err.Error())
		return
	}
	defer upstream.Close()
	if p.Verbose {
		p.logf("ALLOW %s %s:%s", req.Method, host, port)
	}

	if req.Method == http.MethodConnect {
		if _, err := io.WriteString(client, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			return
		}
		_ = client.SetDeadline(time.Time{})
		_ = upstream.SetDeadline(time.Time{})
		tunnel(client, br, upstream)
		return
	}

	p.forward(client, req, upstream)
}

func requestTarget(req *http.Request) (host, port string, err error) {
	if req.Method == http.MethodConnect {
		target := req.Host
		if target == "" {
			target = req.URL.Host
		}
		host, port, err = net.SplitHostPort(target)
		if err != nil {
			return "", "", fmt.Errorf("CONNECT requires host:port: %q", target)
		}
	} else {
		if req.URL == nil || req.URL.Host == "" {
			return "", "", errors.New("proxy requires an absolute URL")
		}
		host = req.URL.Hostname()
		port = req.URL.Port()
		if port == "" {
			port = "80"
		}
	}
	if host == "" {
		return "", "", errors.New("empty destination")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", "", fmt.Errorf("invalid port: %q", port)
	}
	return host, port, nil
}

func (p *Proxy) dialAny(ctx context.Context, ips []net.IP, port string) (net.Conn, error) {
	dialer := p.Dialer
	if dialer == nil {
		dialer = &net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second}
	}
	var lastErr error
	for _, ip := range ips {
		addr := net.JoinHostPort(ip.String(), port)
		conn, err := dialer.DialContext(ctx, "tcp4", addr)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no address to dial")
	}
	return nil, lastErr
}

func (p *Proxy) forward(client net.Conn, req *http.Request, upstream net.Conn) {
	req.RequestURI = ""
	req.URL.Scheme = ""
	req.URL.Host = ""
	req.Close = true
	removeHopHeaders(req.Header)
	req.Header.Set("Connection", "close")

	_ = upstream.SetDeadline(time.Now().Add(headerTimeout))
	if err := req.Write(upstream); err != nil {
		p.writeError(client, http.StatusBadGateway, "failed to send to the destination")
		return
	}
	_ = upstream.SetDeadline(time.Time{})
	_ = client.SetDeadline(time.Time{})
	if _, err := io.Copy(client, upstream); err != nil {
		return
	}
}

func removeHopHeaders(h http.Header) {
	for _, name := range []string{
		"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
		"Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade",
	} {
		h.Del(name)
	}
}

func tunnel(client net.Conn, clientReader io.Reader, upstream net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(upstream, clientReader)
		closeWrite(upstream)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, upstream)
		closeWrite(client)
		done <- struct{}{}
	}()
	<-done
	<-done
}

func closeWrite(c net.Conn) {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		_ = cw.CloseWrite()
	}
}

func (p *Proxy) deny(conn net.Conn, host, reason string) {
	p.logf("DENY %s %s", host, reason)
	p.writeError(conn, http.StatusForbidden, "blocked by the asb allowlist")
}

func (p *Proxy) writeError(conn net.Conn, status int, message string) {
	body := message + "\n"
	_, _ = fmt.Fprintf(conn,
		"HTTP/1.1 %d %s\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		status, http.StatusText(status), len(body), body)
}

func (p *Proxy) logf(format string, args ...any) {
	if p.Logger != nil {
		p.Logger.Printf(format, args...)
	}
}
