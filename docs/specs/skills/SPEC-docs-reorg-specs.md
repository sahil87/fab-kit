# docs-reorg-specs

## Summary

Analyzes spec files for themes and suggests reorganization. Read-only unless user approves. Same pattern as docs-reorg-memory but targeting `docs/specs/`. Scanning recurses into subfolders (e.g. `skills/`, `findings/`), but `docs/specs/skills/SPEC-*.md` mirrors are **reserved paths** — constitution-pinned names derived from their `src/kit/skills/` sources — read for theme analysis only, never renamed/moved/merged/split.

**No compatibility/backfill step.** Unlike `docs-reorg-memory` (which detects pre-fab-kit memory trees missing `description:` frontmatter and orchestrates a backfill), `docs-reorg-specs` has **no** compatibility or frontmatter-backfill step, intentionally: there is no specs-index generator (no counterpart to `fab memory-index`), the specs index is hand-rewritten (Step 5), and a fab-kit design principle keeps specs human-curated — so a spec missing frontmatter breaks nothing and there is no compatibility contract to violate. The skill carries an explicit note so a future contributor does not "fix the asymmetry."

**No FKF frontmatter on spec moves (specs are human-curated — a fab-kit design principle).** FKF (`type: memory` + `description:`) governs `docs/memory/` only; specs are out of FKF scope and stay frontmatter-free, human-curated per that **fab-kit design principle**. When the skill moves a spec file it MUST NOT stamp, add, or synthesize `type:` / `description:` frontmatter — a moved spec carries exactly its prior bytes (only its path and the `index.md` row change). This is the deliberate mirror of `docs-reorg-memory`'s frontmatter-*preserving* moves: memory moves keep FKF frontmatter, spec moves add none. A generated-index model for `docs/specs/index.md` is **not adopted** (specs are human-curated — a fab-kit design principle) — no `fab specs-index` generator; the specs index stays hand-rewritten and spec links stay ordinary repo-relative.

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
   ├─ Write/Edit: reorganized spec files (moved bytes verbatim — no FKF frontmatter stamped, specs are human-curated per a fab-kit design principle)
   └─ Edit: docs/specs/index.md (hand-rewritten — no specs-index generator)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | All spec files and index |
| Write/Edit | Reorganized files (only with approval) |

### Sub-agents

None.
