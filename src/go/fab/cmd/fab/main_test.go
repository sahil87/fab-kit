package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// TestMain isolates the external config layers for the whole cmd/fab test package
// before running any test. The cascade makes internal/config.LoadPath merge
// environment overrides and ~/.fab-kit/config.yaml (via os.UserHomeDir, which
// honors $HOME on unix) into every config read, so a developer's real preferences
// would perturb the exact-byte assertions in
// resolve_agent_test.go, agent_test.go, batch_*_test.go, and dispatch_start_test.go.
// Pointing HOME at a fresh empty temp dir for the package makes those resolved-
// command tests see no system layer; clearing every registry-derived variable does
// the same for the environment layer. Individual tests opt back in with t.Setenv.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "fab-cmd-test-home-")
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
