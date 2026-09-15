# Intake: Right-Size Fab Routing — Anti-Trigger Descriptions + Micro-Change Backstop

**Change**: 260815-slpa-right-size-fab-routing
**Created**: 2026-08-15

## Origin

Created via the shared Create-Intake Procedure in `promptless-defer` mode (promptless dispatch — no questions asked; would-be questions recorded as deferred Unresolved rows). The input is a synthesized description from a prior design discussion in which the key decisions were made explicitly.

> **Right-size fab routing: anti-trigger skill descriptions + intake-entry micro-change backstop.**
> With fab skills always available, agents route trivial fixes through the full fab pipeline. Real example: after a large UI change completed, the user asked to fix a 2px offset — the agent ran an entire fab workflow for it. The routing decision happens in the main session, before any fab skill runs, based on what is always in context: the skill frontmatter `description:` lines. Nothing there says fab is optional or when NOT to use it.
>
> Decisions made in discussion: (1) anti-triggers in the frontmatter descriptions of the routing skills `fab-proceed`, `fab-fff`, `fab-ff`, `fab-new`; (2) a backstop check at intake entry in `/fab-proceed` and `/fab-new` that stops with a message recommending a direct fix when the change meets the micro criteria, requiring an explicit user go-ahead to continue; (3) micro criteria: no Affected Memory impact, no behavior-contract change, ~1-task plan / single-spot edit — tie-breaker default-closed (**when unsure, use fab**), wording minimal and IDENTICAL everywhere it appears. Explicitly REJECTED: a third pipeline lane ("nano-flow"). Explicitly OUT OF SCOPE: the change-types.md micro-tier taxonomy entry and a /fab-setup CLAUDE.md right-sizing rule.

## Why

1. **The pain point**: fab's skills are always available to the host agent, and their frontmatter `description:` lines — the only kit-controlled text in context at routing time — describe only when TO use each skill, never when NOT to. As a result, agents over-route: a trivial follow-up ("fix this 2px offset") triggers a full pipeline run — change folder, branch, intake, plan, review, hydrate, ship — for a single-spot edit. The overhead the pipeline adds is exactly wrong-sized for such work.

2. **The consequence if unfixed**: every micro fix pays minutes of pipeline overhead and pollutes `fab/changes/` with noise changes; users learn to distrust the automation ("it runs a whole workflow for a 2px fix"), which erodes adoption of the pipeline for the changes that genuinely need it.

3. **Why this approach**: the routing decision happens in the **main session, before any fab skill runs**. The one kit-controlled surface the model reads at that moment is the frontmatter `description:` line — so anti-triggers must live there (a rule inside a skill body is read too late). The backstop at intake entry is defense-in-depth for the case where routing already happened: the skill itself recognizes a micro change and stops before creating pipeline state. A third "nano" pipeline lane was rejected because any lane still creates a change folder, branch, and status file — exactly the overhead being complained about. "No-flow" means fab is not invoked at all.

Two conflated cases the wording must distinguish:

1. **Follow-up tweak to a change still in flight / just applied** → amend the current change (same branch, just edit) — never a new pipeline run.
2. **Standalone micro fix** (change already shipped/archived) → direct edit + plain commit, no fab at all ("no-flow").

## What Changes

All edits are to canonical kit sources at `src/kit/skills/*.md` (never `.claude/skills/` — gitignored deployed copies), plus the spec/memory sweep below. No Go code changes. No migration (no user-data restructuring).

### 1. Anti-trigger frontmatter descriptions — four routing skills

Add an explicit anti-trigger to the `description:` frontmatter of exactly these four files:

- `src/kit/skills/fab-proceed.md`
- `src/kit/skills/fab-fff.md`
- `src/kit/skills/fab-ff.md`
- `src/kit/skills/fab-new.md`

Candidate wording from the discussion (final text authored at apply, staying close to this; compressed as needed to keep description lines readable, but the micro-criteria core IDENTICAL across all four):

> "Not for micro fixes — single-spot edits with no spec/memory/behavior-contract impact; make those directly and commit. Not for follow-up tweaks to a change still in flight — amend it instead."

