package agent

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
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
// PROVIDER-AWARE (260806-ywkx). All three built-ins ship fills now, so a fill line
// is checked against the PROVIDER whose block it sits in — the parser tracks the
// enclosing `providers.<name>:` key and compares against
// ResolveProvider(nil, name).Profiles[role]. For claude that is exactly what
// DefaultProfile(role) returns, so this is a strict generalization of the earlier
// claude-only comparison; without it a live codex fill line would be checked against
// claude's table and fail spuriously. A fill line outside any provider block is
// skipped — there is no table to check it against.
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
// inline-YAML `providers.<name>.profiles` sample in § Three built-in providers,
// which since 260806-ywkx shows all three providers' shipped fills (covered here).
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
//	default:  { model: pro }                             # effort half absent
//
// The `, effort: <level>` half is OPTIONAL: gemini's fills carry no effort (that
// CLI has no reasoning-effort flag), and requiring it would leave exactly those
// lines unguarded.
//
// Leading "# " (commented reference blocks) is tolerated by the caller trimming it.
// The fill carries NO provider field — the provider is the map it hangs off (see
// the fill precedence in the package doc), so the expected provider comes from the
// enclosing block that providerKeyLine tracks, not from the line itself.
var inlineRoleYAML = regexp.MustCompile(
	`^(\w+):\s*\{\s*model:\s*([\w.-]*)\s*(?:,\s*effort:\s*(\w*)\s*)?\}`)

// inlineRoleEffortOnlyYAML matches the other half of a sparse map — a fill that
// tunes only the effort and inherits its model from the provider's `default` entry:
//
//	doing:   { effort: xhigh }
//
// It is a separate pattern rather than another optional group in inlineRoleYAML so
// that "no `model:` key at all" stays distinguishable from "`model:` present but
// empty", which is the deliberate inherit-the-session-model signal the caller skips.
var inlineRoleEffortOnlyYAML = regexp.MustCompile(
	`^(\w+):\s*\{\s*effort:\s*(\w*)\s*\}`)

// providerKeyLine matches the bare `<name>:` key that opens a provider's block in
// these samples (`  claude:`, `  # codex:`). The caller has already trimmed the
// indent and any comment prefix, and validates the captured name against the
// resolvable provider set — so a role line such as `doing:` (which also matches
// this shape) can never be mistaken for a provider key.
var providerKeyLine = regexp.MustCompile(`^([\w-]+):\s*$`)

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
	// The provider set the samples may open a block for, resolved once. A name
	// outside it is not a provider key (it is prose, or a role line).
	fills := make(map[string]map[string]config.ProviderProfile, len(ProviderNames(nil)))
	for _, name := range ProviderNames(nil) {
		prov, _ := ResolveProvider(nil, name)
		fills[name] = prov.Profiles
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
			provider := "" // the provider block the current line sits in
			for i, line := range body {
				// Strip comment/indent decoration so a commented reference block
				// (e.g. the `# providers:` sample in a config fence) is still parsed.
				trimmed := strings.TrimSpace(line)
				trimmed = strings.TrimPrefix(trimmed, "# ")
				trimmed = strings.TrimSpace(trimmed)

				// A bare `<name>:` naming a resolvable provider opens that
				// provider's block; every fill line below it is ITS fill until the
				// next such key. A fenced-code boundary ends the block, so a sample
				// that stops mid-provider cannot leak into the next one.
				if strings.HasPrefix(trimmed, "```") {
					provider = ""
					continue
				}
				if m := providerKeyLine.FindStringSubmatch(trimmed); m != nil {
					if _, ok := fills[m[1]]; ok {
						provider = m[1]
					}
					continue
				}

				// An illustrative override is expected to differ from the default.
				if exampleMarker.MatchString(trimmed) {
					continue
				}

				m := inlineRoleYAML.FindStringSubmatch(trimmed)
				if m == nil {
					// A sparse effort-only fill inherits its model from the
					// provider's `default` entry, so the mirror must show no model
					// either — asserting that is what keeps the sparseness itself
					// from drifting.
					m = inlineRoleEffortOnlyYAML.FindStringSubmatch(trimmed)
					if m == nil {
						continue
					}
					m = []string{m[0], m[1], "", m[2]}
				} else if m[2] == "" {
					// An empty model is the deliberate "inherit the session model"
					// signal, not a mirror of a default — skip it.
					continue
				}
				if !known[m[1]] || provider == "" {
					continue
				}
				role, gotModel, gotEffort := m[1], m[2], m[3]
				checked++

				want := fills[provider][role]
				if gotModel != want.Model || gotEffort != want.Effort {
					t.Errorf("%s:%d mirrors providers.%s.profiles.%s as {%s, %s}, but the resolved fill is {%s, %s} (doc drifted — defaults.yaml is canonical)",
						rel, i+1, provider, role, gotModel, gotEffort, want.Model, want.Effort)
				}
			}

			if checked == 0 {
				t.Errorf("%s is listed in mirrorDocs but no role-fill line was found — either the doc stopped mirroring the defaults (remove it from mirrorDocs) or its formatting changed and this guard silently stopped covering it", rel)
			}
			totalChecked += checked
		})
	}

	// Backstop: a doc that mirrors the built-in fills at all mirrors ALL of them, so
	// the floor is the total number of shipped fill entries per doc. Without this, a
	// regex that silently stopped matching most lines would still leave the test
	// green. Derived from the resolved tables, so adding a fill raises the floor.
	shipped := 0
	for _, profiles := range fills {
		shipped += len(profiles)
	}
	if want := shipped * len(mirrorDocs); totalChecked < want {
		t.Errorf("only %d role-fill lines checked across %d doc(s), want at least %d (one per shipped fill per doc) — the parsers likely stopped matching", totalChecked, len(mirrorDocs), want)
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
