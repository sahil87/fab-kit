package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestRoot builds a minimal `fab` root with shellInitCmd registered. This
// mirrors the production wiring in main.go without pulling in every
// subcommand (which would require their own filesystem/runtime setup).
func newTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "fab",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(shellInitCmd())
	return root
}

// runShellInit executes the test root with the given args and returns
// stdout/stderr buffers plus the resulting error.
func runShellInit(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := newTestRoot()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"shell-init"}, args...))
	err := root.Execute()
	return out.String(), errBuf.String(), err
}

func TestShellInit_Bash_NonEmpty(t *testing.T) {
	stdout, _, err := runShellInit(t, "bash")
	if err != nil {
		t.Fatalf("shell-init bash: unexpected error: %v", err)
	}
	if stdout == "" {
		t.Error("shell-init bash: expected non-empty output")
	}
}

func TestShellInit_Zsh_StartsWithCompdef(t *testing.T) {
	stdout, _, err := runShellInit(t, "zsh")
	if err != nil {
		t.Fatalf("shell-init zsh: unexpected error: %v", err)
	}
	if stdout == "" {
		t.Fatal("shell-init zsh: expected non-empty output")
	}
	if !strings.HasPrefix(stdout, "#compdef fab") {
		// Allow a leading comment line — but the canonical Cobra zsh
		// preamble begins with `#compdef <name>`.
		t.Errorf("shell-init zsh: expected output to start with %q, got first line %q",
			"#compdef fab", firstLine(stdout))
	}
}

func TestShellInit_Fish_NonEmpty(t *testing.T) {
	stdout, _, err := runShellInit(t, "fish")
	if err != nil {
		t.Fatalf("shell-init fish: unexpected error: %v", err)
	}
	if stdout == "" {
		t.Error("shell-init fish: expected non-empty output")
	}
}

func TestShellInit_UnknownShell_Errors(t *testing.T) {
	_, _, err := runShellInit(t, "powershell")
	if err == nil {
		t.Error("shell-init powershell: expected an error, got nil")
	}
}

func TestShellInit_MissingArg_Errors(t *testing.T) {
	_, _, err := runShellInit(t)
	if err == nil {
		t.Error("shell-init (no arg): expected an error, got nil")
	}
}

func TestShellInit_TooManyArgs_Errors(t *testing.T) {
	_, _, err := runShellInit(t, "zsh", "extra")
	if err == nil {
		t.Error("shell-init zsh extra: expected an error, got nil")
	}
}

// firstLine returns the first line of s (without the trailing newline) for
// readable assertion messages.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
