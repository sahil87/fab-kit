# Execution Skills

**Domain**: fab-workflow

## Overview

Execution behavior (apply, review, hydrate) is accessed via `/fab-continue`, which dispatches to the appropriate behavior based on the active stage. The pipeline has 8 stages: intake → spec → tasks → apply → review → hydrate → ship → review-pr. The first 6 stages handle planning and execution; `ship` and `review-pr` handle integration (PR creation and PR review feedback). `/fab-continue` also dispatches ship (via `/git-pr` behavior) and review-pr (via `/git-pr-review` behavior). `/fab-archive` exists as a standalone housekeeping skill (not a pipeline stage) for moving completed changes to the archive — it requires `hydrate: done` but does not require ship/review-pr completion. All execution behaviors in `/fab-continue` inherit the optional `[change-name]` argument, which is passed to the preflight script for transient change resolution without modifying `.fab-status.yaml`.

**Status mutations**: All `.status.yaml` progress transitions, checklist updates, and confidence writes use `fab/.kit/bin/fab status` CLI event commands (`start`, `advance`, `finish`, `reset`, `fail`, `skip`, `set-checklist`, `set-confidence`) via the Bash tool, rather than direct file editing. The `skip` event transitions `{pending,active} → skipped` with forward cascade (all downstream pending stages become skipped). Skipped stages are treated as resolved for progression (like `done`). Resetting a skipped stage follows normal reset mechanics (`skipped → active`, downstream cascade to `pending`). This centralizes validation and ensures atomic writes with `last_updated` refresh. The `driver` parameter is optional but skills always pass it. Stage metrics (started_at, completed_at, driver, iterations) are updated automatically as side-effects.

**Hook-backed bookkeeping**: Checklist bookkeeping commands (e.g., `fab status set-checklist`) are now supplemented by a PostToolUse hook (`on-artifact-write.sh`) that fires on Write and Edit events. The hook is a **reliability layer** — it catches bookkeeping the agent forgets. Skills keep their existing bookkeeping instructions unchanged for agent-agnostic portability (non-Claude-Code agents rely on skill instructions only). All bookkeeping commands are idempotent, so both the hook and the skill running the same command produces no conflict.

**Pipeline invocation**: `/fab-fff` runs the full 8-stage pipeline (intake through review-pr); `/fab-ff` runs intake through hydrate (6 stages). All three pipeline skills (`/fab-continue`, `/fab-ff`, `/fab-fff`) dispatch review to a sub-agent in a separate execution context, producing structured findings with three-tier priority (must-fix / should-fix / nice-to-have). `/fab-continue` preserves manual rework on failure; both `/fab-ff` and `/fab-fff` use autonomous rework with bounded retry (3-cycle cap, escalation after 2 consecutive fix-code failures, stop on exhaustion). Both `/fab-ff` and `/fab-fff` have identical confidence gates (intake + spec) and accept `--force` to bypass them. Only `/fab-fff` extends past hydrate to invoke `/git-pr` behavior for the ship stage and `/git-pr-review` behavior for the review-pr stage. Both accept an optional `[change-name]` argument.

**Pipeline orchestrator**: `/fab-proceed` is a context-aware orchestrator that detects the current pipeline state and runs the minimum prefix steps needed before delegating to `/fab-fff`. It accepts no arguments or flags — everything is inferred from context. It does not run preflight or load preamble; it delegates those concerns entirely to `/fab-fff`. State detection follows a 4-step pipeline: (1) active change check via `fab resolve --folder`, (2) branch check via `git branch --show-current` compared to the resolved change folder name, (3) unactivated intake check by scanning `fab/changes/` (excluding `archive/`) for change folders (multiple matches resolved by most-recent date prefix `YYMMDD`), (4) conversation context check — if no intake exists, evaluate whether prior conversation contains substantive technical discussion (empty/greeting-only conversations are treated as "no context"). The dispatch table maps 5 detected states to step sequences:

| Detected state | Steps |
|---|---|
| No context, no intake | Error: "Nothing to proceed with — start a discussion or run /fab-new first." |
| Conversation context, no intake | Synthesize description → `/fab-new` → `/fab-switch` → `/git-branch` → `/fab-fff` |
| Intake exists, change not active | `/fab-switch` → `/git-branch` → `/fab-fff` |
| Active change, no matching branch | `/git-branch` → `/fab-fff` |
| Active change + matching branch | `/fab-fff` |

Each prefix step (`/fab-new`, `/fab-switch`, `/git-branch`) is dispatched as a subagent via the Agent tool per `_preamble.md` § Subagent Dispatch. The final `/fab-fff` delegation is invoked via the Skill tool in the current context (not as a subagent), since it is the terminal operation and its output should be visible to the user. `/fab-proceed` does not pass `--force` or any flags to `/fab-fff`. The skill is idempotent — re-running detects which steps are already complete and skips them. When conversation context synthesis is needed, the skill extracts decisions, rejected alternatives, constraints, and specific values from the conversation to produce a description substantive enough for `/fab-new` to generate a complete intake. Skill file: `fab/.kit/skills/fab-proceed.md`.

**Status confidence display**: `/fab-status` shows stage-aware confidence. At the intake stage, it computes an indicative score on the fly via `calc-score.sh --check-gate --stage intake` and displays `Indicative confidence: {score} (fab-ff gate: {threshold}) — {total} assumptions ({N} certain, {N} confident, {N} tentative)`, appending `, {N} unresolved` only when unresolved > 0. At spec stage or later, it reads the persisted confidence block from `.status.yaml` and displays `Confidence: {score} of 5.0 (...)`. When no confidence data exists and the stage is not intake, it falls back to `Confidence: not yet scored`. The `--check-gate` mode of `calc-score.sh` emits count fields (`certain`, `confident`, `tentative`, `unresolved`) in both the intake and spec branches.

**PR review handling**: `/git-pr-review` (renamed from `/git-review`) is a standalone autonomous skill that processes GitHub PR review comments from any reviewer — human, Copilot, or other bots. It drives the `review-pr` pipeline stage, integrating with statusman for stage tracking (start/finish/fail) and phase sub-state tracking via `stage_metrics.review-pr.phase` (values: waiting, received, triaging, fixing, pushed, replying). It replaces the former `/git-pr-fix` (Copilot-only) and `/git-pr` Step 6 (inline Copilot auto-fix). The detection flow prioritizes existing reviews: (Phase 1) check `GET /reviews` for existing non-PENDING reviews with inline comments via `GET /pulls/{number}/comments` — if found, proceed directly to fetch all comments across all reviewers (Path A); (Phase 2, fallback) if no reviews with comments exist, `POST /requested_reviewers` with `reviewers[]=copilot-pull-request-reviewer[bot]` — any non-2xx response means Copilot is not available, print message and stop; (Phase 3) poll `GET /reviews` for `copilot-pull-request-reviewer[bot]` every 30s, max 16 attempts / 8 minutes, then fetch Copilot-specific comments (Path B). If polling times out, the stage completes as `done` (not `failed`) — the absence of external review is a graceful no-op, same as Copilot unavailable. Human reviews take priority — if any reviewer has commented, the skill processes those comments and does NOT request Copilot. Comments are triaged with a disposition intent: `fix` (will change code — reply confirms with description and short SHA), `defer` (valid concern, out of scope — reply includes reason), `skip` (nitpick, stale, or not applicable — reply includes reason), or informational (no reply). Replies use past-tense to confirm outcomes: `Fixed —`, `Deferred —`, `Skipped —`. After commit and push (Step 5), a new Step 5.5 posts reply comments via REST API (`POST /pulls/{n}/comments` with `in_reply_to`) for each comment with a disposition. Reply posting is best-effort — failures are logged but do not abort the skill. The step also runs when no code changes were made (all deferred/skipped) to close the communication loop. Deduplication on re-run checks existing replies for `Fixed —`/`Deferred —`/`Skipped —` prefixes to avoid posting duplicate replies. The phase sub-state `replying` is set before posting replies. Commit messages are reviewer-aware: `fix: address copilot review feedback`, `fix: address review feedback from @{username}`, or `fix: address PR review feedback` (multiple reviewers). Path A uses `--paginate` for high-comment PRs and fetches `id` and `node_id` fields alongside comment data. Note: the Copilot bot uses different login names across API endpoints — `"Copilot"` in `GET /requested_reviewers`, `"copilot-pull-request-reviewer[bot]"` in `GET /reviews` and `POST /requested_reviewers` body (empirically confirmed). GitHub's REST API does not expose thread resolution state on individual comments — all non-reply comments are processed regardless of resolution. The skill is idempotent — re-running after fixes finds no new modifications and exits cleanly; re-running after replies skips already-replied comments.

