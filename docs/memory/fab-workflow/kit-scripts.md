# Kit Scripts Reference

> Deep reference for agents working on the kit's shell scripts. For calling conventions, see `fab/.kit/skills/_scripts.md`.

## Scripts Overview

| Script | Location | Purpose |
|--------|----------|---------|
| `resolve.sh` | `fab/.kit/scripts/lib/resolve.sh` | Pure change resolution — converts any change reference to canonical output (no side effects) |
| `statusman.sh` | `fab/.kit/scripts/lib/statusman.sh` | Stage management — state transitions, `.status.yaml` accessors and metadata writes. `confidence` accessor outputs `indicative:{true|false}`. `set-confidence`/`set-confidence-fuzzy` accept `[--indicative]` flag |
| `logman.sh` | `fab/.kit/scripts/lib/logman.sh` | History logging — append-only JSON to `.history.jsonl` |
| `changeman.sh` | `fab/.kit/scripts/lib/changeman.sh` | Change management — create, rename, resolve (passthrough), switch, list |
| `calc-score.sh` | `fab/.kit/scripts/lib/calc-score.sh` | Confidence scoring — parse Assumptions tables, compute SRAD scores, gate checks |
| `preflight.sh` | `fab/.kit/scripts/lib/preflight.sh` | Pre-flight validation — project init checks, change resolution, auto-logging, structured YAML output |

## Call Graph

```
resolve.sh     ← universal resolver (no side effects)
   ↑
changeman.sh   ← uses resolve.sh internally, calls logman.sh for new/rename
statusman.sh   ← uses resolve.sh for CLI dispatch, calls logman.sh for review auto-log
logman.sh      ← uses resolve.sh --dir for change resolution
calc-score.sh  ← uses resolve.sh --dir, calls statusman.sh for writes, logman.sh for confidence log
preflight.sh   ← uses changeman.sh resolve, statusman.sh for queries, logman.sh for --driver auto-log
```

## Argument Resolution

### `resolve.sh` (standalone)

Universal resolver called by every other script. Pure function — no file writes, no `.status.yaml` modifications, no logging.

**Input forms**: 4-char change ID, folder name substring (case-insensitive), full folder name, or no argument (reads `.fab-status.yaml` symlink target to extract folder name, single-change guess fallback). `resolve.go` is the sole reader of `.fab-status.yaml` — all other scripts and skills delegate through it or `fab resolve`.

**Output flags** (mutually exclusive):
- `--id` (default) — 4-char change ID
- `--folder` — full folder name
- `--dir` — repo-root-relative directory path (with trailing slash)
- `--status` — repo-root-relative `.status.yaml` path

### `changeman.sh resolve`

Thin passthrough to `resolve.sh --folder` for backward compatibility.

### `statusman.sh` resolution

CLI dispatch resolves `<change>` via `resolve.sh --status`. Also accepts direct file paths for backward compatibility (if `[ -f "$arg" ]`).

## Stage State Machine

### States

`pending` → `active` → `ready` → `done`

Branch: `active` → `failed` (review only), `failed` → `active` (via `start`)

Reset: `done`/`ready`/`skipped` → `active` (cascades downstream to `pending`)

### Transitions

| Command | From | To | Side Effects |
|---------|------|----|-------------|
| `start` | pending, failed | active | Logs stage-transition event (best-effort) |
| `advance` | active | ready | None |
| `finish` | active, ready | done | Auto-activates next pending stage (logs stage-transition for next); review stage auto-logs "passed" via logman |
| `reset` | done, ready, skipped | active | Cascades all downstream stages to pending; logs stage-transition event (best-effort) |
| `fail` | active | failed | Review stage only; auto-logs "failed" [rework] via logman |

### Auto-Activation Chain

`finish` on any stage sets the next stage to `active`. The full chain: intake → spec → tasks → apply → review → hydrate.

## History Logging

### Architecture

`logman.sh` is the sole writer to `.history.jsonl`. It is never called directly by skills — all logging is triggered as side effects:

| Caller | Trigger | Logman call |
|--------|---------|-------------|
| `preflight.sh --driver <skill>` | Skill invocation | `logman.sh command` |
| `statusman.sh finish review` | Review pass | `logman.sh review "passed"` |
| `statusman.sh fail review` | Review fail | `logman.sh review "failed" [rework]` |
| `statusman.sh _apply_metrics_side_effect` | Stage activation | `logman.sh transition` |
| `calc-score.sh` | Score computation | `logman.sh confidence` |
| `changeman.sh new` | Change creation | `logman.sh command` |
| `changeman.sh rename` | Change rename | `logman.sh command` |

### `.history.jsonl` Format

One JSON object per line, appended to `{change_dir}/.history.jsonl`.

