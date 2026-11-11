# Plan: Launch Provider Commands Through the User's Interactive Shell

**Change**: 261111-u1ev-launch-provider-interactive-shell
**Intake**: `intake.md`

## Requirements

### Spawn: Interactive-Shell Wrap Helper

#### R1: One composition helper wraps a provider command in the user's interactive shell
`internal/spawn` SHALL export `WithInteractiveShell(cmd string) string` returning exactly `${SHELL:-/bin/sh} -i -c ` followed by `shellquote.Single(cmd)` (POSIX single quotes, embedded `'` → `'\''`). The helper MUST NOT detect or collapse an already-wrapped command, MUST NOT append any `exec "$SHELL"` tail, and MUST leave the inner command's text unexpanded (all `$…` inside it is evaluated by the inner interactive shell after rc files load). Its doc comment SHALL state: why (rc-only env such as provider auth tokens / PATH), the `${SHELL:-/bin/sh}` fallback, that the interactive shell execs (or forks and waits on) the sole `-c` command so the provider becomes the pane's foreground process and the pane still dies with it, the composition order relative to `WithShellFallback` (wrap inside, fallback outside), double-wrap tolerance, and the non-POSIX `$SHELL` (fish/nushell) limitation.

- **GIVEN** `cmd = claude --model x -n "$(basename "$(pwd)")"`
- **WHEN** `WithInteractiveShell(cmd)` is called
- **THEN** it returns `${SHELL:-/bin/sh} -i -c 'claude --model x -n "$(basename "$(pwd)")"'`

- **GIVEN** `cmd = sh -c 'kimi -p "$(cat)"'` (embedded single quotes)
- **WHEN** wrapped
- **THEN** it returns `${SHELL:-/bin/sh} -i -c 'sh -c '\''kimi -p "$(cat)"'\'''`

- **GIVEN** an already-wrapped `${SHELL:-/bin/sh} -i -c 'claude'`
- **WHEN** wrapped again
- **THEN** the result nests (`${SHELL:-/bin/sh} -i -c '${SHELL:-/bin/sh} -i -c '\''claude'\'''`) and still runs the provider

### Dispatch: Pane Launch

#### R2: Every dispatch pane creator receives the wrapped command
`launchPane` (`cmd/fab/dispatch_start.go`, used by `fab dispatch start` pane mode, `fab dispatch open`, and `restart`) SHALL pass `spawn.WithInteractiveShell(resolvedCmd)` to `pane.OpenSplitPane`, `pane.OpenWindow`, AND `pane.OpenManualWindow` (geometry-floor demotion). The wrap SHALL be applied once, before the shape branch, so no creator branch can miss it. No `WithShellFallback` is added (pane death stays the worker's terminal event). The dispatch record's `SpawnCmd` SHALL remain the UNWRAPPED resolved command.

- **GIVEN** a pane dispatch on a reachable tmux server
- **WHEN** `fab dispatch open` launches the worker (window, split, or demoted window)
- **THEN** the pane's `#{pane_start_command}` is the interactive-shell-wrapped command
- **AND** `rec.SpawnCmd` equals the unwrapped `WithProfile` result

#### R3: The readiness gate is unchanged
`internal/pane/gate.go` SHALL NOT gain login-wall / "Not logged in" text matching. The shell-foreground ⇒ `booting` rule continues to hold because the wrapped command's provider becomes `#{pane_current_command}` once started (rc-loading time reads as the shell → `booting`).

- **GIVEN** a pane launched with the wrap
- **WHEN** the inner shell is still loading rc files
- **THEN** `#{pane_current_command}` is the shell (gate reports `booting`), and becomes the provider binary once it starts

### Dispatch: Headless Launch

#### R4: The headless wrapper runs the provider under the interactive shell
`dispatch.WrapperArgv` (`internal/dispatch/dispatch.go`) SHALL compose `[sh, -c, "[timeout N ]${SHELL:-/bin/sh} -i -c '<cmd>' < '{prompt}' > '{log}' 2>&1; echo $? > '{exit}'"]`, using `spawn.WithInteractiveShell(cmd)` for the `<cmd>` slot. The outer `sh -c` supervisor, `SysProcAttr{Setsid:true}` detach, and `timeout` placement (prefixing the wrapped command) are unchanged. Behavior: stdin redirection reaches the provider, the exit file records the provider's exit code, `timeout` kills the provider (exit 124, no survivors), and nested-shell headless built-ins (`sh -c '… -p "$(cat)"'`) still receive the prompt. Shell startup noise (bash's "no job control in this shell", rc-file output) MAY appear in `{stage}.log` and is acceptable — the result file, not the log, is the success token.

