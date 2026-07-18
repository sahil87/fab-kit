package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// runSkill is driven directly with a bytes.Buffer (the testable seam extracted
// from the cobra factory, mirroring shll's runStandards). No subprocess is
// needed — `fab skill` reads embedded bytes only.

// canonicalSkillRelPath is the repo-relative path of the canonical bundle the
// embedded copy must track byte-for-byte.
const canonicalSkillRelPath = "docs/site/skill.md"

// maxSkillBundleLines is the hard line budget from the toolkit `skill` standard:
// bundles are aggregated across every installed tool, so every line is paid N
// times. Pin it here so a future edit that bloats the bundle fails the build.
const maxSkillBundleLines = 150

// --- Command contract --------------------------------------------------------

// TestSkill_StdoutByteIdentical asserts the command writes the embedded bundle
// to stdout byte-for-byte (raw markdown, no framing) and nothing to stderr.
func TestSkill_StdoutByteIdentical(t *testing.T) {
	var stdout bytes.Buffer
	if err := runSkill(&stdout); err != nil {
		t.Fatalf("runSkill err = %v", err)
	}
	if !bytes.Equal(stdout.Bytes(), skillBundle) {
		t.Errorf("stdout is not byte-identical to the embedded bundle (got %d bytes, want %d)",
			stdout.Len(), len(skillBundle))
	}
}

// TestSkill_EmptyStderrThroughCobra drives the assembled command and asserts the
// standard's success contract: stdout carries the bundle, stderr is empty, and
// the run succeeds. Driving through the real cobra command (not just runSkill)
// also proves the bare invocation with cobra.NoArgs is accepted.
func TestSkill_EmptyStderrThroughCobra(t *testing.T) {
	cmd := skillCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("`fab skill` (no args) err = %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("skill wrote to stderr on success: %q", stderr.String())
	}
	if !bytes.Equal(stdout.Bytes(), skillBundle) {
		t.Errorf("stdout is not the embedded bundle")
	}
}

// TestSkill_RejectsArgs asserts an argued invocation is a usage error. cobra.NoArgs
// returns an error before RunE runs, which main()'s run() classifies as exit 2 —
// the standard says `fab skill` takes no args/flags.
func TestSkill_RejectsArgs(t *testing.T) {
	cmd := skillCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"unexpected"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("`fab skill unexpected` should be a usage error, got nil")
	}
}

// --- Bundle constraints ------------------------------------------------------

// TestSkillBundle_LineBudget pins the ≤150-line hard budget from the standard.
// Counting lines in the embedded bytes so the assertion tracks exactly what ships.
func TestSkillBundle_LineBudget(t *testing.T) {
	// A trailing newline yields one empty final element after Split; count only
	// content lines by trimming the trailing newline first.
	body := strings.TrimRight(string(skillBundle), "\n")
	lines := strings.Count(body, "\n") + 1
	if lines > maxSkillBundleLines {
		t.Errorf("skill bundle is %d lines, exceeds the %d-line budget (toolkit skill standard)",
			lines, maxSkillBundleLines)
	}
}

// TestSkillBundle_StaticOnly is a best-effort static-only sanity check: the
// bundle must not carry obviously dynamic, environment-derived tokens (the
// standard forbids timestamps, environment lookups, and session state — contrast
// run-kit context). This catches an accidental paste of a rendered dynamic
// header; it cannot prove staticness exhaustively (byte-identity across
// invocations is guaranteed by construction — the bytes are embedded and written
// verbatim, never templated).
func TestSkillBundle_StaticOnly(t *testing.T) {
	// Guard against Go template placeholders and shell-substitution artifacts that
	// would only appear if dynamic content leaked in. `$(...)` and `${...}` here
	// would be literal in a static doc, but the standard's genre keeps such
	// environment lookups out of the bundle prose.
	forbidden := []string{"{{", "}}", "captured_at", "current time", "$(pwd)", "os.Getenv"}
	body := string(skillBundle)
	for _, tok := range forbidden {
		if strings.Contains(body, tok) {
			t.Errorf("skill bundle contains a non-static token %q — the bundle must be static-only", tok)
		}
	}
}

// --- Drift guard -------------------------------------------------------------

// TestSkillEmbedMatchesCanonical is the drift guard: the embedded bundle bytes
// MUST equal the canonical docs/site/skill.md. It uses findRepoFile (shared with
// lifecycle_collision_test.go) to walk up from the working directory to the
// repo-relative canonical file, so it is robust to the module/package depth. When
// someone edits docs/site/skill.md without re-running scripts/sync-skill.sh, this
// fails, naming the drifted file. Runs on every `go test ./...` and in CI.
func TestSkillEmbedMatchesCanonical(t *testing.T) {
	canonicalPath := findRepoFile(t, canonicalSkillRelPath)
	canonical, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("read canonical %s: %v", canonicalPath, err)
	}
	if !bytes.Equal(skillBundle, canonical) {
		t.Errorf("embedded skill.md has drifted from canonical %s — run scripts/sync-skill.sh and commit the refreshed copy",
			canonicalSkillRelPath)
	}
}
