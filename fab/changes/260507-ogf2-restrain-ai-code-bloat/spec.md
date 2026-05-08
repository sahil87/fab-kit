# Spec: Restrain AI-driven code bloat

**Change**: 260507-ogf2-restrain-ai-code-bloat
**Created**: 2026-05-07
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/templates.md`, `docs/memory/fab-workflow/hydrate.md`

## Non-Goals

- Hard line-count gates that block apply/review/ship — soft signals + reviewer judgment only (per intake).
- Constitution-level rule on code growth — explicitly out of scope; may follow as a separate change.
- New "shrink"-typed change category — explicitly out of scope.
- Cross-change cumulative deltas in `/fab-status` — per-change net only is sufficient.
- A `parsimony_flags` count on `true_impact` — redundant with the review report itself.
- Auto-deletion of code identified in `## Deletion Candidates` — surfaced for the human reviewer to act on.
- Project-configurable thresholds — `100`, `+50`, and the `docs`/`chore`/`ci` skip list are hard-coded in the kit.

---

## Review: Parsimony Pass

> **Memory home**: `docs/memory/fab-workflow/execution-skills.md` (review behavior).
> **Source touched**: `src/kit/skills/_review.md` (primary).

### Requirement: Parsimony Pass Added to Inward Sub-Agent

The inward sub-agent in `_review.md` SHALL perform an additional **parsimony validation step**, executed alongside the existing six validation checks (Tasks complete, Acceptance items, Run affected tests, Spot-check spec, Memory drift, Code quality). The parsimony step MUST evaluate the apply-stage diff (changed files only) against the question: *"Could the spec's requirements be satisfied with less code?"* The pass MUST cite specific file paths and line ranges; abstract findings (e.g., "the code could be smaller") MUST NOT be emitted.

The parsimony step MUST classify each finding into exactly one of the following four categories with its mapped severity:

| Category | Finding shape | Severity (per `code-review.md`) |
|----------|---------------|---------------------------------|
| `reuse-existing-utility` | Newly added code that duplicates a utility already present in the repo | Should-fix |
| `zero-call-sites` | Newly added function/symbol/branch with zero call sites in the diff | Must-fix |
| `duplicated-logic` | New code added alongside an existing implementation of the same logic | Must-fix |
| `verbosity` | Redundant defensive checks, dead branches, or boilerplate that adds no behavior | Nice-to-have |

Findings emitted by the parsimony step SHALL be merged into the inward sub-agent's structured output and propagated through the existing Findings Merge step in `_review.md`. The merged severity mapping MUST be respected by the orchestrator's pass/fail determination — any `Must-fix` finding (from any source) fails review.

#### Scenario: Reused-utility candidate flagged as Should-fix

- **GIVEN** apply added a new helper `formatTimestamp()` in `src/foo/util.go` that duplicates an existing `time.Format()` wrapper in `src/lib/timefmt.go`
- **WHEN** the parsimony pass runs against the inward diff
- **THEN** the reviewer emits a `Should-fix` finding citing `src/foo/util.go:NN-MM` and naming the existing utility at `src/lib/timefmt.go:NN`
- **AND** review proceeds to merge but does NOT fail on this category alone

#### Scenario: Zero-call-site code flagged as Must-fix

- **GIVEN** apply added a new function `doX()` in `src/bar/x.go` that no diff-introduced caller invokes
- **WHEN** the parsimony pass runs
- **THEN** the reviewer emits a `Must-fix` finding citing `src/bar/x.go:NN-MM`
- **AND** the orchestrator's pass/fail step fails review

#### Scenario: Verbosity flagged as Nice-to-have only

- **GIVEN** apply added redundant nil-checks before a call to a function already returning a default on nil
- **WHEN** the parsimony pass runs
- **THEN** the reviewer emits a `Nice-to-have` finding
- **AND** review pass/fail is unaffected by the finding

### Requirement: Parsimony Threshold (Advisory, Hard-Coded)

The parsimony step's threshold for *encouraging* findings (not blocking) SHALL be **100 net added lines** measured against the inward diff. The threshold value SHALL be hard-coded in `_review.md` (constant in the prompt body) and MUST NOT be project-configurable. Below the threshold, the pass MAY still emit findings; above the threshold, the agent SHOULD scrutinize the diff more aggressively. The threshold is advisory — it is NOT a gate.

