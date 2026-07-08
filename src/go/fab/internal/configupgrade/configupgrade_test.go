package configupgrade

// Behavioral tests over the SHIPPED registry (configref.Fields). These assert the
// engine's contract properties against the real field set — complementing the
// full-document goldens in golden_test.go, which pin exact bytes over a small
// synthetic set. Split out so a real-registry prose edit churns neither the goldens
// nor these (which assert behavior, not bytes).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
)

// fieldsForTest returns the shipped registry, failing the test on a construction
// error (the registry lint / tier invariant).
func fieldsForTest(t *testing.T) []configref.Field {
	t.Helper()
	f, err := configref.Fields()
	if err != nil {
		t.Fatalf("configref.Fields: %v", err)
	}
	return f
}

// TestRender_AppendsFenceToLegacyFile: a legacy config.yaml with no fence gets a
// managed fence appended at the bottom, with byte-exact anchors, while the user's
// live keys and their own comments above are preserved verbatim.
func TestRender_AppendsFenceToLegacyFile(t *testing.T) {
	fields := fieldsForTest(t)
	legacy := `project:
    name: fab-kit
    description: FAB Kit

# pin review to fable — sonnet missed doc-claim regressions
agent:
    tiers:
        review:
            model: claude-fable-5
            effort: xhigh
`
	out, _ := render(legacy, fields, "2.15.0")

	// The user's preamble (live keys + their comment) is preserved verbatim as a prefix.
	if !strings.HasPrefix(out, legacy) {
		t.Errorf("legacy preamble must be preserved verbatim as a prefix.\n--- got ---\n%s", out)
	}
	// Byte-exact BEGIN/END anchors (dash-padded to fenceWidth).
	wantBegin := "# >>> fab reference (kit 2.15.0) >>> " + strings.Repeat("-", fenceWidth-len("# >>> fab reference (kit 2.15.0) >>> "))
	wantEnd := "# <<< end fab reference <<< " + strings.Repeat("-", fenceWidth-len("# <<< end fab reference <<< "))
	if !strings.Contains(out, wantBegin) {
		t.Errorf("missing byte-exact BEGIN anchor %q", wantBegin)
	}
	if !strings.Contains(out, wantEnd) {
		t.Errorf("missing byte-exact END anchor %q", wantEnd)
	}
	// Ends in exactly one trailing newline.
	if !strings.HasSuffix(out, "\n") || strings.HasSuffix(out, "\n\n") {
		t.Error("output must end in exactly one trailing newline")
	}
	// The rendered document must parse as YAML.
	if err := validateYAML(out); err != nil {
		t.Errorf("rendered legacy output does not parse: %v", err)
	}
}

// TestRender_FenceOmitsOverriddenFields: a field the user has overridden live above
// the fence is NOT re-advertised inside the fence (it shows what you could override
// but haven't). agent.tiers is live here, so the fence must not scaffold `agent:`.
func TestRender_FenceOmitsOverriddenFields(t *testing.T) {
	fields := fieldsForTest(t)
	src := `agent:
    tiers:
        review:
            model: claude-fable-5
`
	out, _ := render(src, fields, "2.15.0")

	_, fenceBody, _ := sliceFence(t, out)
	if strings.Contains(fenceBody, "agent:") {
		t.Errorf("fence must omit the already-overridden agent.tiers field.\n--- fence ---\n%s", fenceBody)
	}
	// But an un-overridden advertise field IS scaffolded (fully commented).
	if !strings.Contains(fenceBody, "# providers") {
		t.Errorf("fence must advertise the un-overridden providers field.\n--- fence ---\n%s", fenceBody)
	}
}

// TestRender_FenceFullyComments the C-field scaffold, INCLUDING parent keys — a
// live `agent:` over comment-only children is the `agent: null` masher bug the
// fence design exists to prevent. Every non-blank line inside the fence body is a
// comment.
func TestRender_FenceFullyComments(t *testing.T) {
	fields := fieldsForTest(t)
	out, _ := render("", fields, "2.15.0")
	_, fenceBody, _ := sliceFence(t, out)

	for _, ln := range strings.Split(fenceBody, "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(ln), "#") {
			t.Errorf("fence body must be fully commented; found live line %q", ln)
		}
	}
	// A parent key must appear only in commented form (never a live `providers:`).
	if strings.Contains(fenceBody, "\nproviders:") {
		t.Error("fence must not carry a LIVE providers: parent key (must be commented)")
	}
}

