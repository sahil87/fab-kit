# Intake: Merge Spec Stage into Apply, Frontload SRAD to Intake

**Change**: 260601-j6cs-merge-spec-into-apply
**Created**: 2026-06-01
**Status**: Draft

## Origin

<!-- How was this change initiated? Include the user's raw input/prompt, the interaction
     mode (one-shot vs. conversational), and key decisions from the conversation. -->

Initiated conversationally during a `/fab-discuss` session. The user proposed two coupled structural changes to the fab pipeline:

> "What if we frontload scoring (srad) officially to intake, and merge spec and apply into a single apply stage? ... my reason for merging the stages is that I think most of the brainstorming and spec improvements happen at intake stage."

The user asked for empirical validation against the `loom` repo (`~/code/wvrdz/loom`) before committing. That analysis was performed (see **Why**), confirmed the hypothesis, and the design decisions below were made interactively:

- **Gate model**: The user chose a single intake gate with **per-type thresholds** (over an intake-only fixed gate, or a two-gate model keeping an apply-entry gate). Selected rationale: "frontloads scoring to intake AND preserves per-type rigor."
- **Deliverable**: The user chose to start a real fab change (this one) to dogfood the workflow, rather than a written-only or HTML plan.

Three read-only `Explore` sub-agents mapped every affected file across the Go binary, skills, and docs (see **Impact**).

## Why

**Problem.** The fab core pipeline has 5 stages (`intake → spec → apply → review → hydrate`). The `spec` stage is a near-pass-through that adds a stage boundary, a gate, a `.status.yaml` progress key, a transition, and an artifact-generation skill step — but does little independent refinement work in practice.

**Evidence** (analysis of `~/code/wvrdz/loom`, 613 change-history `.history.jsonl` files):

- `fab-clarify` ran **225 times across 154 changes** (only 25% of changes clarify at all).
- Stage distribution of clarify invocations: **intake 59.6%**, spec 32.0%, all other stages <5% combined.
- **Spec stage median duration: 2 minutes** (mean 8m, p90 10m) vs **intake median 11 minutes** (mean 61m, p90 112m). The deliberation time lives at intake; spec is a rubber-stamp transition.
- Spec was **re-entered for rework in only 7 of 613 changes (~1%)** — once written, the spec is almost never revisited.
- Spec-stage clarification is uniformly low (6–14%) across **all** change types — no type relies on the spec stage for heavy lifting.

This confirms the user's hypothesis: most brainstorming and refinement already happens at intake, and the spec stage rarely earns its cost.

**Consequence of not fixing.** Every change pays for an extra stage, gate, and transition that 99% of the time just advances. New users must learn a 5-stage model where one stage is conceptually redundant with intake.

**Why this approach.** There is a direct, proven precedent: **v1.9.0 already merged the `tasks` stage into `apply`** (migration `src/kit/migrations/1.8.0-to-1.9.0.md`), folding plan generation into apply's entry sub-step and dropping the pipeline from 8→7 stages. This change applies the identical pattern to `spec`: spec generation becomes the *first* apply entry sub-step. Every layer touched by the tasks→apply merge is the layer touched here, so the blast radius and migration shape are known.

**Why not delete the spec *content* entirely.** The 47 loom changes that clarified *only* at spec show a real (if small) second wave of ambiguity that surfaces when writing the spec. This change *relocates* that work rather than deleting it: the requirement discipline (RFC-2119 + scenarios) lives on as `plan.md`'s `## Requirements` section, co-generated at apply entry. The *human* clarification of that ambiguity moves to intake (see §1a); inside apply, the agent resolves it inline as graded SRAD assumptions rather than via markers (see §1a). What is removed is the spec *stage* and the separate `spec.md` *file* — not the requirement-capture practice.

**What the merge trades away — and why the intake gate compensates.** Honesty demands naming a real loss: today the spec stage performs an *independent re-grade* of the intake's assumptions (`_generation.md` §Spec Generation step 6 — the spec agent reads intake's `## Assumptions` and "confirms, upgrades, or overrides" each based on spec-level analysis), and the spec gate scores *that re-graded* table. Co-generating requirements + tasks + acceptance in one apply pass means the same agent that writes requirements immediately consumes them — there is no second, independent re-grade between requirements and tasks. We accept this for three reasons: (1) the new single intake gate is set at **3.0 for all types** — at least as strict as the old intake gate and ≥ the old spec gate for every type (§2 accounting), so the entry bar that the loom data shows does the real filtering is *strengthened*, not weakened; (2) the loom evidence shows the spec re-grade rarely changes the outcome — spec was re-entered for rework in only ~1% of changes and spec-stage clarification was 6–14% across all types; (3) requirement-*correctness* (as opposed to tasks-requirements *alignment*) is still caught at **review**, which re-reads `## Requirements` against the implementation. The cost we knowingly accept: a wrong-but-internally-consistent requirement is now caught at review (after code is written) rather than at a pre-planning checkpoint — a later, somewhat more expensive catch point, judged acceptable given (1)–(2). <!-- See review finding #25/#35: lost independent re-grade, acknowledged not hidden. -->

## What Changes

### 1. Pipeline shape (core: 5 → 4 stages)

```
BEFORE:  intake → spec → apply → review → hydrate   (+ ship, review-pr)
AFTER:   intake →        apply → review → hydrate   (+ ship, review-pr)
```