Descriptions are read without loading files, so they carry a **compressed self-contained** form of the criteria (the owner-or-pointer rule's pointer form is unusable at routing time). `fab-draft` and `fab-dedupe` are deliberately excluded — the discussion enumerated the four routing skills only.

<!-- assumed: posture-matched backstop mechanics — fab-proceed stops with a message (zero-prompt posture), fab-new may confirm inline (interactive posture); discussion pinned only fab-proceed's stop-not-prompt behavior -->

### 2. Intake-entry micro-change backstop — `/fab-proceed` and `/fab-new`

Add a backstop check at intake entry to `src/kit/skills/fab-proceed.md` and `src/kit/skills/fab-new.md` (these two only — not `_intake.md`'s other consumers `fab-draft`/`fab-dedupe`, and not `fab-ff`/`fab-fff`, which receive descriptions only):

- **Trigger point**: only on the path that would CREATE a new intake. For `/fab-new`, before executing Create-Intake Steps 0–9. For `/fab-proceed`, only when the dispatch table selects a create-new (`_intake`) row — resuming an existing intake or an active change never triggers the backstop (that is case 1, "amend in flight", handled by the anti-trigger description text, not by this check).
- **Behavior on trigger**: if the described change meets the micro criteria, the skill STOPS with a message recommending a direct fix instead — e.g. `This looks like a direct fix — handle it without fab unless you want a tracked change.` — and requires an explicit user go-ahead to continue into the pipeline.
- **`/fab-proceed` posture constraint (hard requirement)**: the backstop respects `/fab-proceed`'s zero-prompt posture. It stops with a message **like a gate failure** — it does NOT interactively prompt, and it does NOT itself perform the fix. Continuation is via explicit user go-ahead in conversation (e.g. "use fab anyway") detected on re-invocation; no new argument or flag is added (`/fab-proceed`'s no-args/no-flags contract is untouched, and conversation is already its inference surface).
- **`/fab-new`** is interactive by posture; its backstop may confirm inline (an inline "continue anyway?" IS the explicit go-ahead) or stop identically — resolved at apply, with the criteria wording identical either way.

### 3. Micro criteria — minimal, identical, default-closed

The criteria (no canonical spec home yet — the `change-types.md` micro-tier entry was deliberately deferred to a later change):

- no Affected Memory impact,
- no behavior-contract change,
- would be a ~1-task plan / single-spot edit.

Tie-breaker is **default-closed: when unsure, use fab.** The escape hatch must not become a hydrate-skipping loophole — memory drift is the expensive failure fab exists to prevent.

Wording rules: keep the criteria minimal and IDENTICAL everywhere they appear. Per the owner-or-pointer rule (`fab/project/code-quality.md`), the skill-body backstop text has a **single owner with pointers** from the other body; frontmatter descriptions carry the compressed self-contained form (they are read without loading files, so a pointer cannot work there). The owner file for the skill-body criteria text is a deferred decision (see Open Questions / Assumptions #10).

### Rejected alternative (from discussion)

- **Third pipeline lane ("nano-flow")** — REJECTED. Any lane still creates a change folder, branch, and `.status.yaml` — exactly the overhead being complained about. No-flow means fab is not invoked at all.

### Out of scope (deferred to later changes)

- The `docs/specs/change-types.md` micro-tier taxonomy entry (discussion item 2).
- `/fab-setup` appending a right-sizing rule to host-project CLAUDE.md (discussion item 4).

### Spec + sweep obligations

- `docs/specs/skills.md`: the backstop is a **flow change** for `/fab-proceed` and `/fab-new` — update their sections (and any restatement of the four skills' one-line purposes that the new descriptions contradict).
- **Sibling sweep** (code-quality.md § Sibling Sweeps): `fab-ff` ↔ `fab-fff` are a twin class — sweep both together; grep repo-wide for every phrase being changed (aggregate specs `skills.md`/`glossary.md`/`architecture.md` restate per-skill facts); include user-facing string literals and `*_test.go` comments in the sweep.

## Affected Memory

- `pipeline/planning-skills`: (modify) `/fab-new` gains an intake-entry micro-change backstop before Steps 0–9; description anti-trigger noted; `_intake` itself unchanged (backstop stays at the call sites per the EXTRACTION BOUNDARY).
- `pipeline/execution-skills`: (modify) `/fab-proceed` state-detection/dispatch gains the create-new-path backstop (stop-with-message, zero-prompt preserved); `/fab-ff`/`/fab-fff` description anti-triggers noted.

## Impact

- **Files edited (apply)**: `src/kit/skills/fab-proceed.md`, `src/kit/skills/fab-new.md`, `src/kit/skills/fab-ff.md`, `src/kit/skills/fab-fff.md` (frontmatter for all four; body backstop sections in fab-proceed + fab-new only), `docs/specs/skills.md` (flow-change sections).
- **Memory (hydrate)**: the two files above under Affected Memory.
- **Not touched**: Go binaries (no CLI change → no `_cli-fab.md`/test obligations), templates, migrations (no user-data restructuring), `.claude/skills/` (deployed copies — regenerated by `fab sync`), `docs/specs/change-types.md` (out of scope).
- **Behavioral risk**: over-aggressive anti-triggers could deflect legitimate small-but-memory-affecting changes away from fab — mitigated by the default-closed tie-breaker ("when unsure, use fab") being part of the identical criteria wording.
- **Scale**: markdown-only, ~4 skill files + 1 spec file; likely a light-lane (~≤5-task) change.

## Open Questions

- Which file owns the canonical skill-body micro-criteria/backstop text (single owner with pointers)? Candidates: `fab-proceed.md` body with `fab-new.md` pointing (or vice versa); `_intake.md` (but a pre-step there would leak to `fab-draft`/`fab-dedupe` or require a second knob, contradicting `_intake`'s single-fork design); `_preamble.md` (always-loaded by both, but expands the always-load surface for ALL skills with routing-only content). Deferred — promptless dispatch.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Anti-triggers go in the frontmatter `description:` of exactly four skills — `fab-proceed`, `fab-fff`, `fab-ff`, `fab-new` (canonical `src/kit/skills/*.md`); `fab-draft`/`fab-dedupe` excluded | Discussed — explicit enumerated list; descriptions are the one kit-controlled surface read at routing time | S:90 R:85 A:90 D:90 |
| 2 | Certain | No third pipeline lane ("nano-flow"); no-flow means fab is not invoked at all | Discussed — explicitly REJECTED: any lane still creates folder/branch/status file, the exact overhead complained about | S:95 R:80 A:90 D:95 |
| 3 | Certain | Out of scope: `change-types.md` micro-tier taxonomy entry and `/fab-setup` host-CLAUDE.md right-sizing rule | Discussed — both explicitly deferred to later changes | S:95 R:90 A:90 D:95 |
| 4 | Certain | Micro criteria = no Affected Memory impact + no behavior-contract change + ~1-task/single-spot edit; tie-breaker default-closed ("when unsure, use fab"); wording minimal and IDENTICAL everywhere | Discussed — criteria and tie-breaker fixed verbatim; loophole risk named (memory drift is the expensive failure) | S:90 R:80 A:85 D:90 |
| 5 | Certain | `/fab-proceed` backstop stops with a gate-failure-style message: no interactive prompt, does not perform the fix itself | Discussed — hard requirement on the zero-prompt posture | S:90 R:80 A:90 D:90 |
| 6 | Confident | Go-ahead mechanic for `/fab-proceed`: explicit user instruction in conversation (e.g. "use fab anyway") detected on re-invocation; no new flag/argument | Skill's documented no-args/no-flags contract + conversation-inference design make this the obvious fit; a flag would change a published contract | S:45 R:70 A:70 D:60 |
| 7 | Confident | Backstop triggers only on the create-new path: `/fab-new` before Steps 0–9; `/fab-proceed` only on `_intake` dispatch-table rows; resume/active-change paths never trigger it | "Intake entry" from discussion + case-1 (amend in flight) is handled by description text, not the backstop; matches the EXTRACTION BOUNDARY | S:60 R:75 A:80 D:70 |
| 8 | Confident | Frontmatter descriptions carry a compressed self-contained criteria form; skill-body backstop text has a single owner + pointers (owner-or-pointer, code-quality.md) | Discussed — descriptions are read without loading files, so pointers cannot work there; body text follows the project's owner-or-pointer convention | S:70 R:75 A:85 D:75 |
| 9 | Confident | Exact anti-trigger sentence and backstop stop-message wording finalized at apply, staying close to the discussed candidate text ("Not for micro fixes — … Not for follow-up tweaks … — amend it instead." / "This looks like a direct fix — handle it without fab unless you want a tracked change.") | Discussed "along the lines of" — authoring latitude within a fixed shape; easily revised | S:65 R:85 A:75 D:60 |
| 10 | Unresolved | Canonical owner file for the shared skill-body micro-criteria/backstop text (`fab-proceed.md` vs `fab-new.md` vs `_intake.md` vs `_preamble.md`) | Deferred — promptless dispatch. Discussion said "consider the owner-or-pointer rule" without picking the owner; each candidate has a real structural cost (see Open Questions). Front-runner: one of the two consumer skill bodies owns, the other points | S:45 R:55 A:50 D:30 |
| 11 | Confident | `/fab-new`'s backstop is posture-matched: it MAY confirm inline (its interactive posture makes an inline "continue anyway?" a valid explicit go-ahead), unlike `/fab-proceed`'s stop-only form; criteria wording identical in both | Discussion pinned stop-not-prompt to `/fab-proceed`'s posture specifically; each skill keeping its own posture follows existing contracts | S:55 R:75 A:75 D:60 |
| 12 | Certain | Sweep obligations: `docs/specs/skills.md` updated for the `/fab-proceed`/`/fab-new` flow change; `fab-ff` ↔ `fab-fff` twin-class swept together; repo-wide grep for changed phrases | Project docs mandate (constitution + code-quality.md § Sibling Sweeps + known constraints in the request) | S:90 R:85 A:95 D:90 |

12 assumptions (6 certain, 5 confident, 0 tentative, 1 unresolved).
