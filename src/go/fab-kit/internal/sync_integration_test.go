package internal

// Integration-style tests for the Sync orchestrator (260612-tb6f, F45) — the
// previously-untested top of fab-kit's riskiest call chain. The harness
// builds a real temp git repo plus a fake cached kit (HOME-rooted, the same
// seam as cache_test.go) and runs the REAL Sync end to end:
//
//   - run it twice and assert the second run is a content-identical no-op,
//     directly encoding constitution III's idempotency MUST (content compared,
//     not mtimes — syncAgentSkills rewrites only on content mismatch);
//   - cover the shimOnly/projectOnly branch split;
//   - cover cleanLegacyAgents' deletion scoping inside the same harness.
//
// Feasibility seams used: FAB_AGENTS env override (deploySkills),
// HOME=t.TempDir() cache root, systemVersion="dev" versionGuard bypass, and
// PATH shims for the yq/direnv prerequisites (real git/bash stay on PATH).

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// prependPrereqShims puts fake yq (v4) and direnv binaries on PATH while
// keeping the real PATH (git and bash must actually work for the harness).
func prependPrereqShims(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	yq := "#!/bin/sh\necho 'yq (https://github.com/mikefarah/yq/) version v4.44.1'\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "yq"), []byte(yq), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "direnv"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// setupSyncRepo creates a real git repo with a fab project config, chdirs
// into it, and populates a HOME-rooted "dev" kit cache (VERSION, two skills,
// a scaffold with one copy-if-absent file and one line-ensure fragment).
func setupSyncRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)

	repo := t.TempDir()
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	kitDir := filepath.Join(home, ".fab-kit", "versions", "dev", "kit")
	for _, d := range []string{
		filepath.Join(kitDir, "skills"),
		filepath.Join(kitDir, "scaffold", "docs", "specs"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeOrFatal := func(path, content string, mode os.FileMode) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	writeOrFatal(filepath.Join(home, ".fab-kit", "versions", "dev", "fab-go"), "#!/bin/sh\n", 0755)
	writeOrFatal(filepath.Join(kitDir, "VERSION"), "dev\n", 0644)
	writeOrFatal(filepath.Join(kitDir, "skills", "fab-new.md"), "# /fab-new\n", 0644)
	writeOrFatal(filepath.Join(kitDir, "skills", "fab-help.md"), "# /fab-help\n", 0644)
	writeOrFatal(filepath.Join(kitDir, "scaffold", "docs", "specs", "index.md"), "# Specs Index\n", 0644)
	writeOrFatal(filepath.Join(kitDir, "scaffold", "fragment-.gitignore"), "# managed entries\n.claude/\n", 0644)

	// Project config (existing-project shape) + an idempotent project sync script.
	if err := os.MkdirAll(filepath.Join(repo, "fab", "project"), 0755); err != nil {
		t.Fatal(err)
	}
	writeOrFatal(filepath.Join(repo, "fab", "project", "config.yaml"), "fab_version: \"dev\"\n", 0644)
	if err := os.MkdirAll(filepath.Join(repo, "fab", "sync"), 0755); err != nil {
		t.Fatal(err)
	}
	writeOrFatal(filepath.Join(repo, "fab", "sync", "10-mark.sh"), "#!/usr/bin/env bash\nprintf 'ran\\n' > .project-sync-ran\n", 0755)

	prependPrereqShims(t)
	t.Setenv("FAB_AGENTS", "claude")
	chdir(t, repo)
	return repo
}

// snapshotTree maps every file under root (excluding .git/) to its content.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	tree := make(map[string]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		tree[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return tree
}

