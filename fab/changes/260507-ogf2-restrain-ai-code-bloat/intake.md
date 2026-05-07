# Intake: Restrain AI-driven code bloat

**Change**: 260507-ogf2-restrain-ai-code-bloat
**Created**: 2026-05-07
**Status**: Draft

## Origin

> User raised this in `/fab-discuss`: "AI is great at adding code, but not so good at removing code. This leads to extremely large code bases. When humans write code they're extremely conscious of the volume of code because of the physical limitations they're bound with. One happy side effect is that the volume of code remains very controlled. I want a way to impose this restriction on AI also."

Mode: conversational. After the assistant proposed five possible interventions (parsimony pass in review, net-line accounting in `.status.yaml`, deletion-candidate prompt at hydrate, constitution-level rule, "shrink" change type), the user selected **#1 + #2 + #3 combined** as the scope for this intake. The remaining two (constitution rule, new change type) are explicitly out of scope for this change — they may follow as a separate change once we see whether interventions 1–3 move the needle.

The motivating concern: large codebases hamper everyone's ability to find root causes fast — humans and AI alike. Without a governor on additive growth, fab-driven projects will accumulate code faster than humans can prune it.

## Why

**Problem**: AI agents (including the ones driving fab) reliably add code to satisfy a spec but rarely remove existing code that the new code makes redundant. There is no physical typing-cost governor, no review pressure to "make it smaller," and no measurement infrastructure that surfaces growth as a signal worth attending to. Over many changes, this compounds.

**Consequence if unfixed**:
1. Codebases grow monotonically. Even AI's own ability to navigate, reason about, and root-cause issues degrades as code volume rises.
2. Refactor changes that *should* shrink the codebase silently grow it instead, because nobody is watching the line-delta number.
3. Dead code accumulates because the agent that wrote the *replacement* never asks "what did this make redundant?"
4. The healthy human instinct of "could I do this with less code?" is absent from the AI loop.

**Why this approach (the three interventions, combined)**:
- **#1 Parsimony pass** intervenes at *review time* — the cheapest moment to catch over-additive code, before it ships. Adversarial framing ("could the spec be satisfied with less?") is different from the existing review focus ("is this correct?").
- **#2 Net-line accounting** creates the *measurement infrastructure*. Without numbers, there's no signal. Surfacing per-change net deltas in `/fab-status` and flagging suspicious patterns (e.g., refactor changes that grow code) is what makes #1 and #3 meaningful over time.
- **#3 Deletion-candidate prompt** intervenes at *hydrate time* — the last moment before the change is sealed. A single prompt ("what existing code did this change make redundant?") is cheap, occasionally produces real findings, and (more importantly) trains the agent loop to think about deletion as a first-class output.

These three are mutually reinforcing: #2 produces the numbers, #1 and #3 produce the prompts that act on them. Doing any one alone would be weaker.

The user explicitly rejected hard line-count gates: they incentivize gaming (one-line stuffing, whitespace removal) and don't reflect engineering judgment. Soft signals + reviewer judgment is the chosen approach.

## What Changes

### Intervention #1: Parsimony pass in review sub-agent

**Location**: `.claude/skills/_review/SKILL.md` (the shared review behavior used by `/fab-continue` review stage and `/fab-fff` / `/fab-ff`).

**Behavior**:
- Add a new review pass (or new section within the existing pass) that evaluates the apply-stage diff against a parsimony lens.
- Specific question the reviewer agent must answer: *"Could the spec's requirements be satisfied with less code? Identify candidates for: (a) reusing existing utilities instead of new code, (b) collapsing duplication, (c) removing dead branches, (d) removing now-redundant existing code."*
- Findings classified per existing severity tiers (`code-review.md`):
  - Reusing an existing utility you didn't notice → **Should-fix**
  - New code that has zero call sites in the diff → **Must-fix**
  - Duplicated logic added alongside existing implementation → **Must-fix**
  - Verbosity / redundant defensive checks → **Nice-to-have**
- The pass MUST cite specific file paths and line ranges. No abstract "the code could be smaller" findings.

**Configuration**: A new optional section in `fab/project/code-review.md`:

```markdown
## Parsimony Pass

- Enabled: true  (default; set to false to skip)
- Threshold: flag when net additions exceed 100 lines without explicit spec justification
```

The threshold is advisory, not a gate.

### Intervention #2: Net-line accounting per change

**Schema addition to `.status.yaml`**:

