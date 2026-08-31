# Plan: Config Upgrade — System Scaffold Regeneration

**Change**: 260830-m4ai-config-upgrade-system-scaffold
**Intake**: `intake.md`

## Requirements

### CLI: `fab config upgrade` Modes

#### R1: `--system` Selects the System Config
`fab config upgrade` SHALL accept a `--system` flag that targets `~/.fab-kit/config.yaml`
instead of `fab/project/config.yaml`. Bare `fab config upgrade` SHALL retain its current
project-only behavior unchanged. `--project` SHALL remain accepted as the explicit opposite.
Passing `--system` together with `--project` SHALL be a usage error.

- **GIVEN** a machine with an existing `~/.fab-kit/config.yaml`
- **WHEN** the user runs `fab config upgrade --system`
- **THEN** `~/.fab-kit/config.yaml` is reconciled against the field registry
- **AND** `fab/project/config.yaml` is left untouched

#### R2: The System Path Requires No Fab Repo
`fab config upgrade --system` SHALL NOT require a fab project and SHALL NOT call
`resolve.FabRoot()`. This matches the sibling verbs `config init --system`, `config set --system`,
and `config unset --system`, all of which already operate outside a repo (verified 2026-08-30).

- **GIVEN** a working directory outside any fab project
- **WHEN** the user runs `fab config upgrade --system`
- **THEN** the reconciliation succeeds
- **AND** no `ERROR: fab/ directory not found` is emitted
- **AND** bare `fab config upgrade` in that same directory still fails closed with that error

#### R3: `--check` Composes With Every Mode
`--check` SHALL compose with `--system` and `--all`, computing the identical reconciliation via
the shared `computeUpgrade` path and writing nothing. It SHALL exit non-zero on drift and zero
when clean.

- **GIVEN** a drifted `~/.fab-kit/config.yaml`
- **WHEN** the user runs `fab config upgrade --system --check`
- **THEN** the drift is reported and the exit code is non-zero
- **AND** the file on disk is byte-identical to before the command

#### R4: `--all` Reconciles Both Layers
`fab config upgrade --all` SHALL reconcile the project and system layers in one invocation,
reporting each on its own labelled line. With `--check` it SHALL exit non-zero when **either**
layer has drifted, and zero only when both are clean. `--all` includes the project layer and
therefore SHALL require a fab repo. `--all` combined with `--system` or `--project` SHALL be a
usage error.

- **GIVEN** a fab repo with a clean project config and a drifted system config
- **WHEN** the user runs `fab config upgrade --check --all`
- **THEN** both layers are reported on separate labelled lines
- **AND** the exit code is non-zero because the system layer drifted

### Engine: Fence & Field Filtering

#### R5: The System Layer Carries a Managed Fence
The regenerated `~/.fab-kit/config.yaml` SHALL delimit its generated reference region with the
same BEGIN/END anchor format the project layer uses (`fenceWidth = 76` dash padding, matched by
the existing `beginLineRe`). The fence body's explanatory preamble SHALL name the `--system`
form of the command rather than the bare project form.

- **GIVEN** an unfenced `~/.fab-kit/config.yaml`
- **WHEN** `fab config upgrade --system` runs
- **THEN** the output contains a BEGIN anchor stamped with the current kit version and a matching END anchor
- **AND** the text between them instructs the reader to run `fab config upgrade --system`

#### R6: Only System-Overridable Fields Enter the System Fence
The system fence SHALL contain only registry fields whose scope is `system` or `both`, matching
`renderSystemScaffold`'s existing predicate. A `scope: project` field SHALL NOT appear.

- **GIVEN** the registry contains the `scope: project` field `stage_hooks`
- **WHEN** `fab config upgrade --system` renders the fence
- **THEN** no `stage_hooks` block appears anywhere in `~/.fab-kit/config.yaml`
- **AND** `dispatch.mode` (`scope: both`) does appear

#### R7: The System File Keeps Its Own Header
The regenerated file SHALL retain `configupgrade.SystemScaffoldHeader` as its preamble, not the
project `referenceHeader`. The header's sentence naming `fab config init --system` as the
generator SHALL be reworded to name both doors, since upgrade now also writes it.

