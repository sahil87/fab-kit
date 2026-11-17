# Plan: Delete fab's Operator (Plan Step F2)

**Change**: 261117-egu4-delete-fab-operator
**Intake**: `intake.md`

## Requirements

### CLI: `fab operator` removal

#### R1: The `fab operator` command family is removed
The `fab` binary SHALL NOT register an `operator` command. Every `src/go/fab/cmd/fab/operator*.go`
file (sources and tests) SHALL be deleted. Symbols still needed by other files (`exitCodeError`
used at `main.go:117`, plus any helper a compile break reveals) MUST be moved next to their
remaining caller, never kept in an `operator*` file. Comments in other files citing deleted
symbols (`pr_sync.go:16` `rkOperatorPath`, `pane_map.go:176` `operatorStatePathOverride`,
`dispatch.go:33` `fab operator`) SHALL be reworded.

- **GIVEN** a build of `src/go/fab`
- **WHEN** a user runs `fab operator track list`
- **THEN** cobra reports an unknown command and exits non-zero
- **AND** `go build ./... && go test ./...` pass in `src/go/fab`

#### R2: Kept surfaces keep working
`fab agent` (including the `operator` role, `RoleOperator`, resolved by `fab agent operator -o yaml`),
`fab resolve-agent`, `fab pane *`, `fab dispatch *` and `fab status set-base-branch` SHALL be
unchanged in behaviour. Only comments in `internal/agent/agent.go` that cite the deleted
`fab operator` launcher SHALL be reworded.

- **GIVEN** the change applied
- **WHEN** `fab agent operator -o yaml` runs
- **THEN** it resolves exactly as before (existing `agent` tests pass unchanged in assertions)

#### R3: Operator state file untouched
fab SHALL NOT delete or modify `$XDG_STATE_HOME/fab/operator/<slug>.yaml`; no code path remains
that reads or writes it.

