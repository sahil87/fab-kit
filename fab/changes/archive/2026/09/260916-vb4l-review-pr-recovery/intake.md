# Intake: Review-PR Recovery — Terminalize a Stuck Review-PR Stage, One Bounded Recovery Shape for the Operator

**Change**: 260916-vb4l-review-pr-recovery
**Created**: 2026-09-16

## Origin

Synthesized from a user conversation and dispatched promptless (`/fab-proceed` create-new
dispatch, `{questioning-mode} = promptless-defer`). No questions were asked at intake; every
decision below was made in the conversation and is captured here verbatim where it had specific
values. Interaction mode: conversational diagnosis + design, then a one-shot dispatch.

**Resolves two backlog items** (`fab/backlog.md`):

- `[vb4l]` (2026-09-16) — review gate degraded/unavailable: 5 PRs across two plans stuck at
  `review-pr` when GitHub silently dropped every Copilot review request; the user hand-instructed a
  rebase-wait-merge loop per PR because the skill had no built-in fallback.
- `[hf6x]` (2026-08-29) — on a review-pr-stage CI failure the operator halts and escalates
  immediately with no "send it back to the authoring pane as a fix request" step, even for
  addressable failures (case: run-kit PR #759, CI run 33188652965, E2E shard 4/4 — once asked, the
  agent found the root cause in under a minute).

This change carries `vb4l` as its change ID, so `/fab-archive`'s exact-ID backlog mark handles that
entry; `[hf6x]` is marked done by hand at the same archive step (see § What Changes → Backlog).

> Review-PR Recovery — terminalize a stuck review-pr stage and give the operator one bounded
> recovery shape (resolves backlog items vb4l and hf6x). Treat "review-pr is stuck" as ONE
> condition with a reason, give each reason ONE bounded recovery action, and guarantee the stage
> always ends in a terminal state. Nothing new is invented — the three actions plug into existing
> steps.

### Agreed diagnosis (verified against the current skill text)

The `review-pr` stage has three ways to get stuck, and today each dead-ends differently:

