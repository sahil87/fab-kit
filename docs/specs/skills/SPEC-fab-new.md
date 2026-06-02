# fab-new

## Summary

Creates a new change from a natural language description, Linear ticket, or backlog ID. Generates the change folder, writes `intake.md`, infers change type, computes the authoritative intake confidence (no `indicative` flag — 1.10.0), advances intake to `ready`, activates the change, and creates the matching git branch.

**Helpers**: Declares `helpers: [_generation]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

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
├─ Step 6: Infer Change Type
│  └─ Bash: fab status set-change-type <change> <type>    ◄── bookkeeping
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
└─ Step 11: Create Git Branch
   ├─ Bash: git rev-parse --is-inside-work-tree   (repo check — skip if fails)
   ├─ Bash: git branch --show-current
   ├─ Bash: git rev-parse --verify "{name}"        (target exists check)
   ├─ Bash: git config branch.{current}.remote     (upstream check)
   └─ Bash: git checkout -b / git checkout / git branch -m   (per case)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Load preamble, templates, backlog, project files |
| Write | Write `intake.md` |
| Bash | `fab change new`, `fab status set-change-type`, `fab score`, `fab status advance`, `fab status add-issue`, `fab change switch` |
| Bash (git) | `git rev-parse --is-inside-work-tree`, `git branch --show-current`, `git rev-parse --verify`, `git config branch.{current}.remote`, `git checkout -b`, `git checkout`, `git branch -m` |
| MCP (Linear) | Fetch issue details (optional path) |

### Sub-agents

None.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| 6 | `fab status set-change-type` | After intake.md write |
| 7 | `fab score --stage intake` | After intake.md write |
| 9 | `fab status advance` | After all intake work complete |
| 10 | `fab change switch` | After intake advanced to ready |
| 11 | `git checkout -b` / `git checkout` / `git branch -m` | After change activated |
