// Package configref generates the fully-commented reference config.yaml that
// `fab config reference` prints to stdout. The reference is the single
// discoverability surface for the config.yaml schema — it documents BOTH the
// binary-consumed keys (modeled on internal/config.Config) and the
// skill-consumed keys (read by markdown skills, invisible to Go reflection).
//
// GENERATED, NOT HAND-WRITTEN. Every default/example value that has a canonical
// Go constant is injected from that constant — spawn.DefaultSpawnCommand, the
// per-tier default profiles via agent.DefaultTier over agent.TierNames, and the
// pipeline stage names via agent.StageNames. There is no second copy of these
// values to drift, so no drift-guard test is needed for them (unlike the
// hand-written 2.2.0-to-2.3.0 migration block this supersedes).
//
// Render() output is BYTE-STABLE for a given binary version: the template body
// is fixed, the injected tier/stage lists come from the already-sorted
// agent.TierNames/agent.StageNames accessors, and no map is range-iterated
// during rendering.
package configref

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
)

// tierDefault is one row of the injected default-tier table: the tier name, its
// fixed stage grouping (for the reference comment), and its default profile.
type tierDefault struct {
	Name   string
	Stages string
	Model  string
	Effort string
}

// refData carries every value injected into the reference template. Populated
// from the canonical constants in gatherData — nothing here is a literal that
// duplicates a Go constant.
type refData struct {
	SpawnCommand string
	Tiers        []tierDefault
	Stages       []string
}

// tierStages is the human-readable stage grouping shown next to each tier in the
// reference comment. It restates the FIXED stage→tier mapping owned by
// internal/agent (thinking={intake,review}, doing={apply,review-pr,hydrate},
// fast={ship}) — reference prose only, not a behavioral second source.
var tierStages = map[string]string{
	agent.TierThinking: "intake, review",
	agent.TierDoing:    "apply, review-pr, hydrate",
	agent.TierFast:     "ship",
}

// gatherData builds the injected data from the canonical constants. Ordering is
// deterministic: agent.TierNames and agent.StageNames both return sorted slices,
// so the rendered output is byte-stable across invocations.
//
// It fails loudly on a broken invariant rather than silently emitting a degraded
// reference: agent.DefaultTier must know every tier agent.TierNames reports, and
// tierStages (a separate map maintained here) must carry a stage grouping for
// each tier. Adding a tier to agent.defaultTiers without a matching tierStages
// entry — the one drift these two maps allow — is caught here, not shipped as an
// empty grouping in the reference.
func gatherData() (refData, error) {
	tiers := make([]tierDefault, 0, len(agent.TierNames()))
	for _, name := range agent.TierNames() {
		p, ok := agent.DefaultTier(name)
		if !ok {
			return refData{}, fmt.Errorf("configref: tier %q from agent.TierNames has no agent.DefaultTier profile", name)
		}
		stages, ok := tierStages[name]
		if !ok {
			return refData{}, fmt.Errorf("configref: tier %q has no tierStages grouping (add one when adding a tier)", name)
		}
		tiers = append(tiers, tierDefault{
			Name:   name,
			Stages: stages,
			Model:  p.Model,
			Effort: p.Effort,
		})
	}

	return refData{
		SpawnCommand: spawn.DefaultSpawnCommand,
		Tiers:        tiers,
		Stages:       agent.StageNames(),
	}, nil
}

