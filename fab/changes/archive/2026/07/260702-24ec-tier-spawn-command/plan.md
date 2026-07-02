# Plan: Per-Tier spawn_command — Cross-Harness Stage Dispatch Opt-In

**Change**: 260702-24ec-tier-spawn-command
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md § What Changes / Impact. RFC 2119. Every requirement
     carries at least one GIVEN/WHEN/THEN scenario and a stable R# ID. -->

### Config: The widened tier profile

#### R1: `TierProfile`/`Profile` carry an opt-in `spawn_command`
The `TierProfile` struct (`internal/config`) and the resolution-side `Profile` struct
(`internal/agent`) SHALL each grow a `SpawnCommand` string field (`yaml:"spawn_command"` on
`TierProfile`). fab-kit's built-in `defaultTiers` MUST NOT carry any `spawn_command` — the field is
populated **exclusively** from user config (`agent.tiers.<tier>.spawn_command`). A config with no
tier `spawn_command` MUST continue to load and resolve exactly as today.

- **GIVEN** a `fab/project/config.yaml` with `agent.tiers.doing.spawn_command: "codex exec ..."`
- **WHEN** the config is loaded and `GetAgentTier("doing")` is called
- **THEN** the returned `TierProfile.SpawnCommand` equals `"codex exec ..."`
- **AND** a config with no `spawn_command` on any tier loads with an empty `SpawnCommand`, unchanged.

#### R2: `agent.Resolve` per-field-merges `spawn_command`
`agent.Resolve` SHALL extend its existing per-field merge so an override tier's non-empty
`SpawnCommand` wins over the default (which is always empty), following the exact `model`/`effort`
merge pattern (override field set → wins; omitted → inherits default). A tier resolving from a
fab-kit default MUST have an empty `SpawnCommand`.

- **GIVEN** a `doing` tier override that sets **only** `spawn_command` (no model/effort)
- **WHEN** `Resolve(cfg, "apply")` runs (apply ∈ doing)
- **THEN** the resolved `Profile` carries the override's `SpawnCommand` **and** the default model
  (`claude-opus-4-8`) and default effort (`high`) — per-field merge.
- **AND** `Resolve(nil, "apply")` yields a `Profile` with an empty `SpawnCommand`.

### CLI: `fab resolve-agent` third output line

#### R3: `spawn=` line emitted only when the resolved tier carries a `spawn_command`
`fab resolve-agent <stage>` SHALL emit a third stdout line `spawn=<command>` **only when** the
resolved tier's `SpawnCommand` is non-empty, mirroring the existing "`effort=` omitted when empty"
rule. When the resolved tier has no `spawn_command`, output MUST be byte-identical to today (two
lines, no `spawn=`). The emitted command's `{model}`/`{effort}` placeholders MUST be substituted by
**reusing** `internal/spawn`'s existing template resolution (`spawn.WithProfile`) — never a
reimplementation. Output MUST remain byte-stable for the same config.

- **GIVEN** a config with `agent.tiers.doing.spawn_command: "codex exec -m {model} -c model_reasoning_effort={effort}"`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** stdout is `model=claude-opus-4-8\neffort=high\nspawn=codex exec -m claude-opus-4-8 -c model_reasoning_effort=high\n`
- **AND** for a config with no tier `spawn_command`, `fab resolve-agent apply` emits exactly two
  lines (no `spawn=`).

