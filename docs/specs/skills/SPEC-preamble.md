# _preamble

## Summary

Shared context preamble loaded by every Fab skill. Defines path conventions, test-build guard, context loading layers (always-load, change context, memory lookup, source code), next-steps convention with state table, skill invocation protocol, subagent dispatch pattern with standard subagent context, SRAD autonomy framework, and confidence scoring.

This is an internal partial (`user-invocable: false`) — it is never invoked directly. Skills reference it via the opening instruction: "Read `src/kit/skills/_preamble.md` first."

## Flow

```
Skill reads _preamble.md
│
├─ Path Convention
│  (all paths relative to repo root)
│
├─ Test-Build Guard
│  Read: kit.conf (removed)
│  [if build-type=test]
│    Bash: fab preflight
│    STOP
│
├─ Context Loading
│  ├─ Layer 1: Always Load
│  │  Read: config.yaml, constitution.md,
│  │        context.md*, code-quality.md*,
│  │        code-review.md*, memory/index.md,
│  │        specs/index.md, _cli-fab.md, _naming.md
│  │
│  ├─ Layer 2: Change Context
│  │  Bash: fab preflight [change-name]
│  │  Bash: fab log command "<skill>" "<id>"
│  │  Read: change artifacts (intake, spec, tasks)
│  │
│  ├─ Layer 3: Memory File Lookup
│  │  Read: intake/spec affected memory refs
│  │  Read: docs/memory/{domain}/index.md
│  │  Read: docs/memory/{domain}/{file}.md
│  │
│  └─ Layer 4: Source Code Loading
│     Read: source files from task/spec refs
│     Read: neighboring files (pattern context)
│
├─ Next Steps Convention
│  (state table lookup → "Next:" line)
│
├─ Skill Invocation Protocol
│  ([AUTO-MODE] prefix for inter-skill calls)
│
├─ Subagent Dispatch
│  ├─ Dispatch pattern (6 items)
│  └─ Standard Subagent Context
│     Read: config.yaml, constitution.md,
│           context.md*, code-quality.md*,
│           code-review.md*
│     (applied at every nesting level)
│
├─ SRAD Autonomy Framework
│  (scoring, grades, artifact markers)
│
└─ Confidence Scoring
   Bash: fab score <change>
   (gate thresholds for fab-ff / fab-fff)

* = optional, skip if missing
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | kit.conf (build guard), all context layer files |
| Bash | `fab preflight`, `fab log command`, `fab score` |

### Sub-agents

None — `_preamble.md` is a convention document consumed by skills, not an executor. Subagent dispatch patterns are defined here but executed by the consuming skill.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Change context | `fab log command "<skill>" "<id>"` | After preflight parse |
| Confidence scoring | `fab score <change>` | After spec generation (invoked by consuming skill) |
