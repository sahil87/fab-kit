# Plan: Slim _cli-external.md to Fab-Owned Content

**Change**: 260718-clix-slim-external-md
**Intake**: `intake.md`

## Requirements

Markdown-only refactor: replace the tool-owned per-tool gists in `_cli-external.md` with use-time `<tool> skill` delegation (mirroring the existing `help-dump` delegation), retaining only fab-owned content, and align every cross-reference in the mirror/sweep class. No Go code, no templates, no migrations, no tests.

### _cli-external: Reference Model (the delegation contract)

#### R1: `<tool> skill` delegation instruction
The § Reference Model SHALL carry a new use-time delegation instruction — a sibling of the existing `help-dump` paragraph — directing agents to run `<tool> skill` for a tool's usage knowledge beyond the retained fab-owned content: **bare** for `wt`/`idea` (assumed-present class), **`command -v`-gated fail-silent** for `rk`/`hop` (genuinely-optional class).

- **GIVEN** an operator loading `_cli-external.md` needing a tool's usage knowledge not in the retained fab-owned content
- **WHEN** it reads § Reference Model
- **THEN** it finds an instruction to run `wt skill` / `idea skill` bare and `command -v rk >/dev/null 2>&1 && rk skill` / `command -v hop >/dev/null 2>&1 && hop skill` gated
- **AND** the instruction states the delegation is scoped to the four owned binaries (`tmux`/`—/loop` excluded, exactly like the `help-dump` scope note)

#### R2: version-skew fallback
The delegation instruction SHALL require a capability-probe of `<tool> skill` (failing = non-zero exit or no output) with a **silent** fallback to the shll.ai bundle-page pointer `https://shll.ai/<tool>/skill`; operator context loading MUST NOT break or surface an error on an older binary predating its `skill` subcommand. For `rk`/`hop` the probe composes with the existing `command -v` gate (absent binary → skip entirely; present-but-old → fallback pointer).

- **GIVEN** an installed tool whose binary predates its `skill` subcommand
- **WHEN** the delegation instruction runs `<tool> skill`
- **THEN** the failure is caught silently and the instruction falls back to the `https://shll.ai/<tool>/skill` pointer
- **AND** no error or warning is surfaced to the operator (fail-silent, matching the file's absent-binary discipline)

#### R3: absent-binary discipline retained and coupled to `skill`
The § Reference Model absent-binary discipline (two install classes: `wt`/`idea` assumed-present → bare; `rk`/`hop` genuinely-optional → `command -v`-gated fail-silent) SHALL be retained and MUST govern the new `skill` delegation exactly as it governs `help-dump` (both are per-tool invocations subject to the same gating).

- **GIVEN** § Reference Model documents two install classes
- **WHEN** the `skill` delegation is added
- **THEN** the two-class discipline covers `<tool> skill` invocations identically to `<tool> help-dump`
- **AND** the discipline is not generalized to gate `wt`/`idea`

### _cli-external: Moved-out tool-owned gists

#### R4: wt gist delegated (choreography retained)
The `wt` section SHALL remove the tool-owned Commands table, the `wt create` Flags table, and the generic probe-and-route recipe *examples*, replacing them with a `wt skill` delegation pointer. It SHALL retain all fab-owned operator spawning choreography: run `wt create` in the TARGET repo's directory, `fab agent --print --repo <target-repo>` (never the operator's own config.yaml), tmux `new-window` with `$SPAWN_CMD`, the Operator Spawning Rules (known-change vs backlog-respawn routing) **including** the `/fab-new` Step 11 disposable-branch rename semantics and the do-NOT-send-`/git-branch` rule. The routing rule necessarily keeps stating *which wt form to use when* (existing branch → `--checkout <change-folder-name>`; missing → positional) because that decision is fab's.

- **GIVEN** the slimmed `wt` section
- **WHEN** an operator reads it
- **THEN** the tool-owned Commands/Flags tables and generic recipe examples are gone, replaced by a `wt skill` delegation
- **AND** every fab-owned spawning-choreography element (target-repo cwd, `fab agent --print --repo`, tmux new-window, Operator Spawning Rules with Step 11 semantics and the no-`/git-branch` rule, and the `--checkout`-vs-positional routing decision) remains

#### R5: idea gist delegated
The `idea` section SHALL remove the verb table, persistent-flags table, query-matching prose, backlog-format block, and output-formats block, replacing them with an `idea skill` delegation pointer (bare, assumed-present). The one-line "what it is" identity prose MAY remain.