```yaml
line_stats:
  added: 142
  deleted: 38
  net: +104
  excluding_fab_docs:
    added: 87
    deleted: 38
    net: +49
  computed_at: 2026-05-07T14:32:00Z
  computed_at_stage: apply  # the stage at which this snapshot was taken
```

Fields:
- `added` / `deleted` / `net`: raw `git diff --shortstat` from merge-base to current HEAD.
- `excluding_fab_docs`: the same numbers with `fab/` and `docs/` excluded — this overlaps with the sister intake (`260507-asvz`). The two intakes SHOULD coordinate on a single source-of-truth helper that both consume.
- `computed_at` / `computed_at_stage`: bookkeeping for staleness detection.

**When computed**:
- At end of apply stage (`fab status finish <change> apply` runs the helper).
- Recomputed at hydrate stage to capture any review-stage edits.

**Surfaced in**:
- `/fab-status` — adds a "Code delta" line under the existing change summary, showing net addition with a yellow highlight when net > 100 (raw) or net > 50 (excluding fab/docs).
- `fab change list` — adds an optional column for net delta, behind a `--show-stats` flag to keep the default view compact.

**Flagging suspicious patterns**:
- A `refactor`-typed change with net > +50 lines (excluding fab/docs) → surfaced in `/fab-status` as a soft warning: "Refactor changes typically shrink or stay flat — review whether this growth is intentional."
- The flag is informational. No gate, no block.

### Intervention #3: Deletion-candidate prompt at hydrate

**Location**: `.claude/skills/_review/SKILL.md` hydrate sub-stage, OR `.claude/skills/fab-continue/SKILL.md` hydrate behavior — to be decided at spec stage based on which file owns the hydrate prompt today.

**Behavior**:
- Before the hydrate agent finalizes memory updates, it MUST answer the question: *"What existing code (files, functions, branches, config) did this change make redundant or unused? List candidates for removal in a `## Deletion Candidates` section."*
- The output is a structured list: each candidate names a specific symbol, file path, or block, with a one-line justification.
- The agent is permitted to answer "None — this change adds new functionality without making existing code redundant" when truthful. The prompt's value is in *forcing the question*, not in always producing answers.
- Candidates are written to `fab/changes/{name}/deletion-candidates.md` (a new artifact, parallel to spec/tasks/checklist).
- The hydrate agent does NOT auto-delete. The artifact is a prompt for the human reviewer to act on, either in the same PR or a follow-up `chore` change.

**Why a separate artifact, not a section in spec.md**: the spec is pre-implementation design intent (per constitution principle VI). Deletion candidates are a post-implementation finding that should not retroactively pollute the spec.

### Coordination with sister intake (`260507-asvz`)

Both this change and the sister change need a "diff stats excluding fab/docs" helper. To avoid duplication:
- The first of the two changes to merge SHOULD introduce a small helper (e.g., `fab impact stats <base> <head>`) that reads `pr_impact_exclude` from config and returns shortstat numbers.
- The second change reuses it.

The order of merging is not load-bearing — whichever merges first wins the helper; the other gets refactored to consume it.

## Affected Memory

- `fab-workflow/clarify`: (modify) — wait, this is review, not clarify. Removing.
- `fab-workflow/execution-skills`: (modify) Document the parsimony pass as a review sub-behavior.
- `fab-workflow/hydrate`: (modify) Document the deletion-candidate prompt and the new `deletion-candidates.md` artifact.
- `fab-workflow/schemas`: (modify) Document the new `line_stats` block in `.status.yaml`.
- `fab-workflow/configuration`: (modify) Document the new `parsimony` section in `code-review.md`.
- `fab-workflow/templates`: (modify) Document the new `deletion-candidates.md` artifact template.

## Impact

**Code areas touched**:
- `src/kit/skills/_review/SKILL.md` — parsimony pass + deletion-candidate prompt
- `src/kit/skills/fab-status/SKILL.md` — surface line_stats
- `src/kit/skills/fab-continue/SKILL.md` — possibly the hydrate prompt insertion point
- `src/kit/templates/status.yaml` — add `line_stats` block initialized empty
- `src/kit/templates/deletion-candidates.md` — new template
- `src/cmd/fab/status.go` (or wherever `fab status finish` lives) — invoke the line-stats computation hook
- `src/cmd/fab/` — possibly a new subcommand `fab line-stats` or `fab impact stats` for the helper
- `docs/specs/skills/SPEC-fab-continue.md`, `SPEC-fab-status.md`, `SPEC-_review.md` — spec updates per constitution rule
- `docs/memory/fab-workflow/*` — memory updates per Affected Memory section

