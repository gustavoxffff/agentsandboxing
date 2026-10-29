// Package bridge expõe o socket Unix do proxy como um proxy HTTP em
// 127.0.0.1 dentro do namespace de rede isolado do sandbox.
package bridge

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const dialTimeout = 10 * time.Second

// Run sobe a bridge em uma porta efêmera e executa o comando filho com as
// variáveis de proxy apontando para ela.
func Run(socket string, command []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if len(command) == 0 {
		return 0, errors.New("bridge: comando vazio")
	}
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("bridge: listen: %w", err)
	}
	return Serve(l, socket, command, stdin, stdout, stderr)
}

// Serve é Run com um listener injetável (usado nos testes).
func Serve(l net.Listener, socket string, command []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if len(command) == 0 {
		_ = l.Close()
		return 0, errors.New("bridge: comando vazio")
	}
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		_ = l.Close()
		return 0, fmt.Errorf("bridge: listener não é TCP: %s", l.Addr())
	}
	port := addr.Port

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = WithProxyEnv(os.Environ(), port)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	if err := cmd.Start(); err != nil {
		_ = l.Close()
		return 0, fmt.Errorf("bridge: iniciar %s: %w", command[0], err)
	}

	stop := make(chan struct{})
	go acceptLoop(l, socket, stop)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err := cmd.Wait()
	close(stop)
	_ = l.Close()

	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 1, fmt.Errorf("bridge: esperar comando: %w", err)
}

func acceptLoop(l net.Listener, socket string, stop <-chan struct{}) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		select {
		case <-stop:
			_ = conn.Close()
			return
		default:
		}
		go forward(conn, socket)
	}
}

func forward(client net.Conn, socket string) {
	defer client.Close()
	upstream, err := net.DialTimeout("unix", socket, dialTimeout)
	if err != nil {
		return
	}
	defer upstream.Close()

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(upstream, client)
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

// WithProxyEnv devolve env com as variáveis de proxy da bridge substituindo
// qualquer valor herdado. NODE_USE_ENV_PROXY cobre toolchains Node.
func WithProxyEnv(env []string, port int) []string {
	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	noProxy := "localhost,127.0.0.1,::1"
	values := map[string]string{
		"HTTP_PROXY":  proxyURL,
		"HTTPS_PROXY": proxyURL,
		"ALL_PROXY":   proxyURL,
		"http_proxy":  proxyURL,
		"https_proxy": proxyURL,
		"all_proxy":   proxyURL,
		"NO_PROXY":    noProxy,
		"no_proxy":    noProxy,

		"NODE_USE_ENV_PROXY": "1",
	}

	out := make([]string, 0, len(env)+len(values))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if _, replaced := values[name]; replaced {
			continue
		}
		out = append(out, kv)
	}
	for name, value := range values {
		out = append(out, name+"="+value)
	}
	return out
}
