package main

import (
	"bytes"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
)

func TestFabKitArgs(t *testing.T) {
	// Verify the allowlist contains exactly the fab-kit commands
	expected := []string{"init", "upgrade-repo", "sync", "update"}
	for _, cmd := range expected {
		if !fabKitArgs[cmd] {
			t.Errorf("expected fabKitArgs to contain %q", cmd)
		}
	}

	// Verify workflow commands are not in the allowlist (they go to fab-go)
	workflow := []string{"status", "preflight", "resolve", "log", "change", "score"}
	for _, cmd := range workflow {
		if fabKitArgs[cmd] {
			t.Errorf("expected fabKitArgs to NOT contain %q (belongs to fab-go)", cmd)
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
