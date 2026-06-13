---
description: "Operator coordination skill (`/fab-operator`, superseding the historical operator4) — multi-repo / multi-session coordination on one tmux server via the `(session, repo, pane)` addressing tuple, the server-keyed operator state file (term defined once in skill §4), 3-file context loading (§1 exception, zc9m), multi-agent monitoring, auto-answer model with strategic escalation, repo-targeted spawning (spawning rules homed in `_cli-external.md` wt section; spawn sequence stated once in skill §6 with a 3-row entry-form table, szxd), two-tier dependency resolution (fetch-first, resolved-default-branch cherry-pick/rebase base — g8st), repo-spanning autopilot with a single spawn-embedded dispatch point (gate before the tab opens), nearest-same-repo-predecessor queue chaining, and the `/fab-fff <change>` entry-form command (w7dp), the markdown status frame (skill §4 Status Frame Format subsection, szxd), and tmux tab-naming, the git-optional / `fab/`-optional launcher (window cwd = git root else `os.Getwd()`; spawn command = project config else `spawn.DefaultSpawnCommand`) running its coordinating agent on the doing tier (`fab resolve-agent apply` + `spawn.WithProfile`, built-in `{claude-opus-4-8, high}` fallback — first non-pipeline consumer of the l3ja agent-tier system, 2sdj). Historical operator4 context plus the operator design-decision lineage."
---
# Operator

**Domain**: runtime

## Overview

The operator is a standalone, long-lived coordination skill (`/fab-operator`) — NOT a pipeline stage. It runs in a dedicated tmux pane, coordinates agents **across multiple repos and multiple tmux sessions on a single tmux server**, observes them via `fab pane map --all-sessions`, routes commands via `tmux send-keys`, monitors progress via `/loop`, and auto-answers idle agent prompts (escalating strategic ones to the user). Every agent is addressed by the `(session, repo, pane)` tuple: the **pane ID is the primary key** (server-global, stable), with `repo` (the agent's absolute main-worktree root) and `session` (its tmux session name) as added dimensions. There is **one operator per tmux server**, owning **one server-keyed state file** that spans every repo it coordinates. This file documents the operator's behavior, its tick lifecycle and state model, and the design-decision lineage from the historical `/fab-operator4` through the current multi-repo `/fab-operator`. For the `.fab-runtime.yaml` agent schema and `fab pane` primitives the operator builds on, see [runtime-agents.md](runtime-agents.md) and [pane-commands.md](pane-commands.md). For the Go-side mechanism (server-keyed state path derivation, the `repo` JSON field, `fab spawn-command --repo`), see [kit-architecture.md](../distribution/kit-architecture.md) → "Operator State File" and [pane-commands.md](pane-commands.md).

## Requirements

### `/fab-operator4` (Standalone Coordination Skill) — *superseded by `/fab-operator`*

> **Note**: Operator4 has been superseded by `/fab-operator` (v7). The skill file and launcher script have been removed. This section is preserved as historical context for the design decisions that evolved into the current operator. See `fab-operator.md` for authoritative behavior.

`/fab-operator4` was a standalone, self-contained coordination skill — NOT a pipeline stage. It ran as a long-lived Claude session in a dedicated tmux pane, observing agents via `fab pane-map`, routing commands via `tmux send-keys`, monitoring progress via `/loop`, and auto-answering idle agent prompts.

Operator4 was the first standalone operator skill. Previous iterations (operator1, operator2, operator3) were removed — their behavior was fully inlined into operator4 as a standalone file.

#### Principles

**Coordinate, don't execute.** The operator routes user instructions to the right agent — it never implements work directly. If the target is ambiguous, ask.

**Not a lifecycle enforcer.** Individual agents self-govern via their own pipeline skills. The operator does not validate stage transitions or enforce pipeline rules.

**Context discipline.** The operator never reads change artifacts (intakes, specs, tasks). Its context window is reserved for coordination state — pane maps, stage snapshots, monitoring state.

**State re-derivation.** Before every action, re-query live state via `fab pane-map` (or `wt list` + `fab change list` outside tmux). Panes die, stages advance, agents finish — stale state leads to wrong actions.

#### Context Loading

As of 260611-zc9m the operator loads only **three** project files: `fab/project/config.yaml`, `fab/project/constitution.md`, and `fab/project/context.md` (optional — skip gracefully if missing). It is a listed exception to the `_preamble.md` §1 always-load layer — `code-quality.md`, `code-review.md`, and both doc indexes serve artifact generation and review, which the operator never does (per its own §1 Context discipline principle), and a long-lived session re-pays every loaded file after each `/clear`. This was a deliberate, verifier-endorsed behavior change (loading-only — no principle, safety-model, or spawn-procedure text changed); before zc9m the operator loaded the full 7-file layer. Helpers declared in frontmatter: `_cli-fab` (fab command reference) and `_cli-external` (external tool reference for `wt`, `idea`, `tmux`, and `/loop` — loaded only by operator, not by pipeline skills). It does NOT run preflight. It does NOT load change-specific artifacts.

#### Orientation

On invocation, runs `fab pane-map` and displays the output, then signals readiness. Outside tmux (`$TMUX` unset), falls back to `wt list` + `fab change list` for status queries only — monitoring is disabled.

#### Safety Model

| Tier | Examples | Behavior |
|------|----------|----------|
| Read-only | Status check, pane map | No confirmation |
| Recoverable | Send `/fab-continue`, rebase | Announce before sending |
| Destructive | Merge PR, archive, delete worktree, autopilot | Confirm before executing |

**Pre-send validation**: Before sending keys to any pane, the operator MUST (1) verify the pane exists via refreshed pane map (dead panes fail silently), (2) check the agent is idle via the Agent column. If busy, warn and require explicit confirmation.

**Bounded retries**: Every automatic action has a bounded retry count. Unbounded retries compound errors.

| Situation | Max retries | Escalation |
|-----------|-------------|------------|
| Stuck agent nudge | 1 | "Appears stuck at {stage}. Manual investigation recommended." |
| Rebase conflict | 0 | Immediately flag to user |
| Pane death (non-autopilot) | 0 | Report pane gone. No respawn outside autopilot |
| Send to busy agent | 0 | Warn user, require explicit confirmation |

#### Monitoring System (historical operator4 framing — see "Multi-Repo Monitoring Model" below for the current model)

Operator4 maintained a monitored set persisted to a repo-rooted `.fab-operator.yaml` (mirrored in conversation context for the active tick), each entry tracking: change ID, pane, last-known stage, last-known agent state, enrolled-at timestamp, last-transition-at timestamp. The enrollment/removal mechanics (window-name prefix on enroll, done-marker swap on removal) below are unchanged by the multi-repo reframe — only the addressing model and state-file location moved (see the next section).

**Enrollment triggers**: operator sends a command to it, user requests monitoring, operator triggers an automatic action toward it (including autopilot and watch spawns). Read-only actions do not enroll. **Spawned agents are always auto-enrolled** — the operator MUST NOT ask the user whether to monitor a spawned agent. This constraint is reinforced in both the §1 principles and the "Spawning an Agent" procedural subsection to ensure proximity-based LLM adherence.

**Enrollment also applies the `»` window-name prefix**: after writing the monitored entry to the state file, the operator invokes `fab pane window-name ensure-prefix <pane> »` (U+00BB). The primitive enforces the literal-prefix idempotent guard internally: operator-spawned windows (already `»<wt>` from the spawn step), `/clear`-restored entries, and re-enrollment after transient removal all no-op through the guard. A non-zero exit — pane vanished between refresh and rename (exit 2) or any other tmux error (exit 3, including tmux not running / socket unreachable) — is logged as `"{change}: window rename skipped ({error})."` and does not roll back the enrollment. Window markers (`»` / `›`) are **unchanged by the multi-repo model** — they key on server-global pane IDs, which are unique across every repo and session on the server.

**Removal triggers**: change reaches a terminal stage (hydrate, ship, review-pr), pane dies, user explicitly stops monitoring. On every removal path the operator invokes `fab pane window-name replace-prefix <pane> » ›` (U+203A, single right guillemet — the done-marker), swapping the active-monitoring `»` prefix for the trail-preserved `›`. The primitive's literal-prefix guard protects user-renamed windows: if the user renamed the window mid-monitoring so it no longer starts with `»`, the swap silently no-ops without clobbering the user's name. Exit 2 (pane missing — the window is gone anyway) is treated as a successful removal; other non-zero exits log `"{change}: window rename skipped ({error})."`. This keeps the tab bar an accurate at-a-glance map: `»` for currently-tracked, `›` for operator-touched-but-done, and untouched names for windows the operator never marked.

#### Multi-Repo Monitoring Model (current `/fab-operator`)

> Introduced by `260607-oy0k-operator-multi-repo-skill`; consumes the Go primitives shipped by `260607-h3jk`. This is the authoritative monitored-set model; the operator4 framing above is historical context.

**One operator per tmux server.** The isolation unit is the tmux server — a single operator spans every session and every repo on that server. A second operator means a second tmux server (`tmux -L <label>`). "Multiple sessions, same server" share one operator and one state file. There is no `--name` dimension; the server boundary is the only isolation knob. This matches the server-wide singleton already enforced by the `operator` window (`fab operator` switches to the existing window rather than creating a second one).

**Server-keyed state file.** The monitored set, autopilot queue, `branch_map`, and watches persist in **one server-keyed file** — `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/fab/operator/<server-slug>.yaml`), keyed by the tmux socket path so the one operator-per-server owns one file across all the repos it coordinates. The binary derives this path (`fab operator tick-start` reads/writes it via `StatePath()`); the operator never computes it. The skill defines the term **operator state file** once, in §4 (under the `### Operator State File` heading), and refers to the live file by that term everywhere else (uliv — the ~9 stale live-file `.fab-operator.yaml` mentions were replaced; `.fab-operator.yaml` now appears only in the deliberate legacy not-read/not-migrated context). See [kit-architecture.md](../distribution/kit-architecture.md) → "Operator State File" for the derivation mechanism. Old repo-rooted `.fab-operator.yaml` files from before the server-keyed model are **not migrated** — they are abandoned in place (the monitored set is re-derivable from live `»`-prefixed panes).