`apply` entry generates a **single unified `plan.md`** (with `## Requirements` + `## Tasks` + `## Acceptance` sections) in one pass, then executes the tasks. Today apply entry generates `plan.md` (tasks + acceptance) from a separate `spec.md`; this change folds the requirements section *into* `plan.md`, so spec and plan are co-generated in one file. `spec.md` ceases to exist as a separate artifact (see §4).

### 1a. Automation invariant (the design north star)

The deeper goal driving this merge: **all human judgment is pulled forward to intake; everything after intake is fully automated.**

Post-change the pipeline is **6 stages** (`spec` removed; `hydrate` retained):

```
            ┌─────────────── automated (gated on intake) ───────────────┐
intake  →   apply  ↔  review  →  hydrate  →  ship  →  review-pr
 (1)         (2)       (3)         (4)        (5)        (6)
manual      └──────────────────────────────────────────────────┘
```

Orchestrator scopes under this model:
- `/fab-proceed` — all 6 stages from intake.
- `/fab-fff` — everything after intake (apply → review → hydrate → ship → review-pr).
- `/fab-ff` — apply → review → hydrate (stops before the PR stages).
- `/fab-continue` — one stage at a time, after intake.

- **Intake is the sole stage that requires human judgment to proceed.** It is where the developer's brainstorming, decisions, and clarification happen. The per-type intake confidence gate (§2) is the guard.
- **Apply's inner workings are hidden from the user.** Requirements and plan are co-generated *inside* apply (into `plan.md`), consumed by apply/review, and never surfaced as stage boundaries. The user sees: intake in, PR out.
- **`(apply ↔ review)` is an autonomous bidirectional loop.** Review feeds rework back into apply with bounded retry (per `code-review.md` rework budget). For that loop to *converge* rather than thrash, it needs a target — the parser contract + traceability (§1b) provide it.
- **Honest scope of "automated":** post-intake stages run *unattended*, but two existing touchpoints still surface to a human — review-rework **exhaustion** (the loop bails to `/fab-continue` manual rework after the cap) and **review-pr** (processing human/bot PR comments). The invariant is precisely: *intake is the only stage that requires human judgment to **proceed**; everything after runs unattended unless rework exhausts or PR feedback arrives.* It is not "zero human judgment after intake."

**Clarification is an intake-only, human-facing activity. There is no clarify step inside apply.**

- `/fab-clarify` is a skill that refines `intake.md`. It does **not** run after intake — not even in `[AUTO-MODE]`. Both `fab-ff`/`fab-fff` auto-clarify invocations are deleted (the post-spec-gen one AND the on-plan one — see §5).
- `[NEEDS CLARIFICATION]` markers are an **intake-only construct** — they mean "a human still needs to decide this," and humans only decide at intake. They never appear in `plan.md` (including its `## Requirements` section).
- Inside apply, ambiguity resolution is **not a separate skill or ceremony** — it is the apply agent's normal behavior while generating the plan: encounter an under-specified requirement → make a graded SRAD decision → record it as an assumption (Certain/Confident/Tentative) in `plan.md`'s `## Assumptions`. Apply does not *clarify*; it *decides and records*.

**The "bounce-back" valve is the intake gate itself — there is no new runtime mechanism to build.** The earlier framing (apply detects an Unresolved mid-run and resets to intake) was wrong. The actual guard is simpler and already exists:

- Intake is SRAD-scored. If it fails the per-type gate, **intake never reaches `done`** — it stays `ready`/`active`.
- Because intake is not `done`, the orchestrators **cannot advance**: `/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-proceed` all gate on `fab score --check-gate` and refuse to enter apply.
- So an under-resolved change is *prevented from entering the automated bracket in the first place*, rather than entering and bouncing out. The human re-clarifies at intake, re-scores, and only a passing intake unlocks automation.
- This means **apply needs no Unresolved/bail logic** — the gate upstream is the safety valve. (Consequence: the SRAD *Unresolved* "must always be asked" Critical Rule applies at **intake** — `/fab-new`/`/fab-clarify` — exactly as it does today; nothing new is required at apply.)

### 1b. Parser contract + traceability (what makes the hidden loop autonomous)

Hiding the stage must NOT dissolve the artifact structure. The merge folds the *stage* away while keeping the generated artifacts rigidly **machine-parseable and traceable**, because with apply hidden, the machine's grip on these files is the only interface between intake and PR.

- **Parser contract (enables autonomy):** stable `## Tasks` / `## Acceptance` headings and `T{NNN}` / `A-{NNN}` ID formats are the contract apply (reads Tasks) and review (reads Acceptance) depend on. Unchanged by this merge — but now load-bearing rather than incidental.
- **Traceability (makes the autonomous loop *converge*):** the chain `R# (requirement) → T# (task) → test → A# (acceptance)` lets the review loop localize a failing acceptance item back to its requirement and forward to the task that satisfies it. Without it, autonomous rework over-corrects or stalls; with it, the loop has a gradient to follow. This change makes trace annotations **required** rather than optional (today `_generation.md` treats cross-linking as OPTIONAL).

> Distinction worth recording: the **parser contract** makes the pipeline *mechanizable*; **traceability** makes the autonomous `(apply ↔ review)` loop *convergent*. They are different properties — autonomy needs the first; safe unattended autonomy needs both.

### 2. Confidence gate model

Replace the current **two-gate** model:

| | When | Threshold | Scores |
|---|---|---|---|
| Intake gate (old) | before pipeline | fixed **3.0** | `intake.md` |
| Spec gate (old) | after spec gen | per-type (fix 2.0, feat/refactor 3.0, rest 2.0) | `spec.md` |