- **GIVEN** the slimmed `idea` section
- **WHEN** an operator reads it
- **THEN** the tool-owned verb/flags/format content is gone, replaced by an `idea skill` delegation

#### R6: hop gist delegated
The `hop` section SHALL remove the discovery gist (the `ls`/`ls --trees`/`where` table and the discovery-subset prose), replacing it with a `command -v hop`-gated fail-silent `hop skill` delegation. The genuinely-optional identity and the `command -v` gate discipline MAY remain in condensed form.

- **GIVEN** the slimmed `hop` section
- **WHEN** an operator reads it
- **THEN** the tool-owned discovery table/prose is gone, replaced by a gated `hop skill` delegation, and the fail-silent optionality is preserved

#### R7: rk gist delegated (escalation usage retained)
The `rk` section SHALL remove the tool-owned `rk notify` *contract* (usage line, fail-silent-by-contract bullet, delivery model) and the static `rk context` pointers (server-URL discovery snippet, iframe `@rk_type`/`@rk_url` recipes, the `/proxy/{port}/` pattern, the Visual Display Recipe pointer + visual-explainer integration), replacing them with a `command -v rk`-gated fail-silent `rk skill` delegation. It SHALL retain the **operator's escalation usage of `rk notify`** — the gated send with the operator's `{change}: {summary} ({repo})` / `Operator: strategic question` message/title template — because that is fab-specific usage, not the tool contract.

- **GIVEN** the slimmed `rk` section
- **WHEN** an operator reads it
- **THEN** the tool-owned notify contract + static context/iframe/proxy/visual pointers are gone, replaced by an `rk skill` delegation
- **AND** the fab-owned operator escalation-send (gated `rk notify` with the operator's message/title template) remains

#### R8: tmux and /loop unchanged
The `tmux` section (third-party, no `skill` bundle ever; `fab pane` internalization notes are fab-owned) and the `/loop` section (a Claude Code skill, not a binary) SHALL remain unchanged.

- **GIVEN** the slim
- **WHEN** the file is edited
- **THEN** the `tmux` and `/loop` sections are byte-unchanged, and the delegation scope-note continues to exclude them

### Frontmatter and cross-reference alignment

#### R9: `_cli-external.md` frontmatter description updated
The `_cli-external.md` frontmatter `description:` SHALL be updated to reflect the slimmed reality (fab-owned content + `<tool> skill` delegation), replacing the "hand-authored gist per tool" framing.

- **GIVEN** the slimmed file
- **WHEN** the frontmatter is read
- **THEN** the `description:` describes the fab-owned-plus-`skill`-delegation model, not the pre-slim per-tool gist model

#### R10: SPEC-_cli-external.md mirror aligned
`docs/specs/skills/SPEC-_cli-external.md` (the Summary and the per-section Command Inventory table) SHALL be rewritten to the slimmed reality: the Summary's "hand-authored gist per tool" framing and each row that describes moved-out gist content (notably the `rk` row's "full body the `_preamble.md` § Run-Kit pointer forwards to") updated to describe fab-owned content + `<tool> skill` use-time delegation.

- **GIVEN** the slimmed `_cli-external.md`
- **WHEN** its SPEC mirror is read
- **THEN** the Summary and Command Inventory describe the delegation model, and no row claims a tool-owned gist body still lives in the file

#### R11: `_preamble.md` § Run-Kit pointer + SPEC mirror aligned
`src/kit/skills/_preamble.md` § Run-Kit (rk) Reference SHALL update its "Command Reference (full body in `_cli-external.md`)" pointer: after the slim, `_cli-external.md` § rk carries the fab-owned escalation usage + an `rk skill` delegation, not the full command bodies. The pointer text MUST be corrected to stay accurate (cross-reference rule). `docs/specs/skills/SPEC-_preamble.md` SHALL mirror the corrected pointer.

- **GIVEN** the slimmed `_cli-external.md` § rk
- **WHEN** `_preamble.md` § Run-Kit is read
- **THEN** its pointer no longer claims the "full `rk` command reference" body lives in `_cli-external.md`, and SPEC-_preamble.md mirrors the change

