# docs-reorg-specs

## Summary

Analyzes spec files for themes and suggests reorganization. Read-only unless user approves. Same pattern as docs-reorg-memory but targeting `docs/specs/`. Scanning recurses into subfolders (e.g. `skills/`, `findings/`), but `docs/specs/skills/SPEC-*.md` mirrors are **reserved paths** — constitution-pinned names derived from their `src/kit/skills/` sources — read for theme analysis only, never renamed/moved/merged/split.

**No compatibility/backfill step.** Unlike `docs-reorg-memory` (which detects pre-fab-kit memory trees missing `description:` frontmatter and orchestrates a backfill), `docs-reorg-specs` has **no** compatibility or frontmatter-backfill step, intentionally: there is no specs-index generator (no counterpart to `fab memory-index`), the specs index is hand-rewritten (Step 5), and Constitution VI keeps specs human-curated — so a spec missing frontmatter breaks nothing and there is no compatibility contract to violate. The skill carries an explicit note so a future contributor does not "fix the asymmetry."

## Flow

```
User invokes /docs-reorg-specs
│
├─ Pre-flight: docs/specs/index.md and spec files must exist
├─ Read: all spec files (recursing into subfolders: skills/, findings/)
├─ (identify themes, propose reorganization — never migrating reserved docs/specs/skills/SPEC-*.md)
├─ (present plan, ask for approval)
│
└─ [if approved]
   ├─ Write/Edit: reorganized spec files
   └─ Edit: docs/specs/index.md
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | All spec files and index |
| Write/Edit | Reorganized files (only with approval) |

### Sub-agents

None.
