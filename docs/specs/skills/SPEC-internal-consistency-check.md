# internal-consistency-check

## Summary

Scans for inconsistencies between the three sources of truth — implementation (paths from `source_paths` in `fab/project/config.yaml`), memory (`docs/memory/`), and specs (`docs/specs/`) — by dispatching three parallel read-only audit agents, then synthesizing their reports into a summary table, critical/minor findings, and suggested actions grouped by Fix / Add / Remove / Rename.

Read-only with respect to the repo: the skill reports findings and suggestions; it never applies fixes itself. The skill source `src/kit/skills/internal-consistency-check.md` is canonical.

## Flow

```
User invokes /internal-consistency-check
│
├─ Pre-flight
│  ├─ Read: fab/project/config.yaml (extract source_paths)
│  └─ [source_paths missing/empty] STOP:
│     "No source_paths defined in fab/project/config.yaml."
│
├─ Parallel Dispatch (Task tool, subagent_type: Explore,
│  │                  thoroughness: very thorough)
│  ├─ Agent 1: Specs ↔ Implementation
│  │  (missing/undocumented implementations, naming mismatches,
│  │   behavioral contradictions, stale references)
│  ├─ Agent 2: Memory ↔ Implementation
│  │  (stale/missing memory, wrong paths, contradicted behavior,
│  │   orphaned memory)
│  └─ Agent 3: Specs ↔ Memory
│     (terminology drift, coverage gaps, contradictions,
│      stale cross-references, glossary drift)
│
└─ Synthesis (agent reasoning — no writes)
   ├─ 1. Summary table (findings / critical / minor per dimension)
   ├─ 2. Critical findings
   ├─ 3. Minor findings
   └─ 4. Suggested actions (Fix / Add / Remove / Rename)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | `fab/project/config.yaml` |
| Agent | Three parallel `Explore` sub-agents (read-only audits) |

### Sub-agents

| Agent | Purpose |
|-------|---------|
| Specs ↔ Implementation | Audit `docs/specs/` against `source_paths` directories |
| Memory ↔ Implementation | Audit `docs/memory/` against `source_paths` directories |
| Specs ↔ Memory | Audit `docs/specs/` against `docs/memory/` |

### Classification

- **Critical**: implementation contradicts spec; memory instructs something that fails; referenced file/command/path doesn't exist.
- **Minor**: naming mismatch without behavioral impact; coverage gap; stale reference without user confusion; orphaned content.

### Bookkeeping commands (hook candidates)

None — the skill performs no `fab status` transitions and no file writes.
