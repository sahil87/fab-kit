# internal-skill-optimize

## Summary

Condenses a skill (or, in batch mode, all skills) to its core — removing verbosity, redundant examples, and concepts re-explained from the shared partials — without losing any behavioral step, error case, or decision point. Runs two kinds of pass with **different scoping**: **content optimization** (the bloat-signal trim) evaluates each target against seven content bloat signals (redundant re-explanation, excessive output examples, obvious instructions, redundant argument docs, over-specified error tables, verbose step narration, duplicate examples) and applies ten optimization rules; **structural checks** add a `## Contents` table-of-contents to any file over 100 lines and report (never auto-fix) reference chains deeper than one level.

The two passes scope differently: **content optimization never touches a `_*.md` partial** (a partial is shared reference context, never a trim target) and skips files under 80 lines, whereas **structural checks run on all files including partials and sub-80-line files** — a long partial with no TOC (e.g. a 700+ line `_cli-fab.md`) or a deeply-nested reference is a real structural defect the content rule would wrongly exempt; a structural pass only adds a Contents block or reports a depth chain, never trims a partial's prose. The partial set is derived by globbing `src/kit/skills/_*.md`, never from a hardcoded list (a list drifts as partials are added). Writes require explicit user approval (AskUserQuestion) in both modes; depth findings are report-only (flattening a chain is a separate content-moving change). Targets are the canonical sources at `src/kit/skills/` — never the deployed `.claude/skills/` copies. The skill source `src/kit/skills/internal-skill-optimize.md` is canonical.

## Flow

```
User invokes /internal-skill-optimize [<skill-name>]
│
├─ Pre-flight
│  ├─ Read: _*.md partials — set derived by globbing
│  │        src/kit/skills/_*.md (reference context, not targets)
│  └─ [named skill missing] STOP:
│     "Skill not found: src/kit/skills/{skill-name}.md"
│
├─ Single skill mode (<skill-name> given)
│  ├─ Read: src/kit/skills/{skill-name}.md
│  ├─ Content analysis (skip if _*.md partial) + structural
│  │  checks (TOC > 100 lines, reference depth — run on partials too)
│  ├─ Produce before/after line count + change summary +
│  │  TOC action + depth findings (report-only)
│  ├─ AskUserQuestion: "Apply these optimizations to {skill-name}?"
│  └─ [approved] Write: optimized file (trim + TOC; depth reported only)
│
└─ Batch mode (no argument)
   ├─ Read: all src/kit/skills/*.md, sorted by line count descending
   ├─ Content trim skips files under 80 lines ("Already lean") AND
   │  skips _*.md partials; STRUCTURAL checks run on EVERY file
   │  (incl. partials + sub-80-line files)
   ├─ Present consolidated summary table
   │  (| Skill | Before | After | Reduction | TOC | Depth findings |)
   ├─ List report-only depth chains ({file} → {mid} → {leaf}) below the table
   ├─ Ask: "Apply all optimizations, or select specific skills?"
   └─ [approved] Write: all approved files (trims + TOC; depth chains left for maintainer)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Partials (reference), target skill files |
| Write | Optimized skill files (only after user approval) |
| AskUserQuestion | Approval gate before any write |

### Sub-agents

None.

### Constraints (mirror of the skill's own list)

- Never change logical behavior, remove error handling, or move content between skills
- **Content optimization** never touches a `_*.md` partial; **structural checks are the sole exception** — TOC insertion may add a `## Contents` block to a partial, and the depth check may report on one (neither trims a partial's prose)
- The reference-depth check is **report-only** — never restructures or moves content (flattening a deep chain is a separate change)
- Files under 80 lines are skipped for **content** optimization (reported "Already lean") but still receive **structural** checks

### Bookkeeping commands (hook candidates)

None — no `fab status` transitions. Edits to `src/kit/skills/*.md` trigger the constitution's SPEC-mirror rule for the affected skills.