// TestRender_ParksUnknownLiveKey: a live top-level key absent from the registry is
// removed from the live YAML and parked in a comment block below the fence, its
// value serialized. The live key must be gone from the active config.
func TestRender_ParksUnknownLiveKey(t *testing.T) {
	fields := fieldsForTest(t)
	src := `project:
    name: t

legacy_mode: true
`
	out, report := render(src, fields, "2.15.0")

	preamble, _, postfence := sliceFence(t, out)
	if strings.Contains(preamble, "legacy_mode:") {
		t.Errorf("unknown live key must be removed from the active config.\n--- preamble ---\n%s", preamble)
	}
	if !strings.Contains(postfence, "# removed in") || !strings.Contains(postfence, "#   legacy_mode: true") {
		t.Errorf("unknown key must be parked below the fence with its value.\n--- postfence ---\n%s", postfence)
	}
	// The user's real field is preserved.
	if !strings.Contains(preamble, "name: t") {
		t.Errorf("the user's live project field must be preserved.\n--- preamble ---\n%s", preamble)
	}
	if len(report) == 0 || !strings.Contains(strings.Join(report, "\n"), "legacy_mode") {
		t.Errorf("the report must note the parked key, got %v", report)
	}
}

// TestRender_PreservesUserCommentOnLiveField: a user's own comment on a live A
// field is preserved byte-for-byte (outside the fence is the user's).
func TestRender_PreservesUserCommentOnLiveField(t *testing.T) {
	fields := fieldsForTest(t)
	comment := "# pin review to fable — sonnet missed doc-claim regressions"
	src := comment + `
agent:
    tiers:
        review:
            model: claude-fable-5
`
	out, _ := render(src, fields, "2.15.0")
	if !strings.Contains(out, comment) {
		t.Errorf("user comment on a live field must be preserved verbatim.\n--- got ---\n%s", out)
	}
}

// TestRender_EmptyFileWritesFenceOnly: a fresh (empty) file becomes a fence-only
// document — no preamble, valid anchors, ends in one newline.
func TestRender_EmptyFileWritesFenceOnly(t *testing.T) {
	fields := fieldsForTest(t)
	out, _ := render("", fields, "2.15.0")
	if !strings.HasPrefix(out, "# >>> fab reference (kit 2.15.0) >>> ") {
		t.Errorf("empty file should start with the BEGIN anchor, got:\n%s", out)
	}
	if !strings.HasSuffix(out, "\n") || strings.HasSuffix(out, "\n\n") {
		t.Error("output must end in exactly one trailing newline")
	}
}

// TestRender_BelowFenceLiveOverrideHoisted (R2.1 regression, the review's must-fix
// #1): a live override the user appends BELOW the fence is HOISTED above it and
// preserved — never silently dropped. The exact empirical scenario the review
// confirmed (branch_prefix appended below the fence vanished on re-render).
func TestRender_BelowFenceLiveOverrideHoisted(t *testing.T) {
	fields := fieldsForTest(t)
	first, _ := render("project:\n    name: t\n", fields, "2.15.0")
	withBelow := first + "\nbranch_prefix: feature/\n"

	out, _ := render(withBelow, fields, "2.15.0")
	if !strings.Contains(out, "branch_prefix: feature/") {
		t.Fatalf("a live override appended below the fence must NOT be dropped.\n--- got ---\n%s", out)
	}
	preamble, fenceBody, _ := sliceFence(t, out)
	if !strings.Contains(preamble, "branch_prefix: feature/") {
		t.Errorf("below-fence override must be hoisted ABOVE the fence (live A field).\n--- preamble ---\n%s", preamble)
	}
	if strings.Contains(fenceBody, "branch_prefix") {
		t.Errorf("the now-live branch_prefix must be omitted from the regenerated fence.\n--- fence ---\n%s", fenceBody)
	}
	// Idempotent: a third run over the hoisted document is byte-identical.
	third, _ := render(out, fields, "2.15.0")
	if third != out {
		t.Errorf("hoisted below-fence content must be idempotent.\n--- out ---\n%s\n--- third ---\n%s", out, third)
	}
}

