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
- fab config (show / explain / set / unset / init / upgrade)
- fab pane
- fab doctor
- fab migrations-status
- fab kit-path
- fab shell-init
- fab skill
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

`fab <command> <subcommand> [args...]`. `fab` is a router dispatching workspace commands (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`) to `fab-kit` and everything else to the per-version `fab-go` binary resolved from the pinned version in `fab/.fab-version` (a one-line plain-text sibling of `fab/.kit-migration-version`; the sole version source — a stale `fab_version:` key left in `fab/project/config.yaml` is no longer read). `--version`/`-v`/`--help`/`-h`/`help` are handled inline. `fab-go` auto-fetches from GitHub releases on cache miss.

`fab -h` composes help from both binaries. `fab --version` prints the system binary version; inside a fab repo a second line shows the project-pinned version.

### Exit-Code Convention (`fab-go` commands)

The `fab-go` binary (everything the router does not dispatch to `fab-kit`) follows the toolkit convention: **`0` success / `1` operational failure / `2` usage error**. A **usage error** is a malformed invocation caught at parse/validation time — an unknown/malformed flag (`fab score --nope`), an arg-count violation (`fab score` with no args), an unknown subcommand (`fab nonsense`), or a mutually-exclusive flags-group conflict (`fab resolve --status --folder`). An **operational failure** is a syntactically valid invocation that fails on a runtime/data condition (a missing change, a failed preflight, a below-gate `--check-gate`, a tmux/gh/filesystem error). Success is `0`.

Classification follows execution phase, never message text: cobra failures before `RunE` exit `2`; errors from inside `RunE` exit `1`. The testable `run()` helper records whether execution began, and no path inspects stderr to classify.

**Coexistence with in-handler domain schemes (no renumbering)**: the pane family (`2` = pane missing, `3` = other tmux failure) and `fab memory-index --check` (`0`/`1`/`2`, `2` = destructive loss) set their non-1 codes via `os.Exit` *inside* the handler, which bypasses `main()`'s usage/operational mapping entirely — their codes are unchanged. For those subcommands exit `2` is therefore intentionally ambiguous between "usage error" (at parse time) and the domain meaning (in-handler); this is documented per subcommand, which principle №4 sanctions, rather than renumbered (renumbering would break the pinned pane test and downstream consumers). A usage error is a static caller bug fixable at authoring time, not a runtime condition scripts branch on, and stderr wording disambiguates.

### Workspace Command Exit Semantics

Lifecycle commands fail loudly — a non-zero exit is the failure signal scripts and skills rely on. **Exception**: `sync` and `fab-kit migrations-status` reserve a distinct exit `3` for the "not a fab-managed repo" precondition (see the `sync` row and the `fab migrations-status` section) — that is *not* a failure but a "not applicable here" signal, so a caller branching on the exit code MUST treat exit `3` separately from the generic exit `1` = failure. All other lifecycle failures use the generic non-zero (exit `1`) path.

| Command | Failure behavior |
|---------|------------------|
| `init` | Refuses to run while `FAB_KIT_PATH` is non-empty, with an unset-first error before repository checks, downloads, config writes, or version stamps. Otherwise requires a git repository — exits non-zero with `fab init requires a git repository — run 'git init' first` BEFORE any download or config write. Sync failure during init also exits non-zero |
| `update` | Exits `0` with a degrade message (`fab-kit was not installed via Homebrew` + manual-update guidance) when the binary is not brew-installed (go-install/manual/CI) — not brew's to upgrade, so not a failure (shll update standard: exit non-zero only on genuine failure; keeps a composed `shll update` run honest). Exits non-zero only on genuine brew failure (`brew update`/`brew info`/`brew upgrade`); brew runs unbounded — no timeout, no kill path (the standard forbids SIGKILLing a package manager mid-transaction). Internally `Update()` still returns the `ErrNotBrewInstalled` sentinel — the exit-0 mapping is command-layer only, so `sync`'s version guard keeps treating a too-old non-brew binary as a guard failure |
| `upgrade-repo` | Refuses to run while `FAB_KIT_PATH` is non-empty, before resolution, cache, sync, config, or version-stamp work, because release-stamping arbitrary override content would mix kit provenance. Otherwise runs sync first, then (only AFTER sync succeeds) stamps `fab/.fab-version` and auto-runs `fab config upgrade` against the pinned fab-go to reconcile `config.yaml`'s reference fence (fail-open: a fab-go predating the subcommand prints a reminder and the upgrade continues). On sync failure: exits non-zero with `sync failed: ... — run 'fab sync' to repair, then re-run 'fab upgrade-repo'`, never prints `Updated: x -> y`, stamps nothing, and a re-run retries (no "Already on the latest version" short-circuit of the broken state). **Unaffected by the sync/migrations-status exit-`3` contract**: run outside a fab-managed repo, `upgrade-repo` still exits the generic `1` with the same `not in a fab-managed repo. Run 'fab init' to set one up` stderr message — it deliberately does NOT use `RequireManagedRepo()`/exit `3`, because its guard tolerates a repo with a `config.yaml` but no resolvable pinned version (a partially-managed, not fully-unmanaged, state), a distinct semantic left out of scope. Do not conflate the two: only `sync` and `migrations-status` carry the exit-`3` signal |
| `sync` | When `FAB_KIT_PATH=<dir>` is non-empty, resolves it to an absolute existing directory, prints `kit: <dir> (FAB_KIT_PATH override)`, serves scaffolding/skills from it, and skips both the version guard and `EnsureCached`; invalid overrides fail loudly without cache fallback, while prerequisites, deployment, direnv, project scripts, flags, and failure propagation remain unchanged. With no override, cache behavior is byte-identical. Exits non-zero when any skill deployment write fails (per-skill `WARN:` lines on stderr, `failed N` in the agent tally) or when scaffolding writes fail. The version guard exits non-zero whenever it trips: either `fab-kit was updated to vX — re-run 'fab sync'` (auto-update landed; the current run still ran old code) or actionable too-old instructions (non-brew install, Homebrew tap release lag) — it never continues on a binary older than the pinned version (`fab/.fab-version`, the sole version source). **Branchable exit code** (mirroring the pane family's use of distinct branchable exit codes — though with the opposite polarity: for the pane verbs exit `3` is the real failure and `2` the benign signal, whereas here `3` *is* the benign "not applicable" signal and no exit `2` is involved): run outside a fab-managed repo (no `fab/project/config.yaml` on any ancestor), `sync` prints `not in a fab-managed repo. Run 'fab init' to set one up` to stderr and exits `3` — a distinct "not applicable here" signal, NOT the generic `1` = failure — so callers (e.g. `wt`'s default init, operator scripts) can branch on "not a fab-managed repo" vs. "a real sync failure" without replicating fab's config walk-up. This holds unconditionally, including outside any git repository: the managed-repo check is a `config.yaml` walk-up gated before the git-root resolution, so `sync` is symmetric with `fab-kit migrations-status` (which has no git precondition). The value is the `internal.ExitNotManaged` constant, shared with `fab-kit migrations-status` (below) via `RequireManagedRepo()`. Genuine sync failures above stay exit `1` |

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

`fab preflight`, `fab score`, `fab log command`, `fab change`, `fab resolve`, `fab status` — headline coverage lives there. Sections below document the remaining commands (`fab pane`, `fab doctor`, `fab migrations-status`, `fab kit-path`, `fab shell-init`, `fab skill`, `fab impact`, `fab pr-meta`, `fab memory-index`, `fab fab-help`, `fab help-dump`, `fab operator`, `fab agent`, `fab batch`) and extended flag details for the above.

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
| `set-summary` / `get-summary` | `set-summary <change> <text>` / `get-summary <change> [--json]` | Per-change one-line log summary (`.status.yaml` `summary:` field — the FKF C-lite `log.md` source, §6.3). `set-summary` writes it (the conflict-free write path — each change touches only its own `.status.yaml`); `get-summary` prints it (empty line when absent — the generator then falls back to the change slug). `omitempty`: an empty summary round-trips to absent. No stage auto-populates it. `get-summary --json` → `{"summary":"…"}` (object-wrapped so fields can be added additively; empty summary → `{"summary":""}`) |
| `set-acceptance` | `set-acceptance <change> <field> <value>` | Updates `plan:` block. Valid fields: `generated` (bool), `task_count`, `acceptance_count`, `acceptance_completed` (int) |
| `set-checklist` | `set-checklist [args...]` | **Removed** — exits 1 with `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.` Use `set-acceptance` |
| `set-confidence` | `set-confidence <change> <counts...> <score> [--indicative]` | Basic confidence block. `--indicative` is an accepted-but-ignored no-op |
| `set-confidence-fuzzy` | `set-confidence-fuzzy <change> <counts...> <score> <dims...> [--indicative]` | With SRAD dimensions. `--indicative` is a deprecated no-op (see above) |
| `add-issue` / `get-issues` | `<change> <id>` / `<change> [--json]` | Issue ID array — idempotent / one per line. `get-issues --json` → `["DEV-988"]` (empty → `[]`, never `null`) |
| `add-pr` / `get-prs` | `<change> <url>` / `<change> [--json]` | PR URL array — idempotent / one per line. `get-prs --json` → `["https://…/pull/42"]` (empty → `[]`, never `null`) |
| `progress-line` | `progress-line <change>` | Single-line visual progress. *(No `--json` — visual decoration, not programmatic data.)* |
| `current-stage` | `current-stage <change> [--json]` | Detect active stage. `--json` → `{"stage":"apply"}` |
| `all-stages` | `all-stages [--json]` | List all stage IDs in order (no `<change>` argument). `--json` → `["intake","apply","review","hydrate","ship","review-pr"]` |
| `progress-map` | `progress-map <change> [--json]` | Extract `stage:state` pairs, one per line. `--json` → `[{"stage":"intake","state":"done"},…]` (an ordered array — a Go map would alphabetize and lose stage order) |
| `display-stage` | `display-stage <change> [--json]` | Display stage as `stage:state`. `--json` → `{"stage":"apply","state":"active"}` |
| `plan` | `plan <change> [--json]` | Extract `plan:` fields — `generated`, `task_count`, `acceptance_count`, `acceptance_completed` (one `key:value` per line). `--json` → `{"generated":true,"task_count":12,"acceptance_count":10,"acceptance_completed":3}` (same live-acceptance read path as the text output) |
| `confidence` | `confidence <change> [--json]` | Extract `confidence:` fields — `certain`, `confident`, `tentative`, `unresolved`, `score` (one `key:value` per line). `--json` → `{"certain":2,"confident":3,"tentative":1,"unresolved":0,"score":4.2}` |
| `validate-status-file` | `validate-status-file <change>` | Validate `.status.yaml` against the schema; non-zero exit on violation. *(No `--json` — its contract is the exit code; it emits no data.)* |

**`--json` on the read-only query surface**: the nine read-only query subcommands above (`confidence`, `plan`, `progress-map`, `get-issues`, `get-prs`, `get-summary`, `current-stage`, `display-stage`, `all-stages`) accept a `--json` flag (`Output as JSON`) emitting a stable object/array schema, following the `fab dispatch status --json` precedent (indented `json.NewEncoder`, snake_case keys matching the `.status.yaml` fields). Keys evolve additively (new fields optional) — there is no `schema_version` field. The default (no-flag) text output is byte-identical to before. The two non-data query subcommands are deliberately excluded: `progress-line` (visual decoration) and `validate-status-file` (exit-code contract only).

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
| Normal | `fab score <change>` | Parse `intake.md` (the sole scoring source; `--stage` defaults to `intake`), compute, write `.status.yaml`. No `indicative` key is written. Exits non-zero (error on stderr) when `.status.yaml` fails to load, the confidence write-back or `.history.jsonl` confidence-log append fails, or `intake.md` cannot be read — no silent partial success; the YAML report appears on stdout only when scoring *and* persistence succeed |
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

> `confidence.indicative` is not written. An `indicative: true` key on disk is tolerated on read and dropped on the next save.

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

The `status.yaml` template (in the kit cache at `$(fab kit-path)/templates/status.yaml`) includes the confidence block initialized to zero counts and score 0.0. `/fab-new`, `/fab-draft`, and `/fab-dedupe` write the intake score after intake generation (all three via the shared `_intake` Step 7; `/fab-dedupe` once per accepted cluster group); `/fab-clarify` re-writes it after resolving intake assumptions.

---

## fab preflight (extended)

`fab preflight [<change-name>]` — validates config.yaml, constitution.md, active change resolution, `.status.yaml` existence. Outputs YAML with `id`, `name`, `change_dir`, `stage`, `display_stage`, `display_state`, `progress`, `plan`, `confidence`. Non-zero exit on failure (error on stderr). Validation plus a **best-effort state refresh**: preflight is the **orient seam**, so before reading it runs the artifact-derived recompute under the status flock and persists `change_type`/`confidence`/plan counts when they are dirty. Every refresh failure is swallowed — preflight must still orient when the recompute cannot run. It moves **no stage pointers** and performs no transitions.

---

## fab log (extended)

Append-only JSON logging to `.history.jsonl`.

```
fab log command <cmd> [change] [args]
fab log confidence <change> <score> <delta> <trigger>
fab log review <change> <result> [rework]
fab log transition <change> <stage> <action> [from] [reason] [driver]
```

`command` is pure telemetry and **always exits 0** (given valid usage — cobra arg-count errors are usage errors that exit `2` before RunE) — it owns its best-effort contract. On any internal failure (no fab root, an explicit `[change]` that doesn't resolve, unwritable `.history.jsonl`) it prints a one-line `Warning: fab log command: …` to stderr and still exits 0, so call sites need no `2>/dev/null || true` guard and a telemetry hiccup can never become a pipeline failure mode. When `[change]` is omitted, the active change resolves from `.fab-status.yaml` (silent no-op if absent/dangling). `review`/`confidence`/`transition` keep fail-loud non-zero exits (they are auto-logged by `fab status`/`fab score` — skills never call them directly).

**Common callers** — skills per `_preamble.md` Context Loading §2 (`fab log command "<skill>" "<change>"`); `finish/fail review` auto-log; `score` auto-logs confidence; `change new`/`change rename` auto-log.

---

## fab resolve (extended)

Pure query, no side effects.

```
fab resolve [--id|--folder|--dir|--status|--pane] [--or-none] [--server <name>] [<change>]
```

| Flag | Output |
|------|--------|
| `--id` (default) | 4-char change ID |
| `--folder` | Full folder name |
| `--dir` | Directory path (`fab/changes/.../`) |
| `--status` | `.status.yaml` path |
| `--pane` | Tmux pane ID (errors `ERROR: no tmux pane found for change "<folder>"` if no matching pane) |
| `--or-none` | Absence-as-data opt-in: maps **state-sentinel** resolution failures to the exact token `(none)` on stdout + exit `0` — not-found always (bare and with an explicit `<change>` override); ambiguous **only bare** ("multiple changes exist, none active" IS the no-active-change state — a named-but-multi-matching override stays a non-zero error). Infrastructure errors (missing `fab/` root, I/O) stay non-zero, flag or no flag |
| `--server <name>` / `-L <name>` | Pane mode only: target tmux socket (`tmux -L <name>`), searched server-wide across all sessions; skips the `$TMUX` requirement. Without it, pane lookup is current-session-only and requires `$TMUX` (`ERROR: not inside a tmux session` otherwise) |

The five output-mode flags are **mutually exclusive** — passing two (e.g. `--status --folder`) exits `2` (a usage error — a flags-group conflict, caught before RunE) instead of silently picking one. `--or-none` is **not** part of that group: it composes with every output mode and with `--server`, and on the none path `(none)` replaces the mode-specific output. In `--pane` mode the mapping applies to the **change-resolution** step only — a pane-lookup failure after successful resolution (`no tmux pane found …`, `not inside a tmux session`) is not a state sentinel and stays non-zero.

The token is exactly `(none)` — not `none` (a legal 4-char change ID, so it would collide with `--id` output) and not empty output (illegible in transcripts, and hazardous in command substitution: `cd $(fab resolve --dir …)` with empty output cds to `$HOME`). Without the flag, absence stays an error (the hard stop `$(…)` consumers rely on) — absence-as-data is strictly opt-in. Conceptual split: **`fab preflight` is the validation gate (non-zero on a bad state by design); `fab resolve` is the pure query that can answer "none" when asked.**

`fab change resolve` is a thin wrapper over this same implementation with `--folder` mode fixed — and deliberately flag-free (no `--or-none`; the query flags live on top-level `fab resolve` only). Callers needing the probe form use `fab resolve --folder --or-none`.

---

## fab resolve-agent

Pure query (no side effects) — resolves a pipeline **stage** (or an agent **role** name) to its `{provider, model, effort}` agent profile for sub-agent dispatch. Consumed by the orchestrators (`/fab-ff`, `/fab-fff`, `/fab-proceed`) and `/fab-continue`'s sub-agent dispatch, which call it immediately before dispatching each stage's sub-agent, and by `fab agent` / the operator launcher (role-name resolution).

```
fab resolve-agent <stage|role> [--alias] [--provider <name>] [--model <id>] [--effort <level>]
```

The positional argument is either one of the six pipeline stages (`intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`) or one of the six role names (`default`, `operator`, `doing`, `review`, `hydrate`, `fast`). A role name is accepted positionally alongside a stage name: a stage maps through the fixed stage→role mapping; a role resolves directly. The two name sets overlap only at **fixed points** — a name shared by a stage and a role (`review`, `hydrate`) is one where the stage maps to that same-named role (`stageRoles[name] == name`), so role-first dispatch resolves such a name identically either way. (`ship` is a stage but not a role — it maps to the `fast` role.)

**Vocabulary**: a **role** is one of the six fixed slot names; a **profile** is a concrete `{provider, model, effort}` value; a **provider** carries independent `session_command`, `dispatch_command`, and `native` launch capabilities plus its per-role fills — capability presence says how, while `dispatch.mode` selects through the descending `pane → native → headless` ladder; **Tier 1 / Tier 2** is agent *depth* (what you talk to vs. what stages dispatch to), which is what the two `agent.session` / `agent.workers` knobs select a provider by.

**Resolution**: maps a stage → its role via the FIXED fab-owned stage→role mapping (`default`: intake (advisory) / `doing`: apply, review-pr / `review`: review / `hydrate`: hydrate / `fast`: ship — NOT user-overridable), then resolves the role → `{provider, model, effort}` through the fill precedence below. Built-in defaults today: `default`: claude/claude-fable-5/high, `operator`: claude/claude-sonnet-5/medium, `doing`: claude/claude-opus-5/high, `review`: claude/claude-opus-5/high, `hydrate`: claude/claude-opus-5/high, `fast`: claude/claude-sonnet-5/medium. The stage→role mapping and the role→depth partition are both fab-owned (no `stage_roles`, no per-stage escape hatch); the three launch capabilities and per-role fills live in the top-level `providers:` table, while `dispatch.mode` owns adapter preference. See `docs/specs/stage-models.md`.

**Fill precedence** — where the resolved profile comes from, most specific first:

```
provider:       invocation --provider
                >  agent.profiles.<role>.provider
                >  the role's depth knob (agent.session | agent.workers)
                >  the built-in claude

model / effort  invocation flag
(per field):    >  agent.profiles.<role>.<field>
                >  providers.<p>.profiles.<role>.<field>
                >  providers.<p>.profiles.default.<field>
                >  empty
```

`<p>` is the **resolved** provider. `empty` keeps its existing meaning (an empty `model=` line = "inherit the session model"; on a `dispatch=` command the placeholder's token, plus a preceding `-`-flag, is dropped so the CLI's own default applies).

**One cross-role fallback chain, on the provider side.** `agent.profiles` is a SPARSE per-role override: an unset field is simply not an override, and `agent.profiles.default` is the `default` **role's** own override — never a fallback source for the other five. Cross-role fallback is `providers.<p>.profiles.default`. Because model/effort always come from the resolved provider's own fills, **no value can be foreign to the provider that will run it** — which is why the former cross-provider *cutoff rule* (and its per-field ownership tracking and cross-scope limitation) is gone. The one value that survives a provider change is an explicit `agent.profiles.<role>.model`: a pin the user wrote by hand, not inheritance.

**Legacy spellings still read**: `agent.tiers` is the pre-2.17.0 name of `agent.profiles` (read per role, so a half-migrated config resolves every role), and the flat `providers.<name>.model`/`.effort` is read as an alias for `providers.<name>.profiles.default`. The `2.16.19-to-2.17.0` migration rewrites both.

**Output** (a `model=` line always, then optional `effort=`, `provider=`, and `dispatch=` lines; byte-stable for the same config):

```
model=<id>
effort=<level>
provider=<name>
dispatch=<command>
```

- The `effort=` line is **omitted** when the resolved profile has no effort (empty/absent); the `provider=` line is omitted when it has no provider.
- An **empty model** emits an empty `model=` line — signals "inherit the session/orchestrator model" (today's foreground/no-override behavior). Callers omit the dispatch `model` param in that case.
- `dispatch=` is derived from the resolved `dispatch.mode`, the provider's three independent capabilities, and `$TMUX`. Selection starts at the configured preference and descends only through `pane → native → headless`: pane requires `$TMUX` plus `session_command`, native requires `native: true`, and headless requires `dispatch_command`. The line is **absent iff native resolves**; pane emits the profile-substituted `session_command`, and headless emits the profile-substituted `dispatch_command`. Dispatch sites still branch only on line presence and never execute its value. A provider with no reachable rung is a non-zero actionable error naming the provider and capability keys; an explicit unknown `--provider` retains the earlier lookup error.
- **`dispatch.mode` (preference ceiling)** — a `pane | native | headless` config field, **default `native`**, **scope `both`** (settable once machine-wide in `~/.fab-kit/config.yaml`). `pane` means “prefer watchable”: inside tmux a provider with `session_command` resolves pane; outside tmux it descends to native when `native: true`, otherwise headless when `dispatch_command` exists. `native` never ascends to pane and descends to headless only when the provider lacks native capability. `headless` never ascends. Capability-field presence says **how** a rung runs, never **which** rung to prefer; adding `dispatch_command` alone cannot move a native-capable provider off native. No command field substitutes for another.

**`--alias` (Claude-Code Agent-tool adapter)**: when set, the `model=` line emits the Claude-Code **short alias** (`opus` / `sonnet` / `haiku` / `fable`) instead of the full versioned ID. This exists because the Claude Code **Agent tool's `model` parameter is a hard enum** that rejects full IDs — sub-agent dispatch must pass an alias. The mapping is prefix-based (`claude-opus-` → `opus`, etc.), so dated variants like `claude-haiku-4-5-20251001` resolve to `haiku`. The **default (flag absent) is** the full ID (the `claude` CLI `--model` flag, used by the operator launcher / `fab agent`, accepts full IDs and keeps resolving WITHOUT `--alias`). The **`effort=`/`provider=` lines are unaffected** by `--alias`. **Empty / non-Claude models pass through verbatim** (an empty `model=` line stays empty — the inherit signal; an unrecognized/non-Claude ID like `gpt-5` is emitted unchanged) — `--alias` is a best-effort adapter, not a Claude-only validator. The **`dispatch=` line ALWAYS embeds the FULL model ID even under `--alias`** — CLI dispatch never aliases (an external CLI's `--model` flag takes a full ID); aliasing is the Agent-tool-only adaptation. So under `--alias` the `model=` line is aliased while the `dispatch=` command still carries the full resolved ID.

```
$ fab resolve-agent apply
model=claude-opus-5
effort=high
provider=claude

$ fab resolve-agent apply --alias
model=opus
effort=high
provider=claude

# with the workers knob (or agent.profiles.doing.provider) pointed at a
# non-native provider (`agent.workers: codex` — no `providers:` block
# needed, since codex ships its own fills) the dispatch= line appears, and
# --alias leaves a non-Claude model= verbatim. Values are illustrative — run
# `fab config explain` for the fills your binary actually ships:
$ fab resolve-agent apply --alias
model=<codex-model-id>
effort=xhigh
provider=codex
dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m <codex-model-id> -c model_reasoning_effort=xhigh
```

**Invocation-time overrides** — `--provider <name>`, `--model <id>`, `--effort <level>` are the top rung of the fill precedence above, riding the same single resolution call:

- **`--provider`** swaps the resolved provider and **re-derives `dispatch=` from `dispatch.mode` plus the NAMED provider's capabilities** — so the selected rung and emitted-line presence can differ from the stage's unoverridden result. That is a **query result, not an adapter move**: `fab dispatch start` accepts no override flags and re-resolves the stage from config itself, so only a **config** override (`agent.workers`/`agent.session`, or `agent.profiles.<role>.provider`) actually relocates a stage. An override-only `dispatch=` line is not actionable, and the two remedies are **not interchangeable**: dispatching the stage natively with overridden model/effort works only when the overridden provider/model has the native Agent-tool seam, so for a cross-provider `--provider` override the config override above is the **sole executable path** (see § fab dispatch and `_preamble.md` § Per-Stage Model Resolution). A swap **re-derives** an unoverridden `model`/`effort` from the NEW provider's own per-role fills (`profiles.<role>`, then `profiles.default`, then empty). An explicit `agent.profiles.<role>.model` still wins. **Swapping to a built-in lands on that built-in's own shipped fill**; only a provider with no fills anywhere lands on an empty `model=`.
- **`--model` / `--effort` are valid WITHOUT `--provider`** here — a within-role override of the profile this pure query would otherwise print. **Deliberate asymmetry with `fab agent`**, where they stay a usage error without `--provider`: `resolve-agent` is a pure query whose whole output is a profile (overriding one field is unambiguous), while `fab agent` is a session launcher with two mutually exclusive addressing modes, where a bare `--model` would invent an undocumented role-override surface. See § fab agent.
- All three guards key on whether the flag was **supplied** (not on value emptiness), so `--model=` explicitly clears the model (emitting the inherit signal) rather than being ignored, and `--provider=` resolves an empty provider (the lookup failure below) rather than falling back to the depth knob.
- **An unknown `--provider` name is a LOOKUP failure** — non-zero exit naming the resolvable set (fab-kit's built-in table ∪ the project's `providers:` keys, sorted), mirroring `fab agent`'s error. It is not validation of any command's content: resolved strings still pass through verbatim.

```
# swap a single stage onto codex for this run, pinning a specific model
# (no config change; the explicit flags beat codex's own shipped fill):
$ fab resolve-agent review --provider codex --model <codex-model-id> --effort high
model=<codex-model-id>
effort=high
provider=codex
dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m <codex-model-id> -c model_reasoning_effort=high

# no --model/--effort → the swapped-to provider's OWN shipped fills apply
# (values illustrative; `fab config explain` prints what your binary ships):
$ fab resolve-agent apply --provider codex
model=<codex-model-id>
effort=xhigh
provider=codex
dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m <codex-model-id> -c model_reasoning_effort=xhigh

# a bare within-role override (valid here; a usage error on `fab agent`):
$ fab resolve-agent apply --effort medium
model=claude-opus-5
effort=medium
provider=claude
```

**No validation — verbatim pass-through**: `fab resolve-agent` does NOT validate the provider, model, or effort against any provider's accepted set (provider neutrality — a fab-kit design principle). It echoes the strings as-is — `xhigh`, `reasoning_effort:high`, an empty effort, whatever. A misconfigured pair (e.g. Sonnet + `xhigh`) is NOT corrected by fab; it surfaces as a dispatch-time error in the harness. There is no effort-enum enforcement and no degrade-gracefully drop. (An unknown `--provider` NAME is the one lookup-shaped exception above — naming the resolvable set is not validating a command's content.)

**Effort is advisory on the NATIVE arm.** The resolved `effort=` is only reliably honored where it rides a composed CLI command (`fab dispatch` headless/pane, the operator launcher). On native Agent-tool dispatch it can only be injected as a prompt instruction, and the session-level effort dominates — a known Claude Code limitation (GitHub issues #64033/#39220). Model differentiation works on both arms; treat a per-role effort as binding on the CLI arms and advisory on the native one. See `docs/specs/stage-models.md` § Effort asymmetry.

**Exit code**: non-zero only on a real error — an unreadable/malformed config, an unknown stage/role name, or a supplied `--provider` that resolves to no provider. A stage/role resolving to a default is success (exit 0).

---

## fab config

`config` is a six-verb command group: `show`/`explain` inspect, `set`/`unset` surgically modify, and `init`/`upgrade` own whole-file lifecycle.

```
fab config explain [key]      # registry documentation; --json for field rows
fab config show [key]         # effective config; --origin adds provenance
fab config set <key> <value>  # set one project override; --system targets user config
fab config unset <key>        # restore inheritance; --system targets user config
fab config init               # generate fab/project/config.yaml from the registry
fab config init --system      # write ~/.fab-kit/config.yaml scaffold
fab config upgrade            # reconcile fab/project/config.yaml against the registry
```

### fab config explain `[key]` (`reference` alias)

Pure query (no side effects, no file writes) — prints a **fully-commented reference `config.yaml`** to stdout, documenting every available option so users can discover the whole schema from one place.

```
fab config explain            # commented YAML (default)
fab config explain --json     # machine-readable field table
fab config explain <key>      # owning rendered segment only
```

Accepts at most one dotted key. A keyed query resolves rows rendered inside a shared segment to that owning segment; keyed `--json` returns the matching owning row(s) in the same array shape. Unknown keys fail non-zero naming the key. `reference` remains an invisible Cobra alias so historical pointers keep working. Runs from any directory — it reads no project config and depends on no environment state.

**Generated from a per-field metadata table, not hand-written**: the reference is generated by walking an ordered **per-field metadata table** (`internal/configref`) — each row carries the field's canonical `default`, expected YAML value kind, `description`, `scope` (`project`/`system`/`both`), `advertise` flag, and `renamed_from` carry-forward. Every default that has a canonical Go symbol is sourced from that symbol (`agent.DefaultSessionCommand`, the default role profiles via `agent.DefaultProfile`/`agent.RoleNames`, the pipeline stage names via `agent.StageNames`), so there is no second copy of those values and the shown defaults **cannot drift**. A field's `default` is the canonical built-in default, not the value the reference shows as an example: `source_paths`/`test_paths` render an example but their binary default is empty. See `docs/specs/config.md` for the schema.

**`--json`**: emits the same public field table as a flat, deterministic JSON array (table/rendering order) — per-field objects `{key, default, description, scope, advertise, renamed_from}` with `renamed_from` omitted when empty. Expected kind remains an internal validation signal, so this established JSON wire shape is unchanged. Both renderings are pure and byte-stable; the JSON key set is guarded against drift from the YAML reference's documented keys.

**Full schema coverage**: covers BOTH the binary-consumed keys (modeled on the `Config` struct) AND the skill-consumed keys (read by markdown skills, invisible to Go reflection) — `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `providers.*` (`session_command`/`dispatch_command`/`native`/`profiles.<role>.{model,effort}`), `agent.session`, `agent.workers`, `agent.profiles.*` (`provider`/`model`/`effort`), `dispatch.mode` + `dispatch.column_width` + `dispatch.reap_done`, `stage_hooks.*`, `branch_prefix`. (The retired `review_tools` block moved to `fab/project/code-review.md` § Review Tools; `agent.spawn_command` moved to `providers.claude.session_command`; `fab_version` moved OUT of config.yaml to the plain-text sibling `fab/.fab-version` in 2.15.0 and is no longer a config-file key.) Baseline keys appear live with example values (including the provider capability grammars and the two `agent:` depth knobs); the opt-in override blocks (`agent.profiles`, `stage_hooks`, `branch_prefix`) appear commented-out with fab-kit's built-in defaults shown, so uncommenting is opting in. **The reference is the FULL schema; a project's managed fence is slimmer** — `agent.profiles` and `providers` are `advertise: false` as of 2.17.0, so they are documented here but no longer scaffolded into every repo (see § fab config upgrade and `docs/specs/config.md`).

The `providers:` block documents fab-kit's **three built-in providers** — `claude` (the default), `codex`, and `gemini` — each carrying its independent launch capabilities plus per-role fills in the binary (`internal/agent`'s built-in provider table), so **naming `codex`/`gemini` on a depth knob, in `agent.profiles.<role>.provider`, or on a `--provider` flag resolves a real model for every role with no `providers:` block at all**. Claude ships `session_command`, `native: true`, and the stdin-driven `dispatch_command` `claude -p --dangerously-skip-permissions --model {model} --effort {effort}`; codex/gemini ship both command fields and no native capability. Presence describes how each rung runs and never selects `dispatch.mode`: under the default `native`, claude resolves native while codex/gemini descend to headless. Claude's fill map is exhaustive (all six roles); codex's and gemini's are **sparse** — a role they omit resolves that provider's `default` entry. Non-claude fills are **refreshed at kit-release cadence** and pass through unvalidated; pin a newer model with one config line (`providers.<name>.profiles.<role>.model`, scope `both` — settable once per machine in `~/.fab-kit/config.yaml`). The provider blocks render **commented** because they merely restate built-in defaults. All three ship a deliberate full-auto posture because unattended stage workers cannot answer approval prompts: claude carries `--dangerously-skip-permissions`, codex `--dangerously-bypass-approvals-and-sandbox`, and gemini `--approval-mode=yolo`. Override the corresponding command to restore approvals. Gemini's commands carry no `{effort}` placeholder and no `-p` on `dispatch_command` (gemini reads the stdin-piped prompt in non-TTY mode).

