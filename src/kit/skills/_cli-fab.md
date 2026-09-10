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
- fab doctor
- fab migrations-status
- fab kit-path
- fab setup
- fab shell-init
- fab skill
- fab impact
- fab pr-meta
- fab docs-index
- fab fab-help
- fab help-dump
- fab batch
- Common Error Messages

The `fab pane` and `fab dispatch` families live in `_cli-fab-pane.md`; `fab operator` and `fab agent` in `_cli-fab-operator.md`.

---

## Calling Convention

`fab <command> <subcommand> [args...]`. `fab` is a router dispatching workspace commands (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`) to `fab-kit` and everything else to the per-version `fab-go` binary resolved from the pinned version in `fab/.fab-version` (a one-line plain-text sibling of `fab/.kit-migration-version`; the sole version source — a stale `fab_version:` key left in `fab/project/config.yaml` is no longer read). `--version`/`-v`/`--help`/`-h`/`help` are handled inline. `fab-go` auto-fetches from GitHub releases on cache miss.

`fab -h` composes help from both binaries. `fab --version` prints the system binary version; inside a fab repo a second line shows the project-pinned version.

### Exit-Code Convention (`fab-go` commands)

The `fab-go` binary (everything the router does not dispatch to `fab-kit`) follows the toolkit convention: **`0` success / `1` operational failure / `2` usage error**. A **usage error** is a malformed invocation caught at parse/validation time — an unknown/malformed flag (`fab score --nope`), an arg-count violation (`fab score` with no args), an unknown subcommand (`fab nonsense`), or a mutually-exclusive flags-group conflict (`fab resolve --status --folder`). An **operational failure** is a syntactically valid invocation that fails on a runtime/data condition (a missing change, a failed preflight, a below-gate `--check-gate`, a tmux/gh/filesystem error). Success is `0`.

Classification follows execution phase, never message text: cobra failures before `RunE` exit `2`; errors from inside `RunE` exit `1`. The testable `run()` helper records whether execution began, and no path inspects stderr to classify.

**Coexistence with in-handler domain schemes (no renumbering)**: the pane family (`2` = pane missing, `3` = other tmux failure) and `fab docs-index --check` (`0`/`1`/`2`, `2` = destructive loss) set their non-1 codes via `os.Exit` *inside* the handler, which bypasses `main()`'s usage/operational mapping entirely — their codes are unchanged. For those subcommands exit `2` is therefore intentionally ambiguous between "usage error" (at parse time) and the domain meaning (in-handler); this is documented per subcommand, which principle №4 sanctions, rather than renumbered (renumbering would break the pinned pane test and downstream consumers). A usage error is a static caller bug fixable at authoring time, not a runtime condition scripts branch on, and stderr wording disambiguates.

### Workspace Command Exit Semantics

Lifecycle commands fail loudly — a non-zero exit is the failure signal scripts and skills rely on. **Exception**: `sync` and `fab-kit migrations-status` reserve a distinct exit `3` for the "not a fab-managed repo" precondition (see the `sync` row and the `fab migrations-status` section) — that is *not* a failure but a "not applicable here" signal, so a caller branching on the exit code MUST treat exit `3` separately from the generic exit `1` = failure. All other lifecycle failures use the generic non-zero (exit `1`) path.

| Command | Failure behavior |
|---------|------------------|
| `init` | Refuses to run while `FAB_KIT_PATH` is non-empty, with an unset-first error before repository checks, downloads, config writes, or version stamps. Otherwise requires a git repository — exits non-zero with `fab init requires a git repository — run 'git init' first` BEFORE any download or config write. Sync failure during init also exits non-zero |
| `update` | Exits `0` with a degrade message (`fab-kit was not installed via Homebrew` + manual-update guidance) when the binary is not brew-installed (go-install/manual/CI) — not brew's to upgrade, so not a failure (shll update standard: exit non-zero only on genuine failure; keeps a composed `shll update` run honest). Exits non-zero only on genuine brew failure (`brew update`/`brew info`/`brew upgrade`); brew runs unbounded — no timeout, no kill path (the standard forbids SIGKILLing a package manager mid-transaction). Internally `Update()` still returns the `ErrNotBrewInstalled` sentinel — the exit-0 mapping is command-layer only, so `sync`'s version guard keeps treating a too-old non-brew binary as a guard failure |
| `upgrade-repo` | Refuses to run while `FAB_KIT_PATH` is non-empty, before resolution, cache, sync, config, or version-stamp work, because release-stamping arbitrary override content would mix kit provenance. Otherwise runs sync first, then (only AFTER sync succeeds) stamps `fab/.fab-version` and auto-runs `fab config upgrade` against the pinned fab-go to reconcile `config.yaml`'s reference fence (fail-open: a fab-go predating the subcommand prints a reminder and the upgrade continues). On sync failure: exits non-zero with `sync failed: ... — run 'fab sync' to repair, then re-run 'fab upgrade-repo'`, never prints `Updated: x -> y`, stamps nothing, and a re-run retries (no "Already on the latest version" short-circuit of the broken state). **Unaffected by the sync/migrations-status exit-`3` contract**: run outside a fab-managed repo, `upgrade-repo` still exits the generic `1` with the same `not in a fab-managed repo. Run 'fab init' to set one up` stderr message — it deliberately does NOT use `RequireManagedRepo()`/exit `3`, because its guard tolerates a repo with a `config.yaml` but no resolvable pinned version (a partially-managed, not fully-unmanaged, state), a distinct semantic left out of scope. Do not conflate the two: only `sync` and `migrations-status` carry the exit-`3` signal |
| `sync` | Always deploys the portable agents directory; Claude Code and OpenCode targets are gated on their CLIs (`FAB_AGENTS` overrides PATH lookup). A missing Claude CLI prints `Skipping Claude Code: claude not found in PATH` and preserves existing Claude content; the same gate skips Claude settings scaffolding and legacy agent cleanup. When `FAB_KIT_PATH=<dir>` is non-empty, resolves it to an absolute existing directory, prints `kit: <dir> (FAB_KIT_PATH override)`, serves scaffolding/skills from it, and skips both the version guard and `EnsureCached`; invalid overrides fail loudly without cache fallback, while prerequisites, deployment, direnv, project scripts, flags, and failure propagation remain unchanged. With no override, cache behavior is byte-identical. After each fired target's deploy succeeds, sync writes a whole-file-owned generated `.gitignore` inside the target listing exactly the skills it deployed (`/{name}/` or `/{name}.md` entries plus the self-entry); that file doubles as fab's ownership manifest, so stale pruning removes an entry only when the previous manifest recorded it AND the kit no longer ships it — user-added entries are never pruned, and a target with no manifest yet prunes nothing (printing a one-line note when non-kit entries exist). Exits non-zero when any skill deployment write fails (per-skill `WARN:` lines on stderr, `failed N` in the agent tally), when a manifest write fails, or when scaffolding writes fail. The version guard exits non-zero whenever it trips: either `fab-kit was updated to vX — re-run 'fab sync'` (auto-update landed; the current run still ran old code) or actionable too-old instructions (non-brew install, Homebrew tap release lag) — it never continues on a binary older than the pinned version (`fab/.fab-version`, the sole version source). **Branchable exit code** (mirroring the pane family's use of distinct branchable exit codes — though with the opposite polarity: for the pane verbs exit `3` is the real failure and `2` the benign signal, whereas here `3` *is* the benign "not applicable" signal and no exit `2` is involved): run outside a fab-managed repo (no `fab/project/config.yaml` on any ancestor), `sync` prints `not in a fab-managed repo. Run 'fab init' to set one up` to stderr and exits `3` — a distinct "not applicable here" signal, NOT the generic `1` = failure — so callers (e.g. `wt`'s default init, operator scripts) can branch on "not a fab-managed repo" vs. "a real sync failure" without replicating fab's config walk-up. This holds unconditionally, including outside any git repository: the managed-repo check is a `config.yaml` walk-up gated before the git-root resolution, so `sync` is symmetric with `fab-kit migrations-status` (which has no git precondition). The value is the `internal.ExitNotManaged` constant, shared with `fab-kit migrations-status` (below) via `RequireManagedRepo()`. Genuine sync failures above stay exit `1` |

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

`fab preflight`, `fab score`, `fab log command`, `fab change`, `fab resolve`, `fab status` — headline coverage lives there. Sections below document the remaining commands (`fab doctor`, `fab migrations-status`, `fab kit-path`, `fab setup`, `fab shell-init`, `fab skill`, `fab impact`, `fab pr-meta`, `fab docs-index`, `fab fab-help`, `fab help-dump`, `fab batch`) and extended flag details for the above.

---

## fab change (extended subcommand details)

See `_preamble.md` § Common fab Commands for the headline. Full subcommand table:

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `new` | `new --slug <slug> [--change-id <4char>] [--log-args <desc>]` | Create new change — `<slug>` must be 2-6 lowercase kebab-case words |
| `rename` | `rename --folder <current-folder> --slug <new-slug>` | Rename slug (prefix immutable) — `<new-slug>` must be 2-6 lowercase kebab-case words |
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
| `refresh` | `refresh [<change>]` | Recompute the artifact-derived fields — `change_type` + `confidence` (from `intake.md`) and `plan.generated`/`task_count`/`acceptance_count`/`acceptance_completed` (from `plan.md`) — from on-disk artifacts, under the status flock (single load-mutate-save). The change argument is optional: omitted, it resolves the active change via the `.fab-status.yaml` symlink (same resolution as bare `fab preflight`); with no active change the resolution errors non-zero. The pull-based successor to the removed `artifact-write` hook: heals a hook-bypassing edit (sed, direct write) or a non-Claude agent's artifact write. Respects `change_type_source: explicit` (keeps an explicitly-set type). A missing artifact is a safe no-op; exits non-zero only on a genuine failure (unresolvable change, unreadable `.status.yaml`). Self-healed automatically at `advance`/`finish`/`preflight`, so skills need not call it directly |
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

There are **no hard-fail short-circuits** — no `Unresolved → 0.0` and no `R<25 ∧ A<25` Critical Rule. Blocking is emergent from the curve: a `composite < 20` row penalizes ≥ 2.0, which alone drops a change to the 3.0 gate or below. Reversibility is carried by its 0.30 weight in the composite (low-R decisions land in a worse band and are penalized harder), not by a separate rule. There is **no coverage factor and no minimum-decision requirement** — a thin-but-strong intake (two well-resolved decisions) genuinely scores 5.0; quality is measured per decision, so row count is not a proxy for it. The grade (Certain/Confident/Tentative/Unresolved) is **derived from the composite** (bands 80/50/20) and is indicative only — never read by the formula. Range: 0.0 to 5.0. `expected_min` is documentation-only and not part of the score path.

### Template

The `status.yaml` template (in the kit cache at `$(fab kit-path)/templates/status.yaml`) includes the confidence block initialized to zero counts and score 0.0. `/fab-new` and `/fab-draft` write the intake score after intake generation (both via the shared `_intake` Step 7); `/fab-clarify` re-writes it after resolving intake assumptions.

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

> **Deprecated for skill dispatch.** Pipeline skills consume `fab agent <stage|role> -o yaml` (`_cli-fab-operator.md` § fab agent), which exposes the same shared resolution as structured data. `fab resolve-agent` remains working with its frozen line protocol for out-of-band callers and muscle memory; retirement is a later post-window change.

Pure query (no side effects) — resolves a pipeline **stage** (or an agent **role** name) to its `{provider, model, effort}` agent profile and renders the legacy line projection. `fab resolve-agent` and `fab agent -o yaml` render projections of the same shared resolution engine.

```
fab resolve-agent <stage|role> [--alias] [--provider <name>] [--model <id>] [--effort <level>]
```

The positional argument is either one of the six pipeline stages (`intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`) or one of the six role names (`default`, `operator`, `doing`, `review`, `hydrate`, `fast`). A role name is accepted positionally alongside a stage name: a stage maps through the fixed stage→role mapping; a role resolves directly. The two name sets overlap only at **fixed points** — a name shared by a stage and a role (`review`, `hydrate`) is one where the stage maps to that same-named role (`stageRoles[name] == name`), so role-first dispatch resolves such a name identically either way. (`ship` is a stage but not a role — it maps to the `fast` role.)

**Vocabulary**: a **role** is one of the six fixed slot names; a **profile** is a concrete `{provider, model, effort}` value; a **provider** carries independent `interactive_command`, `headless_command`, and `native` launch capabilities plus its per-role fills — capability presence says how, while `dispatch.mode` selects through the descending `pane → native → headless` ladder; **Tier 1 / Tier 2** is agent *depth* (what you talk to vs. what stages dispatch to), which is what the two `agent.session` / `agent.workers` knobs select a provider by.

**Runs OUTSIDE a fab project** — config-only, same degradation as `fab agent` (env > system > built-in defaults with no project tier). See `_cli-fab-operator.md` § fab agent.

**Resolution**: maps a stage → its role via the FIXED fab-owned stage→role mapping (`default`: intake (advisory) / `doing`: apply, review-pr / `review`: review / `hydrate`: hydrate / `fast`: ship — NOT user-overridable), then resolves the role → `{provider, model, effort}` through the fill precedence below. The built-in fills themselves are NOT restated here — they move at kit-release cadence; read one structured resolution with `fab agent <stage|role> -o yaml`, the legacy line projection with `fab resolve-agent <stage|role>`, or every provider's whole map with `fab config explain providers --json`. The stage→role mapping and the role→depth partition are both fab-owned (no `stage_roles`, no per-stage escape hatch); the three launch capabilities and per-role fills live in the top-level `providers:` table, while `dispatch.mode` owns adapter preference.

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
- `dispatch=` is derived from the resolved `dispatch.mode`, the provider's three independent capabilities, and `$TMUX`. Selection starts at the configured preference and descends only through `pane → native → headless`: pane requires `$TMUX` plus `interactive_command`, native requires `native: true`, and headless requires `headless_command`. The line is **absent iff native resolves**; pane emits the profile-substituted `interactive_command`, and headless emits the profile-substituted `headless_command`. Legacy line-protocol consumers branch on line presence and never execute its value; pipeline skills use the equivalent `dispatch:` key-presence rule from `_cli-fab-operator.md` § fab agent instead. A provider with no reachable rung is a non-zero actionable error naming the provider and capability keys; an explicit unknown `--provider` retains the earlier lookup error.
- **`dispatch.mode` (preference ceiling)** — a `pane | native | headless` config field, **default `native`**, **scope `both`** (settable once machine-wide in `~/.fab-kit/config.yaml`, where it outranks the project file). `pane` means “prefer watchable”: inside tmux a provider with `interactive_command` resolves pane; outside tmux it descends to native when `native: true`, otherwise headless when `headless_command` exists. `native` never ascends to pane and descends to headless only when the provider lacks native capability. `headless` never ascends. Capability-field presence says **how** a rung runs, never **which** rung to prefer; adding `headless_command` alone cannot move a native-capable provider off native. No command field substitutes for another.

**`--alias` (Claude-Code Agent-tool adapter)**: when set, the `model=` line emits the Claude-Code **short alias** (`opus` / `sonnet` / `haiku` / `fable`) instead of the full versioned ID. This remains the adapter for legacy line-protocol callers; `fab agent -o yaml` exposes the same value natively as `model_alias`. The mapping is prefix-based (`claude-opus-` → `opus`, etc.), so dated variants like `claude-haiku-4-5-20251001` resolve to `haiku`. The **default (flag absent) is** the full ID (the `claude` CLI `--model` flag and the operator launcher's in-process composition consume full IDs). The **`effort=`/`provider=` lines are unaffected** by `--alias`. **Empty / non-Claude models pass through verbatim** (an empty `model=` line stays empty — the inherit signal; an unrecognized/non-Claude ID like `gpt-5` is emitted unchanged) — `--alias` is a best-effort adapter, not a Claude-only validator. The **`dispatch=` line ALWAYS embeds the FULL model ID even under `--alias`** — CLI dispatch never aliases (an external CLI's `--model` flag takes a full ID); aliasing is the Agent-tool-only adaptation. So under `--alias` the `model=` line is aliased while the `dispatch=` command still carries the full resolved ID.

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
effort=high
provider=codex
dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m <codex-model-id> -c model_reasoning_effort=high
```

**Invocation-time overrides** — `--provider <name>`, `--model <id>`, `--effort <level>` are the top rung of the fill precedence above, riding the same single resolution call:

- **`--provider`** swaps the resolved provider and **re-derives `dispatch=` from `dispatch.mode` plus the NAMED provider's capabilities** — so the selected rung and emitted-line presence can differ from the stage's unoverridden result. That is a **query result, not an adapter move**: `fab dispatch start` accepts no override flags and re-resolves the stage from config itself, so only a **config** override (`agent.workers`/`agent.session`, or `agent.profiles.<role>.provider`) actually relocates a stage. An override-only `dispatch=` line is not actionable, and the two remedies are **not interchangeable**: dispatching the stage natively with overridden model/effort works only when the overridden provider/model has the native Agent-tool seam, so for a cross-provider `--provider` override the config override above is the **sole executable path** (see `_cli-fab-pane.md` § fab dispatch and `_preamble.md` § Per-Stage Model Resolution). A swap **re-derives** an unoverridden `model`/`effort` from the NEW provider's own per-role fills (`profiles.<role>`, then `profiles.default`, then empty). An explicit `agent.profiles.<role>.model` still wins. **Swapping to a built-in lands on that built-in's own shipped fill — where it ships one.** Three of the four built-ins carry per-role fills (claude and codex exhaustively; agy with a sparse map), so `--provider codex` resolves a real model rather than the inherit signal. A provider with **no** fills anywhere lands on an empty `model=` — that is `kimi` among the built-ins (it deliberately ships none, so the empty value drops the `-m` pair and kimi falls back to the user's own `default_model`) as well as any project-defined `providers:` entry carrying grammar alone.
- **`--model` / `--effort` are valid WITHOUT `--provider`** here — a within-role override of the profile this pure query prints. `fab agent` accepts the same bare overrides (a final post-refill layer on every addressing form), so the two commands now agree on override legality. See `_cli-fab-operator.md` § fab agent.
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
effort=high
provider=codex
dispatch=codex exec --dangerously-bypass-approvals-and-sandbox -m <codex-model-id> -c model_reasoning_effort=high

# a bare within-role override (equally legal on `fab agent` — final
# post-refill layer there too):
$ fab resolve-agent apply --effort medium
model=claude-opus-5
effort=medium
provider=claude
```

**No validation — verbatim pass-through**: `fab resolve-agent` does NOT validate the provider, model, or effort against any provider's accepted set (provider neutrality — a fab-kit design principle). It echoes the strings as-is — `xhigh`, `reasoning_effort:high`, an empty effort, whatever. A misconfigured pair (e.g. Sonnet + `xhigh`) is NOT corrected by fab; it surfaces as a dispatch-time error in the harness. There is no effort-enum enforcement and no degrade-gracefully drop. (An unknown `--provider` NAME is the one lookup-shaped exception above — naming the resolvable set is not validating a command's content.)

**Effort is advisory on the NATIVE arm.** The resolved `effort=` is only reliably honored where it rides a composed CLI command (`fab dispatch` headless/pane, the operator launcher). On native Agent-tool dispatch it can only be injected as a prompt instruction, and the session-level effort dominates — a known Claude Code limitation (GitHub issues #64033/#39220). Model differentiation works on both arms; treat a per-role effort as binding on the CLI arms and advisory on the native one.

**Exit code**: non-zero only on a real error — an unreadable/malformed config, an unknown stage/role name, or a supplied `--provider` that resolves to no provider. A stage/role resolving to a default is success (exit 0).

---

## fab config

`config` is a six-verb command group: `show`/`explain` inspect, `set`/`unset` surgically modify, and `init`/`upgrade` own whole-file lifecycle.

```
fab config explain [key]      # registry documentation; --json for field rows
fab config show [key]         # effective config; --origin adds provenance
fab config set <key> <value>  # set one project override; --system targets user config
fab config unset <key>        # restore inheritance; --system targets user config
fab config init               # generate fab/project/config.yaml from the registry (--print previews, --force overwrites)
fab config init --system      # write ~/.fab-kit/config.yaml scaffold (same --print/--force)
fab config upgrade            # reconcile fab/project/config.yaml (repo required; --project is explicit)
fab config upgrade --system   # reconcile ~/.fab-kit/config.yaml (works outside a repo)
fab config upgrade --all      # reconcile both layers (repo required); --check probes either for drift
```

### fab config explain `[key]` (`reference` alias)

Pure query (no side effects, no file writes) — prints a **fully-commented reference `config.yaml`** to stdout, documenting every available option so users can discover the whole schema from one place.

```
fab config explain            # commented YAML (default)
fab config explain --json     # machine-readable field table
fab config explain <key>      # owning rendered segment only
```

Accepts at most one dotted key. A keyed query resolves rows rendered inside a shared segment to that owning segment; keyed `--json` returns the matching owning row(s) in the same array shape. Unknown keys fail non-zero naming the key. `reference` remains an invisible Cobra alias so historical pointers keep working. Runs from any directory — it reads no project config and depends on no environment state.

**Generated from a per-field metadata table, not hand-written**: the reference is generated by walking an ordered **per-field metadata table** (`internal/configref`) — each row carries the field's canonical `default`, expected YAML value kind, `description`, `scope` (`project`/`system`/`both`), `advertise` flag, and `renamed_from` carry-forward. Every default that has a canonical Go symbol is sourced from that symbol (`agent.DefaultInteractiveCommand`, the default role profiles via `agent.DefaultProfile`/`agent.RoleNames`, the pipeline stage names via `agent.StageNames`), so there is no second copy of those values and the shown defaults **cannot drift**. A field's `default` is the canonical built-in default, not the value the reference shows as an example: `source_paths`/`test_paths` render an example but their binary default is empty. The generated reference this command prints IS the deployed schema.

**`--json`**: emits the same public field table as a flat, deterministic JSON array (table/rendering order) — per-field objects `{key, default, description, scope, advertise, renamed_from}` with `renamed_from` omitted when empty. Expected kind remains an internal validation signal, so this established JSON wire shape is unchanged. Both renderings are pure and byte-stable; the JSON key set is guarded against drift from the YAML reference's documented keys.

**Full schema coverage**: covers BOTH the binary-consumed keys (modeled on the `Config` struct) AND the skill-consumed keys (read by markdown skills, invisible to Go reflection) — `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `providers.*` (`interactive_command`/`headless_command`/`native`/`profiles.<role>.{model,effort}`), `agent.session`, `agent.workers`, `agent.profiles.*` (`provider`/`model`/`effort`), `dispatch.mode` + `dispatch.column_width` + `dispatch.min_cols` + `dispatch.min_rows` + `dispatch.reap_done`, `autopilot.merge_mode`, `stage_hooks.*`. (The retired `review_tools` block moved to `fab/project/code-review.md` § Review Tools; `agent.spawn_command` moved to `providers.claude.interactive_command`; `branch_prefix` was retired outright in 2.20.0 with no destination — `fab batch switch` names branches by the change folder name; `fab_version` moved OUT of config.yaml to the plain-text sibling `fab/.fab-version` in 2.15.0 and is no longer a config-file key.) Baseline keys appear live with example values (including the provider capability grammars and the two `agent:` depth knobs); the opt-in override blocks (`agent.profiles`, `stage_hooks`) appear commented-out with fab-kit's built-in defaults shown, so uncommenting is opting in. **The reference is the FULL schema; a project's managed fence is slimmer** — `agent.profiles` and `providers` are `advertise: false` as of 2.17.0, so they are documented here but no longer scaffolded into every repo (see § fab config upgrade).

The `providers:` block documents fab-kit's **four built-in providers** — `claude` (the default), `codex`, `agy`, and `kimi` — each carrying its independent launch capabilities plus per-role fills in the binary (`internal/agent`'s built-in provider table), so **naming one on a depth knob, in `agent.profiles.<role>.provider`, or on a `--provider` flag resolves with no `providers:` block at all**. Claude ships `interactive_command`, `native: true`, and the stdin-driven `headless_command` `claude -p --permission-mode bypassPermissions --model {model} --effort {effort}`; codex, agy, and kimi ship both command fields and no native capability. Presence describes how each rung runs and never selects `dispatch.mode`: under the default `native`, claude resolves native while the non-claude built-ins descend to headless. agy's pane command is `agy --dangerously-skip-permissions --model {model}`. Its fresh-workspace trust prompt is an ordinary readiness-gate judgment round; the answer is remembered for the exact workspace path, and an operator may pre-seed that exact path in `trustedWorkspaces` in `~/.gemini/antigravity-cli/settings.json` (fab does not write the trust store). Backlog `[agik]` retains the live open → ready → deliver verification after the current agy quota resets. kimi's equivalent probe closed on 2026-08-10: its `Trust this folder?` wall is one readiness-gate judgment round (remembered per folder) and its side-bordered input box verifies under delivery's box-drawing-tolerant echo check. Claude's and codex's fill maps are exhaustive (all six roles — codex's deliberately runs its author and critic roles on different models); agy's is **sparse** — a role it omits resolves agy's `default` entry. `kimi` ships **no fills at all**, deliberately: its `-m` takes a *user-config* model alias rather than a vendor catalog ID, so a pinned value would break non-managed installs; the empty `{model}` drops the `-m` token pair and kimi falls back to the user's own `default_model`. Non-claude fills are **refreshed at kit-release cadence** and pass through unvalidated; pin a newer model with one config line (`providers.<name>.profiles.<role>.model`, scope `both` — settable once per machine in `~/.fab-kit/config.yaml`, which outranks the project file), which is also how you give `kimi` per-role differentiation. The non-claude blocks render **commented** because they merely restate built-in defaults — a commented block registers no project override, and a built-in provider is inert until a knob, a role override, or a flag names it. All four ship a deliberate full-auto posture because unattended stage workers cannot answer approval prompts: claude carries `--permission-mode bypassPermissions`, codex `--dangerously-bypass-approvals-and-sandbox` on both its forms, agy `--dangerously-skip-permissions` on both forms, and kimi `--auto` on its **interactive** command only — its headless form carries no approval flag at all, because `kimi -p` already auto-approves tools and *errors* when combined with `--yolo`/`--auto`. Override the corresponding provider command to restore approvals. Two grammar specifics are load-bearing: agy's commands carry no `{effort}` placeholder (its model IDs *embed* the reasoning level, e.g. `gemini-3.1-pro-high`, so its fills carry no effort either), and both agy's and kimi's `headless_command`s **nest a shell** — `sh -c '<cli> … -p "$(cat)"'` — because each CLI takes the prompt as the `-p` ARGUMENT and ignores stdin, while POSIX expands `$(cat)` before `fab dispatch`'s stdin redirect applies.

