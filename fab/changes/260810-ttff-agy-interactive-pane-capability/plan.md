# Plan: agy Interactive/Pane Capability — Ship the Template, Retire the Dispatch-Only Framing

**Change**: 260810-ttff-agy-interactive-pane-capability
**Intake**: `intake.md`

## Requirements

### Agent: agy provider capability

#### R1: agy MUST ship a built-in `interactive_command`
`src/go/fab/internal/agent/defaults.yaml` SHALL carry
`providers.agy.interactive_command: 'agy --dangerously-skip-permissions --model {model}'`, and
`internal/agent` SHALL export it as `DefaultAgyInteractiveCommand` sourced from the parsed file
(no literal copy), matching the existing `DefaultCodexInteractiveCommand` /
`DefaultKimiInteractiveCommand` pattern.

- **GIVEN** a binary built from this tree with no `providers:` config at all
- **WHEN** `agent.ResolveProvider(nil, "agy")` runs
- **THEN** `InteractiveCommand` is `agy --dangerously-skip-permissions --model {model}`
- **AND** `DefaultAgyInteractiveCommand` equals the value parsed from `defaults.yaml`
- **AND** the command carries no `{effort}` placeholder — agy's model IDs embed the reasoning level

#### R2: The `defaults.yaml` provider prose MUST record the shipped grammar and the trust wall
The providers-block note and the agy block comment SHALL delete the "agy deliberately carries NO
interactive_command" claim and the "Backlog [agik] owns that probe" reference, and SHALL instead
record that agy's first-run trust prompt is an ordinary readiness-gate judgment round (the kimi
precedent), noting the trust store is additionally user-seedable at
`~/.gemini/antigravity-cli/settings.json` (exact-match paths).

- **GIVEN** a reader of `defaults.yaml`
- **WHEN** they read the providers-block note and the agy block
- **THEN** neither claims agy lacks an `interactive_command` nor cites backlog `[agik]`
- **AND** the trust wall is described as gate-cleared, not as a blocker

#### R3: The roster pins MUST flip by VALUE for agy
`defaults_test.go` and `agent_test.go` (`internal/agent`) SHALL assert agy's `interactive_command`
is PRESENT with that exact value — the same pin-by-value treatment kimi's grammar gets — and SHALL
assert all four built-ins are pane-capable.

- **GIVEN** the test suite in `internal/agent`
- **WHEN** it checks the built-in `interactive_command` roster
- **THEN** claude, codex, agy and kimi are all asserted present, agy's pinned by value
- **AND** no assertion or comment claims a built-in is dispatch-only or unprobed

#### R4: agy MUST be session- and pane-eligible at the consuming seams
`fab agent --provider agy` SHALL compose a session command instead of erroring, and
`modeCommand(ModePane, …)` SHALL compose for agy instead of raising the
`providers.agy.interactive_command` hint. The no-`interactive_command` error SHALL remain reachable
and tested — via a **project-defined** provider, which is now its only subject.

- **GIVEN** a repo with no `providers:` config
- **WHEN** `fab agent --provider agy --print` runs
- **THEN** it prints `agy --dangerously-skip-permissions` (the empty model drops the `--model` pair)
  and exits 0
- **WHEN** pane-mode composition runs for agy
- **THEN** it composes the shipped interactive grammar without error
- **AND** `fab pane open --provider <project-defined grammar-less provider>` still raises the
  `providers.<name>.interactive_command` hint verbatim

### Config reference: rendered providers block

#### R5: The rendered reference MUST show agy's `interactive_command`
`internal/configref`'s providers block SHALL render agy's `interactive_command` line above its
`headless_command` (the codex/kimi ordering), interpolated from `agent.DefaultAgyInteractiveCommand`
with no literal copy, and the block SHALL stay valid YAML when uncommented.

- **GIVEN** `fab config explain` output
- **WHEN** the agy block is read
- **THEN** it opens on `interactive_command:` carrying the YAML-single-quoted built-in value,
  followed by `headless_command:`
- **AND** the byte-stable rendering and the fence tests still pass

#### R6: The reference prose MUST retire the DISPATCH-ONLY framing entirely
The providers-block prose (roster sentence, capability enumeration, per-provider agy note) SHALL
state that **all four built-ins are pane-capable** and SHALL NOT describe any shipped provider as
dispatch-only. It SHALL preserve the fact that an absent `interactive_command` in a **user's**
`providers:` block is a valid configuration the descent ladder handles.