#### Scenario: Diff under threshold still scanned

- **GIVEN** the apply diff has 40 net added lines
- **WHEN** the parsimony pass runs
- **THEN** the pass is still executed and MAY emit findings
- **AND** no extra severity escalation is applied

#### Scenario: Diff over threshold

- **GIVEN** the apply diff has 250 net added lines
- **WHEN** the parsimony pass runs
- **THEN** the pass executes with stricter scrutiny
- **AND** findings are still classified into the four categories above (no new severity tier)

### Requirement: Parsimony Skip List (Hard-Coded)

The parsimony step SHALL be skipped (no findings emitted) when the change's `change_type` (read from `.status.yaml`) is one of: `docs`, `chore`, `ci`. The skip list SHALL be hard-coded in `_review.md` and MUST NOT be project-configurable. The skip list is shared with the deletion-candidate prompt (see *Review: Deletion Candidates* below).

#### Scenario: docs-typed change skips parsimony

- **GIVEN** `.status.yaml` `change_type: docs`
- **WHEN** the inward sub-agent runs
- **THEN** the parsimony step is silently skipped (no findings emitted, no error)
- **AND** the other validation checks (Tasks complete, Acceptance items, etc.) still run

### Requirement: Parsimony Toggle in `code-review.md`

`fab/project/code-review.md` MAY contain a section `## Parsimony Pass` with a single field:

```markdown
## Parsimony Pass

- Enabled: true
```

When the field is `true`, absent (default), or the section is missing entirely, the parsimony pass SHALL run. When the field is `false`, the parsimony pass SHALL be silently skipped (treated identically to a skip-list match — other validation checks still run).

This is the **only** parsimony-related project-level knob. Threshold values and the skip list are NOT exposed as config.

#### Scenario: Toggle off via code-review.md

- **GIVEN** `fab/project/code-review.md` contains `## Parsimony Pass\n- Enabled: false`
- **WHEN** review runs on a non-skipped change type
- **THEN** the parsimony step is silently skipped
- **AND** the rest of inward review runs normally

#### Scenario: Toggle absent (default-on)

- **GIVEN** `fab/project/code-review.md` has no `## Parsimony Pass` section
- **WHEN** review runs on a non-skipped change type
- **THEN** the parsimony step runs (default-enabled)

---

## Review: Deletion Candidates

> **Memory home**: `docs/memory/fab-workflow/execution-skills.md` (review behavior) and `docs/memory/fab-workflow/templates.md` (`plan.md` parser contract).
> **Source touched**: `src/kit/skills/_review.md`, `src/kit/templates/plan.md`.

### Requirement: Deletion-Candidate Prompt at Review

The inward sub-agent SHALL answer the question: *"What existing code (files, functions, branches, config) did this change make redundant or unused?"* The output SHALL be a structured list of candidates, each naming a specific symbol, file path, or block, with a one-line justification. The agent MAY answer "None — this change adds new functionality without making existing code redundant" when truthful.

The deletion-candidate prompt SHALL run in the **review stage** (co-located with the parsimony pass in `_review.md`), NOT in hydrate. Both passes share a diff-critique cognitive mode.

The review sub-agent MUST NOT auto-delete any code identified by this prompt. Findings are surfaced for the human reviewer to act on (in the same PR or a follow-up `chore` change).

#### Scenario: Found candidates emitted

- **GIVEN** apply added a new replacement implementation of `parseConfig()` in `src/lib/config.go` and the original `parseLegacyConfig()` in `src/lib/legacy.go` is no longer called
- **WHEN** review's deletion-candidate prompt runs
- **THEN** the prompt emits one candidate: `src/lib/legacy.go:parseLegacyConfig — superseded by parseConfig() in src/lib/config.go`

#### Scenario: No candidates — explicit "None"

- **GIVEN** apply added a new feature with no impact on existing code paths
- **WHEN** the deletion-candidate prompt runs
- **THEN** the section is written with the literal contents `None — this change adds new functionality without making existing code redundant`

### Requirement: Deletion Candidates Section in `plan.md`

