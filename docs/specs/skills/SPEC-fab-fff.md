# fab-fff

## Summary

Full pipeline with no confidence gates. Same stages as fab-ff but frontloads all SRAD questions in a single batch, interleaves auto-clarify between planning stages, and forces through regardless of scores. Max 3 rework cycles on review failure with escalation rule.

## Flow

```
User invokes /fab-fff [change-name]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ Step 1: Frontload Questions
│  └─ (SRAD analysis → batch Unresolved → 1 Q&A round)
│
├─ Steps 2-10: Identical to /fab-ff except:
│  ├─ No intake gate (Step 1 of fab-ff)
│  ├─ No spec gate (Step 1 of fab-ff)
│  ├─ fab-clarify dispatched between spec AND tasks
│  └─ Driver argument is "fab-fff" instead of "fab-ff"
│
└─ (see fab-ff.md for full pipeline diagram)
```

### Sub-agents

Same as fab-ff: /fab-clarify [AUTO-MODE], /fab-continue (Apply, Review, Hydrate), /git-pr, /git-pr-review.

### Bookkeeping commands (hook candidates)

Same as fab-ff.
