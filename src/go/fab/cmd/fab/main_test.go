package main

import (
	"fmt"
	"os"
	"testing"
)

// TestMain isolates $HOME for the whole cmd/fab test package before running any
// test. The lpb5 cascade made internal/config.LoadPath merge
// ~/.fab-kit/config.yaml (via os.UserHomeDir, which honors $HOME on unix) into
// every config read, so a developer's real system config with an agent.tiers /
// providers override would perturb the exact-byte assertions in
// resolve_agent_test.go, agent_test.go, batch_*_test.go, and dispatch_start_test.go.
// Pointing HOME at a fresh empty temp dir for the package makes those resolved-
// command tests see only the project config. Individual tests that need to WRITE
// a system config still set their own HOME (t.Setenv, restored on cleanup), which
// wins over this package default. Only providers/agent are ScopeBoth; every other
// key is pruned from the system layer, so this is the complete isolation surface.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "fab-cmd-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: creating temp HOME:", err)
		os.Exit(1)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