**`(session, repo, pane)` addressing.** Every monitored agent, `branch_map` value, and watch is repo-qualified. The pane ID remains the primary key (server-global, stable); `repo` (absolute main-worktree root) and `session` (tmux session name) are added dimensions, not replacements. Schema additions over the operator4 entry:

- Each `monitored` entry gains `repo` (absolute main-worktree root) and `session` (tmux session name).
- The `branch_map` value becomes `{ branch, repo }` (was a bare branch string). The `repo` is required to disambiguate a dependency's branch across repos and to choose same-repo (cherry-pick) vs. cross-repo (ordering-only) dependency resolution.
- Each `watches` entry gains `target_repo` — the repo a watch's spawned changes land in. A watch with no `target_repo` cannot spawn.

#### Multi-Repo Coordination: Spawning, Dependencies, Autopilot (current `/fab-operator`)

> Introduced by `260607-oy0k-operator-multi-repo-skill`. Supersedes the single-repo spawning/autopilot prose in the historical operator4 sections below.

**Spawning-rules home.** The **Operator Spawning Rules** — the known-change vs new-change worktree/branch naming strategy (known change: pass the change folder name as the branch argument to `wt create`; new change from backlog: `wt create` on the default branch, then `/fab-new` in the agent — fab-new Step 11 renames the worktree's disposable branch inline, so the operator does NOT send a post-intake `/git-branch`; w7dp removed that stale step from `_cli-external.md` and the skill's lockstep sites) — live in `_cli-external.md`'s `wt` section as of 260611-zc9m (moved out of the always-loaded `_preamble.md` § Naming Conventions; only `fab-operator` declares `_cli-external`, so only it pays). `fab-operator.md` §6 remains the normative step-by-step spawn procedure. The `fab spawn-command --repo <target-repo>` / "never the operator's own config.yaml" repo-targeting rule appears exactly once in `_cli-external.md` (the duplicate at its tmux `new-window` bullet was dropped).

