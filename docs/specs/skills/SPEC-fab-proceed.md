# fab-proceed
Context-aware orchestrator — detects pipeline state, runs prefix steps (create-intake via `_intake`, `/fab-switch`, `/git-branch`) as subagents, then delegates to `/fab-fff` via the Skill tool. Takes no arguments; idempotent.
## Flow
```
User invokes /fab-proceed
├─ Bash: fab resolve --folder --or-none
│  ├─ folder → branch check: matches → dispatch /fab-fff only / mismatch → dispatch /git-branch → /fab-fff
│  └─ "(none)" → classify conversation substantive vs empty/thin
├─ Scan unactivated intakes; substantive + candidates → relevance assessment (clearly-relevant wins; ambiguous → not relevant; date-descending tiebreak)
├─ Prefix dispatch (subagents): _intake Steps 0–9 in promptless-defer mode (no questions; unresolved decisions deferred, surfaced before /fab-fff) → /fab-switch → /git-branch
└─ Terminal delegation: Skill: /fab-fff (main context, NOT subagent)
```
### Tools used
Bash (`fab resolve --folder --or-none`, `git branch --show-current`, intake scan), Agent (prefix dispatches), Skill (`/fab-fff`).
### Sub-agents
`_intake` (Create-Intake, promptless-defer — stops at `ready`), `/fab-switch` (activate), `/git-branch` (branch) — dispatched per the dispatch table.