- **GIVEN** an existing fab operator state file
- **WHEN** any fab command or `fab sync` runs
- **THEN** the file is unchanged (HexoKit's `rk operator adopt` may still read it)

### Kit: skills and partials

#### R4: Operator skill partials deleted, `fab agent` docs relocated
`src/kit/skills/_operator-legacy.md` and `src/kit/skills/_cli-fab-operator.md` SHALL be deleted.
The `## fab agent` section of `_cli-fab-operator.md` SHALL move verbatim (adjusted only for
removed operator references) into `src/kit/skills/_cli-fab.md` as `## fab agent` (with a Contents
entry). Every citation of "`_cli-fab-operator.md` § fab agent" SHALL point to "`_cli-fab.md` §
fab agent". `_cli-fab-operator` SHALL be removed from `_preamble.md` § Skill Helper Declaration
"Allowed values".

- **GIVEN** the kit after the change
- **WHEN** grepping `src/kit/skills` for `_cli-fab-operator` or `_operator-legacy`
- **THEN** there are no hits

#### R5: Operator-only prose removed from shared partials
`_preamble.md`, `_cli-external.md`, `_cli-agents.md`, `_cli-fab-pane.md`, `_cli-fab.md`,
`fab-incognito.md`, `fab-setup.md` SHALL drop every reference to `fab operator` verbs,
`_operator-legacy`, the operator launcher exception, the `/fab-operator` reduced context set /
status frame examples, and the user-level pointer kit-path fallback — keeping content still used
by dispatch/pane (e.g. `_cli-agents.md` § Pre-Send Validation, `rk notify` usage by the pipeline's
escalation). Deployed content MUST NOT cite fab-kit-only paths (`docs/`, `src/go/`).

- **GIVEN** the kit after the change
- **WHEN** grepping `src/kit/skills` for `fab operator `
- **THEN** no hit describes a live command (only `/fab-operator` pointer mentions remain)

#### R6: `/fab-operator` is an rk-only project-level pointer *(superseded by R14)*
`src/kit/skills/fab-operator.md` SHALL keep its name and Step 1 probe
(`command -v rk >/dev/null 2>&1 && rk skill operator`; success = exit 0 and first heading
`# HexoKit skill: operator`) and Step 2 (follow rk's output; the `rk operator adopt --dry-run`
hint MAY stay). Step 3 SHALL NOT fall back to any legacy skill: it tells the user the operator lives
in HexoKit and needs an rk that serves `rk skill operator` (install or upgrade rk), then stops. The
frontmatter `description` SHALL not mention a legacy fallback.

- **GIVEN** rk is absent
- **WHEN** a user runs `/fab-operator` in a fab project
- **THEN** the agent reports that HexoKit rk with `rk skill operator` is required and stops

### Distribution: user-level skill dropped

#### R7: No user-level skill is deployed
`src/kit/templates/user-skill-fab-operator.md` SHALL be deleted, along with
`deployUserOperatorSkill` and its call site in `src/go/fab-kit/internal/skills.go` (keep
`writeFileAtomic` only if still used elsewhere). `fab sync` SHALL write nothing under
`~/.agents/skills/` or `~/.claude/skills/`. `rk operator` typing `/fab-operator` outside a fab project
stops resolving — by user decision.

- **GIVEN** a temp `$HOME` with no `fab-operator` skill
- **WHEN** `fab sync` runs in a project
- **THEN** no `~/.agents/skills/fab-operator/` or `~/.claude/skills/fab-operator/` is created

#### R8: `fab sync` removes stale fab-generated user-level copies
`fab sync` SHALL remove `~/.agents/skills/fab-operator/SKILL.md` and
`~/.claude/skills/fab-operator/SKILL.md` when the file exists, is a regular file (not a symlink),
and contains the fab-generated marker string ``Generated by `fab sync` `` together with frontmatter
`name: fab-operator`; it SHALL then remove the `fab-operator` directory if it is empty. Files
without the marker, symlinks, and non-empty directories SHALL be left alone. On removal sync prints
`User-level skill (fab-operator): removed (retired)`. Removal errors are warnings printed to stderr,
never a sync failure. The claude-availability gate does not apply to cleanup (both tiers are checked).

- **GIVEN** `~/.claude/skills/fab-operator/SKILL.md` written by an older `fab sync`
- **WHEN** `fab sync` runs
- **THEN** the file and its now-empty directory are removed
- **GIVEN** a user-authored `~/.agents/skills/fab-operator/SKILL.md` without the marker
- **WHEN** `fab sync` runs
- **THEN** it is left untouched

#### R9: The `user-skills` setup check is removed
`ProbeUserSkills` in `src/go/fab/internal/setupcheck/probes.go`, its registration in
`setupcheck.go`, and its tests SHALL be removed.

- **GIVEN** the change applied
- **WHEN** the setup check runs
- **THEN** no `user-skills` check is reported and setupcheck tests pass

### Config: `autopilot.merge_mode` retired

#### R10: `autopilot.merge_mode` is removed from every fab surface
`AutopilotConfig`, the config struct's `Autopilot` field, `DefaultAutopilotMergeMode`,
`ValidAutopilotMergeModes`, `GetAutopilotMergeMode` (`internal/config/config.go`), the defaults push
(`internal/agent/agent.go` ~L94–117), the `autopilot:` block in `src/go/fab/defaults.yaml`, the
configref entry (`internal/configref/configref.go` + `defaultsmap_test.go`), the configscope row and
key (`internal/configscope/configscope.go` + test), the validate case
(`cmd/fab/config_validate.go:30`), and related cases in `cmd/fab/config*_test.go` SHALL be removed.
The commented `autopilot:` block in `fab/project/config.yaml` (and any scaffold config template in
`src/kit/`) SHALL be removed.

- **GIVEN** a config file still containing `autopilot: {merge_mode: stacked-prs}`
- **WHEN** any fab command loads config
- **THEN** the block is ignored silently (yaml.v3 unknown key) — no error
- **GIVEN** the change applied
- **WHEN** a user runs `fab config set autopilot.merge_mode merge-auto`
- **THEN** it fails as an unknown key

### Migrations

#### R11: Migration note for the retirement
A new file `src/kit/migrations/2.29.0-to-2.30.0.md` SHALL follow the catalog's Summary / Pre-check /
Changes / Verification shape, state that the range is release-owned, perform no automated edits, and
spell out: (1) `rk settings set operator.merge_mode <value>` for anyone who set
`autopilot.merge_mode`, otherwise rk's autopilot silently uses `cherry-pick-ladder`; (2) delete the
inert `autopilot:` block by hand; (3) `rk operator adopt` for un-imported items in fab's state file;
(4) `fab operator` is gone (use `rk operator`), `/fab-operator` is rk-only and project-level, and
`fab sync` removes fab-generated user-level copies. Verification: `fab operator` is unknown;
`/fab-operator` in a project opens rk's skill; the `~/…/fab-operator/` dirs are gone after sync;
`rk settings get operator.merge_mode` shows the intended mode. `fab migrations-status` MUST report
no overlaps.

- **GIVEN** the migrations directory after the change
- **WHEN** `DiscoverMigrations` scans it (existing migrations tests)
- **THEN** no overlap is reported

### Governance and docs

#### R12: Constitution and code-quality updated
`fab/project/constitution.md` SHALL drop the user-level `fab-operator` skill sentence (~L18) and
clause (~L34), and list only `_cli-fab.md` (core, incl. `fab agent`) and `_cli-fab-pane.md`
(`fab pane`, `fab dispatch`) as CLI reference partials (~L33, and any duplicate around L123), with an
amendment comment `<!-- 2026-11-17 (261117-egu4): … -->` in the file's existing style.
`fab/project/code-quality.md` anti-patterns (L17, L19) SHALL be updated the same way.

- **GIVEN** the change applied
- **WHEN** grepping `fab/project/` for `_cli-fab-operator` or `user-skill-fab-operator`
- **THEN** only the amendment comment mentions them

#### R13: Specs, site docs and README updated
`docs/specs/operator.md` SHALL become a short stub (operator owned by HexoKit — `rk operator`,
`rk skill operator`, `rk operator autopilot`; fab keeps the project-level `/fab-operator` pointer and
the `operator` agent role; no user-level skill). `docs/specs/skills.md`, `architecture.md`,
`config.md` (remove § Autopilot merge-mode preference and `autopilot.merge_mode` rows),
`glossary.md`, `overview.md`, `harness-adapters.md`, `stage-models.md`, `docs/site/*.md`, `README.md`
and `src/go/fab/cmd/fab/skill.md` SHALL no longer describe `fab operator` verbs or the user-level
pointer as live. `docs/specs/findings/*` and migration files stay as historical record.

- **GIVEN** the change applied
- **WHEN** grepping `docs/specs docs/site README.md src/go/fab/cmd/fab/skill.md` (excluding `findings/`) for `fab operator \|autopilot.merge_mode\|user-skill-fab-operator`
- **THEN** remaining hits are only "removed / moved to HexoKit" statements

#### R14: The `/fab-operator` skill is deleted entirely
**Supersedes R6** (user decision after apply started: run-kit is dropping its `/fab-operator` kickoff,
so fab ships no `/fab-operator` skill at any level). `src/kit/skills/fab-operator.md` SHALL be deleted.
Its row in `src/go/fab/cmd/fab/fab_help.go` (+ `fab_help_test.go`) and every skill catalog / listing
that names it (`_preamble.md`, `fab-incognito.md`, `fab-help` content, `docs/specs/skills.md` incl. the
`## /fab-operator` section and helper table row, `docs/specs/{overview,glossary,architecture,operator,superpowers-comparison}.md`,
`docs/site/*.md`, `README.md`, `src/go/fab/cmd/fab/skill.md`) SHALL drop it or state it was removed in
favour of `rk operator` / `rk skill operator`. Every remaining description of a live "machine-level /
user-level `fab-operator` pointer" written by `fab sync` (e.g. `src/kit/skills/fab-setup.md:95`,
`src/kit/skills/_cli-fab.md:~699` setup-check row, `docs/site/install.md:124`, `docs/site/skill.md:118`,
`src/go/fab/cmd/fab/skill.md:118`, `docs/specs/architecture.md:500`) SHALL be removed. `fab sync` SHALL
prune a project's deployed `.agents/skills/fab-operator/` and `.claude/skills/fab-operator/` copies via
its existing manifest pruning (verify with a test or an existing one; fix if pruning does not cover a
removed kit skill). `spawn_test.go` sample strings and `launcher_prompt_test.go` may keep
`fab-operator` only as inert sample text. The user-level stale-copy cleanup (R8) stays.

