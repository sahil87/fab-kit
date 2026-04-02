package main

import (
	"testing"
)

func TestGitRepoRoot_ReturnsPath(t *testing.T) {
	// This test runs inside the fab-kit repo, so gitRepoRoot should succeed
	root, err := gitRepoRoot()
	if err != nil {
		t.Skipf("not in a git repo: %v", err)
	}
	if root == "" {
		t.Error("gitRepoRoot() returned empty string")
	}
}

func TestOperatorCmd_Structure(t *testing.T) {
	cmd := operatorCmd()
	if cmd.Use != "operator" {
		t.Errorf("Use = %q, want %q", cmd.Use, "operator")
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}
}
