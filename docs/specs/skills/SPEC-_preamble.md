# _preamble
Shared context preamble loaded by every Fab skill — path/context/helper conventions, per-stage profile resolution, the cross-adapter dispatch procedure, worker continuation, pane readiness gate, confidence scoring. Internal partial; never invoked directly.
## Flow
```
Skill reads _preamble.md
├─ Context Loading (4 layers): always-load → change context (fab preflight, fab log command) → memory lookup (≤3-hop) → source code
├─ Conventions: helpers: frontmatter, naming, rk reference, common fab commands, next steps
├─ Subagent Dispatch
│  ├─ Dispatch pattern + Standard Subagent Context
│  ├─ Per-stage model resolution: fab resolve-agent <stage> --alias → branch on dispatch= (absent ⇒ native Agent / present ⇒ fab dispatch CLI adapter)
│  ├─ Worker continuation (apply rework only): native SendMessage / pane deliver / else fresh
│  ├─ CLI-adapter dispatch: fab dispatch start → wait; pane open → ready gate → deliver; stage-aware reap
│  └─ Dispatch-prompt obligations (all adapters): {stage}-result.yaml, context files, terminal fab status refresh, no status TRANSITIONs
├─ SRAD Autonomy Framework (pointer → _srad.md)
└─ Confidence scoring — fab score <change>
```
### Tools used
Read context layer files; Bash `fab preflight`, `fab log command`, `fab resolve-agent`, `fab score`.
### Sub-agents
None — conventions only; dispatch patterns are defined here but executed by the consuming skill.
