package agent

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
)

// defaultTiers is canonical (agent.go). TestDocTablesMatchAgentMaps guards its
// ONE structured mirror — the 4-column table in docs/specs/stage-models.md. But
// the same profiles are restated in several other docs in two looser shapes that
// no test covered, so each drifted silently whenever a model was bumped:
//
//	inline YAML   default:  { provider: claude, model: claude-fable-5, effort: high }
//	cell triple   | `default` | ...role prose... | `claude` / `claude-fable-5` / `high` |
//
// This file guards BOTH shapes across every doc that carries them. It is
// deliberately shape-driven rather than file-driven for the restated profiles: a
// new doc that copies a tier line in either shape is picked up by the same
// assertion set as soon as it is added to mirrorDocs.
//
// Scope note: only lines naming a KNOWN tier are checked, and only where the
// line carries a model. Historical upgrade notes that quote a superseded default
// as past state are intentionally NOT in scope (they are prose about a prior
// release, not a mirror of today's table) — they live outside mirrorDocs.
//
// Two kinds of tier line deliberately do NOT mirror the defaults and are skipped
// (see exampleMarker): an illustrative OVERRIDE ("run doing cheaper") must differ
// from the default — that is the whole point of the example — and an empty model
// is the "inherit the session model" signal. Both are marked in-doc with an
// `# example:` / `# e.g.` trailing comment, which is the opt-out this guard reads.

// mirrorDocs are the docs that still restate the built-in default profiles in one
// of the two loose shapes, and so must be checked against defaultTiers.
//
// Kept deliberately SHORT: the fix for doc drift is usually to stop mirroring
// (point at `fab config reference`, which renders live from defaultTiers) rather
// than to add another file here. docs/specs/architecture.md,
// docs/memory/_shared/configuration.md, and docs/memory/runtime/providers-and-tiers.md
// were all removed from this list once they were converted to pointers — they now
// show shape-only YAML and own the tier ROLES, not the profiles.
//
// stage-models.md stays: it is the one human-readable mirror of the values, in two
// shapes — the 4-column table (covered by TestDocTablesMatchAgentMaps) and the
// inline-YAML reference sample (covered here).
//
// _cli-fab.md is deliberately absent: it carries only the third, run-on shape,
// guarded by TestCLIFabReferenceListsDefaultTiers below.
var mirrorDocs = []string{
	"docs/specs/stage-models.md",
}

// inlineTierYAML matches an inline-YAML tier line, tolerating the column-aligned
// padding these docs use:
//
//	doing:    { provider: claude, model: claude-opus-5,   effort: xhigh }
//
// Leading "# " (commented reference blocks) is tolerated by the caller trimming it.
var inlineTierYAML = regexp.MustCompile(
	`^(\w+):\s*\{\s*provider:\s*([\w-]+),\s*model:\s*([\w.-]*),\s*effort:\s*(\w*)\s*\}`)

// cellTriple matches a markdown table row whose LAST cell is a
// `provider` / `model` / `effort` triple, with the tier name in the first cell:
//
//	| `doing` | apply, review-pr — ... | `claude` / `claude-opus-5` / `xhigh` |
var cellTriple = regexp.MustCompile(
	"^\\|\\s*`?(\\w+)`?\\s*\\|.*\\|\\s*`?([\\w-]+)`?\\s*/\\s*`?([\\w.-]+)`?\\s*/\\s*`?(\\w+)`?\\s*\\|\\s*$")

// exampleMarker flags a tier line that is an ILLUSTRATIVE OVERRIDE rather than a
// mirror of the built-in default. Such a line is expected to differ from
// defaultTiers (an example that matched the default would demonstrate nothing),
// so it is excluded from the comparison.
var exampleMarker = regexp.MustCompile(`#\s*(example|e\.g\.)`)

func TestMirrorDocsMatchDefaultTiers(t *testing.T) {
	known := make(map[string]bool, len(TierNames()))
	for _, tier := range TierNames() {
		known[tier] = true
	}

	totalChecked := 0
	for _, rel := range mirrorDocs {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			body, err := lines.ReadFileLines(findDocFile(t, rel))
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}

			checked := 0
			for i, line := range body {
				// Strip comment/indent decoration so a commented reference block
				// (e.g. the `# agent:` sample in a config fence) is still parsed.
				trimmed := strings.TrimSpace(line)
				trimmed = strings.TrimPrefix(trimmed, "# ")
				trimmed = strings.TrimSpace(trimmed)

				// An illustrative override is expected to differ from the default.
				if exampleMarker.MatchString(trimmed) {
					continue
				}

				tier, got, ok := parseMirrorLine(trimmed)
				if !ok || !known[tier] {
					continue
				}
				// An empty model is the deliberate "inherit the session model"
				// signal, not a mirror of a default — skip it.
				if got.Model == "" {
					continue
				}
				checked++

				want, _ := DefaultTier(tier)
				if got != want {
					t.Errorf("%s:%d mirrors tier %q as {%s, %s, %s}, but code defaultTiers says {%s, %s, %s} (doc drifted — defaultTiers in internal/agent/agent.go is canonical)",
						rel, i+1, tier,
						got.Provider, got.Model, got.Effort,
						want.Provider, want.Model, want.Effort)
				}
			}

			if checked == 0 {
				t.Errorf("%s is listed in mirrorDocs but no tier-profile line was found — either the doc stopped mirroring the defaults (remove it from mirrorDocs) or its formatting changed and this guard silently stopped covering it", rel)
			}
			totalChecked += checked
		})
	}

	// Backstop: a doc that mirrors the defaults at all mirrors ALL of them, so the
	// floor is one line per tier per doc. Without this, a regex that silently
	// stopped matching five of six lines would still leave the test green.
	if want := len(TierNames()) * len(mirrorDocs); totalChecked < want {
		t.Errorf("only %d tier-profile lines checked across %d doc(s), want at least %d (one per tier per doc) — the parsers likely stopped matching", totalChecked, len(mirrorDocs), want)
	}
}

// parseMirrorLine tries both mirror shapes, returning the tier and the profile
// the line asserts.
func parseMirrorLine(trimmed string) (string, Profile, bool) {
	if m := inlineTierYAML.FindStringSubmatch(trimmed); m != nil {
		return m[1], Profile{Provider: m[2], Model: m[3], Effort: m[4]}, true
	}
	if m := cellTriple.FindStringSubmatch(trimmed); m != nil {
		return m[1], Profile{Provider: m[2], Model: m[3], Effort: m[4]}, true
	}
	return "", Profile{}, false
}

// TestCLIFabReferenceListsDefaultTiers guards the prose enumeration in
// _cli-fab.md § resolve-agent, which restates every default as a
// `tier: provider/model/effort` run-on list rather than a table or YAML — a
// third shape, in the file the constitution requires to track CLI behavior.
func TestCLIFabReferenceListsDefaultTiers(t *testing.T) {
	const rel = "src/kit/skills/_cli-fab.md"
	body, err := lines.ReadFileLines(findDocFile(t, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	joined := strings.Join(body, "\n")

	for _, tier := range TierNames() {
		p, _ := DefaultTier(tier)
		want := fmt.Sprintf("`%s`: %s/%s/%s", tier, p.Provider, p.Model, p.Effort)
		if !strings.Contains(joined, want) {
			t.Errorf("%s does not contain %q — the built-in default enumeration drifted from defaultTiers", rel, want)
		}
	}
}
