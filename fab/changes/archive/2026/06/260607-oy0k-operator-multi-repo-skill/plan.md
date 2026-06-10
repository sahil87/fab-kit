# Plan: Operator Multi-Repo Skill + Specs

**Change**: 260607-oy0k-operator-multi-repo-skill
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

> This is a **skill + specs** change (no Go code). It re-frames the `/fab-operator` skill
> and its spec around a multi-repo / multi-session model on one tmux server, consuming the
> Go primitives shipped by change 1 (`260607-h3jk`): server-keyed XDG operator state path,
> a `repo` field in `fab pane map --json`, `--all-sessions` on `fab pane map`, and a
> `fab spawn-command --repo` helper. The canonical skill source is
> `src/kit/skills/fab-operator.md` — `.claude/skills/` copies are never edited (constitution).

### Operator: Addressing & Schema

#### R1: `(session, repo, pane)` addressing tuple
The operator SHALL address every monitored agent, branch-map entry, and watch as a
`(session, repo, pane)` tuple. Pane ID (server-global, stable) SHALL remain the primary key;
`repo` (absolute main-worktree root) and `session` (tmux session name) are added dimensions,
not replacements. §1 Principles SHALL carry a "Multi-repo aware" principle stating the operator
spans repos and sessions on one tmux server.

- **GIVEN** a tmux server with agents in repos `~/code/foo` (session `work`) and `~/code/bar` (session `side`)
- **WHEN** the operator enrolls each agent
- **THEN** each monitored entry records its `pane`, `repo`, and `session`
- **AND** the pane ID alone still uniquely identifies the entry across the whole server

#### R2: Repo/session-qualified `.fab-operator.yaml` schema
The `.fab-operator.yaml` schema (§4) SHALL extend each `monitored` entry with `repo` (absolute
main-worktree root) and `session` (tmux session name); the `branch_map` value SHALL become
`{branch, repo}` (was a bare branch string); each `watches` entry SHALL gain `target_repo`.

- **GIVEN** the §4 schema block
- **WHEN** a reader inspects a `monitored` entry, a `branch_map` value, or a `watches` entry
- **THEN** `monitored.<id>` shows `repo:` and `session:` keys, `branch_map.<id>` is `{ branch, repo }`, and `watches.<name>` shows a `target_repo:` key

### Operator: Tick & Status Frame

#### R3: Tick snapshots all sessions via `fab pane map --all-sessions --json`
The tick's snapshot step (§4 Tick Behavior step 1) SHALL replace `fab pane map` with
`fab pane map --all-sessions --json`, then group rows by `repo` (using change 1's `repo` JSON
field), then by `session`. Health glyphs, the autopilot `▶` marker, and the `⚠` stuck marker
are unchanged.

- **GIVEN** agents running across two repos in two sessions on one server
- **WHEN** the tick runs its snapshot step
- **THEN** it invokes `fab pane map --all-sessions --json` and parses the `repo` field on each row
- **AND** rows are grouped first by `repo`, then by `session`

#### R4: Status frame uses repo-section headers
The status frame (§4) SHALL render a per-repo header line (with the session noted) followed by
indented entry rows beneath it — NOT a per-row `repo`/`session` column. Each watch row notes its
`target_repo`.

- **GIVEN** the reshaped frame
- **WHEN** the operator prints a tick
- **THEN** each repo appears as a header (e.g., `  ~/code/foo (session: work)`) with its changes indented below
- **AND** a watch row shows its target repo (e.g., `[watch] linear-bugs → ~/code/foo`)

### Operator: Repo-Targeted Spawning

