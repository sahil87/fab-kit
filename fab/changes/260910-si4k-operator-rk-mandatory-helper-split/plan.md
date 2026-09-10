# Plan: Operator loading and dependency — rk-mandatory skill, `_cli-fab.md` split by command family, live clock cadence

**Change**: 260910-si4k-operator-rk-mandatory-helper-split
**Intake**: `intake.md`

<!-- Co-generated inline by /fab-fff at apply entry (2026-09-11). Sections ## Requirements,
     ## Tasks, ## Acceptance are the parser contract. Task IDs T{NNN}, acceptance A-{NNN},
     requirements R#. No [NEEDS CLARIFICATION] markers — graded decisions are in ## Assumptions. -->

## Requirements

Everything in this change is markdown under `src/kit/skills/`, `fab/project/`, and `docs/`. No `.go` file changes. `.agents/skills/` and `.claude/skills/` are deployed copies and are never edited directly. The §6 choreography of `fab-operator.md` (Dependency Resolution, Dependency Declaration, Working a Change, Autopilot and everything under it) is untouched by this change.

### Operator skill: run-kit dependency

#### R1: One rk gate at startup
`fab-operator.md` §2 Startup SHALL carry a `### rk Gate` subsection, placed immediately after `### Tmux Gate` and before `### Role Mark`, that probes once — `command -v rk >/dev/null 2>&1 && rk cron list --json >/dev/null 2>&1` — and STOPs with `Error: the operator requires run-kit — brew install sahil87/tap/run-kit` when either half fails. The subsection MUST state that this is the operator's deliberate exception to `_preamble.md` § Run-Kit (rk) Reference's fail-silent rule (rk is the operator's substrate, not an optional enhancement). No later call site in the skill is individually gated.

- **GIVEN** an operator started on a machine without rk, or with an rk predating `rk cron`
- **WHEN** §2 Startup runs
- **THEN** it stops at the rk Gate with the install-hint error and never reaches Role Mark or Init
- **AND** on a machine with a capable rk the gate passes silently and startup continues

#### R2: rk-absent, raw-tmux and version-skew fallback prose is deleted
`fab-operator.md` MUST no longer describe what to do when rk is absent, when an installed rk predates a verb the gate's floor already covers, or when a raw `tmux send-keys` must stand in for an rk verb the operator uses. The deletion set is the intake § 1 table: the §0 intro's "degrading to raw `tmux send-keys`" clause and `command -v rk`-gated qualifiers; the §2 Role Mark `command -v` prefix and version-skew sentence (the `|| true` and the "staleness and radio conflicts are rk's" sentence stay); §2 Init step 4's fail-silent/degraded clauses; §3 Pre-Send Validation item 2's "with rk installed"/"with rk absent" clauses (the `--answer`/`--force` mapping stays); the whole §4 Degraded Fallback subsection plus its Contents entry and the cross-references in §4 Tick Payload and §4 Post-Compaction Reload step 3; the §4 Tick Behavior step 1 two-rung version-skew paragraph; the §5 Notification Send rk-absent paragraph and its four channel bullets (ntfy.sh, Discord webhook, PushNotification, Slack MCP); §5 Sending Auto-Answers' gate qualifiers and three rk-absent clauses; §6 Spawning step 2's `command -v rk` probe framing and the `_rk-*` name-prefix fallback sentence (the exclusion rule itself stays: never the operator's own session, never infrastructure sessions; candidate source is `rk mux sessions --json` `role: "user"` rows); §8 One Operator Per Server's raw-path parenthetical; the §8 Settings `Notify channel` row; §9 Key Properties' "rk-absent fallback launcher" wording and the Cadence row's `/loop` clause. The skill MUST keep one sentence stating that `rk notify` fails silently by contract (a notification that cannot be delivered never crashes or stalls the operator — it logs one line and keeps ticking). References to the Go binary's own fallbacks (the built-in launcher behind bare `fab operator`, `fab pane map`'s tmux enumeration path, dispatch-internal `fab pane capture/kill/process`) are binary behaviour and stay, shortened to pointers at the CLI reference.

- **GIVEN** the edited `fab-operator.md`
- **WHEN** it is grepped for `rk is absent|rk-absent|when rk is absent|raw tmux|send-keys|/loop|Notify channel|version-skew|ntfy|Discord webhook|PushNotification`
- **THEN** every remaining hit is a binary-side fallback pointer, the judgment-round raw-send carve-out owned by `_preamble.md` § The pane readiness gate, or a historical note — none is an operator-skill fallback arm

#### R3: One ready-line template rendered from `rk cron list --json`
§2 Init step 4 SHALL read `rk cron list --json` (already proven working by the gate), select the row whose `target` is `role:operator`, and read its `schedule_summary`, `deliver`, `muted`, and `muted_until` fields. §2 Init step 5 SHALL render exactly one ready-line template — `Operator ready. Clock: rk cron "operator tick" · {schedule_summary} · {deliver}[ · muted[ until HH:MM]]` — copying the JSON values verbatim (` · muted` appended when `muted` is true; ` until HH:MM` in local time appended when `muted_until` is present). The four literal variants (including the `Clock: none` form) are deleted. A missing `operator tick` entry after the gate passed MUST STOP with `Error: no operator-tick cron entry on this server — run rk operator to seed it`. The step-4 unmute rule (issue `rk cron mute <id> --off` when the entry is muted while tracked work exists) is unchanged.

- **GIVEN** the live entry `{"schedule_summary":"backoff 1m→30m","deliver":"immediate","muted":false}`
- **WHEN** Init step 5 renders the ready line
- **THEN** it prints `Operator ready. Clock: rk cron "operator tick" · backoff 1m→30m · immediate`
- **AND** with `muted: true` and `muted_until` set it prints `… · immediate · muted until 14:30`
- **AND** with no `role:operator` row it stops with the seed-it error

#### R4: Compact frame carries the live cadence, re-read each tick
§4 Status Frame Format's compact frame SHALL be `🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · {schedule_summary} · no change[ · {W} waiting]`; the Element/Format table's "Compact frame" row and the fenced documentation example MUST match. The full-frame header line is unchanged. §4 Tick Behavior step 7 SHALL become the per-tick clock read: read `rk cron list --json` once per tick for the `role:operator` row and render its `schedule_summary` on the frame — cadence is never carried from a previous tick or composed from memory; no lifecycle to manage (the tracked-set verbs mute/unmute, rk evaluates the schedule). §9 Key Properties' Cadence row names the live-cadence render.

- **GIVEN** a quiet tick and the live entry above
- **WHEN** the frame renders
- **THEN** the compact line reads `🛰️ **Operator** · 22:06 · tick #5122 · **6 tracked** · backoff 1m→30m · no change`
- **AND** after a user `rk cron edit` changes the schedule, the next tick's line shows the new summary without an operator reload

### Helpers: `_cli-agents` and `_cli-external`

#### R5: `_cli-agents.md` loses its rk-absent arms
`_cli-agents.md` (declared only by `fab-operator`) SHALL be swept under the same rule as R2 — intake § 2 table: § Scope Boundary's "each with a raw-`tmux` fallback" clause (the dispatch-internal sentence stays); § Pre-Send Validation step 2's gate qualifier and rk-absent sentence (`rk mux send` is the gate); step 3's "the rk-absent fallback here," fragment (the step itself stays — judgment rounds and manual driving are live consumers); § Delivery Probe's gate qualifier and "every send when rk is absent included" (the manual recipe stays as the judgment-round procedure); § Peek Output's and § Peek Process-tree's rk-absent clauses; § Await's gate qualifiers, with item 2 renamed to "**Poll** (when no `await` signal fits — e.g. a completion signal that is a screen pattern)" and `rk notify`'s fail-silent note kept without the gate. The frontmatter `description` MUST no longer mention raw-tmux fallbacks.

- **GIVEN** the edited `_cli-agents.md`
- **WHEN** grepped for `rk is absent|when rk is absent|command -v rk|raw tmux|fail open to raw`
- **THEN** no hit remains except the pre-delivery judgment-round mechanics (pane-mode clear guard, manual probe-and-retype recipe), which are kept because `_preamble.md` § The pane readiness gate types into a not-yet-delivered pane by design

#### R6: `_cli-external.md` § /loop is deleted
`_cli-external.md` SHALL delete `## /loop`, its Contents entry, and the "/loop"/"fallback-scoped" phrases in its frontmatter description; its § rk (run-kit) subsections SHALL drop `command -v rk`-gate qualifiers that exist only to describe absence, while § Reference Model's generic `<tool> skill` capability-probe / shll.ai bundle-page delegation stays (tool-version handling for four binaries, not an rk-absent arm). Intake-time grep established that the only consumers of § /loop were the `fab-operator.md` passages R2 deletes; `_cli-agents.md` § Skill Prompts' mention of `/loop` as an example native TUI control is not a consumer and stays.

- **GIVEN** the edited `_cli-external.md` and a repo-wide grep for `/loop` over `src/kit`, `docs/specs`, `fab/project`
- **WHEN** hits are inspected
- **THEN** the only remaining hits are `_cli-agents.md` § Skill Prompts' example mention and historical Design Decisions / archived findings

### CLI reference: split by command family

#### R7: `_cli-fab.md` is split into three files with one home per section
`src/kit/skills/_cli-fab.md` SHALL be cut at its `## ` boundaries: `## fab pane` and `## fab dispatch` move verbatim to a new `src/kit/skills/_cli-fab-pane.md`; `## fab operator` and `## fab agent` move verbatim to a new `src/kit/skills/_cli-fab-operator.md`; every other section (Calling Convention, change, status, score, preflight, log, resolve, resolve-agent, config, doctor, migrations-status, kit-path, setup, shell-init, skill, impact, pr-meta, docs-index, fab-help, help-dump, batch, Common Error Messages) stays in `_cli-fab.md`. Moved text is byte-identical apart from re-pointed cross-references. Each new file carries the partial frontmatter shape of `_cli-fab.md` today (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`), a header blurb naming its sibling files, and a `## Contents` list; `_cli-fab.md`'s header blurb and `## Contents` shrink to the sections it keeps and gain one pointer line naming the two family files. The `## Calling Convention` router line MUST stay in `_cli-fab.md` unchanged (the Go lifecycle-collision test parses it there). Intra-file references that become cross-file after the cut MUST be re-pointed: `## fab resolve-agent` → `_cli-fab-operator.md § fab agent` (×4) and `_cli-fab-pane.md § fab dispatch` (×1); `## fab pane` → `_cli-fab-operator.md § fab operator tick-start`; `## fab dispatch` → `_cli-fab-operator.md § fab agent`. References between `fab pane` and `fab dispatch` stay intra-file.

- **GIVEN** the three files after the cut
- **WHEN** `grep -c '^## fab pane\|^## fab dispatch\|^## fab operator\|^## fab agent'` is run across them
- **THEN** each of the four headings appears exactly once across the set, in the file R7 names
- **AND** `wc -l` of the three files sums to today's 1,475 ± the added frontmatter/header/Contents lines
- **AND** the new deployed files cite no `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` path (Constitution V; the Go portability guard passes)

#### R8: Every pointer into a moved section is re-targeted
Every `_cli-fab.md § fab (pane|dispatch|operator|agent)` pointer (and the `§ reap` / `§ fab dispatch ready` / `§ fab pane · questions` forms) SHALL name the section's new file: `src/kit/skills/fab-operator.md` (13), `_cli-agents.md` (10), `_preamble.md` (6 — § Per-Stage Model Resolution and § CLI-Adapter Dispatch), `_cli-external.md` (4), `fab-archive.md` (1), `fab/project/constitution.md` Principle V example (1 → "`_cli-fab-pane.md` § fab dispatch"), `docs/specs/hooks.md` (2), `docs/specs/stage-models.md` (2), `docs/memory/runtime/pane-commands.md` (2), `docs/memory/distribution/kit-architecture.md` (1). Pointers to sections that stay in core are untouched. `docs/specs/findings/*` is historical and is not edited.

- **GIVEN** a repo-wide grep for `_cli-fab(\.md)?` followed by `§ fab (dispatch|pane|operator|agent)` excluding `fab/changes/`, `docs/wiki/`, `fab/plans/`, `docs/specs/findings/`, `*/log.md`, `*/log.seed.md`
- **WHEN** hits are inspected
- **THEN** there are none: every such pointer names `_cli-fab-pane.md` or `_cli-fab-operator.md`

#### R9: `helpers:` declarations and the allowed-values list
`fab-operator.md` SHALL declare `helpers: [_cli-fab-operator, _cli-fab-pane, _cli-agents, _cli-external]` and its §2 Context Loading paragraph SHALL list those helpers (dropping "/loop" from `_cli-external`'s description). No other skill's `helpers:` list changes: `fab-fff`/`fab-ff`/`fab-continue`/`fab-adopt` declare no `_cli-fab` family today and reach `§ fab dispatch` through `_preamble.md` § CLI-Adapter Dispatch pointers, which R8 re-targets — today's access path, unchanged. `_preamble.md` § Skill Helper Declaration's allowed values SHALL read `_generation`, `_review`, `_cli-fab`, `_cli-fab-operator`, `_cli-fab-pane`, `_cli-external`, `_cli-agents`, `_srad`, `_pipeline`, `_intake` (8 → 10); its other `_cli-fab` mentions (§ Common fab Commands, § Confidence Scoring's `§ fab score`) stay because those sections remain in core.

- **GIVEN** the edited `fab-operator.md` and `_preamble.md`
- **WHEN** an operator reload loads the declared helpers
- **THEN** the CLI reference loaded is `_cli-fab-operator.md` + `_cli-fab-pane.md` (≈585 lines) and not `_cli-fab.md` (1,475)

#### R10: Constitution and its two sibling restatements name the family files
`fab/project/constitution.md` Additional Constraints SHALL read: "Changes to the `fab` CLI (Go binary) MUST include corresponding test updates and MUST update the CLI reference partial that owns the command's family — `src/kit/skills/_cli-fab.md` (core), `_cli-fab-pane.md` (`fab pane`, `fab dispatch`) or `_cli-fab-operator.md` (`fab operator`, `fab agent`) — with any new or changed command signatures". This is a wording amendment: no MUST rule is added or changed, the version stays 1.8.0, `Last Amended` becomes 2026-09-11, and a dated HTML comment `<!-- 2026-09-11 (260910-si4k): … -->` is appended explaining the family-file rename (udwv / jjg0 / t513 / yd9s precedent). The same wording SHALL be applied to `fab/project/code-quality.md` § Anti-Patterns (project-specific) "Changing a CLI command without updating `_cli-fab.md` + tests" and `fab/project/code-review.md` § Project-Specific Review Rules "CLI ⇒ docs + tests".

- **GIVEN** the three project files after edit
- **WHEN** grepped for `_cli-fab.md`
- **THEN** every hit names the core file explicitly as core, or lists all three family files; the constitution's `**Version**` is still `1.8.0`

### Docs: specs and mechanical memory pointers

#### R11: Specs record v10 and stop restating the deleted arms
`docs/specs/operator.md` Version History SHALL gain `v10 — rk-mandatory skill (one startup gate; 27 fallback passages deleted), CLI reference split by command family (_cli-fab-operator / _cli-fab-pane), live cadence from rk cron list --json in the ready line and compact frame`, and "The current operator (v9) evolved through nine iterations" SHALL be updated accordingly. `docs/specs/skills.md` § Skill Helpers SHALL list ten allowed values and the updated `fab-operator` helper mapping; its `/fab-operator` section's lines that restate `command -v rk`-gating, raw `tmux send-keys`, or the Claude-only `/loop` fallback SHALL be rewritten to the rk-mandatory posture; § Adding a skill checklist item 3's helper list SHALL name the family files. `docs/specs/hooks.md` and `docs/specs/stage-models.md` get pointer re-targets only (R8). Mechanical pointer re-targets in `docs/memory/runtime/pane-commands.md` and `docs/memory/distribution/kit-architecture.md` are part of R8; the narrative memory rewrite (operator.md Design Decisions, context-loading allowed values, kit-architecture helper tables, agent-primitives arms) is hydrate's and is listed in intake § Affected Memory.

- **GIVEN** `docs/specs/skills.md` and `docs/specs/operator.md` after edit
- **WHEN** grepped for `/loop` and `command -v rk`
- **THEN** no hit describes an operator fallback; the version table's last row is v10

### Verification

#### R12: The change verifies mechanically before apply finishes
Apply SHALL run, and record the results of: (a) the R2/R5/R6 fallback grep over `src/kit`, `docs/specs`, `fab/project` with the exclusions in R8; (b) the R8 pointer grep; (c) the R7 heading-uniqueness and line-sum checks; (d) `(cd src/go/fab-kit && go test ./cmd/fab/ -run 'TestKitContentCitesNoRepoLocalPaths|TestKitPortabilityMatcher')` and `(cd src/go/fab && go test ./cmd/fab/ -run TestNoTopLevelCommandCollidesWithRouterAllowlist)` (existing tests only — none added or changed; the two module roots are `src/go/fab-kit` and `src/go/fab`); (e) `FAB_KIT_PATH=$PWD/src/kit fab sync` deploys `_cli-fab-operator` and `_cli-fab-pane` into `.agents/skills/` with no warnings (`fab kit-path` otherwise resolves the released kit under `~/.fab-kit/versions/`, which does not contain the new files; both deploy targets are gitignored generated copies, so the sync is safe).

- **GIVEN** the completed edits
- **WHEN** checks (a)–(e) run
- **THEN** all pass and the outcome of each is stated in the apply result summary

### Non-Goals

- Change B of `fab/plans/sahil/26-09-11-operator-generic-tracking.md`: the generic `tracked` item model, `fab operator track` verbs, the one-table frame with Kind/Checked/Next columns, `rk cron edit`-derived schedules, `rk tab new`/`rk tab mark` adoption, `has_agent`, `rk mux capture --classify`, any state-file or tick-document change.
- Any `.go` change; deleting the binary's built-in launcher or `fab pane map`'s tmux fallback; any rk-side probe/wake mechanism (decided against by the user).
- Moving or trimming `fab-operator.md` §6 choreography (user-decided: it stays inline and untouched).
- Editing `_preamble.md` beyond § Skill Helper Declaration's allowed values and the R8 pointer re-targets; editing `_preamble.md` § Run-Kit (rk) Reference (the operator states its exception in its own §2 rk Gate).
- The full-frame header line (only the compact line gains the cadence).

### Design Decisions

#### Two family files, `fab agent` rides with `fab operator`
**Decision**: `_cli-fab-pane.md` (pane + dispatch) and `_cli-fab-operator.md` (operator + agent); no separate `_cli-fab-agent.md`.
**Why**: the only skill that *loads* a family file is `fab-operator`, which uses both `fab operator` and `fab agent`; every pipeline-side consumer of `fab agent` reaches it by pointer and loads nothing, so a third file would split 58 lines for no loader that benefits.
**Rejected**: three files (intake's recommendation) — correct if pipeline skills ever declare a family helper; revisit then. Bundling `fab agent` with pane/dispatch — it is a resolution verb, not a pane primitive.
*Introduced by*: 260910-si4k-operator-rk-mandatory-helper-split

#### Pipeline skills keep pointer-only access to `§ fab dispatch`
**Decision**: `fab-fff`/`fab-ff`/`fab-continue`/`fab-adopt` `helpers:` lists are unchanged; their `_preamble.md` pointers are re-targeted to `_cli-fab-pane.md`.
**Why**: this is a refactor with no behaviour change; today those skills declare no `_cli-fab` family at all and reach dispatch details by pointer. Adding `_cli-fab-pane` unconditionally would add ≈335 lines to every pipeline run on the shipped `dispatch.mode: native` default, which never enters the CLI-adapter branch.
**Rejected**: unconditional declaration (the plan doc's first draft); stage-conditional in-body read (a new loading mechanism for four skills — defer until a run shows the pointer is insufficient).
*Introduced by*: 260910-si4k-operator-rk-mandatory-helper-split

#### A missing operator-tick entry is a STOP, not a clock-less ready line
**Decision**: after the rk Gate passes, no `role:operator` row in `rk cron list --json` stops startup with `Error: no operator-tick cron entry on this server — run rk operator to seed it`.
**Why**: the user asked for the `Clock: none` form to go; the operator cannot tick without the entry, and `rk operator` seeds it idempotently — an actionable stop beats a session that silently never ticks.
**Rejected**: keeping a `Clock: none` variant (the state it describes is non-functional); auto-seeding from the skill (`rk cron add` is the user's and `rk operator`'s, per §4 Ownership).
*Introduced by*: 260910-si4k-operator-rk-mandatory-helper-split

#### Cadence is re-read every tick
**Decision**: Tick Behavior step 7 reads `rk cron list --json` once per tick and the frame renders `schedule_summary` from it.
**Why**: §1 "Re-derive state" and the never-composed-from-memory rule; a user `rk cron edit` must show on the next frame; one cheap local command.
**Rejected**: carrying the Init-time value (goes stale on edit); rendering cadence only in the full frame (the compact line is the one seen on every quiet tick).
*Introduced by*: 260910-si4k-operator-rk-mandatory-helper-split

## Tasks

### Phase 1: Setup

- [x] T001 Split `src/kit/skills/_cli-fab.md`: create `src/kit/skills/_cli-fab-pane.md` (frontmatter + header blurb + Contents, then `## fab pane` and `## fab dispatch` moved verbatim) and `src/kit/skills/_cli-fab-operator.md` (same shape, then `## fab operator` and `## fab agent` moved verbatim); remove those four sections from `_cli-fab.md`, shrink its header blurb and `## Contents` to the kept sections and add one pointer line naming the two family files; keep `## Calling Convention`'s router line byte-identical; re-point the intra-file references that became cross-file (`## fab resolve-agent` → `_cli-fab-operator.md § fab agent` ×4 and `_cli-fab-pane.md § fab dispatch` ×1; `## fab pane` → `_cli-fab-operator.md § fab operator tick-start`; `## fab dispatch` → `_cli-fab-operator.md § fab agent`). Verify with `grep -c` that each moved `## ` heading appears exactly once across the three files and that `wc -l` sums to 1,475 ± the added header lines. <!-- R7 -->

### Phase 2: Core Implementation

- [x] T002 Edit `src/kit/skills/fab-operator.md` for the rk dependency: insert `### rk Gate` after `### Tmux Gate` (probe, STOP text, the exception-to-fail-silent sentence pointing at `_preamble.md` § Run-Kit (rk) Reference); apply every row of the intake § 1 deletion table (§0 intro, §2 Role Mark, §2 Init step 4, §3 Pre-Send item 2, §4 Degraded Fallback + Contents entry + the two cross-references, §4 Tick step 1 version-skew paragraph, §5 Notification Send rk-absent paragraph and four bullets — keeping the one `rk notify` fail-silent sentence, §5 Sending Auto-Answers clauses, §6 Spawning step 2 probe framing and name-prefix fallback, §8 raw-path parenthetical, §8 Settings `Notify channel` row, §9 launcher wording and Cadence `/loop` clause, §9 new `Requires run-kit?` row); leave §6 Dependency Resolution through Autopilot untouched; verify with the R2 grep. <!-- R1, R2 -->
- [x] T003 Edit `src/kit/skills/fab-operator.md` for live cadence: §2 Init step 4 reads the `role:operator` row's `schedule_summary`/`deliver`/`muted`/`muted_until` and STOPs on a missing row with the seed-it error; §2 Init step 5 renders the single template (bare values, ` · muted[ until HH:MM]` suffix rules); §4 Status Frame Format compact line + Element/Format table "Compact frame" row + fenced example gain ` · {schedule_summary}`; §4 Tick Behavior step 7 becomes the per-tick clock read; §9 Cadence row names the live render. Full-frame header unchanged. <!-- R3, R4 -->
- [x] T004 [P] Sweep `src/kit/skills/_cli-agents.md` per the intake § 2 table (Scope Boundary clause, Pre-Send step 2 and step 3 fragments, Delivery Probe qualifiers, Peek Output and Process-tree clauses, Await qualifiers with the "Poll" rename, frontmatter description); keep the judgment-round raw-tmux mechanics; verify with the R5 grep. <!-- R5 -->
- [x] T005 [P] Edit `src/kit/skills/_cli-external.md`: delete `## /loop`, its Contents entry and the description phrases; drop absence-only `command -v rk` qualifiers in § rk (run-kit); keep § Reference Model's `<tool> skill` delegation; verify with the R6 grep. <!-- R6 -->

### Phase 3: Integration & Edge Cases

- [x] T006 <!-- rework: review cycle 1 — 4 stale `_cli-fab.md` pointers into moved sections survived (line-wrapped file/§ refs the grep missed) + architecture.md partials enumeration --> Re-target every pointer into a moved section (R8 file list: `fab-operator.md`, `_cli-agents.md`, `_preamble.md`, `_cli-external.md`, `fab-archive.md`, `fab/project/constitution.md` Principle V example, `docs/specs/hooks.md`, `docs/specs/stage-models.md`, `docs/memory/runtime/pane-commands.md`, `docs/memory/distribution/kit-architecture.md`); set `fab-operator.md` frontmatter `helpers: [_cli-fab-operator, _cli-fab-pane, _cli-agents, _cli-external]` and rewrite its §2 Context Loading helper sentence; update `_preamble.md` § Skill Helper Declaration allowed values to the ten names; leave all other skills' `helpers:` unchanged; verify with the R8 grep (zero hits). <!-- R8, R9 -->
- [x] T007 [P] Amend `fab/project/constitution.md` Additional Constraints wording to the three-family form, set `Last Amended: 2026-09-11` (version stays 1.8.0), append the dated `<!-- 2026-09-11 (260910-si4k): … -->` comment; apply the same wording to `fab/project/code-quality.md` § Anti-Patterns (project-specific) and `fab/project/code-review.md` § Project-Specific Review Rules. <!-- R10 -->
- [x] T008 [P] Update `docs/specs/operator.md` (v10 row, "ten iterations" sentence) and `docs/specs/skills.md` (§ Skill Helpers allowed values 10 + `fab-operator` mapping; `/fab-operator` section lines restating rk-gating / raw `send-keys` / `/loop`; § Adding a skill checklist item 3). <!-- R11 -->

### Phase 4: Polish

- [x] T009 Run the verification set and record outcomes: fallback grep (R2/R5/R6 pattern, exclusions per R8), pointer grep (R8), heading-uniqueness + line-sum (R7), `(cd src/go/fab-kit && go test ./cmd/fab/ -run 'TestKitContentCitesNoRepoLocalPaths|TestKitPortabilityMatcher')` and `(cd src/go/fab && go test ./cmd/fab/ -run TestNoTopLevelCommandCollidesWithRouterAllowlist)`, and `FAB_KIT_PATH=$PWD/src/kit fab sync` showing both new partials in `.agents/skills/`; fix any residue the greps surface in the class (not just the listed sites). <!-- R12 -->

## Execution Order

- T001 blocks T006 (pointer targets must exist) and T009
- T002 blocks T003 (same file, sequential edits); T004/T005 run alongside them
- T006, T007, T008 are independent after T001–T005
- T009 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-operator.md` §2 carries `### rk Gate` between `### Tmux Gate` and `### Role Mark` with the exact probe, the exact STOP text, and the fail-silent-exception sentence pointing at `_preamble.md` § Run-Kit (rk) Reference
- [x] A-002 R2: every row of the intake § 1 deletion table is applied; the `rk notify` fail-silent sentence survives; binary-side fallback references survive as pointers
- [x] A-003 R3: §2 Init step 5 has exactly one ready-line template with the `schedule_summary`/`deliver`/`muted`/`muted_until` render rules and the missing-entry STOP; no `Clock: none` or `(backoff 60s–30m, …)` literal remains
- [x] A-004 R4: the compact frame line, its Element/Format row, the fenced example, Tick step 7 and the §9 Cadence row all carry the per-tick `schedule_summary` read; the full-frame header is unchanged
- [x] A-005 R5: `_cli-agents.md` rk-absent arms are gone per the intake § 2 table; the judgment-round raw-tmux mechanics remain; the description no longer mentions raw-tmux fallbacks
- [x] A-006 R6: `_cli-external.md` has no `## /loop`, no Contents entry for it, no `/loop` in its description; § Reference Model's `<tool> skill` delegation is intact
- [x] A-007 R7: `_cli-fab-pane.md` and `_cli-fab-operator.md` exist with the partial frontmatter shape, header blurb and Contents; `_cli-fab.md` keeps `## Calling Convention` byte-identical and lists the two family files
- [x] A-008 R8: the R8 grep over `src`, `docs`, `fab/project` (with the stated exclusions) returns zero `_cli-fab(.md) § fab (pane|dispatch|operator|agent)` hits — re-verified in re-review (cycle 2) with a wrap-tolerant sweep (2-line window around every `_cli-fab` mention): the cycle-1 stragglers (`_cli-agents.md:39`, `hooks.md:9-10`, `harness-adapters.md:136-137`, `stage-models.md:754-755`, `architecture.md:503`, `configuration.md:234`, `glossary.md:104`) are all re-pointed; every remaining `§ fab (pane|dispatch|operator|agent)` ref either names `_cli-fab-pane.md`/`_cli-fab-operator.md` or is a memory-internal section pointer
- [x] A-009 R9: `fab-operator.md` declares `[_cli-fab-operator, _cli-fab-pane, _cli-agents, _cli-external]`; `_preamble.md` lists ten allowed values; no other skill's `helpers:` changed
- [x] A-010 R10: constitution, code-quality.md and code-review.md carry the three-family wording; constitution `**Version**` is 1.8.0, `Last Amended` 2026-09-11, dated comment present
- [x] A-011 R11: `docs/specs/operator.md` has the v10 row; `docs/specs/skills.md` has ten allowed values, the updated mapping, and no operator-fallback restatement
- [x] A-012 R12: the apply result summary states the outcome of each verification check (a)–(e)

### Behavioral Correctness

- [x] A-013 R2: `fab-operator.md` §6 Dependency Resolution, Dependency Declaration, Working a Change and Autopilot (incl. Queue Completion Summary, Ordered Merge, Auto-Merge Choreography) are byte-identical to before this change except for R8 pointer re-targets
- [x] A-014 R7: moved sections are byte-identical to their `_cli-fab.md` originals apart from the enumerated cross-file re-points (diff the moved blocks against `git show HEAD:src/kit/skills/_cli-fab.md`)
- [x] A-015 R3: the ready-line template copies JSON values verbatim and never states a literal cadence

### Removal Verification

- [x] A-016 R2: the R2 grep pattern over `src/kit/skills/fab-operator.md` returns only binary-side pointers or the judgment-round carve-out
- [x] A-017 R6: repo-wide `/loop` grep over `src/kit`, `docs/specs`, `fab/project` returns only `_cli-agents.md` § Skill Prompts' example mention and historical text
- [x] A-018 R7: no `## fab pane|dispatch|operator|agent` heading remains in `_cli-fab.md`

### Scenario Coverage

- [x] A-019 R1: the gate text handles both failure halves (rk absent; rk present but `rk cron list --json` fails) with the single install-hint error
- [x] A-020 R3: the three ready-line renders (unmuted; muted; muted with lease) and the missing-entry STOP are each stated in the skill

### Edge Cases & Error Handling

- [x] A-021 R2: the pre-delivery judgment-round raw `tmux send-keys` carve-out (owned by `_preamble.md` § The pane readiness gate) is not deleted or reworded by the sweep
- [x] A-022 R8: `docs/specs/findings/*`, `fab/changes/*`, `docs/wiki/*`, `fab/plans/*`, `log.md`, `log.seed.md` are not edited by the sweep

### Code Quality

- [x] A-023 Pattern consistency: the new partials follow `_cli-fab.md`'s frontmatter, header-blurb and Contents conventions; Contents present on every file over 100 lines
- [x] A-024 No unnecessary duplication: no moved section text exists in two files; no rule is both stated and pointed at (owner-or-pointer, `fab/project/code-quality.md`)
- [x] A-025 Canonical source only: no edit under `.agents/skills/` or `.claude/skills/`; all skill edits are under `src/kit/skills/`
- [x] A-026 Deployed content cites no fab-kit-only path: `_cli-fab-pane.md`, `_cli-fab-operator.md` and every edited `src/kit/**` file pass `TestKitContentCitesNoRepoLocalPaths` (Constitution V)
- [x] A-027 Sibling sweep: the whole class was swept — constitution + code-quality + code-review wording; every pointer site in the R8 list; specs restating operator fallbacks; the deployed-partials enumeration at `docs/specs/architecture.md:503`; the wrap-tolerant re-sweep in re-review (cycle 2) found no further stragglers beyond the ones rework cycle 1 fixed (`configuration.md:234`, `glossary.md:104`, kit-architecture.md enumerations)
- [x] A-028 Markdown-only artifacts, CommonMark; no `.go` file changed (the CLI⇒tests rule is not triggered and no test is added)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- The narrative memory rewrite (operator.md Design Decisions incl. the superseding DD for the `/loop` fallback and role-mark version-skew decisions, `_shared/context-loading.md` allowed values and mapping row, `distribution/kit-architecture.md` helper tables, `runtime/agent-primitives.md` arms) is hydrate's per intake § Affected Memory; apply performs only the mechanical pointer re-targets in memory files (R8).

## Deletion Candidates

- `docs/memory/runtime/operator.md` present-truth sections (Overview l.11, 17; Requirements/Context l.31, 51, 94, 130, 197, 208, 260, 275–279; `_cli-external` contents mention l.327; frontmatter `description`) — still restating the deleted rk-absent / raw-tmux / `/loop` / `Notify channel` / version-skew arms the skill no longer carries; removal already assigned to hydrate (intake §5). Re-review note: sweep the whole file at hydrate — l.11, 17 and 327 were not in the original enumeration
- `docs/memory/runtime/operator.md` DD "Claude-Only /loop Fallback Instead of Hard-Requiring rk" (l.587) and the role-mark version-skew DD (l.512) — superseded by the rk-mandatory posture; supersede-via-new-DD at hydrate, do not silently delete
- `docs/memory/runtime/agent-primitives.md` (l.34, 79–83, 95) — requirement sections restating `_cli-agents.md`'s deleted rk-absent pre-send/peek/await arms; hydrate-scope per R11
- `docs/memory/_shared/context-loading.md` (l.37 "Allowed values (eight)", l.51 `fab-operator` mapping row) — stale against the ten-value list and new `helpers:`; hydrate-scope per R11
- `docs/memory/distribution/kit-architecture.md` (l.26–28 skills tree "/loop fallback-scoped", l.344 "`_cli-fab.md` … Used only by `fab-operator` currently", l.355 rk-absent `capture-pane` fallback restatement, l.360 mapping, l.363 "eight allowed values") — stale against the family split and the deleted rk-absent arms; hydrate-scope per R11
- `docs/memory/runtime/pane-commands.md` l.11 (Overview's "fail-open to raw `tmux send-keys` plus manual probing" restating the deleted `_cli-agents.md` rk-absent pre-send arm) — found in the re-review sweep; not in intake's original per-file scope (pane-commands was pointer-only), assign to hydrate
- `docs/memory/runtime/index.md` / `docs/memory/index.md` rows carrying "/loop fallback" descriptions — regenerated by `fab docs-index` at hydrate, never hand-edited

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Two family files; `## fab agent` rides in `_cli-fab-operator.md` (intake #14 resolved) | Only `fab-operator` loads a family file and it uses both sections; pipeline consumers reach `fab agent` by pointer; a third file would serve no loader today | S:70 R:85 A:75 D:70 |
| 2 | Confident | Pipeline skills' `helpers:` unchanged; their `_preamble.md` pointers are re-targeted (intake #16 resolved as pointer-only) | Refactor with no behaviour change; unconditional declaration adds ≈335 lines to every native-mode run for a branch never entered; stage-conditional read is a new mechanism — defer until a run shows need | S:70 R:80 A:70 D:65 |
| 3 | Confident | Ready line prints the bare `{deliver}` value (`· immediate`), not `· deliver immediate` (intake #15 resolved) | Matches the template as written and the `schedule_summary` field's own bare rendering; shorter line | S:65 R:95 A:70 D:60 |
| 4 | Confident | Missing `role:operator` row after the gate → STOP with the seed-it error (intake #11 carried) | User deleted the `Clock: none` form; the operator cannot tick without the entry; `rk operator` seeds idempotently | S:60 R:90 A:65 D:55 |
| 5 | Confident | Cadence re-read once per tick in Tick step 7; compact line only (intake #12 carried) | §1 Re-derive state; `rk cron edit` must show on the next frame; full-frame header unchanged per the user | S:60 R:90 A:65 D:55 |
| 6 | Confident | Apply performs mechanical pointer re-targets in `docs/memory/*` (R8) but leaves narrative memory rewrites to hydrate | Matches this repo's apply/hydrate split; the R8 grep must reach zero before review, which requires the memory pointer sites too | S:65 R:85 A:75 D:65 |
| 7 | Confident | Verification uses `FAB_KIT_PATH=$PWD/src/kit fab sync` to prove deployment of the new partials | `fab kit-path` resolves the released kit (`~/.fab-kit/versions/2.24.5/kit`), which lacks the new files; the env override is the existing dev seam (`setup_test.go`); deploy targets are gitignored generated copies | S:60 R:80 A:70 D:60 |
| 8 | Certain | No `.go` change; existing portability and lifecycle-collision tests are run as verification only | Intake-verified: `fab sync` enumerates every `*.md`, `fab help` skips `_*`, the router line stays in core | S:85 R:90 A:95 D:95 |

8 assumptions (1 certain, 7 confident, 0 tentative).
