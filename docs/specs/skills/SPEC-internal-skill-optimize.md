# internal-skill-optimize

## Summary

Condenses a skill (or, in batch mode, all skills) to its core — removing verbosity, redundant examples, and concepts re-explained from the shared partials — without losing any behavioral step, error case, or decision point. Evaluates each target against seven bloat signals (redundant re-explanation, excessive output examples, obvious instructions, redundant argument docs, over-specified error tables, verbose step narration, duplicate examples) and applies eight optimization rules (never remove functionality, preserve frontmatter, preserve H1 + preamble reference, reference shared docs, merge small steps, one output example max, keep error tables, preserve tone).

All `_*.md` partials (`_preamble`, `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`) are reference context, never optimization targets. Writes require explicit user approval (AskUserQuestion) in both modes. Targets are the canonical sources at `src/kit/skills/` — never the deployed `.claude/skills/` copies. The skill source `src/kit/skills/internal-skill-optimize.md` is canonical.

## Flow

```
User invokes /internal-skill-optimize [<skill-name>]
│
├─ Pre-flight
│  ├─ Read: .claude/skills/ _*.md partials (reference context,
│  │        not targets)
│  └─ [named skill missing] STOP:
│     "Skill not found: src/kit/skills/{skill-name}.md"
│
├─ Single skill mode (<skill-name> given)
│  ├─ Read: src/kit/skills/{skill-name}.md
│  ├─ Produce before/after line count + change summary
│  ├─ AskUserQuestion: "Apply these optimizations to {skill-name}?"
│  └─ [approved] Write: optimized file
│
└─ Batch mode (no argument)
   ├─ Read: all src/kit/skills/*.md except _*.md partials,
   │        sorted by line count descending
   ├─ Skip files under 80 lines ("Already lean — skipped")
   ├─ Present consolidated summary table
   │  (| Skill | Before | After | Reduction | Key changes |)
   ├─ Ask: "Apply all optimizations, or select specific skills?"
   └─ [approved] Write: all approved files
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
- Never touch any `_*.md` partial — they are the reference, not the target
- Files already under 80 lines are reported as lean and skipped

### Bookkeeping commands (hook candidates)

None — no `fab status` transitions. Edits to `src/kit/skills/*.md` trigger the constitution's SPEC-mirror rule for the affected skills.