**Output**: byte-stable for a given binary version (same convention as `fab resolve` / `fab resolve-agent`). The emitted document round-trips — its live keys parse cleanly back into `Config`.

**Exit code**: 0 on success. Unknown keys fail operationally; more than one positional argument is a usage error. Writes no file.

### The config cascade (environment > project > system > defaults)

`fab config show` and `show --origin` display the **effective** config after resolving the four-layer cascade the loader (`internal/config.LoadPath`) applies to *every* config read:

1. **environment** — YAML-valued variables derived generically from registry keys (highest precedence)
2. **project** — `fab/project/config.yaml`
3. **system** — `~/.fab-kit/config.yaml` (user-global, all repos on the machine)
4. **built-in defaults** — the Go tables in the binary

Layers merge by **per-field deep merge**: maps merge per-key (the `agent.profiles` precedent), lists replace (never concatenate), scalars replace — environment wins. Environment names are mechanically `FAB_` + the uppercase dotted registry key with `.` replaced by `_`; values are parsed as YAML, so scalars, lists, and maps retain config types. The loader walks only the ordered registry keys, honors only `scope: system`/`both`, treats an empty value as unset, and fails open: a malformed value or project-scoped override emits a `fab: warning:` on stderr and is skipped; unrelated/unknown `FAB_*` variables are never scanned. The deliberately advertised variables are `FAB_AGENT_WORKERS` and `FAB_AGENT_SESSION`; similarly named `FAB_AGENTS` is unrelated and has no config meaning. The file-layer behavior remains unchanged: an absent or malformed/unreadable system file is skipped (with a warning for malformed/unreadable), while a malformed *project* file still errors. Warnings never change stdout contracts or exit codes.

