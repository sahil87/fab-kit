package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
)

// TestRegisteredCommandsMatchLifecycleTable cross-checks the cobra
// registrations in rootCmd() against the shared internal.LifecycleCommands
// table in both directions, including each command's Short description. This
// is the check that catches a command added to fab-kit but not the table
// (which would silently mis-route through the shim to fab-go) and a help
// line drifting from the cobra Short (the migrations-status divergence that
// motivated the table).
func TestRegisteredCommandsMatchLifecycleTable(t *testing.T) {
	registered := make(map[string]string) // name → Short
	for _, c := range rootCmd().Commands() {
		registered[c.Name()] = c.Short
	}

	tableShorts := make(map[string]string, len(internal.LifecycleCommands))
	for _, lc := range internal.LifecycleCommands {
		tableShorts[lc.Name] = lc.Short
	}

	// Direction 1: every table entry is registered, with a matching Short.
	for name, wantShort := range tableShorts {
		gotShort, ok := registered[name]
		if !ok {
			t.Errorf("LifecycleCommands lists %q but rootCmd() does not register it", name)
			continue
		}
		if gotShort != wantShort {
			t.Errorf("command %q Short drifted:\n  cobra: %q\n  table: %q", name, gotShort, wantShort)
		}
	}

	// Direction 2: every registered command is in the table (a new fab-kit
	// command must be added to LifecycleCommands or the router mis-routes it).
	for name := range registered {
		if _, ok := tableShorts[name]; !ok {
			t.Errorf("rootCmd() registers %q but it is missing from LifecycleCommands (router would mis-route it to fab-go)", name)
		}
	}
}

// TestFabKitCommandsDerivedFromLifecycleTable pins the derived map to the
// table (no hand-maintained copy remains).
func TestFabKitCommandsDerivedFromLifecycleTable(t *testing.T) {
	if len(fabKitCommands) != len(internal.LifecycleCommands) {
		t.Fatalf("fabKitCommands has %d entries, LifecycleCommands has %d",
			len(fabKitCommands), len(internal.LifecycleCommands))
	}
	for _, c := range internal.LifecycleCommands {
		if !fabKitCommands[c.Name] {
			t.Errorf("fabKitCommands missing lifecycle command %q", c.Name)
		}
	}
}

func TestVersion(t *testing.T) {
	// The version variable should be set (defaults to "dev")
	if version == "" {
		t.Error("version should not be empty")
	}
}

// TestRootVersionFlagShape pins the shll version standard's verify checklist:
// `fab-kit --version` exits successfully with the version token on the first
// stdout line, shaped `fab-kit version vX.Y.Z` (no banner above it).
func TestRootVersionFlagShape(t *testing.T) {
	orig := version
	version = "2.16.5"
	defer func() { version = orig }()

	root := rootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("--version returned error (must exit 0): %v", err)
	}

	firstLine := strings.SplitN(out.String(), "\n", 2)[0]
	shape := regexp.MustCompile(`^fab-kit version v\d+(\.\d+)*$`)
	if !shape.MatchString(firstLine) {
		t.Errorf("--version first line = %q, want match for %s", firstLine, shape)
	}
}

// TestUpdateHelpMentionsSkipBrewUpdate pins the shll update standard's frozen
// textual contract: `update --help` output contains the literal substring
// `--skip-brew-update` (substring presence, not regex).
func TestUpdateHelpMentionsSkipBrewUpdate(t *testing.T) {
	root := rootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"update", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("update --help returned error: %v", err)
	}
	if !strings.Contains(out.String(), "--skip-brew-update") {
		t.Errorf("update --help must contain the literal %q (got: %q)", "--skip-brew-update", out.String())
	}
}

// TestUpdateNotBrewInstalledExitsZero pins the shll update standard's degrade
// clause: on a non-brew install, `fab-kit update` prints the guidance and
// exits 0 (RunE returns nil). Under `go test` the test binary's path never
// contains /Cellar/, so the real isBrewInstalled guard fires naturally —
// internal.Update returns ErrNotBrewInstalled (pinned separately by
// internal's TestUpdate_NotBrewInstalledReturnsSentinel, which guards the
// versionGuard contract) and updateCmd maps it to nil.
func TestUpdateNotBrewInstalledExitsZero(t *testing.T) {
	root := rootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"update", "--skip-brew-update"})

	if err := root.Execute(); err != nil {
		t.Errorf("update on a non-brew install must exit 0 (RunE nil), got: %v", err)
	}
}

func TestDisplayVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1.9.4", "v1.9.4"},
		{"v1.9.4", "v1.9.4"},
		{"dev", "dev"},
	}
	for _, tc := range cases {
		if got := displayVersion(tc.in); got != tc.want {
			t.Errorf("displayVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