**PR shipping**: `/git-pr` drives the `ship` pipeline stage, integrating with statusman for stage tracking (start/finish). All PRs are created as drafts (`gh pr create --draft`) — this is unconditional with no configuration toggle. Developers mark PRs ready for review manually after inspecting the agent-generated implementation. The ship stage has no `failed` state — git-pr fails fast and the user retries. Statusman calls are best-effort (silently ignored on failure). After PR creation, Step 4 executes four sub-steps in order: 4a (record PR URL via `fab status add-pr`), 4b (finish ship stage via `fab status finish` — best-effort), 4c (commit and push both `.status.yaml` and `fab/changes/{change-name}/.history.jsonl` to git), 4d (write `.pr-done` sentinel). All status mutations (4a, 4b) occur before the commit boundary (4c), ensuring no uncommitted fab state files remain after PR creation completes.

**PR type system**: `/git-pr` supports 7 PR types (feat, fix, refactor, docs, test, ci, chore) derived from Conventional Commits. Types are resolved via a four-step chain: explicit argument → read from `.status.yaml` → infer from fab change intake → infer from diff. The type controls the PR title prefix (`{type}: {title}`). All types use a single unified template with conditional field population based on artifact availability (not type). The template includes: a Summary section (from intake or commits), an optional Changes section (from intake subsections), a Change section (conditional on `{has_fab}` — see below), a Stats table with Type/Confidence/Checklist/Tasks/Review columns (`—` for unavailable fields), and a Pipeline progress line showing completed stages with intake/spec as hyperlinks. Blob URLs use `https://github.com/{owner}/{repo}/blob/{branch}/...` to resolve against the feature branch instead of main. Title derivation uses intake heading when available, commit subject otherwise, regardless of type.

**PR change metadata**: When an active fab change resolves (gated on `{has_fab}`), `/git-pr` generates a "Change" section placed above the "Stats" section in the PR body. The section contains a three-column table (ID, Name, Issue) showing the change's identity and linked Linear issues. ID is the 4-char change ID from `.status.yaml`; Name is the full change folder name; Issue shows linked issues from `fab status get-issues`. When `linear_workspace` is configured in `fab/project/config.yaml`, issue IDs are rendered as hyperlinks using `https://linear.app/{linear_workspace}/issue/{ISSUE_ID}`. Without `linear_workspace`, bare issue IDs are shown. Multiple issues are comma-separated. Missing fields show `—`. When no active fab change exists, the entire Change section is omitted.

## Requirements

### Apply Behavior (via `/fab-continue`)

`/fab-continue` dispatches to apply behavior when the active stage is `tasks` or `apply`. It executes tasks from `tasks.md` in dependency order, running tests after each completed task.

#### Pattern Extraction

Before executing the first unchecked task, the agent reads existing source files in the areas the change will touch and extracts: naming conventions, error handling style, typical structure, and reusable utilities. These patterns are held as context for all subsequent task execution. If `config.yaml` defines a `code_quality` section, its `principles` are loaded as additional constraints and `test_strategy` governs test timing (default: `test-alongside`). Pattern extraction is skipped when resuming mid-apply.

#### Task Execution

1. Parse `tasks.md` for unchecked items (`- [ ]`)
2. Execute tasks in dependency order
3. Respect parallel markers `[P]`
4. For each unchecked task:
   1. Read source files relevant to this task
   2. Implement per spec, constitution, and extracted patterns
   3. Prefer reusing existing utilities over creating new ones
   4. Keep functions focused — consider extracting if implementation exceeds the codebase's typical function size
   5. Write tests per `code_quality.test_strategy` (default: `test-alongside`)
   6. Run tests, fix failures
   7. Mark `[x]` immediately
5. Update `.status.yaml` progress after each task

#### Resumability

Apply behavior is inherently resumable. If the agent is interrupted mid-run, re-invoking `/fab-continue` picks up from the first unchecked item. The markdown checklist *is* the progress state — no separate tracking needed.

#### Context

Loads: config, constitution, `specs/index.md`, `tasks.md`, `spec.md`, relevant source code (files referenced in tasks), neighboring files for pattern extraction.

### Review Behavior (via `/fab-continue`)

`/fab-continue` dispatches to review behavior after apply completes. Review validation is dispatched to a **sub-agent running in a separate execution context**. The sub-agent provides a fresh perspective — it has no shared context with the applying agent beyond the explicitly provided artifacts.

The orchestrating LLM MAY use any review agent available in its environment (e.g., a `code-review` skill, a general-purpose sub-agent with review instructions, or any equivalent). The skill files do not hardcode a specific agent name or tool.

The review sub-agent performs capable-tier work: deep reasoning, code analysis, spec comparison, and checklist validation.

**Context**: The review sub-agent receives standard subagent context files (per `_preamble.md` § Standard Subagent Context) plus change-specific files: `spec.md`, `tasks.md`, `checklist.md`, relevant source files (files touched by the change), and target memory file(s) from `docs/memory/`. The `fab/project/**` files are no longer listed ad-hoc in the review behavior — they are inherited from the standard subagent context template in `_preamble.md`.

#### Validation Checks

