# Intake: CLI Single-Sourcing & Doc Conformance

**Change**: 260612-ye8r-cli-single-sourcing-doc-conformance
**Created**: 2026-06-12

## Origin

> /fab-new ye8r

One-shot invocation from backlog ID `ye8r` — **Binary-review batch B4/6 — CLI single-sourcing + doc conformance** (filed 2026-06-12). Absorbs backlog item `[x8c9]` (gating/scoring data single-sourcing). The authoritative detail source is `docs/specs/findings/binary-review-2026-06-12.md` §B4 (F23–F30), adversarially verified against commit `1431a9c3`; all verifier corrections from that report are folded into this intake as decisions.

**Wave-2 dependency satisfied**: `k4ge` (skills-audit batch 1/5, CLI exit-code contract conformance) merged as PR #395 (`5c054b5d`), and this branch is cut from exactly that commit. k4ge owned the doc-side fix of the invalid `fab change resolve --folder` canonical form (now fixed in `_preamble.md:232`) and the `fab hook sync` exit-0 overclaim — both are **excluded** here. This batch owns the binary side.

**Reality-sync (clarify session 2026-06-12)**: all sibling binary-review batches have since merged — #396 (mz4q), #397 (dn2c), #398 (pw3k/B5), #399, #400 (hv7t/B2) — plus release v2.1.7. origin/main is at `a5f26b06`, 7 commits ahead of this branch's base, and every Go file this change targets churned in that window. **Rebase onto origin/main before apply**; every `file:line` citation in this intake predates that churn and MUST be re-verified against the rebased tree (the findings themselves were re-verified to survive — see per-finding notes).

## Why

The set of hand-maintained duplicates of CLI-surface data drifts silently, and the drift has **already shipped**:

1. `src/kit/skills/_cli-fab.md:17` documents the router allowlist as `(init, upgrade-repo, sync, update, doctor)` — omitting `migrations-status` — while line 236 of the same file asserts `migrations-status` is "registered in the router's `fabKitArgs` allowlist". This is a live constitution-MUST violation (constitution.md:31) introduced by change `260610-9733`.
2. The shim's hardcoded help line for `migrations-status` (`fab-kit/cmd/fab/main.go:111`) already diverges from the cobra `Short` (`migrations_status.go:34`).
3. The predicted `change resolve` doc/flag drift materialized (`_preamble.md` documented a `--folder` flag the command doesn't have) and had to be patched doc-side by k4ge — the structural cause (two independently implemented spellings of the same operation) is still in place.

For an agent-driven CLI, the markdown reference **is** the discoverability mechanism: undocumented surface is effectively unusable, and wrongly-documented surface is hallucination bait. Without single sources of truth plus drift tests, every future command addition re-rolls these dice. The repo already has the right pattern (`internal/score/changetypes_doc_test.go` derives canonical sets from code and fails on doc drift) — this change generalizes it.

**Goal**: every hand-maintained duplicate of CLI-surface data gets one source of truth + a drift test; documented surface == actual surface.

## What Changes

### F23 — Collapse the 5x-duplicated workspace-command allowlist [high/small]

The 6 workspace commands are hand-maintained in five places: shim routing map `fabKitArgs` (`src/go/fab-kit/cmd/fab/main.go:17-24`), shim `printHelp` hardcoding names AND Shorts (`main.go:104-111`), fab-kit's `fabKitCommands` (`cmd/fab-kit/main.go:26-33`, never consulted by `main()`), and two tautological tests (`cmd/fab/main_test.go:10-26`, `cmd/fab-kit/main_test.go:7-23`) that re-declare the same 6 strings and can never catch real skew.

- Create one shared table in the fab-kit module's internal package: `var LifecycleCommands = []struct{ Name, Short string }{...}` (module `github.com/sahil87/fab-kit/src/go/fab-kit`; `cmd/fab` and `cmd/fab-kit` deliberately share it per `docs/memory/distribution/kit-architecture.md:451`).
- Shim derives `fabKitArgs` and its workspace-help section from the table. **Do NOT exec `fab-kit --help` for help text** — derive from the Go table in-process so the help renders even when the fab-kit binary is absent (verifier caution; today's static help always renders).
- `cmd/fab-kit` derives `fabKitCommands` from the table; a test asserts each registered cobra command's `Short` matches the table entry (catches the already-diverged `migrations-status` help line).
- Fix `_cli-fab.md:17` to include `migrations-status` in the router list.
- Add a contract test modeled on `changetypes_doc_test.go` parsing the `_cli-fab.md` router line against the canonical set.
- Add a collision test asserting no fab-go top-level command name appears in the allowlist, sourced from `fab help-dump` output (NB: the hidden command is `help-dump`, not `helpdump` — invoke `fab help-dump`). The negative-match-routing design decision (kit-architecture.md, "Rejected: Positive match") concerns router *runtime*, not test-time checks — no conflict.

### F24 — Document undocumented surface in `_cli-fab.md` [medium/small]

constitution.md:31 mandates `_cli-fab.md` updates for CLI changes, yet three surfaces are absent:

- `fab pane window-name ensure-prefix|replace-prefix` (`src/go/fab/cmd/fab/pane_window_name.go:17-50`) including its `--json` flag and bespoke exit-code scheme (2 = pane missing, 3 = other tmux failures, `tmuxExitCode` at :132-144). `_cli-fab.md` currently lists the pane family as `map|capture|send|process` only.
- `fab shell-init <bash|zsh|fish>` (`shellinit.go:12-16`) — appears nowhere under `src/kit/`.
- `fab change list --show-stats` (`change.go:142`) — missing from the `_cli-fab.md` change table and from `_preamble.md`'s `fab change` row (`list [--archive]`).

Same pass: fix the memory-vs-code inconsistency at `docs/memory/runtime/pane-commands.md:217`, which describes the scheme as "1 (no tmux) / 2 / 3" while the code routes tmux-not-running → 3. This consolidates known findings f008 + f136 from `docs/specs/findings/skills-review-2026-06-11.md` — reference them as consolidated here rather than opening a parallel track.

### F25 — `fab resolve` output-mode flags: mutual exclusivity; dead `--id` [medium/small]

`resolve.go:72-91` registers five booleans (`--id --folder --dir --status --pane`) encoding a single output enum, resolved by a manual PreRunE priority chain (folder>dir>status>pane). Multiple flags are silently tolerated (`--status --folder` prints the folder), and `--id` is registered but **never read** (behavioral no-op). This contradicts the codebase's own convention (`panemap.go:29`, `pane_capture.go:24` use `MarkFlagsMutuallyExclusive`).

- Implement the minimum: `cmd.MarkFlagsMutuallyExclusive("id", "folder", "dir", "status", "pane")` so conflicting flags fail loudly, and wire `--id` into the selection chain so it is a real (explicit-default) flag rather than dead surface.
- **Defer** the `-o/--output id|folder|dir|status|pane` enum consolidation: it touches the documented agent-facing interface in ≥4 doc files (`_cli-fab.md`, `_preamble.md:233`, `kit-architecture.md:300`), exceeding this batch's budget; the one-line minimum is the verifier-endorsed defensible part. <!-- clarified: user confirmed minimum scope — -o/--output enum deferred -->
- Update `kit-architecture.md:396` (documents the PreRunE priority chain) and the `_cli-fab.md` resolve rows.

All current callers pass exactly one flag, so no breakage.

### F26 — Consolidate config.yaml parsers behind `internal/config` [medium/medium]

`fab/project/config.yaml` is parsed at six independent sites with one-off structs. Consolidate the **five fab-module sites** (the fab-kit module's `readFabVersion` stays — Go internal-package visibility forbids cross-module reuse; verifier correction):

- `internal/config/config.go:17-21` (stage_hooks/true_impact_exclude/test_paths — the shared Config)
- `internal/spawn/spawn.go:13-17` (agent.spawn_command)
- `cmd/fab/batch_switch.go:146-158` (branch_prefix)
- `internal/prmeta/prmeta.go:461-479` (project.linear_workspace — note `buildDerivation` parses the same file twice in one function: shared `config.Load` at :336 then local `readLinearWorkspace` at :340)
- `internal/preflight/preflight.go:137-152` (fab_version staleness check)

Widen `internal/config.Config` to model `branch_prefix`, `agent.spawn_command`, `project.linear_workspace`, `fab_version` (yaml.v3 ignores unknown keys — widening is free) and convert the four satellite parsers into accessors over a single `config.Load` result, following the existing nil-safe `GetStageHook` accessor pattern. Per-caller fallback semantics MUST survive as accessor behavior: spawn's default command, empty branch prefix, preflight's silent skip (`docs/memory/pipeline/preflight.md:40`). Known caveat to carry: a single Unmarshal couples failure modes — a yaml type error on any modeled key sends other accessors to their fallbacks; the documented fallbacks make this safe but it is a deliberate, recorded semantic change for malformed configs. `fab spawn-command --repo` builds the path from an arbitrary repo root — the widened `Load(fabRoot)` accommodates it.

### F27 — Deduplicate the resolve surface [medium/small]

- `fab change resolve` (`change.go:147-169`) is a separately implemented cobra command whose body (`internal/change/change.go:344-347`) is a one-line passthrough to `resolve.ToFolder`. Make it a **thin cobra wrapper** that literally sets the folder output mode on the shared resolve implementation, so help/flag surface can never drift again. Deprecation in favor of `fab resolve --folder` is **rejected** — skills depend heavily on `change resolve` (`git-branch.md`, `fab-new.md`, `git-pr.md`, `git-pr-review.md`).
- `fab resolve --pane` (`resolve.go:43-66`) reuses pane discovery but hardcodes `server=""` (`resolve.go:49`) and is current-session-only, unlike every `fab pane` subcommand. Plumb `--server`/`-L` through `fab resolve` (matching the pane family's persistent flag at `pane.go:14`) rather than adding a new `fab pane find` surface — the gap is latent (no current skill invokes `resolve --pane`), so the smaller change wins.

### F28 — `fab log command` owns its best-effort contract [medium/small]

`fab log command` is pure telemetry yet exits non-zero on several paths (explicit change arg fails resolve, `internal/log/log.go:18-22`; unwritable `.history.jsonl`, `log.go:122-125`; FabRoot failure, `cmd/fab/log.go:28-31`), forcing `_preamble.md` to mandate `2>/dev/null || true` on every call site — one forgotten guard turns a telemetry hiccup into a pipeline STOP under the preamble failure rule.

- Make `fab log command` (the telemetry event **only** — not `log review`/`log transition`, which stay as-is) always exit 0, printing a one-line warning to stderr on internal failure. This matches the documented design posture (`docs/memory/pipeline/schemas.md:136`: "telemetry hooks never become new failure modes") and the existing `WriteTrueImpact` pattern (stderr warning + return nil).
- Then delete the `2>/dev/null || true` boilerplate: `_preamble.md` Common-fab-Commands row + key-behaviors bullet + step-4 template (coordinate the exact wording), the 5 skill files with explicit guarded calls (`fab-help.md`, `fab-switch.md`, `fab-setup.md`, `fab-operator.md`, `fab-discuss.md`), `_cli-fab.md`'s fab-log row (exit-1 asymmetry note at :142), and each touched skill's `docs/specs/skills/SPEC-*.md` mirror — same PR.

### F29 — `fab batch switch` in-process resolution; batch no-arg defaults [medium/small]

- `batch_switch.go:83` resolves each change by exec'ing `fab change resolve` through PATH — the only self-exec in either Go module. The round-trip runs the shim's `ResolveConfig` + `EnsureCached` (which can trigger a network download on cache miss) per change, fails when `fab` is not on PATH, and `.Output()` discards the resolver's specific stderr ("Multiple changes match…") into a generic warning. Replace with a direct `resolve.ToFolder(fabRoot, change)` call — `resolve` is already imported (`batch_switch.go:10`), `fabRoot` is in scope (:37), and the sibling `batch_archive.go:77` was already deliberately fixed this way (the in-process pattern is the documented design direction, `kit-architecture.md:141`). Keep warn-and-skip, now surfacing the specific error.
- Align `batch archive`'s no-arg default with its siblings: default to `--list`, require explicit `--all` for the bulk action (`batch new`/`batch switch` default `--list`; archive defaults `--all` — the one bulk-mutating member acts on everything implicitly, and archive moves are effectively irreversible within `archiveLoop`). Verifier notes the current default is a deliberate, documented UX choice (code comment + `_cli-fab.md:494`; the acted-on set is pre-filtered to hydrate done|skipped) — this is a consistency judgment call, taken here in the safer direction; the spec example (`assembly-line.md:128`) already uses explicit `--all`. <!-- clarified: user confirmed — batch archive no-arg defaults to --list, explicit --all required -->

### F30 — Unify exit-code semantics and error formatting [low/medium]

- Extend `window-name`'s exit-code scheme (2 = pane missing / 3 = other tmux failure, a deliberate decision from `260423-rxu3`) to `pane capture` and `pane send`, which currently exit 1 on the identical `ValidatePane` failure — preserving the one exit-code-sensitive consumer (fab-operator treats window-name exit 2 as successful removal; that semantic is untouched).
- For plain exit-1 paths, return errors from `RunE` instead of in-handler `os.Exit` so all failures flow through the single `main.go` formatter (`ERROR: %s`; today in-handler paths print a second `Error: ...` format). The pattern is systemic beyond the cited files (`batch_switch.go:59/68`, `batch_new.go:60/69`, `operator.go:33`, `pane_process.go:79`, `batch_archive.go:67/94/99`) — convert plain exit-1 sites; reserve in-handler `os.Exit` for commands that genuinely need non-1 codes. Conform to the exit-code scheme k4ge (#395) established.
- Same-PR doc updates: `_cli-fab.md` pane rows (":204/:212 Pane not found → exit 1" become 2/3) and `docs/memory/runtime/pane-commands.md` (scheme + the exact documented error strings).
- **Post-#398 note**: pw3k/B5 already routed capture/send tmux failures through `pane.RunCmd`/`pane.StderrError` with pane-named errors — the error-*surfacing* half is partially done. What remains for F30 on current main: the exit-code scheme (capture/send `ValidatePane` paths still `os.Exit(1)` at `pane_capture.go:49/55`, `pane_send.go:33/50`) and the single-formatter RunE funneling. <!-- clarified: F30 re-verified against post-#398 main — exit-code split and os.Exit-in-RunE sites all survive -->

### +x8c9 — DROPPED: already shipped on main

The absorbed backlog item is already implemented on origin/main (post-#399/#400): `internal/score/changetypes_doc_test.go` now parses the `docs/specs/change-types.md` expected_min and `## Gate Thresholds` tables and asserts them per canonical type against `getExpectedMin`/`getGateThreshold` ("doc drifted" failures) — exactly the drift test this batch planned. Nothing remains to implement; note the satisfied `[x8c9]` absorption when marking backlog at ship time. The former SEAM with `hv7t` is dissolved — hv7t merged as #400; this branch rebases onto origin/main instead. <!-- clarified: x8c9 dropped — drift test verified present on origin/main changetypes_doc_test.go; hv7t seam dissolved by merge -->

## Affected Memory

- `distribution/kit-architecture.md`: (modify) shim allowlist → shared `LifecycleCommands` table + derived help (replaces the static-help asymmetry note); resolve PreRunE priority chain (:396) → mutual exclusivity; batch switch resolution now in-process (extend the :141 batch-archive note to the whole family)
- `runtime/pane-commands.md`: (modify) unified pane-family exit-code scheme (2/3 extended to capture/send), corrected "1 (no tmux)" inconsistency (:217), updated documented error-string formats
- `pipeline/schemas.md`: (modify) `fab log command` now fully owns the best-effort contract (always exit 0, stderr warning) — the ":136 telemetry posture" note becomes literally true; call-site guard boilerplate retired
- `pipeline/preflight.md`: (modify) staleness check now reads `fab_version` via the shared `internal/config` accessor (silent-skip semantics unchanged — minor wording touch only)

## Impact

- **Go, fab-kit module** (`src/go/fab-kit/`): `cmd/fab/main.go` (routing map, printHelp), `cmd/fab-kit/main.go`, new `internal` LifecycleCommands table, both main_test.go files replaced with table-derived + cross-check tests
- **Go, fab module** (`src/go/fab/`): `cmd/fab/resolve.go`, `change.go`, `batch_switch.go`, `batch_archive.go`, `log.go`, `pane_capture.go`, `pane_send.go`, plus the broader os.Exit-in-RunE sites; `internal/config/config.go` (widened), `internal/spawn`, `internal/prmeta`, `internal/preflight` (accessor conversions); `internal/log/log.go`; new contract/collision tests throughout
- **Kit skills** (`src/kit/skills/`): `_cli-fab.md` (router line F23; window-name/shell-init/--show-stats F24; resolve rows F25/F27; log-command row F28; pane exit-code rows F30; batch rows F29), `_preamble.md` (fab-change row `--show-stats`, log-command boilerplate retirement), `fab-help.md`, `fab-switch.md`, `fab-setup.md`, `fab-operator.md`, `fab-discuss.md` (guard removal)
- **Spec mirrors** (`docs/specs/skills/`): SPEC-*.md for every touched skill file (constitution.md:32)
- **Specs/docs**: `docs/specs/change-types.md` (only if the drift test reveals current drift — otherwise untouched per Principle VI)
- **Coordination**: `d9rs` (skills-audit 5/5, not yet shipped) collides on the `_cli-fab.md` index line — second-to-merge rebases. All binary-review siblings are merged; rebase onto origin/main (≥ `a5f26b06`) before apply
- **Constraint**: every CLI signature/flag change updates `_cli-fab.md` same-PR (constitution.md:31); skills calling changed commands + their SPEC mirrors updated same-PR

## Open Questions

None — all decision points are resolved. The clarify session of 2026-06-12 resolved both former Tentative assumptions (#11 F25 scope, #12 F29 archive default) to the recommended options and bulk-confirmed all Confident assumptions.

## Clarifications

### Session 2026-06-12

**Q1 (F25, assumption #11)**: Scope of the `fab resolve` output-flag fix? → **Minimum scope**: `MarkFlagsMutuallyExclusive` over the 5 bools + wire `--id` into the selection chain; the `-o/--output` enum consolidation is deferred.
**Q2 (F29, assumption #12)**: `fab batch archive` no-arg default? → **Flip to `--list`**; explicit `--all` required for the bulk action.

Also this session (reality-sync, agent-verified against origin/main `a5f26b06`): x8c9 dropped — its drift test already shipped; hv7t seam dissolved (#400 merged); rebase onto origin/main before apply; F30 re-scoped (error-surfacing half shipped in #398, exit codes + RunE funneling remain).

### Session 2026-06-12 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 9 | Confirmed | — |
| 10 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope = findings report §B4 F23–F30 + absorbed x8c9, with all adversarial-verifier corrections applied (`help-dump` not `helpdump`; fab-kit config reader out of F26; f008/f136 consolidated into F24) | Backlog entry enumerates ACTIONS explicitly; report verified vs `1431a9c3` | S:95 R:85 A:95 D:95 |
| 2 | Certain | Wave-2 dependency satisfied; k4ge-owned items excluded; all sibling binary batches now merged (#396–#400) — rebase onto origin/main (≥ `a5f26b06`) before apply | Verified via `gh pr list` + origin/main log during clarify; all findings re-verified to survive the churn | S:95 R:90 A:95 D:90 |
| 3 | Certain | F23: shim workspace help derived from the shared Go table in-process — no `fab-kit --help` subprocess | Backlog mandates ("do NOT exec"); verifier endorses (keeps no-binary help working) | S:95 R:80 A:90 D:90 |
| 4 | Certain | F28: always-exit-0 scoped to `fab log command` only; `log review`/`log transition` unchanged; boilerplate removal spans `_preamble.md` + 5 skill files + `_cli-fab.md:142` + SPEC mirrors, same PR | Backlog explicit incl. the `_preamble.md` wording coordination; documented posture (`schemas.md:136`) aligns | S:95 R:75 A:90 D:90 |
| 5 | Certain | F27: `change resolve` becomes a thin wrapper over the shared resolve implementation — deprecation rejected | Clarified — user confirmed | S:95 R:70 A:90 D:85 |
| 6 | Certain | F27: pane mode gets `--server`/`-L` plumbed through `fab resolve` (and noting the session-scoping fork), not a new `fab pane find` command | Clarified — user confirmed | S:95 R:80 A:75 D:65 |
| 7 | Certain | F26: consolidate the 5 fab-module parsers only; fallback semantics preserved as accessors; coupled-Unmarshal caveat accepted and recorded | Clarified — user confirmed | S:95 R:70 A:85 D:80 |
| 8 | Certain | x8c9 DROPPED — the drift test already exists on origin/main (`changetypes_doc_test.go` asserts both maps vs `change-types.md`) | Clarified — verified on origin/main during clarify session | S:95 R:90 A:95 D:95 |
| 9 | Certain | F30: extend 2/3 scheme to capture/send + funnel plain exit-1 `os.Exit` paths through RunE across cited and verifier-noted sites; operator's window-name exit-2 contract preserved; conform to k4ge's scheme | Clarified — user confirmed | S:95 R:65 A:75 D:70 |
| 10 | Certain | Scope OUT the generic CI help-dump-vs-`_cli-fab.md` diff; rely on F23 contract+collision tests and the already-shipped change-types drift test | Clarified — user confirmed | S:95 R:85 A:75 D:70 |
| 11 | Certain | F25: minimum scope — `MarkFlagsMutuallyExclusive` over the 5 bools + wire `--id` into the chain; `-o/--output` enum deferred | Clarified — user chose recommended option (minimum; enum deferred) | S:95 R:70 A:55 D:45 |
| 12 | Certain | F29: `batch archive` no-arg default flips to `--list` (explicit `--all` required), aligning with siblings | Clarified — user chose recommended option (flip to `--list`) | S:95 R:65 A:50 D:40 |
| 13 | Certain | Coordination: `d9rs` `_cli-fab.md` index-line collision → second-to-merge rebases; constitution same-PR doc rule governs throughout; hv7t seam dissolved (merged as #400) | Backlog constraint, refreshed against merge state during clarify | S:95 R:80 A:90 D:90 |

13 assumptions (13 certain, 0 confident, 0 tentative, 0 unresolved).
