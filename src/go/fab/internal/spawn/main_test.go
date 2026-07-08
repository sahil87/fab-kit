package spawn

import (
	"fmt"
	"os"
	"testing"
)

// TestMain isolates $HOME for the whole internal/spawn test package. spawn.Command
// resolves providers/agent.tiers through internal/config.LoadPath, whose lpb5
// cascade merges ~/.fab-kit/config.yaml (via os.UserHomeDir) into every read. A
// developer's real system config with a providers/agent.tiers override would
// perturb the resolved-command assertions here. Pointing HOME at a fresh empty
// temp dir makes those tests see only the project config.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "fab-spawn-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: creating temp HOME:", err)
		os.Exit(1)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
