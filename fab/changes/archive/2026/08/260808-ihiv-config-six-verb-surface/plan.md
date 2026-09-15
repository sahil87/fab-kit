# Plan: Six-Verb `fab config` Surface

**Change**: 260808-ihiv-config-six-verb-surface
**Intake**: `intake.md`

## Requirements

### CLI: Intent-Grouped Six-Verb Surface

#### R1: Six Visible Verbs and Compatibility Alias

The `fab config` command group MUST expose exactly the visible day-one verbs `show`, `explain`, `set`, `unset`, `init`, and `upgrade`, grouped in help as inspect, modify, and lifecycle operations. `reference` SHALL remain functional only as an invisible Cobra alias of `explain`; no `edit`, `fix`, `doctor`, or `validate` verb SHALL be introduced.

- **GIVEN** the live Cobra tree
- **WHEN** a user reads `fab config --help`
- **THEN** the six intent-grouped verbs are visible and `reference` is absent
- **AND** invoking `fab config reference` produces the same result as the corresponding `explain` invocation

#### R2: Effective Single-Key Inspection

`fab config show` MUST retain its existing bare behavior and SHALL accept at most one dotted key. The keyed form MUST compose built-in defaults, print scalar/list values without a key prefix, print map-valued results as YAML, support `--origin`, and return a non-zero error naming an unknown key.

- **GIVEN** project, system, and built-in config layers
- **WHEN** `fab config show <dotted.key>` runs
- **THEN** it prints the effective post-cascade value for only that key
- **AND** `--origin` identifies the winning layer without changing bare `show` behavior

#### R3: Per-Key Schema Explanation

`fab config explain [<key>] [--json]` MUST render the registry documentation currently produced by `reference`. A key whose registry row owns a segment SHALL select that segment; a key represented inside another row's segment SHALL resolve to the owning segment. JSON key lookup MUST return the rows represented by that owning segment in the existing field-object shape.

- **GIVEN** an exact registry key or a supported dotted descendant of a map-valued registry row
- **WHEN** `fab config explain <key>` runs
- **THEN** only the owning rendered segment is emitted
- **AND** the JSON form emits only the matching owning-segment row set in deterministic table order

### Config Engine: Validated Surgical Mutation

#### R4: Registry-Key and YAML-Value Validation