**Repo-targeted spawning.** Every spawn flow first establishes **which repo** the work targets (an existing change's `repo`, a watch's `target_repo`, or the repo the user names — defaulting to the operator's launch repo), then runs each step against that repo, not the operator's own:

1. Run `wt create --non-interactive` **with the target repo as the working directory**, so the worktree lands under `$(dirname <target-repo>)/<repo-name>.worktrees/` rather than the operator's repo.
2. Read **that repo's** `agent.spawn_command` via `fab spawn-command --repo <target-repo>` (see [kit-architecture.md](../distribution/kit-architecture.md)) — never the operator's own `config.yaml`, since each repo may configure a different spawn command.
3. Open the agent tab and enroll with `repo` and `session` recorded, plus `{ branch, repo }` added to `branch_map`.

All three work paths (existing change, raw text, backlog/Linear) and watch-driven spawns use this same repo-targeted sequence. Since szxd the skill states the spawn sequence **once**, in §6 "Spawning an Agent": the three Working-a-Change walkthroughs that restated it are replaced by a 3-row table mapping entry form → initial command (`/fab-fff <change>` for an existing change — a single parseable command since w7dp: `&&`-chained slash commands have no chaining semantics, and the change-name override makes a `/fab-switch` pre-step unnecessary — `/fab-new <shell_escaped_description>` for raw text, `/fab-new <id>` for backlog/Linear) + "run the §6 spawn sequence", and Autopilot steps 1–2 and Watches step 4 are one-line §6 references. Variant-specific extras are preserved: the shell-escaping requirement for raw text, the idea-lookup pre-step for backlog/Linear, `--reuse` for autopilot respawns, and the watch-enrollment extras (`stop_stage`/`spawned_by`). (One small residual remains and is tracked as a szxd plan deletion candidate: the Working-a-Change preamble's parenthetical arrow-restatement of the §6 sequence.)

**Two-tier dependency resolution.** Each `depends_on` entry is classified by comparing the dependency's `repo` (from its `branch_map` `{ branch, repo }` pair, or its monitored entry) against this change's `repo`:

- **Same-repo dependency** → **cherry-pick**: fetch + resolve the base (step 0 below), then `git cherry-pick --no-commit origin/{default_branch}..<dep-branch>` into the worktree.
- **Cross-repo dependency** → **ordering-only barrier**: wait until the dependency reaches its `stop_stage` (terminal stage when `stop_stage` is null), then spawn. **No code is merged.**

> **REQUIRED caveat — cross-repo deps give the dependent agent NO code.** A cross-repo `depends_on` is a pure *sequencing* constraint; the dependent worktree receives nothing from the dependency. This is correct only for **logical** dependencies ("don't start the frontend change until the API change merges"), never for **code-level** ones. Cross-repo branches share no common default-branch base to cherry-pick across, so there is no sound way to make the dependency's code available. For code sharing across repos, the dependency must merge and be consumed as a normal upstream artifact (package, vendored copy), outside the operator's scope.

**Step-0 fetch + default-branch resolution (g8st).** Same-repo resolution begins, in the target worktree, by refreshing the remote and resolving the repo's **actual** default branch — never a hardcoded `origin/main`:

```bash
git fetch origin
default_branch=$(git symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|^origin/||')
[ -n "$default_branch" ] || default_branch=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name 2>/dev/null)
# Literal fallback when both commands fail: probe the just-fetched refs — main when origin/main exists, else master
[ -n "$default_branch" ] || default_branch=$(git rev-parse --verify -q origin/main >/dev/null && echo main || echo master)
```

`origin/{default_branch}` is the cherry-pick base (and the `--merge-on-complete` rebase target — see Autopilot below). Fetching first prevents a stale base even on correctly-defaulted repos; resolving the name makes autopilot usable on repos whose default branch isn't `main`. (Previously all three sites — the cherry-pick range, the "Why `origin/main` as base" rationale block, and the `--merge-on-complete` rebase — hardcoded `origin/main` with no fetch step, making autopilot unusable on non-main-default repos and the base stale everywhere else.) The resolution chain matches `/git-pr`'s (see [change-lifecycle.md](../pipeline/change-lifecycle.md) § Git Integration); the operator's literal fallback probes the just-fetched refs rather than assuming, because the fetch has already run.

Ancestor-pruning (`git merge-base --is-ancestor`) is scoped to the **same-repo subset** of the dependency set — it is meaningless across repos with no shared history. The `origin/{default_branch}..<dep-branch>` transitive-closure argument (only direct/leaf deps need cherry-picking) holds only within a repo; cross-repo deps carry no such transitive content.

**Repo-spanning autopilot.** An autopilot queue **may span repos** with mixed dependency semantics: implicit `--base`/`depends_on` chaining cherry-picks **within** a repo and **degrades to an ordering-only barrier across** repo boundaries. Ordered merge tracks **per-repo PR sequences** — within each repo, base-first in dependency order; across repos, cross-repo barriers are honored (a cross-repo dependent's PR merges only after its barrier dependency reaches its target repo's main). The queue-completion summary annotates each PR with its repo and suggests a per-repo merge order.

**CI-failure = halt-dependents-only.** During ordered merge, a CI failure halts the failing repo's merge sub-sequence AND any repo whose queued items carry a cross-repo `depends_on` into the failed chain — **transitively** over the cross-repo `depends_on` graph (a repo halts if any of its queued items depends, directly or via another already-halted item, on a PR in the failed chain). **Truly independent repos' sub-sequences continue merging.** The operator isolates the blast radius to the failure's dependency cone, reports which sub-sequences halted vs. completed, and escalates the failure to the user.

**`/loop` lifecycle**: Start when first change enrolled (no loop running) — `/loop 3m "operator tick"`. Stop when monitored set empty. One-loop invariant: at most one active `/loop` at any time.

**Tick snapshot is server-wide.** The tick's snapshot step uses `fab pane map --all-sessions --json` (not bare `fab pane map`), so the operator sees agents in **every** session on its server, not just its own. `--json` exposes the per-row `repo` field (the agent's absolute main-worktree root, `null` when the pane is not in a git repo — see [pane-commands.md](pane-commands.md)). Rows are grouped first by `repo`, then by `session`. The `pr_url`/`pr_number` JSON fields (added by `260609-r7ju-pane-map-pr-fields` — see [pane-commands.md](pane-commands.md)) surface the change's PR once it ships. The multi-repo work altered only the grouping; the frame's *presentation* (markdown tables + emoji health) is defined by the **Frame rendering** model below.

**Repo-section status frame.** The status frame renders one **repo section** per repo — an anchor line `📂 **{repo-path}** · {session}` followed by a markdown table of that repo's changes — then a **Watches** section. Grouping by repo (not per-row repo/session columns) is chosen for scannability. A pane whose main-worktree root could not be resolved renders under a `📂 **(unresolved repo)**` anchor rather than being dropped. At runtime the operator emits this as **bare markdown** — never wrapped in a ` ``` ` fence, which would render the tables as literal text. Example (fenced here only to display the source):

```
🛰️ **Operator** · 17:32 · tick #47 · **7 tracked**

📂 **~/code/foo** · work

| | ID | Health | Stage | PR |
|:--:|---|:--:|---|---|
| ▶ | `r3m7` | 🟢 | apply → review | |
| | `ab12` | ✅ | hydrate | https://github.com/acme/foo/pull/412 |

📂 **~/code/bar** · side

| | ID | Health | Stage | PR |
|:--:|---|:--:|---|---|
| ▶ | `k8ds` | 🟡 | review · idle 8m | |
| | `ef56` | 🔴 | apply · idle 32m ⚠️ | |

👁️ **Watches**

| Watch | Target | Health | Status |
|---|---|:--:|---|
| `linear-bugs` | ~/code/foo | 🟢 | 2 known · 1 completed · 3m ago |
```

**Frame rendering (markdown-native).** The frame is emitted as an assistant message and rendered by the agent harness as GitHub-flavored markdown — **ANSI escapes do not survive this path** (stripped as literal text *and* as real ESC bytes; empirically verified), and markdown **headings** (`#`/`##`/`###`) render as literal text and are unusable. The only channels that render are tables, **emoji**, **bold**, *italic*, `code spans`, and links. So the frame uses: a header line `🛰️ **Operator** · {HH:MM} · tick #{N} · **{N} tracked**`; a `📂 **{repo}** · {session}` anchor + change table per repo; a `👁️ **Watches**` anchor + table. Emoji are the sole color channel — health is 🟢 active/healthy · 🟡 idle/new-items · 🔴 stuck/errored · ✅ complete · ⚪ paused (geometric glyphs like `●◌✗` render monochrome and are not used). Change-table columns: autopilot `▶` (own column) · `ID` (code span) · Health (emoji) · Stage (with `⚠️` trailing on stuck rows) · PR (full `pr_url` as plain text — *not* a `[#N](url)` link, so it is selectable/copyable in a plain xterm; blank until shipped). The 🛰️/📂/👁️ emoji are the prominence/landmark anchors that headings would otherwise provide. Degrades cleanly: strip emoji and the Stage text still names the state; the URL is plain text regardless. Since szxd the authoritative column spec, frame example, and health-emoji tables live in `fab-operator.md` §4's `### Status Frame Format` subsection — extracted from tick step 1 (which now ends "emit the status frame — see Status Frame Format"), with the formerly 4x-repeated render-path rationale collapsed into one rule (emit bare markdown — no code fence, no headings, no ANSI; channels: tables, emoji, bold, italic, code spans, plain URLs) plus the distinct agent-critical runtime no-fence rule. The emitted frame is unchanged — only the skill's internal organization moved (the "Why emoji + table, not ANSI" design history lives here, in the Status Frame design decision below).

**Monitoring tick** (on each `/loop` tick or "any updates?"):

1. **Stage advance detection** — compare current stage to last-known. Report transitions, update baseline.
2. **Pipeline completion detection** — stage is hydrate, ship, or review-pr. Report and remove from monitored set.
3. **Review failure detection** — stage went from review back to apply. Report rework.
4. **Pane death detection** — change no longer in pane map. Report and remove from monitored set.
5. **Auto-nudge** — for each idle agent, run question detection and answer model (see below). (No post-intake `/git-branch` send — fab-new Step 11 creates or renames the branch inline; the former backlog-spawn nudge step was removed in w7dp. Only a detected branch/change mismatch warrants a `/git-branch` send, per pre-send validation.)
6. **Stuck detection** — for agents NOT detected as input-waiting in step 5, check idle duration. If idle at non-terminal stage for >15m, report as potentially stuck. Advisory only — an agent waiting for input is not stuck.

After processing all changes: if the monitored set is empty, stop the loop and report "All monitored changes complete."

**Watch tick dedup (`known` PLUS `completed`)**: On the tick's watch pass (skill §7 step 2; the §4 Tick Behavior watch step states the same union rule since uliv — its earlier "compare against `known`" summary contradicted §7), new-item detection deduplicates spawns against the **union of the watch's `known` and `completed` lists**. When a watch-spawned change reaches its `stop_stage`, the item ID moves from `known` to `completed` — but the source item (e.g., a Linear issue) may still match the watch query, and it MUST NOT be respawned. (Dedup against `known` alone re-enabled spawning at exactly that moment, producing a respawn loop.) Item IDs are added to `known` only after a successful spawn; `known` is capped at 200 entries (oldest pruned first); `completed` additionally answers "what did this watch produce?".

#### Auto-Nudge

The operator acts as a proxy for the user on routine operational questions.

**Question detection** — for each idle monitored agent:

1. Capture: `tmux capture-pane -t <pane> -p -S -20` (wide window compensates for line wrapping)
2. Claude turn boundary guard: if `^\s*>\s*$` appears in last 2 lines, skip (normal human-turn boundary)
3. Blank capture guard: if output is entirely blank/whitespace, skip (treat as "cannot determine")
4. Scan for question indicators: lines ending with `?` (tightened — last non-empty line only, <120 chars, skip comment/log prefixes), `[Y/n]`/`[y/N]`/`(y/n)`/`(yes/no)`, `Allow?`/`Approve?`/`Confirm?`/`Proceed?`, Claude Code permission prompts, `Do you want to...`/`Should I...`/`Would you like...`, lines ending with `:`/`:\s*$`, enumerated options (`[1-9]\)` patterns), `Press.*key`/`press.*enter`/`hit.*enter` (case-insensitive)
5. No match → normal idle behavior (stuck detection applies)
6. Match found → proceed to answer model. Bottom-most (most recent) indicator evaluated when multiple match.

**Answer model** — most detected questions are auto-answered. Rule 4 (numbered menus) classifies the prompt before answering: Routine prompts auto-answer, Strategic prompts escalate. Rule 6 escalates when the operator cannot determine what keystrokes to send. Evaluate in order:

1. Binary yes/no or confirmation prompt → `y`
2. `[Y/n]` or `[y/N]` prompt → `y`
3. Claude Code permission/approval prompt → `y`
4. Numbered menu or multi-choice → classify as **Routine** or **Strategic** using LLM judgment over the terminal capture, weighing four signals: option text length, semantic distinctness of options, surrounding agent context, and reversibility of the choice. No hardcoded keyword list, no agent-side sentinel/marker protocol. **Routine** (tool/permission prompts, binary-framed menus, synonymous-option menus) → `1` (first/default option). **Strategic** (multi-option choices representing materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach) → escalate to user. On classification uncertainty, treat as Strategic and escalate (asymmetric cost: false-negative strategic commits the queue to an unchosen direction; false-positive strategic is recovered by the 30m idle auto-default below).
5. Open-ended question where a concrete answer is determinable from visible terminal context → send that answer
6. Question where the operator cannot determine what keystrokes to send → escalate

No cooldown or retry limit — each question is evaluated independently. Worktree isolation and human PR merge provide the safety gate for auto-answered prompts.

**Idle Auto-Default on Strategic Escalations**: When rule 4 escalates a prompt as Strategic, the operator runs a per-prompt real-time idle timer from the moment the escalation log line is written. If the prompt remains idle for 30 minutes, the operator auto-answers and logs using the distinct `auto-defaulted` format (see Logging below).

- **Threshold**: 30 minutes, hardcoded. No operator-state-file field, no per-change override, no environment variable exposes this value — the §4 operator state file schema is unchanged.
- **Idle clock reset**: the timer resets on any terminal-state change in the pane — new content appended by the agent, user keystrokes that alter the prompt display, or the prompt's own redraw. The timer watches pane-idle-ness, not escalation-open-ness. Tick cadence already provides sub-minute resolution — no new polling infrastructure is required.
- **Answer selection** (priority order): (1) if the prompt visibly states a default (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), send that stated default; (2) otherwise, send `1`. This matches rule 4's existing "first/default" semantics for routine menus.
- **Scope (hard exclusion)**: applies ONLY to rule 4 Strategic escalations. Rule 6 ("cannot determine keystrokes") escalations MUST NOT trigger the idle auto-default — the operator does not know what the correct keystrokes are, so sending `1` or the stated default would emit nonsense into the pane. Rule-6 escalations remain open pending user action regardless of idle duration.

**Re-capture before send**: Before sending an auto-answer via `tmux send-keys`, MUST re-capture the terminal. If output changed since initial capture, abort — the agent is no longer waiting. Eliminates the race condition between detection and send.

**Logging**: Every auto-answer: `"{change}: auto-answered '{summary}' → {answer}"`. Escalated (rule 6 or rule 4 Strategic): `"{change}: can't determine answer for '{summary}'. Please respond."`. Auto-default after 30m idle on a Strategic escalation (distinct from `auto-answered` for grep-based after-action review): `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`.

#### Modes of Operation

Every mode follows the same rhythm: interpret user intent → refresh state → validate preconditions → execute → report → enroll in monitoring (if work dispatched).

| Mode | Description |
|------|-------------|
| **Broadcast** | Send command to all idle agents. Filter pane map, announce targets, send to each, enroll all |
| **Sequenced rebase** | "When X finishes, rebase Y on main." Enroll trigger change. When monitoring detects target stage, send rebase, enroll target |
| **Merge PRs** | Merge completed PRs at ship/review-pr stage. Retrieve URLs, confirm (destructive), merge from operator's shell |
| **Spawn agent** | New worktree + agent from backlog idea. Look up idea, create worktree, open tmux tab with Claude session running `/fab-new` |
| **Status dashboard** | Concise summary of all agents: change name, tab, stage, agent state. Include monitored set if active |
| **Unstick agent** | Nudge a stuck agent with `/fab-continue`. Verify idle first. If second nudge for same agent, warn. Send only on explicit insistence |
| **Notification** | "Tell me when X finishes." Enroll in monitoring. Loop handles notification automatically |
| **Autopilot** | Drive a queue of changes through the full pipeline. See below |

#### Autopilot

> The autopilot mechanics below describe the operator4 single-repo framing. Under the current `/fab-operator`, an autopilot queue may span repos with mixed dependency semantics, ordered merge tracks per-repo PR sequences, and CI-failure halts dependents only (not the whole sequence) — see "Multi-Repo Coordination" above for the authoritative behavior.

Drives a queue of changes through the full pipeline — spawning agents, monitoring progress, and collecting PRs for review. The default mode is **stack-then-review**: all queued changes build on each other via implicit `depends_on` chaining, PRs are created but NOT merged until the user explicitly requests merging. Confirm queue before starting (destructive tier). Default confirmation: "Confirm upfront (creates PRs — merge after review)."

**Queue ordering**: User-provided (exact order given), confidence-based (descending score), or hybrid (partial user constraints, confidence tiebreaker). User-provided ordering implies implicit `--base` chaining — each queued change after the first gets `depends_on: [<nearest-same-repo-predecessor>]`: the closest earlier queue entry in the **same repo** (cherry-picked); when no earlier entry shares the repo, the immediately previous entry as an ordering-only barrier (cross-repo → no code). (w7dp reconciled the rule text, worked example, Dependency Declaration paths, `--merge-on-complete` paragraph, and the spawn-next item to this one semantics — strict queue-previous would silently break same-repo stacking whenever a cross-repo entry sits in between, since a cross-repo dep contributes no code.)

**Per-change loop (stack-then-review, default)**: **Gate check confidence BEFORE anything spawns** (below the gate → flag to user; no worktree, no tab, no dispatch) → spawn worktree (`--reuse` for respawns; chaining per the nearest-same-repo rule above) → resolve dependencies (cherry-pick same-repo `depends_on` entries into the worktree; cross-repo = ordering-only barrier) → open the agent tab with the pipeline command **embedded at spawn** (§6 step 5 — `/fab-fff <change>`; the loop has no separate Dispatch item, so the command reaches the pane exactly once — w7dp fixed the #393 double-dispatch regression and renumbered the loop 9→8 items) → monitor → on completion, record branch in `branch_map`, collect PR URL → spawn next change (implicit `depends_on` per the chaining rule, its command likewise embedded at spawn) → report `"ab12: PR ready. 1 of 3 complete. Starting cd34."`.

**Queue completion summary**: When all changes in a stack-then-review queue complete, the operator displays a summary with all PR links and suggested merge order (base-first). The user can merge individually, or ask the operator to merge all in dependency order. When merging in order, the operator merges each PR sequentially, waiting for CI to pass before proceeding to the next. CI failure halts the merge sequence.

**`--merge-on-complete` opt-in**: Reverts to the previous merge-as-you-go behavior — merge each PR on completion, then `git fetch origin` and rebase the next change onto `origin/{default_branch}` (resolved per the step-0 chain in "Multi-Repo Coordination" above — never a hardcoded `origin/main`; g8st). Confirmation text changes to "Confirm upfront (merges PRs on completion)." Natural language equivalent: "merge as you go".

**Failure matrix**:

| Failure | Action | Resume? |
|---------|--------|---------|
| Confidence below gate | Flag to user: run `/fab-fff` or skip | Wait for user input |
| Review fails (rework exhausted) | Flag, skip to next | Yes |
| Cherry-pick conflict (stack-then-review) | Escalate, do not spawn | No — queue halts, wait for user input |
| Rebase conflict (merge-on-complete) | Flag, skip to next | Yes |
| Agent pane dies | 1 respawn attempt, then flag and skip | Yes |
| Stage timeout (>30 min same stage) | Flag regardless of retry state | Yes |
| Total timeout (>2 hr per change) | Flag for review | Yes |

**Interruptibility**: `"stop after current"` (finish active, halt queue), `"skip <change>"`, `"pause"` (stop new commands, running agents continue), `"resume"`. Interrupts acknowledged immediately.

**Resumability**: If the operator session restarts, state is reconstructable from `fab pane-map`. Resume from first non-completed change.

#### Configuration

| Setting | Default | Override |
|---------|---------|----------|
| Monitoring interval | 5m | "check every {N}m" |
| Stuck threshold | 15m | "flag agents stuck for more than {N} minutes" |
| Autopilot tick interval | 2m | "autopilot check every {N}m" |

All settings are session-scoped — they reset when the operator session restarts.

#### Design Constraints

- **Pane-map only**: Uses `fab pane-map` as its sole observation primitive — no `fab runtime is-idle`
- **No change artifacts**: Never reads intakes, specs, or tasks — context window reserved for coordination state
- **No persistent audit trail for v1**: Per-answer logging is inline only — no file-backed log
- **Hardcoded patterns**: Question indicator patterns embedded in skill file, not configurable via config.yaml

#### Launcher

The operator is launched via `fab operator` — a `fab-go` subcommand (source: `src/go/fab/cmd/fab/operator.go`). It creates a singleton tmux window named "operator" running the resolved spawn command (via `internal/spawn/`) with `'/fab-operator'`. The singleton check is an **exact, server-wide window-name match** (pw3k): it enumerates `tmux list-windows -a -F '#{window_id}\t#{window_name}'` and compares names exactly — never tmux target resolution, whose prefix/glob fallback would let a window like `operator-logs` falsely satisfy the check (and was session-scoped, missing an operator in another session on the same server). On exact match it selects the window by its server-global window ID (grammar-exempt) with a best-effort `switch-client` so a cross-session match moves the user's client; absence launches; a tmux enumeration error is surfaced distinctly (with the child's stderr), never conflated with "absent". Requires an active tmux session (`ERROR: not inside a tmux session` via the central RunE error path, exit 1). Previous shell launcher scripts (`fab-operator4.sh`, `fab-operator5.sh`, `fab-operator.sh`) have been removed.

**Launch preconditions: neither a git repo nor a `fab/` project is required** (2sdj). The launcher matches the per-tmux-server, cross-repo singleton model — its natural launch point is a neutral parent directory (e.g. `~/code`) with no git and no `fab/` project. `runOperator` resolves two things gracefully:

- **Window cwd** (`tmux new-window -c <dir>`): try `gitRepoRoot()` (`git rev-parse --show-toplevel`) first, fall back to `os.Getwd()` on failure. Inside a repo the window opens in the repo root (today's behavior, unchanged); outside any repo it opens in the current directory. The old hard `cannot determine repo root` error is gone — it errors only when BOTH `gitRepoRoot()` and `os.Getwd()` fail (`cannot determine working directory`, a genuinely broken environment). The git repo root was only ever the `-c` argument — it is NOT part of the operator state-file path (still socket-keyed under `XDG_STATE_HOME`, unchanged — no migration).
- **Spawn command**: when a `fab/` project resolves (`resolve.FabRoot()` succeeds), read `agent.spawn_command` from that project's `fab/project/config.yaml` via `spawn.Command(configPath)` (today's behavior). When `resolve.FabRoot()` fails — launched with no `fab/` project up the tree — fall back to `spawn.DefaultSpawnCommand` (`claude --dangerously-skip-permissions`) rather than erroring. A `fab/`-less launch reads **no** project `agent.spawn_command`/`agent.tiers` — there is no project to customize from.

**Coordinating agent runs on the doing tier** (2sdj). The operator launches its agent on a deliberately-chosen model rather than whatever the spawn command happened to specify. `runOperator` shells `fab resolve-agent apply` and parses the byte-stable `model=`/`effort=` stdout into the doing-tier `{model, effort}`. **`apply` is probed because it is the canonical member of the fab-owned, FIXED `doing` tier** in the stage→tier mapping (`fab resolve-agent` takes a STAGE, not a tier name) — a prominent call-site comment flags this coupling so a future remapping surfaces the dependency. The pure, unit-testable `resolveDoingProfile(stdout string) agent.Profile` does the parse + fallback (the live shell-out stays in `runOperator`, which passes `""` on command error); the resolved flags are appended to the END of the spawn command via `spawn.WithProfile(spawnCmd, model, effort)` (last-wins; duplicate `--effort` is accepted by the claude CLI, so the deliberate tier choice overrides whatever the configured spawn command pinned; each flag omitted when its value is empty, per the `empty ⇒ omit` convention). On **any** failure — the installed `fab` predating `resolve-agent`, no resolvable fab project, or empty/unparseable output — it falls back to the in-process built-in doing default `agent.DefaultTier(agent.TierDoing)` = `{claude-opus-4-8, high}`. This makes the operator the **first non-orchestrator (non-pipeline) consumer** of the `l3ja` agent-tier system (`resolve-agent`). A `fab/`-less launch therefore composes a fully-defaulted command: `spawn.DefaultSpawnCommand` + the doing default `{model, effort}`.

> **Caveat — installed-binary skew.** `fab resolve-agent` is shelled from PATH, so an installed `fab` that predates the `l3ja` `resolve-agent` subcommand makes the command error; the operator routes that (via `resolveDoingProfile("")`) to the built-in doing default rather than failing. This is the deliberate reason for the in-process fallback — the operator never assumes the PATH `fab` is new enough.

`fab operator` is a parent command with two subcommands:

- **`fab operator tick-start`** — Called at the start of each operator tick (step 1 of tick behavior). Derives the server-keyed operator state file path via `StatePath()` (no repo-root resolution), reads the file into `map[string]interface{}` using `gopkg.in/yaml.v3` (absent file treated as empty), increments `tick_count` by 1, writes `last_tick_at` as an RFC3339 UTC timestamp (`time.RFC3339`), writes the updated map back atomically preserving all other fields (monitored set, autopilot queue, branch_map, watches). Outputs `tick: N\nnow: HH:MM` to stdout using local time. Write failure → stderr error + exit 1. No flags.

- **`fab operator time`** — Pure clock query with no file I/O or side effects. Always outputs `now: HH:MM` (local 24-hour time). With `--interval <duration>` (Go duration string, e.g. `3m`), also outputs `next: HH:MM` = now + interval. Invalid duration string → stderr error + exit 1.

**Usage in tick lifecycle**: The agent invokes `fab operator tick-start` at step 1 of each tick and parses its stdout for the tick count (`tick: N`) and current time (`now: HH:MM`). Between ticks (idle message), the agent runs `fab operator time --interval {interval}` to obtain both `now:` and `next:` values for the idle message line `Waiting for next tick. Time: HH:MM · next tick: HH:MM`. Separation of concerns: `tick-start` has side effects (writes YAML state), `time` is a pure query (no writes).

## Design Decisions

### Auto-Answer Model with Strategic Escalation (rule 4 classification)
**Decision**: Detected questions are auto-answered by default via a numbered decision list (items 1-6, evaluated in priority order). Rule 4 (numbered menus) further classifies the prompt as Routine or Strategic before answering. Routine prompts (tool/permission, binary-framed, synonymous-option menus) auto-answer `1`. Strategic prompts (multi-option choices representing materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach) escalate to the user. Classification is LLM-judged over four signals in the terminal capture (option text length, semantic distinctness, surrounding agent context, reversibility); no hardcoded keyword list, no agent-side sentinel protocol. Classification uncertainty MUST escalate (asymmetric cost structure: silently committing the queue to an unchosen branch of work is more expensive than an extra user nudge).
**Why**: Worktree isolation and human PR merge are sufficient safety gates for routine operational prompts, but not for prompts that commit the queue to a direction the user never inspected (scope, PR split, spec/approach). A pure all-auto-answer model traded correctness for throughput in the exact scenarios where correctness matters most. Principle-based LLM classification adapts to novel prompt text without maintaining a keyword list or coupling the operator to every skill's surface area.
**Rejected**: Pure all-auto-answer (original model) — loses correctness on strategic prompts. Hardcoded keyword list — brittle, fails on novel prompts, high-maintenance. Agent-side `[STRATEGIC]` sentinel protocol — couples the operator to every skill and fails on Claude Code native + third-party prompts the operator cannot modify.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge (original model); 260422-hin2-operator-strategic-menu-escalation (Strategic classification + escalate-on-uncertainty)

### 30-Minute Idle Auto-Default on Strategic Escalations
**Decision**: When rule 4 escalates a prompt as Strategic, the operator starts a per-prompt real-time idle timer from the escalation log time. If the prompt remains idle for 30 minutes (no terminal-state change in the pane), the operator auto-answers — sending the prompt's stated default if visible (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), otherwise option `1`. The auto-default logs with a distinct format — `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"` — so after-action review tooling can distinguish confidently-auto-answered decisions from decisions taken because the user never returned. The idle clock resets on any terminal-state change in the pane (new agent output, user keystrokes, prompt redraw). The threshold is hardcoded 30 minutes — no operator-state-file field, no per-change override, no environment variable exposes it.
**Why**: Strategic escalations stall forward progress if the user is asleep, in meetings, or otherwise away from the terminal. Headless autopilot runs (overnight, multi-hour) become unreliable when every strategic escalation halts the pipeline. A 30-minute idle watchdog trades oversight for throughput in exactly the scenario where the alternative is zero throughput. The grep-distinct log format preserves auditability — `auto-defaulted` entries are recoverable separately from `auto-answered` entries.
**Rejected**: Configurable threshold (`.fab-operator.yaml`, per-change override, environment variable) — added surface area for marginal benefit; one threshold serves the single motivating "user is asleep / in a meeting" scenario well. Shorter threshold — risks auto-defaulting while the user is mid-reply. Longer threshold — defeats the feature. Uniform auto-default across all escalation types — conflates rule 4 Strategic (known-good default `1`) with rule 6 "cannot determine keystrokes" (auto-defaulting `1` would emit nonsense into the pane). Reusing the `auto-answered` log line — muddies audit trails by conflating confidently-answered and fell-back decisions.
*Introduced by*: 260422-hin2-operator-strategic-menu-escalation

### Re-Capture Before Send
**Decision**: The operator re-captures terminal output immediately before sending an auto-answer. If the output changed, the send is aborted.
**Why**: Eliminates the race condition between idle check and send. Single-tick grace period was rejected — it adds latency without fully solving the race.
**Rejected**: Single-tick grace period — delays answers by one full monitoring cycle and doesn't guarantee safety.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge

### Claude Turn Boundary Guard
**Decision**: If a Claude Code `>` prompt cursor (`^\s*>\s*$`) appears in the last 2 lines of captured output, question detection is skipped.
**Why**: Claude's output often contains question-like phrasing ("Would you like me to...?") that triggers detection. The `>` cursor indicates the agent is at a normal human-turn boundary, not a blocking prompt.
**Rejected**: Excluding all question-mark lines from Claude — too broad, would miss genuine blocking prompts from Claude.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge

### Operator Uses /fab-fff for Autopilot
**Decision**: Operator4 uses `/fab-fff` instead of `/fab-ff` for autopilot gate checks and pipeline invocations.
**Why**: `/fab-fff` is the more autonomous pipeline variant, fitting for operator-driven autopilot where human interaction is minimized.
**Rejected**: Keeping `/fab-ff` — its interactive fallback on review failure conflicts with the operator's autonomous mode.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge

### Standalone Operator Over Inheritance Chain
**Decision**: Operator4 is a fully self-contained skill file. Previous iterations (operator1, operator2, operator3) were deleted — their behavior is inlined into operator4. The skill file loads `_cli-external.md` (operator-only) for external tool references (`wt`, `tmux`, `/loop`).
**Why**: Understanding the operator previously required reading 4 files in sequence (operator1 -> 2 -> 3 -> 4), mentally applying overrides. The standalone version is readable from a single file plus standard `_` files. Dead operator files in the skills directory risked ghost triggers via sync.
**Rejected**: Keeping operator1/2/3 as archived files — git history preserves them; dead files risk agents loading them. Extracting a shared base — adds indirection for a single-consumer pattern.
*Introduced by*: 260315-a2b2-standalone-operator4-rewrite

### Use Case Registry Over Single-Purpose Monitoring
**Decision**: The operator uses a use case registry instead of single-purpose monitoring — named, toggleable concerns checked on each `/loop` tick. The loop is the operator's heartbeat (runs while any use case is enabled), not tied to the monitored set.
**Why**: Real workflows have multiple concurrent monitoring concerns (change progress, Linear inbox, PR staleness) that all need periodic attention. A registry model lets users toggle concerns without operator restarts. Three built-in use cases (fixed set, not user-extensible).
**Rejected**: CLI-level branch resolution (`fab resolve --search-branches`) — fab operates on change folders, not git branches; branch awareness belongs in the operator skill.
*Introduced by*: 260317-yrgo-operator5-branch-fallback

### Branch Fallback in Operator, Not CLI
**Decision**: Branch fallback resolution lives in the operator skill (user-initiated only), not in the `fab` CLI. When `fab resolve` fails, the operator scans branch names as a fallback before reporting failure.
**Why**: `fab` is orthogonal to git — it operates on change folders (filesystem/YAML). Branch name scanning is a coordination concern (finding where a change lives), not a CLI concern. The operator already has the context to decide between read-only (`git show`) and action (worktree creation) responses.
**Rejected**: `fab resolve --search-branches`, `--branch` output mode, automatic fallback in CLI — all rejected because they couple the CLI to git branch semantics.
*Introduced by*: 260317-yrgo-operator5-branch-fallback

### Dependency-Aware Agent Spawning (operator7)
**Decision**: `/fab-operator` (v7) adds pre-spawn dependency resolution to the operator. When spawning an agent for a change with `depends_on` entries, the operator cherry-picks dependency content into the worktree before opening the agent tab. Uses `git cherry-pick --no-commit origin/{default_branch}..<dep-branch> && git commit -m "operator: cherry-pick <dep> dependency"` (the base was a hardcoded `origin/main` until g8st resolved it). On conflict: abort, escalate, do not spawn.
**Why**: Without dependency awareness, agents working on dependent changes start from a baseline missing the dependency code, causing build failures, spec divergence, and manual intervention. This defeats the operator's "automate the routine" principle.
**Rejected**: `git merge --squash` — rejected for unattended sessions where merge machinery introduces risk. Transitive dependency resolution — rejected because leaf dependency branches already carry transitive content via the operator's own cherry-picking when those deps were spawned; `origin/{default_branch}..<dep-branch>` gives the complete transitive closure.
*Introduced by*: 260324-prtv-operator7-dep-aware-spawning; *Updated by*: 260612-g8st-git-state-hardening (base = fetched, resolved default branch — see "Operator Git Ops Fetch First and Resolve the Default Branch")

### Operator7 Schema Additions
**Decision**: `.fab-operator.yaml` gains three new fields: `depends_on` (list of change IDs per monitored entry), `branch` (change's branch name per monitored entry), and `branch_map` (top-level map persisting change ID → branch name after changes leave the monitored set). Redundant deps are pruned via `git merge-base --is-ancestor` before cherry-picking. The `--base` autopilot flag implies `depends_on`.
**Why**: Branch names must persist after dependencies complete (merged/archived) so downstream changes can still cherry-pick from them. Redundant dep pruning prevents duplicate cherry-picks in chains (B's branch already contains A's content).
*Introduced by*: 260324-prtv-operator7-dep-aware-spawning

### Operator7 Direct fab-new for Raw Text Spawns
**Decision**: When spawning agents from raw text descriptions, the operator passes the description directly to `/fab-new` instead of creating an intermediate backlog entry via `idea add`. The "From raw text" spawn path now follows the same structure as "From backlog ID": worktree → resolve deps → spawn with `/fab-new <description>` → enroll → completion.
**Why**: The `idea add` step created orphaned backlog entries in `fab/backlog.md` that served no further purpose — the intake's Origin section already captures the raw input for traceability. `/fab-new` natively accepts natural language descriptions, making the backlog indirection redundant overhead.
**Rejected**: Keeping `idea add` for backlog traceability — the intake artifact is the real record of a change's origin, not the backlog entry.
*Introduced by*: 260326-13ro-operator7-direct-fab-new-spawn

### Pipeline-First Routing Principle (operator7)
**Decision**: `/fab-operator` (v7) §1 Principles gains a "Pipeline-first routing" principle requiring the operator to route all new work through `/fab-new` then a pipeline command (`/fab-fff`, `/fab-ff`, `/fab-continue`). The operator MUST NOT dispatch raw inline implementation instructions to agent panes and MUST NOT use `/fab-continue` to skip intake for new work. Operational maintenance commands (merge PR, archive, delete worktree, rebase, `/git-branch`, `/fab-switch`) are exempt. A reinforcing blockquote in §6 "Working a Change" references the §1 principle.
**Why**: Without an explicit prohibition, an operator (especially after `/clear` or under time pressure) could shortcut by sending freeform implementation instructions directly to an agent pane — bypassing intake generation, confidence scoring, and the full pipeline. This violates the fab workflow's core value: specification-driven development with traceability (Constitution §II).
*Introduced by*: 260326-u3un-operator-enforce-pipeline-routing

### Stack-Then-Review Autopilot Default (operator7)
**Decision**: The autopilot queue defaults to **stack-then-review** mode. All queued changes after the first implicitly get `depends_on:` (equivalent to implicit `--base` chaining) — since w7dp naming the **nearest same-repo predecessor** (the closest earlier same-repo queue entry, cherry-picked; no same-repo predecessor → the immediately previous entry as an ordering-only barrier), not blindly the previous queue entry. PRs are created but not merged until the user reviews and explicitly requests merging. The previous merge-as-you-go behavior is preserved via `--merge-on-complete` opt-in flag. Queue completion produces a summary with all PR links and suggested merge order (base-first). Ordered merge waits for CI on each PR before proceeding to next.
**Why**: The previous merge-as-you-go default caused two problems: (1) rebase conflicts when rebasing dependent changes onto freshly-merged `origin/main` re-linearized commits that cherry-pick resolution had already handled, and (2) no opportunity for holistic review of the full change set before any code merged to `main`. Stack-then-review gives the user full review control over the entire queue.
**Rejected**: Keeping merge-as-you-go as default — too many rebase conflicts and no review control. Available as opt-in for users who want it.
*Introduced by*: 260327-gwg9-operator-base-chaining-default; *Updated by*: 260612-w7dp-orchestrator-dispatch-review-pr-recovery (chaining reconciled to nearest-same-repo-predecessor — the strict queue-previous rule contradicted the worked example and silently broke same-repo stacking across an intervening cross-repo entry)

### Standardized Tmux Tab Naming (operator7)
**Decision**: All agent tab names in `/fab-operator` use `»<wt>` format (right guillemet + worktree name, no space). Replaces the previous `fab-<id>` naming which was unreliable for new changes where the change ID doesn't exist at spawn time. The worktree name is always available at spawn time and unique across panes, making it a consistent identifier for all three spawn paths (existing change, raw text, backlog). Originally used `⚡` (zap emoji) as prefix, but switched to `»` (U+00BB) because the emoji's double-width rendering caused tmux tab bar misalignment and console output formatting issues.
**Why**: The `fab-<id>` format had two issues: (1) for new changes, the ID doesn't exist until `fab-new` runs inside the spawned agent, and (2) the raw-text path already used `fab-<wt>` as a workaround, creating inconsistency. The `»` prefix makes agent tabs visually distinct from other tmux windows while being single-width for consistent terminal rendering.
**Rejected**: Keeping `fab-<id>` with worktree fallback — adds conditional logic without benefit since worktree name is always available. `⚡` emoji — double-width rendering breaks tmux tab alignment.
*Introduced by*: 260328-iqt8-standardize-tmux-tab-naming

### `»` Prefix Extends to Enrolled Windows
**Decision**: The `»` convention applies to **every** monitored window, not just operator-spawned ones. On enrollment the operator invokes `fab pane window-name ensure-prefix <pane> »` (U+00BB). The primitive's literal-prefix idempotent check makes the step a no-op when the name already starts with `»`, covering operator-spawned, `/clear`-restored, and re-enrolled entries. The rename runs after the monitored entry is durably written to the operator state file (repo-rooted `.fab-operator.yaml` at the time of this decision; server-keyed since oy0k); a non-zero primitive exit logs one line and leaves the enrollment intact.
**Why**: With the original decision, the `»` prefix was only half-enforced — windows enrolled via direct command dispatch, user request ("watch this pane"), autopilot spawns, and watch spawns kept their original names, so the monitored set split visually into two indistinguishable populations (prefixed vs unprefixed). Extending the convention to every enrollment path makes the tab bar an accurate at-a-glance map of what the operator is currently tracking. Rename-after-YAML-write ordering guarantees that a partial failure leaves a tracked entry without the cosmetic prefix, never a prefix without a tracked entry.
**Rejected**: (1) Inventing a second signal (status-bar marker, pane title) — violates parity with the existing convention, adds surface area. (2) A generic "already-marked" regex (`^[»⚡…]`) — would silently absorb legacy or user-chosen markers; the literal `»` check keeps naming sovereignty with the user.
*Introduced by*: 260422-jyyg-operator-prefix-enrolled-windows. *Amended by*: 260423-rxu3-window-prefix-primitives (extracted the inline tmux shell into `fab pane window-name ensure-prefix`).

### Done-Marker Swap on Removal
**Decision**: On every removal path (change reaches its `stop_stage` or a terminal stage when `stop_stage` is null, pane dies, user explicitly stops monitoring), the operator invokes `fab pane window-name replace-prefix <pane> » ›` to swap the active-monitoring `»` prefix for the done-marker `›` (U+203A, single right guillemet). The primitive's literal-prefix guard silently no-ops on user-renamed windows (if the user renamed the window mid-monitoring so it no longer starts with `»`, the swap is skipped). Exit 2 (pane missing) is treated as successful removal; other non-zero exits log `"{change}: window rename skipped ({error})."` and the operator continues.
**Why**: Leaving `»` on removed windows made the tab bar lie about what the operator was currently tracking. The entire purpose of the prefix is at-a-glance coordination — an honest signal means "`»` = tracking now, `›` = operator touched but done, untouched = untouched." The done-marker `›` was chosen over `✓` because `✓` already appears in the operator status frame as the stage-done signal (`● apply → review ✓`); reusing it on window names would create a semantic collision. `›` preserves the guillemet visual family (`»` → `›`, double → single), is single-width BMP (consistent with the 260328 / 260416 decisions), and reads as "was-active, now trail-preserved." The literal-prefix guard inside `replace-prefix` replaces the prior "no restore on removal" rule: no state-storage (`original_name`) is needed, because the guard protects user-renamed windows inherently — if the current name no longer starts with `»`, the swap doesn't match and doesn't fire.
**Rejected**: (1) Leave `»` on removal (260422-jyyg's original rule) — self-defeating signal staleness; tab bar lies about what is tracked. (2) Restore the original name — requires storing `original_name` with a "user-renamed-mid-monitoring" ambiguity that the guard-based approach side-steps. (3) `✓` as the done-marker — collides with the stage-done signal in the operator status frame. (4) Per-project config option for the done-marker character — speculative, no current demand; the character is a skill constant in `src/kit/skills/fab-operator.md`.
*Introduced by*: 260423-rxu3-window-prefix-primitives

### Multi-Repo / Multi-Session Isolation = Tmux Server (one operator per server)
**Decision**: The operator's isolation unit is the **tmux server** — exactly one operator per server, spanning every session and repo on it, owning one **server-keyed** state file at `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/...`), keyed by the tmux socket path (the binary derives it via `StatePath()` — see [kit-architecture.md](../distribution/kit-architecture.md)). A second operator means a second tmux server (`tmux -L <label>`). There is **no `--name` dimension** — the server boundary is the only isolation knob, matching the server-wide singleton already enforced by the `operator` window. Old repo-rooted `.fab-operator.yaml` files are **not migrated** — abandoned in place (the monitored set is re-derivable from live `»`-prefixed panes). Every monitored entry, `branch_map` value (`{ branch, repo }`), and watch (`target_repo`) is repo-qualified under the `(session, repo, pane)` addressing tuple, with pane ID as the server-global primary key.
**Why**: A single operator coordinating multiple repos/sessions is the central value of the operator (one pane of glass). A repo-rooted state file is single-repo-only; a fixed global path would force a machine-wide singleton. Keying the file by the tmux socket scopes one owner across all repos on a server while still allowing a second server to host an independent operator. No migration is acceptable because operators don't survive a binary upgrade anyway.
**Rejected**: Per-repo operators (loses the single pane of glass). A `--name` operator dimension (redundant with the server boundary). Repo-rooted `.fab-operator.yaml` (single-repo only). A fixed global state path (machine-wide singleton). Migrating old state files (unnecessary — monitored set is re-derivable).
*Introduced by*: 260607-oy0k-operator-multi-repo-skill

### Cross-Repo Dependencies = Ordering-Only (no code merge)
**Decision**: `depends_on` resolution is two-tier, split by repo. A **same-repo** dependency cherry-picks (`git cherry-pick --no-commit origin/{default_branch}..<dep-branch>` — base fetched and resolved since g8st). A **cross-repo** dependency is an **ordering-only barrier**: the operator waits until the dependency reaches its `stop_stage`, then spawns — **no code is merged**. The skill states the REQUIRED caveat that a cross-repo dependency gives the dependent agent no code (pure logical sequencing), correct only for logical deps ("don't start the frontend until the API merges"), never code-level ones. Ancestor-pruning (`git merge-base --is-ancestor`) is scoped to the same-repo subset only.
**Why**: Cross-repo branches share no common default-branch base, so there is no sound cross-repo cherry-pick; logical sequencing is the only sound cross-repo semantic. Forbidding cross-repo deps would be too restrictive for real multi-repo workflows.
**Rejected**: Forbid cross-repo deps (too restrictive). Full cross-repo code merge (unsound — no shared base). Cross-repo ancestor-pruning (meaningless across repos with no shared history).
*Introduced by*: 260607-oy0k-operator-multi-repo-skill

### Repo-Spanning Autopilot CI-Failure = Halt-Dependents-Only
**Decision**: An autopilot queue may span repos with mixed dependency semantics (within-repo cherry-pick chaining degrades to cross-repo ordering-only barriers); ordered merge tracks per-repo PR sequences. On a CI failure during ordered merge, the operator halts the failing repo's merge sub-sequence AND any repo whose queued items carry a cross-repo `depends_on` into the failed chain — **transitively** over the cross-repo `depends_on` graph. Truly independent repos' sub-sequences **continue merging**. The completion summary reports halted vs. completed sub-sequences and escalates to the user.
**Why**: Maximizes independent-repo throughput while still respecting cross-repo ordering barriers — the failure's blast radius is isolated to its dependency cone rather than throttling the whole queue.
**Rejected**: Halt-all (conservative, throttles independent repos — the earlier lean, superseded during clarify). Halt-only-failing-repo (ignores cross-repo ordering barriers, would merge a dependent ahead of its failed barrier).
*Introduced by*: 260607-oy0k-operator-multi-repo-skill

### Operator Context Loading Trimmed to Three Files (zc9m)
**Decision**: `fab-operator.md` §2 Context Loading loads only `config.yaml`, `constitution.md`, and `context.md` (optional), and `fab-operator` is named in the `_preamble.md` §1 exception list with its 3-file load. The Operator Spawning Rules moved from `_preamble.md` § Naming Conventions into `_cli-external.md`'s `wt` section (single repo-targeting rule — the duplicate `fab spawn-command --repo` note at the tmux bullet was dropped); `fab-operator.md` §6 stays the normative spawn procedure. Loading-only change — no §1 principle, safety-model, or spawn-procedure text was altered.
**Why**: The operator loaded all 7 always-load files, but `code-quality.md`, `code-review.md`, and both doc indexes are used nowhere in the skill — against its own §1 "Context discipline" principle — and the operator re-pays the whole layer after every `/clear` in a long-lived session (it had the largest per-invocation context of any skill: 136,967B, now ~123KB effective at startup/`/clear`). The spawning rules served only the operator yet sat in the always-load preamble paid by every skill; `_cli-external` is loaded exclusively by the operator, making it the natural home. Deliberate behavior change, verifier-endorsed (finding f117; spawning-rules move f040).
**Rejected**: Keeping the 7-file load (pays for files the skill never reads). Leaving the spawning rules in the preamble (every skill pays for operator-only content). Keeping the duplicate repo-targeting rule at the tmux bullet (verbatim drift risk).
*Introduced by*: 260611-zc9m-preamble-context-diet

### Status Frame = Markdown Tables + Emoji (ANSI does not render)
**Decision**: The operator status frame is rendered as **markdown** — a header line, one `📂` repo-anchor + change table per repo, and a `👁️` Watches table. Health is shown with **emoji** (🟢🟡🔴✅⚪); IDs are `code spans`; the header/repo anchors use emoji + **bold**; the PR column holds the **full `pr_url` as plain text**. Earlier specs colored an ANSI-wrapped glyph (`\e[32m●\e[0m`) and a later iteration broadened ANSI to many fields ("structural color"). **Both were non-functional**: the frame is an assistant message rendered as markdown by the agent harness, and ANSI escapes are stripped on that path — verified empirically that neither literal `\e[` text nor real ESC bytes render, and that markdown headings also render as literal text. Emoji (glyphs, not escapes) are the only surviving color channel; markdown tables give real column alignment and absorb the wide PR URL.
**Why**: The operator never writes to a TTY directly — it emits text that the harness markdown-renders. Color/visual hierarchy therefore must come from channels that survive that render (emoji, tables, bold, italic, code spans), not ANSI or headings. PR URLs are surfaced as plain text (not `[#N](url)` links) so they are selectable/copyable in a plain xterm, which shows only the link display text, not the target.
**Rejected**: ANSI SGR codes (stripped by the markdown renderer — the bug this fixes). Markdown headings for the header/sections (render as literal `##` text). `[#N](url)` markdown links for PRs (xterm shows only `#N`, not a copyable URL). A sparse dedicated-then-folded PR cell / footnote block (chose a full-URL column for always-copyable PRs at the cost of table width). Geometric glyphs `●◌✗` for health (render monochrome — no color).
*Introduced by*: PR #387 follow-up (markdown-native operator frame; supersedes the merged-but-non-functional "structural color" ANSI spec)

### Operator Git Ops Fetch First and Resolve the Default Branch (g8st)
**Decision**: Every operator git sequence that uses the default branch as a base — the same-repo dependency cherry-pick and the `--merge-on-complete` rebase — runs `git fetch origin` first and resolves the repo's **actual** default branch instead of hardcoding `origin/main`: `git symbolic-ref --short refs/remotes/origin/HEAD` (strip `origin/`) → `gh repo view --json defaultBranchRef -q .defaultBranchRef.name` → probe the just-fetched refs (`main` when `origin/main` exists, else `master`). The resolved `origin/{default_branch}` replaces all three former hardcoded sites: the cherry-pick range, the "Why `origin/{default_branch}` as base" rationale block, and the `--merge-on-complete` rebase target.
**Why**: The cherry-pick range and rebase previously hardcoded `origin/main` with **no fetch step** — autopilot was unusable on any repo whose default branch isn't `main`, and even on main-defaulted repos the base could be stale. Fetch-first fixes the staleness; name resolution fixes the portability. The chain matches `/git-pr`'s default-branch resolution (one convention across autonomous git paths — see [change-lifecycle.md](../pipeline/change-lifecycle.md) § Git Integration); the operator's literal fallback probes refs rather than assuming, because the fetch has already run.
**Rejected**: A shared `_` helper file for the chain (two consuming skills, three lines each — a helper would add loading cost for every other skill; the chain is inlined per consumer). Keeping hardcoded `origin/main` with a config override (the resolution is mechanical; no config knob needed).
*Introduced by*: 260612-g8st-git-state-hardening

### Autopilot Single Dispatch Point: Gate Before Spawn, Command Embedded at Spawn (w7dp)
**Decision**: A spawned change's initial pipeline command has exactly **one** dispatch point — §6 spawn-sequence step 5 embeds `<command>` in the `tmux new-window` invocation. The autopilot per-change loop's separate "Dispatch" item is removed (the loop renumbered 9→8 items, dispatch folded into the open-tab item), and the confidence **Gate moved before the tab opens** — a below-threshold change gets no worktree, no tab, and no dispatch. The entry-form table's Existing-change initial command is the single parseable `/fab-fff <change>` (the change-name override) — `&&`-chained slash commands have no chaining semantics and MUST NOT be sent.
**Why**: The #393 (szxd) refactor left both the spawn-embedded command and the loop's separate Gate+Dispatch items live — the command fired twice into the same pane. All three entry forms and the Watches flow already send the initial command via the spawn tab, so keeping spawn-embedding required no new machinery, while gating after spawn would open tabs for work the gate then rejects. The former `/fab-switch <change> && /fab-proceed` chain was unparseable as a slash command; `/fab-fff <change>` targets the change directly (no switch pre-step) and picks up from its current stage.
**Rejected**: Bare tab + post-spawn `fab pane send` dispatch (adds an idle-detection/ready-state dependency to every spawn). Keeping the loop's Dispatch item with a "skip if already sent" note (preserves the double-dispatch hazard). Keeping the `&&` chain (no chaining semantics — the trailing command lands as literal text).
*Introduced by*: 260612-w7dp-orchestrator-dispatch-review-pr-recovery

### Operator Launch Is Git-Optional, `fab/`-Optional, and Doing-Tiered (2sdj)
**Decision**: `fab operator` requires neither a git repo nor a resolvable `fab/` project, and it launches its coordinating agent on the **doing tier**. (1) Window cwd: try `gitRepoRoot()`, fall back to `os.Getwd()`, error only if both fail — the old `cannot determine repo root` hard-fail is removed. (2) Spawn command: `agent.spawn_command` from the project config when `resolve.FabRoot()` succeeds, else `spawn.DefaultSpawnCommand` (non-fatal `FabRoot()` failure). (3) Model: shell `fab resolve-agent apply` (`apply` = canonical doing-tier stage in the FIXED stage→tier mapping), parse `model=`/`effort=`, and append `--model`/`--effort` to the END of the spawn command via `spawn.WithProfile` (last-wins, empty ⇒ omit); on ANY failure fall back to the in-process `agent.DefaultTier(agent.TierDoing)` = `{claude-opus-4-8, high}`. The parse+fallback is the pure `resolveDoingProfile(stdout) agent.Profile` (caller passes `""` on command error); `spawn.WithProfile` is a new reusable helper in `internal/spawn` (where `spawn.Command` already has 4 call sites). No state-file path or config-schema change — state stays socket-keyed; no migration.
**Why**: The git root was incidental — used ONLY as the tmux window `-c <dir>`, never in the socket-keyed state path — and forcing one repo root (and one owning `fab/` project) contradicted the operator's per-tmux-server, cross-repo singleton design, whose natural home is a neutral parent dir. The doing tier is the right cognitive mode for "execution that coordinates"; reusing `fab resolve-agent` (the canonical l3ja tier-resolution surface) picks up project `agent.tiers.doing` overrides for free, and the built-in fallback keeps the operator working when the PATH `fab` predates `resolve-agent` or no project is resolvable. This makes the operator the first non-orchestrator (non-pipeline) consumer of the agent-tier system.
**Rejected**: Keeping the hard git-repo (and `fab/`-project) precondition (contradicts the cross-repo singleton model; the git root was never essential). A new `fab resolve-tier <tier>` command (unneeded new surface — the `apply` stage probe already resolves the doing tier). Inline `--model`/`--effort` concatenation in `runOperator` (not reusable across `spawn.Command`'s 4 call sites, not unit-testable) — hence the shared `spawn.WithProfile` helper. Returning an error from the pure parse fn (forces caller branching for no benefit — empty/garbage ⇒ default is the single path).
*Introduced by*: 260613-2sdj-operator-doing-tier-no-git-dep


## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260613-2sdj-operator-doing-tier-no-git-dep | 2026-06-13 | **`fab operator` launch is git-optional, `fab/`-optional, and doing-tiered.** `runOperator` (`src/go/fab/cmd/fab/operator.go`) no longer hard-fails outside a git repo or without a `fab/` project: window cwd resolves via `gitRepoRoot()` → `os.Getwd()` fallback (errors only if both fail; the `cannot determine repo root` hard-fail is gone), and `resolve.FabRoot()` failure is non-fatal — spawn command is the project's `agent.spawn_command` when resolvable, else `spawn.DefaultSpawnCommand`. The operator now launches its coordinating agent on the **doing tier**: it shells `fab resolve-agent apply` (canonical doing-tier stage in the FIXED stage→tier mapping, flagged with a prominent WHY-apply call-site comment), parses `model=`/`effort=` via the new pure `resolveDoingProfile(stdout) agent.Profile`, and appends `--model`/`--effort` to the spawn command via the new reusable `spawn.WithProfile(spawnCmd, model, effort)` helper (last-wins, empty ⇒ omit). On ANY failure (installed `fab` lacking `resolve-agent`, no resolvable fab project, or unparseable output) it falls back to the built-in `agent.DefaultTier(agent.TierDoing)` = `{claude-opus-4-8, high}`. This makes the operator the **first non-orchestrator (non-pipeline) consumer** of the `l3ja` agent-tier system. State stays socket-keyed under `XDG_STATE_HOME` — no migration. New design decision "Operator Launch Is Git-Optional, `fab/`-Optional, and Doing-Tiered". `fab-operator.md`, `SPEC-fab-operator.md`, and `_cli-fab.md` `## fab operator` updated in lockstep; `_shared/configuration.md` records the non-pipeline doing-tier consumption. |
| 260612-w7dp-orchestrator-dispatch-review-pr-recovery | 2026-06-12 | **Autopilot single dispatch point** (skills-audit batch 2/5; fixes the #393/szxd f049 double-dispatch regression): the confidence **Gate moved before the tab opens** (below-threshold → no worktree, no tab, no dispatch), the initial pipeline command is **embedded at spawn** (§6 step 5) and the per-change loop's separate Dispatch item is removed (loop renumbered 9→8) — the command reaches the pane exactly once. Entry-form Existing-change command is the single parseable **`/fab-fff <change>`** (`&&`-chained slash commands have no chaining semantics; the change-name override makes the `/fab-switch` pre-step unnecessary). **Queue chaining reconciled to nearest-same-repo-predecessor** — rule text, worked example (`ab12 → cd34 → ef56`), Dependency Declaration paths, `--merge-on-complete` paragraph, and the spawn-next item all state the same semantics (closest earlier same-repo entry cherry-picked; none → immediately previous entry as ordering-only barrier). Stale post-intake `/git-branch` sends removed in lockstep (tick auto-nudge step, §6 pipeline-commands line, §2 routing example, `_cli-external.md` "New change (from backlog)" flow — fab-new Step 11 creates/renames the branch inline since #322). New design decision "Autopilot Single Dispatch Point"; "Stack-Then-Review Autopilot Default" updated in place. `SPEC-fab-operator.md` mirror updated same-PR. |
| 260612-g8st-git-state-hardening | 2026-06-12 | **Operator git ops fetch first + resolve the default branch** (skills-audit batch 3/5, behavior-flagged): same-repo dependency resolution gained a **step 0** — `git fetch origin`, then resolve `{default_branch}` via `git symbolic-ref --short refs/remotes/origin/HEAD` → `gh repo view --json defaultBranchRef` → probe the just-fetched refs (main when `origin/main` exists, else master). The resolved `origin/{default_branch}` replaces hardcoded `origin/main` at all three sites: the cherry-pick range (`git cherry-pick --no-commit origin/{default_branch}..<dep-branch>`), the "Why … as base" rationale block, and both `--merge-on-complete` rebase mentions (which now also fetch before rebasing). Fixes autopilot being unusable on non-main-default repos and the stale-base risk everywhere (no fetch step existed). New design decision "Operator Git Ops Fetch First and Resolve the Default Branch"; the dep-aware-spawning and cross-repo-ordering decisions' command text updated in place. Chain shared with `/git-pr` — see [change-lifecycle.md](../pipeline/change-lifecycle.md). `SPEC-fab-operator.md` mirror updated same-PR. |
| 260612-pw3k-operator-pane-perf-error-surfacing | 2026-06-12 | `fab operator` singleton check hardened (binary-review B5, F33): the old `tmux select-window -t operator` guard (prefix/glob target resolution — `operator-logs` falsely satisfied it; session-scoped — missed an operator in another session) is replaced by an **exact, server-wide** window-name match — `tmux list-windows -a -F '#{window_id}\t#{window_name}'` enumerated via the pure `findWindowExact` parser (`SplitN(line, "\t", 2)`), selecting an exact match by server-global window ID with best-effort `switch-client` for cross-session jumps. Code now matches the documented per-SERVER singleton invariant (§Multi-Repo Monitoring Model). Also: the `$TMUX` guard returns an error through RunE (central `ERROR: not inside a tmux session`, exit 1) instead of `os.Exit(1)` (F38), and `tmux new-window` / `gitRepoRoot` failures carry the child's stderr (F35). Launcher mechanism only — no skill-behavior change. |
| 260611-szxd-skills-twins-self-duplication-refactor | 2026-06-12 | **Spawn sequence stated once (f049)**: `fab-operator.md`'s canonical 6-step spawn sequence now appears only in §6 "Spawning an Agent"; the three Working-a-Change walkthroughs became a 3-row entry-form → initial-command table (`/fab-switch <change> && /fab-proceed` · `/fab-new <escaped-text>` · `/fab-new <id>` *(the existing-change command was later replaced by the single parseable `/fab-fff <change>` in 260612-w7dp — `&&` has no slash-command chaining semantics)*) + "run the §6 spawn sequence", and Autopilot steps 1–2 / Watches step 4 became one-line §6 references — variant extras preserved (shell-escaping, idea-lookup pre-step, `--reuse`, watch `stop_stage`/`spawned_by`). **Status Frame Format extracted (f116)**: the ~74-line frame spec moved out of tick step 1 into a `### Status Frame Format` subsection after Tick Behavior; the 4x-repeated render-path rationale collapsed to one rule (bare markdown — no fence/headings/ANSI; channels: tables, emoji, bold, italic, code spans, plain URLs), keeping the distinct runtime no-fence rule, the frame example, and both column tables; the "Why emoji + table, not ANSI" design-history paragraph was dropped from the skill (it lives in this file's Status Frame design decision). Emitted frame unchanged — skill-internal organization only. `SPEC-fab-operator.md` mirror updated. |
| 260611-zc9m-preamble-context-diet | 2026-06-11 | **Operator context loading trimmed 7 → 3 files** (skills-review batch 3/4, f117 — deliberate, verifier-endorsed): `fab-operator.md` §2 now loads only `config.yaml`, `constitution.md`, `context.md` (optional); `fab-operator` added to the `_preamble.md` §1 exception list. Loading-only change — no principle, safety-model, or spawn-procedure text altered. **Operator Spawning Rules relocated** (f040): the known-change/new-change worktree-branch strategy moved from `_preamble.md` § Naming Conventions into `_cli-external.md`'s `wt` section (operator-only helper, so only the operator pays); `fab-operator.md` §6 remains the normative spawn procedure; the `fab spawn-command --repo` repo-targeting rule now appears exactly once in `_cli-external.md` (tmux-bullet duplicate dropped). New design decision "Operator Context Loading Trimmed to Three Files". `SPEC-fab-operator.md` mirror updated (3-file context; spawn citation repointed to `_cli-external.md` § Operator Spawning Rules). |
| 260611-uliv-skills-staleness-sweep-frontmatter-fixes | 2026-06-11 | **Operator state file term (f114, B)**: `fab-operator.md` now defines the term **operator state file** once in §4 (`### Operator State File` heading, with the server-keyed path `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`) and uses the term for every live-file reference — the ~9 stale live `.fab-operator.yaml` mentions were replaced; `.fab-operator.yaml` remains only in the deliberate legacy not-read/not-migrated context. **Tick dedupe summary (M.3)**: the §4 Tick Behavior watch step now reads "compare against `known` + `completed` (§7 step 2's dedupe rule)", aligned with §7 — the prior "compare against `known`" summary contradicted it. This memory file's own live references were updated to the term (idle-auto-default threshold notes, the `»`-prefix enrollment decision, and the `fab operator tick-start` description — which derives the path via `StatePath()`, not `gitRepoRoot()` + a repo-rooted YAML). `SPEC-fab-operator.md` mirror updated in the same change. |
| 260611-9u91-skills-correctness-idempotency-fixes | 2026-06-11 | Closed the watch-respawn dedup hole (f018): `/fab-operator` §7 Tick Behavior step 2 now deduplicates spawns against the union of `known` **plus** `completed` — an item ID that moved from `known` to `completed` at `stop_stage` but still matches the watch query is skipped, not respawned in a loop. Skill + `SPEC-fab-operator.md` updated; no schema or Go change. |
| PR #387 follow-up | 2026-06-10 | Replaced the operator status frame's ANSI-based styling with a **markdown-native** design: header line + `📂` repo-anchor + per-repo change table + `👁️` Watches table, **emoji** health (🟢🟡🔴✅⚪), `code-span` IDs, and a **full-URL PR column** (from the `pr_url` JSON field). Roots out the non-functional "structural color" ANSI spec (escapes are stripped by the agent's markdown render path; headings render as literal text) — see the "Status Frame = Markdown Tables + Emoji" decision. Skill + memory + spec change only; no behavior change beyond rendering. |
| 260608-memory-domain-restructure | 2026-06-08 | Created `runtime/operator.md` by extracting the operator coordination content from `pipeline/execution-skills.md` (the `/fab-operator4` historical section, Monitoring System, tick lifecycle via `fab operator tick-start`/`time`, and the operator design-decision lineage from "Auto-Answer Model with Strategic Escalation" through "Done-Marker Swap on Removal"). Part of the memory-domain restructure that replaced the single `fab-workflow/` pseudo-domain with `pipeline/`, `memory-docs/`, `distribution/`, `runtime/`, and `_shared/`. The per-change history of these design decisions is preserved in `execution-skills.md`'s changelog (where the content lived when those changes shipped); this row records the extraction only. No operator behavior changed. |
| 260607-oy0k-operator-multi-repo-skill | 2026-06-08 | Re-framed `/fab-operator` for **multi-repo / multi-session coordination on one tmux server**. Addressing moved to the `(session, repo, pane)` tuple (pane ID stays primary key; `repo` + `session` are added dimensions). Resolved the h3jk deferral note and rewrote the repo-rooted "Monitoring System" prose into a server-keyed model: one operator per server, one server-keyed state file `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (no migration of old repo-rooted files), `monitored` entries gain `repo`+`session`, `branch_map` value becomes `{ branch, repo }`, watches gain `target_repo`. Tick snapshots via `fab pane map --all-sessions --json` grouped by repo→session with repo-section-header status frame. Added repo-targeted spawning (`wt create` in the target repo dir, `fab spawn-command --repo`), two-tier dependency resolution (same-repo cherry-pick / cross-repo ordering-only with the no-code caveat; ancestor-pruning scoped to same-repo subset), and repo-spanning autopilot with per-repo PR sequences and halt-dependents-only CI semantics (transitive over the cross-repo `depends_on` graph). Added three Design Decisions (tmux-server isolation / one-operator-per-server, cross-repo deps ordering-only, halt-dependents-only CI). Skill+specs change only; consumes change 1's (`260607-h3jk`) Go primitives, whose mechanism docs live in [kit-architecture.md](../distribution/kit-architecture.md) and [pane-commands.md](pane-commands.md). |