**Command event**: `{"ts":"ISO-8601","event":"command","cmd":"fab-ff","args":"optional"}`

**Confidence event**: `{"ts":"ISO-8601","event":"confidence","score":3.5,"delta":"+0.3","trigger":"calc-score"}`

**Review event**: `{"ts":"ISO-8601","event":"review","result":"passed"}` or `{"ts":"ISO-8601","event":"review","result":"failed","rework":"fix-code"}`. Canonical `result` values: `"passed"` and `"failed"`. Validation is permissive — non-canonical values (e.g., `"smoke-test"`, `"test-pass"`) are accepted for ad-hoc debugging.

**Stage-transition event**: `{"ts":"ISO-8601","event":"stage-transition","stage":"spec","action":"enter","driver":"fab-ff"}` (first entry) or `{"ts":"ISO-8601","event":"stage-transition","stage":"apply","action":"re-entry","from":"review","reason":"fix-code","driver":"fab-ff"}` (re-entry). `action` is `"enter"` when `iterations == 1`, `"re-entry"` when `iterations > 1`. `from`/`reason` present only on re-entries. `driver` present when provided.

### `logman.sh` Subcommands

All resolve `<change>` via `resolve.sh --dir`. The `command` subcommand's no-change-arg path delegates to `resolve.sh` (reads `.fab-status.yaml` symlink internally) with a symlink-existence guard to skip the single-change guess fallback — best-effort logging only fires when an explicit active change pointer exists.

- `logman.sh command <cmd> [change] [args]` — logs command invocation
- `logman.sh confidence <change> <score> <delta> <trigger>` — logs confidence change
- `logman.sh review <change> <result> [rework]` — logs review outcome
- `logman.sh transition <change> <stage> <action> [from] [reason] [driver]` — logs stage transition. `action` is `"enter"` (first activation, `iterations == 1`) or `"re-entry"` (`iterations > 1`). `from`/`reason` are present only on re-entries. `driver` is optional. Called automatically by `statusman.sh _apply_metrics_side_effect` — skills do not call this directly.

## Confidence Scoring

### Formula

```
if unresolved > 0: score = 0.0
else:
  base = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
  cover = min(1.0, total_decisions / expected_min)
  score = base * cover
```

Range: 0.0 to 5.0. `expected_min` varies by `{stage, change_type}`.

### DRY Helpers

`calc-score.sh` uses two internal helpers to avoid duplicating the formula:

- `count_grades <file>` — parse Assumptions table, output grade counts and dimension sums
- `compute_score <confident> <tentative> <unresolved> <total> <expected_min>` — apply the formula, return score

### Indicative Flag

When `calc-score.sh` runs in normal mode with `--stage intake`, it passes `--indicative` to `statusman.sh` write calls, persisting `confidence.indicative: true` in `.status.yaml`. When running without `--stage intake` (spec scoring), `--indicative` is not passed, which clears the flag (via `del(.confidence.indicative)` or omission in the AWK block).

### Gate Check

`--check-gate` parses the relevant artifact (intake.md or spec.md) and computes the score inline. Read-only — does not write to `.status.yaml`. Returns YAML with `gate: pass|fail`, `score`, `threshold`, `change_type`, and grade counts.

## Design Rationale

### 5-Script Architecture

Each script has exactly one responsibility with no overlap:

- **resolve.sh** owns change resolution (pure query, no side effects)
- **changeman.sh** owns change identity (folders, naming, pointer)
- **statusman.sh** owns stage state (`.status.yaml` reads and writes)
- **logman.sh** owns history logging (`.history.jsonl` append)
- **calc-score.sh** owns confidence scoring (SRAD formula, gate checks)
- **preflight.sh** owns validation (project init, change resolution, structured output)

### Auto-Logging over Explicit Logging

`log-command` was previously called manually by every skill as boilerplate. `log-review` was always paired with `finish/fail review`. Making these implicit (via `preflight.sh --driver` and statusman auto-log) eliminates manual logging lines across skill files.

### resolve.sh as Standalone Script

`resolve.sh` is the universal dependency (~180 lines including help text). Every other script calls it first. Embedding it in changeman would force every script to load 400+ lines for a ~95-line resolution function.

### Centralized `.fab-status.yaml` Access

`resolve.go` is the sole reader of the `.fab-status.yaml` symlink; `change.go` is the sole writer (`os.Remove` + `os.Symlink`). This was enforced by 260302-a8ay-centralize-current-pointer (centralized access to the former `fab/current` pointer), and updated by 260307-x2tx-status-symlink-pointer (replaced text file with symlink). No script or skill manipulates the symlink directly — all access goes through `fab resolve` and `fab change switch`.

---

*Last updated: 2026-03-05*