#### R5: Spawn flows establish target repo, create worktree there, read that repo's spawn command
Every spawn flow (§6) SHALL first establish which repo the work targets, then (1) run
`wt create --non-interactive` **in that repo's directory** so the worktree lands under
`$(dirname target-repo)/<repo>.worktrees/`, (2) read **that repo's** `agent.spawn_command` via
`fab spawn-command --repo <target-repo>` (NOT the operator's own `config.yaml`), and (3) enroll
with `repo` and `session` recorded. Window markers (`»` / `›`) are unchanged — they key on
server-global pane IDs.

- **GIVEN** a request to spawn work targeting `~/code/bar`
- **WHEN** the operator runs the spawn sequence
- **THEN** `wt create` runs in `~/code/bar`, the spawn command comes from `fab spawn-command --repo ~/code/bar`, and the enrolled entry records `repo: ~/code/bar` and its `session`

### Operator: Two-Tier Dependencies

#### R6: Same-repo deps cherry-pick; cross-repo deps are ordering-only
Dependency resolution (§6) SHALL split `depends_on` by repo: a dependency in the **same repo** as
the change cherry-picks as today (`git cherry-pick --no-commit origin/main..<dep-branch>`); a
dependency in a **different repo** is an **ordering-only barrier** — the operator waits until the
dependency reaches its `stop_stage`, then spawns, with **NO code merge**. Ancestor-pruning
(`git merge-base --is-ancestor`) SHALL apply only within the same-repo subset of the dependency set.

- **GIVEN** a change in `~/code/foo` with `depends_on` containing one same-repo dep and one cross-repo dep (in `~/code/bar`)
- **WHEN** the operator resolves dependencies before spawning
- **THEN** the same-repo dep is cherry-picked into the worktree
- **AND** the cross-repo dep is treated as an ordering barrier (wait for its `stop_stage`, then spawn, no cherry-pick)
- **AND** ancestor-pruning considers only the same-repo dep branches

#### R7: Cross-repo dependency caveat (no code) is documented
The skill SHALL state explicitly that an ordering-only cross-repo dependency gives the dependent
agent **no code** from its dependency — it is a pure sequencing constraint, correct only for
logical dependencies (e.g., "don't start the frontend change until the API change merges"), never
for code-level dependencies.

- **GIVEN** §6 Dependency Resolution prose
- **WHEN** a reader looks for the cross-repo dependency semantics
- **THEN** an explicit caveat states that cross-repo `depends_on` provides no code, only ordering

### Operator: Repo-Scoped Autopilot

#### R8: Autopilot queue spans repos with mixed dependency semantics
The autopilot queue (§6) MAY span repos. Implicit `--base` chaining SHALL cherry-pick within a
repo and degrade to ordering-only across repo boundaries. Ordered merge (`gh pr merge` + CI wait)
SHALL track **per-repo PR sequences** so merge order respects each repo's own dependency chain;
the queue-completion summary SHALL annotate each PR with its repo.

- **GIVEN** an autopilot queue containing changes in two repos
- **WHEN** the operator chains and later merges them
- **THEN** within-repo successors get cherry-picked `depends_on`, cross-repo successors get ordering-only barriers
- **AND** the completion summary annotates each PR with its repo and the merge follows per-repo sequences

#### R9: CI-failure scope is halt-dependents-only
During ordered merge (§6), a CI failure SHALL halt the failing repo's merge sub-sequence AND any
repo whose queued items carry a cross-repo `depends_on` into the failed chain (transitively);
truly independent repos' sub-sequences continue merging. The completion summary SHALL report which
sub-sequences halted vs. completed and escalate the failure to the user.

- **GIVEN** an ordered merge where repo A's PR fails CI, repo B has a cross-repo dep into A's chain, and repo C is independent
- **WHEN** CI fails on A's PR
- **THEN** A's sub-sequence halts, B's sub-sequence halts (transitive cross-repo dependent), and C's sub-sequence continues
- **AND** the summary reports halted vs. completed sub-sequences and escalates

### Operator: Watches & Conversational Management

#### R10: Watches gain `target_repo`; conversational management gains repo targeting
The watch schema (§7) SHALL include `target_repo` — the repo a watch's spawned changes land in.
Conversational management SHALL accept repo targeting (e.g., "watch Linear project DEV, spawn into
~/code/foo, stop at intake" sets `target_repo`), and "What are you watching?" SHALL list each
watch's `target_repo`. A watch's spawn action SHALL use `target_repo` as the spawn target repo
(per R5).

- **GIVEN** a watch created with "spawn into ~/code/foo"
- **WHEN** the operator records the watch and later spawns from it
- **THEN** the watch entry carries `target_repo: ~/code/foo` and spawns land in that repo
- **AND** "What are you watching?" lists the watch's `target_repo`

### Operator: Configuration & Key Properties

#### R11: One-operator-per-server documented; state file is server-keyed XDG path
§8 Configuration SHALL document the one-operator-per-server model (isolation unit = tmux server;
a second operator means a second tmux server via `tmux -L <label>`; "multiple sessions, same
server" share one operator and one state file). §9 Key Properties SHALL update the
`.fab-operator.yaml` row to note the server-keyed XDG location
`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (falling back to `~/.local/state` when
`XDG_STATE_HOME` is unset). No migration of old repo-rooted `.fab-operator.yaml` files.

- **GIVEN** §8 and §9
- **WHEN** a reader inspects the isolation model and state-file location
- **THEN** §8 states one operator per tmux server (no `--name`) and §9 names the server-keyed XDG path

### Specs & Helper (Constitution-mandated)

#### R12: `SPEC-fab-operator.md` updated for the multi-repo model
Per the constitution ("Changes to skill files MUST update the corresponding
`docs/specs/skills/SPEC-*.md`"), `docs/specs/skills/SPEC-fab-operator.md` SHALL be updated for the
multi-repo model: the `(session, repo, pane)` addressing tuple, the repo/session-qualified schema,
repo-targeted spawning, two-tier dependency resolution, repo-scoped autopilot with
halt-dependents-only CI semantics, and watch `target_repo`. Updates SHALL match the spec's existing
section structure.

- **GIVEN** the skill changes in R1–R11
- **WHEN** the spec is read
- **THEN** it reflects the multi-repo addressing tuple, schema, spawning, dependency, autopilot, and watch model

#### R13: `_cli-external.md` notes `--all-sessions` and repo-targeted `wt create`
`src/kit/skills/_cli-external.md` SHALL note `fab pane map --all-sessions` usage in the tick and the
repo-targeted `wt create` flow (worktree created in the target repo's directory).

- **GIVEN** `_cli-external.md`
- **WHEN** a reader looks for the tick's pane-map invocation and the spawn worktree flow
- **THEN** it documents `fab pane map --all-sessions` and that `wt create` runs in the target repo's directory

### Non-Goals

- No Go code, templates, or migrations (that was change 1, `260607-h3jk`).
- No `--name` operator dimension — isolation unit is the tmux server.
- No migration of old repo-rooted `.fab-operator.yaml` — abandoned in place.
- No mechanism to make cross-repo dependency code available — cross-repo deps are ordering-only by design.

### Design Decisions

1. **Isolation unit = tmux server (one operator per server)**: matches the existing
   `tmux select-window -t operator` server-wide singleton — *Why*: a fixed global path would force
   a machine-wide singleton (rejected); keying by tmux socket lets a second `tmux -L <label>` server
   host a second operator — *Rejected*: per-repo operators (loses the single pane of glass), a
   `--name` dimension (redundant with the server boundary).
2. **State file keyed by tmux socket path under XDG**: `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`
   (slug from `#{socket_path}`); path derivation is change 1's `StatePath()` — *Why*: server-scoped
   single owner across repos — *Rejected*: repo-rooted `.fab-operator.yaml` (single-repo only), fixed
   global path (machine-wide singleton).
3. **Cross-repo deps = ordering-only**: same-repo `depends_on` cherry-picks; cross-repo waits for
   `stop_stage` with no code merge — *Why*: cross-repo branches share no `origin/main` base to
   cherry-pick from; logical sequencing is the only sound cross-repo semantic — *Rejected*: forbid
   cross-repo deps (too restrictive), full cross-repo code merge (no shared base, unsound).
4. **CI-failure scope = halt-dependents-only**: halt the failing repo's sub-sequence + any repo with
   a transitive cross-repo `depends_on` into the failed chain; independents continue — *Why*:
   maximizes independent-repo throughput while respecting cross-repo ordering barriers — *Rejected*:
   halt-all (conservative, throttles independent repos), halt-only-failing-repo (ignores cross-repo
   ordering barriers).
5. **Status frame = repo-section headers**: per-repo header line with indented entries — *Why*:
   scannability over per-row columns — *Rejected*: per-row `repo`/`session` columns (wider, harder
   to scan).

## Tasks

### Phase 1: Skill core re-framing (`src/kit/skills/fab-operator.md`)

- [x] T001 §1 Principles — add a "Multi-repo aware" principle (operator spans repos + sessions on one tmux server; pane ID is the server-global primary key; repo + session are added dimensions; every monitored entry, branch_map entry, and watch is repo-qualified) in `src/kit/skills/fab-operator.md` <!-- R1 -->
- [x] T002 §4 `.fab-operator.yaml` schema — add `repo` + `session` to each `monitored` entry, change `branch_map` values to `{ branch, repo }`, add `target_repo` to each `watches` entry; update the "Monitored Set" and "Branch Map" prose to mention repo/session in `src/kit/skills/fab-operator.md` <!-- R2 -->

### Phase 2: Tick, spawning, dependencies, autopilot, watches (`src/kit/skills/fab-operator.md`)

- [x] T003 §4 Tick Behavior step 1 — replace `fab pane map` with `fab pane map --all-sessions --json`; group rows by `repo` then `session`; reshape the status-frame example into repo-section headers with indented entries (per R4); add a `target_repo` annotation to watch rows; preserve health glyphs, `▶`, `⚠` semantics; adjust the column-layout table to reflect repo-section grouping in `src/kit/skills/fab-operator.md` <!-- R3 R4 -->
- [x] T004 §6 Spawning an Agent / Working a Change — establish target repo first; run `wt create` in the target repo's directory; read the target repo's `agent.spawn_command` via `fab spawn-command --repo <target-repo>` (replace the "read from config.yaml" prose); enroll with `repo` + `session`; note `»`/`›` markers key on server-global pane IDs in `src/kit/skills/fab-operator.md` <!-- R5 -->
- [x] T005 §6 Dependency Resolution — split resolution by repo: same-repo cherry-pick as today, cross-repo ordering-only barrier (wait for `stop_stage`, no merge); restrict ancestor-pruning to the same-repo subset; add the REQUIRED cross-repo "no code, ordering-only" caveat in `src/kit/skills/fab-operator.md` <!-- R6 R7 -->
- [x] T006 §6 Autopilot + Ordered Merge — queue may span repos with mixed dependency semantics (within-repo cherry-pick chaining degrades to cross-repo ordering-only); ordered merge tracks per-repo PR sequences; completion summary annotates each PR with its repo; replace the "CI failure halts the merge sequence" prose with halt-dependents-only semantics (transitive over the cross-repo `depends_on` graph) and the halted-vs-completed escalation summary in `src/kit/skills/fab-operator.md` <!-- R8 R9 -->
- [x] T007 §7 Watches — add `target_repo` to the watch schema table; update Tick Behavior step 4 (Act) to spawn into `target_repo` (per R5); add repo targeting to Conversational Management ("spawn into ~/code/foo" sets `target_repo`; "What are you watching?" lists `target_repo`) in `src/kit/skills/fab-operator.md` <!-- R10 -->

### Phase 3: Configuration, key properties, init (`src/kit/skills/fab-operator.md`)

- [x] T008 §8 Configuration — document the one-operator-per-server model (isolation unit = tmux server; second operator = second `tmux -L <label>` server; multiple sessions on one server share one operator + one state file) in `src/kit/skills/fab-operator.md` <!-- R11 -->
- [x] T009 §9 Key Properties + §2 Init — update the `.fab-operator.yaml` row to the server-keyed XDG location `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state`); update §2 Init and §4 prose that say "repo root" to reference the server-keyed state file; note no migration of old repo-rooted files in `src/kit/skills/fab-operator.md` <!-- R11 -->

### Phase 4: Spec + helper (Constitution-mandated)

- [x] T010 `docs/specs/skills/SPEC-fab-operator.md` — update for the multi-repo model: `(session, repo, pane)` addressing tuple, repo/session-qualified schema, `--all-sessions` tick + repo-section frame, repo-targeted spawning, two-tier dependency resolution (with cross-repo no-code caveat), repo-scoped autopilot with halt-dependents-only CI semantics, watch `target_repo`; match the spec's existing section structure <!-- R12 -->
- [x] T011 [P] `src/kit/skills/_cli-external.md` — note `fab pane map --all-sessions` usage in the tick (replace/augment the raw `tmux capture-pane` and `new-window` notes where they reference single-session/single-repo assumptions) and document that `wt create` runs in the target repo's directory for repo-targeted spawning <!-- R13 -->

### Phase 5: Validation

- [x] T012 Verify markdown is well-formed and internally consistent: confirm `fab` still parses (`fab preflight oy0k`), the skill's internal section cross-references (§1/§4/§6/§7/§8/§9) remain consistent, and the schema example in §4 matches the prose; spot-check the spec and `_cli-external.md` for broken intra-doc references <!-- R1 R2 R3 R4 R5 R6 R7 R8 R9 R10 R11 R12 R13 -->

## Execution Order

- Phase 1 (T001–T002) before Phase 2 (the schema is referenced by tick/spawn/dep/autopilot/watch prose).
- T003–T007 are sequential within Phase 2 (all edit `fab-operator.md`; sequential edits to the same file avoid conflicting context).
- T008–T009 (Phase 3) after Phase 2.
- T010 (spec) after the skill is final (Phases 1–3) so the spec matches the shipped skill.
- T011 [P] is independent of T010 (different file) but should follow Phase 2 so its tick/spawn notes match.
- T012 last (validation over all edits).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/fab-operator.md` §1 carries a "Multi-repo aware" principle stating the operator spans repos + sessions on one server, pane ID is the server-global primary key, and monitored/branch_map/watch entries are repo-qualified
- [x] A-002 R2: §4 schema shows `repo` + `session` on each `monitored` entry, `branch_map` values as `{ branch, repo }`, and `target_repo` on each `watches` entry; surrounding "Monitored Set"/"Branch Map" prose matches
- [x] A-003 R3: §4 Tick Behavior step 1 invokes `fab pane map --all-sessions --json` and groups rows by `repo` then `session`
- [x] A-004 R4: §4 status-frame example uses per-repo header lines with indented entries (not per-row columns); watch rows note `target_repo`; the column-layout table reflects repo-section grouping
- [x] A-005 R5: §6 spawn flows establish target repo, run `wt create` in the target repo's dir, read `agent.spawn_command` via `fab spawn-command --repo <target-repo>`, and enroll with `repo` + `session`
- [x] A-006 R6: §6 Dependency Resolution splits same-repo (cherry-pick) vs cross-repo (ordering-only, no merge); ancestor-pruning is scoped to the same-repo subset
- [x] A-007 R7: §6 states the explicit caveat that cross-repo `depends_on` gives the dependent agent no code (logical ordering only)
- [x] A-008 R8: §6 Autopilot/Ordered Merge documents queues spanning repos with mixed dependency semantics, per-repo PR sequences, and per-PR repo annotation in the summary
- [x] A-009 R9: §6 Ordered Merge documents halt-dependents-only CI semantics (transitive over cross-repo `depends_on`), independents continue, and a halted-vs-completed escalation summary
- [x] A-010 R10: §7 watch schema includes `target_repo`; Tick Act step spawns into `target_repo`; Conversational Management supports repo targeting and lists `target_repo`
- [x] A-011 R11: §8 documents one-operator-per-server; §9 names the server-keyed XDG state path; no-migration note present
- [x] A-012 R12: `docs/specs/skills/SPEC-fab-operator.md` reflects the multi-repo model (addressing tuple, schema, spawning, dependencies, autopilot CI semantics, watch target_repo)
- [x] A-013 R13: `src/kit/skills/_cli-external.md` notes `fab pane map --all-sessions` and repo-targeted `wt create`

### Behavioral Correctness

- [x] A-014 R6: The two-tier dependency split is unambiguous — a reader can tell, for any dep, whether it cherry-picks (same repo) or waits (cross repo)
- [x] A-015 R9: The halt-dependents-only rule is stated as transitive over the cross-repo dependency graph, so a reader can determine which sub-sequences halt vs. continue for any failure
- [x] A-016 R5: The spawn prose no longer instructs reading the operator's own `config.yaml` for the spawn command — it reads the target repo's via `fab spawn-command --repo`

### Scenario Coverage

- [x] A-017 R3: The §4 reshaped status-frame example concretely shows ≥2 repos with sessions and indented change/watch rows (matching the intake's example)
- [x] A-018 R10: A conversational example shows setting `target_repo` via natural language ("spawn into ~/code/foo")

### Edge Cases & Error Handling

- [x] A-019 R3: The skill notes the `repo` JSON field may be em-dash (unresolved) for panes not in a git repo, and the frame handles such rows gracefully (consistent with change 1's contract)

### Code Quality

- [x] A-020 Pattern consistency: Edits follow the skill file's existing prose conventions (section numbering, blockquote callouts, code-block style, glyph notation) — no structural drift
- [x] A-021 No unnecessary duplication: The cross-repo caveat and CI-failure semantics are stated once in the canonical location and cross-referenced, not copy-pasted across sections
- [x] A-022 Readability/maintainability (code-quality.md): edits favor clarity; no "god" sections introduced — new content is scoped to the relevant existing section
- [x] A-023 No magic strings (code-quality.md anti-pattern): the XDG path, `<server-slug>` token, and glyphs are written consistently with their existing definitions
- [x] A-024 Documentation accuracy (config extra_categories): documented command signatures (`fab pane map --all-sessions --json`, `fab spawn-command --repo`, `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`) match change 1's actual Go contract
- [x] A-025 Cross-references (config extra_categories): intra-doc section references (§1/§4/§6/§7/§8/§9) and the skill↔spec correspondence remain valid after edits

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- The canonical skill source is `src/kit/skills/fab-operator.md`; never edit `.claude/skills/` copies (constitution + context.md).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change 1's Go primitives are present in this worktree (verified `fab pane map --all-sessions --json` with a `repo` field, `fab spawn-command --repo`, and `StatePath()` deriving `<stateDir>/fab/operator/<server-slug>.yaml`) — document the real contract verbatim | Verified via `--help` and source grep in `src/go/fab/cmd/fab/{panemap,spawn_command,operator}.go`; not assumed-absent | S:98 R:85 A:95 D:95 |
| 2 | Certain | The `repo` JSON field is the absolute main-worktree root (em-dash when the pane is not in a git repo); group the tick by this field | Read directly from `panemap.go` (`Repo *string json:"repo"`, `mainRootForPane`, em-dash fallback) | S:95 R:80 A:95 D:92 |
| 3 | Certain | State path documented as `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` with `~/.local/state` fallback when `XDG_STATE_HOME` is unset/relative | Matches `stateDir()`/`StatePath()` in `operator.go` and the existing memory note in `runtime/operator.md` | S:95 R:80 A:92 D:90 |
| 4 | Certain | Update `SPEC-fab-operator.md` for the multi-repo model while preserving its existing section structure; do NOT rewrite unrelated stale content (e.g., the "fab-operator4" title, the stale "all-auto-answer" decision) beyond what the multi-repo re-framing touches | Intake says "update ... match its structure"; constitution mandates the spec update for skill changes but not a full audit; minimizes blast radius | S:85 R:70 A:80 D:80 |
| 5 | Certain | Within-repo autopilot `--base` chaining keeps cherry-pick semantics; cross-repo successors in a queue get ordering-only barriers (degrade) | Direct extension of locked design decision 4 (cross-repo = ordering-only) applied to the autopilot queue per B5 | S:95 R:65 A:88 D:90 |
| 6 | Certain | "Dependent" for CI-failure halt is transitive over the cross-repo `depends_on` graph | Stated verbatim in intake B5 implementation note | S:98 R:70 A:92 D:95 |
| 7 | Certain | Watch spawns reuse the §6 repo-targeted spawn sequence with `target_repo` as the target | B6 + R5 composition; the only consistent reading | S:92 R:75 A:88 D:88 |

7 assumptions (7 certain, 0 confident, 0 tentative, 0 unresolved).
