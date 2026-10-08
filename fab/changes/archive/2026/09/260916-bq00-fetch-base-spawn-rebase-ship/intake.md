# Intake: Fetch the Base Before Spawning a Worktree, Rebase onto the Base at Ship

**Change**: 260916-bq00-fetch-base-spawn-rebase-ship
**Created**: 2026-09-16

## Origin

User-reported, then verified by research in the same session. The user has repeatedly hit merge
conflicts at the end of operator-driven fab changes because the change's base branch was stale by
the time the work landed. The discussion established the mechanism at both ends of the change's
life — worktree creation and ship — and the user directed: *"Go ahead with both the changes in the
single fab change."*

> Fetch the base before spawning a worktree, and rebase onto the base at ship — stale-base merge
> conflicts in operator queues.

Interaction mode: conversational research + design, dispatched promptless. No questions were asked
at intake; the deliberation that produced the mechanics below happened in the discussion.

### What the research established (five findings)

1. **Worktree creation is off the main checkout's LOCAL HEAD in every merge mode.**
   `fab-operator.md` §6 spawn step 3 runs the probe-and-route procedure in `_cli-external.md` § wt
   → Operator Spawning Rules, which issues `wt create --non-interactive --name <wt>
   [<change-folder-name>]` with **no `--base`**. In wt's source
   (`~/code/sahil87/wt/src/internal/worktree/crud.go`) an empty start-point means
   `git worktree add -b <branch> <path>` → the new branch is cut from **HEAD**. wt fetches only for
   `--checkout` of a remote-only branch. So a queue item starts from whatever commit the target
   repo's main checkout happens to be sitting on; nothing guarantees it is current.

2. **Dependency Resolution step 0's `git fetch origin` does not fix it.** It runs *only* when
   `depends_on` is non-empty — never for the first queue item, and never in `merge-auto` (which
   disables implicit chaining) — and it runs *after* the worktree already exists. It refreshes
   `origin/{default_branch}` for the cherry-pick range; it never moves the worktree's base. The
   g8st Design Decision "Fetching first prevents a stale base"
   (`docs/memory/runtime/operator.md` § Design Decisions → *Operator Git Ops Fetch First and
   Resolve the Default Branch*) is about **cherry-pick/rebase targets**, not the worktree base.

3. **`merge-auto`'s promise is incoherent with its mechanism.** `fab-operator.md` § Queues and
   `docs/site/merge-topologies.md` both claim "the next change starts from the advanced line", and
   the arming `github-pr` item's `then` "fetches, rebases, and spawns the next change". But for a
   backlog / raw-text queue item **the branch does not exist before spawn** (there is nothing to
   rebase), and a fetch moves `origin/main`, not local `main` — so the subsequent `wt create` still
   branches off stale local HEAD.

4. **`stacked-prs` is correct within a stack but not at its root.** Dependents are created with
   `--checkout <dep-branch>` (correct). The **root** of the stack has the same stale-HEAD problem.
   Its merge-all `git fetch origin && git rebase --onto origin/{default_branch} …` is the one place
   the base is genuinely refreshed — at merge time, which is exactly where the conflicts surface.

5. **Nothing downstream reconciles.** `/fab-new` Step 11 and `/git-branch` Step 4 (the
   branch-creation twins, kept in sync via in-file comments) do `git checkout -b` off the current
   branch with **no fetch**. `/git-pr` (ship) pushes and creates the PR **without fetching or
   rebasing**: Step 1 Gather State resolves `default_branch` and `base_branch`
   (`fab status get-base-branch` with the default-branch fallback), Step 3b pushes with
   `git push -u origin <branch>` or `git push`, Step 3c runs
   `gh pr create --draft --base "$base_branch"`. The per-change `base_branch` field (vy27) records a
   branch **name**, not a commit — so it pins the PR target, not the base *content*.

## Why

**The problem.** A change's base is fixed at two moments — when its worktree/branch is created, and
when its commits are pushed — and today neither moment consults the remote. An operator queue can
run for hours; by the time item *n* ships, `origin/main` has moved by every earlier item in the
queue plus anything a human merged. The change's diff is computed against a base that no longer
exists on the remote, and the divergence surfaces as a merge conflict at the very end of the
pipeline: at `gh pr merge`, or at `merge-all`, where the **operator** discovers it with none of the
change's context and the worker pane that *had* that context is long gone.