`_review.md` SHALL append the prompt output as a new top-level section `## Deletion Candidates` to `plan.md` (the as-built artifact already produced at apply stage). No new artifact file SHALL be created. The section heading `## Deletion Candidates` is a **stable parser contract** — orchestrators and hydrate consumers parse this heading by name. The section SHALL be appended below the existing `## Notes` section (or at end of file if `## Notes` is absent).

The section format SHALL be either:

```markdown
## Deletion Candidates

- `{file:line or symbol}` — {one-line justification}
- `{file:line or symbol}` — {one-line justification}
```

or, when no candidates are found:

```markdown
## Deletion Candidates

None — this change adds new functionality without making existing code redundant.
```

The section is appended exactly once per review cycle. On rework loops, the existing `## Deletion Candidates` section SHALL be replaced (not duplicated). The section is distinct from `## Acceptance > ### Removal Verification`: removal-verification covers *planned* removals declared in `spec.md`; deletion-candidates covers *discovered* opportunities the apply agent missed.

#### Scenario: First review cycle appends section

- **GIVEN** `plan.md` does not yet contain `## Deletion Candidates`
- **WHEN** review's deletion-candidate prompt runs
- **THEN** a new `## Deletion Candidates` section is appended after `## Notes`
- **AND** `plan.md`'s `## Tasks` and `## Acceptance` sections are unchanged

#### Scenario: Rework cycle replaces section

- **GIVEN** `plan.md` already has a `## Deletion Candidates` section from a prior review cycle
- **WHEN** review re-runs after rework
- **THEN** the existing section is replaced in place (not duplicated)

#### Scenario: Distinct from Removal Verification

- **GIVEN** spec.md declared a planned removal of `legacyHandler()` and apply removed it
- **WHEN** review runs
- **THEN** the planned removal is verified under `## Acceptance > ### Removal Verification`
- **AND** `## Deletion Candidates` lists only *discovered* (unplanned) opportunities

### Requirement: Deletion-Candidate Skip List

The deletion-candidate prompt SHALL share the parsimony pass's skip list: it SHALL be silently skipped when `change_type` is `docs`, `chore`, or `ci`. When skipped, NO `## Deletion Candidates` section is appended to `plan.md` (omitted entirely, not written as "None").

#### Scenario: Skipped for chore-typed change

- **GIVEN** `.status.yaml` `change_type: chore`
- **WHEN** review runs
- **THEN** no `## Deletion Candidates` section is appended to `plan.md`
- **AND** the rest of review runs normally

### Requirement: Hydrate Reads Deletion Candidates When Present

The hydrate behavior in `/fab-continue` SHALL read `## Deletion Candidates` from `plan.md` when present, so memory updates MAY reference findings. Hydrate MUST NOT generate or modify the section — generation is review's responsibility. Hydrate MUST treat an absent section as "no findings" and proceed without error.

#### Scenario: Hydrate parses candidates

- **GIVEN** `plan.md` has `## Deletion Candidates` listing `src/lib/legacy.go:parseLegacyConfig`
- **WHEN** hydrate runs
- **THEN** hydrate MAY reference the candidate in memory updates (e.g., a Design Decision noting follow-up cleanup)
- **AND** the section in `plan.md` is unchanged

#### Scenario: Hydrate handles absent section

- **GIVEN** `plan.md` has no `## Deletion Candidates` section (e.g., skipped because `change_type: docs`)
- **WHEN** hydrate runs
- **THEN** hydrate proceeds without error, treating the absence as "no findings"

### Requirement: `plan.md` Template Documents the Section

`src/kit/templates/plan.md` SHALL be updated to document the `## Deletion Candidates` section as part of the parser contract. The template SHALL describe (in a guidance comment) that the section is appended by review (not by apply or by the template scaffold) and that it is omitted entirely when the change type is in the skip list. The template MUST NOT include a placeholder `## Deletion Candidates` section in the scaffolded output (it is review-generated, not template-generated).

#### Scenario: Template references the section in guidance

- **GIVEN** the updated `src/kit/templates/plan.md`
- **WHEN** a new change generates `plan.md` at apply entry
- **THEN** the generated `plan.md` does NOT contain a `## Deletion Candidates` section (review will append it later)
- **AND** the template's guidance comment names the section as a review-owned parser contract

---

## Schema: `true_impact` Block in `.status.yaml`