- **GIVEN** a regenerated system config
- **WHEN** the user reads its first lines
- **THEN** the system-layer header is present, including the "Resolves ABOVE any project's" precedence note
- **AND** the generator sentence names both `fab config init --system` and `fab config upgrade --system`

### Engine: Adoption & Preservation

#### R8: Legacy Unfenced Files Are Adopted Without Duplication
On the first `--system` run against an unfenced file, the reconciliation SHALL recognize and
discard the previously generated scaffold region (the `SystemScaffoldHeader` plus the
`CommentOutSegment(ShortSegment)` blocks) and replace it with the fenced region, rather than
appending a fence beneath the stale blocks. The result SHALL NOT contain duplicated field
adverts.

- **GIVEN** a `~/.fab-kit/config.yaml` generated by an older kit, with no fence
- **WHEN** `fab config upgrade --system` runs
- **THEN** the file contains exactly one advert block per system-overridable field
- **AND** exactly one fence BEGIN anchor and one END anchor

#### R9: Live Keys Are Preserved Verbatim and Hoisted Above the Fence
Adoption SHALL preserve every live key and its value byte-exactly, including user comments
attached to those keys, and SHALL place them **above** the fence per project convention. The
effective configuration MUST be unchanged by the reconciliation.

- **GIVEN** a system config whose live keys sit below the comment scaffold
- **WHEN** `fab config upgrade --system` runs
- **THEN** those keys appear above the fence with identical values
- **AND** `fab config show` produces byte-identical output before and after the run

#### R10: Unrecognized Comments Are Preserved, Never Discarded
A comment line that is not part of the recognized generated scaffold SHALL be preserved above
the fence. Losing a user's own note is a worse failure than leaving a stale duplicate.

- **GIVEN** a system config containing a hand-written comment such as `# work laptop only`
- **WHEN** `fab config upgrade --system` runs
- **THEN** that comment still appears in the file

#### R10a: Line-Complete Accounting Governs Every Discard (precedence rule)
R8 (discard the generated scaffold) and R10 (preserve unknown comments) are in direct tension
whenever a comment block only partly resembles generated output. This requirement fixes the
precedence, so the tie is never broken by implementer judgment.

A comment paragraph MAY be discarded **only when every line in it is individually accounted
for** as generated output — matched against a rendering the registry can produce, for this or
any historical kit version. If **any single line** in the paragraph is unaccounted for, the
**entire paragraph SHALL be preserved** as user content, even when the remaining lines match a
generated advert exactly and even when this leaves a visible stale duplicate below the fence.

Duplication is the accepted cost of this rule. Duplication is visible and a user can delete it;
a silently deleted note is unrecoverable and the user may never know it existed. Structural
resemblance alone SHALL NOT authorize a discard.

- **GIVEN** a legacy advert with a user's note appended directly beneath it and no blank-line separator
- **WHEN** `fab config upgrade --system` runs
- **THEN** the whole paragraph — note and advert together — is preserved above the fence
- **AND** the regenerated fence's own copy of that advert may appear as a duplicate

- **GIVEN** a user-edited copy of a generated advert (correct scope suffix and `# Full prose:` pointer, but a hand-modified value)
- **WHEN** `fab config upgrade --system` runs
- **THEN** the edited copy is preserved as user content, because its modified line is unaccounted for

- **GIVEN** a paragraph in which every line matches a historical generated rendering
- **WHEN** `fab config upgrade --system` runs
- **THEN** that paragraph is discarded and replaced by the regenerated fence

- **AND** this rule SHALL hold identically for unfenced adoption, later `--system` upgrades, and `set --system` / `unset --system`

#### R11: Removed Keys Are Parked, Not Deleted
A live key no longer present in the registry SHALL be parked in a comment block below the
fence with its value serialized, exactly as the project layer does, and SHALL NOT be silently
deleted. Parked blocks SHALL be appended once and never regenerated away.

- **GIVEN** a system config containing a live key absent from the registry
- **WHEN** `fab config upgrade --system` runs
- **THEN** the key appears in a parked comment block below the fence with its value
- **AND** a second run leaves that parked block byte-identical

