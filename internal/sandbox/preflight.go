package sandbox

import (
	"fmt"
	"os/exec"
	"strings"
)

// Check é o resultado de uma verificação de preflight.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

// Preflight verifica bwrap, user namespaces e o binário true. Retorna erro
// agregado se algo indispensável faltar.
func Preflight() ([]Check, error) {
	checks := []Check{}
	var problems []string

	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		checks = append(checks, Check{Name: "bwrap", OK: false, Detail: "não encontrado no PATH"})
		problems = append(problems, "bwrap não encontrado no PATH")
	} else {
		checks = append(checks, Check{Name: "bwrap", OK: true, Detail: bwrap})
	}

	truePath, err := exec.LookPath("true")
	if err != nil {
		truePath = "/bin/true"
	}

	if bwrap != "" {
		// O probe usa o mesmo caminho de criação de namespace da execução real.
		cmd := exec.Command(bwrap,
			"--ro-bind", "/", "/",
			"--proc", "/proc",
			"--dev", "/dev",
			"--", truePath,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = err.Error()
			}
			checks = append(checks, Check{Name: "user namespaces", OK: false, Detail: detail})
			problems = append(problems, "bwrap não conseguiu criar o sandbox (user namespaces desabilitados?)")
		} else {
			checks = append(checks, Check{Name: "user namespaces", OK: true, Detail: "ok"})
		}
	}

	if len(problems) > 0 {
		return checks, fmt.Errorf("%s; veja docs/asb.md (troubleshooting)", strings.Join(problems, "; "))
	}
	return checks, nil
}

// RequireBwrap falha cedo com mensagem clara se o backend não existir.
func RequireBwrap() error {
	_, err := Preflight()
	if err != nil {
		return fmt.Errorf("%w. Instale bubblewrap e habilite user namespaces (kernel.unprivileged_userns_clone=1 ou bwrap setuid)", err)
	}
	return nil
}
