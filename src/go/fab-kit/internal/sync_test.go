package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldTreeWalk_CopyIfAbsent(t *testing.T) {
	scaffoldDir := t.TempDir()
	repoRoot := t.TempDir()

	// Create scaffold file
	os.MkdirAll(filepath.Join(scaffoldDir, "docs", "memory"), 0755)
	os.WriteFile(filepath.Join(scaffoldDir, "docs", "memory", "index.md"), []byte("# Index\n"), 0644)

	// Run tree-walk
	if err := scaffoldTreeWalk(scaffoldDir, repoRoot); err != nil {
		t.Fatalf("scaffoldTreeWalk failed: %v", err)
	}

	// Verify file was copied
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "memory", "index.md"))
	if err != nil {
		t.Fatal("expected index.md to be created")
	}
	if string(data) != "# Index\n" {
		t.Errorf("unexpected content: %s", string(data))
	}
}

func TestScaffoldTreeWalk_CopyIfAbsentSkip(t *testing.T) {
	scaffoldDir := t.TempDir()
	repoRoot := t.TempDir()

	// Create scaffold file
	os.WriteFile(filepath.Join(scaffoldDir, "existing.md"), []byte("scaffold content\n"), 0644)

	// Create destination file with different content
	os.WriteFile(filepath.Join(repoRoot, "existing.md"), []byte("user content\n"), 0644)

	// Run tree-walk
	if err := scaffoldTreeWalk(scaffoldDir, repoRoot); err != nil {
		t.Fatalf("scaffoldTreeWalk failed: %v", err)
	}

	// Verify existing file was NOT overwritten
	data, _ := os.ReadFile(filepath.Join(repoRoot, "existing.md"))
	if string(data) != "user content\n" {
		t.Errorf("existing file should not be overwritten, got: %s", string(data))
	}
}

func TestJsonMergePermissions_CreateNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	srcJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read"},
		},
	}
	srcData, _ := json.MarshalIndent(srcJSON, "", "  ")
	os.WriteFile(src, srcData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal("expected dest file to be created")
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)
	if len(allow) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(allow))
	}
}

func TestJsonMergePermissions_Merge(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	srcJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read", "Write"},
		},
	}
	destJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Edit"},
		},
	}
	srcData, _ := json.MarshalIndent(srcJSON, "", "  ")
	destData, _ := json.MarshalIndent(destJSON, "", "  ")
	os.WriteFile(src, srcData, 0644)
	os.WriteFile(dest, destData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	// Read merged result
	data, _ := os.ReadFile(dest)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)

	// Should have 4: Edit (existing), Bash(git *) (existing/deduped), Read (new), Write (new)
	if len(allow) != 4 {
		t.Errorf("expected 4 permissions after merge, got %d: %v", len(allow), allow)
	}
}

func TestJsonMergePermissions_NoDuplicates(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	// Same permissions in both — no change expected
	perms := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read"},
		},
	}
	srcData, _ := json.MarshalIndent(perms, "", "  ")
	os.WriteFile(src, srcData, 0644)
	os.WriteFile(dest, srcData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)
	if len(allow) != 2 {
		t.Errorf("expected 2 permissions (no duplicates), got %d", len(allow))
	}
}

func TestLineEnsureMerge_CreateNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("# comment\nnode_modules/\n.env\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal("expected dest file to be created")
	}

	content := string(data)
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

