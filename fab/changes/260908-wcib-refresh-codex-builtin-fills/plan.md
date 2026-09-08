# Plan: Refresh Codex Built-in Fills

**Change**: 260908-wcib-refresh-codex-builtin-fills
**Intake**: `intake.md`

## Requirements

### Agent defaults: codex fill table

#### R1: Dense codex fill map with GPT-6 Astra as `default`
`src/go/fab/defaults.yaml` `providers.codex.profiles` MUST carry exactly six rows, each naming both `model` and `effort`: `default` and `doing` → `gpt-6-astra` @ `high`; `review` → `gpt-5.6-sol` @ `xhigh`; `hydrate` → `gpt-5.6-sol` @ `high`; `operator` → `gpt-5.6-luna` @ `medium`; `fast` → `gpt-5.6-luna` @ `low`. No row MAY use effort `ultra`. The block comment MUST state the four points from intake § What Changes 1 (pinned catalog slugs verified against `~/.codex/models_cache.json`; dense on purpose; author/critic split; `ultra` never, `max` override-only). The claude, agy, and kimi blocks and both codex command lines MUST be byte-unchanged.

- **GIVEN** a locally built `fab` binary from this tree
- **WHEN** `fab agent <role> --provider codex -o yaml` runs for each of the six roles
- **THEN** it resolves exactly the six target `{model, effort}` pairs, and `fab agent <role> -o yaml` under the shipped claude knobs is unchanged

#### R2: Tests pin the new table and enforce density
`TestNonClaudeProviderFillsArePinned` (`src/go/fab/internal/agent/agent_test.go`) MUST pin the six new codex rows; agy and kimi pins stay as-is. `src/go/fab/internal/agent/defaults_test.go` MUST gain an assertion that every codex role fill names a model (agy exempt, kimi still asserted empty). Doc comments in `agent_test.go` (above both pin tests) and `src/go/fab/cmd/fab/config_test.go` (above `TestConfigReferenceDocumentsProviderFill`) MUST NOT name the non-existent `TestMirrorDocsMatchDefaultProfiles` / `TestCLIFabReferenceListsDefaultRoles` and MUST NOT call codex's map sparse. The legacy fixture `legacyV2198ToV2220ProvidersAdvert` in `configupgrade_test.go` MUST NOT be edited.

- **GIVEN** the edited `defaults.yaml`
- **WHEN** `go test ./internal/agent/ ./internal/configref/ ./internal/configupgrade/ ./cmd/fab/` runs from `src/go/fab`
- **THEN** all packages pass, and reverting any codex row to effort-only would fail the new density assertion

#### R3: Go comments and the rendered reference call only agy sparse
The comments in `src/go/fab/internal/agent/agent.go` (`defaultProviders`) and `src/go/fab/internal/configref/configref.go` (the empty-default convention comment and `providerDefault`), and the rendered `providers:` prose string in `configref.go` ("the non-claude maps are SPARSE…"), MUST describe codex's map as dense (every role pins a model) and agy's as the sparse example.

- **GIVEN** the locally built binary
- **WHEN** `fab config explain providers` runs
- **THEN** the codex block shows six `model`+`effort` rows in `RoleNames()` order and the prose no longer calls the codex map sparse

### Specs and skills: mirrors and sibling sweep

#### R4: Spec mirrors reflect the dense shape and record the rationale
`docs/specs/stage-models.md` § Built-in providers MUST show the codex sample in the dense shape (IDs elided as `...`), rewrite the "maps are SPARSE" prose so agy is the sparse exemplar, rewrite the role-differentiation consequence bullet role-relatively, add a short role-relative "Why these codex fills" paragraph (incl. the inferred-cost limitation and the `ultra` exclusion), add an upgrade note (no migration) for the flat `providers.codex.model` alias now reaching only `default`, correct § Drift guard (no `TestMirrorDocsMatchDefaultProfiles`; the sample is shape-only, the rendered reference is guarded by `TestConfigReferenceDocumentsProviderFill`), and extend the decision-lineage note with `260908-wcib`. `docs/specs/glossary.md` `providers` entry and the codex `profiles:` comment in `docs/specs/architecture.md` MUST stop calling codex fills sparse. No literal codex model ID MAY appear in any spec.

- **GIVEN** the edited specs
- **WHEN** `grep -rn 'gpt-5.6\|gpt-6' docs/specs` runs
- **THEN** it returns nothing, and `grep -in 'sparse' docs/specs/stage-models.md docs/specs/glossary.md docs/specs/architecture.md` matches only agy (or user `agent.profiles`) contexts

