# Plan: Operator Hand-over to HexoKit (Plan Step F1)

**Change**: 261113-915b-operator-hexokit-handover
**Intake**: `intake.md`

## Requirements

> Ground rule for every requirement (plan Rule 1): apply MUST NOT run `fab sync`, `brew upgrade`,
> restart the rk daemon, or invoke a real `rk cron`/`rk operator` write against the live tmux
> server. All rk behaviour is exercised through the existing injectable seams (`rkCronRunner`,
> `rkOperatorPath`) and new seams stubbed in tests.

### Operator CLI: Capability Probe

#### R1: Hand-over capability probe
The `fab` binary SHALL decide "rk ≥ R5's release is present" with one injectable, side-effect-free
probe seam (e.g. `rkOperatorHandover func() bool` in `src/go/fab/cmd/fab/operator.go` or a new
`operator_handover.go`).
- The probe MUST `exec.LookPath("rk")` first.
- It MUST then run `rk operator --help`, bounded by a timeout. Reuse `rkCronTimeout`.
- It MUST answer true only when the help output's command listing carries **both** a `check` row
  and a `brief` row. A row is a line whose first whitespace-delimited token is the subcommand name;
  a bare substring is not enough.
- It MUST answer false, silently, on any error or non-match.
- It MUST be memoized per process.
- The pure matcher (help text → bool) MUST be a separate function so tests can drive it with
  captured help text.

- **GIVEN** rk v3.22.12 help (lists `adopt`, `request`, `track`, but no `check`/`brief`)
- **WHEN** the matcher runs
- **THEN** it returns false
- **AND** for rk v3.22.13 help (lists `adopt brief check request track`) it returns true
- **AND** with rk absent from PATH the probe returns false and prints nothing

### Operator CLI: Deprecate and Delegate

#### R2: Deprecation line
When the R1 probe is true, every `fab operator` **subcommand** (`track …`, `tick-start`,
`clock …`, `state`, `branch-map …`, `time`) SHALL print exactly one deprecation line to
**stderr** before doing anything else, naming the rk replacement where one exists. Example:
`fab operator track is deprecated — delegating to 'rk operator track' (HexoKit now owns the operator)`.

The deprecation line MUST NOT appear in any of these cases:
- the R1 probe is false;
- on stdout;
- on the bare `fab operator` launcher (it already delegates to `rk operator` via
  `rkOperatorPath`);
- on any `fab agent` command.

- **GIVEN** a capable rk
- **WHEN** `fab operator time` runs
- **THEN** stderr carries one deprecation line, and stdout and the exit code are unchanged from
  today

#### R3: `track` verb delegation
When the R1 probe is true, `fab operator track <verb>` SHALL delegate to rk and SHALL NOT read or
write fab's operator state file:
- rk is executed argv-only (no shell string);
- stdin/stdout/stderr are inherited;
- fab exits with rk's exit code (non-zero rk exit → the same code; failure to start rk → exit 1).

The rk argv is rebuilt from the cobra-parsed **changed** flags plus positional args:
- `stringArray` flags repeat (`--argv a --argv b`);
- `strings` flags join with commas.

Mapping:

| fab | rk |
|---|---|
| `track add <id> [flags]` | `operator track add <id> [same flags]` |
| `track list [--json] [--kind k]` | `operator track list [--json] [--kind k]` |
| `track rm <id>` | `operator track rm <id>` |
| `track observe <id> [--json j] [--error e] [--seen s]…` | `operator track observe <id> [same flags]` |
| `track update <id> [other flags]` | `operator track update <id> [other flags]` |
| `track update <id> --pause` | `operator track pause <id>` |
| `track update <id> --resume` | `operator track resume <id>` |
| `track update <id> --text t` | `operator track note <id> t` |

A mixed `update` runs `rk operator track update` first, when any other flag changed. Then, in
order, it runs `pause`, `resume`, then `note`. The sequence stops at the first non-zero exit and
returns that code. `--pause` together with `--resume` remains a usage error (exit 2) before any rk
call, matching fab's existing validation.

- **GIVEN** a capable rk
- **WHEN** `fab operator track add pr-1 --kind github-pr --scope '{"pr":1}' --argv gh --argv pr --fields state,mergedAt`
  runs
