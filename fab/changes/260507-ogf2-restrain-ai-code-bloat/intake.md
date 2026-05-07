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
- **#3 Deletion-candidate prompt** intervenes at *review time* — co-located with the parsimony pass. A single prompt ("what existing code did this change make redundant?") is cheap, occasionally produces real findings, and (more importantly) trains the agent loop to think about deletion as a first-class output. Generating it at review (not hydrate) keeps it in the diff-critique cognitive mode and surfaces findings before the change is sealed.

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

**Configuration**: A single optional toggle in `fab/project/code-review.md`:

```markdown
## Parsimony Pass

- Enabled: true  (default; set to false to skip)
```

Thresholds and the change-type skip list are hard-coded in the kit (not project-configurable):
- Parsimony pass threshold: flag when net additions exceed **100 lines** without explicit spec justification
- Refactor-growth warning threshold: **+50 net lines** (excluding fab/docs)
- Skip list (no parsimony pass, no deletion-candidate prompt): `docs`, `chore`, `ci`

These can be revisited if real-world usage shows the defaults are wrong, but until then a single set of numbers across projects keeps the surface area small. The threshold is advisory, not a gate.

### Intervention #2: Net-line accounting per change

**Schema addition to `.status.yaml`**:

```yaml
true_impact:
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

### Intervention #3: Deletion-candidate prompt at review

**Location**: `.claude/skills/_review/SKILL.md` — co-located with the parsimony pass (intervention #1). Both interventions are diff-critique passes asking "what could be smaller / what's now redundant?"

**Behavior**:
- During review (after apply, before hydrate), the review sub-agent MUST answer: *"What existing code (files, functions, branches, config) did this change make redundant or unused? List candidates for removal in a `## Deletion Candidates` section."*
- The output is a structured list: each candidate names a specific symbol, file path, or block, with a one-line justification.
- The agent is permitted to answer "None — this change adds new functionality without making existing code redundant" when truthful. The prompt's value is in *forcing the question*, not in always producing answers.
- Candidates are appended as a new top-level `## Deletion Candidates` section in `plan.md` (the as-built artifact already produced at apply stage). No new artifact file.
- The review agent does NOT auto-delete. The section is a prompt for the human reviewer to act on, either in the same PR or a follow-up `chore` change.
- Hydrate stage reads `## Deletion Candidates` from `plan.md`, so memory updates can reference findings.
- The section is distinct from `## Acceptance > ### Removal Verification` (which covers *planned* removals from spec.md). Deletion candidates are *discovered* removal opportunities that the apply agent missed.

**Why a section in `plan.md`, not a separate artifact and not in spec.md**:
- Not in `spec.md`: spec is pre-implementation design intent (per constitution principle VI). Discovered deletion candidates are post-implementation findings — different cognitive mode, different stage, different ownership. *Planned* deletions (known at spec time) still go in spec.md as normal "remove X" requirements; only discovered ones land here.
- Not a separate artifact: `plan.md` already records the as-built state for this change (Tasks + Acceptance + Notes). Adding a new file per change for a single section would proliferate artifacts — exactly the bloat-restraint behavior we're trying to instill. Co-locating with the apply-stage artifact keeps the change folder tidy.

**Why review (not hydrate)**: deletion-candidate identification is a critique of the diff — same cognitive mode as the parsimony pass. Hydrate is about updating memory, a different job. Co-locating with #1 also enables the review agent to share context across both passes.

### Coordination with sister intake (`260507-asvz`)

Both this change and the sister change need a "diff stats excluding fab/docs" helper. To avoid duplication:
- The first of the two changes to merge SHOULD introduce a small helper (e.g., `fab impact stats <base> <head>`) that reads `pr_impact_exclude` from config and returns shortstat numbers.
- The second change reuses it.

The order of merging is not load-bearing — whichever merges first wins the helper; the other gets refactored to consume it.

## Affected Memory

