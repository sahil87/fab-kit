package configupgrade

// Full-document golden tests for the config-upgrade fence renderer. R7.1 / T011
// mandate the internal/memoryindex/golden_test.go precedent: pin the COMPLETE
// reconciled document with a literal `got != want` comparison, so any rendering or
// splice change that would churn every user's config.yaml fails loudly HERE first.
//
// These goldens run over a SMALL SYNTHETIC field set (goldenFields) rather than the
// shipped registry, so the pinned bytes do not churn every time a real registry
// row's prose is edited (the shipped-registry byte-stability is covered by the
// idempotence/freeze tests). The only registry-derived bytes are the anchor lines,
// which are composed from the same beginPrefix/endPrefix/fenceWidth constants the
// engine uses — so a deliberate anchor-format change updates both in lockstep and a
// golden that hard-coded the dash count would not silently rot.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
)

// goldenFields is the fixed synthetic registry the full-document goldens render
// over: one non-advertised identity field (project.name — an A field, never fenced)
// and two advertised C fields with short, stable segments. Small on purpose so the
// pinned documents stay readable and do not churn on real-registry prose edits.
func goldenFields() []configref.Field {
	return []configref.Field{
		{Key: "project.name", Description: "name", Scope: configref.ScopeProject, Advertise: false, InitSeed: true},
		{Key: "branch_prefix", Description: "prefix", Scope: configref.ScopeProject, Advertise: true,
			Segment: "# branch_prefix — worktree branch prefix.\n# branch_prefix: \"\""},
		{Key: "test_paths", Description: "tests", Scope: configref.ScopeProject, Advertise: true,
			Segment: "# test_paths — test globs.\ntest_paths:\n  - \"**/*_test.go\""},
	}
}

// beginAnchor / endAnchor rebuild the exact anchor lines the engine emits for a kit
// version, from the same constants — so the goldens track a deliberate anchor-format
// change instead of rotting silently.
func beginAnchor(v string) string { return anchorLine(fmt.Sprintf(beginPrefix, v)) }
func endAnchor() string           { return anchorLine(endPrefix) }

// fenceBlock is the fixed fence body the synthetic field set renders (BEGIN anchor
// through END anchor, inclusive), for a given kit version. Assembled from the same
// constants + segments the engine walks, so a golden document is `preamble + sep +
// fenceBlock + parked`.
func fenceBlock(v string) string {
	return beginAnchor(v) + "\n" +
		fenceHeaderComment + "\n" +
		"#\n" +
		"# branch_prefix — worktree branch prefix.\n" +
		"# branch_prefix: \"\"\n" +
		"#\n" +
		"# test_paths — test globs.\n" +
		"# test_paths:\n" +
		"#   - \"**/*_test.go\"\n" +
		endAnchor()
}

func TestGolden_LegacyNoFence_FullDocument(t *testing.T) {
	got, _ := render("project:\n    name: myproj\n", goldenFields(), "2.15.0")
	want := "project:\n    name: myproj\n\n" + fenceBlock("2.15.0") + "\n"
	if got != want {
		t.Errorf("legacy-no-fence full-document golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGolden_EmptyFile_FullDocument(t *testing.T) {
	got, _ := render("", goldenFields(), "2.15.0")
	want := fenceBlock("2.15.0") + "\n"
	if got != want {
		t.Errorf("empty-file full-document golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGolden_ExistingFence_RegionRewrittenOutsidePreserved(t *testing.T) {
	// A file with an existing (stale-version) fence: the fence region is rewritten
	// (re-stamped, regenerated) while the user's preamble outside it is preserved.
	first, _ := render("project:\n    name: myproj\n", goldenFields(), "2.14.0")
	got, _ := render(first, goldenFields(), "2.15.0")
	want := "project:\n    name: myproj\n\n" + fenceBlock("2.15.0") + "\n"
	if got != want {
		t.Errorf("existing-fence rewrite full-document golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGolden_LiveOverrideOmittedFromFence_FullDocument(t *testing.T) {
	// branch_prefix is live above the fence, so the fence must omit it (advertise
	// only what you could override but haven't). The fence carries test_paths only.
	got, _ := render("branch_prefix: feature/\n", goldenFields(), "2.15.0")
	want := "branch_prefix: feature/\n\n" +
		beginAnchor("2.15.0") + "\n" +
		fenceHeaderComment + "\n" +
		"#\n" +
		"# test_paths — test globs.\n" +
		"# test_paths:\n" +
		"#   - \"**/*_test.go\"\n" +
		endAnchor() + "\n"
	if got != want {
		t.Errorf("live-override-omitted full-document golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGolden_UserCommentOnAField_Preserved_FullDocument(t *testing.T) {
	// A user comment on a live A field is preserved byte-for-byte above the fence.
	src := "# pin the project name — do not change\nproject:\n    name: myproj\n"
	got, _ := render(src, goldenFields(), "2.15.0")
	want := "# pin the project name — do not change\nproject:\n    name: myproj\n\n" + fenceBlock("2.15.0") + "\n"
	if got != want {
		t.Errorf("comment-preserved full-document golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGolden_UnknownKeyParked_FullDocument(t *testing.T) {
	src := "project:\n    name: p\n\nlegacy_mode: true\n"
	got, _ := render(src, goldenFields(), "2.15.0")
	want := "project:\n    name: p\n\n" + fenceBlock("2.15.0") + "\n\n" +
		"# removed in " + parkedVersionPlaceholder + " (parked by fab config upgrade — delete when done):\n" +
		"#   legacy_mode: true\n"
	if got != want {
		t.Errorf("unknown-key-parked full-document golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGolden_BelowFenceContentHoisted_FullDocument(t *testing.T) {
	// R2.1 regression: a live override the user appended BELOW the fence is hoisted
	// above it (never dropped). The reconciled document carries branch_prefix live
	// above the fence, and the fence omits it. The exact scenario the review flagged.
	first, _ := render("project:\n    name: myproj\n", goldenFields(), "2.15.0")
	withBelow := first + "\nbranch_prefix: feature/\n"
	got, _ := render(withBelow, goldenFields(), "2.15.0")
	want := "project:\n    name: myproj\nbranch_prefix: feature/\n\n" +
		beginAnchor("2.15.0") + "\n" +
		fenceHeaderComment + "\n" +
		"#\n" +
		"# test_paths — test globs.\n" +
		"# test_paths:\n" +
		"#   - \"**/*_test.go\"\n" +
		endAnchor() + "\n"
	if got != want {
		t.Errorf("below-fence-hoist full-document golden mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// sliceFence splits rendered output into (preamble, fenceBody, postfence) using the
// same anchors the engine emits. fenceBody is the content between (exclusive) the
// anchors; postfence is everything after the END anchor. Fails the test when no
// fence is present. Retained for the non-golden behavioral tests.
func sliceFence(t *testing.T, out string) (preamble, fenceBody, postfence string) {
	t.Helper()
	lines := strings.Split(out, "\n")
	beginIdx, endIdx := -1, -1
	for i, ln := range lines {
		if beginIdx == -1 && beginLineRe.MatchString(ln) {
			beginIdx = i
			continue
		}
		if beginIdx != -1 && endLineRe.MatchString(ln) {
			endIdx = i
			break
		}
	}
	if beginIdx == -1 || endIdx == -1 {
		t.Fatalf("rendered output has no managed fence:\n%s", out)
	}
	preamble = strings.Join(lines[:beginIdx], "\n")
	fenceBody = strings.Join(lines[beginIdx+1:endIdx], "\n")
	postfence = strings.Join(lines[endIdx+1:], "\n")
	return preamble, fenceBody, postfence
}
