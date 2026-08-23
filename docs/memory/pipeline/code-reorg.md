---
type: memory
description: "`/code-reorg [<path>]` — read-only source-tree structure review: sweeps a scoped path (default: combined `source_paths`) for prediction failures (junk drawers, singleton folders, sibling naming inconsistency, co-change/import-direction misplacement) and presents a ranked report of move/rename/merge/split proposals. Suggestions only — applies, drafts, and routes nothing; docs/memory + docs/specs refused with docs-reorg-* pointers; duplication smells point to /fab-dedupe."
---
# Code-Reorg Skill

**Domain**: pipeline

## Overview

`/code-reorg [<path>]` treats folder structure as a **prediction interface** — a structure is good when a reader can predict where a thing lives from its name and what a file contains from its path. It sweeps a scoped source tree for concrete prediction failures and presents a ranked report of evidence-backed proposals (moves, renames, folder merges/splits), each with rationale, blast radius, and an SRAD-graded confidence. The report is the skill's terminal output and entire effect: it applies nothing, drafts nothing, and never decides how a finding gets fixed. It is the source-tree sibling of `/docs-reorg-memory` / `/docs-reorg-specs` (same analyze-and-propose posture, source-tree scope) and the structural complement of `/fab-dedupe` ([dedupe](/pipeline/dedupe.md)), which judges content duplication.

## Requirements

### Requirement: Identity and the report-only contract

The canonical source is `src/kit/skills/code-reorg.md`, frontmatter `name: code-reorg` with `helpers: [_srad]` and one optional `[<path>]` argument. `fab fab-help` groups it under **Maintenance** via `fabhelp.go`'s `skillToGroupMap` — alongside the `docs-reorg-*` analysis skills, not under Planning where `/fab-dedupe` sits (dedupe drafts intakes; code-reorg drafts nothing). It does not join fabhelp's hardcoded TYPICAL FLOW "Maintain docs:" line, which enumerates docs-tree commands.

The skill is fully read-only: it modifies no files, creates no changes, runs no `fab status` transition, and creates no git state. Pre-flight verifies `config.yaml` and `constitution.md` exist (STOPping with the standard uninitialized message otherwise) and MUST NOT run `fab preflight` — the skill operates with no active change and must not resolve or disturb one. Routing each finding (micro change vs `/fab-new` vs ignore) is the user's per-finding choice; each proposal MAY carry an informational suggested-next-action line (e.g. `micro: rename directly and commit` or a ready-to-paste `/fab-new <description>`), which the skill never executes. The skill emits no `Next:` pipeline line — a documented opt-out per `_preamble.md` § Next Steps Convention, like `/fab-discuss`. A clean tree closes with `no proposals — structure predicts well`; that is a success, not a failure.

### Requirement: Scope resolution with a path-based docs carve-out

With no argument the skill sweeps all `source_paths` from `fab/project/config.yaml` as one combined scope; an argument resolves relative to the repo root and must exist inside the repo. The resolved scope and file count are echoed before analysis — a silently mis-resolved scope wastes the run. The carve-out is **path-based, not content-based**: a resolved path inside `docs/memory/` or `docs/specs/` is refused with a pointer to `/docs-reorg-memory` / `/docs-reorg-specs`; a scope that contains either tree (e.g. `.` or `docs/`) is accepted with those two subtrees pruned from the swept file set, the pruning recorded in the scope echo. Those two trees are the only exclusions — markdown inside a source tree is just source, judged by the same placement/naming frames. Errors: a nonexistent or outside-repo path STOPs showing the resolved attempt; missing/empty `source_paths` with no argument STOPs naming the config key.

#### Scenario: Docs-tree path refused
- **GIVEN** the argument `docs/memory/pipeline`
- **WHEN** the skill starts
- **THEN** it refuses with the `/docs-reorg-memory` pointer and performs no analysis

### Requirement: Four signal families, co-change behind mandatory noise controls

Step 2 gathers mechanical, read-only signals:

- **Tree shape** — depth, fan-out, singleton folders, oversized flat dirs, junk drawers (`utils/`, `helpers/`, `misc/` with unrelated contents). Junk-drawer and oversized-dir criteria are judgment guidance, not numeric thresholds — the evidence is named (e.g. "6 of 8 files share no import or topic relation").
- **Sibling naming** — casing, plural/singular, and stutter inconsistencies among siblings (e.g. `pane/pane_send.go` in a `pane/` package).
- **Static import-direction** — a file whose imports and importers concentrate in a different folder than its own is placement evidence; the deterministic complement to co-change, which only proxies coupling behaviorally.
- **Co-change coupling** — from `git log`, behind the full noise-control set: a history window (default 12 months), bulk-commit exclusion (default: commits touching > 20 files), rename following (`-M`), a **mandated-coupling carve-out** derived at run time from the constitution and code-quality files (pairs whose co-change is required by project rules are expected coupling, not misplacement evidence), and a **cross-layer whitelist** (tests, docs, and specs legitimately co-change with the source they cover; colocating across layers against ecosystem convention is never proposed). The carve-out and whitelist are applied before interpreting any pair, and results are read as a **fraction** of the commits touching either file, not a raw count.

