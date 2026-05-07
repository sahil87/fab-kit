# _preamble

## Summary

Shared context preamble loaded by every Fab skill. Defines path conventions, context loading layers (always-load, change context, memory lookup, source code), the **Skill Helper Declaration** frontmatter convention, inlined **Naming Conventions**, inlined **Run-Kit (rk) Reference**, the **Common fab Commands** headline table, next-steps convention with state table, skill invocation protocol, subagent dispatch pattern with standard subagent context, SRAD autonomy framework, and confidence scoring.

This is an internal partial (`user-invocable: false`) — it is never invoked directly. Skills reference it via the opening instruction: "Read `src/kit/skills/_preamble.md` first."

## Subsection Inventory

Post-260418-or0o, `_preamble.md` contains four additional subsections inlined from previously-separate helpers or lifted out of `_cli-fab.md`. Each is a canonical source within `_preamble`:

| Subsection | Purpose | Canonical source |
|------------|---------|------------------|
| `## Skill Helper Declaration` | Documents the per-skill `helpers:` frontmatter field, its 4 allowed values (`_generation`, `_review`, `_cli-fab`, `_cli-external`), semantics (read each helper after `_preamble`, before body), and default (empty → load only `_preamble`). Explicitly states that `_naming` and `_cli-rk` are inlined (not allowed as values) and that `_preamble` is implicit. | `_preamble.md` itself |
| `## Naming Conventions` | Change folder pattern (`{YYMMDD}-{XXXX}-{slug}`), git branch naming (matches folder name), worktree directory naming (`{adjective}-{noun}`), and operator spawning rules (known change vs new from backlog). | `_preamble.md` (inlined from the deleted `_naming.md`) |
| `## Run-Kit (rk) Reference` | Silent-fail detection (`command -v rk`), iframe window creation, proxy URL pattern, server URL discovery at use-time, 4-step visual display recipe. | `_preamble.md` (inlined from the deleted `_cli-rk.md`) |
| `## Common fab Commands` | Headline table of 6 most-used fab command families (`preflight`, `score`, `log command`, `change`, `resolve`, `status`) with purpose and canonical invocation form. Cross-references `_cli-fab` for exhaustive flag documentation. | `_preamble.md` |

## Flow

```
Skill reads _preamble.md
│
├─ Path Convention
│  (all paths relative to repo root)
│
├─ Context Loading
│  ├─ Layer 1: Always Load
│  │  Read: config.yaml, constitution.md,
│  │        context.md*, code-quality.md*,
│  │        code-review.md*, memory/index.md,
│  │        specs/index.md
│  │  (no other helper — additional helpers
│  │   declared per-skill via frontmatter)
│  │
│  ├─ Layer 2: Change Context
│  │  Bash: fab preflight [change-name]
│  │  Bash: fab log command "<skill>" "<id>"
│  │  Read: change artifacts (intake, spec, plan)
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
├─ Skill Helper Declaration
│  (defines the `helpers:` frontmatter field —
│   allowed: _generation, _review,
│            _cli-fab, _cli-external)
│
├─ Naming Conventions (inlined from _naming)
│  (change folder / git branch / worktree /
│   operator spawning patterns)
│
├─ Run-Kit (rk) Reference (inlined from _cli-rk)
│  (detection, iframe, proxy, server URL,
│   4-step visual display recipe — fail silent)
│
├─ Common fab Commands
│  (headline table for 6 most-used families:
│   preflight, score, log command, change,
│   resolve, status — see _cli-fab for rest)
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