**What happens if we don't fix it.** The failure keeps recurring and keeps being discovered in the
worst possible place. `merge-auto` continues to advertise a guarantee ("the next change starts from
the advanced line") that its mechanism does not deliver — the single most misleading claim in the
merge-topology documentation. Every queue item after the first inherits accumulated staleness, so
the conflict probability grows with queue length: the exact workload the operator exists to run is
the one it handles worst.

**Why this approach over the alternatives.**

- *Fix the base at creation (Fix 1)* is the root-cause fix: a branch cut from the current remote tip
  never accumulates drift in the first place. It is one fetch and one flag on a command the operator
  already runs, and `wt --base` already exists (tool-owned, no wt change needed).
- *Rebase at ship (Fix 2)* is the reconciliation fix, and is needed **in addition** because a change
  can be long-lived (a worker pane may run for hours after its worktree was created) and because
  non-operator flows — `/fab-new`, `/git-branch`, a hand-created branch — never get a spawn-time
  fetch at all. Ship is the last moment before the change becomes someone else's problem.
- *Rejected: rely on the ship rebase alone.* It would leave the whole apply/review pass running
  against a stale tree — work built on old assumptions, conflicts discovered only after the
  implementation is finished.
- *Rejected: rely on the spawn fetch alone.* It fixes only the operator path and only at t=0.
- *Rejected: make `base_branch` record a commit SHA.* That inverts vy27's design (the field is the
  PR *target*, a branch name that `gh pr create --base` consumes) and would need a Go/CLI change,
  a schema change, and a migration for a problem two skill edits solve.

**Why the conflict surfaces in the worker pane, by design.** Fix 2 deliberately moves conflict
discovery from merge-all to the ship stage of the pane that authored the change. That pane holds the
change's intent, its plan, and its diff; the operator holds none of it. A ship-stage failure with a
named file list is recoverable by the agent that wrote the code; a merge-all conflict is an
escalation to the user.

## What Changes

Skill sources only (`src/kit/skills/*.md`) plus the fab-kit docs that describe them. **No Go/CLI
change** is expected, so no `_cli-fab*.md` reference update and no migration. `wt --base` is
tool-owned and already exists (`wt create --base string  Git ref (branch, tag, SHA) to use as
start-point for new branch`, wt v0.1.7) — no wt change.

### Fix 1 — spawn-time fetch + `--base <sha>` (operator, mode-agnostic)

**Where**: `src/kit/skills/fab-operator.md` §6 → Spawning an Agent, step 3; and
`src/kit/skills/_cli-external.md` § wt → Operator Spawning Rules (the two `wt create` example
lines).

**Behavior**: before `wt create`, **in the target repo's main-worktree root**, run `git fetch
origin`, resolve the default branch with the existing default-branch chain, and pass the **fetched
commit SHA** as `--base` to **both** new-branch `wt create` forms.

```sh
# In the TARGET repo's main-worktree root (never the operator's own repo), before wt create:
git fetch origin
# {default_branch} resolved per the existing chain — Dependency Resolution step 0 owns it;
# point at it, do not restate the three lines here.
base_sha=$(git rev-parse "origin/${default_branch}")

# branch missing → create it (new-branch positional), now off the fetched tip
wt create --non-interactive --name <name> --base "$base_sha" <change-folder-name>