- **THEN** rk is invoked as
  `rk operator track add pr-1 --kind github-pr --scope {"pr":1} --argv gh --argv pr --fields state,mergedAt`
  (argv elements, flag order may differ)
- **AND** fab's state file is neither created nor modified
- **GIVEN** a capable rk
- **WHEN** `fab operator track update w1 --then x --pause` runs
- **THEN** rk receives `operator track update w1 --then x`, then `operator track pause w1`

#### R4: `tick-start` delegation
When the R1 probe is true, `fab operator tick-start [--diff] [--quiet]` SHALL delegate to
`rk operator check`, with inherited stdio and rk's exit code:
- fab's `--diff` and `--quiet` are accepted and dropped;
- `tick_count`/`last_tick_at` are not written;
- no fab clock reconcile runs.

- **GIVEN** a capable rk
- **WHEN** `fab operator tick-start --diff --quiet` runs
- **THEN** rk receives exactly `operator check`, and fab's state file is untouched

#### R5: `track clock` refusal
When the R1 probe is true, `fab operator track clock …` SHALL write nothing and call no rk verb.
Besides the deprecation line, it prints one stderr line pointing at the replacement: tune the
`operator loop` entry with `rk cron edit <id>` or `rk cron mute <id> --for <dur>` (find the id with
`rk cron list`). It exits 1.

- **GIVEN** a capable rk
- **WHEN** `fab operator track clock --every 5m --for 1h` runs
- **THEN** exit 1, the pointer is on stderr, and no `rkCronRunner` call is recorded

#### R6: `clock sync` under a capable rk
When the R1 probe is true, `fab operator clock sync`:
- SHALL NOT seed, mute, unmute, or edit any cron entry;
- SHALL NOT require fab's state file;
- SHALL run `rk cron list` with inherited stdio and exit with rk's code.

- **GIVEN** a capable rk and a fab state file with live tracked items
- **WHEN** `fab operator clock sync` runs
- **THEN** the only rk invocation is `cron list`, and no `cron add`/`cron mute`/`cron edit` is
  issued

### Operator CLI: Clock Side-Effect Gate

#### R7: No fab writes to rk's cron file under a capable rk
When the R1 probe is true, fab SHALL issue no `rk cron add`, `rk cron mute` (either direction) or
`rk cron edit` through any path. The gate MUST sit at the shared entry points so every caller is
covered: `ensureOperatorCronRow` / `seedOperatorCronRow`, `syncOperatorClock`,
`muteOperatorClockIfUntracked`, `reconcileOperatorMute`, `reconcileOperatorSchedule`, and any
clock-override apply/expire path that calls `rk cron edit`. Each answers its no-op/not-ok result
silently.

- **GIVEN** a capable rk (probe stubbed true) and a fab state whose tracked set flips
  untracked→tracked
- **WHEN** a non-delegated code path calls `syncOperatorClock(false, true)` or
  `reconcileOperatorMute(tracked, row)`