#### R12: Idempotence and Byte-Stability
Running `fab config upgrade --system` twice SHALL produce byte-identical output, and a run
against an already-current file SHALL report no change.

- **GIVEN** a freshly upgraded system config
- **WHEN** `fab config upgrade --system` runs again
- **THEN** the file is byte-identical and the command reports "already up to date"

### Writers: Fence Awareness

#### R13: `set --system` / `unset --system` Respect the Fence
`SetSystem` and `UnsetSystem` SHALL write live keys above the fence and leave the fence intact,
neither duplicating nor corrupting it, by routing through the same fence-aware render path the
project `Set`/`Unset` use, parameterized with the system field filter and header.

- **GIVEN** a fenced `~/.fab-kit/config.yaml`
- **WHEN** the user runs `fab config set --system agent.workers codex`
- **THEN** the key is written above the fence
- **AND** the fence anchors remain present exactly once each
- **AND** a subsequent `fab config upgrade --system --check` reports no drift

### Wizard

#### R14: `fab setup` Self-Heals the System Scaffold
`runSetupWizard` SHALL run the `upgrade --system` reconciliation on entry, before the scope
banner and first question. It SHALL print nothing when the reconciliation reports no change,
print exactly one advisory line when it does change the file, and SHALL NOT abort the wizard on
a reconciliation error (warn and continue). It SHALL NOT prompt.

- **GIVEN** a stale system config and a user running `fab setup`
- **WHEN** the wizard starts
- **THEN** one advisory line reports the refresh and the interview proceeds normally
- **AND GIVEN** an already-current system config
- **WHEN** the wizard starts
- **THEN** no refresh output appears and the all-Enter zero-write invariant still holds

### Documentation

#### R15: CLI Reference and Spec Are Updated
Per the constitution's Additional Constraints, `src/kit/skills/_cli-fab.md` SHALL document the
new `--system` and `--all` modes, and `docs/specs/config.md` SHALL describe them within the
six-verb surface.

- **GIVEN** the change is complete
- **WHEN** a reader consults `_cli-fab.md` for `fab config upgrade`
- **THEN** the `--system`, `--all`, and `--check` composition are documented with their repo requirements

### Non-Goals

- Adding the two commented examples the originating conversation started from — `agent.profiles.operator.provider` and `providers.claude.profiles.default` already ship in 2.23.7; the defect is the missing regeneration path, not missing content.
- A `src/kit/migrations/` file — see Design Decisions.
- Changing bare `fab config upgrade` or bare `--check` semantics.
- Extending the wizard beyond the entry warm-up (no new questions, no pane/trust seeding).
- A both-layer mode for any verb other than `upgrade`.

### Design Decisions

#### Managed Fence Over Header-Anchored Region
**Decision**: Delimit the system layer's regenerable region with the same BEGIN/END anchors the project layer uses, rather than treating the contiguous comment block after the header as implicitly owned.
**Why**: The fence is an explicit, machine-recognizable boundary, so fab can distinguish its own generated prose from a user's hand-written note. It also reuses `renderFence`, `beginLineRe`, and the parking machinery wholesale instead of forking a second delimitation scheme.
**Rejected**: A header-anchored comment region (cannot tell user prose from generated prose — silently eats hand-written notes); a whole-file rebuild (discards all user comments unconditionally).
*Introduced by*: 260830-m4ai-config-upgrade-system-scaffold

#### Live Keys Hoist Above the Fence on Adoption
**Decision**: Adoption reorders existing live keys above the fence, matching `fab/project/config.yaml` convention, accepting a one-time visible reshuffle of the user's file.
**Why**: One mental model across both config layers. The project engine already hoists non-parked content above the fence, so this is the existing behavior rather than new logic.
**Rejected**: Preserving the current below-comments position — the two layers would then read differently forever, and `set --system` would keep its divergent append behavior.
*Introduced by*: 260830-m4ai-config-upgrade-system-scaffold

