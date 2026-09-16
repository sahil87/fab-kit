package main

import (
	"bytes"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/kitpath"
)

// fab kit-path prints the resolved kit directory as one newline-terminated
// line (8nd2) — chained callers must never see the path fused with the next
// command's output.
func TestKitPathCmd_PrintsNewlineTerminatedLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(kitpath.KitPathEnv, dir)

	cmd := kitPathCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("kit-path: %v", err)
	}
	if got, want := buf.String(), dir+"\n"; got != want {
		t.Fatalf("output = %q, want %q (exactly one trailing newline)", got, want)
	}
}
