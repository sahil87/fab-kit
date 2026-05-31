package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeBrewOnPath installs a fake `brew` executable on PATH that appends each
// invocation's first argument (the brew subcommand) to a log file, then writes
// a minimal valid `brew info --json=v2` payload to stdout. It returns the path
// to the log file. The fake lets us assert which brew subcommands Update ran
// without touching a real Homebrew install.
func fakeBrewOnPath(t *testing.T, stableVersion string) string {
	t.Helper()

	binDir := t.TempDir()
	logFile := filepath.Join(binDir, "brew-calls.log")

	script := "#!/bin/sh\n" +
		"echo \"$1\" >> \"" + logFile + "\"\n" +
		"if [ \"$1\" = \"info\" ]; then\n" +
		"  printf '%s' '{\"formulae\":[{\"versions\":{\"stable\":\"" + stableVersion + "\"}}]}'\n" +
		"fi\n" +
		"exit 0\n"

	brewPath := filepath.Join(binDir, "brew")
	if err := os.WriteFile(brewPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake brew: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logFile
}

// brewCalls returns the recorded brew subcommands (one per line). Missing log
// file means no brew invocations were made.
func brewCalls(t *testing.T, logFile string) []string {
	t.Helper()
	data, err := os.ReadFile(logFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read brew log: %v", err)
	}
	var calls []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			calls = append(calls, line)
		}
	}
	return calls
}

func contains(calls []string, sub string) bool {
	for _, c := range calls {
		if c == sub {
			return true
		}
	}
	return false
}

// stubBrewSeams overrides the package-level seams so Update believes fab-kit was
// brew-installed and that the latest stable version is latest. It restores the
// originals via t.Cleanup.
func stubBrewSeams(t *testing.T, latest string) {
	t.Helper()
	origInstalled, origLatest := isBrewInstalledFn, brewLatestVersionFn
	isBrewInstalledFn = func() bool { return true }
	brewLatestVersionFn = func() (string, error) { return latest, nil }
	t.Cleanup(func() {
		isBrewInstalledFn = origInstalled
		brewLatestVersionFn = origLatest
	})
}

// TestUpdate_SkipBrewUpdate_SkipsRefreshButUpgrades verifies the cross-toolkit
// contract: with skipBrewUpdate=true, the internal `brew update` is NOT run,
// while the version check and `brew upgrade` still execute.
func TestUpdate_SkipBrewUpdate_SkipsRefreshButUpgrades(t *testing.T) {
	logFile := fakeBrewOnPath(t, "2.0.0")
	stubBrewSeams(t, "2.0.0")

	if err := Update("1.0.0", true); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	calls := brewCalls(t, logFile)

	if contains(calls, "update") {
		t.Errorf("brew update should NOT be invoked with --skip-brew-update; calls=%v", calls)
	}
	if !contains(calls, "upgrade") {
		t.Errorf("brew upgrade should still run with --skip-brew-update; calls=%v", calls)
	}
}

// TestUpdate_Default_RunsBrewUpdate verifies default behavior (flag absent) is
// preserved: `brew update` runs, followed by `brew upgrade`.
func TestUpdate_Default_RunsBrewUpdate(t *testing.T) {
	logFile := fakeBrewOnPath(t, "2.0.0")
	stubBrewSeams(t, "2.0.0")

	if err := Update("1.0.0", false); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	calls := brewCalls(t, logFile)

	if !contains(calls, "update") {
		t.Errorf("brew update should run by default; calls=%v", calls)
	}
	if !contains(calls, "upgrade") {
		t.Errorf("brew upgrade should run by default; calls=%v", calls)
	}
}

// TestUpdate_SkipBrewUpdate_UpToDateShortCircuits verifies the version check and
// the up-to-date short-circuit still run when the refresh is skipped: when
// latest == current, Update returns without invoking `brew upgrade`.
func TestUpdate_SkipBrewUpdate_UpToDateShortCircuits(t *testing.T) {
	logFile := fakeBrewOnPath(t, "1.0.0")
	stubBrewSeams(t, "1.0.0")

	if err := Update("1.0.0", true); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	calls := brewCalls(t, logFile)

	if contains(calls, "update") {
		t.Errorf("brew update should NOT be invoked with --skip-brew-update; calls=%v", calls)
	}
	if contains(calls, "upgrade") {
		t.Errorf("brew upgrade should NOT run when already up to date; calls=%v", calls)
	}
}