- **GIVEN** the rendered reference
- **WHEN** the roster and agy notes are read
- **THEN** no phrase claims agy ships no `interactive_command` or that its first run is unprobed
- **AND** the descent-ladder consequence of an absent `interactive_command` survives as a
  user-config possibility, not as a shipped state

### Documentation: capability-framing sweep

#### R7: Every dispatch-only roster claim in the sweep class MUST be corrected
The "agy is dispatch-only / ships no `interactive_command`" claim SHALL be corrected wherever it
appears in kit skills, their SPEC mirrors, and the aggregate specs:
`src/kit/skills/_cli-agents.md`, `src/kit/skills/_cli-fab.md`,
`docs/specs/skills/SPEC-_cli-agents.md`, `docs/specs/skills/SPEC-_cli-fab.md`,
`docs/specs/stage-models.md`, `docs/specs/architecture.md`, `docs/specs/config.md`,
`docs/specs/glossary.md`.

- **GIVEN** a repo-wide grep for `dispatch-only` / `DISPATCH-ONLY` / `no interactive_command` /
  `unprobed` / `[agik]` outside `fab/changes/` and `docs/memory/`
- **WHEN** the sweep completes
- **THEN** every surviving occurrence is about the general mechanism or a user-defined provider —
  none about a built-in
- **AND** the agy dictionary entry in `_cli-agents.md` records the shipped interactive form, the
  trust wall as a gate judgment round, and the optional trust-store seeding

#### R8: The corrected capability MODEL MUST be stated where the wrong one lived
`docs/specs/stage-models.md`'s "**One of the four is dispatch-only**" section SHALL be replaced with
the pane-by-default framing: **every agent CLI has an interactive mode, so pane capability is the
default expectation for any provider — it rides launch grammar plus the generic readiness gate;
headless grammar is what varies per CLI and needs probing.** The per-provider open question is
first-run behavior, now answered for all four.

- **GIVEN** a reader of `docs/specs/stage-models.md`
- **WHEN** they read the provider-capability section
- **THEN** it teaches pane-by-default and names first-run behavior (not prompt grammar, not
  "interactive mode") as the per-provider unknown
- **AND** it records that agy's first-run wall is an ordinary readiness-gate judgment round, with
  the trust store seedable as a user-side optimization

### Non-Goals