func TestSync_FullRunProducesExpectedTree(t *testing.T) {
	repo := setupSyncRepo(t)

	// A legacy agent file matching a kit skill must be cleaned; a custom one kept.
	if err := os.MkdirAll(filepath.Join(repo, ".claude", "agents"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(repo, ".claude", "agents", "fab-new.md"), []byte("legacy\n"), 0644)
	os.WriteFile(filepath.Join(repo, ".claude", "agents", "custom-agent.md"), []byte("mine\n"), 0644)

	var err error
	out := captureStdout(t, func() {
		err = Sync("dev", "dev", false, false)
	})
	if err != nil {
		t.Fatalf("Sync: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Done.") {
		t.Errorf("expected 'Done.' in output, got:\n%s", out)
	}

	// Skills deployed for the claude agent (directory format).
	for _, skill := range []string{"fab-new", "fab-help"} {
		data, err := os.ReadFile(filepath.Join(repo, ".claude", "skills", skill, "SKILL.md"))
		if err != nil {
			t.Errorf("skill %s not deployed: %v", skill, err)
			continue
		}
		if !strings.Contains(string(data), skill) {
			t.Errorf("deployed %s content mismatch: %q", skill, data)
		}
	}

	// Scaffolding: directories, copy-if-absent file, line-ensure fragment.
	for _, p := range []string{
		filepath.Join("fab", "changes", ".gitkeep"),
		filepath.Join("fab", ".kit-migration-version"),
		filepath.Join("docs", "specs", "index.md"),
		".gitignore",
	} {
		if _, err := os.Stat(filepath.Join(repo, p)); err != nil {
			t.Errorf("expected %s to exist after sync: %v", p, err)
		}
	}

	// Hook sync wrote agent settings.
	if _, err := os.Stat(filepath.Join(repo, ".claude", "settings.local.json")); err != nil {
		t.Errorf("expected hook sync to write settings.local.json: %v", err)
	}

	// Project sync script ran.
	if _, err := os.Stat(filepath.Join(repo, ".project-sync-ran")); err != nil {
		t.Errorf("expected project sync script marker: %v", err)
	}

	// cleanLegacyAgents scoping: skill-named legacy file removed, custom kept.
	if _, err := os.Stat(filepath.Join(repo, ".claude", "agents", "fab-new.md")); !os.IsNotExist(err) {
		t.Error("legacy agent file matching a kit skill should be deleted")
	}
	if _, err := os.Stat(filepath.Join(repo, ".claude", "agents", "custom-agent.md")); err != nil {
		t.Error("custom agent file (not a kit skill) must be preserved")
	}
}

// TestSync_SecondRunIsContentIdenticalNoop is the constitution III idempotency
// contract: running Sync twice with identical inputs must leave every file's
// CONTENT identical (mtimes may differ — syncAgentSkills rewrites only on
// content mismatch, the merges only append missing entries).
func TestSync_SecondRunIsContentIdenticalNoop(t *testing.T) {
	repo := setupSyncRepo(t)

	var err error
	captureStdout(t, func() { err = Sync("dev", "dev", false, false) })
	if err != nil {
		t.Fatalf("first Sync: %v", err)
	}
	first := snapshotTree(t, repo)

	captureStdout(t, func() { err = Sync("dev", "dev", false, false) })
	if err != nil {
		t.Fatalf("second Sync: %v", err)
	}
	second := snapshotTree(t, repo)

	for rel, content := range first {
		got, ok := second[rel]
		if !ok {
			t.Errorf("second run deleted %s", rel)
			continue
		}
		if got != content {
			t.Errorf("second run changed content of %s:\n--- first ---\n%s\n--- second ---\n%s", rel, content, got)
		}
	}
	for rel := range second {
		if _, ok := first[rel]; !ok {
			t.Errorf("second run created new file %s", rel)
		}
	}
}

func TestSync_ProjectOnlyRunsOnlyProjectScripts(t *testing.T) {
	repo := setupSyncRepo(t)

	var err error
	captureStdout(t, func() { err = Sync("dev", "dev", false, true) })
	if err != nil {
		t.Fatalf("Sync --project: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repo, ".project-sync-ran")); err != nil {
		t.Error("projectOnly run must execute fab/sync scripts")
	}
	if _, err := os.Stat(filepath.Join(repo, ".claude", "skills")); !os.IsNotExist(err) {
		t.Error("projectOnly run must not deploy skills (steps 1-5 skipped)")
	}
	if _, err := os.Stat(filepath.Join(repo, "fab", ".kit-migration-version")); !os.IsNotExist(err) {
		t.Error("projectOnly run must not scaffold the workspace")
	}
}

func TestSync_ShimOnlySkipsProjectScripts(t *testing.T) {
	repo := setupSyncRepo(t)

	var err error
	captureStdout(t, func() { err = Sync("dev", "dev", true, false) })
	if err != nil {
		t.Fatalf("Sync --shim: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repo, ".claude", "skills", "fab-new", "SKILL.md")); err != nil {
		t.Error("shimOnly run must still deploy skills")
	}
	if _, err := os.Stat(filepath.Join(repo, ".project-sync-ran")); !os.IsNotExist(err) {
		t.Error("shimOnly run must not execute fab/sync scripts (step 6 skipped)")
	}
}

func TestSync_FailedProjectScriptPropagates(t *testing.T) {
	repo := setupSyncRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "fab", "sync", "20-fail.sh"), []byte("#!/usr/bin/env bash\nexit 7\n"), 0755); err != nil {
		t.Fatal(err)
	}

	var err error
	captureStdout(t, func() { err = Sync("dev", "dev", false, false) })
	if err == nil {
		t.Fatal("expected a failing project sync script to fail Sync")
	}
	if !strings.Contains(err.Error(), "20-fail.sh") {
		t.Errorf("expected failing script name in error, got: %v", err)
	}
}

// --- cleanLegacyAgents deletion scoping (direct, same harness style) ---

func TestCleanLegacyAgents_DeletesOnlySkillNamedMarkdown(t *testing.T) {
	repo := t.TempDir()
	kitDir := t.TempDir()
	os.MkdirAll(filepath.Join(kitDir, "skills"), 0755)
	os.WriteFile(filepath.Join(kitDir, "skills", "fab-new.md"), []byte("# New\n"), 0644)

	agentsDir := filepath.Join(repo, ".claude", "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "fab-new.md"), []byte("legacy\n"), 0644)      // skill-named → deleted
	os.WriteFile(filepath.Join(agentsDir, "custom-agent.md"), []byte("mine\n"), 0644)   // not a skill → kept
	os.WriteFile(filepath.Join(agentsDir, "notes.txt"), []byte("not markdown\n"), 0644) // non-.md → kept
	// A neighboring project file outside .claude/agents must never be touched.
	os.WriteFile(filepath.Join(repo, ".claude", "fab-new.md"), []byte("outside scope\n"), 0644)

	cleanLegacyAgents(repo, kitDir)

	if _, err := os.Stat(filepath.Join(agentsDir, "fab-new.md")); !os.IsNotExist(err) {
		t.Error("skill-named legacy agent file should be deleted")
	}
	for _, keep := range []string{
		filepath.Join(agentsDir, "custom-agent.md"),
		filepath.Join(agentsDir, "notes.txt"),
		filepath.Join(repo, ".claude", "fab-new.md"),
	} {
		if _, err := os.Stat(keep); err != nil {
			t.Errorf("%s must survive cleanLegacyAgents: %v", keep, err)
		}
	}
}

func TestCleanLegacyAgents_NoAgentsDirIsNoop(t *testing.T) {
	repo := t.TempDir()
	kitDir := t.TempDir()
	os.MkdirAll(filepath.Join(kitDir, "skills"), 0755)

	// Must not create the directory or panic.
	cleanLegacyAgents(repo, kitDir)
	if _, err := os.Stat(filepath.Join(repo, ".claude")); !os.IsNotExist(err) {
		t.Error("cleanLegacyAgents must not create .claude/")
	}
}
