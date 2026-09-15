# Plan: fab setup Interactive Wizard

**Change**: 260811-stpw-setup-interactive-wizard
**Intake**: `intake.md`

## Requirements

### CLI: The bare `fab setup` wizard

#### R1: Bare `fab setup` runs the interactive wizard
Bare `fab setup` MUST replace the "Yet to be implemented" placeholder with the interactive wizard: probe → scope banner → 4-question default path → opt-in advanced section → diff-before-write summary → writes. All detection MUST come from `setupcheck.Run(setupCheckInput())` (the same call `fab setup check` makes) — the wizard SHALL add no probing of its own and never asks a capability question. `fab setup check` and its no-writes invariant MUST remain untouched.

- **GIVEN** a TTY session in any directory (fab repo or not)
- **WHEN** the user runs `fab setup`
- **THEN** the wizard runs the probe once, shows the scope banner, and enters the interview
- **AND** `fab setup check` behaves byte-identically to before this change

#### R2: Scope banner and `--project` retarget
The wizard MUST print a scope banner stating the write target before any question. The default target is the **system tier** (`~/.fab-kit/config.yaml`); a `--project` flag MUST retarget writes to `fab/project/config.yaml`. `--project` outside a fab repo MUST fail with a clear error (no project config to write).

- **GIVEN** a run of bare `fab setup` with no flags
- **WHEN** the banner renders
- **THEN** it names the system tier path and mentions `--project` as the alternative
- **GIVEN** `fab setup --project` inside a fab repo
- **THEN** the banner names `fab/project/config.yaml` and writes go there

#### R3: Default-path provider questions (`agent.session`, `agent.workers`)
Q1 and Q2 MUST ask for `agent.session` and `agent.workers` respectively. Options MUST be the provider names from the probe `Report`, **filtered to detected providers** (`ProviderProbe.Found()` true — undetected providers are dropped, not annotated as missing), each **annotated** with its declared capabilities (e.g. `claude (interactive, headless, native)`). Every question MUST default to the **current effective value with its origin** shown (Enter keeps it), and MUST carry a footer pointing at `fab config explain <key>`. Prompts MUST be bare stdin line reads against the command's `InOrStdin()` — no TUI dependency.

- **GIVEN** providers claude (detected) and codex (binary not on PATH)
- **WHEN** Q1 renders
- **THEN** the option list contains claude with capability annotation and omits codex
- **AND** the prompt shows the current effective `agent.session` value and its origin as the default
- **GIVEN** the user presses Enter
- **THEN** the answer is the current effective value (no change recorded)

#### R4: `dispatch.mode` question with viability filtering and ladder semantics
Q3 MUST ask for `dispatch.mode` with options filtered by viability: `pane` is offered only when the probe's tmux signal is present; `native` and `headless` follow the roster's capability columns (offer a rung only when some detected provider can run it). The question text MUST state the ladder semantics: the value is a preference *ceiling* over `pane → native → headless` — resolution descends and never ascends.

- **GIVEN** `$TMUX` is absent
- **WHEN** Q3 renders
- **THEN** `pane` is not among the options and the ladder-semantics line is present

#### R5: Opt-in advanced section with the skip rule
Q4 MUST offer the advanced section (`Configure advanced options? [y/N]`, default no). When accepted, the section covers `agent.profiles.operator` (provider), `agent.profiles.review` (provider), `dispatch.column_width`, and `dispatch.reap_done` — but MUST **skip** any key whose current value equals the built-in default and was never overridden at any tier (winning tier == built-in defaults). When every advanced question skips, the wizard MUST print a one-line note naming the skipped keys with the `fab config explain <key>` pointer instead of silence.

- **GIVEN** a fresh machine (no overrides anywhere) and the user answers `y` to Q4
- **WHEN** the advanced section runs
- **THEN** no advanced question is asked and the note lists the four keys with the explain pointer
- **GIVEN** `dispatch.column_width` is overridden at the system tier
- **WHEN** the advanced section runs
- **THEN** the `dispatch.column_width` question is asked (default = current value + origin) and the other three skip

