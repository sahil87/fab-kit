# Plan: Kimi Pane Enablement — Box-Drawing-Tolerant Squeeze + Ship interactive_command

**Change**: 260810-ki9v-kimi-pane-enablement
**Intake**: `intake.md`

## Requirements

### Dispatch: Pane echo verification

#### R1: `squeeze` MUST drop box-drawing runes as well as whitespace
`squeeze` (`src/go/fab/internal/dispatch/gate.go`) SHALL remove every rune in the Unicode
box-drawing block (U+2500–U+257F) in addition to every whitespace rune, so `countWrapped`
tolerates a TUI whose input box draws vertical side borders between the halves of a wrapped
line.

- **GIVEN** a pane capture in which the needle is split across two lines bordered by `│`
  (so the raw capture interleaves `││` between the wrapped halves)
- **WHEN** `countWrapped(capture, needle)` runs
- **THEN** it returns 1 — the same answer it returns for the borderless claude/agy shapes
- **AND** `Probe` reports `ready` and `newlyEchoed` reports true for such a capture

#### R2: The narrow range MUST be the whole normalization change
The drop SHALL be exactly the U+2500–U+257F range; no broader class (punctuation,
all non-alphanumerics) SHALL be normalized away, so the verifier's false-positive surface
grows only by the frame runes a boxed TUI actually draws.

- **GIVEN** a capture containing ordinary ASCII punctuation the needle does not have
- **WHEN** `countWrapped` compares squeezed strings
- **THEN** that punctuation still participates in the match (no false positive from it)
- **AND** a wrong answer remains a loud double failure into the gate's escalation, never a
  false success

#### R3: `countWrapped`'s comment MUST record the kimi probe
The comment block above `countWrapped` SHALL state that kimi is probed (2026-08-10, kimi
0.34.0: `│` side borders interleaving `││` across a wrap) and that dropping box-drawing runes
is what admits boxed TUIs — replacing the "kimi is unprobed and rides backlog [agik]"
prediction.

- **GIVEN** a reader of `gate.go`
- **WHEN** they read the `countWrapped` comment
- **THEN** it describes the probed provider set and the box-rune tolerance, with no claim
  that kimi is unprobed

### Agent: kimi provider capability

#### R4: kimi MUST ship a built-in `interactive_command`
`src/go/fab/internal/agent/defaults.yaml` SHALL carry
`providers.kimi.interactive_command: 'kimi --auto -m {model}'`, and `internal/agent` SHALL
export it as `DefaultKimiInteractiveCommand` sourced from the parsed file (no literal copy),
matching the existing `DefaultCodexInteractiveCommand` pattern.

- **GIVEN** a binary built from this tree with no `providers:` config at all
- **WHEN** `agent.ResolveProvider(nil, "kimi")` runs
- **THEN** `InteractiveCommand` is `kimi --auto -m {model}`
- **AND** `DefaultKimiInteractiveCommand` equals the value parsed from `defaults.yaml`
- **AND** kimi still ships NO fills, so the empty `{model}` drops the `-m` token pair

#### R5: The roster assertions MUST flip for kimi and hold for agy
The defaults/agent tests SHALL assert kimi's `interactive_command` is PRESENT with that exact
value, and SHALL keep asserting agy's is ABSENT (PR #564 is not merged in this tree).

- **GIVEN** the test suite in `internal/agent`
- **WHEN** it checks the built-in interactive_command roster
- **THEN** claude, codex and kimi are asserted present and agy asserted absent
- **AND** the comments state kimi's probe result rather than backlog `[agik]`'s pending probe

#### R6: kimi MUST be session- and pane-eligible at the consuming seams
`fab agent --provider kimi` SHALL compose a session command instead of erroring, and
`modeCommand(ModePane, …)` SHALL compose for kimi instead of raising the
`providers.kimi.interactive_command` hint; agy SHALL keep both errors.

- **GIVEN** a repo with no `providers:` config
- **WHEN** `fab agent --provider kimi --print` runs
- **THEN** it prints `kimi --auto` (the empty model drops the `-m` pair) and exits 0
- **WHEN** pane-mode composition runs for kimi
- **THEN** it composes without error, while agy still errors with its config-key hint

### Config reference: rendered providers block

#### R7: The rendered reference MUST show kimi's `interactive_command`
`internal/configref`'s providers block SHALL render kimi's `interactive_command` line above its
`headless_command` (the codex ordering), interpolated from `agent.DefaultKimiInteractiveCommand`
with no literal copy, and the block SHALL stay valid YAML when uncommented.

- **GIVEN** `fab config explain` output
- **WHEN** the kimi block is read
- **THEN** it opens on `interactive_command:` carrying the YAML-single-quoted built-in value,
  followed by `headless_command:`
