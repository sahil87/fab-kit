// Package configref generates the fully-commented reference config.yaml that
// `fab config reference` prints to stdout. The reference is the single
// discoverability surface for the config.yaml schema — it documents BOTH the
// binary-consumed keys (modeled on internal/config.Config) and the
// skill-consumed keys (read by markdown skills, invisible to Go reflection).
//
// GENERATED, NOT HAND-WRITTEN. Every default/example value that has a canonical
// Go constant is injected from that constant — the default provider session
// command (agent.DefaultSessionCommand), the per-tier default profiles via
// agent.DefaultTier over agent.TierNames, and the pipeline stage names via
// agent.StageNames. There is no second copy of these values to drift, so no
// drift-guard test is needed for them (unlike the hand-written 2.2.0-to-2.3.0
// migration block this supersedes).
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
)

// tierDefault is one row of the injected default-tier table: the tier name, its
// fixed stage grouping (for the reference comment), and its default profile.
type tierDefault struct {
	Name     string
	Stages   string
	Provider string
	Model    string
	Effort   string
}

// refData carries every value injected into the reference template. Populated
// from the canonical constants in gatherData — nothing here is a literal that
// duplicates a Go constant.
type refData struct {
	SessionCommand string
	Tiers          []tierDefault
	Stages         []string
}

// tierStages is the human-readable stage grouping shown next to each tier in the
// reference comment. It restates the FIXED stage→tier mapping owned by
// internal/agent (default={intake advisory}, operator={fab operator},
// doing={apply,review-pr,hydrate}, review={review}, fast={ship}) — reference prose
// only, not a behavioral second source.
var tierStages = map[string]string{
	agent.TierDefault:  "intake (advisory), fab batch, fab agent",
	agent.TierOperator: "fab operator (coordinator session)",
	agent.TierDoing:    "apply, review-pr, hydrate",
	agent.TierReview:   "review",
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
			Name:     name,
			Stages:   stages,
			Provider: p.Provider,
			Model:    p.Model,
			Effort:   p.Effort,
		})
	}

	return refData{
		SessionCommand: agent.DefaultSessionCommand,
		Tiers:          tiers,
		Stages:         agent.StageNames(),
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

# providers — named agent invocation grammars. Each provider MAY carry two
# command fields (they are NOT merged — session and dispatch are different
# invocations of the same binary):
#   session_command  — opens an interactive agent SESSION (fab operator /
#                       fab batch / fab agent). {model}/{effort} placeholders are
#                       substituted from the resolved tier profile (the built-in
#                       claude default below is templated this way); a command
#                       carrying NO placeholder instead gets --model/--effort
#                       appended.
#   dispatch_command — runs ONE headless stage task via fab dispatch. ABSENT →
#                      native Agent-tool dispatch (the default). There is NO
#                      fallback from dispatch_command to session_command. fab
#                      dispatch pipes the stage prompt to the command's STDIN.
# Provider names are opaque, user-chosen strings — fab NEVER infers a provider
# from a model string. The one footgun (documented, not validated): if you
# override a tier's model to another provider, override that tier's provider too.
#
# fab-kit ships the claude provider as the built-in default (session_command
# shown LIVE below). codex and gemini are shown fully commented as a starter
# TEMPLATE — uncomment and adapt a block to add that provider. Anything whose
# uncommenting would change default BEHAVIOR ships commented: claude's
# dispatch_command (uncommenting flips claude's stages from native Agent-tool
# dispatch to headless CLI dispatch) and the whole codex/gemini blocks (opt-in
# providers). No new built-in providers are added in Go — codex/gemini are
# template text only until you uncomment them.
#
# Per-provider notes (kept out of the blocks below so uncommenting a whole block
# yields valid YAML — strip the leading '# ' from every line of a block):
#   claude.dispatch_command — claude -p reads the prompt from stdin; uncommenting
#     runs claude's stages as headless CLI processes instead of native sub-agents.
#   codex — codex exec reads the prompt from stdin. Substitute a current model ID
#     for {model} (e.g. gpt-5.3-codex); {model}/{effort} come from the tier.
#   gemini — no {effort} (the gemini CLI has no reasoning-effort flag) and no -p:
#     gemini's -p takes prompt TEXT (appended after stdin), whereas fab dispatch
#     pipes the prompt to stdin, which gemini reads as the prompt in non-TTY mode.
providers:
  claude:
    session_command: '{{ .SessionCommand }}'
    # dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'
  # codex:
  #   session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
  #   dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
  # gemini:
  #   session_command: 'gemini -m {model}'
  #   dispatch_command: 'gemini -m {model}'   # no {effort} flag; no -p (fab dispatch pipes the prompt to stdin)

# agent.tiers — per-stage model override. A "tier" is a named
# {provider, model, effort} profile (the invocation command lives on the provider,
# above — NOT on the tier). fab-kit owns the FIXED, non-overridable stage→tier
# mapping below; you override only WHAT EACH TIER MEANS. Omit any tier (or the
# whole tiers: block) to use fab-kit's built-in default. An omitted field within a
# tier inherits the project's ` + "`default`" + ` tier, then fab-kit's built-in; an empty
# model means "inherit the session model". Resolved per stage by
# ` + "`fab resolve-agent <stage>`" + ` at sub-agent dispatch time. Values pass through
# verbatim — fab does no provider validation. See docs/specs/stage-models.md.
#
# FIXED stage→tier mapping (fab-owned, NOT overridable — shown for reference):
{{- range .Tiers }}
#   {{ printf "%-9s" .Name }} {{ .Stages }}
{{- end }}
#
# fab-kit's built-in default profiles (today):
agent:
  tiers:
{{- range .Tiers }}
    {{ printf "%-9s" (printf "%s:" .Name) }} { provider: {{ .Provider }}, model: {{ .Model }}, effort: {{ .Effort }} }
{{- end }}

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
