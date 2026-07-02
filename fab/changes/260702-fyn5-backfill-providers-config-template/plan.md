# Plan: Backfill Providers Config Template Migration

**Change**: 260702-fyn5-backfill-providers-config-template
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md. Change type: chore (comment-only YAML backfill +
     VERSION bump + new markdown migration instruction file). No Go/binary,
     no tests, no skill/spec changes. -->

### Migration: File & Slot

#### R1: New migration instruction file at the correct slot
The change SHALL add a new kit migration file `src/kit/migrations/2.13.1-to-2.13.2.md` following the standard Summary / Pre-check / Changes / Verification shape, with FROM = the real current VERSION (`2.13.1`) and TO = a patch bump (`2.13.2`).

- **GIVEN** the current `src/kit/VERSION` is `2.13.1` and no migration targets `2.13.0→2.13.1` or `2.13.1→2.13.2`
- **WHEN** the migration file is authored
- **THEN** its filename is exactly `2.13.1-to-2.13.2.md` and its title line reads `# Migration: 2.13.1 to 2.13.2`
- **AND** its range does not overlap any existing migration file's range (`fab migrations-status` reports no overlap)

#### R2: VERSION bump to 2.13.2
`src/kit/VERSION` SHALL be bumped from `2.13.1` to `2.13.2` (patch — comment-only backfill, no binary change).

- **GIVEN** `src/kit/VERSION` currently contains `2.13.1`
- **WHEN** the version bump is applied
- **THEN** `src/kit/VERSION` contains exactly `2.13.2` (bare semver, trailing newline preserved) and the migration's TO matches it

### Migration: Pre-check (idempotency & applicability)

#### R3: Pre-check gates on config presence, providers block, and codex/gemini sentinel
The migration's Pre-check SHALL (a) skip entirely when `fab/project/config.yaml` is absent; (b) STOP with run-the-chain guidance when no top-level `providers:` key exists; and (c) skip (sentinel) when the config already carries a `codex` or `gemini` provider — live (`codex:` / `gemini:` mapping keys) or as the commented starter marker (`# codex:` / `# gemini:`).

- **GIVEN** a project running `/fab-setup migrations`
- **WHEN** `fab/project/config.yaml` does not exist
- **THEN** the migration prints `Skipped: fab/project/config.yaml not present.` and is a complete no-op
- **AND WHEN** the config exists but has no top-level `providers:` key
- **THEN** the migration STOPs and directs the user to run the `2.12.1-to-2.13.0` migration first
- **AND WHEN** the config already contains a `codex`/`gemini` key or `# codex:`/`# gemini:` marker
- **THEN** the migration prints `Skipped: codex/gemini provider template already present.` and makes no edit (re-run no-op)

### Migration: Changes (comment-only content)

#### R4: Providers explanatory header refresh/insert
The migration SHALL ensure the v2.13.1 providers explanatory header (including the per-provider-notes paragraph, detection line `# Per-provider notes (kept out of the blocks below so uncommenting a whole block`) is present above the `providers:` line — replacing a pre-#467 header when detected (distinctive old line `# dispatch; ABSENT → native Agent-tool dispatch). The two are NOT merged.`), or inserting the full header when no providers header exists at all. The header text SHALL be the verbatim scaffold wording.

- **GIVEN** a config whose `providers:` block has no per-provider-notes paragraph
- **WHEN** the pre-#467 header is present
- **THEN** the migration replaces that header paragraph with the v2.13.1 wording
- **AND WHEN** no providers header exists at all (a `2.12.1-to-2.13.0`-migrated config)
- **THEN** the migration inserts the full v2.13.1 header immediately above the `providers:` line

#### R5: Claude commented dispatch_command line
When a `claude:` provider exists and carries no `dispatch_command` (live or commented), the migration SHALL append the commented `dispatch_command` line directly after its `session_command` (replacing the old `# no dispatch_command → …` note when present). When no `claude:` provider exists, this piece SHALL be skipped while the codex/gemini blocks are still appended.

- **GIVEN** a config whose `claude:` provider has a `session_command` and no `dispatch_command`
- **WHEN** the migration applies
- **THEN** the commented line `# dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'` is appended directly after the `session_command`, at the provider's field indent
- **AND WHEN** no `claude:` provider exists (e.g. `UNNAMED_PROVIDER` from the 2.12.1→2.13.0 halt-and-ask path)
- **THEN** this piece is skipped and the codex/gemini blocks are still appended

#### R6: Codex/gemini commented starter blocks
The migration SHALL append the commented codex and gemini starter blocks after the last entry of the `providers:` mapping (before the next top-level key), with the exact scaffold content.