#### R6: Diff-before-write summary and zero-write idempotence
After the interview the wizard MUST print a diff summary — one line per key whose answer differs from the current effective value (`<key>: <old> → <new>` plus the target tier) — and confirm before writing. When no answer differs (the all-Enter run), the wizard MUST print a "nothing to change" line and exit 0 **without touching any file** (Constitution III: repeated all-Enter runs are byte-identical no-ops).

- **GIVEN** an interview where every question was answered with Enter
- **WHEN** the wizard reaches the write step
- **THEN** it prints "nothing to change", writes no file, and exits 0
- **GIVEN** the user changed `agent.workers` from `claude` to `codex`
- **THEN** the diff shows `agent.workers: claude → codex`, and writing proceeds only after confirmation

#### R7: Writes reuse the surgical config-set path
Confirmed writes MUST go through the existing `fab config set` write path **in-process** — `configupgrade.SetSystem(path, key, value)` for the system tier, `configupgrade.Set(path, key, value, version)` for the project tier, with the target path from `configMutationPath(system)` — never a whole-file rewrite, never a child-process exec. The existing shadow warning (`warnIfShadowed`) MUST fire per written key, so a write shadowed by a higher tier (e.g. an env var) says so.

- **GIVEN** a confirmed change to `agent.workers` with default (system) scope
- **WHEN** the wizard writes
- **THEN** `~/.fab-kit/config.yaml` gains/updates only that key via the fence-aware engine
- **GIVEN** `FAB_AGENT_WORKERS` is set in the environment
- **THEN** the write succeeds and the shadow warning names the env tier

#### R8: Non-interactive parity — `--defaults` and the non-TTY guard
`fab setup --defaults` MUST run the full flow non-interactively, accepting every question's default (the current effective value) — printing the banner, the resolved answers, and the "nothing to change" summary (a zero-write run by R6). It MUST compose with `--project`. When stdin is not a TTY and `--defaults` was not passed, the wizard MUST fail with a usage hint (mention `--defaults`) rather than hanging on a read. TTY detection follows the existing stdlib-only `os.ModeCharDevice` pattern (`batch_archive.go`).

- **GIVEN** `fab setup --defaults` in CI (no TTY)
- **WHEN** it runs
- **THEN** it completes without reading stdin, writes nothing, and exits 0
- **GIVEN** `echo | fab setup` (non-TTY, no `--defaults`)
- **THEN** it exits non-zero with a hint naming `--defaults`

### Docs: Skill and reference updates