- **AND** the byte-stable rendering and the fence tests still pass

#### R8: The reference prose MUST stop calling kimi dispatch-only
The providers-block prose (the roster sentence and the per-provider kimi note) SHALL name agy
as the only dispatch-only built-in and SHALL record kimi's shipped interactive command plus the
first-run trust wall the readiness gate clears.

- **GIVEN** the rendered reference
- **WHEN** the roster and kimi notes are read
- **THEN** neither claims kimi ships no `interactive_command` nor that its input echo is unprobed
- **AND** the `DISPATCH-ONLY` explanation survives for agy

### Documentation: roster sweep

#### R9: Every roster claim in the sweep class MUST be updated
The "agy and kimi are dispatch-only" claim SHALL be corrected wherever it appears in kit skills,
their SPEC mirrors, and the aggregate specs: `src/kit/skills/_cli-agents.md`,
`src/kit/skills/_cli-fab.md`, `docs/specs/skills/SPEC-_cli-agents.md`,
`docs/specs/skills/SPEC-_cli-fab.md`, `docs/specs/stage-models.md`, `docs/specs/architecture.md`,
`docs/specs/config.md`, `docs/specs/glossary.md`.

- **GIVEN** a repo-wide grep for `dispatch-only` / `DISPATCH-ONLY` / "not been probed" /
  roster-count phrasings ("only claude and codex ship")
- **WHEN** the sweep completes
- **THEN** every surviving occurrence is about agy alone (or about the general concept), and
  none claims kimi lacks an `interactive_command`
- **AND** the kimi dictionary entry records the shipped interactive form and the first-run
  trust wall (`parked` → one Enter → ready, remembered per folder)

### Non-Goals

- Any `{prompt}` placeholder or `spawn.DeliverPrompt` work — that is PR #564 (agik) territory;
  kimi carries no `{prompt}` either way.
- agy's capability flip — this change touches the kimi half only.
- `docs/memory/` updates — they belong to the hydrate stage.
- Relocating the gate code (`260810-1lah` is gated on this change merging).

### Design Decisions

#### Range-based box-drawing drop
**Decision**: extend `squeeze` to drop runes in U+2500–U+257F alongside whitespace.
**Why**: the needle (a `ReadySentinel` or a prompt-file pointer line) never legitimately
contains box-drawing runes, so dropping them from both sides cannot mask a genuine mismatch;
it is the narrowest normalization that admits a side-bordered input box.
**Rejected**: verifying submission instead of echo (bigger contract change, loses the pre-Enter
safety check); per-provider delivery choreography (unautomatable); a broader
"drop all non-alphanumerics" normalization (raises false-positive surface with no probed need).
*Introduced by*: 260810-ki9v-kimi-pane-enablement