- **GIVEN** `cmd = codex exec`, timeout 600
- **WHEN** `WrapperArgv` composes
- **THEN** argv[2] is `timeout 600 ${SHELL:-/bin/sh} -i -c 'codex exec' < 'p.md' > 'l.log' 2>&1; echo $? > 'e.exit'`

- **GIVEN** `$SHELL` is `/bin/bash`, `zsh`, or unset
- **WHEN** the wrapper runs a stand-in provider that reads stdin and exits 7
- **THEN** the log contains the prompt-derived output and the exit file contains `7`

### Pane Primitive

#### R5: `fab pane open` launches the wrapped command
`cmd/fab/pane_open.go` SHALL pass `spawn.WithInteractiveShell(resolvedCmd)` to `pane.OpenPlainPane`, with no shell fallback.

- **GIVEN** `fab pane open --provider kimi`
- **WHEN** it spawns
- **THEN** the tmux spawn argv's shell-command argument is `${SHELL:-/bin/sh} -i -c 'kimi --auto'`

### Interactive Spawns

#### R6: Operator, batch new, and batch switch launch under the wrap, fallback outside
`cmd/fab/operator.go`, `cmd/fab/batch_new.go`, `cmd/fab/batch_switch.go` SHALL compose `spawn.WithShellFallback(spawn.WithInteractiveShell(withWorkersEnv(spawnCmd + " " + quotedPrompt, …)))`: the workers env prefix inside the wrap (scoped to the agent, set after rc loads), the `; exec "$SHELL"` fallback outside it (unchanged pane-survival behavior).

- **GIVEN** `fab operator` with `--workers codex`
- **WHEN** the tmux `new-window` command is composed
- **THEN** it is `${SHELL:-/bin/sh} -i -c 'FAB_AGENT_WORKERS=codex claude … '\''/fab-operator'\'''; exec "$SHELL"`

