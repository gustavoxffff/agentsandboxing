package sandbox

import (
	"fmt"
	"os/exec"
	"strings"
)

// Check is the result of a preflight check.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

// Preflight checks bwrap, user namespaces and the true binary. It returns an
// aggregated error if something indispensable is missing.
func Preflight() ([]Check, error) {
	checks := []Check{}
	var problems []string

	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		checks = append(checks, Check{Name: "bwrap", OK: false, Detail: "not found in PATH"})
		problems = append(problems, "bwrap not found in PATH")
	} else {
		checks = append(checks, Check{Name: "bwrap", OK: true, Detail: bwrap})
	}

	truePath, err := exec.LookPath("true")
	if err != nil {
		truePath = "/bin/true"
	}

	if bwrap != "" {
		// The probe uses the same namespace creation path as the real execution.
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
			problems = append(problems, "bwrap could not create the sandbox (user namespaces disabled?)")
		} else {
			checks = append(checks, Check{Name: "user namespaces", OK: true, Detail: "ok"})
		}
	}

	if len(problems) > 0 {
		return checks, fmt.Errorf("%s; see docs/asb.md (troubleshooting)", strings.Join(problems, "; "))
	}
	return checks, nil
}

// RequireBwrap fails early with a clear message if the backend does not exist.
func RequireBwrap() error {
	_, err := Preflight()
	if err != nil {
		return fmt.Errorf("%w. Install bubblewrap and enable user namespaces (kernel.unprivileged_userns_clone=1 or bwrap setuid)", err)
	}
	return nil
}