#### Ship `kimi --auto -m {model}` verbatim
**Decision**: ship the exact command the user's system config has been running.
**Why**: proven end-to-end through the wll4 pipeline's kimi pane workers; kimi ships no fills,
so `{model}` resolves empty and the token-drop rule removes `-m`, leaving kimi on the user's own
`default_model`.
**Rejected**: `kimi --yolo -m {model}` (the doc's illustrative form) — unproven against the
delivery choreography, and `--auto` is what the live probe used.
*Introduced by*: 260810-ki9v-kimi-pane-enablement

## Tasks

### Phase 1: Core Implementation

- [x] T001 Extend `squeeze` in `src/go/fab/internal/dispatch/gate.go` to drop U+2500–U+257F alongside `unicode.IsSpace`, and rewrite the `countWrapped` comment block to record the kimi probe (2026-08-10, kimi 0.34.0, `│` side borders) and the box-rune tolerance <!-- R1 R2 R3 -->
- [x] T002 Add wrap cases to `src/go/fab/internal/dispatch/gate_test.go` covering a kimi-style boxed wrap for both call sites — the `Probe` sentinel echo and `newlyEchoed`/`Deliver`'s pointer echo — alongside the existing borderless case <!-- R1 R2 -->
- [x] T003 Add `providers.kimi.interactive_command: 'kimi --auto -m {model}'` to `src/go/fab/internal/agent/defaults.yaml` and rewrite the kimi block comment plus the providers-block roster note (agy-only dispatch-only) <!-- R4 -->
- [x] T004 Export `DefaultKimiInteractiveCommand` from `src/go/fab/internal/agent/agent.go` sourced from `defaultProviders[providerKimi].InteractiveCommand`, and update the surrounding roster comments (`defaultProviders` doc, the non-claude command var block) <!-- R4 -->

### Phase 2: Test Flips

- [x] T005 Flip the roster assertions in `src/go/fab/internal/agent/defaults_test.go` — kimi's `interactive_command` present with the exact value, agy's still absent, `DefaultKimiInteractiveCommand` added to the command-var wiring table <!-- R5 -->
- [x] T006 Flip `TestResolveProvider_NonClaudeBuiltIns` and the pane-eligibility loop in `src/go/fab/internal/agent/agent_test.go` so kimi asserts its session command and agy alone asserts ineligibility <!-- R5 -->
- [x] T007 Update `src/go/fab/cmd/fab/agent_test.go` — `TestAgentProviderDispatchOnlyBuiltInsError` narrows to agy, plus a positive case asserting `fab agent --provider kimi --print` yields `kimi --auto` <!-- R6 -->
- [x] T008 Update `TestModeCommand_DispatchOnlyBuiltInsAreHeadlessOnly` in `src/go/fab/cmd/fab/dispatch_start_test.go` so kimi composes for pane and agy alone errors <!-- R6 -->

### Phase 3: Rendered Reference

- [x] T009 Render kimi's `interactive_command` in `providersYAML` and rewrite the roster/per-provider prose in `providersSegment` (`src/go/fab/internal/configref/configref.go`), including the `providersSegment` doc comment <!-- R7 R8 -->
- [x] T010 Update `TestConfigReferenceDocumentsBuiltInProviders` in `src/go/fab/cmd/fab/config_test.go` — kimi's block now opens on `interactive_command`, its command var is asserted rendered, and the dispatch-only prose assertions narrow to agy <!-- R7 R8 -->

### Phase 4: Documentation Sweep

- [x] T011 [P] Sweep `src/kit/skills/_cli-agents.md` (§ Spawn Composition roster note, § Dictionary Discipline `**Built-ins:**` bullet, § kimi dictionary entry) and its mirror `docs/specs/skills/SPEC-_cli-agents.md` <!-- R9 -->
- [x] T012 [P] Sweep `src/kit/skills/_cli-fab.md` (§ providers roster prose, § fab agent no-interactive_command error, § deliver echo-tolerance claim) and its mirror `docs/specs/skills/SPEC-_cli-fab.md` <!-- R9 -->
- [x] T013 [P] Sweep the aggregate specs carrying the roster claim: `docs/specs/stage-models.md`, `docs/specs/architecture.md`, `docs/specs/config.md`, `docs/specs/glossary.md` <!-- R9 -->
- [x] T014 Run `gofmt -l` over every touched Go file and the affected package tests (`./internal/dispatch/... ./internal/agent/... ./internal/configref/... ./internal/configupgrade/... ./cmd/fab/... ./internal/spawn/...`), then re-grep the retired claims repo-wide <!-- R1 R4 R7 R9 -->

## Execution Order

- T001 blocks T002; T003 blocks T004, which blocks T005–T010
- T011–T013 are independent of each other and of the Go work
- T014 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `squeeze` drops U+2500–U+257F alongside whitespace, and `countWrapped` returns 1 for a `│`-bordered wrapped needle
- [x] A-002 R3: the `countWrapped` comment records the 2026-08-10 kimi probe and the box-rune tolerance, with no "unprobed" claim
- [x] A-003 R4: `defaults.yaml` carries `providers.kimi.interactive_command: 'kimi --auto -m {model}'` and `agent.DefaultKimiInteractiveCommand` exports it without a literal copy
- [x] A-004 R7: the rendered reference's kimi block opens on an `interactive_command` line interpolated from the agent var
- [x] A-005 R9: no file in the sweep class claims kimi is dispatch-only or that its input echo is unprobed — re-verified cycle 2: `_cli-agents.md:134` now reads "codex and kimi are non-native (pane + headless); agy is headless-only"; repo-wide greps for `dispatch-only`/`headless-only`/`not been probed`/`unprobed`/`agy and kimi` leave every surviving occurrence about agy alone

### Behavioral Correctness

- [x] A-006 R5: `internal/agent` tests assert kimi's interactive_command PRESENT and agy's ABSENT
- [x] A-007 R6: `fab agent --provider kimi --print` yields `kimi --auto`; `--provider agy` still errors with the config-key hint
- [x] A-008 R6: pane-mode composition succeeds for kimi and still errors for agy
- [x] A-009 R8: the reference prose names agy as the sole dispatch-only built-in and records kimi's shipped command

### Scenario Coverage

- [x] A-010 R1: a test exercises the boxed-wrap capture through `Probe` (sentinel echo) and through delivery (`newlyEchoed`/`Deliver` pointer echo)
- [x] A-011 R2: a test pins that the drop is range-scoped — non-box characters still participate in the match

### Edge Cases & Error Handling

- [x] A-012 R4: kimi still ships NO fills, so the empty `{model}` drops the `-m` pair (existing token-drop behavior unchanged)
- [x] A-013 R7: uncommenting the rendered kimi block still yields valid YAML (single quotes doubled by `YAMLSingleQuoted`)

### Code Quality

- [x] A-014 Pattern consistency: new code follows the naming and structural patterns of surrounding code (kimi's interactive command var mirrors `DefaultCodexInteractiveCommand`)
- [x] A-015 No unnecessary duplication: command strings are interpolated from the canonical `agent.Default*` vars, never copied as literals
- [x] A-016 Canonical source only: kit edits land in `src/kit/skills/`, never `.claude/skills/`
- [x] A-017 SPEC-mirror sync: every `src/kit/skills/*.md` edit carries its `docs/specs/skills/SPEC-*.md` update
- [x] A-018 Go changes ship tests: every touched `.go` file has accompanying test updates, and `gofmt -l` reports nothing
- [x] A-019 Sibling & mirror sweeps: the roster claim was grepped repo-wide and every occurrence in the class updated — cycle-1 miss (`_cli-agents.md:134`) fixed; skill and `SPEC-_cli-agents.md:59` now agree. The cycle-2 must-fix (deliver's echo-tolerance wording) is fixed in both members: `_cli-fab.md:680` and `SPEC-_cli-fab.md:33` now say the echo check ignores whitespace AND box-drawing runes. Re-verified cycle 3: `git grep` for `kimi` outside `fab/`, `docs/memory/`, `src/`, `docs/specs/` returns only `.gitignore`; all four SPEC/skill pairs changed together (`SPEC-_cli-agents.md` 37/56/59, `SPEC-_cli-fab.md` 13/31/33/45)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/kit/skills/_cli-agents.md` § kimi — the `providers: kimi: interactive_command: 'kimi --yolo -m {model}'` opt-in YAML block and its surrounding "add one yourself" prose were made redundant by shipping the built-in and are already deleted by this change. The parallel block under § agy stays live: agy is still dispatch-only.
- `src/go/fab/internal/agent/agent.go:206-208` — `DefaultKimiHeadlessCommand`'s "kimi likewise ships no interactive command, so this is its only invocation grammar" clause, made false by the flip and already deleted by this change. The parallel clause on `DefaultAgyHeadlessCommand:203-204` stays live: agy is still dispatch-only.
- None otherwise — the change flips a capability on and widens one verifier predicate; it retires no function, branch, or config key. `boxDrawing` is new with exactly one call site (`squeeze`), and the whitespace-only condition it extends was one predicate, not separable code. The `{agy, kimi}` loops in `defaults_test.go`, `agent_test.go`, `cmd/fab/agent_test.go` and `dispatch_start_test.go` were narrowed to agy rather than deleted, which is correct: the dispatch-only rule still has a built-in subject.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | agy's `interactive_command` stays ABSENT (PR #564 unmerged in this tree) | Verified at branch time: `defaults.yaml` carries no agy `interactive_command` and `git log` shows no agik commit — intake Assumption 7's expected branch | S:95 R:85 A:95 D:95 |
| 2 | Certain | kimi's rendered block orders `interactive_command` before `headless_command` | Matches claude's and codex's shipped ordering in `providersYAML`; the config_test assertion that agy/kimi open on `headless_command` is the thing being flipped | S:80 R:90 A:95 D:90 |
| 3 | Certain | Export `DefaultKimiInteractiveCommand` rather than inlining the string in configref | The package's no-duplicate-literal rule is explicit in `agent.go` and `configref.go`; codex is the exact precedent | S:90 R:85 A:95 D:95 |
| 4 | Confident | `docs/specs/harness-adapters.md` needs no edit | Grepped: it carries no per-provider roster claim, only the provider-neutral `interactive_command` prerequisite — the intake listed it conditionally ("if they carry the claim") | S:70 R:85 A:90 D:80 |
| 5 | Confident | The seam tests that loop over `{agy, kimi}` narrow to agy plus a new positive kimi case, rather than being deleted | The dispatch-only rule still has a built-in subject (agy) and a user-defined one; deleting the loop would drop coverage the flip does not invalidate | S:75 R:80 A:85 D:80 |
| 6 | Confident | The wrap tests use a synthetic kimi-shaped capture, not a recorded fixture | `gate_test.go`'s existing wrap case is synthetic (`ReadySentinel[:5] + "\n" + …`); a byte fixture would add a file the package has no precedent for | S:70 R:85 A:85 D:75 |

6 assumptions (3 certain, 3 confident, 0 tentative).