**Output**: byte-stable for a given binary version (same convention as `fab resolve` / `fab resolve-agent`). The emitted document round-trips — its live keys parse cleanly back into `Config`.

**Exit code**: 0 on success. Unknown keys fail operationally; more than one positional argument is a usage error. Writes no file.

### The config cascade (environment > system > project > defaults)

`fab config show` and `show --origin` display the **effective** config after resolving the four-tier cascade the loader (`internal/config.LoadPath`) applies to *every* config read:

1. **environment** — YAML-valued variables derived generically from registry keys (highest precedence)
2. **system** — `~/.fab-kit/config.yaml` (user-global, all repos on the machine)
3. **project** — `fab/project/config.yaml`
4. **built-in defaults** — the Go tables in the binary

**The system tier outranks the project file**, so a personal machine-wide preference beats a repo's committed suggestion. Scope enforcement is what makes that safe: only preference-class fields (`scope: system`/`both`) are honored in the system file at all, so a semantics-class key (`source_paths`, `test_paths`, `stage_hooks`, …) is untouched by the order and the repo stays reproducible for teammates and CI.

Tiers merge **per leaf**: maps merge per-key (the `agent.profiles` precedent), lists replace (never concatenate), scalars replace, and each leaf takes the value of the highest tier that defines it **non-empty**. An **empty leaf — `null`, `""`, `[]`, `{}` — falls through** to the tier below instead of shadowing it, at every tier including the environment; there is no explicit-null override. `false` and `0` are real values and are never skipped (a `dispatch.reap_done: false` resolves `false`). Environment names are mechanically `FAB_` + the uppercase dotted registry key with `.` replaced by `_`; values are parsed as YAML, so scalars, lists, and maps retain config types. The loader walks only the ordered registry keys, honors only `scope: system`/`both`, treats an empty value as unset, and fails open: a malformed value or project-scoped override emits a `fab: warning:` on stderr and is skipped; unrelated/unknown `FAB_*` variables are never scanned. The deliberately advertised variables are `FAB_AGENT_WORKERS` and `FAB_AGENT_SESSION`; similarly named `FAB_AGENTS` is unrelated and has no config meaning. The file-layer behavior is otherwise unchanged: an absent or malformed/unreadable system file is skipped (with a warning for malformed/unreadable), while a malformed *project* file still errors. Warnings never change stdout contracts or exit codes.