- **GIVEN** the kit after the change
- **WHEN** listing `src/kit/skills/`
- **THEN** no `fab-operator.md` exists and `fab help` lists no `fab-operator`

### Non-Goals

- Changing HexoKit/run-kit (e.g. making `rk operator` stop typing `/fab-operator`) — another repo.
- Deleting fab's operator state file or its directory.
- Hydrating `docs/memory/` during apply — the hydrate stage owns memory.
- Rewriting historical `log.md` / `findings/*` / older migration files.

### Design Decisions

#### Drop the user-level `fab-operator` skill and clean stale copies
**Decision**: `fab sync` stops writing `~/.{agents,claude}/skills/fab-operator/SKILL.md` and removes
copies carrying its own generated-by marker.
**Why**: User decision — `rk operator` typing `/fab-operator` outside a project should stop working;
rk owns the operator's entry point. Leaving stale copies would keep it working silently.
**Rejected**: Deploying `fab-operator.md` itself as the user-level skill (keeps a seam the plan
removes); deleting without cleanup (stale pointers keep resolving).
*Introduced by*: 261117-egu4-delete-fab-operator

#### New migration slot rather than folding into F1's file
**Decision**: `2.29.0-to-2.30.0.md`, range release-owned.
**Why**: `2.28.16-to-2.29.0.md` is unreleased and ranges must not overlap.
**Rejected**: `2.28.18-to-2.29.0` (overlaps); editing F1's file (mixes two steps' notes).
*Introduced by*: 261117-egu4-delete-fab-operator