- `fab-workflow/clarify`: (modify) — wait, this is review, not clarify. Removing.
- `fab-workflow/execution-skills`: (modify) Document the parsimony pass as a review sub-behavior.
- `fab-workflow/execution-skills`: also documents the deletion-candidate prompt as a review sub-behavior (folded into the entry above).
- `fab-workflow/hydrate`: (modify) Document that hydrate reads `deletion-candidates.md` when present.
- `fab-workflow/schemas`: (modify) Document the new `true_impact` block in `.status.yaml`.
- `fab-workflow/configuration`: (modify) Document the new `parsimony` section in `code-review.md`.
- `fab-workflow/templates`: (modify) Document the new `## Deletion Candidates` section in `plan.md`.

## Impact

**Code areas touched**:
- `src/kit/skills/_review/SKILL.md` — parsimony pass + deletion-candidate prompt
- `src/kit/skills/fab-status/SKILL.md` — surface true_impact
- `src/kit/skills/fab-continue/SKILL.md` — possibly the hydrate prompt insertion point
- `src/kit/templates/status.yaml` — add `true_impact` block initialized empty
- `src/kit/templates/plan.md` — add new `## Deletion Candidates` parser-contract section (no new template file)
- `src/cmd/fab/status.go` (or wherever `fab status finish` lives) — invoke the line-stats computation hook
- `src/cmd/fab/` — possibly a new subcommand `fab line-stats` or `fab impact stats` for the helper
- `docs/specs/skills/SPEC-fab-continue.md`, `SPEC-fab-status.md`, `SPEC-_review.md` — spec updates per constitution rule
- `docs/memory/fab-workflow/*` — memory updates per Affected Memory section

**APIs / contracts**:
- `.status.yaml` schema gains `true_impact` block. Backwards-compatible — consumers without awareness of the field ignore it.
- `plan.md` gains a new top-level `## Deletion Candidates` section (parser contract). Optional — present with "None" when no candidates, omitted entirely when the change type is in the skip list.
- New optional section in `code-review.md`: `## Parsimony Pass`. Backwards-compatible.

**Dependencies**:
- `git` (already required) for shortstat.
- `yq` for config parsing.
- No new binaries.

**Migration**:
- Existing `.status.yaml` files do not have `true_impact` — handled gracefully (computed lazily at next stage transition).
- Existing `code-review.md` files do not have the parsimony section — defaults to enabled with default threshold.
- No destructive migration required.

## Open Questions