### fab config show `[key] [--origin]`

Pure query (no file writes) for the current repo. Config resolves over a four-tier cascade — environment > system `~/.fab-kit/config.yaml` > project > built-in defaults — with per-leaf deep merge and empty-skip (an empty value never overrides a set one).

| Mode | Output |
|------|--------|
| `fab config show` | Fully composed environment-over-system-over-project-over-built-in-defaults config as YAML |
| `fab config show --origin` | Effective values including defaults, WINNER only — one line per leaf with origin `$ENV_VARIABLE` / `system path` / `project path` / `default`; map fields (`agent.profiles`, `providers`) drill down per key |
| `fab config show <key>` | Effective value including the built-in default; scalar/list values are raw, map values are YAML subtrees |
| `fab config show <key> --origin` | The key's FULL STACK — one line per tier that defines it, highest first, as `key = value  # <tier> <label>  (effective\|shadowed)`; a map-valued key drills down per leaf, each leaf listing its own tiers |

```
fab config show                        # fully composed effective config as YAML
fab config show --origin               # each field: value + origin ($ENV_VARIABLE / system path / project path / default)
fab config show agent.workers          # one effective value
fab config show agent.workers --origin # every tier that defines it, winner first
```

The keyed `--origin` listing is the answer to "why is my override not taking effect" — a shadowed tier is printed rather than inferred:

```
agent.workers = codex    # env $FAB_AGENT_WORKERS  (effective)
agent.workers = kimi3    # system /home/u/.fab-kit/config.yaml  (shadowed)
agent.workers = claude   # default  (shadowed)
```

Accepts at most one known dotted key; unknown keys fail non-zero naming the key. Requires a fab repo (walks up for `fab/`, like `fab preflight`). Writes no file. Bare output is the fully composed YAML view; `--origin` changes presentation by adding provenance, not composition.

### fab config set / unset

```
fab config set <key> <value> [--system]
fab config unset <key> [--system]
```

Both route through `internal/configupgrade`'s comment-preserving path splicer and atomic writer; neither marshals the whole document. `set` accepts only documented **scalar leaves** and a single-line, comment-free YAML scalar value (`string`, `bool`, `int`, or `float`). It refuses structural map keys, collection-valued leaves, collection/null values, multiline input, and YAML comments with an explicit manual-edit remedy. An **EMPTY value is refused** as well, naming `fab config unset` as the verb that was meant: an empty leaf falls through to the next tier and can never be effective, so writing one is never what a user wants. The test is applied to the **parsed** value, not to the argument string, so the quoted-empty spellings (`''`, `""`) and an explicit `null` are refused alongside a blank or whitespace-only argument. Fence-only deep fields materialize every ancestor from the registry renderer before the scalar is inserted; opaque provider names remain supported through quoted YAML mapping keys.

