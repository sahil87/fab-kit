# Plan: Reflect the run-kit rk→run-kit Formula Rename

**Change**: 260709-j0n1-run-kit-formula-rename
**Intake**: `intake.md`

## Requirements

<!-- docs-type change: the intake's What Changes section specifies the exact edits.
     Requirements below restate those edits as verifiable outcomes. -->

### Docs Accuracy: rk install-identity in `_cli-external.md`

#### R1: Reference Model parenthetical names the current formula
The § Reference Model bullet describing `rk` SHALL state the current formula identity — `rk` is run-kit, formula `sahil87/tap/run-kit` since run-kit v3.0.0, with `rk` kept as a symlink alias — while leaving the `command -v`-gate / fail-silent rule of that bullet untouched.

- **GIVEN** `src/kit/skills/_cli-external.md` § Reference Model → Absent-binary discipline, the "Genuinely-optional — `rk`, `hop`" bullet
- **WHEN** an agent reads the `rk` parenthetical for install identity
- **THEN** it reads `` `rk` is run-kit — formula `sahil87/tap/run-kit` since run-kit v3.0.0, with `rk` kept as a symlink alias``
- **AND** the sentence "**Every `rk`/`hop` invocation … MUST be `command -v`-gated and fail silently**" and the "Do NOT generalize this gate to `wt`/`idea`" clause are unchanged

#### R2: rk (run-kit) section intro pins the alias status and why `rk` is still used
The § rk (run-kit) opening paragraph SHALL carry one added sentence, immediately after the existing opening sentence, stating that since run-kit v3.0.0 the Homebrew formula and primary binary are named `run-kit` (`sahil87/tap/run-kit`), `rk` is kept as a symlink alias, and `rk` remains the invocation form used throughout fab skills.

- **GIVEN** `src/kit/skills/_cli-external.md` § rk (run-kit), first paragraph
- **WHEN** an agent reads the section intro
- **THEN** the added sentence follows "run-kit is the tmux session manager with a web UI that hosts the operator's session." and reads: "Since run-kit v3.0.0 the Homebrew formula and primary binary are named `run-kit` (`sahil87/tap/run-kit`); `rk` is kept as a symlink alias and remains the invocation form used throughout fab skills."
- **AND** it makes explicit that fab skills keep writing `rk` deliberately (not stale)

### Docs Accuracy: SPEC mirror (constitution-mandated)

#### R3: SPEC-_cli-external.md mirrors the rename fact
Per the constitution ("Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file"), `docs/specs/skills/SPEC-_cli-external.md` SHALL gain a minimal factual addition on the `rk` row of its § Command Inventory table reflecting the formula rename (formula `sahil87/tap/run-kit` since run-kit v3.0.0, `rk` kept as an alias / invocation form).

- **GIVEN** `docs/specs/skills/SPEC-_cli-external.md` § Command Inventory, the `rk (run-kit)` row
- **WHEN** the skill-file edit ships
- **THEN** the SPEC's `rk (run-kit)` row carries the same one-line rename fact
- **AND** the addition is minimal (no restructuring), keeping the SPEC a faithful mirror of the partial

### Non-Goals

- **`rk` command invocations everywhere** (`command -v rk`, `rk notify`, `rk context`, `rk agent-setup`, `rk help-dump` in `_preamble.md`, `_cli-external.md`, `fab-operator.md`, SPECs, memory) — the alias is kept deliberately; invocations stay `rk`, NOT rewritten to `run-kit`.
- **`src/kit/skills/_preamble.md` § Run-Kit (rk) Reference** (and its `SPEC-_preamble.md` mirror) — carries only the detection/fail-silent rule + pointer, no formula-identity claim. No edit.
- **`@rk_agent_state` pane option** and all `rk agent-setup` references (Go sources, pane tests, `_cli-fab.md`, `runtime/runtime-agents.md`, migration `2.13.6-to-2.14.0.md`) — run-kit-owned data convention; rename covers formula/binary only. Unchanged.
- **Historical references** — `rk v2.3.2` attributions (`SPEC-fab-operator.md`, `docs/memory/runtime/operator.md`), archived change artifacts (`fab/changes/archive/**`), memory `log.md`/`log.seed.md` entries. Historical facts stay verbatim.
- **Formula-path references** — zero hits for `sahil87/tap/rk`, `Formula/rk.rb`, `brew install rk` anywhere in fab-kit. Nothing to update.
- **Config & fixtures** — `fab/project/config.yaml` and Go testdata contain no rk formula references. Nothing to update.
- **`.claude/skills/` deployed copies** — gitignored; refresh via `fab sync` on release. Canonical source is `src/kit/skills/`.

### Design Decisions

1. **Surgical docs-accuracy edit, not a rename sweep**: An intake-time repo-wide sweep proved fab-kit never targets the formula by name (zero `sahil87/tap/rk` / `brew install rk` / `Formula/rk.rb` hits). — *Why*: the only stale claim is the install-identity parenthetical in `_cli-external.md`; a mass edit would be wrong. — *Rejected*: rewriting `rk` invocations to `run-kit` (the alias is deliberately kept; invocations are correct as-is).

