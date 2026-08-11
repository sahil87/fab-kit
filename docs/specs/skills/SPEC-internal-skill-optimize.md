# internal-skill-optimize
Condenses a skill (or all consumer skills in batch mode) without losing behavior, error cases, or decision points: content trim plus structural checks (TOC over 100 lines, report-only reference-depth audit). Writes require approval; targets are `src/kit/skills/`.
## Flow
```
User invokes /internal-skill-optimize [<skill-name>]
├─ Pre-flight: Read _*.md partials; [named skill missing] STOP
├─ Single: Read target → content + structural analysis → AskUserQuestion → [approved] Write optimized file
└─ Batch: Read all skills → content trim skips <80-line files + partials (structural checks on all) → summary table → approval → Write approved files
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read/Write | Partials + target files; approved optimized files |
| AskUserQuestion | Approval gate before any write |
### Sub-agents
None.
