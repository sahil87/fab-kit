# git-pr

## Summary

Autonomously commits, pushes, and creates a draft GitHub PR. No prompts, no questions. Resolves PR type from status/intake/diff. Generates PR body from fab artifacts when available. Records PR URL in `.status.yaml`.

**Re-run contract** (Constitution III, declared in the skill's Key Properties section): re-run after ship is a no-op — the "already shipped" path (no uncommitted changes, no unpushed commits, PR exists) re-records the existing PR URL silently and stops; `fab status add-pr` is idempotent and the Step 4c status commit is guarded by `git diff --cached --quiet`.

## Flow

```
/git-pr invoked (user or sub-agent)
│
├─ Step 0: Resolve Change Context (260611-szxd f094 — the ONLY
│  │        fab change resolve in the skill; later steps reference
│  │        the variables and MUST NOT re-resolve)
│  ├─ Bash: fab change resolve 2>/dev/null → {has_fab}, {name}
│  ├─ {has_intake} — fab/changes/{name}/intake.md exists?
│  └─ {change_type} — from fab/changes/{name}/.status.yaml
│
├─ Step 0a: Start Ship Stage (if {has_fab})
│  └─ Bash: fab status start <change> ship git-pr
│
├─ Step 0b: Resolve PR Type
│  ├─ {change_type} from Step 0 (status source)
│  ├─ Read: fab/changes/{name}/intake.md (keyword match, if {has_intake})
│  └─ Bash: git diff --name-only (fallback)
│
├─ Step 1: Gather State
│  ├─ Bash: git branch --show-current
│  ├─ Bash: git status --porcelain
│  ├─ Bash: git log --oneline -5
│  ├─ Bash: git log --oneline @{u}..HEAD
│  ├─ Bash: gh pr view --json
│  └─ Bash: fab status get-issues {name}   (if {has_fab})
│
├─ Step 1b: Branch Mismatch Nudge (compares branch vs {name}; skip if !{has_fab})
│
├─ Step 2: Branch Guard (STOP if main/master)
│
├─ Step 3: Execute Pipeline
│  ├─ 3a. Commit (if uncommitted)
│  │  ├─ Bash: git add -A
│  │  └─ Bash: git commit -m "<message>"
│  ├─ 3b. Push (if unpushed)
│  │  └─ Bash: git push [-u origin <branch>]
│  └─ 3c. Create PR (if no PR exists)
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
│
├─ Step 4a: Record PR URL (if {has_fab})
│  └─ Bash: fab status add-pr {name} <url>
│
├─ Step 4b: Finish Ship Stage
│  └─ Bash: fab status finish <change> ship git-pr
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