- `docs/memory/` updates — they belong to the **hydrate** stage (the intake's Affected Memory list
  is hydrate's input). Apply touches `src/` and `docs/specs/` only.
- Trust-store seeding code — the generic gate suffices (intake Assumption 4).
- A live agy pane-worker run — agy quota is exhausted until ~2026-08-16 (intake Assumption 5); a
  recorded follow-up, not part of this change's verification.
- `fab/backlog.md` edits — backlog bookkeeping is committed outside the pipeline in this repo
  (verified: neither `260810-ki9v` nor `260810-agik` touched it).

### Design Decisions

#### Ship `agy --dangerously-skip-permissions --model {model}` verbatim
**Decision**: ship the full-auto interactive grammar with no `{effort}` and no `{prompt}` placeholder.
**Why**: verified against the installed CLI (`agy --help`, v1.1.11 family: `--dangerously-skip-permissions`,
`--model`, and a separate `--effort` that would fight the model-ID suffix). It mirrors agy's own
shipped headless posture, which is the convention all four built-ins follow. No initial-prompt
grammar is needed: since 3oz7/1lah the stage prompt is delivered after launch by
`fab dispatch deliver`, so `-i`/`--prompt-interactive` would add an unused, unverifiable channel.
**Rejected**: `agy -i {prompt}` (the rpsr probe's spawn-time form) — superseded by post-launch
delivery, and `{prompt}`-carrying grammars have their own operator-spawn constraints;
`--effort {effort}` (fights the ID suffix).
*Introduced by*: 260810-ttff-agy-interactive-pane-capability

#### Retire the dispatch-only concept for BUILT-INS, keep it for user config
**Decision**: delete every "one of the four is dispatch-only" claim, but keep the
no-`interactive_command` error, its descent reason, and their tests — re-subjected to a
project-defined provider.
**Why**: the shape is still reachable and still needs its actionable error; what changed is that no
*shipped* provider presents it. Deleting the tests would drop coverage the flip does not invalidate
(the same narrowing ki9v applied, taken one provider further).
**Rejected**: deleting `TestModeCommand_DispatchOnlyBuiltInsAreHeadlessOnly` and the pane-open error
subtest outright (loses the actionable-error guarantee); keeping agy as their subject (the assertion
would be false).
*Introduced by*: 260810-ttff-agy-interactive-pane-capability

#### Pane capability is the default expectation; headless is what varies
**Decision**: state the capability model as "every agent CLI has an interactive mode — pane
capability rides launch grammar plus the generic readiness gate; headless grammar is the
per-CLI variable that needs probing."
**Why**: the old prose conflated "the CLI has an interactive mode" (universally true of a TUI) with
"fab ships an `interactive_command` for it" (a fab decision), and that framing would judge the next
provider by the wrong axis. The four built-ins' headless forms genuinely diverge (stdin vs `-p`
argument, nested shells, per-form approval flags) while their interactive forms are uniform.
**Rejected**: keeping a per-provider "is it pane-capable?" question (answered yes four times over);
framing the trust wall as a capability gap (it is a gate input).
*Introduced by*: 260810-ttff-agy-interactive-pane-capability

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add `interactive_command: 'agy --dangerously-skip-permissions --model {model}'` to the agy block in `src/go/fab/internal/agent/defaults.yaml`, and rewrite the providers-block note plus the agy block comment (delete the "agy deliberately carries NO interactive_command" paragraph and every `[agik]` reference; record the trust wall as a readiness-gate judgment round and the seedable trust store) <!-- R1 R2 -->
- [x] T002 Export `DefaultAgyInteractiveCommand` from `src/go/fab/internal/agent/agent.go` sourced from `defaultProviders[providerAgy].InteractiveCommand`, and rewrite the surrounding roster comments (the non-claude command-var block header, the `DefaultAgyHeadlessCommand` doc, and the `defaultProviders` table doc's agy bullet) <!-- R1 R2 -->

### Phase 2: Test Flips

- [x] T003 Flip the roster assertions in `src/go/fab/internal/agent/defaults_test.go` — agy's `interactive_command` present and pinned BY VALUE, all four built-ins in the presence loop, `DefaultAgyInteractiveCommand` added to the command-var wiring table, and the `[agik]`/dispatch-only comments rewritten <!-- R3 -->
- [x] T004 Flip `TestResolveProvider_NonClaudeBuiltIns` in `src/go/fab/internal/agent/agent_test.go` — agy's `session` becomes `DefaultAgyInteractiveCommand` with its `--dangerously-skip-permissions` session bypass, the trailing agy-ineligibility assertion becomes a pane-eligibility assertion, and the `{effort}`-free pin extends to the interactive grammar <!-- R3 R1 -->
- [x] T005 Update `src/go/fab/cmd/fab/agent_test.go` — replace `TestAgentProviderDispatchOnlyBuiltInsError` with a positive `TestAgentProviderBuiltinAgyNoConfig` asserting `fab agent --provider agy --print` yields `agy --dangerously-skip-permissions`, and re-point the two "the built-in agy half of the same rule is …" cross-references on the project-defined-provider tests <!-- R4 -->
- [x] T006 Convert `TestModeCommand_DispatchOnlyBuiltInsAreHeadlessOnly` in `src/go/fab/cmd/fab/dispatch_start_test.go` into `TestModeCommand_AgyComposesBothModes` (mirroring the kimi test: both rungs compose from the shipped defaults), and rewrite the shape-(b) tie-in comment so it names the synthetic `cli` provider as the grammar-less shape's only remaining subject <!-- R4 --> <!-- deviation (post-pass annotation): shipped as TestModeCommand_NonClaudeBuiltInsComposeBothModes (dispatch_start_test.go:612), which also absorbed+deleted TestModeCommand_KimiComposesBothModes and added codex coverage — strict improvement over the per-provider mirror; R4/A-012 met -->
- [x] T007 Re-subject the "provider without interactive_command is a hard error" subtest in `src/go/fab/cmd/fab/pane_open_test.go` to a project-defined provider (a temp fab repo whose `providers.myagent` carries `headless_command` alone), keeping the verbatim error assertion <!-- R4 -->

### Phase 3: Rendered Reference

- [x] T008 Render agy's `interactive_command` in `providersYAML` and rewrite the roster/capability/per-provider prose in `providersSegment` plus the `providersSegment` and `YAMLSingleQuoted`/`profilesLines` doc comments (`src/go/fab/internal/configref/configref.go`) <!-- R5 R6 -->
- [x] T009 Update `TestConfigReferenceDocumentsBuiltInProviders` in `src/go/fab/cmd/fab/config_test.go` — agy's block now opens on `interactive_command`, `agent.DefaultAgyInteractiveCommand` is asserted rendered, the `agy ships NO interactive_command` / `DISPATCH-ONLY` prose assertions are replaced with the pane-capable framing, and the bad-substring guard keeps agy free of `{effort}` on BOTH command lines <!-- R5 R6 -->

### Phase 4: Documentation Sweep

- [x] T010 [P] Sweep `src/kit/skills/_cli-agents.md` (§ Spawn Composition roster note L61, § Dictionary Discipline `**Built-ins:**` bullet, § agy Interactive row + § Dispatch-only block + opt-in YAML + caveat) and its mirror `docs/specs/skills/SPEC-_cli-agents.md` (rows 37, 55, **56**, 59) <!-- R7 --> <!-- rework cycle 1 DONE: SPEC-_cli-agents.md:56 kimi row "unlike agy" → "like every built-in" (mirror-only — the canonical skill never carried the contrast); contrastive-phrase class re-grepped clean -->
- [x] T011 [P] Sweep `src/kit/skills/_cli-fab.md` (§ providers roster prose L413, § fab agent provider-addressed L1222, § fab agent error L1238) and its mirror `docs/specs/skills/SPEC-_cli-fab.md` (rows 13, 30, 45) <!-- R7 -->
- [x] T012 Rewrite the "One of the four is dispatch-only" section of `docs/specs/stage-models.md` as the pane-by-default capability model, and update its inline-YAML samples (the `agy:` block at ~L245 and the commented sample at ~L446) plus the full-auto-posture paragraph <!-- R7 R8 -->
- [x] T013 [P] Sweep the remaining aggregate specs: `docs/specs/architecture.md` (rendered-reference mirror ~L227–290), `docs/specs/config.md` (~L224 capability sentence, ~L358–363 `--json` default description), `docs/specs/glossary.md` (`providers` row) <!-- R7 --> <!-- rework cycle 2 DONE: glossary.md:86 splice repaired (kimi's no-fills parenthetical moved back onto the kimi clause; the absent-interactive_command sentence now ends at "…the ladder descends past"); config.md:224 gained the environment-prerequisite clause; config.md:358 comma-splice fixed and "both commands" → "both command fields"; architecture.md 99-col line re-wrapped to the block's ~78-col convention and the agy YAML inline comment trimmed into the block's width family; deferred-live-probe clause added to defaults.yaml, _cli-agents.md § agy and its SPEC mirror; config_test.go retired-framing guard narrowed to discriminating literals -->
- [x] T014 Run `gofmt -l` over every touched Go file, run the affected package tests (`./internal/agent/... ./internal/configref/... ./internal/dispatch/... ./internal/configupgrade/... ./internal/spawn/... ./cmd/fab/...`), then re-grep the retired claims repo-wide (`dispatch-only`, `DISPATCH-ONLY`, `agik`, `unprobed`, `no interactive_command`, plus the contrastive phrase class: `unlike agy`, `agy alone`, `except agy`, `only built-in`) outside `fab/changes/` and `docs/memory/` <!-- R1 R3 R4 R5 R6 R7 R8 --> <!-- rework cycles 1-2 DONE: re-run last each cycle with the widened grep set — gofmt clean, six affected packages green (incl. ./cmd/fab/... after the guard narrowing), both the literal and contrastive classes clean; the only surviving literal hits are config_test.go's own guard asserting their absence -->

## Execution Order

- T001 blocks T002, which blocks T003–T009
- T010–T013 are independent of each other and of the Go work
- T014 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` carries `providers.agy.interactive_command: 'agy --dangerously-skip-permissions --model {model}'` and `agent.DefaultAgyInteractiveCommand` exports it without a literal copy
- [x] A-002 R2: the `defaults.yaml` providers-block note and agy block record the shipped grammar and the gate-cleared trust wall, with no `[agik]` reference and no "carries NO interactive_command" claim
- [x] A-003 R5: the rendered reference's agy block opens on an `interactive_command` line interpolated from the agent var
- [x] A-004 R7: no file outside `fab/changes/` and `docs/memory/` claims a built-in provider is dispatch-only or that agy's first run is unprobed — met at cycle 2: the cycle-1 miss (`docs/specs/skills/SPEC-_cli-agents.md:56` kimi row, "Carries a **§ Pane-capable** posture, **unlike agy**") now reads "like every built-in". Re-verified with a grep set widened past retired *literals* to the contrastive phrase class (`unlike agy`, `agy alone`, `except agy`, `but agy`, `agy is the one/only`, `save agy`, `other than agy`): zero hits. The surviving `dispatch-only`/`agik` literals are in `config_test.go`, the guard asserting their absence from the rendered reference — narrowed at cycle 2 from the bare substring `dispatch only` to the discriminating hyphenated forms, so it no longer trips on innocent prose. Re-verified clean at cycle 2 after the glossary/config/architecture edits
- [x] A-005 R8: `docs/specs/stage-models.md` states the pane-by-default capability model and names first-run behavior as the per-provider unknown

### Behavioral Correctness

- [x] A-006 R3: `internal/agent` tests assert agy's `interactive_command` PRESENT and pinned by value, with all four built-ins pane-capable
- [x] A-007 R4: `fab agent --provider agy --print` yields `agy --dangerously-skip-permissions` (verified against a binary built from this tree)
- [x] A-008 R4: pane-mode composition succeeds for agy and returns the shipped interactive grammar
- [x] A-009 R6: the reference prose describes all four built-ins as pane-capable and keeps the absent-`interactive_command` descent as a user-config possibility

### Removal Verification

- [x] A-010 R6: the `DISPATCH-ONLY` / `agy ships NO interactive_command` literals are gone from the rendered reference, and `config_test.go` no longer asserts them
- [x] A-011 R4: no test names a BUILT-IN as the subject of the no-`interactive_command` error; the error itself and its descent reason remain live and covered

### Scenario Coverage

- [x] A-012 R4: a test exercises `modeCommand` for agy on both the headless and pane rungs against the shipped defaults
- [x] A-013 R4: a test exercises the no-`interactive_command` hard error through `fab pane open` against a project-defined grammar-less provider

### Edge Cases & Error Handling

- [x] A-014 R1: agy's interactive grammar carries no `{effort}` placeholder, and agy's fills stay effort-free (its model IDs embed the reasoning level)
- [x] A-015 R5: uncommenting the rendered agy block still yields valid YAML (single quotes doubled by `YAMLSingleQuoted`) — re-parsed the rendered block with `yq`
- [x] A-016 R4: with no fills supplied on the provider-addressed path, the empty `{model}` drops the `--model` token pair while the fixed `--dangerously-skip-permissions` survives

### Code Quality

- [x] A-017 Pattern consistency: new code follows the naming and structural patterns of surrounding code (`DefaultAgyInteractiveCommand` mirrors `DefaultCodexInteractiveCommand`/`DefaultKimiInteractiveCommand`)
- [x] A-018 No unnecessary duplication: command strings are interpolated from the canonical `agent.Default*` vars, never copied as literals
- [x] A-019 Canonical source only: kit edits land in `src/kit/skills/`, never `.claude/skills/`
- [x] A-020 SPEC-mirror sync: every `src/kit/skills/*.md` edit carries its `docs/specs/skills/SPEC-*.md` update (both `_cli-agents.md` and `_cli-fab.md` mirrors edited in this change; row-level accuracy tracked by A-004/A-022)
- [x] A-021 Go changes ship tests: every touched `.go` file has accompanying test updates, and `gofmt -l` reports nothing (full `go test ./...` green in `src/go/fab`)
- [x] A-022 Sibling & mirror sweeps: the capability claim was grepped repo-wide (including `docs/specs/glossary.md` and `src/kit/skills/_cli-fab.md`) and every occurrence in the class updated — met at cycle 2. Cycle-1 lesson: a literal-only grep set cannot see a claim carried by CONTRAST, and the mirror had drifted from the canonical skill in a row the skill never carried (`SPEC-_cli-agents.md:56`), so the SPEC-mirror pair needs reading row-by-row, not just grepping. The sweep now covers both classes; the roster-count phrasings that survive (`three of the four`, `the one built-in`) are all about FILLS, which this change does not touch
- [x] A-023 No migration needed: built-in defaults are embedded data and user `providers.agy.*` overrides still win per-field, so no `src/kit/migrations/` file is required

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- **Open PR #564 (`260809-agik-agy-interactive-command`)** — superseded in whole by this change on the
  contested field: it sets `providers.agy.interactive_command: 'agy … --model {model} -i {prompt}'`,
  the spawn-time form this plan's Design Decisions explicitly reject. Whichever branch merges second
  silently overrides the other's grammar (and would break this change's pin-by-value tests), so #564
  should be closed or rebased-and-narrowed rather than merged alongside. Recorded here, not acted on:
  cross-PR bookkeeping is outside this change's tree.
- `fab/backlog.md:31` (`[agik]` item) — its remaining scope ("Give agy/kimi an `interactive_command`", "probe each CLI live before shipping") is fully satisfied by #564/#566 plus this change; the item can be closed. Declared a plan Non-Goal (backlog bookkeeping is committed outside the pipeline here), so recorded for the archive step rather than deleted now.
- No code deletions — the grammar-less shape (`missingCommandError` in `validatePane`, the `pane unavailable: no interactive_command` descent reason, and both `fab agent` error paths) stays reachable through a user's own `providers:` block, so its branches and tests were re-subjected rather than retired.
- `src/kit/skills/_cli-agents.md` § agy's opt-in `providers.agy.interactive_command` YAML block and its "add one yourself" prose — made redundant by shipping the grammar; already deleted in this change (T010).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Command value `agy --dangerously-skip-permissions --model {model}` (intake Assumption 3, promoted after verification) | Verified against the installed CLI: `agy --help` lists `--dangerously-skip-permissions`, `--model`, and a separate `--effort` that would fight the model-ID suffix | S:95 R:85 A:95 D:95 |
| 2 | Certain | `docs/memory/` edits are hydrate's, not apply's — this plan touches `src/` and `docs/specs/` only | Pipeline convention (constitution: memory is hydrated post-review) and the ki9v precedent, which listed memory as an explicit apply Non-Goal; the intake's Affected Memory section is hydrate's input | S:90 R:90 A:95 D:90 |
| 3 | Certain | No shipped built-in is dispatch-only after this change, so those tests are re-subjected to a project-defined provider rather than deleted | The grammar-less shape is still reachable via user config and still owes an actionable error; ki9v narrowed the same tests rather than deleting them | S:85 R:85 A:90 D:90 |
| 4 | Confident | `fab/backlog.md` is not edited by this change | Verified: neither the ki9v (#566) nor the agik (#564) commit touched `fab/backlog.md`; backlog bookkeeping lands in separate commits, and the intake's Impact list omits it | S:75 R:95 A:85 D:85 |
| 5 | Confident | The agy rendered block orders `interactive_command` before `headless_command` and keeps its inline `# no {effort} flag; nested shell …` note on the headless line | Matches claude/codex/kimi ordering in `providersYAML`; the "dispatch only;" clause of that inline note is the part being retired | S:80 R:90 A:90 D:85 |
| 6 | Confident | `docs/specs/harness-adapters.md`, `docs/specs/skills/SPEC-_preamble.md`, `src/kit/skills/_preamble.md` and the Go non-test files need no edit | Grepped: their `no interactive_command` occurrences are the provider-neutral descent reason and the generic error string, both of which survive unchanged as a user-config path | S:80 R:85 A:90 D:85 |
| 7 | Confident | The live agy pane probe is deferred past the ~2026-08-16 quota reset and recorded as a follow-up, not verified here | Intake Assumption 5; quota exhaustion is external, and the kimi precedent plus pin-by-value tests carry verification | S:80 R:80 A:75 D:80 |

7 assumptions (3 certain, 4 confident, 0 tentative).