#### R5: Skill prose is swept
`src/kit/skills/_cli-fab.md` MUST say codex's fills are exhaustive (both the `--provider` bullet and the `providers:` block paragraph) and its two illustrative codex `apply` outputs MUST show `effort=high`. `src/kit/skills/_cli-agents.md`'s codex Model-discovery cell MUST list all six roles and add `~/.codex/models_cache.json` as the verified catalog source, and its flat-alias sentence MUST say the alias reaches only `default`. Only canonical `src/kit/skills/` files are edited.

- **GIVEN** the edited skills
- **WHEN** `grep -n 'sparsely\|codex.s and agy.s are \*\*sparse\|effort=xhigh' src/kit/skills/_cli-fab.md` runs
- **THEN** it returns nothing

#### R6: Repo-wide sweep is clean and resolution is verified
Before apply finishes, the sweep greps from intake § What Changes 7 MUST be run over `src/`, `docs/specs/`, `docs/memory/` (memory is hydrate's, but the sweep must list its hits for hydrate), excluding `fab/changes/`, `.claude/`, `docs/findings/`, `docs/memory/**/log*.md`, `src/kit/migrations/`, and the legacy fixture; every non-excluded codex-sparse/effort-only/inherits claim MUST be gone. `gofmt -l` MUST be clean on touched Go files.

- **GIVEN** the finished apply
- **WHEN** `go test ./...` runs from `src/go/fab`, the local binary is built, and the six-role check runs
- **THEN** tests pass and the six profiles match R1

### Non-Goals
- agy, kimi, and claude fills; any command grammar — unchanged
- Effort-enum or model validation — provider neutrality stands
- A migration file — fills live in the binary, never seeded into user config
- A/B of `gpt-5.6-terra` — follow-up idea only

### Design Decisions

#### The Codex Fill Map Is Dense by Policy; Author and Critic Run Different Models
**Decision**: `providers.codex.profiles` ships all six roles with both `model` and `effort` set. `default`/`doing` take the catalog's top model at `high`; `review` takes a different model at `xhigh`; `hydrate` the same second model at `high`; `operator`/`fast` the cheap model. A structural test rejects any codex row without a model. `ultra` is never shipped; `max` is override-only.
**Why**: Effort-only rows made a `default` bump silently repoint `doing`/`review`; the reader could not see what a role resolved to without knowing the per-field merge rule. Putting the critic on a different model than the author keeps the reviewer from sharing the author's blind spots (the 2026-08-10 four-provider comparison recorded reviewer strictness tracking the worker model on every arm). `ultra` means automatic task delegation, which breaks the single-worker dispatch contract. Per-model codex pricing was not available; the tier ladder is inferred from codex's own descriptions and priority order.
**Rejected**: Keeping effort-only rows (the inheritance is the defect); same model for author and critic (shared blind spots); `gpt-5.6-terra` (no repo evidence); a migration (nothing on disk restructures).
*Introduced by*: 260908-wcib-refresh-codex-builtin-fills

## Tasks

### Phase 2: Core Implementation

- [x] T001 Replace the codex `profiles:` block and its comment in `src/go/fab/defaults.yaml` with the six dense rows and the four-point comment; leave every other block byte-identical <!-- R1 -->
- [x] T002 Update `src/go/fab/internal/agent/agent_test.go` (pin table + both doc comments), add the codex density assertion in `src/go/fab/internal/agent/defaults_test.go`, fix the comment in `src/go/fab/cmd/fab/config_test.go`, reword the comments/prose (incl. the registry-row `Description`) in `src/go/fab/internal/agent/agent.go` and `src/go/fab/internal/configref/configref.go`, retarget the two flat-alias tests in `agent_test.go`/`resolve_agent_test.go` to `intake`, and append the new providers-advert digest in `src/go/fab/internal/configupgrade/configupgrade.go` <!-- R2 R3 -->

### Phase 3: Integration & Edge Cases

- [x] T003 [P] Update `docs/specs/stage-models.md` (sample, sparse prose, consequence bullet, why-paragraph, upgrade note, drift guard, lineage), `docs/specs/glossary.md`, and `docs/specs/architecture.md` <!-- R4 --> <!-- rework: stale sparse/effort-only claims at config.md:122-125, architecture.md:232, stage-models.md:235-236 (review cycle 1) --> <!-- rework: stage-models.md:291/352 effort-ceiling claim, 288-290 role wording, upgrade-note placement (review cycle 2) -->
- [x] T004 [P] Update `src/kit/skills/_cli-fab.md` (two sparse claims, two `effort=xhigh` examples) and `src/kit/skills/_cli-agents.md` (six roles, catalog cache recipe, flat-alias reach) <!-- R5 --> <!-- rework: _cli-fab.md:349 `sparsely` still matched the R5 grep; _cli-agents.md:187 must enumerate the six roles (review cycle 1) -->
- [x] T005 Run the sibling-sweep greps, `gofmt -l`, scoped then full `go test` from `src/go/fab`, build the local binary, and verify the six codex roles plus `config explain providers` <!-- R6 --> <!-- rework: sweep missed config.md/architecture.md/config_test.go:899 (review cycle 1) -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` codex block has exactly six rows, each with model and effort, matching the target table; no `ultra`
- [x] A-002 R2: `TestNonClaudeProviderFillsArePinned` pins the six rows; a density assertion exists for codex in `defaults_test.go`
- [x] A-003 R3: Go comments and the rendered `providers:` prose call agy sparse and codex dense
- [x] A-004 R4: `stage-models.md` sample, prose, consequence bullet, why-paragraph, upgrade note, drift guard, and lineage are updated; glossary and architecture no longer call codex fills sparse
- [x] A-005 R5: `_cli-fab.md` and `_cli-agents.md` updated as specified

### Behavioral Correctness

- [x] A-006 R1: Local binary resolves `default`/`doing` → astra@high, `review` → sol@xhigh, `hydrate` → sol@high, `operator` → luna@medium, `fast` → luna@low under `--provider codex`
- [x] A-007 R1: Claude, agy, kimi resolution unchanged (pin tests for claude/agy/kimi untouched and passing)

### Scenario Coverage

- [x] A-008 R2: Scoped and full `go test` pass from `src/go/fab`
- [x] A-009 R3: `fab config explain providers` (local binary) shows six codex rows in role order

### Edge Cases & Error Handling

- [x] A-010 R2: `legacyV2198ToV2220ProvidersAdvert` fixture and `src/kit/migrations/2.16.19-to-2.17.0.md` are unedited
- [x] A-011 R4: No literal codex model ID appears under `docs/specs/`

### Code Quality

- [x] A-012 Pattern consistency: new test assertion mirrors the existing claude density check's style
- [x] A-013 No unnecessary duplication: values live only in `defaults.yaml` plus the pin table
- [x] A-014 Canonical source only: no edits under `.claude/skills/`
- [x] A-015 Owner-or-pointer: no skill or spec restates the codex table; role-relative prose points at `fab config explain`
- [x] A-016 Sibling sweep: every codex-sparse / effort-only / inherits-default claim outside the exclusions is gone (R6)
- [x] A-017 Go changes ship tests; `gofmt -l` clean

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change refreshes data, tests, and their documentation without making existing code redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Five tasks: data, tests+Go prose, specs, skills, verify — the light-lane fork applies | Each task is one focused file group; the intake enumerates exactly these surfaces | S:90 R:90 A:95 D:90 |
| 2 | Confident | The `_cli-fab.md` `providers:`-block paragraph (line ~421, "codex's and agy's are **sparse**") is in the sweep class though the intake cited only line 349 | Sweep grep found it; same claim, same file | S:60 R:90 A:90 D:85 |
| 3 | Confident | Memory-file hits from the sweep are listed for hydrate, not edited at apply | Hydrate owns `docs/memory/`; the intake's Affected Memory already names them | S:70 R:85 A:90 D:85 |
| 4 | Certain | The two flat-alias regression tests (`TestFlatProviderFillBeatsBuiltInDefault`, `TestResolveAgentOverrideProviderTakesFill`) are retargeted from `hydrate`/`apply` to `intake` (→ `default`), and each gains an assertion that a built-in role fill now outranks the flat alias on apply/hydrate | Constitution VII: tests conform to the spec, which says a built-in role fill outranks the folded alias; with a dense codex map `default` is the only role the alias still reaches — exactly the upgrade note this change documents | S:85 R:95 A:90 D:90 |
| 5 | Certain | The reworded rendered `providers:` advert changes the system-scaffold paragraph digest, so its sha256 is appended to `knownGeneratedSystemParagraphDigests` in `internal/configupgrade/configupgrade.go` (append-only catalog) | `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` is the self-enforcing guard the catalog's own comment describes; older digests stay | S:90 R:95 A:95 D:95 |
| 6 | Confident | The `providers` registry-row `Description` string in `configref.go` ("codex and agy sparse maps…") is in the sweep class and reworded | User-facing string literal carrying the stale sparse claim (fab-recurring-lessons: sweeps include string literals) | S:70 R:90 A:90 D:85 |

6 assumptions (3 certain, 3 confident).
