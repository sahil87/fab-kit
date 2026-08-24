---
type: memory
description: "`/fab-dedupe` — sweeps a scoped area for duplicated utilities, clusters them by behavioral shape into a shared core plus opt-in layers, drafts one intake per accepted cluster group. Read-only until acceptance (a conversational reply). `consolidate.detectors` only SEED the sweep: probe-then-fail-silent, non-zero exit is a finding not a STOP. Thin `_intake` call-site (Steps 0–9, interactive) with a fan-out tail; the `_shared/utilities.md` home is hardcoded, hydrate-written."
---
# Dedupe Skill

**Domain**: pipeline

## Overview

`/fab-dedupe [scope]` finds code that should have been a shared utility and wasn't: it sweeps a scoped area for duplicate and near-duplicate functions, groups them into **clusters**, decomposes each into a shared core plus opt-in layers, proposes a canonical home, and drafts one change intake per accepted cluster group. It does **not** refactor — the drafted changes run the normal pipeline, with review in the loop (4v91).

Structurally it is `/fab-draft` with a cluster-analysis front end and a **fan-out tail** — a thin `_intake` call-site running Steps 0–9 once per accepted cluster group rather than once per invocation ([planning-skills.md](/pipeline/planning-skills.md) § The `_intake` Shared Create-Intake Procedure).

The boundary with `/code-reorg` ([code-reorg](/pipeline/code-reorg.md)) splits content from structure: `/code-reorg` judges where files live and what they are called — placement, naming, consolidation — and never clusters content duplication; duplication smells it encounters surface in its report's `For /fab-dedupe` section as pointers into this skill, which owns semantic-duplication judgment.

## Requirements

### Requirement: Identity, pre-flight, and read-only-until-accept

The canonical source is `src/kit/skills/fab-dedupe.md`, frontmatter `name: fab-dedupe` with `helpers: [_generation, _srad, _intake]` — the triple `/fab-draft` declares — and one optional `[scope]` argument. `fab fab-help` groups it under **Planning** via `fab_help.go`'s `skillToGroupMap`.

Pre-flight verifies `config.yaml` and `constitution.md`, STOPping with the standard uninitialized message when either is missing. It MUST NOT run `fab preflight` — the skill operates with no active change and must not resolve or disturb one. Logging is `fab log command "fab-dedupe"`, no change ID.

Everything before Step 5 SHALL be read-only: the skill SHALL NOT refactor code, write memory files, activate a change, create a git branch, or modify `.fab-status.yaml`. Its only writes are the intakes the Create-Intake Procedure produces for accepted clusters. Accepting nothing is a valid, complete outcome — the report is the deliverable, nothing is written, no `Next:` line is emitted.

### Requirement: Scope filters where clusters are FOUND, not where members LIVE

Step 1 resolves `[scope]` to a concrete path set: an existing path or glob directly; natural language against `source_paths`/`test_paths`, confirmed with the user first; a bare invocation defaults to `source_paths` with a warning that a full-repo sweep produces a long cluster list. The resolved set and file count are echoed before sweeping. The sweep is intra-repo only — duplication across separate repos is out of scope.

A cluster seeded inside the scope MAY include members outside it, and those members MUST be reported and flagged as out-of-scope — consolidating half a cluster is worse than not consolidating it.

### Requirement: Detectors seed the sweep — probe-then-fail-silent, non-zero-exit tolerant

Step 2 reads `consolidate.detectors` from project config, defaulting to jscpd alone when absent ([_shared/configuration.md](/_shared/configuration.md) § `consolidate`). Each entry is a shell command template with `{paths}` (resolved scope paths, each shell-quoted, space-joined) and `{out}` (a scratch dir created for the run, shell-quoted) substituted before execution — quoting is required, so a path carrying a space or a shell metacharacter reaches the detector as one intact argument instead of splitting or injecting into the command. Two rules bind:

- **Probe then fail silent.** `command -v <bin> >/dev/null 2>&1` runs before invoking each detector; a missing binary is skipped **silently** — never an error, never a warning — mirroring the `rk` discipline in `_preamble.md` § Run-Kit Reference. Which detectors ran is reported (ran / skipped-missing / ran-with-exit-code-N), so a skip reads as information, not failure.
- **A non-zero detector exit is a finding, not a STOP.** Duplication tools conventionally exit non-zero for "threshold exceeded". The skill parses whatever output exists, notes the exit code, and continues. This is an **explicit per-skill exception** to `_preamble.md` § Common fab Commands' failure rule, which governs `fab` commands rather than third-party tools — the skill says so in those words.

Detector output is a **seed set** only — a hint where to look, never the cluster list.

