# docs-distill-memory
Rewrites an existing `docs/memory/` domain's topic files to the FKF present-truth style. `<domain>` named forces a full read of that one domain; omitted runs a heuristic survey and loops every flagged domain — read-only until per-domain approval.
## Flow
```
User invokes /docs-distill-memory [<domain>]
├─ [omitted] Survey: Bash: fab memory-index --check --json → flagged-domain worklist → per-domain loop in main session; [none flagged] STOP
├─ [given] skip survey, full read of that one domain (no loop); Pre-flight: index.md + ≥1 topic file
├─ Read: domain files + $(fab kit-path)/reference/fkf.md → classify (narration / superseded prose / description: defects / duplicates / DD changelog bullets / TODOs / rationale)
├─ Report → approval gate → [declined] stop, no mutation
└─ [approved] Edit rewrites; relocate TODOs → fab/backlog.md; then once: Bash: fab memory-index --check → fab memory-index; Next: remaining flagged domains
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read | Domain files + fkf.md reference; survey JSON output |
| Edit/Write | Approved present-truth rewrites; TODO relocation into `fab/backlog.md` |
| Bash | `fab memory-index --check --json` survey; refuse-guarded `--check` + `fab memory-index` regen |
### Sub-agents
None — runs inline, including the no-arg all-domains loop.