#### No Migration File for Machine-Level Config
**Decision**: Ship the adoption path inside `upgrade --system` itself, with no `src/kit/migrations/` file, despite `fab/project/code-quality.md` requiring a migration for "restructuring existing user data".
**Why**: Migrations are applied per project by `/fab-setup migrations`. `~/.fab-kit/config.yaml` is machine-level and shared across every project on the host, so a per-project migration would run N times against one file — wrong cardinality and non-idempotent in intent. The migration rule is scoped to project-local user data.
**Rejected**: A migration file (wrong vehicle for machine-level state); a standalone one-shot repair subcommand (a second mechanism to keep in sync with the reconciliation engine — exactly the drift `--check` exists to catch).
*Introduced by*: 260830-m4ai-config-upgrade-system-scaffold

#### `--all` Rather Than Repurposing Bare `--check`
**Decision**: Spell the both-layer check as `fab config upgrade --check --all`, leaving bare `--check` project-only.
**Why**: Bare `--check` is an established CI probe; changing its scope would silently alter the verdict of every existing invocation — a drifted system config would begin failing project CI that never asked about it. `--all` is additive and non-breaking.
**Rejected**: Making bare `--check` cover both layers (breaking, and couples repo CI to machine state).
*Introduced by*: 260830-m4ai-config-upgrade-system-scaffold

## Tasks

### Phase 1: Engine Parameterization

- [x] T001 Extract the system-scope field predicate out of `renderSystemScaffold` in `src/go/fab/cmd/fab/config.go` into a shared, testable filter (colocate with `configref` or `configupgrade`) so both `init --system` and the new upgrade path use one definition <!-- R6 -->
- [x] T002 Introduce a target descriptor in `src/go/fab/internal/configupgrade/` carrying {path, header, field filter, fence-preamble text} and thread it through `Upgrade`, `Check`, and `computeUpgrade`, keeping the existing project behavior byte-identical as the default target <!-- R1 -->
- [x] T003 Parameterize `fenceHeaderComment` (`configupgrade.go:92`) so the system target renders "REGENERATED by `fab config upgrade --system`" without duplicating the constant <!-- R5 -->
- [x] T004 Reword `SystemScaffoldHeader` (`mutation.go:26`) to name both `fab config init --system` and `fab config upgrade --system` as generators <!-- R7 -->

### Phase 2: System Reconciliation

- [x] T005 Implement the system target's render path: system header, system-scoped fence, live keys hoisted above the fence, reusing `render`/`renderFence` <!-- R5 R6 R7 R9 -->
- [x] T006 Implement legacy-scaffold recognition and discard under R10a's line-complete accounting rule: discard a comment paragraph ONLY when every line in it is accounted for as generated output (this or any historical version); any unaccounted line preserves the WHOLE paragraph <!-- R8 R10 R10a --> <!-- rework cycle 2: cycle 1's structural paragraph match over-corrected — it deletes a user note appended to a legacy advert without a blank separator, and deletes a user-EDITED advert copy. Both reproduced by review. R10a (new) now fixes the precedence: preservation wins every ambiguity, duplication is the accepted cost. --> <!-- rework: recognition uses exact equality against TODAY's SystemScaffoldHeader/ShortSegments, so a scaffold generated by an older kit (e.g. `claude --dangerously-skip-permissions` vs today's `--permission-mode bypassPermissions`) is not recognized, survives as 'user content' above the fence, and is DUPLICATED by the regenerated fence. Confirmed against a real 2.20.x-era config. Recognition must tolerate historical drift in generated prose. -->
- [x] T007 Verify parking works on the system target (removed live key parked below the fence with serialized value, appended once) and fix any target-coupling in the parking path <!-- R11 -->
- [x] T008 Route `SetSystem` and `UnsetSystem` (`mutation.go`) through the fence-aware `renderMutation` with the system target, replacing their fence-free write path <!-- R13 -->

### Phase 3: CLI Surface

- [x] T009 Add `--system`, `--project`, and `--all` flags to `configUpgradeCmd` (`src/go/fab/cmd/fab/config.go:60`), with mutual-exclusion usage errors for the contradictory pairs <!-- R1 R4 -->
- [x] T010 Gate `resolve.FabRoot()` on the target so `--system` runs repo-free while bare and `--all` keep failing closed outside a repo <!-- R2 R4 -->
- [x] T011 Implement `--all`: reconcile both layers, one labelled report line each, non-zero exit when either drifted under `--check`, and an applying counterpart when `--check` is absent <!-- R3 R4 -->
- [x] T012 Update the `configUpgradeCmd` `Short`/`Long` help text to describe all three modes and their differing repo requirements <!-- R1 R15 -->