### Deprecated Requirements

#### `fab operator` verbs (track, tick-start, clock sync, state, branch-map rm, time, bare launcher)
**Reason**: HexoKit owns the operator.
**Migration**: `rk operator`, `rk operator track|check`, `rk cron …`, `rk operator autopilot`.

#### `autopilot.merge_mode`
**Reason**: rk reads `operator.merge_mode` from rk settings; fab's key was never read.
**Migration**: `rk settings set operator.merge_mode <value>`.

#### User-level `fab-operator` pointer skill
**Reason**: User decision; rk owns its entry point.
**Migration**: N/A (`fab sync` removes fab-generated copies).

## Tasks

### Phase 1: Go — delete the operator command

- [x] T001 Delete all `src/go/fab/cmd/fab/operator*.go` (sources + tests); remove `operatorCmd()` from `src/go/fab/cmd/fab/main.go`; move `exitCodeError` (type + `Error()`) into `main.go` (or drop the branch if dead); fix `main_test.go` / `launcher_prompt_test.go` references; reword comments in `pr_sync.go`, `pane_map.go`, `dispatch.go`, `internal/agent/agent.go`; review `fab_help.go` (+ test) for operator rows (keep `fab-operator` skill row). Fix any compile break by relocating the helper to its caller. `cd src/go/fab && go build ./... && go vet ./cmd/fab && go test ./...` green <!-- R1 -->
- [x] T002 Verify kept surfaces: run `go test ./internal/agent/... ./cmd/fab/ -run 'Agent|Pane|Dispatch|Status|ResolveAgent'` and confirm no assertion changes were needed in agent/pane/dispatch/status tests; confirm no remaining code references the operator state path <!-- R2, R3 -->

### Phase 2: Go — config and sync

- [x] T003 Retire `autopilot.merge_mode`: `internal/config/config.go`, `internal/agent/agent.go` (defaults push), `src/go/fab/defaults.yaml`, `internal/configref/configref.go` + `defaultsmap_test.go`, `internal/configscope/configscope.go` + test, `cmd/fab/config_validate.go`, `cmd/fab/config.go`/`config_test.go`/`config_json_test.go`, `internal/config/config_test.go`; add a test that a config containing an `autopilot:` block still loads without error <!-- R10 -->
- [x] T004 `src/go/fab-kit/internal/skills.go`: delete `deployUserOperatorSkill` + call site; add the stale-copy cleanup per R8 (both tiers, marker + `name: fab-operator` match, regular file only, rmdir if empty, warning-only errors, `removed (retired)` line); update `skills_test.go` and `sync_integration_test.go` (drop deploy tests; add cleanup tests: generated copy removed, unmarked file kept, symlink kept). `cd src/go/fab-kit && go build ./... && go test ./...` green <!-- R7, R8 -->
- [x] T005 [P] Remove `ProbeUserSkills` from `src/go/fab/internal/setupcheck/probes.go`, its registration in `setupcheck.go`, and its tests in `setupcheck_test.go` <!-- R9 -->