…with a **single gate at intake**. To avoid silently *weakening* the entry bar, the single gate threshold is set to **preserve the old fixed intake bar of 3.0 for every type**, keeping `feat`/`refactor` at 3.0 and *raising* `fix`/`docs`/`test`/`ci`/`chore` to 3.0 (the old per-type spec-gate values were 2.0 for these — but those gated `spec.md` *in addition to* the 3.0 intake gate; with one gate, we keep the stronger 3.0):

| | When | Threshold | Scores |
|---|---|---|---|
| Intake gate (new) | before pipeline | **3.0 for all 7 types** | `intake.md` |

**Gate-strength accounting (old → new), per type** — so the consolidation is honestly no-weaker:

| Type | Old intake gate | Old spec gate | New single gate |
|------|-----------------|---------------|-----------------|
| feat, refactor | 3.0 | 3.0 | **3.0** (unchanged) |
| fix, docs, test, ci, chore | 3.0 | 2.0 | **3.0** (≥ both old gates) |

Every type's new single gate is ≥ its old intake gate, so no type's entry bar is relaxed. (A flat 3.0 also keeps `gateThresholds` trivial; per-type divergence can be revisited later if evidence warrants — see Open Questions.)

- **`expectedMin` (resolved):** adopt the higher `expectedMinSpec` values (`feat:7, refactor:6, fix:5`) as the single intake gate's `expectedMin`; **delete the lower `expectedMinIntake` table** (feat:5, refactor:4, fix:3). For the four types absent from `expectedMinSpec` (`docs/test/ci/chore`), the single intake `expectedMin` uses the current spec-path default fallback of **3** (matching `getExpectedMin`'s `else`-branch default today). Rationale: intake is now the sole authoritative gate, so it should demand spec-level decision coverage. **Reconcile with `docs/specs/change-types.md`** (whose table currently lists *different* values — Intake feat:4/refactor:3/fix:2, Spec feat:6/refactor:5/fix:4 — a pre-existing code↔doc drift): the Go values are authoritative; the change-types.md table is rewritten to match.
- The `confidence.indicative` flag is **retired** — intake scoring becomes authoritative, not indicative.
- `fab score` default `--stage` changes from `spec` to `intake`. `getExpectedMin`'s now-vestigial `stage` param is retained as `intake`-only (or simplified to a single map — decide at planning); `CheckGate`'s intake branch changes from the hardcoded `threshold = 3.0` to `threshold = getGateThreshold(changeType)` (which now returns 3.0 for all, but routes through the per-type table so future divergence is a one-line data change).

### 3. Go binary (`src/go/fab/`)

**State machine:**
- `internal/statusfile/statusfile.go`: remove `"spec"` from `StageOrder` (line ~13–15). Note `StageNumber`/`NextStage`/`GetProgressMap` derive from `StageOrder`, so they update automatically; an orphan `progress.spec` key on an un-migrated file is tolerated (raw-node passthrough, `Validate()` skips unknown keys) and removed only by the migration — assert this in a test.
- `internal/status/status.go`: remove `spec` from `AllowedStates` and `stageTransitions`; add a `spec`-deprecation branch to `validateStage` (mirroring the `tasks` branch at line ~93–94): `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".`
- `internal/change/change.go`: drop the `spec` case from `defaultCommand()` routing (line ~406, the `case "intake", "spec", "apply", "review"`). **Also `fab change list` row formatter (lines ~289–293)** emits a positional `name:display_stage:display_state:score:indicative` 5-field row — retiring indicative changes this output ABI (see §indicative below).

**Scoring (`internal/score/score.go`, `cmd/fab/score.go`):**
- `score.go`: set `gateThresholds` to 3.0 for all 7 types (flat); change `CheckGate` intake branch (line ~100) from hardcoded `threshold = 3.0` to `threshold = getGateThreshold(changeType)`; replace `expectedMinIntake`/`expectedMinSpec` with a single `expectedMin` map seeded from the old spec values (feat:7/refactor:6/fix:5, default 3 for the rest); simplify `getExpectedMin` (the `stage` param is now vestigial — keep as intake-only or drop); remove the `indicative := stage == "intake"` derivation (line ~186) and stop threading it.
- `cmd/fab/score.go`: change default `--stage` flag from `"spec"` to `"intake"` (line ~44); the `spec.md`-read paths in `Compute` (line ~148/150) and `CheckGate` (line ~102) are removed/repointed to `intake.md`.