1. **Reviewer never shows up.** `src/kit/skills/git-pr-review.md` Step 2 Phase 2 requests a
   Copilot review and polls 30 s × 20 (10 minutes). On exhaustion it exits through Step 6 with
   outcome **timeout** — "Leave `review-pr` `active`: no finish/fail; preserve the explicit-change
   re-run guidance". `/fab-fff` Step 5 then reports `Review-PR pending (Copilot review requested,
   timed out waiting) — re-run /git-pr-review {name} when ready` and stops. The operator's built-in
   done predicate for a tracked `pane` item with a change is "`review-pr` done/skipped" (§4 Tracked
   Items → Kinds), so a dead review gate makes every such item unreachable by construction — nothing
   ever finishes or fails the stage, and nothing re-runs the skill.
2. **CI fails on a PR during Ordered Merge / an armed Auto-Merge.** `fab-operator.md` §6 → Ordered
   Merge → "CI failure during ordered merge (halt-dependents-only)" halts the repo's sub-sequence
   plus its transitive cross-repo dependent cone, disarms per Auto-Merge Choreography rule 5, and
   escalates: `ab12: CI failed (~/code/foo). Halted: … Fix foo and retry.` No fix round.
3. **Branch goes stale while armed.** Auto-Merge Choreography rule 3: `unchanged ≥ 3` on the armed
   `github-pr` item with no failed check → `gh pr view --json mergeable`; `CONFLICTING` → "disarm
   and escalate".

### Decisions made in the conversation (D1–D7)

| ID | Decision |
|----|----------|
| D1 | Reviewer never shows up → **terminalize**. `/git-pr-review`'s `timeout` outcome gets a retry budget of two invocations on the same PR (two consecutive timeouts ≈ 20 minutes). On exhaustion, finish `review-pr` exactly the way the existing `copilot: false` path does — the clean **no-reviews** outcome (`fab status finish <change> review-pr git-pr-review`) — with a logged reason `review-gate-unavailable`. The first timeout keeps today's behavior. The budget rides an EXISTING mechanism (an event written to the change history via `fab log`, counted by the next invocation) — no new `.status.yaml` field. |
| D2 | **No confirmation before merging unreviewed PRs** — drop any batch confirm. merge-all / merge-auto are already user-directed and carry the Destructive-tier confirm for the sequence; the change already passed the sub-agent `review` stage, so review-pr is a second gate, not the only one; a degraded second gate must not add a second prompt. Non-negotiable: the operator's tick report MUST name it — `merged with review gate unavailable: <PRs>` — so it never happens silently. |
| D3 | CI fails → **one bounded fix round before escalating**. The operator sends the authoring pane a fix request naming the PR and the failing run/check (and shard where applicable): "CI failed on run X / check Y — diagnose and fix, push, report". Exactly one round. Pane gone (reaped, `agent_exited`) → spawn a fresh agent in that change's worktree with the same fix prompt via the existing spawn machinery. Green after the round → the merge sequence continues (an armed PR STAYS armed during the round — do NOT disarm on the first failure; auto-merge fires when green). Still red → today's behavior (halt-dependents-only, disarm per rule 5, escalate) **with the agent's diagnosis attached**. |
| D4 | Stale branch while armed → **rebase and re-arm once**. On `CONFLICTING` (rule 3) the operator rebases the branch onto the resolved default branch in the change's worktree, pushes with `--force-with-lease`, and re-arms once. Only if it conflicts again (or the rebase itself conflicts and cannot be resolved mechanically) does it disarm and escalate as today. |
| D5 | **Detection uses signals that already exist only**: the `/git-pr-review` `timeout` outcome, the `gh pr checks` failed-required-check probe, and `gh pr view --json mergeable` == `CONFLICTING`. Rejected: any GraphQL timeline / `reviewRequests` / `review_requested` heuristic (GraphQL omits bot reviewers — the trap is already documented in git-pr-review.md), and any new "N consecutive drops" counter. |
| D6 | **Skills only + hydrate.** `git-pr-review.md` (budget, second-timeout exit, outcome table; the 10-minute synchronous-poll discipline untouched); `fab-operator.md` (ONE short new section "Review-PR Recovery" with a reason table — no-reviewer / ci-failed / conflicting — and the report wording; Ordered Merge's CI-failure paragraph and Auto-Merge rules 3 and 5 amended to route through it, owner-or-pointer; recovery stays distinct from the dependency-chain merge choreography — per item, independent stuck items are not a queue, halt-dependents-only still applies after a failed recovery); sweep of every `/fab-fff` Step 5 / `_pipeline` / `fab-continue` / `_preamble` text describing the `timeout` outcome; docs/memory (pipeline + runtime) and docs/specs (`skills.md`, `operator.md`) at hydrate. Canonical sources are `src/kit/skills/*.md`; deployed copies are never edited. |
| D7 | **Rejected alternatives**: (a) a standalone "review gate degraded" playbook / fleet-level condition with its own rebase-wait-merge loop — duplicates Ordered Merge + Auto-Merge; (b) parking `review-pr` as `failed` and waiting for the user — keeps items unreachable and pages the user, the opposite of the goal; (c) a batch confirm before merging unreviewed PRs — see D2; (d) timeline-based detection — see D5. |

## Why

**The pain.** `review-pr` is the pipeline's terminal stage and the operator's completion predicate.
It has one exit that is neither `done` nor `failed` — the Copilot `timeout` — and that exit is
absorbing: `/fab-fff` stops, the pane goes idle, and nothing in the system ever invokes
`/git-pr-review` again. When GitHub drops review requests (observed 2026-09-16: every request across
5 PRs silently dropped), every tracked item on the server becomes unreachable at once, and the
operator — whose whole job is to finish work without the user — can only wait. The user ended up
hand-driving a rebase → wait for CI → merge loop, per PR, which is exactly the choreography the
operator already owns (Ordered Merge + Auto-Merge). Separately, an addressable CI failure on a
shipped PR (a real assertion failure, not infra flake) is escalated to the user on first sight,
although the agent that wrote the change — still idle in its pane, with the change's whole context —
can usually fix it in a minute when asked.

**If we don't fix it.** Every review-gate outage stalls the whole fleet until a human notices; CI
failures keep paging the user for work an agent can do; and the "stuck" states keep being handled
ad hoc, differently each time, with no report line that says what happened. The operator's
`merge-auto` promise ("arm on completion") is hollow whenever the second gate is down, because
completion never arrives.

**Why this shape.** The three stuck reasons already have owners: `/git-pr-review` owns the review
gate's outcome table; Ordered Merge owns CI-failure handling; Auto-Merge rule 3 owns the
`CONFLICTING` probe. Each gets exactly one bounded recovery action and a guaranteed terminal state,
plugged into the step that already detects the condition. A new fleet-level "review gate degraded"
condition (D7a) would need its own detection heuristic (D5 rejects the only candidates — GraphQL
timeline reads omit bot reviewers), its own loop (duplicating Ordered Merge), and its own
confirmation (D2 rejects a second prompt). Parking as `failed` (D7b) keeps the item unreachable —
`failed` is not `done`/`skipped`, so the predicate never fires — and pages the user. Terminalizing
via the existing `no-reviews` path is what the `copilot: false` configuration already does on every
run; a *degraded* gate should behave like an *absent* one, with the reason logged and reported, not
like a broken pipeline. The sub-agent `review` stage (the first gate) has already passed by the
time `review-pr` runs, so merging unreviewed-by-Copilot is a bounded, named risk rather than an
unreviewed merge.

**Why one round, once, per item.** Constitution I (pure prompt play) and the operator's
Bounded-Retries doctrine (§3): every autonomous retry is a fixed small budget with a named
escalation. Two review requests, one CI fix round, one rebase-and-re-arm — then today's escalation,
carrying more evidence than today (the agent's diagnosis, the rebase result). Recovery is
**per item**: independent stuck PRs are not a queue and are not sequenced against each other;
halt-dependents-only still governs what a *failed* recovery blocks.

## What Changes

Skill sources only (`src/kit/skills/*.md`) plus the fab-kit docs that describe them. **No Go/CLI
change** and no new `fab` verb: every mechanism named below exists today (`fab log command`,
`fab status finish`, `fab operator track update --scope`, `gh pr checks`, `gh pr view --json
mergeable`, `gh pr merge --auto` / `--disable-auto`, `rk mux send`, the §6 spawn sequence). No
migration (no user-data restructuring). No change to the 10-minute poll window, the dependency-chain
merge ordering, or the synchronous-poll discipline.

### 1. `src/kit/skills/git-pr-review.md` — timeout retry budget → terminalize (D1)

**Budget carrier (existing mechanism, confirmed).** `fab log command <cmd> [change] [args]` appends
`{"args":"<args>","cmd":"<cmd>","event":"command","ts":"…"}` to
`fab/changes/{name}/.history.jsonl` (CLI help; `_cli-fab.md` § fab log; `internal/log`; the
`command` event shape is documented in the change-lifecycle memory). It is pure telemetry — always
exits 0, prints a stderr warning on internal failure, needs no shell guard — so it cannot become a
new failure mode (Constitution III). The skill today never calls it; this change adds two calls.

**Step 2 Phase 2 → "Copilot request and poll" step 2, the 20-attempts-exhausted branch** becomes:

1. **Record the timeout** for this PR:

   ```bash
   fab log command "git-pr-review" {name} "timeout pr={number}"
   ```

   (Only when Step 0 resolved a change — with no change there is no history file and no stage, so
   the branch below is skipped and the pre-change behavior — print the pending message — stands.)

2. **Count consecutive timeouts on this PR within the current stage activation.** The window
   opens at the most recent `review-pr` stage-transition line (written when `fab status finish
   <change> ship` auto-activates `review-pr`, or when Step 0's `fab status start` moves
   `pending`/`failed → active`; a `start` on an already-`active` stage is a no-op and writes
   nothing, so two back-to-back invocations share one window). Count `timeout pr={number}` markers
   after it:

   ```bash
   hist="fab/changes/{name}/.history.jsonl"
   timeouts=$(awk -v pr="{number}" '
     /"event":"stage-transition"/ && /"stage":"review-pr"/ { n = 0; next }
     index($0, "\"args\":\"timeout pr=" pr "\"") { n++ }
     END { print n + 0 }' "$hist")
   ```

   (The `awk` matches the JSON line by substring; key order is irrelevant. A `fab status reset
   review-pr` or a re-ship writes a fresh transition line and so resets the count — "consecutive"
   is per activation, never lifetime.)

3. **Branch on the count:**
   - `timeouts < 2` → **today's behavior, verbatim**: print `Copilot review requested but not yet
     available. Re-run /git-pr-review to process when ready.` (explicit-change form when
     `<change>` was passed) and go to Step 6 with outcome **timeout** (stage left `active`).
   - `timeouts ≥ 2` → **terminalize**: print

     ```
     Review gate unavailable — no Copilot review landed after 2 requests on PR #{number}
     (~20 minutes). Finishing review-pr unreviewed (reason: review-gate-unavailable).
     ```

     record the reason — `fab log command "git-pr-review" {name} "review-gate-unavailable
     pr={number}"` — and go to Step 6 with outcome **no-reviews**. Step 6's existing `no-reviews`
     row runs `fab status finish <change> review-pr git-pr-review 2>/dev/null || true` and its
     "Commit status in Step 6.5? **Yes**" column commits `.status.yaml` + `.history.jsonl` (the
     reason marker ships with the PR's branch). Nothing new is added to Step 6's stage actions —
     the second timeout *is* a `no-reviews` outcome with a reason.

**Step 6 outcome table** — amend the two rows' "Includes" cells (stage actions unchanged):

| Outcome | Includes | Stage action | Commit status in Step 6.5? |
|---------|----------|--------------|----------------------------|
| **no-reviews** | No reviews; no actionable inline comments; no automated reviewer; **second consecutive Copilot timeout on the same PR (reason `review-gate-unavailable`, logged)** | `fab status finish … 2>/dev/null \|\| true` (successful no-op) | Yes |
| **timeout** | Copilot still pending after 10 minutes — **first timeout in this activation only**; the timeout marker `timeout pr=<n>` is logged to the change history so the next invocation can count it | Leave `review-pr` `active`: no finish/fail; preserve the explicit-change re-run guidance | No |

**Result-file summary (dispatched arms).** When run as a `/fab-fff` Step 5 / `/fab-continue` worker,
the terminalized exit writes `outcome: no-reviews` with a `summary` that names the reason, e.g.
`summary: "review gate unavailable: 2 Copilot timeouts on PR #681; finished review-pr unreviewed"`
— the orchestrators already treat `no-reviews` as a successful no-op, so no new orchestrator branch
is needed; only the reported string changes.

**Idempotency (Constitution III).** Re-running `/git-pr-review` after the terminalized finish: Step 0's
`start` is a no-op on `done`; Phase 1 finds either a review that landed late (processed normally —
the stage is already `done`, `finish` no-ops) or none (Phase 2 requests again, may time out again;
the count is now ≥ 2 so it exits via `no-reviews` → `finish` no-ops → Step 6.5 finds nothing staged).
Clean no-op on every path; no marker is ever removed.

**Untouched:** the 30 s × 20 poll, the synchronous-poll discipline note, the two-login predicate,
the REST-not-GraphQL confirmation note, Steps 3–5.5, Phase Sub-State Tracking (no new `phase` value
— the history event is the durable record; `stage_metrics.review-pr.phase` keeps its five values).

**Skill header/description**: the frontmatter description gains the clause "…waits up to 10 minutes
for it to appear; a second consecutive timeout on the same PR finishes the stage unreviewed
(reason `review-gate-unavailable`)". `allowed-tools` needs no change (the skill already runs
`fab status` and `yq` outside the listed `git`/`gh`/`command` patterns; `fab log` and `awk` are the
same class).

### 2. `src/kit/skills/fab-operator.md` — ONE new section `#### Review-PR Recovery` (D2–D5)

Placed under §6 Coordination Patterns immediately after `#### Auto-Merge Choreography` (added to the
Contents list only if §6's sub-headings are listed there — today they are not, so no Contents
change). Short: one lead paragraph, one reason table, one report-wording list, one state note.

**Lead paragraph (intent, verbatim-grade):**

> A `review-pr` stage that cannot finish is ONE condition with a **reason**; each reason has ONE
> bounded recovery action, after which the stage is terminal or the item escalates with evidence.
> Recovery is **per item** — independent stuck items are not a queue and are never sequenced against
> each other; a *failed* recovery still applies Ordered Merge's halt-dependents-only policy and rule 5's
> disarm. Detection uses only signals that already exist: `/git-pr-review`'s `timeout` outcome, the
> `gh pr checks` failed-required-check probe, and `gh pr view --json mergeable` == `CONFLICTING`
> (never a GraphQL timeline / `reviewRequests` read — GraphQL omits bot reviewers — and never a new
> drop counter).

**Reason table:**

| Reason | Detected by | One bounded action | Then |
|--------|-------------|--------------------|------|
| `no-reviewer` | A tracked `pane` item observed at stage `review-pr` (`scope.stage`), agent `idle` past the §8 stuck threshold (15 m default) — the pipeline stopped on `/git-pr-review`'s first `timeout` and is waiting for a re-run that nothing else will send | **Re-send once**: the routed skill send `/git-pr-review <change>` to the authoring pane (rendered per `_cli-agents.md` § Skill Prompts; §3 pre-send gate — `idle` only). Pane gone (`pane_death` / `agent_exited` acked, or the item already removed) → one recovery spawn per § Recovery spawn below with the same invocation. `/git-pr-review`'s own budget terminalizes on the second consecutive timeout (`review-gate-unavailable`) — the operator never finishes or skips the stage itself | The item's built-in predicate fires (`review-pr` done) → normal merge mode takes over: `merge-auto` arms on completion, `cherry-pick-ladder` / `stacked-prs` wait for "merge all". Budget: 1 re-send per item per activation; a second idle-at-review-pr after it is a stuck-agent report (§3), not another send |
| `ci-failed` | Auto-Merge rule 3's failed required check on an armed `github-pr` item (`gh pr checks <n> --required --json name,state,link`, any `state == "FAILURE"`), or the foreground CI wait in Ordered Merge (stacked-prs / arming unavailable) turning red | **One fix round**: send the authoring pane the fix request (below) naming the PR, each failing check and its run link (`link`), and the shard where the check name carries one. **The armed PR stays armed** — no disarm on the first failure; GitHub merges it when the round goes green. Pane gone → one recovery spawn with the same fix prompt. Record the round on the `github-pr` item: `fab operator track update pr-<n> --scope '{"recovery":"ci-fix","fix_head":"<headRefOid at request>"}'` (`--scope` merges per key) | **Green** (auto-merge fires → the item's `done` delta; or the foreground wait passes) → the sequence continues; the merge report appends ` · after CI fix round`. **Still red** — a failed required check on a head SHA ≠ `fix_head` (the agent pushed and CI failed again), or 30 m elapsed with no new push (the §6 Failures stage-timeout bound) → today's path: halt-dependents-only, rule 5 disarm, escalate **with the agent's diagnosis** (`rk mux capture <pane> --lines 40`'s last report, or "no report — pane gone") |
| `conflicting` | Auto-Merge rule 3: `unchanged ≥ 3` on the armed item, no failed check, `gh pr view <n> --json mergeable` == `CONFLICTING` | **Rebase and re-arm once**: in the change's worktree (the live pane's worktree, else `wt create --non-interactive --name <name> --checkout <branch>` per `_cli-external.md` § wt's existing-branch route — `branch`/`repo` from the pane item or `branch_map`): `git fetch origin && git rebase origin/{default_branch} && git push --force-with-lease` (`{default_branch}` per Dependency Resolution step 0's chain — armed PRs always target the default branch, rule 1), then re-arm idempotently `gh pr merge --auto --squash <n>` (an "already enabled" rejection is fine). Record `fab operator track update pr-<n> --scope '{"recovery":"rebase"}'`. Recoverable tier: announce, no confirm — the user's merge-all / merge-auto confirm covers the sequence | The `mergeable` field flip resets the binary's `unchanged` counter on its own. **`CONFLICTING` again** with `scope.recovery == "rebase"` already set, or the rebase itself conflicts (`git rebase --abort`, §3 Bounded Retries "Rebase conflict → 0") → today's path: disarm, halt-dependents-only, escalate naming the conflicting files |

**Fix request text (sent verbatim, one Enter-terminated send, no skill prefix):**

```
CI failed on PR #{n} ({repo}): check `{check name}` — run {run link}{, shard S/N when present}.
Diagnose and fix on this branch, push, and report the root cause in one line.
```

This is the one sanctioned raw send into a fab-project agent: it **amends a change that already
completed the pipeline** — the same class as `/git-pr-review`'s own fix commits — not new work, so
Pipeline-first (§6 The pane Kind and Spawn Rules) is not bypassed. State this carve-out in one clause
on the Pipeline-first bullet ("…never send raw implementation instructions to a fab-project agent
— the single exception is § Review-PR Recovery's CI fix request, which amends a shipped change").

**Recovery spawn (shared by the three rows).** When the authoring pane is gone, run the §6 spawn
sequence steps 1–8 with: step 3 = the existing-branch route (`--checkout <branch>`); step 7's prompt =
the row's invocation (`/git-pr-review <change>` rendered per Skill Prompts for `no-reviewer`; the raw
fix request for `ci-failed`; `conflicting` needs no agent — the rebase is an operator maintenance
action); step 8 tracks it as a **change-less** pane item `<change>-fix --kind pane --pane … --repo …
--branch <branch>` (no `--change`, so the built-in completion — already `done` for a shipped change —
does not remove it on the next tick); `track rm <change>-fix` when the round resolves (PR merged, or
escalation). Budget: **1 recovery spawn per item per reason**. This is a carve-out to §3 Bounded
Retries' "Pane death → 0; respawn only for a queue-driven change" row — add one row there pointing
here: `Review-PR Recovery (§6) | 1 per reason | per its table`.

**Locating the authoring pane.** Join by change, never by memory: `fab pane map --all-sessions`
enriches every pane with its change/stage — the pane whose `change` equals the PR's change id is the
authoring pane (its tracked item is typically already `track rm`'d on the tick `review-pr` finished;
the pane itself survives, idle). The PR → change link rides the `github-pr` item's opaque `scope`:
amend rule 4's `track add` line to carry `"change":"<id>"` in `--scope` (`'{"repo":…,"pr":n,"change":"ab12"}'`).

**Report wording (tick footnote / merge report — bare markdown, §4 render rule):**

- `no-reviewer` re-send: `"{change}: review-pr idle {N}m — re-sent /git-pr-review (retry 2 of 2)"`
  · spawn form: `"{change}: review-pr idle {N}m, pane gone — spawned {wt} → %N for /git-pr-review"`
- Unreviewed merge (D2, MUST): on the verified-merge tick, for every PR merged this tick with no
  non-`PENDING` review (`gh pr view <n> --json reviews -q '[.reviews[] | select(.state != "PENDING")] | length'`
  == `0`) the per-merge line appends ` · unreviewed` and the tick emits one summary line
  `"merged with review gate unavailable: #{n} ({change}), #{m} ({change})"`. Never silent, never a
  prompt.
- `ci-failed` round: `"{change}: CI failed (check `{name}`, run {id}) — fix round sent to %N"` ·
  exhausted: `"{change}: CI still failing after fix round (check `{name}`, head {sha}). Halted:
  {sub-sequences}. Completed: {sub-sequences}. Diagnosis: {agent's line}. Fix and retry."`
- `conflicting`: `"{change}: PR #{n} CONFLICTING — rebased onto {default_branch}, pushed {sha},
  re-armed"` · exhausted: `"{change}: PR #{n} CONFLICTING again after rebase — disarmed. Halted:
  {sub-sequences}. Resolve by hand."` · rebase conflict: `"{change}: rebase conflict ({files}) —
  aborted, disarmed. Halted: {sub-sequences}."`

**Recovery state.** The per-item budget lives on the `github-pr` item's `scope.recovery` (written by
`track update --scope`, merged per key, durable across compaction — the items ARE the sequence state
per rule 4); the `no-reviewer` re-send budget is per activation and needs no persisted field (the
second idle-at-`review-pr` reads as a stuck agent under §3). `merge-auto` has a one-PR sequence:
"halt-dependents-only" degenerates to disarm-this-PR + escalate, and the queue's next spawn (which
waits on the verified merge) stays deferred until the user acts — today's behavior.

**Amend, owner-or-pointer (no duplicated text):**

- §6 Ordered Merge → "CI failure during ordered merge (halt-dependents-only)": open with "A failed
  required check first runs § Review-PR Recovery's `ci-failed` round; only an exhausted round
  halts:" then today's text verbatim; the example escalation line gains the diagnosis clause.
- §6 Auto-Merge Choreography rule 3: replace "disarm per rule 5 and apply the halt-dependents-only
  policy" with "→ § Review-PR Recovery `ci-failed`" and "`CONFLICTING` → disarm and escalate" with
  "`CONFLICTING` → § Review-PR Recovery `conflicting`".
- Rule 5: "Any halt or escalation — CI failure, stall, conflict — **after § Review-PR Recovery's
  bounded action is exhausted** — MUST run `gh pr merge --disable-auto`…" (an armed PR is NOT disarmed
  while a round is in flight).
- Rule 4's `track add` line: `--scope '{"repo":…,"pr":n,"change":"<id>"}'`.
- §1 Principles → "Coordinate, don't execute" allowlist: "rebase/cherry-pick for dependency
  resolution **and § Review-PR Recovery** (§6)".
- §3 Bounded Retries: one pointer row (above). §3 Confirmation Tiers unchanged (rebase is already
  Recoverable; merge already Destructive and covered by the sequence confirm).
- §9 Key Properties: no row change.

### 3. Sweep — every site that describes the review-pr `timeout` outcome (sibling class)

Grep `timeout` / `no-reviews` / `Copilot review requested` across `src/kit/skills/`, `docs/memory/`,
`docs/specs/` before finishing apply (code-quality.md § Sibling Sweeps). Known sites:

- `src/kit/skills/fab-fff.md` Step 5 "**If timeout**" paragraph + Error Handling "Review-PR timeout"
  row + CLI-arm `outcome: timeout` / `outcome: no-reviews` rows: add "(first timeout in the
  activation; a second consecutive timeout on the same PR is a `no-reviews` outcome with reason
  `review-gate-unavailable` — report its `summary`)". `/fab-fff` does NOT re-dispatch on the first
  timeout — today's stop stands (D1).
- `src/kit/skills/fab-continue.md` `review-pr`/`active` row: the "Timeout … deliberately left
  `active`" clause gains the same parenthetical.
- `src/kit/skills/_preamble.md` § Dispatch-Prompt Obligations review-pr result comment: "a timeout
  maps to the orchestrator's existing leave-`active` + pending-message path" → "a first timeout maps
  …; the second consecutive timeout on the same PR exits as `no-reviews`".
- `src/kit/skills/_pipeline.md`: no `timeout` text today (verified) — nothing to sweep; re-check.
- `docs/memory/pipeline/execution-skills.md` (git-pr-review Phase 2 paragraph; DD "Copilot-Only
  Phase 2"), `docs/memory/runtime/operator.md`, `docs/memory/pipeline/change-lifecycle.md` § Event
  History, `docs/specs/skills.md` (`/git-pr-review` Behavior 2 & 5 + Flow; `/fab-operator`),
  `docs/specs/operator.md` version history — at hydrate.
- `docs/specs/harness-adapters.md`, `docs/memory/runtime/dispatch.md`, `docs/memory/pipeline/planning-skills.md`
  mention `timeout` in other senses (dispatch wait / poll) — verify, leave unless they restate the
  review-pr outcome.

### 4. Backlog

`vb4l` is this change's ID → `/fab-archive` marks `[vb4l]` by exact-ID match. `[hf6x]` is marked by
hand at the same archive step: `- [x] [hf6x] … — SHIPPED via 260916-vb4l-review-pr-recovery` (the
file's existing convention). Not done at apply — archive is the usual point.

### Non-goals

- No Go/CLI change, no new `fab` verb, no new `.status.yaml` field, no migration.
- No change to the 10-minute poll window or the synchronous-poll discipline.
- No change to dependency-chain merge ordering or `stacked-prs`' own retarget/rebase steps (its
  rebase conflict already halts and escalates; D4 is scoped to **armed** PRs, rule 3).
- No autonomous second `/git-pr-review` dispatch inside `/fab-fff` — the second invocation comes from
  the operator's `no-reviewer` re-send or the user's re-run.
- No `rk notify` on unreviewed merges — the tick report line is the mandated surface.
- No detection of "review gate degraded" as a fleet-level condition (D7a).

## Affected Memory

- `pipeline/execution-skills`: (modify) `/git-pr-review` Phase 2 + Step 6 prose — two-invocation
  timeout budget carried by `fab log command` markers in `.history.jsonl` (per-activation count),
  second consecutive timeout → `no-reviews` with reason `review-gate-unavailable`; `/fab-fff` Step 5 and
  `/fab-continue` review-pr rows swept; DD "Copilot-Only Phase 2 in `/git-pr-review`" gains
  *Updated by* (timeout is no longer absorbing) — or a new four-field DD "Review-PR Always
  Terminalizes: the Second Consecutive Timeout Finishes Unreviewed" (Decision / Why / Rejected: park
  as `failed`, batch confirm / *Introduced by*).
- `runtime/operator`: (modify) § Requirements — new "Review-PR Recovery" subsection (reason table,
  report wording, per-item recovery state on `scope.recovery`, recovery spawn, Pipeline-first
  carve-out); "CI-failure = halt-dependents-only" paragraph routes through the fix round; `merge-auto`
  paragraph gains the unreviewed-merge report line; DDs "Repo-Spanning Queue CI-Failure =
  Halt-Dependents-Only" and "Auto-Merge Choreography: Sequential Arming…" gain *Updated by*; new DD
  "Review-PR Recovery: One Bounded Action per Stuck Reason, No Batch Confirm" (Rejected: fleet-level
  degraded-gate condition, park-as-failed, batch confirm, timeline detection).
- `pipeline/change-lifecycle`: (modify) § Event History — one sentence: `command` events with
  `args` are also consumed as a per-activation counter by `/git-pr-review` (`timeout pr=<n>`,
  `review-gate-unavailable pr=<n>`), windowed by the `review-pr` `stage-transition` line.

## Impact

**Skill sources (canonical, deployed by `fab sync`):**

- `src/kit/skills/git-pr-review.md` — Step 2 Phase 2 poll-exhausted branch (log marker, per-activation
  `awk` count, ≥ 2 → `no-reviews` with reason), Step 6 outcome table (two "Includes" cells), result
  `summary` guidance, frontmatter description.
- `src/kit/skills/fab-operator.md` — new `#### Review-PR Recovery`; pointer amendments in Ordered
  Merge (CI-failure paragraph), Auto-Merge rules 3/4/5, §1 allowlist clause, §3 Bounded Retries
  pointer row, Pipeline-first carve-out clause.
- `src/kit/skills/fab-fff.md`, `fab-continue.md`, `_preamble.md` — timeout-outcome sweep (prose only).

**Not touched:** Go (`src/go/**`), `_cli-fab*.md` (no command signature changes), templates,
migrations, `.agents/skills/` / `.claude/skills/` (deployed copies), `docs/site/merge-topologies.md`
(no CI/review text today — verify at sweep).

**Docs (hydrate):** `docs/memory/pipeline/execution-skills.md`, `docs/memory/runtime/operator.md`,
`docs/memory/pipeline/change-lifecycle.md`, generated indexes via `fab docs-index docs/memory`;
`docs/specs/skills.md` (`/git-pr-review`, `/fab-operator`), `docs/specs/operator.md` (version-history
row).

**Constraints honored:** Constitution I (markdown only, existing single-binary tools `gh`/`awk`/`fab`);
III (every path re-runs as a clean no-op — see § What Changes 1 Idempotency; recovery budgets are
recorded before acting so a crash between record and action costs at most one repeat); V citation
rule (skills cite kit skills, `fab` commands, `gh`, `rk` — never `docs/specs/*`, `docs/memory/*`,
`src/go/*`; the Go portability guard runs in CI); fab-operator.md conventions (§-numbered sections,
owner-or-pointer, Destructive/Recoverable tier vocabulary, bare-markdown report lines).

**Operational scale:** three markdown skills, ~120 lines net; three memory files; two spec files.
Lane: likely FULL (multi-file, cross-skill sweep, review-critical owner-or-pointer edits).

## Open Questions

- None blocking. See § Assumptions for the design details decided at intake (all Certain or
  Confident — no Tentative or Unresolved rows); `/fab-clarify` may revisit rows 10, 11, 13, 14 if
  the user wants different bounds or wording.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Terminalize the second consecutive Copilot timeout through the existing `no-reviews` outcome (`fab status finish … review-pr git-pr-review`), reason `review-gate-unavailable`; first timeout keeps today's behavior | Discussed — user decision D1, verbatim | S:95 R:85 A:95 D:95 |
| 2 | Certain | No batch confirm before merging unreviewed PRs; the tick report MUST name them (`merged with review gate unavailable: <PRs>`) | Discussed — user decision D2, explicitly non-negotiable | S:95 R:80 A:95 D:95 |
| 3 | Certain | One CI fix round to the authoring pane (respawn if gone), armed PR stays armed during the round, still-red → today's halt/disarm/escalate with the diagnosis attached | Discussed — user decision D3 | S:95 R:80 A:90 D:95 |
| 4 | Certain | `CONFLICTING` → rebase onto the resolved default branch, `--force-with-lease`, re-arm once; second conflict or an unresolvable rebase → disarm + escalate | Discussed — user decision D4 | S:95 R:80 A:90 D:95 |
| 5 | Certain | Detection signals: `timeout` outcome, `gh pr checks` failed required check, `gh pr view --json mergeable` == `CONFLICTING`; no GraphQL timeline / `reviewRequests`, no new drop counter | Discussed — user decision D5 | S:95 R:90 A:95 D:95 |
| 6 | Certain | Skills-only change (`git-pr-review.md`, `fab-operator.md` ONE new section + pointer amendments, timeout-outcome sweep) + hydrate; no Go, no new verb, no new `.status.yaml` field, no migration | Discussed — user decision D6 + Non-goals; Constitution I | S:95 R:85 A:95 D:95 |
| 7 | Certain | Retry-budget carrier = `fab log command "git-pr-review" {name} "timeout pr={number}"` markers in `.history.jsonl`, counted per activation (window opens at the last `review-pr` `stage-transition` line) with a single `awk` | Confirmed existing mechanism (CLI help, `_cli-fab.md` § fab log, `internal/log`, `fab status finish ship` auto-logs the review-pr transition); the user named this exact mechanism as the preferred example; alternative `stage_metrics.review-pr.phase` rejected — single-valued, would add a new value to an existing field | S:80 R:85 A:85 D:80 |
| 8 | Certain | Change ID = `vb4l` (archive's exact-ID backlog mark handles it); `[hf6x]` marked done by hand at archive with the file's `— SHIPPED via …` convention | `internal/archive.ArchiveWithBacklog` marks by exact change-ID only (verified); two IDs cannot both be the folder prefix | S:85 R:90 A:90 D:85 |
| 9 | Certain | Operator's per-item recovery budget lives on the `github-pr` item's `scope.recovery` (`ci-fix` + `fix_head`, or `rebase`) via `fab operator track update --scope` (merged per key); the PR→change link rides `scope.change` added to rule 4's `track add` | `--scope` merge-per-key verified in CLI help + `operator_track.go`; scope is documented as opaque kind-specific metadata; durable across compaction unlike an in-context counter | S:70 R:85 A:85 D:75 |
| 10 | Confident | Unreviewed-merge detection for the D2 report = zero non-`PENDING` reviews on the merged PR (`gh pr view --json reviews`), probed on the verified-merge tick | Same `reviews` query `/git-pr-review` already relies on (bot reviews DO appear there); the alternative — reading the branch's `.history.jsonl` marker — needs a worktree the operator may not have; a `copilot: false` project reads the same line, which is still true | S:60 R:90 A:80 D:60 |
| 11 | Confident | `no-reviewer` operator action = one routed `/git-pr-review <change>` re-send to the idle authoring pane (idle past the §8 stuck threshold, 15 m default), recovery spawn if gone; `/fab-fff` itself does NOT re-dispatch on the first timeout | D1 says the first timeout keeps today's behavior (stop); the second invocation must come from somewhere for the budget to be reachable under the operator — the §3 stuck-nudge budget of 1 is the existing shape; user re-run covers the manual path | S:65 R:85 A:80 D:70 |
| 12 | Certain | The CI fix request is a raw send into a fab-project agent — record it as the single sanctioned exception to Pipeline-first (amends a shipped change, same class as `/git-pr-review`'s fix commits) | D3's wording ("send the authoring pane a fix request … diagnose and fix, push, report") is a raw prompt by design; `/fab-new` would create a new change for a follow-up tweak, which the fab skills explicitly call not-new-work | S:80 R:85 A:80 D:80 |
| 13 | Confident | Fix-round end conditions: green → continue; failed required check on a head SHA ≠ `fix_head` → exhausted; 30 m with no new push → exhausted (reuses §6 Failures "Stage timeout (>30m)") | A round needs a bound the tick can evaluate mechanically; both signals already exist (`headRefOid`, `gh pr checks`, the 30 m constant) | S:55 R:85 A:75 D:60 |
| 14 | Confident | Recovery spawns are tracked as change-less `<change>-fix` pane items (no `--change`) and removed when the round resolves; budget 1 spawn per item per reason, recorded as a pointer row in §3 Bounded Retries | A `--change` item would be `done` at the next tick (review-pr already done) and be acked away; change-less pane items are the existing shape that "never done on its own" | S:60 R:85 A:85 D:70 |
| 15 | Certain | D4 scoped to armed PRs (rule 3); `stacked-prs` merge-all keeps its own explicit `rebase --onto` step and its halt-and-escalate on conflict | Rule 3 is Auto-Merge-only and `stacked-prs` is excluded from arming by design; its rebase is already operator-sequenced | S:70 R:90 A:90 D:80 |
| 16 | Certain | `change_type` left to `fab status refresh`'s keyword inference (`fix` expected from the intake text; `feat` acceptable) — no explicit override | Both types fit a change that closes two backlog gaps by adding recovery behavior; the type affects PR tier only | S:70 R:95 A:85 D:75 |

16 assumptions (12 certain, 4 confident, 0 tentative, 0 unresolved).
