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
- fab config (reference, show, init --system)
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
- fab agent
- fab batch
- Common Error Messages

---

## Calling Convention

`fab <command> <subcommand> [args...]`. `fab` is a router dispatching workspace commands (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`) to `fab-kit` and everything else to the per-version `fab-go` binary resolved from the pinned version in `fab/.fab-version` (a one-line plain-text sibling of `fab/.kit-migration-version`; for a not-yet-migrated repo the reader falls back to a legacy `fab_version:` key in `fab/project/config.yaml` for one compat window). `--version`/`-v`/`--help`/`-h`/`help` are handled inline. `fab-go` auto-fetches from GitHub releases on cache miss.

`fab -h` composes help from both binaries. `fab --version` prints the system binary version; inside a fab repo a second line shows the project-pinned version.

### Workspace Command Exit Semantics

Lifecycle commands fail loudly — a non-zero exit is the failure signal scripts and skills rely on. **Exception**: `sync` and `fab-kit migrations-status` reserve a distinct exit `3` for the "not a fab-managed repo" precondition (see the `sync` row and the `fab migrations-status` section) — that is *not* a failure but a "not applicable here" signal, so a caller branching on the exit code MUST treat exit `3` separately from the generic exit `1` = failure. All other lifecycle failures use the generic non-zero (exit `1`) path.

| Command | Failure behavior |
|---------|------------------|
| `init` | Requires a git repository — exits non-zero with `fab init requires a git repository — run 'git init' first` BEFORE any download or config write. Sync failure during init also exits non-zero |
| `update` | Exits non-zero with `fab-kit was not installed via Homebrew` when the binary is not brew-installed (go-install/manual/CI); brew failures also exit non-zero |
| `upgrade-repo` | Runs sync first, then (only AFTER sync succeeds) stamps `fab/.fab-version` and auto-runs `fab config upgrade` against the pinned fab-go to reconcile `config.yaml`'s reference fence (fail-open: a fab-go predating the subcommand prints a reminder and the upgrade continues). On sync failure: exits non-zero with `sync failed: ... — run 'fab sync' to repair, then re-run 'fab upgrade-repo'`, never prints `Updated: x -> y`, stamps nothing, and a re-run retries (no "Already on the latest version" short-circuit of the broken state). **Unaffected by the sync/migrations-status exit-`3` contract**: run outside a fab-managed repo, `upgrade-repo` still exits the generic `1` with the same `not in a fab-managed repo. Run 'fab init' to set one up` stderr message — it deliberately does NOT use `RequireManagedRepo()`/exit `3`, because its guard tolerates a repo with a `config.yaml` but no resolvable pinned version (a partially-managed, not fully-unmanaged, state), a distinct semantic left out of scope. Do not conflate the two: only `sync` and `migrations-status` carry the exit-`3` signal |
| `sync` | Exits non-zero when any skill deployment write fails (per-skill `WARN:` lines on stderr, `failed N` in the agent tally) or when scaffolding writes fail. The version guard exits non-zero whenever it trips: either `fab-kit was updated to vX — re-run 'fab sync'` (auto-update landed; the current run still ran old code) or actionable too-old instructions (non-brew install, Homebrew tap release lag) — it never continues on a binary older than the pinned version (`fab/.fab-version`, with the legacy config.yaml `fab_version:` fallback). **Branchable exit code** (mirroring the pane family's use of distinct branchable exit codes — though with the opposite polarity: for the pane verbs exit `3` is the real failure and `2` the benign signal, whereas here `3` *is* the benign "not applicable" signal and no exit `2` is involved): run outside a fab-managed repo (no `fab/project/config.yaml` on any ancestor), `sync` prints `not in a fab-managed repo. Run 'fab init' to set one up` to stderr and exits `3` — a distinct "not applicable here" signal, NOT the generic `1` = failure — so callers (e.g. `wt`'s default init, operator scripts) can branch on "not a fab-managed repo" vs. "a real sync failure" without replicating fab's config walk-up. This holds unconditionally, including outside any git repository: the managed-repo check is a `config.yaml` walk-up gated before the git-root resolution, so `sync` is symmetric with `fab-kit migrations-status` (which has no git precondition). The value is the `internal.ExitNotManaged` constant, shared with `fab-kit migrations-status` (below) via `RequireManagedRepo()`. Genuine sync failures above stay exit `1` |

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

`fab preflight`, `fab score`, `fab log command`, `fab change`, `fab resolve`, `fab status` — headline coverage lives there. Sections below document the remaining commands (`fab pane`, `fab doctor`, `fab migrations-status`, `fab kit-path`, `fab shell-init`, `fab impact`, `fab pr-meta`, `fab memory-index`, `fab fab-help`, `fab help-dump`, `fab operator`, `fab agent`, `fab batch`) and extended flag details for the above.

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
| `archive` | `archive <change> [--description "..."]` | Move to `archive/`, delete the change's `.fab-dispatch/{id}/` dispatch state (transient comms, not history — not recreated on restore; best-effort), update index, mark backlog item done, clear pointer. `--description` is optional — defaults to the intake title (humanized-slug fallback). Re-archiving an already-archived change is a soft skip (exit 0) that still re-attempts the backlog mark (idempotent — recovers a previously-failed mark; silent, the plain soft-skip line is unchanged). |
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
| `refresh` | `refresh <change>` | Recompute the artifact-derived fields — `change_type` + `confidence` (from `intake.md`) and `plan.generated`/`task_count`/`acceptance_count`/`acceptance_completed` (from `plan.md`) — from on-disk artifacts, under the status flock (single load-mutate-save). The pull-based successor to the removed `artifact-write` hook: heals a hook-bypassing edit (sed, direct write) or a non-Claude agent's artifact write. Respects `change_type_source: explicit` (keeps an explicitly-set type). A missing artifact is a safe no-op; exits non-zero only on a genuine failure (unresolvable change, unreadable `.status.yaml`). Self-healed automatically at `advance`/`finish`/`preflight`, so skills need not call it directly |
| `set-change-type` | `set-change-type <change> <type>` | Sets `change_type` AND marks `change_type_source: explicit`, so `fab status refresh` (and the self-healing transitions that run it) stops re-inferring/overwriting it — it only re-infers when the source is absent or `inferred` |
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

`fab status start` and `fab status finish` honor an optional `stage_hooks` map in `fab/project/config.yaml` (not seeded by the scaffold — add the key by hand). This is a pipeline-transition mechanism, unrelated to Claude Code settings hooks (the `fab hook` command family was removed in 2.14.0):

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

Pure query (no side effects) — resolves a pipeline **stage** (or a role **tier** name) to its `{provider, model, effort}` agent profile for sub-agent dispatch. Consumed by the orchestrators (`/fab-ff`, `/fab-fff`, `/fab-proceed`) and `/fab-continue`'s sub-agent dispatch, which call it immediately before dispatching each stage's sub-agent, and by `fab agent` / the operator launcher (tier-name resolution).

```
fab resolve-agent <stage|tier> [--alias]
```

The positional argument is either one of the six pipeline stages (`intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`) or one of the five role-tier names (`default`, `operator`, `doing`, `review`, `fast`). The two sets are **disjoint**, so a tier name is accepted positionally alongside a stage name: a stage maps through the fixed stage→tier mapping; a tier resolves directly.

**Resolution**: maps a stage → its tier via the FIXED fab-owned stage→tier mapping (`default`: intake (advisory) / `doing`: apply, review-pr, hydrate / `review`: review / `fast`: ship — NOT user-overridable), then resolves the tier → `{provider, model, effort}`: the project's `agent.tiers.<tier>` override **per-field merged** over the project's `default` tier, over fab-kit's built-in default (`default`: claude/claude-fable-5/xhigh, `operator`: claude/claude-sonnet-5/medium, `doing`: claude/claude-opus-4-8/xhigh, `review`: claude/claude-fable-5/xhigh, `fast`: claude/claude-sonnet-5/low). `agent.tiers` is the sole tier-override surface (no `stage_tiers`, no per-stage escape hatch); the command grammar lives in the top-level `providers:` table. See `docs/specs/stage-models.md`.

**Output** (a `model=` line always, then optional `effort=`, `provider=`, and `dispatch=` lines; byte-stable for the same config):

```
model=<id>
effort=<level>
provider=<name>
dispatch=<command>
```

- The `effort=` line is **omitted** when the resolved tier has no effort (empty/absent); the `provider=` line is omitted when the resolved tier has no provider.
- An **empty model** emits an empty `model=` line — signals "inherit the session/orchestrator model" (today's foreground/no-override behavior). Callers omit the dispatch `model` param in that case.
- The `dispatch=` line is emitted **ONLY when the resolved tier's provider carries a `dispatch_command`** (the CLI-dispatch opt-in), mirroring the effort-omit rule. Its **absence** is the signal for **native Agent-tool dispatch** — and there is **NO fallback to a session command** (a provider's `session_command` is a separate, independent field; `resolve-agent` never falls back to it for dispatch). The emitted command has its `{model}`/`{effort}` placeholders **already substituted** via `internal/spawn`'s template resolution (reused, not reimplemented). Consumed by the `fab dispatch` command family; dispatch-seam skills that only inject `model=`/`effort=` do not read it.

**`--alias` (Claude-Code Agent-tool adapter)**: when set, the `model=` line emits the Claude-Code **short alias** (`opus` / `sonnet` / `haiku` / `fable`) instead of the full versioned ID. This exists because the Claude Code **Agent tool's `model` parameter is a hard enum** that rejects full IDs — sub-agent dispatch must pass an alias. The mapping is prefix-based (`claude-opus-` → `opus`, etc.), so dated variants like `claude-haiku-4-5-20251001` resolve to `haiku`. The **default (flag absent) is** the full ID (the `claude` CLI `--model` flag, used by the operator launcher / `fab agent`, accepts full IDs and keeps resolving WITHOUT `--alias`). The **`effort=`/`provider=` lines are unaffected** by `--alias`. **Empty / non-Claude models pass through verbatim** (an empty `model=` line stays empty — the inherit signal; an unrecognized/non-Claude ID like `gpt-5` is emitted unchanged) — `--alias` is a best-effort adapter, not a Claude-only validator. The **`dispatch=` line ALWAYS embeds the FULL model ID even under `--alias`** — CLI dispatch never aliases (an external CLI's `--model` flag takes a full ID); aliasing is the Agent-tool-only adaptation. So under `--alias` the `model=` line is aliased while the `dispatch=` command still carries the full resolved ID.

```
$ fab resolve-agent apply
model=claude-opus-4-8
effort=xhigh
provider=claude

$ fab resolve-agent apply --alias
model=opus
effort=xhigh
provider=claude

# with the doing tier pointing at a provider that has a dispatch_command
# (apply ∈ doing) — the dispatch= line appears, aliased model= but full-ID dispatch=:
$ fab resolve-agent apply --alias
model=opus
effort=xhigh
provider=codex
dispatch=codex exec -m claude-opus-4-8 -c model_reasoning_effort=xhigh
```

**No validation — verbatim pass-through**: `fab resolve-agent` does NOT validate the provider, model, or effort against any provider's accepted set (provider neutrality — a fab-kit design principle). It echoes the strings as-is — `xhigh`, `reasoning_effort:high`, an empty effort, whatever. A misconfigured pair (e.g. Sonnet + `xhigh`) is NOT corrected by fab; it surfaces as a dispatch-time error in the harness. There is no effort-enum enforcement and no degrade-gracefully drop.

**Exit code**: non-zero only on a real error — an unreadable/malformed config, or an unknown stage/tier name. A stage/tier resolving to a default is success (exit 0).

---

## fab config

`config` is a command group over the project configuration: the pure queries `reference` and `show`, the scaffold/generator `init` (`--system` scaffold, `--project` generator), and the reconciling writer `upgrade` (the group name leaves room for a future `fab config validate`).

```
fab config reference          # commented YAML reference (all options)
fab config reference --json   # machine-readable field table
fab config show               # effective (post-cascade) config, as YAML
fab config show --origin      # effective config + per-field provenance
fab config init --system      # write ~/.fab-kit/config.yaml scaffold
fab config init --project ... # generate fab/project/config.yaml from the registry
fab config upgrade            # reconcile fab/project/config.yaml against the registry
```

### fab config reference

Pure query (no side effects, no file writes) — prints a **fully-commented reference `config.yaml`** to stdout, documenting every available option so users can discover the whole schema from one place.

```
fab config reference          # commented YAML (default)
fab config reference --json   # machine-readable field table
```

Only the `--json` flag; no positional arguments (`fab config reference extra-arg` is rejected, with or without `--json`). Runs from any directory — it reads no project config and depends on no environment state.

**Generated from a per-field metadata table, not hand-written**: the reference is generated by walking an ordered **per-field metadata table** (`internal/configref`) — each row carries the field's canonical `default`, `description`, `scope` (`project`/`system`/`both`), `advertise` flag, and `renamed_from` carry-forward. Every default that has a canonical Go constant is sourced from that constant (`agent.DefaultSessionCommand`, the default tier profiles via `agent.DefaultTier`/`agent.TierNames`, the pipeline stage names via `agent.StageNames`), so there is no second copy of those values and the shown defaults **cannot drift** — strictly stronger than a drift-guard test on hand-written copies. A field's `default` is the *canonical* built-in default (what the config cascade falls back to), not the value the reference shows as an example: `source_paths`/`test_paths` render an example (`- src/`) but their binary default is empty. See `docs/specs/config.md` for the schema.

**`--json`**: emits the same field table as a flat, deterministic JSON array (table/rendering order) — per-field objects `{key, default, description, scope, advertise, renamed_from}` with `renamed_from` omitted when empty (empty on every row today). This is the tooling surface the config-upgrade cascade/upgrade commands and external tools consume. Without the flag, output is the commented YAML exactly as before. Both renderings are pure queries and byte-stable for a given binary version; the JSON key set is guarded against drift from the YAML reference's documented keys. All three metadata fields now have behavioral consumers: `scope` drives the cascade resolver's scope enforcement (`config show`/`show --origin` below) and the `config init --system` scaffold filter; `advertise` drives which fields `fab config upgrade` / `init --project` scaffold into the managed reference fence; `renamed_from` drives `fab config upgrade`'s mechanical rename carry-forward (empty on every row today — it serves future renames).

**Full schema coverage**: covers BOTH the binary-consumed keys (modeled on the `Config` struct) AND the skill-consumed keys (read by markdown skills, invisible to Go reflection) — `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `providers.*` (`session_command`/`dispatch_command`), `agent.tiers.*` (`provider`/`model`/`effort`), `stage_hooks.*`, `branch_prefix`. (The retired `review_tools` block moved to `fab/project/code-review.md` § Review Tools; `agent.spawn_command` moved to `providers.claude.session_command`; `fab_version` moved OUT of config.yaml to the plain-text sibling `fab/.fab-version` in 2.15.0 and is no longer a config-file key.) Baseline keys appear live with example values (including the `providers:` claude entry and the five `agent.tiers`); the opt-in override blocks (`stage_hooks`, `branch_prefix`) appear commented-out with fab-kit's built-in defaults shown, so uncommenting is opting in. The `providers:` block ships as a **three-provider starter template** — `claude` (built-in default: `session_command` live, `dispatch_command` commented), plus fully-commented `codex` and `gemini` blocks each showing both command fields — so adding a non-claude provider is a copy-and-adapt rather than composing command grammar from scratch (codex/gemini remain template text only; no new built-in providers are added in Go). Gemini's commands carry no `{effort}` placeholder (the gemini CLI has no reasoning-effort flag) and no `-p` on `dispatch_command` (gemini reads the `fab dispatch` stdin-piped prompt in non-TTY mode).

**Output**: byte-stable for a given binary version (same convention as `fab resolve` / `fab resolve-agent`). The emitted document round-trips — its live keys parse cleanly back into `Config`.

**Exit code**: 0 on success (pure query — no runtime error paths). A usage error (e.g. an extra positional argument, rejected by `cobra.NoArgs`) exits non-zero. Writes no file.

### The config cascade (project > system > defaults)

`fab config show` and `show --origin` display the **effective** config after resolving the three-layer cascade the loader (`internal/config.LoadPath`) applies to *every* config read:

1. **project** — `fab/project/config.yaml` (highest precedence)
2. **system** — `~/.fab-kit/config.yaml` (user-global, all repos on the machine)
3. **built-in defaults** — the Go tables in the binary

The two files merge by **per-field deep merge**: maps merge per-key (the `agent.tiers` precedent), lists replace (never concatenate), scalars replace — project wins. The cascade is **fail-open** (config must never brick): an absent system file is byte-identical to before; a malformed/unreadable system file emits a `fab: warning:` on stderr and is skipped; a malformed *project* file still errors as before. **Scope enforcement**: a project-scoped field placed in the system file is ignored with a `fab: warning:` (only `scope: system`/`both` fields — today `agent.tiers`, `providers` — are honored there). Warnings go to stderr and never change stdout contracts or exit codes.

### fab config show `[--origin]`

Pure query (no file writes) — resolves the config for the current repo and prints it. Without a flag it prints the merged config of the two **files** (project over system) as YAML; built-in defaults are **not** materialized into this output — they apply at point-of-use and are surfaced explicitly only by `--origin`. With `--origin` it prints, per field, the effective value (built-in defaults composed in) alongside its **provenance** — the project path, the system path, or `default` (the `git config --show-origin` precedent) — with **per-key drill-down for map-valued fields** (`agent.tiers`, `providers`), the honest granularity where maps merge per-key. `--origin` is the way to see *why* an override did or didn't take: a typo'd override (`agent.teirs:`) leaves the intended field showing origin `default`, surfacing a mistake that silently no-ops today.

```
fab config show               # effective config as YAML
fab config show --origin      # each field: value + origin (project path / system path / default)
```

No positional arguments (`cobra.NoArgs`). Requires a fab repo (walks up for `fab/`, like `fab preflight`). Writes no file.

### fab config init (`--system` | `--project`)

`fab config init` has two mutually-exclusive modes. Bare `fab config init` (neither flag) is a **usage error**; passing both errors.

**`--system`** writes a `~/.fab-kit/config.yaml` **scaffold** — a header explaining the system layer, then **only** the system-overridable fields (`scope: system`/`both` — today `agent.tiers` and `providers`), **all commented**, generated from the same per-field metadata table as `fab config reference` so it cannot drift from the schema. It is the user's answer to "what can I safely override at the system level".

**`--project`** generates a fresh `fab/project/config.yaml` from the registry — the retirement path for the hand-maintained scaffold `config.yaml` (deleted in 2.15.0). It writes the **A-class identity fields** (`--name`, `--description`, `--source-path` repeatable, `--test-path` repeatable) **live** above the managed reference fence, then the fence of commented C fields. `agent.tiers` is **not** pinned (presence=intent — an init-pinned tier would be an accidental override that stops tracking fab-kit's defaults). It shares the same fence renderer as `fab config upgrade`, so a generated file and an upgraded file carry a byte-identical fence. This is the shell-out target `fab init` (the fab-kit binary) calls to bootstrap a project config; when the installed fab-go predates it, `fab init` falls open to a minimal embedded stub instead (a fresh repo never fails preflight for lack of a config.yaml).

```
fab config init --system      # write the ~/.fab-kit/config.yaml scaffold (refuses to overwrite)
fab config init --project --name X --description Y --source-path src/ [--test-path "**/*_test.go"]
```

Both modes **refuse to overwrite** an existing target file (non-zero exit, message naming the path) — the file is user-owned once created; there is no `--force`. The `--system` scaffold is fully commented (inert until uncommented).

### fab config upgrade

The **single, comment-aware writer** of `fab/project/config.yaml` going forward (2.15.0). It reconciles the file against the binary's field registry mechanically — retiring the comment-clobbering whole-file rewrite (`setFabVersion`) at the root and replacing the hand-written comment-backfill migration pattern.

```
fab config upgrade            # reconcile config.yaml against the registry (idempotent)
```

Reconciliation, under the A/B/C field-category model:

- **Live (A) fields kept verbatim**, including the user's own comments. *Presence = intent*: a live field is an override even when its value equals the default — it is NEVER auto-removed (B-hygiene "equals default — remove?" is an advisory report line only).
- **The managed fence (C fields)**: `advertise: true` fields not currently overridden are regenerated as a fully-commented scaffold (including parent keys — a live `agent:` over comment-only children is exactly the `agent: null` the old masher produced) inside byte-exact splice anchors: `# >>> fab reference (kit X.Y.Z) >>> …` / `# <<< end fab reference <<< …`. Upgrade rewrites ONLY between the anchors; everything outside is the user's. The fence omits fields already overridden above it (at top-level-key granularity — a live top-level key suppresses the whole scaffolded block under it). A legacy file with no fence gets one appended at the bottom. Content the user places BELOW the fence is never dropped — it is **hoisted above** the fence on the next run and classified like any other live key.
- **Unknown fields parked, never deleted**: a live key no longer in the registry is parked in a `# removed in … (parked by fab config upgrade — delete when done):` block below the fence, its value serialized — appended exactly once, never regenerated away.
- **Renames carried mechanically**: a live field matching a registry row's `renamed_from` is carried to the new key, value verbatim (empty on every row today). A carry is **skipped** (and reported) if the target key is already live, so it never emits a duplicate top-level key.

**Byte-stable and idempotent** — running it twice yields a byte-identical file (the `fab memory-index` discipline). Before writing, the reconciled document is **validated as YAML** and a run that would produce an unparseable file is **refused** (original left untouched) rather than bricking the repo. The write is atomic (`internal/atomicfile`). Requires a fab repo (walks up for `fab/`); `cobra.NoArgs`. `fab upgrade-repo` **auto-runs** it after sync (fail-open: if the installed fab-go predates the subcommand, it prints a reminder and the upgrade continues).

---

## fab hook (removed in 2.14.0)

The `fab hook` command family — `session-start`, `stop`, `user-prompt`, `artifact-write`, and `sync` — was **removed outright** in 2.14.0 (no deprecation shim period). fab no longer registers, writes, or owns any Claude Code hook. An un-migrated `.claude/settings.local.json` that still invokes `fab hook <x>` will now get a cobra *unknown command* error (exit 1) until the `2.13.6-to-2.14.0` migration removes the entries; that migration strips the three session-scoped entries (both the inline `fab hook …` and legacy `on-*.sh` forms) and deletes the dead `.fab-runtime.yaml`/`.lock` files. Two facts that outlived the hooks:

**Agent state is a consumed convention, not a produced hook (ioku).** The former `session-start`/`stop`/`user-prompt` handlers used to WRITE `.fab-runtime.yaml` `_agents` agent active/idle state that the `fab pane` commands read. That producer subsystem — the hook writes, the throttled GC sweep, the grandparent PID walker, the runtime file, and the `_agents` matching — was deleted wholesale. fab is now a pure CONSUMER of the `@rk_agent_state` tmux pane-option convention written by run-kit's `rk agent-setup` (see `fab pane` below).

**Artifact bookkeeping is no longer a hook.** The former `artifact-write` PostToolUse handler that recomputed `change_type`/confidence (from `intake.md`) and the `plan.*` counts (from `plan.md`) is gone — that state is correctness-critical (a hook fires only in the Claude harness, so a sed edit or a non-Claude agent left it stale). It is now recomputed by the pull-based **`fab status refresh`** (see the `fab status` family above), self-healed at the transition seams (`fab status advance`/`finish`, `fab preflight`), which preserves the `change_type_source: explicit` guard. The plan counters remain a **write-time cache**: readers (`fab preflight`, `fab pr-meta`, `fab status plan`) prefer a **live count derived from `plan.md` `## Acceptance` checkboxes at read time** and fall back to the cached counter only when `plan.md` (or its `## Acceptance` section) is absent — so a hook-bypassing edit cannot make those readers report a stale acceptance count.

---

## fab pane

Tmux pane operations with fab context enrichment. `fab pane <map|capture|send|process|window-name> [flags...]`

**Pane-family exit codes** (capture, send, window-name): pane validation failures use a shared scheme so callers can branch on cause — `2` = pane missing, `3` = any other tmux failure (dead server, bad socket). `map` and `process` use plain `ERROR:`-formatted exit 1. (Non-tmux usage errors — bad flag values, cobra arg-count — exit 1 per command; see the per-verb rows.)

**Persistent flag** (all subcommands): `--server <name>` / `-L <name>` (default `""`) — target tmux socket (`tmux -L <name>`). Defaults to `$TMUX` / tmux default. Lets daemons on one tmux server inspect panes on another.

**§ agent state (`@rk_agent_state` convention — read-only, ioku).** `map`/`capture`/`send` resolve a pane's agent lifecycle state by READING the tmux pane user option `@rk_agent_state` (value `"<state>:<epoch_seconds>"`, `state ∈ active | waiting | idle`), written by run-kit's `rk agent-setup` global agent-harness hooks (covering Claude Code, Codex, Copilot, Gemini, OpenCode — not just Claude). fab is a pure CONSUMER: it never writes the option and needs no run-kit software installed — it reads with plain tmux (`map` via the `#{@rk_agent_state}` field on its existing `list-panes -F` call; `send`/`capture` via `tmux show-options -pv -t <pane> @rk_agent_state`). `active` = turn in progress, `waiting` = blocked on a human (permission prompt / menu / elicitation), `idle` = turn complete. The epoch suffix is mandatory — idle duration is `now - epoch`; only `idle` carries a duration. An absent option, unknown token, or missing/non-integer epoch is **unknown** (`—` in tables, `null` in JSON, refused by `send` without `--force`). No staleness heuristic: a stale `active` (e.g. an Esc-interrupted agent) still refuses sends — `--force` is the escape hatch. (fab no longer produces this state — the old `.fab-runtime.yaml` `_agents` hook pipeline was divested; see § fab hook.)

### map — `fab pane map [--json] [--session <name>] [--all-sessions] [--server <name>]`

All tmux panes with pipeline state. Non-git/non-fab panes included with `---` fallbacks.

| Flag | Description |
|------|-------------|
| `--json` | JSON array (snake_case: `session`, `window_index`, `window_id`, `pane`, `tab`, `worktree`, `repo`, `change`, `stage`, `display_state`, `agent_state`, `agent_idle_duration`, `pr_url`, `pr_number`). `window_id` (`string\|null`) is the tmux `@N` window ID (server-assigned, stable for the window's lifetime and travels with it across `swap-window`/`move-window` — so consumers join on stable identity instead of the positional `window_index`), `null` when unavailable — `--json` only, no table column. `agent_state` (`string\|null`) is `active` / `waiting` / `idle` from the pane's `@rk_agent_state` option (see § agent state above), `null` when unset/unparseable; `agent_idle_duration` (`string\|null`) is populated only for `idle` (`null` for `active`/`waiting`/unset). `repo` is the absolute main-worktree root for the pane's repo (`null` when unresolved) — `--json` only, no human-table column. `display_state` (`string\|null`) is the state half of the display-stage derivation (the `stage` field is the name half): `active`, `ready`, `done`, `failed`, `pending`, or `skipped`; `null` whenever `stage` is `null` (no resolvable change / unloadable `.status.yaml`) — `--json` only, no table column. Distinguishes an actively-worked stage from a parked finished change (e.g. a fully-shipped change is `stage: "review-pr"` + `display_state: "done"`, while one whose review-pr is running is `"active"`). `pr_url` (`string\|null`) is the last entry of the change's `.status.yaml` `prs:` list (most recent), `null` when the list is absent/empty or the pane has no resolvable change; `pr_number` (`number\|null`) is parsed from the URL's trailing `/pull/<n>` segment, `null` when there is no URL or it is unparseable. Both are `--json` only (no table column), sourced from the already-loaded status file — **no `gh`/`git`, no network, no PR status (open/merged/CI)**; consumers fetch live PR state themselves. |
| `--session <name>` | Target specific session (skips `$TMUX` check) |
| `--all-sessions` | Query all sessions (skips `$TMUX` check; mutually exclusive with `--session`) |

Without `--session`/`--all-sessions` → current session only (`-s` scope, requires `$TMUX`). Table columns: `Session` (only with `--all-sessions`), `Pane`, `WinIdx`, `Tab`, `Worktree` (relative; `(main)` for main; `basename/` non-git), `Change`, `Stage`, `Agent`. The `Worktree` relative path is computed **per repo** — each pane's display path is relative to its own repo's main-worktree root (cached by git worktree root), so panes from multiple repos render correct paths. Agent: `active`, `waiting`, `idle ({dur})`, or `—` (em dash for unknown). Change: folder name, `(no change)` for fab worktree with no active change, or `—` for non-fab panes. Idle duration: `{N}s`/`{N}m`/`{N}h` floor division (idle only). Change and Agent resolve on independent axes: Change comes from `.fab-status.yaml`; Agent comes from the pane's `@rk_agent_state` option (read from the SAME `list-panes` call via the `#{@rk_agent_state}` format field — zero extra subprocesses, and server disambiguation evaporates since a pane option lives on exactly one server's pane; see § agent state above) — so a pane running any instrumented agent in discussion mode (no active change) shows `(no change)` in Change but a populated Agent column. `$TMUX` unset without targeting flag → exit 1 (`ERROR: not inside a tmux session`). No panes → exit 0 `No tmux panes found.`

### capture — `fab pane capture <pane> [-l N] [--json] [--raw] [--server <name>]`

`<pane>` required (e.g., `%5`). `-l/--lines N` (default 50). `--json` = content + metadata (`worktree`/`change`/`stage`/`agent_state`/`agent_idle_duration` — `agent_state` ∈ `active`/`waiting`/`idle`/`null`, read from the pane's `@rk_agent_state` option; see § agent state above). `--raw` = plain `tmux capture-pane -p`, no enrichment. `--json`/`--raw` mutually exclusive. Pane not found → exit 2 (`Error: pane <id> not found`); other tmux validation failure → exit 3. `--lines < 1` → exit 1 (`ERROR: --lines must be >= 1`).

### send — `fab pane send <pane> <text> [--no-enter] [--force] [--server <name>]`

Validation pipeline: (1) pane exists via a single targeted probe — `tmux display-message -t <pane> -p '#{pane_id}'`, output must equal the argument exactly (ID-exact: window names / target-grammar args resolve to a different pane ID and are rejected; no server-wide enumeration) — pane missing → exit 2 (`Error: pane <id> not found`), other tmux failure → exit 3; (2) three-state agent gate (unless `--force`): read the pane's `@rk_agent_state` option — `idle` → send; `active`/`waiting` → refuse with `ERROR: agent in pane <id> is not idle (state: <state>)` (exit 1, three-state aware); unset/unparseable → refuse with a **distinct** unknown-state message naming `--force` (exit 1); (3) `tmux send-keys`. `--no-enter` skips the trailing Enter. `--force` bypasses the state check only — pane-existence still enforced (a missing pane still exits 2 even with `--force`). Agent resolution reads `@rk_agent_state` via `tmux show-options -pv -t <pane> @rk_agent_state` (see § agent state above); a pane with no option = unknown (refused without `--force`). Change state is independent — panes in discussion mode (no active change) accept sends when idle. Success: `Sent to <pane>`.

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

## fab dispatch

Headless, tmux-independent process manager for CLI-dispatched pipeline stages — the **CLI adapter** for cross-harness stage dispatch (a stage running headless on a different agent CLI). Parallel to and independent of `fab pane` / `fab operator` (which stay the interactive path). `fab dispatch <start|status|logs|kill|clean> [args...]`. **POSIX-only (v1)** — `start` errors clearly on Windows (`fab dispatch requires a POSIX shell (setsid/timeout); Windows is not supported in v1`) rather than half-working. Full cross-adapter contract: `docs/specs/harness-adapters.md`.

**State layout** — `.fab-dispatch/{4-char-change-id}/` at the **repo root** (alongside `.fab-status.yaml`, already gitignored via the scaffold `.fab-*` pattern — no gitignore/scaffold/migration work). Keyed by the stable 4-char change ID (stable across `fab change rename`); one dir per worktree. Per-stage files:

| File | Written by | Contents |
|------|-----------|----------|
| `{stage}-prompt.md` | `start` (from stdin) | the stage prompt piped to the dispatched command's stdin |
| `{stage}.yaml` | `start` (via `internal/atomicfile`) | `pid`, `pgid`, `spawn_cmd` (resolved), `started_at`, `timeout` (secs, omitted when unset) |
| `{stage}.log` | the wrapper | combined stdout+stderr of the dispatched command |
| `{stage}.exit` | the wrapper | the exit code (`echo $? > ...`) — its presence is the "process finished" signal |
| `{stage}-result.yaml` | the dispatched agent (contract) | the stage result; presence is required for `done` (see states) |

### start — `fab dispatch start <change> <stage> [--timeout <secs>]`

Resolves `<change>` → 4-char ID; reads the stage prompt on **stdin** → `{stage}-prompt.md`; resolves the stage's tier → provider → `dispatch_command` internally (via `internal/agent` + `internal/spawn` `{model}`/`{effort}` substitution — the same resolution `fab resolve-agent` performs); launches it **DETACHED**, cwd = repo root:

```sh
sh -c '<resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'
```

The shell is launched with `setsid` semantics (Go's `SysProcAttr{Setsid:true}`, not a `setsid` binary prefix — prefixing it would double-fork and leave the recorded pid pointing at a process that exits immediately), detaching it into a new session/process group so the dispatch **survives the orchestrator dying** — no Go supervisor remains, the shell records the exit code itself and the recorded `pid`/`pgid` track the live worker. `start` writes `{stage}.yaml` before returning. `--timeout N` wraps the resolved command in POSIX `timeout N <cmd>` **inside the wrapper** (no Go timer/daemon); a timed-out command exits `124`, surfacing as `failed`.

- **No `dispatch_command` → error, no fallback**: if the resolved tier's provider has no `dispatch_command`, `start` errors (`stage <stage> resolves to tier <tier> (provider <name>), which has no dispatch_command; configure providers.<name>.dispatch_command to dispatch this stage`) and does NOT fall back to that provider's `session_command`.
- **Concurrency = refuse-if-running + last-attempt-only**: refuses if a dispatch for the exact `(change, stage)` pair is already `running` (`a dispatch for <change>/<stage> is already running (pid N); run fab dispatch kill first`). A `start` over a **completed** prior attempt (done / failed / orphaned) **overwrites** its files — no per-attempt history. Different stages of the same change share `.fab-dispatch/{id}/` via distinct `{stage}.*` filenames and do not collide.

### status — `fab dispatch status <change> <stage> [--json]`

Byte-stable poll surface. Reads `{stage}.yaml` / `{stage}.exit`, probes `pid` liveness (POSIX `kill(pid,0)`), and reports exactly one of five states:

| State | Condition |
|-------|-----------|
| `running` | pid alive AND `{stage}.exit` absent |
| `done` | `{stage}.exit` == `0` AND `{stage}-result.yaml` present |
| `failed` | `{stage}.exit` present AND != `0` (includes `124` timeout) |
| `failed (no-result)` | `{stage}.exit` == `0` BUT `{stage}-result.yaml` absent — a **contract violation, NOT done** |
| `orphaned` | pid dead AND `{stage}.exit` absent (reboot / `kill -9` / crash) |

A clean exit (code 0) is necessary but **not sufficient** for `done` — the result file must exist. Human output is the bare state string on stdout; `--json` emits `{change, stage, state, pid, pgid, exit?}`.

### logs — `fab dispatch logs <change> <stage> [--tail N]`

Prints `.fab-dispatch/{id}/{stage}.log`. `--tail N` prints the last N lines (Go-side, no external `tail`). Missing log → `no dispatch log for <change>/<stage>`.

### kill — `fab dispatch kill <change> <stage>`

Sends `SIGTERM` to the **process group** (`pgid` from `{stage}.yaml`) so the detached command and its children die together. Idempotent: killing an already-dead dispatch is a benign no-op with a clear report (`dispatch <change>/<stage> already dead (pid N); nothing to kill`).

### clean — `fab dispatch clean [<change>] [--orphans]`

Manual cleanup — one of exactly **two** cleanup paths (the other is archive-time deletion; there is **no automatic GC** anywhere):

- `fab dispatch clean <change>` — removes `.fab-dispatch/{id}/` for the named change.
- `fab dispatch clean` (no arg) — removes all `.fab-dispatch/*/` dirs.
- `fab dispatch clean --orphans` — prunes any `.fab-dispatch/{id}/` whose ID no longer resolves to a non-archived change (covers a change archived/deleted upstream leaving a local state dir orphaned).

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

Migration discovery. Lives in `fab-kit` (registered in the router's `fabKitArgs` allowlist). Resolves `fab/.kit-migration-version` (local) and the engine `VERSION` from the cached kit for the pinned version (`fab/.fab-version`, with the legacy config.yaml `fab_version:` fallback), scans the engine `migrations/` dir, and runs the discovery algorithm. Consumed by both `/fab-setup migrations` (via `--json`) and as a standalone query.

```
fab migrations-status [--json]
```

**Human output**: `Local version` / `Engine version`, then either `No migrations apply.` or `Migrations to apply (N):` with an ordered `[i/N] FROM -> TO (file)` list, followed by any gap-skip lines and any overlap warning.

**`--json` output**: `{local, engine, applicable:[{from,to,file}], gap_skips, overlaps}` — `applicable` is the ordered chain to apply (FROM ascending), `gap_skips` are skip log lines, `overlaps` are conflicting filename pairs (non-empty = malformed migration set).

**Exit code**: `0` on any clean query — including the no-op case AND the overlap case (overlap is surfaced via the `overlaps` field). Non-zero only on a genuine error (missing `fab/.kit-migration-version`, missing engine `VERSION`, unreadable migrations dir). **Distinct exit `3` for the unmanaged-repo precondition**: run outside a fab-managed repo, `migrations-status` prints `not in a fab-managed repo. Run 'fab init' to set one up` to stderr and exits `3` — the same "not applicable here" signal `sync` uses, shared via `internal.RequireManagedRepo()` (the `internal.ExitNotManaged` constant), distinct from the generic exit `1` = failure above. This is the same branchable-code contract documented in the `sync` row of § Workspace Command Exit Semantics; contrast `fab upgrade-repo`, which is unaffected and still exits `1` in that scenario. Read-only — never writes `fab/.kit-migration-version`.

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

Consumers: `fab pr-meta` (which renders the PR body `**Impact**` line via the same `internal/impact` package) and the apply-finish, hydrate-finish, and ship-finish hooks (write the result into `.status.yaml` `true_impact`; ship-finish is the authoritative write in the standard pipeline — the earlier writes see `HEAD == merge-base` until commits exist). `/git-pr` no longer calls `fab impact` directly — it delegates the whole `## Meta` block to `fab pr-meta`.

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
- Impact: a single normalized markdown table whose first-column header is `Impact` (so the table self-labels — there is no `**Impact**:` lead-in line), columns `Impact | +/− | Net` (numeric columns right-aligned, Net kept on every row), followed by a `<sub>` provenance caption. The locked taxonomy is `raw / true / impl / tests / excluded` with `raw = true + excluded` and `true = impl + tests`; `true` is ALWAYS the post-exclude diff (the fix for the prior "total flips meaning" bug). The table adapts by DROPPING rows, never reshaping: the `raw` row is shown whenever excludes are configured (the `Excluding` pass is present), even when its figures equal `true` — with NO excludes configured, `true` is definitionally identical to `raw`, so no redundant duplicate row is rendered; the `**true**` row is always present and bold, and the nested `└ impl` / `└ tests` rows appear only when a `tests` pair exists (`impl` is the per-component `max(0, true − tests)` residual, Unicode minus `−`, clamp-annotated when net-negative). The caption co-locates the excludes note and the version stamp — `<sub>excludes `…` · generated by fab-kit vX.Y.Z</sub>` — with the excludes list built from the actual `true_impact_exclude` values each backtick-wrapped (the `excludes …` clause omitted when none are configured) and the binary version stamped from the running `fab` (`main.version`, threaded via `prmeta.Data.Version`; a dev build renders `fab-kit vdev` honestly). The whole block is omitted entirely on `+0/−0` `true`, missing merge-base, or impact failure. Only **bold** is used for emphasis (`<sub>` is on GitHub's HTML allowlist; the sanitizer strips row backgrounds and text color).
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
stops the per-row / per-changelog-row merge conflicts on the hot `description` cells. The index
is a pure function of content (no git dates), so it is branch-independent and idempotent. It
produces the generated half of the **FKF** format (Fab Knowledge Format — see
`$(fab kit-path)/reference/fkf.md`): per-folder `log.md`, the `type: memory` round-trip mechanism, and the
root-index `fkf_version` frontmatter.

What it writes:
- **Root `docs/memory/index.md`** — **domains-only** (`| Domain | Description |`), prefixed with
  the FKF `fkf_version: "0.1"` frontmatter block (the **only** `index.md` permitted frontmatter
  beyond the generator's own output — FKF §8; no domain/sub-domain index carries it). The legacy
  inlined per-file "Memory Files" column is dropped (it silently drifts). Each domain row's
  Description is read from that domain `index.md`'s `description:` frontmatter.
- **Every `docs/memory/{domain}/index.md`** — file rows (`| File | Description |`)
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
- The **`log.md` history** comes from ONE batched
  `git log --date=short --name-status -- docs/memory` pass (newest-first): the log takes the
  full per-path commit list (date + subject + name-status) — no per-file `git log` spawns. The
  **index** consumes none of this — it carries no dates (a pure function of content), so the
  batched pass now serves `log.md` only. When the whole batched pass fails, **no
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

Frontmatter warnings (stderr — the malformed-detection + description-length work, 260715-xu0k). All
are gathered in the same pass as the shape warnings and printed to stderr on **both** the write and
`--check` paths; **none change the byte-stable rendered index output** (a malformed value keeps
rendering exactly as it does now — validation is stderr/exit-code only):
- `✖ docs/memory/<domain>/<file>.md has malformed frontmatter — unclosed frontmatter block (no closing \`---\`)`
  when a file opens with `---` (line 1) but has no subsequent standalone `---`. The loom glued-fence
  corruption (`description: "…"---` on one line) is an instance — gluing the fence onto the value
  removes the closing fence entirely.
- `✖ docs/memory/<domain>/<file>.md has malformed frontmatter — \`description:\` value fails quote-stripping (unterminated quote): <value>`
  when the extracted `description:` value begins with a quote (`"`/`'`) but does not end with the
  matching quote (the specific glued-fence diagnostic, e.g. a value ending in `"---`).
- `⚠ docs/memory/<domain>/<file>.md has a <N>-character \`description:\` (soft cap: 500) — trim to a one-liner; detail belongs in the file body`
  when a `description:` value exceeds **500 characters** (measured in runes on the quote-stripped
  value; hardcoded package const `DescriptionLenWarnThreshold`, NOT config-overridable). This one is
  **advisory only** (see the `--check` asymmetry below).
- The two `✖` **malformed** warnings are a **blocking** class (they fail `--check` — see Exit codes);
  the `⚠` length warning is **advisory** (it never fails `--check`). Both the index-row `description:`
  frontmatter on topic files AND the domain/sub-domain `index.md` stub descriptions are validated.

Flags:
- `--check` — write nothing; classify the rendered-vs-existing **index drift** (across every index
  **and `log.md`** target) by **severity** and encode it in the **exit code** (see Exit codes).
  Useful as a staleness guard (CI / preflight) AND as a destructive-loss guard (refuse-before-regen).
  The drift detection is the same byte-compare the write path uses; the destructive-loss half is a
  classifier + a small parser over the *existing* index rows/headings (pure functions in
  `internal/memoryindex`, unit-tested like `RenderRoot`/`Gather`) — and is skipped for `log.md`
  targets (always benign drift). **Malformed frontmatter is a separate blocking signal** — it floors
  the `--check` exit at 1 independent of index drift (see Exit codes); the advisory over-length
  `description:` warning never affects the exit code.
- `--json` (with `--check`) — emit the loss report as a single JSON object on **stdout** and
  suppress the human-readable text; the exit code is unchanged. Mirrors the `fab pane` /
  `fab migrations-status` `--json` convention (snake_case). Shape:
  `{"tier": 0|1|2, "drift": bool, "losses": [{"category": "description"|"tombstone"|"grouping", "path": "<repo-rel index>", "detail": "<lost text | dropped link target | flattened heading>"}], "malformed": [{"kind": "malformed-fence"|"malformed-description", "path": "<repo-rel file>", "detail": "<offending value, omitted for fence>"}]}`.
  The `malformed` array is **additive** (260715-xu0k): the `tier`/`drift`/`losses` keys are unchanged,
  so `/docs-reorg-memory`'s compatibility detection (which branches on `tier` / reads `losses`) is
  unaffected. `losses` and `malformed` are always present (empty arrays, never `null`).
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
  `description:`, a stale `log.md`, a `log.md` gaining merged
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

**Malformed frontmatter — a distinct BLOCKING signal, not a drift tier (260715-xu0k).** The two `✖`
malformed warnings above (unclosed fence, quote-strip failure) are **source-file corruption**,
orthogonal to the index-drift tier. They **floor the `--check` exit at 1 even when index drift is
clean (tier 0)** — the loom case is provably tier 0 (the committed garbage row is byte-identical to
what regeneration produces from the corrupted source), so a pure drift comparison exits 0 and would
never catch it; the malformed check runs independent of drift. When malformed frontmatter co-occurs
with a tier-2 destructive loss, **exit 2 still wins** (the malformed files are enumerated either way).
Malformed frontmatter is **NOT a tier-2 category** and does **NOT** extend the `losses[]` category
enum: it is fixed by repairing the file's frontmatter (restore the closing `---` / matching quotes),
not by `/docs-reorg-memory`, so it carries its own fix-the-file remediation and does **not** fire the
hydrate/reorg refuse-before-regen guards (which key on exit == 2). The over-length `description:`
warning is **advisory only** — it never affects the exit code (corruption blocks, over-length nags).

Callers pick a threshold: **CI / pre-commit** fails on exit ≥ 1 (any drift **or** malformed
frontmatter); the **hydrate / reorg refuse-before-regen guards** fail only on exit == 2 (destructive
loss — malformed frontmatter does not reach them). A **born-FKF / born-compatible fab-kit
tree is provably never exit 2** (frontmatter present, no off-disk rows, domains-only root, native
`log.md` exactly what the generator produces) — so the refuse-before-regen guards are no-ops on
native trees and only ever fire on a pre-fab-kit tree. (A born-FKF tree that *later* has its
frontmatter mangled by a bad edit exits 1 on the malformed floor — the corruption this change exists
to block — never exit 2.)

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

Singleton tmux-tab launcher for `/fab-operator`. Requires `$TMUX` (else exit 1, `ERROR: not inside a tmux session`). The singleton check is an **exact, server-wide** window-name match: `tmux list-windows -a` enumerated and compared exactly (never tmux target resolution, whose prefix/glob fallback would let e.g. `operator-logs` mask the real check; `-a` enforces the one-operator-per-SERVER invariant across sessions). If a window named exactly `operator` exists anywhere on the server → select it by window ID, switching the client to its session when needed (`Switched to existing operator tab.`); else create the window running `{operator-session-command} '/fab-operator'` (`Launched operator.`).

**Launch cwd (no git-repo dependency)**: the new window's working directory (`tmux new-window -c <dir>`) is resolved by trying `git rev-parse --show-toplevel` first and falling back to `os.Getwd()` when that fails — so the operator launches **inside a git repo** (cwd = repo root, today's behavior) **or from a neutral parent directory** (cwd = current directory). It no longer hard-fails with `cannot determine repo root`; it errors only if both git-root resolution AND `os.Getwd()` fail. This matches the per-tmux-server, cross-repo singleton model: the operator's natural launch point is a neutral dir with no `fab/` project.

**Session command resolution (no `fab/`-project dependency) + operator-tier profile**: the operator resolves the **operator tier** in-process (`agent.ResolveTier(cfg, "operator")`) → its provider → that provider's `session_command`, then injects the tier's `{model, effort}` via `spawn.WithProfile`. When a `fab/` project is resolvable (`resolve.FabRoot()` succeeds) the config supplies the operator tier + provider; when `resolve.FabRoot()` **fails** — the operator is launched from a neutral directory with no `fab/` project anywhere up the tree (its natural cross-repo home) — this is **non-fatal**: `config.Load` returns an empty config, so `ResolveTier`/`ResolveProvider` degrade to fab-kit's built-in operator tier (`claude-sonnet-5`/`medium`) + built-in claude provider (`spawn.DefaultSpawnCommand`). `WithProfile` is grammar-forgiving: for a **template** `session_command` containing `{model}`/`{effort}` — including the built-in claude default, which is templated — it **substitutes** the resolved values in place (all-or-nothing; an empty value drops the placeholder's token and a preceding `-`-flag); for a command carrying **no placeholder** (e.g. a user's plain-form config carried forward by the 2.13.0 migration) it instead **appends** `--model <model> --effort <effort>` to the END (last-wins; each flag omitted when its value is empty, per the `empty ⇒ omit` convention). A provider without a `session_command` falls back to `spawn.DefaultSpawnCommand` (the templated claude default, still profile-substituted). So a `fab/`-less launch composes a fully-defaulted command: default session command + operator default `{model, effort}` (byte-identical whether resolved by substitution or, for a plain user command, by append).

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

## fab agent

```
fab agent [tier] [--print] [--repo <path>]
```

Launch (or `--print`) the resolved agent **session** command in the current shell. Replaces `fab spawn-command`, with a semantic upgrade: the printed/exec'd command is **profile-resolved** (model/effort substituted), not placeholder-stripped.

- Resolves the tier profile (`default` when the positional `[tier]` is omitted; any of the five role-tier names accepted: `default`, `operator`, `doing`, `review`, `fast`), then composes `providers.<profile.provider>.session_command` with `{model}`/`{effort}` substituted (or Claude-style `--model`/`--effort` appended for a non-templated command) via `internal/spawn.WithProfile` — the same substitution `fab resolve-agent`'s `dispatch=` line and the operator launcher use.
- **Default (exec)**: replaces this process with the composed command via `sh -c` (so shell expansions like `$(basename "$(pwd)")` expand at invocation). `fab agent` starts the default-tier agent right here; `fab agent operator` starts the coordinator profile. **No TTY guard** — exec-and-let-the-agent-CLI-handle-it (document-don't-validate).
- **`--print`**: prints the fully-resolved command instead of executing (the `fab spawn-command` replacement — profile-resolved, not stripped). Lets the operator compose a worker spawn from a real profile.
- **`--repo <path>`**: reads `<path>/fab/project/config.yaml` instead of the current repo (the operator's fetch-another-repo's-command use case, carried over from `fab spawn-command --repo`).
- **Error**: a resolved provider with no `session_command` (and not the built-in claude) errors with a config-key hint (`configure providers.<name>.session_command`); an unknown tier name errors and names it.

*(`fab spawn-command` is removed in this release with no deprecation alias — its only CLI consumer was the operator skill, updated in the same kit. `fab batch` and the operator launcher use the internal `spawn` package, not this CLI command.)*

---

## fab batch

Multi-target operations: `fab batch <new|switch|archive> [flags] [targets...]`. The `new` and `switch` subcommands take `[--list] [--all]`, create tmux windows, and require `$TMUX`; `archive` runs in-process (no `$TMUX`) and has its own flag surface (`[--yes|-y] [--dry-run]` — see below), having diverged from `--list`/`--all` for safety.

- **`new`** — parse `fab/backlog.md` pending items (`- [ ] [xxxx]`), create worktrees, open tmux windows, start agents with `/fab-new {description}`. No args → `--list`. IDs → one worktree tab each (`wt create --non-interactive --worktree-name {id}`, window `fab-{id}`, `{worker-session-command} '/fab-new {description}'`). `--all` → all pending. Handles continuation lines. Launch failures are surfaced per item: a failed `wt create` or `tmux new-window` prints `[{id}] FAILED: ...` (the tmux line names the already-created worktree path as the cleanup/recovery hint) with the child's stderr included, never aborts the remaining items, and the command exits non-zero when any item failed (`ERROR: {N} of {M} item(s) failed to launch`). Unknown/empty backlog IDs remain warn-and-skip (exit 0). Requires `$TMUX` (else exit 1, `ERROR: not inside a tmux session`); empty pending backlog with `--all` → exit 1, `ERROR: No pending backlog items found.`. **Profile injection**: the worker spawn command is composed from the **default tier's** provider `session_command` with the default tier's `{model}`/`{effort}` **substituted** (a templated command) or **appended** (a non-templated command) via `internal/spawn` — workers finally spawn WITH a profile; substitution resolves every placeholder so no literal `{model}`/`{effort}` braces reach tmux.
- **`switch`** — resolve change names (in-process via `resolve.ToFolder`, like the rest of the family — no `fab`-on-PATH dependency; an unresolvable name warns with the resolver's specific error, e.g. `Multiple changes match…`, and skips), create worktrees with branch names (applying `branch_prefix` from config), start agents with `/fab-switch {change}`. No args → `--list`. `--all` → all active changes (excludes `archive/`); empty set → exit 1, `ERROR: No changes found.`. Branch naming: `{branch_prefix}{folder_name}`. Requires `$TMUX` (else exit 1, `ERROR: not inside a tmux session`). **Profile injection**: same as `new` — the worker spawn command is composed from the default tier's provider `session_command` with the default tier's `{model}`/`{effort}` substituted/appended via `internal/spawn`, so a profile rides the worker and no literal braces reach tmux.
- **`archive`** — find changes with `hydrate: done|skipped`, then archive each mechanically in a Go loop via `internal/archive.ArchiveWithBacklog` (move, index, backlog mark-done, pointer). No agent or Claude session is spawned; resolution uses `resolve.ToFolder` (no `fab`-on-PATH dependency). **Flag surface (diverges from new/switch):** archive is the one bulk-mutating member whose moves are effectively irreversible within the loop, so instead of staying list-by-default behind `--all` it uses a list-then-confirm model with a `--yes` escape hatch (apt/npm/gh-style):
  - **bare invocation (interactive stdin)** → lists the archivable set, then prompts `Archive these N? [y/N]` with **default No** — a bare Enter or any non-`y`/`yes` (case-insensitive) answer aborts (exit 0, nothing archived); `y`/`yes` archives all.
  - **`--yes` / `-y`** → archives all archivable changes with no prompt (the non-interactive escape hatch; resolved behavior of the former `--all`).
  - **`--dry-run`** → lists what would be archived; no prompt, no action (the former `--list`).
  - **non-TTY stdin without `--yes`** → refuses rather than hangs: returns a single multi-line error so `main()`'s centralized printer emits it once as `ERROR: refusing to prompt for confirmation on a non-interactive stdin.` followed by `Re-run with --yes to archive non-interactively` on stderr, then exits non-zero (the handler does not print its own `ERROR:` lines, avoiding a doubled prefix). This matters because the tmux/operator runtime is frequently non-interactive — those call sites pass `--yes`.
  - **explicit args** (`fab batch archive foo bar`) → archive the named changes with **no prompt and no TTY guard** (naming them IS the opt-in; the prompt applies only to the bare/archive-all path).
  - **`--dry-run --yes`** → mutually exclusive → exits non-zero (`ERROR: --dry-run and --yes are mutually exclusive`).

  Per change prints `{name} — archived` (with ` (backlog marked done)` when applicable; when a post-archive step — index update or backlog mark — fails, the change still prints `archived` plus a stderr `warning:` line and counts as archived, not failed), `already archived, skipping` (covers genuinely-archived names — counted as skipped), or `FAILED: {err}`; a single failure never aborts the batch. Footer: `Archived {N}, skipped {M}, failed {K}.`. Exit semantics: an empty archivable set (bare or `--yes`) is a benign no-op (`No archivable changes found.` + zero footer, exit 0) checked **before** any prompt or non-TTY guard (finding F49); after the loop runs, non-zero when `failed > 0` (`ERROR: {K} change(s) failed to archive`); explicitly named targets where none resolves to an active *or* archived change → exit 1, `ERROR: No valid changes to archive.`.

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
