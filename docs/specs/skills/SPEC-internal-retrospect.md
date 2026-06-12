# internal-retrospect

## Summary

Reviews the current session end-to-end and produces a retrospective across four areas: (1) scriptable repetition (manual steps repeated 2+ times), (2) skill & prompt quality (skills whose output needed correction or re-runs), (3) context & documentation gaps (wrong assumptions traceable to missing `docs/memory/`, `docs/specs/`, `_preamble.md`, `CLAUDE.md`, or `constitution.md` content), and (4) workflow friction (awkward stage transitions, unnecessary clarification loops). Findings cite actual conversation moments, not generic advice.

Read-only and conversational: the skill writes no files and runs no commands — its output is the retrospective itself. The skill source `src/kit/skills/internal-retrospect.md` is canonical.

## Flow

```
User invokes /internal-retrospect
│
├─ (agent reasoning — reviews the conversation history;
│   no tool calls required)
│
└─ Output
   ├─ Per area: Findings (with citations + suggested actions) or Clean
   ├─ Suggested Actions section, e.g.:
   │  ├─ Run /internal-skill-optimize {skill-file} to condense {Y}
   │  ├─ Add {Z} to docs/memory/{domain}/{name}.md
   │  └─ Add {W} to CLAUDE.md
   └─ [no findings anywhere] "Clean session — no actions needed."
```

### Tools used

None required — pure conversation analysis. (The agent MAY Read cited files to verify a claim before reporting it.)

### Sub-agents

None.

### Bookkeeping commands (hook candidates)

None — no `fab status` transitions, no writes.