> **Memory home**: `docs/memory/fab-workflow/schemas.md` (`.status.yaml` schema), with surfacing in `docs/memory/fab-workflow/execution-skills.md` (apply/hydrate stage transitions).
> **Source touched**: `src/kit/templates/status.yaml`, `src/go/fab/cmd/fab/status.go` (or the `internal/status` package — wherever `Finish` lives), `src/kit/skills/fab-status.md`, `src/kit/skills/git-pr.md` (helper consumer), and a new `fab impact` Go subcommand.

### Requirement: `true_impact` Block Added to `.status.yaml`

`.status.yaml` SHALL gain a new top-level optional block `true_impact`. The block name `true_impact` SHALL be used (renamed from `line_stats` in the original intake draft, per intake assumption #4). Schema:

```yaml
true_impact:
    added: 142
    deleted: 38
    net: 104
    excluding:
        added: 87
        deleted: 38
        net: 49
    computed_at: 2026-05-07T14:32:00Z
    computed_at_stage: apply
```

Field semantics:
- `added`, `deleted`, `net` — raw `git diff --shortstat` from merge-base to current HEAD. `net` is `added - deleted`, signed integer (positive on growth, negative on shrink, zero on no-op).
- `excluding` — same fields, but with paths in `fab/project/config.yaml` `true_impact_exclude` (the field shipped by sister change `asvz`, default `[fab/, docs/]`) excluded via `git diff` `:(exclude)<pattern>` pathspec.
- `computed_at` — RFC 3339 UTC timestamp of computation.
- `computed_at_stage` — the pipeline stage at which the snapshot was taken. Valid values: `apply`, `hydrate`.

When `true_impact_exclude` is absent, `null`, or empty in `config.yaml`, the `excluding` sub-block SHALL be omitted entirely from the `true_impact` block (the consumer treats "no excludes" identically to "excluding == raw"; emitting an extra block adds no signal).

The block SHALL be omitted entirely from `.status.yaml` until first computed. Existing `.status.yaml` files without the block remain valid (backwards compatible).

The `status.yaml` template (`src/kit/templates/status.yaml`) SHALL NOT initialize the block (no empty `true_impact: {}` placeholder) — the block is created lazily on first computation.

#### Scenario: Block absent on new change

- **GIVEN** a freshly created change at intake stage
- **WHEN** `.status.yaml` is read
- **THEN** the file has no `true_impact` key
- **AND** all consumers proceed without error

#### Scenario: Block populated at end of apply

- **GIVEN** `fab status finish <change> apply` is invoked
- **WHEN** the apply-finish handler runs
- **THEN** `.status.yaml` is updated with a `true_impact` block where `computed_at_stage: apply`
- **AND** `added`, `deleted`, `net` reflect the merge-base-to-HEAD diff
- **AND** if `true_impact_exclude` is non-empty, `excluding` is populated with the exclude-pathspec'd diff

#### Scenario: Block recomputed at hydrate

- **GIVEN** `.status.yaml` has a `true_impact` block from apply with `computed_at_stage: apply`
- **WHEN** `fab status finish <change> hydrate` is invoked
- **THEN** the block is recomputed
- **AND** `computed_at` is updated to the new timestamp
- **AND** `computed_at_stage` becomes `hydrate`

#### Scenario: Empty exclude list omits sub-block

- **GIVEN** `fab/project/config.yaml` has `true_impact_exclude: []` (or the field is absent)
- **WHEN** `true_impact` is computed
- **THEN** the written block contains `added`, `deleted`, `net`, `computed_at`, `computed_at_stage` — but NO `excluding` sub-block

### Requirement: `fab impact` Go Subcommand (Helper Extraction)

A new Go subcommand `fab impact <base> <head>` SHALL be added to the `fab` binary. It SHALL compute `git diff --shortstat` line counts from `<base>` to `<head>` (three-dot range semantics: `<base>...<head>`), reading `true_impact_exclude` from `fab/project/config.yaml` to apply pathspec exclusions for the `excluding` pass. The subcommand SHALL emit a YAML document on stdout matching the `true_impact` block schema (without the `computed_at_stage` field — that is the caller's responsibility):

```yaml
added: 142
deleted: 38
net: 104
excluding:
    added: 87
    deleted: 38
    net: 49
computed_at: 2026-05-07T14:32:00Z
```

The subcommand SHALL omit the `excluding` block when `true_impact_exclude` is absent/null/empty, mirroring the schema rule above. The subcommand SHALL exit non-zero with an actionable stderr message when `git merge-base` cannot be resolved or `git diff` fails; the caller decides whether to abort or skip.

`src/kit/skills/git-pr.md`'s Step 3c-impact SHALL be refactored to consume `fab impact` instead of inlining the shell pipeline (`yq` + two `git diff --shortstat` invocations + line parsing). The PR-body rendering logic (the `**Impact**: +A/−D code (excluding ...) · +A_total/−D_total total` line, the `+0/−0` omission rule, and the no-fab-context fallback) remains in `git-pr.md` — only the *measurement* moves to `fab impact`.

The status-finish handler (apply/hydrate) SHALL invoke `fab impact` (in-process via the same Go package, or via subprocess; spec defers the choice to apply) to produce the data written into the `true_impact` block.

#### Scenario: Two consumers share the helper

- **GIVEN** `fab impact origin/main HEAD` is invoked
- **WHEN** both `/git-pr` Step 3c-impact and the apply-finish handler call it
- **THEN** they observe identical numbers for the same merge-base + HEAD
- **AND** neither consumer reimplements `git diff --shortstat` parsing or pathspec-exclude assembly

#### Scenario: Merge-base resolution failure

- **GIVEN** the working tree has no `origin/main` and no `origin/master`
- **WHEN** `fab impact $(git merge-base HEAD origin/main) HEAD` is invoked with an empty `<base>` (because `git merge-base` failed upstream)
- **THEN** `fab impact` exits non-zero with a stderr message identifying the missing base
- **AND** the apply-finish handler skips the `true_impact` write (no partial block)
- **AND** `/git-pr` Step 3c-impact omits the `**Impact**` line silently (existing behavior preserved)

### Requirement: Computation Hook at Apply-Finish and Hydrate-Finish

`fab status finish <change> apply` SHALL invoke the `fab impact` helper (with `<base>` = `git merge-base HEAD origin/main` falling back to `origin/master`, `<head>` = `HEAD`) and write the result plus `computed_at_stage: apply` into `.status.yaml`'s `true_impact` block. `fab status finish <change> hydrate` SHALL repeat the computation, overwriting the block with `computed_at_stage: hydrate`.

The hook SHALL be best-effort: if the helper exits non-zero (e.g., no merge-base resolvable in a fresh worktree), `finish` SHALL log a one-line warning to stderr and proceed — the stage transition itself MUST NOT fail because of a `true_impact` computation failure. This preserves existing `finish` semantics and keeps `true_impact` from becoming a new failure mode.

#### Scenario: Apply-finish writes block

- **GIVEN** a change at apply stage with merge-base resolvable
- **WHEN** `fab status finish <id> apply` runs
- **THEN** `.status.yaml` gains a `true_impact` block with `computed_at_stage: apply`
- **AND** the apply stage transitions to `done` and review activates

#### Scenario: Apply-finish on broken worktree

- **GIVEN** a change at apply stage with no resolvable merge-base
- **WHEN** `fab status finish <id> apply` runs
- **THEN** the stage transition still completes (apply → done; review → active)
- **AND** `.status.yaml` does NOT gain a `true_impact` block
- **AND** stderr emits a one-line warning

### Requirement: `/fab-status` Surfaces `true_impact`

`fab/project/skills/fab-status.md` SHALL render an additional line under the existing change summary block, sourced from `.status.yaml` `true_impact`:

```
Impact: +N (raw {added}/-{deleted}, excluding fab/docs +M ({excl_added}/-{excl_deleted}))
```

When `true_impact` is absent, the line SHALL be omitted entirely (no "not yet computed" placeholder). When `excluding` is absent in the block (because the project's `true_impact_exclude` is empty), the line SHALL render only the raw figures.

The line SHALL be highlighted in yellow (terminal `\e[33m...\e[0m` or equivalent) when EITHER:
- `true_impact.net > 100`, OR
- `true_impact.excluding.net > 50` (when `excluding` is present)

These thresholds are advisory and hard-coded in `fab-status.md` (matching the parsimony threshold and refactor-warning threshold respectively); they are NOT project-configurable.

#### Scenario: Block present, under thresholds

- **GIVEN** `true_impact: {added: 80, deleted: 20, net: 60, excluding: {net: 30}, ...}`
- **WHEN** `/fab-status` runs
- **THEN** the impact line is rendered without yellow highlighting

#### Scenario: Block present, raw over threshold

- **GIVEN** `true_impact.net: 150`
- **WHEN** `/fab-status` runs
- **THEN** the impact line is rendered in yellow

#### Scenario: Block absent

- **GIVEN** `.status.yaml` has no `true_impact` block
- **WHEN** `/fab-status` runs
- **THEN** no impact line is rendered

### Requirement: `fab change list --show-stats` Flag

`fab change list` SHALL gain an optional `--show-stats` boolean flag. When set, the output SHALL include an additional column showing `true_impact.net` (or `excluding.net` when present) for each listed change; absent blocks render as `—`. The default `fab change list` output SHALL remain unchanged (no extra column) to keep the compact view.

#### Scenario: Default list compact

- **GIVEN** several changes with and without `true_impact`
- **WHEN** `fab change list` is invoked without `--show-stats`
- **THEN** the output matches today's format (no impact column)

#### Scenario: With --show-stats

- **GIVEN** the same changes
- **WHEN** `fab change list --show-stats` is invoked
- **THEN** the output adds a per-row impact column showing `excluding.net` when present, else `net`, else `—`

### Requirement: Refactor-Growth Soft Warning

`/fab-status` SHALL emit a one-line **soft warning** below the impact line when ALL of the following hold:
- `.status.yaml` `change_type` equals `refactor`
- `true_impact.excluding.net > 50` (when `excluding` is present) OR `true_impact.net > 50` (when `excluding` is absent)
- The `true_impact` block is present (from any stage)

The warning text SHALL be exactly:

```
Refactor changes typically shrink or stay flat — review whether this growth is intentional.
```

The threshold of `+50` is hard-coded in the kit; it MUST NOT be project-configurable (per intake assumption #9). The warning is informational only — no gate, no block.

#### Scenario: Refactor with growth fires warning

- **GIVEN** `change_type: refactor` and `true_impact.excluding.net: 80`
- **WHEN** `/fab-status` runs
- **THEN** the soft-warning line is emitted

#### Scenario: Non-refactor change does not fire

- **GIVEN** `change_type: feat` and `true_impact.excluding.net: 200`
- **WHEN** `/fab-status` runs
- **THEN** the impact line is rendered (yellow if over threshold) but the refactor warning is NOT shown

#### Scenario: Refactor at threshold does not fire

- **GIVEN** `change_type: refactor` and `true_impact.excluding.net: 50` (equal, not greater)
- **WHEN** `/fab-status` runs
- **THEN** the soft warning is NOT shown

---

## Documentation: SPEC files for Touched Skills

> **Memory home**: not a memory change — a constitutional rule (Additional Constraints, line 32) requires that changes to `src/kit/skills/*.md` update the corresponding `docs/specs/skills/SPEC-*.md`.

### Requirement: SPEC Files Updated for Touched Skills

Per the constitution's Additional Constraints, changes to `src/kit/skills/_review.md`, `src/kit/skills/fab-status.md`, and `src/kit/skills/git-pr.md` SHALL be accompanied by updates to their corresponding `docs/specs/skills/SPEC-_review.md`, `SPEC-fab-status.md`, and `SPEC-git-pr.md`. SPEC files MUST be edited by hand at apply time (per constitution principle VI: "specs MUST NOT be auto-generated"). This spec deliberately does NOT enumerate the SPEC content; tasks at apply time will derive the SPEC edits from the requirements above.

The new `fab impact` Go subcommand SHALL be documented in `src/kit/skills/_cli-fab.md` (per the constitution's CLI rule: "Changes to the `fab` CLI MUST … update `src/kit/skills/_cli-fab.md`").

#### Scenario: Apply-stage tasks include SPEC edits

- **GIVEN** the tasks artifact for this change
- **WHEN** apply runs
- **THEN** SPEC files for `_review`, `fab-status`, and `git-pr` are edited
- **AND** `_cli-fab.md` is updated with the `fab impact` subcommand signature

---

## Design Decisions

1. **Extract impact math into `fab impact` Go subcommand (rejecting "shared shell helper")**:
   - *Why*: The kit ships markdown skills + Go binary only — there is no `src/kit/scripts/lib/` directory and no precedent for shared shell helpers in the kit. Two consumers (`/git-pr` and the apply-finish hook) need identical merge-base + shortstat + pathspec-exclude logic. A Go subcommand keeps consumers shell-free, gives the helper proper unit tests via `internal/`, and matches the binary-first distribution model. The intake recommended extraction with two-consumer justification; the spec-stage decision is *which form*: Go subcommand wins because the kit has no shell-helper precedent.
   - *Rejected*: Inline duplication in both call sites — duplicates merge-base resolution, two-call shortstat orchestration, and `--shortstat` line parsing across two languages (markdown-shell + Go), increasing drift risk for ~20 lines of logic per consumer.
   - *Rejected*: Shared shell helper at `src/kit/scripts/lib/impact.sh` — no such directory exists in `src/kit/`, and introducing one establishes a parallel distribution surface that the system-cache + `fab sync` pipeline does not currently cover. Adding a new shell-helper distribution path is itself code bloat — exactly the behavior this change restrains.

2. **`true_impact` lazily created (no template placeholder)**:
   - *Why*: The block is only meaningful after apply completes; an empty `true_impact: {}` in the template adds noise to every change folder for no signal, and the schema is already optional/backwards-compatible. Consistent with how the change started life — `.status.yaml` has no `confidence` data at template creation either, just the zero-counts placeholder.
   - *Rejected*: Initialize `true_impact: {}` in the template — adds a never-read empty key for the entire intake/spec span.

3. **Skip list shared between parsimony and deletion-candidate passes**:
   - *Why*: Both passes are diff-critique with the same applicability profile. A single hard-coded skip list (`docs`, `chore`, `ci`) keeps the configuration surface small (per intake assumption #10).
   - *Rejected*: Independent skip lists per pass — duplicates the rationale across two prompts and risks drift if either is tuned in isolation.

4. **Plan template guidance comment, not placeholder section, for `## Deletion Candidates`**:
   - *Why*: The section is review-generated, not template-generated. A placeholder section would create an empty heading on every plan.md that gets overwritten or deleted; reuses the same pattern as the spec template's optional sections (Non-Goals, Design Decisions) — guidance comment, not scaffold.
   - *Rejected*: Always-present `## Deletion Candidates\n_TBD_` placeholder — clutter for changes that will skip the section entirely (`docs`/`chore`/`ci`).

5. **`true_impact` recomputed at hydrate (not just apply)**:
   - *Why*: Review-stage edits (rework loops, reviewer fixes) shift the diff between apply-finish and hydrate-finish. Recomputing at hydrate captures the final shipped diff. The cost is one extra `git diff --shortstat` invocation per change — negligible.
   - *Rejected*: Compute only at apply — produces a stale snapshot when review edits land.

6. **Best-effort hook on `fab status finish` (no failure mode)**:
   - *Why*: A `true_impact` computation failure (e.g., no merge-base in a fresh worktree) MUST NOT block stage transition. The block is informational; the pipeline must continue. Same posture as `fab log command` (best-effort, silent on failure).
   - *Rejected*: Hard-fail `finish` on computation error — adds a new failure mode for an advisory feature.

---

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is limited to interventions #1, #2, #3 from intake | Confirmed from intake #1 — discussed and locked | S:95 R:80 A:90 D:95 |
| 2 | Certain | No hard line-count gates — soft signals only | Confirmed from intake #2 — locked-down by user | S:95 R:90 A:95 D:95 |
| 3 | Certain | Parsimony pass lives in `_review` skill (inward sub-agent), integrated with existing severity tiers | Confirmed from intake #3; spec adds the four-category mapping verbatim | S:95 R:75 A:85 D:75 |
| 4 | Certain | `true_impact` block goes in `.status.yaml` (renamed from `line_stats`) | Confirmed from intake #4 — name and location locked | S:95 R:75 A:85 D:80 |
| 5 | Certain | Deletion candidates go in a new `## Deletion Candidates` section of `plan.md` | Confirmed from intake #5 — co-located with as-built artifact, distinct from Removal Verification | S:95 R:75 A:90 D:80 |
| 6 | Certain | `true_impact_exclude` config field reused from sister change asvz | Confirmed from intake #6 — config field exists in `fab/project/config.yaml`, default `[fab/, docs/]` | S:90 R:80 A:85 D:80 |
| 7 | Certain | Refactor-type changes with net > +50 (excluding fab/docs) get a soft warning in `/fab-status` | Confirmed from intake #7 — exact warning wording specified | S:95 R:85 A:80 D:70 |
| 8 | Certain | Parsimony pass threshold: 100 net added lines (hard-coded) | Confirmed from intake #8 — hard-coded in `_review.md` constant | S:95 R:85 A:60 D:55 |
| 9 | Certain | Refactor warning threshold: +50 net excluding fab/docs (hard-coded) | Confirmed from intake #9 — hard-coded in `fab-status.md` | S:95 R:85 A:60 D:55 |
| 10 | Certain | Deletion-candidate prompt shares parsimony pass's skip list | Confirmed from intake #10 — `docs`, `chore`, `ci` apply to both | S:95 R:75 A:65 D:55 |
| 11 | Certain | Parsimony pass and deletion-candidate prompt skipped for `docs`/`chore`/`ci` (hard-coded) | Confirmed from intake #11 — single shared skip list per #10 | S:95 R:85 A:65 D:55 |
| 12 | Certain | Deletion-candidate prompt lives in `_review` (review stage, not hydrate) | Confirmed from intake #12 — co-located with parsimony pass; hydrate reads but doesn't generate | S:95 R:55 A:50 D:35 |
| 13 | Certain | No cross-change cumulative deltas in `/fab-status` (per-change net only) | Confirmed from intake #13 — per-change signal sufficient | S:95 R:75 A:55 D:50 |
| 14 | Certain | Helper extracted as `fab impact <base> <head>` Go subcommand (resolves intake #6 sub-question) | Spec-stage decision: kit has no `src/kit/scripts/lib/` precedent — Go subcommand matches the binary-first distribution model. Two consumers (`/git-pr` Step 3c-impact and apply-finish hook) justify extraction per intake recommendation. See Design Decision #1 | S:90 R:70 A:90 D:85 |
| 15 | Certain | `true_impact` block is lazily created; not initialized as `{}` placeholder in `status.yaml` template | Spec-stage decision: aligns with backwards-compat semantics ("absent block remains valid") and avoids template noise. See Design Decision #2 | S:90 R:80 A:85 D:80 |
| 16 | Certain | `true_impact` recomputed at both apply-finish and hydrate-finish (not apply-only) | Spec-stage decision: review edits shift the diff between apply-finish and hydrate-finish; recomputing keeps the surfaced number aligned with shipped diff. See Design Decision #5 | S:90 R:75 A:85 D:80 |
| 17 | Certain | `fab status finish` hook for `true_impact` is best-effort (logs warning, never fails the stage transition) | Spec-stage decision: matches the project pattern for telemetry hooks (e.g., `fab log command`). A new advisory feature MUST NOT introduce a new failure mode. See Design Decision #6 | S:90 R:80 A:85 D:80 |
| 18 | Certain | `## Deletion Candidates` documented in `plan.md` template via guidance comment, not placeholder section | Spec-stage decision: review-owned section, not template-generated. Mirrors how Non-Goals and Design Decisions are handled in the spec template. See Design Decision #4 | S:90 R:80 A:85 D:80 |
| 19 | Certain | `excluding` sub-block omitted when `true_impact_exclude` is absent/null/empty | Spec-stage decision: avoids redundant data (`excluding` would equal raw); consistent with `/git-pr`'s Impact-line omission rule for the same edge case | S:90 R:80 A:85 D:85 |
| 20 | Certain | `/fab-status` impact-line yellow-highlight thresholds: raw>100 OR excluding>50 | Spec-stage decision: reuses the parsimony-pass threshold (100) and refactor-warning threshold (+50) — no new magic number | S:90 R:80 A:75 D:70 |
| 21 | Certain | `## Deletion Candidates` section is a stable parser contract heading; appended below `## Notes`; replaced (not duplicated) on rework | Spec-stage decision: matches the existing parser-contract pattern for `## Tasks`/`## Acceptance` (named-heading stability); placement after `## Notes` keeps the file's narrative flow (work → acceptance → notes → discovered cleanup) | S:90 R:75 A:85 D:75 |

21 assumptions (21 certain, 0 confident, 0 tentative, 0 unresolved).
