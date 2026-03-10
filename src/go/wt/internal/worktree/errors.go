package worktree

import (
	"fmt"
	"os"
)

// Exit codes matching the bash wt scripts.
const (
	ExitSuccess          = 0
	ExitGeneralError     = 1
	ExitInvalidArgs      = 2
	ExitGitError         = 3
	ExitRetryExhausted   = 4
	ExitByobuTabError    = 5
	ExitTmuxWindowError  = 6
)

// ANSI color codes, disabled when NO_COLOR is set.
var (
	ColorRed   = "\033[0;31m"
	ColorYellow = "\033[0;33m"
	ColorGreen = "\033[0;32m"
	ColorBold  = "\033[1m"
	ColorReset = "\033[0m"
)

func init() {
	if os.Getenv("NO_COLOR") != "" {
		ColorRed = ""
		ColorYellow = ""
		ColorGreen = ""
		ColorBold = ""
		ColorReset = ""
	}
}

// WtError formats a structured error message and writes it to stderr.
// Format: "Error: {what}\n  Why: {why}\n  Fix: {fix}"
func WtError(what, why, fix string) string {
	msg := fmt.Sprintf("%sError:%s %s\n  %sWhy:%s %s",
		ColorRed, ColorReset, what,
		ColorBold, ColorReset, why)
	if fix != "" {
		msg += fmt.Sprintf("\n  %sFix:%s %s", ColorBold, ColorReset, fix)
	}
	return msg
}

// PrintError writes a structured error to stderr.
func PrintError(what, why, fix string) {
	fmt.Fprintln(os.Stderr, WtError(what, why, fix))
}

// ExitWithError prints a structured error and exits with the given code.
func ExitWithError(code int, what, why, fix string) {
	PrintError(what, why, fix)
	os.Exit(code)
}
