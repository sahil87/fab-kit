---
type: memory
description: "`/code-dedupe` — read-only duplication review: sweeps a scoped area for duplicated utilities, clusters them by behavioral shape into a shared core plus opt-in layers, and presents a ranked consolidation report with SRAD-graded confidence per cluster. Suggestions only — refactors nothing, drafts nothing; `consolidate.detectors` only SEED the sweep (probe-then-fail-silent, non-zero exit is a finding not a STOP); the `_shared/utilities.md` home is hardcoded, hydrate-written."
---
# Code-Dedupe Skill

**Domain**: pipeline

## Overview

`/code-dedupe [scope]` finds code that should have been a shared utility and wasn't: it sweeps a scoped area for duplicate and near-duplicate functions, groups them into **clusters**, decomposes each into a shared core plus opt-in layers, proposes a canonical home, and presents a ranked, evidence-backed consolidation report. The report is the skill's terminal output and entire effect: it refactors nothing, drafts nothing, and never decides how a cluster gets consolidated — that routing (micro change vs `/fab-new` vs ignore) is the user's per-cluster choice, optionally guided by informational suggested-next-action lines the skill never executes.

It is the content-duplication sibling of `/code-reorg` ([code-reorg](/pipeline/code-reorg.md)) — same analyze-and-report posture, same read-only contract. The boundary splits content from structure: `/code-reorg` judges where files live and what they are called and points duplication smells here (its report's `For /code-dedupe` section); `/code-dedupe` owns semantic-duplication judgment and points structural smells back (its report's `For /code-reorg` section).

## Requirements

### Requirement: Identity, pre-flight, and full read-only

The canonical source is `src/kit/skills/code-dedupe.md`, frontmatter `name: code-dedupe` with `helpers: [_srad]` (SRAD grades cluster confidence — no `_intake`/`_generation`) and one optional `[scope]` argument. `fab fab-help` groups it under **Maintenance** via `fab_help.go`'s `skillToGroupMap`, alongside `code-reorg` and the `docs-reorg-*` analysis skills.

Pre-flight verifies `config.yaml` and `constitution.md`, STOPping with the standard uninitialized message when either is missing. It MUST NOT run `fab preflight` — the skill operates with no active change and must not resolve or disturb one. Logging is `fab log command "code-dedupe"`, no change ID.

The skill SHALL be fully read-only: it modifies no files, refactors no code, creates no changes, activates nothing, creates no git state, and emits no `Next:` line (a documented opt-out per `_preamble.md` § Next Steps Convention). A sweep finding nothing closes with `no consolidation candidates in {scope}` — a success, not a failure.

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

The **already-done guard** checks each cluster against `docs/memory/_shared/utilities.md` (when present) and in-flight changes (`fab/changes/`): a cluster already consolidated, or with a change in flight for it, is dropped from the proposals and noted in the report.

### Requirement: Ranking counts base-layer-only members as LOW divergence; the report is terminal

Step 4 ranks by consolidation value — high call-site count and low divergence first — treating "this member needs only the base layer" as **low** divergence even when its body reads differently: textual difference is not behavioral divergence, and the layer profile is what ranks. Each cluster carries an SRAD-graded confidence (identical members + obvious home → Certain; minor divergence with a clear unified API → Confident; divergence that might drop a behavior → Tentative; cannot tell whether members are the same → Unresolved), with per-dimension scores recorded alongside the grade — a mostly-Tentative cluster is reported as not worth consolidating wholesale, never talked up.

The report shows the resolved scope, the detector run/skip/exit-code record, the scanned counts, and per cluster its member and call-site counts, divergence rating, member locations, canonical home + unified API, base/layers breakdown, confidence, and an optional informational suggested-next-action line (e.g. a ready-to-paste `/fab-new consolidate {shared behavior} into {canonical home}`, which SHOULD name `_shared/utilities` for the change's Affected Memory). Structural/placement smells appear only in a separate `For /code-reorg` section, never as clusters. Presenting the report ends the skill.

#### Scenario: Textually divergent bodies, shared base
- **GIVEN** a 12-member cluster whose bodies differ textually but where 8 need only the base layer
- **THEN** it rates low divergence and outranks a 3-member cluster with genuinely divergent semantics

### Requirement: Error handling

Config/constitution missing, or a scope resolving to no files → STOP (the latter showing the resolved set); all detectors missing → continue unaided, noted in the report; a detector exiting non-zero or emitting unparseable output → continue (the latter as no seed); no clusters found → the `no consolidation candidates` success close.

### Requirement: The utilities memory home is hardcoded and written by hydrate

Consolidated utilities are recorded in **`docs/memory/_shared/utilities.md`** — fixed in the skill prose, with **no** `consolidate.memory_file` override. The skill only reads the file (the already-done guard): it is written by hydrate on the normal pipeline path when a consolidation change the user routed from the report ships, carrying a coverage header naming what has and has not been swept (`Coverage: swept {paths} {date} · not yet swept: {paths}`).

### Requirement: Re-run contract

The sweep is idempotent — it writes nothing, and re-reading the same scope over the same code yields the same clusters. The already-done guard keeps repeat sweeps quiet about work that has since shipped or entered flight.

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

### Report-only: the skill suggests; the user routes
**Decision**: The skill presents a ranked report and stops — the `code-reorg` posture. It drafts no intakes: consolidation routing (micro change vs `/fab-new` vs ignore) is the user's per-cluster choice, guided by informational suggested-next-action lines. The name is `code-dedupe`, grouped under Maintenance with the other analysis skills.
**Why**: The original drafted-intake handoff (shipped 2026-07-28 as `/fab-dedupe`) forced routing decisions the skill shouldn't own — a two-member trivial cluster is a micro change that must skip fab entirely — coupled the skill to the `_intake`/`_generation` machinery, and its funnel produced zero drafted intakes in four weeks of use. `/code-reorg` shipped report-only for exactly these reasons (its review panel rejected borrowing the dedupe handoff), and the pair reads as siblings: one judges structure, one judges content duplication, both stop at suggestions. The pipeline's gates still apply to any consolidation the user routes through `/fab-new` — nothing is lost but the unused fan-out.
**Rejected**: Keeping the fan-out `_intake` call-site — unexercised, and it made the skill the only `fab-*`-named surface operating on code rather than the pipeline. A skill with its own refactor loop — it bypasses the intake gate, `code-quality.md`, and the reviewer. A structured accept-picker — moot once nothing is accepted in-skill.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill; *Updated by*: direct change 2026-08-24 (renamed `/fab-dedupe` → `/code-dedupe`; intake drafting removed in favor of the report-only contract)

### The utilities memory home is hardcoded, and lives in `_shared`
**Decision**: `docs/memory/_shared/utilities.md` is fixed in the skill prose. There is no `consolidate.memory_file` config key, and hydrate — not the skill — writes the file.
**Why**: `_shared` is defined as cross-cutting concerns spanning all domains, which is what a utility index is. A config key would double the registry surface for a knob with no known consumer, and adding one later is cheap and non-breaking. Hydrate-writes means the record cannot drift from what shipped, and the coverage header keeps an under-covering index honest — a consumer trusts it identically at 10% and 90% coverage.
**Rejected**: A top-level `utilities` domain — every existing domain is organized by subject matter, whereas this would be organized by code role and cut across all of them; promotion defers to `/docs-reorg-memory` if the file outgrows itself. A `consolidate.memory_file` key. A skill-owned write path.
*Introduced by*: 260728-4v91-add-fab-dedupe-skill
