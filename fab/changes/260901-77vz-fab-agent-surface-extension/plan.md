# Plan: fab agent Surface Extension

**Change**: 260901-77vz-fab-agent-surface-extension
**Intake**: `intake.md`

## Requirements

### CLI: fab agent selector grammar

#### R1: Stage selectors on the positional
The `[role]` positional SHALL additionally accept a stage name (`intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`), resolved role-first via the existing `agent.RoleForName` (`internal/agent/agent.go:360` — role set checked first, then `stageRoles`; the `review`/`hydrate` collisions are fixed points, so order is immaterial). An unknown selector SHALL keep a non-zero-exit error naming the valid names.

- **GIVEN** a default config
- **WHEN** `fab agent apply --print` runs
- **THEN** stdout is the `doing` role's composed interactive command, byte-identical to `fab agent doing --print`
- **AND** `fab agent bogus --print` exits non-zero naming valid role and stage names

#### R2: Selector + `--provider` re-resolves fills
Supplying both a selector (role or stage) and `--provider <name>` SHALL cease to be a usage error: the role's profile SHALL re-resolve from the named provider's own fills via `agent.ResolveRoleWith(cfg, role, Overrides{Provider, ProviderSet: true})` (`agent.go:464`). Bare `--provider` (no selector) SHALL keep today's total fill-bypass semantics untouched (profile = exactly the passed flags; `providers.<name>.profiles` deliberately not consulted).

- **GIVEN** default config (built-in kimi ships no fills)
- **WHEN** `fab agent apply --provider kimi --print` runs
- **THEN** stdout is `kimi --auto` (empty `{model}` drops the `-m` pair — kimi's own fills, not claude's model patched in)
- **AND** `fab agent --provider kimi --print` (bare) prints the same via the bypass path, unchanged from today