A `set` whose key a **higher tier already defines** still writes and still exits 0, but prints a `fab: warning:` on stderr naming the tier that wins (`fab: warning: agent.workers is shadowed by env $FAB_AGENT_WORKERS — the written value is not in effect`). Writing `--system` over a project-file value is NOT shadowed — the system tier outranks it.

`unset` stays kind-ungated so it can repair any current value shape, removes only the live override, and lets the fence advertise the field again. Unsetting a known absent key exits 0 with a notice that names the tier where the key IS live and the command that would remove it (`live in system ~/.fab-kit/config.yaml — use: fab config unset agent.workers --system`; for an environment tier it says so and notes that `unset` cannot remove a variable). A key supplied only by the built-in default gets the bare no-op notice. The shared value parser remains collection-aware (flow or block style) solely because environment overlays accept lists and maps; that broader ENV contract does not widen `set`. Relative to the pre-2.17.3 env layer, four degenerate spellings now warn-and-skip instead of resolving to surprising values: whitespace-only, comment-only, multi-document, and bare-date (`!!timestamp`) env values are rejected per-variable, fail-open.

Without `--system`, the target is `fab/project/config.yaml`. With it, the target is `~/.fab-kit/config.yaml`; only `scope: system`/`both` keys are accepted, and a missing file is created with the canonical fence-owned system header inside its managed reference fence. Live keys stay above that fence, and `set --system` / `unset --system` regenerate it through the same target-aware writer as upgrade. Unknown keys fail naming the key and point to `fab config explain`.

### fab config init `[--system] [--print] [--force]`

Bare `fab config init` selects project mode. `--project` is retained for compatibility; `--system` selects the system scaffold, and passing both explicit flags errors.

**`--system`** writes a `~/.fab-kit/config.yaml` **scaffold** — **only** the system-overridable fields (`scope: system`/`both` — today `dispatch`, `agent`, `providers`, and `autopilot`), **all commented inside a managed fence** whose preamble carries the canonical precedence header — generated from the same per-field metadata table as `fab config explain` so it cannot drift from the schema — rendering each field's SHORT file-bound advert (see § fab config upgrade), not the explain essays. It is the user's answer to "what can I safely override at the system level".

