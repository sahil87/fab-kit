# internal-consistency-check
Scans for inconsistencies between implementation (`source_paths` from `fab/project/config.yaml`), memory, and specs via three parallel read-only audit agents, then synthesizes a summary table, critical/minor findings, and Fix/Add/Remove/Rename actions. Read-only — reports, never fixes.
## Flow
```
User invokes /internal-consistency-check
├─ Pre-flight: Read config.yaml source_paths (missing → STOP)
├─ Parallel dispatch: 3 Explore agents — Specs↔Impl, Memory↔Impl, Specs↔Memory
└─ Synthesis (no writes): summary table → critical findings → minor findings → suggested actions
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read | `fab/project/config.yaml` |
| Agent | Three parallel Explore audits |
### Sub-agents
Three parallel Explore agents: Specs↔Implementation, Memory↔Implementation, Specs↔Memory (read-only audits).
