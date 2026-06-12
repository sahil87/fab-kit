package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
)

const cliFabRelPath = "src/kit/skills/_cli-fab.md"

// routerLineRe anchors on the documented router dispatch sentence and
// captures the parenthesized workspace-command list.
var routerLineRe = regexp.MustCompile(`router dispatching workspace commands \(([^)]*)\)`)

// backtickedRe extracts each backticked command name from the captured list.
var backtickedRe = regexp.MustCompile("`([a-z][a-z0-9-]*)`")

// TestRouterDocMatchesLifecycleCommands guards against drift between the
// canonical internal.LifecycleCommands table and the router allowlist
// documented in _cli-fab.md's Calling Convention section. The Go table is
// canonical; this test fails when the doc's parenthesized command list
// disagrees (the drift that shipped when migrations-status was added to the
// allowlist but not the doc). Modeled on the fab module's
// changetypes_doc_test.go code↔doc pattern.
func TestRouterDocMatchesLifecycleCommands(t *testing.T) {
	docPath := findRepoFile(t, cliFabRelPath)

	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}

	m := routerLineRe.FindStringSubmatch(string(data))
	if m == nil {
		t.Fatalf("no 'router dispatching workspace commands (...)' sentence found in %s — the contract anchor moved", docPath)
	}

	var docNames []string
	for _, tok := range backtickedRe.FindAllStringSubmatch(m[1], -1) {
		docNames = append(docNames, tok[1])
	}

	var tableNames []string
	for _, c := range internal.LifecycleCommands {
		tableNames = append(tableNames, c.Name)
	}

	sort.Strings(docNames)
	sort.Strings(tableNames)

	if strings.Join(docNames, ",") != strings.Join(tableNames, ",") {
		t.Errorf("_cli-fab.md router allowlist drifted from LifecycleCommands:\n  doc:   %v\n  table: %v", docNames, tableNames)
	}
}

// findRepoFile resolves a repo-relative path by walking up from the test's
// working directory until the file is found (the changetypes_doc_test.go
// pattern — robust to package depth changes).
func findRepoFile(t *testing.T, relPath string) string {
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
