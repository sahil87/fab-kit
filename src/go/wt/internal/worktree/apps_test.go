package worktree

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
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

	// Set WT_WRAPPER=1 so the hint is suppressed (original test behavior)
	t.Setenv("WT_WRAPPER", "1")

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

func TestOpenInApp_OpenHere_WithWrapper(t *testing.T) {
	path := "/tmp/test-worktree"

	t.Setenv("WT_WRAPPER", "1")

	// Capture stdout
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}

	// Capture stderr
	origStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}

	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rOut.Close()
		_ = wOut.Close()
		_ = rErr.Close()
		_ = wErr.Close()
	})
	os.Stdout = wOut
	os.Stderr = wErr

	openErr := OpenInApp("open_here", path, "repo", "wt-name")

	wOut.Close()
	wErr.Close()

	if openErr != nil {
		t.Fatalf("OpenInApp returned error: %v", openErr)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := io.Copy(&stdoutBuf, rOut); err != nil {
		t.Fatalf("io.Copy stdout: %v", err)
	}
	if _, err := io.Copy(&stderrBuf, rErr); err != nil {
		t.Fatalf("io.Copy stderr: %v", err)
	}

	// stdout should have the cd command
	expected := fmt.Sprintf("cd -- %q\n", path)
	if stdoutBuf.String() != expected {
		t.Errorf("expected stdout %q, got %q", expected, stdoutBuf.String())
	}

	// stderr should NOT have the hint
	if stderrBuf.String() != "" {
		t.Errorf("expected no stderr output with WT_WRAPPER=1, got %q", stderrBuf.String())
	}
}

func TestOpenInApp_OpenHere_WithoutWrapper(t *testing.T) {
	path := "/tmp/test-worktree"

	// Ensure WT_WRAPPER is not set
	t.Setenv("WT_WRAPPER", "")

	// Capture stdout
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}

	// Capture stderr
	origStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}

	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rOut.Close()
		_ = wOut.Close()
		_ = rErr.Close()
		_ = wErr.Close()
	})
	os.Stdout = wOut
	os.Stderr = wErr

	openErr := OpenInApp("open_here", path, "repo", "wt-name")

	wOut.Close()
	wErr.Close()

	if openErr != nil {
		t.Fatalf("OpenInApp returned error: %v", openErr)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := io.Copy(&stdoutBuf, rOut); err != nil {
		t.Fatalf("io.Copy stdout: %v", err)
	}
	if _, err := io.Copy(&stderrBuf, rErr); err != nil {
		t.Fatalf("io.Copy stderr: %v", err)
	}

	// stdout should still have the cd command
	expected := fmt.Sprintf("cd -- %q\n", path)
	if stdoutBuf.String() != expected {
		t.Errorf("expected stdout %q, got %q", expected, stdoutBuf.String())
	}

	// stderr should have the hint
	stderrStr := stderrBuf.String()
	if !strings.Contains(stderrStr, `hint: "Open here" requires the shell wrapper`) {
		t.Errorf("expected stderr to contain hint, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, `eval "$(wt shell-setup)"`) {
		t.Errorf("expected stderr to contain eval instruction, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, `Add it to your ~/.zshrc or ~/.bashrc`) {
		t.Errorf("expected stderr to contain profile hint, got %q", stderrStr)
	}
}