### Phase 4: Wizard

- [x] T013 Add the `upgrade --system` warm-up to `runSetupWizard` (`src/go/fab/cmd/fab/setup_wizard.go:93`): silent when unchanged, one advisory line when changed, warn-and-continue on error, never prompting <!-- R14 -->

### Phase 5: Tests

- [x] T014 [P] Adoption tests in `config_upgrade_test.go`: unfenced legacy file gains exactly one fence and one advert per field; live keys preserved verbatim and hoisted above the fence; hand-written comments survive <!-- R8 R9 R10 R10a --> <!-- rework cycle 2: add ADVERSARIAL regression coverage for R10a — (a) user note appended to a legacy advert with NO blank separator, (b) user-edited advert copy carrying a valid scope suffix and `# Full prose:` pointer, (c) a fully-accounted historical paragraph that SHOULD still be discarded. Keep the existing multi-version historical fixtures — they pass and must not regress. --> <!-- rework: the fixture at configupgrade_test.go:1071 is synthesized from TODAY's registry, so it structurally cannot catch historical drift. Add a fixture with VERBATIM older-kit scaffold text (hardcoded, not registry-generated) covering the claude/agy command-string rename. -->
- [x] T015 [P] Idempotence and drift tests: two consecutive `--system` runs byte-identical; `--check --system` exit codes on clean and drifted files; `--check` writes nothing <!-- R3 R12 -->
- [x] T016 [P] Scope-filter test: no `scope: project` field (e.g. `stage_hooks`) appears in the system fence; `dispatch.mode` does <!-- R6 -->
- [x] T017 [P] Parking test on the system target: removed live key parked with value, second run leaves the block byte-identical <!-- R11 -->
- [x] T018 [P] No-repo test in `noproject_config_test.go`: `--system` succeeds outside a fab repo; bare and `--all` still fail with `fab/ directory not found` <!-- R2 R4 -->
- [x] T019 [P] Writer tests: `set --system` / `unset --system` on a fenced file write above the fence, leave anchors intact once each, and leave `--check --system` reporting no drift <!-- R13 -->
- [x] T020 [P] Wizard tests in `setup_test.go`: silent when clean, one advisory line when changed, non-fatal on reconciliation error <!-- R14 -->
- [x] T021 `--all` tests: both layers reported on separate lines; non-zero exit when either drifted; contradictory flag pairs rejected <!-- R4 -->

### Phase 6: Documentation

- [x] T022 [P] Update `src/kit/skills/_cli-fab.md` with the `fab config upgrade` `--system` / `--all` / `--check` matrix and repo requirements <!-- R15 --> <!-- rework: _cli-fab.md:902 still promises `fab setup --defaults` and all-Enter runs perform ZERO writes, but the R14 warm-up writes a stale system scaffold before the interview. Qualify the promise to clean-or-missing system configs. -->
- [x] T023 [P] Update `docs/specs/config.md`'s six-verb surface with the upgrade modes and the system fence <!-- R15 -->

## Execution Order

