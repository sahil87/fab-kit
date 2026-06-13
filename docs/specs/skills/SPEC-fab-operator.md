# fab-operator

## Summary

Standalone multi-agent coordination layer with proactive monitoring and auto-nudge. Runs in a dedicated tmux pane, observes all running fab agents across every session on its tmux server via `fab pane map --all-sessions`, routes commands via `tmux send-keys`, monitors progress via `/loop`, auto-answers routine agent questions, and drives autopilot queues through the full pipeline.

**Multi-repo / multi-session model.** The operator coordinates agents across **multiple repos and multiple tmux sessions on a single tmux server** — one operator per server (the isolation unit; a second operator means a second `tmux -L <label>` server). Every agent is addressed by the `(session, repo, pane)` tuple: the **pane ID is the primary key** (server-global, stable), with `repo` (the agent's absolute main-worktree root) and `session` (its tmux session name) layered on as dimensions. Every monitored entry, every `branch_map` value (`{ branch, repo }`), and every watch (`target_repo`) is repo-qualified. State lives in **one server-keyed file** — `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/...`), keyed by the tmux socket path — not a per-repo `.fab-operator.yaml`. This model consumes the Go primitives shipped by change 1 (`260607-h3jk`): `fab pane map --all-sessions --json` with a per-row `repo` field, `fab spawn-command --repo`, and the binary-derived server-keyed state path. Old repo-rooted state files are not migrated.

Self-contained — does not inherit from any other operator skill. All behavior is defined in `src/kit/skills/fab-operator.md` plus the standard `_` files loaded via `_preamble.md`. External tool reference (`_cli-external.md`) is loaded in the operator's own startup section.

Not a lifecycle enforcer — the operator coordinates across agents and proxies routine user input, not advancing stages or making pipeline decisions.

**Helpers**: Declares `helpers: [_cli-fab, _cli-external]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

---

## Section Structure

The skill is organized into 9 sections:

1. **Principles** — identity (coordinate don't execute), **multi-repo aware** (`(session, repo, pane)` addressing; pane ID is the server-global primary key; repo + session are added dimensions), routing discipline, context discipline (never loads change artifacts), state re-derivation via `fab pane map --all-sessions` (why: stale state = wrong actions)
2. **Startup** — trimmed context layer (config/constitution/context only — a `_preamble.md` §1 exception since 260611-zc9m), `_cli-external.md` load, orientation (`fab pane map --all-sessions` + ready signal), reads the server-keyed state file, tmux gate (hard stop when `$TMUX` is unset). **Launch requires neither a git repo nor a resolvable `fab/` project** (260613-2sdj): (a) window cwd — `fab operator` opens its tmux window in the repo root when launched inside a repo, else `os.Getwd()` (the neutral parent dir of the cross-repo singleton), erroring only if both resolutions fail — the old `cannot determine repo root` hard-fail is gone; (b) spawn command — read from the project's `agent.spawn_command` when a `fab/` project is resolvable, else the built-in `spawn.DefaultSpawnCommand` (`claude --dangerously-skip-permissions`), so a missing `fab/` project is non-fatal (no project `agent.spawn_command`/`agent.tiers` is read on a `fab/`-less launch — `resolve.FabRoot()` failure is treated as non-fatal). The launcher also runs the coordinating agent on the **doing tier**: it resolves `fab resolve-agent apply` (the canonical doing-tier stage in the fixed stage→tier mapping), appends `--model`/`--effort` to the spawn command (last-wins; empty value ⇒ omit), and falls back to the built-in doing default `{claude-opus-4-8, high}` on any failure (installed `fab` lacking `resolve-agent`, no resolvable fab project, or unparseable output). A `fab/`-less launch therefore composes a fully-defaulted command: default spawn command + doing default `{model, effort}`. This makes the operator the first non-pipeline consumer of the agent-tier system shipped by `260613-l3ja`.
3. **Safety Model** — confirmation tiers (read-only / recoverable / destructive), pre-send validation (pane exists + agent idle), bounded retries & escalation table
4. **Monitoring System** — server-keyed state file (`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`), monitored set (fields incl. `repo` + `session`; enrollment triggers, removal triggers), `branch_map` value is `{ branch, repo }`, `/loop` lifecycle (start/extend/stop, one-loop invariant), monitoring tick with 7 steps (snapshot, auto-nudge, watches, autopilot dispatch, removals, persist, loop lifecycle). The tick snapshot uses `fab pane map --all-sessions --json` and groups rows by `repo` then `session`. Since 260611-szxd (f116) the frame spec lives in a dedicated **`Status Frame Format` subsection** after Tick Behavior (tick step 1 ends "emit the status frame — see Status Frame Format"), with the render-path rationale collapsed to one rule (bare markdown — no fence, no headings, no ANSI; channels: tables, emoji, bold, italic, code spans, plain URLs) and the design history deferred to `runtime/operator.md`. The status frame is rendered **markdown-native** (the operator emits it as an assistant message that the agent harness renders as markdown — ANSI escapes and markdown headings do NOT survive that path, so neither is used). It is a header line (`🛰️ **Operator** · {time} · tick #{N} · **{N} tracked**`), one `📂 **{repo-path}** · {session}` anchor + change table per repo, and a `👁️ **Watches**` anchor + table — grouped by repo for scannability. Health is shown with **emoji** (the only surviving color channel): 🟢 active/healthy, 🟡 idle/new-items, 🔴 stuck/errored, ✅ complete, ⚪ paused. Change-table columns: autopilot `▶` · `ID` (code span) · Health (emoji) · Stage (with `⚠️` trailing on stuck rows) · PR (full `pr_url` as plain text — selectable/copyable in xterm, blank until shipped). Degrades cleanly: strip emoji and the Stage text still names the state; the PR URL is plain text regardless. (Supersedes the earlier non-functional ANSI "structural color" spec — see `runtime/operator.md` → "Status Frame = Markdown Tables + Emoji".) Window-name rename on enrollment: prefix `»` to the tmux window name via `fab pane window-name ensure-prefix` (idempotent; keys on server-global pane IDs, unchanged by the multi-repo model). Removal replaces `»` with `›` via `fab pane window-name replace-prefix`, guarded to skip user-renamed windows.
5. **Auto-Nudge** — question detection (capture -S -20, guards, pattern matching), answer model (decision list items 1-6 with rule 4 Routine/Strategic classification), idle auto-default on strategic escalations, re-capture before send, per-answer logging
6. **Coordination Patterns / Modes of Operation** — shared rhythm + compact table: broadcast, sequenced rebase, merge PRs, spawn agent, status dashboard, unstick agent, notification, autopilot. **Repo-targeted spawning**: each spawn establishes the target repo first, runs `wt create` in that repo's directory, reads the target repo's `agent.spawn_command` via `fab spawn-command --repo <target-repo>`, and enrolls with `repo` + `session`. Since 260611-szxd (f049) the 6-step spawn sequence is stated **once** in §6; the three Working-a-Change forms are a 3-row table mapping entry form → initial command (`/fab-fff <change>`, `/fab-new <shell_escaped_description>`, `/fab-new <id>`) + "run the §6 spawn sequence", and Autopilot / Watches step 4 are one-line §6 references (variant extras preserved: shell-escaping, idea-lookup pre-step, `--reuse`, watch enrollment fields). Since 260612-w7dp the Existing-change initial command is the single parseable `/fab-fff <change>` — the former `/fab-switch <change> && /fab-proceed` chain is gone (`&&` has no slash-command chaining semantics; the change-name override needs no switch). **Two-tier dependency resolution**: same-repo `depends_on` cherry-picks against the **resolved default branch** after a fetch (260612-g8st: `git fetch origin`, then resolve `{default_branch}` via `git symbolic-ref --short refs/remotes/origin/HEAD` → `gh repo view --json defaultBranchRef` → probed literal fallback: `main` when `origin/main` exists, else `master`, then `git cherry-pick --no-commit origin/{default_branch}..<dep-branch>` — never a hardcoded, unfetched `origin/main`); cross-repo `depends_on` is an ordering-only barrier (wait for stop_stage, no code merge) — with the REQUIRED caveat that a cross-repo dependency gives the dependent agent no code, only logical sequencing. Ancestor-pruning (`git merge-base --is-ancestor`) is scoped to the same-repo subset.
7. **Autopilot** — queue ordering (user-provided / confidence-based / hybrid); implicit chaining is **nearest-same-repo-predecessor** (260612-w7dp — the closest earlier queue entry in the same repo, cherry-picked; the immediately previous entry only as a cross-repo ordering-only fallback, so same-repo stacking survives interleaved cross-repo entries); queue may span repos with mixed dependency semantics (within-repo cherry-pick chaining degrades to cross-repo ordering-only); per-change loop with a **single dispatch point** (260612-w7dp — gate first, then the §6 step-5 spawn embeds the pipeline command; no separate post-spawn send); per-repo PR sequences for ordered merge; CI-failure is **halt-dependents-only** (halt the failing repo's sub-sequence + any repo with a transitive cross-repo `depends_on` into the failed chain; independent repos continue; summary reports halted vs completed and escalates); failure matrix; interruptibility; resumability. Pipeline uses `/fab-fff`
8. **Configuration** — one operator per tmux server (isolation unit; second operator = second `tmux -L` server; no `--name` dimension), loop interval (3m), stuck threshold (15m), session-scoped
9. **Key Properties** — standard properties table, incl. server-keyed XDG state file and multi-repo/multi-session row

---

## Primitives

All tool references are in shared `_` files — operator4 does not duplicate tool tables.

| Primitive | Reference |
|-----------|-----------|
| `fab pane map --all-sessions --json` (per-row `repo` field; nullable per-row `display_state` — the change's stage state, `active|ready|done|failed|pending|skipped`, `null` under the same conditions as `stage` (`failed` reachable since the DisplayStage failed tier shipped with 260612-dkn3)), `fab spawn-command --repo`, `fab resolve`, `fab change list`, `fab status`, `fab score`, `fab operator tick-start` (server-keyed state path) | `_cli-fab.md` |
| `wt list`, `wt create` (run in the target repo's directory), `wt delete`, `tmux` commands, `/loop` | `_cli-external.md` |
| Change folder, branch, worktree naming | `_preamble.md` § Naming Conventions |

The multi-repo primitives (`--all-sessions`, the `repo` JSON field, `fab spawn-command --repo`, and the server-keyed state path) are provided by change 1 (`260607-h3jk`); this skill is the policy layer over them.

---

## Monitoring Tick

The snapshot uses `fab pane map --all-sessions --json` and groups rows by `repo` then `session` before computing status. The tick's detection concerns are fully specified inline:

1. Stage advance detection
2. Pipeline completion detection
3. Review failure detection
4. Pane death detection
5. Auto-nudge (input-waiting detection + answer model) — no post-intake `/git-branch` send (260612-w7dp: `/fab-new` Step 11 creates/renames the branch inline; `/git-branch` is sent only for a detected branch/change mismatch per §3 pre-send validation)
6. Stuck detection (excludes input-waiting agents)

---

## Watches (§7)

Per-tick source polling (Linear/Slack via MCP) with spawn dedup. **Dedup checks `known` plus `completed`**: when a watch-spawned agent reaches its `stop_stage`, the item ID moves from `known` to `completed`, but the source item may still match the watch query — items present in either list are skipped, so completed items are never respawned.

---

## Auto-Nudge

### Question Detection

- Capture window: `tmux capture-pane -t <pane> -p -S -20`
- Guards: Claude turn boundary (`>` cursor in last 2 lines), blank capture, idle-only
- Pattern matching: `?` on last non-empty line <120 chars with comment/log exclusions, plus inherited patterns (Y/n, approval, phrasing) and new patterns (`:` endings, enumerated options, `Press.*key`)
- Bottom-most indicator rule

### Answer Model

Decision list (all auto-answer except undeterminable or strategic):

1. Binary yes/no -> `y`
2. `[Y/n]`/`[y/N]` -> `y`
3. Claude Code permission -> `y`
4. Numbered menu -> classify then act:
   - **Routine** (tool/permission prompts, binary-framed menus, synonymous-option menus) -> `1`
   - **Strategic** (multi-option menus where options represent materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach decisions) -> escalate to user
   - Classification uses LLM judgment over the terminal capture, weighing: option text length, semantic distinctness of options, surrounding agent context, and reversibility of the choice. No hardcoded keyword list. No agent-side sentinel/marker protocol.
   - On classification uncertainty, treat as Strategic and escalate. False-negative strategic commits the queue to an unchosen direction; false-positive strategic costs at most a user nudge, recovered by the 30m idle auto-default below.
5. Determinable from context -> send answer
6. Cannot determine keystrokes -> escalate

### Idle Auto-Default on Strategic Escalations

When rule 4 escalates as Strategic, the operator runs a per-prompt idle timer. If the prompt stays idle for 30 minutes, the operator auto-answers and logs with a distinct `auto-defaulted` format.

- **Threshold**: 30 minutes, hardcoded. No operator-state-file field, no per-change override, no environment variable. The §4 operator state file schema is unchanged.
- **Idle clock reset**: timer resets on any terminal-state change in the pane (new content appended by the agent, user keystrokes that alter the prompt display, prompt redraw). The timer watches pane-idle-ness, not escalation-open-ness.
- **Answer selection priority**: (1) if the prompt visibly states a default (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), send that default; (2) otherwise, send `1`.
- **Scope exclusion**: applies ONLY to rule 4 Strategic escalations. Rule 6 ("cannot determine keystrokes") escalations MUST NOT trigger idle auto-default — sending `1` would emit nonsense into the pane. Rule-6 escalations remain open pending user action.
- **Distinct log format**: `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`. This is grep-distinguishable from the normal `auto-answered` line for after-action review.

### Safety

- Re-capture before send eliminates detection-to-send race condition
- No cooldown or retry limit — PR review is the safety net
- Per-answer logging for all auto-answers, escalations, and auto-defaults

---

## Autopilot

- Pipeline: `/fab-fff` (not `/fab-ff`)
- Gate: confidence score threshold per change type — evaluated **before anything spawns** (260612-w7dp)
- Per-change loop (single dispatch point, 260612-w7dp): gate -> spawn (in target repo) -> resolve deps (same-repo cherry-pick / cross-repo ordering-only) + open tab with the pipeline command **embedded at spawn** (§6 step 5 is the dispatch — no separate send) -> monitor -> record `{ branch, repo }` + collect PR -> spawn next -> report
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
| Requires a `fab/` project? | No (260613-2sdj) — spawn command is the project's `agent.spawn_command` when `fab/` is resolvable, else `spawn.DefaultSpawnCommand` (`claude --dangerously-skip-permissions`); `resolve.FabRoot()` failure is non-fatal (no project `agent.spawn_command`/`agent.tiers` read on a `fab/`-less launch) |
| Coordinating-agent model | Doing tier (260613-2sdj) — `fab resolve-agent apply` → `--model`/`--effort` appended (last-wins, empty ⇒ omit); built-in fallback `{claude-opus-4-8, high}` on any failure (incl. no resolvable `fab/` project) |
| Uses `/loop`? | Yes — 3m heartbeat |
| State file | Server-keyed: `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/...`), keyed by the tmux socket path. Binary-derived; old repo-rooted files not migrated |
| Multi-repo / multi-session? | Yes — one operator per tmux server spans all its sessions and repos via the `(session, repo, pane)` addressing tuple |

---

## Resolved Design Decisions

1. **Standalone over inheritance chain.** Reading operator4 previously required mentally merging ~800 lines across 4 files (operator1->2->3->4). The standalone rewrite contains all behavior in ~280 lines by offloading tool references to shared `_` files and explaining constraints concisely.

2. **All-auto-answer over two-tier classification** *(superseded)*. The original standalone rewrite auto-answered everything, leaning on worktree isolation and human PR merge as the safety gate. Superseded: answer-model rule 4 now classifies numbered menus as **Routine** (auto-answer `1`) vs **Strategic** (escalate to user, with a 30m idle auto-default) — see Answer Model above.

3. **Re-capture before send over single-tick grace period.** Eliminates the race condition between detection and send without adding latency.

4. **`/fab-fff` for autopilot.** The more autonomous pipeline variant, fitting for operator-driven autopilot where human interaction is minimized.

5. **`/git-branch` after new change** *(superseded in 260612-w7dp)*. The operator no longer sends `/git-branch` after intake advancement — `/fab-new` Step 11 creates or renames the branch inline, making the post-intake send a guaranteed no-op. `_cli-external.md` § Operator Spawning Rules documents the inline behavior; `/git-branch` sends remain only for detected branch/change mismatches (§3 pre-send validation item 4).

6. **Isolation unit = tmux server (one operator per server).** Matches the existing server-wide `operator`-window singleton. A fixed global state path was rejected (forces a machine-wide singleton); keying the state file by the tmux socket path lets a second `tmux -L <label>` server host an independent operator. No `--name` dimension — the server boundary is the only isolation knob.

7. **State file keyed by tmux socket path, under XDG.** `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/...`), derived by change 1's binary (`StatePath()`). Rejected: repo-rooted `.fab-operator.yaml` (single-repo only, can't span repos) and a fixed global path (machine-wide singleton). Old repo-rooted files are abandoned in place — no migration (the monitored set is re-derivable from live `»`-prefixed panes).

8. **Cross-repo dependencies = ordering-only.** Same-repo `depends_on` cherry-picks as today; cross-repo `depends_on` is a pure sequencing barrier (wait for stop_stage, no code merge). Cross-repo branches share no common default-branch base, so there is no sound cross-repo cherry-pick — the dependent agent gets no code, only ordering. Rejected: forbidding cross-repo deps (too restrictive) and full cross-repo code merge (unsound, no shared base).

9. **CI-failure scope = halt-dependents-only.** A CI failure halts the failing repo's merge sub-sequence + any repo with a transitive cross-repo `depends_on` into the failed chain; truly independent repos continue. Rejected: halt-all (throttles independent repos) and halt-only-failing-repo (ignores cross-repo ordering barriers). Chosen to maximize independent-repo throughput while respecting cross-repo barriers.

10. **Status frame = repo-section headers.** Changes render grouped under per-repo header lines (noting the session) with indented rows, rather than per-row repo/session columns. Chosen for scannability.
