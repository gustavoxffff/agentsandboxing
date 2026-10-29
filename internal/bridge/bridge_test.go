package bridge

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func unixEcho(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proxy.sock")
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
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
	return path
}

func TestServeForwardsToUnixSocket(t *testing.T) {
	socket := unixEcho(t)
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	codeCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		code, err := Serve(l, socket, []string{"sleep", "0.4"}, os.Stdin, io.Discard, io.Discard)
		codeCh <- code
		errCh <- err
	}()

	conn, err := net.Dial("tcp4", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.WriteString(conn, "ping"); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "ping" {
		t.Fatalf("echo = %q", buf)
	}

	select {
	case code := <-codeCh:
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
		if code != 0 {
			t.Fatalf("código = %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bridge não terminou")
	}
}

func TestWithProxyEnvReplacesInheritedValues(t *testing.T) {
	env := WithProxyEnv([]string{
		"PATH=/usr/bin",
		"HTTP_PROXY=http://evil",
		"HTTPS_PROXY=http://evil",
		"NO_PROXY=*",
	}, 4321)
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "evil") || strings.Contains(joined, "NO_PROXY=*") {
		t.Fatalf("valores herdados não foram substituídos:\n%s", joined)
	}
	for _, want := range []string{
		"HTTP_PROXY=http://127.0.0.1:4321",
		"HTTPS_PROXY=http://127.0.0.1:4321",
		"NO_PROXY=localhost,127.0.0.1,::1",
		"NODE_USE_ENV_PROXY=1",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("faltou %q:\n%s", want, joined)
		}
	}
	if !strings.Contains(joined, "PATH=/usr/bin") {
		t.Fatalf("PATH perdido:\n%s", joined)
	}
}