#### R9: Skill/spec/reference prose tracks the new surface
The change MUST update: `src/kit/skills/_cli-fab.md` (bare `fab setup` signature + `--defaults`/`--project` flags — the standing CLI constraint), `src/kit/skills/fab-setup.md` (the `/fab-setup` skill delegates its config-interview portion for the wizard-covered preference keys to `fab setup`; the identity-field create-mode is untouched), and `docs/specs/skills.md` § fab-setup (the mirror-tree successor obligation). Any repo prose claiming bare `fab setup` prints a placeholder MUST be swept in the same pass (kit skills + specs; `docs/memory/` claims are hydrate's job).

- **GIVEN** the change is complete
- **WHEN** grepping the repo for the placeholder claim ("Yet to be implemented" / "placeholder" near `fab setup`)
- **THEN** no kit-skill or spec file still asserts the placeholder behavior

#### R10: Tests ship with the wizard
The wizard MUST land with table/flow tests in `src/go/fab/cmd/fab/setup_test.go` driven through injected stdin (`cmd.SetIn`) and a fixture `Report`/config where needed: all-Enter zero-write, changed-answer diff + surgical write, `--defaults` non-interactive, non-TTY-without-`--defaults` error, provider-option filtering, `dispatch.mode` filtering without tmux, advanced skip rule (all-skipped note + overridden-key surfaced). Constitution: tests conform to these requirements, never the reverse.

- **GIVEN** the affected packages
- **WHEN** `go test ./src/go/fab/cmd/fab/...` runs
- **THEN** the new wizard tests pass alongside the existing setup-check tests

### Non-Goals

- Pane warm-up / trust seeding — deferred to C3 per the backlog entry
- Per-provider fill editing (`providers.<name>.profiles.*`) — C3
- Section entry points (`fab setup agents|dispatch`) and per-key answer flags — C3
- Any change to `fab setup check`, the `fab config` verbs, the registry, or migrations

### Design Decisions

#### Wizard consumes the C1 Report — capability detected, never asked
**Decision**: All detection comes from one `setupcheck.Run` call; the interview only asks preference questions over pre-filtered options.
**Why**: The probe package was built for exactly this consumer ("the future wizard consumes the same `Report` to filter its interview options without shelling out"); the wizard cannot configure what the machine can't run.
**Rejected**: Wizard-owned probing or shelling out to `fab setup check` — duplicated detection logic, parse coupling to human-readable output.
*Introduced by*: 260811-stpw-setup-interactive-wizard

#### Wizard lives in package main beside the set/origin seams
**Decision**: The interview loop lands in a new `src/go/fab/cmd/fab/setup_wizard.go` in package main (sibling of `setup.go`), not a new internal package.
**Why**: The seams it needs — `configMutationPath`, `effectiveTierFor`, `warnIfShadowed`, `stdinIsTTY`-style helper, `version` — are unexported members of package main; an internal package would force exporting write-path plumbing for one consumer. C1's split (cmd owns wiring/rendering, internal owns probing) is preserved: the wizard IS wiring/rendering.
**Rejected**: A new `internal/setupwizard` package — export churn for no reuse; nothing else consumes an interview loop.
*Introduced by*: 260811-stpw-setup-interactive-wizard

#### Advanced section reviews existing customizations (literal skip rule)
**Decision**: Skip advanced questions whose winning tier is the built-in defaults (at-default and never overridden); the all-skipped case prints a note naming the keys with the `fab config explain` pointer.
**Why**: The backlog entry's wording, read literally; keeps the opt-in section short on typical machines while never yielding silence. First-time advanced setup remains a `fab config set` away.
**Rejected**: Asking every advanced key each time (noise, contradicts the entry); the inverted reading (ask only at-default keys) — contradicts the entry's wording.
*Introduced by*: 260811-stpw-setup-interactive-wizard

#### Non-TTY without --defaults errors
**Decision**: When stdin is not a TTY and `--defaults` was not passed, fail with a usage hint.
**Why**: Predictable failure beats hanging on a read or silently pretending to be interactive; `--defaults` is the sanctioned non-interactive path.
**Rejected**: Auto-degrading to `--defaults` — makes CI invocations silently succeed in a mode the caller didn't choose.
*Introduced by*: 260811-stpw-setup-interactive-wizard

## Tasks

### Phase 1: Setup

- [x] T001 Add `--defaults` and `--project` flags to `setupCmd()` in `src/go/fab/cmd/fab/setup.go`; replace the placeholder `RunE` with a call into the wizard entry point (new file `src/go/fab/cmd/fab/setup_wizard.go`); wire the non-TTY guard using the existing `os.ModeCharDevice` pattern from `batch_archive.go` <!-- R1, R2, R8 -->

### Phase 2: Core Implementation

- [x] T002 Implement the wizard skeleton in `src/go/fab/cmd/fab/setup_wizard.go`: run `setupcheck.Run(setupCheckInput())` once, print the scope banner (system default, `--project` retarget with the outside-a-repo error), and the shared question primitive — bare stdin line read from `cmd.InOrStdin()`, rendering current effective value + origin as the default (via `effectiveTierFor`/`cascadeTier` origin rendering) and the `fab config explain <key>` footer <!-- R1, R2, R3 -->
- [x] T003 Implement Q1/Q2 (`agent.session`, `agent.workers`): option list from `Report.Providers` filtered to `Found()`, capability annotations from the `Interactive`/`Headless`/`Native` flags; validate the typed answer against the option list <!-- R3 -->
- [x] T004 Implement Q3 (`dispatch.mode`): filter `pane` on the probe's tmux signal, filter rungs by roster capability presence, state the ladder-ceiling semantics in the question text <!-- R4 -->
- [x] T005 Implement Q4 + the advanced section (`agent.profiles.operator`, `agent.profiles.review`, `dispatch.column_width`, `dispatch.reap_done`): skip keys whose winning tier is the built-in defaults; print the all-skipped note naming the keys with the explain pointer <!-- R5 -->
- [x] T006 Implement the write step: diff summary (`<key>: <old> → <new>` + target tier), confirmation prompt, zero-write short-circuit ("nothing to change", exit 0, no file touched), then per-key writes via `configupgrade.SetSystem`/`configupgrade.Set` + `configMutationPath` + `warnIfShadowed` <!-- R6, R7 -->
- [x] T007 Implement `--defaults`: non-interactive full flow accepting every default (banner + resolved answers + "nothing to change"), composable with `--project`; the non-TTY-without-`--defaults` error message names `--defaults` <!-- R8 -->

### Phase 3: Integration & Edge Cases

- [x] T008 Tests in `src/go/fab/cmd/fab/setup_test.go` with injected stdin and fixture config/Report: all-Enter zero-write idempotence, changed answer → diff + surgical write (temp HOME/system path), `--defaults` run, non-TTY error, provider filtering (undetected dropped, annotation present), `dispatch.mode` without tmux, advanced skip rule both ways (fresh machine note; overridden key asked); run `go test ./src/go/fab/cmd/fab/...` <!-- R10 -->

### Phase 4: Polish

- [x] T009 [P] Update `src/kit/skills/_cli-fab.md` (bare `fab setup` wizard signature, `--defaults`/`--project`, unchanged `check`), `src/kit/skills/fab-setup.md` (delegate the config-interview portion for wizard-covered keys to `fab setup`), and `docs/specs/skills.md` § fab-setup <!-- R9 -->
- [x] T010 [P] Sibling sweep: grep kit skills + specs for placeholder claims about bare `fab setup` ("Yet to be implemented", wizard-is-planned phrasing) and update every non-memory occurrence; leave `docs/memory/` for hydrate <!-- R9 -->

## Execution Order

- T001 → T002 → (T003, T004, T005) → T006 → T007 → T008
- T009/T010 are independent of the Go tasks and each other ([P])

## Acceptance

### Functional Completeness

- [x] A-001 R1: Bare `fab setup` runs the wizard (probe → banner → questions → advanced opt-in → diff → write); `fab setup check` output and exit contract are byte-identical to before
- [x] A-002 R2: The scope banner names the system tier by default; `--project` retargets to the project config and errors outside a fab repo
- [x] A-003 R3: Q1/Q2 option lists come from the probe roster filtered to detected providers with capability annotations; defaults show current effective value + origin; footers point at `fab config explain <key>`
- [x] A-004 R4: Q3 omits `pane` when tmux is absent and states the ladder-ceiling semantics
- [x] A-005 R5: Advanced section skips at-default never-overridden keys and prints the all-skipped note naming the keys
- [x] A-006 R6: Diff-before-write summary renders old → new per changed key; all-Enter run prints "nothing to change" and writes no file
- [x] A-007 R7: Writes go through `configupgrade.SetSystem`/`Set` in-process with `warnIfShadowed` per key; no whole-file rewrite
- [x] A-008 R8: `--defaults` completes non-interactively (zero-write) and composes with `--project`; non-TTY without `--defaults` exits non-zero naming the flag

### Behavioral Correctness

- [x] A-009 R1: The "Yet to be implemented" placeholder string is gone from the binary's bare-setup path (and from swept prose per R9)

### Scenario Coverage

- [x] A-010 R10: Every GIVEN/WHEN/THEN scenario above is exercised by a test in `setup_test.go` (or explicitly justified as manual-only)

### Edge Cases & Error Handling

- [x] A-011 R5: Fresh-machine advanced opt-in yields the note, never silence or a crash
- [x] A-012 R8: Non-TTY stdin without `--defaults` fails fast with the hint (no hang)
- [x] A-013 R2: `--project` outside a fab repo errors clearly instead of writing nothing silently

### Code Quality

- [x] A-014 Pattern consistency: New code follows naming and structural patterns of surrounding code (C1's cmd-owns-wiring split, stdlib-only TTY check)
- [x] A-015 No unnecessary duplication: Existing utilities reused (`setupcheck.Run`, `effectiveTierFor`, `configupgrade.Set*`, `configMutationPath`, `warnIfShadowed`, TTY helper) — no parallel probing or write path
- [x] A-016 CLI ⇒ docs + tests: `_cli-fab.md` updated and tests ship in the same change (project review rule)
- [x] A-017 Canonical source only: no edits under `.claude/skills/`; kit changes in `src/kit/`
- [x] A-018 Owner-or-pointer: skill-prose edits state a rule they own or point at its owner, never both
- [x] A-019 No god functions: the interview loop is decomposed (question primitive, section runners, write step), no >50-line monolith without reason

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (the placeholder `RunE` body and `TestSetupCmd_BarePrintsPlaceholder` were planned removals already executed in-diff; `effectiveTierFor` retains live callers via `warnIfShadowed`/`liveTierNotice`).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Wizard code lands in package main as `setup_wizard.go`, not an internal package | The set/origin seams it reuses (`configMutationPath`, `effectiveTierFor`, `warnIfShadowed`) are unexported in package main; exporting them for one consumer is churn | S:60 R:85 A:80 D:70 |
| 2 | Confident | TTY detection via the existing stdlib-only `os.ModeCharDevice` pattern (`batch_archive.go`) | Two in-repo precedents explicitly reject `x/term`/`go-isatty`; consistency wins | S:70 R:90 A:90 D:85 |
| 3 | Confident | "Never overridden" for the advanced skip rule = the key's winning tier is the built-in defaults tier (per `effectiveTierFor`) | The cascade has no history; the only observable meaning of "never overridden" is "no tier above defaults defines it" | S:60 R:80 A:75 D:65 |
| 4 | Confident | `native`/`headless` rungs in Q3 are offered when any *detected* provider declares the capability; `pane` additionally requires the tmux signal | Mirrors the descent ladder's own viability inputs; offering an unrunnable rung would contradict "capability is detected never asked" | S:55 R:80 A:70 D:60 |
| 5 | Confident | The Q3/Q4 confirmation prompts and `--defaults` answers render to stdout in both modes (auditable non-interactive runs) | Matches `setup check`'s stdout-is-the-data principle (toolkit №2) | S:50 R:90 A:75 D:70 |
| 6 | Confident | The advanced profile questions target the scalar leaf keys `agent.profiles.operator.provider` / `agent.profiles.review.provider`, not the map keys the intake names | The intake's "agent.profiles.operator (provider)" reads as the provider field of that role profile; the surgical set path only accepts scalar leaf keys, and `<role>.provider` is the registry's settable unit | S:65 R:80 A:75 D:60 |
| 7 | Confident | Q3's tmux gate reads `setupCheckInput().TmuxEnv` (the same Input handed to `setupcheck.Run`) rather than re-deriving it from the Report's finding strings | The Report carries tmux only as a rendered finding; parsing it back would couple to human-readable prose, and the Input field IS the probe's tmux signal | S:70 R:85 A:80 D:70 |
| 8 | Confident | Outside a fab repo the wizard loads its read model with an empty project path (system+env+defaults), mirroring `setupcheck.Run`'s degradation | R1's GIVEN covers "any directory (fab repo or not)"; `config.LoadLayers("")` is the established no-repo pattern | S:60 R:85 A:75 D:65 |
| 9 | Confident | Invalid typed answers re-ask (with the option list named); EOF on stdin falls back to the question's default | Re-asking matches interview expectations; EOF-as-default guarantees the loop can never hang on an exhausted reader | S:50 R:85 A:65 D:55 |
| 10 | Confident | The write confirmation defaults to yes (`[Y/n]`) — the diff it follows is the real review step | The change summary already shows every pending write; a default-yes confirm matches `config set`'s zero-confirmation posture while still gating the mutation | S:40 R:85 A:55 D:45 |

10 assumptions (0 certain, 10 confident, 0 tentative).
