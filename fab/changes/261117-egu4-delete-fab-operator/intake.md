# Intake: Delete fab's Operator (Plan Step F2)

**Change**: 261117-egu4-delete-fab-operator
**Created**: 2026-11-17

## Origin

> Implement step F2 (Delete fab operator) for fab-kit. Plans live in the hexokit checkout: /home/sahil/code/sahil87/hexokit/fab/plans/sahil/26-11-10-operator-to-hexokit.md (section F2, plus its rules) and /home/sahil/code/sahil87/hexokit/fab/plans/sahil/26-11-16-operator-autopilot-restructure.md (section F2). Scope: delete fab operator code (operator*.go, except anything fab pane still needs), the fab-operator skill and _cli-fab-operator.md, the user-level pointer template, update memory/specs; keep fab agent, fab pane and fab status set-base-branch; retire fab autopilot.merge_mode with a migration note in src/kit/migrations/. The hexokit side is fully shipped (R1-R6c merged; rk skill operator and rk operator autopilot are the replacement). The user explicitly waived the C1 + 1 release wait and wants this ASAP; bugs are acceptable, speed matters. Keep the /fab-operator pointer behaviour working so rk skill operator is the entry point. After the intake, stop and report; the operator will send /fab-fff.

One-shot `/fab-new`, spawned by a HexoKit operator. No prior conversation. The two plan files are
**not** in this repo; read them at the absolute paths above if more context is needed (parent plan
§ "F2", § "Decisions already taken", § "Rules for the operator executing this plan"; restructure
plan § "F2" and § "Decisions" row `merge_mode` location).

Plan text, verbatim:

- Parent plan F2: *"After one release with F1's wrappers: delete `operator*.go` (except anything
  `fab pane` still needs), the operator skill and `_cli-fab-operator.md`, the user-level pointer
  template, and update memory/specs. Keep `fab agent`, `fab pane`, `fab status set-base-branch`."*
- Restructure plan F2: *"Unchanged from the parent plan, plus: retire fab's `autopilot.merge_mode`
  config key (rk owns `operator.merge_mode` now) with a migration note in `src/kit/migrations/` that
  spells out the manual step — fab cannot write rk's settings, so anyone who set the fab key must run
  `rk settings set operator.merge_mode <value>` or silently fall back to `cherry-pick-ladder`."*
- Restructure plan decision: *"`merge_mode` location — rk settings key `operator.merge_mode`; fab's
  `autopilot.merge_mode` is never read."*

The plan's "user gate" and "C1 + 1 release" dependency are **waived by the user** for this run.

## Why

F1 (#701, `261113-915b-operator-hexokit-handover`, shipped in v2.28.17/18) handed the operator to
HexoKit: `fab operator …` verbs deprecate-and-delegate to `rk operator …` when a capable rk is
present, fab stopped seeding rk's cron, and `/fab-operator` became a pointer that follows
`rk skill operator`, falling back to fab's old operator skill (now `_operator-legacy.md`) only when
rk can't serve it. HexoKit has since shipped the whole replacement (R1–R6c merged: state engine,
check, Operator Tasks, run brief + chat skill, loop, Go autopilot incl. stacked-prs merge-all and PR
recovery). rk's autopilot reads `operator.merge_mode` from rk settings and never reads fab's
`autopilot.merge_mode`.

What remains in fab is dead weight that still costs: ≈ 10,400 lines of operator Go + tests
(`src/go/fab/cmd/fab/operator*.go`), the 877-line `_operator-legacy.md` skill, the 355-line
`_cli-fab-operator.md`, and an `autopilot.merge_mode` config key nothing honours — a config key that
silently does nothing is a trap (users set it and get `cherry-pick-ladder`). The two-repo seam was the
historical source of breakage (plan § Problem); deleting fab's side finishes the move so there is one
owner.

Alternative rejected: keep the legacy fallback indefinitely. That keeps two operators alive and the
seam this plan exists to remove; HexoKit is a hard prerequisite for the operator anyway (`rk operator`
launches it).

## What Changes

### 1. Delete the `fab operator` command family (Go)