### Phase 3: Kit content

- [x] T006 Move `## fab agent` from `src/kit/skills/_cli-fab-operator.md` into `src/kit/skills/_cli-fab.md` (+ Contents entry); delete `_cli-fab-operator.md` and `_operator-legacy.md`; delete `src/kit/templates/user-skill-fab-operator.md`; remove `autopilot.merge_mode` docs from `_cli-fab.md` <!-- R4, R7, R10 -->
- [x] T007 Rewrite `src/kit/skills/fab-operator.md` as the rk-only pointer per R6 (frontmatter description updated) <!-- R6 -->
- [x] T008 Sweep `_preamble.md`, `_cli-external.md`, `_cli-agents.md`, `_cli-fab-pane.md`, `_cli-fab.md`, `fab-incognito.md`, `fab-setup.md` (and any other `src/kit/skills/*.md` hit) for `fab operator`, `_cli-fab-operator`, `_operator-legacy`, user-level pointer / kit-path fallback for neutral directories, operator launcher exception; repoint `§ fab agent` citations to `_cli-fab.md`; remove `_cli-fab-operator` from the allowed helpers list. Keep dispatch/pane-used content <!-- R4, R5 -->
- [x] T009 Remove the commented `autopilot:` block from `fab/project/config.yaml` and from any scaffold config under `src/kit/` <!-- R10 -->
- [x] T010 Write `src/kit/migrations/2.29.0-to-2.30.0.md` per R11; run `fab migrations-status` (or the fab-kit migrations tests) to confirm no overlap <!-- R11 -->

### Phase 4: Governance and docs

- [x] T011 Update `fab/project/constitution.md` and `fab/project/code-quality.md` per R12 <!-- R12 -->
- [x] T012 Update specs/site/README/`cmd/fab/skill.md` per R13: stub `docs/specs/operator.md`; edit `docs/specs/{skills,architecture,config,glossary,overview,harness-adapters,stage-models}.md`, `docs/site/{skill,workflows,install,merge-topologies}.md`, `README.md`, `src/go/fab/cmd/fab/skill.md` <!-- R13 -->
- [x] T013 Final sweep: `grep -rn "fab operator \|_cli-fab-operator\|_operator-legacy\|autopilot.merge_mode\|user-skill-fab-operator\|deployUserOperatorSkill\|ProbeUserSkills" src/ fab/project docs/specs docs/site README.md --exclude-dir=findings` — only intentional hits remain (migrations, "removed" statements, amendment comments); run full `go test ./...` in both `src/go/fab` and `src/go/fab-kit`, plus any kit content tests (e.g. the deployed-content fab-kit-path guard) <!-- R1, R4, R5, R10, R13 -->

### Phase 5: Amendment (user decision after apply)

- [x] T014 Delete `src/kit/skills/fab-operator.md` and sweep every listing/description per R14 (help row + test, skill catalogs, specs, site, README, `cmd/fab/skill.md`, `fab-setup.md:95`, `_cli-fab.md` setup-check row, `docs/specs/architecture.md:500`); confirm `fab sync` prunes the deployed `fab-operator` copy (test); rerun `go test ./...` in `src/go/fab` and `src/go/fab-kit` <!-- R14 -->

## Execution Order

