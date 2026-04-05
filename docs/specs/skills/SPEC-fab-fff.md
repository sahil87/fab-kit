# fab-fff

## Summary

Full pipeline with confidence gates (identical to fab-ff). Extends through ship and review-pr (fab-ff stops at hydrate). No frontloaded questions — proceeds directly to spec generation. Interleaves auto-clarify between planning stages. Max 3 rework cycles on review failure with escalation rule. Accepts `--force` to bypass confidence gates.

## Flow

```
User invokes /fab-fff [change-name] [--force]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ Gate 1: Intake Gate (skip if --force)
│  └─ Bash: fab score --check-gate --stage intake <change>
│     └─ STOP if < 3.0
│
├─ Steps 1-7: Same as /fab-ff Steps 1-7 (spec, tasks, checklist, planning complete, apply, review, hydrate)
│  ├─ Gate 2: Spec Gate after spec generation (skip if --force)
│  ├─ fab-clarify dispatched between spec AND tasks
│  └─ Driver argument is "fab-fff" instead of "fab-ff"
│
├─ Step 8: Ship
│  └─ SUB-AGENT: /git-pr (commit, push, create PR)
│
└─ Step 9: Review-PR
   └─ SUB-AGENT: /git-pr-review (process PR review comments)
```

### Sub-agents

Same as fab-ff: /fab-clarify [AUTO-MODE], /fab-continue (Apply, Review, Hydrate), /git-pr, /git-pr-review.

> Step 6 review behavior (inward spec/tasks/checklist validation and outward holistic diff review) is defined in `_review.md`. `/fab-continue` Review Behavior delegates to `_review.md` — the authoritative source for inward + outward sub-agent dispatch and findings merge.

### Bookkeeping commands (hook candidates)

Same as fab-ff.
