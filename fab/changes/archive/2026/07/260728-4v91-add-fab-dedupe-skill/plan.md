# Plan: Add /fab-dedupe skill

**Change**: 260728-4v91-add-fab-dedupe-skill
**Intake**: `intake.md`

## Requirements

### Skill: `/fab-dedupe` identity and invocation contract

#### R1: Canonical skill file, name, and helper declaration
The skill SHALL exist at `src/kit/skills/fab-dedupe.md` (the canonical kit source — Constitution V; `.claude/skills/` MUST NOT be edited). Its frontmatter `name` MUST be `fab-dedupe`, matching the filename, its `description` MUST name the actual behavior (sweep → cluster → draft one intake per accepted cluster, read-only until accept), and its `helpers:` list MUST be `[_generation, _srad, _intake]` — the same set `/fab-draft` declares, since the skill is a thin call-site over the shared Create-Intake Procedure. The body MUST open with the standard preamble-read blockquote.

- **GIVEN** the kit source tree
- **WHEN** an agent reads `src/kit/skills/fab-dedupe.md`
- **THEN** the frontmatter reads `name: fab-dedupe` with `helpers: [_generation, _srad, _intake]`
- **AND** no draft-era occurrence of the working name `fab-consolidate` remains anywhere in the file

#### R2: Pre-flight without `fab preflight`
The skill SHALL verify `fab/project/config.yaml` and `fab/project/constitution.md` exist and STOP with the standard uninitialized message when either is missing. It MUST NOT run `fab preflight` — the skill operates with no active change and must not resolve or disturb one.

- **GIVEN** a repo with no active change (no `.fab-status.yaml`)
- **WHEN** `/fab-dedupe` is invoked
- **THEN** the skill proceeds on config/constitution presence alone
- **AND** the pre-flight section explicitly forbids `fab preflight`

#### R3: Read-only until cluster acceptance
The skill SHALL NOT refactor code, write memory files, activate a change, create a git branch, or modify `.fab-status.yaml`. Everything before Step 5 SHALL be read-only. The only writes are the intakes created by the shared Create-Intake Procedure for accepted clusters.

- **GIVEN** a sweep where the user accepts no clusters
- **WHEN** the skill finishes
- **THEN** nothing has been written and no `Next:` line is emitted

### Skill: sweep behavior (Steps 1–5)

#### R4: Scope resolution with member-vs-finding distinction
Step 1 SHALL resolve `[scope]` (natural language, path, or glob) to a concrete path set: an existing path/glob is used directly; natural language maps against `source_paths`/`test_paths` and is confirmed with the user before sweeping; a bare invocation defaults to `source_paths` with a warning about list length. The resolved path set and file count SHALL be echoed before sweeping. Scope SHALL filter *where clusters may be found*, not *where their members may live*: a cluster seeded inside the scope MAY include members outside it, and those members MUST be reported.

- **GIVEN** `/fab-dedupe "test setup helpers in src/go"`
- **WHEN** Step 1 runs
- **THEN** the natural-language scope is mapped to paths and confirmed with the user, and the resolved set + file count are echoed
- **AND** a cluster seeded in `src/go` that has a member in `scripts/` reports that member, flagged as out-of-scope

#### R5: Detector layer — config-driven, probe-then-fail-silent, non-zero-exit tolerant
Step 2 SHALL read `consolidate.detectors` from project config, defaulting to jscpd alone when absent. For each detector the skill MUST probe with `command -v <bin> >/dev/null 2>&1` before invoking, skip silently on absence (mirroring the `rk` discipline in `_preamble.md` § Run-Kit Reference), substitute `{paths}` (space-joined resolved scope paths) and `{out}` (a scratch dir created for the run), run it, and record per detector: ran / skipped-missing / ran-with-exit-code-N. A detector that runs and exits non-zero SHALL NOT be a STOP — the skill parses available output, notes the exit code, and continues; the skill MUST state in those words that this is an explicit per-skill exception to `_preamble.md` § Common fab Commands' failure rule, which governs `fab` commands rather than third-party tools. Detector output is a seed set only, never the cluster list.

- **GIVEN** `consolidate.detectors` is absent and jscpd is not installed
- **WHEN** Step 2 runs
- **THEN** the detector is skipped silently (no error, no warning) and the sweep continues unaided, with the skip recorded in the report
- **AND GIVEN** jscpd is installed and exits 1 because its threshold was exceeded
- **WHEN** Step 2 runs
- **THEN** the skill parses the output, notes exit code 1, and continues