// referenceTemplate is the fixed body of the reference config.yaml. Baseline
// keys every project sets appear live with example/default values; the opt-in
// override blocks (agent.tiers, stage_hooks, branch_prefix) appear commented-out
// with fab-kit's defaults shown — mirroring the 2.2.0-to-2.3.0 style so
// uncommenting is opting in. fab_version is documented as machine-managed.
//
// Injection points ({{ . }}) draw exclusively on refData (constant-sourced).
const referenceTemplate = `# Full reference of all available options: fab config reference
#
# This is the canonical, generated reference for fab/project/config.yaml. Every
# key below is documented — baseline keys appear live with example values;
# optional override blocks (agent.tiers, stage_hooks, branch_prefix) are shown
# commented-out with fab-kit's built-in defaults. Uncomment a block to opt in.
# Values here are examples/defaults, not your project's settings.

# project — identity and PR metadata.
project:
  name: "My Project"                 # skills: orientation, PR bodies
  description: "One-line project description"  # skills: orientation, PR bodies
  # linear_workspace: myteam         # optional — enables Linear issue links in
                                     # PR bodies (https://linear.app/<slug>/issue/<ID>).
                                     # Omit or leave null for bare issue-ID text.

# source_paths — directories containing implementation code (relative to repo
# root). Read by skills to scope apply context.
source_paths:
  - src/

# test_paths — glob/pathspec patterns identifying test files. Used by the
# /git-pr true-impact breakdown to attribute lines to tests vs. implementation
# (impl = total − tests). Language-specific — no kit default. Patterns are
# :(glob) magic pathspecs, so ` + "`**`" + ` matches across directories and ` + "`*`" + ` does
# NOT match ` + "`/`" + `. When absent/empty, the breakdown collapses to a single total.
test_paths:
  - "**/*_test.go"                   # example (Go — ` + "`_test.go`" + ` suffix)

# true_impact_exclude — pathspec exclusions used by /git-pr to compute the "true
# impact" line counts in the PR body (excluding noise directories). Optional —
# when absent or empty, the impact block is omitted.
true_impact_exclude:
  - fab/
  - docs/

# checklist.extra_categories — project-specific quality categories appended to
# the built-in plan.md ## Acceptance categories (functional_completeness,
# behavioral_correctness, scenario_coverage, edge_cases, code_quality, security).
# Each becomes an extra subsection under plan.md ## Acceptance.
checklist:
  extra_categories: []               # example: [performance, accessibility, i18n]

# review_tools — automated PR reviewer toggles consumed by /git-pr-review. An
# absent key defaults to enabled. (codex/claude are legacy keys, silently
# ignored by /git-pr-review Phase 2; the pre-ship review-stage cascade is not
# configurable here.)
review_tools:
  claude: true
  codex: true
  copilot: true

# agent — agent spawn command and the optional per-stage model override.
agent:
  # spawn_command — base command used by fab operator / fab batch /
  # fab spawn-command to spawn agent sessions. Shell expansions (e.g.
  # $(basename "$(pwd)")) expand at invocation time. Optional {model}/{effort}
  # placeholders make it provider-forgiving: when either is present, the
  # resolved profile is SUBSTITUTED in place (e.g.
  # 'codex -m {model} -c model_reasoning_effort={effort}') and an empty value
  # drops the placeholder's token plus a preceding -flag; with no placeholder,
  # Claude-style --model/--effort are appended. Falls back to
  # ` + "`{{ .SpawnCommand }}`" + ` when absent.
  spawn_command: 'claude --dangerously-skip-permissions --effort xhigh -n "$(basename "$(pwd)")"'

  # agent.tiers — per-stage model override (optional). A "tier" is a named
  # {model, effort} profile. fab-kit owns the FIXED, non-overridable stage→tier
  # mapping below; you override only WHAT EACH TIER MEANS (model + effort).
  # Omit any tier (or the whole tiers: block) to use fab-kit's built-in default.
  # An omitted field within a tier inherits that tier's default; an empty model
  # means "inherit the session model". Resolved per stage by
  # ` + "`fab resolve-agent <stage>`" + ` at sub-agent dispatch time. Values pass through
  # verbatim — fab does no provider validation. See docs/specs/stage-models.md.
  #
  # FIXED stage→tier mapping (fab-owned, NOT overridable — shown for reference):
{{- range .Tiers }}
  #   {{ printf "%-9s" .Name }} {{ .Stages }}
{{- end }}
  #
  # fab-kit's built-in default profiles (today):
{{- range .Tiers }}
  #   {{ printf "%-9s" .Name }} { model: {{ .Model }}, effort: {{ .Effort }} }
{{- end }}
  #
  # Override shape (uncomment + edit any tier):
  # tiers:
  #   doing: { model: claude-sonnet-4-6, effort: medium }   # example: run doing cheaper

# stage_hooks — optional per-stage pre/post shell commands honored by
# ` + "`fab status start`" + ` / ` + "`fab status finish`" + `. Each command runs as ` + "`sh -c`" + ` from
# the repo root. A failing ` + "`pre`" + ` hook blocks ` + "`fab status start`" + `; a ` + "`post`" + ` hook
# runs after ` + "`finish`" + ` saves. Not seeded by the scaffold — add by hand.
# Valid stage keys: {{ range $i, $s := .Stages }}{{ if $i }}, {{ end }}{{ $s }}{{ end }}.
# stage_hooks:
#   apply:
#     pre: ./scripts/check-clean-tree.sh
#     post: make test

# branch_prefix — optional prefix applied by ` + "`fab batch switch`" + ` when creating
# worktree branches (branch name = ` + "`{branch_prefix}{folder_name}`" + `). Empty when
# absent.
# branch_prefix: ""

# fab_version — MACHINE-MANAGED: the project-pinned engine version the router
# uses to resolve which cached fab-go binary to exec. Set by ` + "`fab init`" + ` and
# ` + "`fab upgrade-repo`" + ` — do NOT hand-edit.
fab_version: "0.0.0"
`

var tmpl = template.Must(template.New("configref").Parse(referenceTemplate))

// Render returns the fully-commented reference config.yaml. The output is
// byte-stable for a given binary version and is what `fab config reference`
// prints to stdout. It returns an error rather than emitting partial/degraded
// output if an invariant breaks: gatherData surfaces a tier map drift, and
// tmpl.Execute surfaces a template/data mismatch (both are today unreachable
// given template.Must at init and the constant-sourced data, but are propagated
// so a future edit that breaks the invariant fails loudly instead of silently
// shipping a malformed reference).
func Render() (string, error) {
	data, err := gatherData()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("configref: rendering reference template: %w", err)
	}
	return buf.String(), nil
}