The skill file ships a worked co-change command — the pairwise aggregation is the one signal too fiddly to improvise per run. The command pipes `git log --since="12 months ago" -M --format='commit %H' --name-only -- <paths>` into an awk pass that accumulates each commit's file list and flushes pairwise combinations per commit — the final commit included, via an `END{flush()}` block — then `sort | uniq -c | sort -rn`; commits over the 20-file cap are excluded inside `flush()`. Reference density (what links/imports into a file) is **not** a detection signal — it is computed lazily in Step 4, for clustered proposals only, as the blast-radius input. Shallow or absent git history skips the co-change signal with an explicit note in the report; tree-shape, naming, and import-direction signals still run.

### Requirement: Frame evaluation behind a taste guard

Step 3 evaluates candidate findings against frames in priority order: (a) the project's stated conventions (`fab/project/context.md`'s documented repo structure), (b) ecosystem convention (e.g. Go project layout), (c) internal sibling consistency. Every finding MUST name the Step-2 signal or named frame violation it cites. The **taste guard** is load-bearing: a finding with no cited signal and no named frame violation is dropped — "I would have organized it differently" does not qualify. A finding citing *only* frame (c) additionally requires a quantified sibling majority (e.g. "7 of 9 siblings use pattern X") — sibling consistency is otherwise the loophole that re-admits taste.

### Requirement: Proposal schema with blast radius

Step 4 clusters related findings into coherent units — one proposal = "rename these 4 files for sibling consistency", not four proposals. Each proposal carries: the **prediction failure** it fixes with its cited signal/frame; the concrete **move/rename list** (`from → to`); the **blast radius** — breaking references (imports, doc cross-links; reference density computed now, for these proposals only) plus **in-flight exposure** (open branches and active fab changes touching the affected files); and an **SRAD-graded confidence** with the per-dimension scores recorded alongside the grade. Folder merges/renames in package-scoped languages (e.g. Go package directories, where import paths change and identifier collisions become possible) carry a **mandatory elevated blast grade** — such moves are never content-neutral.

### Requirement: The ranked report is the terminal output

Step 5 ranks proposals by highest confidence × lowest blast radius, presents, and stops. Content-duplication smells appear **only** in a separate `For /fab-dedupe` section — never as proposals. A co-change-skipped run states the skip note in the report. The error table enumerates: missing config/constitution (STOP), nonexistent/outside-repo path (STOP with the resolved attempt), missing/empty `source_paths` (STOP naming the config key), docs-tree path (refusal with the sibling-skill pointer), shallow history (co-change skipped, noted, other signals run), and no findings surviving the taste guard (the `no proposals` success close).

## Design Decisions

### Report-only: the skill finds and presents, never routes
**Decision**: `/code-reorg` ends at its ranked report; it applies nothing, drafts no intakes, and never decides whether a fix is a micro change or a pipeline change.
**Why**: Routing is the user's per-finding call — a one-rename proposal is a micro change that must skip fab, a cross-cutting move deserves a pipeline change, and the skill cannot know which without owning policy it shouldn't. Decoupling also drops the `_intake`/`_generation` dependency entirely.
**Rejected**: The fab-dedupe drafted-intake handoff — it forces routing decisions, conflicts with the micro-change doctrine, and the precedent funnel has produced zero drafted intakes since shipping.
*Introduced by*: 260823-ekp3-code-reorg-skill

### Path-based docs carve-out, not content-based
**Decision**: The only scope exclusions are `docs/memory/` and `docs/specs/`; everything else in the scoped path is in scope regardless of file type.
**Why**: The skill targets repos consuming fab-kit, where those two trees are the only fab-convention prose with dedicated FKF-aware skills; markdown inside a source tree is just source, judged by the same placement/naming frames. A content-type heuristic would be invented, unpredictable, and unneeded.
**Rejected**: Content-based exclusion of doc-like trees — ambiguous (an apply agent must guess "how much markdown makes a docs tree") and wrong for repos whose source legitimately includes prose.
*Introduced by*: 260823-ekp3-code-reorg-skill

### Co-change is evidence only after noise controls
**Decision**: The co-change signal ships with a mandatory control set (window, bulk-commit cap, `-M`, constitution-derived mandated-coupling carve-out, cross-layer whitelist) and a worked command in the skill file.
**Why**: Raw co-change is systematically poisoned on sweep-heavy repos with mandated doc-mirror coupling — measured here: a doc mirror co-changes with its CLI source in ~81% of commits while co-location is constitutionally prohibited. Without controls, the strongest-looking signal is the least trustworthy.
**Rejected**: "Weighted highest" raw co-change — admits false positives that pass the taste guard by construction; per-run improvised aggregation — the pairwise computation is the one signal too fiddly to improvise.
*Introduced by*: 260823-ekp3-code-reorg-skill
