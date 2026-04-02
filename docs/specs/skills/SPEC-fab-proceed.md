# fab-proceed

## Summary

Context-aware orchestrator — detects pipeline state via a 4-step detection pipeline, runs prefix steps (fab-new, fab-switch, git-branch) as subagents, then delegates to `/fab-fff` via the Skill tool. No arguments, no flags — infers everything from context. Idempotent — re-running detects completed steps and skips them. Does not load `_preamble.md` or run preflight.

## Flow

```
User invokes /fab-proceed
│
├─ Step 1: Active Change Check
│  └─ Bash: fab resolve --folder 2>/dev/null
│     ├─ exits 0 → active change found, go to Step 2
│     └─ exits non-zero → go to Step 3
│
├─ Step 2: Branch Check (only if active change found)
│  └─ Bash: git branch --show-current
│     ├─ matches change name → dispatch /fab-fff only
│     └─ does not match → dispatch /git-branch → /fab-fff
│
├─ Step 3: Unactivated Intake Check (only if no active change)
│  └─ Scan fab/changes/ (exclude archive/) for intake.md
│     ├─ exactly one → use it
│     ├─ multiple → select most recent by YYMMDD prefix
│     └─ none → go to Step 4
│  └─ Dispatch: /fab-switch → /git-branch → /fab-fff
│
├─ Step 4: Conversation Context Check (only if no intakes)
│  ├─ substantive discussion → synthesize description
│  │  └─ Dispatch: /fab-new → /fab-switch → /git-branch → /fab-fff
│  └─ no context → STOP: "Nothing to proceed with"
│
├─ Prefix Dispatch (subagents)
│  ├─ ┌──────────────────────────────────────────┐
│  │  │ SUB-AGENT: /fab-new (if needed)          │
│  │  │  Read: src/kit/skills/fab-new.md        │
│  │  │  Input: synthesized description          │
│  │  │  Returns: created change folder name     │
│  │  └──────────────────────────────────────────┘
│  ├─ ┌──────────────────────────────────────────┐
│  │  │ SUB-AGENT: /fab-switch (if needed)       │
│  │  │  Read: src/kit/skills/fab-switch.md     │
│  │  │  Bash: fab change switch "<change-name>" │
│  │  │  Returns: switch confirmation            │
│  │  └──────────────────────────────────────────┘
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /git-branch (if needed)       │
│     │  Read: src/kit/skills/git-branch.md     │
│     │  Returns: branch action result           │
│     └──────────────────────────────────────────┘
│
└─ Terminal Delegation (Skill tool, NOT subagent)
   └─ Skill: /fab-fff
      └─ Runs in main context with full user visibility
```

### Dispatch Table

| Detected state | Prefix steps | Terminal |
|----------------|--------------|----------|
| Active change + matching branch | (none) | /fab-fff |
| Active change + no matching branch | /git-branch | /fab-fff |
| Unactivated intake | /fab-switch → /git-branch | /fab-fff |
| Conversation context, no intake | /fab-new → /fab-switch → /git-branch | /fab-fff |
| No context, no intake | (error — stop) | — |

### Sub-agents

| Agent | When | Purpose |
|-------|------|---------|
| /fab-new | Conversation context exists, no intake | Create change from synthesized description |
| /fab-switch | Unactivated intake or newly created change | Activate the change |
| /git-branch | Active change without matching branch | Create or checkout the matching branch |

### Key differences from /fab-fff and /fab-ff

- Does NOT load `_preamble.md` or run preflight — delegates that to `/fab-fff`
- Does NOT accept arguments or flags — infers everything from state detection
- Prefix steps are subagents; terminal `/fab-fff` is via Skill tool (not subagent)