#### R7: Non-provider and caller-shell launches are not wrapped
`fab agent` (execs the provider in the caller's own interactive shell) and `internal/hooks` (`sh -c` of user stage hooks, not provider commands) SHALL NOT be wrapped. The apply survey SHALL confirm no other provider-launch site exists (any found gets the wrap + a test).

- **GIVEN** a grep of `src/go/fab` for `new-window`, `split-window`, `"sh", "-c"`, `syscall.Exec`/`exec.Command` of provider commands
- **WHEN** the survey completes
- **THEN** every provider launch by tmux/sh is wrapped, and `fab agent` / hooks are untouched

### Docs: Launch Prose

#### R8: Prose describing the launch string is updated; per-repo wrap documented as unnecessary
`src/go/fab/defaults.yaml` provider prose, `src/kit/skills/_cli-fab-pane.md` (the pane-mode "verbatim" table cell, `open`'s "reaches tmux verbatim" sentence, the headless wrapper line), `src/kit/skills/_cli-fab.md` § providers paragraph, and `docs/specs/harness-adapters.md` / `docs/specs/stage-models.md` sentences that describe the LAUNCH as verbatim or as plain `sh -c '<cmd>'` SHALL state that fab launches the composed command under `${SHELL:-/bin/sh} -i -c '…'`, and that hand-wrapping an `interactive_command`/`headless_command` in a repo config is no longer necessary (it still works — it nests). "Verbatim" in the RESOLVER sense (no prompt appended; no model/effort validation) MUST be preserved. Deployed `src/kit/` prose MUST NOT cite fab-kit `docs/` or `src/go/` paths.

- **GIVEN** a reader of `_cli-fab-pane.md` § fab dispatch
- **WHEN** they look up how the pane command is launched
- **THEN** it says nothing is appended to `interactive_command` AND that it runs under the user's interactive shell

### Non-Goals

- Login-wall text matching in the readiness gate — rejected by user decision 3.
- Changing built-in `interactive_command`/`headless_command` values in `defaults.yaml` — the wrap is a launch mechanism, not provider grammar.
- Detecting/collapsing a pre-wrapped user command — double wrap works; detection would be fragile string sniffing.
- Handling non-POSIX `$SHELL` (fish/nushell) — documented limitation only.
- A config migration — no config value or schema changes.

### Design Decisions

#### Wrap at the Launch Seams, Not in Provider Grammar
**Decision**: One helper, `spawn.WithInteractiveShell`, applied where fab hands a provider command to tmux or `sh -c`; the resolved command, the dispatch record, and `fab agent -o yaml` stay unwrapped.
**Why**: The missing env is a property of how fab launches, so one seam covers every provider (built-in and user-defined) and every override; records keep documenting the provider command.
**Rejected**: Baking `${SHELL} -i -c` into the built-in `defaults.yaml` commands — misses user-defined providers/overrides and pushes `{model}`/`{effort}` token-drop into a quoted nested shell. Gate text matching — treats the symptom.
*Introduced by*: 261111-u1ev-launch-provider-interactive-shell

#### Headless Wrap Lives Inside `WrapperArgv`
**Decision**: `WrapperArgv` itself wraps its `cmd` slot (internal/dispatch imports internal/spawn — no import cycle), so the headless argv contract is pinned in one tested function.
**Why**: Keeps the full headless launch grammar in one place and testable by exact string; callers cannot forget the wrap.
**Rejected**: Wrapping in `launchHeadless` — splits the argv contract across two packages and leaves `WrapperArgv`'s exact-argv test blind to it.
*Introduced by*: 261111-u1ev-launch-provider-interactive-shell

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add `WithInteractiveShell` to `src/go/fab/internal/spawn/spawn.go` (reusing `internal/shellquote.Single`), with doc comment per R1; add table tests in `src/go/fab/internal/spawn/spawn_test.go` (plain, embedded `'`, `$(…)` left unexpanded, empty string, double wrap, composition with `WithShellFallback`) <!-- R1 -->
- [x] T002 Update `WrapperArgv` in `src/go/fab/internal/dispatch/dispatch.go` to wrap `cmd` via `spawn.WithInteractiveShell` (timeout prefixes the wrapped command); rewrite its doc comment (drop "inserted verbatim" for the launch); update `TestWrapperArgv`/`TestWrapperArgvQuotesPathsWithSpaces` exact strings and add a case with an embedded single-quote command (agy/kimi shape) in `src/go/fab/internal/dispatch/dispatch_test.go` <!-- R4 -->
- [x] T003 Add a live headless test in `src/go/fab/internal/dispatch/dispatch_test.go` executing `WrapperArgv`'s argv via `exec.Command` (hermetic: `SHELL` set per subtest via the cmd env to `/bin/sh`, `bash`, `zsh` when found, and unset; `HOME` pointed at a temp dir so no user rc loads) asserting stdin reaches the stand-in provider, the exit file holds its code (7), `timeout 1` on `sleep` yields 124, and a nested `sh -c '… "$(cat)"'` receives the prompt; skip when a shell / `timeout` is absent <!-- R4 -->
- [x] T004 In `src/go/fab/cmd/fab/dispatch_start.go` `launchPane`, wrap `resolvedCmd` once with `spawn.WithInteractiveShell` before the shape branch so split, window, and manual-window demotion all receive it; keep `SpawnCmd` unwrapped; update the `launchHeadless` comment; extend a pane integration test in `src/go/fab/cmd/fab/dispatch_open_test.go` (window shape, and the below-floor demotion test) to assert `#{pane_start_command}` carries the wrap and `rec.SpawnCmd` does not <!-- R2 -->
- [x] T005 In `src/go/fab/cmd/fab/pane_open.go`, pass `spawn.WithInteractiveShell(resolvedCmd)` to `pane.OpenPlainPane`; update `TestPaneOpen_SpawnShapes` in `src/go/fab/cmd/fab/pane_open_test.go` to require the wrapped fragment (`-i -c 'kimi --auto`) <!-- R5 -->
- [x] T006 In `src/go/fab/cmd/fab/operator.go`, `batch_new.go`, `batch_switch.go`, insert `shellCmd = spawn.WithInteractiveShell(shellCmd)` between `withWorkersEnv` and `WithShellFallback` (update adjacent comments); update `launcher_prompt_test.go`, `batch_new_test.go`, `batch_switch_test.go` assertions (prefix `${SHELL:-/bin/sh} -i -c '`, suffix `'; exec "$SHELL"`, workers env inside the quotes) — note `launcher_prompt_test.go` evaluates the captured command, so adapt its unquoting to the extra quoting layer <!-- R6 -->

### Phase 2: Integration & Edge Cases

- [x] T007 Survey `src/go/fab` for every other provider-launch site (`new-window`, `split-window`, `"sh", "-c"`, `syscall.Exec`, `exec.Command` running a resolved provider command, incl. `cmd/fab/agent.go`, `internal/agent`, `internal/spawn`); confirm `fab agent` execs in the caller's shell and `internal/hooks` runs user hooks (both unwrapped); wrap + test any other provider launch found. Record the survey result in this plan's Notes <!-- R7 -->
- [x] T008 Update `src/go/fab/internal/pane/create.go` creator doc comments (OpenWindow/OpenManualWindow/OpenSplitPane/OpenPlainPane) and `spawn.WithShellFallback`'s doc to mention callers pass the interactive-shell-wrapped command; confirm `internal/pane/gate.go` is unchanged <!-- R3 -->
- [x] T009 Run `go test ./...` from `src/go/fab` (scoped packages first: `./internal/spawn ./internal/dispatch ./cmd/fab`), fix failures (tests conform to the requirements) <!-- R2 -->

### Phase 3: Polish

- [x] T010 Update launch prose per R8: `src/go/fab/defaults.yaml` provider comment block (launch under the interactive shell; per-repo wrap unnecessary but harmless), `src/kit/skills/_cli-fab-pane.md` (pane-mode table cell, `open` paragraph, headless wrapper line ~151, `fab pane open`), `src/kit/skills/_cli-fab.md` § providers paragraph, `docs/specs/harness-adapters.md` (lines ~139, ~170, ~264) and `docs/specs/stage-models.md` where they describe the launch; sweep `grep -rn "verbatim\|sh -c '<"` across `src/kit docs/specs src/go/fab/defaults.yaml` and fix every launch-sense hit, preserving resolver-sense "verbatim"; re-run any defaults.yaml / config-explain / kit-citation guard tests <!-- R8 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `spawn.WithInteractiveShell` exists, returns `${SHELL:-/bin/sh} -i -c '<cmd>'` with `'\''` quoting via `shellquote.Single`, and is covered by table tests including double wrap
- [ ] A-002 R2: `launchPane` wraps once before the shape branch; split, window, and manual-window paths all receive the wrapped command; `SpawnCmd` stays unwrapped
- [ ] A-003 R4: `WrapperArgv` composes `[timeout N ]${SHELL:-/bin/sh} -i -c '<cmd>' < prompt > log 2>&1; echo $? > exit` under `sh -c`, pinned by exact-string tests
- [ ] A-004 R5: `fab pane open` passes the wrapped command to `OpenPlainPane`, pinned by `TestPaneOpen_SpawnShapes`
- [ ] A-005 R6: operator / batch new / batch switch compose env → wrap → fallback, pinned by updated tests
- [ ] A-006 R8: launch prose in defaults.yaml, `_cli-fab-pane.md`, `_cli-fab.md`, and the specs describes the interactive-shell launch and states the per-repo wrap is unnecessary

### Behavioral Correctness

- [ ] A-007 R4: live headless test passes for `/bin/sh`, bash, zsh (when present), and unset `SHELL`: stdin reaches the provider, exit code is the provider's, `timeout` yields 124, nested `"$(cat)"` receives the prompt
- [ ] A-008 R2: a pane integration test observes the wrapped `#{pane_start_command}`

### Scenario Coverage

- [ ] A-009 R3: `internal/pane/gate.go` has no diff; the foreground-command behavior under the wrap is evidenced (Notes § Verification Evidence)
- [ ] A-010 R7: the spawn-site survey is recorded in Notes; `fab agent` and `internal/hooks` are unwrapped

### Edge Cases & Error Handling

- [ ] A-011 R1: double wrap works in pane and headless shapes (test + evidence)
- [ ] A-012 R4: agy/kimi nested-shell headless commands survive the wrap (test case with embedded `'`)

### Code Quality

- [ ] A-013 Pattern consistency: new code follows the naming/doc-comment style of `WithShellFallback` / `WithProfile`
- [ ] A-014 No unnecessary duplication: quoting reuses `shellquote.Single`; no new quoting helper
- [ ] A-015 Readability over cleverness: composition order is explicit and commented at each interactive-spawn site
- [ ] A-016 CLI reference + tests: `_cli-fab-pane.md` / `_cli-fab.md` updated alongside Go changes; every changed `.go` file ships test updates
- [ ] A-017 Deployed content cites no fab-kit-only paths: `src/kit/` edits restate rules and point at no `docs/` / `src/go/` path

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

### Verification Evidence (pre-apply live probes, 2026-11-11, linux, tmux 3.8)

Headless — `setsid sh -c '[timeout 1 ]${SHELL:-/bin/sh} -i -c '\''…'\'' < p.md > l.log 2>&1; echo $? > e.exit'`, prompt `hello prompt\n`:

| `$SHELL` | stdin → `wc -c; exit 7` | `timeout 1 … sleep` | nested `sh -c 'echo got:"$(cat)"'` | double wrap `wc -c; exit 5` | log noise |
|---|---|---|---|---|---|
| `/bin/bash` | `13`, exit 7 | exit 124, 0 survivors | `got:hello prompt` | `13`, exit 5 | `bash: cannot set terminal process group … / bash: no job control in this shell` (per shell layer) |
| `/usr/bin/zsh` | `13`, exit 7 | exit 124, 0 survivors | `got:hello prompt` | `13`, exit 5 | `can't change option: monitor` + the user's rc output (a p10k/gitstatus error) |
| unset (`/bin/sh`) | `13`, exit 7 | exit 124, 0 survivors | `got:hello prompt` | `13`, exit 5 | `/bin/sh: 0: can't access tty; job control turned off` |

Pane — scratch server (`tmux -L`), `new-window "${SHELL:-/bin/sh} -i -c 'sleep 33'"` and its double wrap, with `SHELL` = bash, zsh, `/bin/sh`: `#{pane_current_command}` reads the shell right after spawn (→ `booting`) and `sleep` after ~2.5s in all six cases. The pane closed when the wrapped command exited (`sleep 2; exit 3` → pane gone), so pane-death semantics hold. Inside tmux, `$SHELL` in the pane environment is tmux's `default-shell` (the user's shell).

### Spawn-Site Survey (T007)

`grep` of `src/go/fab` for `new-window`, `split-window`, `"sh", "-c"`, `syscall.Exec`, and `exec.Command(` launches:

| Site | Launches | Wrapped? |
|------|----------|----------|
| `cmd/fab/dispatch_start.go` `launchPane` → `pane.OpenWindow` / `OpenSplitPane` / `OpenManualWindow` | provider `interactive_command` via tmux | yes (once, before the shape branch) |
| `internal/dispatch` `WrapperArgv` (via `launchHeadless`) | provider `headless_command` via `sh -c` | yes (inside `WrapperArgv`) |
| `cmd/fab/pane_open.go` → `pane.OpenPlainPane` | provider `interactive_command` via tmux | yes |
| `cmd/fab/operator.go`, `batch_new.go`, `batch_switch.go` → `tmux new-window` | session provider + skill prompt via tmux | yes, fallback outside |
| `cmd/fab/agent.go` `execAgent` (`syscall.Exec`) | provider in the caller's own shell process | no (by decision) |
| `cmd/fab/operator.go` `execOperator` → `rk operator` | rk owns the launch (run-kit wraps its own agent panes) | no |
| `internal/hooks` `sh -c` | user stage hooks, not a provider | no |

### Apply-time finding

With `/bin/sh` (dash) as tmux's `default-shell`, an explicit `exec` inside the wrapped command leaves the outer `sh -c` as the foreground process-group leader (`#{pane_current_command}` = `sh`), while a plain simple command (every built-in) is forked by the interactive inner shell into its own process group and reads as the provider. The parked-wall test fixture therefore uses `exec sleep 300` under raw tmux (not wrapped) to model "provider is foreground" independent of the developer's shell; `TestMain` pins `SHELL=/bin/sh` so tmux-backed tests never start the developer's interactive shell (zsh's new-user wizard appears under the temp HOME otherwise).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Headless wrap is applied inside `WrapperArgv` (internal/dispatch imports internal/spawn) | `go list -deps ./internal/spawn` shows no dependency on internal/dispatch, so no cycle; keeps the argv contract in one tested function | S:80 R:90 A:90 D:80 |
| 2 | Certain | Pane wrap is applied once at the top of `launchPane`, leaving `internal/pane` creators mechanism-only | All three creators are reached only through launchPane's branches; wrapping before the branch makes a miss impossible | S:85 R:90 A:90 D:85 |
| 3 | Confident | Live headless tests run hermetically (`HOME` → temp dir, `SHELL` per subtest) and skip when a shell is absent | Avoids loading the developer's rc in CI/tests; zsh may be absent in CI | S:70 R:90 A:80 D:75 |
| 4 | Confident | Shell startup noise in `{stage}.log` is accepted and documented, not suppressed | Result file is the success token; suppressing stderr would hide real rc errors; probes show the provider output and exit code are intact | S:70 R:85 A:75 D:70 |

4 assumptions (2 certain, 2 confident, 0 tentative).