- **GIVEN** the `providers:` mapping ends before the next top-level key
- **WHEN** the migration applies
- **THEN** the `# codex:` and `# gemini:` commented blocks (each with `session_command` and `dispatch_command` lines, the gemini `dispatch_command` carrying the trailing `# no {effort} flag; no -p …` note) are inserted after the last provider entry

#### R7: Indent adaptation to the file's own mapping indent
The migration SHALL detect the config's mapping indent from the existing `providers:` block children (2-space scaffold vs 4-space go-yaml-written configs) and emit all commented lines so that stripping the leading `# ` from every line of a block yields valid YAML at the file's own indent.

- **GIVEN** a config whose `providers:` children are indented 4 spaces (e.g. fab-kit's own, go-yaml-written)
- **WHEN** the migration emits the codex/gemini blocks
- **THEN** the block lines are `    # codex:` and `    #     session_command: …` so that stripping `# ` per line yields `    codex:` / `        session_command: …` — valid YAML at the file's 4-space indent
- **AND** the migration includes the 4-space worked example (proven by the in-session hand-patch)

#### R8: Value preservation (comment-only, semantics unchanged)
The migration SHALL add no live key and remove/modify none — all live keys, values, and unrelated comments are preserved verbatim, and `yq '.providers'` (and `.agent`) semantics are identical before and after.

- **GIVEN** any config the migration edits
- **WHEN** the edit completes
- **THEN** `yq '.'` still parses, and `yq '.providers'` / `yq '.agent'` (comments stripped) return semantically identical values to pre-migration

### Migration: Verification section

#### R9: Verification steps documented in the migration file
The migration file's `## Verification` section SHALL document: YAML still parses; live `.providers`/`.agent` semantics unchanged; the per-provider-notes detection line, commented claude `dispatch_command` (when a claude provider exists), and `# codex:` / `# gemini:` markers present; a mechanical uncomment-and-parse round-trip of a starter block yields valid YAML; and re-run is a complete no-op (sentinel trips).

- **GIVEN** a reader applying or auditing the migration
- **WHEN** they read the `## Verification` section
- **THEN** all five verification steps above are present and actionable

### Repo: fab-kit's own config.yaml (worked example)

#### R10: fab-kit's own config.yaml already carries the migration's intended output
`fab/project/config.yaml` in this repo SHALL match the migration's intended output — the full v2.13.1 header, the commented claude `dispatch_command`, and the codex/gemini starter blocks, adapted to the file's 4-space indent — serving as the migration's worked example.

- **GIVEN** this repo's `fab/project/config.yaml`
- **WHEN** its `providers:` section is inspected
- **THEN** it carries the full header (per-provider-notes paragraph), the commented `dispatch_command` after claude's `session_command`, and the 4-space-adapted `# codex:` / `# gemini:` blocks
- **AND** uncommenting a starter block (strip `# ` per line) yields valid YAML that `yq` parses with `.providers.codex` / `.providers.gemini` as mappings

### Non-Goals

- No Go/binary changes, no tests (`test_paths` untouched) — comment-only YAML + markdown.
- No skill changes, no `_cli-fab.md` or SPEC-mirror updates — `/fab-setup migrations` applies any migration file generically.
- No change to the `agent.tiers` comment block or any other scaffold section (#467's scaffold diff touched only the providers section).
- No live provider entries written — a user who wants codex/gemini uncomments and adapts.
- No memory edits — the `distribution/migrations.md` catalog entry is hydrate's job, not apply's.

### Design Decisions

1. **Deliverable is a kit migration file, not a script/subcommand** — mandated by `context.md` § Migrations + Constitution I (Pure Prompt Play). *Why*: user-config restructuring ships as an LLM-driven migration instruction file. *Rejected*: ad-hoc script / new subcommand (violates Pure Prompt Play).
2. **Sentinel on codex/gemini presence (live or commented)** — the migration's own output is the codex/gemini blocks, so their presence is the idempotency marker. *Why*: comment-sentinel precedent (`2.2.0-to-2.3.0`, `2.11.0-to-2.12.0`); the live-key check also protects users who already configured those providers. *Rejected*: sentinel on the header alone (would miss users who added live codex/gemini).
3. **Indent-adapt to the file's detected mapping indent** — emit commented lines whose `# `-stripped form is valid YAML at 2-space or 4-space. *Why*: go-yaml-written configs (fab-kit's own) are 4-space; the scaffold is 2-space; proven by the in-session hand-patch and the round-trip test. *Rejected*: hardcoding 2-space (breaks 4-space configs on uncomment).

## Tasks

### Phase 1: Migration file

- [x] T001 Create `src/kit/migrations/2.13.1-to-2.13.2.md` with the `# Migration: 2.13.1 to 2.13.2` title and a `## Summary` section describing the comment-only providers-template backfill (header + commented claude `dispatch_command` + codex/gemini starter blocks), sentinel-guarded, config-only, patch bump, referencing the comment-only precedents (`2.9.2-to-2.10.0`, `2.11.0-to-2.12.0`) and the comment-sentinel precedent (`2.2.0-to-2.3.0`) <!-- R1 -->
- [x] T002 Write the `## Pre-check` section: (1) skip when `fab/project/config.yaml` absent (`Skipped: fab/project/config.yaml not present.`); (2) STOP with run-`2.12.1-to-2.13.0`-first guidance when no top-level `providers:` key; (3) sentinel skip when a `codex`/`gemini` key or `# codex:`/`# gemini:` marker is already present (`Skipped: codex/gemini provider template already present.`) <!-- R3 -->
- [x] T003 Write `## Changes` step 1 (header refresh/insert): detect the per-provider-notes line; replace a pre-#467 header when the distinctive old line is present, else insert the full v2.13.1 header above `providers:` when no header exists; embed the verbatim scaffold header block <!-- R4 -->
- [x] T004 Write `## Changes` step 2 (claude commented `dispatch_command`): append the commented line after claude's `session_command` (replace the old `# no dispatch_command →` note when present); skip when no `claude:` provider exists; embed the exact line <!-- R5 -->
- [x] T005 Write `## Changes` step 3 (codex/gemini starter blocks): append after the last `providers:` entry; embed the exact 2-space scaffold blocks <!-- R6 -->
- [x] T006 Write `## Changes` step 4 (indent adaptation): detect the file's mapping indent from `providers:` children; emit commented lines so `# `-stripping yields valid YAML at the file's own indent; include the 4-space worked example. Add step 5 (value preservation): comment-only, `yq '.providers'` semantically identical before/after; use an atomic temp+rename write <!-- R7 -->
- [x] T007 Write the `## Verification` section: (1) `yq '.' fab/project/config.yaml` parses; (2) `.providers`/`.agent` semantics unchanged; (3) per-provider-notes detection line + commented claude `dispatch_command` (when claude exists) + `# codex:`/`# gemini:` markers present; (4) uncomment-a-block-and-parse round-trip yields valid YAML; (5) re-run is a complete no-op <!-- R9 -->

### Phase 2: VERSION bump

- [x] T008 Bump `src/kit/VERSION` from `2.13.1` to `2.13.2` (bare semver, trailing newline preserved) <!-- R2 -->

### Phase 3: Verification & worked-example confirmation

- [x] T009 Verify this repo's `fab/project/config.yaml` already matches the migration's intended output (full v2.13.1 header, commented claude `dispatch_command`, 4-space-adapted codex/gemini blocks); no edit needed if already conformant. Confirm `fab migrations-status` reports no overlap for the new slot <!-- R10 -->
- [x] T010 Round-trip verify: on a temp copy of this repo's config, `yq '.'` parses; uncomment the codex block (strip `# ` per line) → `yq '.providers.codex'` returns a mapping; `yq '.providers'` semantics identical before uncomment; confirm the migration file's Verification steps are mechanically executable <!-- R8 R10 -->

## Execution Order

- T001–T007 build the single migration file top-down (Summary → Pre-check → Changes → Verification) and are ordered; they may be authored in one pass over the file.
- T008 (VERSION) is independent of the migration-file tasks.
- T009–T010 (verification) run last, after the file and VERSION exist.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/migrations/2.13.1-to-2.13.2.md` exists with title `# Migration: 2.13.1 to 2.13.2` and the four standard sections (Summary / Pre-check / Changes / Verification)
- [x] A-002 R2: `src/kit/VERSION` contains exactly `2.13.2` and the migration's TO matches
- [x] A-003 R3: the Pre-check documents all three gates (config-absent skip, no-`providers:` STOP, codex/gemini sentinel skip) with the exact skip/stop messages
- [x] A-004 R4: the Changes section embeds the verbatim v2.13.1 header and documents both the pre-#467-header-replace and no-header-insert paths, keyed on the per-provider-notes detection line
- [x] A-005 R5: the Changes section documents appending the commented claude `dispatch_command` after `session_command`, the old-note-replace case, and the no-`claude:`-provider skip, with the exact line
- [x] A-006 R6: the Changes section embeds the exact codex/gemini commented starter blocks appended after the last `providers:` entry
- [x] A-007 R7: the Changes section documents indent detection and includes the 4-space worked example whose `# `-stripped form is valid YAML
- [x] A-008 R9: the Verification section lists all five verification steps (parse, semantics-unchanged, markers-present, uncomment-round-trip, re-run-no-op)

### Behavioral Correctness

- [x] A-009 R8: uncommenting a starter block (strip `# ` per line) on this repo's 4-space config yields valid YAML (`yq '.providers.codex'` returns a mapping); `yq '.providers'` semantics are identical before uncomment
- [x] A-010 R1: `fab migrations-status` (or manual range check) reports no overlap between `2.13.1-to-2.13.2` and any existing migration range

### Scenario Coverage

- [x] A-011 R3: re-running the migration on a config that already carries the codex/gemini blocks is a complete no-op (sentinel trips) — verified against this repo's already-conformant config
- [x] A-012 R7: the same migration content adapts to both 2-space (scaffold) and 4-space (fab-kit's own) mapping indents

### Edge Cases & Error Handling

- [x] A-013 R5: the no-`claude:`-provider case (e.g. `UNNAMED_PROVIDER`) skips the dispatch_command piece while still appending the codex/gemini blocks
- [x] A-014 R3: the no-`providers:`-block case STOPs with guidance to run `2.12.1-to-2.13.0` first (only reachable via direct-file invocation, not the chained flow)

### Code Quality

- [x] A-015 Pattern consistency: the migration file follows the catalog's Summary / Pre-check / Changes / Verification structure and prose conventions of neighboring migrations (`2.9.2-to-2.10.0`, `2.11.0-to-2.12.0`, `2.2.0-to-2.3.0`, `2.12.1-to-2.13.0`)
- [x] A-016 No unnecessary duplication: the embedded header / dispatch_command / codex-gemini blocks are copied verbatim from the scaffold source (`src/kit/scaffold/fab/project/config.yaml`), not paraphrased
- [x] A-017 documentation_accuracy: every literal (skip/stop messages, sentinel markers, detection lines, embedded YAML) in the migration matches the actual scaffold and precedent files
- [x] A-018 cross_references: precedent migrations cited in the Summary (`2.9.2-to-2.10.0`, `2.11.0-to-2.12.0`, `2.2.0-to-2.3.0`, `1.9.1-to-1.9.2` for patch-target) name real, existing files

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- fab-kit's own `fab/project/config.yaml` was already backfilled and committed on this branch (commit `3a4b7372`), so T009 is a verify-only task — no config edit is expected during apply.

## Assumptions

<!-- Apply-time graded decisions (three grades only). Scores column required. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Deliverable is a kit migration file `2.13.1-to-2.13.2.md` (Summary/Pre-check/Changes/Verification), not a script | Mandated by intake A1 + context.md § Migrations + Constitution I; intake is explicit | S:90 R:85 A:95 D:90 |
| 2 | Confident | Slot `2.13.1-to-2.13.2` — FROM = real current VERSION `2.13.1`, TO = patch bump | FROM per `2.9.2-to-2.10.0` chaining precedent; patch per `1.9.1-to-1.9.2` (comment-only, no binary change). Release author may re-slot before release | S:65 R:85 A:75 D:60 |
| 3 | Confident | Sentinel skips when any codex/gemini key (live or `# codex:`/`# gemini:`) is present | Comment-sentinel precedent (`2.2.0-to-2.3.0`, `2.11.0-to-2.12.0`); the migration's own output is the marker; live-key check protects already-configured users | S:70 R:80 A:80 D:70 |
| 4 | Certain | Indent-adapt to the file's detected mapping indent (2-space scaffold vs 4-space go-yaml) so uncommenting yields valid YAML | Proven in-session on fab-kit's 4-space config and re-verified via a round-trip parse test in this apply run | S:85 R:90 A:95 D:90 |
| 5 | Certain | Embed header / dispatch_command / codex-gemini content verbatim from `src/kit/scaffold/fab/project/config.yaml` | The scaffold (#467, d7a87acb) is the single source of the template text; paraphrasing would drift | S:90 R:85 A:95 D:90 |
| 6 | Certain | fab-kit's own config.yaml needs no edit during apply — already backfilled and committed on this branch (commit `3a4b7372`) | Verified: HEAD config already carries the full 4-space-adapted template; git status clean; T009 is verify-only | S:95 R:90 A:95 D:95 |
| 7 | Confident | Change type is `chore` → no `## Deletion Candidates` section is templated (review-owned, skipped for chore) | Per dispatch instructions (change type chore) and the plan-template parser contract note | S:75 R:85 A:85 D:80 |

7 assumptions (4 certain, 3 confident, 0 tentative).
