# Plan: Re-tier Claude `doing` and `hydrate` Fills to Sonnet 5.5

**Change**: 261030-lc7k-retier-doing-hydrate-sonnet
**Intake**: `intake.md`

## Requirements

### Defaults: Claude role fills

#### R1: `doing` and `hydrate` resolve to Sonnet 5.5 at `high`
The built-in `providers.claude.profiles` in `src/go/fab/defaults.yaml` SHALL fill `doing` and `hydrate` with `{ model: claude-sonnet-5-5, effort: high }`. `default`/`review` SHALL stay `claude-opus-5-5`/`high`; `operator`/`fast` SHALL stay `claude-sonnet-5-5`/`medium`. Every effort MUST remain explicit. No other provider block changes.

- **GIVEN** a project with no `agent.profiles` or `providers.claude.profiles` override
- **WHEN** `fab agent apply -o yaml` (or `fab agent hydrate -o yaml`) runs
- **THEN** it resolves `model: claude-sonnet-5-5`, `effort: high`, `model_alias: sonnet`
- **AND** `fab agent review -o yaml` still resolves `claude-opus-5-5`/`high`

#### R2: The fills comment stays accurate
The comment above `profiles:` in `defaults.yaml` MUST state, without contradiction: efforts are explicit because fab never relies on a model's default; Opus 5.5 defaults to `medium` (explicit `high` holds the Opus rows); Sonnet 5.5 defaults to `high` (explicit `medium` holds `operator`/`fast`); `doing`/`hydrate`'s `high` equals Sonnet 5.5's default today and stays explicit anyway.

- **GIVEN** the edited `defaults.yaml`
- **WHEN** a reader compares the comment to the six rows
- **THEN** no sentence describes all Sonnet rows as `effort: medium`

#### R3: The pinned-defaults test tracks the new fills
`TestDefaultRoleProfilesArePinned` (`src/go/fab/internal/agent/agent_test.go`) MUST pin `RoleDoing` and `RoleHydrate` to `claude`/`claude-sonnet-5-5`/`high`; the other four rows stay.

- **GIVEN** the edited `defaults.yaml`
- **WHEN** `go test ./internal/agent/...` runs
- **THEN** `TestDefaultRoleProfilesArePinned` and `TestDocTablesMatchAgentMaps` pass

### Specs: stage-models

#### R4: `docs/specs/stage-models.md` records the shipped split honestly
The § Default role profiles mirror table SHALL show `doing`/`hydrate` on `claude-sonnet-5-5`/`high`. The "Why these defaults" paragraph, § apply↔review coupling, and § Model upgrade path SHALL no longer claim `doing`/`hydrate` run Opus or that `doing` and `review` resolve to the same profile. The coupling mechanism is kept; the section states the accepted, unmeasured rework risk and the one-row per-project override (`agent.profiles.doing`).

- **GIVEN** the edited spec
- **WHEN** the drift-guard test parses the mirror table
- **THEN** it matches `defaults.yaml`
- **AND** a grep of the spec for `doing` running Opus finds no present-truth claim

### Kit skills: CLI reference

#### R5: `_cli-fab.md` `fab resolve-agent apply` examples show the new `doing` fill
`src/kit/skills/_cli-fab.md` examples that run `fab resolve-agent apply` SHALL print `model=claude-sonnet-5-5` (full form) and `model=sonnet` (`--alias` form); the `--effort medium` example keeps `effort=medium`.

- **GIVEN** the edited `_cli-fab.md`
- **WHEN** a reader runs the shown command against the new defaults
- **THEN** the printed `model=` lines match

### Config upgrade: digest catalog

#### R6: The rendered providers advert digest is cataloged (append-only)
`knownGeneratedSystemParagraphDigests` in `src/go/fab/internal/configupgrade/configupgrade.go` MUST gain the new current providers-advert digest as its final entry (labelled `261030-lc7k`); the prior `ab93e3cc…` entry is relabelled historical; no entry is removed or altered.

- **GIVEN** the edited `defaults.yaml`
- **WHEN** `go test ./internal/configupgrade/...` runs
- **THEN** `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` passes

### Non-Goals

- `default`/`review`/`operator`/`fast` fills, other providers, command templates — out of scope (owner-scoped)
- Go logic, CLI surface, migration — built-in defaults are not user data
- A rework-rate measurement harness — possible follow-up

### Design Decisions