- T001 before T002; T003 after T001 (shared files in `cmd/fab`).
- T006 before T008 (citations repoint to the moved section).
- T013 last.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: No `operator*.go` file exists under `src/go/fab/cmd/fab/`; `main.go` registers no operator command; `fab operator` is an unknown command
- [ ] A-002 R2: `fab agent operator -o yaml`, `fab pane`, `fab dispatch`, `fab status set-base-branch` behave as before; their tests pass without assertion changes
- [ ] A-003 R4: `_operator-legacy.md` and `_cli-fab-operator.md` are deleted; `_cli-fab.md` carries `## fab agent`; no `_cli-fab-operator` citation remains in `src/kit`
- [ ] A-004 R6: `fab-operator.md` probes `rk skill operator`, follows it on success, and on failure reports rk is required and stops — no legacy fallback
- [ ] A-005 R7: The user-level template and `deployUserOperatorSkill` are gone; `fab sync` writes nothing under `~/` skills
- [ ] A-006 R8: `fab sync` removes marker-matched user-level copies and leaves unmarked files and symlinks; tests cover all three
- [ ] A-007 R9: `ProbeUserSkills` and its registration/tests are removed
- [ ] A-008 R10: `autopilot.merge_mode` appears in no Go source, defaults, configref, configscope, validate path, or project config
- [ ] A-009 R11: `2.29.0-to-2.30.0.md` exists with the four sections and the `rk settings set operator.merge_mode <value>` step; no migration overlap
- [ ] A-010 R12: Constitution and code-quality list only `_cli-fab.md` / `_cli-fab-pane.md` and no user-level pointer; amendment comment present
- [ ] A-024 R14: `src/kit/skills/fab-operator.md` is deleted; no skill catalog, help row, spec, site page or README lists `/fab-operator` as a live skill; no text describes a live user-level `fab-operator` pointer; sync pruning of the deployed copy is verified
- [ ] A-011 R13: `docs/specs/operator.md` is a stub; specs/site/README describe no live `fab operator` verbs or user-level pointer

### Removal Verification

- [ ] A-012 R1: No dead helpers left over from the operator files (nothing unused that only operator code called)
- [ ] A-013 R10: No residual `Autopilot*` identifiers in `src/go/fab`

### Scenario Coverage

- [ ] A-014 R10: A test loads a config containing an `autopilot:` block without error
- [ ] A-015 R8: Cleanup tests use a temp `$HOME` and assert removed / kept outcomes

### Edge Cases & Error Handling

- [ ] A-016 R8: A cleanup failure (e.g. permission denied) is a warning and `fab sync` still exits 0
- [ ] A-017 R3: No code path deletes or writes the fab operator state file

### Code Quality

- [ ] A-018 Pattern consistency: New/edited Go follows surrounding naming, error-wrapping and test-helper patterns
- [ ] A-019 No unnecessary duplication: Existing utilities reused where applicable
- [ ] A-020 Owner-or-pointer: Edited skill files state a rule or point at its owner, never both
- [ ] A-021 No fab-kit-only paths cited from deployed `src/kit/**` content
- [ ] A-022 CLI changes ship with reference-partial and test updates (Constitution Additional Constraints)
- [ ] A-023 Config-data change ships with a `src/kit/migrations/` file (no ad-hoc script)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | User-level skill dropped; `rk operator` from a neutral dir stops resolving `/fab-operator` | Clarified by the user after intake | S:95 R:75 A:90 D:90 |
| 2 | Confident | Stale-copy cleanup matches the template's `Generated by \`fab sync\`` marker plus `name: fab-operator`; symlinks and unmarked files kept; errors are warnings | Avoids deleting user-authored files; mirrors the old deploy's symlink caution | S:65 R:75 A:70 D:60 |
| 3 | Confident | `exitCodeError` moves to `main.go` rather than being dropped, unless apply finds the branch dead | Only real cross-file dependency found by grep | S:70 R:90 A:75 D:70 |
| 4 | Confident | Unknown `autopilot:` blocks ignored silently, no warning | yaml.v3 unknown-key precedent (`fab_version:`) | S:75 R:85 A:80 D:70 |
| 5 | Tentative | Migration slot `2.29.0-to-2.30.0.md` | Non-overlap with the unreleased F1 slot; release owner may renumber | S:40 R:70 A:40 D:35 |
| 6 | Confident | `docs/specs/operator.md` reduced to a stub, not deleted | Keeps the index link; specs are human-curated | S:50 R:90 A:65 D:55 |
| 7 | Certain | `fab-operator.md` deleted; R14 supersedes R6 | User decision after apply started | S:95 R:75 A:90 D:90 |

7 assumptions (2 certain, 4 confident, 1 tentative).
