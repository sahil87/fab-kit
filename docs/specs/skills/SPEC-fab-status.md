# fab-status

## Summary

Read-only status display. Shows change name, branch, stage progress (out of 6 total stages), plan progress (tasks + acceptance counts), confidence score, optional impact line (sourced from `.status.yaml` `true_impact` block), optional refactor-growth soft warning, version drift warning, and next command suggestion.

## Flow

```
User invokes /fab-status [change-name]
│
├─ Bash: fab preflight [change-name]
├─ Read: kit VERSION (via fab kit-path), fab/.kit-migration-version (if exists)
├─ Bash: git branch --show-current
│
└─ Render status display
   ├─ Stage line: "Stage: {stage} ({n}/6) — {state}"
   ├─ Progress table (6 rows: intake, apply, review, hydrate, ship, review-pr)
   │  Glyphs: ✓ done, ● active, ◷ ready, ○ pending, ✗ failed,
   │  ⏭ skipped (260612-w7dp — skipped glyph matches the Go
   │  renderer's ProgressLine)
   ├─ Plan counts: "Tasks: {plan.task_count}", "Acceptance: {plan.acceptance_completed}/{plan.acceptance_count}"
   │  (or "Plan: not yet generated" when plan absent)
   ├─ Confidence line (from .status.yaml confidence block)
   ├─ Impact line (when .status.yaml `true_impact` present)
   │  ⚠️-prefixed + bold when raw net > 100 OR excluding.net > 50
   │  (emoji + bold are the surviving channels — ANSI SGR is
   │   stripped by the render path; thresholds hard-coded,
   │   not project-configurable)
   ├─ Refactor-growth warning (soft, informational)
   │  Fires when change_type==refactor AND
   │  (excluding.net > 50 if present, else net > 50)
   │  Text (hard-coded): "Refactor changes typically shrink
   │  or stay flat — review whether this growth is intentional."
   └─ (agent formatting — no further tool calls)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab preflight`, `git branch --show-current` |
| Read | VERSION, migration-version |

### Sub-agents

None.

## Related `fab status` CLI verbs

Beyond the read-only `/fab-status` display skill above, the `fab status` CLI carries
the per-change `summary:` write/read verbs (kept in sync with `_cli-fab.md` § fab
status):

| Verb | Usage | Notes |
|------|-------|-------|
| `set-summary` | `set-summary <change> <text>` | Sets the `.status.yaml` `summary:` field — the per-change one-line log summary (FKF C-lite `log.md` source, see `docs/specs/fkf.md` §6.3). Conflict-free write path: each change touches only its own `.status.yaml`. An empty text clears the field (drop-when-empty round-trip via `omitempty`) |
| `get-summary` | `get-summary <change>` | Prints the `summary:` field. An absent/empty summary prints an empty line (graceful absence — the `log.md` generator falls back to the change slug) |

No stage auto-populates `summary` — it is set once during the change (authored at
hydrate, or carried from the intake) via the CLI. The `summary:` field models exactly
on the optional-string `change_type_source` field (`yaml:"summary,omitempty"`,
drop-when-empty, insert-when-absent before `last_updated`).
