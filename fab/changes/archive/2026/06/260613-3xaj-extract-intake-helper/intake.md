# Intake: Extract `_intake.md` Shared Helper for Pre-Boundary Intake Creation

**Change**: 260613-3xaj-extract-intake-helper
**Created**: 2026-06-13

## Origin

This is a one-shot `/fab-draft` invocation from a natural-language description (no Linear/backlog ID). It is **"Change B"** in the execution plan of an architecture discussion about fab-kit's skill structure that produced three coordinated refactors. The full analysis lives in `docs/findings/intake-is-the-context-boundary.md` (read it — especially the **"Pre-boundary de-duplication: extract `_intake.md`"** section) and its companion `docs/findings/per-stage-model-tier-application.md`.

> refactor: Extract `_intake.md` shared helper for pre-boundary intake creation.
>
> `/fab-new` Steps 0–9 ARE the "create an intake" procedure, but they are duplicated across the pre-boundary skill family via two INCONSISTENT reuse mechanisms both pointing at `fab-new.md`. Lift Steps 0–9 into a new internal helper `_intake.md` parameterized by a single `{questioning-mode}` knob, then rewire the three consumers (`fab-new`, `fab-draft`, `fab-proceed`) to thin call-sites. This is the lowest-risk of the three coordinated refactors and the recommended starting point. It is INDEPENDENT of the other two changes (different files, no shared seam) and touches ZERO Go code — a pure skill restructure.

The principle behind the finding: there is exactly **one context-bearing boundary** in the whole pipeline — **intake**. Up to and including intake creation runs in the **main session context** (it needs the live conversation): this is `/fab-new`, `/fab-draft`, the intake-creation prefix of `/fab-proceed`, and `/fab-clarify`. After intake, the intake artifact **IS the context** and every subsequent stage runs as a dispatched subagent. The post-boundary family's shared orchestration was already extracted into `_pipeline.md`; the **pre-boundary family has the symmetric duplication and is not yet cleanly extracted**. This change extracts it.

## Why

`/fab-new` Steps 0–9 ARE the "create an intake" procedure, but they are duplicated across the pre-boundary skill family via **two INCONSISTENT reuse mechanisms both pointing at `fab-new.md`**:

1. **`/fab-draft` is a prose delta**: "execute fab-new's Steps 0–9 exactly as written, read self-name as `/fab-draft`", then skip Steps 10–11. This is **fragile** — it carries an explicit warning about the **"run activation by momentum" failure mode**, precisely BECAUSE the steps it must NOT run (activate / branch) live in the same body it executes. (See `fab-draft.md` § Behavior delta #2: *"running activation or branch creation by momentum is the known failure mode of this delta form — before any `fab change switch` or `git` invocation, re-check that you are executing `/fab-draft`."*)

2. **`/fab-proceed` dispatches `/fab-new` as a subagent** with a promptless defer-and-surface contract that replaces Step 8's interactive questioning.

So `fab-new.md` is **simultaneously a skill AND the de-facto shared library** the other two reach into — by two different routes.

The artifact-**GENERATION** mechanics are *already* extracted: `_generation.md` § **Intake Generation Procedure** handles read-template / fill-metadata / write-sections / append-`## Assumptions` / write-file (Step 5 only). What is NOT extracted is the surrounding **ORCHESTRATION** — Steps 0–4, 6–7, 9.

**What happens if we don't fix it**: the duplication persists across two routes, `fab-draft` keeps carrying its momentum-warning band-aid, and `fab-new.md` stays a library masquerading as a skill. Editing the intake-creation procedure means reasoning about three call sites with two reuse mechanisms.

**Why this approach (mirror `_pipeline.md`)**: the post-boundary family already proved this exact shape — a shared body parameterized by one knob, with call-site-specific tails staying in the call-site files (just as `fab-fff`'s ship/review-pr Steps 4–5 stay in `fab-fff.md`, and `_pipeline.md` is parameterized by `{driver}`/`{terminal}`). Completing the symmetry gives one shared helper per phase:

| Phase | Helper | Consumers |
|-------|--------|-----------|
| artifact mechanics | `_generation.md` | new, draft, continue, ff, fff |
| review mechanics | `_review.md` | continue, ff, fff |
| post-intake orchestration | `_pipeline.md` | ff, fff |
| **pre-intake orchestration** | **`_intake.md`** *(this change)* | **new, draft, proceed** |

## What Changes

### 1. Create new internal helper `src/kit/skills/_intake/SKILL.md`

> NOTE on path: the **canonical source** is `src/kit/skills/_intake/SKILL.md` (per the constitution, `src/kit/` is canonical). `.claude/skills/` holds gitignored deployed copies produced by `fab sync` — do NOT create or edit the file there. The directory-per-skill layout (`_intake/SKILL.md`) matches every other source skill (e.g. `src/kit/skills/_pipeline/SKILL.md`).

Frontmatter MUST match the established internal-helper pattern of `_pipeline`/`_review`/`_generation`/`_srad`:

```yaml
---
name: _intake
description: "..."   # describe the Create-Intake Procedure (Steps 0–9, parameterized by {questioning-mode})
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
```

The body defines the **"Create-Intake Procedure" = fab-new Steps 0–9**, parameterized by a **SINGLE knob**:

- **`{questioning-mode}`**:
  - `interactive` — used by `fab-new` and `fab-draft`. Step 8 asks the user via SRAD (the existing behavior: SRAD-driven question selection, no fixed cap, conversational mode when 5+ Unresolved).
  - `promptless-defer` — used by `fab-proceed`. Step 8 records each would-be-asked Unresolved decision as a **deferred Unresolved row** instead of asking, per the **`_srad.md` promptless-dispatch carve-out**. That carve-out (verbatim from `_srad.md` § Critical Rule): *"each would-be-asked Unresolved decision is recorded as an Unresolved row with Rationale `Deferred — promptless dispatch` and surfaced to the user by the dispatcher. The intake gate is the structural backstop — `fab score` returns 0.0 whenever any Unresolved row exists, so a deferred decision always blocks the automated bracket until resolved via `/fab-clarify`."*

This is the **ONLY behavioral fork** in intake creation, and it is legitimately **invocation-level** (who resolves ambiguity: human-now vs. defer-and-surface). It is exactly parallel to the post-boundary autonomy fork (interactive rework menu vs. autonomous auto-rework).

**Steps 0–9 to lift into `_intake.md`** (from `fab-new.md`, with the `{questioning-mode}` parameterization applied only at Step 8):

- **Step 0 — Parse Input**: detect input type in order — Linear ticket ID (`[A-Z]+-\d+`, fetch via `mcp__claude_ai_Linear__get_issue`, fall back to NL on failure); Backlog ID (`[a-z0-9]{4}`, search `fab/backlog.md` for `\[{id}\]`, check optional trailing `[ISSUE_ID]` bracket); Natural language (as-is).
- **Step 1 — Generate Slug**: 2–6 word kebab slug, no articles/prepositions, SHALL NOT include the Linear issue ID; passed to `fab change new --slug`.
- **Step 2 — Gap Analysis**: check for existing mechanisms / scope concerns; if covered present findings and let user decide, else proceed.
- **Step 3 — Create Change**: includes the re-run/collision check (backlog ID via `fab resolve --id {id}` equality check + `fab resolve --folder {id}`; Linear ID via `grep -lw "{ISSUE_ID}" fab/changes/*/.status.yaml`), the route-to-resume behavior, the natural-language-re-run-creates-new semantics, the `fab change new` flags (`--slug`, `--change-id` only when backlog ID, `--log-args`), and `fab status add-issue` when a Linear ticket was detected.
- **Step 4 — Conversation Context Mining**: extract Decisions made / Alternatives rejected / Constraints identified / Specific values agreed from the conversation into graded SRAD assumptions. (The finding reframes this as the load-bearing **context-flush at the boundary**, not merely a scoring optimization — capture that framing.)
- **Step 5 — Generate `intake.md`**: delegate to `_generation.md` § Intake Generation Procedure (already extracted; `_intake.md` references it, does not inline it).
- **Step 6 — Verify Change Type**: the PostToolUse intake-write hook owns `change_type`; verify by reading `.status.yaml`, override only if wrong via `fab status set-change-type` (re-verify after later intake writes).
- **Step 7 — Confidence**: `fab score --stage intake <change>` (normal mode, not `--check-gate`), persist + display `Confidence: {score} / 5.0 ({N} decisions)`.
- **Step 8 — SRAD-Based Question Selection** *(the parameterized step)*: `interactive` vs. `promptless-defer` per `{questioning-mode}` above.
- **Step 9 — Advance Intake to Ready**: `fab status advance {name} intake`.

> A self-name handling note belongs in `_intake.md`: where the lifted Step 4 text refers to "this `/fab-new` invocation", the helper text should read generically (it is invoked by new/draft/proceed). This is the existing `fab-draft.md` "read self-name mentions as `/fab-draft`" concern, now resolved structurally by genericizing the helper rather than via a per-consumer prose instruction.

### 2. Rewire `fab-new.md` to a thin call-site

`fab-new.md` becomes **`_intake(interactive)` + Step 10 (activate) + Step 11 (git branch) as its own tail.**

- Replace the inline Steps 0–9 body with a reference: read `_intake.md` and execute the Create-Intake Procedure with `{questioning-mode} = interactive`.
- **The activate (Step 10) + branch (Step 11) tail STAYS in `fab-new.md`** — it is a **different responsibility** (make the change active + checked out vs. queue it), NOT a questioning-mode parameter. Keep Step 10 (`fab change switch "{name}"`), Step 11 (the full git-branch table — verify-in-repo guard, the 6-row evaluate-in-order table, the fab-new-specific `{dirty_count}` derivation excluding `fab/changes/{name}/`, the dirty-tree note, the keep-in-sync-with-git-branch.md comment), the Output block (with `Activated:` and `Branch:` lines), and the activation/git Error Handling rows in `fab-new.md`.
- Add `_intake` to `fab-new.md`'s frontmatter `helpers:` list. `fab-new` currently declares `helpers: [_generation, _srad]`. Decision needed (see Open Questions / Assumptions): whether `fab-new` still declares `_generation`/`_srad` directly or inherits them transitively via `_intake`.

### 3. Rewire `fab-draft.md` to a thin call-site

`fab-draft.md` becomes **`_intake(interactive)`, stop at ready.**

- Replace the prose-delta body ("execute fab-new's Pre-flight, Arguments, and Steps 0–9 … skip 10–11") with: read `_intake.md` and execute the Create-Intake Procedure with `{questioning-mode} = interactive`; do NOT activate; do NOT create a git branch; stop after Step 9.
- **The "don't run Steps 10–11 by momentum" warning should EVAPORATE**, because those steps no longer live in the body draft executes — they live in `fab-new.md`'s tail, which `fab-draft` never reads. Remove delta #2's momentum warning.
- Keep `fab-draft`'s own Output block (fab-new's Output minus `Activated:`/`Branch:` lines, ending with the Activation Preamble `Next:` line per `_preamble.md` § Activation Preamble), Key Properties table, and Error Handling (no activation/git rows).
- Update `fab-draft.md`'s frontmatter `helpers:` to add `_intake` (currently `helpers: [_generation, _srad]`; same transitive-vs-direct decision as fab-new).

### 4. Rewire `fab-proceed.md`'s fab-new subagent dispatch

`fab-proceed.md`'s fab-new subagent dispatch becomes **`_intake(promptless-defer)`.**

- The subagent that today is dispatched as "`/fab-new` with a promptless defer-and-surface contract" instead dispatches the **`_intake` Create-Intake Procedure with `{questioning-mode} = promptless-defer`**.
- **`/fab-proceed`'s state-detection + relevance-assessment logic STAYS** in `fab-proceed.md` — its dispatch table, asymmetric-bias rule, and bypass notes (the bulk of its ~241 lines) are NOT intake creation. They decide **WHETHER** to call `_intake` at all (create new vs. activate an existing draft). Only the create-an-intake sub-operation is rerouted through `_intake`.
- Keep `/fab-proceed`'s Conversation Context Synthesis behavior as the dispatcher-side context-flush (it surfaces deferred Unresolved rows to the user).

### EXTRACTION BOUNDARY (critical — do NOT over-extract)

`_intake.md` = **"given I've decided to create an intake, do it (Steps 0–9), with `{questioning-mode}` as the one knob."** Whether to create one, and what to do after (activate / branch), stays at the call site. Two things explicitly stay at the call site, or the extraction recreates the dual-mode problem on the intake side:

- **Activate (Step 10) + branch (Step 11)** — different responsibility, stays as a tail in `fab-new.md`.
- **`/fab-proceed`'s state detection + relevance assessment** — decides *whether* to call `_intake`, stays in `fab-proceed.md`.

Mirror the proven `_pipeline.md` shape exactly: shared body parameterized by one knob, call-site-specific tail stays in the call-site file.

### 5. Update `_preamble.md` "Allowed values" helper allowlist

`_preamble.md` § Skill Helper Declaration currently enumerates the **Allowed values** for the `helpers:` frontmatter key as: `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`, `_pipeline`. Adding `_intake` as a new helper means **this allowlist MUST be updated** to include `_intake`. (Edit the canonical source `src/kit/skills/_preamble.md`.)

### 6. Spec updates (constitution-mandated)

Per the constitution, *"Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file."* The following spec files exist and must be reconciled with the skill edits:

- **`docs/specs/skills/SPEC-_intake.md`** *(new)* — create the spec for the new helper (mirror the format of `SPEC-_pipeline.md` / `SPEC-_generation.md` / `SPEC-_srad.md`).
- **`docs/specs/skills/SPEC-fab-new.md`** *(modify)* — reflect the thin-call-site rewire + retained activate/branch tail.
- **`docs/specs/skills/SPEC-fab-draft.md`** *(modify)* — reflect the thin-call-site rewire + removed momentum warning.
- **`docs/specs/skills/SPEC-fab-proceed.md`** *(modify)* — reflect the `_intake(promptless-defer)` dispatch rewire.
- **`docs/specs/skills/SPEC-_preamble.md`** *(modify)* — reflect the updated `helpers:` allowlist.

## Affected Memory

- `pipeline/{file}`: (modify) — the planning/clarify/execution skills domain. The `fab-new` / `fab-draft` / `fab-proceed` intake-creation behavior changes from three duplicated/delta routes to a single shared `_intake` helper parameterized by `{questioning-mode}`. Record the new helper and the call-site/tail boundary.
- `memory-docs/{file}`: (modify) — the relationship between the already-extracted `_generation` generation mechanics and the new `_intake` orchestration helper (one shared helper per phase: generation mechanics vs. pre-intake orchestration). Capture the completed helper symmetry table (`_generation` / `_review` / `_pipeline` / `_intake`).

> Exact file names within these domains are to be resolved at hydrate against `docs/memory/{domain}/index.md` (3-hop walk per `_preamble.md` § Memory File Lookup). Listed at domain granularity because spec-level behavior of the pre-boundary skill family changes.

## Impact

**Affected files (skill sources — `src/kit/skills/` canonical only; never `.claude/skills/`):**
- `src/kit/skills/_intake/SKILL.md` — NEW helper file.
- `src/kit/skills/fab-new.md` — rewire to `_intake(interactive)` + activate/branch tail; add `_intake` to `helpers:`.
- `src/kit/skills/fab-draft.md` — rewire to `_intake(interactive)`, stop at ready; remove momentum warning; add `_intake` to `helpers:`.
- `src/kit/skills/fab-proceed.md` — reroute the fab-new subagent dispatch to `_intake(promptless-defer)`; keep state-detection/relevance-assessment.
- `src/kit/skills/_preamble.md` — add `_intake` to the `helpers:` Allowed-values allowlist.

**Affected files (specs — constitution-mandated):**
- `docs/specs/skills/SPEC-_intake.md` (new), `SPEC-fab-new.md`, `SPEC-fab-draft.md`, `SPEC-fab-proceed.md`, `SPEC-_preamble.md` (modify).

**Zero Go code.** Pure skill restructure. The finding confirms the Go state machine is already caller-agnostic (`driver` is metrics-only) and this change does not touch dispatch/state behavior at all.

**Deployment note**: after editing `src/kit/skills/`, the deployed copies in `.claude/skills/` are regenerated by `fab sync` (out of scope for the diff — the diff touches only canonical sources + specs).

**Risk**: lowest of the three coordinated refactors. Independent of the other two (per-stage-model-tier-application and the post-intake single-mode change) — different files, no shared seam. Reversible (skill prose only). The main correctness concern is **behavioral parity**: the lifted Steps 0–9 must produce byte-identical behavior for `interactive` mode (new/draft) as the current inline versions, and the `promptless-defer` mode must exactly preserve `/fab-proceed`'s current contract.

## Open Questions

- Should `fab-new.md` / `fab-draft.md` continue to declare `_generation` and `_srad` directly in their `helpers:` lists, or inherit them transitively through `_intake` (which itself would reference/declare them)? `_pipeline.md` precedent: consumers (`fab-ff`/`fab-fff`) still declare the underlying helpers (`_generation`, `_review`, `_srad`, `_pipeline`) directly. See Assumptions row 4.
- Does `_intake.md` declare its own `helpers:` (`_generation`, `_srad`), or does the helper-declaration model only apply to user-invocable skills? Internal helpers (`_pipeline`, `_review`, `_generation`) today carry no `helpers:` frontmatter and rely on the consumer to have loaded what they reference. See Assumptions row 5.

## Assumptions

<!-- STATE TRANSFER: sole continuity mechanism between this intake-stage agent and the
     apply-entry agent. Every row substantive. Intake includes all four grades. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Canonical source path is `src/kit/skills/_intake/SKILL.md` (directory-per-skill); never create/edit in `.claude/skills/` (gitignored deployed copies). | Constitution: `src/kit/` is canonical, `.claude/skills/` is `fab sync` output. Description states this explicitly. Every existing source skill uses the `{name}/SKILL.md` layout. | S:100 R:90 A:100 D:100 |
| 2 | Certain | `_intake` frontmatter is `user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`. | Description states it explicitly and it matches `_pipeline`/`_review`/`_generation`/`_srad` verbatim. | S:100 R:85 A:100 D:100 |
| 3 | Certain | `{questioning-mode}` (`interactive` \| `promptless-defer`) is the SOLE parameter; activate/branch and proceed's state-detection stay at the call site. | Description's EXTRACTION BOUNDARY section is explicit and emphatic ("do NOT over-extract"); finding restates it. | S:100 R:80 A:95 D:100 |
| 4 | Confident | `fab-new.md` and `fab-draft.md` keep declaring `_generation` and `_srad` directly in `helpers:` AND add `_intake`. | `_pipeline.md` precedent: its consumers declare the underlying helpers directly rather than relying on transitive loading. Low blast radius — frontmatter only, trivially reversible. Front-runner over transitive inheritance. | S:55 R:90 A:70 D:65 |
| 5 | Confident | `_intake.md` carries NO `helpers:` frontmatter; it references `_generation`/`_srad` in-body and relies on the consumer having loaded them (every consumer already declares both). | Matches existing internal helpers (`_pipeline`, `_review`, `_generation` carry no `helpers:`). The `_preamble.md` helper model is consumer-declared, not transitively chained. | S:60 R:85 A:75 D:70 |
| 6 | Confident | Spec updates required: new `SPEC-_intake.md` + modify `SPEC-fab-new.md`, `SPEC-fab-draft.md`, `SPEC-fab-proceed.md`, `SPEC-_preamble.md`. | Constitution mandates SPEC-* updates for skill-file changes. The five edited skill files map 1:1 to these specs (all exist except `SPEC-_intake.md`, which is new). | S:80 R:85 A:90 D:85 |
| 7 | Confident | `_preamble.md` § Skill Helper Declaration "Allowed values" line gains `_intake` (now `_generation, _review, _cli-fab, _cli-external, _srad, _pipeline, _intake`). | Description's third constitution constraint states this explicitly; the current allowlist is verbatim in `_preamble.md` line 105. | S:90 R:90 A:95 D:90 |
| 8 | Confident | Behavioral parity is the acceptance bar: `interactive` mode reproduces current `fab-new`/`fab-draft` Steps 0–9 exactly; `promptless-defer` preserves `/fab-proceed`'s current contract exactly. No behavior change intended — pure restructure. | Finding frames it as de-duplication, not behavior change. The `promptless-defer` carve-out already exists verbatim in `_srad.md` § Critical Rule; this change references it rather than redefining it. | S:75 R:70 A:80 D:75 |
| 9 | Confident | The lifted Step 4 (Conversation Context Mining) text is genericized in `_intake.md` (referring to "the invoking skill" rather than "this `/fab-new` invocation"), structurally retiring `fab-draft`'s "read self-name as `/fab-draft`" instruction. The `{self-name}` parameter alternative is rejected — the text only needs to be invocation-agnostic, not invocation-named. | **User-endorsed (2026-06-13)**: genericize over parameterize. Exact wording is ordinary apply-time prose, not an open design decision. <!-- decided: genericize self-name references in the lifted body; reject {self-name} parameter — user-endorsed 2026-06-13 --> | S:80 R:80 A:75 D:80 |

9 assumptions (3 certain, 6 confident, 0 tentative).
