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

// TestCleanStaleSkills_Directory: pruning is scoped to the previous manifest —
// an entry is removed only when fab recorded it (in prev) AND the kit dropped
// it. A user-added directory fab never recorded must survive, and a manifest
// entry with no on-disk directory is a silent no-op.
func TestCleanStaleSkills_Directory(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Dir(baseDir)

	// On disk: fab-new (still in kit), fab-old (kit dropped it), my-team-skill (user's).
	os.MkdirAll(filepath.Join(baseDir, "fab-new"), 0755)
	os.WriteFile(filepath.Join(baseDir, "fab-new", "SKILL.md"), []byte("# New\n"), 0644)
	os.MkdirAll(filepath.Join(baseDir, "fab-old"), 0755)
	os.WriteFile(filepath.Join(baseDir, "fab-old", "SKILL.md"), []byte("# Old\n"), 0644)
	os.MkdirAll(filepath.Join(baseDir, "my-team-skill"), 0755)

	// Previous manifest recorded fab-new, fab-old, and gone-skill (no dir on disk).
	prev := map[string]bool{"fab-new": true, "fab-old": true, "gone-skill": true}
	skills := []string{"fab-new"}
	cleanStaleSkills(baseDir, "directory", prev, skills, true, repoRoot)

	if _, err := os.Stat(filepath.Join(baseDir, "fab-old")); !os.IsNotExist(err) {
		t.Error("expected fab-old (previously manifested, dropped from kit) to be removed")
	}
	if _, err := os.Stat(filepath.Join(baseDir, "fab-new", "SKILL.md")); err != nil {
		t.Error("expected fab-new skill to still exist")
	}
	if _, err := os.Stat(filepath.Join(baseDir, "my-team-skill")); err != nil {
		t.Error("user-added directory fab never manifested must NOT be removed")
	}
}

// TestCleanStaleSkills_Flat: same manifest scoping for the flat format —
// a user's .md that fab never recorded must survive.
func TestCleanStaleSkills_Flat(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Dir(baseDir)

	os.WriteFile(filepath.Join(baseDir, "fab-new.md"), []byte("# New\n"), 0644)
	os.WriteFile(filepath.Join(baseDir, "old-skill.md"), []byte("# Old\n"), 0644)
	os.WriteFile(filepath.Join(baseDir, "mine.md"), []byte("# Mine\n"), 0644)

	prev := map[string]bool{"fab-new": true, "old-skill": true, "gone": true}
	skills := []string{"fab-new"}
	cleanStaleSkills(baseDir, "flat", prev, skills, true, repoRoot)

	if _, err := os.Stat(filepath.Join(baseDir, "old-skill.md")); !os.IsNotExist(err) {
		t.Error("expected old-skill.md (previously manifested, dropped from kit) to be removed")
	}
	if _, err := os.Stat(filepath.Join(baseDir, "fab-new.md")); err != nil {
		t.Error("expected fab-new.md to still exist")
	}
	if _, err := os.Stat(filepath.Join(baseDir, "mine.md")); err != nil {
		t.Error("user-added mine.md fab never manifested must NOT be removed")
	}
}

// TestCleanStaleSkills_NoManifestPrunesNothing: without a manifest fab has no
// ownership record, so NOTHING is pruned — and when the directory held a
// non-kit entry, exactly one note is printed instead.
func TestCleanStaleSkills_NoManifestPrunesNothing(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Dir(baseDir)
	os.MkdirAll(filepath.Join(baseDir, "fab-new"), 0755)
	os.MkdirAll(filepath.Join(baseDir, "unknown"), 0755)

	var noteCount int
	out := captureStdout(t, func() {
		cleanStaleSkills(baseDir, "directory", nil, []string{"fab-new"}, false, repoRoot)
	})
	noteCount = strings.Count(out, "has no fab manifest yet")
	if noteCount != 1 {
		t.Errorf("expected the no-manifest note exactly once, got %d in:\n%s", noteCount, out)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "unknown")); err != nil {
		t.Error("no-manifest run must prune nothing — unknown/ must survive")
	}
}

