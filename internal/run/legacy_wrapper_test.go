package run

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLegacyWrapperRemoved is a regression test for the removed
// `opencode-sandbox` wrapper and its implicit `--profile host` fallback. The
// script once still existed and silently restored host networking. `asb run`
// is the only entry point and fails without an explicit profile.
func TestLegacyWrapperRemoved(t *testing.T) {
	wrapper := filepath.Join("..", "..", "opencode-sandbox")
	_, err := os.Stat(wrapper)
	switch {
	case err == nil:
		t.Fatalf("legacy wrapper %s exists; there is no implicit --profile host", wrapper)
	case !os.IsNotExist(err):
		t.Fatalf("stat %s: %v", wrapper, err)
	}
}