- T002 blocks T005, T009, T010, T011 (the target descriptor is the seam everything else threads through)
- T001 blocks T005 (the filter defines what the system fence contains)
- T005 blocks T006, T007, T008
- T009 blocks T010, T011, T012
- Phase 5 tasks are all `[P]` except T021, which depends on T011
- Phase 6 is independent of Phase 5 and may run alongside it

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab config upgrade --system` reconciles `~/.fab-kit/config.yaml` and leaves the project config untouched; bare `fab config upgrade` behavior is unchanged
- [x] A-002 R2: `--system` succeeds outside a fab repo; bare `upgrade` there still fails with `ERROR: fab/ directory not found`
- [x] A-003 R3: `--check` composes with `--system` and `--all`, writing nothing and exiting non-zero only on drift
- [x] A-004 R4: `--all` reconciles both layers in one invocation with per-layer report lines and either-drifted exit semantics
- [x] A-005 R5: The regenerated system file carries BEGIN/END anchors in the project format, with a preamble naming the `--system` command
- [x] A-006 R6: The system fence contains only `system`/`both` scoped fields
- [x] A-007 R7: The system header is retained and reworded to name both generator doors
- [x] A-008 R14: `fab setup` refreshes the scaffold on entry — silent when clean, one line when changed, never fatal
- [x] A-009 R15: `_cli-fab.md` and `docs/specs/config.md` document the new modes

### Behavioral Correctness

- [x] A-010 R8: Adopting an unfenced legacy file yields exactly one fence and no duplicated adverts
- [x] A-011 R9: Live keys survive adoption byte-exactly and sit above the fence; `fab config show` output is identical before and after
- [x] A-012 R13: `set --system` / `unset --system` write above an existing fence without duplicating or corrupting it

### Scenario Coverage

- [x] A-013 R12: Two consecutive `--system` runs are byte-identical and the second reports "already up to date"
- [x] A-014 R11: A removed live key is parked below the fence with its value and is not regenerated away on a second run
- [x] A-015 R2: The no-repo path is covered by a test in `noproject_config_test.go`

### Edge Cases & Error Handling

- [x] A-016 R10: A hand-written comment in the system config survives adoption
- [x] A-025 R10a: A user note appended to a legacy advert with no blank-line separator survives, together with its paragraph
- [x] A-026 R10a: A user-edited copy of a generated advert survives as user content
- [x] A-027 R10a: A paragraph whose every line is accounted for as generated IS still discarded (the rule does not disable adoption)
- [x] A-028 R10a: The line-complete rule holds identically for `set --system` / `unset --system`, not only for adoption
- [x] A-017 R4: Contradictory flag pairs (`--system --all`, `--system --project`, `--project --all`) are rejected with a usage error
- [x] A-018 R14: A reconciliation failure during the wizard warm-up warns and continues into the interview rather than aborting
- [x] A-019 R5: The existing `validateYAML` refusal still guards the system target — unparseable rendered output leaves the file untouched

### Code Quality

- [x] A-020 Pattern consistency: New code follows the existing `configupgrade` target/render conventions and the surrounding cobra flag idiom
- [x] A-021 No unnecessary duplication: The scaffold renderer, fence renderer, scope filter, and header constant each have exactly one definition shared by `init --system` and `upgrade --system`
- [x] A-022 CLI ⇒ docs + tests: Per the constitution, the command-signature change ships `_cli-fab.md` updates and Go test coverage
- [x] A-023 Owner-or-pointer: No skill file both states a rule and points at its owner; the `--system` documentation lives in one place
- [x] A-024 Migration exemption is recorded: The no-migration decision is captured as a Design Decision so review does not flag it as a missing migration

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Manual smoke check worth running before ship: back up `~/.fab-kit/config.yaml`, run `--system`, confirm `fab config show` diffs clean, then restore.

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | A target descriptor threaded through `computeUpgrade` is the seam, rather than forking a parallel `UpgradeSystem` function | The intake requires one reconciliation engine; a second function is the drift class `--check` exists to catch. `Set`/`SetSystem` already show the fork cost | S:85 R:75 A:85 D:80 |
| 2 | Confident | Legacy-scaffold recognition matches on the rendered `CommentOutSegment(ShortSegment)` blocks rather than a version stamp | Old files carry no stamp — there is no fence — so the generated text itself is the only available signal | S:70 R:65 A:75 D:65 |
| 3 | Confident | The `set --system` indentation defect is fixed only if the fix is local to the system write path being rewritten in T008 | Intake assumption 10 flagged it may root in `setLivePath`; T008 rewrites that path anyway, so a local fix is likely — but a deeper root belongs in its own change and apply should say so rather than expand scope | S:65 R:75 A:70 D:60 |
| 4 | Confident | `--all` without `--check` upgrades both layers rather than erroring | A check mode with no applying counterpart is a trap; symmetry is free here | S:70 R:80 A:80 D:75 |
| 5 | Confident | The wizard warm-up reuses the same engine entry point as the CLI rather than shelling out to `fab config upgrade --system` | In-process reuse keeps one code path and avoids a subprocess in an interactive flow | S:70 R:75 A:85 D:75 |

5 assumptions (1 certain, 4 confident, 0 tentative).
