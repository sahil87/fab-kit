# git-pr

## Summary

Autonomously commits, pushes, and creates a draft GitHub PR. No prompts, no questions. Resolves PR type from status/intake/diff. Generates PR body from fab artifacts when available. Records PR URL in `.status.yaml`.

**State hardening** (260612-g8st): verifies git state before mutating — detached-HEAD STOP before any commit/push; the branch guard uses the *resolved* default branch (symbolic-ref → `gh repo view` → probed literal `main`/`master`; always non-empty); staging is scoped (`git add -u` + expected-area guard for untracked files — never `git add -A`); Step 3 branches on the PR's `state` (OPEN short-circuit / CLOSED fresh PR / MERGED STOP).

**Re-run contract** (idempotency — a fab-kit design principle; declared in the skill's Key Properties section): re-run after ship is a no-op — the "already shipped" path (no uncommitted changes, no unpushed commits, an **OPEN** PR exists) re-records the existing PR URL silently and stops; `fab status add-pr` is idempotent and the Step 4c status commit is guarded by `git diff --cached --quiet`.

**Dispatch-target hardening** (260612-w7dp): accepts an optional explicit `<change>` argument (any argument that isn't one of the 7 PR types), resolved transiently in Step 0 — `.fab-status.yaml` untouched; an explicit argument that fails to resolve STOPs (caller error), while argless failure keeps the silent `{has_fab}=false` degradation. Callers SHOULD pass the change folder name, not a bare 4-char id (an id spelling a type word — `feat`/`docs`/`test` — would classify as a type); `/fab-fff` Step 4 passes the folder name through (`/git-pr {name}`). Step 0 ends with a **branch-matches-change guard** (exact folder-name equality, or the folder name as a branch substring) that STOPs before any status mutation, commit, or push on mismatch — no autonomous checkout; an empty branch (detached HEAD) likewise STOPs there, before Step 0a's `fab status start` (verify-before-mutate parity with git-pr-review — Step 2's own guard still covers the no-fab path). It supersedes the former Step 1b non-blocking nudge, which is removed.

**Memory-index date-drift fix** (260620-o203): a new sub-step **3a-bis. Refresh Memory Indexes** sits between 3a (Commit) and 3b (Push). `fab memory-index` stamps each `index.md` row's "Last Updated" cell from `git log` (committed dates only); the hydrate-stage regen runs entirely pre-commit, so every file the change touched is stamped one regen behind until the content commit lands — a benign tier-1 `fab memory-index --check` drift at review-pr. 3a-bis closes the gap at the only pipeline position where `git log` already knows the change's real commit date: immediately after 3a commits, before 3b pushes. It regenerates byte-stably and, when `docs/memory/` actually drifted (`git diff --quiet -- docs/memory` non-zero), makes a **separate** `docs: refresh memory indexes` follow-up commit (never `--amend` — squash collapses the pair on merge); a no-drift regen produces no diff and no commit (Constitution III). It performs no push of its own — 3b's "if has_unpushed or just committed" trigger pushes both commits together. It is **gated on `{has_fab}` AND 3a-having-just-committed**, so it is a silent no-op for standalone `/git-pr` (`{has_fab}` false) and for the no-change re-run paths. On regen/commit failure it reports + STOPs with the 3a content commit intact (a benign stale-date index recoverable by re-running `fab memory-index` — never a torn state).

## Flow

```
/git-pr [<change>] [<type>] invoked (user or sub-agent)
│
├─ Step 0: Resolve Change Context (260611-szxd f094 — the ONLY
│  │        fab change resolve in the skill; later steps reference
│  │        the variables and MUST NOT re-resolve)
│  ├─ Bash: fab change resolve [<change>] 2>/dev/null → {has_fab}, {name}
│  │        (explicit <change> = transient override, 260612-w7dp;
│  │         explicit-arg resolution failure → STOP, never a silent
│  │         fallback to the active change; argless failure → {has_fab}=false)
│  ├─ {has_intake} — fab/changes/{name}/intake.md exists?
│  ├─ {change_type} — from fab/changes/{name}/.status.yaml
│  └─ Branch-matches-change guard (260612-w7dp, if {has_fab}):
│     branch must equal {name} or contain it as a substring →
│     mismatch STOPs BEFORE Step 0a's status mutation (guidance:
│     /git-branch, /fab-switch, or /git-pr <change>); empty branch
│     (detached HEAD) STOPs here too — before Step 0a — with Step 2's
│     detached-HEAD message (Step 2's guard covers the no-fab path)
│
├─ Step 0a: Start Ship Stage (if {has_fab})
│  └─ Bash: fab status start {name} ship git-pr
│
├─ Step 0b: Resolve PR Type
│  ├─ {change_type} from Step 0 (status source)
│  ├─ Read: fab/changes/{name}/intake.md (keyword match, if {has_intake})
│  └─ Bash: git diff --name-only (fallback)
│
├─ Step 1: Gather State
│  ├─ Bash: git branch --show-current   (empty = detached HEAD → Step 2 guard)
│  ├─ Bash: git status --porcelain
│  ├─ Bash: git log --oneline -5
│  ├─ Bash: git log --oneline @{u}..HEAD
│  ├─ Bash: gh pr view --json number,state,url
│  │        → {pr_state} (OPEN/CLOSED/MERGED/none), {number}, {url}
│  │          ({number}/{url} feed the Step 3 MERGED STOP + already-shipped output)
│  ├─ Bash: git symbolic-ref --short refs/remotes/origin/HEAD → {default_branch}
│  │        (fallbacks: gh repo view --json defaultBranchRef, then the probed
│  │         literal — main if refs/remotes/origin/main exists, else master;
│  │         always non-empty)
│  └─ Bash: fab status get-issues {name}   (if {has_fab})
│
├─ (Step 1b removed in 260612-w7dp — the non-blocking mismatch nudge is
│   superseded by Step 0's hard branch-matches-change guard)
│
├─ Step 2: Branch Guard (260612-g8st)
│  ├─ detached HEAD (empty branch) → STOP before any commit/push:
│  │  "Cannot ship from a detached HEAD — check out a branch first"
│  └─ on {default_branch} (resolved; probed fallback keeps the other
│     literal as a safety net) → STOP
│
├─ Step 3: Execute Pipeline (branches on {pr_state} — 260612-g8st:
│  │        MERGED → STOP with new-change/branch guidance, no git mutations;
│  │        CLOSED → proceed, 3c creates a fresh PR;
│  │        OPEN → "already shipped" short-circuit when nothing else to do)
│  ├─ 3a. Commit (if uncommitted)
│  │  ├─ Expected-area guard FIRST (before anything is staged — the
│  │  │  STOP path leaves no staged index): any untracked file outside
│  │  │  source_paths (config.yaml) / docs/ / fab/ → STOP with the
│  │  │  file list
│  │  ├─ Bash: git add -u   (never git add -A)
│  │  ├─ Bash: git add <in-area untracked files from the guard>
│  │  └─ Bash: git commit -m "<message>"
│  ├─ 3a-bis. Refresh Memory Indexes (260620-o203 — if {has_fab} AND 3a just committed)
│  │  ├─ Bash: fab memory-index  (byte-stable; writes only docs/memory/ index + log files)
│  │  └─ Bash: if ! git diff --quiet -- docs/memory; then
│  │           git add docs/memory && git commit -m "docs: refresh memory indexes"
│  │     (no --amend — keeps 3a's content commit; squash collapses on merge;
│  │      first moment git log knows the real commit date; lives in ship not
│  │      hydrate because hydrate is entirely pre-commit; no push here — 3b
│  │      ("if has_unpushed or just committed") pushes both commits together;
│  │      silent no-op when {has_fab} false → standalone /git-pr unaffected;
│  │      no-drift regen → diff guard suppresses an empty commit;
│  │      regen/commit failure → report + STOP, 3a commit intact, no torn state)
│  ├─ 3b. Push (if unpushed)
│  │  └─ Bash: git push [-u origin <branch>]  (pushes 3a + 3a-bis commits together)
│  └─ 3c. Create PR (if no OPEN PR exists — {pr_state} none or CLOSED)
│     ├─ Read: intake.md (PR title + Summary + Changes)
│     ├─ Render ## Meta block (gated on {has_fab}):
│     │  └─ Bash: META=$(fab pr-meta <change> --type <type> --issues "<issues>")
│     │     (subcommand is self-contained: reads .status.yaml, parses plan.md
│     │      (or legacy tasks.md) checkboxes, reads config.yaml
│     │      (true_impact_exclude, test_paths, project.linear_workspace),
│     │      computes impact via internal/impact against the internal merge-base,
│     │      and resolves git/gh context (branch, owner/repo) itself;
│     │      emits the full table + **Pipeline** + optional **Issues** +
│     │      optional **Impact** as final markdown;
│     │      three-row impl/tests/total Impact form when a tests pair exists,
│     │      single total line otherwise; ← excludes annotation from actual
│     │      true_impact_exclude config values, never hardcoded;
│     │      gh failure → plain-text Pipeline labels; missing merge-base or
│     │      +0/−0 total → Impact line dropped;
│     │      non-zero exit / empty stdout → Meta block omitted entirely)
│     ├─ Assemble body: $META verbatim,
│     │                 ## Summary (from intake ## Why), ## Changes (from intake ## What Changes)
│     │                 (Meta block omitted entirely when {has_fab} is false or $META empty)
│     └─ Bash: gh pr create --draft --title --body
│        (body-generation failure → silent gh pr create --draft --fill
│         fallback, evaluated BEFORE the creation-failure STOP — a body
│         failure never reaches it; PR-creation failure → report + STOP)
│
├─ Step 4a: Record PR URL (if {has_fab})
│  └─ Bash: fab status add-pr {name} <url>
│
├─ Step 4b: Finish Ship Stage
│  └─ Bash: fab status finish {name} ship git-pr
│
└─ Step 4c: Commit Status Update
   ├─ Bash: git add fab/changes/{name}/.status.yaml fab/changes/{name}/.history.jsonl
   ├─ Bash: git commit
   └─ Bash: git push
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Intake (for PR title + the agent-generated `## Summary` and `## Changes` sections) |
| Bash | All git operations, gh CLI, fab status commands. Step 3c renders the body's entire `## Meta` block by calling `fab pr-meta <change> --type <type> --issues "<issues>"` and pasting its stdout verbatim — the subcommand is self-contained (reads `.status.yaml`, `plan.md`/`tasks.md`, `config.yaml`, computes impact via `internal/impact`, resolves git/`gh` context) and renders the table, `**Pipeline**`, optional `**Issues**`, and optional `**Impact**` deterministically. When a `tests` pair is present the Impact renders as a three-row impl / tests / total breakdown (`impl = max(0, total − tests)`, per component, never stored; `← excludes …` from the actual `true_impact_exclude` values); otherwise it collapses to the single `total` line. A non-zero exit / empty stdout means the Meta block is omitted. `/git-pr` no longer calls `fab impact` directly. |

### Sub-agents

None.
