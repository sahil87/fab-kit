package worktree

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestBuildAvailableApps_OpenHereFirst(t *testing.T) {
	apps := BuildAvailableApps()
	if len(apps) == 0 {
		t.Fatal("BuildAvailableApps returned no apps")
	}
	first := apps[0]
	if first.Name != "Open here" || first.Cmd != "open_here" {
		t.Errorf("expected first app to be {\"Open here\", \"open_here\"}, got {%q, %q}", first.Name, first.Cmd)
	}
}

func TestDetectDefaultApp_SkipsOpenHere(t *testing.T) {
	apps := BuildAvailableApps()
	if len(apps) < 2 {
		t.Skip("need at least 2 apps to test fallback")
	}

	// Clear environment to force fallback path
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TMUX", "")
	t.Setenv("BYOBU_BACKEND", "")
	// Remove last-app cache to ensure fallback
	t.Setenv("HOME", t.TempDir())

	idx := DetectDefaultApp(apps)
	if idx > 0 && idx <= len(apps) {
		if apps[idx-1].Cmd == "open_here" {
			t.Errorf("DetectDefaultApp returned 'open_here' as default (index %d)", idx)
		}
	}
}

func TestOpenInApp_OpenHere(t *testing.T) {
	path := "/tmp/test-worktree"

	// Capture stdout
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		os.Stdout = origStdout
		_ = r.Close()
		_ = w.Close()
	})
	os.Stdout = w

	openErr := OpenInApp("open_here", path, "repo", "wt-name")

	w.Close()

	if openErr != nil {
		t.Fatalf("OpenInApp returned error: %v", openErr)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}

	expected := fmt.Sprintf("cd -- %q\n", path)
	if buf.String() != expected {
		t.Errorf("expected stdout %q, got %q", expected, buf.String())
	}
}
