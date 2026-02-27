# Kit Script Invocation Guide

> Loaded by every skill via `_preamble.md`. Defines calling conventions for all kit shell scripts.

---

## `<change>` Argument Convention

All stageman commands accept a unified `<change>` argument. Accepted forms (resolved by `changeman.sh resolve` internally):

| Form | Example |
|------|---------|
| 4-char change ID | `yobi` |
| Folder name substring | `fix-kit` |
| Full folder name | `260227-yobi-fix-kit-scripts` |
| `.status.yaml` path | `fab/changes/260227-yobi-fix-kit-scripts/.status.yaml` |

**Not accepted**: bare directory paths like `fab/changes/260227-yobi-fix-kit-scripts/` — use the folder name or change ID instead.

---

## stageman.sh

Stage Manager — manages workflow stages, states, and `.status.yaml`.

```
stageman.sh <subcommand> <change> [args...]
```

### Key subcommands

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `finish` | `finish <change> <stage> [driver]` | Mark stage done, auto-activate next |
| `start` | `start <change> <stage> [driver]` | pending/failed → active |
| `advance` | `advance <change> <stage> [driver]` | active → ready |
| `reset` | `reset <change> <stage> [driver]` | done/ready → active (cascades downstream) |
| `fail` | `fail <change> <stage> [driver]` | active → failed (review only) |
| `log-command` | `log-command <change> <cmd> [args]` | Log a command invocation |
| `log-review` | `log-review <change> <result> [rework]` | Log review outcome |
| `log-confidence` | `log-confidence <change> <score> <delta> <trigger>` | Log confidence change |
| `set-change-type` | `set-change-type <change> <type>` | Set change type |
| `set-checklist` | `set-checklist <change> <field> <value>` | Update checklist field |
| `progress-line` | `progress-line <change>` | Single-line visual progress |
| `current-stage` | `current-stage <change>` | Detect active stage |

### Stage transition side effects

Each `finish` auto-activates the next pending stage. No separate `start` call needed:

```
finish intake  → spec becomes active
finish spec    → tasks becomes active
finish tasks   → apply becomes active
finish apply   → review becomes active
finish review  → hydrate becomes active
finish hydrate → pipeline complete
```

**Common mistake**: calling `start <stage>` after `finish <previous-stage>` — this is redundant because `finish` already activated it.

---

## changeman.sh

Change Manager — manages change folders, naming, and the `fab/current` pointer.

```
changeman.sh <subcommand> [flags...]
```

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `new` | `new --slug <slug> [--change-id <4char>] [--log-args <desc>]` | Create new change |
| `rename` | `rename --folder <current-folder> --slug <new-slug>` | Rename change slug |
| `resolve` | `resolve [<override>]` | Resolve change name (from override, `fab/current`, or single-change guess) |
| `switch` | `switch <name> \| --blank` | Switch active change |
| `list` | `list [--archive]` | List changes with stage info |

---

## calc-score.sh

Confidence scorer — computes SRAD confidence score from Assumptions tables.

```
calc-score.sh [--check-gate] [--stage <stage>] <change-dir>
```

| Mode | Usage | Behavior |
|------|-------|----------|
| Normal | `calc-score.sh <change-dir>` | Parse spec.md, compute score, write to .status.yaml |
| Intake scoring | `calc-score.sh --stage intake <change-dir>` | Parse intake.md, compute score, write to .status.yaml |
| Gate check | `calc-score.sh --check-gate <change-dir>` | Parse artifact, compute score, compare threshold. Read-only (no .status.yaml write) |
| Intake gate | `calc-score.sh --check-gate --stage intake <change-dir>` | Intake gate with fixed threshold 3.0 |

**Note**: `calc-score.sh` takes `<change-dir>` (the directory path), not a change ID. It derives `.status.yaml` internally.

---

## preflight.sh

Pre-flight validator — validates project state and outputs structured YAML.

```
preflight.sh [<change-name>]
```

Validates: config.yaml exists, constitution.md exists, active change resolved, `.status.yaml` exists. Outputs YAML with `name`, `change_dir`, `stage`, `progress`, `checklist`, `confidence` fields. Non-zero exit on failure with error message on stderr.

---

## Common Error Messages

| Error | Cause | Fix |
|-------|-------|-----|
| "Status file not found: {path}" | Passed a path that doesn't exist as a file | Use a change ID or folder name instead |
| "Cannot resolve change '{arg}'" | Change ID/name doesn't match any folder in `fab/changes/` | Check `changeman.sh list` for available changes |
| "Multiple changes match" | Ambiguous substring matched multiple folders | Use a more specific identifier |
| "No active changes found" | `fab/current` is empty/missing and no changes exist | Run `/fab-new` first |
