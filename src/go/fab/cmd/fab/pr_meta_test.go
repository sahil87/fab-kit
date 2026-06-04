package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPrMetaCmd_RegisteredWithExpectedUse(t *testing.T) {
	cmd := prMetaCmd()
	if !strings.HasPrefix(cmd.Use, "pr-meta ") {
		t.Errorf("prMetaCmd Use = %q, want prefix \"pr-meta \"", cmd.Use)
	}
	if cmd.Flags().Lookup("type") == nil {
		t.Error("prMetaCmd missing --type flag")
	}
	if cmd.Flags().Lookup("issues") == nil {
		t.Error("prMetaCmd missing --issues flag")
	}
}

func TestPrMetaCmd_TypeRequired(t *testing.T) {
	cmd := prMetaCmd()
	cmd.SetArgs([]string{"some-change"}) // no --type
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --type is omitted")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Errorf("error should mention the required type flag, got: %v", err)
	}
}

func TestPrMetaCmd_NoFabContextExitsNonZero(t *testing.T) {
	// Run from a temp dir that contains no fab/ ancestor → FabRoot fails or the
	// change cannot be resolved. Either way the command must error (non-zero
	// exit) and print nothing to stdout.
	tmp := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	cmd := prMetaCmd()
	// Mirror the production root (main.go sets SilenceUsage) so the assertion
	// targets command stdout, not cobra's usage echo on RunE error.
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"nonexistent", "--type", "feat"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected non-zero exit when there is no fab context")
	}
	if out.Len() != 0 {
		t.Errorf("expected empty stdout on no-fab-context error, got %q", out.String())
	}
}
