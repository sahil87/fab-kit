package main

import (
	"bytes"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
)

func TestFabKitArgsDerivedFromLifecycleTable(t *testing.T) {
	// fabKitArgs must be exactly the LifecycleCommands name set — the router
	// allowlist has no hand-maintained copy of its own. (Real drift guards:
	// the registration cross-check in cmd/fab-kit, the _cli-fab.md router-line
	// contract test, and the fab module's collision test.)
	if len(fabKitArgs) != len(internal.LifecycleCommands) {
		t.Fatalf("fabKitArgs has %d entries, LifecycleCommands has %d",
			len(fabKitArgs), len(internal.LifecycleCommands))
	}
	for _, c := range internal.LifecycleCommands {
		if !fabKitArgs[c.Name] {
			t.Errorf("fabKitArgs missing lifecycle command %q", c.Name)
		}
	}
}

func TestVersion(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
}

func TestPrintVersion(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		var buf bytes.Buffer
		printVersion(&buf, "1.3.1", nil)
		got := buf.String()
		if got != "fab 1.3.1\n" {
			t.Errorf("expected %q, got %q", "fab 1.3.1\n", got)
		}
	})

	t.Run("matching versions", func(t *testing.T) {
		var buf bytes.Buffer
		printVersion(&buf, "1.3.1", &internal.ConfigResult{FabVersion: "1.3.1"})
		got := buf.String()
		want := "fab 1.3.1\nproject: 1.3.1\n"
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})

	t.Run("differing versions", func(t *testing.T) {
		var buf bytes.Buffer
		printVersion(&buf, "1.4.0", &internal.ConfigResult{FabVersion: "1.3.1"})
		got := buf.String()
		want := "fab 1.4.0\nproject: 1.3.1\n"
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})
}
