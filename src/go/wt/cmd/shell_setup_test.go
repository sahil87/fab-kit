package main

import (
	"testing"
)

func TestShellSetup_OutputsWrapperFunction(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, []string{"SHELL=/bin/zsh"}, "shell-setup")
	assertExitCode(t, r, 0)

	// Verify the wrapper function is present in stdout
	assertContains(t, r.Stdout, "wt() {")
	assertContains(t, r.Stdout, `command wt "$@"`)
	assertContains(t, r.Stdout, `eval "$last"`)
	assertContains(t, r.Stdout, "export WT_WRAPPER=1")

	// Verify the full output matches the expected wrapper
	if r.Stdout != ShellWrapperFunc {
		t.Errorf("stdout does not match expected wrapper function.\nExpected:\n%s\nGot:\n%s", ShellWrapperFunc, r.Stdout)
	}

	// No stderr for recognized shell
	if r.Stderr != "" {
		t.Errorf("expected no stderr for zsh, got %q", r.Stderr)
	}
}

func TestShellSetup_BashShell(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, []string{"SHELL=/bin/bash"}, "shell-setup")
	assertExitCode(t, r, 0)

	if r.Stdout != ShellWrapperFunc {
		t.Errorf("stdout does not match expected wrapper function.\nExpected:\n%s\nGot:\n%s", ShellWrapperFunc, r.Stdout)
	}

	if r.Stderr != "" {
		t.Errorf("expected no stderr for bash, got %q", r.Stderr)
	}
}

func TestShellSetup_EmptyShell(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, []string{"SHELL="}, "shell-setup")
	assertExitCode(t, r, 0)

	if r.Stdout != ShellWrapperFunc {
		t.Errorf("stdout does not match expected wrapper function.\nExpected:\n%s\nGot:\n%s", ShellWrapperFunc, r.Stdout)
	}

	// No warning for empty SHELL
	if r.Stderr != "" {
		t.Errorf("expected no stderr for empty SHELL, got %q", r.Stderr)
	}
}

func TestShellSetup_UnsupportedShell(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, []string{"SHELL=/usr/bin/fish"}, "shell-setup")
	assertExitCode(t, r, 0)

	// Still outputs the bash/zsh wrapper
	if r.Stdout != ShellWrapperFunc {
		t.Errorf("stdout does not match expected wrapper function.\nExpected:\n%s\nGot:\n%s", ShellWrapperFunc, r.Stdout)
	}

	// Warning on stderr
	assertContains(t, r.Stderr, `warning: unsupported shell "fish"`)
	assertContains(t, r.Stderr, "outputting bash/zsh wrapper")
}
