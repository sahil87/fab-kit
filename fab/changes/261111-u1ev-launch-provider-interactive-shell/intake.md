# Intake: Launch Provider Commands Through the User's Interactive Shell

**Change**: 261111-u1ev-launch-provider-interactive-shell
**Created**: 2026-11-11

## Origin

> Launch provider commands through the user's interactive shell.
>
> Problem (from hexokit backlog [sca2], seen 2026-11-11): `fab dispatch open` / `fab dispatch start` pane workers hand the composed `providers.<p>.interactive_command` straight to tmux as the new-window/split-window shell-command argument (internal/pane/create.go: OpenWindow, OpenSplitPane, OpenManualWindow; `fab pane open` uses OpenPlainPane). tmux runs it via its default-shell `-c` — non-interactive, no rc files — so env exported only by interactive-shell startup (e.g. CLAUDE_CODE_OAUTH_TOKEN in ~/.zshrc) is missing and a claude worker sits at "Not logged in". The headless rung has the same gap: internal/dispatch/dispatch.go WrapperArgv runs the command under plain `sh -c`. hexokit currently works around it per-repo by wrapping its interactive_command in `${SHELL:-/bin/sh} -i -c '…'` in fab/project/config.yaml; run-kit's own agent panes launch the same way (hexokit app/backend/internal/riff/shell.go buildSkillShellString). The fix belongs in fab-kit so no repo needs that override.

One-shot invocation (`/fab-new` → `/fab-fff`, driven from a prompt file) with four decisions **already made by the user** (recorded verbatim below and as Certain assumptions) and a list of behaviors the user explicitly asked to be **verified, not assumed**:

1. Add ONE composition helper (alongside `spawn.WithShellFallback` / `WithProfile`) that wraps a resolved provider command as `${SHELL:-/bin/sh} -i -c '<single-quoted cmd>'` (quote with the `'\''` idiom). Use it at EVERY site where fab hands a provider command to tmux or to `sh -c`: the dispatch pane path (`launchPane` in `cmd/fab/dispatch_start.go`, covering `start --pane` and `open`, including the manual-window demotion), `fab pane open`, and the headless `WrapperArgv` inner command. Also survey other spawn sites (`operator.go`, `batch_switch.go`, `agent.go`, `internal/agent`, `internal/spawn`) and apply it wherever a provider command is launched by tmux/sh rather than by the user's own interactive shell. `fab agent` itself execs in the caller's shell and does not need it.
2. NO `exec "$SHELL"` tail for dispatch workers — their state machine treats pane death as the worker's terminal event (see `spawn.WithShellFallback`'s doc). Interactive spawns that already use `WithShellFallback` keep it, composed after the wrap.
3. Do NOT add login-wall / "Not logged in" text matching to the readiness gate (`internal/pane/gate.go`). The gate stays text-free by design; fixing the env removes the cause.
4. Must stay idempotent-safe for users who already wrap their `interactive_command` (like hexokit does): a double wrap must still work (verify), and document that the per-repo wrap is now unnecessary.

Things to verify, not assume:
- With the wrap, `#{pane_current_command}` still becomes the provider binary once it starts (bash/zsh exec the sole `-c` command), so the gate's shell-foreground ⇒ booting rule still holds; rc-file loading time reads as booting.
- Headless: `timeout N ${SHELL} -i -c '…' < prompt > log 2>&1; echo $? > exit` — stdin redirection still reaches the provider, the exit code is the provider's, timeout still kills the provider, and job-control noise ("no job control in this shell") either does not appear or is acceptable in the log. Check both bash and zsh as `$SHELL`, and unset `$SHELL` (falls back to `/bin/sh`).
- The agy/kimi `headless_command` values are themselves `sh -c '…'` — wrapping must not break them.
- Update docs/specs and defaults.yaml prose that describes the command as "launched verbatim", tests for the helper and for each call site's composed argv, and memory per the pipeline.

## Why

**The pain.** Every place fab launches a provider CLI on the user's behalf goes through a *non-interactive* shell:

- **tmux spawns** (`new-window` / `split-window` with a shell-command argument) run that argument as `$default-shell -c '<cmd>'`. A non-interactive `-c` shell sources no `~/.zshrc` / `~/.bashrc` (zsh reads only `~/.zshenv`; bash reads nothing unless `BASH_ENV` is set).
- **headless dispatch** runs `sh -c '<cmd> < prompt > log 2>&1; echo $? > exit'` from a detached Go process — again no rc files, and `sh` rather than the user's shell.

