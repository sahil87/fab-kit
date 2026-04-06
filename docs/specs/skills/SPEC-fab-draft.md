# fab-draft

## Summary

Creates a new change intake without activating the change. Identical to `/fab-new` through Step 9, but stops there — no activation, no git branch. Used to queue changes for later without switching the active context. After creation, run `/fab-switch {name}` to activate.

## Flow

```
User invokes /fab-draft <description>
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
├─ Step 7: Indicative Confidence
│  └─ Bash: fab score --stage intake <change>             ◄── bookkeeping
│
├─ Step 8: SRAD Questions
│  └─ (agent reasoning, possible user interaction)
│
└─ Step 9: Advance Intake to Ready
   └─ Bash: fab status advance <change> intake
   (change is NOT activated — no .fab-status.yaml symlink created)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Load preamble, templates, backlog, project files |
| Write | Write `intake.md` |
| Bash | `fab change new`, `fab status set-change-type`, `fab score`, `fab status advance`, `fab status add-issue` |
| MCP (Linear) | Fetch issue details (optional path) |

### Sub-agents

None.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| 6 | `fab status set-change-type` | After intake.md write |
| 7 | `fab score --stage intake` | After intake.md write |
| 9 | `fab status advance` | After all intake work complete |

### Difference from /fab-new

`/fab-draft` omits Steps 10 and 11 from `/fab-new`:
- **No Step 10** — change is not activated (`.fab-status.yaml` symlink is not created)
- **No Step 11** — git branch is not created

The output `Next:` line uses the activation preamble: `/fab-switch {name} to make it active, then /fab-continue, /fab-fff, /fab-ff, or /fab-clarify`.