#### R3: `--model`/`--effort` as general post-refill overrides
`--model` and `--effort` SHALL be legal with any addressing form (bare, role, stage, with or without `--provider`), applied as final verbatim overrides after role resolution / provider refill (`ResolveRoleWith`'s `ModelSet`/`EffortSet`). The today-only-with-`--provider` guard is removed; `Flag.Changed` keying is preserved (an explicitly-empty `--model=` clears the field rather than being ignored). No validation, no enum, no pair correction.

- **GIVEN** a default config
- **WHEN** `fab agent review --model claude-sonnet-5 --effort max --print` runs
- **THEN** the printed command carries `claude-sonnet-5` and `max` in the review role's command shape
- **AND** `fab agent --model claude-haiku-4-5-20251001 --print` resolves the default role with the model overridden

#### R4: `-t, --template` prints the unsubstituted template
`-t, --template` SHALL print the selected provider's command template with `{model}`/`{effort}` placeholders intact (a tap before the fill step), implying print-mode (never execs). It SHALL combine with any selector, `--provider`, and `--headless` (they pick which template). It SHALL reject `--model`/`--effort` with a usage error (they feed a step that never runs).

- **GIVEN** a default config
- **WHEN** `fab agent apply -t` runs
- **THEN** stdout is claude's raw `interactive_command` with placeholders intact
- **AND** `fab agent apply --provider kimi -t` prints `kimi --auto -m {model}`
- **AND** `fab agent apply --model k3 -t` exits non-zero: `--model has no effect with --template (substitution never runs)`

#### R5: `--headless` resolves `headless_command`
`--headless` SHALL select the provider's `headless_command` instead of `interactive_command`, valid only in the print-family modes (`--print`, `-t`, `-o yaml`); combining `--headless` with exec (no print-family flag) SHALL be a usage error. A provider without `headless_command` SHALL hard-error naming the config key `providers.<name>.headless_command` (mirror of the existing interactive hint) — never a silent descent.

- **GIVEN** a default config
- **WHEN** `fab agent doing --headless --print` runs
- **THEN** stdout is claude's substituted `headless_command`
- **AND** `fab agent doing --headless` (no print flag) exits non-zero as a usage error
- **AND** a provider lacking the capability yields `configure providers.<name>.headless_command`

#### R6: `-o yaml` structured resolution output (minimal)
`-o, --output <format>` SHALL accept exactly `yaml` (any other value is a usage error), implying print-mode. It SHALL emit a YAML document with keys `selector`, `kind` (`role`|`stage`), `role`, `provider`, `model`, `effort`, `command` — the minimal projection; Change 2 (`mp8d`) owns the full schema and extends additively (`model_alias`/`template`/`fill_mode`/`source`/`dispatch` are NOT emitted here). `-o` SHALL be mutually exclusive with `--print` and with `-t` (usage errors — one output sink per invocation). For bare-`--provider` mode, `kind` and `role` report the provider-addressed form (`kind: provider`, `role` empty).

- **GIVEN** a default config
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** the document reports `selector: apply`, `kind: stage`, `role: doing`, the resolved provider/model/effort, and the composed `command`
- **AND** `fab agent apply -o json` exits non-zero as a usage error
- **AND** `fab agent apply -o yaml --print` exits non-zero as a usage error

#### R7: `-p` shorthand for `--print`
`--print` SHALL gain the `-p` short flag. Long-form behavior and output are untouched.

- **GIVEN** any invocation legal with `--print`
- **WHEN** the same invocation runs with `-p`
- **THEN** output is byte-identical

#### R8: Byte-stability and unchanged behaviors
`--print` output SHALL remain byte-identical for every invocation legal today (operator spawn seam + cross-repo spawning consume it). Golden tests pinning today's bytes MUST be written BEFORE the surface rework. Unchanged: bare `fab agent` (exec default role — no picker), `--` passthrough (shell-quoted append), `--repo`, `--workers` (env replace), project-free cascade, exec via `sh -c`, no-TTY-guard, the shared `unknownProviderError` phrasing, `roleSessionCommand`'s contract, selector case-sensitivity. `fab resolve-agent` is not touched.

- **GIVEN** the golden test matrix over today-legal invocations (default role, named roles, bare `--provider` with/without `--model`/`--effort`, `--repo`, `--` passthrough)
- **WHEN** the reworked surface runs the matrix
- **THEN** every output is byte-identical to the pinned pre-rework bytes

### Docs: skill reference and spec sweep

#### R9: `_cli-fab.md` § fab agent updated
`src/kit/skills/_cli-fab.md` § fab agent SHALL document the new selector grammar and flags, and the now-stale claims SHALL be rewritten: the `:348` asymmetry note (bare `--model`/`--effort` asymmetry with resolve-agent is erased by R3 — rewrite both sides), the `:1442` "require `--provider`" bullet, and the `:1444` mutual-exclusion bullet (replaced by re-resolve semantics).

- **GIVEN** the shipped `_cli-fab.md`
- **WHEN** grepped for the mutual-exclusion and require-`--provider` claims about `fab agent`
- **THEN** no stale occurrence remains and the new surface (stage selectors, `-t`, `--headless`, `-o yaml`, `-p`, override layering) is documented

#### R10: `_cli-agents.md` updated
`src/kit/skills/_cli-agents.md` (§§ ~:46-67) SHALL rewrite the `:63` mutual-exclusion / only-with-`--provider` claims and note the new stage/template forms available to spawn composition.

- **GIVEN** the shipped `_cli-agents.md`
- **WHEN** grepped for the `:63` claims
- **THEN** they match the new surface

#### R11: `stage-models.md` mentions the launcher grammar
`docs/specs/stage-models.md` SHALL gain a brief mention of `fab agent`'s new selector grammar (surgical edit — human-curated spec).

- **GIVEN** the spec's launcher/resolve-agent discussion
- **WHEN** read after this change
- **THEN** it acknowledges stage selectors on `fab agent` without restating the CLI reference

#### R12: Repo-wide sibling sweep
Before review, a contrastive sweep SHALL run over the phrase classes ("mutually exclusive" + role/provider, "require --provider", "deliberate asymmetry", "fab agent") across `src/kit/skills/`, `docs/specs/` (incl. aggregates `skills.md`/`glossary.md`/`architecture.md`), `docs/memory/`, `*_test.go` comments, and `src/go/fab/defaults.yaml`/`cmd/fab` doc comments — every occurrence describing the old surface updated in this same change. (Memory-file content updates land at hydrate; the sweep still identifies them.)

- **GIVEN** the completed implementation
- **WHEN** `grep -rn "mutually exclusive" src/kit docs src/go | grep -i "provider\|role"` and the sibling phrase greps run
- **THEN** zero occurrences still describe the pre-change `fab agent` surface

### Non-Goals

- No mutation of `fab resolve-agent`'s flags, arguments, or output (frozen; parity is Change 2, migration Change 3, retirement later).
- No interactive picker (bare `fab agent` unchanged; plan open decision 2 deferred).
- No case-insensitive selector matching.
- No change to the operator launcher path (`operator.go` / `WithProfile` direct), `fab pane open`, `fab batch`, `fab dispatch start` composition, or `spawn.WithProfile` semantics.
- No provider grammar validation; verbatim pass-through holds on `-t` and `-o yaml` too.
- No `json` output format (spelling reserved for later).

### Design Decisions

#### One output sink per invocation
**Decision**: `-o yaml` is mutually exclusive with `--print` and `-t` (usage errors); `-t` with `--print` is tolerated (`-t` implies print).
**Why**: each print-family flag selects a different sink (command line / template / structured doc); silently preferring one over another would make output shape depend on flag order.
**Rejected**: precedence ordering (surprising, undocumentable cheaply); allowing `-o yaml -t` to add a template key (that key is Change 2's schema surface).
*Introduced by*: 260901-77vz-fab-agent-surface-extension

#### Reuse `RoleForName`, no new hoist
**Decision**: selector-kind detection reuses the exported `agent.RoleForName` (+ `agent.IsRoleName` for `kind` reporting); no new `internal/agent` symbol unless the `-o yaml` renderer needs one.
**Why**: the role-first/stage-fallback resolution the intake called for already exists at `agent.go:360` with the fixed-point guarantee tested.
**Rejected**: a new `SelectorKind` helper in `internal/agent` — premature; Change 2's `agent.Resolution` struct is the right home for kind provenance.
*Introduced by*: 260901-77vz-fab-agent-surface-extension

## Tasks

### Phase 1: Setup

- [x] T001 Golden tests pinning today's `--print` bytes across every today-legal invocation shape (bare/default role, each named role, bare `--provider` with and without `--model`/`--effort`, `--repo`, `--` passthrough, `--workers` with `--print`) in `src/go/fab/cmd/fab/agent_test.go`; run them GREEN against the untouched implementation before any rework commit <!-- R8 -->

### Phase 2: Core Implementation

- [x] T002 Rework `cmd/fab/agent.go` addressing: positional resolves via `agent.RoleForName` (stage or role; record selector + kind for `-o yaml`), remove the role×`--provider` mutual-exclusion guard, remove the `--model`/`--effort`-require-`--provider` guard; selector paths resolve through `agent.ResolveRoleWith` with `Overrides{Provider/Model/Effort + *Set}` from `Flag.Changed`; bare-`--provider` path byte-for-byte unchanged <!-- R1 -->
- [x] T003 Wire override layering and update the command's doc comment + `Use`/flag help text for the new addressing model (role|stage positional, provider re-resolve vs bare bypass, override layering) <!-- R3 -->
- [x] T004 Add `-t, --template` flag: template selection (interactive vs headless per `--headless`; provider from selector resolution or `--provider`), print unsubstituted, usage errors for `--model`/`--effort` combination <!-- R4 -->
- [x] T005 Add `--headless` flag: resolve `headless_command` in print-family modes, usage error when combined with exec, missing-capability error naming `providers.<name>.headless_command` <!-- R5 -->
- [x] T006 Add `-o, --output` flag: `yaml`-only value gate, minimal struct (`selector`/`kind`/`role`/`provider`/`model`/`effort`/`command`) marshalled via `gopkg.in/yaml.v3`, print-mode implied, mutual-exclusion guards vs `--print`/`-t`; `kind: provider` + empty `role` for bare-provider mode <!-- R6 -->
- [x] T007 Add `-p` shorthand to the `--print` flag registration <!-- R7 -->

### Phase 3: Integration & Edge Cases

- [x] T008 Test coverage for every new capability and usage error in `cmd/fab/agent_test.go`: stage selector (print + exec seam), selector+`--provider` refill (kimi empty-fill case), bare/role/stage `--model`/`--effort` overrides incl. explicit-empty `--model=`, `-t` matrix (+ error), `--headless` matrix (+ both errors), `-o yaml` golden output (+ `-o json` and combination errors), `-p` ≡ `--print`, unknown-selector error text <!-- R1 -->
- [x] T009 Run `gofmt`, `go vet`, and the affected package tests (`cmd/fab`, `internal/agent`); widen to the full module suite once green <!-- R8 -->

### Phase 4: Polish

- [x] T010 Update `src/kit/skills/_cli-fab.md` § fab agent: new synopsis, selector grammar, flag table, override layering, error surfaces; rewrite the asymmetry note at ~:348 (both directions) and the `:1442`/`:1444` bullets <!-- R9 -->
- [x] T011 Update `src/kit/skills/_cli-agents.md` ~:46-67: rewrite the `:63` exclusion/override claims; note stage/template forms for spawn composition <!-- R10 -->
- [x] T012 Add the launcher selector-grammar mention to `docs/specs/stage-models.md` (surgical) <!-- R11 -->
- [x] T013 Repo-wide contrastive sweep: grep `fab agent` + phrase classes ("mutually exclusive", "require --provider", "asymmetry", "only valid alongside") across `src/kit/`, `docs/specs/` (aggregates included), `docs/memory/`, `src/go/` comments and `*_test.go`; fix every stale claim in the same change (memory-file body updates deferred to hydrate but enumerated in the apply result) <!-- R12 -->

## Execution Order

- T001 blocks everything (golden pins must exist and pass first)
- T002 blocks T003–T007 (the addressing rework is the base)
- T008–T009 after Phase 2; T010–T013 after tests are green

## Acceptance

### Functional Completeness

- [x] A-001 R1: Stage names resolve on the positional to the mapped role's command; unknown selectors error naming valid names
- [x] A-002 R2: Selector+`--provider` refills from the named provider's fills; bare `--provider` bypass byte-identical to today
- [x] A-003 R3: `--model`/`--effort` legal and layered correctly in every addressing form, `Flag.Changed`-keyed
- [x] A-004 R4: `-t` prints the correct unsubstituted template in all combinations and rejects `--model`/`--effort`
- [x] A-005 R5: `--headless` resolves the headless template, errors on exec-mode use and on missing capability with the config-key hint
- [x] A-006 R6: `-o yaml` emits exactly the seven minimal keys with correct values in role, stage, and bare-provider modes
- [x] A-007 R7: `-p` is byte-equivalent to `--print`

### Behavioral Correctness

- [x] A-008 R8: Golden `--print` tests pass unchanged post-rework for every today-legal invocation
- [x] A-009 R8: Bare `fab agent` execs the default role; no picker; exec path still `sh -c` with `--workers` env replace

### Scenario Coverage

- [x] A-010 R1: `fab agent apply --print` ≡ `fab agent doing --print` covered by test
- [x] A-011 R2: kimi refill scenario (`apply --provider kimi --print` → `kimi --auto`) covered by test
- [x] A-012 R6: `-o json`, `-o yaml --print`, `-o yaml -t` usage errors covered by tests

### Edge Cases & Error Handling

- [x] A-013 R3: Explicit-empty `--model=` clears the field (inherit/drop semantics) rather than being ignored
- [x] A-014 R5: `--headless` with a fills-less provider drops placeholder tokens per `WithProfile` (no panic, no validation)
- [x] A-015 R4: `-t` with unknown `--provider` still yields the shared `unknownProviderError` phrasing

### Documentation Accuracy

- [x] A-016 R9: `_cli-fab.md` § fab agent matches the shipped surface; no stale exclusion/asymmetry claims anywhere in the file
- [x] A-017 R10: `_cli-agents.md` claims match the shipped surface
- [x] A-018 R11: `stage-models.md` mentions the launcher selector grammar
- [x] A-019 R12: Phrase-class sweep clean — no occurrence in `src/kit/`, `docs/specs/`, `src/go/` comments describing the old surface

### Code Quality

- [x] A-020 Pattern consistency: new flags/guards follow `agent.go`'s existing `Flag.Changed` keying, error phrasing, and doc-comment style
- [x] A-021 No unnecessary duplication: resolution stays in `internal/agent` (`ResolveRoleWith`/`RoleForName`); no re-implemented precedence chain in `cmd/fab`
- [x] A-022 CLI ⇒ docs + tests: `_cli-fab.md` updated and tests ship in the same change (constitution constraint)
- [x] A-023 Canonical source only: all skill edits under `src/kit/skills/`, none under `.claude/skills/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/cmd/fab/agent.go:370-380 (roleSlotError)` — dead helper with zero call sites (the selector path formats its slot error inline at `resolveAgentInvocation`); flagged as must-fix parsimony `zero-call-sites` and removed on rework
- `docs/memory/runtime/providers-and-profiles.md:345/:361/:521` — stale "mutually exclusive addressing modes" claims describing the pre-change `fab agent` surface; enumerated per T013/R12, body updates deferred to hydrate by design

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `-o yaml` mutually exclusive with `--print` and `-t`; `-t --print` tolerated (implied) | One sink per invocation; precedence ordering rejected as surprising | S:55 R:85 A:80 D:60 |
| 2 | Certain | Reuse exported `agent.RoleForName`/`IsRoleName` for selector detection; no new hoist | The intake's "possible hoist" already exists at `agent.go:360` with fixed-point tests | S:85 R:90 A:95 D:90 |
| 3 | Confident | Bare-provider mode reports `kind: provider` with empty `role` in `-o yaml` | Minimal schema must represent the third addressing form; Change 2 can refine | S:50 R:80 A:75 D:60 |
| 4 | Certain | YAML rendering via `gopkg.in/yaml.v3` (already a module dependency) | Existing tests import it; no new dependency | S:80 R:90 A:95 D:95 |

4 assumptions (2 certain, 2 confident, 0 tentative).
