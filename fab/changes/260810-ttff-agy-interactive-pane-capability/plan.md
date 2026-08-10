# Plan: agy-interactive-pane-capability

**Change**: 260810-ttff-agy-interactive-pane-capability
**Intake**: `intake.md`

## Requirements

### Providers: agy ships an `interactive_command`

#### R1: `defaults.yaml` carries agy's interactive grammar
`src/go/fab/internal/agent/defaults.yaml` SHALL ship `interactive_command: 'agy --dangerously-skip-permissions --model {model}'` in the `providers.agy` block — the full-auto posture flag matching its own headless form, **no** `{effort}` placeholder (agy's model IDs embed the reasoning level as a suffix), and **no** initial-prompt placeholder (delivery is post-launch via `fab dispatch deliver`). The surrounding comments SHALL be rewritten: the "agy deliberately carries NO interactive_command" block and the stale "Backlog [agik] owns that probe" references are deleted; the first-run trust wall is recorded as an ordinary readiness-gate judgment round (the kimi precedent), with the trust store noted as additionally user-seedable.

- **GIVEN** the embedded `defaults.yaml` in `internal/agent`
- **WHEN** `ResolveProvider(nil, "agy")` runs with no user config
- **THEN** the resolved provider carries a non-empty `InteractiveCommand` equal to `agy --dangerously-skip-permissions --model {model}`
- **AND** automatic mode resolution inside tmux may select pane for agy instead of descending with `pane unavailable: no interactive_command`

#### R2: The command is pinned by value and exported like its siblings
`internal/agent` SHALL export a `DefaultAgyInteractiveCommand` var (the per-name sibling of `DefaultCodexInteractiveCommand`/`DefaultKimiInteractiveCommand`, sourced from `defaults.yaml`), `internal/configref` SHALL interpolate it into the rendered reference's agy block, and `defaults_test.go` SHALL pin the shipped value **by value** (the same treatment `kimi --auto -m {model}` gets), so an edit cannot reach pane workers unnoticed.

- **GIVEN** the agy block ships an `interactive_command`
- **WHEN** `fab config explain` renders the providers block
- **THEN** the agy block opens on `interactive_command` (like codex's and kimi's), interpolated from `agent.DefaultAgyInteractiveCommand` with no literal copy
- **AND** `go test ./internal/agent` fails if the shipped value drifts from the pinned string

#### R3: First-run trust wall rides the generic readiness gate — no agy-specific code
The change SHALL NOT add provider-specific Go machinery: agy's fresh-workspace trust prompt is an ordinary readiness-gate judgment round (one answer, workspace-scoped), exactly the shape that clears kimi's identical `Trust this folder?` wall. No trust-store seeding ships; seeding (`~/.gemini/antigravity-cli/settings.json`, exact-match) is documented as a user-side optimization only.

- **GIVEN** a pane worker launched on the agy built-in in a fresh worktree
- **WHEN** `fab dispatch ready` probes the pane
- **THEN** the trust wall is classifiable as a judgment round the gate can answer, with no agy-specific branch in fab's code

### Docs: the capability framing is corrected at its sources

#### R4: The "dispatch-only built-in" claim is swept repo-wide
Every prose occurrence of the claims "agy is dispatch-only", "agy ships/carries no `interactive_command`", and "backlog [agik] owns that probe" SHALL be rewritten to the shipped state: all four built-ins are pane-capable. The sweep class is: `defaults.yaml` comments, `internal/agent/agent.go` + `agent_test.go` + `defaults_test.go` comments, `internal/configref/configref.go` rendered prose, `cmd/fab` test pins, `docs/specs/stage-models.md`, `docs/specs/architecture.md`, `docs/specs/config.md`, `src/kit/skills/_cli-agents.md`, `src/kit/skills/_cli-fab.md`, `docs/specs/skills/SPEC-_cli-agents.md`, `docs/specs/skills/SPEC-_cli-fab.md`, and the three runtime memory files plus `docs/memory/_shared/configuration.md`. Files carrying only the generic descent-reason vocabulary (`pane unavailable: no interactive_command` — `docs/specs/harness-adapters.md`, `src/kit/skills/_preamble.md`, `docs/specs/skills/SPEC-_preamble.md`, `internal/dispatch`) stay unchanged: a user-defined provider with no `interactive_command` remains a valid configuration the ladder handles.

- **GIVEN** the flip in R1
- **WHEN** `grep -r "dispatch-only\|agik" src/ docs/ fab/backlog.md` runs after the sweep
- **THEN** no occurrence still describes agy as the dispatch-only built-in or defers its roster flip to `[agik]`
- **AND** each edited `src/kit/skills/*.md` carries its `docs/specs/skills/SPEC-*.md` mirror update in the same change (constitution mirror rule)

#### R5: Memory states the corrected capability model
`docs/memory/runtime/providers-and-profiles.md` (primarily), `dispatch.md`, `agent-primitives.md`, and `_shared/configuration.md` SHALL state the corrected framing in present truth: **every agent CLI has an interactive mode — pane capability is the default expectation for any provider, because it rides launch grammar plus the generic readiness gate; headless grammar is the capability that varies per CLI and needs probing.** An absent `interactive_command` in a user's own `providers:` block remains a valid configuration the descent ladder handles; it is just no longer a shipped state. The per-provider open question is first-run behavior, answered for agy by the kimi-precedent gate mechanics.

- **GIVEN** a reader learning the provider model from memory
- **WHEN** they read the built-in providers table and the agy entry
- **THEN** they find all four built-ins pane-capable, agy's shipped interactive grammar, and the trust-wall note — never the claim that agy "has no interactive mode" or is dispatch-only

### Tests: the pins flip with the capability

#### R6: Tests asserting agy's dispatch-only posture are replaced, not deleted
Tests that pin the old posture SHALL be updated to the new spec (Constitution VII — tests conform to spec): `defaults_test.go` gains agy in the interactive-presence loop and the by-value pin; `agent_test.go`'s `TestResolveProvider_NonClaudeBuiltIns` asserts agy's session command + bypass flag and its pane eligibility; `cmd/fab` tests that used agy as "the built-in with no `interactive_command`" (`TestAgentProviderDispatchOnlyBuiltInsError`, the `modeCommand` agy case in `dispatch_start_test.go`, the agy error case in `pane_open_test.go`, the agy rendering pins in `config_test.go`) are rewritten — error-path coverage moves to user-defined dispatch-only providers (the pattern `TestAgentProviderNoInteractiveCommandErrors` already uses), and agy cases assert the composed command instead.

- **GIVEN** the new built-in table
- **WHEN** `go test ./src/go/fab/internal/agent/... ./src/go/fab/internal/configref/... ./src/go/fab/cmd/fab/...` runs
- **THEN** the suite is green with agy pane-capable, and the no-`interactive_command` error path is still covered against user-defined providers

### Non-Goals

- Live agy pane-worker verification — agy quota is exhausted until ~2026-08-16; verification rides tests + the kimi-precedent gate mechanics, and the live probe is recorded as a backlog follow-up.
- Trust-store seeding in Go or config — stays a user-side optimization note (intake assumption 4).
- Changes to the descent ladder, readiness gate, or delivery machinery — generic; no agy-specific branch is added.

## Tasks

### Phase 1: Core data + `internal/agent`

- [x] T001 Add `interactive_command: 'agy --dangerously-skip-permissions --model {model}'` to the agy block in `src/go/fab/internal/agent/defaults.yaml`; rewrite the providers-block note (delete the "deliberately carries NO interactive_command" + "[agik]" prose; record the trust wall as a readiness-gate judgment round with user-seedable trust store) and the agy block comment <!-- R1, R3 -->
- [x] T002 Add the `DefaultAgyInteractiveCommand` export in `src/go/fab/internal/agent/agent.go`; update the non-claude commands comment block and the provider-roster doc comment (agy entry: interactive AND headless, no pane caveat) <!-- R2, R4 -->
- [x] T003 Flip `src/go/fab/internal/agent/defaults_test.go`: include agy in the interactive-presence loop, replace the absence assertion with a by-value pin, drop the "agy exports no session-command var" comment, add `DefaultAgyInteractiveCommand` to the var-match table <!-- R2, R6 -->
- [x] T004 Update `TestResolveProvider_NonClaudeBuiltIns` in `src/go/fab/internal/agent/agent_test.go`: agy case gains `session: DefaultAgyInteractiveCommand` + `sessionBypass: "--dangerously-skip-permissions"`; rewrite the dispatch-only prose and the pane-eligibility seam assertion <!-- R6 -->

### Phase 2: Rendered reference + `cmd/fab` pins

- [x] T005 Update `src/go/fab/internal/configref/configref.go`: agy block in `providersYAML` gains the interpolated `interactive_command` line; rewrite the `providersSegment` header comment ("agy is dispatch-only"), the block prose ("agy ships NO interactive_command — DISPATCH-ONLY", "agy's is the one still open", opt-in-ahead-of-probe advice), and the per-provider agy note <!-- R2, R4 -->
- [x] T006 Flip the agy rendering pins in `src/go/fab/cmd/fab/config_test.go` (~L759-780): the agy block opens on `interactive_command`; the prose-phrase assertions track the new wording; verify `TestConfigReferenceDocumentsProviderFill` still passes <!-- R6 -->
- [x] T007 Rewrite `TestAgentProviderDispatchOnlyBuiltInsError` in `src/go/fab/cmd/fab/agent_test.go`: agy composes a session command; the no-`interactive_command` error path stays covered by `TestAgentProviderNoInteractiveCommandErrors` (user-defined provider) <!-- R6 -->
- [x] T008 Flip the agy cases in `src/go/fab/cmd/fab/dispatch_start_test.go` (`modeCommand` pane for agy now composes) and `src/go/fab/cmd/fab/pane_open_test.go` (`--provider agy` resolution error case → user-defined provider or success assertion) <!-- R6 -->

### Phase 3: Specs + kit skills sweep

- [x] T009 Rewrite `docs/specs/stage-models.md`: the "**One of the four is dispatch-only**" section (all four pane-capable; first-run behavior answered for agy), the agy YAML sample (`interactive_command` added, "NO interactive_command" comment removed), the kimi-probe follow-on paragraph, and the § config-reference sample (~L446) <!-- R4, R5 -->
- [x] T010 Update `docs/specs/architecture.md` (~L238-285: agy block comment + per-provider note) and `docs/specs/config.md` (~L357-364: the `--json` providers description) to the shipped state <!-- R4 -->
- [x] T011 Rewrite the agy dictionary entry in `src/kit/skills/_cli-agents.md` (**Dispatch-only.** paragraph, the user-config block, the caveat, and the § Spawn Composition "three of the four" note) + mirror `docs/specs/skills/SPEC-_cli-agents.md` (agy row, § Spawn Composition row, trailing capability paragraph) <!-- R4, R5 -->
- [x] T012 Update `src/kit/skills/_cli-fab.md` (§ fab config explain providers paragraph, § fab agent error bullet "since 260810-ki9v that is agy alone") + mirror `docs/specs/skills/SPEC-_cli-fab.md` (the command-policy paragraph, the `fab config` row, the `fab agent` row) <!-- R4 --> <!-- rework: review cycle 1 — the § fab agent Provider-addressed bullet at _cli-fab.md:1222 still claims `--provider agy` / `--provider kimi` error "because those two ship dispatch grammar only"; fix it (and re-check the SPEC mirror row stays consistent) -->
- [x] T016 Fix `docs/specs/glossary.md:86` — the `providers` glossary row still describes agy as "dispatch grammar only (no `interactive_command`, so no pane capability — its interactive first-run has not been probed)"; rewrite to the shipped state (all four built-ins pane-capable, agy's first-run trust wall an ordinary readiness-gate judgment round). Then re-grep the whole repo (`dispatch-only`, `dispatch grammar only`, `no interactive_command`, `agik`) excluding `fab/changes/` and archives, and fix any remaining stale instance <!-- R4 --> <!-- rework: review cycle 1 — glossary.md was a sweep-class member no task covered -->
- [x] T017 After T012/T016, re-run the affected Go tests if any Go/testdata files changed (none expected) and update plan.md acceptance items A-004/A-017 checkmarks truthfully <!-- R4 --> <!-- rework: review cycle 1 bookkeeping -->

### Phase 4: Memory + backlog

- [x] T013 Rewrite `docs/memory/runtime/providers-and-profiles.md`: built-in table agy row, § "The dispatch-only built-in" → pane-by-default framing, the agy-descends scenario, the embedded-defaults test-pin mention, and DD "agy Is Dispatch-Only Until Its Interactive First Run Is Probed" → present-truth replacement <!-- R5 -->
- [x] T014 Update `docs/memory/runtime/dispatch.md` (the no-`interactive_command` example loses its shipped instance — rephrase as a user-config possibility), `docs/memory/runtime/agent-primitives.md` (agy grammar entry gains the interactive form, pane caveat dropped), and `docs/memory/_shared/configuration.md` (built-in capability enumeration) <!-- R5 -->
- [x] T015 Close backlog `[agik]` in `fab/backlog.md` (both halves shipped: kimi via ki9v, agy via this change) and record the deferred live agy pane probe (post-quota-reset, ~2026-08-16) as a new backlog item <!-- R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` ships `providers.agy.interactive_command: 'agy --dangerously-skip-permissions --model {model}'` with no `{effort}` and no prompt placeholder; `[agik]`/dispatch-only comments are gone from the file
- [x] A-002 R2: `agent.DefaultAgyInteractiveCommand` exists, is interpolated by `internal/configref` into the agy block, and is pinned by value in `defaults_test.go`
- [x] A-003 R3: No agy-specific branch exists in dispatch/readiness Go code (`grep -rn "agy" src/go/fab/internal/dispatch src/go/fab/internal/pane` finds nothing provider-specific — only a generic wrapping comment in `pane/gate.go:350`)
- [x] A-004 R4: `grep -rn "dispatch-only\|dispatch grammar only\|agik" src/ docs/ fab/backlog.md` shows no surviving claim that agy is dispatch-only or pending a probe (rework cycle 1 closed the two missed spots: `_cli-fab.md` § fab agent Provider-addressed bullet and `glossary.md` providers row); every edited kit skill carries its SPEC-mirror edit
- [x] A-005 R5: The three runtime memory files + `_shared/configuration.md` state all four built-ins as pane-capable with the pane-by-default framing, in present truth (no transition narration)
- [x] A-006 R6: All flipped tests pass and the no-`interactive_command` error path remains covered against user-defined providers

### Behavioral Correctness

- [x] A-007 R1: `ResolveProvider(nil, "agy").InteractiveCommand == "agy --dangerously-skip-permissions --model {model}"` (asserted by test)
- [x] A-008 R6: `fab agent --provider agy --print` composes `agy --dangerously-skip-permissions --model <fill>` instead of erroring (asserted by test)
- [x] A-009 R6: `fab config explain` renders the agy block opening on `interactive_command` with YAML-single-quoted grammar (asserted by test)

### Removal Verification

- [x] A-010 R4: The "agy deliberately carries NO interactive_command" comment block, the "Backlog [agik] owns that probe" references in `src/`, and the § "The dispatch-only built-in" memory section no longer exist

### Scenario Coverage

- [x] A-011 R1: `go test ./src/go/fab/internal/agent/` exercises agy pane eligibility (`InteractiveCommand` non-empty, bypass flag present, by-value pin)
- [x] A-012 R6: `go test ./src/go/fab/cmd/fab/` exercises the agy session composition and the user-defined-provider error path

### Edge Cases & Error Handling

- [x] A-013 R6: A user-defined provider with only `headless_command` still errors with `configure providers.<name>.interactive_command` on `fab agent --provider` / `fab pane open` (covered by test)
- [x] A-014 R1: A user override of `providers.agy.interactive_command` still wins per-field over the new built-in (presence=intent merge — covered by existing `TestResolveProvider_ProfilesMerge`-family behavior; no regression)

### Code Quality

- [x] A-015 Pattern consistency: the agy export/comment/test shapes mirror the codex/kimi siblings (`DefaultKimiInteractiveCommand` treatment)
- [x] A-016 No unnecessary duplication: rendered-reference prose interpolates the canonical vars; no literal copy of the command string outside `defaults.yaml` and the test pin
- [x] A-017 Mirror sweep: `src/kit/skills/_cli-agents.md` and `_cli-fab.md` edits pair with `SPEC-_cli-agents.md`/`SPEC-_cli-fab.md`; aggregate specs (`stage-models.md`, `architecture.md`, `config.md`, `glossary.md` — the last added in rework cycle 1) and memory files updated in the same change
- [x] A-018 Canonical sources only: no edit under `.claude/skills/`; `gofmt -l` clean on every touched `.go` file

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Verification constraint (from intake): agy quota is exhausted until ~2026-08-16 — no live pane run in this change; the live probe is the backlog follow-up recorded by T015.

## Deletion Candidates

- None — this change already deletes the prose its flip made redundant (the "Dispatch-only" section and the user-side `providers.agy.interactive_command` opt-in block in `_cli-agents.md`, the retired defaults.yaml comment blocks, and the dispatch-only test pins); no Go symbols, files, or config keys became unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Command value `agy --dangerously-skip-permissions --model {model}` verified against the installed CLI (`agy --help`: both flags exist; `--effort` exists but stays unused per the ID-suffix rule) | Installed binary confirms the flags; intake settled the shape; mirrors the headless posture | S:90 R:85 A:95 D:90 |
| 2 | Certain | Error-path test coverage moves to user-defined dispatch-only providers rather than deleting the cases | `TestAgentProviderNoInteractiveCommandErrors` already establishes the pattern; the error path itself is unchanged code | S:85 R:90 A:90 D:85 |
| 3 | Confident | Rendered agy block orders `interactive_command` first (codex/kimi order) with a short trailing comment | Matches the existing block convention; exact comment wording is presentational | S:70 R:90 A:85 D:75 |
| 4 | Confident | Backlog `[agik]` is closed by this change (kimi half shipped in ki9v, agy half here) and the deferred live probe becomes a new backlog item | Intake records agik as shipped (PR #564) except the agy flip, and the live probe as "a recorded follow-up" | S:75 R:85 A:75 D:70 |
| 5 | Certain | Memory files are edited during apply (the intake's What Changes items 3–4 make the prose correction the bulk of this change); hydrate still owns index regen and `set-summary` | Intake explicitly scopes memory edits into the change; hydrate's own pass follows the pipeline | S:85 R:80 A:85 D:85 |
| 6 | Confident | `docs/specs/harness-adapters.md`, `_preamble.md`, `SPEC-_preamble.md`, and `internal/dispatch` need no edit — they carry only the generic descent-reason vocabulary, which stays valid for user-defined providers | Grep shows no agy-specific or "shipped dispatch-only" claim in those files | S:75 R:85 A:85 D:75 |

6 assumptions (3 certain, 3 confident, 0 tentative).