Users routinely export provider credentials and PATH additions only in their interactive rc (the reported case: `CLAUDE_CODE_OAUTH_TOKEN` in `~/.zshrc`). A dispatched claude worker therefore boots unauthenticated and parks at "Not logged in" — the pane gate sees a parked screen, the pipeline escalates, and the stage never runs. Headless workers fail the same way, surfacing as `failed` with an auth error in the log.

**If we don't fix it.** Every consuming repo has to copy hexokit's workaround — hand-wrapping `providers.<p>.interactive_command` in `${SHELL:-/bin/sh} -i -c '…'` in its own `fab/project/config.yaml` — which (a) only fixes the provider/field it was applied to (the headless command and every other provider stay broken), (b) has to be re-derived per repo, and (c) silently overrides the built-in command so future default changes (new flags, `{model}`/`{effort}` templating fixes) stop reaching that repo.

**Why this approach.** The launch environment is a property of *how fab launches*, not of any provider's grammar, so it belongs in one composition helper applied at the launch seams — exactly where `WithShellFallback` already lives for the interactive-spawn pane-survival concern. Alternatives rejected by the user:
- **Readiness-gate text matching for "Not logged in"** — rejected (decision 3): the gate is deliberately text-free; matching a login wall treats the symptom and the gate would still escalate.
- **An `exec "$SHELL"` tail for dispatch workers** — rejected (decision 2): dispatch's pane state machine treats pane death as the worker's terminal event (`orphaned`/done detection), so the pane must die with the provider.
- **Changing the built-in `interactive_command` strings in `defaults.yaml`** to carry the wrap — rejected implicitly by decision 1 (one helper at the launch seams): it would not cover user-defined providers or user overrides, and would make the `{model}`/`{effort}` token-drop grammar operate inside a quoted nested shell.

## What Changes

### 1. New helper: `spawn.WithInteractiveShell` (`src/go/fab/internal/spawn/spawn.go`)

One composition helper next to `WithShellFallback` / `WithProfile`:

```go
// WithInteractiveShell wraps a resolved provider command so it runs under the
// user's interactive shell: `${SHELL:-/bin/sh} -i -c '<cmd>'`. ...
func WithInteractiveShell(cmd string) string {
	return `${SHELL:-/bin/sh} -i -c ` + shellquote.Single(cmd)
}
```

