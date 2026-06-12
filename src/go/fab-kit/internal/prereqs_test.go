package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequiredToolsUpdated(t *testing.T) {
	// Verify jq and gh are not in the required tools list
	for _, tool := range requiredTools {
		if tool == "jq" {
			t.Error("jq should not be in requiredTools (removed: was used by old shell-based hook sync)")
		}
		if tool == "gh" {
			t.Error("gh should not be in requiredTools (removed: only needed by download)")
		}
	}

	// Verify expected tools are present
	expected := map[string]bool{"git": false, "bash": false, "yq": false, "direnv": false}
	for _, tool := range requiredTools {
		expected[tool] = true
	}
	for tool, found := range expected {
		if !found {
			t.Errorf("expected %s in requiredTools", tool)
		}
	}
}

// shimPath builds a PATH-only shim directory containing all required tools,
// with yq reporting the given --version output, and points PATH at it
// exclusively for the test.
func shimPath(t *testing.T, yqVersionOutput string) {
	t.Helper()
	dir := t.TempDir()
	noop := "#!/bin/sh\nexit 0\n"
	for _, tool := range []string{"git", "bash", "direnv"} {
		if err := os.WriteFile(filepath.Join(dir, tool), []byte(noop), 0755); err != nil {
			t.Fatal(err)
		}
	}
	yq := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' '%s'\nexit 0\n", yqVersionOutput)
	if err := os.WriteFile(filepath.Join(dir, "yq"), []byte(yq), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestCheckPrerequisites_MissingToolsListed(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir — nothing on PATH

	err := checkPrerequisites()
	if err == nil {
		t.Fatal("expected error when no required tool is on PATH")
	}
	for _, tool := range requiredTools {
		if !strings.Contains(err.Error(), tool) {
			t.Errorf("expected %s in missing-tools error, got: %v", tool, err)
		}
	}
}

func TestCheckPrerequisites_YqV4Passes(t *testing.T) {
	shimPath(t, "yq (https://github.com/mikefarah/yq/) version v4.44.1")
	if err := checkPrerequisites(); err != nil {
		t.Errorf("yq v4 should pass: %v", err)
	}
}

func TestCheckPrerequisites_YqV3Rejected(t *testing.T) {
	shimPath(t, "yq version 3.4.1")
	err := checkPrerequisites()
	if err == nil {
		t.Fatal("expected yq v3 to be rejected")
	}
	if !strings.Contains(err.Error(), "yq version 4+") {
		t.Errorf("expected yq version error, got: %v", err)
	}
}

// Regression (260612-tb6f, F44): the major-version check used a lexicographic
// string compare (`major < "4"`), which misordered multi-digit majors —
// a hypothetical yq v10 was rejected ("10" < "4" as strings).
func TestCheckPrerequisites_YqV10Passes(t *testing.T) {
	shimPath(t, "yq (https://github.com/mikefarah/yq/) version v10.0.0")
	if err := checkPrerequisites(); err != nil {
		t.Errorf("yq v10 must pass the v4+ check (numeric compare): %v", err)
	}
}
