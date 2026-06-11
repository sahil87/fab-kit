# fab-draft

## Summary

Creates a new change intake without activating the change. Identical to `/fab-new` through Step 9, but stops there — no activation, no git branch. Used to queue changes for later without switching the active context. After creation, run `/fab-switch {name}` to activate.

**Re-run contract** (Constitution III): a backlog/Linear-ID re-run detects the existing non-archived change and routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of erroring; a natural-language re-run intentionally creates a new change each run. Declared in the skill's Key Properties section.

**Helpers**: Declares `helpers: [_generation]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

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
│  ├─ [backlog ID detected] collision check first:
│  │  Bash: fab change resolve {id}  (4-char ID is in the folder prefix)
│  ├─ [Linear ID detected] collision check first:
│  │  Bash: grep -l "{ISSUE_ID}" fab/changes/*/.status.yaml
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
├─ Step 6: Infer Change Type
│  └─ Bash: fab status set-change-type <change> <type>    ◄── bookkeeping
│
├─ Step 7: Confidence (authoritative — intake is the sole scoring source)
│  └─ Bash: fab score --stage intake <change>             ◄── bookkeeping (no indicative flag, 1.10.0)
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
