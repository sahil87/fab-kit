package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeBrewScript is a portable /bin/sh stand-in for the `brew` executable. It
// appends each invocation's subcommand ($1) to the log file named by
// $FAB_BREW_LOG, and for `info` emits valid --json=v2 output so
// brewLatestVersion() parses a stable version (9.9.9) different from the
// currentVersion the test passes to Update — so the up-to-date short-circuit
// does not fire and `brew upgrade` is reached.
const fakeBrewScript = `#!/bin/sh
printf '%s\n' "$1" >> "$FAB_BREW_LOG"
if [ "$1" = "info" ]; then
  printf '%s' '{"formulae":[{"versions":{"stable":"9.9.9"}}]}'
fi
exit 0
`

func TestUpdateSkipBrewUpdateGating(t *testing.T) {
	tests := []struct {
		name           string
		skipBrewUpdate bool
		wantUpdate     bool // whether the brew log should contain "update"
	}{
		{name: "skip omits update but keeps upgrade", skipBrewUpdate: true, wantUpdate: false},
		{name: "default runs all three", skipBrewUpdate: false, wantUpdate: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Place a fake `brew` on PATH that logs subcommands.
			brewPath := filepath.Join(tmpDir, "brew")
			if err := os.WriteFile(brewPath, []byte(fakeBrewScript), 0755); err != nil {
				t.Fatalf("write fake brew: %v", err)
			}
			logPath := filepath.Join(tmpDir, "brew.log")
			t.Setenv("FAB_BREW_LOG", logPath)
			t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			// Override the brew-install guard so Update reaches the brew sequence.
			// Under `go test` the test binary's path never contains /Cellar/, so
			// the real guard would short-circuit before any brew call.
			orig := isBrewInstalled
			isBrewInstalled = func() bool { return true }
			defer func() { isBrewInstalled = orig }()

			if err := Update("1.0.0", tt.skipBrewUpdate); err != nil {
				t.Fatalf("Update returned error: %v", err)
			}

			data, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("read brew log: %v", err)
			}
			log := string(data)

			if got := strings.Contains(log, "update"); got != tt.wantUpdate {
				t.Errorf("brew log update-present = %v, want %v (log: %q)", got, tt.wantUpdate, log)
			}
			// info (version check) and upgrade must run in both cases.
			if !strings.Contains(log, "info") {
				t.Errorf("brew log must contain %q (log: %q)", "info", log)
			}
			if !strings.Contains(log, "upgrade") {
				t.Errorf("brew log must contain %q (log: %q)", "upgrade", log)
			}
		})
	}
}
