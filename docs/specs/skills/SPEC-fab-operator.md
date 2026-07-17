# fab-operator

## Contents

- Summary
- Section Structure
- Primitives
- Monitoring Tick
- Watches (§7)
- Auto-Nudge
- Autopilot
- Key Properties
- Resolved Design Decisions

## Summary

Standalone multi-agent coordination layer with proactive monitoring and auto-nudge. Runs in a dedicated tmux pane, observes all running fab agents across every session on its tmux server via `fab pane map --all-sessions`, routes commands via `tmux send-keys`, monitors progress via `/loop`, auto-answers routine agent questions, and drives autopilot queues through the full pipeline.

**Multi-repo / multi-session model.** The operator coordinates agents across **multiple repos and multiple tmux sessions on a single tmux server** — one operator per server (the isolation unit; a second operator means a second `tmux -L <label>` server). Every agent is addressed by the `(session, repo, pane)` tuple: the **pane ID is the primary key** (server-global, stable), with `repo` (the agent's absolute main-worktree root) and `session` (its tmux session name) layered on as dimensions. Every monitored entry, every `branch_map` value (`{ branch, repo }`), and every watch (`target_repo`) is repo-qualified. State lives in **one server-keyed file** — `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/...`), keyed by the tmux socket path — not a per-repo `.fab-operator.yaml`. This model consumes the Go primitives shipped by change 1 (`260607-h3jk`): `fab pane map --all-sessions --json` with a per-row `repo` field, `fab agent --print --repo` (the profile-resolved session command; formerly `fab spawn-command --repo`), and the binary-derived server-keyed state path. Old repo-rooted state files are not migrated.

Self-contained — does not inherit from any other operator skill. All behavior is defined in `src/kit/skills/fab-operator.md` plus the standard `_` files loaded via `_preamble.md`. External tool reference (`_cli-external.md`) is loaded in the operator's own startup section.

Not a lifecycle enforcer — the operator coordinates across agents and proxies routine user input, not advancing stages or making pipeline decisions.

**Helpers**: Declares `helpers: [_cli-fab, _cli-external]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

**Prose optimization** (260620-skop): skill content trimmed to remove re-explanation of partial-owned concepts (launcher degraded behavior → `_cli-fab.md` § fab operator + §9; state-file path/migration → §2/§9; tick-step field semantics → `_cli-fab.md` § fab pane map; `rk notify` contract → `_cli-external.md` § rk + `_preamble.md` § Run-Kit; implicit `--base` chaining defined once in Queue ordering) and a `## Contents` TOC added; no behavioral change (Flow / Tools / Sub-agents unchanged). A `## Contents` TOC was also added to this SPEC (>100 lines, structural rule).

---

## Section Structure

The skill is organized into 9 sections:

1. **Principles** — identity (coordinate don't execute), **multi-repo aware** (`(session, repo, pane)` addressing; pane ID is the server-global primary key; repo + session are added dimensions), routing discipline, context discipline (never loads change artifacts), state re-derivation via `fab pane map --all-sessions` (why: stale state = wrong actions)
2. **Startup** — trimmed context layer (config/constitution/context only — a `_preamble.md` §1 exception since 260611-zc9m), `_cli-external.md` load, orientation (`fab pane map --all-sessions` + ready signal), reads the server-keyed state file, tmux gate (hard stop when `$TMUX` is unset). **Launch requires neither a git repo nor a resolvable `fab/` project** (260613-2sdj): (a) window cwd — `fab operator` opens its tmux window in the repo root when launched inside a repo, else `os.Getwd()` (the neutral parent dir of the cross-repo singleton), erroring only if both resolutions fail — the old `cannot determine repo root` hard-fail is gone; (b) session command — read from the project's `providers.claude.session_command` when a `fab/` project is resolvable, else the built-in `spawn.DefaultSpawnCommand` (the template `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`), so a missing `fab/` project is non-fatal (no project `providers`/`agent.tiers` is read on a `fab/`-less launch — `resolve.FabRoot()` failure is treated as non-fatal). The launcher also runs the coordinating agent on its **own operator tier** (previously it borrowed the doing tier): it resolves the `operator` tier via `agent.ResolveTier`, reads that tier's provider's `session_command`, and injects the profile via the grammar-forgiving `spawn.WithProfile` — **substituting** into a `{model}`/`{effort}` template session command (the built-in claude default is templated, and a non-Claude worker CLI templates the same way; all-or-nothing, an empty value drops the placeholder token + a preceding `-`-flag), or **appending** `--model`/`--effort` to a plain session command carrying no placeholder (last-wins; empty value ⇒ omit). A provider without a `session_command` falls back to `spawn.DefaultSpawnCommand` (still profile-substituted); an unresolvable `fab/` project degrades to fab-kit's built-in operator tier + built-in claude provider. A `fab/`-less launch therefore composes a fully-defaulted command: default session command + built-in operator-tier `{model, effort}`. This makes the operator the first non-pipeline consumer of the agent-tier system shipped by `260613-l3ja`.
3. **Safety Model** — confirmation tiers (read-only / recoverable / destructive), pre-send validation (pane exists + **three-state agent gate**: only `idle` sends unattended; `active`/`waiting`/unknown require confirmation — mirrors `fab pane send`'s `@rk_agent_state` semantics, ioku), bounded retries & escalation table
4. **Monitoring System** — server-keyed state file (`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`), monitored set (fields incl. `repo` + `session`; enrollment triggers, removal triggers), `branch_map` value is `{ branch, repo }`, `/loop` lifecycle (start/extend/stop, one-loop invariant), monitoring tick with 7 steps (snapshot, auto-nudge, watches, autopilot dispatch, removals, persist, loop lifecycle). **Adaptive cadence** (260615-mmmt; `waiting`-driven since ioku): the heartbeat is not fixed — `3m` normally, tightened to `90s` (§8, overridable) the moment a tick detects **any** monitored agent in the **`waiting`** Agent-column state (the pane's `@rk_agent_state` is `waiting` — blocked on a human; the event-driven primary trigger that replaces the old capture-based menu detection, which stays a fallback for uninstrumented panes), relaxed back to `3m` when none is. The tick's loop-lifecycle step (step 7) adapts the interval; the one-loop invariant is preserved by **re-establishing the single loop** at the new interval (never a second concurrent loop), and autopilot's own `2m` cadence composes unchanged. The §4 Idle Message reflects the currently-active interval (via its existing `--interval {interval}`), so a tightened cadence shows the nearer next-tick time. The tick snapshot uses `fab pane map --all-sessions --json` and groups rows by `repo` then `session`. Since 260611-szxd (f116) the frame spec lives in a dedicated **`Status Frame Format` subsection** after Tick Behavior (tick step 1 ends "emit the status frame — see Status Frame Format"), with the render-path rationale collapsed to one rule (bare markdown — no fence, no headings, no ANSI; channels: tables, emoji, bold, italic, code spans, plain URLs) and the design history deferred to `runtime/operator.md`. The status frame is rendered **markdown-native** (the operator emits it as an assistant message that the agent harness renders as markdown — ANSI escapes and markdown headings do NOT survive that path, so neither is used). It is a header line (`🛰️ **Operator** · {time} · tick #{N} · **{N} tracked**`), one `📂 **{repo-path}** · {session}` anchor + change table per repo, and a `👁️ **Watches**` anchor + table — grouped by repo for scannability. Health is shown with **emoji** (the only surviving color channel): 🟢 active/healthy, 🟡 waiting/idle/new-items, 🔴 stuck/errored, ✅ complete, ⚪ paused. Change-table columns: autopilot `▶` · `ID` (code span) · Health (emoji) · Stage (with `⚠️` trailing on stuck rows) · PR (full `pr_url` as plain text — selectable/copyable in xterm, blank until shipped). Degrades cleanly: strip emoji and the Stage text still names the state; the PR URL is plain text regardless. (Supersedes the earlier non-functional ANSI "structural color" spec — see `runtime/operator.md` → "Status Frame = Markdown Tables + Emoji".) Window-name rename on enrollment: prefix `»` to the tmux window name via `fab pane window-name ensure-prefix` (idempotent; keys on server-global pane IDs, unchanged by the multi-repo model). Removal replaces `»` with `›` via `fab pane window-name replace-prefix`, guarded to skip user-renamed windows.
5. **Auto-Nudge** — question detection (capture -S -20, guards, pattern matching), answer model (decision list items 1-6 with rule 4 Routine/Strategic classification), **non-blocking strategic handling** (260615-mmmt: a Strategic classification never ends the operator's turn — auto-pick-and-notify when a defensible recommendation exists, else leave-open-and-notify; the operator keeps ticking and picks up the async answer on a later tick via the existing re-capture path), **notification send** (one fail-silent shell command, default `rk notify` gated on `command -v rk`, documented fallbacks ntfy.sh / Discord / `PushNotification` / Slack MCP), idle auto-default on left-open strategic prompts, re-capture before send, per-answer logging
6. **Coordination Patterns / Modes of Operation** — shared rhythm + compact table: broadcast, sequenced rebase, merge PRs, spawn agent, status dashboard, unstick agent, notification, autopilot. **Repo-targeted spawning**: each spawn establishes the target repo first, runs `wt create` in that repo's directory, reads the target repo's profile-resolved session command via `fab agent --print --repo <target-repo>` (from `providers.claude.session_command`), and enrolls with `repo` + `session`. **Existence-guarded pointer activation** (260617-5xnx): the spawn sequence's step 3 (between `wt create` and opening the agent tab) runs `fab change switch <change>` **in the new worktree's directory** — guarded on `fab resolve --folder <change>` succeeding (the change folder already exists) and **fail-soft** (log one line and continue on failure) — so an operator-spawned worktree for an existing change sets its **own** `.fab-status.yaml` and is self-describing after the pipeline completes, instead of being left pointer-less. Each operator worktree is a dedicated single-change checkout that owns its own per-worktree pointer, so there is no cross-tab collision; the raw-text/backlog forms skip the switch (the folder doesn't exist yet) and defer activation to `/fab-new` inside the spawned agent. The transient `<change>` override on `/fab-fff` — not the pointer — remains what targets the pipeline (see Resolved Design Decision 12). Since 260611-szxd (f049) the spawn sequence is stated **once** in §6 (now 7 steps with the activation step); the three Working-a-Change forms are a 3-row table mapping entry form → initial command (`/fab-fff <change>`, `/fab-new <shell_escaped_description>`, `/fab-new <id>`) + "run the §6 spawn sequence", and Autopilot / Watches step 4 are one-line §6 references (variant extras preserved: shell-escaping, idea-lookup pre-step, `--reuse`, watch enrollment fields). Since 260612-w7dp the Existing-change initial command is the single parseable `/fab-fff <change>` — the former `/fab-switch <change> && /fab-proceed` chain is gone (the spawn embeds one prompt, where `&&` is not a shell operator and Claude reads one leading `/command`, so the `&& …` tail is absorbed into the first command's argument rather than running as a second command; the change-name override needs no switch anyway). Two slash commands *can* be sequentialized via separate sends, but the operator prefers the synchronous `fab change switch` CLI verb (step 3) over a slash-command switch — a one-line symlink write should not cost an agent round-trip, and a post-spawn send would regress the single-dispatch-at-spawn property (see Resolved Design Decision 12). **Two-tier dependency resolution**: same-repo `depends_on` cherry-picks against the **resolved default branch** after a fetch (260612-g8st: `git fetch origin`, then resolve `{default_branch}` via `git symbolic-ref --short refs/remotes/origin/HEAD` → `gh repo view --json defaultBranchRef` → probed literal fallback: `main` when `origin/main` exists, else `master`, then `git cherry-pick --no-commit origin/{default_branch}..<dep-branch>` — never a hardcoded, unfetched `origin/main`); cross-repo `depends_on` is an ordering-only barrier (wait for stop_stage, no code merge) — with the REQUIRED caveat that a cross-repo dependency gives the dependent agent no code, only logical sequencing. Ancestor-pruning (`git merge-base --is-ancestor`) is scoped to the same-repo subset.
7. **Autopilot** — queue ordering (user-provided / confidence-based / hybrid); implicit chaining is **nearest-same-repo-predecessor** (260612-w7dp — the closest earlier queue entry in the same repo, cherry-picked; the immediately previous entry only as a cross-repo ordering-only fallback, so same-repo stacking survives interleaved cross-repo entries); queue may span repos with mixed dependency semantics (within-repo cherry-pick chaining degrades to cross-repo ordering-only); per-change loop with a **single dispatch point** (260612-w7dp — gate first, then the §6 step-6 spawn embeds the pipeline command; no separate post-spawn send); per-repo PR sequences for ordered merge; CI-failure is **halt-dependents-only** (halt the failing repo's sub-sequence + any repo with a transitive cross-repo `depends_on` into the failed chain; independent repos continue; summary reports halted vs completed and escalates); failure matrix; interruptibility; resumability. Pipeline uses `/fab-fff`
8. **Configuration** — one operator per tmux server (isolation unit; second operator = second `tmux -L` server; no `--name` dimension), loop interval (3m), stuck threshold (15m), **waiting/menu heartbeat (90s)** and **notify channel (`rk`, auto-fallback)** (260615-mmmt), session-scoped. The operator-state-file schema is unchanged by the new settings, and the strategic auto-default threshold stays hardcoded at 30m (no setting added)
9. **Key Properties** — standard properties table, incl. server-keyed XDG state file and multi-repo/multi-session row

---

## Primitives

All tool references are in shared `_` files — operator4 does not duplicate tool tables.

| Primitive | Reference |
|-----------|-----------|
| `fab pane map --all-sessions --json` (per-row `repo` field; nullable per-row `display_state` — the change's stage state, `active|ready|done|failed|pending|skipped`, `null` under the same conditions as `stage` (`failed` reachable since the DisplayStage failed tier shipped with 260612-dkn3)), `fab agent --print --repo`, `fab resolve`, `fab change list`, `fab status`, `fab score`, `fab operator tick-start` (server-keyed state path) | `_cli-fab.md` |
| `wt list`, `wt create` (run in the target repo's directory; branch-selection probe-and-route per wt's 260717-2af2 contract — an existing branch takes `--checkout <branch>`, a missing one the new-branch-only positional), `wt delete`, `tmux` commands, `/loop` | `_cli-external.md` |
| Change folder, branch, worktree naming | `_preamble.md` § Naming Conventions |

The multi-repo primitives (`--all-sessions`, the `repo` JSON field, `fab agent --print --repo`, and the server-keyed state path) are provided by change 1 (`260607-h3jk`); this skill is the policy layer over them.

---

## Monitoring Tick

The snapshot uses `fab pane map --all-sessions --json` and groups rows by `repo` then `session` before computing status. The tick's detection concerns are fully specified inline:

1. Stage advance detection
2. Pipeline completion detection
3. Review failure detection
4. Pane death detection
5. Auto-nudge (`waiting`-state + input-waiting detection + answer model, incl. non-blocking strategic handling per Auto-Nudge below) — no post-intake `/git-branch` send (260612-w7dp: `/fab-new` Step 11 creates/renames the branch inline; `/git-branch` is sent only for a detected branch/change mismatch per §3 pre-send validation)
6. Stuck detection (excludes `waiting`/input-waiting agents)

The loop-lifecycle step then adapts the cadence (260615-mmmt; `waiting`-driven since ioku): tighten the single loop to `90s` if any monitored agent is `waiting` (or was detected menu-waiting this tick), relax to `3m` otherwise — re-establishing the one loop, never a second.

---

## Watches (§7)

Per-tick source polling (Linear/Slack via MCP) with spawn dedup. **Dedup checks `known` plus `completed`**: when a watch-spawned agent reaches its `stop_stage`, the item ID moves from `known` to `completed`, but the source item may still match the watch query — items present in either list are skipped, so completed items are never respawned.

---

## Auto-Nudge

### Question Detection

- **Primary signal (ioku)**: the `waiting` Agent-column state — the pane's `@rk_agent_state` is `waiting` (blocked on a human: permission prompt / menu / elicitation). Event-driven across all instrumented harnesses (Claude/codex/copilot/gemini); a `waiting` pane is capture-scanned and run through the answer model. The per-tick sweep is `waiting`+idle only (idle is the per-tick fallback); the capture-based patterns below **remain applicable** to `active`/unknown (`—`) panes — uninstrumented, or not yet flipped to `waiting` — but those are not swept every tick.
- Capture window: `tmux capture-pane -t <pane> -p -S -20`
- Guards: Claude turn boundary (`>` cursor in last 2 lines), blank capture
- Pattern matching: `?` on last non-empty line <120 chars with comment/log exclusions, plus inherited patterns (Y/n, approval, phrasing) and new patterns (`:` endings, enumerated options, `Press.*key`)
- Bottom-most indicator rule

### Answer Model

Decision list (all auto-answer except undeterminable or strategic):

1. Binary yes/no -> `y`
2. `[Y/n]`/`[y/N]` -> `y`
3. Claude Code permission -> `y`
4. Numbered menu -> classify then act:
   - **Routine** (tool/permission prompts, binary-framed menus, synonymous-option menus) -> `1`
   - **Strategic** (multi-option menus where options represent materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach decisions) -> **non-blocking handling** (below) — a Strategic classification never ends the operator's turn (260615-mmmt)
   - Classification uses LLM judgment over the terminal capture, weighing: option text length, semantic distinctness of options, surrounding agent context, and reversibility of the choice. No hardcoded keyword list. No agent-side sentinel/marker protocol.
   - On classification uncertainty, treat as Strategic. False-negative strategic commits the queue to an unchosen direction; false-positive strategic costs at most a notification (and, if auto-picked, a reversal at PR review).
5. Determinable from context -> send answer
6. Cannot determine keystrokes -> escalate (left open; excluded from auto-pick and from the 30m idle auto-default — sending a guess would emit nonsense into the pane)

### Non-Blocking Strategic Handling (260615-mmmt)

A Strategic classification (rule 4) does not block the loop — the operator handles the prompt out-of-band within the current tick and proceeds to the next monitored change in the **same** tick, so one strategic question on one change no longer freezes the queue. Two branches:

- **Defensible recommendation -> auto-pick-and-notify**: the operator picks its recommended option (LLM judgment over the capture using rule 4's signals), sends it (after the re-capture-before-send guard), fires a notification, and keeps ticking. PR review is the reversal point (§1).
- **No defensible default -> leave open and notify**: the operator leaves the prompt open, fires a notification, and keeps ticking; the 30m idle auto-default is the backstop.

The user answers asynchronously (notification guidance or direct keystrokes into the pane); the operator picks up the resolution on a later tick via the existing re-capture/re-detection — no new pickup mechanism is added.

### Notification Send (260615-mmmt)

A single fail-silent out-of-band shell send. **Default channel `rk notify`** — a run-kit external contract: `rk notify <message> [--title string]` (run-kit Web Push, `rk v2.3.2`; full command reference in `_cli-external.md` § rk (run-kit)). The send is gated on `command -v rk` and runs fail-silent per `_preamble.md` § Run-Kit (rk) Reference (which documents the gate and the fail-silent discipline; the `notify` subcommand itself is documented in `_cli-external.md` § rk): `rk notify "{change}: {summary} ({repo})" --title "Operator: strategic question"` — real background Web Push, fail-silent by contract. When `rk` is absent, fall back to the first available documented alternative (configurable via the §8 `Notify channel` setting): **ntfy.sh** (high-entropy topic REQUIRED — public topics are world-readable, the topic name is the only secret), **Discord webhook**, the **`PushNotification`** harness tool (personal push, not a shared feed), or **Slack MCP** (searchable, but an interactively-authed MCP may be absent in headless/cron runs, so not a headless default). **All sends fail silently** — a send that cannot be delivered (server unreachable, no subscriptions, missing `curl`/tool) logs one line and the loop keeps ticking; it never crashes or stalls. Mirrors the `_preamble.md` § Run-Kit (rk) Reference "fail silently" discipline.

### Idle Auto-Default on Strategic Escalations

Watchdog for a **left-open** Strategic prompt (rule 4's no-defensible-default branch; auto-picked prompts are already resolved). When rule 4 leaves a prompt open as Strategic, the operator runs a per-prompt idle timer. If the prompt stays idle for 30 minutes, the operator auto-answers and logs with a distinct `auto-defaulted` format. The timer runs in the background — it does not block the loop (260615-mmmt).

- **Threshold**: 30 minutes, hardcoded. No operator-state-file field, no per-change override, no environment variable, **no §8 setting** (explicitly unchanged by 260615-mmmt). The §4 operator state file schema is unchanged.
- **Idle clock reset**: timer resets on any terminal-state change in the pane (new content appended by the agent, user keystrokes that alter the prompt display, prompt redraw). The timer watches pane-idle-ness, not escalation-open-ness.
- **Answer selection priority**: (1) if the prompt visibly states a default (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), send that default; (2) otherwise, send `1`.
- **Scope exclusion**: applies ONLY to left-open rule 4 Strategic prompts. Does NOT apply to auto-picked Strategic prompts (already resolved) or to rule 6 ("cannot determine keystrokes") escalations — sending `1` would emit nonsense into the pane. Rule-6 escalations remain open pending user action.
- **Distinct log format**: `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`. This is grep-distinguishable from the normal `auto-answered` line for after-action review.

### Safety

- Re-capture before send eliminates detection-to-send race condition
- No cooldown or retry limit — PR review is the safety net
- Per-answer logging for all auto-answers, auto-picks, left-open strategic prompts, escalations, auto-defaults, and notify-send failures. Five answer-line shapes + a fail-silent notify line (260615-mmmt): `auto-answered` (routine), `auto-picked strategic … · notified`, `strategic … left open · notified. Please respond.`, `can't determine answer … Please respond.` (rule 6), `auto-defaulted after 30m idle …`, and `notify failed ({channel}). Continuing.`

---

## Autopilot

- Pipeline: `/fab-fff` (not `/fab-ff`)
- Gate: confidence score threshold per change type — evaluated **before anything spawns** (260612-w7dp)
- Per-change loop (single dispatch point, 260612-w7dp): gate -> spawn (in target repo) -> resolve deps (same-repo cherry-pick / cross-repo ordering-only) + open tab with the pipeline command **embedded at spawn** (§6 step 6 is the dispatch — no separate send) -> monitor -> record `{ branch, repo }` + collect PR -> spawn next -> report
- **Repo-spanning queue**: a queue may span repos with mixed dependency semantics — within-repo `--base`/`depends_on` chaining cherry-picks, cross-repo chaining degrades to an ordering-only barrier; implicit chaining picks each change's **nearest same-repo predecessor**, falling back to the immediately previous entry across repo boundaries (260612-w7dp)
- **Per-repo ordered merge**: the completion summary annotates each PR with its repo and suggests a per-repo merge order; ordered merge waits for CI on each PR within a repo's sequence, honoring cross-repo barriers across repos
- **CI failure = halt-dependents-only**: halt the failing repo's sub-sequence + any repo with a transitive cross-repo `depends_on` into the failed chain; truly independent repos continue; the summary reports halted vs completed sub-sequences and escalates
- Failure matrix covers: confidence below gate, review fails, rebase conflict, cherry-pick conflict, pane death, stage timeout, total timeout
- Interruptible: stop/skip/pause/resume
- Resumable from `fab pane map --all-sessions` state reconstruction

---

## Key Properties

| Property | Value |
|----------|-------|
| Requires active change? | No |
| Runs preflight? | No |
| Read-only? | No — sends commands to other agents, auto-answers questions |
| Idempotent? | Yes — state is re-derived before every action |
| Advances stage? | No |
| Outputs `Next:` line? | No — ends with ready signal |
| Loads change artifacts? | No — coordination context only |
| Requires tmux? | Yes — hard stop without it |
| Requires a git repo? | No (260613-2sdj) — window cwd is the repo root inside a repo, else `os.Getwd()`; errors only if both fail |
| Requires a `fab/` project? | No (260613-2sdj) — session command is the project's `providers.claude.session_command` when `fab/` is resolvable, else `spawn.DefaultSpawnCommand` (the template `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`); `resolve.FabRoot()` failure is non-fatal (no project `providers`/`agent.tiers` read on a `fab/`-less launch) |
| Coordinating-agent model | Operator tier (260702-tykw; previously borrowed the doing tier) — `agent.ResolveTier` on the `operator` tier → its provider's `session_command` → profile injected via `spawn.WithProfile` (**substitutes** into a `{model}`/`{effort}` template — the built-in claude default is templated — or **appends** `--model`/`--effort` to a plain command carrying no placeholder, last-wins/empty ⇒ omit); a provider without a `session_command` falls back to `spawn.DefaultSpawnCommand` (still profile-substituted), and an unresolvable `fab/` project degrades to the built-in operator tier + built-in claude provider |
| Uses `/loop`? | Yes — adaptive heartbeat (260615-mmmt; `waiting`-driven since ioku): `3m` normally, tightens to `90s` (§8) when any monitored agent is `waiting` (`@rk_agent_state`) or menu-waiting, relaxes back to `3m`; one loop at a time |
| State file | Server-keyed: `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/...`), keyed by the tmux socket path. Binary-derived; old repo-rooted files not migrated |
| Multi-repo / multi-session? | Yes — one operator per tmux server spans all its sessions and repos via the `(session, repo, pane)` addressing tuple |

---

## Resolved Design Decisions

1. **Standalone over inheritance chain.** Reading operator4 previously required mentally merging ~800 lines across 4 files (operator1->2->3->4). The standalone rewrite contains all behavior in ~280 lines by offloading tool references to shared `_` files and explaining constraints concisely.

2. **All-auto-answer over two-tier classification** *(superseded)*. The original standalone rewrite auto-answered everything, leaning on worktree isolation and human PR merge as the safety gate. Superseded: answer-model rule 4 now classifies numbered menus as **Routine** (auto-answer `1`) vs **Strategic** — and Strategic is handled **non-blockingly** (260615-mmmt: auto-pick-and-notify when a defensible recommendation exists, else leave-open-and-notify, with the 30m idle auto-default as the backstop for left-open prompts) rather than parking the loop — see Answer Model and Non-Blocking Strategic Handling above, and Resolved Design Decision 11.

3. **Re-capture before send over single-tick grace period.** Eliminates the race condition between detection and send without adding latency.

4. **`/fab-fff` for autopilot.** The more autonomous pipeline variant, fitting for operator-driven autopilot where human interaction is minimized.

5. **`/git-branch` after new change** *(superseded in 260612-w7dp)*. The operator no longer sends `/git-branch` after intake advancement — `/fab-new` Step 11 creates or renames the branch inline, making the post-intake send a guaranteed no-op. `_cli-external.md` § Operator Spawning Rules documents the inline behavior; `/git-branch` sends remain only for detected branch/change mismatches (§3 pre-send validation item 4).

6. **Isolation unit = tmux server (one operator per server).** Matches the existing server-wide `operator`-window singleton. A fixed global state path was rejected (forces a machine-wide singleton); keying the state file by the tmux socket path lets a second `tmux -L <label>` server host an independent operator. No `--name` dimension — the server boundary is the only isolation knob.

7. **State file keyed by tmux socket path, under XDG.** `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/...`), derived by change 1's binary (`StatePath()`). Rejected: repo-rooted `.fab-operator.yaml` (single-repo only, can't span repos) and a fixed global path (machine-wide singleton). Old repo-rooted files are abandoned in place — no migration (the monitored set is re-derivable from live `»`-prefixed panes).

8. **Cross-repo dependencies = ordering-only.** Same-repo `depends_on` cherry-picks as today; cross-repo `depends_on` is a pure sequencing barrier (wait for stop_stage, no code merge). Cross-repo branches share no common default-branch base, so there is no sound cross-repo cherry-pick — the dependent agent gets no code, only ordering. Rejected: forbidding cross-repo deps (too restrictive) and full cross-repo code merge (unsound, no shared base).

9. **CI-failure scope = halt-dependents-only.** A CI failure halts the failing repo's merge sub-sequence + any repo with a transitive cross-repo `depends_on` into the failed chain; truly independent repos continue. Rejected: halt-all (throttles independent repos) and halt-only-failing-repo (ignores cross-repo ordering barriers). Chosen to maximize independent-repo throughput while respecting cross-repo barriers.

10. **Status frame = repo-section headers.** Changes render grouped under per-repo header lines (noting the session) with indented rows, rather than per-row repo/session columns. Chosen for scannability.

11. **Non-blocking strategic escalation (260615-mmmt).** A Strategic menu no longer parks the operator's turn (the old "escalate to user" froze the single loop, stalling every other monitored change until a human returned). Escalation is now non-blocking: the operator either **auto-picks-and-notifies** (when it has a defensible recommendation — reversible at PR review per §1) or **leaves the prompt open and notifies** (no defensible default — the 30m idle auto-default remains the backstop), then **keeps ticking** and picks up the async answer on a later tick via the existing re-capture path. Paired with an **adaptive heartbeat** (tighten to 90s when any monitored agent is menu-waiting, relax to 3m) that bounds worst-case detection latency without paying that cadence when idle, preserving the one-loop invariant. The notification **send** is abstracted behind one fail-silent shell command (default `rk notify`, gated on `command -v rk`; ntfy.sh / Discord / `PushNotification` / Slack MCP as documented fallbacks) so the channel can evolve without touching the escalation logic. Rejected: parking every strategic menu (the freeze this removes); a second concurrent loop for the watchdog (violates the one-loop invariant); hardcoding ntfy.sh as the default channel (world-readable topic — now a fallback). The strategic auto-default threshold stays hardcoded at 30m (no new setting); run-kit Web Push itself was a separate run-kit change (`rk v2.3.2`, backlog `[xd9r]`), consumed here as the default channel.

12. **Activate the change pointer at spawn for existing changes (260617-5xnx).** *Gap*: an operator-spawned worktree for an *existing* change was left **pointer-less** — its `.fab-status.yaml` was never written. The §6 spawn sequence created the worktree (whose branch already matches the change folder) and embedded `/fab-fff <change>`, relying entirely on the transient `<change>` override to resolve the pipeline; that override deliberately never writes `.fab-status.yaml` (it protects parallel tabs sharing one pointer from clobbering each other). So the pipeline ran correctly, but a human who later `cd`d into the finished worktree and ran a bare `fab`/`/fab-*` got `No active change (multiple changes exist — use /fab-switch)` and had to name the change on every follow-up (the highest-friction case being the natural post-pipeline `/fab-archive`). *Chosen fix*: add an **existence-guarded** `fab change switch <change>` step (§6 step 3, between `wt create` and opening the agent tab) that runs **in the new worktree's directory** only when `fab resolve --folder <change>` succeeds, and is **fail-soft** (log + continue) — the worktree sets its own pointer and becomes self-describing while the override stays the load-bearing resolution mechanism. *No-collision rationale*: each operator worktree is a dedicated, single-change checkout that owns its **own** per-worktree `.fab-status.yaml`, so setting the pointer there carries zero cross-tab collision risk — the parallel-tabs concern the override path guarded against does not arise within one dedicated worktree. *Existence guard*: the raw-text and backlog entry forms reach spawn before the change folder exists, so the guard skips the switch and activation is deferred to `/fab-new` inside the spawned agent, which creates the change and then activates it (activation at fab-new Step 10) — the operator must not (and cannot) switch to a not-yet-existing change. *Rejected*: leave-as-is / rely on the `<change>` arg (zero new writes, but pushes friction onto every human follow-up — the ergonomic papercut this removes). *Rejected — sequentialized slash commands*: a `/fab-switch <change>` slash dispatch before `/fab-fff`, whether as a `&&`-joined string or two separate Enter-terminated sends. The `&&`-joined string does not work at all (the spawn embeds one prompt; `&&` is not a shell operator there and Claude reads one leading `/command`, so the tail is absorbed into the first command's argument). Two separate sends *would* work, but a slash-command switch is a full agent round-trip for a one-line symlink write, and a post-spawn send regresses the single-dispatch-at-spawn property (Decision 5 / w7dp). The synchronous `fab change switch` **CLI verb** does the identical write directly, with no agent turn and no pane-timing fragility — and setting the pointer is operator *coordination* state (which worktree owns which change), which the operator already owns directly (like `wt create` and window renames), not pipeline work to delegate. The fix is therefore a separate spawn-sequence step running the CLI verb, **not** a chained slash command — the `&&`-no-slash-command-chaining rule (260612-w7dp; see the §6 Section-Structure entry and the entry-form table, where it lives alongside the sibling `/git-branch`-removal captured in Decision 5) is unchanged.
