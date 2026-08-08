package spawn

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// TestMain isolates every external config layer for the whole internal/spawn test
// package. spawn.Command resolves providers/agent through config.LoadPath, whose
// cascade merges registry-derived environment overrides and
// ~/.fab-kit/config.yaml into every read. Tests that need either layer opt back in
// explicitly.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "fab-spawn-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: creating temp HOME:", err)
		os.Exit(1)
	}
	os.Setenv("HOME", home)
	for _, key := range configscope.DottedKeys() {
		os.Unsetenv("FAB_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_")))
	}
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
