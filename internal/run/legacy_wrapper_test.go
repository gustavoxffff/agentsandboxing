package run

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLegacyWrapperRemoved is a regression test for the drift recorded in
// RFC-0002 §2.2 and fixed for RFC-0001 acceptance: ADR-0001 removed the
// `opencode-sandbox` wrapper and its implicit `--profile host` fallback, but
// the script still existed and silently restored host networking. `asb run` is
// the only entry point and fails without an explicit profile.
func TestLegacyWrapperRemoved(t *testing.T) {
	wrapper := filepath.Join("..", "..", "opencode-sandbox")
	_, err := os.Stat(wrapper)
	switch {
	case err == nil:
		t.Fatalf("legacy wrapper %s exists: ADR-0001 removed it; there is no implicit --profile host", wrapper)
	case !os.IsNotExist(err):
		t.Fatalf("stat %s: %v", wrapper, err)
	}
}
