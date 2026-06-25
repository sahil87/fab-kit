---
name: _cli-fab
description: "Fab CLI command reference — calling conventions, flag details, and commands not covered by the Common fab Commands subsection of _preamble."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Fab CLI Reference

> Loaded selectively via a skill's `helpers: [_cli-fab]` frontmatter. See `_preamble.md` § Common fab Commands for the 6 most-used commands (`preflight`, `score`, `log command`, `change`, `resolve`, `status`). This file documents the remaining commands and exhaustive flag details.

## Contents

- Calling Convention
- fab change (extended subcommand details)
- fab status (extended subcommand details)
- fab score (extended)
- fab preflight (extended)
- fab log (extended)
- fab resolve (extended)
- fab resolve-agent
- fab hook
- fab pane
- fab doctor
- fab migrations-status
- fab kit-path
- fab shell-init
- fab impact
- fab pr-meta
- fab memory-index
- fab fab-help
- fab help-dump
- fab operator
- fab spawn-command
- fab batch
- Common Error Messages

---

## Calling Convention

`fab <command> <subcommand> [args...]`. `fab` is a router dispatching workspace commands (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`) to `fab-kit` and everything else to the per-version `fab-go` binary resolved from `fab_version` in `fab/project/config.yaml`. `--version`/`-v`/`--help`/`-h`/`help` are handled inline. `fab-go` auto-fetches from GitHub releases on cache miss.

`fab -h` composes help from both binaries. `fab --version` prints the system binary version; inside a fab repo a second line shows the project-pinned version.

### Workspace Command Exit Semantics

Lifecycle commands fail loudly — a non-zero exit is the failure signal scripts and skills rely on:

| Command | Failure behavior |
|---------|------------------|
| `init` | Requires a git repository — exits non-zero with `fab init requires a git repository — run 'git init' first` BEFORE any download or config write. Sync failure during init also exits non-zero |
| `update` | Exits non-zero with `fab-kit was not installed via Homebrew` when the binary is not brew-installed (go-install/manual/CI); brew failures also exit non-zero |
| `upgrade-repo` | Runs sync first and stamps `fab_version` only AFTER sync succeeds. On sync failure: exits non-zero with `sync failed: ... — run 'fab sync' to repair, then re-run 'fab upgrade-repo'`, never prints `Updated: x -> y`, and a re-run retries (no "Already on the latest version" short-circuit of the broken state) |
| `sync` | Exits non-zero when any skill deployment write fails (per-skill `WARN:` lines on stderr, `failed N` in the agent tally) or when scaffolding writes fail. The version guard exits non-zero whenever it trips: either `fab-kit was updated to vX — re-run 'fab sync'` (auto-update landed; the current run still ran old code) or actionable too-old instructions (non-brew install, Homebrew tap release lag) — it never continues on a binary older than `fab_version` |

The auto-download path (any uncached `fab <cmd>`) is bounded by HTTP timeouts, serialized per version via an advisory lock, installed atomically (temp dir + rename), and verified against the release's `SHA256SUMS` asset — checksum mismatch refuses to install; releases predating checksum publishing install with a stderr warning.

### `upgrade-repo` Version Resolution

`fab upgrade-repo` resolves its target version by this precedence (first match wins):

| Invocation | Resolves to | Network? |
|------------|-------------|----------|
| `fab upgrade-repo <version>` | the explicit `<version>` (wins over everything; `--latest` is ignored when an arg is given) | No |
| `fab upgrade-repo --latest` | the newest published GitHub release (`releases/latest`) — the pre-2.3.x default, now opt-in | Yes |
| `fab upgrade-repo` (no arg) | the **installed binary's own version** (offline, authoritative) — reconciles the repo's kit to the `brew`-installed `fab-kit` | No |
| `fab upgrade-repo` when the binary is `dev`/unstamped | falls back to the newest GitHub release (a `just build` shim has no real release tag to sync to) | Yes |

The no-arg default is offline-first: it answers "match my repo to the installed binary" without a GitHub round-trip, avoiding the unauthenticated API rate limit (60 req/hr/IP, surfaced as a misleading `HTTP 403`). Use `--latest` to deliberately discover and jump to the newest upstream release. The *fetch* of a resolved-but-uncached target still downloads on demand; only *resolution* is offline.

### `<change>` Argument

All commands accept the unified `<change>`: 4-char ID (`yobi`), folder substring (`fix-kit`), or full folder name (`260227-yobi-fix-kit-scripts`). Bare directory paths and `.status.yaml` paths are NOT accepted.

### Commands covered in `_preamble` Common fab Commands

`fab preflight`, `fab score`, `fab log command`, `fab change`, `fab resolve`, `fab status` — headline coverage lives there. Sections below document the remaining commands (`fab hook`, `fab pane`, `fab doctor`, `fab migrations-status`, `fab kit-path`, `fab shell-init`, `fab impact`, `fab pr-meta`, `fab memory-index`, `fab fab-help`, `fab help-dump`, `fab operator`, `fab spawn-command`, `fab batch`) and extended flag details for the above.

---

## fab change (extended subcommand details)

See `_preamble.md` § Common fab Commands for the headline. Full subcommand table:

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `new` | `new --slug <slug> [--change-id <4char>] [--log-args <desc>]` | Create new change |
| `rename` | `rename --folder <current-folder> --slug <new-slug>` | Rename slug (prefix immutable) |
| `resolve` | `resolve [<override>]` | Thin wrapper over `fab resolve --folder` — the same shared implementation, identical output and error strings |
| `switch` | `switch <name> \| --none` | Switch active change (writes `.fab-status.yaml` symlink) |
| `list` | `list [--archive] [--show-stats]` | List changes with stage info; `--show-stats` appends the `true_impact` net column |
| `archive` | `archive <change> [--description "..."]` | Move to `archive/`, update index, mark backlog item done, clear pointer. `--description` is optional — defaults to the intake title (humanized-slug fallback). Re-archiving an already-archived change is a soft skip (exit 0) that still re-attempts the backlog mark (idempotent — recovers a previously-failed mark; silent, the plain soft-skip line is unchanged). |
| `restore` | `restore <change> [--switch]` | Move from `archive/`, remove index entry, optionally activate |
| `archive-list` | `archive-list` | List archived folder names |

`archive` and `restore` output structured YAML to stdout — skills parse it for user-facing reports. The `archive` YAML adds a `backlog: {marked|already|not_found}` field alongside `action`, `name`, `move`, `index`, and `pointer`. **Exception**: on the soft-skip path (re-archiving an already-archived change), `archive` prints a plain `already archived: {change}` line instead of YAML and exits 0 — skills parsing stdout must handle this non-YAML case (the `/fab-archive` skill treats it as a clean no-op). The soft skip covers both the half-completed case (archive destination already exists) and the genuinely-archived case (the change folder is gone from `fab/changes/` but matches an archive entry). **Partial failure**: when the archive move succeeds but the backlog mark fails (e.g., unreadable `fab/backlog.md`), `archive` prints the YAML report (so the completed move is visible) AND exits non-zero with the backlog error on stderr — the folder is already archived at that point; re-running soft-skips. An `archive/index.md` write failure follows the same print-then-error pattern on both commands: the YAML reports `index: failed` AND the command exits non-zero with the index error on stderr (for `archive` the move already succeeded; for `restore` the folder is already back in `fab/changes/`). `restore --switch` reports `pointer: {switched|failed}` — `failed` means the restore completed but activation could not create the `.fab-status.yaml` symlink (run `/fab-switch {name}` manually); `pointer: skipped` strictly means `--switch` was not requested.

---

## fab status (extended subcommand details)

Full subcommand table (headline in `_preamble` § Common fab Commands):

| Subcommand | Usage | Notes |
|------------|-------|-------|
| `finish` | `finish <change> <stage> [driver]` | Done + auto-activate next. Review auto-logs `passed` |
| `start` | `start <change> <stage> [driver] [from] [reason]` | pending/failed → active |
| `advance` | `advance <change> <stage> [driver]` | active → ready. Rejected (non-zero, no write) for `ship`/`review-pr` — `ready` is not in those stages' allowed states |
| `reset` | `reset <change> <stage> [driver] [from] [reason]` | done/ready/skipped → active (cascades downstream to pending; `stage_metrics` entries with a non-zero `iterations` keep that counter — only timing fields are cleared) |
| `skip` | `skip <change> <stage> [driver]` | {pending,active} → skipped (cascades pending→skipped downstream). Rejected (non-zero, no write) for `intake` — `skipped` is not allowed for intake |
| `fail` | `fail <change> <stage> [driver] [rework]` | active → failed (review/review-pr only). Auto-logs `failed` |
| `set-change-type` | `set-change-type <change> <type>` | Sets `change_type` AND marks `change_type_source: explicit`, so the PostToolUse intake-write hook stops re-inferring/overwriting it on later intake edits (it only re-infers when the source is absent or `inferred`) |
| `set-summary` / `get-summary` | `set-summary <change> <text>` / `get-summary <change>` | Per-change one-line log summary (`.status.yaml` `summary:` field — the FKF C-lite `log.md` source, §6.3). `set-summary` writes it (the conflict-free write path — each change touches only its own `.status.yaml`); `get-summary` prints it (empty line when absent — the generator then falls back to the change slug). `omitempty`: an empty summary round-trips to absent. No stage auto-populates it |
| `set-acceptance` | `set-acceptance <change> <field> <value>` | Updates `plan:` block. Valid fields: `generated` (bool), `task_count`, `acceptance_count`, `acceptance_completed` (int) |
| `set-checklist` | `set-checklist [args...]` | **Removed** — exits 1 with `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.` Use `set-acceptance` |
| `set-confidence` | `set-confidence <change> <counts...> <score> [--indicative]` | Basic confidence block. `--indicative` is a deprecated accepted-but-ignored no-op (1.10.0) — it writes nothing |
| `set-confidence-fuzzy` | `set-confidence-fuzzy <change> <counts...> <score> <dims...> [--indicative]` | With SRAD dimensions. `--indicative` is a deprecated no-op (see above) |
| `add-issue` / `get-issues` | `<change> <id>` / `<change>` | Issue ID array — idempotent / one per line |
| `add-pr` / `get-prs` | `<change> <url>` / `<change>` | PR URL array — idempotent / one per line |
| `progress-line` | `progress-line <change>` | Single-line visual progress |
| `current-stage` | `current-stage <change>` | Detect active stage |
| `all-stages` | `all-stages` | List all stage IDs in order (no `<change>` argument) |
| `progress-map` | `progress-map <change>` | Extract `stage:state` pairs, one per line |
| `display-stage` | `display-stage <change>` | Display stage as `stage:state` |
| `plan` | `plan <change>` | Extract `plan:` fields — `generated`, `task_count`, `acceptance_count`, `acceptance_completed` (one `key:value` per line) |
| `confidence` | `confidence <change>` | Extract `confidence:` fields — `certain`, `confident`, `tentative`, `unresolved`, `score` (one `key:value` per line) |
| `validate-status-file` | `validate-status-file <change>` | Validate `.status.yaml` against the schema; non-zero exit on violation |

**Target-state validation**: every event command validates the resolved target state against the stage's allowed states — a schema-forbidden combination (e.g., `advance ship`, `advance review-pr`, `skip intake`) exits non-zero with `Cannot {event} stage '{stage}' — target state '{state}' is not allowed for this stage` and writes nothing, instead of bricking `fab preflight` with a permanently invalid `.status.yaml`.

**Side effects of `finish`**: `intake→apply`, `apply→review`, `review→hydrate` (+auto-log `passed`), `hydrate→ship`, `ship→review-pr`. Never call `start` after `finish`. Legacy `tasks` event invocations exit 1 with `"tasks" stage was removed — run "fab status <event> <change> apply" instead. plan.md is now generated at apply entry.` Legacy `spec` event invocations exit 1 with `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".`

**Auto-logs**: `finish review|review-pr`→`passed`; `fail review|review-pr`→`failed`; every `active` transition is best-effort logged. Skills do NOT manually call `fab log review` or `fab log transition`.

### stage_hooks (project-config pre/post stage commands)

`fab status start` and `fab status finish` honor an optional `stage_hooks` map in `fab/project/config.yaml` (not seeded by the scaffold — add the key by hand; not to be confused with `fab hook`, the Claude Code harness handlers below):

```yaml
stage_hooks:
  apply:
    pre: ./scripts/check-clean-tree.sh   # any sh -c command line
    post: make test
```

| Hook | Fires | On failure (non-zero exit) |
|------|-------|---------------------------|
| `pre` | Before `start`'s transition is applied | **Blocks the stage from starting** — the transition is not applied, the command errors |
| `post` | After `finish`'s transition **is saved** (stage already `done`, next stage already auto-activated) | The command errors, but the saved transition stands |

- **Execution**: `sh -c "<command>"` from the repo root, stdout/stderr passed through. An empty/absent hook (or a missing config file) is a silent no-op.
- **Auto-activation caveat**: pre hooks fire only on an explicit `fab status start` — `finish`'s auto-activation of the next pending stage does NOT run that stage's pre hook.
- **Failing-post-hook re-run trap**: by the time a post hook runs, the stage is already `done` — re-running `fab status finish <change> <stage>` after fixing the hook does NOT re-fire it (`done` is not a valid `finish` source state; the re-run errors). Run the hook command by hand instead, or `reset` the stage first if the transition genuinely needs replaying.

---

## fab score (extended)

See `_preamble.md` § Common fab Commands. Modes:

| Mode | Usage | Behavior |
|------|-------|----------|
| Normal | `fab score <change>` | Parse `intake.md` (the sole scoring source; `--stage` defaults to `intake`), compute, write `.status.yaml`. No `indicative` key is written (retired 1.10.0). Exits non-zero (error on stderr) when `.status.yaml` fails to load, the confidence write-back or `.history.jsonl` confidence-log append fails, or `intake.md` cannot be read — no silent partial success; the YAML report appears on stdout only when scoring *and* persistence succeed |
| Gate | `fab score --check-gate [--stage intake] <change>` | Read-only threshold compare; non-zero below the flat 3.0 intake gate (the single gate — `--stage` defaults to `intake`, so the flag is optional). An `intake.md` read failure also exits non-zero (distinguishable on stderr from a gate fail) rather than gating on a partial Assumptions table |

### Schema (in `.status.yaml`)

```yaml
confidence:
  certain: 12      # count of Certain-graded SRAD decisions (grade DERIVED from composite)
  confident: 3     # count of Confident-graded decisions
  tentative: 2     # count of Tentative-graded decisions
  unresolved: 0    # count of Unresolved-graded decisions
  score: 2.1       # derived score (see formula below), computed from intake.md
```

> The grade counts are **derived** from each row's composite (the 80/50/20 bands), not read from the hand-written Grade column, and are informational — only `score` gates the pipeline.

> The `confidence.indicative` flag is retired (1.10.0): intake scoring is now authoritative, not indicative, so the flag's distinction is meaningless with one scoring source. It is no longer written; a legacy `indicative: true` key on disk is tolerated on read and harmlessly dropped on the next save.

### Formula

Demerit model — the score starts at a perfect 5.0 and each decision subtracts a **penalty** keyed on its composite. Strong decisions cost nothing; weak ones cost, and the cost cannot be refunded by surrounding strong rows (so a single risky decision stays visible, never averaged away):

```
for each Assumptions row with parseable dimensions:
  composite = 0.20 * S + 0.30 * R + 0.30 * A + 0.20 * D            # 0–100; R and A up-weighted

  penalty(c) =  0                            if c >= 80            # Certain  → free
                (80 - c) / 30 * 0.50         if 50 <= c < 80       # Confident → ≤ 0.5
                0.50 + (50 - c)/50 * 2.50    if c < 50             # Tentative / Unresolved

score = clamp(5.0 - Σ penalty(composite), 0.0, 5.0)               # sum over parseable rows
```

There are **no hard-fail short-circuits** — no `Unresolved → 0.0` and no `R<25 ∧ A<25` Critical Rule. Blocking is emergent from the curve: a `composite < 20` row penalizes ≥ 2.0, which alone drops a change to the 3.0 gate or below. Reversibility is carried by its 0.30 weight in the composite (low-R decisions land in a worse band and are penalized harder), not by a separate rule. There is **no coverage factor and no minimum-decision requirement** — a thin-but-strong intake (two well-resolved decisions) genuinely scores 5.0; quality is measured per decision, so row count is not a proxy for it. The grade (Certain/Confident/Tentative/Unresolved) is **derived from the composite** (bands 80/50/20) and is indicative only — never read by the formula. Range: 0.0 to 5.0. `expected_min` (in `docs/specs/change-types.md`) is no longer part of the score path; it remains documented only.

### Template

The `status.yaml` template (in the kit cache at `$(fab kit-path)/templates/status.yaml`) includes the confidence block initialized to zero counts and score 0.0. `/fab-new` writes the intake score after intake generation; `/fab-clarify` re-writes it after resolving intake assumptions.

---

## fab preflight (extended)

`fab preflight [<change-name>]` — validates config.yaml, constitution.md, active change resolution, `.status.yaml` existence. Outputs YAML with `id`, `name`, `change_dir`, `stage`, `display_stage`, `display_state`, `progress`, `plan`, `confidence`. Non-zero exit on failure (error on stderr). Pure validation — no side effects.

---

## fab log (extended)

Append-only JSON logging to `.history.jsonl`.

```
fab log command <cmd> [change] [args]
fab log confidence <change> <score> <delta> <trigger>
fab log review <change> <result> [rework]
fab log transition <change> <stage> <action> [from] [reason] [driver]
```

`command` is pure telemetry and **always exits 0** (given valid usage — cobra arg-count errors exit non-zero before RunE) — it owns its best-effort contract. On any internal failure (no fab root, an explicit `[change]` that doesn't resolve, unwritable `.history.jsonl`) it prints a one-line `Warning: fab log command: …` to stderr and still exits 0, so call sites need no `2>/dev/null || true` guard and a telemetry hiccup can never become a pipeline failure mode. When `[change]` is omitted, the active change resolves from `.fab-status.yaml` (silent no-op if absent/dangling). `review`/`confidence`/`transition` keep fail-loud non-zero exits (they are auto-logged by `fab status`/`fab score` — skills never call them directly).

**Common callers** — skills per `_preamble.md` Context Loading §2 (`fab log command "<skill>" "<change>"`); `finish/fail review` auto-log; `score` auto-logs confidence; `change new`/`change rename` auto-log.

---

## fab resolve (extended)

Pure query, no side effects.

```
fab resolve [--id|--folder|--dir|--status|--pane] [--server <name>] [<change>]
```

| Flag | Output |
|------|--------|
| `--id` (default) | 4-char change ID |
| `--folder` | Full folder name |
| `--dir` | Directory path (`fab/changes/.../`) |
| `--status` | `.status.yaml` path |
| `--pane` | Tmux pane ID (errors `ERROR: no tmux pane found for change "<folder>"` if no matching pane) |
| `--server <name>` / `-L <name>` | Pane mode only: target tmux socket (`tmux -L <name>`), searched server-wide across all sessions; skips the `$TMUX` requirement. Without it, pane lookup is current-session-only and requires `$TMUX` (`ERROR: not inside a tmux session` otherwise) |

The five output-mode flags are **mutually exclusive** — passing two (e.g. `--status --folder`) exits non-zero with a flags-group error instead of silently picking one. `fab change resolve` is a thin wrapper over this same implementation with `--folder` mode fixed.

---

## fab resolve-agent

Pure query (no side effects) — resolves a pipeline stage to its `{model, effort}` agent profile for sub-agent dispatch. Consumed by the orchestrators (`/fab-ff`, `/fab-fff`, `/fab-proceed`) and `/fab-continue`'s sub-agent dispatch, which call it immediately before dispatching each stage's sub-agent.

```
fab resolve-agent <stage> [--alias]
```

`<stage>` is one of the six pipeline stages: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`.

**Resolution**: maps the stage → its tier via the FIXED fab-owned stage→tier mapping (`thinking`: intake, review / `doing`: apply, review-pr, hydrate / `fast`: ship — NOT user-overridable), then resolves the tier → `{model, effort}`: the project's `agent.tiers.<tier>` override **per-field merged** over fab-kit's built-in default (`thinking`: claude-opus-4-8/xhigh, `doing`: claude-opus-4-8/high, `fast`: claude-sonnet-4-6/low), else the default. `agent.tiers` is the sole override surface — there is no `stage_tiers` and no per-stage escape hatch. See `docs/specs/stage-models.md`.

**Output** (two stdout lines, byte-stable for the same config):

```
model=<id>
effort=<level>
```

- The `effort=` line is **omitted** when the resolved tier has no effort (empty/absent).
- An **empty model** emits an empty `model=` line — signals "inherit the session/orchestrator model" (today's foreground/no-override behavior). Callers omit the dispatch `model` param in that case.

**`--alias` (Claude-Code Agent-tool adapter)**: when set, the `model=` line emits the Claude-Code **short alias** (`opus` / `sonnet` / `haiku` / `fable`) instead of the full versioned ID. This exists because the Claude Code **Agent tool's `model` parameter is a hard enum** that rejects full IDs — sub-agent dispatch must pass an alias. The mapping is prefix-based (`claude-opus-` → `opus`, etc.), so dated variants like `claude-haiku-4-5-20251001` resolve to `haiku`. The **default (flag absent) is unchanged** — the full ID, byte-identical to today (the `claude` CLI `--model` flag, used by the operator launcher, accepts full IDs and keeps resolving WITHOUT `--alias`). The **`effort=` line is unaffected** by `--alias`. **Empty / non-Claude models pass through verbatim** (an empty `model=` line stays empty — the inherit signal; an unrecognized/non-Claude ID like `gpt-5` is emitted unchanged) — `--alias` is a best-effort adapter, not a Claude-only validator.

```
$ fab resolve-agent apply
model=claude-opus-4-8
effort=high

$ fab resolve-agent apply --alias
model=opus
effort=high
```

**No validation — verbatim pass-through**: `fab resolve-agent` does NOT validate the model or effort against any provider's accepted set (provider neutrality — a fab-kit design principle). It echoes both strings as-is — `xhigh`, `reasoning_effort:high`, an empty effort, whatever. A misconfigured pair (e.g. Sonnet + `xhigh`) is NOT corrected by fab; it surfaces as a dispatch-time error in the harness. There is no effort-enum enforcement and no degrade-gracefully drop.

**Exit code**: non-zero only on a real error — an unreadable/malformed config, or an unknown stage name. A stage resolving to a default is success (exit 0).

---

## fab hook

Claude Code hook handlers. Each subcommand is registered as inline `fab hook <subcommand>` in `.claude/settings.local.json`. **All hook subcommands exit 0** so they never block the agent — the four event handlers swallow errors silently; `sync` (setup-facing) surfaces failures on stderr but still exits 0.

| Subcommand | Event | Purpose |
|------------|-------|---------|
| `session-start` | SessionStart | Delete `_agents[session_id]` entry in `.fab-runtime.yaml` |
| `stop` | Stop | Write `_agents[session_id]` with `idle_since` plus optional tmux/pid/change/transcript fields |
| `user-prompt` | UserPromptSubmit | Remove only `idle_since` from `_agents[session_id]`; other fields preserved |
| `artifact-write` | PostToolUse (Write/Edit) | Per-artifact bookkeeping from stdin JSON |
| `sync` | n/a | Register inline hook entries in `.claude/settings.local.json`; migrates old-style bash scripts; idempotent |

The three session-scoped hooks (`session-start`, `stop`, `user-prompt`) read a JSON payload on stdin with at least a `session_id` field (UUID) and optionally `transcript_path`. Malformed JSON or a missing `session_id` is silently skipped. Each handler also invokes a throttled GC sweep (≤ once per 180 s via `last_run_gc`) that prunes entries whose stored `pid` no longer exists (`kill(pid, 0)` returning ESRCH). `artifact-write` is unchanged — it parses a different payload shape (`tool_input.file_path`) and does not participate in `_agents` writes; it emits `{"additionalContext":"Bookkeeping: ..."}` on stdout. After bookkeeping it **auto-stages the change's `.status.yaml` and `.history.jsonl`** (best-effort `git add`, errors ignored) so hook-driven bookkeeping writes never block subsequent git operations.

**`artifact-write` bookkeeping** (per artifact):
- **intake.md** → infers `change_type` from keywords and recomputes the intake confidence score, persisting both in one Save. **change_type is re-inferred ONLY when `.status.yaml` has `change_type_source` absent or `inferred`** — when a human ran `fab status set-change-type` (which marks `change_type_source: explicit`), the hook keeps the explicit type and does NOT overwrite it. This stops the previous behavior where every intake edit silently reverted a manual type correction. The `fix` keyword inference excludes a passing `must-fix`/`must fix` mention in a feature intake (it still classifies `bug-fix`/`hot-fix`/`bug-free`/standalone `fix` as `fix`).
- **plan.md** → recomputes `task_count`, `acceptance_count`, and `acceptance_completed` (acceptance counts from the `## Acceptance` checkboxes). This durable counter is a **write-time cache**: readers (`fab preflight`, `fab pr-meta`, `fab status plan`) prefer a **live count derived from `plan.md` `## Acceptance` checkboxes at read time** and fall back to the cached counter only when `plan.md` (or its `## Acceptance` section) is absent — so a hook-bypassing edit (`sed`, direct file edit) cannot make those readers report a stale acceptance count. (`fab score` reads `intake.md` only and is unaffected.)

`sync` output: `Created`, `Updated`, or `.claude/settings.local.json hooks: OK` on stdout; on failure (no fab root, unwritable settings) a `hook sync: {error}` line on stderr — exit code stays 0 either way.

---

## fab pane

Tmux pane operations with fab context enrichment. `fab pane <map|capture|send|process|window-name> [flags...]`

**Pane-family exit codes** (capture, send, window-name): pane validation failures use a shared scheme so callers can branch on cause — `2` = pane missing, `3` = any other tmux failure (dead server, bad socket). `map` and `process` use plain `ERROR:`-formatted exit 1. (Non-tmux usage errors — bad flag values, cobra arg-count — exit 1 per command; see the per-verb rows.)

**Persistent flag** (all subcommands): `--server <name>` / `-L <name>` (default `""`) — target tmux socket (`tmux -L <name>`). Defaults to `$TMUX` / tmux default. Lets daemons on one tmux server inspect panes on another.

### map — `fab pane map [--json] [--session <name>] [--all-sessions] [--server <name>]`

All tmux panes with pipeline state. Non-git/non-fab panes included with `---` fallbacks.

| Flag | Description |
|------|-------------|
| `--json` | JSON array (snake_case: `session`, `window_index`, `pane`, `tab`, `worktree`, `repo`, `change`, `stage`, `display_state`, `agent_state`, `agent_idle_duration`, `pr_url`, `pr_number`). `repo` is the absolute main-worktree root for the pane's repo (`null` when unresolved) — `--json` only, no human-table column. `display_state` (`string\|null`) is the state half of the display-stage derivation (the `stage` field is the name half): `active`, `ready`, `done`, `failed`, `pending`, or `skipped`; `null` whenever `stage` is `null` (no resolvable change / unloadable `.status.yaml`) — `--json` only, no table column. Distinguishes an actively-worked stage from a parked finished change (e.g. a fully-shipped change is `stage: "review-pr"` + `display_state: "done"`, while one whose review-pr is running is `"active"`). `pr_url` (`string\|null`) is the last entry of the change's `.status.yaml` `prs:` list (most recent), `null` when the list is absent/empty or the pane has no resolvable change; `pr_number` (`number\|null`) is parsed from the URL's trailing `/pull/<n>` segment, `null` when there is no URL or it is unparseable. Both are `--json` only (no table column), sourced from the already-loaded status file — **no `gh`/`git`, no network, no PR status (open/merged/CI)**; consumers fetch live PR state themselves. |
| `--session <name>` | Target specific session (skips `$TMUX` check) |
| `--all-sessions` | Query all sessions (skips `$TMUX` check; mutually exclusive with `--session`) |

Without `--session`/`--all-sessions` → current session only (`-s` scope, requires `$TMUX`). Table columns: `Session` (only with `--all-sessions`), `Pane`, `WinIdx`, `Tab`, `Worktree` (relative; `(main)` for main; `basename/` non-git), `Change`, `Stage`, `Agent`. The `Worktree` relative path is computed **per repo** — each pane's display path is relative to its own repo's main-worktree root (cached by git worktree root), so panes from multiple repos render correct paths. Agent: `active`, `idle ({dur})`, or `—` (em dash). Change: folder name, `(no change)` for fab worktree with no active change, or `—` for non-fab panes. Idle duration: `{N}s`/`{N}m`/`{N}h` floor division. Change and Agent resolve on independent axes: Change comes from `.fab-status.yaml`; Agent comes from `_agents[*].tmux_pane` matching in `.fab-runtime.yaml` — so a pane with a running Claude in discussion mode (no active change) now shows `(no change)` in Change but a populated Agent column. `$TMUX` unset without targeting flag → exit 1 (`ERROR: not inside a tmux session`). No panes → exit 0 `No tmux panes found.`

### capture — `fab pane capture <pane> [-l N] [--json] [--raw] [--server <name>]`

`<pane>` required (e.g., `%5`). `-l/--lines N` (default 50). `--json` = content + metadata (`worktree`/`change`/`stage`/`agent_state`/`agent_idle_duration`). `--raw` = plain `tmux capture-pane -p`, no enrichment. `--json`/`--raw` mutually exclusive. Pane not found → exit 2 (`Error: pane <id> not found`); other tmux validation failure → exit 3. `--lines < 1` → exit 1 (`ERROR: --lines must be >= 1`).

### send — `fab pane send <pane> <text> [--no-enter] [--force] [--server <name>]`

Validation pipeline: (1) pane exists via a single targeted probe — `tmux display-message -t <pane> -p '#{pane_id}'`, output must equal the argument exactly (ID-exact: window names / target-grammar args resolve to a different pane ID and are rejected; no server-wide enumeration) — pane missing → exit 2 (`Error: pane <id> not found`), other tmux failure → exit 3; (2) agent is idle (rejects `active`/`unknown` unless `--force`) — not idle → exit 1 (`ERROR: agent in pane <id> is not idle (state: <state>)`); (3) `tmux send-keys`. `--no-enter` skips the trailing Enter. `--force` bypasses idle check only — pane-existence still enforced. Agent resolution matches `_agents[*].tmux_pane` in `.fab-runtime.yaml` at the worktree root; a pane with no matching entry = `unknown` (non-idle). Change state is independent — panes in discussion mode (no active change) now accept sends when idle, instead of being rejected as `unknown`. Success: `Sent to <pane>`.

### process — `fab pane process <pane> [--json] [--server <name>]`

OS-level process tree. Linux: walks `/proc/<pid>/task/<tid>/children`, reads `/proc/<pid>/comm` + `/cmdline`. macOS: `ps -o pid,ppid,comm -ax` PPID traversal, plus one batched `ps -axo pid=,args=` pass joined by PID for full cmdlines (two `ps` spawns total — no per-node lookups; a process exiting between the passes degrades to cmdline `""`). Classification: `claude`/`claude-code` → `agent`, `node` → `node`, `git`/`gh` → `git`, else `other`. JSON: `{pane, pane_pid, processes (tree), has_agent}`. Pane not found → exit 1 (`ERROR: pane <id> not found`). `--server` scopes tmux lookup only; `/proc`/`ps` walk is socket-independent.

### window-name — `fab pane window-name <ensure-prefix|replace-prefix> [--json] [--server <name>]`

Guarded, idempotent rewrites of the tmux window name — used by `/fab-operator` to mark enrolled (`»`) and done-monitoring (`›`) windows.

| Verb | Usage | Behavior |
|------|-------|----------|
| `ensure-prefix` | `ensure-prefix <pane> <char>` | Idempotent prepend: if the window name already begins with the literal `<char>`, no-op; else `rename-window` to `<char><name>`. `<char>` must be non-empty (else exit 3) |
| `replace-prefix` | `replace-prefix <pane> <from> <to>` | Atomic guarded swap: if the name begins with `<from>`, rename to `<to><name-without-from>`; else silent no-op (the user-rename-mid-monitoring guard). `<to>` may be empty (prefix strip); `<from>` must be non-empty (else exit 3) |

**Exit codes** (both verbs): `0` = renamed OR no-op; `2` = pane missing (tmux stderr propagated); `3` = any other tmux failure (tmux not running, socket error, rename failed, argument usage error — e.g., empty `<char>` or `<from>`). The 2/3 split lets `/fab-operator`'s removal path treat "pane gone" (exit 2) as successful removal. No `$TMUX` gate — tmux's own exec failure surfaces as exit 3, so the verbs work via `--server` targeting from outside a tmux client.

**Output**: plain `renamed: <old> -> <new>` on rename, empty stdout on no-op; `--json` always emits one `{"pane","old","new","action"}` object (`action`: `renamed`|`noop`).

---

## fab doctor

Prerequisite check. Lives in `fab-kit` so it works before `config.yaml` exists; used as `/fab-setup` Phase 0 gate.

```
fab doctor [--porcelain]
```

**Checks** (7): git, fab, bash, yq (v4+), jq, gh, direnv (with zsh/bash hook detection).

**Output**: `  ✓ {tool} {version}` (pass) / `  ✗ {tool} — not found` + install hint (fail) / summary line. Exit code = failure count.

`--porcelain`: errors only (no passes/hints/summary). Exit code still = failure count. Empty stdout + exit 0 = all good.

---

## fab migrations-status

Migration discovery. Lives in `fab-kit` (registered in the router's `fabKitArgs` allowlist). Resolves `fab/.kit-migration-version` (local) and the engine `VERSION` from the cached kit for `fab_version`, scans the engine `migrations/` dir, and runs the discovery algorithm. Consumed by both `/fab-setup migrations` (via `--json`) and as a standalone query.

```
fab migrations-status [--json]
```

**Human output**: `Local version` / `Engine version`, then either `No migrations apply.` or `Migrations to apply (N):` with an ordered `[i/N] FROM -> TO (file)` list, followed by any gap-skip lines and any overlap warning.

**`--json` output**: `{local, engine, applicable:[{from,to,file}], gap_skips, overlaps}` — `applicable` is the ordered chain to apply (FROM ascending), `gap_skips` are skip log lines, `overlaps` are conflicting filename pairs (non-empty = malformed migration set).

**Exit code**: `0` on any clean query — including the no-op case AND the overlap case (overlap is surfaced via the `overlaps` field). Non-zero only on a genuine error (missing `fab/.kit-migration-version`, missing engine `VERSION`, unreadable migrations dir). Read-only — never writes `fab/.kit-migration-version`.

---

## fab kit-path

```
fab kit-path
```

Prints absolute path to the resolved kit directory (exe-sibling `kit/` next to `fab-go`). No trailing newline, no decoration. Exit 0 on success; non-zero with stderr error on failure. Used by skills to reference kit content: `$(fab kit-path)/templates/`, `$(fab kit-path)/migrations/`, etc.

---

## fab shell-init

```
fab shell-init <bash|zsh|fish>
```

Emits the shell-completion script for the given shell on stdout — the `tu`-style verb equivalent of (and delegated to) Cobra's auto-generated `fab completion <shell>`. Recommended install: add `eval "$(fab shell-init zsh)"` to `~/.zshrc` (or the bash/fish equivalent). Config-independent — works outside a fab repo. Human-setup-facing; no skill invokes it.

---

## fab impact

```
fab impact <base> <head>
```

Computes `git diff --shortstat <base>...<head>` line counts and emits a YAML document on stdout matching the `.status.yaml` `true_impact` block schema (minus `computed_at_stage`):

```yaml
added: 142
deleted: 38
net: 104
excluding:
    added: 87
    deleted: 38
    net: 49
tests:
    added: 40
    deleted: 0
    net: 40
computed_at: "2026-05-07T14:32:00Z"
```

The `excluding` sub-block is emitted only when `fab/project/config.yaml`'s top-level `true_impact_exclude` list is non-empty; the subcommand applies each entry as a `:(exclude)<pattern>` pathspec when running the second `git diff --shortstat` pass.

The `tests` sub-block is emitted only when `fab/project/config.yaml`'s top-level `test_paths` list is non-empty. It is computed by a third `git diff --shortstat` pass whose pathspec combines the `test_paths` includes with the same `:(exclude)<pattern>` arguments as the `excluding` pass — so test lines are counted *within the scaffolding-excluded universe* (a test fixture under an excluded path is not double-counted). Each include is applied as a `:(glob)<pattern>` magic pathspec so wildcards behave like `.gitignore`-style globs — notably `**` matches across directory boundaries (so `**/*_test.go` matches both `foo_test.go` and `pkg/foo_test.go`). When `true_impact_exclude` is empty, the test pass runs with the includes alone (tests are then attributed within the raw universe). No `impl` field is emitted: the implementation residual (`impl = max(0, total − tests)`, per component) is derived at render time by consumers — the YAML stores only the measured passes. Emitted after `excluding`, before `computed_at`.

Three-dot range semantics (`<base>...<head>`) — "changes on this branch only".

Exit codes:
- `0` — success; YAML document on stdout.
- non-zero — `<base>` is empty/invalid or `git diff` failed; actionable message on stderr (e.g., `base ref is empty`). The subcommand does not run `git merge-base` itself — callers must resolve the merge-base upstream and pass the result. The caller decides whether to abort or skip.

Consumers: `fab pr-meta` (which renders the PR body `**Impact**` line via the same `internal/impact` package) and the apply-finish + hydrate-finish hooks (write the result into `.status.yaml` `true_impact`). `/git-pr` no longer calls `fab impact` directly — it delegates the whole `## Meta` block to `fab pr-meta`.

---

## fab pr-meta

```
fab pr-meta <change> --type <type> [--issues "DEV-1 DEV-2"]
```

Renders the complete `## Meta` block of a fab-generated PR as final markdown on stdout — the deterministic replacement for the natural-language Meta formatting that previously lived in `/git-pr` Step 3c. The block is byte-for-byte stable across runs, so the Meta block stops drifting between PRs.

Arguments and flags:
- `<change>` — 4-char ID, folder substring, or full folder name (resolved via the same `resolve` package as every other subcommand).
- `--type <type>` — **required**. The resolved PR type (`feat|fix|refactor|docs|test|ci|chore`). `/git-pr` resolves type via its Step 0b chain (which depends on the user's argument and the diff) and passes it in; the binary does not re-derive it.
- `--issues "<space-joined IDs>"` — optional. When non-empty, renders the `**Issues**` line. When absent/empty, the line is omitted.

Self-contained data sourcing — the command reads everything else itself:
- `.status.yaml` (via the `statusfile` package): `id`, `confidence.score`, `plan.acceptance_count`/`acceptance_completed`, `progress.*`, `stage_metrics.review.iterations`.
- `plan.md`: parses the `## Tasks` checkboxes (`- [x]` vs `- [ ]`) for the `{done}/{total} tasks` count. Legacy `tasks.md` fallback for pre-1.9.0 changes.
- `fab/project/config.yaml`: `true_impact_exclude`, `test_paths`, and `project.linear_workspace`.
- Impact math: reuses `internal/impact` (`ComputeForRepo`) against the merge-base of HEAD vs `origin/main` (falling back to `origin/master`), computed internally.
- Git/`gh` context: branch (`git branch --show-current`) and owner/repo (`gh repo view --json nameWithOwner`) for blob URLs.

Output — the exact `## Meta` block markdown, in element order **table → Impact → optional Issues → Pipeline** (each block blank-line separated so GitHub renders them as distinct elements):
- The 5-column table (`Change ID | Type | Confidence | Plan | Review`) with `—` fallbacks, the `Change ID` value backtick-wrapped when present (the bare `—` fallback is not), a ` ✓` Plan completion suffix when both task and acceptance pairs are complete, and a `✓/✗ {N} cycle{s}` Review cell.
- Impact: a single normalized markdown table whose first-column header is `Impact` (so the table self-labels — there is no `**Impact**:` lead-in line), columns `Impact | +/− | Net` (numeric columns right-aligned, Net kept on every row), followed by a `<sub>` provenance caption. The locked taxonomy is `raw / true / impl / tests / excluded` with `raw = true + excluded` and `true = impl + tests`; `true` is ALWAYS the post-exclude diff (the fix for the prior "total flips meaning" bug). The table adapts by DROPPING rows, never reshaping: the `raw` row is shown only when it differs from `true` (excludes engaged), the `**true**` row is always present and bold, and the nested `└ impl` / `└ tests` rows appear only when a `tests` pair exists (`impl` is the per-component `max(0, true − tests)` residual, Unicode minus `−`, clamp-annotated when net-negative). The caption co-locates the excludes note and the version stamp — `<sub>excludes `…` · generated by fab-kit vX.Y.Z</sub>` — with the excludes list built from the actual `true_impact_exclude` values each backtick-wrapped (the `excludes …` clause omitted when none are configured) and the binary version stamped from the running `fab` (`main.version`, threaded via `prmeta.Data.Version`; a dev build renders `fab-kit vdev` honestly). The whole block is omitted entirely on `+0/−0` `true`, missing merge-base, or impact failure. Only **bold** is used for emphasis (`<sub>` is on GitHub's HTML allowlist; the sanitizer strips row backgrounds and text color).
- `**Issues**` (only when `--issues` is non-empty): Linear-linked when `project.linear_workspace` is set, bare comma-joined IDs otherwise; positioned between Impact and Pipeline.
- `**Pipeline:**` (colon inside the bold span): the six stages in fixed order with ` ✓` per `done` stage; `intake`/`apply` labels hyperlink to blob URLs when the artifact exists and owner/repo resolved. Rendered LAST in the block.

Exit codes:
- `0` — success; the `## Meta` block on stdout.
- non-zero — no fab context (change unresolved or `.status.yaml` absent); nothing on stdout. `/git-pr` treats this (or empty stdout) as "omit the Meta block", matching the legacy `{has_fab} = false` path.

Graceful degradation: an unreachable `gh` leaves owner/repo empty so Pipeline stages render as plain-text labels (never a hard error); a missing/failed merge-base drops only the Impact block.

Consumers: `/git-pr` Step 3c (renders the PR body `## Meta` block, pasted verbatim).

---

## fab memory-index

```
fab memory-index [--check [--json]] [--rebuild]
```

Deterministically (re)generates the `docs/memory/` index **and log** files so agents never
hand-edit them — the deterministic replacement for the hand-maintained index rows (and per-file
`## Changelog` tables) that previously lived in the hydrate / `docs-reorg-memory` skill prose.
Modeled on `fab pr-meta` (pure `RenderRoot`/`RenderDomain`/`RenderLog` + a `Gather` I/O
orchestrator in `internal/memoryindex`), so the output is byte-for-byte stable across runs and
stops the per-row / per-changelog-row merge conflicts on the hot `description` / `Last Updated`
cells. It produces the generated half of the **FKF** format (Fab Knowledge Format — see
`$(fab kit-path)/reference/fkf.md`): per-folder `log.md`, the `type: memory` round-trip mechanism, and the
root-index `fkf_version` frontmatter.

What it writes:
- **Root `docs/memory/index.md`** — **domains-only** (`| Domain | Description |`), prefixed with
  the FKF `fkf_version: "0.1"` frontmatter block (the **only** `index.md` permitted frontmatter
  beyond the generator's own output — FKF §8; no domain/sub-domain index carries it). The legacy
  inlined per-file "Memory Files" column is dropped (it silently drifts). Each domain row's
  Description is read from that domain `index.md`'s `description:` frontmatter.
- **Every `docs/memory/{domain}/index.md`** — file rows (`| File | Description | Last Updated |`)
  for each non-`index` `.md` file, plus a `description:` frontmatter line carrying the domain's
  curated one-liner (round-tripped so the root row survives regen). When the domain contains
  sub-domains, a `## Sub-Domains` table is appended referencing each (`[sub](sub/index.md)`) —
  emitted only when sub-domains exist, so a flat domain index is byte-identical to before.
- **Every `docs/memory/{domain}/{sub-domain}/index.md`** — a sub-domain is a folder one level
  under a domain dir holding ≥1 non-`index` `.md`. It gets its own generated index using the
  same file-row contract as a domain index (relative `[file](file.md)` links are correct from
  the sub-domain folder). Recursion is one level only: `{domain}/{sub-domain}/{topic}.md`
  (depth 3, the max bound). Deeper nesting is surfaced as a depth warning, not an extra index
  tier. An empty sub-folder (no `.md`) is skipped — no spurious index.
- **A per-folder `log.md`** (FKF §6, **C-lite**) for every domain **and** sub-domain folder that
  has attributable git history — `# Log — {Title}` + a `Do not hand-edit` generated-comment
  header, then date-grouped (`## YYYY-MM-DD`, newest first) entries. Each entry is an optional
  leading bold **verb** (`**Creation**` / `**Deprecation**` / `**Update**`, derived from the
  commit's git name-status: `A`→Creation, `D`→Deprecation, `M`/`R`/`C`→Update; omitted when
  ambiguous), a **bundle-relative** link `[base](/{domain}[/{sub}]/base.md)` (beginning with `/`,
  FKF §7), the change's one-line **summary**, and the `(change-id)` in parens. A folder with no
  attributable history is skipped (no empty `log.md`). `log.md` is a single-writer generated
  artifact, same discipline as `index.md` — it replaces the per-file `## Changelog` tables FKF
  removes.
- **Freeze-on-write `log.md` (FKF §6.4).** The existing `log.md` is **authoritative and
  write-once** — a pure projection of *live* git is not deterministic (squash-merge rewrites commit
  subjects/counts and branch-delete makes the originals unreachable), so a from-scratch regen churns
  every contributor's `log.md`. Instead, `fab memory-index` reads the existing `log.md` back into
  entries (parsing the §6.2 render — same grammar as `log.seed.md`), treats those entries as
  **immutable** (never reworded / re-dated / dropped), and **appends only** newly-discovered entries.
  The append/dedup key is **`(file-base, change-id)`** (NOT the commit hash `%H` — squash +
  branch-delete makes the hash unreachable, the exact operation being defended; the change-id
  survives in the folder name + registry): an attributable projected entry is appended only when no
  existing entry already records that `(file-base, change-id)` pair, so a re-run, or a re-projection
  after a squash that preserved the change token, is a no-op. **Unattributable commits are frozen,
  not re-projected**: an entry with no registry change-id already in `log.md` stays verbatim, and a
  NEW unattributable commit (a migration, a direct-`main` edit, a squash that dropped the branch
  token) is NOT projected after first write (accepted tradeoff: tooling commits leave no log trace).
  **Bootstrap is not a special mode** — the first run on a folder with no `log.md` is just the first
  append into an empty log (unattributable commits ARE projected and frozen there); there is no
  `--first-generation` flag, and bootstrap shares one code path with every later run. The
  `log.seed.md` seed-merge is preserved (merged beneath the projection at first write / `--rebuild`).
- **Seed-merge (FKF §6 — `log.seed.md`).** A folder MAY carry a curated `log.seed.md` sidecar in
  the §6.2 entry format (`## YYYY-MM-DD` headings + `- {**Verb** }[base](/bundle/rel.md) — summary
  ({id})` lines). It is a **read-only input** — like `description:` frontmatter — never written by
  the generator, so the single-writer discipline holds (`fab memory-index` remains the sole writer
  of `log.md`; the seed is just another gathered input). Its entries are parsed and **merged
  beneath the git-projected entries** into the generated `log.md`: unioned by date (newest first;
  within a date the git-projected lines render before the seed lines), de-duplicating any seed entry
  byte-equal to a projected one. The merge is **idempotent** — a seed entry that already matches a
  projected entry is dropped, so a re-run is byte-stable and `--check` stays clean. The seed
  preserves its OWN authored dates (independent of git), which is why it can carry pre-FKF history
  that no live `.status.yaml` `summary:` could regenerate (the oovf cutover seeds the pre-FKF
  `## Changelog` rows here — DECISION b). A folder whose only history is a `log.seed.md` (no
  attributable git commits) still emits a `log.md`; `log.seed.md` is excluded from topic-file
  gathering (never an index row), exactly like `index.md` / `log.md`.
- **`type: memory` frontmatter** is **preserved** (round-tripped) when present on a file the
  generator owns — `fab memory-index` ships the *mechanism* only. It does **not** author or
  bulk-stamp `type:` into topic files. Authoring is the memory writers' job: the canonical
  memory-file template (`$(fab kit-path)/templates/memory.md`) carries the `type: memory`
  constant, which hydrate and `/docs-hydrate-memory` stamp onto the new files they author, and
  `docs-reorg-memory` stamps onto any genuinely new topic file a split creates — while
  **preserving** the `type: memory`/`description:` frontmatter byte-for-byte on moved files
  (a move never re-stamps; FKF §3.1, §7). Bulk-stamping the existing tree is a separate,
  later FKF-adoption change — `fab memory-index` provides the preserve-when-present round-trip,
  not the authoring.

Data sourcing (all read by the command itself):
- Each topic file's **H1** (first `# ` line) and **`description:` frontmatter** (via
  `internal/frontmatter`). A file with no `description:` renders `—` in that cell (never errors).
- **"Last Updated"** and the **`log.md` history** both come from ONE batched
  `git log --date=short --name-status -- docs/memory` pass (newest-first). The index takes the
  first date seen per path (equivalent to the old per-file `git log -1 --date=short --format=%ad
  -- <file>` defaults, kept only as the per-file fallback when the batched call fails); the log
  takes the full per-path commit list (date + subject + name-status) from the **same** pass — no
  per-file `git log` spawns. "Last Updated" degrades to `—` when git records nothing for a file —
  uncommitted file, worktree, shallow clone, squash/rebase, or git unavailable — mirroring how
  `fab pr-meta` degrades on missing git/gh context; when the whole batched pass fails, **no
  `log.md` is written** (the log surface degrades to absent, never an error).
- The **`log.md` summary + change-id** are joined from two sources, neither hand-edited (FKF §6):
  each change's `.status.yaml` **`summary:`** field (the *what* — set via `fab status
  set-summary`; absent → the change **slug** is projected instead, FKF §6.3), and the
  **change-id** recovered from the commit and **gated against the change registry**
  (`fab/changes/*` + `fab/changes/archive/**` give the canonical `(id, folder)` set). The id is
  recovered from a `{YYMMDD}-{XXXX}-{slug}` (or registered `{XXXX}`) token in the commit message.
  The merge-commit branch token (`Merge pull request #N from owner/<folder>`) is the **only
  recoverable token shape**, and it is effective **only on legacy true-merge history** — against
  this repo's now-squash-merged history it recovers ≈0 change-ids in practice, so most entries
  take the degraded path. A commit that resolves to no registered change (a direct edit on
  `main`, pre-FKF history, or — the common case here — a squash-merge whose subject is
  `feat: … (#NNN)` with no branch token) **degrades gracefully**: the `(change-id)` token is
  **omitted** and the descriptive line falls back to the **commit subject** (still a
  conflict-free git projection), or to `—` when even that is empty.

Shape warnings (non-fatal, stderr — the "detect" half of the memory-tree-shape work):
- `⚠ docs/memory/<domain> has <N> topic files (soft bound: ~12) — consider splitting into sub-domains`
  when a folder holds more than ~12 topic files.
- `⚠ docs/memory/<domain>/<sub>/<deep> exceeds depth 3 — consider flattening` when nesting
  exceeds 3 levels under `docs/memory/`.
- Reserved domains **`_shared/`** and **`_unsorted/`** are **exempt** from the width warning.
- Warnings are advisory: they never block, never modify files, and never affect the byte-stable
  index output (so a regen-with-warnings is still idempotent).

Flags:
- `--check` — write nothing; classify the rendered-vs-existing drift (across every index **and
  `log.md`** target) by **severity** and encode it in the **exit code** (see Exit codes). Useful
  as a staleness guard (CI / preflight) AND as a destructive-loss guard (refuse-before-regen). The
  drift detection is the same byte-compare the write path uses; the destructive-loss half is a
  classifier + a small parser over the *existing* index rows/headings (pure functions in
  `internal/memoryindex`, unit-tested like `RenderRoot`/`Gather`) — and is skipped for `log.md`
  targets (always benign drift).
- `--json` (with `--check`) — emit the loss report as a single JSON object on **stdout** and
  suppress the human-readable text; the exit code is unchanged. Mirrors the `fab pane` /
  `fab migrations-status` `--json` convention (snake_case). Shape:
  `{"tier": 0|1|2, "drift": bool, "losses": [{"category": "description"|"tombstone"|"grouping", "path": "<repo-rel index>", "detail": "<lost text | dropped link target | flattened heading>"}]}`.
  Consumed by `/docs-reorg-memory`'s compatibility detection.
- `--rebuild` — **DESTRUCTIVE** freeze-on-write escape hatch (FKF §6.4): discard the accumulated
  frozen `log.md` state and re-project every `log.md` from current git (the pre-freeze behavior, made
  explicit and opt-in — it re-projects unattributable commits too). It can rewrite or drop frozen
  lines, so use it only for a corrupted frozen log or a deliberate re-baseline — never the default
  path. The `log.seed.md` seed-merge still applies beneath the re-projection. **Ignored with
  `--check`** (which never writes): `--check` always compares against the non-destructive
  freeze-on-write merge. The 2.5.5→2.6.0 re-baseline migration runs `fab memory-index --rebuild` +
  commit once to move an existing project onto freeze-on-write, after a pre-check that the running
  binary understands `--rebuild` (probe `fab memory-index --help`; abort with "upgrade the binary
  first" if absent).

Tiered `--check` exit codes (loss is a strict subset of drift — one render pass serves both;
`log.md` and the root `index.md` `fkf_version` frontmatter are classified too, but only ever as
benign drift — see below):
- **`0`** — clean: every index **and `log.md`** file is byte-identical to its regenerated form
  (no regen needed).
- **`1`** — **benign drift**: regen would change content but destroy nothing (e.g. an *improved*
  `description:`, a refreshed `Last Updated` date, a stale `log.md`, a `log.md` gaining merged
  `log.seed.md` entries, or absent/changed FKF frontmatter). This is the former "out of date"
  condition — existing consumers treating "non-zero = stale" still work unchanged. **All `log.md`
  and FKF-frontmatter drift is benign (tier 1)** — a `log.md` is a C-lite git projection (plus any
  merged seed), not a row-table index, so the three destructive-loss detectors below are skipped for
  it, and FKF added **no new tier-2 category** (FKF / OQ4 decision); a preserved seed is never
  reported as destructive loss. **Under freeze-on-write (FKF §6.4) `--check` compares the committed
  `log.md` against the freeze-on-write MERGE, not a from-scratch projection**: a committed log that
  is a valid **superset** of the merge (it carries frozen lines the live history no longer shows)
  **PASSES** (the case byte-equality false-fails today). A `log.md` benign FAIL (tier 1) means the
  committed log is **missing** a projected attributable `(file-base, change-id)` entry (forgot to
  regenerate-and-commit), or a frozen line was **hand-edited** in a render-unstable way (single-writer
  discipline violated — a clean reword that round-trips through the §6.2 grammar is accepted as the
  new frozen truth).
- **`2`** — **destructive loss**: regen would wipe curated/historical content. Three
  **index-only** categories, the mechanical form of `/docs-reorg-memory`'s prose signals: (1) a
  curated **description** that would regenerate to `—` (the file lacks `description:` frontmatter);
  (2) a **tombstone** row whose `docs/memory/`-relative link target is absent on disk
  (external/absolute links excluded — no false positives); (3) a custom structural **grouping**
  heading in the root `index.md` beyond the domains-only table. (`log.md` targets never reach
  these.) Writes nothing; enumerates each loss to stderr by category; the human-readable output
  ends with the pointer `→ run /docs-reorg-memory to remediate (it relocates removal-history rows
  to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory)
  before regenerating.` (`/docs-reorg-memory` is the orchestrator that handles all three categories
  — it relocates tombstone rows itself and dispatches `/docs-hydrate-memory` backfill mode for the
  descriptions; backfill alone does not relocate tombstones.)

Callers pick a threshold: **CI / pre-commit** fails on exit ≥ 1 (any drift); the **hydrate /
reorg refuse-before-regen guards** fail only on exit == 2. A **born-FKF / born-compatible fab-kit
tree is provably never exit 2** (frontmatter present, no off-disk rows, domains-only root, native
`log.md` exactly what the generator produces) — so the refuse-before-regen guards are no-ops on
native trees and only ever fire on a pre-fab-kit tree.

Other exit codes:
- non-zero (1) — an operational error: `docs/memory/` not found (or another `Gather` failure), or a
  write failed. `Gather` runs before the `--check` branch, so a `--check` run also exits 1 on these —
  the exit-1 / exit-2 *tier* codes above apply only once gather succeeds and the comparison runs.
  Writes happen only on non-`--check` runs, so a write failure is non-`--check`-only.

Consumers: the hydrate skills (`/docs-hydrate-memory` Step 4 + its refuse-before-regen guard,
`/fab-continue` hydrate + its defense-in-depth guard) and `/docs-reorg-memory` (compatibility
detection via `--check --json`, index regen after diagnosis) — all call `fab memory-index`
instead of hand-maintaining index rows.

---

## fab fab-help

```
fab fab-help
```

Scans skill frontmatter from the cache kit, groups skills by category (Start & Navigate, Planning, Completion, Maintenance, Setup, Batch Operations), renders formatted overview. Excludes `_`-prefix and `internal-` prefix skills. Batch entries read dynamically from `fab batch` cobra subcommands. Unmapped → "Other".

Output: version header, workflow diagram, grouped commands, typical flow, packages section (wt, idea).

(The command name is `fab-help` — not overriding cobra's built-in `help`.)

---

## fab help-dump

```
fab help-dump
```

**Hidden, CI/build-time-only command.** Marked `Hidden: true`, so it does not appear in `fab --help` and is excluded from its own dumped tree. Takes no arguments. Walks the live cobra command tree of the rich `fab` CLI programmatically (not by regex-parsing `-h` text) and writes the frozen shll.ai "command reference" contract JSON to stdout.

Contract shape (`schema_version: 1`):

```json
{
  "tool": "fab",
  "version": "<main.version, from ldflags>",
  "captured_at": "<RFC3339 UTC>",
  "schema_version": 1,
  "root": {
    "name": "fab",
    "path": "fab",
    "short": "...",
    "usage": "...",
    "text": "<raw -h body, byte-for-byte>",
    "commands": [ /* recursive Node[]; [] for a leaf, never null */ ]
  }
}
```

Per node: `name=cmd.Name()`, `path=cmd.CommandPath()`, `short=cmd.Short`, `usage=cmd.UseLine()`, `text=cmd.UsageString()`. At every level the walk drops `completion`, `help`, and any `Hidden` command, then sorts surviving children by `Name()` for byte-stable output. JSON is 2-space indented with HTML escaping disabled, so `<`, `>`, `&` in help text are preserved verbatim.

`tool` is the literal `"fab"` (the user-facing binary); the *output file* is named `help/fab-kit.json` (the repo/site slug) — these intentionally differ. Consumed by `.github/workflows/release.yml` (Help-dump → shll.ai step) to deliver an auto-merging PR into `sahil87/shll.ai`.

---

## fab operator

```
fab operator
```

Singleton tmux-tab launcher for `/fab-operator`. Requires `$TMUX` (else exit 1, `ERROR: not inside a tmux session`). The singleton check is an **exact, server-wide** window-name match: `tmux list-windows -a` enumerated and compared exactly (never tmux target resolution, whose prefix/glob fallback would let e.g. `operator-logs` mask the real check; `-a` enforces the one-operator-per-SERVER invariant across sessions). If a window named exactly `operator` exists anywhere on the server → select it by window ID, switching the client to its session when needed (`Switched to existing operator tab.`); else create the window running `{spawn_command} '/fab-operator'` (`Launched operator.`).

**Launch cwd (no git-repo dependency)**: the new window's working directory (`tmux new-window -c <dir>`) is resolved by trying `git rev-parse --show-toplevel` first and falling back to `os.Getwd()` when that fails — so the operator launches **inside a git repo** (cwd = repo root, today's behavior) **or from a neutral parent directory** (cwd = current directory). It no longer hard-fails with `cannot determine repo root`; it errors only if both git-root resolution AND `os.Getwd()` fail. This matches the per-tmux-server, cross-repo singleton model: the operator's natural launch point is a neutral dir with no `fab/` project.

**Spawn command resolution (no `fab/`-project dependency) + doing-tier model**: when a `fab/` project is resolvable (`resolve.FabRoot()` succeeds), the base command is `agent.spawn_command` from that project's `fab/project/config.yaml` (falls back to `claude --dangerously-skip-permissions` if the key is missing/null/empty). When `resolve.FabRoot()` **fails** — the operator is launched from a neutral directory with no `fab/` project anywhere up the tree (its natural cross-repo home) — this is **non-fatal**: the base command is the built-in `spawn.DefaultSpawnCommand` (`claude --dangerously-skip-permissions`) and no project `agent.spawn_command`/`agent.tiers` is read. The operator then launches its coordinating agent on the **doing tier**: it shells `fab resolve-agent apply` (`apply` is the canonical doing-tier stage in the fixed stage→tier mapping), parses the `model=`/`effort=` profile, and appends `--model <model> --effort <effort>` to the END of the spawn command (last-wins; each flag omitted when its value is empty, per the `empty ⇒ omit` convention). On any failure (the installed `fab` lacks `resolve-agent`, no resolvable fab project, or unparseable output) the doing tier falls back to the built-in default `{claude-opus-4-8, high}`. So a `fab/`-less launch composes a fully-defaulted command: default spawn command + doing default `{model, effort}`.

### fab operator tick-start

```
fab operator tick-start
```

Called at start of each operator tick. Increments `tick_count`, writes `last_tick_at` (ISO 8601 UTC) to the **server-keyed** state file (not the old repo-rooted `.fab-operator.yaml`). Stdout:

```
tick: N
now: HH:MM
```

**State path** (server-keyed, XDG): `<XDG_STATE_HOME>/fab/operator/<server-slug>.yaml`, where the base is `$XDG_STATE_HOME` (when set and absolute) else `$HOME/.local/state` — uniform on Linux and macOS (never `~/Library/...`). `<server-slug>` is derived from the tmux socket path (`#{socket_path}`) by escaping literal `-` to `--` then mapping separators to a single `-` (e.g. `/tmp/tmux-1000/default` → `tmp-tmux--1000-default`); the escape keeps the mapping collision-free so distinct sockets never share a state file. One operator-per-tmux-server gets one state file that survives a server restart (same `-L` label → same socket path). Falls back to slug `default` when tmux can't be queried. No migration of old repo-rooted `.fab-operator.yaml` files — they are abandoned in place.

### fab operator time

```
fab operator time [--interval <duration>]
```

Pure time query (no writes).

- Without `--interval`: `now: HH:MM`
- With `--interval 3m`: `now: HH:MM\nnext: HH:MM` (now + interval)

Duration is Go format (`3m`, `5m`, `2m`). Invalid → exit 1.

---

## fab spawn-command

```
fab spawn-command [--repo <path>]
```

Prints a repo's configured agent spawn command to stdout. With `--repo <path>`, reads `agent.spawn_command` from `<path>/fab/project/config.yaml`; without `--repo`, resolves the current repo's config via upward `fab/` search (same source as `fab operator`). Falls back to `claude --dangerously-skip-permissions` when the key is missing/empty or the file is unreadable. Lets the operator fetch a **target** repo's spawn command (e.g. to spawn an agent into a different repo with that repo's configuration) instead of only its own.

---

## fab batch

Multi-target operations: `fab batch <new|switch|archive> [--list] [--all] [targets...]`. The `new` and `switch` subcommands create tmux windows and require `$TMUX`; `archive` runs in-process and does not.

- **`new`** — parse `fab/backlog.md` pending items (`- [ ] [xxxx]`), create worktrees, open tmux windows, start agents with `/fab-new {description}`. No args → `--list`. IDs → one worktree tab each (`wt create --non-interactive --worktree-name {id}`, window `fab-{id}`, `{spawn_command} '/fab-new {description}'`). `--all` → all pending. Handles continuation lines. Launch failures are surfaced per item: a failed `wt create` or `tmux new-window` prints `[{id}] FAILED: ...` (the tmux line names the already-created worktree path as the cleanup/recovery hint) with the child's stderr included, never aborts the remaining items, and the command exits non-zero when any item failed (`ERROR: {N} of {M} item(s) failed to launch`). Unknown/empty backlog IDs remain warn-and-skip (exit 0). Requires `$TMUX` (else exit 1, `ERROR: not inside a tmux session`); empty pending backlog with `--all` → exit 1, `ERROR: No pending backlog items found.`.
- **`switch`** — resolve change names (in-process via `resolve.ToFolder`, like the rest of the family — no `fab`-on-PATH dependency; an unresolvable name warns with the resolver's specific error, e.g. `Multiple changes match…`, and skips), create worktrees with branch names (applying `branch_prefix` from config), start agents with `/fab-switch {change}`. No args → `--list`. `--all` → all active changes (excludes `archive/`); empty set → exit 1, `ERROR: No changes found.`. Branch naming: `{branch_prefix}{folder_name}`. Requires `$TMUX` (else exit 1, `ERROR: not inside a tmux session`).
- **`archive`** — find changes with `hydrate: done|skipped`, then archive each mechanically in a Go loop via `internal/archive.ArchiveWithBacklog` (move, index, backlog mark-done, pointer). No agent or Claude session is spawned; resolution uses `resolve.ToFolder` (no `fab`-on-PATH dependency). No args → `--list` (aligned with new/switch; the bulk action requires explicit `--all` — archive moves are effectively irreversible within the loop). Per change prints `{name} — archived` (with ` (backlog marked done)` when applicable; when a post-archive step — index update or backlog mark — fails, the change still prints `archived` plus a stderr `warning:` line and counts as archived, not failed), `already archived, skipping` (covers genuinely-archived names — counted as skipped), or `FAILED: {err}`; a single failure never aborts the batch. Footer: `Archived {N}, skipped {M}, failed {K}.`. Exit semantics: an empty `--all` set is a benign no-op (`No archivable changes found.` + zero footer, exit 0); after the loop runs, non-zero when `failed > 0` (`ERROR: {K} change(s) failed to archive`); explicitly named targets where none resolves to an active *or* archived change → exit 1, `ERROR: No valid changes to archive.`.

---

## Common Error Messages

All strings below match `internal/resolve/resolve.go` verbatim (placeholders shown as `{arg}`):

| Error | Cause | Fix |
|-------|-------|-----|
| `No change matches "{arg}".` | An override was given but matches no folder in `fab/changes/` (exact match tried first, then substring — both case-insensitive) | Check `fab change list` |
| `Multiple changes match "{arg}": {list}.` | Ambiguous substring matched multiple folders | Use a more specific identifier (4-char ID or full folder name) |
| `No active changes found.` | An override was given but `fab/changes/` contains no change folders at all | Run `/fab-new` or `/fab-draft` |
| `No active change. Run /fab-new <description> to start one, or /fab-switch to activate an existing one.` | No override, `.fab-status.yaml` symlink absent **or dangling** (its target `.status.yaml` no longer exists — e.g. change archived/deleted underneath), and zero candidate changes (a single candidate would auto-resolve) | Follow the message — `/fab-new` or `/fab-switch` |
| `No active change (multiple changes exist — use /fab-switch).` | No override, symlink absent **or dangling**, and multiple changes exist (no single-change guess possible) | Run `/fab-switch` |
| `fab/changes/ not found.` | The `fab/changes/` directory is missing | Run `fab init` or check the CWD is the repo root |

> **Typed resolution errors**: the `No change matches` / `No active change` messages are classified `ErrNotFound`, and the `Multiple changes match` / `multiple changes exist` messages are classified `ErrAmbiguous` (the surfaced text is unchanged). Internal callers branch on these with `errors.Is` — e.g. archive soft-skip treats only `ErrNotFound` as "maybe already archived" (idempotent skip) and surfaces `ErrAmbiguous` as a real error instead of conflating the two.
