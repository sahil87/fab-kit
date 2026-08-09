package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// TestAgentAvailable_AnyCandidate: the generic `.agents/skills` target names the
// several CLIs that read that directory, and ONE of them being declared is enough.
// Without any-match semantics a kimi-only or agy-only workspace would silently get
// no skills deployed at all.
func TestAgentAvailable_AnyCandidate(t *testing.T) {
	generic := []string{"codex", "agy", "kimi"}

	for _, only := range generic {
		t.Setenv("FAB_AGENTS", only)
		if !agentAvailable(generic...) {
			t.Errorf("with FAB_AGENTS=%q, the generic target must deploy — any candidate suffices", only)
		}
	}

	t.Setenv("FAB_AGENTS", "claude opencode")
	if agentAvailable(generic...) {
		t.Error("with no generic-directory CLI declared, the generic target must be skipped")
	}
}

// TestMissingCLIs: the skip message adapts to the candidate count. Only the generic
// `.agents/skills` target is gated on more than one CLI, so the plural "none of"
// phrasing would read as a wart on the single-candidate targets — but it must still
// name EVERY candidate when there are several, since that list is what tells a user
// with none installed what would enable the target.
func TestMissingCLIs(t *testing.T) {
	if got, want := missingCLIs([]string{"claude"}), "claude not found in PATH"; got != want {
		t.Errorf("missingCLIs(single) = %q, want %q — a one-candidate target reads oddly as \"none of\"", got, want)
	}

	got := missingCLIs([]string{"codex", "agy", "kimi"})
	if want := "none of codex, agy, kimi found in PATH"; got != want {
		t.Errorf("missingCLIs(generic) = %q, want %q", got, want)
	}
	for _, cli := range []string{"codex", "agy", "kimi"} {
		if !strings.Contains(got, cli) {
			t.Errorf("the generic target's skip message %q must name candidate %q", got, cli)
		}
	}
}

// TestDeploySkills_GenericDirForNonCodexCLI is the change's headline distribution
// behavior (260808-rpsr): agy and kimi read the GENERIC `.agents/skills` directory
// natively, so they deploy there and get NO per-brand directory of their own. The
// per-brand `.gemini/skills` target that used to exist is what made every synced
// skill appear twice to that CLI, which is the warning class this asserts is gone.
func TestDeploySkills_GenericDirForNonCodexCLI(t *testing.T) {
	for _, cli := range []string{"agy", "kimi"} {
		t.Run(cli, func(t *testing.T) {
			kitDir := t.TempDir()
			os.MkdirAll(filepath.Join(kitDir, "skills"), 0755)
			os.WriteFile(filepath.Join(kitDir, "skills", "fab-new.md"), []byte("# New\n"), 0644)

			repoRoot := t.TempDir()
			t.Setenv("FAB_AGENTS", cli)
			if err := deploySkills(repoRoot, kitDir); err != nil {
				t.Fatalf("deploySkills: %v", err)
			}

			if _, err := os.Stat(filepath.Join(repoRoot, ".agents", "skills", "fab-new", "SKILL.md")); err != nil {
				t.Errorf("%s must deploy to the generic .agents/skills directory: %v", cli, err)
			}
			// No per-brand directory for any of the generic-dir CLIs — one target
			// per skill set is what makes duplicate discovery impossible.
			for _, brandDir := range []string{".gemini", ".agy", ".kimi", ".codex"} {
				if _, err := os.Stat(filepath.Join(repoRoot, brandDir)); !os.IsNotExist(err) {
					t.Errorf("%s must not be created — %s reads .agents/skills natively", brandDir, cli)
				}
			}
		})
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

func TestSyncAgentSkills_FailedReplaceDoesNotWriteThroughSymlink(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	skillsDir := t.TempDir()
	os.WriteFile(filepath.Join(skillsDir, "fab-new.md"), []byte("# New\n"), 0644)

	// dest is a symlink pointing at a cache file; its directory is read-only
	// so the replace's os.Remove fails. WriteFile must not follow the
	// leftover symlink and modify its target.
	target := filepath.Join(t.TempDir(), "cached.md")
	os.WriteFile(target, []byte("# Cached\n"), 0644)
	baseDir := filepath.Join(t.TempDir(), "commands")
	os.MkdirAll(baseDir, 0755)
	if err := os.Symlink(target, filepath.Join(baseDir, "fab-new.md")); err != nil {
		t.Fatal(err)
	}
	roDir(t, baseDir)

	agent := agentConfig{Label: "Test", BaseDir: baseDir, Format: "flat", Mode: "copy"}
	err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
	if err == nil {
		t.Fatal("expected the failed replace to surface as an error")
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "# Cached\n" {
		t.Errorf("symlink target was modified (write-through): %q", string(got))
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