- ~~Where exactly does the deletion-candidate prompt live~~ — resolved: lives in `_review` alongside the parsimony pass.
- Should `true_impact` also track the parsimony-flagged threshold violations (e.g., a `parsimony_flags` count)? Possibly redundant with the review report itself.
- ~~Should `/fab-status` display growth trends across multiple changes~~ — resolved: dropped. Per-change net is sufficient signal; branch-level rollup is overreach.
- ~~Is the refactor-growth threshold configurable per-project~~ — resolved: hard-coded at +50 (excluding fab/docs).
- ~~Should the parsimony pass run on `docs`/`chore` changes~~ — resolved: hard-coded skip list of `docs`/`chore`/`ci`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is limited to interventions #1, #2, #3 from the discussion | Discussed — user explicitly chose these three; the other two (constitution rule, shrink change type) are out of scope | S:95 R:80 A:90 D:95 |
| 2 | Certain | No hard line-count gates — soft signals only | Discussed — user explicitly rejected hard gates due to gaming risk | S:95 R:90 A:95 D:95 |
| 3 | Certain | Parsimony pass lives in `_review` skill, integrated with existing severity tiers | Clarified — user confirmed | S:95 R:75 A:85 D:75 |
| 4 | Certain | `true_impact` block goes in `.status.yaml` (not a separate file); renamed from `line_stats` | Clarified — user confirmed and renamed for clarity (it's the true-impact signal, not just raw line stats) | S:95 R:75 A:85 D:80 |
| 5 | Certain | Deletion candidates go in a new `## Deletion Candidates` section of `plan.md` (not a separate artifact, not in spec.md) | Clarified — user reframed: discovered (vs planned) deletions are post-implementation findings; co-locate with the existing as-built artifact (`plan.md`) rather than creating a new file. Avoids artifact proliferation, which is exactly the bloat we're restraining | S:95 R:75 A:90 D:80 |
| 6 | Confident | The "excluding fab/docs" stats are computed using the same helper as the sister intake (260507-asvz). Helper signature is locked in once asvz merges; this intake refactors to consume it before final spec | User confirmed pending asvz merge — coordination resolved at that point | S:80 R:80 A:85 D:80 |
| 7 | Certain | Refactor-type changes with net > +50 (excluding fab/docs) get a soft warning in `/fab-status` | Clarified — user confirmed | S:95 R:85 A:80 D:70 |
| 8 | Certain | Parsimony pass threshold: 100 net added lines (hard-coded in kit, not project-configurable) | Clarified — user chose hard-coded defaults; revisit if real-world usage shows the value is wrong | S:95 R:85 A:60 D:55 |
| 9 | Certain | Refactor warning threshold: +50 net excluding fab/docs (hard-coded in kit) | Clarified — user chose hard-coded defaults; consistent with #8 | S:95 R:85 A:60 D:55 |
| 10 | Certain | Deletion-candidate prompt shares the parsimony pass's skip list (skipped for `docs`/`chore`/`ci` change types) | Clarified — user confirmed: consistent config surface, both passes are diff-critique with the same applicability profile | S:95 R:75 A:65 D:55 |
| 11 | Certain | Parsimony pass and deletion-candidate prompt skipped for `docs`/`chore`/`ci` change types (hard-coded in kit) | Clarified — user chose hard-coded skip list; both passes share the list per #10 | S:95 R:85 A:65 D:55 |
| 12 | Certain | Deletion-candidate prompt lives in `_review` skill, co-located with the parsimony pass (review stage, not hydrate) | Clarified — user confirmed: deletion-candidate identification is a diff-critique task, same cognitive mode as parsimony; hydrate reads the artifact but doesn't generate it | S:95 R:55 A:50 D:35 |
| 13 | Certain | No cross-change cumulative deltas in `/fab-status` (per-change net only) | Clarified — user dropped the feature: per-change signal is sufficient, branch-level rollup is overreach | S:95 R:75 A:55 D:50 |

13 assumptions (12 certain, 1 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-05-07

| # | Question | Resolution |
|---|----------|------------|
| 12 | Where does the deletion-candidate prompt live? | Moved from hydrate to review stage, co-located with parsimony pass in `_review`. User noted it's a diff-critique task and belongs in review's cognitive mode. Hydrate reads the artifact when present. |
| 10 | Is the deletion-candidate prompt mandatory? | Skippable via the same skip list as the parsimony pass (skips `docs`/`chore`/`ci`). Single shared config surface. |
| 13 | Cross-change cumulative deltas in /fab-status? | Dropped. Per-change net is sufficient; branch-level rollup is overreach. |
| 8, 9, 11 | Configurable thresholds and skip list, or hard-coded? | All three hard-coded in the kit at current defaults (100 lines / +50 refactor / docs+chore+ci). Revisit if real-world usage shows the values are wrong. `code-review.md` keeps only the on/off toggle. |
| 4 | Confirmed; renamed `line_stats` → `true_impact` in `.status.yaml` to better reflect intent (the diff stats *are* the true-impact signal, not raw line stats). |
| 6 | Confirmed pending asvz merge; signature of the shared helper locked when asvz lands and this intake's spec consumes it. |
| 3, 7 | Confirmed as-is. |
| 5 | Reframed: separated *planned* deletions (stay in spec.md) from *discovered* deletions (new `## Deletion Candidates` section in `plan.md`). No new artifact file — co-locating with `plan.md` keeps the change folder tidy and matches the as-built role `plan.md` already plays. |
