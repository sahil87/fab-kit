package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

func runLogs(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := dispatchLogsCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestDispatchLogs_PrintsAndTails(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	mustWrite(t, dispatch.LogPath(dir, "apply"), "line1\nline2\nline3\n")

	// Full log.
	out, err := runLogs(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if out != "line1\nline2\nline3\n" {
		t.Errorf("full log = %q", out)
	}

	// Tail 2.
	out, err = runLogs(t, "abcd", "apply", "--tail", "2")
	if err != nil {
		t.Fatalf("logs --tail: %v", err)
	}
	if out != "line2\nline3\n" {
		t.Errorf("tail 2 = %q", out)
	}
}

func TestDispatchLogs_MissingLogClearMessage(t *testing.T) {
	setupDispatchRepo(t, "sh -c 'exit 0'")
	_, err := runLogs(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error for a missing log")
	}
	if !strings.Contains(err.Error(), "no dispatch log") {
		t.Errorf("error = %q, want the clear no-log message", err.Error())
	}
}
