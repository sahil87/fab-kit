# fab-new

## Summary

Creates a new change from a natural language description, Linear ticket, or backlog ID. Generates the change folder, writes `intake.md`, verifies the hook-inferred change type (the PostToolUse intake-write hook owns `change_type`; the skill overrides via `set-change-type` only if wrong), computes the authoritative intake confidence (no `indicative` flag — 1.10.0), advances intake to `ready`, activates the change, and creates the matching git branch.

**Re-run contract** (Constitution III): a backlog/Linear-ID re-run detects the existing non-archived change and routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of erroring; a natural-language re-run intentionally creates a new change each run. Declared in the skill's Key Properties section.

**Helpers**: Declares `helpers: [_generation, _srad]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

## Flow

```
User invokes /fab-new <description>
│
├─ Read: _preamble.md (always-load layer: 7 project files)
│
├─ Step 0: Parse Input
│  ├─ Linear ID? ──► MCP: mcp__claude_ai_Linear__get_issue
│  ├─ Backlog ID? ──► Read: fab/backlog.md
│  └─ Natural language ──► use as-is
│
├─ Step 1: Generate Slug
│  └─ (agent reasoning — no tools)
│
├─ Step 2: Gap Analysis
│  └─ Read/Grep: existing skills, specs, memory
│
├─ Step 3: Create Change
│  ├─ [backlog ID detected] collision check first:
│  │  Bash: fab change resolve {id}  (4-char ID is in the folder prefix)
│  ├─ [Linear ID detected] collision check first:
│  │  Bash: grep -lw "{ISSUE_ID}" fab/changes/*/.status.yaml
│  │  (-w word-anchors: DEV-123 won't match DEV-1234)
│  │  (Linear IDs never appear in folder names — they live in
│  │   .status.yaml issues arrays; the single-level glob
│  │   naturally excludes archive/)
│  ├─ [existing non-archived change found by either check]
│  │  → route to resume: report it + point to
│  │    /fab-switch {name} then /fab-continue — STOP
│  │    (no duplicate created; `Change ID already in use`
│  │     stays as safety net for backlog IDs only — Linear
│  │     re-runs pass no --change-id, so the scan is the
│  │     only collision guard)
│  │  (NL re-run intentionally creates a new change each run)
│  └─ Bash: fab change new --slug <slug> --log-args <desc>
│     └─ (creates folder, .status.yaml from template)
│  └─ [if Linear] Bash: fab status add-issue <change> <id>
│
├─ Step 4: Conversation Context Mining
│  └─ (agent reasoning — scans conversation history)
│
├─ Step 5: Generate intake.md
│  ├─ Read: $(fab kit-path)/templates/intake.md
│  └─ Write: fab/changes/{name}/intake.md          ◄── HOOK CANDIDATE
│
├─ Step 6: Verify Change Type (hook-owned — the intake-write
│  │        hook already set it in Step 5's Write)
│  ├─ Bash: grep '^change_type:' fab/changes/{name}/.status.yaml
│  └─ [only if wrong] Bash: fab status set-change-type <change> <type>
│
├─ Step 7: Confidence (authoritative — intake is the sole scoring source)
│  └─ Bash: fab score --stage intake <change>             ◄── bookkeeping (no indicative flag, 1.10.0)
│
├─ Step 8: SRAD Questions
│  └─ (agent reasoning, possible user interaction)
│
├─ Step 9: Advance Intake to Ready
│  └─ Bash: fab status advance <change> intake
│
├─ Step 10: Activate Change
│  └─ Bash: fab change switch "{name}"
│
└─ Step 11: Create Git Branch (single first-match-wins table —
   │         260611-szxd f032; kept in sync with git-branch.md Step 4
   │         via an in-file comment; same five cases, commands, and
   │         report strings)
   ├─ Bash: git rev-parse --is-inside-work-tree   (repo check — skip if fails)
   ├─ Context reads: git branch --show-current ·
   │  git rev-parse --verify "{name}" ·
   │  git config branch.{current}.remote ·
   │  fab change resolve "$(git branch --show-current)"
   └─ Evaluate the 5-row table in order, first match wins:
      already-on-target (no-op) / target-exists (checkout) /
      on-main (checkout -b) / local-only + rename guard passes
      (branch -m) / other-change's local-only branch or pushed
      branch (checkout -b, leaving {old_branch} intact)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Load preamble, templates, backlog, project files |
| Write | Write `intake.md` |
| Bash | `fab change new`, `fab status set-change-type` (override only), `fab score`, `fab status advance`, `fab status add-issue`, `fab change switch` |
| Bash (git) | `git rev-parse --is-inside-work-tree`, `git branch --show-current`, `git rev-parse --verify`, `git config branch.{current}.remote`, `git checkout -b`, `git checkout`, `git branch -m` |
| MCP (Linear) | Fetch issue details (optional path) |

### Sub-agents

None.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| 6 | `fab status set-change-type` | Only if the hook-inferred type is wrong (the intake-write hook owns `change_type`) |
| 7 | `fab score --stage intake` | After intake.md write |
| 9 | `fab status advance` | After all intake work complete |
| 10 | `fab change switch` | After intake advanced to ready |
| 11 | `git checkout -b` / `git checkout` / `git branch -m` | After change activated |
