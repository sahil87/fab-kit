package main

import (
	"os"
	"testing"
)

// chdirTestEnv isolates env vars and cwd for a test and restores them on
// t.Cleanup. Shared across command tests that need a controlled working
// directory (for resolve.FabRoot) and/or scrubbed TMUX/TMUX_PANE env.
func chdirTestEnv(t *testing.T, cwd string, envOverrides map[string]string) {
	t.Helper()
	origEnv := map[string]string{}
	for k := range envOverrides {
		origEnv[k] = os.Getenv(k)
	}
	t.Cleanup(func() {
		for k, v := range origEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})
	for k, v := range envOverrides {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
	// Chdir for resolve.FabRoot()
	origWd, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })
}
