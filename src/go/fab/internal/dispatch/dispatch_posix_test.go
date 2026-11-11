//go:build !windows

package dispatch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// launchWrappedWorker runs the production headless launch (WrapperArgv through
// Launch, i.e. setsid) under shellPath with a hermetic HOME whose interactive rc
// sleeps for rcDelay, so a test can act while the interactive shell is still
// starting. Returns the state dir and the recorded pgid.
func launchWrappedWorker(t *testing.T, shellPath, rcDelay, cmd string, timeout int) (dir string, pgid int) {
	t.Helper()
	dir = t.TempDir()
	rc := "sleep " + rcDelay + "\n"
	for _, name := range []string{".bashrc", ".zshrc", ".shrc"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(rc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", dir)
	t.Setenv("ZDOTDIR", dir)
	t.Setenv("ENV", filepath.Join(dir, ".shrc"))
	t.Setenv("SHELL", shellPath)
	prompt := filepath.Join(dir, "p.md")
	if err := os.WriteFile(prompt, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := WrapperArgv(cmd, prompt, filepath.Join(dir, "l.log"), filepath.Join(dir, "e.exit"), timeout)
	_, pgid, err := Launch(argv, dir)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	return dir, pgid
}

func wrapShells() map[string]string {
	shells := map[string]string{"sh": "/bin/sh"}
	for _, name := range []string{"bash", "zsh"} {
		if path, err := exec.LookPath(name); err == nil {
			shells[name] = path
		}
	}
	return shells
}

// TestKillGroupStopsAnRcLoadingWorker pins KillGroup against the
// interactive-shell launch: interactive shells ignore SIGTERM, so a kill that
// lands while the worker's shell is still loading its rc must not let the shell
// go on to start the provider afterwards.
func TestKillGroupStopsAnRcLoadingWorker(t *testing.T) {
	for name, shellPath := range wrapShells() {
		t.Run(name, func(t *testing.T) {
			dir, pgid := launchWrappedWorker(t, shellPath, "1", "touch started; sleep 30", 0)
			time.Sleep(300 * time.Millisecond) // the rc is still sleeping
			if err := KillGroup(pgid); err != nil {
				t.Fatalf("KillGroup: %v", err)
			}
			time.Sleep(1500 * time.Millisecond) // past the rc delay
			if _, err := os.Stat(filepath.Join(dir, "started")); err == nil {
				t.Error("the provider started after the kill — the interactive shell survived it")
			}
		})
	}
}

// TestWrapperArgvTimeoutLeavesNoSurvivor pins R4's timeout clause on the real
// setsid launch: `timeout` kills the wrapped provider (exit 124) and nothing it
// started outlives it.
func TestWrapperArgvTimeoutLeavesNoSurvivor(t *testing.T) {
	if _, err := exec.LookPath("timeout"); err != nil {
		t.Skip("timeout not available")
	}
	for name, shellPath := range wrapShells() {
		t.Run(name, func(t *testing.T) {
			dir, _ := launchWrappedWorker(t, shellPath, "0", "sleep 2; touch late", 1)
			time.Sleep(3 * time.Second)
			if _, err := os.Stat(filepath.Join(dir, "late")); err == nil {
				t.Error("the provider outlived its timeout")
			}
			exit, _ := os.ReadFile(filepath.Join(dir, "e.exit"))
			if got := strings.TrimSpace(string(exit)); got != "124" {
				t.Errorf("exit = %q, want 124", got)
			}
		})
	}
}