Delete every `src/go/fab/cmd/fab/operator*.go` file and its tests:

```
operator.go              operator_clock.go        operator_clock_sync.go
operator_handover.go     operator_migrate.go      operator_state.go
operator_tick_start.go   operator_time.go         operator_track.go
operator_track_types.go
operator_test.go         operator_clock_test.go   operator_handover_test.go
operator_migrate_test.go operator_tick_diff_test.go operator_track_test.go
operator_track_types_test.go
```

- Remove `operatorCmd()` from the root command list in `main.go` (line ~39). After this,
  `fab operator …` is an unknown command (cobra's normal error). No deprecation shim, no alias.
- **Symbols used outside `operator*.go`** (verified by grep over non-operator `cmd/fab/*.go`):
  - `exitCodeError` (defined in `operator_handover.go`) is used by `main.go:117` — **move** it (type +
    `Error()` method) into `main.go` or a small shared file, or drop the `main.go` branch if it only
    served rk delegation. Apply decides by reading `main.go`.
  - `rkOperatorPath` — referenced only in a comment in `pr_sync.go:16`; reword the comment.
  - `operatorStatePathOverride` — referenced only in a comment in `pane_map.go:176`; reword.
  - `operatorCmd` / `rkOperatorHandover` — referenced by `launcher_prompt_test.go` and
    `main_test.go`; delete/adjust those test cases.
  - Any other compile breaks surfaced by `go build ./... && go test ./...` are fixed by moving the
    needed helper next to its remaining caller (e.g. `gitRepoRoot`, `slugify`, `nowRFC3339`), never by
    keeping an operator file.
- `fab pane *`, `fab dispatch *`, `fab agent`, `fab resolve-agent`, `fab status set-base-branch` stay
  and must keep passing their tests.
- `src/go/fab/cmd/fab/fab_help.go` — remove the `"fab-operator": "Maintenance"` row only if the skill
  is gone — it is (§3), so remove the row. `fab_help_test.go` follows.
- `src/go/fab/cmd/fab/dispatch.go:33` comment mentioning `fab operator` — reword.
- `src/go/fab/cmd/fab/skill.md` (the `fab skill` bundle) — drop the operator verbs.
- `internal/agent`: **keep the `operator` role** (`RoleOperator`) — HexoKit's `rk operator` launches
  via `fab agent operator -o yaml`. Only reword comments that cite the `fab operator` launcher
  (`agent.go:4`, `:126`, `:264`).
- `internal/spawn/spawn_test.go` uses `'/fab-operator'` as sample command text — harmless, leave.

The on-disk state file `$XDG_STATE_HOME/fab/operator/<slug>.yaml` is **not touched** — fab never
deletes it; `rk operator adopt` reads it on its own path logic (run-kit `internal/cron/dir.go`).

### 2. Delete the operator skills and CLI partial

- Delete `src/kit/skills/_operator-legacy.md` (the F1 fallback copy of the old operator skill).
- Delete `src/kit/skills/_cli-fab-operator.md`. **Relocate its `## fab agent` section** (lines
  ~298–355: resolver role/depth/fill precedence, YAML schema, `dispatch:` ladder, no-rung errors) into
  `src/kit/skills/_cli-fab.md` as `## fab agent` — `_preamble.md` and others cite
  "`_cli-fab-operator.md` § fab agent" and must be repointed to "`_cli-fab.md` § fab agent".
- `_preamble.md`:
  - § Skill Helper Declaration "Allowed values" — remove `_cli-fab-operator`.
  - § Context Loading — drop the `/fab-operator` reduced-set example (the pointer loads nothing).
  - § HexoKit (rk) Reference — drop the "operator's escalation `rk notify` send" / "operator's startup
    role self-mark" pointers into `_cli-external.md`/`fab-operator.md` §2.
  - § Next Steps Convention — drop "`/fab-operator`'s status frame" example.
  - § Per-Stage Model Resolution — repoint "§ fab agent" to `_cli-fab.md`; drop "The operator launcher
    is the deliberate exception …" sentence (no fab-side launcher remains).
- `_cli-external.md` § rk (HexoKit), `_cli-agents.md`, `_cli-fab-pane.md`, `_cli-fab.md`,
  `fab-incognito.md`, `fab-setup.md` — remove every reference to `fab operator`, `_operator-legacy`,
  `_cli-fab-operator`, and operator-only procedures; keep the parts still used by dispatch / pane
  (e.g. `_cli-agents.md` § Pre-Send Validation is cited by the pane readiness gate).
- `_cli-fab.md` config-key docs — remove `autopilot.merge_mode` (§4).

### 3. `/fab-operator` is deleted entirely (project and user level)

**User decisions (2026-11-17, after the first intake draft):** (a) drop the user-level skill entry;
(b) delete the project-level `fab-operator` skill too — run-kit has been told to stop typing
`/fab-operator` from `rk operator` (that run-kit change ships separately). `rk skill operator` is the
only operator entry point; fab ships no `/fab-operator` skill at all.

- Delete `src/kit/skills/fab-operator.md`. Remove its row from `src/go/fab/cmd/fab/fab_help.go`
  (`"fab-operator": "Maintenance"`) and `fab_help_test.go`, and any skill-catalog/list that names it
  (`_preamble.md`, `fab-help`, `fab-incognito.md` catalog, `docs/specs/skills.md`, `docs/site/skill.md`,
  `README.md`, Go tests enumerating kit skills such as `src/go/fab-kit/internal/skills_test.go`).
- `fab sync`'s existing manifest-based pruning should drop the deployed
  `.agents/skills/fab-operator/` / `.claude/skills/fab-operator/` copies from projects on next sync —
  verify (and fix if needed) that removing a kit skill prunes its deployed copy.
- **User-level skill removed**:
  - Delete the template `src/kit/templates/user-skill-fab-operator.md`.
  - Delete `deployUserOperatorSkill` in `src/go/fab-kit/internal/skills.go` and its call site;
    `fab sync` no longer prints `User-level skill (fab-operator): …`.
  - **Clean up previously written copies**: `fab sync` removes `~/.agents/skills/fab-operator/` and
    `~/.claude/skills/fab-operator/` when they exist **and** their `SKILL.md` is the fab-generated
    pointer (frontmatter `name: fab-operator` plus the template's generated-by marker
    `Generated by \`fab sync\``); anything else at that path is left alone. Report the removal in sync
    output (e.g. `User-level skill (fab-operator): removed (retired)`). Best-effort: a removal error is
    a warning, never a sync failure.
    <!-- assumed: fab sync actively removes stale fab-written user-level copies so rk operator from $HOME genuinely stops resolving /fab-operator, per the user's "should stop working" -->
  - Delete `ProbeUserSkills` in `src/go/fab/internal/setupcheck/probes.go` and its registration in
    `setupcheck.go` + tests (`setupcheck_test.go`) — no user-level skill to check.
  - Update `src/go/fab-kit/internal/skills_test.go` and `sync_integration_test.go`: drop user-level
    deploy tests, add a test for the stale-copy cleanup (fab-generated copy removed; foreign file kept).
  - `fab-setup.md` / `_preamble.md` § Skill Helper Declaration: remove the "skill launched outside any
    `fab sync`-deployed project, e.g. the operator from a neutral directory via its user-level pointer
    skill … `$(fab kit-path)/skills/{helper}.md`" fallback wording that only existed for the user-level
    pointer (keep any generic kit-path fallback only if another consumer uses it — apply checks).

### 4. Retire `autopilot.merge_mode`

Remove the key from every fab surface:

- `src/go/fab/internal/config/config.go` — `AutopilotConfig`, the `Autopilot` field on the config
  struct, `DefaultAutopilotMergeMode`, `ValidAutopilotMergeModes`, `GetAutopilotMergeMode` (and their
  comments).
- `src/go/fab/internal/agent/agent.go:94–117` — stop pushing `builtinDefaults.Autopilot.MergeMode`.
- `src/go/fab/defaults.yaml` — remove the `autopilot:` block.
- `src/go/fab/internal/configref/configref.go` (+ `defaultsmap_test.go`) — remove the
  `autopilot.merge_mode` reference entry (`fab config explain` / advertise list).
- `src/go/fab/internal/configscope/configscope.go:79–114` (+ test) — remove the `autopilot` scope row
  and key.
- `src/go/fab/cmd/fab/config_validate.go:30` — remove the case; `fab config set autopilot.merge_mode …`
  then fails as an unknown key (normal behaviour).
- `config.go`, `config_test.go`, `config_json_test.go` in `cmd/fab` — drop autopilot cases.
- `fab/project/config.yaml` lines ~77–83 — remove the commented `autopilot:` block; also the scaffold
  config template if it carries one.
- An existing `autopilot:` block in a user's project or system config is **ignored silently** —
  yaml.v3 drops unknown keys (the stale `fab_version:` precedent in `config.go:379/790`). No load-time
  error, no file rewrite by the binary.

### 5. Migration file

New file in `src/kit/migrations/`. The F1 file `2.28.16-to-2.29.0.md` is still an unreleased slot and
migration ranges MUST NOT overlap (`fab migrations-status` refuses on overlap), so a new file must start
at or after `2.29.0` — use **`2.29.0-to-2.30.0.md`**, range release-owned (the `2.27.0-to-2.28.0` /
`2.28.16-to-2.29.0` precedent: Summary describes the change, not the bump).
<!-- assumed: new migration slot 2.29.0-to-2.30.0 rather than folding into the unreleased 2.28.16-to-2.29.0 file — avoids overlap; release owner may renumber -->

Content (Summary / Pre-check / Changes / Verification, matching the catalog's shape):

- **Summary**: `fab operator` and its verbs are gone; the operator lives in HexoKit (`rk operator`,
  `rk skill operator`, `rk operator autopilot`). The `/fab-operator` skill is removed (project and
  user level); open the operator with `rk operator` / `rk skill operator`. The user-level `fab-operator` skill under `~/`
  is no longer written and `fab sync` removes fab-generated copies. `autopilot.merge_mode` is retired.
- **Pre-check**: does `fab/project/config.yaml` (or `fab config show --system`) set
  `autopilot.merge_mode`? Did the user still run the legacy fab operator (no capable rk)?
- **Changes** (manual, spelled out — fab cannot write rk's settings):
  1. If `autopilot.merge_mode` was set to `<value>`, run `rk settings set operator.merge_mode <value>`;
     otherwise rk's autopilot silently uses `cherry-pick-ladder`.
  2. Delete the now-inert `autopilot:` block from `fab/project/config.yaml` / the system config
     (`fab config unset --system autopilot.merge_mode` will no longer recognise the key — edit the file).
  3. If fab's operator state file still holds items not yet imported, run `rk operator adopt`
     (it only reads fab's file).
- **Verification**: `fab operator` → unknown command; `/fab-operator` no longer exists in the project; `~/.agents/skills/fab-operator/` and `~/.claude/skills/fab-operator/` are gone after
  `fab sync` (remove them by hand if they were edited);
  `rk settings get operator.merge_mode` shows the intended mode.

The migration performs no automated file edits.

### 6. Constitution, memory, specs, site docs

- `fab/project/constitution.md`:
  - Line ~18 (Principle on kit/user layout): **remove** the sentence about the machine carrying one
    user-level `fab-operator` skill — fab writes nothing under `~/` skills any more.
  - Line ~33 (CLI ⇒ docs rule): partial list becomes `_cli-fab.md` (core, incl. `fab agent`) and
    `_cli-fab-pane.md` (`fab pane`, `fab dispatch`) — drop `_cli-fab-operator.md`.
  - Line ~34: **remove** the user-level `fab-operator` pointer clause.
  - Add an amendment comment in the file's existing style (`<!-- 2026-11-17 (261117-egu4): … -->`).
- Memory (hydrate stage) — see Affected Memory.
- Specs: `docs/specs/operator.md` → reduce to a short stub (operator owned by HexoKit; fab keeps the
  `operator` agent role; no `/fab-operator` skill at any level) or remove and drop
  from `docs/specs/index.md`; update `skills.md`, `architecture.md`, `config.md` (remove
  § Autopilot merge-mode preference and the `autopilot.merge_mode` rows in the scope/advertise
  tables), `glossary.md`, `overview.md`, `harness-adapters.md`, `stage-models.md` where they describe
  `fab operator`. `docs/specs/findings/*` are historical — leave.
- `docs/site/*` (`skill.md`, `workflows.md`, `install.md`, `merge-topologies.md`) and `README.md`:
  remove `fab operator` usage; point to `rk operator`.

### 7. Verification

- `cd src/go/fab && go build ./... && go test ./...`; same for `src/go/fab-kit`.
- `grep -rn "fab operator\|_cli-fab-operator\|_operator-legacy\|autopilot.merge_mode\|user-skill-fab-operator" src/ docs/specs docs/site README.md fab/project`
  returns only intentional hits (migration files, historical findings, this change).
- `fab sync` in a scratch project writes no user-level skill, and removes a pre-existing
  fab-generated `~/.agents/skills/fab-operator/SKILL.md` (test with a temp `$HOME`).

## Affected Memory

- `runtime/operator`: (modify) collapse to present truth — the operator is owned by HexoKit; fab keeps
  no `/fab-operator` skill at all (project or user level), only the `operator` agent role for
  `fab agent operator -o yaml`, and fab's state file is left on disk for `rk operator adopt`. Remove all
  verb/tick/clock/track/autopilot content.
- `runtime/index`: (modify) operator row + domain description.
- `runtime/agent-primitives`: (modify) drop `fab operator` launcher/role references beyond the kept role.
- `runtime/dispatch`: (modify) drop `fab operator` mentions.
- `runtime/providers-and-profiles`: (modify) drop operator-launcher exception; `fab agent` doc lives in
  `_cli-fab.md`.
- `distribution/kit-architecture`: (modify) skills/partials list (no `_operator-legacy`,
  `_cli-fab-operator`), user-level pointer and its template removed, sync cleans stale copies.
- `distribution/setup`: (modify) `fab sync` no longer deploys a user-level skill; removes stale
  fab-generated copies; `user-skills` setupcheck probe removed.
- `distribution/distribution`: (modify) operator references.
- `distribution/migrations`: (modify) catalog entry for the new migration.
- `_shared/configuration`: (modify) remove `autopilot.merge_mode`.
- `_shared/context-loading`: (modify) remove `/fab-operator` reduced-set and `_cli-fab-operator` helper.
- `pipeline/preflight`, `pipeline/change-lifecycle`: (modify) drop stale operator references if any
  describe current behaviour.
- `docs/memory/index.md`: (modify) runtime row description.

(`log.md` / `log.seed.md` files are historical logs — append per hydrate convention, don't rewrite.)

## Impact

- **CLI surface removed**: `fab operator` (bare launcher and `track`, `tick-start`, `clock sync`,
  `state`, `branch-map rm`, `time`). Breaking for anyone still scripting them; HexoKit replacements
  exist (`rk operator`, `rk operator track|check`, `rk cron …`).
- **Config surface removed**: `autopilot.merge_mode` (project + system). Unknown-key semantics apply.
- **Skills**: `_operator-legacy`, `_cli-fab-operator` removed; `fab-operator` removed too (no `/fab-operator` at any level). `fab sync` deploys removed partials no more — stale deployed copies in `.agents/skills/`
  are handled by sync's existing manifest/cleanup (verify).
- **HexoKit contract kept**: `fab agent operator -o yaml [--headless]`, `.status.yaml` reads,
  `fab status set-base-branch`, `/fab-new` / `/fab-proceed` spawn prompts.
- **HexoKit contract broken on purpose**: `rk operator` typing `/fab-operator` (anywhere — project or
  `$HOME`) no longer resolves a skill — user decision; run-kit is dropping that kickoff separately.
- Packages: `src/go/fab` (cmd/fab, internal/config, configref, configscope, agent, setupcheck),
  `src/go/fab-kit` (internal/skills.go + tests), `src/kit/skills`, `src/kit/templates`,
  `src/kit/migrations`, `fab/project`, `docs/`.
- Size: large deletion, modest edits. Speed over polish per the user; review should still catch
  compile/test breaks and dangling references.

## Open Questions

- Should `docs/specs/operator.md` be reduced to a stub or removed outright? (Defaulting to a short stub.)
- Should the bare `/fab-operator` failure path (no capable rk) suggest an install command? (Defaulting
  to a plain "requires HexoKit rk with `rk skill operator`; install or upgrade rk" message.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete all `cmd/fab/operator*.go` + tests and the `fab operator` command; no shim | Plan F2 verbatim; grep shows only `exitCodeError` (main.go) is a real cross-file dependency, the rest are comments/tests | S:90 R:70 A:90 D:90 |
| 2 | Certain | Keep `fab agent` (incl. the `operator` role), `fab pane`, `fab dispatch`, `fab status set-base-branch` | Plan F2 + parent plan's "fab touchpoints HexoKit keeps"; run-kit `rk operator` calls `fab agent operator -o yaml` | S:95 R:80 A:95 D:95 |
| 3 | Certain | Delete `_operator-legacy.md` and `_cli-fab-operator.md`; move § fab agent into `_cli-fab.md` | Plan names the operator skill and `_cli-fab-operator.md`; § fab agent documents a kept command and is cited by `_preamble` | S:85 R:80 A:85 D:80 |
| 4 | Certain | Delete the project-level `fab-operator` skill entirely (and its help row / catalog mentions) | Clarified — user: "rk operator has also been informed to stop typing /fab-operator … So you can delete the project level fab-operator" | S:95 R:75 A:90 D:90 |
| 5 | Certain | Drop the user-level skill entirely: delete template, `deployUserOperatorSkill`, the `user-skills` probe; `rk operator` typing `/fab-operator` outside a project stops working by design | Clarified — user: "Drop the user level skill entry. rk operator should stop working in case it still types /fab-operator" | S:95 R:75 A:90 D:90 |
| 6 | Confident | Retire `autopilot.merge_mode` from config struct, defaults, configref, configscope, validate, docs; existing blocks ignored silently as unknown keys | Restructure plan F2 + `merge_mode` location decision; yaml.v3 unknown-key precedent (`fab_version:`) | S:85 R:75 A:80 D:70 |
| 7 | Tentative | New migration slot `2.29.0-to-2.30.0.md` (release-owned range), manual steps only | `2.28.16-to-2.29.0` is unreleased and ranges must not overlap; folding into the F1 file is the alternative | S:40 R:70 A:40 D:35 |
| 8 | Certain | Migration spells out `rk settings set operator.merge_mode <value>` and `rk operator adopt`; binary moves no data | Restructure plan F2 text verbatim; F1 migration precedent | S:90 R:85 A:80 D:80 |
| 9 | Confident | Leave fab's operator state file on disk; never delete it | `rk operator adopt` reads it; F1 rollback relies on it being untouched | S:70 R:60 A:80 D:80 |
| 10 | Confident | Amend constitution lines on the user-level pointer and the CLI-partials list | Constitution names `_cli-fab-operator.md` and the user-level pointer; leaving them makes the rule unsatisfiable | S:70 R:80 A:80 D:75 |
| 11 | Confident | `docs/specs/operator.md` reduced to a short stub rather than deleted | Specs are human-curated; a stub keeps the index link and says where the operator went | S:50 R:90 A:65 D:55 |
| 13 | Confident | `fab sync` removes previously written fab-generated user-level copies (marker-matched; foreign files kept; best-effort) | Follows from the user's "should stop working" — stale copies would keep resolving `/fab-operator`; marker match avoids deleting user-authored files | S:65 R:75 A:70 D:60 |
| 12 | Certain | Plan's user gate and "C1 + 1 release" wait are waived | User explicitly waived them; speed over polish | S:95 R:70 A:95 D:95 |

13 assumptions (6 certain, 6 confident, 1 tentative, 0 unresolved).