#### R4: `spawn=` always embeds the FULL model ID, even under `--alias`
Under `--alias`, the `model=` line SHALL stay aliased (`opus`/`sonnet`/…) exactly as today, but the
`spawn=` line's `{model}` placeholder MUST be substituted with the **full resolved model ID**, never
the alias. CLI dispatch never aliases (an external CLI's `--model` flag takes a full ID); aliasing is
the Agent-tool-only adaptation.

- **GIVEN** the R3 config
- **WHEN** `fab resolve-agent apply --alias` runs
- **THEN** stdout is `model=opus\neffort=high\nspawn=codex exec -m claude-opus-4-8 -c model_reasoning_effort=high\n`
  (aliased `model=`, full-ID `spawn=`).

#### R5: No cross-fallback to `agent.spawn_command`
The stage-dispatch resolution path MUST NOT read `agent.spawn_command`. The absence of a resolved
tier `spawn_command` is the sole signal for "native Agent-tool dispatch"; there is no fallback from
the tier to the project-wide `agent.spawn_command`. The two fields remain independent surfaces
(session boundary vs. stage dispatch).

- **GIVEN** a config that sets `agent.spawn_command` but NO tier `spawn_command`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** no `spawn=` line is emitted (the project-wide `agent.spawn_command` is never consulted by
  `resolve-agent`).

### Config reference: `fab config reference`

#### R6: The generated reference documents the tier `spawn_command`
The `internal/configref` `agent.tiers` reference block SHALL document the new `spawn_command` field
and its opt-in semantics (present → CLI dispatch; absent → native Agent-tool dispatch; no fallback to
`agent.spawn_command`). Because the reference is generated from constants and defaults carry no
`spawn_command`, the field MUST appear as documented-but-commented guidance, not a shipped default
value. The coverage test suite MUST continue to pass (the reflected `spawn_command` segment is already
required; the new field on `TierProfile` reuses the same segment name).

- **GIVEN** the widened `TierProfile`
- **WHEN** `fab config reference` renders
- **THEN** the output documents `agent.tiers.<tier>.spawn_command` opt-in CLI-dispatch semantics
- **AND** `TestConfigReferenceCoversBinaryKeys` and the round-trip/byte-stable tests still pass.

### Distribution: migration + version bump

#### R7: Ship a config-only, sentinel-guarded, idempotent migration in the correct slot
A new migration file SHALL append a SHORT commented block under an existing repo's
`fab/project/config.yaml` `agent:` section documenting the tier `spawn_command` field and its opt-in
semantics, ending with a pointer to `fab config reference`. It MUST follow the `2.2.0-to-2.3.0`
precedent (comment-sentinel idempotency; skip when config absent or the sentinel is already present;
insert under the `agent:` block). Because 3a (PR #457) already claimed the `2.10.1-to-2.11.0` slot and
bumped VERSION to `2.11.0`, this change MUST use the **next** slot: `2.11.0-to-2.12.0.md`, and bump
`src/kit/VERSION` to `2.12.0` (MINOR, config-additive).

- **GIVEN** a repo whose `config.yaml` has an `agent:` block and no tier-spawn_command sentinel
- **WHEN** the `2.11.0-to-2.12.0` migration is applied
- **THEN** the commented block is inserted under `agent:` and re-running is a complete no-op (sentinel
  trips); a repo with no `config.yaml` is skipped.

### Documentation: mirror-sweep of the resolve-agent contract

#### R8: Every canonical description of the "two-line" resolve-agent contract reflects the optional third line
The full mirror-sweep class SHALL be updated so no canonical source asserts the output is "exactly two
lines". `src/kit/skills/_cli-fab.md` (§ fab resolve-agent: third-line contract, full-ID-under-alias
rule, worked example), `docs/specs/stage-models.md` (§ Config schema, § Resolution, § Harness-adapter
boundary), `docs/specs/skills/SPEC-_cli-fab.md`, `docs/specs/architecture.md`,
`src/kit/skills/_preamble.md` § Per-Stage Model Resolution, `docs/specs/skills/SPEC-_preamble.md`, and
the `resolve_agent.go` doc comment MUST all describe the "two lines plus an optional third `spawn=`
line" contract with the no-cross-fallback semantics. Dispatch-seam skills that only consume
`model=`/`effort=` MUST NOT grow `spawn=` handling (that is 3c/3d) — only contract-describing prose
changes.

- **GIVEN** the intake's grepped mirror-sweep class
- **WHEN** apply finishes
- **THEN** a repo-wide grep for "two byte-stable"/"two stdout lines"/"exactly two" against the
  resolve-agent contract finds no stale claim in a canonical contract-describing source.

### Non-Goals

- The dispatch execution itself (`fab dispatch`, 3c) — this change only *emits* the command.
- Skill dispatch-seam wiring and the result protocol (3d).
- Any validation/quoting-guarantee of the spawn command string — verbatim (post-substitution)
  pass-through per Constitution I and the no-validation principle.
- Touching `docs/memory/` — hydrate is a later stage.

### Design Decisions

1. **Field-by-field merge, not struct embedding** (intake Open Question a): `Resolve` continues to
   copy field-by-field with the default-then-override merge — *Why*: it is today's established pattern
   for model/effort; adding one `if override.SpawnCommand != ""` line is the minimal, consistent
   extension. *Rejected*: struct embedding of `TierProfile` into `Profile` — introduces a new pattern
   for no benefit and would couple the two package structs.
2. **Verbatim `spawn=` output, no shell-quoting guarantee** (intake Open Question b): the `spawn=`
   line is the post-substitution command string emitted verbatim. *Why*: 3c owns execution and any
   quoting it needs; the no-validation principle (Constitution I) says fab does not massage the
   string. *Rejected*: shell-escaping the command at resolve time — premature (no consumer has
   specified a need) and would embed shell-grammar knowledge into a provider-neutral resolver.
3. **Migration slot is `2.11.0-to-2.12.0`, VERSION → `2.12.0`**: 3a (PR #457, in this branch's
   history) already took `2.10.1-to-2.11.0` and bumped VERSION to `2.11.0`. *Why*: the intake's
   Migration-slot note mandates re-confirming the tip and using the actual current version as `from`.
   *Rejected*: reusing/overwriting `2.10.1-to-2.11.0.md` — it belongs to a different, already-shipped
   change (artifact-write hook removal).

## Tasks

### Phase 1: Core widening (config + agent)

- [x] T001 Add `SpawnCommand string \`yaml:"spawn_command"\`` to `TierProfile` in `src/go/fab/internal/config/config.go`; update the struct doc comment to name the new field. <!-- R1 -->
- [x] T002 Add `SpawnCommand string` to `Profile` and extend the per-field merge in `Resolve` (`src/go/fab/internal/agent/agent.go`) with `if override.SpawnCommand != "" { resolved.SpawnCommand = override.SpawnCommand }`; update the `Profile`/`Resolve` doc comments. `defaultTiers` stays `{model, effort}` only. <!-- R2, R5 -->

### Phase 2: Resolve-agent output

- [x] T003 In `src/go/fab/cmd/fab/resolve_agent.go`: extend `formatAgentProfile` (or the command) to emit `spawn=<command>` only when the resolved tier's `SpawnCommand` is non-empty, with `{model}`/`{effort}` substituted via `spawn.WithProfile(profile.SpawnCommand, fullModelID, effort)`. The full model ID (pre-alias) MUST feed the substitution even under `--alias`; the `model=` line uses the aliased value. Update the command's doc comment (two lines → optional third line). <!-- R3, R4 -->

### Phase 3: Config reference

- [x] T004 Update the `agent.tiers` reference block in `src/go/fab/internal/configref/configref.go` to document the `spawn_command` field and its present=CLI / absent=native / no-fallback semantics (commented guidance, no default value). <!-- R6 -->

### Phase 4: Distribution (migration + version)

- [x] T005 Create `src/kit/migrations/2.11.0-to-2.12.0.md` — a config-only, sentinel-guarded, idempotent migration appending a SHORT commented block under `agent:` documenting the tier `spawn_command` opt-in, ending with a pointer to `fab config reference`, per the `2.2.0-to-2.3.0` precedent. <!-- R7 -->
- [x] T006 Bump `src/kit/VERSION` to `2.12.0`. <!-- R7 -->

### Phase 5: Documentation mirror-sweep

- [x] T007 Update `src/kit/skills/_cli-fab.md` § fab resolve-agent — document the third `spawn=` line (present-only-when-tier-has-spawn_command), the full-ID-under-`--alias` rule, the no-cross-fallback semantics, and an updated worked example. <!-- R8 -->
- [x] T008 [P] Update `docs/specs/stage-models.md` — § Config schema (`agent.tiers` gains `spawn_command`), § Resolution (optional third `spawn=` line + omit-when-absent + no-cross-fallback), § Harness-adapter boundary (`spawn=` is the CLI-dispatch adapter, never aliases). <!-- R8 -->
- [x] T009 [P] Update `docs/specs/skills/SPEC-_cli-fab.md` — the fab resolve-agent inventory row reflects the third `spawn=` line. <!-- R8 -->
- [x] T010 [P] Update `docs/specs/architecture.md` — the `agent.tiers` config block documents the `spawn_command` field. <!-- R8 -->
- [x] T011 [P] Update `src/kit/skills/_preamble.md` § Per-Stage Model Resolution — "two byte-stable stdout lines" softens to include the optional third `spawn=` line. <!-- R8 -->
- [x] T012 [P] Update `docs/specs/skills/SPEC-_preamble.md` — the § Per-Stage Model Resolution summary reflects the optional third `spawn=` line. <!-- R8 -->
- [x] T013 [P] Update the `resolve_agent.go` doc comment — "two stdout lines" → optional third `spawn=` line. <!-- R8 -->

### Phase 6: Tests

- [x] T014 Config parse tests in `src/go/fab/internal/config/config_test.go`: a tier `spawn_command` round-trips via `GetAgentTier`; a tier without it yields an empty `SpawnCommand`. <!-- R1 -->
- [x] T015 Agent merge tests in `src/go/fab/internal/agent/agent_test.go`: tier with `spawn_command` → resolved `Profile` carries it; tier without → empty; per-field merge with only `spawn_command` set keeps default model/effort; default resolve carries empty `SpawnCommand`. <!-- R2, R5 -->
- [x] T016 `resolve_agent_test.go` cases: no-tier-spawn (two lines unchanged); tier-spawn present (three lines, substituted); `--alias` with tier-spawn (aliased `model=`, full-ID `spawn=`); byte-stability; empty-value token-drop inherited from `spawn`. <!-- R3, R4, R5 -->
- [x] T017 Confirm `internal/configref` coverage + round-trip + byte-stable tests still pass after the reference edit (no new test needed unless a `spawn_command`-in-tiers assertion adds value; add one asserting the tiers block mentions `spawn_command`). <!-- R6 -->

## Execution Order

- T001 → T002 (agent merge depends on the config field existing conceptually, though structs are
  independent packages; keep the order for clarity).
- T002 → T003 (resolve-agent reads the resolved `Profile.SpawnCommand`).
- T001/T002/T003 → T014/T015/T016 (tests follow their code).
- Phase 5 (T007–T013) is documentation-only and independent of the Go build; all `[P]`.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `TierProfile` and `agent.Profile` carry a `SpawnCommand` field; `defaultTiers` carry none; a config's tier `spawn_command` round-trips through `GetAgentTier`.
- [x] A-002 R2: `agent.Resolve` per-field-merges `spawn_command` (override wins when set; default resolves empty); a `spawn_command`-only override keeps default model/effort.
- [x] A-003 R3: `fab resolve-agent <stage>` emits a `spawn=` line only when the resolved tier carries a `spawn_command`, with `{model}`/`{effort}` substituted via `internal/spawn` (reused, not reimplemented); byte-stable.
- [x] A-004 R4: under `--alias`, `model=` is aliased while `spawn=` embeds the full model ID.
- [x] A-005 R6: `fab config reference` documents `agent.tiers.<tier>.spawn_command` opt-in semantics as commented guidance.
- [x] A-006 R7: a `2.11.0-to-2.12.0.md` migration exists (config-only, sentinel-guarded, idempotent, `2.2.0-to-2.3.0` shape) and `src/kit/VERSION` is `2.12.0`.
- [x] A-007 R8: no canonical contract-describing source still asserts the resolve-agent output is exactly two lines; the third `spawn=` line and no-cross-fallback semantics are documented across the mirror-sweep class.

### Behavioral Correctness

- [x] A-008 R5: `fab resolve-agent` never consults `agent.spawn_command`; absence of a resolved tier `spawn_command` yields no `spawn=` line (native dispatch).
- [x] A-009 R3: for a config with no tier `spawn_command`, `fab resolve-agent apply` output is byte-identical to today (two lines).

### Scenario Coverage

- [x] A-010 R3 R4: `resolve_agent_test.go` exercises no-spawn, spawn-present, and `--alias`+spawn cases with exact-byte assertions.
- [x] A-011 R2: `agent_test.go` exercises the `spawn_command`-only per-field merge and the empty-default cases.

### Edge Cases & Error Handling

- [x] A-012 R3: an empty `{model}`/`{effort}` substitution in a templated `spawn_command` inherits `internal/spawn`'s token-drop behavior (not reimplemented) — verified by a test case. <!-- Met via reuse: resolve_agent.go:71 delegates to spawn.WithProfile, and TestResolveAgentSpawnSubstitutionReusesSpawnPackage exercises the seam (documenting that the empty-value token-drop OUTCOME is unreachable through resolve-agent — an empty override is a no-op merge that keeps the default, so a RESOLVED model/effort is never empty with today's defaults — while spawn's own token-drop path stays unit-tested in spawn_test.go). The inheritance of behavior, not a duplicate token-drop test, is the acceptance target and it holds. -->


### Code Quality

- [x] A-013 Pattern consistency: new code follows the existing `model`/`effort` per-field-merge and omit-when-empty patterns in `agent.Resolve` and `formatAgentProfile`.
- [x] A-014 No unnecessary duplication: `{model}`/`{effort}` substitution reuses `spawn.WithProfile` — no second copy of the template/token-drop logic (Constitution I; code-quality anti-pattern).
- [x] A-015 Migrations for user-data restructuring: the config-doc block ships as a `src/kit/migrations/` file, not an ad-hoc script.
- [x] A-016 CLI ⇒ docs + tests: the `fab resolve-agent` output change updates `src/kit/skills/_cli-fab.md` and ships Go test updates.
- [x] A-017 Canonical source only: no edits under `.claude/skills/`; kit changes live in `src/kit/`.
- [x] A-018 Go changes ship tests: config/agent/resolve-agent packages carry the new test cases in the same change.

### documentation_accuracy

- [x] A-019 R8: the `_cli-fab.md` worked example and the stage-models.md worked example show the aliased-`model=` / full-ID-`spawn=` output correctly; the migration slot/version numbers (`2.11.0-to-2.12.0`, `2.12.0`) are accurate against the branch's actual VERSION.

### cross_references

- [x] A-020 R8: the SPEC mirrors (`SPEC-_cli-fab.md`, `SPEC-_preamble.md`) stay in sync with their skill sources; `stage-models.md`/`architecture.md`/`_preamble.md` describe the same three-line contract with no residual "two lines" claim in the sweep class.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. The `SpawnCommand` field, its per-field-merge branch, and the `spawn=` emission are pure additive extensions of the existing `{model, effort}` tier surface; no prior symbol, branch, or config key is superseded. (The 3a shim / orphaned hooklib functions noted in memory belong to a different change's next-release cleanup, not this one.)

## Assumptions

<!-- Graded SRAD decisions made while co-generating this plan. Three grades only. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Field-by-field merge in `Resolve` (add `if override.SpawnCommand != ""`), NOT struct embedding — resolves intake Open Question (a). | Today's established `model`/`effort` merge pattern; one-line extension; embedding introduces a new pattern for no benefit and couples the two package structs. Directed by the task instructions. | S:90 R:85 A:95 D:90 |
| 2 | Certain | `spawn=` emitted verbatim (post-substitution), no shell-quoting/escaping guarantee — resolves intake Open Question (b). | No-validation principle (Constitution I); 3c owns execution + any quoting it needs. Low blast radius (a later formatting refinement 3c can request). Directed by the task instructions (lean verbatim). | S:80 R:85 A:85 D:75 |
| 3 | Certain | Migration slot is `2.11.0-to-2.12.0` and VERSION bumps to `2.12.0` (NOT `2.10.1-to-2.11.0`/`2.11.0`). | 3a (PR #457) is in this branch's git history: it already created `2.10.1-to-2.11.0.md` and bumped VERSION to `2.11.0`. The intake's Migration-slot note mandates using the actual current version as `from` and the next MINOR as `to` when the tip moved. Verified via git log + VERSION read. | S:95 R:80 A:95 D:95 |
| 4 | Confident | Feed the FULL (pre-alias) model ID into `spawn.WithProfile` in `resolve_agent.go` by substituting from `profile.Model` captured BEFORE the `--alias` transform overwrites it. | Intake assumption #4 + § 3 worked example are explicit: `spawn=` always embeds the full ID; `model=` stays aliased. Implementation detail (capture-before-alias) is the natural way to preserve both; Confident because the exact code shape is authored at apply. | S:85 R:80 A:90 D:85 |
| 5 | Confident | The `internal/configref` coverage test does not newly fail (the reflected `spawn_command` segment already exists via `AgentConfig.SpawnCommand`); add one small assertion that the tiers block mentions `spawn_command` for documentation_accuracy coverage. | Reflection over `Config` collects leaf segment names; `spawn_command` is already required. The new `TierProfile.SpawnCommand` reuses that segment name, so no new coverage gap — but an explicit tiers-block assertion guards the intended documentation. Confident because it depends on the exact rendered wording authored at apply. | S:80 R:85 A:85 D:75 |
| 6 | Confident | Exact wording of the migration's commented block (intake assumption #10, Tentative there) — settled to a short 5-6 line block mirroring the illustrative shape in intake § 5, ending "See: fab config reference." | User said "short" + "pointer to reference" + named the `2.2.0-to-2.3.0` precedent; multiple acceptable phrasings exist but the intake supplies a near-final draft. Graded up to Confident here because apply is the authoring point and the draft is adopted nearly verbatim. | S:70 R:85 A:75 D:70 |
| 7 | Confident | Mirror-sweep class is exactly: `_cli-fab.md`, `stage-models.md`, `SPEC-_cli-fab.md`, `architecture.md`, `_preamble.md`, `SPEC-_preamble.md`, and the `resolve_agent.go` doc comment. Dispatch-seam skills consuming only model=/effort= are NOT edited (3c/3d own spawn= handling). | Grep for "two byte-stable"/"two stdout lines"/"exactly two" + "resolve-agent" across canonical sources (src/kit, docs/specs) enumerated the class; `docs/memory/` excluded (hydrate stage); `docs/specs/findings/*` and `*/log*.md` excluded (not contract-describing prose). | S:85 R:80 A:85 D:80 |

7 assumptions (3 certain, 4 confident, 0 tentative).