## Tasks

### Phase 1: Skill edits

- [x] T001 Edit `src/kit/skills/_cli-external.md` § Reference Model → Absent-binary discipline: change the `rk` parenthetical in the "Genuinely-optional — `rk`, `hop`" bullet from "(`rk` is run-kit; `hop` is the multi-repo navigator)" to "(`rk` is run-kit — formula `sahil87/tap/run-kit` since run-kit v3.0.0, with `rk` kept as a symlink alias; `hop` is the multi-repo navigator)", leaving the gate rule untouched <!-- R1 -->
- [x] T002 Edit `src/kit/skills/_cli-external.md` § rk (run-kit): add one sentence after the existing opening sentence — "Since run-kit v3.0.0 the Homebrew formula and primary binary are named `run-kit` (`sahil87/tap/run-kit`); `rk` is kept as a symlink alias and remains the invocation form used throughout fab skills." <!-- R2 -->

### Phase 2: SPEC mirror

- [x] T003 Edit `docs/specs/skills/SPEC-_cli-external.md` § Command Inventory: add the minimal rename fact to the `rk (run-kit)` row (formula `sahil87/tap/run-kit` since run-kit v3.0.0, `rk` kept as an alias/invocation form) <!-- R3 -->

### Phase 3: Verification

- [x] T004 Grep the affected claims repo-wide to confirm the mirror class is fully covered and no other live install-identity claim was missed (respecting the intake's non-goals: historical rows and archived changes stay verbatim; `.claude/skills/` deployed copies excluded) <!-- R1 R2 R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/_cli-external.md` § Reference Model `rk` parenthetical names formula `sahil87/tap/run-kit`, run-kit v3.0.0, and the `rk` binary-alias status; the gate/fail-silent rule is unchanged
- [x] A-002 R2: `src/kit/skills/_cli-external.md` § rk (run-kit) intro carries the added rename/alias sentence immediately after the existing opening sentence
- [x] A-003 R3: `docs/specs/skills/SPEC-_cli-external.md` § Command Inventory `rk (run-kit)` row carries the matching one-line rename fact

### Behavioral Correctness

- [x] A-004 R2: The added sentence pins that fab skills keep writing `rk` deliberately (alias kept), so a reader does not mistake the retained `rk` invocations for stale usage

### Scenario Coverage

- [x] A-005 R1: An agent reading `_cli-external.md` for run-kit install guidance now names the formula `sahil87/tap/run-kit` (not `rk`)

### Documentation Accuracy

- [x] A-006 No `rk` invocation was rewritten to `run-kit`; `@rk_agent_state`, historical `rk v2.3.2` attributions, and archived changes are unchanged (intake non-goals honored)
- [x] A-007 The skill edit and its SPEC mirror stay in sync (constitution SPEC-mirror rule); no other live install-identity claim for `rk` exists elsewhere in the repo

### Cross-References

- [x] A-008 `_preamble.md` § Run-Kit and its `SPEC-_preamble.md` mirror remain unedited (they carry only the detection rule + pointer, no formula-identity claim), so the pointer to `_cli-external.md` § rk stays accurate

### Code Quality

- [x] A-009 Pattern consistency: edits match the surrounding prose style/wrapping of `_cli-external.md` and the table-row style of `SPEC-_cli-external.md`
- [x] A-010 No unnecessary duplication: the rename fact is added only at the single live claim site + its SPEC mirror, not duplicated into `_preamble.md`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Markdown-only change — no test surface; verification is by grepping the edited claims.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Blast radius is exactly `_cli-external.md` (2 edits) + `SPEC-_cli-external.md` (1 mirror) — no other live install-identity claim or formula-path reference exists | Confirmed by re-running the intake's sweep at apply: zero `tap/rk`/`Formula/rk`/`brew install rk` hits outside the intake's own quotes; the "sibling formula" hits in memory are about `wt`/`idea` and the ones in `_cli-external.md`/archives are about `hop` or historical | S:90 R:90 A:95 D:90 |
| 2 | Certain | SPEC mirror addition lands on the `rk (run-kit)` row of the § Command Inventory table (minimal factual clause), not the overview prose | The intake offers "row and/or overview prose"; the row is the most targeted, minimal mirror site and keeps the SPEC's mirror-faithfulness; overview prose would restate more than needed | S:85 R:90 A:90 D:80 |
| 3 | Certain | `_preamble.md` / `SPEC-_preamble.md` and the `rk v2.3.2` historical refs in `SPEC-fab-operator.md` stay unedited | Verified those sites carry only the detection rule + pointer or a historical attribution — no formula-identity claim; editing them would duplicate the single-claim-site or falsify history (intake non-goals + 4rtx precedent) | S:85 R:95 A:90 D:90 |

3 assumptions (3 certain, 0 confident, 0 tentative).
