# Plan: Operator Question-Sweep Command (`fab pane questions`)

**Change**: 260823-dckc-pane-questions-sweep
**Intake**: `intake.md`

## Requirements

### CLI: `fab pane questions` command

#### R1: Command surface and flags
A new `fab pane questions` subcommand SHALL exist under the `fab pane` group (new file `src/go/fab/cmd/fab/pane_questions.go`, registered in `pane.go`'s `AddCommand` list), with flags `--all-sessions` (bool), `--panes <id>...` (StringSlice, repeatable/comma-separated), and `--json` (bool), consuming the pane family's existing persistent `--server`/`-L` flag. `--all-sessions` and `--panes` MUST be mutually exclusive (`cmd.MarkFlagsMutuallyExclusive`). With neither flag, the command MUST require `$TMUX` (current-session default) and error when unset; `--panes` and `--all-sessions` MUST work without `$TMUX`.

- **GIVEN** a shell outside tmux
- **WHEN** `fab pane questions` runs with no flags
- **THEN** it exits non-zero with a "not inside a tmux session" error
- **AND** `fab pane questions --panes %3` in the same shell proceeds (no `$TMUX` guard)

#### R2: Candidate population
When discovering its own candidates (`--all-sessions` or the no-flag current-session default), the command SHALL enumerate panes via the existing `collectPaneRows` helper (`panemap.go` — no changes to that file) and filter the candidate set to panes whose resolved `agent_state` is `waiting` or `idle`; unknown-state (`""`/`—`) and `active` panes are excluded by construction. With `--panes`, the given IDs are the candidate set verbatim (no filter at selection time; per-pane state is checked during the sweep, R3).

- **GIVEN** a server with panes in `waiting`, `idle`, `active`, and unknown states
- **WHEN** `fab pane questions --all-sessions` runs
- **THEN** only the `waiting` and `idle` panes are swept

#### R3: Per-pane sweep procedure and skip reasons
For each candidate pane, the sweep SHALL apply, in this exact order, short-circuiting at the first hit: (1) pane absent from the up-front `collectPaneRows` snapshot → skip `capture_failed`; (2) pane present but `agent_state` not `waiting`/`idle` → skip `state_changed`; (3) capture the last **20** lines (fixed, not a flag) via a package-level injectable seam `paneCaptureFn` defaulting to `pane.Capture` — capture error → skip `capture_failed`; (4) empty/all-whitespace capture → skip `blank_capture`; (5) either of the last 2 lines matching `^\s*>\s*$` → skip `turn_boundary`; (6) no indicator match (R4) → skip `no_indicator`; otherwise emit a match. The five skip-reason strings are exactly `turn_boundary`, `blank_capture`, `no_indicator`, `capture_failed`, `state_changed`.

- **GIVEN** a `--panes` list naming a dead pane, an `active` pane, and a `waiting` pane showing `Do you want to proceed?`
- **WHEN** the sweep runs
- **THEN** the dead pane skips `capture_failed`, the active pane skips `state_changed`, and the waiting pane matches

#### R4: Mechanical indicator scanner
A pure, I/O-free scanner function SHALL implement the skill's §5 indicator classes minus class 4 (Claude Code permission/tool-approval prompts — non-mechanical by design): (1) `?`-ending last non-empty line only, <120 chars, excluding `#`/`//`/`*`/`>`/timestamp-prefixed lines; (2) `[Y/n]`/`[y/N]`/`(y/n)`/`(yes/no)` case-insensitive; (3) `Allow?`/`Approve?`/`Confirm?`/`Proceed?`; (4) `Do you want to`/`Should I`/`Would you like` case-insensitive; (5) `:`-ending lines; (6) enumerated options `[1-9])`; (7) `Press.*key`/`press.*enter`/`hit.*enter` case-insensitive. Scanning walks non-blank lines bottom-most upward; class 1 is tested only against the actual last line; the first (bottom-most) matching line wins, reporting indicator names `question_mark`, `yes_no`, `action_word`, `imperative_question`, `colon_prompt`, `enumerated_options`, `press_key`.

- **GIVEN** a capture whose last line is `1) yes  2) no` and an earlier line ends with `?`
- **WHEN** the scanner runs
- **THEN** the match is the bottom-most line with indicator `enumerated_options`

#### R5: Output schema and exit codes
`--json` SHALL emit `{"matches": [{"pane","agent_state","indicator","snippet"}...], "skipped": [{"pane","reason"}...]}` (empty arrays, never null). Human output SHALL print one line per match (`pane [agent_state] indicator: snippet`), one per skip, then `N matched, M skipped`; an empty candidate set prints `No candidate panes.` The command SHALL exit 0 on any clean sweep regardless of match/skip counts (the pane-family per-pane 2/3 exit scheme does NOT apply); non-zero only on usage errors or a hard discovery failure, mirroring `fab pane map`.

- **GIVEN** a sweep where every candidate is skipped
- **WHEN** the command completes
- **THEN** it exits 0 with the skips listed as data

#### R6: Test coverage
`src/go/fab/cmd/fab/pane_questions_test.go` (new) SHALL cover: both guards (turn-boundary in last-2-lines, blank capture); table-driven indicator-class cases over the pure scanner including bottom-most-wins and class-1 last-line-only scoping; dead pane ⇒ `capture_failed`; state-changed ⇒ `state_changed`; `--json` schema stability (field names, empty-array encoding); and the seeded-row/capture-seam stubbing path (no live tmux).

- **GIVEN** `go test ./src/go/fab/cmd/fab/`
- **WHEN** the suite runs without tmux
- **THEN** all pane-questions tests pass using the stubbed seams

### Skills: operator detection rewire

#### R7: `fab-operator.md` §5 rewrite
`src/kit/skills/fab-operator.md` § Question Detection SHALL replace its manual steps 1–4 (per-pane capture + the two guards + the indicator list applied by the LLM) with a single `fab pane questions --panes <ids>` sweep over the tick's `candidates:` block (population policy unchanged), keeping steps 5–6 (no match → stuck detection; match → Answer Model) and the whole Answer Model LLM-side. The section MUST state the class-4 split (Claude Code permission/tool prompts are not mechanized — covered in practice by the yes/no, action-word, imperative, and enumerated classes; novel shapes remain operator judgment via on-demand sweep or manual capture). The §3 pre-send gate and § Sending Auto-Answers re-capture-before-send guard are retained unchanged, and § Sending Auto-Answers gains one sentence stating `questions` output is detection input only, never a license to send blind. §4's version-skew fallback line MUST name `fab pane questions` alongside `tick-start --diff` (extend the existing line only if its wording doesn't already cover both).

- **GIVEN** the rewritten §5
- **WHEN** an operator follows it on a tick with candidates
- **THEN** detection is one `fab pane questions` call and the delivery guards still run before any send

#### R8: `_cli-fab.md` command reference entry
`src/kit/skills/_cli-fab.md` § fab pane SHALL gain a `questions` entry in the family's existing format (signature, flags, JSON fields, skip-reason enum, exit-code note), documented as a first-class skill-facing verb — NOT dispatch-internal — per the constitution's CLI ⇒ docs rule.

- **GIVEN** the new entry
- **WHEN** a skill needs the command contract
- **THEN** signature and schema are documented without reading Go source

#### R9: `_cli-agents.md` layering carve-out
`src/kit/skills/_cli-agents.md` § Peek SHALL gain a short paragraph naming `fab pane questions` as a policy-bearing sweep (fab-operator's own indicator patterns, not raw peek) and therefore the one exception to "skill-facing peek rides run-kit's substrate twins".

- **GIVEN** the updated § Peek
- **WHEN** a reader checks the fab-vs-rk layering rule
- **THEN** the `questions` exception is named with its rationale

### Non-Goals

- Indicator class 4 (Claude Code permission/tool-approval prompts) as a regex — non-mechanical per the plan; stays operator judgment.
- Any change to `panemap.go`, `operator_tick_start.go`, `operator_note.go`, or `internal/pane` — the capture seam lives in the new file.
- Configurable capture depth or indicator patterns — fixed at 20 lines / hardcoded patterns by design.
- Memory rewrites (`runtime/pane-commands.md` fence, `runtime/operator.md` B2 half) — hydrate-stage work, not apply tasks.

### Design Decisions

#### Seed the implementation from the stashed prior attempt
**Decision**: Apply starts by extracting `src/go/fab/cmd/fab/pane_questions.go` and the `pane.go` registration from stash commit `1749af376ee48709fcc0678e32f56ec93ad5796e` (`git show 1749af37:<path>`), then verifies it against R1–R5 rather than rewriting from scratch.
**Why**: The stashed file is complete, well-commented, and verified compatible with current main (uses B1's `collectPaneRows`; `pane.Capture`/`AgentState*` unchanged); rewriting invites drift from an already-reviewed shape.
**Rejected**: Fresh rewrite — pure waste; `git stash apply` of the whole entry — it drags in the dead change's `fab/changes/260823-ekh9-*` artifacts, which must not resurrect.
*Introduced by*: 260823-dckc-pane-questions-sweep

#### Exit-code contract follows `pane map`, not the per-pane 2/3 scheme
**Decision**: Clean sweeps always exit 0; matches and skips are data.
**Why**: A multi-pane sweep has no single target to be "missing"; a dead candidate is an expected `capture_failed` skip.
**Rejected**: Pane-family exit 2/3 — would make routine sweep outcomes look like failures to the operator loop.
*Introduced by*: 260823-dckc-pane-questions-sweep

## Tasks

### Phase 1: Setup

- [x] T001 Seed `src/go/fab/cmd/fab/pane_questions.go` from stash commit `1749af37` (`git show 1749af376ee48709fcc0678e32f56ec93ad5796e:src/go/fab/cmd/fab/pane_questions.go`) and register `paneQuestionsCmd()` in `src/go/fab/cmd/fab/pane.go`'s `AddCommand` list (update the pane group's verb-list prose if it enumerates subcommands); `go build ./src/go/...` compiles <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 Verify the seeded implementation against R1–R5 line-by-line (flag wiring + mutual exclusivity + `$TMUX` guard scope; discovery filter; sweep order and the five skip reasons; scanner classes/ordering/last-line scoping; JSON schema incl. empty-array encoding; human output; exit codes) and fix any divergence in `src/go/fab/cmd/fab/pane_questions.go` <!-- R3 -->
- [x] T003 Write `src/go/fab/cmd/fab/pane_questions_test.go`: table-driven pure-scanner cases (all 7 classes, bottom-most wins, class-1 last-line-only + <120-char + prefix/timestamp exclusions), guard tests, command-level tests over stubbed `paneCaptureFn` + seeded pane rows (dead pane → `capture_failed`, active → `state_changed`, blank → `blank_capture`, turn boundary → `turn_boundary`, no match → `no_indicator`, match rows), `--json` schema, `--panes`/`--all-sessions` mutual exclusivity, `$TMUX` guard <!-- R6 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Rewrite `src/kit/skills/fab-operator.md` § Question Detection onto `fab pane questions --panes <ids>` (steps 5–6 and Answer Model unchanged), add the class-4 split line, add the § Sending Auto-Answers cross-reference sentence, and extend §4's version-skew fallback line to name `fab pane questions` if not already covered <!-- R7 -->
- [x] T005 [P] Add the `questions` entry to `src/kit/skills/_cli-fab.md` § fab pane (signature, flags, JSON fields, skip-reason enum, exit codes; first-class skill-facing verb) <!-- R8 -->
- [x] T006 [P] Add the policy-bearing-sweep carve-out paragraph to `src/kit/skills/_cli-agents.md` § Peek <!-- R9 -->
- [x] T007 Sibling sweep: grep repo-wide (`src/kit/skills/`, `docs/specs/`) for restatements of §5's manual capture-and-scan detection (e.g. `skills.md`/`glossary.md` operator entries, `_cli-agents.md` peek phrasing) and update every stale occurrence in the class <!-- R7 -->

### Phase 4: Polish

- [x] T008 `gofmt -l src/go/` clean; `go vet ./src/go/...`; run `go test ./src/go/fab/cmd/fab/ ./src/go/fab/internal/pane/` (scoped first, widen if cross-cutting signals appear) <!-- R6 -->

## Execution Order

- T001 blocks T002 and T003; T004–T006 are independent of Go work after T002 fixes the contract; T007 follows T004–T006; T008 last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab pane questions` exists with `--all-sessions`/`--panes`/`--json`, consumes `--server`, enforces mutual exclusivity, and applies the `$TMUX` guard only to the no-flag default
- [x] A-002 R2: Discovery modes sweep only `waiting`/`idle` panes via `collectPaneRows`; `--panes` takes IDs verbatim; `panemap.go` untouched
- [x] A-003 R3: The sweep applies the six checks in the specified order with exactly the five skip-reason strings; capture fixed at 20 lines through the `paneCaptureFn` seam
- [x] A-004 R4: The pure scanner implements classes 1–7 (minus prose class 4) with bottom-most-wins and class-1 last-line-only scoping, emitting the seven named indicators
- [x] A-005 R5: JSON and human output match the specified shapes (empty arrays, `No candidate panes.`, count line); exit 0 on clean sweeps
- [x] A-006 R7: `fab-operator.md` §5 detection is a single `questions` sweep; Answer Model, §3 gate, and re-capture guard retained; class-4 split and skew-fallback naming present
- [x] A-007 R8: `_cli-fab.md` documents `questions` in the family format as a skill-facing verb
- [x] A-008 R9: `_cli-agents.md` § Peek names the `questions` carve-out with rationale

### Scenario Coverage

- [x] A-009 R6: Table-driven tests cover all indicator classes, both guards, all five skip reasons, and `--json` schema stability, passing without live tmux
- [x] A-010 R3: Race-closing behavior verified — a candidate that went `active` or died between listing and sweep is skipped (`state_changed`/`capture_failed`), never matched

### Edge Cases & Error Handling

- [x] A-011 R1: Outside tmux, the no-flag form errors; `--panes`/`--all-sessions` forms proceed
- [x] A-012 R5: All-skip and empty-candidate sweeps exit 0 with correct output

### Code Quality

- [x] A-013 Pattern consistency: new command follows the pane-family file/registration/flag patterns; seam follows the `rkPanesRunner` precedent
- [x] A-014 No unnecessary duplication: discovery reuses `collectPaneRows`; no new tmux/rk enumeration code
- [x] A-015 CLI ⇒ docs + tests: `_cli-fab.md` updated and tests ship in the same change (constitution)
- [x] A-016 Canonical source only: all skill edits under `src/kit/skills/`, none under `.claude/skills/`
- [x] A-017 Owner-or-pointer: the §5 rewrite and the `_cli-agents.md` carve-out don't restate owned rules alongside pointers; sibling sweep (T007) covers the twin/aggregate class

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality (new subcommand + tests + skill rewires) without making existing code redundant. The only deletions are the §5 manual capture-and-scan steps in `fab-operator.md`, removed by the change itself.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Memory edits (pane-commands fence, operator.md B2 half) are hydrate-stage work, not apply tasks | Pipeline contract — hydrate owns docs/memory writes; intake's Affected Memory carries the list | S:85 R:90 A:95 D:90 |
| 2 | Confident | T007 sibling sweep is a distinct task rather than folded into T004–T006 | code-quality.md names sibling sweeps the top rework cause and says sweep the whole class up front; a named task makes review verifiable | S:70 R:85 A:85 D:80 |
| 3 | Confident | Seeding via `git show` of the single file, never `git stash apply` | Stash entry also carries the dead ekh9 change folder which must not resurrect; single-file extraction is surgical | S:75 R:80 A:90 D:85 |

3 assumptions (1 certain, 2 confident).