#### Scenario: Detector absent, then present and exiting non-zero
- **GIVEN** `consolidate.detectors` absent and jscpd not installed
- **THEN** the detector is skipped with no error or warning, the skip is recorded, and the sweep continues unaided
- **AND GIVEN** jscpd installed and exiting 1 on a threshold breach
- **THEN** the skill parses the output, notes exit code 1, and continues to Step 3

### Requirement: Clusters are layered — a shared core plus opt-in variation layers

Step 3 clusters by **behavioral shape**: it reads function bodies in scope — seeded by but not limited to detector output — and groups members that solve the same problem regardless of name, signature, or implementation. Per cluster it records members (`file:line` + current name), the shared behavior in one line, **divergences** (the field carrying the risk), the proposed canonical home (per the repo's own layout conventions, read from neighboring directories), the call-site count, out-of-scope members, and a **layered decomposition**: the **base** behavior every member needs with its member list; each named **opt-in layer** with exactly which members require it; and the **unified API** expressing that shape — functional options, an options struct, a base helper plus thin wrappers, or an explicit "these two should stay separate" when no single API is honest. A flat per-cluster "proposed signature" is forbidden.

Two rules bind the grouping: **do not cluster on name similarity alone** (two `parseConfig` functions in different packages may be unrelated; `mustTempRepo` and `newFixtureDir` may be identical — read the bodies), and **one member is not a cluster**.

### Requirement: Ranking counts base-layer-only members as LOW divergence

Step 4 ranks by consolidation value — high call-site count and low divergence first — treating "this member needs only the base layer" as **low** divergence even when its body reads differently: textual difference is not behavioral divergence, and the layer profile is what ranks. The read-only report shows the resolved scope, the detector run/skip/exit-code record, the scanned counts, and per cluster its member and call-site counts, divergence rating, member locations, canonical home + unified API, and base/layers breakdown. Acceptance is a **plain conversational question** — reply grammar `all` / a comma-separated list (`1,3`) / `none`, never a structured picker, so the cluster count is uncapped.

#### Scenario: Textually divergent bodies, shared base
- **GIVEN** a 12-member cluster whose bodies differ textually but where 8 need only the base layer
- **THEN** it rates low divergence and outranks a 3-member cluster with genuinely divergent semantics

### Requirement: Step 5 binds the shared Create-Intake Procedure and stops at `ready`

Per accepted cluster group, Step 5 reads `.claude/skills/_intake/SKILL.md` and runs the **Create-Intake Procedure (Steps 0–9)** with `{questioning-mode} = interactive`, then **STOPs after Step 9** — no activation, no git branch. Steps 10–11 live only in `fab-new.md`'s tail, which this skill never reads, so there is no momentum hazard. Grouping is by refactor coherence, not count — clusters sharing a canonical home in one change, unrelated ones separate — with a stated preference for **one intake per cluster** (bundling makes apply all-or-nothing and forces one review verdict over independent refactors). The per-step binding:

- **Step 0 (parse input)** — natural language, `consolidate {shared behavior} into {canonical home}`. No backlog or Linear ID, so no collision check ever fires and each invocation creates a fresh change.
- **Step 2 (gap analysis)** — the duplicate-work guard: check `_shared/utilities.md` and in-flight changes per cluster; already consolidated or in flight → report and skip that cluster.
- **Step 5 (generate `intake.md`)** — Origin ← scope + detector commands + exit codes (falsifiable: a reader can re-run the sweep); Why ← member/call-site counts and the cost of leaving it; What Changes ← canonical home, layered API, per-member migration notes; Impact ← every call site plus out-of-scope members; Affected Memory ← `_shared/utilities`, `(new)` then `(modify)`; Open Questions ← unresolved divergences.
- **Step 7 (confidence)** — persists the intake score via `fab score --stage intake`, per drafted intake.
- **Step 8 (SRAD)** — identical members + obvious home → Certain; minor divergence with a clear unified API → Confident; divergence that might drop a behavior → Tentative; cannot tell whether members are the same → Unresolved, and ask (the Critical Rule applies here). A mostly-Tentative cluster is left to fail the gate, never talked up.

#### Scenario: Two clusters accepted
- **GIVEN** the user replies `1,3`
- **THEN** two intakes are created at `ready`, with no `.fab-status.yaml` symlink and no git branch, each naming `_shared/utilities` in `## Affected Memory`

### Requirement: Output and error handling

Per drafted intake the skill reports name and confidence, then emits the Activation Preamble `Next:` line (`_preamble.md` § Activation Preamble; the post-switch command list derived at runtime per § Lookup Procedure, never hardcoded). With no clusters accepted it ends with the report and **no** `Next:` line.

Error handling: config/constitution missing, or a scope resolving to no files → STOP (the latter showing the resolved set); all detectors missing → continue unaided, noted in the report; a detector exiting non-zero or emitting unparseable output → continue (the latter as no seed); no clusters found, or none accepted → report and stop. A `fab change new` failure for one cluster is reported, the remaining accepted clusters still drafted, and the failures listed at the end.

### Requirement: The utilities memory home is hardcoded and written by hydrate

Consolidated utilities are recorded in **`docs/memory/_shared/utilities.md`** — fixed in the skill prose, with **no** `consolidate.memory_file` override. The skill never writes the file: it appears in each drafted intake's `## Affected Memory` and hydrate writes it on the normal pipeline path. It comes into existence the first time a real `/fab-dedupe`-drafted consolidation is hydrated, carrying a coverage header naming what has and has not been swept (`Coverage: swept {paths} {date} · not yet swept: {paths}`).

### Requirement: Re-run contract

The sweep is idempotent — it writes nothing, and re-reading a scope yields the same clusters. Re-running **after** accepting clusters creates *new* changes rather than resuming (Step 0 carries no ID, so no collision check); Step 2's gap analysis is the guard.

## Design Decisions

### Detectors seed; the agent clusters
**Decision**: Duplicate-detection CLIs are an optional, fail-silent *seed*; the agent's reading of function bodies is the load-bearing clustering mechanism. `consolidate.detectors` defaults to jscpd alone.
**Why**: No shipping CLI performs true Type-4 (semantically-equivalent) clone detection — claims to the contrary resolve to unmaintained research artifacts — and a dry run on `src/go` confirmed a token-based detector would have contributed nothing on the target cluster (its common run is ~4 lines split by divergent code). jscpd is the only universal option (Type-1/2); `similarity-ts` is the strongest Type-3 tool (TypeScript, no JSON output); Go's best tier is Type-1/2, so Go sweeps lean harder on the agent. Optional detectors also make the skill polyglot on day one and correct in a repo with nothing installed. The per-language suggestion table lives in the skill.
**Rejected**: Requiring a detector — it gates the skill on per-language tooling and still misses the Type-3/4 clusters worth consolidating. Treating a non-zero exit as a STOP — it means "threshold exceeded", the finding the sweep wants.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill

### Clusters decompose into a shared core plus opt-in variation layers
**Decision**: Step 3 records a base behavior every member needs plus named opt-in layers with their member lists, and Step 4 counts "needs only the base layer" as LOW divergence.
**Why**: A dry run on four `src/go` fixtures proved the layered shape (tempdir + `fab/changes` for all, then +project config, +change dir, +active symlink, +kit override for narrowing subsets), and `resolve_test.go` had already factored `createChange` out — the right decomposition arrived at independently. A single flat signature per cluster forces every member's needs into one parameter list and produces a bad API, while making base-only members look artificially divergent.
**Rejected**: A flat "proposed signature" field per cluster — it distorts layered clusters and mis-ranks members whose bodies differ only by which layers they opt into.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill

### The skill drafts intakes; the pipeline refactors
**Decision**: `/fab-dedupe` is a thin `_intake` call-site (Steps 0–9, interactive) with a cluster-analysis front end and a fan-out tail, stopping at intake `ready`. Cluster acceptance is a conversational reply, not a picker.
**Why**: Collapsing overlapping helpers is design work with real semantic risk, and the pipeline already has a gated stage for it. The intake's `## Assumptions` SRAD table is a natural per-cluster risk record, and `fab score`'s flat 3.0 gate already refuses to auto-run a change whose divergences grade Tentative. A conversational reply matches every other fab skill and takes an arbitrary cluster count.
**Rejected**: A skill with its own refactor loop — it bypasses the intake gate, `code-quality.md`, and the reviewer, which exist for exactly this class of change. A bespoke report artifact — it reinvents the gate. A structured picker — it caps at four options, which sweeps routinely exceed.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill

### The utilities memory home is hardcoded, and lives in `_shared`
**Decision**: `docs/memory/_shared/utilities.md` is fixed in the skill prose. There is no `consolidate.memory_file` config key, and hydrate — not the skill — writes the file.
**Why**: `_shared` is defined as cross-cutting concerns spanning all domains, which is what a utility index is. A config key would double the registry surface for a knob with no known consumer, and adding one later is cheap and non-breaking. Hydrate-writes means the record cannot drift from what shipped, and the coverage header keeps an under-covering index honest — a consumer trusts it identically at 10% and 90% coverage.
**Rejected**: A top-level `utilities` domain — every existing domain is organized by subject matter, whereas this would be organized by code role and cut across all of them; promotion defers to `/docs-reorg-memory` if the file outgrows itself. A `consolidate.memory_file` key. A skill-owned write path.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill
