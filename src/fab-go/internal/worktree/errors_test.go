package worktree

import (
	"strings"
	"testing"
)

func TestWtError_Format(t *testing.T) {
	// Save original colors
	origRed := ColorRed
	origBold := ColorBold
	origReset := ColorReset
	defer func() {
		ColorRed = origRed
		ColorBold = origBold
		ColorReset = origReset
	}()

	// Disable colors for testing
	ColorRed = ""
	ColorBold = ""
	ColorReset = ""

	msg := WtError("Something failed", "Because of X", "Do Y instead")

	if !strings.Contains(msg, "Error: Something failed") {
		t.Errorf("error message should contain 'Error: Something failed', got: %s", msg)
	}
	if !strings.Contains(msg, "Why: Because of X") {
		t.Errorf("error message should contain 'Why: Because of X', got: %s", msg)
	}
	if !strings.Contains(msg, "Fix: Do Y instead") {
		t.Errorf("error message should contain 'Fix: Do Y instead', got: %s", msg)
	}
}

func TestWtError_WithoutFix(t *testing.T) {
	origRed := ColorRed
	origBold := ColorBold
	origReset := ColorReset
	defer func() {
		ColorRed = origRed
		ColorBold = origBold
		ColorReset = origReset
	}()

	ColorRed = ""
	ColorBold = ""
	ColorReset = ""

	msg := WtError("Something failed", "Because of X", "")

	if strings.Contains(msg, "Fix:") {
		t.Errorf("error message should not contain 'Fix:' when fix is empty, got: %s", msg)
	}
}

func TestWtError_NoColor(t *testing.T) {
	// Set NO_COLOR and manually apply
	origRed := ColorRed
	origBold := ColorBold
	origReset := ColorReset
	defer func() {
		ColorRed = origRed
		ColorBold = origBold
		ColorReset = origReset
	}()

	ColorRed = ""
	ColorBold = ""
	ColorReset = ""

	msg := WtError("Test", "Why", "Fix")

	// Should not contain ANSI codes
	if strings.Contains(msg, "\033[") {
		t.Errorf("NO_COLOR: message should not contain ANSI codes, got: %s", msg)
	}
}

func TestWtError_WithColor(t *testing.T) {
	origRed := ColorRed
	origBold := ColorBold
	origReset := ColorReset
	defer func() {
		ColorRed = origRed
		ColorBold = origBold
		ColorReset = origReset
	}()

	ColorRed = "\033[0;31m"
	ColorBold = "\033[1m"
	ColorReset = "\033[0m"

	msg := WtError("Test", "Why", "Fix")

	if !strings.Contains(msg, "\033[0;31m") {
		t.Errorf("color: message should contain ANSI red code, got: %s", msg)
	}
	if !strings.Contains(msg, "\033[1m") {
		t.Errorf("color: message should contain ANSI bold code, got: %s", msg)
	}
}

func TestExitCodes(t *testing.T) {
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess = %d, want 0", ExitSuccess)
	}
	if ExitGeneralError != 1 {
		t.Errorf("ExitGeneralError = %d, want 1", ExitGeneralError)
	}
	if ExitInvalidArgs != 2 {
		t.Errorf("ExitInvalidArgs = %d, want 2", ExitInvalidArgs)
	}
	if ExitGitError != 3 {
		t.Errorf("ExitGitError = %d, want 3", ExitGitError)
	}
	if ExitRetryExhausted != 4 {
		t.Errorf("ExitRetryExhausted = %d, want 4", ExitRetryExhausted)
	}
	if ExitByobuTabError != 5 {
		t.Errorf("ExitByobuTabError = %d, want 5", ExitByobuTabError)
	}
	if ExitTmuxWindowError != 6 {
		t.Errorf("ExitTmuxWindowError = %d, want 6", ExitTmuxWindowError)
	}
}