// TestRender_BelowFenceUnknownKeyParked: an UNKNOWN key the user appends below the
// fence is hoisted, then parked (not left dangling below the fence, and not dropped).
func TestRender_BelowFenceUnknownKeyParked(t *testing.T) {
	fields := fieldsForTest(t)
	first, _ := render("project:\n    name: t\n", fields, "2.15.0")
	withBelow := first + "\nmy_custom_key: 99\n"

	out, report := render(withBelow, fields, "2.15.0")
	_, _, postfence := sliceFence(t, out)
	if !strings.Contains(postfence, "#   my_custom_key: 99") {
		t.Errorf("an unknown key appended below the fence must be parked, not dropped.\n--- postfence ---\n%s", postfence)
	}
	if len(report) == 0 || !strings.Contains(strings.Join(report, "\n"), "my_custom_key") {
		t.Errorf("report must note the parked below-fence key, got %v", report)
	}
}

// TestRender_InteriorColumn0CommentInLiveBlock (SF-c): a live block with an interior
// column-0 comment between its indented lines is captured as ONE block — the
// trailing indented lines are not orphaned. The rendered output parses as YAML.
func TestRender_InteriorColumn0CommentInLiveBlock(t *testing.T) {
	fields := fieldsForTest(t)
	// An unknown live block whose value carries an interior column-0 comment.
	src := "custom_block:\n    a: 1\n# a comment the user wrote inside the block\n    b: 2\n"
	out, _ := render(src, fields, "2.15.0")
	_, _, postfence := sliceFence(t, out)
	// Both indented children must be parked together (b must not orphan above the fence).
	for _, want := range []string{"#   custom_block:", "#       a: 1", "#       b: 2"} {
		if !strings.Contains(postfence, want) {
			t.Errorf("interior-comment block must be parked whole; missing %q.\n--- postfence ---\n%s", want, postfence)
		}
	}
	if err := validateYAML(out); err != nil {
		t.Errorf("rendered output must parse as YAML: %v\n%s", err, out)
	}
}

// TestRender_BHygieneFlagsEqualsDefault (SF-d): a live field whose value equals the
// built-in default is flagged in the advisory report — but never removed
// (presence=intent). Uses providers, whose default is the built-in claude session
// command.
func TestRender_BHygieneFlagsEqualsDefault(t *testing.T) {
	fields := fieldsForTest(t)
	src := "providers:\n    claude:\n        session_command: '" + agent.DefaultSessionCommand + "'\n"
	out, report := render(src, fields, "2.15.0")

	if !strings.Contains(out, "providers:") {
		t.Error("the live providers field must be kept (presence=intent — never auto-removed)")
	}
	joined := strings.Join(report, "\n")
	if !strings.Contains(joined, "providers") || !strings.Contains(joined, "equals the current default") {
		t.Errorf("B-hygiene must flag providers==default, got %v", report)
	}
}

// TestRender_BHygieneSilentOnRealOverride: a genuinely different live value is NOT
// flagged (no false positive).
func TestRender_BHygieneSilentOnRealOverride(t *testing.T) {
	fields := fieldsForTest(t)
	src := "providers:\n    claude:\n        session_command: 'my-custom-agent --flag'\n"
	_, report := render(src, fields, "2.15.0")
	if strings.Contains(strings.Join(report, "\n"), "equals the current default") {
		t.Errorf("a real override must not be flagged as equals-default, got %v", report)
	}
}

// TestUpgrade_RefusesUnparseableOutput (SF-c): Upgrade validates the reconciled
// bytes and REFUSES to overwrite the file when the result would not parse as YAML,
// leaving the original file byte-untouched. Exercised via a live block whose value
// is broken YAML (an unclosed flow mapping) — it survives the line-splice but fails
// the parser, so a config that would brick every fab command is never written.
func TestUpgrade_RefusesUnparseableOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := "project: {name: t\n" // unclosed flow map → the assembled doc won't parse
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Upgrade(path, "2.15.0")
	if err == nil {
		t.Fatal("Upgrade must REFUSE to write output that does not parse as YAML")
	}
	if !strings.Contains(err.Error(), "does not parse") {
		t.Errorf("refusal error should explain the parse failure, got: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != original {
		t.Errorf("a refused upgrade must leave the original file untouched.\n--- original ---\n%s\n--- after ---\n%s", original, string(after))
	}
}