func TestLineEnsureMerge_AppendNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("node_modules/\n.env\n"), 0644)
	os.WriteFile(dest, []byte("node_modules/\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	content := string(data)
	// Should contain .env but not duplicate node_modules/
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

func TestLineEnsureMerge_SkipComments(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "entries")
	dest := filepath.Join(destDir, "entries")

	os.WriteFile(src, []byte("# this is a comment\nactual-entry\n"), 0644)

	if err := lineEnsureMerge(src, dest, "entries"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	content := string(data)
	// Should only contain "actual-entry", not the comment
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

func TestListSkills(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "fab-new.md"), []byte("# New\n"), 0644)
	os.WriteFile(filepath.Join(dir, "_preamble.md"), []byte("# Preamble\n"), 0644)
	os.WriteFile(filepath.Join(dir, "fab-setup.md"), []byte("# Setup\n"), 0644)
	os.WriteFile(filepath.Join(dir, "README.txt"), []byte("Not a skill\n"), 0644)

	skills := listSkills(dir)
	if len(skills) != 3 {
		t.Errorf("expected 3 skills (.md files), got %d: %v", len(skills), skills)
	}
}

func TestAgentAvailable_FABAgentsOverride(t *testing.T) {
	t.Setenv("FAB_AGENTS", "claude codex")

	if !agentAvailable("claude") {
		t.Error("expected claude to be available via FAB_AGENTS")
	}
	if !agentAvailable("codex") {
		t.Error("expected codex to be available via FAB_AGENTS")
	}
	if agentAvailable("opencode") {
		t.Error("expected opencode to NOT be available when FAB_AGENTS is set without it")
	}
}

func TestCleanStaleSkills_Directory(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Dir(baseDir)

	// Create directory-format skill entries
	os.MkdirAll(filepath.Join(baseDir, "fab-new"), 0755)
	os.WriteFile(filepath.Join(baseDir, "fab-new", "SKILL.md"), []byte("# New\n"), 0644)
	os.MkdirAll(filepath.Join(baseDir, "old-skill"), 0755)
	os.WriteFile(filepath.Join(baseDir, "old-skill", "SKILL.md"), []byte("# Old\n"), 0644)

	// Canonical skills: only fab-new
	skills := []string{"fab-new"}
	cleanStaleSkills(baseDir, "directory", skills, repoRoot)

	// old-skill should be removed
	if _, err := os.Stat(filepath.Join(baseDir, "old-skill")); !os.IsNotExist(err) {
		t.Error("expected old-skill directory to be removed")
	}
	// fab-new should still exist
	if _, err := os.Stat(filepath.Join(baseDir, "fab-new", "SKILL.md")); err != nil {
		t.Error("expected fab-new skill to still exist")
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.44.10", "0.44.10", 0},
		{"0.44.9", "0.44.10", -1},
		{"0.44.10", "0.44.9", 1},
		{"0.45.0", "0.44.10", 1},
		{"0.44.0", "0.45.0", -1},
		{"1.0.0", "0.99.99", 1},
		{"v0.44.10", "0.44.10", 0},
	}
	for _, tt := range tests {
		got := compareSemver(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareSemver(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestVersionGuard_DevBypass(t *testing.T) {
	if err := versionGuard("0.99.0", "dev"); err != nil {
		t.Errorf("expected dev build to bypass guard, got: %v", err)
	}
}

func TestVersionGuard_SufficientVersion(t *testing.T) {
	if err := versionGuard("0.44.10", "0.44.10"); err != nil {
		t.Errorf("expected equal versions to pass, got: %v", err)
	}
	if err := versionGuard("0.44.9", "0.45.0"); err != nil {
		t.Errorf("expected older fab_version to pass, got: %v", err)
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"0.44.10", [3]int{0, 44, 10}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"0.0.0", [3]int{0, 0, 0}},
	}
	for _, tt := range tests {
		got := parseSemver(tt.input)
		if got != tt.want {
			t.Errorf("parseSemver(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

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

// scaffoldKitDir builds a minimal cached-kit layout under tmp with the given VERSION.
func scaffoldKitDir(t *testing.T, version string) string {
	t.Helper()
	kitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(kitDir, "VERSION"), []byte(version+"\n"), 0644); err != nil {
		t.Fatalf("cannot write kit VERSION: %v", err)
	}
	return kitDir
}

func TestScaffoldDirectories_FreshProject(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if err != nil {
		t.Fatalf("expected .kit-migration-version to be created: %v", err)
	}
	if want := "1.6.1\n"; string(got) != want {
		t.Errorf("fresh project: got %q, want %q", string(got), want)
	}
}

func TestScaffoldDirectories_PreExistingMigrationVersion(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	// Simulate Init() having already stamped both config.yaml and .kit-migration-version.
	if err := os.MkdirAll(filepath.Join(fabDir, "project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, "project", "config.yaml"), []byte("fab_version: 1.6.1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, ".kit-migration-version"), []byte("1.6.1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	// Pre-existing .kit-migration-version must be preserved, not overwritten with 0.1.0.
	got, _ := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if want := "1.6.1\n"; string(got) != want {
		t.Errorf("post-init: got %q, want %q (must preserve, not write 0.1.0)", string(got), want)
	}
}

func TestScaffoldDirectories_ExistingProjectWithoutMigrationVersion(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	// Pre-migration-version-era project: config.yaml exists, no .kit-migration-version.
	// This is the legitimate "existing project" branch (e.g., manual `fab sync` on an old project).
	if err := os.MkdirAll(filepath.Join(fabDir, "project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, "project", "config.yaml"), []byte("fab_version: 0.43.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if want := "0.1.0\n"; string(got) != want {
		t.Errorf("legacy project: got %q, want %q", string(got), want)
	}
}

func TestCleanStaleSkills_Flat(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Dir(baseDir)

	// Create flat-format skill entries
	os.WriteFile(filepath.Join(baseDir, "fab-new.md"), []byte("# New\n"), 0644)
	os.WriteFile(filepath.Join(baseDir, "old-skill.md"), []byte("# Old\n"), 0644)

	skills := []string{"fab-new"}
	cleanStaleSkills(baseDir, "flat", skills, repoRoot)

	// old-skill.md should be removed
	if _, err := os.Stat(filepath.Join(baseDir, "old-skill.md")); !os.IsNotExist(err) {
		t.Error("expected old-skill.md to be removed")
	}
	// fab-new.md should still exist
	if _, err := os.Stat(filepath.Join(baseDir, "fab-new.md")); err != nil {
		t.Error("expected fab-new.md to still exist")
	}
}

// overrideGuardSeams overrides the two versionGuard seams and restores them on cleanup.
func overrideGuardSeams(t *testing.T, brewInstalled bool, installedVersion string, installedErr error) {
	t.Helper()
	origBrew := isBrewInstalled
	origInstalled := installedBinaryVersion
	isBrewInstalled = func() bool { return brewInstalled }
	installedBinaryVersion = func() (string, error) { return installedVersion, installedErr }
	t.Cleanup(func() {
		isBrewInstalled = origBrew
		installedBinaryVersion = origInstalled
	})
}

func TestVersionGuard_NotBrewInstalledFails(t *testing.T) {
	// Non-brew install: Update returns ErrNotBrewInstalled, installed binary
	// stays old — the guard must fail with actionable instructions (it used
	// to silently pass on Update's old nil return).
	overrideGuardSeams(t, false, "0.9.0", nil)

	err := versionGuard("1.0.0", "0.9.0")
	if err == nil {
		t.Fatal("expected guard to fail for non-brew install with too-old binary")
	}
	if !strings.Contains(err.Error(), "manually") {
		t.Errorf("expected actionable manual-update instructions, got: %v", err)
	}
}

func TestVersionGuard_PostStateUpdatedFailsCurrentSync(t *testing.T) {
	// The installed binary on PATH is new enough after the update attempt —
	// post-state decides (even though Update itself errored). The guard must
	// still fail the CURRENT sync so the next run uses the new binary.
	overrideGuardSeams(t, false, "1.0.0", nil)

	err := versionGuard("1.0.0", "0.9.0")
	if err == nil {
		t.Fatal("expected guard to fail the current sync after a successful update")
	}
	if !strings.Contains(err.Error(), "re-run 'fab sync'") {
		t.Errorf("expected re-run guidance, got: %v", err)
	}
}

func TestVersionGuard_BrewReleaseLagFails(t *testing.T) {
	// Brew-installed, but the tap's latest equals the current version
	// (release lag): Update returns nil having upgraded nothing. The
	// post-state check must catch the still-old binary.
	tmpDir := t.TempDir()
	brewScript := "#!/bin/sh\nif [ \"$1\" = \"info\" ]; then printf '%s' '{\"formulae\":[{\"versions\":{\"stable\":\"0.9.0\"}}]}'; fi\nexit 0\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "brew"), []byte(brewScript), 0755); err != nil {
		t.Fatalf("write fake brew: %v", err)
	}
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	overrideGuardSeams(t, true, "0.9.0", nil)

	err := versionGuard("1.0.0", "0.9.0")
	if err == nil {
		t.Fatal("expected guard to fail on brew release lag (Update no-op, binary still old)")
	}
	if !strings.Contains(err.Error(), "still older") {
		t.Errorf("expected release-lag error, got: %v", err)
	}
}

func TestVersionGuard_UnverifiablePostStateFails(t *testing.T) {
	// If the installed version cannot be verified after the update attempt,
	// the guard must fail rather than trust the nil return.
	overrideGuardSeams(t, false, "", fmt.Errorf("fab-kit not found on PATH"))

	err := versionGuard("1.0.0", "0.9.0")
	if err == nil {
		t.Fatal("expected guard to fail when post-state cannot be verified")
	}
}

func TestScaffoldDirectories_MissingKitVersionFails(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := t.TempDir() // no VERSION file

	// New-project branch (no config.yaml) reads kit VERSION — a failed read
	// must propagate, not silently stamp an empty .kit-migration-version.
	err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1")
	if err == nil {
		t.Fatal("expected error when kit VERSION is unreadable")
	}
	if !strings.Contains(err.Error(), "VERSION") {
		t.Errorf("expected kit VERSION read error, got: %v", err)
	}
}

// roDir makes dir read-only for the duration of the test.
func roDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })
}

func TestSyncAgentSkills_CopyWriteFailureCounted(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	skillsDir := t.TempDir()
	os.WriteFile(filepath.Join(skillsDir, "fab-new.md"), []byte("# New\n"), 0644)

	baseDir := filepath.Join(t.TempDir(), "commands")
	os.MkdirAll(baseDir, 0755)
	roDir(t, baseDir) // flat copy into read-only dir fails

	agent := agentConfig{Label: "Test", BaseDir: baseDir, Format: "flat", Mode: "copy"}
	err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
	if err == nil {
		t.Fatal("expected write failure to surface as an error (was silently counted as created)")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected failure count in error, got: %v", err)
	}
}

func TestSyncAgentSkills_SymlinkFailureCounted(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	skillsDir := t.TempDir()
	os.WriteFile(filepath.Join(skillsDir, "fab-new.md"), []byte("# New\n"), 0644)

	baseDir := filepath.Join(t.TempDir(), "commands")
	os.MkdirAll(baseDir, 0755)
	roDir(t, baseDir)

	agent := agentConfig{Label: "Test", BaseDir: baseDir, Format: "flat", Mode: "symlink"}
	err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
	if err == nil {
		t.Fatal("expected symlink failure to surface as an error")
	}
}

func TestSyncAgentSkills_UnreadableSourceCounted(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	skillsDir := t.TempDir()
	src := filepath.Join(skillsDir, "fab-new.md")
	os.WriteFile(src, []byte("# New\n"), 0644)
	os.Chmod(src, 0000)
	t.Cleanup(func() { os.Chmod(src, 0644) })

	baseDir := filepath.Join(t.TempDir(), "skills")
	agent := agentConfig{Label: "Test", BaseDir: baseDir, Format: "flat", Mode: "copy"}
	err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
	if err == nil {
		t.Fatal("expected unreadable source to be counted as a failure (was a silent continue)")
	}
}

func TestSyncAgentSkills_BaseDirCreationFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	skillsDir := t.TempDir()
	os.WriteFile(filepath.Join(skillsDir, "fab-new.md"), []byte("# New\n"), 0644)

	parent := t.TempDir()
	roDir(t, parent)
	agent := agentConfig{Label: "Test", BaseDir: filepath.Join(parent, "skills"), Format: "flat", Mode: "copy"}
	err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
	if err == nil {
		t.Fatal("expected BaseDir MkdirAll failure to surface as an error")
	}
}

func TestDeploySkills_PropagatesAgentFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	kitDir := t.TempDir()
	os.MkdirAll(filepath.Join(kitDir, "skills"), 0755)
	os.WriteFile(filepath.Join(kitDir, "skills", "fab-new.md"), []byte("# New\n"), 0644)

	repoRoot := t.TempDir()
	// .claude exists read-only so MkdirAll(.claude/skills) fails for the claude agent.
	claudeDir := filepath.Join(repoRoot, ".claude")
	os.MkdirAll(claudeDir, 0755)
	roDir(t, claudeDir)

	t.Setenv("FAB_AGENTS", "claude")
	err := deploySkills(repoRoot, kitDir)
	if err == nil {
		t.Fatal("expected deploySkills to propagate the agent deployment failure (Sync must exit non-zero)")
	}
}
