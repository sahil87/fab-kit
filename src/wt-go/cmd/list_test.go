package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestList_ShowsRepoNameAndLocation(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "Worktrees for:")
	assertContains(t, r.Stdout, filepath.Base(repo))
	assertContains(t, r.Stdout, "Location:")
}

func TestList_ShowsMainRepo(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "(main)")
	assertContains(t, r.Stdout, "main")
}

func TestList_ShowsTotalCount(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "Total: 1 worktree(s)")
}

func TestList_MultipleWorktrees(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "test-wt1")
	createWorktreeViaWt(t, repo, "test-wt2")

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "test-wt1")
	assertContains(t, r.Stdout, "test-wt2")
	assertContains(t, r.Stdout, "Total: 3 worktree(s)")
}

func TestList_ShowsBranchNames(t *testing.T) {
	repo := createTestRepo(t)

	gitRun(t, repo, "checkout", "-b", "feature/test")
	gitRun(t, repo, "checkout", "main")
	runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "my-feature", "feature/test")

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "my-feature")
	assertContains(t, r.Stdout, "feature/test")
}

func TestList_SucceedsWithNoWorktrees(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "Total: 1 worktree(s)")
}

func TestList_ErrorOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	r := runWt(t, dir, nil, "list")
	if r.ExitCode == 0 {
		t.Error("expected failure outside git repo")
	}
	assertContains(t, r.Stderr, "Not a git repository")
}

// --path flag tests

func TestList_PathReturnsAbsolutePath(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "path-test")

	r := runWtSuccess(t, repo, nil, "list", "--path", "path-test")
	path := strings.TrimSpace(r.Stdout)
	if !strings.HasSuffix(path, "/path-test") {
		t.Errorf("expected path ending in /path-test, got %q", path)
	}
	assertDirExists(t, path)
}

func TestList_PathSingleLineOnly(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "single-line-test")

	r := runWtSuccess(t, repo, nil, "list", "--path", "single-line-test")
	lines := strings.Split(strings.TrimSpace(r.Stdout), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}
}

func TestList_PathNonexistent(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "list", "--path", "nonexistent")
	if r.ExitCode == 0 {
		t.Error("expected failure for nonexistent worktree --path lookup")
	}
	assertContains(t, r.Stderr, "not found")
}

// --json flag tests

func TestList_JSONOutputValid(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "list", "--json")
	entries := parseJSONList(t, r.Stdout)
	if len(entries) == 0 {
		t.Error("expected at least 1 entry in JSON output")
	}
}

func TestList_JSONIncludesMainRepo(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "list", "--json")
	entries := parseJSONList(t, r.Stdout)

	mainCount := 0
	for _, e := range entries {
		if isMain, ok := e["is_main"].(bool); ok && isMain {
			mainCount++
		}
	}
	if mainCount != 1 {
		t.Errorf("expected exactly 1 main entry, got %d", mainCount)
	}
}

func TestList_JSONAllFields(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "json-fields-test")

	r := runWtSuccess(t, repo, nil, "list", "--json")
	entries := parseJSONList(t, r.Stdout)

	found := false
	for _, e := range entries {
		if name, ok := e["name"].(string); ok && name == "json-fields-test" {
			found = true
			// Check all required fields exist
			requiredFields := []string{"name", "branch", "path", "is_main", "is_current", "dirty", "unpushed"}
			for _, f := range requiredFields {
				if _, ok := e[f]; !ok {
					t.Errorf("missing field %q in JSON entry", f)
				}
			}
			// Check types
			if _, ok := e["is_main"].(bool); !ok {
				t.Error("is_main should be boolean")
			}
			if _, ok := e["dirty"].(bool); !ok {
				t.Error("dirty should be boolean")
			}
			if _, ok := e["unpushed"].(float64); !ok {
				t.Error("unpushed should be number")
			}
		}
	}
	if !found {
		t.Error("json-fields-test not found in JSON output")
	}
}

func TestList_JSONDetectsDirty(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "dirty-json-test")

	// Make the worktree dirty
	os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty"), 0644)

	r := runWtSuccess(t, repo, nil, "list", "--json")
	entries := parseJSONList(t, r.Stdout)

	for _, e := range entries {
		if name, ok := e["name"].(string); ok && name == "dirty-json-test" {
			if dirty, ok := e["dirty"].(bool); !ok || !dirty {
				t.Error("expected dirty=true for dirty worktree")
			}
			return
		}
	}
	t.Error("dirty-json-test not found in JSON output")
}

func TestList_JSONIsCurrentField(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "list", "--json")
	entries := parseJSONList(t, r.Stdout)

	for _, e := range entries {
		if name, ok := e["name"].(string); ok && name == "main" {
			if isCurrent, ok := e["is_current"].(bool); !ok || !isCurrent {
				t.Error("expected is_current=true for main when running from main repo")
			}
			return
		}
	}
	t.Error("main not found in JSON output")
}

// mutual exclusivity

func TestList_PathAndJSONMutuallyExclusive(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "list", "--path", "foo", "--json")
	if r.ExitCode == 0 {
		t.Error("expected failure for --path and --json together")
	}
	assertContains(t, r.Stderr, "mutually exclusive")
}

// dirty/status indicators

func TestList_DirtyIndicator(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "dirty-status-test")

	os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty"), 0644)

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "dirty-status-test")
	// Should show * dirty indicator on the dirty worktree line
	for _, line := range strings.Split(r.Stdout, "\n") {
		if strings.Contains(line, "dirty-status-test") {
			if !strings.Contains(line, "*") {
				t.Errorf("expected dirty indicator '*' on dirty-status-test line, got: %s", line)
			}
			return
		}
	}
	t.Fatal("dirty-status-test line not found in output")
}

// NO_COLOR support

func TestList_NoColorSupport(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, []string{"NO_COLOR=1"}, "list")
	// Should not contain ANSI escape codes
	if strings.Contains(r.Stdout, "\033[") {
		t.Error("output contains ANSI color codes despite NO_COLOR=1")
	}
}
