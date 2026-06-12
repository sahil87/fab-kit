package main

import (
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
