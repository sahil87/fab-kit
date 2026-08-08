# internal-skill-optimize

## Summary

Condenses a skill (or, in batch mode, all consumer skills) to its core — removing verbosity, redundant examples, and concepts re-explained from the shared partials — without losing any behavioral step, error case, or decision point. Runs two kinds of pass with **different scoping**: **content optimization** (the bloat-signal trim) evaluates each target against nine content bloat signals (redundant re-explanation, transition narration, excessive output examples, obvious instructions, redundant argument docs, over-specified error tables, verbose step narration, duplicate examples, sibling duplication) and applies ten optimization rules; **structural checks** add a `## Contents` table-of-contents to any file over 100 lines and report (never auto-fix) reference chains deeper than one level. Transition narration is rewritten as present truth per `$(fab kit-path)/reference/fkf.md` §3.3, preserving the current prohibition and rationale while dropping history. In every content pass, the sibling-duplication signal compares the analyzed file with the already-loaded `_*.md` partials, so it operates only on files loaded for that pass. The self-contained ownership rule is that a file may state a rule it owns or point at the owner, never both; all duplication findings remain report-only.

The two passes scope differently: a consumer-skill or batch **content optimization** pass treats `_*.md` partials as read-only shared reference context, while a dedicated pass invoked explicitly with a partial's name (for example, `/internal-skill-optimize _preamble`) legitimately applies the same content signals to that partial. Batch content optimization also skips files under 80 lines. **Structural checks run on all files including partials and sub-80-line files** — a long partial with no TOC (e.g. a 700+ line `_cli-fab.md`) or a deeply-nested reference is a real structural defect; a structural pass only adds a Contents block or reports a depth chain. The partial set is derived by globbing `src/kit/skills/_*.md`, never from a hardcoded list (a list drifts as partials are added). Optimization Rule 6 limits illustrative output examples only: literals that the skill or a sibling greps/string-matches are protected contracts, including `git-pr`'s `✓ commit`/`✓ push`/`✓ pr`/`✓ meta`/`✓ status` lines and `git-pr-review`'s `Fixed —`/`Deferred —`/`Skipped —` prefixes. Writes require explicit user approval (AskUserQuestion) in both modes; duplication and depth findings are report-only (pointer reduction, extraction, or flattening is a separate content-moving change). Targets are the canonical sources at `src/kit/skills/` — never the deployed `.claude/skills/` copies. The skill source `src/kit/skills/internal-skill-optimize.md` is canonical.

## Flow

```
User invokes /internal-skill-optimize [<skill-name>]
│
├─ Pre-flight
│  ├─ Read: _*.md partials — set derived by globbing
│  │        src/kit/skills/_*.md (consumer/batch: read-only context;
│  │        explicitly named partial: dedicated optimization target)
│  └─ [named skill missing] STOP:
│     "Skill not found: src/kit/skills/{skill-name}.md"
│
├─ Single skill mode (<skill-name> given)
│  ├─ Read: src/kit/skills/{skill-name}.md
│  ├─ Content analysis against loaded partials (including an explicitly
│  │  named partial; no sibling-skill comparison) + structural checks
│  │  (TOC > 100 lines, reference depth — partials too)
│  ├─ Produce before/after line count + change summary +
│  │  TOC action + duplication/depth findings (report-only)
│  ├─ AskUserQuestion: "Apply these optimizations to {skill-name}?"
│  └─ [approved] Write: optimized file (trim + TOC; findings reported only)
│
└─ Batch mode (no argument)
   ├─ Read: all src/kit/skills/*.md, sorted by line count descending
   ├─ Content trim skips files under 80 lines ("Already lean — skipped") AND
   │  skips _*.md partials; STRUCTURAL checks run on EVERY file
   │  (incl. partials + sub-80-line files)
   ├─ Per-file analysis detects consumer↔loaded-partial duplication
   ├─ Compare analyzed skills for cross-skill/twin clusters (batch only)
   ├─ Present consolidated summary table
   │  (| Skill | Before | After | Reduction | TOC | Depth findings |)
   ├─ List report-only duplication findings and depth chains below the table
   ├─ Ask: "Apply all optimizations, or select specific skills?"
   └─ [approved] Write: all approved files (trims + TOC; duplication and
      depth findings left for the maintainer)
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
- **Content optimization** never trims a `_*.md` partial as a side effect of a consumer or batch pass; an explicitly named partial is a legitimate dedicated content-optimization target
- Structural checks always include partials; TOC insertion may add a `## Contents` block and the depth check may report a chain
- Duplication and reference-depth findings are **report-only** — never reduce pointers, restructure, extract, or move content (pointer reduction/flattening/extraction is a separate change)
- Files under 80 lines are skipped for **content** optimization (reported "Already lean — skipped") but still receive **structural** checks

### Bookkeeping commands (hook candidates)

None — no `fab status` transitions. Edits to `src/kit/skills/*.md` trigger the constitution's SPEC-mirror rule for the affected skills.
