# Plan: docs-distill-memory Skill

**Change**: 260717-dgp8-docs-distill-memory-skill
**Intake**: `intake.md`

## Requirements

<!-- Requirements co-generated from intake.md. This change ships a new user-invocable
     skill (markdown-driven, no runtime surface) plus its constitution-required SPEC mirror,
     the aggregate/inventory surfaces, and the minimal fabhelp.go registration (intake
     Assumption #11 recommended default = include). Requirements are grouped by the two
     domains the change touches: the skill's behavior, and the New-Skill-Checklist sweep. -->

### Skill Behavior: docs-distill-memory

#### R1: New user-invocable skill file
The kit SHALL carry a new canonical skill source at `src/kit/skills/docs-distill-memory.md` (never `.claude/skills/`, which is gitignored deployed copies — Constitution V). It MUST be user-invocable with frontmatter `name: docs-distill-memory` and a behavior-naming `description` one-liner that is itself free of change-ids.

- **GIVEN** the kit source tree
- **WHEN** the change ships
- **THEN** `src/kit/skills/docs-distill-memory.md` exists with valid FKF-independent skill frontmatter (`name` matching filename, behavior-naming change-id-free `description`)
- **AND** no edit is made under `.claude/skills/`

#### R2: One-domain-per-run, propose-then-apply posture
The skill SHALL operate on exactly one memory domain per invocation (named as an argument, e.g. `/docs-distill-memory pipeline`), running read-only analysis first, emitting a per-file proposed-rewrite report (per-file diffs/summaries), and applying rewrites ONLY on explicit user approval — the same propose-then-apply idiom as `/docs-reorg-memory` (its Step 4 report → Step 5 confirm-and-apply). It MUST NOT perform autonomous bulk rewriting.

- **GIVEN** a user invokes `/docs-distill-memory <domain>`
- **WHEN** the skill runs
- **THEN** it reads the domain's topic files read-only, reports proposed per-file rewrites, and mutates no file until the user approves
- **AND** an invocation naming no domain (or an unknown domain) is handled gracefully per Error Handling

#### R3: Present-truth rewrite semantics (FKF §3.2/§3.3)
A rewrite SHALL transform each topic file to the FKF present-truth style, citing the shipped extract `$(fab kit-path)/reference/fkf.md` (§3.2/§3.3) as deployed skills do. It MUST: remove transition narration ("renamed X→Y in {id}", "this supersedes/inverts {id}", "was `old.value`", "superseding the historical …"); remove superseded-state descriptions (previous states live in the per-folder generated `log.md`, git history, and archived change folders); keep allowed provenance — trailing `(change-id)` citations and the `*Introduced by*: {change-name}` field on Design Decisions; and strip change-ids from `description:` frontmatter (§3.2 ban) while compressing over-cap descriptions to the ≤500-character routing-signal shape, moving displaced detail into the body where it is not already present.

- **GIVEN** a topic file with transition narration, superseded-state prose, an over-cap description, and/or change-ids in `description:`
- **WHEN** a rewrite is proposed and applied
- **THEN** narration and superseded-state prose are removed, the `description:` is ≤500 chars and change-id-free, and trailing `(change-id)` / `*Introduced by*` provenance is preserved
- **AND** bare 4-char ids are treated identically to dated ids — kept in trailing-citation position, removed when woven into narration (Assumption #8)

#### R4: Rationale-preservation guard (the critical constraint)
The skill SHALL preserve rationale. Token savings come from dropping narration, NEVER rationale. Deliberate-behavior / "don't re-break this" content MUST be RELOCATED into Design Decisions (`Why` / `Rejected`) as present-tense design intent (a rejected alternative is a design fact, not transition narration — FKF §3.3 verbatim), never deleted. Deletion is safe ONLY for narration whose content is already recorded elsewhere (per-folder `log.md`, git history, archived change folders); content recorded nowhere else and carrying intent is relocated, not dropped.

- **GIVEN** a topic file whose narration encodes a deliberate-behavior defense (e.g. a poll-predicate "don't simplify this") recorded nowhere else
- **WHEN** a rewrite is proposed
- **THEN** that content is relocated into a Design Decisions `Why`/`Rejected` entry as present-tense intent, not deleted
- **AND** narration whose content already lives in `log.md`/git/archives may be deleted

#### R5: Generated files never hand-edited; regen via `fab memory-index` with the refuse-before-regen guard
The skill SHALL never hand-edit the generated files `index.md` (root/domain/sub-domain tiers) and `log.md`. `log.seed.md` is a curated READ-ONLY SEED INPUT the generator reads during the seed-merge but never writes (like `description:` frontmatter) — it is not a generated file, and distillation excludes it like a ledger (the same exclusion posture as `removed-domains.md`), never rewriting it. After applying rewrites the skill MUST regenerate via `fab memory-index`, and MUST honor the refuse-before-regen convention: consult `fab memory-index --check` first and on exit 2 (destructive loss) refuse to regenerate, surfacing the existing `→ run /docs-reorg-memory to remediate …` pointer. Regeneration derives the **index tiers** from folder contents + `description:` frontmatter and each **`log.md`** from the C-lite join of git history + per-change `.status.yaml` summaries (freeze-on-write, append-only). `docs/memory/_shared/removed-domains.md` is EXEMPT from rewrite (the §3.3 tombstone carve-out — its body is a citation-carrying removal ledger).

- **GIVEN** rewrites have been applied to a domain's topic files
- **WHEN** the skill regenerates indexes
- **THEN** it runs `fab memory-index --check` first, regenerates via `fab memory-index` on exit 0/1, and refuses (surfacing the reorg pointer) on exit 2
- **AND** it never hand-edits the generated `index.md`/`log.md`, never rewrites the curated read-only `log.seed.md` seed input, and never rewrites `_shared/removed-domains.md`

#### R6: Idempotent re-runs (Constitution III)
Re-running the skill on an already-distilled domain SHALL find nothing to rewrite and report that; `fab memory-index` regeneration is byte-stable, so a no-op re-run produces no diff.

- **GIVEN** a domain already distilled by a prior run
- **WHEN** the skill is re-invoked on that domain
- **THEN** it reports no proposed rewrites and mutates no file (byte-stable)

#### R7: House skill conventions & reduced Context Loading override
The skill file SHALL follow house conventions: the standard `_preamble`-read blockquote; a `## Contents` table-of-contents (the file exceeds 100 lines); an explicit `## Context Loading` section that overrides the always-load layer (loads the memory indexes + the target domain's files + `$(fab kit-path)/reference/fkf.md`; requires NO active change, config, or constitution — the skill file wins per `_preamble` §1); no `helpers:` declaration (referencing `_cli-fab` § fab memory-index by in-body pointer, the `docs-reorg-memory` sibling style); a state-derived `Next:` line; and closing Error Handling + Key Properties tables (mirroring `docs-reorg-memory.md`, e.g. "Advances stage? No", "Requires active change? No", "Idempotent? Yes", "Indexes hand-edited? No — regenerated by `fab memory-index`").

- **GIVEN** the New Skill Checklist items 1–5 (`docs/specs/skills.md` § New Skill Checklist)
- **WHEN** the skill file is authored
- **THEN** it carries valid frontmatter, the preamble-read line, a `## Contents` TOC, an explicit `## Context Loading` override, a `Next:` line, and Error Handling + Key Properties tables

### New Skill Checklist Sweep: mirror & inventory surfaces

#### R8: Constitution-required SPEC mirror
The change SHALL create `docs/specs/skills/SPEC-docs-distill-memory.md` (Checklist item 6; Constitution Additional Constraints — every `src/kit/skills/*.md` edit requires its SPEC mirror in the same change). The filename follows the mechanical `SPEC-{source-filename}.md` policy; the content follows the `SPEC-docs-reorg-memory.md` shape (Summary + behavior sections + Flow + Tools/Sub-agents tables).

- **GIVEN** the new skill file
- **WHEN** the change ships
- **THEN** `docs/specs/skills/SPEC-docs-distill-memory.md` exists and accurately mirrors the skill's behavior

#### R9: Aggregate/inventory surface updates (the sweep class)
The change SHALL update every aggregate surface that inventories doc skills: `docs/specs/skills.md` (add the skill's own `## /docs-distill-memory` section — Checklist item 7 — no § Skill Helpers row needed, no `helpers:` declared), `docs/specs/glossary.md` (add a `## Skills` command row), and `README.md` (add a row to the `### Documentation` command table). Verified no-edit surfaces (`src/kit/skills/fab-help.md` — dynamic frontmatter scan; `docs/specs/user-flow.md` — no `docs-*` inventory) SHALL NOT be edited.

- **GIVEN** the sibling doc-skill inventory rows
- **WHEN** the change ships
- **THEN** `docs/specs/skills.md`, `docs/specs/glossary.md`, and `README.md` each carry a `docs-distill-memory` entry, and `fab-help.md` / `user-flow.md` are unchanged

#### R10: Minimal fabhelp.go registration + test (Checklist item 8)
The change SHALL add the minimal Go registration (intake Assumption #11 recommended default): a `"docs-distill-memory": "Maintenance"` entry in `skillToGroupMap` (`src/go/fab/cmd/fab/fabhelp.go`), list the new command on the hardcoded `Maintain docs:` TYPICAL FLOW line, and add `docs-distill-memory` to the `expectedMapped` list in `fabhelp_test.go` (`TestFabHelp_GroupMapping`). No command-signature change ⇒ NO `_cli-fab.md` edit. The affected Go package tests MUST pass.

- **GIVEN** the fabhelp help-grouping registry
- **WHEN** the change ships
- **THEN** `docs-distill-memory` maps to the "Maintenance" group, appears on the `Maintain docs:` flow line, `fabhelp_test.go` asserts it in `expectedMapped`, and the fabhelp package tests pass
- **AND** `/fab-help` lists it under Maintenance (not the "Other" bucket)

### Non-Goals

- Running the distillation on fab-kit's own `docs/memory/` corpus — a later invocation of the shipped skill, out of scope here.
- Any `fab memory-index` validation extension, new subcommand, or migration (`src/kit/migrations/` untouched — no user data restructured).
- Any Go change beyond the fabhelp.go registration above; no `_cli-fab.md` edit (no command-signature change).
- Rewriting specs (FKF governs `docs/memory/` only; specs are human-curated — Constitution VI).
- `docs/memory/` topic-file updates for this change — those belong to hydrate, NOT apply. `memory-docs/distill` (new) and `memory-docs/templates` (modify) are the change's Affected Memory targets, authored at the hydrate stage.

### Design Decisions

1. **Separate discoverable skill, not a `docs-reorg-memory` mode** — *Why*: user explicitly required discoverability ("If its a mode it wont be discoverable"). *Rejected*: a `docs-reorg-memory` mode (binding, per intake).
2. **Per-domain human gate over autonomous bulk rewrite** — *Why*: memory files encode load-bearing behavioral contracts; a human approves per domain seeing per-file diffs. *Rejected*: one-run full-corpus autonomous rewrite (too risky).
3. **Include the minimal fabhelp.go registration** (Assumption #11 recommended default) — *Why*: it is registration, not behavior; no command-signature change (so no `_cli-fab.md` edit); skipping it lands the skill in the "Other" bucket and violates the tree's own eight-point checklist. *Rejected*: prose-only with no Go touch.
4. **Cite the shipped FKF extract `$(fab kit-path)/reference/fkf.md`, not `docs/specs/fkf.md`** — *Why*: the skill ships to user repos where only the kit extract is reachable; deployed sibling skills cite it the same way. *Rejected*: citing the dev-repo `docs/specs/fkf.md` (absent in user repos).

## Tasks

### Phase 1: Skill source

- [x] T001 Write the new canonical skill `src/kit/skills/docs-distill-memory.md`: <!-- reworked (dgp8 cycle 2): log.seed.md reclassified as a curated read-only seed input (skill lines 51/52, 95/96, 133/135); Step 5 regen states index.md/log.md only; log.md derivation split (index tiers from folder contents + description:; each log.md from git history + per-change summaries, freeze-on-write append-only); stamp-type:memory hoisted into its own per-file Step 4 clause. --> frontmatter (`name`/behavior-naming change-id-free `description`/`user-invocable`), preamble-read blockquote, `## Contents` TOC, explicit `## Context Loading` override (memory indexes + target-domain files + `$(fab kit-path)/reference/fkf.md`; no active change/config/constitution), Purpose, Arguments (one domain per run), Pre-flight, Behavior (read-only analysis → per-file proposed-rewrite report → user confirm → apply the FKF §3.2/§3.3 rewrite with the rationale-preservation relocation guard → `fab memory-index --check` refuse-before-regen → `fab memory-index`), the `_shared/removed-domains.md` tombstone exemption, Output, `Next:` line, and closing Error Handling + Key Properties tables. <!-- R1 R2 R3 R4 R5 R6 R7 -->

### Phase 2: Sweep class — mirror & inventory surfaces

- [x] T002 [P] Create the SPEC mirror `docs/specs/skills/SPEC-docs-distill-memory.md` (Summary + behavior sections + Flow + Tools/Sub-agents tables) mirroring the skill, following the `SPEC-docs-reorg-memory.md` shape and the mechanical `SPEC-{name}.md` naming. <!-- R8 --> <!-- reworked (dgp8 cycle 2): log.seed.md reclassified as curated read-only seed input (lines 34→35, Flow 49→50); log.md derivation corrected in Flow; the two Bash steps (--check guard + regen) dedented out of the per-file branch into a "once, after ALL approved rewrites (Step 5)" block. -->
- [x] T003 [P] Add the `## /docs-distill-memory` section to `docs/specs/skills.md` (sibling: `## /docs-reorg-memory`). <!-- R9 --> <!-- reworked (dgp8 cycle 2): line 774 reworded — index.md/log.md named as the generated pair never hand-edited; log.seed.md described as a curated read-only seed input excluded from distillation. -->
- [x] T004 [P] Add a `docs-distill-memory` command row to `docs/specs/glossary.md` § Skills (sibling: the `/docs-reorg-memory` row). <!-- R9 -->
- [x] T005 [P] Add a `docs-distill-memory` row to `README.md` § Documentation command table (the `/docs-*` block). <!-- R9 -->

### Phase 3: Go registration + test

- [x] T006 Register the skill in `src/go/fab/cmd/fab/fabhelp.go`: add `"docs-distill-memory": "Maintenance"` to `skillToGroupMap`, and add the new command to the hardcoded `Maintain docs:` TYPICAL FLOW line. <!-- R10 -->
- [x] T007 Add `docs-distill-memory` to the `expectedMapped` list in `fabhelp_test.go` (`TestFabHelp_GroupMapping`), then run the fabhelp package tests (`go test ./src/go/fab/cmd/fab/` scoped to the affected package). <!-- R10 -->

## Execution Order

- T001 is the source of truth for T002 (the SPEC mirrors it); author T001 first, then T002.
- T003–T005 are independent inventory edits ([P]) — parallelizable with each other and with T002.
- T006 then T007 (test asserts the map entry T006 adds); run tests after T007.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/docs-distill-memory.md` exists as a user-invocable skill with `name: docs-distill-memory` and a change-id-free behavior-naming `description`; nothing under `.claude/skills/` is edited.
- [x] A-002 R2: The skill runs one domain per run, read-only analysis → per-file report → apply-on-approval; it never bulk-rewrites autonomously.
- [x] A-003 R3: A rewrite removes transition narration and superseded-state prose, strips change-ids from `description:` and caps it at ≤500 chars, and keeps trailing `(change-id)` + `*Introduced by*` provenance, citing `$(fab kit-path)/reference/fkf.md` §3.2/§3.3.
- [x] A-004 R4: The rationale-preservation guard is explicit — deliberate-behavior/"don't re-break" content is relocated into Design Decisions (`Why`/`Rejected`), never deleted; deletion is confined to narration recorded elsewhere.
- [x] A-005 R5: The generated files `index.md`/`log.md` are never hand-edited (and `log.seed.md` — a curated read-only seed input the generator never writes — is excluded from distillation, not classified as generated); the skill runs `fab memory-index --check` and refuses on exit 2 (surfacing the reorg pointer) before regenerating via `fab memory-index`; `_shared/removed-domains.md` is exempt.
- [x] A-006 R6: The skill is idempotent — a re-run on an already-distilled domain reports no rewrites and mutates nothing.
- [x] A-007 R7: The skill file carries the preamble-read line, a `## Contents` TOC, an explicit `## Context Loading` override, no `helpers:`, a `Next:` line, and Error Handling + Key Properties tables per the New Skill Checklist items 1–5.
- [x] A-008 R8: `docs/specs/skills/SPEC-docs-distill-memory.md` exists and mirrors the skill (Summary + Flow + Tools/Sub-agents tables). <!-- met (dgp8 cycle 2): the `--check` guard + `fab memory-index` regen are now dedented into a "once, after ALL approved rewrites (Step 5)" block, no longer implying per-file regen -->
- [x] A-009 R9: `docs/specs/skills.md`, `docs/specs/glossary.md`, and `README.md` each carry a `docs-distill-memory` entry; `fab-help.md` and `user-flow.md` are unchanged.
- [x] A-010 R10: `fabhelp.go` maps `docs-distill-memory` to "Maintenance" and lists it on the `Maintain docs:` line; `fabhelp_test.go` asserts it in `expectedMapped`.

### Behavioral Correctness

- [x] A-011 R5: On a born-compatible fab-kit tree `fab memory-index --check` returns exit 0/1 (the refuse-before-regen guard is a documented no-op there, not dead code); the exit-2 refuse path and pointer text match `_cli-fab` § fab memory-index. <!-- verified live: exit 0 on this tree; pointer text verbatim vs _cli-fab.md:831 -->
- [x] A-012 R3: Bare 4-char ids are handled identically to dated ids — kept in trailing-citation position, removed with the narration when woven into prose (Assumption #8).

### Scenario Coverage

- [x] A-013 R10: The fabhelp package tests pass (`go test ./src/go/fab/cmd/fab/`), including `TestFabHelp_GroupMapping` with the new `expectedMapped` entry.

### Code Quality

- [x] A-014 Pattern consistency: The skill and SPEC follow the `docs-reorg-memory`/`docs-hydrate-memory` sibling structure (frontmatter, Contents, Context Loading override, Key Properties, `Next:` posture) and house CommonMark conventions.
- [x] A-015 No unnecessary duplication: The skill references `_cli-fab` § fab memory-index and the FKF extract by pointer rather than restating exit tiers / normative rules; the SPEC does not duplicate skill body prose verbatim.

### Documentation Accuracy

- [x] A-016 R3 R5: Every FKF/CLI claim in the skill and SPEC (the §3.2 500-char cap, the §3.2 change-id ban, the §3.3 tombstone carve-out, the `fab memory-index --check` exit-2 pointer) matches the shipped `$(fab kit-path)/reference/fkf.md` and `_cli-fab` § fab memory-index. <!-- met (dgp8 cycle 2): (1) `log.seed.md` is now correctly a curated read-only seed input, never written by the generator, excluded from distillation (skill:52,96,135; SPEC:35,50; skills.md:774) — verified vs _cli-fab § fab memory-index seed-merge and templates.md:147; (2) log.md derivation corrected to the C-lite join of git history + per-change summaries (freeze-on-write, append-only), index tiers from folder contents + description: (skill:144; SPEC:65-66). -->

### Cross References

- [x] A-017 R8 R9: The SPEC mirror and every inventory surface (skills.md, glossary.md, README.md) are internally consistent with the skill's frontmatter `description` and behavior; the SPEC-mirror sweep class is complete (constitution-required).

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The new skill fills a gap no existing mechanism covers — per the intake's gap analysis, `/docs-reorg-memory` never rewrites body prose, `/docs-hydrate-memory` backfill is body-preserving, hydrate touches only its change's sections — so nothing is superseded; the fabhelp.go registration is purely additive.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Include the minimal fabhelp.go registration (skillToGroupMap Maintenance entry + Maintain docs: flow line + fabhelp_test.go expectedMapped), no `_cli-fab.md` edit | Intake Assumption #11 recommended default; registration not behavior; no command-signature change; skipping violates the eight-point checklist | S:70 R:90 A:80 D:70 |
| 2 | Certain | `/fab-help` group = "Maintenance" (with the reorg skills), not "Setup" | Intake Assumption #6; maintenance/cleanup semantics match the reorg skills | S:70 R:95 A:85 D:75 |
| 3 | Confident | Skill declares no `helpers:`; carries an explicit `## Context Loading` override + in-body `_cli-fab` § fab memory-index pointer | Intake Assumption #7; sibling pattern (`docs-reorg-memory` pointer style); skill file wins over always-load per `_preamble` §1 | S:60 R:90 A:80 D:70 |
| 4 | Confident | SPEC mirror content follows the `SPEC-docs-reorg-memory.md` shape (Summary + behavior sections + Flow + Tools/Sub-agents tables) and carries a trailing `(260717-dgp8)` provenance note | SPEC format policy (Checklist item 6); sibling SPEC is the closest pattern | S:65 R:85 A:85 D:75 |
| 5 | Certain | Skill cites `$(fab kit-path)/reference/fkf.md` §3.2/§3.3 (the shipped extract), not `docs/specs/fkf.md` | Deployed skills cite the kit extract (reachable in user repos); dev-repo spec is absent there | S:85 R:90 A:90 D:85 |

5 assumptions (3 certain, 2 confident, 0 tentative).