#### R6: Layered cluster decomposition (shared core + variation layers)
Step 3 SHALL cluster by behavioral shape — reading function bodies in scope, seeded by but not limited to detector output — and MUST record per cluster: members (`file:line` + current name), shared behavior (one line), divergences, proposed canonical home (following the repo's own layout conventions, read from neighboring directories), call-site count, out-of-scope members, and a **layered decomposition**: the base behavior every member needs, each opt-in layer with the members requiring it, and the unified API expressing that shape (functional options, options struct, base + wrappers, or an explicit "these two should stay separate"). The skill MUST NOT record a single flat "proposed signature" per cluster — real clusters are layered, and one signature per cluster produces a bad API. Two rules MUST be stated: do not cluster on name similarity alone (read the bodies), and a cluster of one is not a cluster (drop it).

- **GIVEN** a cluster of four `setupXFixture` helpers where all four need a tempdir + `fab/changes` but only two need a project config and only one needs an active symlink
- **WHEN** Step 3 records the cluster
- **THEN** it records a base layer (tempdir + `fab/changes`, all four) plus named opt-in layers with their member lists, and a single unified `newFabRoot(t, opts...)`-shaped API
- **AND** it does NOT record one flat signature covering all four

#### R7: Ranking treats base-layer-only members as LOW divergence
Step 4 SHALL rank clusters by consolidation value — high call-site count and low divergence first — and MUST treat "this member needs only the base layer" as **low** divergence even when the member's body looks different. The presented report SHALL be read-only and SHALL show the resolved scope, the detector run/skip/exit-code record, the scanned counts, and per cluster its member count, call-site count, divergence rating, member locations, proposed canonical home + unified API, and the base/layers breakdown.

- **GIVEN** a 12-member cluster whose bodies differ textually but where 8 members need only the base layer
- **WHEN** Step 4 ranks it
- **THEN** it is rated low divergence and ranks above a 3-member cluster with genuinely divergent semantics

#### R8: Cluster acceptance is a plain conversational reply
Step 4 SHALL collect cluster acceptance as a **plain conversational reply** (`all` / `1,3` / `none`) — NOT a structured multi-select / AskUserQuestion picker. Accepting nothing SHALL be a valid, complete outcome.

- **GIVEN** a report listing 7 clusters
- **WHEN** the skill asks which to act on
- **THEN** it asks in prose with the `(all / 1,3 / none)` reply grammar and does not open a structured picker
- **AND** the count of clusters is not capped by the interaction mechanism

#### R9: Step 5 binds into the shared Create-Intake Procedure and stops at `ready`
Step 5 SHALL, per accepted cluster group, read `.claude/skills/_intake/SKILL.md` and execute the **Create-Intake Procedure (Steps 0–9)** with `{questioning-mode} = interactive`, then **STOP after Step 9** — no activation, no git branch (those steps live only in `fab-new.md`'s tail, which this skill never reads). Grouping SHALL be by refactor coherence, not count, and the skill MUST state a preference for **one intake per cluster**. The binding SHALL be specified per procedure step: Step 0 takes natural-language input (`consolidate {shared behavior} into {canonical home}` — no backlog/Linear ID, so no collision check and a fresh change per invocation); Step 2's gap analysis checks the utilities memory file and in-flight changes per cluster and skips those already handled; Step 5 populates the template from cluster data (Origin ← scope + detector commands + exit codes; Why ← member/call-site counts and the cost of leaving it; What Changes ← canonical home, layered API, per-member migration notes; Impact ← every call site plus out-of-scope members; Affected Memory ← the utilities file, `(new)` first time then `(modify)`; Open Questions ← unresolved divergences); Step 8 grades honestly per the stated Certain/Confident/Tentative/Unresolved criteria and lets a mostly-Tentative cluster fail the gate rather than talking the score up.

- **GIVEN** the user accepts clusters 1 and 3
- **WHEN** Step 5 runs
- **THEN** two intakes are created, each at intake `ready`, with no `.fab-status.yaml` symlink and no git branch
- **AND** each intake's `## Affected Memory` names `_shared/utilities`

#### R10: Memory home is hardcoded; no `consolidate.memory_file` key
The utilities memory home SHALL be hardcoded to `docs/memory/_shared/utilities.md` in the skill prose. The skill MUST NOT reference `consolidate.memory_file` as an override — that key was deliberately not shipped. The skill SHALL NOT write this file; it appears in each drafted intake's `## Affected Memory` and hydrate writes it on the normal pipeline path. The skill SHALL document the honest coverage header the file carries and the deliberate rejection of a top-level `utilities` domain.

- **GIVEN** the shipped skill and SPEC files
- **WHEN** they are grepped for `memory_file`
- **THEN** there are zero matches

#### R11: Output and error handling
Per drafted intake the skill SHALL report name and confidence, then emit the Activation Preamble `Next:` line per `_preamble.md` § Activation Preamble. When no clusters are accepted it SHALL end with the report and **no** `Next:` line. The skill SHALL carry an Error Handling table covering: config/constitution missing → STOP; scope resolves to no files → STOP with the resolved set shown; all detectors missing → continue unaided, noted in the report; a detector exits non-zero → continue; unparseable detector output → continue, treat as no seed; no clusters found → report and stop; user accepts nothing → report and stop; `fab change new` fails for one cluster → report, continue with the rest, list failures at the end. It SHALL also carry the standard Key Properties table.

- **GIVEN** `fab change new` fails for the second of three accepted clusters
- **WHEN** Step 5 runs
- **THEN** the failure is reported, the third cluster is still drafted, and the failures are listed at the end

### Config registry: `consolidate.detectors`

#### R12: One new registry row following the `checklist.extra_categories` shape
`src/go/fab/internal/configref/configref.go` SHALL gain exactly ONE new `Field` row, `consolidate.detectors`, following the `checklist.extra_categories` entry shape (`Key` / `Default` / `Description` / `Scope` / `Advertise` / `Segment`) with `Default: nil`, `Scope: ScopeProject`, `Advertise: true`, and the intake's verbatim Description and Segment text. No `consolidate.memory_file` row SHALL be added.

- **GIVEN** the registry
- **WHEN** `configref.Fields()` is called
- **THEN** it returns a `consolidate.detectors` row with nil Default, project scope, and Advertise true
- **AND** no `consolidate.memory_file` row exists

#### R13: `consolidate` is registered in the single-source scope taxonomy
`src/go/fab/internal/configscope/configscope.go`'s `keyScopes` table SHALL gain `"consolidate": ScopeProject`. This is required, not optional: `configref.lintFields` fails loud when a row's Scope disagrees with `configscope.ScopeFor(topLevelKey)`, and an unknown top-level key yields the empty scope, so without this entry `configref.Fields()` returns an error and every consumer (`fab config reference`, `fab config upgrade`, `fab config init`) breaks.

- **GIVEN** the `consolidate.detectors` registry row with `Scope: ScopeProject`
- **WHEN** `configref.Fields()` runs its lint
- **THEN** `configscope.ScopeFor("consolidate")` reports `(project, true)` and the lint passes
- **AND** the loader's `pruneProjectScoped` strips a `consolidate:` block found in the system config file, with a warning

#### R14: Rendered surfaces pick the key up automatically and remain byte-stable
Because both renderings walk the one registry table, `fab config reference` (commented YAML) and `fab config reference --json` SHALL both include `consolidate.detectors` with no second edit, and the `fab config upgrade` managed fence SHALL advertise it. Rendering SHALL remain deterministic and idempotent.

- **GIVEN** the new advertised row
- **WHEN** `fab config reference` renders
- **THEN** the `consolidate.detectors` segment appears exactly once, commented-out (opting in requires uncommenting), and `--json` carries a matching entry
- **AND** re-running `fab config upgrade` over its own output is byte-identical

#### R15: Go changes ship tests
Per `fab/project/code-review.md` ("a `.go` change without accompanying test updates is a must-fix gap") and Constitution VII, the Go changes SHALL ship test updates in the same change. There is no test file in `configref/`, so coverage lands in the existing suites: `src/go/fab/cmd/fab/config_test.go` (registry scope-assignment pin and the reference/JSON surface), `src/go/fab/internal/configscope/configscope_test.go` (the taxonomy pin), and `src/go/fab/internal/configupgrade/` (fence rendering over the shipped registry). Golden fixtures in `configupgrade/golden_test.go` SHALL be verified against the new registry and updated if they churn.

- **GIVEN** the new registry key and scope entry
- **WHEN** `go test ./internal/configscope/... ./internal/configupgrade/... ./cmd/fab/... ./internal/config/...` runs
- **THEN** every package passes, and the pinned taxonomy/scope maps name `consolidate`

### Sibling & mirror sweep (constitution + `code-quality.md` § Sibling & Mirror Sweeps)

#### R16: SPEC mirror
`docs/specs/skills/SPEC-fab-dedupe.md` SHALL exist and mirror the shipped skill (Constitution Additional Constraints). It SHALL follow the house shape of `SPEC-fab-draft.md` — Summary, design rationale, Flow diagram, Tools used, Sub-agents, Bookkeeping commands, difference-from-`/fab-draft` table, Re-run contract, Known limitations — and SHALL carry the corrected layered-decomposition and conversational-acceptance semantics, with no `fab-consolidate` naming and no `consolidate.memory_file` reference.

- **GIVEN** the shipped skill file
- **WHEN** the SPEC mirror is read alongside it
- **THEN** every behavioral claim in the SPEC matches the skill (name, layered Step 3, conversational Step 4, hardcoded memory home, one config key)

#### R17: Aggregate spec + help sweep, done up front
The whole sibling/mirror class SHALL be swept in this change: `docs/specs/skills.md` (a `/fab-dedupe` section plus its `helpers:` row in § Skill Helpers), `docs/specs/glossary.md` (`/fab-dedupe` in § Skills; *cluster*, *detector*, and *canonical home* as terms; `consolidate.detectors` where config keys are enumerated), `docs/specs/config.md` (the new key in the scope-taxonomy and not-set-live enumerations), **`README.md` § Command Quick Reference → § Pipeline** (a constitution-governed surface per § Toolkit Standards, and a COMPLETE 9-skill inventory — a new Planning-group skill MUST appear there or the table silently under-reports the command surface), and `src/go/fab/cmd/fab/fabhelp.go`'s `skillToGroupMap` (point 8 of skills.md § New Skill Checklist — an unmapped skill falls into the "Other" bucket) with its `fabhelp_test.go` pin.

**Count words, coverage notes, and parallel twin tables are part of the sweep class, not separate from it** (added rework cycle 2). Adding a helper consumer invalidates three things beyond name lists, and all three SHALL be swept together: (a) **count words** that quantify an enumeration ("all six declaring skills", "used by five skills", "the remaining two"); (b) **coverage notes** that assert a subset is accounted for by other means — these carry *functional* content (posture, interruption budget, escape valve), so a consumer missing from a coverage note has no behavioral coverage at all, not merely a stale count; and (c) **parallel twin tables** — byte-parallel copies of the same table in more than one file (the phase-symmetry table exists in BOTH `docs/memory/_shared/context-loading.md` § Skill Helper Declaration AND `docs/memory/memory-docs/templates.md` § `helpers:` and the One-Shared-Helper-Per-Phase Decomposition; every row of the class MUST move together, including rows for helpers other than the one that prompted the edit). No sentence in a live (non-archival) file SHALL assert a count that contradicts its own adjacent enumeration. `src/kit/skills/fab-help.md` and `docs/specs/skills/SPEC-fab-help.md` SHALL be verified: both describe a delegation to `fab fab-help` which discovers skills by frontmatter scan, so neither needs a per-skill edit — the grouping edit is the real sibling.

**The sweep unit is the CLASS OF ENUMERATIONS, never a file list** (added rework cycle 3 — the root cause of three consecutive failures). Three cycles each fixed the sites a reviewer named and shipped; each time the same defect survived one hop away, because the sweep tracked *the file it had just edited* rather than *the class of prose constructs the change invalidates*. The binding procedure is therefore **enumerate-the-class-then-verify**, and it is not satisfied by fixing a reported site list:

1. **Enumerate the class first, before editing anything.** Grep live prose for every construct that enumerates a planning-skill set or a shared-helper consumer set — name lists, count words, coverage notes, `§ Invocation`-style "who does X" lists, helper-declaration tables, twin/parallel tables, and **ASCII flow diagrams** (a diagram line is live prose and drifts identically — cycle 3 found stale consumer lists inside `SPEC-_srad.md`'s and `SPEC-_generation.md`'s flow diagrams while both files' Summary paragraphs were already correct). Useful patterns: `fab-new.*fab-draft`, `fab-draft.*fab-clarify`, `remaining (two|three)`, `all (six|seven)`, `(five|six) skills`, `(three|four) consumers`, `planning skills`, `declare .?_(srad|generation|intake)`, `persist(ed)? the intake score`, `after intake generation`, `assumed: \{description\}`.
2. **Adjudicate every hit with explicit reasoning** — belongs (fix) or does not belong (say why). A per-skill-scoped statement (`/fab-new` produces `intake.md`), an error string, or historical `*Introduced by*`/`*Why*` narration is NOT an enumeration and MUST be left alone; a "who does X" list where `/fab-dedupe` does X IS one.
3. **Sweep the parallel restatement of every line edited.** A canonical line in `src/kit/skills/` almost always has a spec-side twin (`docs/specs/`) AND a memory-side summary (`docs/memory/`) that restates it in different words — grep the *claim*, not the phrase. Cycle 3's `_preamble.md § Invocation` fix required parallel edits in `docs/specs/srad.md`, `docs/memory/_shared/context-loading.md`, `docs/memory/memory-docs/templates.md`, `docs/specs/templates.md`, and `src/kit/skills/_cli-fab.md` — five restatements of one sentence.
4. **Self-verify by re-running the greps.** Assert: no live sentence contradicts its own adjacent enumeration; no file names `/fab-dedupe` in one enumeration while omitting it from a parallel one; every count word matches the list it quantifies.

- **GIVEN** the new skill
- **WHEN** the aggregate specs are read
- **THEN** `/fab-dedupe` appears in `skills.md`, `glossary.md`, and `skillToGroupMap`, and `consolidate.detectors` appears in `config.md` and `glossary.md`
- **AND** the enumerate-the-class-then-verify procedure above was executed and its verification greps returned no surviving contradiction

#### R18: `_cli-fab.md` verified, not assumed
`src/kit/skills/_cli-fab.md` SHALL be checked for a needed update. The constitution's CLI constraint binds on "new or changed command signatures". This change adds no `fab` subcommand and changes no command signature — it adds a config registry row that existing commands render — so the expected outcome is **no change**, and the verification result SHALL be recorded as a plan assumption rather than left implicit.

- **GIVEN** the change's Go diff
- **WHEN** `_cli-fab.md` is checked against it
- **THEN** no command signature documented there has changed, and the file is left unmodified with the verification recorded

#### R19: Backlog disposition
`fab/backlog.md`'s `[ruaw]` entry SHALL be annotated as substantially absorbed by `/fab-dedupe` — its inventory arrives as consolidation output rather than as a curated artifact — naming the residue that is NOT built here (the reuse ledger in `plan.md`, apply-entry inventory priming, the `reuse` acceptance category). It SHALL NOT be silently left as though untouched, and SHALL NOT be deleted. `[cmap]` SHALL be left unchanged.

- **GIVEN** `fab/backlog.md`
- **WHEN** `[ruaw]` is read after this change
- **THEN** it carries an annotation naming this change and the surviving residue
- **AND** `[cmap]`'s text is byte-identical to before

#### R20: Shared-helper consumer sets name `/fab-dedupe`
Adding a skill that declares `helpers: [_generation, _srad, _intake]` makes it a **consumer** of two shared helpers, and every file that *enumerates* a helper's consumer set SHALL name it. For `_intake` that is: `src/kit/skills/_intake.md` (frontmatter `description:`, the header blockquote consumer list, the Step 8 `interactive` binding), its constitution-required mirror `docs/specs/skills/SPEC-_intake.md` (Summary, the `{questioning-mode}` parameter table, the `interactive` bullet, the Helpers paragraph, the Flow-diagram consumer line), and `docs/memory/pipeline/planning-skills.md` (frontmatter `description:`, the `interactive` bullet, the per-consumer call-site table, the helper-declaration mechanics paragraph, the `Pre-Boundary Intake-Creation Extracted to _intake` Design Decision). For `_srad` that is `src/kit/skills/_srad.md`, `src/kit/skills/_preamble.md` § SRAD pointer, their SPEC mirrors, `docs/specs/glossary.md` § SRAD, and `docs/memory/_shared/context-loading.md`. `/fab-dedupe` is the **first fan-out consumer** of `_intake` — the one behavioral fact the enumerations must carry beyond the name: it runs Steps 0–9 once per accepted cluster group, N times per invocation. Append-only change logs (`log.md`, `log.seed.md`) and dated finding archives (`docs/specs/findings/`) are frozen historical records and SHALL NOT be swept.

**The consumer-set class is defined BEHAVIORALLY, not by helper name** (added rework cycle 3). A file enumerates a "consumer set" whenever it lists *which skills do a thing that `/fab-dedupe` does* — not merely when it lists `helpers:` declarations. The file lists above are **examples of the class, never its definition**; membership is decided by asking, for each enumeration found by the R17 sweep, "does `/fab-dedupe` do this?" `/fab-dedupe` **does**: declare all three helpers; run `_intake` Steps 0–9 interactively (incl. Step 7 `fab score --stage intake`, hence it **persists intake scores**); reach `_generation` § Intake Generation via `_intake` Step 5; place SRAD `<!-- assumed: -->` **artifact markers**; emit the **Activation Preamble** `Next:` line per drafted intake; and consume the `_srad` autonomy table via the `/fab-new` column. It **does not**: activate a change, create a git branch, take a backlog/Linear ID (so ID-collision enumerations correctly exclude it), or act as a `/fab-proceed` prefix step. Four behavioral enumerations beyond the helper name lists bind, each with several parallel restatements: **score persisters** (`_preamble.md` § Confidence Scoring → Invocation, `docs/specs/srad.md` § Confidence Lifecycle, `docs/specs/templates.md`, `docs/memory/_shared/context-loading.md`, `docs/memory/memory-docs/templates.md`, `src/kit/skills/_cli-fab.md` § Template), **artifact-marker placers** (`_srad.md` § Artifact Markers + `docs/specs/srad.md`'s parallel — functionally consequential: a `/fab-dedupe` reading that it places no markers would leave its own Step 5 Tentative rows unmarked and invisible to `/fab-clarify`'s scan), **Activation-Preamble emitters** (`_preamble.md` § Activation Preamble + `docs/memory/_shared/context-loading.md`), and the **`_srad` autonomy covering note** (`_srad.md`, `docs/specs/srad.md`, `SPEC-_srad.md` Summary *and* its flow diagram, `planning-skills.md`'s SRAD Design Decision). Append-only change logs (`log.md`, `log.seed.md`) and dated finding archives (`docs/specs/findings/`, `docs/findings/`) are frozen historical records and SHALL NOT be swept.

- **GIVEN** the shipped `/fab-dedupe` skill declaring `helpers: [_generation, _srad, _intake]`
- **WHEN** `src/kit/skills/_intake.md`, `SPEC-_intake.md`, and `docs/memory/pipeline/planning-skills.md` are read
- **THEN** each names four `_intake` consumers, not three, and identifies `/fab-dedupe` as the fan-out call site
- **AND** no live (non-archival) file still claims the `_intake` consumer set is exactly three skills
- **AND** every behavioral enumeration `/fab-dedupe` belongs to — score persisters, artifact-marker placers, Activation-Preamble emitters, and the `_srad` covering note — names it in **every** parallel restatement across `src/kit/skills/`, `docs/specs/`, and `docs/memory/`

### Non-Goals

- **No refactor of the `setupXRepo` test-helper cluster.** That is the first *use* of the skill and belongs in its own change (the dry run that discovered it is evidence, not scope).
- **No `fab` subcommand, no new binary command surface, no Constitution amendment.**
- **No `consolidate.memory_file` registry key** — resolved at intake (O1): hardcode the memory home; adding the key later is cheap and non-breaking.
- **No structured multi-select for cluster acceptance** — resolved at intake (O2): conversational reply.
- **No cross-repo duplication detection** — a materially harder problem, explicitly out of scope.
- **No creation of `docs/memory/_shared/utilities.md`** — this change creates the skill; the file comes into existence the first time a real `/fab-dedupe`-drafted consolidation is hydrated.
- **No fix for the PRE-EXISTING `_srad`/helper-mapping drift caused by `/fab-adopt`.** `fab-adopt.md` declares `helpers: [_srad, _generation, _review, _pipeline]` but is absent from the `_srad` consumer enumerations and from `docs/memory/_shared/context-loading.md`'s helper-mapping table (whose "All others (16 skills)" count is correspondingly stale — the true residue is 18 of 27 user-facing skills). Reproduced at HEAD, independent of this change. R20's sweep adds `/fab-dedupe` and drops the now-wrong word "six" from the `_srad` enumerations, but does NOT backfill `fab-adopt` or recount the table — that is a distinct pre-existing defect and belongs in its own change (Constitution: fix root causes, not adjacent symptoms opportunistically).

### Design Decisions

#### Detectors seed; the agent clusters
**Decision**: Duplicate-detection CLIs are an optional, fail-silent *seed* for the sweep; the agent's reading of function bodies is the load-bearing clustering mechanism.
**Why**: Verified 2026-07-28 that no shipping CLI performs true Type-4 (semantically-equivalent) clone detection, and the `src/go` dry run confirmed a token-based detector would have contributed nothing on the target cluster (the common run is ~4 lines split by divergent code). Making detectors optional also makes the skill polyglot on day one and correct in a repo with nothing installed.
**Rejected**: Requiring a detector — it would gate the skill on per-language tooling and still miss the Type-3/4 clusters that are worth consolidating.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill

#### Clusters decompose into a shared core plus opt-in variation layers
**Decision**: Step 3 records a base behavior every member needs plus named opt-in layers with their member lists, and Step 4 counts "needs only the base layer" as LOW divergence.
**Why**: The dry run on four `src/go` fixtures proved the layered shape (tempdir+`fab/changes` for all; +project config for two; +change dir for three; +active symlink for one; +kit override for one) and that `resolve_test.go` had already factored `createChange` out — the right decomposition. A single flat signature per cluster forces every member's needs into one API and produces a bad one.
**Rejected**: The draft's flat "proposed signature" field — it distorts layered clusters and mis-ranks members whose bodies differ only by which layers they opt into.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill

#### The skill drafts intakes; the pipeline refactors
**Decision**: `/fab-dedupe` is a thin call-site over the `_intake` Create-Intake Procedure (Steps 0–9, interactive) with a cluster-analysis front end and a fan-out tail, stopping at intake `ready`.
**Why**: Collapsing overlapping helpers is design work with real semantic risk, and the pipeline already has a gated stage for it. The intake's `## Assumptions` SRAD table is a natural per-cluster risk record and `fab score`'s flat 3.0 gate already refuses to auto-run a change whose divergences grade Tentative.
**Rejected**: A skill with its own refactor loop (bypasses the SRAD gate, `code-quality.md`, and the reviewer); a bespoke report artifact (reinvents the gate).
*Introduced by*: 260728-4v91-add-fab-dedupe-skill

#### `consolidate` is registered in `configscope`, not only in `configref`
**Decision**: Adding the `consolidate.detectors` registry row requires a paired `"consolidate": ScopeProject` entry in `internal/configscope`'s `keyScopes`.
**Why**: `configref.lintFields` cross-checks every row's Scope against `configscope.ScopeFor(topLevel(key))` and rejects a mismatch; an unregistered top-level key resolves to the empty scope, so omitting the entry makes `configref.Fields()` return an error and breaks `fab config reference`/`upgrade`/`init` wholesale. The taxonomy is deliberately single-sourced in the leaf package.
**Rejected**: Adding the row to `configref` alone (the intake's literal "ONE registry key" reading) — it fails loud at construction; the constraint is one *registry row*, not one *file touched*.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill

## Tasks

### Phase 1: Setup & Verification

- [x] T001 [P] Verify the sibling/mirror sweep class up front: grep the repo for `fab-consolidate`, `consolidate.memory_file`, and the skill-inventory surfaces (`docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/config.md`, `src/kit/skills/fab-help.md`, `docs/specs/skills/SPEC-fab-help.md`, `src/go/fab/cmd/fab/fabhelp.go`); record the concrete edit list before touching any file <!-- R17 -->
- [x] T002 [P] Verify `src/kit/skills/_cli-fab.md` against this change's Go diff (no new/changed `fab` command signature is introduced — only a config registry row rendered by existing commands); record the verification outcome as a plan assumption rather than editing the file <!-- R18 -->

### Phase 2: Core Implementation — the skill and its mirror

- [x] T003 Rewrite `src/kit/skills/fab-dedupe.md` from the draft: rename `fab-consolidate` → `fab-dedupe` throughout (frontmatter `name`, `# /fab-dedupe [scope]`, `fab log command`, all prose), keep `helpers: [_generation, _srad, _intake]`, keep the no-`fab preflight` pre-flight and the read-only-until-accept framing <!-- R1 -->
- [x] T004 In `src/kit/skills/fab-dedupe.md`, keep/confirm the Pre-flight, Arguments (scope resolution + member-vs-finding rule), Context Loading, and Command Logging sections and the Detector Configuration section (config-driven default, `{paths}`/`{out}` substitution, probe-then-fail-silent, non-zero-exit-is-a-finding exception stated in those words, per-language suggested-detector table, and the two plain facts about Type-4 and Go's Type-1/2 ceiling) <!-- R2 R4 R5 -->
- [x] T005 Fix the Step 3 flat-signature bug in `src/kit/skills/fab-dedupe.md`: replace the single "Proposed signature" cluster field with a **layered decomposition** — base behavior (all members) + named opt-in layers with their member lists + the unified API expressing that shape (functional options / options struct / base + wrappers / explicit "keep separate") — and state the dry-run rationale; retain the no-name-clustering and cluster-of-one rules <!-- R6 -->
- [x] T006 Fix Step 4 in `src/kit/skills/fab-dedupe.md`: rank by call-site count ↑ / divergence ↓ with "member needs only the base layer" counted as **low** divergence; replace the report sample with one showing `Base:` and `Layers:` lines; collect acceptance as a **plain conversational reply** (`all` / `1,3` / `none`), explicitly not a structured multi-select <!-- R7 R8 -->
- [x] T007 Confirm/repair Step 5 in `src/kit/skills/fab-dedupe.md`: the Create-Intake Procedure binding (Steps 0/2/5/8), one-intake-per-cluster preference, STOP-after-Step-9, plus the Output, Error Handling, and Key Properties sections and the trailing `Next:` line <!-- R9 R11 -->
- [x] T008 Remove the `consolidate.memory_file` override from `src/kit/skills/fab-dedupe.md` § Memory Home — hardcode `docs/memory/_shared/utilities.md`, keep the rejected-top-level-domain rationale, the hydrate-writes-it property, and the coverage header <!-- R10 -->
- [x] T009 Rewrite `docs/specs/skills/SPEC-fab-dedupe.md` to mirror the corrected skill: rename to `fab-dedupe`, replace the flat-signature language with layered decomposition (shared core + variation layers, and the ranking consequence), state conversational acceptance, drop the `consolidate.memory_file` override, and keep the house shape (Summary, rationale, Flow, Tools, Sub-agents, Bookkeeping, difference-from-`/fab-draft`, Re-run contract, Known limitations) <!-- R16 -->

### Phase 3: Core Implementation — the config registry

- [x] T010 Add `"consolidate": ScopeProject` to `keyScopes` in `src/go/fab/internal/configscope/configscope.go` (required by the `configref` lint's single-source cross-check) <!-- R13 -->
- [x] T011 Add the single `consolidate.detectors` `Field` row to `Fields()` in `src/go/fab/internal/configref/configref.go` after the `checklist.extra_categories` row, following that row's shape exactly: `Default: nil`, `Scope: ScopeProject`, `Advertise: true`, the intake's verbatim Description, and the intake's verbatim Segment. Add no `consolidate.memory_file` row <!-- R12 R14 -->

### Phase 4: Tests

- [x] T012 [P] Extend `src/go/fab/internal/configscope/configscope_test.go`'s `TestScopeFor` taxonomy pin with `"consolidate": ScopeProject` <!-- R13 R15 -->
- [x] T013 [P] Extend `src/go/fab/cmd/fab/config_test.go`: add `consolidate.detectors` to `TestConfigReferenceScopeAssignments`' pinned map, and add a test asserting the rendered reference documents `consolidate.detectors` commented-out (not parsed live), that the `--json` dump carries it with `default: null` / `advertise: true` / `scope: project`, and that no `consolidate.memory_file` key exists in the registry <!-- R12 R14 R15 -->
- [x] T014 Verify the `configupgrade` golden fixtures against the new registry: run `go test ./internal/configupgrade/...`, and if the shipped-registry-backed tests churn, regenerate/update them; add a `configupgrade` test asserting the managed fence advertises `consolidate.detectors` <!-- R14 R15 -->
- [x] T015 Run the scoped Go suites — `go test ./internal/configscope/... ./internal/configref/... ./internal/configupgrade/... ./internal/config/... ./cmd/fab/...` — from `src/go/fab`, and fix failures to conform to the spec (never the reverse) <!-- R15 -->

### Phase 5: Sibling & mirror sweep

- [x] T016 Add the `/fab-dedupe` section to `docs/specs/skills.md` (purpose, context, creates, arguments, example, behavior — matching the `/fab-draft` section's shape) and add its `helpers:` row to § Skill Helpers § Current mapping <!-- R17 -->
- [x] T017 [P] Update `docs/specs/glossary.md`: add `/fab-dedupe` to § Skills, add *Cluster*, *Detector*, and *Canonical home* to the appropriate groups, and name `consolidate.detectors` in the `config.yaml` row <!-- R17 -->
- [x] T018 [P] Update `docs/specs/config.md`: add `consolidate.detectors` to the § Scope taxonomy project-scope enumeration and to the § Advertise semantics not-set-live enumeration <!-- R17 -->
- [x] T019 Add `"fab-dedupe": "Planning"` to `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go` (skills.md § New Skill Checklist point 8) and add `fab-dedupe` to the `expectedMapped` list in `TestFabHelp_GroupMapping` in `src/go/fab/cmd/fab/fabhelp_test.go`; leave `src/kit/skills/fab-help.md` and `docs/specs/skills/SPEC-fab-help.md` unedited (both describe a frontmatter-scan delegation that picks the skill up automatically) and record that verification <!-- R17 R15 -->
- [x] T020 Annotate `[ruaw]` in `fab/backlog.md` as substantially absorbed by `/fab-dedupe`, naming the surviving residue (plan.md reuse ledger, apply-entry inventory priming, the `reuse` acceptance category); leave `[cmap]` untouched <!-- R19 -->

### Phase 6: Polish

- [x] T021 Final consistency sweep: grep the repo for `fab-consolidate` and `consolidate.memory_file` (expect zero hits outside `fab/changes/`), confirm no file under `.claude/skills/` was edited, and re-run the scoped Go suites <!-- R1 R10 R15 -->

### Phase 7: Rework cycle 1 — complete the sibling/mirror sweep (review must-fix + should-fix)

> Added after review cycle 1 failed on an incomplete sweep. Requirements R17 (aggregate sweep) and R16 (SPEC-mirror sync) already bind these files; the gap was coverage, not intent. R20 below states the newly-discovered class member explicitly.

- [x] T022 Declare `/fab-dedupe` as the fourth `_intake` consumer in `src/kit/skills/_intake.md`: frontmatter `description:`, the header blockquote (`three skills` → `four skills`, noting the fan-out call site), and the Step 8 `{questioning-mode} = interactive` binding <!-- R20 -->
- [x] T023 Mirror T022 into `docs/specs/skills/SPEC-_intake.md` (constitution-required): the Summary consumer sentence, the `{questioning-mode}` parameter table (three consumer columns → four), the `interactive` bullet, the Helpers paragraph, and the Flow-diagram consumer line <!-- R20 R16 -->
- [x] T024 [P] Add `/fab-dedupe [scope]` to the `docs/specs/overview.md` § Quick Reference skill table, between `/fab-draft` and `/fab-continue` <!-- R17 -->
- [x] T025 Update `docs/memory/pipeline/planning-skills.md` for the fourth `_intake` consumer: frontmatter `description:` skill list, the `interactive` mode bullet, the per-consumer call-site table (three → four rows, with the fan-out semantics), the helper-declaration mechanics paragraph, and an `*Updated by*:` suffix on the `Pre-Boundary Intake-Creation Extracted to _intake` Design Decision. Also correct the intake's `## Affected Memory` path (`pipeline/skills-overview` → `pipeline/planning-skills`) so hydrate does not miss it <!-- R20 R17 -->
- [x] T026 Extend `TestScope_PruneAllProjectScopedFields` in `src/go/fab/internal/config/config_test.go` to cover R13's second GIVEN/THEN clause: add `consolidate` to the seeded system-config map and to the pruned-key enumeration, assert a NAMED per-key pruning warning for every enumerated key, and bump the warning-count pin 7 → 8 <!-- R13 R15 -->
- [x] T027 [P] Sweep the `_srad` declared-by enumeration for the new consumer (`/fab-dedupe` declares `helpers: [_generation, _srad, _intake]`): `src/kit/skills/_srad.md` header, `src/kit/skills/_preamble.md` § SRAD pointer, their SPEC mirrors (`SPEC-_srad.md`, `SPEC-_preamble.md`), `docs/specs/skills/SPEC-_preamble.md`'s helper-allowlist row, `docs/specs/glossary.md` § SRAD, `docs/memory/_shared/context-loading.md` (helper mapping row, phase-symmetry table, SRAD paragraph), and `docs/memory/pipeline/planning-skills.md`'s SRAD Design Decision <!-- R20 R17 -->
- [x] T028 Reorder the three new `docs/specs/glossary.md` § Core Concepts rows (*Canonical home*, *Cluster*, *Detector*) into the group's stated alphabetical order, leaving the pre-existing misordering (*Memory files*, *Design specs*) untouched <!-- R17 -->

### Phase 8: Rework cycle 2 — count words, coverage notes, and parallel twin tables

> Added after review cycle 2 failed on the SAME defect class one hop out: cycle 1 swept the `_intake`/`_srad` consumer **name lists** but left the **count words** and **coverage notes** attached to them, two of them inside sentences cycle 1 itself edited (a sentence listing `fab-dedupe` in the helpers enumeration while asserting "all six declaring skills" two clauses earlier). R17 now states the count/coverage-note/twin-table rule explicitly. The lesson: **adding a helper consumer means sweeping counts, coverage notes, and parallel twin tables — not just name lists.**

- [x] T029 Add `fab-dedupe` to the `pre-intake orchestration` row's Consumers cell in the phase-symmetry table at `docs/memory/memory-docs/templates.md` § `helpers:` and the One-Shared-Helper-Per-Phase Decomposition — the byte-parallel twin of the table at `docs/memory/_shared/context-loading.md` § Skill Helper Declaration that cycle 1 updated, matching the twin's wording exactly <!-- R20 R17 -->
- [x] T030 Fix the § Skill-Specific Autonomy Levels **covering note** in `src/kit/skills/_srad.md`: correct the count word (`remaining two` → `remaining three`) and name `/fab-dedupe` as following the **fab-new** column (exactly like `fab-draft`), giving it explicit posture / interruption-budget / escape-valve coverage. This closes a live FUNCTIONAL gap, not a stale count: `/fab-dedupe` declares `helpers: [_generation, _srad, _intake]`, so it loads this file, reads the 4-column table, and previously found no coverage for itself <!-- R20 -->
- [x] T031 Mirror T030's wording into both of its parallel surfaces, which cycle 1 edited into self-contradiction by inserting `fab-dedupe` into an adjacent helpers list while leaving "all six declaring skills" intact: `docs/specs/skills/SPEC-_srad.md` § Summary (the constitution-required SPEC mirror of T030's `src/kit/skills/*.md` edit) and the § SRAD Autonomy Framework Design Decision in `docs/memory/pipeline/planning-skills.md` <!-- R20 R16 -->
- [x] T032 Add `/fab-dedupe [scope]` to `README.md` § Command Quick Reference → § Pipeline (after `/fab-draft`) — a COMPLETE 9-skill inventory that omitted the new skill even though it maps to `Planning` in `skillToGroupMap` alongside two skills that have rows; README.md is constitution-governed (§ Toolkit Standards). Also add README.md to R17's sweep-class list so the gap cannot recur <!-- R17 -->
- [x] T033 Trim `docs/memory/pipeline/planning-skills.md`'s frontmatter `description:` to ≤500 runes (it reached 511 when cycle 1 inserted `/fab-dedupe`, breaching the FKF §3.2 cap and raising a `fab memory-index --check` warning absent at HEAD). Index regeneration at hydrate does NOT fix this — only trimming does. Drop the `/fab-adopt` Intake-from-Diff/Plan-from-Diff parenthetical (detail already in the body); verify with `fab memory-index --check` <!-- R17 -->
- [x] T034 Sweep the count word and twin-table rows the cycle-1/cycle-2 name-list passes left behind on the OTHER helper `/fab-dedupe` declares — found by the mandated self-check grep, same defect class: `src/kit/skills/_generation.md`'s header (`five skills` → `six skills`, naming `/fab-dedupe` and how it reaches the Intake Generation Procedure via `_intake.md` Step 5, exactly as its structural twin `/fab-draft` does) plus its constitution-required mirror `docs/specs/skills/SPEC-_generation.md`, and the `artifact mechanics` row of BOTH copies of the phase-symmetry twin table (`docs/memory/_shared/context-loading.md`, `docs/memory/memory-docs/templates.md`), whose `_generation` cells omitted `fab-dedupe` while the `_intake` row directly beneath them named it <!-- R20 R16 R17 -->
- [x] T035 Verify no NEW `fab memory-index --check` warning is introduced by this change: diff the full warning set against HEAD. Cycle 2's own T031 edit traded the description-length warning for a `narration-density` warning (6 markers) by writing an unsanctioned prose change-id (`(c5tr; extended for fab-dedupe in 4v91)` — a change-id inside a multi-token parens group counts as narration, whereas a lone-parenthesized `(4v91)` is sanctioned per FKF §3.3). Rewrite it, plus the two cycle-1 prose ids on the same file (`(3xaj; /fab-dedupe added 4v91)` and `four as of 4v91 with /fab-dedupe`), into sanctioned lone-paren citations <!-- R17 -->

### Phase 9: Rework cycle 3 — class-wide exhaustiveness (the FINAL cycle)

> Added after review cycle 3 failed on the SAME defect class a third time, now including `docs/specs/srad.md` — the live SRAD spec, untouched by any prior cycle, carrying a byte-parallel copy of the exact sentence cycle 2 rewrote in `_srad.md`. The reviewer's diagnosis is the root cause: **"the sweep tracks the file it just edited rather than the class of enumerations."** This cycle therefore works from the CLASS: enumerate every live-prose construct the change invalidates (name lists, count words, coverage notes, § Invocation-style lists, twin tables, AND flow diagrams), adjudicate each with reasoning, fix every member, then re-run the greps. R17 and R20 above are rewritten to make enumerate-the-class-then-verify the binding requirement rather than a file list. Scope enumerated up front: 60 matching sites across 19 live files, individually adjudicated.

- [x] T036 Fix the `_srad` § Artifact Markers "Placed by" enumeration in `src/kit/skills/_srad.md` — add `fab-dedupe` to `All planning skills (...)`. **Functionally consequential, not a stale count**: `/fab-dedupe` declares `_srad`, loads this file, and previously read that it does not place `<!-- assumed: -->` markers, which would leave the Tentative divergence rows its own Step 5 produces unmarked and invisible to `/fab-clarify`'s scan <!-- R20 -->
- [x] T037 Mirror T036 into `docs/specs/srad.md`'s parallel artifact-marker table row (spec-side twin of the same enumeration) <!-- R20 R16 -->
- [x] T038 Fix `docs/specs/srad.md`'s § Skill-Specific Autonomy Levels covering note (`remaining two` → `remaining three`, naming `/fab-dedupe` as following the `/fab-new` column) — the byte-parallel spec twin of `_srad.md:73`, which cycle 2 fixed while leaving this copy false. The file cites `_srad.md` as canonical, so it contradicted its own declared source <!-- R20 R17 -->
- [x] T039 Fix the score-persister enumeration in `src/kit/skills/_preamble.md` § Confidence Scoring → Invocation — `/fab-dedupe` runs `_intake` Step 7 (`fab score --stage intake`) per accepted cluster group, so one invocation persists N intake scores <!-- R20 -->
- [x] T040 Sweep ALL FIVE parallel restatements of T039's sentence, found by grepping the *claim* rather than the phrase: `docs/specs/srad.md` § Confidence Lifecycle "Computation" row, `docs/specs/templates.md`'s `confidence` block bullet, `docs/memory/_shared/context-loading.md`'s § Confidence Scoring summary of the preamble's Invocation list, `docs/memory/memory-docs/templates.md`'s confidence-block paragraph, and `src/kit/skills/_cli-fab.md` § fab score (extended) → Template. One sentence, six live homes — the exact shape of the three-cycle failure <!-- R20 R17 -->
- [x] T041 Fix the flow-diagram covering note in `docs/specs/skills/SPEC-_srad.md` (`fab-draft = fab-new's column, fab-clarify = the escape valve`) — the SAME file's Summary already named `fab-draft/fab-dedupe/fab-clarify`, so the file contradicted itself. Establishes that ASCII flow diagrams are live prose in the sweep class <!-- R20 R16 -->
- [x] T042 Sweep the `_generation` consumer enumerations the cycle-2 pass left behind, found by the mandated self-check: `src/kit/skills/_generation.md`'s frontmatter `description:` (omitted `fab-dedupe` while the body header two lines below said "six skills" — intra-file contradiction) and the Intake-Generation-Procedure consumer line in `docs/specs/skills/SPEC-_generation.md`'s flow diagram (contradicting its own Summary) <!-- R20 R16 -->
- [x] T043 Add `/fab-dedupe` to the Intake Generation Procedure consumer list in `docs/memory/pipeline/planning-skills.md` § Shared Generation Partial — it contradicted the same file's `interactive` bullet and helper-declaration paragraph, both of which name `/fab-dedupe` <!-- R20 R17 -->
- [x] T044 Fix the **Activation-Preamble emitter** enumeration — a behavioral consumer set no prior cycle considered, discovered by the class sweep rather than reported: `src/kit/skills/_preamble.md` § Activation Preamble and its memory-side restatement in `docs/memory/_shared/context-loading.md` both listed only `/fab-draft` + `/fab-archive restore`, yet `fab-dedupe.md:231` emits the preamble per drafted intake <!-- R20 R17 -->
- [x] T045 Verify no NEW `fab memory-index --check` warning: diff the full warning set against HEAD via a detached `main` worktree; confirm all `4v91` citations in `docs/memory/**` are FKF §3.3-sanctioned lone-paren `(4v91)` form and every touched `description:` is ≤500 runes <!-- R17 -->
- [x] T046 Final class-wide self-verification: re-run the enumeration greps (count words, `declare _srad|_generation|_intake`, persister phrases, artifact-marker rows, and a per-file intra-contradiction scan over every file naming `fab-dedupe`) and adjudicate every residual hit as scoped-statement / error-string / historical-narration rather than enumeration <!-- R17 R20 -->

## Execution Order

- T001–T002 (verification) precede all edits.
- T003 precedes T004–T008 (they edit the same file).
- T010 blocks T011 (the `configref` lint fails without the `configscope` entry).
- T012–T015 follow T010–T011.
- T019's Go edit is independent of Phase 3 but shares the `cmd/fab` test suite, so T015's rerun in T021 covers both.
- T021 runs last **of the original pass**.
- **Rework cycle 1**: T022 precedes T023 (the SPEC mirrors the skill, so the skill's wording is written first). T024, T027, T028 are independent `[P]`. T026 is Go-only and re-runs the `internal/config` suite. T025 both edits memory and corrects the intake's Affected Memory path — do them together so the path fix and the file it points at land in one step.
- **Rework cycle 3**: the cycle OPENS with the class enumeration (the grep sweep behind T046), not with the reported site list — the site list is a starting point and the greps define the scope. The skill-then-SPEC-mirror rule holds throughout (T036 before T037, T039 before its spec/memory restatements in T040, `_generation.md` before `SPEC-_generation.md` within T042). T038, T041, T043, T044 are independent. **T045 and T046 run last** — T045 is the FKF warning diff-vs-HEAD gate and T046 the class-wide grep re-verification, so both must observe every other task's final bytes. No `.go` file is touched in this cycle; the scoped Go suites are re-run only as a regression guard.
- **Rework cycle 2**: T030 precedes T031 (same skill-then-SPEC-mirror rule as cycle 1 — the covering note's wording is authored in `_srad.md`, then mirrored). Within T034 the same rule applies (`_generation.md` before `SPEC-_generation.md`). T029, T032, T033 are independent. **T035 runs last** — it is the FKF-warning diff-vs-HEAD gate, and T033's description trim plus T031's Design-Decision rewrite both feed it, so it must observe their final bytes. No `.go` file is touched in this cycle.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/fab-dedupe.md` exists with `name: fab-dedupe`, a behavior-naming description, `helpers: [_generation, _srad, _intake]`, and the standard preamble-read blockquote; no `fab-consolidate` string remains
- [x] A-002 R2: The skill's Pre-flight checks config + constitution only and explicitly forbids `fab preflight`
- [x] A-003 R3: The skill states it never refactors, never writes memory, never activates a change, and never creates a branch, and its Key Properties table reflects that
- [x] A-004 R4: Step 1 documents path/glob, natural-language, and bare-invocation scope resolution, echoes the resolved set + file count, and states the scope-filters-findings-not-members rule
- [x] A-005 R5: Step 2 / § Detector Configuration document `consolidate.detectors` with the jscpd default, `{paths}`/`{out}` substitution, `command -v` probe-then-fail-silent, and the non-zero-exit exception stated as an explicit per-skill exception to `_preamble.md` § Common fab Commands
- [x] A-006 R6: Step 3 records a base layer + named opt-in layers + a unified API per cluster, and contains no single flat "proposed signature" field
- [x] A-007 R7: Step 4 ranks by call-sites ↑ / divergence ↓ and states that base-layer-only membership counts as LOW divergence
- [x] A-008 R8: Step 4 collects acceptance as a plain conversational reply with the `(all / 1,3 / none)` grammar and no structured picker
- [x] A-009 R9: Step 5 binds Create-Intake Procedure Steps 0/2/5/8 explicitly and STOPs after Step 9 with no activation and no branch
- [x] A-010 R10: The memory home is hardcoded to `docs/memory/_shared/utilities.md`; `consolidate.memory_file` appears nowhere in the skill or SPEC
- [x] A-011 R11: The skill carries the Output rules (per-intake name + confidence, Activation Preamble `Next:`, no `Next:` when nothing accepted), the Error Handling table with all eight rows, and the Key Properties table
- [x] A-012 R12: `configref.Fields()` returns exactly one new row, `consolidate.detectors`, with nil Default / ScopeProject / Advertise true / the intake's Description and Segment, and no `consolidate.memory_file` row
- [x] A-013 R13: `configscope.keyScopes` carries `"consolidate": ScopeProject` and `configref.Fields()` returns no lint error — verified empirically: deleting the entry makes `Fields()` error with `configscope says ""`
- [x] A-014 R14: `fab config reference` renders the `consolidate.detectors` segment commented-out exactly once, `--json` carries the matching entry, and the managed fence advertises it
- [x] A-015 R16: `docs/specs/skills/SPEC-fab-dedupe.md` mirrors the shipped skill on every behavioral claim (name, layered Step 3, conversational Step 4, hardcoded memory home, one config key)
- [x] A-016 R17: `/fab-dedupe` appears in `docs/specs/skills.md` (section + helpers row), `docs/specs/glossary.md` (skill + the three new terms), and `skillToGroupMap`; `consolidate.detectors` appears in `docs/specs/config.md` and `docs/specs/glossary.md`
- [x] A-017 R19: `fab/backlog.md`'s `[ruaw]` is annotated as substantially absorbed with the residue named; `[cmap]` is unchanged

### Behavioral Correctness

- [x] A-018 R6: A layered cluster (base needed by all, layers needed by subsets) is representable in the Step 3 record without collapsing to one signature — verified against the intake's four-member `setupXFixture` worked example
- [x] A-019 R7: A member whose body differs but which needs only the base layer is rated LOW, not HIGH, divergence
- [x] A-020 R14: `fab config upgrade` remains idempotent over its own output with the new advertised key (byte-identical re-run) — verified by running `config upgrade` twice over a scratch project; second pass byte-identical

### Scenario Coverage

- [x] A-021 R15: `go test ./internal/configscope/... ./internal/configref/... ./internal/configupgrade/... ./internal/config/... ./cmd/fab/...` passes from `src/go/fab` — all packages green except two PRE-EXISTING `cmd/fab` failures (`TestConfigShowOrigin_Provenance`, `TestConfigShowOrigin_HigherLayerScalarReplacesSubtree`), a macOS `/private/var` TMPDIR-symlink artifact in a file this change does not touch, independently reproduced at HEAD with all changes stashed
- [x] A-022 R13: `TestScopeFor` pins `consolidate` → project, and `TestConfigReferenceScopeAssignments` pins `consolidate.detectors` → project
- [x] A-023 R17: `TestFabHelp_GroupMapping` includes `fab-dedupe` in its `expectedMapped` list and it maps to a known group

### Edge Cases & Error Handling

- [x] A-024 R5: The skill's documented behavior for a missing detector binary is a silent skip (no error, no warning) recorded as information in the report
- [x] A-025 R5: The skill's documented behavior for a non-zero detector exit is continue-with-noted-exit-code, not STOP
- [x] A-026 R11: Zero accepted clusters, zero found clusters, and a per-cluster `fab change new` failure each have a documented, non-fatal outcome

### Code Quality

- [x] A-027 Pattern consistency: The new registry row and scope entry follow the surrounding rows' shape and comment style exactly; the skill and SPEC follow the `/fab-draft` + `SPEC-fab-draft.md` house shapes
- [x] A-028 No unnecessary duplication: The skill delegates to the shared `_intake` Create-Intake Procedure rather than restating it, and the registry Segment/Description are the single source both renderings walk
- [x] A-029 Canonical source only: No file under `.claude/skills/` was edited (Constitution V; `code-quality.md` § Anti-Patterns project-specific)
- [x] A-030 SPEC-mirror sync: met after rework cycle 1. Every `src/kit/skills/*.md` file this change touches carries its `SPEC-*.md` update in the same change — `fab-dedupe.md`→`SPEC-fab-dedupe.md` (original pass), plus `_intake.md`→`SPEC-_intake.md`, `_srad.md`→`SPEC-_srad.md`, and `_preamble.md`→`SPEC-_preamble.md` (T022/T023/T027). The three-consumer claim is gone from every live file: repo-wide grep for `three consumers`/`three skills` over `src/kit/` + `docs/` returns zero outside append-only change logs and dated finding archives
- [x] A-031 Go changes ship tests: Every `.go` file changed in this change has accompanying test updates in the same change (`code-review.md` § Project-Specific Review Rules) — `configref.go`→`config_test.go`+`configupgrade_test.go`, `configscope.go`→`configscope_test.go`, `fabhelp.go`→`fabhelp_test.go`; the new `TestConfigReferenceConsolidateDetectors` was mutation-verified non-vacuous
- [x] A-032 R18: `_cli-fab.md` was verified against the Go diff rather than assumed; the verification outcome is recorded and the file is correctly left unmodified — confirmed: `_cli-fab.md` documents command signatures/flags, not registry key contents, and no signature changed
- [x] A-033 No magic strings: The registry row's default detector command lives only in the rendered Segment and the skill prose (the canonical Default stays nil per the registry's empty-default convention), with no third copy

### Rework Cycle 1 — sweep completion

- [x] A-034 R20: `src/kit/skills/_intake.md` names four consumers in its frontmatter `description:`, its header blockquote, and its Step 8 `interactive` binding, and identifies `/fab-dedupe` as the fan-out call site
- [x] A-035 R20/R16: `docs/specs/skills/SPEC-_intake.md` mirrors it on all five surfaces — Summary, the `{questioning-mode}` parameter table (now four consumer columns), the `interactive` bullet, the Helpers paragraph, and the Flow-diagram consumer line
- [x] A-036 R17: `docs/specs/overview.md` § Quick Reference lists `/fab-dedupe [scope]` with its purpose and its per-accepted-cluster output
- [x] A-037 R20: `docs/memory/pipeline/planning-skills.md` documents the fourth consumer across all five surfaces (frontmatter `description:`, the `interactive` bullet, the four-row call-site table with fan-out semantics, the helper-declaration mechanics paragraph, and an `*Updated by*:` suffix on the `_intake`-extraction Design Decision)
- [x] A-038 R20: `intake.md`'s `## Affected Memory` names the real path `pipeline/planning-skills` (the original `pipeline/skills-overview` does not exist), so hydrate resolves it
- [x] A-039 R13/R15: `TestScope_PruneAllProjectScopedFields` seeds a `consolidate:` block into the system-config map, asserts it is pruned with a NAMED `fab: warning: ignoring project-scoped field "consolidate"` warning, and pins the warning count at 8 — closing R13's second GIVEN/THEN clause. Mutation-verified non-vacuous: deleting `"consolidate": ScopeProject` from `configscope.keyScopes` fails the test on all three assertions
- [x] A-040 R20: The `_srad` consumer enumerations name `/fab-dedupe` and no longer claim a stale count — swept across `_srad.md`, `_preamble.md`, `SPEC-_srad.md`, `SPEC-_preamble.md` (both the SRAD pointer line and the helper-allowlist row), `glossary.md` § SRAD, `_shared/context-loading.md` (three surfaces), and `planning-skills.md`'s SRAD Design Decision. The pre-existing `fab-adopt` omission is deliberately out of scope (see § Non-Goals)
- [x] A-041 R17: The three new `glossary.md` § Core Concepts rows sit in the group's stated alphabetical order (Canonical home → Change → Change folder → Cluster → Constitution → Detector); the pre-existing misordering elsewhere in the group is left untouched
- [x] A-042 Canonical source only (rework): No file under `.claude/skills/` was edited during rework — verified via `git status --short` (the tree is gitignored and shows no entries)

### Rework Cycle 2 — count words, coverage notes, twin tables

- [x] A-043 R20/R17: The phase-symmetry table's `pre-intake orchestration` row names `fab-dedupe` in **both** copies of the twin — `docs/memory/_shared/context-loading.md` (cycle 1) and `docs/memory/memory-docs/templates.md` (cycle 2) — with matching wording
- [x] A-044 R20: `src/kit/skills/_srad.md`'s § Skill-Specific Autonomy Levels covering note names `/fab-dedupe` as following the **fab-new** column and reads "remaining three", so every one of the 7 enumerated declaring skills has posture / interruption-budget / escape-valve coverage. Verified arithmetically self-consistent: the 4 table columns (fab-new, fab-continue, fab-fff, fab-ff) plus the 3 covered skills (fab-draft, fab-dedupe, fab-clarify) partition the 7-skill enumeration at line 12 with no overlap and no gap
- [x] A-045 R20/R16: The covering-note wording is mirrored in `docs/specs/skills/SPEC-_srad.md` § Summary ("all seven declaring skills (4 columns + a fab-draft/fab-dedupe/fab-clarify covering note)") and in `docs/memory/pipeline/planning-skills.md`'s § SRAD Autonomy Framework Design Decision; neither sentence any longer contradicts its own adjacent helpers enumeration
- [x] A-046 R17: `README.md` § Command Quick Reference → § Pipeline lists `/fab-dedupe [scope]`, restoring the table to a complete inventory of the Planning-group command surface; R17's sweep-class list now enumerates README.md so the gap cannot recur
- [x] A-047 R17: `docs/memory/pipeline/planning-skills.md`'s frontmatter `description:` is 432 runes (was 511), clearing the FKF §3.2 500-char cap — verified with `fab memory-index --check`, where the `511-character description:` warning is gone
- [x] A-048 R20/R16/R17: The other declared helper's enumerations are swept too — `src/kit/skills/_generation.md` and `docs/specs/skills/SPEC-_generation.md` read "six skills" and name `/fab-dedupe` (noting it reaches the Intake Generation Procedure via `_intake.md` Step 5, like its twin `/fab-draft`), and the `artifact mechanics` row names `fab-dedupe` in both twin-table copies. Verified against frontmatter ground truth: 6 skills declare `_generation`, 3 declare `_intake` (+`fab-proceed` by subagent dispatch = "four skills"), 8 declare `_srad` (7 enumerated; the `fab-adopt` omission is the documented pre-existing drift in § Non-Goals + Assumption 19)
- [x] A-049 R17: This change introduces **no new** `fab memory-index --check` warning — the full warning set is diffed against HEAD and is identical except the pre-existing `planning-skills.md` file-size warning ticking 395→396 lines (same warning kind, already over cap at HEAD). Both warnings cycle 1/cycle 2 introduced are cleared: the 511-char description (T033) and the 6-marker `narration-density` (T035, by rewriting three prose change-ids into FKF §3.3-sanctioned lone-paren citations)
- [x] A-050 Self-check — no self-contradicting sentence survives: a repo-wide grep for count words adjacent to helper-consumer enumerations returns only unrelated subjects (memory-index call consumers, config per-field metadata, batch scripts, default-branch chain). `all six declaring` / `remaining two declaring` / `six planning skills` return zero live hits. Append-only logs (`log.md`, `log.seed.md`), dated findings archives (`docs/specs/findings/`, `docs/findings/`), and `kit-architecture.md` are deliberately untouched — historical records stay as written
- [x] A-051 Canonical source only (rework cycle 2): no file under `.claude/skills/` was edited (Constitution V) — the only `src/kit/skills/` edits are `_srad.md` and `_generation.md`, each carrying its `SPEC-*.md` mirror in the same cycle

### Rework Cycle 3 — class-wide exhaustiveness

- [x] A-052 R20: The **artifact-marker placer** enumeration names `/fab-dedupe` in both copies — `src/kit/skills/_srad.md` § Artifact Markers and `docs/specs/srad.md`'s parallel row. Closes a live FUNCTIONAL gap: `/fab-dedupe` loads `_srad.md` and previously read that it places no `<!-- assumed: -->` markers, which would have left its own Step 5 Tentative divergence rows invisible to `/fab-clarify`'s scan
- [x] A-053 R20/R17: `docs/specs/srad.md`'s § Skill-Specific Autonomy Levels covering note reads "remaining three" and names `/fab-dedupe` as following the `/fab-new` column, matching its declared canonical source `_srad.md:73` (fixed in cycle 2). The 4 table columns + 3 covered skills again partition the 7-skill baseline with no overlap and no gap — verified arithmetically in both files
- [x] A-054 R20: The **score-persister** enumeration names `/fab-dedupe` in all SIX live homes of that one claim — `_preamble.md` § Confidence Scoring → Invocation, `docs/specs/srad.md` § Confidence Lifecycle, `docs/specs/templates.md`, `docs/memory/_shared/context-loading.md`, `docs/memory/memory-docs/templates.md`, and `src/kit/skills/_cli-fab.md` § Template — each stating the `_intake` Step 7 route and the per-cluster-group fan-out
- [x] A-055 R20/R16: No file contradicts itself across a Summary/flow-diagram boundary: `SPEC-_srad.md`'s diagram covering note and `SPEC-_generation.md`'s Intake-Generation consumer line now match their own Summary paragraphs. ASCII flow diagrams are established as live prose in the sweep class
- [x] A-056 R20/R16: The `_generation` consumer enumerations are coherent — `src/kit/skills/_generation.md`'s frontmatter `description:` names `fab-dedupe` (it previously omitted it while the body header two lines below read "six skills"), and `docs/memory/pipeline/planning-skills.md` § Shared Generation Partial names it in the Intake Generation Procedure consumer list (it previously contradicted the same file's `interactive` bullet and helper-declaration paragraph)
- [x] A-057 R20/R17: The **Activation-Preamble emitter** enumeration — a behavioral consumer set found by the class sweep rather than reported by review — names `/fab-dedupe` in `src/kit/skills/_preamble.md` § Activation Preamble and its memory-side restatement in `docs/memory/_shared/context-loading.md`, matching `fab-dedupe.md:231` which emits the preamble per drafted intake
- [x] A-058 R17: This change introduces **no new** `fab memory-index --check` warning — the full warning set was diffed against a detached `main` worktree: 27 warnings at HEAD, 27 now, the sole delta being the pre-existing `planning-skills.md` size warning ticking 395→396 lines (same warning kind, already over cap at HEAD). Every `4v91` citation in `docs/memory/**` is the FKF §3.3-sanctioned lone-paren `(4v91)` form (or an `*Introduced by*:` line), and all three touched memory `description:` fields are under the 500-rune cap (435 / 499 / 496)
- [x] A-059 R17/R20 Self-check — **class-wide, not site-list**: the enumeration greps were re-run after the final edit and every residual hit adjudicated. `remaining two` / `all six declaring` / `six planning skills` / `three consumers are thin` / `five skills` return **zero** live hits; every surviving count word (`remaining three`, `six skills`, `four consumers`, `all seven declaring`) names `/fab-dedupe` or quantifies an unrelated subject; the per-file intra-contradiction scan over all 19 live files naming `fab-dedupe` leaves only per-skill-scoped statements (`/fab-new` produces `intake.md`, idempotency rows), error strings, `/fab-proceed`-prefix lists, and historical `*Introduced by*`/`*Why*` narration — none of which are enumerations. Deliberately untouched: `docs/specs/srad-v1.md` (superseded), `kit-architecture.md` (broadly stale, own change), append-only logs, dated findings, `docs/memory/pipeline/index.md` (generated), `planning-skills.md:11`'s pre-existing narrow `(/fab-new, /fab-clarify)` overview list (already omitted `/fab-draft`/`/fab-ff`/`/fab-fff` at HEAD — unrelated axis, names no helper consumer set)
- [x] A-060 Canonical source only (rework cycle 3): no file under `.claude/skills/` was edited (Constitution V) — verified via `git status --short`. Every `src/kit/skills/` edit carries its SPEC treatment: `_srad.md`→`SPEC-_srad.md`, `_generation.md`→`SPEC-_generation.md`, `_preamble.md`/`_cli-fab.md`→ mirrors verified to document those sections at catalog/section granularity with no consumer name list, so no mirror edit was required (recorded rather than assumed)
- [x] A-061 R15: No `.go` file was touched this cycle; the scoped suites were re-run as a regression guard — `internal/configscope`, `internal/configupgrade`, `internal/config` all green, `cmd/fab` green except the two PRE-EXISTING macOS TMPDIR failures (`TestConfigShowOrigin_Provenance`, `TestConfigShowOrigin_HigherLayerScalarReplacesSubtree`) documented in Assumption 11

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `fab/backlog.md` `[ruaw]` (inventory layer) — the curated-inventory half is superseded by `/fab-dedupe`; the entry was correctly annotated rather than deleted because its process residue (reuse ledger, apply-entry priming, `reuse` acceptance category) is still unbuilt. No deletion recommended, but the residue should be re-scoped to a narrower entry once consumed.
- `fab/plans/sahil/26-07-22-reuse-awareness-codemap.md` (Part 1 section) — the standing plan document this change's Part 1 supersedes. Not touched by this change; it remains the `[cmap]` pickup detail for Part 2, so deleting Part 1 in isolation would orphan Part 2's context. Candidate for a trim, not a delete.
- Nothing else — the change adds a skill, one registry row, and one taxonomy entry. No existing function, branch, or config key was made redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Adding `consolidate.detectors` to `configref` REQUIRES a paired `"consolidate": ScopeProject` entry in `internal/configscope`'s `keyScopes` | Read directly from `configref.lintFields`: it cross-checks each row's Scope against `configscope.ScopeFor(topLevel(key))` and an unknown key yields `""`, which `validScope` rejects — `Fields()` would return an error and break every config command. The intake's "ONE registry key" constraint is about registry rows, not files touched | S:95 R:85 A:100 D:100 |
| 2 | Certain | The `configupgrade` golden fixtures do NOT churn on this change | Read `golden_test.go`: the full-document goldens render over `goldenFields()`, a small SYNTHETIC field set, chosen precisely so pinned bytes do not move when a real registry row is added or edited. The shipped-registry tests (`freeze_test.go`, `configupgrade_test.go`) assert idempotence/structure, not pinned bytes. Verified by running the suite | S:95 R:90 A:100 D:95 |
| 3 | Certain | `src/kit/skills/fab-help.md` and `docs/specs/skills/SPEC-fab-help.md` need no per-skill edit; the real sibling is `skillToGroupMap` in `fabhelp.go` | Read all three: `fab-help.md` delegates to `fab fab-help`, which discovers skills by scanning kit frontmatter (`scanSkills`), so a new skill appears automatically. Only the display grouping is hand-maintained, and skills.md § New Skill Checklist point 8 names exactly that file | S:90 R:85 A:100 D:95 |
| 4 | Certain | `src/kit/skills/_cli-fab.md` needs no update (intake assumption 17, verified at apply) | Verified: the change adds no `fab` subcommand and alters no command signature. The new config key is rendered by the existing `fab config reference` / `upgrade` / `init` commands, whose signatures are unchanged; `_cli-fab.md` documents signatures, not registry contents | S:90 R:80 A:95 D:90 |
| 5 | Certain | The two draft files are rewritten in place rather than preserved (intake assumption 15) | The intake states they are working material; they still carry the pre-rename `fab-consolidate` name, the invalidated flat-signature Step 3, and the dropped `consolidate.memory_file` override — three known defects | S:95 R:85 A:95 D:90 |
| 6 | Confident | `/fab-dedupe` is grouped under **Planning** in `skillToGroupMap` | It produces intakes (planning artifacts) like `/fab-draft`, but `/fab-draft` sits in "Start & Navigate". Chose Planning because the skill is analysis-then-draft rather than a navigation entry point, and it fans out to N changes rather than starting one. Reversible in one line | S:70 R:95 A:70 D:65 |
| 7 | Confident | Glossary terms land as *Cluster* / *Detector* / *Canonical home* under § Core Concepts, and `/fab-dedupe` under § Skills | The intake names the three terms; the glossary's groups are subject-based and these are core concepts of the skill's model rather than files, stages, or SRAD terms | S:80 R:90 A:85 D:75 |
| 8 | Confident | `docs/specs/config.md` gains `consolidate.detectors` in the § Scope taxonomy row and the § Advertise "not set live" enumeration, and no new section | Those are the two places the file enumerates per-key membership; the file is explicitly a metadata-table spec, so the key's per-field data is already covered by the registry it describes | S:80 R:90 A:85 D:80 |
| 9 | Confident | The `docs/specs/skills.md` `/fab-dedupe` section follows the `/fab-draft` section shape (Purpose / Context / Creates / Arguments / Examples / Behavior) | That is the house shape for a change-creating skill in that file, and `/fab-dedupe` is `/fab-draft` with a front end and a fan-out tail | S:85 R:90 A:90 D:85 |
| 10 | Confident | A new `configupgrade` test asserting the fence advertises `consolidate.detectors` is the right place for fence coverage of the new key | `configupgrade` is where the shipped registry is rendered into the managed fence, and the package already has shipped-registry tests (`fieldsForTest`); `configref` has no test file of its own and the intake directs coverage to the existing suites | S:80 R:85 A:90 D:80 |
| 11 | Certain | `cmd/fab`'s `TestConfigShowOrigin_Provenance` and `TestConfigShowOrigin_HigherLayerScalarReplacesSubtree` failures are PRE-EXISTING and unrelated to this change | Reproduced on a stashed clean tree before any Go edit. Cause is the macOS `/private/var` ↔ `/var` `TMPDIR` symlink: the tests compare an origin path against an un-resolved `t.TempDir()`. Every other test in the package passes, and both failures are byte-identical before and after this change. Not fixed here — out of scope, and Constitution VII forbids bending implementation to a test fixture | S:90 R:95 A:95 D:95 |
| 12 | Tentative | `[ruaw]` is annotated in place with a **SUBSTANTIALLY ABSORBED by 260728-4v91** lead rather than checked off or rewritten | The intake calls the disposition a judgment call (assumption 18). Annotating preserves the residue (reuse ledger, `reuse` acceptance category, apply-entry priming) that this change does not build, while making the absorption visible. Checking it off would lose the residue; deleting it would lose the history | S:60 R:85 A:55 D:55 |

| 13 | Certain | The `SPEC-_intake.md` `{questioning-mode}` parameter table gains a **fourth consumer column** rather than being restructured | The table's cells are single tokens (`interactive` / `promptless-defer`), so a fourth column costs ~15 characters of width and keeps the shape parallel to the header list. A transpose would read worse for a one-parameter table and would gratuitously diverge from the SPEC's existing house shape | S:90 R:95 A:95 D:90 |
| 14 | Certain | The real memory file is `docs/memory/pipeline/planning-skills.md`; the intake's `pipeline/skills-overview` names nothing | Enumerated `docs/memory/pipeline/`: the ten files are change-lifecycle, clarify, execution-skills, hooks-may-enhance-never-own, index, log, log.seed, planning-skills, preflight, schemas. `planning-skills.md` is the file carrying the `_intake` consumer set and the per-consumer binding table. Intake assumption 16 flagged the paths as unverified; this resolves it | S:100 R:90 A:100 D:100 |
| 15 | Confident | `/fab-dedupe`'s distinguishing property in the consumer enumerations is **fan-out** (Steps 0–9 run N times per invocation), and it is worth stating rather than just adding a name | All three prior consumers run the procedure exactly once. A reader of `_intake.md` who assumes one-invocation-one-intake would mis-model the re-entrancy of Step 3 (`fab change new` per cluster) and Step 9 (`advance` per intake). Naming it costs one clause | S:85 R:85 A:90 D:80 |
| 16 | Confident | Append-only change logs (`log.md`, `log.seed.md`) and dated finding archives (`docs/specs/findings/`) are excluded from the sweep class | Both are historical records of what a past change did — `log.seed.md`/`log.md` entries are explicitly per-change append-only, and the findings files are dated snapshots. Rewriting them would falsify the record. Only present-truth files are swept | S:85 R:80 A:90 D:85 |
| 17 | Confident | The `config_test.go` pruned-key enumeration stays a hardcoded list (with a drift-tracking comment) rather than being derived from `configscope.keyScopes` | `configscope` exports only `ScopeFor(key)` — no iterator. Adding an exported `Keys()`/`All()` purely for test convenience would widen a deliberately minimal leaf package's API. The real guard against an undeliberate key is `configscope_test.go`'s `TestScopeFor` taxonomy pin (T012); this test's job is the pruner's behavior, and the comment names the coupling | S:80 R:85 A:90 D:75 |
| 18 | Confident | The `/fab-dedupe:293` trailing `Next:` line is left as-is, declining the review's nice-to-have | The line is **byte-identical** to `fab-draft.md:76`, the house pattern for the Activation Preamble (`fab-new.md:162` and `fab-adopt.md:191` use the same runtime-derivation form). The literal-template rendering is deliberate — `_preamble.md` § Lookup Procedure requires the command list be derived at runtime, not hardcoded, and the clean line at :238 is a *sample output* block, not the skill's own terminal instruction. Changing :293 would break the twin and re-hardcode what d9rs deliberately de-hardcoded | S:85 R:90 A:85 D:80 |
| 19 | Tentative | The pre-existing `fab-adopt` gap in the `_srad`/helper-mapping enumerations is left unfixed, with the word "six" removed so the surviving prose is not actively false | Reproduced at HEAD: `fab-adopt.md` declares `_srad` but appears in none of the eight enumerations, and `context-loading.md`'s "All others (16 skills)" is wrong (18 of 27 user-facing skills declare no helpers). Fixing it properly means backfilling a `fab-adopt` row and recounting — a distinct pre-existing defect. The judgment call is whether a reviewer reads the partial sweep as an incomplete sweep; dropping the count word makes every touched sentence true, and § Non-Goals records the residue explicitly | S:60 R:80 A:65 D:50 |

| 20 | Certain | `/fab-dedupe` is covered by the `_srad` autonomy table's **fab-new** column, exactly like `fab-draft` — not given a fifth column | Both are thin call-sites over the same `_intake` Steps 0–9 with `{questioning-mode} = interactive`, so posture (SRAD-driven, 0 questions for clear input), interruption budget, and escape valve (`/fab-clarify`) are identical by construction. The fan-out tail multiplies *invocations* of that posture, not the posture itself. A fifth column would duplicate fab-new's cells verbatim; the covering note is the file's own established mechanism for exactly this case (c5tr) | S:95 R:90 A:95 D:90 |
| 21 | Certain | The `_srad` covering-note gap was a live FUNCTIONAL defect, not a stale count | `/fab-dedupe` declares `helpers: [_srad]`, so it loads `_srad.md` unconditionally before its body and reads the autonomy table. With only `fab-draft`/`fab-clarify` named, an agent running `/fab-dedupe` found no posture, interruption budget, or escape valve for itself anywhere in the file — an actual behavioral hole. This is why it graded must-fix rather than a documentation nit | S:100 R:85 A:95 D:100 |
| 22 | Certain | The phase-symmetry table is a **two-copy** twin class (`_shared/context-loading.md` + `memory-docs/templates.md`), and every row moves together | Both files carry the byte-parallel 4-row table. Cycle 1 updated one copy's `_intake` row only; cycle 2 found the other copy's `_intake` row stale AND both copies' `_generation` row stale — i.e. the twin class spans files *and* rows. Grepping the table's literal row label (`artifact mechanics`) enumerates the whole class in one pass | S:95 R:90 A:95 D:95 |
| 23 | Certain | `_generation.md`'s "five skills" was in scope for this change despite the reviewer not listing it | `/fab-dedupe` declares `_generation` in frontmatter, and `/fab-draft` — its exact structural twin (same `helpers:` triple, same transitive reach via `_intake` Step 5, no in-body `_generation` reference) — IS counted among the five. Consistency requires `/fab-dedupe` too; leaving it would have been the identical defect class a third cycle away. Ground-truthed against frontmatter: 6 skills declare `_generation` | S:90 R:90 A:95 D:85 |
| 24 | Certain | Trimming the description means dropping the `/fab-adopt` Intake-from-Diff/Plan-from-Diff parenthetical, not shortening the skill list | The skill list is the description's load-bearing content (it drives the generated domain-index row and is what a reader routes on); the `/fab-adopt` diff-variant detail is already documented at length in the file body and in `SPEC-_generation.md`. Dropping it yields 432 runes — comfortably under the 500 cap with headroom for a future consumer | S:90 R:95 A:95 D:85 |
| 25 | Certain | A change-id is FKF-sanctioned only as a lone-parenthesized `(id)` citation or on an `*Introduced by*:` line; an id inside a multi-token parens group counts as narration debt | Read directly from `internal/memoryindex`: `parenCitationPattern` is `\(\s*([^()\s]+)\s*\)` (no internal whitespace), and `introducedByLinePattern` matches `*Introduced by*:` only — notably NOT `*Updated by*:`. Confirmed empirically by per-line bisection against the shipped binary: neutralizing each `4v91` occurrence showed lines 111/284/338 each contributing one marker while the lone-paren `(4v91)` and the `*Updated by*:` line contributed none | S:100 R:90 A:95 D:100 |
| 26 | Confident | The two cycle-1 prose change-ids on `planning-skills.md` are rewritten rather than left, even though neither is cycle 2's own edit | The `narration-density` warning is a per-FILE threshold (≥5 markers), so it does not clear unless the file drops below it — fixing only cycle 2's own marker would leave a warning that is new relative to HEAD and attributable to this change. The rewrites are pure citation-form changes (prose id → sanctioned lone-paren) preserving every factual claim and change-id | S:85 R:90 A:90 D:80 |
| 27 | Confident | `docs/findings/intake-is-the-context-boundary.md`'s stale symmetry table is left untouched | It is a dated findings artifact under a findings tree — the same historical-record class as `docs/specs/findings/` and the append-only logs, which § Non-Goals and Assumption 16 already exclude from the sweep. It records what was true when the finding was written; rewriting it would falsify the record. It is also outside `docs/memory/`, so no FKF check reads it | S:85 R:85 A:85 D:80 |

| 28 | Certain | The sweep unit is the **class of enumerations**, not a reported site list — R17/R20 are rewritten to bind enumerate-the-class-then-verify | Three cycles each fixed exactly the sites review named and shipped; each time the identical defect survived one hop out (cycle 1 name lists → cycle 2 count words → cycle 3 a byte-parallel spec copy). The reviewer's diagnosis is mechanical and correct: a file-tracking sweep cannot terminate, because the defect is a property of the *construct class*, not of any file. Fixing cycle 3's 5 reported sites alone would have failed a 4th time — the class sweep found 5 further members those 5 did not include | S:100 R:85 A:95 D:100 |
| 29 | Certain | ASCII **flow diagrams** are live prose inside the sweep class | `SPEC-_srad.md` and `SPEC-_generation.md` each carried a stale consumer list *inside a diagram* while that same file's Summary paragraph was already correct — a self-contradiction no prose-only grep would surface. A diagram line is read by agents exactly as prose is, so it drifts identically and must be swept identically | S:95 R:90 A:95 D:95 |
| 30 | Certain | The `_srad` § Artifact Markers omission was a live FUNCTIONAL defect, not a stale count | `/fab-dedupe` declares `helpers: [_srad]`, so it loads `_srad.md` unconditionally before its body and reads the marker table. Reading that it does not place `<!-- assumed: -->` markers would leave the Tentative divergence rows its own Step 5 produces unmarked — and `/fab-clarify` scans for exactly that marker, so those assumptions would be permanently invisible to the escape valve. Same defect kind as Assumption 21's covering-note hole | S:100 R:85 A:95 D:100 |
| 31 | Certain | One canonical sentence commonly has **five or six** live homes, so the sweep greps the *claim* and not the phrase | The § Invocation score-persister claim exists in `_preamble.md`, `docs/specs/srad.md`, `docs/specs/templates.md`, `docs/memory/_shared/context-loading.md`, `docs/memory/memory-docs/templates.md`, and `_cli-fab.md` — each in different wording ("persist the intake score" / "Computation" / "Computed by" / "who scores, when" / "writes the intake score"), so no single literal grep finds them all. Phrase-grepping is what let each cycle miss the next hop | S:95 R:85 A:90 D:95 |
| 32 | Certain | The **Activation-Preamble emitter** set is a consumer-set enumeration in the class, and `/fab-dedupe` belongs to it | `fab-dedupe.md:231` instructs "Then the Activation Preamble `Next:` line (`_preamble.md` § Activation Preamble)" per drafted intake, and R11 requires it. The § Activation Preamble sentence enumerates *which skills emit it* — a behavioral "who does X" list, exactly the construct R20 now defines. Found by the class sweep, reported by no review cycle: evidence the class method reaches further than the site lists | S:95 R:90 A:95 D:95 |
| 33 | Confident | Membership is decided **behaviorally** ("does `/fab-dedupe` do this?"), which is what correctly EXCLUDES several near-miss enumerations | Adjudicated and deliberately left alone: `planning-skills.md:380`'s ID-collision set (`/fab-dedupe` Step 0 never takes a backlog/Linear ID, per R9 and its own call-site row), `execution-skills.md:51`'s two-skill change-creation inventory (it enumerates `/fab-proceed`'s prefix-step candidates by activation behavior and quotes their frontmatter verbatim), and the 4-column autonomy table headers (the covering note is the file's own mechanism for non-column skills). A name-matching sweep would have wrongly edited all three | S:85 R:85 A:90 D:80 |
| 34 | Confident | `docs/memory/pipeline/planning-skills.md:11`'s narrow `(/fab-new, /fab-clarify)` overview list is left unfixed | Reproduced at HEAD, where it already omits `/fab-draft`, `/fab-ff`, and `/fab-fff` while the same file's frontmatter `description:` lists six skills — pre-existing drift on an axis unrelated to `/fab-dedupe`, and it enumerates no helper-consumer set. Same disposition as the `fab-adopt` residue (§ Non-Goals, Assumption 19): fix root causes, not adjacent symptoms opportunistically. The judgment call is whether a reviewer reads it as part of this sweep; it is not, because adding `/fab-dedupe` alone would leave the sentence just as wrong | S:75 R:85 A:80 D:70 |
| 35 | Confident | `_preamble.md`/`_cli-fab.md` needed **no** SPEC-mirror edit, and that verification is recorded rather than assumed | Read both mirrors directly: `SPEC-_preamble.md` documents § Confidence Scoring at section granularity ("gate threshold + invocation only") and its bookkeeping row is skill-agnostic ("After intake generation / clarify"); it carries no § Activation Preamble enumeration at all. `SPEC-_cli-fab.md:20` describes the `fab score (extended)` section as a catalog entry ("status-template details") with no persister list. Neither mirror restates the edited claim, so the constitution's sync rule is satisfied without an edit — the failure mode this records against is *assuming* that and being wrong (T002/T019 set the precedent of recording such verifications) | S:85 R:90 A:90 D:85 |

35 assumptions (19 certain, 14 confident, 2 tentative).
