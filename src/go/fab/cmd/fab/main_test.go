package main

import (
	"fmt"
	"os"
	"path/filepath"
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
//
// TestMain also forces a deterministic rk-capability default: the pane-ready
// gate delegates to a sentinel-capable rk when one's on PATH, and dev machines
// have one — so without control, the cmd-layer ready tests would exercise the
// delegated arm here and the raw-tmux arm on an rk-less CI. A stub rk whose
// `mux await --help` lacks the `parked` discriminant is prepended to PATH,
// making every test default to the raw arm (the pane package's own TestMain
// precedent, one seam level down). A test that installs its own sentinel-
// capable stub sets FAB_TEST_RK_ARM=1 in its child env, which skips the
// prepend so its binary is not shadowed.
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
	rkDir := ""
	if os.Getenv("FAB_TEST_RK_ARM") != "1" {
		rkDir, err = os.MkdirTemp("", "fab-cmd-test-rk-")
		if err != nil {
			fmt.Fprintln(os.Stderr, "TestMain: creating rk stub dir:", err)
			os.Exit(1)
		}
		stub := "#!/bin/sh\n" +
			"if [ \"$1 $2 $3\" = \"mux await --help\" ]; then\n" +
			"printf 'a settled screen is classified ready\\n'\n" +
			"exit 0\n" +
			"fi\n" +
			"exit 1\n"
		if err := os.WriteFile(filepath.Join(rkDir, "rk"), []byte(stub), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "TestMain: writing rk stub:", err)
			os.Exit(1)
		}
		os.Setenv("PATH", rkDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	code := m.Run()
	os.RemoveAll(home)
	if rkDir != "" {
		os.RemoveAll(rkDir)
	}
	os.Exit(code)
}