#### Kit Default Runs `doing` and `hydrate` on Sonnet 5.5, `review` Stays Opus
**Decision**: The Claude `doing` and `hydrate` fills move to `claude-sonnet-5-5`/`high` kit-wide; `default` and `review` stay `claude-opus-5-5`/`high`, making review the only Opus stage downstream of intake (`intake` rides `default`).
**Why**: `doing` (apply) and `hydrate` are the highest-output-volume stages; hydrate writes memory from a finished, reviewed diff (within Sonnet's range). The owner chose the kit-wide default over a personal override. Accepted tradeoff: per the apply↔review coupling, a cheaper author under an Opus critic may raise must-fix findings and rework cycles (cap 3) — unmeasured; a project that sees it sets `agent.profiles.doing` back to Opus.
**Rejected**: machine-local `--system` override (owner wants it for everyone); hydrate-only re-tier (agent's counter-proposal, owner held on `doing`); dropping the now-default-equal explicit `effort: high` (fab never relies on a model's default).
*Introduced by*: 261030-lc7k-retier-doing-hydrate-sonnet

## Tasks

### Phase 1: Setup

- [x] T001 Edit `src/go/fab/defaults.yaml`: `doing`/`hydrate` → `{ model: claude-sonnet-5-5, effort: high }`, rewrite the comment above `profiles:` per R2 <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 [P] Update `TestDefaultRoleProfilesArePinned` rows for `RoleDoing`/`RoleHydrate` in `src/go/fab/internal/agent/agent_test.go` <!-- R3 -->
- [x] T003 [P] Update `docs/specs/stage-models.md`: mirror table cells, "Why these defaults", § apply↔review coupling, § Model upgrade path (fix inbound anchors if the coupling heading changes) <!-- R4 -->
- [x] T004 [P] Update `src/kit/skills/_cli-fab.md` lines ~326/331/374 (`model=claude-sonnet-5-5`, `model=sonnet`) <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Harvest the new providers-advert digest from the failing catalog test, append it to `knownGeneratedSystemParagraphDigests` in `src/go/fab/internal/configupgrade/configupgrade.go`, relabel `ab93e3cc…` as historical; gofmt; run `go test ./internal/agent/... ./internal/configupgrade/... ./cmd/fab/...` from `src/go/fab` and re-grep `src/go` for stale literals <!-- R6 -->

## Execution Order

- T001 blocks T005 (the digest covers the final rendered bytes of the fills)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` fills `doing`/`hydrate` with `claude-sonnet-5-5`/`high`; the other four Claude rows and all other provider blocks are byte-unchanged
- [x] A-002 R2: The `defaults.yaml` comment names the three effort facts and no longer calls all Sonnet rows `medium`
- [x] A-003 R3: `TestDefaultRoleProfilesArePinned` pins the new values and passes
- [x] A-004 R4: The stage-models mirror table matches `defaults.yaml` (`TestDocTablesMatchAgentMaps` passes)
- [x] A-005 R5: All three `fab resolve-agent apply` examples in `_cli-fab.md` show the Sonnet fill (`model=claude-sonnet-5-5` / `model=sonnet`)
- [x] A-006 R6: The new digest is the catalog's last entry, labelled current with `261030-lc7k`; `ab93e3cc…` relabelled historical; no entry removed

### Behavioral Correctness

- [x] A-007 R1: `fab agent apply -o yaml` / `fab agent hydrate -o yaml` built from the branch resolve `claude-sonnet-5-5`/`high`/`sonnet`; `review` resolves Opus
- [x] A-008 R4: The spec's "Why these defaults", § apply↔review coupling, and § Model upgrade path contain no present-truth claim that `doing`/`hydrate` run Opus or that `doing` and `review` share a profile; the coupling mechanism and the accepted, unmeasured risk + `agent.profiles.doing` override are stated

### Scenario Coverage

- [x] A-009 R6: `go test ./internal/agent/... ./internal/configupgrade/... ./cmd/fab/...` passes from `src/go/fab`

### Edge Cases & Error Handling

- [x] A-010 R4: Any retitled heading has no dangling inbound anchor links in `docs/` or `src/kit/`

### Code Quality

- [x] A-011 Pattern consistency: New catalog comments and YAML alignment follow the surrounding entries' style
- [x] A-012 No unnecessary duplication: Spec prose states each fact once (no restated table values in prose beyond what the paragraph already did)
- [x] A-013 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`
- [x] A-014 Deployed content cites no fab-kit-only paths in the `_cli-fab.md` edit
- [x] A-015 Go files are gofmt-clean

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Requirements R1–R6 mirror intake § What Changes 1–5 exactly | Intake is fully decided (5.0, 0 unresolved) | S:95 R:90 A:95 D:95 |
| 2 | Confident | Sibling sweep found no other present-truth "doing/hydrate run Opus" claim outside stage-models.md and `_cli-fab.md` (migrations are history) | Repo grep over docs/specs, docs/site, src/kit at plan time | S:80 R:90 A:85 D:80 |
| 3 | Confident | Retitle § apply↔review coupling only if the old title becomes false, fixing inbound anchors in the same task | Intake permits retitle; anchors are the risk | S:70 R:95 A:80 D:70 |

3 assumptions (1 certain, 2 confident, 0 tentative).