**`--project`** generates a fresh `fab/project/config.yaml` from the registry — the retirement path for the hand-maintained scaffold `config.yaml` (deleted in 2.15.0). It writes the **A-class identity fields** (`--name`, `--description`, `--source-path` repeatable, `--test-path` repeatable) **live** above the managed reference fence, then the fence of commented C fields. No `agent:` key is pinned (presence=intent — an init-pinned knob or role profile would be an accidental override that stops tracking fab-kit's defaults). It shares the same fence renderer as `fab config upgrade`, so a generated file and an upgraded file carry a byte-identical fence. This is the shell-out target `fab init` (the fab-kit binary) calls to bootstrap a project config; when the installed fab-go predates it, `fab init` fails closed with a non-zero error naming the fab-go upgrade remedy — the minimal embedded stub fallback is retired, no stub is written.

```
fab config init --name X --description Y --source-path src/ [--test-path "**/*_test.go"]
fab config init --system      # write the ~/.fab-kit/config.yaml scaffold (refuses to overwrite without --force)
fab config init --print       # preview the exact file on stdout; zero writes, never blocked by an existing file
```

Both modes **refuse to overwrite** an existing target file (non-zero exit, message naming the path) unless **`--force`** is given — the file is user-owned once created; refusal stays the default. **`--print`** renders the exact file init would write to stdout with **zero writes**: it composes with bare/`--project` (including the seed flags) and `--system`, is never blocked by an existing target (a preview, not a write), and `--print --force` is a pure preview (print wins, nothing is written). The `--system` scaffold is fully commented (inert until uncommented).

### fab config upgrade `[--project|--system|--all] [--check]`

The whole-file reconciliation command for the project config, the system config, or both. It shares `internal/configupgrade`'s comment-aware writing engine and fence renderer with surgical `set`/`unset`.

```
fab config upgrade                  # project layer (default; repo required)
fab config upgrade --project        # explicit project layer (repo required)
fab config upgrade --system         # system layer only (repo-free)
fab config upgrade --all            # project + system, with labelled output (repo required)
fab config upgrade --system --check # zero-write drift probe for the system layer
fab config upgrade --all --check    # zero-write probe; non-zero when either layer drifts
```

`--project`, `--system`, and `--all` are mutually exclusive. Bare upgrade remains project-only. `--system` resolves `~/.fab-kit/config.yaml` directly and never asks for a fab root; `--all` includes the project layer and therefore keeps the repo requirement. In `--all` mode each layer receives its own `project:` / `system:` result line, and applying mode reconciles both just as checking mode computes both.

Reconciliation, under the A/B/C field-category model:

- **Live (A) fields kept verbatim**, including the user's own comments. *Presence = intent*: a live field is an override even when its value equals the default — it is NEVER auto-removed (B-hygiene "equals default — remove?" is an advisory report line only).
- **The managed fence (C fields)**: eligible fields not currently overridden are regenerated as a fully-commented scaffold (including parent keys — a live `agent:` over comment-only children is exactly the `agent: null` the old masher produced) inside byte-exact splice anchors: `# >>> fab reference (kit X.Y.Z) >>> …` / `# <<< end fab reference <<< …`. The project target includes `advertise: true` rows and retains top-level-block suppression. The system target includes every `scope: system`/`both` row (including project-fence-demoted provider machinery) and suppresses live leaves within a shared block, so a live `agent.workers` still leaves `agent.session` discoverable; its fence preamble carries the canonical **fence-owned** precedence header inside the anchors. Each advert is the registry row's SHORT file-bound form (`configref.ShortSegment`, 260809-wll4): 1–4 short description lines ending with a `[project|system|both]` scope tag, then — for `scope: both` fields — a `# Settable machine-wide: fab config set --system <key> <value>` pointer, then a `# Full prose: fab config explain <key>` pointer, then the field's YAML block. The long prose essays live in exactly one place — `fab config explain`'s LONG form (`Segment`), whose output is unchanged. Upgrade rewrites ONLY between the anchors; everything outside is the user's. Content the user places BELOW the fence is never dropped — it is **hoisted above** the fence on the next run and classified like any other live key.
- **Legacy system adoption**: the first `--system` run recognizes the old unfenced generated header and commented short-segment scaffold, discards only those recognized generated lines, preserves live YAML and unrecognized comments byte-for-byte above the new fence, and avoids duplicate adverts. Unknown live keys use the same parking path as project reconciliation.
- **Unknown fields parked, never deleted**: a live key no longer in the registry is parked in a `# removed in … (parked by fab config upgrade — delete when done):` block below the fence, its value serialized — appended exactly once, never regenerated away.
- **Renames carried mechanically**: a live field matching a registry row's `renamed_from` is carried to the new key, value verbatim (empty on every row today). A carry is **skipped** (and reported) if the target key is already live, so it never emits a duplicate top-level key.

**Byte-stable and idempotent** — running a mode twice yields byte-identical files (the `fab docs-index` discipline). Before writing, each reconciled document is **validated as YAML** and a run that would produce an unparseable file is **refused** (original left untouched). Writes are atomic (`internal/atomicfile`). Bare/`--project` and `--all` require a fab repo; `--system` does not. `cobra.NoArgs`. `fab upgrade-repo` **auto-runs** the bare project form after sync (fail-open: if the installed fab-go predates the subcommand, it prints a reminder and the upgrade continues).

**`--check` — the drift probe.** Composes with every target mode and computes exactly the same reconciliation (`configupgrade.Check` shares `Upgrade`'s render/validate path, so the probe can never disagree with a real run about what would change) but **writes nothing**: it prints the would-change report — the same report lines the applying run prints — and exits non-zero (operational, exit 1) when the selected file has drifted: a stale fence kit-version stamp, unparked removed keys, a missing fence, or any rendered-content delta — including a missing config file, which a real run would create. `--all --check` exits non-zero when either layer drifts and zero only when both are clean. On a clean target it prints `already up to date`; files remain byte-identical.

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

## fab setup

```
fab setup              # interactive setup wizard (interview over the setup-check probe)
fab setup --defaults   # non-interactive defaults; zero preference writes (a stale system scaffold may refresh)
fab setup --project    # write fab/project/config.yaml instead of the system tier
fab setup check        # read-only environment doctor
```

Bare `fab setup` runs the **interactive setup wizard**: it first reconciles an existing system scaffold (silent when clean, one advisory when changed, warn-and-continue on error; a missing file stays missing), then probes once via `internal/setupcheck` (the same call `check` makes — capability is detected, never asked), prints a scope banner (write target is the **system tier** `~/.fab-kit/config.yaml` by default; `--project` retargets the repo's config and errors outside a fab repo), and walks a 4-question default path — `agent.session`, `agent.workers` (options filtered to detected providers, annotated with capabilities), `dispatch.mode` (options filtered by $TMUX/capability viability, ladder-ceiling semantics stated), and an opt-in advanced section (`agent.profiles.operator/review.provider`, `dispatch.column_width`, `dispatch.reap_done`; opting in asks all four — a never-set profile key is presented as a depth-correct inherit indication, `(inherit agent.session)` for operator / `(inherit agent.workers)` for review, where Enter keeps the inherit and typing a detected provider — even the currently-inherited one — writes an explicit override). Every question defaults to the current effective value with its origin and points at `fab config explain <key>`; an all-Enter run writes no preference values ("nothing to change"), and the whole run is zero-write when the system scaffold is clean or missing. Changed answers get a diff-before-write summary (`<key>: <old> → <new>` + target tier) and a confirmation; writes go through the surgical `fab config set` path in-process (never a whole-file rewrite), with the shared shadow warning per key. Prompts are bare stdin line reads — no TUI. Non-TTY stdin without `--defaults` fails with a hint naming the flag. `setup` is a fab-go command — NOT in the router's workspace allowlist, so the shim's default route forwards it.

`fab setup check` is the **read-only setup-state doctor**: it writes nothing (no config mutation, no trust-store seeding, no agent/pane launches, no prompts) and is safe to re-run identically. It **coexists with `fab doctor` by distinct job** — `doctor` (fab-kit binary) checks system prerequisites ("is this machine good enough to use fab-kit"); `setup check` diagnoses fab's own setup state. Probes (all in `internal/setupcheck`, the same probe layer the wizard consumes):

- **Provider roster** — every resolvable provider (built-ins ∪ user-defined) with binary presence (leading executable of each command; nested `sh -c '...'` wrappers unwrap to the provider's own binary, e.g. `agy`/`kimi`) and declared interactive/headless/native capabilities. A provider a role actually resolves to (depth knob or `agent.profiles.<role>.provider`) with no binary on PATH is a **failure**; an unconfigured provider's absence is informational.
- **Environment** — `$TMUX` presence (the `internal/dispatch` pane-viability classification), `gh`/`yq` presence (warnings when absent), `rk` (informational only — fail-silent optional tooling).
- **Version triplet + skew** — binary version vs kit-cache `VERSION` vs the `fab/.fab-version` project pin; mismatches warn (a `dev` binary is reported, never compared). Plus the **override-masking** bottle-skew check: a system/project-tier `providers.<name>.{interactive_command,headless_command,native}` override on a key the binary's own embedded defaults do not define is flagged *load-bearing against your installed binary* (the #573 incident shape — unsetting it silently changes behavior). User-defined providers are skipped (a definition is not a mask).
- **Config sanity** — `dispatch.mode` viability via `internal/dispatch.SelectMode` itself, echoing the exact descent reasons (`pane unavailable: no tmux`, …): a working descent warns, no reachable rung fails.

**Exit code**: `0` healthy or warnings-only, `1` when any failure-severity finding exists (CI-able), `2` usage error via the standard `run()` seam. Works outside a fab repo (degrades to the system+env config tiers, no project pin).

---

## fab shell-init

```
fab shell-init <bash|zsh|fish>
```

Emits the shell-completion script for the given shell on stdout — the `tu`-style verb equivalent of (and delegated to) Cobra's auto-generated `fab completion <shell>`. Recommended install: add `eval "$(fab shell-init zsh)"` to `~/.zshrc` (or the bash/fish equivalent). Config-independent — works outside a fab repo. Human-setup-facing; no skill invokes it.

---

## fab skill

```
fab skill [topics]
```

Prints the fab **agent skill bundle** — a one-page, static, agent-first usage briefing (when to reach for fab, a capabilities map keyed to subcommands, composition patterns, the stdout/exit-code contracts, gotchas) to stdout as **raw markdown, byte-stable per release**. **stderr empty on success, exit 0**, no rendering/pager/framing (an agent consumes the bytes directly). The single accepted positional is the standard's **reserved topic `topics`**: `fab skill topics` enumerates the tool's content-topic names one per line, raw to stdout, stderr empty, exit 0 — and fab ships **zero topic pages**, so the output is empty (zero bytes), the standard's scriptable "zero topics" answer. `topics` is a machine affordance reserved by the shll `skill` standard in every tool's topic namespace, never a content topic — it is deliberately not a cobra child command, so it appears in neither `fab skill --help` nor the `help-dump` tree. Any other positional remains a usage error → exit `2` via the binary-wide `run()` classification. No flags. Config-independent — works outside a fab repo.

This is the shll toolkit-wide `skill` standard, published at shll.ai (sibling of the machine-readable `help-dump` contract). The bundle is embedded into the `fab-go` binary at build time, so it is offline and version-locked to the release; the same page renders at `shll.ai/tools/fab-kit/skill`.

**Disambiguation**: `fab skill` (this one static bundle command) is unrelated to fab's own **kit-skills** — the many `/fab-*` markdown prompts `fab sync` deploys to `.agents/skills/`. Same word, two concepts.

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

## fab docs-index

```bash
fab docs-index [<root-path>] [--check] [--json] [--rebuild]
```

With no argument, processes all `docs_index.roots`; a positional argument selects one
configured path. An unconfigured path errors naming `docs_index.roots`. There is no `--root`.
Without `docs_index`, the implicit root is `{path: docs/memory, index_file: index.md,
log: true, max_depth: 3}`. Other explicit roots default to `index_file: index.md`,
`also_accept: []`, `log: false`, `max_depth: 3`, `superseded: []`, and `exclude: []`.
Inspect schema/defaults with `fab config explain docs_index.roots`.

`nav_note` is a per-root free-text markdown note for the root landing: unset or empty
renders nothing, and a non-empty value renders verbatim (on a generic root, right
after the "Generated by" note). The generator emits no navigation links of its own —
consumer-layout paths (READMEs, glossaries) belong in `nav_note`, not the binary.

The generator recurses to arbitrary depth. `max_depth` is an advisory bound, never a
traversal limit. Every regular file under a root becomes a topic row, not just `.md`
(dotfiles, symlinks, landing files, `log.md`/`log.seed.md` on `log: true` roots, and
`exclude` matches are excepted). Metadata is per type: markdown rows read the
`description:` frontmatter; `.html`/`.htm` rows take their label from `<title>` and
their description from `<meta name="description">` (a title is never a description);
every other type renders as filename + `—` and the file is never opened. Missing
descriptions in generic roots produce the file H1 plus `—` and an advisory — the
missing-description advisory fires only for `.md` and HTML, the types that can carry a
description (the HTML message: "has no <meta name=\"description\"> — using <title> and
placeholder"). Legacy memory keeps filename-stem labels for byte compatibility.

Every primary (`index_file`) landing carries a hand-managed **manual block** between
`<!-- fab docs-index:manual:start -->` and `<!-- fab docs-index:manual:end -->` —
always emitted, even empty (seeded with an agent-facing comment on creation), with its
content preserved verbatim on regeneration; `also_accept` alternate landings never get
one. The block holds rows the generator cannot produce — descriptions for non-markdown
files, external links, custom groupings. For one version the legacy
`<!-- fab docs-index:curated:start -->` spelling is still read and rewritten as
`manual`. Later loss checks still protect generated and hand-managed navigation.
The manual block is the one hand-managed region of a generated primary landing:
preserved verbatim by `fab docs-index`, and never rewritten by skills.

**Ownership:** `index_file` landings are whole generated files. An existing `also_accept`
landing (for example `README.md`) receives a table inside
`<!-- fab docs-index:generated:start -->` / `<!-- fab docs-index:generated:end -->`.
Prose outside that block stays human-owned and byte-preserved; no neighboring `index_file`
is created. Only generated index files/blocks are rewritten; spec topic content stays
human-curated. Take generated output wholesale after resolving source-file conflicts.

`superseded` is a list of root-relative slash globs: `**` spans directories, `*` and `?`
match within a segment, and bracket classes follow shell-glob syntax. Use
`"**/archive/**"` for archive subtrees, `"**/Z*/**"` for Z-prefix folders, and YAML
single-quoted `'**/\[archived\]-*'` for literal `[archived]-` filenames. Superseded
folders produce one parent pointer/count, then one index of immediate child versions;
no per-file rows or topic descriptions are read inside. Individual file matches fold
into the parent folder's superseded-file count. Version summaries use numeric bounds.
When a live subtree becomes superseded, regeneration removes its obsolete generated
descendant landings and logs (including the boundary folder's log). Alternate landings
lose only their generated blocks; human prose and seed inputs remain. `--check` reports
this configured retirement as benign drift and writes nothing. `exclude` (default `[]`)
uses the same root-relative slash-glob syntax as `superseded`: a matching file produces
no row or count, and a matching folder is not walked at all — the index is the tree
minus `exclude`.

Only `log: true` roots use FKF metadata, reserved-domain exemptions, description
escalations and per-folder `log.md`. For the log/seed/freeze-on-write contract consult
`$(fab kit-path)/reference/fkf.md` §6. `--rebuild` discards frozen logs and rebuilds from
git, with seed merging; ignored under `--check`, irrelevant to `log: false` roots.

| Exit under `--check` | Meaning | Consumer action |
|---|---|---|
| 0 | Byte-clean, no blocking findings | No regeneration needed |
| 1 | Benign drift or independent blocking floor | Regenerate drift; fix blocking source findings |
| 2 | Destructive loss: description wipe, dropped tombstone, or flattened custom grouping | Refuse regeneration; preserve/remediate the reported navigation first |

Worst root wins. Malformed frontmatter blocks at exit ≥1; for `log: true`, registry-gated
change-ids and descriptions over 1000 runes also block. Advisory findings never fail a
byte-clean check. The exit-2 refuse-before-regen guard is unchanged: blocking findings
remain separate from tier-2 losses. An operational error (missing root, invalid root
configuration, read/write error) returns 1. Parse-time usage errors return 2 under the
binary-wide convention; distinguish those from a valid `--check`'s loss report.

`--json` with `--check` emits a single stdout object with `tier`, `drift`, `losses`,
`malformed` (blocking), and `warnings` (advisory). All three arrays are present, never null.
Loss entries have `category`, `path`, `detail`; malformed entries have `kind`, `path`,
optional `detail`; warning entries have `kind`, `path`, `count`, optional `bytes`/`detail`.
Advisory kinds are `description-length`, `missing-description`, `narration-density`,
`file-size`, `unsorted-nonempty`, and `broken-link`; width/depth warnings remain stderr-only.
Narration-density and bundle-relative broken-link diagnostics apply only to `log: true` roots.
Advisory details are capped at 5 per kind across all selected roots, on stderr and
in the JSON `warnings` array. Stderr prints an "… and N more (M total)" summary for
each truncated kind. The additive `warnings_total` integer counts all JSON-eligible
advisories before sampling (including those omitted); width/depth remain stderr-only
and are excluded from that total. Blocking findings and losses remain complete.
A sampled list is not an exhaustive per-file or per-domain inventory; consumers
must treat counts derived from it as lower bounds when `warnings_total` exceeds
`len(warnings)`. Warnings and alias deprecation notices never contaminate JSON stdout.

Memory-only operations select `fab docs-index docs/memory`; shipping may run bare
`fab docs-index` to refresh every configured root.

**Older-binary version-skew fallback:** If the binary lacks `docs-index`, memory-only
callers may use `fab memory-index` with the same flags. It is the deprecated memory-only
alias, retained for at least one minor version, emitting one stderr notice on current
binaries. If the older alias also lacks the requested machine surface, use the calling
skill's documented legacy fallback and warn to upgrade `fab`. Additional-root generation
requires the new command; it cannot fall back through the alias.

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

## fab batch

Multi-target operations: `fab batch <new|switch|archive> [flags] [targets...]`.

| Subcommand | Flags / default | Operation | Guards and failure behavior |
|------------|-----------------|-----------|-----------------------------|
| `new` | `[--list] [--all] [--workers <provider>] [ids...]`; no args ⇒ `--list`; no `--quiet` | Parse pending `- [ ] [xxxx]` backlog items (including continuation lines), create one worktree/window per ID, run rendered `fab-new` with `{description}`; `--all` selects all pending | Launch path requires `$TMUX`, then `wt` (`brew install sahil87/tap/wt`); list path requires neither. Empty `--all` errors `ERROR: No pending backlog items found.` Unknown IDs and backlog items with empty content warn-and-skip (exit 0). Per-item `wt`/tmux failures include child stderr, continue, then exit non-zero with `ERROR: {N} of {M} item(s) failed to launch` |
| `switch` | `[--list] [--all] [--quiet\|-q] [--workers <provider>] [changes...]`; no args ⇒ `--list` | Resolve active changes in-process, create branch worktrees, run rendered `fab-switch` with `{change}`; `--all` excludes `archive/` | Same launch guards; empty set errors `ERROR: No changes found.` Resolver and `wt` errors warn-and-skip with specific/child stderr. Quiet suppresses progress but not list/data output or stderr |
| `archive` | `[--yes\|-y] [--dry-run] [--quiet\|-q] [changes...]` | Archive `hydrate: done\|skipped` changes in-process via `ArchiveWithBacklog`; no agent, tmux, `wt`, or fab-on-PATH dependency | Uses the consent matrix below. `--dry-run --yes` is mutually exclusive. Quiet suppresses progress only, never consent, data, stderr, or footer |

`new` uses `wt create --non-interactive --worktree-name {id}`, window `fab-{id}`, and `{worker-session-command} {shell-quoted fab-new prompt}`. Initial skill prompts are rendered by `internal/agent.SkillPrompt` for the actual launched provider (`$` for `codex`, `/` otherwise — including Claude when command resolution falls back to the built-in). Both launchers compose the default-role provider `interactive_command` through `internal/spawn`, substituting templated `{model}`/`{effort}` or appending them for a plain command; no placeholders reach tmux. Missing `wt` exits 1 after the tmux guard with `ERROR: wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt` (or `'fab batch switch'`); missing tmux is `ERROR: not inside a tmux session`.

On `new` and `switch`, **`--workers <provider>`** safely prefixes every tmux shell command with `FAB_AGENT_WORKERS='<provider>'`. Embedded single quotes are shell-escaped; the value is otherwise passed through without provider validation. Omitting the flag leaves the launch command byte-for-byte unchanged.

`switch` names branches `{folder_name}` — the branch name IS the change folder name, no prefix, the same convention `/git-branch` and `_preamble.md` § Naming Conventions carry, so the worktree attaches to the change's real branch. It probes local `git show-ref --verify --quiet refs/heads/<b>`, then `git ls-remote --heads origin <b>`: existing branches use `--checkout <branch>`, new branches use the positional, and an offline remote probe degrades to the positional so `wt` re-checks. Quiet leaves successful stdout empty because tmux creation is the result; list output and stderr remain.

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