**Indicative retirement (spans 3 files — correct package attribution):**
- `internal/statusfile/statusfile.go`: the `Confidence.Indicative *bool` field (line ~60) and its `encodeConfidence` emission (lines ~542–546). **Decision (per finding #7):** *keep* the field as an accepted-but-ignored decode target so Load/Save round-trips legacy `indicative: true` harmlessly (mirrors how the precedent kept `checklist:` decode tolerance), rather than hard-removing it and silently stripping the key from every un-migrated/archived file the binary happens to re-save. Stop *writing* it; tolerate *reading* it.
- `internal/status/status.go`: `SetConfidence`/`SetConfidenceFuzzy` live **here** (lines ~335/350), not in statusfile.go — drop their `indicative bool` param.
- `cmd/fab/status.go`: remove the `--indicative` flag from `set-confidence`/`set-confidence-fuzzy` cmd definitions (lines ~385/441) and plumbing (~350/381/390/437) — or keep it as an accepted-but-ignored no-op for script back-compat (decide at planning).

**Hook + artifact matcher (highest-corruption-risk — was missed entirely):**
- `cmd/fab/hook.go`: remove the `case "spec.md": score.Compute(..., "spec")` branch (lines ~270–274). **Why this matters:** the migration leaves stray `spec.md` files on disk ("safe to delete"); if a user edits one, this hook fires and rewrites `.status.yaml` confidence *without* `indicative`, silently overwriting the authoritative intake score. Removing the branch closes that corruption path.
- `internal/hooklib/artifact.go`: drop `"spec.md"` from the `MatchArtifactPath` switch (line ~80, `case "intake.md", "spec.md", "plan.md"`) in lockstep — this is the gate that decides whether the PostToolUse hook fires at all; leaving it makes the hook.go removal an orphaned branch.

**Test updates (Constitution §"Go changes MUST include test updates"):** at minimum — `internal/status/status_test.go` (~186 stage enumeration, ~218–223 finish spec), `internal/statusfile/statusfile_test.go` (~133 `StageOrder`, ~295–298 `NextStage("intake")=="spec"`), `internal/preflight/preflight_test.go` (~84–88 asserts `Stage=="spec"` and `len(Progress)==7`), `internal/log/log_test.go` (~109–183 spec transitions), `internal/change/change_test.go` (~335–339 the 5-field `:indicative` contract), `internal/hooklib/artifact_test.go` (spec.md → convert to rejected-case test, mirroring the tasks rejection), `cmd/fab/hook_test.go`, and `true_impact_test.go` (~263 writes for spec).

### 4. Templates & migration

- `src/kit/templates/status.yaml`: drop `spec: pending` from the `progress:` block.
- **`spec.md` is ABSORBED into `plan.md`, not kept as a separate file.** The `spec.md` template (`src/kit/templates/spec.md`) is removed; its requirement discipline (RFC-2119 + GIVEN/WHEN/THEN scenarios) becomes a `## Requirements` section at the top of `plan.md`. Final artifact set: `intake.md` (human) → `plan.md` (machine, hidden) → code. Rationale: once scoring moves to intake, **nothing reads `spec.md` programmatically**. Full consumer enumeration (verified): `score.Compute` (normal mode, `score.go:148/150`) and `score.CheckGate` (gate mode, `score.go:102`) — the former invoked by the PostToolUse hook (`hook.go:270`) and bare `fab score`; the hook's artifact matcher (`hooklib/artifact.go:80`); and `git-pr.md`'s PR-body Spec link (L174/184/253). *All* of these are removed or repointed by this change (§3, §5). A separate `spec.md` would be generated, never machine-read, hidden from the user — dead weight.
- **`src/kit/templates/plan.md`** (template update — the new home of all three sections):
  - Add a `## Requirements` section (absorbed from spec.md): RFC-2119 statements with stable `R#` IDs, each with ≥1 GIVEN/WHEN/THEN scenario, plus optional `## Non-Goals` / `## Design Decisions` / `## Deprecated Requirements` subsections. **No `[NEEDS CLARIFICATION]` markers** — per §1a they are intake-only; an under-specified requirement at apply becomes a graded SRAD `## Assumptions` row, not a marker. (This resolves the §1a-vs-§4 contradiction the review flagged.)
  - **Scrub the existing spec.md references already baked into the template:** the `**Spec**: \`spec.md\`` frontmatter line (template ~L6) and the Acceptance-derivation comments that cite "spec.md" (~L92/99/118) — repoint all to the in-file `## Requirements` section.
  - Make the **traceability contract** explicit: each `## Tasks` item carries a `<!-- R# -->` trace annotation (today `_generation.md` line ~85 treats cross-linking as OPTIONAL — flip to REQUIRED); each `## Acceptance` item names the requirement it accepts (`A-001 R2: {outcome}`).
  - Stable headings (`## Requirements`/`## Tasks`/`## Acceptance`) and `R#`/`T{NNN}`/`A-{NNN}` ID formats are the parser contract.
- **Artifact names are NOT renamed.** `intake.md`, `plan.md` keep their names. Rationale: artifacts are named for *what they are*, not the *stage* that runs them — renaming `plan.md`→`apply.md` would leak the stage name into the user-visible filename, and churn ~40 files for zero benefit. Matches the tasks→apply precedent.
- **`src/kit/VERSION`** bump (currently `1.9.7`). A stage-model + scoring change is at least a **minor** bump → target `1.10.0`. The VERSION bump and the migration's `TO` version MUST be identical (the migration filename's `TO` is how `/fab-setup migrations` decides the file is in range), and `FROM` must equal the currently-shipped VERSION. So the migration is named **`1.9.7-to-1.10.0.md`** and ships in the same PR as the VERSION bump.
- **New migration** `src/kit/migrations/1.9.7-to-1.10.0.md`: walk in-flight `fab/changes/**` (exclude `archive/**`, untouched). Per change:
  - **Idempotency sentinel:** skip the spec.md→plan.md merge if `plan.md` already contains a `## Requirements` heading OR a `<!-- migrated from spec.md -->` marker.
  - **Four-state case table** (mirroring the precedent's step-2 structure):
    1. *spec.md only (no plan.md)* — change is mid-spec/pre-apply. Do **not** create a plan.md stub (a stub would trip `fab-continue.md`'s "plan.md exists → skip generation" resumability guard and deadlock plan generation). Leave `spec.md` in place; the new Plan Generation Procedure detects a legacy `spec.md` and folds it into `## Requirements` on first apply (this requires a **one-release legacy-spec.md ingestion path** in `_generation.md` — added to §5).
    2. *plan.md only (no spec.md)* — already past apply or never had a spec; only the `.status.yaml`/progress rewrite applies.
    3. *both* — merge `spec.md` body into `plan.md` as a `## Requirements` section (annotate `<!-- migrated from spec.md -->`); leave `spec.md` with a "safe to delete" comment.
    4. *neither* — pre-planning; only the progress rewrite applies.
  - `.status.yaml`: drop `progress.spec` and fold its state into `apply` (tasks-migration logic: if spec was `active`/`ready`, carry that level to apply; if `done`/`skipped`, leave apply as-is). The `confidence.indicative` key is **left on disk** (the binary tolerates it on read; it's simply no longer written) — the migration does not need to strip it.
  - **`stage_directives.spec` in `config.yaml`:** unlike the empty `tasks: []` the precedent pruned, `spec:` carries **4 real directives** (GIVEN/WHEN/THEN, [NEEDS CLARIFICATION], impact docs, methodology synthesis). **Relocate** their content into `stage_directives.apply` (apply now owns requirement generation) rather than silently dropping it — these may be user-customized in real projects. Document the relocation.
  - Fully idempotent (re-run hits the sentinel); `archive/**` untouched.

### 5. Skills (`src/kit/skills/*.md` — canonical source, never edit `.claude/skills/`)

- **Heavy**:
  - `_preamble.md` — broader than first scoped. (a) **State Table** ~L268–282: drop the `spec` row. (b) **Confidence Scoring** section — note it *starts at ~L483* (not 516); the whole ~L483–551 carries `indicative`/spec-stage scoring (Schema comment ~L496, Formula ~L510, Invocation ~L530–533, Indicative-vs-Spec ~L537–539, Bulk Confirm). (c) **Skill Invocation Protocol → "Currently Applicable"** table ~L327–328: remove the two `/fab-ff`/`/fab-fff` → `/fab-clarify [AUTO-MODE]` mappings (this change deletes them). (d) **Skill-Specific Autonomy Levels** table ~L411–412: "Recomputes confidence? — Spec stage only" becomes false → "No (intake-only, via `/fab-clarify`)"; update escape-valve wording. (e) **Context Loading** ~L58 ("load … intake.md, spec.md, plan.md") and **Memory File Lookup** ~L72 ("spec's Affected memory metadata"): drop spec.md. (f) **Assumptions Summary** ~L478 ("in spec.md"): repoint to plan.md.
  - `fab-continue.md` — dispatch table ~47–58 (drop the `spec` rows; intake-ready now starts apply directly), reset flow ~182–202, the **"Spec stage only" scoring step ~74** (remove; no scoring at apply entry — intake is authoritative), and the **`tasks`-deprecation strings ~185/202** that point to `/fab-clarify spec` (repoint to `/fab-continue apply` / `/fab-clarify intake`), and the **review-fail "Revise spec" rework tier ~159** (see below).
  - `fab-ff.md` & `fab-fff.md` — remove the spec generation Step 1 (~50–58); **delete BOTH auto-clarify invocations**: the post-spec-gen one (~58) AND the on-plan one (~66) — per §1a, no clarify runs in the bracket; consolidate to the single intake gate; **redefine the "Revise spec → reset to spec stage" rework tier** (~ff:89 / fff:85) since that target is gone → "Revise requirements → edit `plan.md` `## Requirements` + downstream `## Tasks`/`## Acceptance`, re-run apply" (this keeps the rework tiers distinct, preserving the §1b convergence guarantee); update hardcoded `>= 3.0` stop-message strings (~ff:15/30, fff:15/30) and the dangling `/fab-clarify spec|plan` recovery hints (~ff:105, fff:172/174).
  - `_generation.md` — the **Spec Generation Procedure is merged into the Plan Generation Procedure**: one walk emits `## Requirements` + `## Tasks` + `## Acceptance` into `plan.md`; the standalone Spec Generation Procedure is deleted; Plan Generation step ~3 ("Walk `spec.md` requirements once") → "Walk the `## Requirements` just generated"; drop the step-2 "Keep the Spec link" instruction. **Add a one-release legacy `spec.md` ingestion path** (per §4 migration state 1): if a legacy `spec.md` exists and `plan.md` has no `## Requirements`, fold it in on first apply.
  - `_review.md` — more than a source swap. Repoint each spec.md touchpoint to `plan.md` `## Requirements`: sub-agent context-file list (~L35), "Spot-check spec" step (~L44), parsimony question (~L53), and the removal-verification-vs-deletion-candidates distinction (~L75, "planned removals declared in spec.md" → "in `## Deprecated Requirements`"). (Leave ~L124 "spec files" — that means `docs/specs/`, not the change spec.)
- **Moderate**:
  - `_cli-fab.md` — finish transition chain ~72 (drop `intake→spec`/`spec→apply`), `fab score` modes ~84–87 (the spec/intake mode split + "Intake gate Fixed threshold 3.0" row), the `set-confidence`/`set-confidence-fuzzy` `[--indicative]` signatures ~65–66.
  - `fab-clarify.md` — becomes **intake-only**: `intake` is the sole target; drop `spec` and `plan` targets (and the legacy-`tasks` error string ~26 that references them); remove post-planning artifact-default logic ~36–37; **invert** the recompute-confidence guard ~172–174 — today it *skips* at intake ("Skip this step if at intake stage"); now it must *always* run `fab score --stage intake <change>` (the only stage it operates at).
  - `git-pr.md` — **reclassified from Light to Moderate** (it reads spec.md). Correct the line ref: the seven-stage pipeline enumeration is at **~L249** ("List the seven pipeline stages … intake → spec → apply → …"), not ~70. Drop `spec`, change "seven"→"six". Remove the spec PR-link logic: `{has_spec}` (~L174), Spec blob URL (~L184), the `spec → Spec URL` row (~L253). (The "7 valid *types*" at ~L35/37 are change types — leave them.)
  - `fab-new.md` & `fab-draft.md` — **reclassified from Light to Moderate**. Each has a full `### Step 7: Indicative Confidence` section (fab-new ~92–102, fab-draft ~94–104) whose body is now false: rename to plain "Confidence", drop the `indicative: true` persistence and the "spec-stage score overwrites it" sentences (~102/104), update output-format lines (~204/137).
- **Light**: `fab-status.md` (progress counter `(1/7)` → `(1/6)`), `fab-operator.md` (stage-sequence docs).
- **None**: `fab-archive.md`, `fab-discuss.md`, `fab-help.md`, `git-branch.md`, `git-pr-review.md`, `fab-proceed.md`, `docs-*`, `internal-*`. *(Note: `fab-switch.md` moved OUT of None — it parses the `:indicative` list field, see §3 / Moderate below.)*
- **Moderate (added)**: `fab-switch.md` — parses the `name:display_stage:display_state:score:indicative` 5-field row from `fab change list` (~L90/97); update the format/parsing to the new field set when the `:indicative` ABI changes.

### 6. Specs (`docs/specs/`) and per-skill SPECs

- **Heavy**: `overview.md` (the "5 Core Stages"/"7 with…" section, mermaid, stage table → 6 stages), `srad.md` (gate/scoring/indicative; the "after spec generation" invocation note), `change-types.md` (`expected_min` table — reconcile with the Go values per §2; gate thresholds table → flat 3.0; the Tier-1 PR-template "links to intake and spec" ~L65), `skills.md` + affected `skills/SPEC-*.md`.
- **Medium**: `architecture.md` (`.status.yaml` progress examples, `stage_directives` incl. the relocated spec directives), `glossary.md` (Spec stage definition — now a `plan.md` section, not a stage), `templates.md` (status progress map), `user-flow.md` (diagrams).
- **Non-skill specs that the skill→SPEC coupling rule does NOT catch** (must be named explicitly): `docs/specs/skills/SPEC-hooks.md` (~L37–60 "After spec.md is written → fab score", ~L154 the `intake.md|spec.md|plan.md` matcher, ~L207–227 dispatch tree — all tied to the removed hook/matcher), `docs/specs/skills/SPEC-preamble.md` (State Table, Confidence Scoring, indicative, "after spec generation"), `docs/specs/skills/SPEC-_review.md` (~L20/26/74/93/99 spec.md touchpoints, mirroring `_review.md`), `docs/specs/assembly-line.md` (~L89/121/140 "spec → tasks → apply" pipeline narration), `docs/specs/index.md` (~L18 stage count — already stale, says "6 stages" while code has 7; ~L21 lists `spec` as a template).
- **Constitution coupling note:** skill source changes MUST update the corresponding `docs/specs/skills/SPEC-*.md`. The partials `_generation.md` and `_cli-fab.md` have **no** dedicated SPEC file — their behavior is specified in `skills.md`/`SPEC-hooks.md`/`SPEC-preamble.md`, so the coupling is satisfied by updating those, not a (nonexistent) `SPEC-_generation.md`.

### 7. Memory (`docs/memory/fab-workflow/`)

Hydrate-stage updates (post-implementation): `change-lifecycle.md` (7→6 stages, phase split), `planning-skills.md` (planning = intake only), `execution-skills.md` (apply entry generates unified `plan.md` then executes), `clarify.md` (intake-only; in-loop SRAD; intake gate is the guard), `schemas.md` (progress map drops `spec`, `indicative` retired), **`templates.md`** (has a dedicated `### spec.md` section ~L22, a `spec: pending` progress key ~L80, a "Single spec.md Without Delta Markers" section ~L141, and spec-stage clarify rationale ~L155 — fold requirements into the plan.md doc, drop the spec.md section and progress key), **`kit-architecture.md`** (~L54 lists `templates/spec.md` in the kit tree — remove). These are updated at hydrate, not apply.

### 8. Constitution-mandated coupling

- Go changes MUST include test updates (e.g., `internal/status/status_test.go` line ~186 enumerates valid stages) AND update `src/kit/skills/_cli-fab.md`.
- Skill file changes MUST update the corresponding `docs/specs/skills/SPEC-*.md`.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) pipeline 7→6 stages, planning phase loses spec; artifacts `intake.md → plan.md → code`
- `fab-workflow/planning-skills`: (modify) planning = intake only; `spec.md` absorbed into `plan.md` `## Requirements`, co-generated at apply entry
- `fab-workflow/execution-skills`: (modify) apply entry generates unified `plan.md` (requirements + tasks + acceptance) in one pass, then executes
- `fab-workflow/clarify`: (modify) `/fab-clarify` becomes intake-only; no post-intake clarify; in-loop ambiguity → SRAD assumptions in `plan.md`; the intake gate (not a runtime bounce) is the guard
- `fab-workflow/schemas`: (modify) `.status.yaml` progress map drops `spec`; `confidence.indicative` no longer written (tolerated on read); `plan.md` carries `## Requirements`
- `fab-workflow/configuration`: (modify) `stage_directives.spec` content relocated into `stage_directives.apply`; flat 3.0 intake gate
- `fab-workflow/migrations`: (modify) note the new spec→apply migration (incl. spec.md→plan.md body merge + four-state case table) alongside the tasks→apply precedent
- `fab-workflow/templates`: (modify) drop the `### spec.md` section + `spec: pending` progress key; document `## Requirements` as a `plan.md` section
- `fab-workflow/kit-architecture`: (modify) remove `templates/spec.md` from the kit tree listing

## Impact

- **Go binary** (highest-risk — state-machine + scoring core): `internal/statusfile`, `internal/status`, `internal/change`, `internal/score`, `internal/preflight`, `internal/log`, `internal/hooklib`, `cmd/fab` (`score.go`, `status.go`, `hook.go`) + their `_test.go` files. The hook/matcher (`hooklib/artifact.go`, `cmd/fab/hook.go`) and the `fab change list` 5-field ABI (`internal/change`) are the easily-missed corruption/break surfaces.
- **Skills**: 6 heavy (`_preamble`, `fab-continue`, `fab-ff`, `fab-fff`, `_generation`, `_review`) + 4 moderate (`_cli-fab`, `fab-clarify`, `git-pr`, `fab-new`/`fab-draft` — reclassified up — and `fab-switch` for the `:indicative` ABI) + 2 light (`fab-status`, `fab-operator`) skill files (see §5).
- **Specs**: 4 heavy + 4 medium + 5 non-skill specs (SPEC-hooks, SPEC-preamble, SPEC-_review, assembly-line, index) + affected per-skill SPECs.
- **Migration + VERSION**: new `1.9.7-to-1.10.0.md` migration (idempotent, four-state case table, archive-safe) shipped with the `src/kit/VERSION` bump in the same PR.
- **External dependents**: `confidence.indicative` (retired) — `fab change list` emits it as a positional 5th field parsed by `fab-switch.md`; this is an **output ABI**, so the field's removal (or keep-always-false) must be a deliberate decision, not a silent drop. `progress.spec` consumers read via preflight and update with the schema.
- **Backward compatibility**: existing in-flight changes in *user* projects need the migration. `fab status <event> <change> spec` produces a hard deprecation error (mirroring `tasks`). Legacy `spec.md`/`indicative:` on disk are tolerated, not corrupted (see §3/§4).

## Open Questions

- ~~Exact merged `expectedMin` values?~~ **Resolved** (§2): adopt `expectedMinSpec` (feat:7, refactor:6, fix:5; default 3 for docs/test/ci/chore); drop `expectedMinIntake`.
- ~~Hard deprecation error vs. silent no-op for `fab status … spec`?~~ **Resolved** (Assumption #7, §3, Impact): hard deprecation error, mirroring the `tasks` branch.
- ~~Migration handling of mid-spec / spec.md-without-plan.md changes?~~ **Resolved** (§4 four-state case table): state 1 leaves spec.md for on-apply ingestion (no stub, to avoid the resumability-guard deadlock); states 2–4 enumerated.
- ~~Migration version number?~~ **Resolved** (§4): `1.9.7-to-1.10.0.md`, tied to the `src/kit/VERSION` bump in the same PR.
- **Constitution amendment scope** (Assumption #16): bump `Last Amended` — but does the stage-model change warrant a new constitution *principle/clause*, or only a `Last Amended` date + rationale note? Leaning: date + a short note under Additional Constraints; no new MUST-rule. (Confirm at planning.)
- **`gateThresholds` shape**: flat 3.0 for all 7 types now (§2). Keep the per-type *map* (so future divergence is a data change) vs. collapse to a single constant? Leaning: keep the map. Low-stakes; decide at planning.
- **`getExpectedMin`/`CheckGate`/`Compute` `stage` param**: now vestigial (intake-only). Keep for minimal diff vs. simplify away? Decide at planning.
- **`--indicative` CLI flag**: hard-remove vs. keep as accepted-but-ignored no-op for script back-compat? Leaning: keep-as-no-op for one release, then remove.

## Assumptions

<!-- STATE TRANSFER: continuity between the intake agent and the apply-entry agent that
     generates the unified plan.md. No separate spec-stage agent exists post-change. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
<!-- Grades reconciled with the SRAD composite formula (0.25·S + 0.30·R + 0.25·A + 0.20·D):
     Certain ≥85, Confident 60–84. Rows whose composite lands 60–84 are labeled Confident even
     when the decision feels settled — the low Reversibility (R) of a cascading pipeline change is
     a real signal, not a labeling error. Only #2/#5/#13 legitimately compute ≥85. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Merge `spec` into `apply` as the first apply-entry sub-step (requirements → plan → execute), core pipeline 7→6 stages | Direct user request; exact precedent exists (tasks→apply v1.9.0). Composite 82.3 (low R: cascades widely) | S:95 R:60 A:90 D:90 |
| 2 | Certain | Change type is `refactor` | User specified; matches taxonomy. Composite 94.3 | S:98 R:90 A:95 D:95 |
| 3 | Confident | Single intake gate at **flat 3.0 for all 7 types** (≥ every old gate, so no entry bar relaxed) | User chose single-gate-per-type; refined to flat 3.0 to avoid weakening fix/docs/etc. Composite 80.8 | S:95 R:55 A:90 D:90 |
| 4 | Confident | Stop writing the `confidence.indicative` flag (tolerate on read); intake scoring becomes authoritative | Direct consequence of the chosen gate model; user-confirmed. Composite 78.6 | S:90 R:55 A:88 D:88 |
| 5 | Certain | `fab score` default `--stage` flips from `spec` to `intake` | Required by single-intake-gate model; mechanical. Composite 88.0 | S:92 R:80 A:92 D:90 |
| 6 | Confident | Ship a NEW idempotent migration `1.9.7-to-1.10.0.md` (four-state case table; drop `progress.spec`→apply; relocate `stage_directives.spec`→apply; leave `archive/` untouched) | Constitution mandates migrations for user-data restructuring; tasks→apply is the template. Composite 70.8 | S:85 R:50 A:88 D:80 |
| 7 | Confident | `fab status <event> <change> spec` returns a hard deprecation error mirroring the `tasks` branch in `validateStage` | Consistency with the established `tasks` precedent. Composite 74.7 | S:75 R:65 A:85 D:78 |
| 8 | Confident | Affected-file map is accurate and complete — *strengthened* by a 52-finding adversarial review that added hook.go/artifact.go/status.go/VERSION + test/spec/memory files the first sweep missed | Three Explore sweeps + an adversarial verify pass with codebase checks. Composite 76.0 | S:85 R:70 A:82 D:78 |
| 9 | Confident | `/fab-clarify` becomes intake-only; no clarify runs after intake (not even `[AUTO-MODE]` — both ff/fff auto-clarify steps deleted). `[NEEDS CLARIFICATION]` markers are intake-only, never in `plan.md` | User decision: clarification is not a separate skill in the automated zone. Composite 79.0 | S:90 R:55 A:88 D:90 |
| 10 | Confident | Inside apply, ambiguity is resolved by the apply agent recording graded SRAD assumptions in `plan.md` `## Assumptions` — not by any clarify ceremony | User: "the agent can clarify issues it finds… an inbuilt part of apply." Composite 77.3 | S:88 R:55 A:85 D:88 |
| 11 | Confident | The "bounce-back" guard is the **intake gate itself** — a failing intake never reaches `done`, so the orchestrators (gated on `fab score --check-gate`) cannot enter apply. **No new runtime mechanism is built**; apply needs no Unresolved/bail logic | Corrected from the original (wrong) "apply detects Unresolved and resets" framing — it's the existing gate. SRAD Critical Rule still applies at intake. Composite 74.5 | S:85 R:50 A:85 D:85 |
| 12 | Confident | `spec.md` is absorbed into `plan.md` as a `## Requirements` section; the `spec.md` template is removed. Final artifacts: `intake.md` → `plan.md` → code | User chose "absorb into plan.md"; full consumer set verified (score.go Compute+CheckGate, hook.go, hooklib/artifact.go, git-pr.md) and all removed/repointed. Composite 79.7 | S:95 R:50 A:90 D:92 |
| 13 | Certain | Artifacts are NOT renamed (`plan.md` stays `plan.md`; no `apply.md`) | User-affirmed; artifact-named not stage-named; avoids ~40-file churn. Composite 87.9 | S:92 R:80 A:90 D:92 |
| 14 | Confident | Trace annotations become REQUIRED: each `## Tasks` item carries `<!-- R# -->`; each `## Acceptance` item names its `R#`. Parser contract = stable `## Requirements`/`## Tasks`/`## Acceptance` + `R#`/`T{NNN}`/`A-{NNN}` | Traceability makes the autonomous `(apply↔review)` loop converge; today cross-linking is OPTIONAL (`_generation.md` ~L85). Composite 75.1 | S:80 R:60 A:82 D:78 |
| 15 | Confident | Adopt `expectedMinSpec` (feat:7, refactor:6, fix:5; default 3 for the rest) as the single intake `expectedMin`; drop `expectedMinIntake`. Reconcile `change-types.md` to the Go values | Clarified — user agreed. Composite 74.0 (low R). Open: count ≠ reversibility-realism (see §design tradeoff / Open Questions) | S:95 R:45 A:75 D:90 |
| 16 | Confident | Treat the stage-model change as constitution-touching: bump `Last Amended` + rationale note (scope of any new clause is an Open Question) | Clarified — user confirmed ("it can be"). Composite 75.0 | S:90 R:60 A:70 D:85 |
| 17 | Confident | Accept the lost independent assumption re-grade (old spec stage re-evaluated intake assumptions). Compensated by the flat-3.0 intake gate, the ~1% spec-rework loom evidence, and requirement-correctness still caught at review | New, surfaced by review finding #25. Deliberate tradeoff, documented in Why, not hidden. Composite ~74 | S:80 R:55 A:80 D:75 |

17 assumptions (3 certain, 14 confident, 0 tentative, 0 unresolved).

> **Note on the grade distribution.** Only 3 of 17 decisions compute to *Certain*; the rest are *Confident*, held down by Reversibility scores of 45–70. That is not under-confidence in the *decisions* (most are user-confirmed) — it is an honest reading that this is a **high-blast-radius, hard-to-reverse structural change**: it rewrites the state machine, the scoring core, a public output ABI, and ships a migration over user data. The SRAD formula correctly refuses to call those "Certain." The practical consequence is a *lower* `fab score` than the label-inflated version (see below) — which is the right signal for a change of this scope, and exactly the kind of honesty the proposal itself is trying to build into the pipeline.