The sub-agent performs all of these checks:
1. All tasks in `tasks.md` marked `[x]`
2. All checklist items in `checklist.md` verified and checked off — inspects relevant code/tests per `CHK-*` item, marks `[x]` or reports failure
3. Run tests affected by the change (scoped to modules touched, not the full suite)
4. Features match spec requirements (spot-check key scenarios from `spec.md`)
5. No memory drift detected (implementation doesn't contradict memory files)
6. Code quality check — for each file modified during apply: naming conventions consistent with surrounding code, functions focused and appropriately sized, error handling consistent with codebase style, existing utilities reused. If `config.yaml` defines `code_quality.principles`, check each applicable principle. If `code_quality.anti_patterns` defined, check for violations. Report code quality issues with specific file:line references; classify as should-fix by default, and as must-fix only when they correspond to spec mismatches, functional defects, or violations of hard project constraints

#### Structured Review Output

The sub-agent returns structured findings with a three-tier priority scheme:

- **Must-fix**: Spec mismatches, failing tests, checklist violations — always addressed during rework
- **Should-fix**: Code quality issues, pattern inconsistencies — addressed when clear and low-effort
- **Nice-to-have**: Style suggestions, minor improvements — may be skipped

Each finding includes: severity tier, description, and file:line reference where applicable. Pass/fail determination: if any must-fix findings exist, the review fails. If only should-fix and/or nice-to-have remain, the review MAY pass.

#### On Pass

All checks succeed → stage advances to review done via `statusman.sh finish <change> review`. Review outcomes are auto-logged: `statusman.sh finish review` automatically calls `logman.sh review "passed"`. Skills do not call `log-review` manually.

#### On Failure

Review failure is auto-logged: `statusman.sh fail <change> review [driver] [rework]` automatically calls `logman.sh review "failed" [rework]`. Skills do not call `log-review` manually. Rework behavior differs by invoking skill:

**`/fab-continue` (manual rework)**: Presents the sub-agent's prioritized findings to the user, then offers three rework options:

- **Fix code** — the agent identifies affected tasks, unchecks them in `tasks.md` with `<!-- rework: reason -->` annotations, re-runs apply, then spawns a fresh sub-agent for re-review
- **Revise tasks** — the user edits `tasks.md` (add/modify tasks), then the agent re-runs apply for unchecked tasks and spawns a fresh sub-agent for re-review
- **Revise spec** → `/fab-continue spec` — resets to spec stage, regenerates downstream, re-runs apply, then spawns a fresh sub-agent for re-review

**`/fab-ff` (auto-loop + stop)**: The agent triages the sub-agent's prioritized findings and autonomously selects the rework path (up to 3 cycles). Each cycle = one rework action + one re-review by a fresh sub-agent. On exhaustion after 3 cycles, stops with a per-cycle summary.

**`/fab-fff` (auto-loop + stop)**: Same autonomous triage and rework as `/fab-ff` — identical behavior, identical 3-cycle cap, identical escalation rule. On exhaustion after 3 cycles, stops with a per-cycle summary.

**Escalation rule** (applies to `/fab-ff` and `/fab-fff` auto-loops): After 2 consecutive "fix code" attempts, the agent MUST escalate to "revise tasks" or "revise spec". Non-fix-code actions reset the consecutive counter.

**Comment triage**: The applying agent triages review comments by priority — not all comments need to be implemented. Must-fix items are always addressed. Should-fix items are addressed when clear and low-effort. Nice-to-have items may be acknowledged but deferred.

The general rule: **artifacts at and after the re-entry point are regenerated or updated; artifacts before it are preserved.**

#### Context

Loads: config, constitution, `specs/index.md`, `tasks.md`, `checklist.md`, `spec.md`, target memory file(s) from `docs/memory/`, relevant source code (files touched by the change).

### Hydrate Behavior (via `/fab-continue`)

`/fab-continue` dispatches to hydrate behavior after review passes. It completes the pipeline: validates review passed and hydrates learnings into memory files. The change folder remains in `fab/changes/` after hydrate — archiving is a separate step via `/fab-archive`.

#### Behavior

1. **Final validation** — review MUST have passed (all tasks `[x]`, all checklist items `[x]` including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose specs reference the same memory files. If found, warn: "Change {name} also modifies {file}. After this hydrate, that change's spec was written against a now-stale base. Re-review with `/fab-continue` after switching to it."
3. **Hydrate into `docs/memory/`**:
   - From `spec.md` → integrate new/changed requirements and scenarios into the Requirements section. Remove requirements the spec explicitly deprecates. Extract durable design decisions into Design Decisions section
   - Compare against existing memory file to determine what's new vs changed vs removed — no explicit delta markers needed
   - Minimize edits to unchanged sections to prevent drift
4. **Update status** to `hydrate: done` in `.status.yaml`
5. **Pattern capture** *(optional)* — if the change introduced non-obvious implementation patterns that future changes should follow (e.g., a new error handling approach, a reusable abstraction), note them in the relevant memory file's Design Decisions section with the change name for traceability. Skip for implementations that follow existing patterns

#### Recovery

Hydration modifies memory files in-place. If the merge goes wrong, the only recovery is `git checkout` on the affected memory files. Commit (or at least review the diff) before pushing after hydrate.

#### Context

Loads: config, constitution, `specs/index.md`, `spec.md`, `intake.md`, target memory file(s) from `docs/memory/`, `docs/memory/index.md` and relevant domain indexes.

### `/fab-archive` (Standalone Skill)

`/fab-archive` is a standalone housekeeping command — not a pipeline stage. It supports two modes: **archive** (default) moves completed changes to the archive; **restore** moves archived changes back to active.

#### Archive Mode

##### Precondition

Requires `hydrate: done` in `.status.yaml`. If hydrate is not done, it stops with: "Hydrate has not completed. Run /fab-continue to hydrate memory first."

##### Behavior

1. **Move change folder** — `fab/changes/{name}/` → `fab/changes/archive/{name}/`. Create `archive/` if needed. No rename.
2. **Update archive index** — prepend entry to `fab/changes/archive/index.md` (create with backfill if missing). Format: `- **{folder-name}** — {1-2 sentence description}`. Most-recent-first.
3. **Mark backlog items done** — exact-ID check (always), then keyword scan with interactive confirmation
4. **Clear pointer** — remove `.fab-status.yaml` symlink only if the archived change is the active one

##### Fail-Safe Order of Operations

Steps 1–4 execute in this order for safety. Folder move first (recoverable if interrupted — re-run detects folder already in archive and completes remaining steps). Index after folder is in place. Backlog marking after index. Pointer last.

#### Restore Mode (`/fab-archive restore <change-name> [--switch]`)

Restores an archived change back to `fab/changes/`. Inverse of the archive operation. Preserves all artifacts and `.status.yaml` without modification — no status reset, no artifact regeneration.

##### Precondition

`<change-name>` is required. Resolved via case-insensitive substring matching against folder names in `fab/changes/archive/`. Supports exact/single/ambiguous/no-match flows (same pattern as `/fab-switch`).

##### Behavior

1. **Move change folder** — `fab/changes/archive/{name}/` → `fab/changes/{name}/`. No rename. All artifacts preserved.
2. **Remove archive index entry** — remove the entry for `{name}` from `fab/changes/archive/index.md`. Preserve empty index file.
3. **Update pointer** (conditional) — if `--switch` flag provided, create `.fab-status.yaml` symlink pointing to `fab/changes/{name}/.status.yaml`. Otherwise no-op.

Steps execute 1→3 for safety. If interrupted, re-run detects folder already in `fab/changes/` and completes remaining steps (index cleanup, optional pointer update).

#### Key Properties

- Does NOT modify `.status.yaml` progress (may update `last_updated`)
- Accepts optional `[change-name]` argument for targeting a specific change (archive mode)
- Conditional pointer clearing in archive mode — only removes `.fab-status.yaml` when the archived change is the active one
- Restore mode requires explicit `<change-name>` — no "restore most recent" convenience
- Restore mode optionally activates via `--switch` flag

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

The operator loads the always-load layer (`_preamble.md` §1) plus `fab/.kit/skills/_cli-external.md` (external tool reference for `wt`, `tmux`, and `/loop` — loaded only by operator, not by pipeline skills). It does NOT run preflight. It does NOT load change-specific artifacts.

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

#### Monitoring System

The operator maintains a monitored set in conversation context (not a persistent file). Each entry tracks: change ID, pane, last-known stage, last-known agent state, enrolled-at timestamp, last-transition-at timestamp.

**Enrollment triggers**: operator sends a command to it, user requests monitoring, operator triggers an automatic action toward it. Read-only actions do not enroll. **Spawned agents are always auto-enrolled** — the operator MUST NOT ask the user whether to monitor a spawned agent. This constraint is reinforced in both the §1 principles and the "Spawning an Agent" procedural subsection of each operator skill (operator6, operator7) to ensure proximity-based LLM adherence.

**Removal triggers**: change reaches a terminal stage (hydrate, ship, review-pr), pane dies, user explicitly stops monitoring.

**`/loop` lifecycle**: Start when first change enrolled (no loop running) — `/loop 5m "check monitored agents"`. Stop when monitored set empty. One-loop invariant: at most one active `/loop` at any time.

**Monitoring tick** (on each `/loop` tick or "any updates?"):

1. **Stage advance detection** — compare current stage to last-known. Report transitions, update baseline.
2. **Pipeline completion detection** — stage is hydrate, ship, or review-pr. Report and remove from monitored set.
3. **Review failure detection** — stage went from review back to apply. Report rework.
4. **Pane death detection** — change no longer in pane map. Report and remove from monitored set.
5. **Auto-nudge** — for each idle agent, run question detection and answer model (see below). If a monitored agent was spawned for a new change from backlog and the tick detects the change has advanced past intake, send `/git-branch` to that agent's pane (aligns branch name with newly created change folder).
6. **Stuck detection** — for agents NOT detected as input-waiting in step 5, check idle duration. If idle at non-terminal stage for >15m, report as potentially stuck. Advisory only — an agent waiting for input is not stuck.

After processing all changes: if the monitored set is empty, stop the loop and report "All monitored changes complete."

#### Auto-Nudge

The operator acts as a proxy for the user on routine operational questions.

**Question detection** — for each idle monitored agent:

1. Capture: `tmux capture-pane -t <pane> -p -l 20` (wide window compensates for line wrapping)
2. Claude turn boundary guard: if `^\s*>\s*$` appears in last 2 lines, skip (normal human-turn boundary)
3. Blank capture guard: if output is entirely blank/whitespace, skip (treat as "cannot determine")
4. Scan for question indicators: lines ending with `?` (tightened — last non-empty line only, <120 chars, skip comment/log prefixes), `[Y/n]`/`[y/N]`/`(y/n)`/`(yes/no)`, `Allow?`/`Approve?`/`Confirm?`/`Proceed?`, Claude Code permission prompts, `Do you want to...`/`Should I...`/`Would you like...`, lines ending with `:`/`:\s*$`, enumerated options (`[1-9]\)` patterns), `Press.*key`/`press.*enter`/`hit.*enter` (case-insensitive)
5. No match → normal idle behavior (stuck detection applies)
6. Match found → proceed to answer model. Bottom-most (most recent) indicator evaluated when multiple match.

**Answer model** — all detected questions are auto-answered. The only escalation is when the operator cannot determine what keystrokes to send. Evaluate in order:

1. Binary yes/no or confirmation prompt → `y`
2. `[Y/n]` or `[y/N]` prompt → `y`
3. Claude Code permission/approval prompt → `y`
4. Numbered menu or multi-choice → `1` (first/default option)
5. Open-ended question where a concrete answer is determinable from visible terminal context → send that answer
6. Question where the operator cannot determine what keystrokes to send → escalate

No cooldown or retry limit — each question is evaluated independently. Worktree isolation and human PR merge provide the safety gate.

**Re-capture before send**: Before sending an auto-answer via `tmux send-keys`, MUST re-capture the terminal. If output changed since initial capture, abort — the agent is no longer waiting. Eliminates the race condition between detection and send.

**Logging**: Every auto-answer: `"{change}: auto-answered '{summary}' -> {answer}"`. Escalated (item 6): `"{change}: can't determine answer for '{summary}'. Please respond."`.

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

Drives a queue of changes through the full pipeline — spawning agents, monitoring progress, and collecting PRs for review. The default mode is **stack-then-review**: all queued changes build on each other via implicit `depends_on` chaining, PRs are created but NOT merged until the user explicitly requests merging. Confirm queue before starting (destructive tier). Default confirmation: "Confirm upfront (creates PRs — merge after review)."

**Queue ordering**: User-provided (exact order given), confidence-based (descending score), or hybrid (partial user constraints, confidence tiebreaker). User-provided ordering implies implicit `--base` chaining — each queued change after the first gets `depends_on: [<prev-change-id>]` automatically.

**Per-change loop (stack-then-review, default)**: Spawn worktree (`--reuse` for respawns, `--base <prev-change>` for user-provided ordering) → resolve dependencies (cherry-pick `depends_on` entries into worktree) → open agent tab with `/fab-switch <change>` → gate check confidence (if >= gate, send `/fab-fff`; if < gate, flag to user) → monitor → on completion, record branch in `branch_map`, collect PR URL → dispatch next change (with implicit `depends_on`) → report `"ab12: PR ready. 1 of 3 complete. Starting cd34."`.

**Queue completion summary**: When all changes in a stack-then-review queue complete, the operator displays a summary with all PR links and suggested merge order (base-first). The user can merge individually, or ask the operator to merge all in dependency order. When merging in order, the operator merges each PR sequentially, waiting for CI to pass before proceeding to the next. CI failure halts the merge sequence.

**`--merge-on-complete` opt-in**: Reverts to the previous merge-as-you-go behavior — merge each PR on completion, rebase next change onto `origin/main`. Confirmation text changes to "Confirm upfront (merges PRs on completion)." Natural language equivalent: "merge as you go".

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

#### Launcher Script (removed)

The `fab-operator4.sh` launcher has been removed. The current operator launcher is `fab-operator.sh`.

## Design Decisions

### Checklist Tests Implementation Fidelity and Code Quality
**Decision**: The quality checklist validates "does the code match the spec?" (implementation fidelity) and "is the code well-written?" (code quality). Code Quality is always included with at least two baseline items (pattern consistency, no unnecessary duplication); additional items derive from `config.yaml` `code_quality` section when present.
**Why**: Spec quality is addressed during the spec stage (via `/fab-clarify`), but code quality is only observable at review time. The baseline items are universally applicable; project-specific standards come from config.
**Rejected**: Code quality as opt-in only — would miss quality checks on projects without `code_quality` config. SpecKit-style requirement-quality checklist — duplicates planning-stage work.
*Source*: doc/fab-spec/TEMPLATES.md
*Updated by*: 260215-r8k3-DEV-1024-code-quality-layer

### Sub-Agent Over Inline Review
**Decision**: Review validation is dispatched to a sub-agent in a separate execution context, replacing inline review by the applying agent.
**Why**: Same-context review is fundamentally limited by shared cognitive biases. The sub-agent provides a fresh perspective — it has no shared context with the applying agent beyond the explicitly provided artifacts.
**Rejected**: Multiple inline review passes (still shares context), external review tool integration (too prescriptive, not portable).
*Introduced by*: 260216-gqpp-DEV-1040-code-review-loop

### Standard Subagent Context Template
**Decision**: All subagent prompts include a standard set of `fab/project/**` files (`config.yaml`, `constitution.md`, `context.md`, `code-quality.md`, `code-review.md`) defined in `_preamble.md` § Standard Subagent Context. Review behavior references this template instead of listing files ad-hoc.
**Why**: Previously, each skill that dispatched subagents maintained its own context list, creating silent quality gaps when files were omitted and drift risk as new project files were added. The template in `_preamble.md` centralizes the list so all subagents — including nested sub-subagents — inherit project principles automatically. Total context cost is ~150 lines, negligible.
**Rejected**: Selective per-subagent file lists (maintenance burden, drift risk), loading only for review subagents (apply and other subagents also benefit from project principles).
*Introduced by*: 260318-dzze-standard-subagent-context

### Priority-Based Comment Triage
**Decision**: The applying agent triages review comments by severity (must-fix / should-fix / nice-to-have) rather than implementing all of them.
**Why**: Prevents infinite rework loops over diminishing-return suggestions. Must-fix items ensure correctness; nice-to-have items allow pragmatic completion.
**Rejected**: Fix all comments (leads to infinite loops on style disagreements), ignore all non-critical (misses genuine should-fix quality issues).
*Introduced by*: 260216-gqpp-DEV-1040-code-review-loop

### fab-ff Gains Auto-Loop with Interactive Fallback
**Decision**: `/fab-ff` auto-loops between apply and review (up to 3 cycles) before falling back to interactive rework. Previously, fab-ff always presented interactive rework immediately on failure.
**Why**: Sub-agent review enables tighter automated feedback cycles. The interactive fallback preserves fab-ff's semi-interactive character — the user is never locked out of control.
**Rejected**: Keep fab-ff fully interactive (wastes the fresh-context benefit on simple fixes), make fab-ff fully autonomous like fab-fff (loses the semi-interactive identity).
*Introduced by*: 260216-gqpp-DEV-1040-code-review-loop

### Review Failure Offers Multiple Re-Entry Points
**Decision**: On review failure, the agent presents three options (fix code, revise tasks, revise spec) and the user chooses where to loop back.
**Why**: Not all review failures are implementation bugs. Some require revisiting upstream artifacts. Giving the user explicit choice prevents the agent from guessing wrong about where the problem originated.
**Rejected**: Always looping back to apply — misses cases where the spec was wrong.
*Source*: doc/fab-spec/SKILLS.md

### Hydrate Semantically, Not by Delta Markers
**Decision**: The agent compares `spec.md` against existing memory files to determine what's new, changed, or removed. No ADDED/MODIFIED/REMOVED markers in the spec.
**Why**: The spec reads as a straightforward requirements document. Delta markers would clutter the spec and couple it to the hydration mechanism.
**Rejected**: Explicit delta markers — clutters specs, requires discipline to maintain, fragile to editing.
*Source*: doc/fab-spec/TEMPLATES.md

### Concurrent Change Warning on Hydrate
**Decision**: Before hydrating, scan for other active changes that reference the same memory files and warn the user.
**Why**: Hydration updates the memory files, which may invalidate assumptions in other active changes. The warning prompts re-review rather than allowing silent drift.
**Rejected**: Blocking hydrate if concurrent changes exist — too restrictive, especially for independent changes that happen to touch the same domain.
*Source*: doc/fab-spec/SKILLS.md

### Hydrate is a Pipeline Stage, Archive is Not
**Decision**: Memory hydration (`hydrate`) is a tracked pipeline stage; folder housekeeping (`/fab-archive`) is a standalone skill.
**Why**: Memory hydration is the logical completion of the agent's work — it closes the feedback loop from implementation back to memory files. Folder housekeeping is a user-triggered cleanup action with no bearing on artifact quality.
**Rejected**: Both as pipeline stages — would add a 7th stage for marginal benefit. Neither as pipeline stages — would lose the memory hydration automation.
*Introduced by*: 260213-jc0u-split-archive-hydrate

### Restore as Subcommand, Not Separate Skill
**Decision**: Archive restore is a subcommand of `/fab-archive` (`/fab-archive restore <name>`), not a separate `/fab-restore` skill.
**Why**: Archive and restore are paired inverse operations. Grouping them under the same skill maintains conceptual cohesion and avoids skill proliferation. Users naturally look for restore under the archive command.
**Rejected**: Separate `/fab-restore` skill — adds a new top-level command for a narrow, complementary operation.
*Introduced by*: 260214-v7k3-archive-restore-mode

### fab-archive Clears Pointer Conditionally
**Decision**: `/fab-archive` only removes `.fab-status.yaml` when the archived change is the active one.
**Why**: If archiving a non-active change (via change-name argument), clearing the pointer would disrupt the user's active work context.
**Rejected**: Always clear — would lose active change context when archiving a different change. Never clear — would leave stale pointer after archiving the active change.
*Introduced by*: 260213-jc0u-split-archive-hydrate

### Unified PR Template with Conditional Field Population
**Decision**: `/git-pr` uses a single unified PR body template for all PR types. Fab-linked fields (Stats table columns, Changes section, Pipeline progress line) are conditionally populated based on artifact availability — whether `changeman.sh resolve` succeeds and artifacts exist — not based on PR type. The Stats table has five columns (Type, Confidence, Checklist, Tasks, Review) with `—` for unavailable fields. A pipeline progress line shows completed stages with intake/spec as hyperlinks.
**Why**: A `test` or `docs` change that went through the full fab pipeline deserves the same quality signals as a `feat` change. Gating template richness on type hid real planning work and reduced reviewer confidence. The unified template always shows the same structure, reducing cognitive overhead.
**Rejected**: Two-tier templates gated on type (hides work for non-feat types), keep type-gating but extend Tier 1 to all types (still requires two code paths), omit columns when empty (inconsistent table shape across PRs).
*Introduced by*: 260305-b0xs-unified-pr-template
*Supersedes*: 260225-54vl-smart-git-pr-category-taxonomy (Two-Tier PR Templates with Type Resolution)

### Execution Stage Reset Preserves Task Checkboxes
**Decision**: `/fab-continue apply` re-runs apply behavior starting from the first unchecked task. It does NOT uncheck all tasks.
**Why**: Task checkboxes reflect actual implementation progress. Silently unchecking them would discard valid work. Review rework (Option 1: "Fix code") handles targeted unchecking with `<!-- rework: reason -->` annotations.
**Rejected**: Resetting all checkboxes on apply reset — too destructive, discards completed work.
*Introduced by*: 260212-a4bd-unify-fab-continue

### Review Active Triggers Forward Progression
**Decision**: When the active stage is `review`, `/fab-continue` runs the review behavior (advancing toward hydrate), not re-review. Re-review is available via `/fab-continue review` reset.
**Why**: The normal flow always advances. `review: active` means "review needs to run"; `review: done` means "review passed, hydrate is next." This avoids ambiguity about whether the command should redo or advance.
**Rejected**: Having review active trigger re-review — conflicts with the forward-progression model.
*Introduced by*: 260212-a4bd-unify-fab-continue

### Generic Review Skill Replaces Copilot-Specific Fix
**Decision**: `/git-pr-review` (renamed from `/git-review` to distinguish from the internal `review` stage) is a standalone skill that handles all PR review types (human, Copilot, other bots), replacing `/git-pr-fix` (Copilot-only) and `/git-pr` Step 6 (inline auto-fix). Human reviews take priority — if any reviewer has commented, Copilot is not requested. It drives the `review-pr` pipeline stage with statusman integration.
**Why**: The old architecture had two problems: `/git-pr` had Copilot-specific workflow bolted on (Step 6), and `/git-pr-fix` only handled Copilot reviews. Human review comments required fully manual handling. The new skill consolidates all review handling with a priority-based routing: existing reviews first, Copilot as fallback.
**Rejected**: Extending `/git-pr-fix` to support humans (name implies Copilot-only, awkward evolution), keeping Copilot logic inline in `/git-pr` (bloats shipping skill), separate skills per reviewer type (duplication, same API shape).
*Introduced by*: 260303-i58g-extract-pr-review-skill

### All-Auto-Answer Model
**Decision**: All detected questions are auto-answered unless the operator cannot determine what keystrokes to send. A numbered decision list (items 1-6, evaluated in priority order) provides a consistent, deterministic answer selection process.
**Why**: Worktree isolation and human PR merge provide the safety gate. The operator should not be a bottleneck. A priority-ordered list ensures consistent processing of answer patterns.
**Rejected**: Two-tier auto-answer/escalate classification — adds complexity without meaningful safety improvement given the PR safety net. Prose heuristic — harder to reason about edge cases and priority conflicts.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge

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
**Decision**: `/fab-operator` (v7) adds pre-spawn dependency resolution to the operator. When spawning an agent for a change with `depends_on` entries, the operator cherry-picks dependency content into the worktree before opening the agent tab. Uses `git cherry-pick --no-commit origin/main..<dep-branch> && git commit -m "operator: cherry-pick <dep> dependency"`. On conflict: abort, escalate, do not spawn.
**Why**: Without dependency awareness, agents working on dependent changes start from a baseline missing the dependency code, causing build failures, spec divergence, and manual intervention. This defeats the operator's "automate the routine" principle.
**Rejected**: `git merge --squash` — rejected for unattended sessions where merge machinery introduces risk. Transitive dependency resolution — rejected because leaf dependency branches already carry transitive content via the operator's own cherry-picking when those deps were spawned; `origin/main..<dep-branch>` gives the complete transitive closure.
*Introduced by*: 260324-prtv-operator7-dep-aware-spawning

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
**Decision**: The autopilot queue defaults to **stack-then-review** mode. All queued changes after the first implicitly get `depends_on: [<prev-change-id>]` (equivalent to implicit `--base` chaining). PRs are created but not merged until the user reviews and explicitly requests merging. The previous merge-as-you-go behavior is preserved via `--merge-on-complete` opt-in flag. Queue completion produces a summary with all PR links and suggested merge order (base-first). Ordered merge waits for CI on each PR before proceeding to next.
**Why**: The previous merge-as-you-go default caused two problems: (1) rebase conflicts when rebasing dependent changes onto freshly-merged `origin/main` re-linearized commits that cherry-pick resolution had already handled, and (2) no opportunity for holistic review of the full change set before any code merged to `main`. Stack-then-review gives the user full review control over the entire queue.
**Rejected**: Keeping merge-as-you-go as default — too many rebase conflicts and no review control. Available as opt-in for users who want it.
*Introduced by*: 260327-gwg9-operator-base-chaining-default

### Standardized Tmux Tab Naming (operator7)
**Decision**: All agent tab names in `/fab-operator` use `⚡<wt>` format (zap emoji + worktree name, no space). Replaces the previous `fab-<id>` naming which was unreliable for new changes where the change ID doesn't exist at spawn time. The worktree name is always available at spawn time and unique across panes, making it a consistent identifier for all three spawn paths (existing change, raw text, backlog).
**Why**: The `fab-<id>` format had two issues: (1) for new changes, the ID doesn't exist until `fab-new` runs inside the spawned agent, and (2) the raw-text path already used `fab-<wt>` as a workaround, creating inconsistency. The `⚡` prefix makes agent tabs visually distinct from other tmux windows.
**Rejected**: Keeping `fab-<id>` with worktree fallback — adds conditional logic without benefit since worktree name is always available.
*Introduced by*: 260328-iqt8-standardize-tmux-tab-naming

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260328-iqt8-standardize-tmux-tab-naming | 2026-03-28 | Standardized tmux tab naming in `/fab-operator`: all four `tmux new-window -n` invocations now use `⚡<wt>` (zap emoji + worktree name) instead of `fab-<id>` or `fab-<wt>`. Resolves inconsistency where change ID wasn't available at spawn time for new changes. |
| 260327-gwg9-operator-base-chaining-default | 2026-03-27 | Changed `/fab-operator` autopilot default from merge-as-you-go to **stack-then-review**: queued changes get implicit `depends_on` chaining (implicit `--base`), PRs created but not merged until user reviews. Queue completion summary with ordered merge support. Previous behavior preserved via `--merge-on-complete` opt-in. Confirmation prompt updated. Failure matrix updated: cherry-pick conflict replaces rebase conflict in default mode. |
| 260326-oxgu-unified-tick-status-list | 2026-03-26 | Updated `/fab-operator` tick status frame: replaced two-block layout (changes + 👁 watches) with a single unified list. Every entry gets `[type]` prefix (`[change]`/`[watch]`), consistent column layout (Type, ID, Autopilot, Health, Detail). Autopilot moved from header (`autopilot 1/3`) to per-entry `▶` symbol. Watches gain health emojis (🟢 healthy, 🟡 new items, 🔴 errored, ⏸ paused) matching change emoji column. Watch timestamps use relative format (`3m ago`). Header simplified to `N tracked` total count. |
| 260326-13ro-operator7-direct-fab-new-spawn | 2026-03-26 | Updated operator "From raw text" spawn path: removed `idea add` intermediate step, operator now passes description directly to `/fab-new`. Spawn sequence aligned with structured flow (worktree → deps → spawn → enroll → completion). Explanatory paragraph updated to attribute traceability to `/fab-new` Origin section instead of `idea add`. |
| 260326-u3un-operator-enforce-pipeline-routing | 2026-03-26 | Added "Pipeline-first routing" principle to `/fab-operator` §1: operator MUST route all new work through `/fab-new` then pipeline commands, MUST NOT dispatch raw inline implementation instructions to agent panes, MUST NOT use `/fab-continue` to skip intake. Operational maintenance exempt. Reinforcing blockquote added to §6 "Working a Change" referencing the §1 principle. |
| 260324-prtv-operator7-dep-aware-spawning | 2026-03-24 | New `/fab-operator` (v7) skill — operator6 plus dependency-aware agent spawning. Pre-spawn step cherry-picks dependency content into worktrees via `git cherry-pick --no-commit origin/main..<dep-branch>`. Schema additions: `depends_on`, `branch` on monitored entries; `branch_map` top-level for post-removal persistence. Redundant dep pruning via ancestor check. `--base` implies `depends_on`. Cherry-pick conflict: abort, escalate, do not spawn. Idle message now includes timestamp (`Time: HH:MM · next tick: HH:MM`). New launcher `fab-operator.sh`. |
| 260320-t13m-configurable-agent-spawn-command | 2026-03-20 | Updated operator spawn documentation: operator4 and operator5 launcher descriptions now reference configurable spawn command from `config.yaml` `agent.spawn_command` (via `lib/spawn.sh`) instead of hardcoded `claude --dangerously-skip-permissions`. |
| 260320-tm9h-draft-prs-by-default | 2026-03-20 | `/git-pr` now creates all PRs as drafts via `gh pr create --draft`. Unconditional — no configuration toggle. Both primary and fallback (`--fill`) paths include `--draft`. Updated "PR shipping" overview paragraph. |
| 260318-dzze-standard-subagent-context | 2026-03-18 | Review sub-agent context now references `_preamble.md` § Standard Subagent Context instead of listing `fab/project/**` files ad-hoc. Added Standard Subagent Context Template design decision. All subagents (including nested sub-subagents) inherit project principles via the centralized template. |
| 260317-yrgo-operator5-branch-fallback | 2026-03-17 | Added `/fab-operator5` as operator4's successor with three new capabilities: (1) use case registry — toggleable named concerns (`monitor-changes`, `linear-inbox`, `pr-freshness`) persisted in `.fab-operator.yaml`, with conversational toggling and tick-start status roster; (2) branch fallback resolution — scans local/remote branch names when `fab resolve` fails (user-initiated only), with read-only `git show` path and action worktree-creation path; (3) tab preparation procedure — shared pre-dispatch sequence (verify pane, check idle, check active change, check branch alignment) used by all playbooks and use cases. Operator4's "Modes of Operation" renamed to "Playbooks". Deleted legacy `fab-operator{1,2,3}.sh` launcher scripts. |
| 260315-a2b2-standalone-operator4-rewrite | 2026-03-15 | Rewrote operator documentation: replaced operator1/2/3/4 inheritance chain with standalone `/fab-operator4`. Removed all references to operator1, operator2, operator3 as separate skills. Operator4 is now documented as the single operator skill with all behavior inlined (principles, startup, safety model, monitoring, auto-nudge, modes of operation, autopilot, configuration). Loads `_cli-external.md` (operator-only) for wt/tmux/loop references. Consolidated design decisions (merged "All-Auto-Answer" + "Decision List" into single DD, renamed "operator4"-specific DDs to generic names). Added "Standalone Operator Over Inheritance Chain" design decision. |
| 260314-q5p9-redesign-ff-fff-scopes | 2026-03-14 | Updated pipeline invocation overview: `/fab-ff` now runs intake → hydrate (6 stages), `/fab-fff` runs intake → review-pr (8 stages). Both have identical confidence gates and autonomous rework. Updated review failure rework descriptions: both `/fab-ff` and `/fab-fff` now use identical auto-loop + stop behavior (was: `/fab-ff` had interactive fallback, `/fab-fff` had bail). |
| 260314-007n-redesign-operator-auto-nudge | 2026-03-14 | Added `/fab-operator4` standalone coordination skill — workflow iteration of operator3 with redesigned auto-nudge. Replaces two-tier confidence model with all-auto-answer (decision list items 1-6). Improved question detection: `-l 20` capture window, Claude turn boundary guard (`>` cursor in last 2 lines), tightened `?` pattern (last non-empty line only, <120 chars, skip comment prefixes), new indicator patterns (`:` endings, enumerated options, `Press.*key`), blank capture guard. Re-capture before send eliminates detection-to-send race condition. Per-answer inline logging for debugging reconstruction. Routing discipline (operator must route instructions to agents, never execute directly). Autopilot uses `/fab-fff` instead of `/fab-ff`. ~26% token reduction via delta-only tick description, reduced key properties table, condensed purpose, integrated guards. New launcher `fab-operator4.sh`. New spec `SPEC-fab-operator4.md`. Updated "one operator at a time" constraint to include operator4. |
| 260312-9r3t-pr-change-metadata | 2026-03-12 | `/git-pr` gains "Change" section in PR body (above Stats), conditional on `{has_fab}`. Three-column table (ID, Name, Issue) shows change identity and linked Linear issues. Issue IDs rendered as hyperlinks when `linear_workspace` is configured in `fab/project/config.yaml`, bare text otherwise. Missing fields show `—`. Section omitted entirely when no active fab change. |
| 260312-ngew-add-operator3-auto-nudge | 2026-03-12 | Added `/fab-operator3` standalone coordination skill — workflow iteration of operator2 with auto-nudge capability. Detects input-waiting agents via terminal heuristic (`tmux capture-pane -t <pane> -p -l 10`), classifies questions via two-tier confidence model (auto-answer vs escalate), and adds input-waiting detection as step 5 in the monitoring tick (before stuck detection). No cooldown on auto-answers. Hardcoded question patterns for v1. New launcher `fab-operator3.sh`. Updated "one operator at a time" constraint to include operator3. |
| 260312-kvng-resolve-pane-evolve-panemap | 2026-03-12 | Updated operator skill references: replaced all `fab send-keys` invocations with `fab resolve --pane` + `tmux send-keys` pattern. Updated pre-send validation, interaction primitive, autopilot per-change loop, and operator2 monitor-after-action sections. `fab send-keys` CLI subcommand removed — `resolve --pane` provides pane lookup, raw `tmux send-keys` provides delivery. |
| 260312-wrk6-add-wt-create-base-flag | 2026-03-12 | Updated operator autopilot per-change loop: `wt create` now uses `--base <previous-change-folder-name>` for user-provided ordering so dependent changes branch from the previous change's tip instead of HEAD. Confidence-based ordering omits `--base` (independent changes). |
| 260311-5c11-add-operator2-monitoring-skill | 2026-03-11 | Added `/fab-operator2` standalone coordination skill — workflow iteration of operator1 with proactive monitoring. After every send-keys action, enrolls target change in a monitoring set and runs `/loop` (default 5m) to detect stage advances, completions, failures, pane deaths, and stuck agents. Monitoring-enhanced UC1 (auto-enroll after broadcast), UC2 (loop-driven sequenced rebase), UC6 (monitor recovery after nudge), UC7 (immediate enrollment instead of next-interaction check). Advisory-only stuck detection (15m threshold). Conversation-held monitored set, not file-backed. New launcher `fab-operator2.sh`; renamed `fab-operator.sh` to `fab-operator1.sh` with shared `operator` tab name. |
| 260312-9lci-fix-status-show-fab-current | 2026-03-12 | Replaced `fab status show --all` outside-tmux fallback with `wt list` + `fab change list` in `/fab-operator1` and `/fab-operator2` descriptions. The `show` subcommand has been removed from the Go binary. |
| 260311-ftrh-drop-runtime-idle-from-operator | 2026-03-11 | Removed all `fab runtime is-idle` references from `/fab-operator1` description. The operator now uses the pane-map Agent column exclusively for idle detection — `fab runtime is-idle` reads the wrong worktree's `.fab-runtime.yaml` when called from the operator's pane. Updated state re-derivation, UC6 (unstick), pre-send validation, autopilot per-change loop, and two prior changelog entries (b8ff, qkov). |
| 260310-1ttn-operator-autopilot-uc8 | 2026-03-11 | Added UC8 (Autopilot) to `/fab-operator1`: drives a queue of changes through the full pipeline with per-change spawn, gate check, monitoring, merge, and rebase-next loop. Three ordering strategies (user-provided, confidence-based, hybrid). Failure matrix with 6 failure types. Interruptibility (stop/skip/pause/resume). Session-resumable via `fab pane-map`. Queue state held in conversation context (v1). "Seven Use Cases" heading renamed to "Use Cases". Confirmation model updated to include autopilot as destructive. |
| 260310-b8ff-operator-observation-fixes | 2026-03-10 | Updated `/fab-operator1` observation model: pane-map is now the sole primary observation mechanism (session-scoped via `-s`, 6 columns: Pane, Tab, Worktree, Change, Stage, Agent). `fab status show --all` retained only as outside-tmux fallback. State re-derivation uses `fab pane-map` (replacing `fab status show --all`). Pre-send validation uses the Agent column in the pane map. |
| 260307-8ggm-git-pr-ship-finish-ordering | 2026-03-07 | Fixed git-pr post-PR step ordering: reordered as 4a (record PR URL) → 4b (finish ship stage) → 4c (commit+push .status.yaml and .history.jsonl) → 4d (write .pr-done sentinel). All status mutations now occur before the commit boundary, preventing uncommitted fab state files in the working tree after PR creation. Steps renumbered from 4/4b/4c/4d to 4a/4b/4c/4d. |
| 260306-qkov-operator1-skill | 2026-03-07 | Added `/fab-operator1` standalone coordination skill: user-driven Claude session for cross-agent coordination (not a pipeline stage, not a lifecycle enforcer). Seven use cases (broadcast, sequenced rebase, merge PRs, spawn worktree, status dashboard, unstick agent, notification surface). Three-tier confirmation model. Pre-send validation via pane-map Agent column. Bounded retries with escalation. Context discipline — loads always-load layer only, never change artifacts. Uses `fab resolve --pane` + `tmux send-keys` for agent interaction. |
| 260306-6bba-redesign-hooks-strategy | 2026-03-06 | Added hook-backed bookkeeping note: PostToolUse hook (`on-artifact-write.sh`) supplements skill-instructed checklist bookkeeping as a reliability layer. Skills keep instructions unchanged for agent-agnostic portability; hooks catch what the agent forgets. All commands idempotent. |
| 260305-u8t9-clean-break-go-only | 2026-03-05 | Updated status mutations overview: replaced `fab/.kit/scripts/lib/statusman.sh` reference with `fab/.kit/bin/fab status` CLI. Shell scripts removed — all status mutations now go through Go binary via `fab status` commands. |
| 260305-id4j-review-pr-timeout-done | 2026-03-05 | `/git-pr-review` Copilot polling window increased from 12 attempts / 6 minutes to 16 attempts / 8 minutes. Copilot timeout now results in `finish` (done) instead of `fail` (failed) — absence of external review is a graceful no-op, matching the existing "Copilot unavailable" behavior. Step 6 routing updated: failure case limited to "no PR found" and "processing error". `fab-ff.md` Step 9 and `fab-fff.md` Step 10 updated to reflect new timeout and routing. |
| 260305-b0xs-unified-pr-template | 2026-03-05 | Replaced two-tier PR template system (Tier 1 fab-linked / Tier 2 lightweight) with a single unified template. Template fields conditionally populated based on artifact availability, not PR type. New horizontal Stats table (Type/Confidence/Checklist/Tasks/Review) with `—` for unavailable fields. Pipeline progress line shows completed stages with intake/spec as hyperlinks. Title derivation simplified: intake heading when available, commit subject otherwise, regardless of type. "Fab Pipeline?" and "Template Tier" columns removed from PR Type Reference table. |
| 260303-he6t-extend-pipeline-through-pr | 2026-03-05 | Extended pipeline from 6 to 8 stages: added `ship` (driven by `/git-pr`) and `review-pr` (driven by `/git-pr-review`) after hydrate. `/git-review` renamed to `/git-pr-review`. Both `/fab-ff` and `/fab-fff` extended through ship and review-pr. `/git-pr` gains statusman integration for ship stage. `/git-pr-review` gains statusman integration for review-pr stage with phase sub-state tracking. `review-pr` supports `failed` state; `ship` does not. No confidence gates for integration stages. Backward compatible — statusman tolerates missing stages in old 6-stage changes. |
| 260303-i58g-extract-pr-review-skill | 2026-03-04 | New `/git-pr-review` skill replaces `/git-pr-fix` and `/git-pr` Step 6. Handles all reviewer types (human, Copilot, bots) with priority-based routing: existing reviews processed first, Copilot requested as fallback only when no reviews with comments exist. Reviewer-aware commit messages. Path A (all comments across reviewers) uses `--paginate`. `/git-pr` trimmed to end at Step 5 ("Shipped."), Step 6 and Rules reference removed. `git-pr-fix` skill deleted. |
| 260303-6b7c-update-underscore-skill-references | 2026-03-04 | Standardized top-of-file `_preamble.md` references in execution skill files — removed `./` prefix. No content changes to execution-skills requirements — all references already used correct form. |
| 260303-n30u-smart-copilot-review-detection | 2026-03-03 | `/git-pr-fix` Step 2 rewritten: blind polling replaced with 3-phase detection — (1) check existing reviews, (2) POST `/requested_reviewers` to detect availability (any non-2xx = not available, skip entirely), (3) mode-specific poll. Standalone mode: Phase 3 single-check then bail. Wait mode: Phase 3 polls 30s×12. Eliminates 6-minute waste on repos without Copilot. Login name discrepancy documented inline (`"Copilot"` in requested_reviewers vs `"copilot-pull-request-reviewer[bot]"` in reviews). `/git-pr` Step 6 updated to reference new 3-phase flow. |
| 260303-4ojc-git-pr-copilot-fix | 2026-03-03 | New `/git-pr-fix` standalone skill: waits for `copilot-pull-request-reviewer[bot]` review, triages comments (actionable vs informational), auto-fixes actionable ones, commits and pushes. Two detection modes: first-poll bail (standalone) and wait mode with 30s×12 polling (inline from git-pr). `/git-pr` gains Step 6: best-effort auto-invoke of `/git-pr-fix` behavior after "Shipped." output. Rules updated: "Step 6 (Copilot fix) is best-effort — never blocks shipping." |
| 260302-9fnn-extract-logman-from-preflight | 2026-03-02 | Command logging decoupled from preflight: skills now call `logman.sh command` directly. Preflight-calling skills log via `_preamble.md` §2 step 4 (after YAML parsing); exempt skills (`/fab-switch`, `/fab-setup`, `/fab-discuss`, `/fab-help`) log via per-skill instructions; `/fab-new` logging handled by `changeman.sh new` internally. `logman.sh command` signature flipped to `<cmd> [change] [args]` with optional change resolution via `fab/current` and silent exit 0 on failure. |
| 260228-wyhd-add-skipped-stage-state | 2026-02-28 | `statusman.sh` gains `skip` event command (`{pending,active} → skipped` with forward cascade). Skipped stages treated as resolved for progression. `get_current_stage`, `get_display_stage`, `get_progress_line`, and `_apply_metrics_side_effect` updated to handle `skipped`. Six event commands total: `start`, `advance`, `finish`, `reset`, `fail`, `skip`. |
| 260227-czs0-status-indicative-confidence | 2026-02-27 | `/fab-status` gains stage-aware confidence display: indicative at intake (computed on the fly via `calc-score.sh --check-gate --stage intake`), persisted at spec+, fallback when no data. `calc-score.sh --check-gate` now emits count fields (`certain`, `confident`, `tentative`, `unresolved`) in both intake and spec branches, with `|| echo "0"` guards for robustness under `set -euo pipefail`. |
| 260227-8q33-richer-git-pr-output | 2026-02-27 | `/git-pr` Tier 1 Context table expanded: added Confidence (`{score} / 5.0`) and Pipeline (`done` stages joined with `→`) rows. Fixed Intake/Spec from sibling cells in one row to separate `Field \| Detail` rows. Spec row omitted (not empty) when `spec.md` absent. Reads `confidence.score` and `progress` from `.status.yaml`. Tier 2 unchanged. |
| 260226-6boq-event-driven-statusman | 2026-02-26 | Status mutations updated: replaced `transition`/`set-state` with event commands (`start`, `advance`, `finish`, `reset`, `fail`). Driver parameter now optional (skills always pass it). |
| 260226-3g6f-git-branch-non-interactive-rename | 2026-02-26 | `/git-branch` non-interactive: replaced 3-option interactive menu with deterministic upstream-tracking logic. Local-only branches renamed (`git branch -m`); tracked branches get new branch (`git checkout -b`). Removed "Adopt this branch" concept. Added `renamed from {old}` and `created, leaving {old} intact` report verbs. |
| 260226-i9av-add-ready-state-to-stages | 2026-02-26 | `fab-status.md` updated progress symbols: replaced `—` skipped with `◷` ready. `/fab-ff` review failure behavior changed from interactive fallback to stop after 3 cycles. |
| 260227-gasp-consolidate-status-field-naming | 2026-02-27 | `/git-pr` reads issues via `statusman.sh get-issues` (replaces direct `issue_id` read), uses `statusman.sh add-pr` (replaces `ship`), PR title supports multiple space-joined issues, commit message updated, `.shipped` sentinel → `.pr-done`. `/fab-new` uses `statusman.sh add-issue` (replaces raw `yq`). `/fab-archive` cleans `.pr-done` instead of `.shipped`. |
| 260226-tnr8-coverage-scoring-change-types | 2026-02-26 | `/git-pr` type resolution chain extended from 3 to 4 steps: explicit argument → read `change_type` from `.status.yaml` → infer from intake → infer from diff. Step 2 (new) reads `change_type` written by `/fab-new` at creation time, avoiding re-inference. Resolution source tracking updated to include `status`. |
| 260225-54vl-smart-git-pr-category-taxonomy | 2026-02-25 | `/git-pr` gains PR type system: 7 types (feat, fix, refactor, docs, test, ci, chore) from Conventional Commits. New Step 0 resolves type via three-step chain (explicit arg → intake keyword inference → diff path inference). Step 3c rewritten with two-tier templates: Tier 1 (fab-linked) shows summary/changes/context with blob URL links; Tier 2 (lightweight) shows auto-generated summary with "no design artifacts" note. PR titles prefixed with `{type}: `. Blob URLs resolve against feature branch instead of main, fixing broken relative links. PR Type Reference table added. |
| 260224-1jkh-smart-resolve-and-pr-summary | 2026-02-25 | `changeman.sh resolve` gains single-change guessing fallback when `fab/current` is missing/empty (enumerate valid change folders, return if exactly one). `/git-pr` Step 3c enhanced with intake-aware PR generation: resolves active change, reads `intake.md`, derives PR title from H1 (strips `Intake: ` prefix), generates body with Summary/Changes/Context sections and links to intake + optional spec. Falls back to `gh pr create --fill` when resolution fails or no intake exists. |
| 260224-vx4k-decouple-git-from-fab-switch | 2026-02-24 | `/git-pr` enhanced with Step 1b (branch mismatch nudge — non-blocking note when current branch doesn't match active change, using prefix-stripped comparison) and enhanced Step 2 branch guard (suggests `/git-branch` when on main with active change). New `/git-branch` command documented in change-lifecycle.md. |
| 260222-trdc-git-pr-shipped-sentinel-and-status-commit | 2026-02-22 | `/git-pr` now performs a second commit+push after recording shipped URL to `.status.yaml` (Step 4b), then writes a `.shipped` sentinel file as its final action (Step 4c). Step 4b guards for "nothing to commit" via `git diff --cached --quiet`. The "already shipped" early-exit path now includes Steps 4–4c. Sentinel is gitignored, written unconditionally. Pipeline orchestrator (`run.sh`) polls sentinel existence instead of `statusman is-shipped`. |
| 260222-s90r-add-shipped-tracking | 2026-02-22 | Updated `/git-pr` skill to record PR URLs via `statusman.sh ship` after PR creation (graceful degradation when no active change). Updated `_preamble.md` state table: hydrate row now routes to `/git-pr` as default, `/fab-archive` as alternative. Updated `changeman.sh` `default_command` for hydrate from `/fab-archive` to `/git-pr`. |
| 260221-5tj7-rename-context-to-preamble | 2026-02-21 | Renamed shared skill preamble from `_context.md` to `_preamble.md`. Updated references in status mutations overview and sub-agent context sections. |
| 260217-eywl-fix-statusman-skill-path-refs | 2026-02-18 | All `lib/statusman.sh` references in skill files updated to repo-root-relative `fab/.kit/scripts/lib/statusman.sh`. Also fixed 2 short-form `lib/preflight.sh` references in `_preamble.md` and `fab-status.md`. Memory file references updated to match. |
| 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions | 2026-02-16 | All `Next:` lines in execution skills (`/fab-continue`, `/fab-archive`) now derived from canonical state table in `_preamble.md`. Removed hardcoded suggestions from review pass verdict and archive mode output. `/fab-archive` restore without `--switch` now uses activation preamble before state-derived commands. |
| 260216-gqpp-DEV-1040-code-review-loop | 2026-02-16 | Review dispatched to sub-agent in separate execution context for all three pipeline skills. Structured findings with three-tier priority (must-fix / should-fix / nice-to-have). `/fab-ff` gains auto-loop (3 cycles) with interactive fallback; `/fab-fff` retains auto-loop with bail. Comment triage by applying agent. Escalation rule after 2 consecutive fix-code. |
| 260216-knmw-DEV-1030-swap-ff-fff-review-rework | 2026-02-16 | Swapped pipeline invocation note: `/fab-ff` now presents interactive rework on review failure; `/fab-fff` now uses autonomous rework with bounded retry (3-cycle cap, escalation after 2 consecutive fix-code) |
| 260215-237b-DEV-1027-redefine-ff-fff-scope | 2026-02-16 | Updated pipeline invocation note: `/fab-fff` now presents interactive rework on review failure, `/fab-ff` now bails immediately (swapped from previous behavior) |
| 260215-v4n7-DEV-1025-rename-brief-to-intake | 2026-02-15 | Renamed `brief` stage/artifact to `intake` throughout — stage identifiers, artifact filenames, YAML keys, prose references |
| 260215-r8k3-DEV-1024-code-quality-layer | 2026-02-15 | Added Pattern Extraction to Apply (naming, error handling, structure, utilities), expanded per-task guidance to 7-step sequence, added code quality check as Review step 6, added optional pattern capture to Hydrate step 5, updated "Checklist Tests Implementation Fidelity" design decision to include code quality |
| 260214-r7k3-statusman-yq-metrics | 2026-02-14 | Added `driver` parameter requirement to status mutations overview. Added `log-review` calls to review pass/fail behavior. Stage metrics side-effects documented as automatic |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_statusman.sh` → `lib/statusman.sh` in status mutations overview |
| 260214-w3r8-statusman-write-api | 2026-02-14 | All execution-stage `.status.yaml` transitions now use `_statusman.sh` CLI commands instead of direct file edits |
| 260214-eikh-consistency-fixes | 2026-02-14 | Verified cf13 (contradictory fab-status.sh/statusman.sh changelog entries) — already resolved by prior changes. No behavioral modifications. |
| 260214-r8kv-docs-skills-housekeeping | 2026-02-14 | Removed `fab-status.sh` references from changelog entries (updated to reference `/fab-status` skill instead) |
| 260214-v7k3-archive-restore-mode | 2026-02-14 | Added restore mode to `/fab-archive` — moves archived changes back to active, removes index entry, optional `--switch` flag. Idempotent and resumable. Added Restore as Subcommand design decision. |
| 260213-jc0u-split-archive-hydrate | 2026-02-13 | Replaced Archive Behavior with Hydrate Behavior (steps 1-4 only, change folder stays). Added `/fab-archive` as standalone housekeeping skill. Updated overview, design decisions. |
| 260213-w4k9-explicit-change-targeting | 2026-02-13 | Execution skills now inherit optional `[change-name]` argument via `/fab-continue` preflight override; `/fab-status` also accepts change-name override directly |
| 260212-a4bd-unify-fab-continue | 2026-02-12 | Restructured: apply, review, and archive behavior now accessed via `/fab-continue` instead of standalone skills. Updated all section headings, requirements, and cross-references |
| 260212-ipoe-checklist-folder-location | 2026-02-12 | Updated checklist path references from `checklists/quality.md` to `checklist.md` in `/fab-review` and `/fab-archive` |
| 260212-bk1n-rework-fab-ff-archive | 2026-02-12 | Added note that `/fab-ff` and `/fab-fff` invoke execution skills internally as part of their full-pipeline behavior |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated stage references from proposal/specs to intake/spec |
| 260211-endg-add-created-by-field | 2026-02-11 | `/fab-status` now displays `Created by:` line when `created_by` field is present in `.status.yaml` |
| 260210-7wxx-add-specs-index-context-loading | 2026-02-10 | Added `docs/specs/index.md` to context loading for all three execution skills, aligning with the always-load protocol in `_preamble.md` |
| 260209-r4w8-archive-index-longer-slugs | 2026-02-09 | Added archive index maintenance step to `/fab-archive` — creates/updates `fab/changes/archive/index.md` with searchable change summaries |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Removed auto-guess soft gate from `/fab-apply` — replaced by confidence gating on `/fab-fff` |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added auto-guess soft gate to `/fab-apply` (subsequently removed by 260208-k3m7-add-fab-fff) |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| — | 2026-02-07 | Generated from doc/fab-spec/ (SKILLS.md, TEMPLATES.md) |
