package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// The fab module cannot import the fab-kit module's internal package (Go
// internal-visibility is module-scoped), so the router allowlist is read from
// its documented form: the `_cli-fab.md` router line. The fab-kit module's
// contract test pins that line to the canonical internal.LifecycleCommands
// table, so the two tests together give transitive code↔code coverage with
// no cross-module import and no subprocess.
const collisionDocRelPath = "src/kit/skills/_cli-fab.md"

var collisionRouterLineRe = regexp.MustCompile(`router dispatching workspace commands \(([^)]*)\)`)
var collisionBacktickedRe = regexp.MustCompile("`([a-z][a-z0-9-]*)`")

// TestNoTopLevelCommandCollidesWithRouterAllowlist asserts that no top-level
// fab-go command name appears in the router's workspace-command allowlist.
// The `fab` shim (Homebrew-installed, system-wide) dispatches allowlisted
// names to fab-kit BEFORE fab-go ever sees them, so a colliding fab-go
// command would be silently shadowed forever. Top-level names are sourced
// from the in-process `fab help-dump` tree of the assembled root command
// (the machine-readable contract walk — dumpDoc/buildNode).
func TestNoTopLevelCommandCollidesWithRouterAllowlist(t *testing.T) {
	allowlist := parseRouterAllowlist(t)
	if len(allowlist) == 0 {
		t.Fatal("parsed an empty router allowlist — the _cli-fab.md contract anchor moved")
	}

	doc := dumpDoc(newRootCmd(), version)
	if len(doc.Root.Commands) == 0 {
		t.Fatal("help-dump tree has no top-level commands — root assembly broken")
	}

	for _, node := range doc.Root.Commands {
		if allowlist[node.Name] {
			t.Errorf("fab-go top-level command %q collides with the router's workspace allowlist — the shim would shadow it (rename the command or remove it from the allowlist)", node.Name)
		}
	}
}

// parseRouterAllowlist extracts the backticked workspace-command names from
// the _cli-fab.md router line.
func parseRouterAllowlist(t *testing.T) map[string]bool {
	t.Helper()

	docPath := findCollisionDocFile(t, collisionDocRelPath)
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}

	m := collisionRouterLineRe.FindStringSubmatch(string(data))
	if m == nil {
		t.Fatalf("no 'router dispatching workspace commands (...)' sentence found in %s", docPath)
	}

	set := make(map[string]bool)
	for _, tok := range collisionBacktickedRe.FindAllStringSubmatch(m[1], -1) {
		set[tok[1]] = true
	}
	return set
}

// findCollisionDocFile resolves a repo-relative path by walking up from the
// test's working directory (the changetypes_doc_test.go pattern).
func findCollisionDocFile(t *testing.T, relPath string) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		candidate := filepath.Join(dir, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate %q by walking up to the filesystem root", relPath)
		}
		dir = parent
	}
}
