package worktree

import (
	"bytes"
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

	expected := "cd -- '" + path + "'\n"
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
	expected := "cd -- '" + path + "'\n"
	if stdoutBuf.String() != expected {
		t.Errorf("expected stdout %q, got %q", expected, stdoutBuf.String())
	}

	// stderr should NOT have the hint
	if stderrBuf.String() != "" {
		t.Errorf("expected no stderr output with WT_WRAPPER=1, got %q", stderrBuf.String())
	}
}

func TestBuildAvailableApps_TmuxSession_InTmux(t *testing.T) {
	// Simulate a plain tmux session
	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	t.Setenv("BYOBU_TTY", "")
	t.Setenv("BYOBU_BACKEND", "")
	t.Setenv("BYOBU_SESSION", "")
	t.Setenv("BYOBU_CONFIG_DIR", "")

	apps := BuildAvailableApps()

	found := false
	for _, app := range apps {
		if app.Cmd == "tmux_session" {
			found = true
			if app.Name != "tmux session" {
				t.Errorf("expected display name %q, got %q", "tmux session", app.Name)
			}
			break
		}
	}
	if !found {
		t.Error("expected tmux_session in BuildAvailableApps when IsTmuxSession() is true")
	}
}

func TestBuildAvailableApps_TmuxSession_AfterTmuxWindow(t *testing.T) {
	// Simulate a plain tmux session
	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	t.Setenv("BYOBU_TTY", "")
	t.Setenv("BYOBU_BACKEND", "")
	t.Setenv("BYOBU_SESSION", "")
	t.Setenv("BYOBU_CONFIG_DIR", "")

	apps := BuildAvailableApps()

	windowIdx := -1
	sessionIdx := -1
	for i, app := range apps {
		if app.Cmd == "tmux_window" {
			windowIdx = i
		}
		if app.Cmd == "tmux_session" {
			sessionIdx = i
		}
	}

	if windowIdx == -1 {
		t.Fatal("tmux_window not found in apps")
	}
	if sessionIdx == -1 {
		t.Fatal("tmux_session not found in apps")
	}
	if sessionIdx != windowIdx+1 {
		t.Errorf("expected tmux_session (index %d) immediately after tmux_window (index %d)", sessionIdx, windowIdx)
	}
}

func TestBuildAvailableApps_TmuxSession_AbsentOutsideTmux(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("BYOBU_TTY", "")
	t.Setenv("BYOBU_BACKEND", "")
	t.Setenv("BYOBU_SESSION", "")
	t.Setenv("BYOBU_CONFIG_DIR", "")

	apps := BuildAvailableApps()

	for _, app := range apps {
		if app.Cmd == "tmux_session" {
			t.Error("tmux_session should not appear when not in a tmux session")
		}
	}
}

func TestBuildAvailableApps_TmuxSession_AbsentInByobu(t *testing.T) {
	// Simulate a byobu session
	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	t.Setenv("BYOBU_BACKEND", "tmux")

	apps := BuildAvailableApps()

	for _, app := range apps {
		if app.Cmd == "tmux_session" {
			t.Error("tmux_session should not appear in a byobu session")
		}
	}
}

func TestResolveApp_TmuxSession_ByCmd(t *testing.T) {
	apps := []AppInfo{
		{"Open here", "open_here"},
		{"tmux window", "tmux_window"},
		{"tmux session", "tmux_session"},
	}

	resolved, err := ResolveApp("tmux_session", apps)
	if err != nil {
		t.Fatalf("ResolveApp returned error: %v", err)
	}
	if resolved.Cmd != "tmux_session" {
		t.Errorf("expected Cmd %q, got %q", "tmux_session", resolved.Cmd)
	}
	if resolved.Name != "tmux session" {
		t.Errorf("expected Name %q, got %q", "tmux session", resolved.Name)
	}
}

func TestResolveApp_TmuxSession_ByDisplayName(t *testing.T) {
	apps := []AppInfo{
		{"Open here", "open_here"},
		{"tmux window", "tmux_window"},
		{"tmux session", "tmux_session"},
	}

	resolved, err := ResolveApp("tmux session", apps)
	if err != nil {
		t.Fatalf("ResolveApp returned error: %v", err)
	}
	if resolved.Cmd != "tmux_session" {
		t.Errorf("expected Cmd %q, got %q", "tmux_session", resolved.Cmd)
	}
}

func TestResolveApp_TmuxSession_ByDisplayNameCaseInsensitive(t *testing.T) {
	apps := []AppInfo{
		{"Open here", "open_here"},
		{"tmux window", "tmux_window"},
		{"tmux session", "tmux_session"},
	}

	resolved, err := ResolveApp("Tmux Session", apps)
	if err != nil {
		t.Fatalf("ResolveApp returned error: %v", err)
	}
	if resolved.Cmd != "tmux_session" {
		t.Errorf("expected Cmd %q, got %q", "tmux_session", resolved.Cmd)
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
	expected := "cd -- '" + path + "'\n"
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

func TestOpenInApp_OpenHere_ShellSafeQuoting(t *testing.T) {
	t.Setenv("WT_WRAPPER", "1")

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"dollar sign", "/tmp/$(whoami)", "cd -- '/tmp/$(whoami)'\n"},
		{"backticks", "/tmp/`id`", "cd -- '/tmp/`id`'\n"},
		{"single quote", "/tmp/it's-here", "cd -- '/tmp/it'\\''s-here'\n"},
		{"spaces", "/tmp/my worktree", "cd -- '/tmp/my worktree'\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			openErr := OpenInApp("open_here", tt.path, "repo", "wt")
			w.Close()

			if openErr != nil {
				t.Fatalf("OpenInApp returned error: %v", openErr)
			}

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("io.Copy: %v", err)
			}

			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}