// TestCleanStaleSkills_NoManifestKitOnlyDirPrintsNothing: an all-kit directory
// has nothing to warn about — no note on every fresh checkout.
func TestCleanStaleSkills_NoManifestKitOnlyDirPrintsNothing(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Dir(baseDir)
	os.MkdirAll(filepath.Join(baseDir, "fab-new"), 0755)

	out := captureStdout(t, func() {
		cleanStaleSkills(baseDir, "directory", nil, []string{"fab-new"}, false, repoRoot)
	})
	if strings.Contains(out, "no fab manifest") {
		t.Errorf("kit-only directory must not print the note, got:\n%s", out)
	}
}

// TestWriteReadSkillManifest_RoundTrip: the manifest is byte-stable, anchored
// per format, self-ignoring, and parses back to exactly the deployed set.
func TestWriteReadSkillManifest_RoundTrip(t *testing.T) {
	for _, tc := range []struct {
		format   string
		deployed []string
		want     string
	}{
		{"directory", []string{"_preamble", "fab-new"},
			manifestHeaderSkills + "/.gitignore\n/_preamble/\n/fab-new/\n"},
		{"flat", []string{"_preamble", "fab-new"},
			manifestHeaderCmds + "/.gitignore\n/_preamble.md\n/fab-new.md\n"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			baseDir := t.TempDir()
			if err := writeSkillManifest(baseDir, tc.format, tc.deployed); err != nil {
				t.Fatalf("writeSkillManifest: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(baseDir, ".gitignore"))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tc.want {
				t.Errorf("manifest content mismatch:\n--- want ---\n%s\n--- got ---\n%s", tc.want, data)
			}
			// Byte-stable: a second write with the same list is identical.
			if err := writeSkillManifest(baseDir, tc.format, tc.deployed); err != nil {
				t.Fatal(err)
			}
			data2, _ := os.ReadFile(filepath.Join(baseDir, ".gitignore"))
			if string(data2) != string(data) {
				t.Error("manifest must be byte-stable across two writes")
			}
			// Round-trip: parses back to exactly the deployed set.
			owned, had := readSkillManifest(baseDir, tc.format)
			if !had {
				t.Fatal("readSkillManifest must report the manifest exists")
			}
			if len(owned) != len(tc.deployed) {
				t.Fatalf("owned set %v, want %v", owned, tc.deployed)
			}
			for _, name := range tc.deployed {
				if !owned[name] {
					t.Errorf("owned set missing %q: %v", name, owned)
				}
			}
			if owned[".gitignore"] {
				t.Error("the self-entry must not parse as an owned skill")
			}
		})
	}
}

// TestReadSkillManifest_CorruptLinesSkipped: unparseable lines are skipped,
// not fatal; a missing manifest reports had=false.
func TestReadSkillManifest_CorruptLinesSkipped(t *testing.T) {
	baseDir := t.TempDir()
	content := "# comment\n/.gitignore\n/fab-new/\nno-anchor\n/\n\n/\n"
	os.WriteFile(filepath.Join(baseDir, ".gitignore"), []byte(content), 0644)

	owned, had := readSkillManifest(baseDir, "directory")
	if !had {
		t.Fatal("manifest exists — had must be true")
	}
	if len(owned) != 1 || !owned["fab-new"] {
		t.Errorf("corrupt lines must be skipped, owned = %v", owned)
	}

	_, had = readSkillManifest(filepath.Join(t.TempDir(), "nope"), "directory")
	if had {
		t.Error("missing manifest must report had=false")
	}
}

// TestWriteSkillManifest_FailurePropagates: a failed manifest write surfaces
// as an error (jznd fail-loud contract — Sync must exit non-zero).
func TestWriteSkillManifest_FailurePropagates(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	baseDir := t.TempDir()
	roDir(t, baseDir)
	if err := writeSkillManifest(baseDir, "flat", []string{"fab-new"}); err == nil {
		t.Fatal("expected a manifest write failure to propagate as an error")
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
	_, err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
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
	_, err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
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
	_, err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
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
	_, err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
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
	_, err := syncAgentSkills(agent, []string{"fab-new"}, skillsDir)
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