- **THEN** no `cron mute … --off` argv is recorded (rk's muted legacy entry stays muted)

#### R8: Fallback arm unchanged
When the R1 probe is false, every `fab operator` verb, the clock reconcile, and the launcher SHALL
behave exactly as before this change: same stdout, stderr, exit codes and rk calls. Every existing
operator test MUST pass with the probe stubbed false. The default test stub MUST be false, so
existing tests need no edits beyond the stub.

- **GIVEN** the probe is false
- **WHEN** the existing operator test suite runs
- **THEN** it passes unchanged

### Skills: `/fab-operator` Pointer

#### R9: Project skill becomes a short pointer
`src/kit/skills/fab-operator.md` SHALL be rewritten as a short pointer (≤ ~80 lines):
- `name: fab-operator` (rk's launcher types `/fab-operator` as the kickoff) and an updated,
  trigger-worthy `description` saying it opens the operator chat via HexoKit's operator skill,
  with fab's legacy operator as fallback;
- no `helpers:`, and no always-load layer.

Body:
1. Gate on `command -v rk`, then run `rk skill operator`. Success = exit 0 and the output's first
   heading names the operator topic (`# HexoKit skill: operator`).
2. On success, follow that output as the operator chat skill. One hand-over hint: run
   `rk operator adopt --dry-run`. If it reports items to import, tell the user that
   `rk operator adopt` imports fab's tracked items (the pointer never runs adopt itself).
3. On failure (rk absent, older rk, any other output), read the legacy operator skill and follow
   it: `.agents/skills/_operator-legacy/SKILL.md`, else
   `$(fab kit-path)/skills/_operator-legacy.md`.

- **GIVEN** rk v3.22.13 on PATH
- **WHEN** an agent runs `/fab-operator`
- **THEN** it follows `rk skill operator` and never loads the 108KB legacy body
- **GIVEN** rk absent
- **WHEN** an agent runs `/fab-operator`
- **THEN** it follows the legacy partial, i.e. today's behaviour

#### R10: Legacy operator partial
Today's `fab-operator.md` body SHALL move verbatim to `src/kit/skills/_operator-legacy.md`.
- Frontmatter: `name: _operator-legacy`, `description`, `user-invocable: false`,
  `disable-model-invocation: true`, `metadata: {internal: true}`, plus the original
  `helpers: [_cli-fab-operator, _cli-fab-pane, _cli-agents, _cli-external]`.
- Kit cross-references of the form ``fab-operator.md` §N`` (in `_cli-fab-operator.md`,
  `_cli-fab-pane.md`, `_cli-agents.md`, `_cli-external.md`, `_cli-fab.md`, `_preamble.md`,
  `fab-setup.md`) SHALL be re-pointed to `_operator-legacy.md` wherever they cite legacy operator
  sections.
- References to the `/fab-operator` command name stay.

- **GIVEN** the sweep is done
- **WHEN** `grep -rn 'fab-operator.md' src/kit` runs
- **THEN** every remaining hit refers to the pointer file itself, not to a legacy section

#### R11: User-level pointer probes rk first
`src/kit/templates/user-skill-fab-operator.md` SHALL gain a first step: when `rk skill operator`
succeeds (same success test as R9), follow it. Otherwise it continues with today's two steps
unchanged (project deployed copy, else `fab kit-path` flat copy with the flat-copy rule).
- The "Generated by `fab sync`" header stays.
- `{DESCRIPTION}` substitution stays.

- **GIVEN** `fab sync` renders the template (test with a temp `$HOME`)
- **WHEN** the rendered file is read
- **THEN** it contains the `rk skill operator` step before the project/kit-path steps

### Distribution: Migration

#### R12: Hand-over migration file
A new `src/kit/migrations/2.28.16-to-2.29.0.md` SHALL follow the existing migration shape (Summary
/ Pre-check / Changes / Verification), with the release-owned-slot placeholder note from
`2.27.0-to-2.28.0.md`. It SHALL state that the migration performs **no file edits** and the binary
moves **no data**.

Pre-check: `rk operator --help` lists `check` and `brief`; otherwise the migration is a no-op.

Per-server hand-over steps, done with the user:
1. Stop or idle the old operator chat.
2. Upgrade fab and rk.
3. `rk operator adopt --dry-run`, then `rk operator adopt`. Idempotent; fab's file is only read.
4. The rk daemon seeds the target-less `operator loop` entry and mutes, never deletes, the legacy
   `operator tick` entry once rk's state file exists. Confirm with `rk cron list`.
5. Open the new chat via `rk operator`.
6. Watch one wake → check → run → note cycle.

Rollback: mute the loop entry and unmute the legacy one. fab's state file was never modified.

Verification: exactly one unmuted operator clock, and `fab operator clock sync` prints the
deprecation and unmutes nothing.

The file MUST NOT cite fab-kit-only paths or the hexokit plan path (Constitution V; the
kit-portability Go test guards it).

- **GIVEN** the migration file
- **WHEN** `TestKitContentCitesNoRepoLocalPaths` runs
- **THEN** it passes

### Docs: CLI Reference, Specs, Memory

#### R13: CLI reference and help surfaces
`src/kit/skills/_cli-fab-operator.md` SHALL document:
- the R1 probe;
- the R2 deprecation line;
- the R3 mapping table (including `update` splitting and the note-append semantic shift);
- R4, R5, R6 and the R7 gate;
- that the fallback arm is unchanged.

`fab agent` sections stay untouched. Verb `Long` help text in Go SHALL mention the delegation in
one sentence per delegated verb.

Also update, wherever they describe `/fab-operator`'s content or context loading, to stay
consistent:
- `src/kit/skills/_preamble.md` (the example "`/fab-operator` loads a reduced 3-file set" → now the
  legacy partial's override; the pointer loads nothing);
- `src/kit/skills/fab-help.md` / `src/go/fab/cmd/fab/fab_help.go`;
- `src/go/fab/cmd/fab/skill.md`.

- **GIVEN** the change
- **WHEN** `go test ./...` runs in `src/go/fab` and `src/go/fab-kit`
- **THEN** doc-guard tests (`clifab_doc_test`, `fab_help_test`, `kit_portability_test`) pass

#### R14: Specs updated (human-curated, minimal)
- `docs/specs/operator.md`: one new version-history row ("v14 — hand-over to HexoKit: fab verbs
  deprecate+delegate under rk ≥ v3.22.13, `/fab-operator` → pointer to `rk skill operator` with
  legacy fallback; deletion pending F2") and a one-line note under the command table.
- `docs/specs/skills.md`: its `/fab-operator` section names the pointer behaviour and the legacy
  partial.

No other spec prose is rewritten.

- **GIVEN** the specs
- **WHEN** read
- **THEN** they describe the pointer and the legacy fallback without contradicting the
  implementation

### Non-Goals

- Deleting any operator Go code, the legacy skill, `_cli-fab-operator.md`, or the user-level
  template. That is F2.
- Any data move from fab's state file to rk's (rk adopts).
- Changing `fab agent`, `fab pane`, `fab dispatch`, or `fab status set-base-branch`.
- Running `fab sync`, `brew upgrade`, or restarting rk on the live server. That is C1.
- Editing the constitution. Principle V's pointer sentence stays accurate for the fallback branch.
  Flag it at hydrate if a wording touch is wanted.

### Design Decisions

#### Capability probe over version compare
**Decision**: Detect R5 by `rk operator --help` listing both `check` and `brief`.
**Why**: R1 (`track`, `adopt`) shipped in rk v3.22.12 without the loop. R2/R4/R5 shipped together
in v3.22.13. A help-row probe matches the existing `rkOperatorPath` precedent and works for dev
builds.
**Rejected**: Parsing `rk --version`, which fails for dev builds and duplicates the rk release
calendar into fab. Probing the hidden `run-exec` verb, which is an internal wrapper and not a
contract.
*Introduced by*: 261113-915b-operator-hexokit-handover

#### Gate every cron write, not only the seed
**Decision**: Under a capable rk, fab issues no `rk cron add/mute/edit` from any path.
**Why**: rk mutes, never deletes, fab's legacy `operator tick` entry when it seeds its loop. fab's
mute reconcile would `mute --off` it on a tracked set, so two clocks would fire.
**Rejected**: Stopping only the seed, which leaves the unmute double-fire path.
*Introduced by*: 261113-915b-operator-hexokit-handover

#### Pointer plus legacy partial
**Decision**: `fab-operator.md` becomes a short rk-first pointer. Today's body moves verbatim to
the internal `_operator-legacy` partial, read only when rk can't serve the chat skill.
**Why**: Keeps today's behaviour without a capable rk, while a capable rk never loads the 108KB
legacy body into the operator chat.
**Rejected**: Prepending a pointer to the full body, which still loads 108KB. Deleting the body
now, which breaks rk-less users and is F2's job.
*Introduced by*: 261113-915b-operator-hexokit-handover

## Tasks

### Phase 1: Setup

- [x] T001 Add the hand-over probe in `src/go/fab/cmd/fab/operator_handover.go` (new): a pure help-text matcher `rkHelpHasHandover(help string) bool` (row-anchored `check` + `brief`), the memoized injectable seam `rkOperatorHandover func() bool` (LookPath + `rk operator --help` with `rkCronTimeout`, fail-silent), a `printOperatorDeprecation(cmd, verb, replacement string)` stderr helper, and an argv-only `runRkDelegate(cmd, args ...string) error` that inherits stdio and maps rk's exit status to fab's exit code (start failure → 1). Reuse existing exit-code plumbing in `src/go/fab/cmd/fab` (look for how other commands return a specific exit code) <!-- R1 --> <!-- rework: scope rkHelpHasHandover's row scan to the 'Available Commands:' block (a wrapped Long line starting with check/brief must not count; add a test); map a negative rk ExitCode (signal-killed) to exit 1 -->
- [x] T002 Add `src/go/fab/cmd/fab/operator_handover_test.go`: matcher table test using captured v3.22.12 help (no `check`/`brief` rows; include a decoy line mentioning "check" mid-sentence) → false, and v3.22.13 help → true; probe with rk absent (empty PATH) → false and no output. Set the package-level test default so `rkOperatorHandover` is stubbed false for all existing tests (e.g. in an existing `TestMain`, or per-test helper) <!-- R1, R8 -->

### Phase 2: Core Implementation

- [x] T003 Wire the deprecation line into every `fab operator` subcommand (`track` + all its verbs, `tick-start`, `clock sync`, `state`, `branch-map` verbs, `time`) in `src/go/fab/cmd/fab/operator*.go`, firing only when the probe is true, and never on the bare launcher or `fab agent` <!-- R2 --> <!-- rework: 'track clock' must not print 'delegating to rk cron edit' — it delegates nothing; use no-counterpart/refusal wording and update TestOperatorReplacement -->
- [x] T004 Implement `track` delegation in `src/go/fab/cmd/fab/operator_track.go`: a generic `rkArgvFromChangedFlags(cmd)` builder (stringArray repeats, strings comma-join, skip `help`), per-verb delegation at the top of each RunE for add/list/rm/observe, and the `update` split (update-with-other-flags → pause → resume → note; stop at first non-zero; `--pause`+`--resume` usage error before any rk call). The delegated arm never opens fab's state file <!-- R3 --> <!-- rework: --text → 'rk operator track note <id> -- <text>' (insert '--' so a leading '-' or a bare '-' is not parsed as a flag/stdin); also make extract leaf verbs list/update/pause/resume/note and the deprecation format strings named constants (A-023) -->
- [x] T005 Implement `tick-start` delegation in `src/go/fab/cmd/fab/operator_tick_start.go`: probe true → deprecation + `rk operator check` and nothing else <!-- R4 -->
- [x] T006 Implement the `track clock` refusal (exit 1, pointer line, no rk call, no write) in `src/go/fab/cmd/fab/operator_track.go` (or wherever `track clock` lives) <!-- R5 -->
- [x] T007 Implement the `clock sync` rk arm in `src/go/fab/cmd/fab/operator_clock_sync.go`: probe true → deprecation + `rk cron list` passthrough (inherited stdio, rk's exit code), before the state-file stat <!-- R6 -->
- [x] T008 Add the cron-write gate in `src/go/fab/cmd/fab/operator_clock.go` at `ensureOperatorCronRow`/`seedOperatorCronRow`, `syncOperatorClock`, `muteOperatorClockIfUntracked`, `reconcileOperatorMute`, `reconcileOperatorSchedule`, and any clock-override path that issues `rk cron edit` (grep `rkCronRunner("cron"` to find every write site) <!-- R7 -->
- [x] T009 Tests in `src/go/fab/cmd/fab/operator_handover_test.go` (or the matching `_test.go` files): with probe stubbed true and `rkCronRunner` + the delegate runner stubbed to record argv — add/list/rm/observe argv; update split ordering and stop-on-failure; `--pause --resume` usage error with zero rk calls; tick-start → `operator check`; track clock → exit 1 with zero rk calls; clock sync → only `cron list`; gate → no `cron add`/`cron mute`/`cron edit` from `syncOperatorClock(false,true)`, `reconcileOperatorMute`, `muteOperatorClockIfUntracked`, `ensureOperatorCronRow`, `reconcileOperatorSchedule`; delegated verbs leave a temp `XDG_STATE_HOME` empty; deprecation line on stderr only. The delegate runner needs an injectable seam (e.g. `rkDelegateRunner`) so no real rk runs <!-- R2, R3, R4, R5, R6, R7 --> <!-- rework: cover the '--' note argv, the Available-Commands-scoped matcher decoy, the track-clock deprecation wording, and the negative-exit-code → 1 mapping -->
- [x] T010 Run `go test ./...` in `src/go/fab` with the probe defaulting to false and confirm all pre-existing operator tests pass unchanged <!-- R8 -->

### Phase 3: Integration & Edge Cases

- [x] T011 Move today's `src/kit/skills/fab-operator.md` body verbatim to `src/kit/skills/_operator-legacy.md` with the R10 frontmatter (internal partial, original helpers). Then rewrite `src/kit/skills/fab-operator.md` as the R9 pointer (rk probe with the heading check, adopt `--dry-run` hint, legacy fallback with the deployed-path → `$(fab kit-path)/skills/_operator-legacy.md` rule). No fab-kit-only path citations <!-- R9, R10 -->
- [x] T012 Sweep `src/kit/skills/{_cli-fab-operator,_cli-fab-pane,_cli-agents,_cli-external,_cli-fab,_preamble,fab-setup}.md` and the new legacy partial itself: re-point ``fab-operator.md` §N`` legacy-section citations to `_operator-legacy.md`. Keep `/fab-operator` command-name references. Update `_preamble.md`'s context-loading example (the pointer loads nothing; the legacy partial carries the reduced 3-file set) <!-- R10, R13 -->
- [x] T013 Update `src/kit/templates/user-skill-fab-operator.md` with the rk-first step (R11). Update `src/go/fab-kit/internal/skills_test.go` fixture/assertions if they pin the template body, and add an assertion that the rendered file contains `rk skill operator` before the project/kit-path steps. Verify `src/go/fab/internal/setupcheck` still passes with the new `description` <!-- R11 -->
- [x] T014 Add `src/kit/migrations/2.28.16-to-2.29.0.md` per R12. Check `docs/memory/distribution/migrations.md` for naming/section conventions and any migration-file test <!-- R12 -->

### Phase 4: Polish

- [x] T015 Update `src/kit/skills/_cli-fab-operator.md` (probe, deprecation, mapping table incl. update split + note-append shift, tick-start, track clock refusal, clock sync arm, cron-write gate, fallback unchanged). Add one-sentence delegation notes to the Go `Long` help of delegated verbs. Update `src/kit/skills/fab-help.md`, `src/go/fab/cmd/fab/fab_help.go` and `src/go/fab/cmd/fab/skill.md` where they describe `/fab-operator` <!-- R13 --> <!-- rework: MUST-FIX _cli-fab-operator.md:28 still says 'The operator subcommand family is never delegated' — qualify it to defer to § Hand-over to HexoKit; re-sweep the file for any other never-delegated claim -->
- [x] T016 [P] Update `docs/specs/operator.md` (v14 row + one-line note) and the `/fab-operator` section of `docs/specs/skills.md` per R14 <!-- R14 --> <!-- rework: docs/specs/architecture.md:504's list of deployed underscore partials omits _operator-legacy.md — add it (aggregate-spec sibling sweep) -->
- [x] T017 Run `go test ./...` in both `src/go/fab` and `src/go/fab-kit` (includes `TestKitContentCitesNoRepoLocalPaths`, `clifab_doc_test`, `fab_help_test`) and `gofmt -l`. Do not run `fab sync` <!-- R8, R12, R13 -->

- [x] T018 Rework cycle 2 (revise plan — aggregate-spec sibling sweep): (a) `docs/specs/skills.md` ~:54 helpers mapping row: `fab-operator` now declares no helpers; add/re-point the row so `_operator-legacy` carries `[_cli-fab-operator, _cli-fab-pane, _cli-agents, _cli-external]`; (b) `docs/specs/skills.md` ~:151 always-load exceptions: `/fab-operator` is a pointer that loads nothing, and its `_operator-legacy` fallback loads the reduced config/constitution/context set (match `src/kit/skills/_preamble.md` ~:50); (c) `docs/specs/architecture.md` ~:500 user-level pointer row: add that it tries `rk skill operator` first (match `fab-setup.md` / `docs/site/skill.md` wording); (d) `src/kit/skills/_cli-fab-operator.md` § Hand-over to HexoKit: name the second semantic shift — delegated `track add` without `--mode` leaves mode unset under rk (fab's `autopilot.merge_mode` config default no longer applies); (e) nice-to-have: make `add`/`rm`/`observe` named constants like `rkListVerb`, and add a one-line comment that `update --pause=false`/`--resume=false` alone is an intended no-op. Then grep `docs/specs/` and `src/kit/` once more for any remaining per-skill fact that still describes `/fab-operator` as the full operator <!-- R13, R14 -->

## Execution Order

- T001 blocks T003–T009. T002 sets the false default that T010 relies on.
- T011 blocks T012 (the sweep targets the new partial name).
- T015 follows T003–T008 (it documents the shipped behaviour).

## Acceptance

### Functional Completeness

- [x] A-001 R1: The probe returns true only for help text with `check` and `brief` command rows, is memoized, LookPath-gated and fail-silent
- [x] A-002 R2: Every `fab operator` subcommand prints exactly one stderr deprecation line under a capable rk, and none otherwise; the bare launcher and `fab agent` never print it
- [x] A-003 R3: `track add/list/rm/observe/update` delegate with correctly rebuilt argv, inherited stdio and rk's exit code; `update` splits into update → pause → resume → note
- [x] A-004 R4: `tick-start [--diff] [--quiet]` delegates to exactly `rk operator check` and writes no fab state
- [x] A-005 R5: `track clock` under a capable rk exits 1 with the `rk cron edit`/`mute --for` pointer and makes no rk call
- [x] A-006 R6: `clock sync` under a capable rk runs only `rk cron list`
- [x] A-007 R7: No `rk cron add`, `rk cron mute` (either direction) or `rk cron edit` is issued by any clock path while the probe is true
- [x] A-008 R9: `fab-operator.md` is a short rk-first pointer with the heading-checked probe, adopt hint and legacy fallback
- [x] A-009 R10: `_operator-legacy.md` holds the previous body verbatim under internal-partial frontmatter with the original helpers, and kit cross-references to legacy sections point at it
- [x] A-010 R11: The user-level template probes `rk skill operator` first, then keeps today's two steps
- [x] A-011 R12: The migration file exists with Summary/Pre-check/Changes/Verification, no-edit/no-data-move statements, adopt + loop-seed/mute + rollback steps
- [x] A-012 R13: `_cli-fab-operator.md`, `_preamble.md`, fab-help surfaces and Go help text describe the hand-over consistently
- [x] A-013 R14: `docs/specs/operator.md` and `docs/specs/skills.md` carry the hand-over row/section

### Behavioral Correctness

- [x] A-014 R8: With the probe false, every pre-existing operator test passes without modification beyond the stub default
- [x] A-015 R7: The tracked-set untracked→tracked flip no longer issues `mute --off` under a capable rk (regression test present)

### Scenario Coverage

- [x] A-016 R1: Tests cover v3.22.12 help (false, including a decoy mid-sentence "check"), v3.22.13 help (true), and rk absent (false)
- [x] A-017 R3: Tests cover the mixed `update` split ordering, stop-on-first-failure, and `--pause --resume` usage error with zero rk calls

### Edge Cases & Error Handling

- [x] A-018 R3: A delegate whose rk process cannot start returns exit 1 with an error, and a non-zero rk exit is propagated unchanged
- [x] A-019 R3: Delegated verbs neither create nor modify fab's state file (temp `XDG_STATE_HOME` stays empty)

### Code Quality

- [x] A-020 Pattern consistency: New seams mirror `rkOperatorPath`/`rkCronRunner` (injectable package vars, argv-only, bounded, fail-silent); naming matches the surrounding `operator_*.go` files
- [x] A-021 No unnecessary duplication: One argv builder and one delegate runner serve every delegated verb; `pane.RunCmdContext`/existing exec helpers are reused where they fit
- [x] A-022 Readability over cleverness: Delegation is a short early-return arm at the top of each RunE, not interleaved with fallback logic; no god functions (>50 lines) introduced
- [x] A-023 No magic strings: rk verb names (operator/track/cron/check/brief and the leaf verbs list/update/pause/resume/note), the help-row header, the `--` terminator and the deprecation templates are named constants
- [x] A-024 CLI ⇒ docs + tests: Every changed `fab operator` behaviour is documented in `_cli-fab-operator.md` and covered by tests
- [x] A-025 Canonical source only: No edits under `.agents/skills/` or `.claude/skills/`; kit changes live in `src/kit/`
- [x] A-026 Deployed content cites no fab-kit-only paths: The pointer, legacy partial, template and migration pass `TestKitContentCitesNoRepoLocalPaths`
- [x] A-027 Owner-or-pointer: The probe success rule is stated once in the pointer skill (owner) and referenced, not restated, by the template, or restated only because the template is a separately generated file. A justification comment is acceptable
- [x] A-028 Sibling sweep: All `fab-operator.md §` legacy-section citations across `src/kit/` and the aggregate specs are updated

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds the hand-over arm without making existing code redundant: every fallback path (`runOperatorTrack*`, `runOperatorTickStart`, `runOperatorClockSync`, the clock reconcile, the legacy skill body) stays reachable when the probe is false, and its removal is the plan's later deletion step, not this change's.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | No deprecation line on the bare `fab operator` launcher | It already delegates to `rk operator` when capable. The brief scopes deprecation to subcommand verbs | S:60 R:90 A:75 D:65 |
| 2 | Confident | Delegation rebuilds rk argv from cobra's changed flags, not raw `os.Args` | Reuses fab's flag validation and parsing; argv-only safety | S:60 R:90 A:80 D:70 |
| 3 | Confident | Pointer's adopt hint uses `rk operator adopt --dry-run` instead of `fab operator state` | `fab operator state` creates a skeleton file (a side effect) and now prints the deprecation line. `adopt --dry-run` is read-only and reports what would import | S:60 R:90 A:75 D:70 |
| 4 | Confident | Migration slot named `2.28.16-to-2.29.0.md` (release-owned placeholder) | Follows the `{last-patch}-to-{next-minor}` naming of existing files. A deprecation is minor-bump material | S:65 R:90 A:75 D:70 |
| 5 | Confident | The pointer skill loads no always-load layer and declares no helpers | Its only job is a probe and a redirect; the legacy partial keeps its own reduced context set | S:65 R:90 A:75 D:70 |
| 6 | Certain | Constitution is not edited | Principle V's sentence stays true for the fallback branch. Intake said flag, don't silently edit | S:80 R:90 A:85 D:80 |
| 7 | Confident | `track clock` refusal exits 1 (operational), not 2 (usage) | The invocation is well-formed; the capability moved elsewhere | S:55 R:95 A:70 D:60 |
| 8 | Confident | The deprecation line is one `PersistentPreRunE` on `fab operator`, exempting the command named `operator` (the bare launcher) | One hook covers every subcommand, including verbs added later; avoids touching each RunE | S:60 R:90 A:80 D:70 |
| 9 | Confident | A delegated rk exit code rides a typed `exitCodeError` that `run()` returns without printing; `run()` also now wires `SetErr(errW)` | Inherited rk stdio has already reported; `SetErr` makes subcommand stderr (deprecation lines) land on the injected writer and is a no-op for `main()` | S:55 R:85 A:75 D:65 |
| 10 | Confident | `github.com/spf13/pflag` moves from indirect to a direct require in `src/go/fab/go.mod` | The argv builder walks `cmd.Flags().Visit`; pflag was already vendored transitively through cobra | S:60 R:95 A:85 D:75 |
| 11 | Confident | `track update` with no changed flags still delegates `rk operator track update <id>` | Leaves "nothing to update" validation to rk rather than duplicating it | S:55 R:90 A:70 D:60 |
| 12 | Confident | `docs/site/skill.md` is edited (and synced into the embedded `skill.md` via `scripts/sync-skill.sh`) for the pointer's rk-first note | The embedded bundle is a drift-guarded copy of the canonical docs/site file | S:60 R:90 A:80 D:70 |

12 assumptions (1 certain, 11 confident, 0 tentative).