The registry MUST provide a single key-resolution and expected-value-kind signal for exact rows and supported deep paths, including `agent.profiles.<role>.{provider,model,effort}`, `providers.<name>.{session_command,dispatch_command}`, `providers.<name>.profiles.<role>.{model,effort}`, and `stage_hooks.<stage>.{pre,post}`. **`set` is a scalar-only, leaf-only convenience function** (user-confirmed 2026-08-09; supersedes the intake's flow-collection line): `<key>` MUST name a scalar-valued leaf and `<value>` MUST parse to a single-line, comment-free YAML scalar (string/bool/int/float). `set` MUST reject unknown or quoted keys, scalar values whose kind does not match the registry signal, map/sequence values of any spelling, multi-line or comment-bearing values, and map- or sequence-valued leaves (e.g. `source_paths`) — each refusal names the reason and points to `fab config explain` or to editing `fab/project/config.yaml` directly. The shared value parser retains flow-collection support for the environment layer, which writes no file; `set` applies the stricter gate at its call site.

- **GIVEN** a plausible-looking typo such as `agent.profile.review.model` or `dispatch.colum_width`, a quoted key, a map/sequence value in any spelling (`agent '{workers: codex}'`), a multi-line or comment-bearing value, or a list-valued leaf like `source_paths`
- **WHEN** a set operation validates the request
- **THEN** it refuses before any file write, naming the invalid key, the expected scalar kind, or the edit-the-file-manually fallback
- **AND** valid single-line scalar values parse through the one shared helper, whose flow-collection mode remains available to the env layer only

#### R5: Comment-Preserving Project Set

`fab config set <key> <value>` MUST write through `internal/configupgrade` without a whole-document YAML marshal. It SHALL replace an existing live leaf surgically, materialize a missing live path above the managed fence **rendering the full ancestor chain — exactly the requested leaf and its real ancestors, never a phantom sibling or ancestor key**, preserve sibling keys and user comments, regenerate the fence from the same registry segments used by upgrade, and be byte-identical when the same value is set twice. The managed-fence anchor text MUST remain unchanged.

- **GIVEN** a live map block with sibling leaves, comments above/inside/inline, and a managed fence
- **WHEN** a nested leaf is set or reset to the same value
- **THEN** only the requested leaf changes, every sibling and user comment survives, the fence advertises only non-live surfaces, and the second write is byte-identical

#### R6: Idempotent Project Unset

`fab config unset <key>` MUST remove the live override through `internal/configupgrade`, preserve unrelated content and comments, prune empty structural parents where safe, and regenerate the managed fence so the removed override is advertised again. A known but already-unset key SHALL exit zero and report a no-op. Unset MUST NOT validate the malformed live value's type.

- **GIVEN** either a live known override with an invalid value or a known key that is not live
- **WHEN** the key is unset
- **THEN** the live override is removed regardless of its value type, or the operation reports an exit-zero no-op
- **AND** inheritance/default behavior is restored without comment loss

#### R7: Scope-Aware System Mutation

`set` and `unset` MUST accept `--system`, target `~/.fab-kit/config.yaml`, and enforce the registry scope through `internal/configscope`. Project-scoped keys MUST be refused in system mode. `set --system` SHALL create a missing system file with the canonical scaffold header and the requested live override, while subsequent mutations SHALL remain fence-free and comment-preserving.

- **GIVEN** a missing system config or an existing commented/user-edited system config
- **WHEN** a scope-eligible key is set or unset with `--system`
- **THEN** the file is created or surgically edited without a managed fence
- **AND** a project-scoped key is refused before write

### Lifecycle and Documentation: Current Surface Without Historical Churn

#### R8: Project-Default Init and Stable Upgrade

Bare `fab config init` MUST run project mode while `--project` remains supported, `--system` remains unchanged, `--project --system` remains an error, and overwrite refusal remains intact. `upgrade` behavior MUST remain unchanged apart from help text that reflects the six-verb surface and removes the reserved-validate claim.

- **GIVEN** a fresh fab repo, an existing target file, or mutually exclusive init flags
- **WHEN** `fab config init` is invoked
- **THEN** the bare form creates the same project config as explicit `--project`, existing files are not overwritten, and conflicting flags fail
- **AND** upgrade's file semantics and fence anchors remain byte-identical

#### R9: Mirrored, Canonical Documentation Sweep

Current CLI, skill, spec, site, and affected-memory text MUST describe `explain`, keyed `show`, surgical `set`/`unset`, bare project init, and the dropped validate slot. Every changed `src/kit/skills/*.md` file MUST have its `docs/specs/skills/SPEC-*.md` mirror updated. `docs/site/skill.md` SHALL be canonical and `scripts/sync-skill.sh` MUST keep `src/go/fab/cmd/fab/skill.md` byte-identical. Historical migration files, the quoted seeded pointer `# Full reference of all available options: fab config reference`, and managed-fence anchor/prose text MUST remain verbatim.

- **GIVEN** the repository-wide current-text and mirror classes
- **WHEN** the documentation sweep and sync script complete
- **THEN** current guidance consistently uses the six-verb surface, all required mirrors agree, and the skill embed drift guard passes
- **AND** historical migration text, the seeded pointer quotation, and fence anchors have no diff

### Non-Goals

- Environment-variable overrides and launch flags (C1)
- `dispatch.mode` and watchable dissolution (C3)
- Provider command-field renames (C4)
- `upgrade --check`, fence scope annotations, one-values-file consolidation, and origin drill-down repair (C5)
- `FAB_KIT_PATH` (C6)
- Any new migration file or edits to historical migration files

### Design Decisions

#### Set Is a Scalar-Only, Leaf-Only Convenience Function

**Decision**: `fab config set` accepts only a scalar-valued leaf key and a single-line, comment-free scalar value; map/sequence values, multi-line spellings, and collection-valued leaves are refused with a pointer to manual editing. Materialization renders the full ancestor chain.
**Why**: The raw-text splice writer can prove correctness only over a small input space — every widening (parent-path maps, multi-line flow, comment-bearing scalars) produced a silent-wrong-at-exit-0 review finding (phantom ancestor keys, splice mangling, nested-shape bypass). Shrinking the accepted inputs kills the failure class, not instances; manual edit + `upgrade` remains the first-class path for complex shapes.
**Rejected**: Teaching the writer every YAML shape (three review cycles showed the tail does not converge); quote-serializing exotic values (complexity without a use case); restricting the shared parser globally (the env layer writes no file, so flow collections stay valid there).
*Introduced by*: 260808-ihiv-config-six-verb-surface (user-confirmed 2026-08-09, closing rework cycle)


#### Mutation Stays Inside the Fence Engine

**Decision**: Both CLI mutation verbs delegate validation-aware, comment-preserving writes to `internal/configupgrade`; Cobra remains a thin path/scope/reporting layer.
**Why**: The engine already owns fence parsing, registry rendering, atomic writes, and idempotence, and the change exists to avoid YAML marshal round-trips.
**Rejected**: A second command-local writer or whole-document YAML marshal, either of which recreates the comment-mashing and renderer-drift classes.
*Introduced by*: 260808-ihiv-config-six-verb-surface

#### Dynamic Provider Names Round-Trip as String Keys

**Decision**: The dotted mutation surface keeps provider names unquoted and unescaped at the CLI, while the surgical writer quote-serializes an individual on-disk mapping key when its bare spelling would decode as a non-string YAML key. Opaque names such as `123`, bool-like words, `-local`, and Unicode remain valid.
**Why**: Successful mutation must be effective through the loader. A single path-segment renderer preserves raw document splicing while ensuring every dynamic provider axis decodes back to the exact string the dotted key named.
**Rejected**: Emitting every provider name bare (numeric and bool-like names silently become typed YAML keys), accepting quoted/escaped dotted-key spellings (makes path parsing ambiguous), or marshaling the surrounding document to obtain one key token.
*Introduced by*: 260808-ihiv-config-six-verb-surface

## Tasks

### Phase 1: Regression Harness and Schema Primitives

- [x] T001 <!-- rework cycle 5: renderer regression STILL not genuine (3rd recurrence): configupgrade_test.go:602 asserts Set output against a hand-written literal, then calls registryMaterializationSkeleton independently with an unrelated shifted segment — a duplicate literal renderer in the production Set path would pass both assertions. Rewrite so the test PROVES single-sourcing: derive the expected materialized bytes FROM the fence renderer's own segment output (e.g. render the field's fence segment, strip the comment markers via the inverse of CommentOutSegment, and assert Set's materialized block equals that derivation byte-for-byte), or assert at a seam that both paths call the same function. No hand-written literal may stand in for the renderer's output --> Add the six day-one failing regression cases before writer implementation across `src/go/fab/internal/configupgrade/configupgrade_test.go` and focused command/registry tests as needed: block-form orphaning, single-sourced renderer, key-axis typos, value-type holes, quote holes, and end-to-end comment preservation plus repeated-set byte identity. <!-- R4 R5 R6 -->
- [x] T002 Implement reusable YAML value parsing in `src/go/fab/internal/configvalue/` and registry key/owning-segment/expected-kind resolution in `src/go/fab/internal/configref/configref.go` with unit coverage, making validation tests green without adding writer behavior. <!-- R3 R4 -->

### Phase 2: Core Surgical Mutation

- [x] T003 <!-- rework cycle 10: scanner path-component identity: mutation.go:336-373,386-403,457-468 flattens mapping keys into dot-joined strings, so a literal user-owned key `agent.workers:` (one key WITH a dot in its name — valid YAML, inert unknown to the loader) is indistinguishable from the nested schema path agent:→workers:. Repro: from `agent.workers: claude`, `set agent.workers codex` exits 0, rewrites the unrelated literal key, materializes `agent:\n  workers:` with null, and keyed show returns null; unset can delete the unrelated key (R5/R6 violated). Fix: track path IDENTITY as components (each segment = one real mapping key at its own indent level), never compare flattened joined strings — a literal dotted key is a single component and can never match a multi-segment schema path. Add set/show + unset regressions with literal dotted YAML keys present --> <!-- rework: (1) mutation.go:290-302/354-374/625-631 parses a flow-mapping ancestor into a yaml.Node and re-serializes with yaml.Marshal — a YAML parse/marshal round-trip in the write path, violating the binding raw-splice constraint R5/A-012 and normalizing user flow formatting; replace with raw text splice. (2) mutation.go:420-423 treats any non-empty inline token as a one-line value, so block scalars (| and >) are not extended through their indented body — set replaces only the key line and unset orphans the scalar body; handle block-scalar bodies in the block scanner (R5/R6) --> Implement comment-preserving `Set` and `Unset` entry points and raw YAML path-splice helpers in `src/go/fab/internal/configupgrade/configupgrade.go`, reusing the existing registry fence renderer and atomic writer; satisfy all six T001 regressions. <!-- R5 R6 -->
- [x] T004 Add scope-aware project/system path handling and missing-system scaffold creation support around the mutation engine in `src/go/fab/internal/configupgrade/` and `src/go/fab/cmd/fab/config.go`, with focused system-scope tests. <!-- R7 -->

### Phase 3: CLI Integration and Edge Cases

- [x] T005 <!-- rework cycle 7: keyed --origin empty-mapping fallback: config.go:271 — `FAB_PROVIDERS='{custom: {}}' fab config show providers.custom` prints `{}` but `--origin` prints `null  # default`: flattenOrigin emits no leaf for an empty map, so the zero-selection fallback discards both the effective value and the $FAB_PROVIDERS provenance (R2/A-002). Make the zero-selection path fall back to the EFFECTIVE value already computed (renderCompactValue) with the winning layer's origin resolved via the ancestor walk — never a hardcoded `null  # default`; add the empty-mapping regression --> <!-- rework: config.go:249-258 sends a keyed --origin leaf through renderScalar, so a list key (`fab config show source_paths --origin`) emits `- src/ - scripts/  # <path>` instead of raw/scriptable flow output like `[src/, scripts/]  # <path>`; route keyed origin leaves through the compact renderer and add the missing list-origin test in config_show_init_test.go (R2/A-002) --> Extend keyed effective inspection and origin rendering in `src/go/fab/cmd/fab/config.go` with coverage in `src/go/fab/cmd/fab/config_show_init_test.go`, preserving byte-identical bare `show`. <!-- R2 -->
- [x] T006 <!-- rework cycle 6: registry segment prose still advertises the invisible alias: the configref Segment text (rendered into every managed fence) says `fab config reference` / `fab config reference --json` — every `fab config upgrade` regenerates that stale pointer into user repos (see this repo's fab/project/config.yaml:41,51). Sweep the segment prose in configref.go (and any other segment/comment source, e.g. internal/agent defaults.yaml comments) to `fab config explain`, keeping deliberate historical/alias-rationale text; then run `fab config upgrade` in THIS repo so fab/project/config.yaml's fence regenerates with the corrected prose, and include the regenerated file --> Rename the visible registry command to `explain`, retain `reference` only in `Aliases`, implement keyed YAML/JSON selection, and update command tests in `src/go/fab/cmd/fab/config_test.go`. <!-- R1 R3 -->
- [x] T007 <!-- rework cycle 9 (requirements revised — scalar-only set contract): rewire cmd set/unset to the scalar-only contract: refusal messages per revised R4 (name the reason; point to `fab config explain` or manual editing); update command-level tests --> Wire `fab config set` and `unset` in `src/go/fab/cmd/fab/config.go`, including exact args, `--system`, scope/unknown/type errors, exit-zero unset notices, and command-level tests. <!-- R1 R4 R5 R6 R7 -->
- [x] T008 Make bare `fab config init` select project mode while retaining explicit modes, mutual exclusion, and overwrite behavior; update lifecycle help and tests without changing upgrade semantics. <!-- R1 R8 -->

### Phase 4: Documentation, Mirrors, and Verification

- [x] T009 <!-- rework cycle 9 (requirements revised — scalar-only set contract): docs: _cli-fab.md § fab config set/unset rewritten to the scalar-only leaf-only contract (+ SPEC mirror) — value grammar, refusal cases, manual-edit fallback, env-layer distinction --> Rewrite the `fab config` surface in `src/kit/skills/_cli-fab.md` and its full mirror class in `docs/specs/skills/SPEC-_cli-fab.md`; update `src/kit/skills/fab-setup.md` plus `docs/specs/skills/SPEC-fab-setup.md` only if the command is named there. <!-- R1 R2 R3 R5 R6 R8 R9 -->
- [x] T010 <!-- rework cycle 9 (requirements revised — scalar-only set contract): docs: docs/specs/config.md set/unset framing + docs/memory/_shared/configuration.md updated to the scalar-only contract (present truth, no transition narration); re-run `fab memory-index` if descriptions change --> <!-- rework cycle 6: docs/specs/stage-models.md:845-846 still describes the shipped cascade as three layers (project > system > defaults) — post-C1 the current contract is environment > project > system > default; update the claim (or bound it explicitly as historical), and re-grep the whole changed-doc set for remaining three-layer claims --> Update only the C2-owned surface sections in `docs/specs/config.md`, the config row in `docs/specs/index.md`, and the four affected memory files `docs/memory/_shared/configuration.md`, `docs/memory/distribution/kit-architecture.md`, `docs/memory/runtime/providers-and-profiles.md`, and `docs/memory/distribution/migrations.md`, preserving the quoted seeded pointer verbatim. <!-- R1 R2 R3 R5 R6 R8 R9 -->
- [x] T011 Update canonical `docs/site/skill.md`, run `scripts/sync-skill.sh`, and verify `src/go/fab/cmd/fab/skill.md` is byte-identical; sweep current non-historical `fab config reference` text while leaving `src/kit/migrations/*.md`, fence anchors, and seeded historical quotations untouched. <!-- R1 R3 R9 -->
- [x] T012 Run `gofmt` on changed Go files, the required `shll standards`/`principles`/`help-dump`/`skill` checks, focused package tests under `src/go/fab/...`, the skill drift guard, and widened `go test ./...` for the fab module; fix all failures and confirm scope/history guards. <!-- R1 R2 R3 R4 R5 R6 R7 R8 R9 -->

### Phase 5: Review-Cycle-3 Plan Revision (added after two fix-code cycles — escalation)

- [x] T013 Make the quote-hole regression genuine in `src/go/fab/internal/configupgrade/configupgrade_test.go:662-680`: the current unterminated-quoted-value case returns from `configvalue.Parse` before any file read, so `inlineCommentIndex` (mutation.go:44-46) is never exercised and the test survives a scanner regression. Add a malformed live-value **Unset** case that reaches the leading-quote comment scanner on a real file (e.g. a live leaf whose value is an unterminated quote containing ` # `), assert the surviving bytes, and retain the existing quoted-key refusal case. Re-verify A-014. <!-- R4 R6 -->
- [x] T014 Single-source the origin-line formatting in `src/go/fab/cmd/fab/config.go`: `renderOriginLines` (291-302) duplicates the width-calculation and `key = value  # origin` formatting loop inside `renderShowOrigin` (382-398). Extract one shared formatter over `[]originLine` and route both callers through it; keyed and bare `--origin` output must stay byte-identical (existing tests pin both). Re-verify A-023 and A-027. <!-- R2 -->

### Phase 6: Contract Reverts (cycle 9)

- [x] T015 Revert the `aliases` field added to the frozen help-dump schema-version-1 wire contract (`src/go/fab/cmd/fab/helpdump.go:28-30,73-76`, its tests, and the kit-architecture.md help-dump prose) — an unrelated public-surface expansion; the invisible cobra alias needs no help-dump exposure. Restore the pre-change node shape byte-for-byte. <!-- R1 R9 -->

## Execution Order

- T001 MUST complete before T003 begins; the six regressions are written before surgical writer code.
- T002 blocks T003 and T006 because both mutation and keyed explain consume the registry resolver.
- T003 blocks T004 and T007; T004 blocks the `--system` half of T007.
- T005 and T006 may proceed independently after T002; T008 may proceed independently of T003-T007.
- T009-T011 begin only after the CLI behavior is stable; T012 is terminal verification.

## Acceptance

### Functional Completeness

- [x] A-001 R1: Help exposes exactly the six visible config verbs and the hidden `reference` alias remains compatible.
- [x] A-002 R2: Keyed show resolves effective defaults, raw scalar/list output, YAML map output, origins, and unknown-key failures while bare output is unchanged.
- [x] A-003 R3: Keyed explain resolves owning segments in YAML and deterministic matching row sets in JSON.
- [x] A-004 R4: Registry validation rejects key typos, quoted keys, collection paths/values, multiline/comment-bearing values, and wrong scalar kinds before write while accepting supported single-line scalar leaves; the shared parser retains flow collections for ENV.
- [x] A-005 R5: Project set is surgical, fence-aware, comment/sibling preserving, full-ancestor materializing, and byte-idempotent.
- [x] A-006 R6: Project unset restores inheritance/fence advertising, repairs malformed live values, and treats absent known overrides as an exit-zero notice.
- [x] A-007 R7: System mutation enforces scope, creates the missing scaffold-header file, and remains fence-free and comment-preserving.
- [x] A-008 R8: Bare init selects project mode with explicit flags and overwrite protections retained; upgrade semantics do not change.
- [x] A-009 R9: Current docs, skill mirrors, memory, and the canonical/embedded skill bundle agree on the six-verb surface.

### Behavioral Correctness

- [x] A-010 R1: Existing `fab config reference` invocations continue working without appearing as a visible help command.
- [x] A-011 R2: Existing bare `show` and `show --origin` golden/output contracts remain byte-identical.
- [x] A-012 R5: No set/unset path performs a whole-document YAML marshal or duplicates the registry segment renderer.
- [x] A-013 R8: Existing explicit `init --project`, `init --system`, and `upgrade` paths retain their prior behavior.

### Scenario Coverage

- [x] A-014 R4: Tests cover both named key-axis typos, structural-map and collection-value refusal, multiline/comment-bearing scalar refusal, quoted-key refusal, and malformed quote input.
- [x] A-015 R5: Tests cover nested block sibling preservation, fence materialization, inline/interior/preceding comments, and setting the same value twice.
- [x] A-016 R6: Tests cover removal of valid and malformed live values plus already-unset no-op behavior.
- [x] A-017 R7: Tests cover missing and existing system files and project-scoped refusal under `--system`.

### Edge Cases & Error Handling

- [x] A-018 R2: Unknown or structurally invalid show keys fail non-zero and name the requested key.
- [x] A-019 R3: Segment-less registry rows and supported dynamic descendants resolve to the correct owning segment; unknown keys fail clearly.
- [x] A-020 R4: Validation happens before filesystem mutation, leaving bytes unchanged on every rejected request.
- [x] A-021 R6: Unset prunes only empty structural parents and never discards sibling keys or user comments.

### Code Quality

- [x] A-022 Pattern consistency: New Go code follows existing Cobra thin-wrapper, internal-package, table-test, error-wrapping, and atomic-write patterns.
- [x] A-023 No unnecessary duplication: Existing registry, scope taxonomy, fence renderer, origin helpers, and atomic writer are reused.
- [x] A-024 Readability and maintainability: Mutation/path-scanning functions remain focused and named by responsibility rather than becoming one monolithic writer.
- [x] A-025 Composition: Key resolution, value parsing, raw splicing, fence regeneration, and command wiring are separated into composable units.
- [x] A-026 No God functions: Any function exceeding surrounding package norms is split unless a clear parser state machine warrants the size.
- [x] A-027 No duplicated utilities: The implementation introduces no second registry renderer, scope table, origin composer, or value parser.
- [x] A-028 No magic strings or numbers: User-facing fixed text and structural tokens with shared meaning use named constants/helpers.
- [x] A-029 Canonical-source discipline: No `.claude/skills/` file is edited; every canonical skill edit carries its SPEC mirror.
- [x] A-030 CLI documentation discipline: Changed command signatures are covered by Go tests and `_cli-fab.md` plus its mirror.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None identified. <!-- the earlier renderShowOrigin inline-formatter candidate was consolidated by T014 (config.go now delegates to renderOriginLines) -->

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Put the shared YAML scalar/flow parser in a small `internal/configvalue` package and expose parsed kind alongside the value. | C1 and C2 share semantics but run on parallel branches; a dependency-free package is mechanically reusable when the second branch rebases and avoids coupling parsing to Cobra or the writer. | S:75 R:85 A:85 D:75 |
| 2 | Confident | Registry key resolution explicitly recognizes the documented dynamic axes (provider name, fixed role, fixed stage, and their allowed leaf names) rather than accepting any descendant below a map row. | This is required for typo linting while retaining open provider names; the fixed role/stage sets already have canonical agent symbols and the allowed leaf shapes are the config schema. | S:75 R:80 A:85 D:80 |
| 3 | Confident | `explain --json <key>` returns a JSON array containing all registry rows represented by the selected owning segment. | The intake says matching row(s), and the array preserves the existing object shape/order while handling segment-less rows without inventing a second JSON schema. | S:70 R:85 A:80 D:80 |
| 4 | Confident | Keyed `show --origin` keeps scalar/list output value-first with an origin suffix; map-valued requests use the existing flattened per-leaf origin representation filtered to the requested subtree. | Raw output remains scriptable without `--origin`; origin is inherently per leaf for merged maps, and reusing `flattenOrigin` avoids a second provenance algorithm. | S:60 R:80 A:85 D:65 |
| 5 | Confident | Project mutation regenerates the managed fence after raw live-area splicing, while system mutation uses the same path splicer without any fence. | This composes the existing single-sourced fence renderer with the intake's explicit fence-free system-file rule and keeps the writer idempotent. | S:80 R:80 A:90 D:85 |

5 assumptions (0 certain, 5 confident, 0 tentative).