- Quoting reuses the existing `internal/shellquote.Single` (POSIX single quotes, embedded `'` → `'\''`) — no new quoting code (code-quality: don't duplicate utilities).
- `${SHELL:-/bin/sh}` is expanded by the **outer** shell (tmux's default-shell `-c`, or the headless wrapper's `sh -c`), so it resolves to the user's login shell when `$SHELL` is set and falls back to `/bin/sh` when unset or empty.
- The wrapped command is inserted **unexpanded**: everything inside it (`"$(basename "$(pwd)")"` in the claude built-in, `"$(cat)"` in the agy/kimi headless built-ins) is evaluated by the inner interactive shell, after rc files have loaded.
- **No idempotence detection.** The helper does not sniff for an already-wrapped command; a pre-wrapped user command (hexokit's `${SHELL:-/bin/sh} -i -c '…'`) simply nests — the outer interactive shell runs the inner one, which runs the provider. Decision 4 requires this double wrap to *work* (verified in apply), not to be collapsed. Cost: rc files are sourced twice for such users — acceptable, and the docs tell them the per-repo wrap can be removed.
- Doc comment states: what it fixes (rc-only env such as provider auth tokens), that it adds no `exec "$SHELL"` tail (pane-death semantics for dispatch workers), its composition order relative to `WithShellFallback` (wrap first, fallback outside), and that interactive shells with `-c` exec the sole command (so the provider replaces the shell and `#{pane_current_command}` / pid tracking still see the provider).

Composition orders:

| Site | Composed shell-command |
|------|------------------------|
| dispatch pane (`launchPane`: split, window, manual-window demotion) | `WithInteractiveShell(resolvedCmd)` |
| `fab pane open` | `WithInteractiveShell(resolvedCmd)` |
| headless (`WrapperArgv`) | `sh -c '[timeout N ]${SHELL:-/bin/sh} -i -c '\''<cmd>'\'' < prompt > log 2>&1; echo $? > exit'` — i.e. the script's `<cmd>` slot receives `WithInteractiveShell(resolvedCmd)` |
| operator / batch new / batch switch (interactive spawns) | `WithShellFallback(WithInteractiveShell(withWorkersEnv(spawnCmd + ' ' + quotedPrompt)))` → `${SHELL:-/bin/sh} -i -c 'FAB_AGENT_WORKERS=codex claude …'; exec "$SHELL"` |

### 2. Dispatch pane path (`src/go/fab/cmd/fab/dispatch_start.go`)

`launchPane` receives the resolved command and hands it to `pane.OpenWindow`, `pane.OpenSplitPane`, and `pane.OpenManualWindow` (the geometry-floor demotion). Wrap once at the top of `launchPane` (or at its caller) so all three creators receive the same wrapped string — no creator is missed by a later branch. `internal/pane/create.go`'s creators stay mechanism-only (they keep taking "the whole left-hand side"); their doc comments may mention that callers pass an interactive-shell-wrapped command.

The dispatch **record** (`SpawnCmd: resolvedCmd`, line ~449) keeps the **unwrapped** resolved provider command: it documents what provider command ran; the wrap is a launch-time mechanism, identical for every record. `fab agent <stage> -o yaml`'s `command` / `dispatch.command` output is likewise unchanged (pure resolution output; dispatch sites never execute it).

### 3. Headless path (`src/go/fab/internal/dispatch/dispatch.go` `WrapperArgv`, and `launchHeadless`)

The inner command in the `sh -c` wrapper script becomes the interactive-shell-wrapped command:

```sh
sh -c 'timeout 600 ${SHELL:-/bin/sh} -i -c '\''claude -p --permission-mode bypassPermissions --model claude-opus-… --effort high'\'' < '\''/repo/.fab-dispatch/u1ev/apply-prompt.md'\'' > … 2>&1; echo $? > …'
```

The outer `sh -c` stays (it is the supervisor that records `$?`; `setsid` via `SysProcAttr` unchanged). Where the wrap is applied — inside `WrapperArgv` or by its caller in `launchHeadless` — is an implementation choice for apply (`internal/dispatch` currently imports neither `spawn` nor `agent`; check for an import cycle before importing `spawn` there). Either way `WrapperArgv`'s doc comment (which currently says "cmd is the resolved spawn command inserted verbatim") is updated to describe the wrap, and tests assert the exact composed argv with and without `timeoutSecs`.

### 4. `fab pane open` (`src/go/fab/cmd/fab/pane_open.go`)

`resolvedCmd := spawn.WithProfile(...)` → pass `spawn.WithInteractiveShell(resolvedCmd)` to `pane.OpenPlainPane`. No `WithShellFallback` (unchanged — `fab pane open` is documented as a dispatch-style primitive whose pane dies with the agent).

### 5. Interactive spawns: `operator.go`, `batch_new.go`, `batch_switch.go` (`src/go/fab/cmd/fab/`)

All three compose `spawnCmd + quoted skill prompt`, prefix the workers env (`withWorkersEnv`), append `WithShellFallback`, and pass the string to `tmux new-window`. They are launched by tmux's non-interactive default-shell, so they have the same gap. Insert `spawn.WithInteractiveShell` **between** `withWorkersEnv` and `WithShellFallback`:

```go
shellCmd = withWorkersEnv(shellCmd, workers, workersSet)
shellCmd = spawn.WithInteractiveShell(shellCmd)
shellCmd = spawn.WithShellFallback(shellCmd)
```

The workers env prefix sits **inside** the wrap so it still scopes the agent command only and is set after rc files load (an rc file cannot clobber it). The `; exec "$SHELL"` fallback stays **outside**, so after the agent exits the pane is taken over by the user's interactive shell exactly as today (decision 2).

Apply MUST also survey `cmd/fab/agent.go`, `internal/agent`, `internal/spawn`, and any other `tmux new-window` / `split-window` / `exec.Command("sh", "-c", …)` site that launches a **provider** command. Expected outcome of the survey (from intake-time grep): `fab agent` execs the provider in the caller's own shell (no wrap — decision 1); `internal/hooks/hooks.go`'s `sh -c` runs user stage hooks, not provider commands (no wrap). Any additional provider-launch site found gets the wrap and a test.

### 6. Verification (apply MUST run these — the user asked for evidence, not assumption)

Run live probes in the worktree (scratch tmux server via `tmux -L <name>` so the user's server is untouched), and record the outcomes in plan.md's acceptance evidence:

1. **Pane foreground command** — open a pane with `${SHELL:-/bin/sh} -i -c 'sleep 30'` (and a double-wrapped variant) and read `#{pane_current_command}`: it must become `sleep` (the provider stand-in), proving bash/zsh exec the sole `-c` command and the gate's shell-foreground ⇒ booting rule still holds (rc loading time shows the shell → `booting`). If zsh/bash does NOT exec (e.g. an rc file that installs traps forces a fork), document the observed behavior and its effect on the gate rather than papering over it.
2. **Headless semantics**, for `SHELL=/bin/bash`, `SHELL=$(command -v zsh)` (if installed), and `SHELL` unset (→ `/bin/sh`), using the real `WrapperArgv` composition:
   - stdin redirection reaches the provider (stand-in: `cat` / `wc -c` echoes the prompt file's content into the log);
   - the exit file holds the provider's exit code (stand-in: `sh -c 'exit 7'` → `7`);
   - `timeout` still kills the provider (stand-in: `sleep 30` with `timeout 1` → exit `124`, no surviving `sleep` process);
   - job-control noise: record whether "no job control in this shell" (bash) or similar appears in the log; it is acceptable as a log line but must be documented.
3. **Nested-shell built-ins** — compose the agy and kimi `headless_command` defaults (`sh -c 'agy … -p "$(cat)"'`, `sh -c 'kimi -m {model} -p "$(cat)"'`) through `WithProfile` + `WithInteractiveShell` + `WrapperArgv` and run them with stand-in binaries on PATH: `"$(cat)"` must still receive the prompt file content.
4. **Double wrap** — `WithInteractiveShell("${SHELL:-/bin/sh} -i -c 'claude …'")` runs the provider (stand-in) correctly in both pane and headless shapes.

Go tests cover the deterministic part: exact composed strings/argv for the helper (plain command, command with embedded single quotes, double wrap, empty string) and for each call site (pane path incl. demotion, `fab pane open`, headless with/without timeout, operator/batch-new/batch-switch with and without the workers env). Live shell behavior can be covered by a test that executes `WrapperArgv` with `/bin/sh` stand-ins where hermetic (`SHELL` set explicitly in the test env), skipping when a needed shell is absent.

### 7. Docs, specs, defaults.yaml prose, CLI reference

- `src/go/fab/defaults.yaml` provider prose (lines ~89–112): wherever it says the command is launched/inserted verbatim, state that fab launches it under the user's interactive shell (`${SHELL:-/bin/sh} -i -c '…'`) so rc-exported env (auth tokens, PATH) is present, and that hand-wrapping an `interactive_command` in a repo config is no longer necessary (still works — it just nests). The `fab config explain` text derived from it follows.
- `src/kit/skills/_cli-fab-pane.md` (owner of `fab pane` / `fab dispatch` reference) — the launch description for `fab dispatch start/open` and `fab pane open`; `src/kit/skills/_cli-fab-operator.md` for operator/batch spawn if it describes the launched string. Deployed content must restate rules, never cite fab-kit `docs/`/`src/go/` paths (Constitution V).
- `docs/specs/stage-models.md`, `docs/specs/harness-adapters.md`, `docs/specs/config.md` — human-curated; update only sentences that would become *false* ("launched verbatim" / "`sh -c '<cmd>'`" descriptions of the launch), keeping edits minimal.
- Grep repo-wide for `verbatim`, `sh -c '<cmd>`, `WithShellFallback`, `-i -c` across `docs/`, `src/kit/`, `src/go/fab/defaults.yaml` and sweep the whole sibling class (code-quality § Sibling Sweeps). Note: "verbatim pass-through" in the *resolver* sense (fab does not validate/rewrite model/effort values) remains true and must NOT be changed — only claims about the *launch* string change.

## Affected Memory

- `runtime/agent-primitives`: (modify) profile-resolved spawning — add the interactive-shell launch wrap, its composition order with `WithProfile` / `withWorkersEnv` / `WithShellFallback`, and the double-wrap tolerance
- `runtime/dispatch`: (modify) pane launch (split / window / manual-window demotion) and headless `WrapperArgv` composition now run the provider under `${SHELL:-/bin/sh} -i -c`; record keeps the unwrapped command; gate stays text-free; verified foreground/stdin/exit/timeout behavior
- `runtime/pane-commands`: (modify) `fab pane open` launches the resolved command under the interactive-shell wrap
- `runtime/providers-and-profiles`: (modify) provider command semantics — commands are launched under the user's interactive shell; per-repo `-i -c` wrapping is unnecessary
- `runtime/operator`: (modify) operator launcher spawn string gains the wrap inside the shell fallback

## Impact

- **Go code** (`src/go/fab/`): `internal/spawn/spawn.go` (+ test), `internal/dispatch/dispatch.go` (`WrapperArgv` + test) or `cmd/fab/dispatch_start.go` `launchHeadless`, `cmd/fab/dispatch_start.go` `launchPane` (+ tests), `cmd/fab/pane_open.go` (+ test), `cmd/fab/operator.go`, `cmd/fab/batch_new.go`, `cmd/fab/batch_switch.go` (+ tests). `internal/pane/create.go` doc comments only (creators unchanged).
- **Config prose**: `src/go/fab/defaults.yaml` comment text (no value changes → no migration needed; user data is not restructured).
- **Kit skills**: `src/kit/skills/_cli-fab-pane.md`, possibly `_cli-fab-operator.md` (CLI reference rule).
- **Specs/memory**: as listed above.
- **Runtime behavior**: every provider launch now sources the user's interactive rc files first (startup cost = rc load time, observed by the pane gate as `booting`). Users whose rc prints to stdout will see that output in headless logs / at pane start. Users who already hand-wrap get a nested, still-working launch.
- **Not changed**: `internal/pane/gate.go` (no text matching), dispatch state machine / pane-death semantics, `fab agent` exec path, `fab agent -o yaml` output, stage hooks.

## Open Questions

- None blocking. Platform caveat to record, not resolve: on a host where `$SHELL` names a non-POSIX shell (fish, nushell), `-i -c '<posix cmd>'` would be interpreted by that shell; the built-in commands use POSIX `$(…)` syntax which fish ≥3.4 accepts but nushell does not. hexokit's workaround has the same property. Apply should note this limitation in the helper's doc comment / memory, not add shell detection.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | One helper wraps the resolved provider command as `${SHELL:-/bin/sh} -i -c '<cmd>'` with `'\''` quoting, applied at every tmux/`sh -c` provider-launch site | Discussed — user decision 1 | S:95 R:80 A:90 D:95 |
| 2 | Certain | No `exec "$SHELL"` tail for dispatch workers; interactive spawns keep `WithShellFallback`, composed outside the wrap | Discussed — user decision 2 | S:95 R:85 A:90 D:95 |
| 3 | Certain | Readiness gate (`internal/pane/gate.go`) gets no login-wall text matching | Discussed — user decision 3 | S:95 R:90 A:95 D:95 |
| 4 | Certain | Double wrap must work and be verified; docs state the per-repo wrap is now unnecessary; no wrap-detection logic | Discussed — user decision 4 ("a double wrap must still work") | S:90 R:85 A:85 D:85 |
| 5 | Certain | `fab agent` (execs in caller's shell) and stage hooks (`internal/hooks` `sh -c`, not a provider command) are not wrapped | Discussed — user decision 1 explicitly exempts `fab agent`; hooks run user commands, not provider launches | S:90 R:85 A:85 D:90 |
| 6 | Certain | Helper is `spawn.WithInteractiveShell`, reusing `internal/shellquote.Single` for the quoting | User said "alongside spawn.WithShellFallback / WithProfile"; shellquote.Single already implements the `'\''` idiom | S:80 R:90 A:85 D:75 |
| 7 | Confident | Operator / batch new / batch switch get the wrap too, with order `withWorkersEnv` → `WithInteractiveShell` → `WithShellFallback` (env inside the wrap) | User asked to survey these and apply wherever tmux launches a provider command; all three use tmux `new-window` with a non-interactive default-shell. Env inside keeps it scoped to the agent and immune to rc clobbering | S:75 R:85 A:80 D:70 |
| 8 | Confident | Dispatch record `SpawnCmd` and `fab agent -o yaml` command output stay the unwrapped resolved command; the wrap is applied only at launch | The wrap is a constant launch mechanism, not provider resolution; keeps records/YAML stable and comparable | S:60 R:85 A:75 D:70 |
| 9 | Certain | Headless wrap lands inside `sh -c`'s script in the `<cmd>` slot (after `timeout N`), keeping the outer `sh -c` supervisor and setsid tracking | Decision 1 names "the headless WrapperArgv inner command"; the outer sh records `$?` and must stay | S:85 R:85 A:80 D:80 |
| 10 | Certain | No migration: only comment prose in defaults.yaml changes; no config value or schema change | Migrations are for restructuring user data (context.md); none is restructured | S:80 R:90 A:90 D:90 |
| 11 | Confident | Non-POSIX `$SHELL` (fish/nushell) is documented as a limitation, not handled | Same property as the hexokit workaround the user cited; shell detection would be new scope. | S:40 R:80 A:55 D:50 |

11 assumptions (8 certain, 3 confident, 0 tentative, 0 unresolved).