# no change branch (no-fab/ repo, or a bare spawn) → new branch of the worktree's name, same base
wt create --non-interactive [--name <name>] --base "$base_sha"
```

**The SHA, not the ref name — this is load-bearing.** `--base` must receive
`$(git rev-parse origin/{default_branch})`, **never** the literal ref `origin/{default_branch}`.
Verified this session with git 2.53.0: `git worktree add -b br ../wt origin/main` sets upstream
tracking on the new branch (`branch.br.remote=origin`, via `branch.autoSetupMerge`). That upstream
would:

- break `/fab-new` Step 11 row 5's **rename guard** — the row requires `upstream` empty; a
  tracking branch falls through to row 6 (`created, leaving {old_branch} intact`) instead of
  renaming the disposable worktree branch, leaving the worktree on the wrong branch; and
- confuse `push.default=simple`, whose push target would become the default branch.

A **SHA** start-point sets no upstream. This must be stated as a rule where the flag is documented,
not left as a style preference.

**Untouched**: the `--checkout <branch>` form (an existing change branch; a `stacked-prs` dependent
cut off its dependency branch). wt's own contract rejects `--base` with `--checkout`, and that route
is already correct — the branch it checks out is the intended base.

**Also untouched**: Dependency Resolution step 0's `git fetch origin`. The cherry-pick range still
needs a current `origin/{default_branch}`; the new spawn-time fetch is **additive** and is the one
that fixes the base. Owner-or-pointer: the default-branch resolution chain is owned by Dependency
Resolution step 0 — step 3 points at it and does not restate the three lines.

### Fix 2 — rebase onto the base at ship (`/git-pr`)

**Where**: `src/kit/skills/git-pr.md` — a new rebase sub-step after the commit-producing sub-steps
(3a Commit, 3a-bis Refresh Memory Indexes) and **immediately before 3b Push**, so every commit the
ship stage produces is included in the rebase.

**Applicability**: whenever a base can be resolved — **not** gated on `{has_fab}`. Step 1 already
resolves `base_branch` for both cases (the recorded per-change base when `{has_fab}` and its ref
resolves; `$default_branch` otherwise), so the step has a target either way.

**Mechanics**:

```sh
git fetch origin
# $base_branch is already resolved in Step 1 (recorded per-change base, default-branch fallback).
if git rev-parse --verify -q "refs/remotes/origin/$base_branch" >/dev/null; then
  git rebase "origin/$base_branch"
else
  : # skip the rebase, emit a one-line warning naming $base_branch, continue to 3b