### fab config show `[key] [--origin]`

Pure query (no file writes) for the current repo. `docs/specs/config.md` owns the cascade/provenance rationale and history.

| Mode | Output |
|------|--------|
| `fab config show` | Environment-over-project-over-system merge as YAML; built-in defaults remain point-of-use values and are not materialized |
| `fab config show --origin` | Effective values including defaults, with origin `$ENV_VARIABLE` / `project path` / `system path` / `default`; map fields (`agent.profiles`, `providers`) drill down per key |
| `fab config show <key>` | Effective value including the built-in default; scalar/list values are raw, map values are YAML subtrees |
| `fab config show <key> --origin` | Scalar/list value with one origin suffix; map subtree as per-leaf dotted rows with origins |

```
fab config show               # effective config as YAML
fab config show --origin      # each field: value + origin ($ENV_VARIABLE / project path / system path / default)
fab config show agent.workers # one effective value
```

Accepts at most one known dotted key; unknown keys fail non-zero naming the key. Requires a fab repo (walks up for `fab/`, like `fab preflight`). Writes no file. Bare output remains unchanged.

### fab config set / unset

```
fab config set <key> <value> [--system]
fab config unset <key> [--system]
```

Both route through `internal/configupgrade`'s comment-preserving path splicer and atomic writer; neither marshals the whole document. `set` accepts only documented **scalar leaves** and a single-line, comment-free YAML scalar value (`string`, `bool`, `int`, or `float`). It refuses structural map keys, collection-valued leaves, collection/null values, multiline input, and YAML comments with an explicit manual-edit remedy. Fence-only deep fields materialize every ancestor from the registry renderer before the scalar is inserted; opaque provider names remain supported through quoted YAML mapping keys. `unset` stays kind-ungated so it can repair any current value shape, removes only the live override, and lets the fence advertise the field again. Unsetting a known absent key exits 0 with a notice. The shared value parser remains collection-aware (flow or block style) solely because environment overlays accept lists and maps; that broader ENV contract does not widen `set`. Relative to the pre-2.17.3 env layer, four degenerate spellings now warn-and-skip instead of resolving to surprising values: whitespace-only, comment-only, multi-document, and bare-date (`!!timestamp`) env values are rejected per-variable, fail-open.