#### R12: `fab-operator.md` references re-verified + SPEC mirror aligned
`src/kit/skills/fab-operator.md`'s references to `_cli-external.md § wt` (probe-and-route routing) SHALL be re-verified — they remain accurate because the fab-owned routing choreography stays. Its reference to `_cli-external.md § rk` framed as the "full command reference" (line ~366) MUST be re-pointed to describe the retained fab-owned notify usage + `rk skill` delegation, not a full command body. `docs/specs/skills/SPEC-fab-operator.md`'s matching references (notably § Notification Send's "full command reference in `_cli-external.md` § rk") SHALL be aligned the same way.

- **GIVEN** the slimmed `_cli-external.md`
- **WHEN** `fab-operator.md` and SPEC-fab-operator.md are read
- **THEN** every `_cli-external.md § wt` reference still resolves to retained content, and every `_cli-external.md § rk` reference no longer implies a full tool-command body lives there

#### R13: companions.md verified/aligned
`docs/specs/companions.md` SHALL be checked for restated tool-owned gist content and aligned with the delegation model if any is found. (It documents the wt/idea *pipeline integration* — `fab batch new`/`switch`, backlog feeding `/fab-new` — and defers full command surface to each tool's README; it restates no operator-gist content, so no edit is expected.)

- **GIVEN** companions.md
- **WHEN** it is checked against the slim
- **THEN** it either needs no edit (it defers command surface to tool READMEs) or is aligned to the delegation model

#### R14: aggregate-spec sweep
The repo-wide grep for moved-out phrases (`help-dump`, `probe-and-route`, `` `_cli-external.md` § wt ``/`§ rk`, `rk notify`, hand-authored gist) across the sweep class — skill sources under `src/kit/skills/`, SPEC mirrors under `docs/specs/skills/`, and aggregate specs (`docs/specs/skills.md`, `glossary.md`, `architecture.md`, `companions.md`) — SHALL be run, and every occurrence that restates moved-out per-tool facts updated. Occurrences that merely name `_cli-external` as a helper (helper-lists, deployment inventories) are not moved-out facts and are left unchanged.

- **GIVEN** the full sweep class
- **WHEN** the moved-out phrases are grepped repo-wide
- **THEN** every occurrence restating a moved-out per-tool fact is updated, and helper-name-only mentions are untouched

### Non-Goals

- No change to the two install classes or which tools belong to each (R3 retains them verbatim).
- No change to `fab-operator.md`'s spawn-procedure *semantics* (§6) — only reference accuracy (R12).
- No removal of the `help-dump` delegation — it remains for the exhaustive command tree; `skill` covers usage knowledge (siblings per the standard).
- No shipping of any `skill` subcommand (producer side, already done tool-side per shll [agst]).
- No edit to `docs/memory/` files — that is hydrate's job (a later stage).
- No edit under `.claude/skills/` (gitignored deployed copies) — canonical sources live in `src/kit/skills/` only.

### Design Decisions

1. **Version-skew fallback = silent shll.ai pointer, not retained gists**: the fallback for an old binary points to `https://shll.ai/<tool>/skill` rather than re-inlining a gist — *Why*: retaining gists would defeat the slim, and the retained fab-owned choreography already carries the operator-critical wt semantics; all four installed binaries already serve bundles, so the fallback is a degraded-mode edge — *Rejected*: keeping a retained gist per tool (backlog allowed either form).
2. **`skills.md` line 126 SPEC-exclusion inconsistency left untouched**: `docs/specs/skills.md` still states `_cli-external.md` "carr[ies] no SPEC", yet `SPEC-_cli-external.md` exists (a 260620 backfill) and the intake requires updating it — *Why*: this is a pre-existing inconsistency this change neither introduces nor is scoped to fix; `skills.md` restates no moved-out per-tool gist content, so it is outside the sweep class for *this* change — *Rejected*: fixing the stale exclusion note here (scope creep beyond the intake).

## Tasks

### Phase 1: Primary slim

- [x] T001 Update `src/kit/skills/_cli-external.md` frontmatter `description:` to the slimmed model (fab-owned content + use-time `<tool> skill` delegation), dropping the "hand-authored gist per tool" framing. <!-- R9 -->
- [x] T002 In `src/kit/skills/_cli-external.md` § Reference Model, add the `<tool> skill` delegation instruction as a sibling of the `help-dump` paragraph (bare `wt`/`idea`; `command -v`-gated fail-silent `rk`/`hop`), with the four-owned-binaries scope note; and add the required version-skew fallback (probe `<tool> skill`, silent fallback to `https://shll.ai/<tool>/skill`, composing with the `command -v` gate for `rk`/`hop`). Retain the absent-binary discipline and couple it to `skill`. <!-- R1 --> <!-- R2 --> <!-- R3 -->
- [x] T003 In `src/kit/skills/_cli-external.md` § wt, remove the Commands table, the `wt create` Flags table, and the generic probe-and-route recipe *examples*; add a `wt skill` delegation pointer. Retain ALL fab-owned choreography: the Repo-targeted spawning note, the Operator Spawning Rules (known-change probe-and-route with `--checkout`-vs-positional decision + `/fab-new` Step 11 rename semantics + no-`/git-branch` rule), `fab agent --print --repo`, tmux new-window with `$SPAWN_CMD`. <!-- R4 -->
- [x] T004 In `src/kit/skills/_cli-external.md` § idea, remove the verb table, persistent-flags table, query-matching prose, backlog-format block, and output-formats block; add an `idea skill` delegation pointer (bare). Keep the one-line identity prose. <!-- R5 -->
- [x] T005 In `src/kit/skills/_cli-external.md` § hop, remove the discovery gist (`ls`/`ls --trees`/`where` table + discovery-subset prose); add a `command -v hop`-gated fail-silent `hop skill` delegation. Keep the condensed genuinely-optional identity + `command -v` gate discipline. <!-- R6 -->
- [x] T006 In `src/kit/skills/_cli-external.md` § rk (run-kit), remove the tool-owned `rk notify` contract (usage line, fail-silent-by-contract bullet, delivery model) and the static `rk context` pointers (server-URL discovery, iframe `@rk_type`/`@rk_url`, `/proxy/{port}/`, Visual Display Recipe + visual-explainer integration); add a `command -v rk`-gated fail-silent `rk skill` delegation. RETAIN the fab-owned operator escalation-send (gated `rk notify` with the `{change}: {summary} ({repo})` / `Operator: strategic question` template). <!-- R7 -->
- [x] T007 Verify `src/kit/skills/_cli-external.md` § tmux and § /loop are byte-unchanged and the Reference Model scope-note still excludes them. <!-- R8 -->

### Phase 2: SPEC mirrors + cross-reference alignment

- [x] T008 [P] Rewrite `docs/specs/skills/SPEC-_cli-external.md` — the Summary "hand-authored gist per tool" framing and the Command Inventory rows describing moved-out gist content (esp. the `rk` row's "full body the `_preamble.md` § Run-Kit pointer forwards to" and the Reference Model row's `help-dump`-only framing) to the fab-owned-plus-`skill`-delegation model. <!-- R10 -->
- [x] T009 [P] Update `src/kit/skills/_preamble.md` § Run-Kit (rk) Reference "Command Reference (full body in `_cli-external.md`)" pointer so it no longer claims the full `rk` command body lives in `_cli-external.md § rk` (it now carries fab-owned escalation usage + `rk skill` delegation). <!-- R11 -->
- [x] T010 [P] Mirror the R11 pointer correction into `docs/specs/skills/SPEC-_preamble.md` (§ Run-Kit row of the Subsection Inventory + the Flow block's Run-Kit node). <!-- R11 -->
- [x] T011 [P] Re-verify + re-point `src/kit/skills/fab-operator.md`: confirm the `_cli-external.md § wt` probe-and-route references (lines ~110, ~436, ~539) still resolve to retained content; re-point the § rk "full command reference" reference (line ~366) to the retained fab-owned notify usage + `rk skill` delegation. <!-- R12 -->
- [x] T012 [P] Align `docs/specs/skills/SPEC-fab-operator.md`: the § Notification Send "full command reference in `_cli-external.md` § rk (run-kit)" phrasing (line ~118) and the § wt primitive-table row (line ~54) to match the slimmed `_cli-external.md`. <!-- R12 -->
- [x] T013 [P] Check `docs/specs/companions.md` for restated operator-gist content; align to the delegation model only if found (expected: no edit — it defers command surface to tool READMEs and documents pipeline integration, not operator gists). <!-- R13 -->

### Phase 3: Sweep verification

- [x] T014 Run the repo-wide grep for the moved-out phrases (`help-dump`, `probe-and-route`, `` `_cli-external.md` § wt ``/`§ rk`, `rk notify`, hand-authored gist) across `src/kit/skills/`, `docs/specs/skills/`, and aggregate specs (`docs/specs/skills.md`, `glossary.md`, `architecture.md`, `companions.md`); confirm every moved-out-fact occurrence is updated and helper-name-only mentions are left untouched. <!-- R14 -->

## Execution Order

- T002 depends on T001 (both in the same file; frontmatter then body — do sequentially to avoid edit-context churn).
- T003–T007 are section-local edits to `_cli-external.md`; run after T002 (they reference the § Reference Model contract T002 establishes). Sequential within the file to keep a coherent edit context.
- Phase 2 (T008–T013) depends on Phase 1 being complete (the mirrors must reflect the final slimmed source). T008–T013 touch distinct files and are `[P]`.
- T014 (sweep verification) runs last, after all edits, and gates completion.

## Acceptance

### Functional Completeness

- [x] A-001 R1: § Reference Model carries a `<tool> skill` delegation instruction — bare `wt skill`/`idea skill`, `command -v`-gated `rk skill`/`hop skill` — scoped to the four owned binaries.
- [x] A-002 R2: The delegation instruction requires a `<tool> skill` capability-probe with a silent fallback to `https://shll.ai/<tool>/skill`, composing with the `command -v` gate for `rk`/`hop`; no error surfaces on an old binary.
- [x] A-003 R3: The two-install-class absent-binary discipline is retained and governs the `skill` delegation exactly as it governs `help-dump`; it is not generalized to `wt`/`idea`.
- [x] A-004 R4: § wt's tool-owned Commands/Flags tables and generic recipe examples are gone (replaced by `wt skill`), and all fab-owned spawning choreography (target-repo cwd, `fab agent --print --repo`, tmux new-window, Operator Spawning Rules incl. Step 11 + no-`/git-branch` + `--checkout`-vs-positional routing) remains.
- [x] A-005 R5: § idea's verb/flags/query/backlog-format/output-format content is gone (replaced by `idea skill`).
- [x] A-006 R6: § hop's discovery gist is gone (replaced by a `command -v hop`-gated `hop skill`), fail-silent optionality preserved.
- [x] A-007 R7: § rk's tool-owned notify contract + static context/iframe/proxy/visual pointers are gone (replaced by `rk skill`), and the fab-owned operator escalation-send (gated `rk notify` with the operator's message/title template) remains.
- [x] A-008 R9: `_cli-external.md` frontmatter `description:` describes the slimmed model, not the per-tool-gist model.
- [x] A-009 R10: SPEC-_cli-external.md Summary + Command Inventory describe the delegation model; no row claims a tool-owned gist body still lives in the file.
- [x] A-010 R11: `_preamble.md` § Run-Kit pointer no longer claims the full `rk` command body lives in `_cli-external.md`, and SPEC-_preamble.md mirrors it.
- [x] A-011 R12: Every `fab-operator.md`/SPEC-fab-operator.md `_cli-external.md § wt` reference still resolves to retained content, and every `§ rk` reference no longer implies a full tool-command body lives there.

### Behavioral Correctness

- [x] A-012 R8: § tmux and § /loop are unchanged, and the Reference Model scope-note still excludes them.
- [x] A-013 R7: The retained operator `rk notify` send is the fab-specific escalation usage (message/title template), distinct from the removed tool contract — an operator reading the slimmed file can still perform the escalation send without the deleted contract prose.

### Scenario Coverage

- [x] A-014 R2: An older binary predating `<tool> skill` triggers the silent shll.ai-pointer fallback per the instruction, without breaking operator context loading (verified by reading the instruction's stated behavior).

### Edge Cases & Error Handling

- [x] A-015 R6: An absent `hop`/`rk` binary is skipped entirely by the `command -v` gate (no `command not found`); a present-but-old one hits the fallback pointer — both paths documented.

### Removal Verification

- [x] A-016 R14: The repo-wide sweep confirms no stale occurrence restates a moved-out per-tool fact (help-dump-still-authoritative-for-full-surface phrasing may remain where it correctly describes `help-dump`), and helper-name-only `_cli-external` mentions (skills.md, architecture.md deployment inventory) are untouched.

### Code Quality

- [x] A-017 Pattern consistency: The new `skill` delegation prose mirrors the existing `help-dump` delegation wording/structure (same § Reference Model, same two install classes, same fail-silent discipline) — no new convention invented.
- [x] A-018 No unnecessary duplication: The delegation instruction is stated once in § Reference Model; per-tool sections point to it rather than restating the gate/fallback mechanics.
- [x] A-019 Canonical source only: All skill edits are under `src/kit/skills/` — no `.claude/skills/` edit.
- [x] A-020 SPEC-mirror sync: Every edited `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in this same change (`_cli-external`, `_preamble`, `fab-operator`).

### Documentation Accuracy (checklist.extra_categories)

- [x] A-021 Every cross-reference to `_cli-external.md § wt`/`§ rk` in the sweep class points at content that actually still lives there after the slim (no dangling "full command reference" claims).

### Cross References (checklist.extra_categories)

- [x] A-022 The `_preamble.md` ⇄ `_cli-external.md` rk pointer pair and the `fab-operator.md` ⇄ `_cli-external.md` wt/rk reference pairs are mutually consistent after the slim (both sides describe the same retained content).

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Markdown-only change: no tests to run (no `.go` files touched). Verification is the R14 grep sweep + reading the edited prose for accuracy.

## Deletion Candidates

<!-- Recorded by the review stage (change_type=refactor) — out-of-scope cleanup candidates, NOT rework for this change. -->

- `docs/specs/skills.md:126` — the "Exclusion policy: … `_cli-fab.md` and `_cli-external.md` carry no SPEC" note is falsified by the 260620 backfill and further entrenched by this change's SPEC-_cli-external update; stale-note removal candidate (plan Design Decision 2 deliberately deferred it).
- `--worktree-name` occurrences repo-wide (`_cli-external.md`, `fab-operator.md`, `_preamble.md` § Naming Conventions, `_cli-fab.md` § fab batch, and the Go batch spawner) — the installed `wt` deprecates it in favor of `--name` (still functional); whole-class sweep incl. Go code, beyond this intake.
- `src/kit/skills/fab-operator.md:57` — helper gloss "`_cli-external` (wt, idea, tmux, /loop reference)" omits hop/rk and predates the fab-owned-choreography framing; one-line refresh candidate.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope split (moves-out vs stays) taken verbatim from intake Assumption 1 / backlog [clix]: wt create-contract/flags/recipe-examples, idea verbs/flags/format, hop discovery, rk notify-contract + static context/iframe/proxy pointers move out; absent-binary discipline, all operator spawning choreography (incl. Step 11 + no-`/git-branch`), escalation rk-notify usage, tmux, /loop stay | Enumerated verbatim in the intake; no interpretation | S:95 R:70 A:90 D:95 |
| 2 | Certain | Delegation form = use-time `<tool> skill`, bare for wt/idea, `command -v`-gated fail-silent for rk/hop, mirroring the existing help-dump delegation | Specified in intake Assumption 2; the mirrored pattern already exists in § Reference Model; all four bundles verified shipping this session (`wt/idea/hop/rk skill` each exit 0) | S:95 R:80 A:95 D:95 |
| 3 | Confident | Version-skew fallback = silent pointer to `https://shll.ai/<tool>/skill`, not retained gists | Intake Assumption 4 (backlog allowed either form); retaining gists defeats the slim, retained choreography already carries operator-critical wt semantics, all four binaries already serve bundles so fallback is a degraded-mode edge | S:70 R:85 A:80 D:60 |
| 4 | Confident | Sweep class = SPEC-_cli-external, _preamble § Run-Kit pointer + SPEC-_preamble, fab-operator § wt/§ rk refs + SPEC-fab-operator, companions.md; aggregate specs (skills.md/glossary.md/architecture.md) grepped but expected untouched | Grep confirmed skills.md/architecture.md mention `_cli-external` only as a helper-name / deployment-inventory entry (not moved-out gist facts); glossary.md has zero matches; per code-quality.md § Sibling & Mirror Sweeps | S:85 R:75 A:90 D:85 |
| 5 | Confident | companions.md needs no edit — it documents wt/idea *pipeline integration* (fab batch new/switch, backlog→/fab-new) and defers full command surface to each tool's README; it restates no operator-gist content | Read in full this session; its wt/idea sections are integration prose + "Full command reference … live in [repo]" pointers, none of which move out | S:80 R:80 A:85 D:80 |
| 6 | Confident | `docs/specs/skills.md` line 126's stale "_cli-external.md carries no SPEC" exclusion note is left untouched | Pre-existing inconsistency (SPEC-_cli-external.md exists as a 260620 backfill, and the intake requires updating it); skills.md restates no moved-out per-tool fact, so it is outside this change's sweep class; fixing the note would be scope creep | S:70 R:85 A:85 D:80 |
| 7 | Certain | change_type = refactor (content restructuring/delegation, no new fab capability) | Confirmed by `.status.yaml` `change_type: refactor` (explicit) — matches intake Assumption 6 | S:90 R:85 A:95 D:90 |

7 assumptions (3 certain, 4 confident, 0 tentative).
