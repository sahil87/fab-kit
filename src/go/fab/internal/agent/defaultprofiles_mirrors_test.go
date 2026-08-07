package agent

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
)

// The built-in per-role profiles are canonical in defaults.yaml.
// TestDocTablesMatchAgentMaps guards their ONE structured mirror — the 4-column
// table in docs/specs/stage-models.md. But the same values are restated in a
// looser inline-YAML shape that no test covered, so it drifted silently whenever a
// model was bumped:
//
//	inline YAML   default:  { model: claude-fable-5, effort: high }
//
// This file guards that shape across every doc that carries it. It is deliberately
// shape-driven rather than file-driven: a new doc that copies a role fill line is
// picked up by the same assertion set as soon as it is added to mirrorDocs.
//
// Scope note: only lines naming a KNOWN role are checked, and only where the
// line carries a model. Historical upgrade notes that quote a superseded default
// as past state are intentionally NOT in scope (they are prose about a prior
// release, not a mirror of today's table) — they live outside mirrorDocs.
//
// Two kinds of fill line deliberately do NOT mirror the defaults and are skipped
// (see exampleMarker): an illustrative OVERRIDE ("run ship cheaper") must differ
// from the default — that is the whole point of the example — and an empty model
// is the "inherit the session model" signal. Both are marked in-doc with an
// `# example:` / `# e.g.` trailing comment, which is the opt-out this guard reads.

// mirrorDocs are the docs that still restate the built-in per-role fills in the
// inline-YAML shape, and so must be checked against the resolved defaults.
//
// Kept deliberately SHORT: the fix for doc drift is usually to stop mirroring
// (point at `fab config reference`, which renders live) rather than to add another
// file here. docs/specs/architecture.md, docs/memory/_shared/configuration.md, and
// docs/memory/runtime/providers-and-profiles.md were all removed from this list once
// they were converted to pointers — they now show shape-only YAML and own the role
// TAXONOMY, not the values.
//
// stage-models.md stays: it is the one human-readable mirror of the values, in two
// shapes — the 4-column table (covered by TestDocTablesMatchAgentMaps) and the
// inline-YAML `providers.claude.profiles` sample (covered here).
//
// _cli-fab.md is deliberately absent: it carries only the third, run-on shape,
// guarded by TestCLIFabReferenceListsDefaultRoles below.
var mirrorDocs = []string{
	"docs/specs/stage-models.md",
}

// inlineRoleYAML matches an inline-YAML per-role fill line, tolerating the
// column-aligned padding these docs use:
//
//	doing:    { model: claude-opus-5,   effort: high }
//
// Leading "# " (commented reference blocks) is tolerated by the caller trimming it.
// The fill carries NO provider field — the provider is the map it hangs off (see
// the fill precedence in the package doc), so the expected provider is supplied by
// the resolved default profile rather than parsed from the line.
var inlineRoleYAML = regexp.MustCompile(
	`^(\w+):\s*\{\s*model:\s*([\w.-]*),\s*effort:\s*(\w*)\s*\}`)

// exampleMarker flags a fill line that is an ILLUSTRATIVE OVERRIDE rather than a
// mirror of the built-in default. Such a line is expected to differ from the
// shipped value (an example that matched the default would demonstrate nothing),
// so it is excluded from the comparison.
var exampleMarker = regexp.MustCompile(`#\s*(example|e\.g\.)`)

func TestMirrorDocsMatchDefaultProfiles(t *testing.T) {
	known := make(map[string]bool, len(RoleNames()))
	for _, role := range RoleNames() {
		known[role] = true
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
				// (e.g. the `# providers:` sample in a config fence) is still parsed.
				trimmed := strings.TrimSpace(line)
				trimmed = strings.TrimPrefix(trimmed, "# ")
				trimmed = strings.TrimSpace(trimmed)

				// An illustrative override is expected to differ from the default.
				if exampleMarker.MatchString(trimmed) {
					continue
				}

				m := inlineRoleYAML.FindStringSubmatch(trimmed)
				if m == nil || !known[m[1]] {
					continue
				}
				role, gotModel, gotEffort := m[1], m[2], m[3]
				// An empty model is the deliberate "inherit the session model"
				// signal, not a mirror of a default — skip it.
				if gotModel == "" {
					continue
				}
				checked++

				want, _ := DefaultProfile(role)
				if gotModel != want.Model || gotEffort != want.Effort {
					t.Errorf("%s:%d mirrors role %q as {%s, %s}, but the resolved default is {%s, %s} (doc drifted — defaults.yaml is canonical)",
						rel, i+1, role, gotModel, gotEffort, want.Model, want.Effort)
				}
			}

			if checked == 0 {
				t.Errorf("%s is listed in mirrorDocs but no role-fill line was found — either the doc stopped mirroring the defaults (remove it from mirrorDocs) or its formatting changed and this guard silently stopped covering it", rel)
			}
			totalChecked += checked
		})
	}

	// Backstop: a doc that mirrors the defaults at all mirrors ALL of them, so the
	// floor is one line per role per doc. Without this, a regex that silently
	// stopped matching five of six lines would still leave the test green.
	if want := len(RoleNames()) * len(mirrorDocs); totalChecked < want {
		t.Errorf("only %d role-fill lines checked across %d doc(s), want at least %d (one per role per doc) — the parsers likely stopped matching", totalChecked, len(mirrorDocs), want)
	}
}

// TestCLIFabReferenceListsDefaultRoles guards the prose enumeration in
// _cli-fab.md § resolve-agent, which restates every default as a
// `role: provider/model/effort` run-on list rather than a table or YAML — a
// third shape, in the file the constitution requires to track CLI behavior.
func TestCLIFabReferenceListsDefaultRoles(t *testing.T) {
	const rel = "src/kit/skills/_cli-fab.md"
	body, err := lines.ReadFileLines(findDocFile(t, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	joined := strings.Join(body, "\n")

	for _, role := range RoleNames() {
		p, _ := DefaultProfile(role)
		want := fmt.Sprintf("`%s`: %s/%s/%s", role, p.Provider, p.Model, p.Effort)
		if !strings.Contains(joined, want) {
			t.Errorf("%s does not contain %q — the built-in default enumeration drifted from defaults.yaml", rel, want)
		}
	}
}