Without `--system`, the target is `fab/project/config.yaml`. With it, the target is `~/.fab-kit/config.yaml`; only `scope: system`/`both` keys are accepted, and a missing file is created with the canonical system scaffold header and no project reference fence. Unknown keys fail naming the key and point to `fab config explain`.

### fab config init `[--system]`

Bare `fab config init` selects project mode. `--project` is retained for compatibility; `--system` selects the system scaffold, and passing both explicit flags errors.

**`--system`** writes a `~/.fab-kit/config.yaml` **scaffold** — a header explaining the system layer, then **only** the system-overridable fields (`scope: system`/`both` — today the `agent:` block and `providers`), **all commented**, generated from the same per-field metadata table as `fab config explain` so it cannot drift from the schema. It is the user's answer to "what can I safely override at the system level".

**`--project`** generates a fresh `fab/project/config.yaml` from the registry — the retirement path for the hand-maintained scaffold `config.yaml` (deleted in 2.15.0). It writes the **A-class identity fields** (`--name`, `--description`, `--source-path` repeatable, `--test-path` repeatable) **live** above the managed reference fence, then the fence of commented C fields. No `agent:` key is pinned (presence=intent — an init-pinned knob or role profile would be an accidental override that stops tracking fab-kit's defaults). It shares the same fence renderer as `fab config upgrade`, so a generated file and an upgraded file carry a byte-identical fence. This is the shell-out target `fab init` (the fab-kit binary) calls to bootstrap a project config; when the installed fab-go predates it, `fab init` falls open to a minimal embedded stub instead (a fresh repo never fails preflight for lack of a config.yaml).

```
fab config init --name X --description Y --source-path src/ [--test-path "**/*_test.go"]
fab config init --system      # write the ~/.fab-kit/config.yaml scaffold (refuses to overwrite)
```

Both modes **refuse to overwrite** an existing target file (non-zero exit, message naming the path) — the file is user-owned once created; there is no `--force`. The `--system` scaffold is fully commented (inert until uncommented).

### fab config upgrade

The whole-file reconciliation command for `fab/project/config.yaml`. It shares `internal/configupgrade`'s comment-aware writing engine and fence renderer with surgical `set`/`unset`.

```
fab config upgrade            # reconcile config.yaml against the registry (idempotent)
```

Reconciliation, under the A/B/C field-category model:

- **Live (A) fields kept verbatim**, including the user's own comments. *Presence = intent*: a live field is an override even when its value equals the default — it is NEVER auto-removed (B-hygiene "equals default — remove?" is an advisory report line only).
- **The managed fence (C fields)**: `advertise: true` fields not currently overridden are regenerated as a fully-commented scaffold (including parent keys — a live `agent:` over comment-only children is exactly the `agent: null` the old masher produced) inside byte-exact splice anchors: `# >>> fab reference (kit X.Y.Z) >>> …` / `# <<< end fab reference <<< …`. Upgrade rewrites ONLY between the anchors; everything outside is the user's. The fence omits fields already overridden above it (at top-level-key granularity — a live top-level key suppresses the whole scaffolded block under it). A legacy file with no fence gets one appended at the bottom. Content the user places BELOW the fence is never dropped — it is **hoisted above** the fence on the next run and classified like any other live key.
- **Unknown fields parked, never deleted**: a live key no longer in the registry is parked in a `# removed in … (parked by fab config upgrade — delete when done):` block below the fence, its value serialized — appended exactly once, never regenerated away.
- **Renames carried mechanically**: a live field matching a registry row's `renamed_from` is carried to the new key, value verbatim (empty on every row today). A carry is **skipped** (and reported) if the target key is already live, so it never emits a duplicate top-level key.

**Byte-stable and idempotent** — running it twice yields a byte-identical file (the `fab memory-index` discipline). Before writing, the reconciled document is **validated as YAML** and a run that would produce an unparseable file is **refused** (original left untouched) rather than bricking the repo. The write is atomic (`internal/atomicfile`). Requires a fab repo (walks up for `fab/`); `cobra.NoArgs`. `fab upgrade-repo` **auto-runs** it after sync (fail-open: if the installed fab-go predates the subcommand, it prints a reminder and the upgrade continues).

## fab pane

Tmux pane operations with fab context enrichment. `fab pane <map|capture|send|process|window-name> [flags...]`

**Pane-family exit codes** (capture, send, window-name): pane validation failures use a shared scheme so callers can branch on cause — `2` = pane missing, `3` = any other tmux failure (dead server, bad socket). `map` and `process` use plain `ERROR:`-formatted exit 1. **Usage-error coexistence**: a *usage* error on any pane verb — a bad flag or a cobra arg-count violation — exits `2` at parse time (the binary-wide convention above), caught before the handler runs; the in-handler `2` = pane-missing / `3` = tmux-failure scheme is a separate, in-handler `os.Exit` path that bypasses the usage/operational mapping. Exit `2` on a pane verb is therefore ambiguous between "usage error" (at parse time) and "pane missing" (in-handler) — disambiguate on stderr wording; the codes are not renumbered.

**Persistent flag** (all subcommands): `--server <name>` / `-L <name>` (default `""`) — target tmux socket (`tmux -L <name>`). Defaults to `$TMUX` / tmux default. Lets daemons on one tmux server inspect panes on another.

**§ agent state (`@rk_agent_state` convention — read-only).** `map`/`capture`/`send` resolve a pane's agent lifecycle state by READING the tmux pane user option `@rk_agent_state` (value `"<state>:<epoch_seconds>"`, `state ∈ active | waiting | idle`), written by run-kit's `rk agent-setup` global agent-harness hooks (covering Claude Code, Codex, Copilot, Gemini, OpenCode — not just Claude). fab is a pure CONSUMER: it never writes the option and needs no run-kit software installed — it reads with plain tmux (`map` via the `#{@rk_agent_state}` field on its existing `list-panes -F` call; `send`/`capture` via `tmux show-options -pv -t <pane> @rk_agent_state`). `active` = turn in progress, `waiting` = blocked on a human (permission prompt / menu / elicitation), `idle` = turn complete. The epoch suffix is mandatory — idle duration is `now - epoch`; only `idle` carries a duration. An absent option, unknown token, or missing/non-integer epoch is **unknown** (`—` in tables, `null` in JSON, refused by `send` without `--force`). No staleness heuristic: a stale `active` (e.g. an Esc-interrupted agent) still refuses sends — `--force` is the escape hatch.

### map — `fab pane map [--json] [--session <name>] [--all-sessions] [--server <name>]`

All tmux panes with pipeline state. Non-git/non-fab panes included with `---` fallbacks.

| Flag | Description |
|------|-------------|
| `--json` | Emit the snake-case JSON field set below |
| `--session <name>` | Target specific session (skips `$TMUX` check) |
| `--all-sessions` | Query all sessions (skips `$TMUX` check; mutually exclusive with `--session`) |

| JSON field | Type / meaning |
|------------|----------------|
| `session`, `window_index`, `pane`, `tab`, `worktree`, `change`, `stage` | Table-equivalent identity/context fields |
| `window_id` | `string\|null`; stable server-assigned tmux `@N` identity that follows `swap-window`/`move-window`; JSON-only |
| `repo` | `string\|null`; absolute main-worktree root; JSON-only |
| `display_state` | `string\|null`; `active` / `ready` / `done` / `failed` / `pending` / `skipped`, or `null` with no stage; JSON-only |
| `agent_state` | `string\|null`; `active` / `waiting` / `idle` from `@rk_agent_state`, else `null` |
| `agent_idle_duration` | `string\|null`; populated only for `idle` |
| `pr_url` | `string\|null`; last `.status.yaml` `prs:` entry; JSON-only |
| `pr_number` | `number\|null`; trailing `/pull/<n>` parsed from `pr_url`; JSON-only |

PR fields come from the already-loaded status file: **no `gh`/`git`, no network, no PR status (open/merged/CI)**. Consumers fetch live state themselves. `stage: "review-pr"` plus `display_state: "done"` distinguishes a parked shipped change from `display_state: "active"` review work.

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

Process manager for CLI-dispatched pipeline stages, in **two modes** — the two non-native adapters of cross-harness stage dispatch. `fab dispatch <start|restart|status|wait|logs|kill|reap|clean> [args...]`. Full cross-adapter contract (three adapters: native Agent-tool / headless CLI / interactive pane): `docs/specs/harness-adapters.md`. `restart` is the family's **recovery** verb (§ restart) — it relaunches a non-running dispatch from the prompt `start` persisted; the observation policy that spends it lives in `_preamble.md` § CLI-Adapter Dispatch → *Recovery policy*. `status` and `wait` are the family's two **observation** verbs over one derivation: `status` is the one-shot probe, `wait` its blocking sibling (§ wait). `reap` is the family's **hygiene** verb (§ reap) — it reclaims a *done* pane worker's tmux pane and is a reported no-op in every other case, which is what keeps it distinct from `kill` (§ kill), the *recovery* verb valid in any state.

| Mode | Selection signals (resolved by the § start ladder, in precedence order) | Worker | Command composed | Completion observed via | tmux |
|------|------------------------------------------------------------------------|--------|------------------|-------------------------|------|
| **headless** | `--headless`, `--timeout`, or automatic descent to the headless rung | detached `sh -c` process | the provider's `dispatch_command` | `{stage}.exit` + pid liveness + result file | never touched |
| **pane** | `--pane`, `--server`, or automatic selection of the pane rung | an interactive agent session in a tmux pane you can watch and steer — **split into your own window** when you are a tmux pane on the target server, else a new window | the provider's `session_command` | **result file** + pane liveness | **required** — as is a `session_command`; either prerequisite missing is a hard error when explicit and a skipped rung when automatic |

The two provider command fields are **never merged and never substitute for each other**; `native` is a third, explicit provider capability. Headless mode stays **tmux-independent**; pane mode borrows tmux. Automatic selection starts at `dispatch.mode` (default `native`) and descends only through `pane → native → headless`. `fab dispatch` can launch the two non-native rungs; if its fresh selection lands on native it stops before writing state and tells the caller to re-run `fab resolve-agent`. Headless dispatch remains parallel to and independent of `fab pane` / `fab operator`; pane dispatch borrows tmux as a launch surface but does **not** join the operator's monitored set. **POSIX-only (v1)** — headless `start`/`kill` error clearly on Windows (`fab dispatch requires a POSIX shell (setsid/timeout); Windows is not supported in v1`) rather than half-working.

**State layout** — `.fab-dispatch/{4-char-change-id}/` at the **repo root** (alongside `.fab-status.yaml`, already gitignored via the scaffold `.fab-*` pattern — no gitignore/scaffold/migration work). Keyed by the stable 4-char change ID (stable across `fab change rename`); one dir per worktree. Both modes share the dir, the loader, and the refuse-if-running check. Per-stage files:

| File | Written by | Contents |
|------|-----------|----------|
| `{stage}-prompt.md` | `start` (from stdin) | the stage prompt — piped to the dispatched command's stdin (headless) or **pointed at** by the one-line prompt the pane worker receives (pane). Also **`restart`'s input**, which reads it and leaves it byte-identical (see § restart) |
| `{stage}.yaml` | `start` (via `internal/atomicfile`) | `spawn_cmd` (resolved) + `started_at`, plus the mode's identity: `pid`/`pgid`/`timeout` (headless) or `pane`/`window`/`server` (pane — where `window` holds the `fab-{id}-{stage}` identity string, meaning the tmux **window name** in the new-window shape and the tmux **pane title** in the split shape). Every mode-specific key is omitted when empty, so a headless record's shape is unchanged and the mode is **derived** from which keys are present (no stored discriminator) |
| `{stage}.log` | the wrapper | combined stdout+stderr of the dispatched command — **headless only** (a pane worker's output is tmux scrollback) |
| `{stage}.exit` | the wrapper | the exit code (`echo $? > ...`) — its presence is the "process finished" signal; **headless only** |
| `{stage}-result.yaml` | the dispatched agent (contract) | the stage result; presence is required for `done` in both modes, and is the **sole** completion signal in pane mode |

### start — `fab dispatch start <change> <stage> [--timeout <secs>] [--pane] [--headless] [--server <name>]`

Resolves `<change>` → 4-char ID; reads the stage prompt on **stdin** → `{stage}-prompt.md`; resolves the stage's role → provider internally (via `internal/agent` + `internal/spawn` `{model}`/`{effort}` substitution — the same resolution `fab resolve-agent` performs). **This re-resolution reads CONFIG ONLY — `start` has no `--provider`/`--model`/`--effort` surface**, so `fab resolve-agent`'s invocation-time overrides do NOT reach it: an override binds the **native Agent-tool arm** only, and moving a stage onto CLI dispatch requires a config override (`agent.workers`/`agent.session`, or `agent.profiles.<role>.provider`) rather than a flag (see § fab resolve-agent → Invocation-time overrides). Everything up to the launch is shared by both modes; the launch differs.

**Mode selection = explicit overrides, then the configured descent ladder.** Evaluated in order, the first explicit match wins (each keys on whether the flag was **supplied**, not its value, so `--timeout 0` / `--server ""` still count). With no explicit signal, selection starts at `dispatch.mode` and chooses the first possible rung at or below it:

| # | Signal | Mode | Selection is |
|---|--------|------|--------------|
| 1 | `--pane` | pane | explicit |
| 2 | `--headless` | headless | explicit |
| 3 | `--timeout <secs>` | headless | explicit (the timeout is enforced by the headless wrapper, so it can only mean headless) |
| 4 | `--server <name>` | pane | explicit (the flag exists solely to target a pane's socket) |
| 5 | *(none of the above)* | first possible rung in `pane → native → headless`, beginning at `dispatch.mode` | **automatic** |

- Pane is possible when tmux is available and the provider has `session_command`; native when `native: true`; headless when `dispatch_command` is present. A missing prerequisite skips that rung, and automatic selection never ascends. At the resolver seam `$TMUX` presence represents tmux availability; here a real reachability probe validates an automatically selected pane before launch, and a failed probe re-runs the same descent with `tmux unreachable`. An empty `$TMUX` reads as unset. **`$TMUX_PANE` is separate**: it chooses pane shape only after pane mode resolves.
- An automatically selected pane targets the **current** server — no `-L` is passed unless `--server` was given.
- **`--pane` + `--headless` is a usage error** (exit 2, cobra flag group — fired before any work, so nothing is launched and nothing persisted). `--headless` + `--timeout` **composes** (both select headless).
- An automatic native result is not launchable by this command family: `start`/`restart` error before state writes and tell the caller to re-run `fab resolve-agent <stage> --alias`, whose omitted `dispatch=` line selects the native Agent-tool adapter.

**Headless** — launches the resolved `dispatch_command` **DETACHED**, cwd = repo root:

```sh
sh -c '<resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'
```

The shell is launched with `setsid` semantics (Go's `SysProcAttr{Setsid:true}`, not a `setsid` binary prefix — prefixing it would double-fork and leave the recorded pid pointing at a process that exits immediately), detaching it into a new session/process group so the dispatch **survives the orchestrator dying** — no Go supervisor remains, the shell records the exit code itself and the recorded `pid`/`pgid` track the live worker. `--timeout N` wraps the resolved command in POSIX `timeout N <cmd>` **inside the wrapper** (no Go timer/daemon); a timed-out command exits `124`, surfacing as `failed`.

**`--pane`** — runs the worker **interactively in a tmux pane** instead, composing the provider's `session_command` (the same string `fab agent` composes, so **no new provider config field** exists or is needed). The worker's **pane ID**, identity string, and tmux socket label are persisted in `{stage}.yaml`. **WHERE the pane opens has two shapes**, decided from `$TMUX_PANE` and `--server`:

| # | Condition | Shape | tmux call | Identity carried by |
|---|-----------|-------|-----------|---------------------|
| 1 | `$TMUX_PANE` set **and** no `--server` | **split** — a pane inside the **dispatching agent's own window** | `tmux split-window {-h -l <n>%\|-v} -t <target> -c <repo-root> "<resolved-cmd> <shell-quoted-pointer>"` then `select-pane -T fab-{id}-{stage}` | the **pane title** |
| 2 | `--server <name>` supplied | **new window** | `tmux -L <name> new-window -n fab-{id}-{stage} -c <repo-root> "<resolved-cmd> <shell-quoted-pointer>"` | the **window name** |
| 3 | `$TMUX_PANE` unset | **new window** | `tmux new-window -n fab-{id}-{stage} -c <repo-root> "…"` | the **window name** |

This is the two-tier tmux hierarchy: an operator opens worktree agents as windows; a worktree agent's stage workers appear as panes beside it. Details:

- **Placement:** split workers form a stacked right column keyed on same-socket dispatch-record pane IDs, never mutable pane titles. Intersect recorded panes with the caller window's live panes; split the last live sibling with unsized `-v`, or carve once from `$TMUX_PANE` with `-h -l <n>%` at `dispatch.column_width` (default 35). This is a creation-time column invariant: never `select-layout`, repair, rearrange user panes, or fight manual resizes. Placement is cosmetic and warn-only: probe/read failures retain usable records and name the chosen fallback; absent state is the normal first run; failed `list-panes` still makes a sized carve; tmux versions rejecting percentage size retry unsized.
- **Identity and lifecycle:** `fab-{id}-{stage}` is the `window` record value, pane title for split shape, and window name for new-window shape. Pane ID is the downstream identity, making `status`, `kill`, capture, and refuse-if-running shape-blind; failed title-setting only warns. A split kill leaves the agent window intact. Neither shape carries operator `»`/`›` markers unless an operator explicitly enrolls it with `fab pane window-name ensure-prefix`.
- **Prompt delivery:** stdin is persisted to `{stage}-prompt.md`; the pane worker gets one shell-quoted pointer to it as the command's prompt argument. This preserves identical prompt content, handles quote-bearing checkout paths, avoids argv/send-keys size and printed-prompt hazards, and leaves the resolved `session_command` verbatim so its own shell expansion still occurs.
- **Selection and validation:** pane requires reachable tmux plus `session_command`. Explicit `--pane` or `--server` failure launches and writes nothing; the unreachable error is `pane mode requires a reachable tmux server, but <target> is unreachable; start tmux (or pass --server <name>), or pass --headless (drop --pane/--server) to dispatch headless`, while missing command names `providers.<name>.session_command`. Automatic missing prerequisites descend and print `dispatch selection: <selection-reason>` on stderr. Reachability is a real tmux query; if it fails after an automatic pane choice, selection repeats with reason `pane unavailable: tmux unreachable` and may land on native (actionable re-resolve error) or headless. `--server` / `-L` targets and persists a socket, implies pane, and is ignored by headless. `--pane` + `--timeout` is the usage error `--pane and --timeout are mutually exclusive: --timeout is enforced by the headless launch wrapper (POSIX timeout), which pane mode does not use`; bare `--timeout` selects headless. `--headless` forces detached work inside tmux and remains mutually exclusive with `--pane`.

Common to both modes:

- **No reachable capability → actionable error:** explicit pane/headless still hard-error on their own missing command key. Automatic selection skips unavailable rungs; if none remains, it errors naming the provider and the three capability keys (`providers.<name>.session_command`, `.native`, `.dispatch_command`). This is mode descent, never command-field substitution.
- **Concurrency = refuse-if-running + last-attempt-only**: refuses if a dispatch for the exact `(change, stage)` pair is already `running` (`a dispatch for <change>/<stage> is already running (pid N | pane %N); run fab dispatch kill first`), applying the **prior** record's own mode's finished signal — the same one `status` derives that mode's state from, so `start` and `status` can never disagree. Headless: `{stage}.exit` absent **and** the pid alive. Pane: `{stage}-result.yaml` absent **and** the pane alive — result presence wins over pane liveness there too, since an interactive worker sits at its prompt after finishing and a liveness-only rule would refuse forever after a successful pane run. A `start` over a **completed** prior attempt (done / failed / orphaned) **overwrites** its files — no per-attempt history. `restart` applies this same check and the same overwrite semantics, so it is the argument-free way to take that route when the prompt is already on disk (§ restart). Different stages of the same change share `.fab-dispatch/{id}/` via distinct `{stage}.*` filenames and do not collide.
- Output: `dispatched <id>/<stage> (pid N, pgid N)` (headless), `dispatched <id>/<stage> (pane %N, split, title fab-<id>-<stage>)` (pane, split shape), or `dispatched <id>/<stage> (pane %N, window fab-<id>-<stage>)` (pane, new-window shape — byte-identical to before the split shape existed). Automatic success appends its exact selection reason; explicit selection carries no suffix and remains byte-identical. The selector's complete reason set (some native forms become the actionable `fab dispatch` error rather than a `dispatched` line) is:

  - Direct: `mode: pane (preferred)`, `mode: native (preferred)`, `mode: headless (preferred)`.
  - One-rung pane descent: `mode: native (descended: pane unavailable: no tmux)`, `mode: native (descended: pane unavailable: tmux unreachable)`, `mode: native (descended: pane unavailable: no session_command)`.
  - Native descent: `mode: headless (descended: native unavailable)`.
  - Two-rung descent: `mode: headless (descended: pane unavailable: no tmux; native unavailable)`, `mode: headless (descended: pane unavailable: tmux unreachable; native unavailable)`, `mode: headless (descended: pane unavailable: no session_command; native unavailable)`.

### restart — `fab dispatch restart <change> <stage> [--timeout <secs>] [--pane] [--headless] [--server <name>]`

Relaunches a non-running dispatch through `start`'s shared prologue, validation, flag exclusions, launch, save, record/output shape, stale-attempt clearing, refuse-if-running rule, and last-attempt-only semantics. It has two unique behaviors:

- **Current-environment mode:** mode and pane shape are re-derived from current config, provider capabilities, and environment, never inherited. A pane dispatch orphaned by a dead server descends again; it may relaunch headless or stop for native re-resolution. A restart issued from a tmux pane can split that pane's window when pane is still at or below the configured preference.
- **Persisted-prompt input:** reads `.fab-dispatch/{id}/{stage}-prompt.md` instead of stdin and leaves it byte-identical. Refuse-if-running is checked before prompt presence. Missing prompt returns `no persisted prompt at <path> — nothing to relaunch; run \`fab dispatch start\` with the prompt on stdin` and touches no state.

### status — `fab dispatch status <change> <stage> [--json]`

Byte-stable poll surface. Reads `{stage}.yaml`, then derives the state by the record's **mode**. The five state **strings are the cross-adapter contract** — identical in both modes; what differs is which are reachable.

**Headless** — reads `{stage}.exit`, probes `pid` liveness (POSIX `kill(pid,0)`), reports one of all five:

| State | Condition |
|-------|-----------|
| `running` | pid alive AND `{stage}.exit` absent |
| `done` | `{stage}.exit` == `0` AND `{stage}-result.yaml` present |
| `failed` | `{stage}.exit` present AND != `0` (includes `124` timeout) |
| `failed (no-result)` | `{stage}.exit` == `0` BUT `{stage}-result.yaml` absent — a **contract violation, NOT done** |
| `orphaned` | pid dead AND `{stage}.exit` absent (reboot / `kill -9` / crash) |

A clean exit (code 0) is necessary but **not sufficient** for `done` — the result file must exist.

**Pane** — reads result-file presence and **pane** liveness (no exit file is ever written or consulted), reporting a **subset of three**:

| State | Condition |
|-------|-----------|
| `done` | `{stage}-result.yaml` present |
| `running` | result absent AND the pane is alive |
| `orphaned` | result absent AND the pane is dead (killed / crashed / tmux server gone) |

**Result presence WINS over pane liveness** — an interactive worker that produced its result and is still sitting at its prompt reads `done`, not `running` (a liveness-first rule would never terminate). **`failed` and `failed (no-result)` are UNREACHABLE in pane mode**: there is no exit-code channel, so a crashed or killed worker collapses into `orphaned`.

Human output is the bare state string on stdout. `--json` emits `{change, stage, state, mode, …}` where `mode` is `headless` or `pane` and the mode's identity keys follow: `pid`, `pgid`, `exit?` (headless) or `pane`, `window` (pane). Keys for the other mode are **omitted**, so a headless object is unchanged apart from the added `mode`.

### wait — `fab dispatch wait <change> <stage> [--timeout <secs>] [--json]`

`status`'s **blocking** sibling: it blocks until the dispatch's state leaves `running`, then prints that state **exactly as `status` does** (same text output, same `--json` object) and exits 0. It exists so an orchestrator can be **woken by a state change** instead of burning an inference turn every 30s asking whether one happened — the wiring in `_preamble.md` § CLI-Adapter Dispatch step 2 runs it as a **background command** (the harness's notify-on-exit seam) and falls back to a plain foreground blocking call on harnesses without one.

- **One derivation, shared with `status`.** The block re-derives state on an internal **~2s tick** through the same record loader and the same pure derivations `status` uses — `DeriveState` (headless: `{stage}.exit` + result file + pid liveness) or `DerivePaneState` (pane: result file + pane liveness), selected by the record's derived mode. `wait` and `status` therefore **cannot disagree** about state, by construction. The tick is an internal constant with **no flag and no config field**.
- **Not a file watcher.** A watcher on `{stage}-result.yaml` would see a worker *finish* but never see one *die* — `orphaned` is derived from pid/pane liveness, which needs a periodic probe regardless. The cost being eliminated is inference turns, not syscalls.
- **`--timeout <secs>` bounds the block, and expiry is NOT an error.** On expiry `wait` prints the still-current state — necessarily `running`, since any other state would already have ended the wait — and exits **0**. **The state string is the sole discriminator**: read `running` ⇒ the wait timed out (the wiring treats that wake as its peek-on-suspicion moment); read anything else ⇒ a terminal state was reached. Absent (or `0`) ⇒ wait indefinitely.
- **Already-terminal returns immediately** — no tick is consumed when the state is non-`running` at entry, so the verb is idempotent and safe to re-arm after a `restart` or re-run after an interruption.
- **Exit codes**: 0 for any successfully observed state (terminal **or** timeout); non-zero only for real errors — no dispatch record for the pair, or an unresolvable change — with the **same message surface as `status`** (`no dispatch for <change>/<stage> (run \`fab dispatch start\` first)`).
- **`--json`** emits the same object `status --json` emits, through the same render path (mode discriminator + that mode's identity keys).
- **Does not touch the worker.** `wait --timeout` bounds the *observer*; the unrelated `start`/`restart` `--timeout` is a POSIX `timeout` inside the headless wrapper that **kills** the worker. Waiting has no side effects at all: no state is written and no signal is sent.

### logs — `fab dispatch logs <change> <stage> [--tail N]`

Prints `.fab-dispatch/{id}/{stage}.log`. `--tail N` prints the last N lines (Go-side, no external `tail`). Missing log → `no dispatch log for <change>/<stage>`.

**Pane dispatches keep no log file** — an interactive worker's output is tmux scrollback, not a redirected stream — so `logs` reports that and names the equivalent: `<change>/<stage> is a --pane dispatch and keeps no log file (an interactive worker's output is tmux scrollback); read it with 'fab pane capture <pane>'`. When the record carries a `server` (the dispatch was started with `--server`/`-L`), the suggested command carries the socket too — `fab pane capture -L <server> <pane>` — since a socket-scoped pane is unreachable from a default-socket capture. This report is therefore the copy-pasteable source for the capture command; `status --json` exposes `pane` but not `server`.

### kill — `fab dispatch kill <change> <stage>`

Terminates the dispatch by the mechanism its recorded **mode** implies, idempotently in both cases (an already-dead target is a benign no-op with a clear report; a missing record is a clear error `no dispatch for <change>/<stage>`):

- **Headless** — `SIGTERM` to the **process group** (`pgid` from `{stage}.yaml`) so the detached command and its children die together. Already dead → `dispatch <change>/<stage> already dead (pid N); nothing to kill`.
- **Pane** — kills the **tmux pane** (`tmux kill-pane -t <pane>`), taking the interactive worker down with it. Shape-blind: the target is the pane ID, so a **split** worker's pane dies while the dispatching agent's window (and any sibling worker) survives, and a **new-window** worker's window closes with its only pane. Already gone → `dispatch <change>/<stage> already dead (pane %N); nothing to kill`.

No marker file is written in either mode: with no result file present, a killed dispatch simply reads `orphaned` on the next `status`.

> **`kill` vs `reap`** (§ reap). `kill` is the **recovery** verb: valid in **any** state, ungated by config, and the one the Recovery policy spends on a parked worker. `reap` is the **hygiene** verb: it fires **only** on `done`, is gated on `dispatch.reap_done`, and refuses to touch a live or failed dispatch. Use `kill` to terminate something; use `reap` to reclaim the space something finished with. They share the same `kill-pane` mechanism and the same idempotence, and neither removes any `.fab-dispatch/` state.

### reap — `fab dispatch reap <change> <stage>`

Reclaims a **done pane worker's tmux pane**. It exists because a pane-mode worker never exits on completion — it writes `{stage}-result.yaml` and sits at its prompt (by design, so it stays steerable) — so across a multi-stage pipeline the carved worker column fills with finished panes and the panes the user actually watches shrink with every completed stage. The pipeline calls it at the one deterministic moment that already exists: right after the orchestrator reads a `done` result (`_preamble.md` § CLI-Adapter Dispatch step 3).

**No flags.** No `--json`, and no `--server`: the socket comes from the record, so a `--server`-started dispatch is reaped on the right socket with nothing extra passed.

It kills the pane **only when all three** hold — and owns the whole guard itself, which is why the skill-side call is unconditional and dumb:

1. the record is **pane-mode** (a `pane:`-bearing record), **and**
2. the **derived state is `done`** (`{stage}-result.yaml` present — pane liveness is irrelevant to the state), **and**
3. **`dispatch.reap_done` resolves `true`** (default `true`) through the four-layer config cascade (environment > project > system > built-in defaults) — the reason the policy check lives in Go: a skill reading `fab/project/config.yaml` directly would miss the higher-precedence environment layer and the machine-wide system layer `~/.fab-kit/config.yaml`, which for a `both`-scope preference is exactly where it is usually set.

Every other case is a **no-op that names its reason and exits 0**:

| Case | Behavior |
|------|----------|
| record is **headless** | no-op — the worker process already exited; there is nothing visual to reclaim |
| state is **not `done`** (`running` / `orphaned`, or any headless state) | no-op — reap is NOT kill; it must never terminate a live or failed dispatch. The report points at `fab dispatch kill` for that |
| **`dispatch.reap_done: false`** | no-op — the user opted to keep done-worker panes and their scrollback |
| **pane already gone** (killed by hand, tmux server died) | benign already-gone report — mirrors `kill`'s idempotence, so a re-reap is safe |
| all three hold | `tmux kill-pane -t <pane>` on the record's own socket; reports `reaped pane %N for <change>/<stage>` |

**Exit codes**: 0 for the reap and for every no-op above; non-zero **only** for real errors — no dispatch record for the pair, or an unresolvable change — sharing `status`/`wait`'s message surface (`no dispatch for <change>/<stage> (run \`fab dispatch start\` first)`). An **unreadable `fab/project/config.yaml` is deliberately NOT in that set**: the knob is resolved only once conditions 1–2 already hold, so a headless or not-`done` no-op reads no config at all, and where it *is* read a config that will not parse **warns on stderr and falls back to the built-in default** (`true`) — the same value an absent key resolves to. The wiring calls reap unconditionally after every `done`, so a broken config must not turn pane hygiene into a pipeline failure.

**Reap kills the pane only — it cleans no state.** The record (`{stage}.yaml`), the result (`{stage}-result.yaml`), the prompt file, and the log all remain. That is exactly why a reaped dispatch still reads **`done`** forever (`DerivePaneState` gives result presence precedence over pane liveness) and why reap is pane *hygiene*, not state cleanup: the no-automatic-GC posture — archive-time deletion plus explicit `fab dispatch clean` as the only two cleanup moments (§ clean) — is untouched. It is also **shape-blind**, like `kill`: killing a split worker's pane leaves the dispatching agent's pane, its window, and any sibling worker intact, while killing the only pane of a new-window worker takes the window with it. A `restart` after a reaped attempt needs no special handling — last-attempt-only overwrite already covers a completed prior attempt.

### clean — `fab dispatch clean [<change>] [--orphans]`

Manual cleanup — one of exactly **two** cleanup paths (the other is archive-time deletion; there is **no automatic GC** anywhere):

- `fab dispatch clean <change>` — removes `.fab-dispatch/{id}/` for the named change.
- `fab dispatch clean` (no arg) — removes all `.fab-dispatch/*/` dirs.
- `fab dispatch clean --orphans` — prunes any `.fab-dispatch/{id}/` whose ID no longer resolves to a non-archived change (covers a change archived/deleted upstream leaving a local state dir orphaned).

`clean` is **mode-blind** — it removes state dirs and never inspects a record's mode, so a pane dispatch's dir (prompt file included) is cleaned exactly like a headless one's. As with a live headless process, cleaning a **live** pane dispatch removes the state without killing the worker; `kill` is the verb for that.

---

## fab doctor

Prerequisite check. Lives in `fab-kit` so it works before `config.yaml` exists; used as `/fab-setup` Phase 0 gate.

```
fab doctor [--porcelain]
```

**Checks** (7): git, fab, bash, yq (v4+), jq, gh, direnv (with zsh/bash hook detection).

**Output**: `  ✓ {tool} {version}` (pass) / `  ✗ {tool} — not found` + install hint (fail) / summary line. When `FAB_KIT_PATH` is non-empty, normal output also includes `kit: <absolute-dir> (FAB_KIT_PATH override)` as informational provenance; it is not an eighth check and does not affect the failure count or exit code. The line exposes the configured path even when reader commands would reject it as invalid.

`--porcelain`: errors only (no passes/hints/summary/provenance). Exit code still = failure count. Empty stdout + exit 0 = all good.

---

## fab migrations-status

Migration discovery. Lives in `fab-kit` (registered in the router's `fabKitArgs` allowlist). Resolves `fab/.kit-migration-version` (local) and the engine `VERSION` from `FAB_KIT_PATH` when set, otherwise from the cached kit for the pinned version (`fab/.fab-version`, the sole version source), scans that kit's `migrations/` dir, and runs the discovery algorithm. The override is absolutized and must name an existing directory; invalid values fail loudly with no cache fallback. Consumed by both `/fab-setup migrations` (via `--json`) and as a standalone query.

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

Prints the absolute resolved kit directory. A non-empty `FAB_KIT_PATH` wins over the normal exe-sibling `kit/` next to `fab-go`; relative values are absolutized and a missing/non-directory value errors loudly naming the variable, with no exe-sibling fallback. No trailing newline or decoration. Exit 0 on success; non-zero with stderr error on failure. Used by skills to reference kit content: `$(fab kit-path)/templates/`, `$(fab kit-path)/migrations/`, etc. The same per-process variable is honored by fab-kit's reader paths (`sync`, `migrations-status`), so every kit-content reader follows one source; binary resolution remains version-pinned and unchanged.

---

## fab shell-init

```
fab shell-init <bash|zsh|fish>
```

Emits the shell-completion script for the given shell on stdout — the `tu`-style verb equivalent of (and delegated to) Cobra's auto-generated `fab completion <shell>`. Recommended install: add `eval "$(fab shell-init zsh)"` to `~/.zshrc` (or the bash/fish equivalent). Config-independent — works outside a fab repo. Human-setup-facing; no skill invokes it.

---

## fab skill

```
fab skill
```

Prints the fab **agent skill bundle** — a one-page, static, agent-first usage briefing (when to reach for fab, a capabilities map keyed to subcommands, composition patterns, the stdout/exit-code contracts, gotchas) to stdout as **raw markdown, byte-identical** to the repo's canonical `docs/site/skill.md`. **stderr empty on success, exit 0**, no rendering/pager/framing (an agent consumes the bytes directly). Takes no args/flags (`cobra.NoArgs`); an argued invocation is a usage error → exit `2` via the binary-wide `run()` classification. Config-independent — works outside a fab repo.

This is the shll toolkit-wide `skill` standard (contract: shll `docs/site/standards/skill.md`; sibling of the machine-readable `help-dump` contract). The bundle is embedded into the `fab-go` binary at build time (fab-go's first `go:embed`), so it is offline and version-locked to the release — the sync + drift-guard pattern `shll standards` established: a committed copy at `src/go/fab/cmd/fab/skill.md`, refreshed from the canonical `docs/site/skill.md` by `scripts/sync-skill.sh` (`//go:generate` pointer), pinned byte-honest by `TestSkillEmbedMatchesCanonical`. Because `docs/site/**` is the pulled site surface, the same page renders at `shll.ai/tools/fab-kit/skill` for free.

**Disambiguation**: `fab skill` (this one static bundle command) is unrelated to fab's own **kit-skills** — the many `/fab-*` markdown prompts `fab sync` deploys to `.claude/skills/`. Same word, two concepts.

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

Consumers: `fab pr-meta` (which renders the PR body `**Impact**` line via the same `internal/impact` package) and the apply-finish, hydrate-finish, and ship-finish hooks (write the result into `.status.yaml` `true_impact`; ship-finish is the authoritative write in the standard pipeline — the earlier writes see `HEAD == merge-base` until commits exist). `/git-pr` delegates the whole `## Meta` block to `fab pr-meta`.

---

## fab pr-meta

```
fab pr-meta <change> --type <type> [--issues "DEV-1 DEV-2"]
```

Renders the complete, byte-stable `## Meta` block of a fab-generated PR as final markdown on stdout.

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
- Impact: one normalized `Impact | +/− | Net` table (right-aligned numeric columns, Net retained) followed by a `<sub>` provenance caption; there is no `**Impact**:` lead-in. The table drops rows but never reshapes:

  | Row | Shown when | Contract |
  |-----|------------|----------|
  | `raw` | Excludes are configured | `raw = true + excluded`, even when its values equal `true` |
  | `**true**` | Always when the block exists | Post-exclude diff; `true = impl + tests` |
  | `└ impl` | A tests pair exists | Per-component `max(0, true − tests)` residual; Unicode minus `−`, clamp-annotated when net-negative |
  | `└ tests` | A tests pair exists | Test-path component |
  | `excluded` | Excludes are configured | Excluded-path component; makes `raw = true + excluded` checkable |

  The caption is `<sub>excludes \`…\` · generated by fab-kit vX.Y.Z</sub>` with actual `true_impact_exclude` values individually backtick-wrapped; omit the excludes clause when none apply. Version comes from the running binary (`fab-kit vdev` for a dev build). Omit the whole block for `+0/−0` `true`, missing merge-base, or impact failure. Only **bold** emphasizes rows; `<sub>` is GitHub-allowed HTML.
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
- **Freeze-on-write `log.md` (FKF §6.4).** Existing entries are authoritative and immutable;
  regeneration parses them and appends newly discovered attributable entries only. Dedup uses
  `(file-base, change-id)`, not commit hash; existing unattributable entries stay verbatim and new
  unattributable commits are not projected after first write. First generation is the same append
  path over an empty log, not a separate mode, and `log.seed.md` still merges beneath projection.
  The rationale and full grammar are canonical in `$(fab kit-path)/reference/fkf.md` §6.4.
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
- `⚠ docs/memory/<domain>/<sub>/<deep> is nested <N> levels deep (max: 3) — consider flattening`
  when nesting exceeds 3 levels under `docs/memory/`.
- Reserved domains **`_shared/`** and **`_unsorted/`** are **exempt** from the width warning.
- Warnings are advisory: they never block, never modify files, and never affect the byte-stable
  index output (so a regen-with-warnings is still idempotent).

Content findings are gathered once and printed to stderr on write and `--check`; they never alter byte-stable rendered output.

| Severity | Marker | Fires when | Affects `--check` exit |
|----------|--------|------------|------------------------|
| BLOCKING `✖` | `✖ … has malformed frontmatter — unclosed frontmatter block (no closing \`---\`)` | Line 1 opens `---` with no later standalone close, including glued-fence corruption | Floors at 1 independent of drift |
| BLOCKING `✖` | `✖ … has malformed frontmatter — \`description:\` value fails quote-stripping (unterminated quote): <value>` | A quoted description lacks its matching closing quote | Floors at 1 |
| BLOCKING `✖` | `✖ … \`description:\` carries a change-id (registry match: <id>) — descriptions are routing signals; move citations to the body (FKF §3.2)` | Full folder token or bare 4-char ID matches the change registry; unregistered words never match | Floors at 1 |
| BLOCKING `✖` | `✖ … has a <N>-character \`description:\` (blocking cap: 1000, soft cap: 500) — trim to a one-liner; detail belongs in the file body` | Description exceeds 1000 runes | Floors at 1 |
| ADVISORY `⚠` | `⚠ … has a <N>-character \`description:\` (soft cap: 500) — trim to a one-liner; detail belongs in the file body` | Description is 501–1000 runes; JSON kind `description-length`, `count` = runes | Never |
| ADVISORY `⚠` | `⚠ … has <N> narration markers (threshold: 5) — distillation debt; consider /docs-distill-memory` | Topic body has ≥5 transition stems or registry IDs outside trailing citations / `*Introduced by*:` | Never |
| ADVISORY `⚠` | `⚠ … is <N> lines / <K>KB (soft cap: ~400 lines / ~15KB) — consider splitting; see /docs-reorg-memory` | Topic file exceeds 400 lines or 15KB | Never |
| ADVISORY `⚠` | `⚠ docs/memory/_unsorted holds <N> staged file(s) — triage into domains (staging should trend to empty)` | `_unsorted/` contains at least one topic file | Never |
| ADVISORY `⚠` | `⚠ … links to <target> — target does not exist` | Topic-file bundle-relative link resolves missing under `docs/memory/`; fenced and inline code are skipped | Never |

Frontmatter findings and the description-length advisory run on topic files and domain/sub-domain `index.md` stubs. Narration, size, and broken-link meters run only on topic files (`index.md` / `log.md` / `log.seed.md` excluded); `_unsorted` is per-folder.

Flags:
- `--check` — write nothing; byte-compare every rendered index and `log.md`, classify index-only destructive loss, and apply the independent blocking floor from the severity table. Advisory findings never affect exit. `log.md` drift is always benign.
- `--json` (with `--check`) — emit the loss report as a single JSON object on **stdout** and
  suppress the human-readable text; the exit code is unchanged. Mirrors the `fab pane` /
  `fab migrations-status` `--json` convention (snake_case). Shape:
  `{"tier": 0|1|2, "drift": bool, "losses": [{"category": "description"|"tombstone"|"grouping", "path": "<repo-rel index>", "detail": "<lost text | dropped link target | flattened heading>"}], "malformed": [{"kind": "malformed-fence"|"malformed-description"|"description-change-id"|"description-over-cap", "path": "<repo-rel file>", "detail": "<offending value | matched change-id, omitted for fence/over-cap>"}], "warnings": [{"kind": "description-length"|"narration-density"|"file-size"|"unsorted-nonempty"|"broken-link", "path": "<repo-rel file/folder>", "count": <N — rune length for description-length; line count for file-size; marker count for narration-density; staged-file count for unsorted-nonempty>, "bytes": <N — file-size findings only, omitted otherwise>, "detail": "<broken link target, omitted otherwise>"}]}`.
  The `malformed` array (blocking findings) and `warnings` array (advisory findings) are **additive**:
  `/docs-reorg-memory` compatibility detection continues to branch on `tier` and read `losses`, and
  the five advisory kinds share one `warnings[]` array. `losses`, `malformed`, and `warnings` are always
  present (empty arrays, never `null`).
- `--rebuild` — **DESTRUCTIVE** FKF §6.4 escape hatch: discard frozen `log.md` state and re-project from git, including unattributable commits; use only for corruption or deliberate re-baseline. Seed merge still applies. `--check` ignores it and compares the non-destructive merge. The re-baseline migration probes `fab memory-index --help` before running `fab memory-index --rebuild`; `$(fab kit-path)/reference/fkf.md` owns the rationale.

Tiered `--check` exit codes (loss is a strict subset of drift):

| Exit | Tier | Fires when | Consumer action |
|------|------|------------|-----------------|
| `0` | Clean | Every index and `log.md` matches regeneration and no blocking finding exists | No regeneration |
| `1` | Benign drift / blocking floor | Regeneration changes but destroys nothing (including every `log.md` and root `fkf_version` drift), or a blocking content finding exists. Freeze-on-write accepts a committed log that is a valid superset; failure means a projected attributable pair is missing or a frozen line is render-unstable | CI/pre-commit fails on ≥1; fix the source file for blocking findings |
| `2` | Destructive loss | Index-only: curated description would become `—`; tombstone target is absent (external/absolute excluded); or root custom grouping would flatten. `log.md` never reaches tier 2 | Enumerate losses and print `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.` Hydrate/reorg refuse only on `== 2` |

> **Precedence:** exit-2 destructive loss beats the independent blocking floor; blocking findings are still enumerated, do not extend `losses[]`, and never fire the exit-2 refuse guards. A born-compatible tree cannot reach tier 2 unless it first acquires legacy structural debt.
>
> **Compatibility:** the `--json` key stays `malformed` for the four blocking kinds; `losses`, `malformed`, and `warnings` remain present as arrays.

Other exit codes:
- non-zero (1) — an operational error: `docs/memory/` not found (or another `Gather` failure), or a
  write failed. `Gather` runs before the `--check` branch, so a `--check` run also exits 1 on these —
  the exit-1 / exit-2 *tier* codes above apply only once gather succeeds and the comparison runs.
  Writes happen only on non-`--check` runs, so a write failure is non-`--check`-only.

**Usage-error coexistence**: the `--check` tier-`2` (destructive loss) is an in-handler `os.Exit(2)` that bypasses the binary-wide usage/operational mapping (§ Exit-Code Convention). A *usage* error on `memory-index` — a bad flag or arg-count violation — instead exits `2` at parse time, before the handler runs. Both use code `2`, so it is ambiguous between "usage error" (parse-time) and "destructive loss" (in-handler `--check`) — disambiguate on stderr; the tiered `--check` scheme is not renumbered, and the hydrate guard's tier-2 branch is unaffected (it only ever runs `--check`, which reaches the handler).

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

**Hidden, machine-consumer command** (invoked by shll.ai's puller on a schedule). Marked `Hidden: true`, so it does not appear in `fab --help` and is excluded from its own dumped tree. Takes no arguments. Walks the live cobra command tree of the rich `fab` CLI programmatically (not by regex-parsing `-h` text) and writes the frozen shll.ai "command reference" contract JSON to stdout.

Contract shape (`schema_version: 1`):

```json
{
  "tool": "fab",
  "version": "<main.version, from ldflags>",
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

The envelope is exactly `{tool, version, schema_version, root}`. Per the toolkit help-dump standard it carries **no `captured_at`** — the capture timestamp is owned by shll.ai (a tool cannot know its own capture time; the puller stamps it after capture).

`tool` is the literal `"fab"` (the user-facing binary), which differs from the repo/site slug `fab-kit`. shll.ai's puller invokes `fab help-dump` on a schedule and renders the result as fab-kit's command reference; fab-kit pushes nothing.

---

## fab operator

```
fab operator [--workers <provider>]
```

Singleton tmux-tab launcher for `/fab-operator`. Requires `$TMUX` (else exit 1, `ERROR: not inside a tmux session`). The singleton check is an **exact, server-wide** window-name match: `tmux list-windows -a` enumerated and compared exactly (never tmux target resolution, whose prefix/glob fallback would let e.g. `operator-logs` mask the real check; `-a` enforces the one-operator-per-SERVER invariant across sessions). If a window named exactly `operator` exists anywhere on the server → select it by window ID, switching the client to its session when needed (`Switched to existing operator tab.`); else create the window running `{operator-session-command} '/fab-operator'` (`Launched operator.`).

**`--workers <provider>`** is launch sugar for the new-window path: the shell command is prefixed with the safely quoted assignment `FAB_AGENT_WORKERS='<provider>'`. The value passes through verbatim with no provider lookup or validation. When the singleton already exists, selection is unchanged and no new environment can be injected.

**Launch cwd (no git-repo dependency)**: the new window's working directory (`tmux new-window -c <dir>`) is resolved by trying `git rev-parse --show-toplevel` first and falling back to `os.Getwd()` when that fails. The operator launches **inside a git repo** (cwd = repo root) **or from a neutral parent directory** (cwd = current directory), and errors only if both resolutions fail. This matches the per-tmux-server, cross-repo singleton model: the operator's natural launch point is a neutral dir with no `fab/` project.

**Session command resolution (no `fab/`-project dependency) + operator-role profile**: the operator resolves the **operator role** in-process (`agent.ResolveRole(cfg, "operator")`) → its provider (the `agent.session` knob, since `operator` is a Tier-1 role) → that provider's `session_command`, then injects the role's `{model, effort}` via `spawn.WithProfile`. When a `fab/` project is resolvable (`resolve.FabRoot()` succeeds) the config supplies the knob + provider; when `resolve.FabRoot()` **fails** — the operator is launched from a neutral directory with no `fab/` project anywhere up the tree (its natural cross-repo home) — this is **non-fatal**: `config.Load` returns an empty config, so `ResolveRole`/`ResolveProvider` degrade to fab-kit's built-in operator profile (`claude-sonnet-5`/`medium`) + built-in claude provider (`spawn.DefaultSpawnCommand`). `WithProfile` is grammar-forgiving: for a **template** `session_command` containing `{model}`/`{effort}` — including the built-in claude default, which is templated — it **substitutes** the resolved values in place (all-or-nothing; an empty value drops the placeholder's token and a preceding `-`-flag); for a command carrying **no placeholder** (e.g. a user's plain-form config carried forward by the 2.13.0 migration) it instead **appends** `--model <model> --effort <effort>` to the END (last-wins; each flag omitted when its value is empty, per the `empty ⇒ omit` convention). A provider without a `session_command` falls back to `spawn.DefaultSpawnCommand` (the templated claude default, still profile-substituted). So a `fab/`-less launch composes a fully-defaulted command: default session command + the built-in operator `{model, effort}` (byte-identical whether resolved by substitution or, for a plain user command, by append).

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
fab agent [role] [--provider <name> [--model <id>] [--effort <level>]] [--workers <provider>] [--print] [--repo <path>]
```

Launch (or `--print`) the profile-resolved agent **session** command in the current shell, with model and effort substituted.

Two **mutually exclusive addressing modes** compose the command:

- **Role-addressed** (the `[role]` positional) — resolves the role profile (`default` when the positional is omitted; any of the six role names accepted: `default`, `operator`, `doing`, `review`, `hydrate`, `fast`), then composes `providers.<profile.provider>.session_command` with the role's `{model}`/`{effort}` substituted (or Claude-style `--model`/`--effort` appended for a non-templated command) via `internal/spawn.WithProfile` — the same substitution `fab resolve-agent`'s `dispatch=` line and the operator launcher use. Which provider the role lands on is the role's depth knob (`agent.session` for `default`/`operator`, `agent.workers` for the rest) unless `agent.profiles.<role>.provider` names one.
- **Provider-addressed** (`--provider <name>`) — **bypasses role resolution entirely**: looks up `providers.<name>` directly (project config per-field merged over fab-kit's built-in provider table, exactly as the role path's provider lookup does) and composes its `session_command` with the `--model`/`--effort` values through the same `WithProfile`. This is the "spawn a codex session right here" form — no knob or role need name the provider first, and no `providers:` block either: `codex` and `gemini` are **built-in** providers, so `fab agent --provider codex` works on a fresh project.

Common to both modes:

- **Default (exec)**: replaces this process with the composed command via `sh -c` (so shell expansions like `$(basename "$(pwd)")` expand at invocation). `fab agent` starts the default-role agent right here; `fab agent operator` starts the coordinator profile. **No TTY guard** — exec-and-let-the-agent-CLI-handle-it (document-don't-validate).
- **`--print`**: prints the fully-resolved command instead of executing. Lets the operator compose a worker spawn from a real profile.
- **`--repo <path>`**: reads `<path>/fab/project/config.yaml` instead of the current repo. Composes with either addressing mode.
- **`--workers <provider>`**: sets `FAB_AGENT_WORKERS=<provider>` in the exec environment for the launched session, replacing any value inherited from the parent environment rather than appending a second entry. It is pure pass-through launch sugar: no provider lookup or validation, and `--print` remains exactly the resolved session command with no assignment added.

Provider-mode specifics:

- **Omitted `--model`/`--effort`** leave the value empty, which follows the existing `WithProfile` empty-value rule: in **template** mode the placeholder's whitespace-delimited token is dropped along with a preceding `-`-flag; in **append** mode the flag is simply not appended. So `fab agent --provider codex --print` against `codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` prints `codex --dangerously-bypass-approvals-and-sandbox`: the installed CLI's own default model applies while fixed, non-profile flags remain. This is how you spawn a provider whose current model IDs you do not know.
- **`--model`/`--effort` require `--provider`.** Supplying either without it is a usage error (non-zero exit): on the role path the model and effort ARE the resolved role profile's, so a bare `--model` would either invent an undocumented role-override surface or be silently ignored. **Deliberate asymmetry with `fab resolve-agent`**, where a bare `--model`/`--effort` IS valid — that command is a pure query whose whole output is a profile, so overriding one printed field is unambiguous; this one is a session launcher with two mutually exclusive addressing modes. See § fab resolve-agent.
- **Omitted `--model` on a provider that has fills**: `fab agent --provider <name>` does NOT read `providers.<name>.profiles` — the provider mode's profile is exactly the flags. The fill rungs apply to *role* resolution and to `fab resolve-agent` (see its § Fill precedence); here an omitted value stays empty and the CLI's own default applies.
- **`--provider` and the `[role]` positional are mutually exclusive** — supplying both is a usage error (non-zero exit) naming the exclusion; a role already resolves a provider, so mixing the two has no coherent semantics.
- **Unknown provider name** → non-zero exit listing the **available** provider names (the project's `providers:` keys ∪ fab-kit's built-in table, sorted). This is a **lookup** failure, not validation of the command's content — resolved command strings still pass through verbatim (document-don't-validate; fab never infers a provider from a model string).
- **Error**: a resolved provider with no `session_command` errors with a config-key hint (`configure providers.<name>.session_command`) on either path — reachable only for a provider name that has **no built-in row at all** (a wholly user-defined, dispatch-only `providers:` entry). Naming one of the three built-ins can never reach it: `ResolveProvider` per-field merges over the built-in row, so `providers.codex: {dispatch_command: …}` still inherits the built-in `session_command` and still launches a session. An unknown role name errors and names the valid set.

The procedural knowledge for *using* the composed command — opening it in a tmux window, delivering a prompt reliably, peeking, awaiting — plus the per-provider invocation grammar and model-discovery recipes live in the `_cli-agents` helper (`helpers: [_cli-agents]`).

---

## fab batch

Multi-target operations: `fab batch <new|switch|archive> [flags] [targets...]`.

| Subcommand | Flags / default | Operation | Guards and failure behavior |
|------------|-----------------|-----------|-----------------------------|
| `new` | `[--list] [--all] [--workers <provider>] [ids...]`; no args ⇒ `--list`; no `--quiet` | Parse pending `- [ ] [xxxx]` backlog items (including continuation lines), create one worktree/window per ID, run `/fab-new {description}`; `--all` selects all pending | Launch path requires `$TMUX`, then `wt` (`brew install sahil87/tap/wt`); list path requires neither. Empty `--all` errors `ERROR: No pending backlog items found.` Unknown IDs and backlog items with empty content warn-and-skip (exit 0). Per-item `wt`/tmux failures include child stderr, continue, then exit non-zero with `ERROR: {N} of {M} item(s) failed to launch` |
| `switch` | `[--list] [--all] [--quiet\|-q] [--workers <provider>] [changes...]`; no args ⇒ `--list` | Resolve active changes in-process, create branch worktrees, run `/fab-switch {change}`; `--all` excludes `archive/` | Same launch guards; empty set errors `ERROR: No changes found.` Resolver and `wt` errors warn-and-skip with specific/child stderr. Quiet suppresses progress but not list/data output or stderr |
| `archive` | `[--yes\|-y] [--dry-run] [--quiet\|-q] [changes...]` | Archive `hydrate: done\|skipped` changes in-process via `ArchiveWithBacklog`; no agent, tmux, `wt`, or fab-on-PATH dependency | Uses the consent matrix below. `--dry-run --yes` is mutually exclusive. Quiet suppresses progress only, never consent, data, stderr, or footer |

`new` uses `wt create --non-interactive --worktree-name {id}`, window `fab-{id}`, and `{worker-session-command} '/fab-new {description}'`. Both launchers compose the default-role provider `session_command` through `internal/spawn`, substituting templated `{model}`/`{effort}` or appending them for a plain command; no placeholders reach tmux. Missing `wt` exits 1 after the tmux guard with `ERROR: wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt` (or `'fab batch switch'`); missing tmux is `ERROR: not inside a tmux session`.

On `new` and `switch`, **`--workers <provider>`** safely prefixes every tmux shell command with `FAB_AGENT_WORKERS='<provider>'`. Embedded single quotes are shell-escaped; the value is otherwise passed through without provider validation. Omitting the flag leaves the launch command byte-for-byte unchanged.

`switch` names branches `{branch_prefix}{folder_name}`. It probes local `git show-ref --verify --quiet refs/heads/<b>`, then `git ls-remote --heads origin <b>`: existing branches use `--checkout <branch>`, new branches use the positional, and an offline remote probe degrades to the positional so `wt` re-checks. Quiet leaves successful stdout empty because tmux creation is the result; list output and stderr remain.

Archive consent matrix:
  - **bare invocation (interactive stdin)** → lists the archivable set, then prompts `Archive these N? [y/N]` with **default No** — a bare Enter or any non-`y`/`yes` (case-insensitive) answer aborts (exit 0, nothing archived); `y`/`yes` archives all.
  - **`--yes` / `-y`** → archives all archivable changes with no prompt (the non-interactive escape hatch).
  - **`--dry-run`** → lists what would be archived; no prompt, no action.
  - **non-TTY stdin without `--yes`** → refuses rather than hangs: returns a single multi-line error so `main()`'s centralized printer emits it once as `ERROR: refusing to prompt for confirmation on a non-interactive stdin.` followed by `Re-run with --yes to archive non-interactively` on stderr, then exits non-zero (the handler does not print its own `ERROR:` lines, avoiding a doubled prefix). This matters because the tmux/operator runtime is frequently non-interactive — those call sites pass `--yes`.
  - **explicit args** (`fab batch archive foo bar`) → archive the named changes with **no prompt and no TTY guard** (naming them IS the opt-in; the prompt applies only to the bare/archive-all path).
  - **`--dry-run --yes`** → mutually exclusive → exits non-zero (`ERROR: --dry-run and --yes are mutually exclusive`).
  - **`--quiet` / `-q`** → suppresses the `Archiving N changes...` preamble and every per-change loop line while keeping the `Archived N, skipped N, failed N.` footer, all stderr, the empty-set no-op output, and the `--dry-run` listing. It is **orthogonal to consent**: it does not imply `--yes` and introduces no new mutual-exclusion rule — the bare-invocation listing + `[y/N]` prompt (consent, not progress) still print under `--quiet`. `--quiet --yes` (footer-only) is the expected non-interactive agent invocation.

  Per change prints `{name} — archived` (with ` (backlog marked done)` when applicable; when a post-archive step — index update or backlog mark — fails, the change still prints `archived` plus a stderr `warning:` line and counts as archived, not failed), `already archived, skipping` (covers genuinely-archived names — counted as skipped), or `FAILED: {err}`; a single failure never aborts the batch. Under `--quiet` these per-change stdout lines are suppressed (the `FAILED:`/`warning:` stderr lines are not). Footer: `Archived {N}, skipped {M}, failed {K}.` (always printed, including under `--quiet`). Exit semantics: an empty archivable set (bare or `--yes`) is a benign no-op (`No archivable changes found.` + zero footer, exit 0) checked **before** any prompt or non-TTY guard (finding F49); after the loop runs, non-zero when `failed > 0` (`ERROR: {K} change(s) failed to archive`); explicitly named targets where none resolves to an active *or* archived change → exit 1, `ERROR: No valid changes to archive.`.

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