**APIs / contracts**:
- `.status.yaml` schema gains `line_stats` block. Backwards-compatible — consumers without awareness of the field ignore it.
- New artifact: `deletion-candidates.md` per change. Optional — empty (or absent) when no candidates.
- New optional section in `code-review.md`: `## Parsimony Pass`. Backwards-compatible.

**Dependencies**:
- `git` (already required) for shortstat.
- `yq` for config parsing.
- No new binaries.

**Migration**:
- Existing `.status.yaml` files do not have `line_stats` — handled gracefully (computed lazily at next stage transition).
- Existing `code-review.md` files do not have the parsimony section — defaults to enabled with default threshold.
- No destructive migration required.

## Open Questions

- Where exactly does the deletion-candidate prompt live — inside the existing review sub-agent dispatch, or as its own hydrate-stage step? Spec stage to decide.
- Should `line_stats` also track the parsimony-flagged threshold violations (e.g., a `parsimony_flags` count)? Possibly redundant with the review report itself.
- Should `/fab-status` display growth trends across multiple changes (e.g., "this branch has added net +1200 lines across 3 changes")? Out of scope for v1; defer.
- Is the threshold for "suspicious refactor growth" fixed at +50 (excluding fab/docs), or configurable per-project? Lean toward configurable but with a sensible default.
- Should the parsimony pass run on `docs`/`chore` changes? Probably no value — those changes are inherently additive. Spec stage to decide a type-based skip list.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is limited to interventions #1, #2, #3 from the discussion | Discussed — user explicitly chose these three; the other two (constitution rule, shrink change type) are out of scope | S:95 R:80 A:90 D:95 |
| 2 | Certain | No hard line-count gates — soft signals only | Discussed — user explicitly rejected hard gates due to gaming risk | S:95 R:90 A:95 D:95 |
| 3 | Confident | Parsimony pass lives in `_review` skill, integrated with existing severity tiers | Existing code-review infrastructure already classifies findings; parsimony fits naturally | S:80 R:75 A:85 D:75 |
| 4 | Confident | `line_stats` block goes in `.status.yaml` (not a separate file) | Status.yaml is the existing per-change state container; consistent with `confidence` and `checklist` blocks | S:80 R:75 A:85 D:80 |
| 5 | Confident | Deletion candidates go in a new `deletion-candidates.md` artifact, not in `spec.md` | Constitution principle VI: spec is pre-implementation; deletion candidates are post-implementation findings | S:85 R:75 A:90 D:80 |
| 6 | Confident | The "excluding fab/docs" stats are computed using the same helper as the sister intake (260507-asvz) | DRY principle; both intakes need the same git command | S:80 R:80 A:85 D:80 |
| 7 | Confident | Refactor-type changes with net > +50 (excluding fab/docs) get a soft warning in `/fab-status` | Refactors should generally shrink or stay flat; this is the cheapest detection signal | S:75 R:85 A:80 D:70 |
| 8 | Tentative | Parsimony pass threshold default: 100 net added lines without spec justification | Plausible default; spec stage to validate against real change history | S:55 R:85 A:60 D:55 |
| 9 | Tentative | Refactor warning threshold: +50 net (excluding fab/docs) | Conservative default; might need tuning after observing real changes | S:50 R:85 A:60 D:55 |
| 10 | Tentative | Deletion-candidate prompt is mandatory at hydrate (cannot be skipped) | Forcing the question is the value; allowing skip undermines the intervention | S:60 R:75 A:65 D:55 |
| 11 | Tentative | Parsimony pass is skipped for `docs`/`chore`/`ci` change types | These types are inherently additive; pass would produce noise | S:55 R:85 A:65 D:55 |
| 12 | Unresolved | Where does the deletion-candidate prompt physically live — inside `_review` hydrate sub-stage or as a separate `_hydrate` step? | Asked — spec stage to decide based on existing skill structure | S:30 R:55 A:50 D:35 |
| 13 | Unresolved | Should `/fab-status` show cross-change cumulative deltas (e.g., branch-level totals)? | Asked — defer or include? Lean defer-to-v2, but flagging for spec-stage decision | S:40 R:75 A:55 D:50 |

13 assumptions (2 certain, 5 confident, 4 tentative, 2 unresolved).