fi
```

Using the **recorded** base (not the default branch) is what makes `stacked-prs` correct: a
dependent rebases onto its dependency's branch, not onto main.

**Push after a rebase** (3b): if an upstream exists (a re-ship, or a rework re-run), push with
`git push --force-with-lease` — a rebase rewrites history, so a plain `git push` would be rejected.
The first push is unchanged: `git push -u origin $(git branch --show-current)`.

**Conflict policy** (the design default; recorded as a graded assumption, the user may override):

| Conflict class | Action |
|----------------|--------|
| Generated `docs/memory/**/index.md` or `log.md` | Follow the existing rule already in git-pr: **never hand-merge a generated index/log (FKF §5)** — take either side, re-run `fab docs-index docs/memory`, take its output wholesale, continue the rebase |
| Any other file, resolution **clear** from the change's own intent | The ship agent resolves it and continues |
| Any other file, resolution **not** clear | `git rebase --abort`, report the conflicting file list, **STOP the ship stage** (fail; nothing pushed, no PR created) |

Rationale for the STOP: surfacing the conflict in the worker pane — which holds the change's
context — beats discovering it at merge-all, where the operator has none.

**Twin sweep**: ship is `driver: git-pr`; `/fab-fff` Step 4 and `/fab-ff` (twins) dispatch it. Check
whether either twin **restates** git-pr's ship sub-steps (a read of both files during intake found
delegation, not restatement — they name `/git-pr {name}` and describe the dispatch seam) and sweep
them only if a restatement exists.

### Fix 3 — `merge-auto` wording (`fab-operator.md` § Queues)

The mechanism that delivers "the next change starts from the advanced line" is now the **spawn-time
fetch + `--base <sha>`**, not a pre-spawn rebase. Three sites in `src/kit/skills/fab-operator.md`:

1. The **`merge-auto` bullet** under § Queues (currently: *"…once the merge is verified on a later
   tick …, `git fetch origin` and rebase the next change onto `origin/{default_branch}`"*) — drop
   the pre-spawn rebase wording.
2. The paragraph *"In `merge-auto` mode, steps 5–8 …"* — **keep** "defer the next change's spawn to
   the tick that verifies the merge" (that ordering rule still holds and is still load-bearing);
   drop the "rebase the next change onto `origin/{default_branch}`" clause.
3. The arming `github-pr` item's **`then`** text — *"one `github-pr` item whose `then` fetches,
   rebases, and spawns the next change"* → the rebase is gone; the fetch now rides the spawn.

The incoherence being fixed: for a backlog/raw-text item there is no branch to rebase before spawn,
and fetching moves `origin/main`, not local `main`.

### Sweep class (Sibling Sweeps, `fab/project/code-quality.md`)

Grep the old claim repo-wide and update every occurrence **before finishing apply** — this project's
single most common rework cause.

| Surface | What changes |
|---------|--------------|
| `src/kit/skills/fab-operator.md` | §6 step 3 (Fix 1), § Queues `merge-auto` bullet + steps-5–8 paragraph + arming `then` (Fix 3) |
| `src/kit/skills/_cli-external.md` § wt | Operator Spawning Rules — the two `wt create` example lines + the no-branch-argument sentence (Fix 1) |
| `src/kit/skills/git-pr.md` | The new rebase sub-step, the 3b push branch for `--force-with-lease`, Contents/Key Properties if they enumerate sub-steps (Fix 2) |
| `src/kit/skills/_cli-agents.md` § Spawn Composition | **Verified during intake: carries no `wt create` restatement** (it explicitly disclaims worktree-creation policy). Re-check, expect no edit |
| `docs/memory/runtime/operator.md` | Spawn sequence, § Queues `merge-auto` paragraph, and the g8st Design Decision *Operator Git Ops Fetch First and Resolve the Default Branch* — **extend/update it** (the decision now covers the worktree base, not just cherry-pick/rebase targets); keep the *Introduced by* / *Updated by* trailer convention (`*Updated by*: 260916-bq00-…`) |
| `docs/memory/pipeline/execution-skills.md` | Owns `/git-pr` ship behavior (36 `git-pr` mentions; `pipeline/index.md` routes here) — add the rebase step |
| `docs/memory/pipeline/change-lifecycle.md` § Git Integration | The default-branch-resolution convention paragraph names what each consumer does with the chain; `/git-pr` now also rebases |
| `docs/specs/skills.md` | The `/git-pr` and `/fab-operator` partial flow skeletons |
| `docs/site/merge-topologies.md` | Its `merge-auto` claim (table row line ~8, prose line ~34: *"the next change starts from the advanced line"*) **becomes true** — re-check the wording rather than assuming an edit is needed |
| `docs/memory/runtime/log.md`, `docs/memory/pipeline/log.md` | **Generated** by `fab docs-index` — never hand-edit |

## Affected Memory

- `runtime/operator`: (modify) — §6 spawn sequence step 3 gains the pre-`wt create` fetch +
  `--base <sha>`; § Queues `merge-auto` paragraph drops the pre-spawn rebase; the g8st Design
  Decision *Operator Git Ops Fetch First and Resolve the Default Branch* extends to cover the
  **worktree base** (keep the `*Introduced by*` / `*Updated by*` trailer)
- `pipeline/execution-skills`: (modify) — `/git-pr` ship behavior gains the rebase-onto-base step,
  its conflict policy, and the `--force-with-lease` re-push rule
- `pipeline/change-lifecycle`: (modify) — § Git Integration's default-branch-resolution convention
  paragraph: `/git-pr` now fetches and rebases onto the recorded base, not only guards with the
  resolved `{default_branch}`

## Impact

**Skill sources** (canonical; `.agents/skills/` and `.claude/skills/` are deployed copies and are
**never** edited directly — `fab sync` regenerates them):

- `src/kit/skills/fab-operator.md` — §6 step 3, § Queues (3 sites)
- `src/kit/skills/_cli-external.md` — § wt → Operator Spawning Rules
- `src/kit/skills/git-pr.md` — new rebase sub-step + 3b push branch

**fab-kit docs**: `docs/memory/runtime/operator.md`, `docs/memory/pipeline/execution-skills.md`,
`docs/memory/pipeline/change-lifecycle.md`, `docs/specs/skills.md`, `docs/site/merge-topologies.md`.

**Not touched**: Go/CLI (`src/go/**`), `_cli-fab.md` / `_cli-fab-pane.md` / `_cli-fab-operator.md`
(no command signature changes), migrations, templates, `wt` itself.

**Constraints**:

- **Constitution V (deployed-content citation rule)** — deployed `src/kit/**` files MUST NOT cite
  fab-kit's `docs/specs/*`, `docs/memory/*`, `docs/site/*` or `src/go/*` as an authority or rule
  owner. A Go test guards this. Every rule added to `fab-operator.md`, `_cli-external.md` and
  `git-pr.md` is **restated in the skill**; the fab-kit doc then points at the skill, never the
  reverse.
- **Owner-or-pointer** — state a rule once. The default-branch resolution chain stays owned by
  Dependency Resolution step 0 (operator side) and Step 1 (git-pr side); the new steps point at
  their owner rather than re-inlining the three-line chain.
- **Sibling Sweeps** (`fab/project/code-quality.md`) — sweep the whole class up front; review treats
  a missed sibling as must-fix (`documentation_accuracy` + `cross_references`).
- **Generated files** — `docs/memory/**/log.md` and the index tables are `fab docs-index` output;
  regenerate, never hand-edit.

**Risk**: every operator spawn gains one `git fetch origin` (a network round-trip on a path that is
already doing tmux + worktree + agent-launch work — negligible), and every ship gains one fetch plus
a rebase that can now fail the ship stage where it previously always succeeded. The ship-stage
failure is the intended behavior change, not a regression.

**Testing**: no Go tests to add (no Go change). Verification is behavioral —
`wt create --base <sha>` leaving `branch.<name>.remote` unset (the rename-guard precondition), and
the three `merge-auto` wording sites no longer claiming a pre-spawn rebase.

## Open Questions

- Should the ship-stage rebase be **skipped when an OPEN PR already exists** (i.e. rebase only on
  the first ship)? A `--force-with-lease` push after a rebase rewrites history under an open PR,
  which invalidates in-flight review threads and re-runs CI. The default taken here is *always
  rebase* (a current base matters more than thread stability, and `/git-pr` on an OPEN PR is the
  re-ship/rework path, not the normal one) — recorded as a Tentative assumption below.
- Does `wt create --reuse` (the operator's respawn form) interact with `--base`? Expected: `--reuse`
  reuses the existing worktree, so the start-point is moot and passing `--base` is harmless — not
  verified against wt's reuse code path.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Both the spawn-time fetch (Fix 1) and the ship rebase (Fix 2) ship in ONE fab change, together with the `merge-auto` wording correction (Fix 3) | Discussed — user: "Go ahead with both the changes in the single fab change"; Fix 3 is the wording consequence of Fix 1 and cannot ship separately without leaving the docs incoherent | S:95 R:80 A:90 D:95 |
| 2 | Certain | `--base` receives the fetched COMMIT SHA (`git rev-parse origin/{default_branch}`), never the ref name `origin/{default_branch}` | Verified this session with git 2.53.0: a ref start-point sets `branch.<name>.remote=origin` (branch.autoSetupMerge), which breaks `/fab-new` Step 11 row 5's rename guard (`upstream` non-empty ⇒ row 6 instead of rename) and confuses `push.default=simple`; a SHA sets no upstream | S:95 R:70 A:95 D:90 |
| 3 | Certain | The `--checkout <branch>` route (existing change branch; `stacked-prs` dependent off its dep branch) is untouched | wt's own contract rejects `--base` with `--checkout` (confirmed against `wt help-dump`, v0.1.7: "--checkout cannot be combined with a positional branch argument or with --base"); the route is already correct | S:90 R:85 A:95 D:95 |
| 4 | Certain | Dependency Resolution step 0's `git fetch origin` stays; the spawn-time fetch is additive and is the one that fixes the base | The cherry-pick range still needs a current `origin/{default_branch}`; removing it would break same-repo dependency resolution (g8st) | S:85 R:85 A:90 D:85 |
| 5 | Certain | Every rule added to `fab-operator.md`, `_cli-external.md` and `git-pr.md` is restated in the skill, never cited to fab-kit's own docs | Constitution V, guarded by a Go test | S:90 R:85 A:100 D:90 |
| 6 | Certain | Fix 2 applies whenever a base can be resolved — NOT gated on `{has_fab}` | Discussed — user named this the default and left the call to the intake; the codebase settles it: git-pr Step 1 already resolves `base_branch` in both cases (recorded base, else `$default_branch`), so the non-`{has_fab}` path has a working target | S:80 R:85 A:85 D:80 |
| 7 | Confident | Ship-rebase conflict policy: generated `docs/memory/**/index.md`/`log.md` follow the existing FKF §5 regenerate-wholesale rule; any other conflict is resolved only when clear from the change's own intent, else `git rebase --abort` + report the files + STOP the ship stage | Discussed — user stated this as the design default in full and flagged it overridable; the FKF §5 half is already a rule in git-pr. The invited override is the only gap | S:85 R:75 A:75 D:70 |
| 8 | Certain | Push after a rebase uses `git push --force-with-lease` when an upstream exists; the first push (`git push -u origin <branch>`) is unchanged | Discussed — user specified it verbatim, and it is a git necessity rather than a preference: a rebase rewrites history, so a plain `git push` is rejected | S:90 R:80 A:90 D:90 |
| 9 | Certain | The rebase sub-step sits AFTER 3a (Commit) and 3a-bis (Refresh Memory Indexes) and immediately BEFORE 3b (Push) | Derived, not asked: the user said "before the push (Step 3b)", and the sub-step order forces the rest — a rebase placed before 3a-bis would leave the index-refresh commit outside it. Exactly one placement satisfies both | S:70 R:85 A:85 D:85 |
| 10 | Confident | `/fab-new` Step 11 and `/git-branch` Step 4 are a **Non-Goal** — no fetch is added at branch creation | User's fix list is explicit and enumerated (Fix 1/2/3) and excludes them; Fix 2's ship rebase reconciles those flows at ship, which is the reconciliation point for every non-operator path | S:70 R:80 A:80 D:75 |
| 11 | Confident | `/git-pr-review`'s re-push is a **Non-Goal** — no rebase added there | Not named in the fix list, and the pipeline settles it: review-pr runs directly after ship, so the base was refreshed moments earlier — a second rebase would rewrite history under an active review for no stale-base benefit | S:55 R:85 A:80 D:75 |
| 12 | Certain | No Go/CLI change, no `_cli-fab*.md` reference update, no migration, no `wt` change | Verified during intake: every edit is skill/doc prose, no `fab` command signature changes, and `wt --base` is tool-owned and already exists (v0.1.7). The constitution's CLI⇒docs constraint therefore does not fire | S:80 R:70 A:90 D:85 |
| 13 | Confident | `docs/site/merge-topologies.md` is re-checked rather than assumed-edited — its `merge-auto` claim becomes TRUE under Fix 1 | The claim "the next change starts from the advanced line" is what Fix 1 finally delivers; only the mechanism prose (if any) needs adjusting | S:65 R:90 A:75 D:75 |
| 14 | Confident | The ship rebase runs even when an OPEN PR already exists (the re-ship / rework path) — it is not skipped to protect in-flight review threads <!-- assumed: always-rebase at ship; a current base was judged to matter more than PR review-thread stability, and /git-pr against an OPEN PR is the re-ship path, not the normal one --> | Not addressed head-on, but the user's own `--force-with-lease` clause ("if an upstream exists — re-ship / rework re-run") only makes sense if the rebase runs on a re-ship, so the front-runner is the user's. The cost — invalidated review threads, re-run CI — is the reason it is flagged rather than silent | S:70 R:65 A:55 D:55 |
| 15 | Confident | `wt create --reuse` (operator respawn) may be passed `--base` harmlessly — `--reuse` reuses the existing worktree, so the start-point is moot <!-- assumed: --reuse ignores --base; not verified against wt's reuse code path --> | `wt help-dump` (v0.1.7) documents a conflict for `--checkout` × `--base` and none for `--reuse` × `--base`. The reuse code path was not read, but it is answerable at apply from wt's own source, and a wrong call costs one flag placement | S:45 R:90 A:55 D:60 |

15 assumptions (9 certain, 6 confident, 0 tentative, 0 unresolved).
