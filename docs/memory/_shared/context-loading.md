---
description: "Smart context loading convention — 7-file always-load layer, standard subagent context, selective domain loading, SRAD protocol, state-keyed Next Steps Convention"
---
# Context Loading

**Domain**: _shared

## Overview

The context loading convention defines how fab skills load project context before execution. It is implemented in `$(fab kit-path)/skills/_preamble.md` as a shared preamble read by all skills. The convention uses a layered approach: always-load essentials, change-specific artifacts, and selective domain memory loading.

## Requirements

### Always Load Layer

Every skill (except `/fab-setup`, `/fab-switch`, `/fab-status`, `/docs-hydrate-memory`) reads seven files as baseline context:

1. `fab/project/config.yaml` — project configuration, naming conventions, model tiers
2. `fab/project/constitution.md` — project principles and constraints (MUST/SHOULD/MUST NOT rules)
3. `fab/project/context.md` — free-form project context: tech stack, conventions, architecture *(optional — no error if missing)*
4. `fab/project/code-quality.md` — coding standards for apply/review: principles, anti-patterns, test strategy *(optional — no error if missing)*
5. `fab/project/code-review.md` — review policy: severity definitions, scope, rework budget *(optional — no error if missing)*
6. `docs/memory/index.md` — documentation landscape (which domains exist; a domain may contain sub-domains, surfaced in that domain's index via a `## Sub-Domains` table — see Selective Domain Loading)
7. `docs/specs/index.md` — specifications landscape (pre-implementation design intent, human-curated)

This gives the agent awareness of project settings, constraints, project context, coding standards, review policy, the documentation landscape, and the specifications landscape before generating any artifact.

The only universal helper beyond the 7 project files is `_preamble.md`. Additional helpers are declared per-skill via the `helpers:` frontmatter field — see **Skill Helper Declaration (Opt-In)** below. Naming conventions and run-kit (rk) recipes that were previously separate helpers are now inlined into `_preamble.md` (§ Naming Conventions, § Run-Kit (rk) Reference). Common `fab` commands are inlined into `_preamble.md` § Common fab Commands so most skills do not need `_cli-fab`.

### Skill Helper Declaration (Opt-In)

Skills declare additional helper files via the `helpers:` frontmatter list. Allowed values: `_generation`, `_review`, `_cli-fab`, `_cli-external`. The agent MUST read `.claude/skills/{helper}/SKILL.md` for each declared helper after reading `_preamble` and before executing the skill body.

Current mapping (post-2026-04-18):

| Skill(s) | `helpers:` |
|----------|------------|
| `fab-new`, `fab-draft` | `[_generation]` |
| `fab-continue`, `fab-ff`, `fab-fff` | `[_generation, _review]` |
| `fab-operator` | `[_cli-fab, _cli-external]` |
| All others (19 skills) | omitted / `[]` (load only `_preamble`) |

`_naming` and `_cli-rk` are NOT allowed values — their content is inlined into `_preamble`. `_preamble` itself is implicit and never listed.

### Preflight Script for Change Context

Skills that operate on an active change resolve the change context by running `src/kit/scripts/lib/preflight.sh [change-name]` via Bash. The script accepts an optional first positional argument as a change name override. When provided, the script resolves the change using case-insensitive substring matching against folder names in `fab/changes/` (excluding `archive/`) instead of reading `.fab-status.yaml`. The override is transient — `.fab-status.yaml` is never modified. When no argument is provided, the script falls back to reading `.fab-status.yaml` (backward compatible).

The matching supports full folder names, partial slug matches, and 4-char random IDs (e.g., `r3m7`). Exact match takes priority; single partial match resolves directly; multiple matches or no match produce a non-zero exit with a descriptive error.

The script validates project initialization, the change directory, and `.status.yaml`, then outputs structured YAML with name, stage, branch, progress, and checklist fields. On non-zero exit, the agent stops and surfaces the stderr error message. On success, the agent uses the stdout YAML instead of re-reading `.status.yaml`.

Since the preflight script validates `config.yaml` and `constitution.md` existence, skills using preflight don't need separate existence checks for these files — they only need to read them for content.

The existing 4-step inline validation sequence (check current, check directory, check .status.yaml, check config/constitution) remains documented in `_preamble.md` as reference for what the script validates internally.

### Selective Domain Loading

When operating on an active change, skills selectively load relevant memory files based on the change's scope. An Affected Memory entry is either **flat** (`{domain}/{name}`) or **sub-domained** (`{domain}/{sub-domain}/{name}` — used after an over-wide domain has been split by `docs-reorg-memory`). Loading is an **up-to-3-hop walk**:

1. Read the intake's Affected Memory section to identify relevant domains (and sub-domains)
2. **Domain index**: for each referenced domain, read `docs/memory/{domain}/index.md` — its `## Sub-Domains` table lists any sub-domains the domain contains
3. **Sub-domain index** *(only if the entry is sub-domained)*: when the referenced file lives in a sub-domain (3-part `{domain}/{sub-domain}/{name}` form), read `docs/memory/{domain}/{sub-domain}/index.md` next
4. **File**: read the specific memory file referenced — `docs/memory/{domain}/{name}.md` for a flat entry, or `docs/memory/{domain}/{sub-domain}/{name}.md` for a sub-domained entry
5. If a referenced domain, sub-domain, or file doesn't exist yet, note this and proceed without error (it will be created during hydrate)
6. Do not load unrelated domains — keeps context focused and efficient

A flat domain is just the degenerate 2-hop case (domain index → file); the sub-domain index hop is taken only when the Affected Memory entry carries the 3-part form. This matches `_preamble.md` § Memory File Lookup and `SPEC-preamble.md`. The always-load layer is unchanged in *which* files it loads (root + domain indexes); only its description acknowledges that a domain may contain sub-domains.

This applies to all skills operating on an active change, not just spec-writing skills.

### Standard Subagent Context

When orchestrator skills (`/fab-ff`, `/fab-fff`) or middle agents (`/fab-continue`) dispatch subagents via the Agent tool, the subagent prompt MUST instruct the subagent to read a standard set of project files **before** executing its task. This is defined in `_preamble.md` § Standard Subagent Context and is distinct from the Always Load layer (which is for the parent agent itself).

The standard subagent context includes:

**Required** (subagent reports error if missing):
- `fab/project/config.yaml`
- `fab/project/constitution.md`

**Optional** (skip gracefully if missing):
- `fab/project/context.md`
- `fab/project/code-quality.md`
- `fab/project/code-review.md`

This is a subset of the Always Load layer — it includes the 5 `fab/project/**` files but excludes `docs/memory/index.md` and `docs/specs/index.md` (which are navigation aids for the parent agent, not project principles needed by subagents).

**Nested dispatch**: When a subagent dispatches its own sub-subagent (e.g., review sub-agent within `/fab-continue`), the inner prompt MUST also include the standard subagent context instruction. The same 5 files are loaded at every nesting level.

**Relationship to Always Load**: The Always Load layer is what the parent agent reads. The Standard Subagent Context is what the parent agent instructs its subagents to read. The parent does not re-pass `docs/memory/index.md` or `docs/specs/index.md` to subagents — those are for the parent's own domain awareness.

### SRAD Protocol

The shared context preamble (`_preamble.md`) includes the SRAD autonomy framework, which all planning skills reference during artifact generation. The framework defines:
- **SRAD scoring table** — four dimensions evaluated on a continuous 0–100 scale per decision point
- **Fuzzy-to-grade mapping** — composite score via weighted mean (w_S=0.25, w_R=0.30, w_A=0.25, w_D=0.20), mapped to grades via trapezoidal thresholds (Certain: 85–100, Confident: 60–84, Tentative: 30–59, Unresolved: 0–29)
- **Critical Rule override** — R < 25 AND A < 25 forces Unresolved regardless of composite
- **Confidence grades** — Certain, Confident, Tentative, Unresolved with corresponding artifact markers
- **Worked examples** — demonstrating how numeric dimension scores interact to produce grades
- **Artifact markers** — `<!-- assumed: ... -->` for Tentative, `<!-- clarified: ... -->` for resolved assumptions
- **Assumptions Summary Block** — standard format with required `Scores` column for per-dimension data; all four grades (Certain, Confident, Tentative, Unresolved) recorded
- **Dynamic gate thresholds** — `/fab-fff` threshold varies by change type (bugfix=2.0, feature/refactor=3.0, architecture=4.0)

`calc-score.sh` parses per-dimension scores from the required `Scores` column in the spec's Assumptions table and writes aggregate dimension statistics (`fuzzy: true`, `dimensions:` block) to `.status.yaml`.

This protocol is loaded as part of the "Always Load" layer via `_preamble.md` and does not require separate file loading.

### Next Steps Convention (State Table)

The `_preamble.md` preamble defines a **state-keyed Next Steps Convention** that all skills use to derive their `Next:` output lines. The convention includes:

1. **State Table** — 9 states (none, initialized, intake, spec, tasks, apply, review pass, review fail, hydrate) each mapping to available commands and a default
2. **State derivation rules** — how to determine the current state from `config.yaml` existence, `.fab-status.yaml`, and `.status.yaml` progress map
3. **Lookup procedure** — determine state, look up in table, output default first
4. **Activation preamble** — when a skill creates/restores a change without activating it (e.g., `/fab-new`, `/fab-archive restore` without `--switch`), the `Next:` line includes a `/fab-switch {name}` instruction before state-derived commands

No skill duplicates or maintains its own suggestion logic — all derive from this single canonical table.

### Exception Skills

The following skills skip the standard context loading layers:
- `/fab-setup` — bootstraps structure, doesn't need project memory
- `/fab-switch` — navigation only
- `/fab-status` — read-only status display
- `/docs-hydrate-memory` — ingests sources, doesn't need to load them first

**Special case**: `/fab-discuss` is *not* an exception — it loads the full 7-file always-load layer. However, it is the only skill whose entire purpose is to surface that layer. Other skills load the always-load layer as a preamble to generating or validating artifacts; `fab-discuss` loads it as its primary output, presenting an orientation summary for exploratory discussion sessions. It does not run preflight, does not require an active change, and does not advance any stage.

## Design Decisions

### External Sub-Domain Addressing (Up-to-3-Hop Selective Load)
**Decision**: When an over-wide domain is split into sub-domains, the sub-domain file is addressed **externally** — the Affected Memory contract, the always-load layer, and selective loading all gain a `{domain}/{sub-domain}/{file}` form. Selective domain loading becomes an up-to-3-hop walk: domain index → (only if the entry is sub-domained) sub-domain index → file. A flat domain stays the degenerate 2-hop case (no sub-domain index hop, byte-identical to pre-change behavior).
**Why**: External addressing makes sub-domains first-class and navigable — there is no "find-the-file-anywhere-under-the-domain" resolver ambiguity (the failure mode of the Internal/duplicate-truth-file alternative). loom's historical External-style index churn was the *hand-edited-index* problem, which `tciy` already eliminated (sub-domain indexes are generated by `fab memory-index` too), so External's only historical downside is moot post-`tciy`; its upside (explicit, navigable addressing) stands.
**Rejected**: Internal addressing (sub-domain files resolved by search under the domain) — re-introduces resolver ambiguity and a duplicate-truth-file failure mode. A flat-only model (never sub-dividing) — leaves over-wide domains (e.g. `fab-workflow` at 20 files > ~12) with no structural escape valve.
*Introduced by*: 260607-sx7a-reorg-memory-shape-rebalance

### Smart Loading for All Skills on Active Changes
**Decision**: Expanded "Memory Lookup" from spec-writing-only to all skills operating on an active change.
**Why**: Agents need domain awareness for planning, implementation, and review — not just spec writing.
**Rejected**: Per-skill opt-in — too much maintenance overhead and easy to miss new skills.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Always Load docs/specs/index.md
**Decision**: Added `docs/specs/index.md` to the "Always Load" layer as a 4th baseline file.
**Why**: Gives every skill awareness of the specifications landscape (pre-implementation design intent) alongside the documentation landscape. The index is lightweight and human-curated, so context cost is minimal.
**Rejected**: Loading design index only when relevant — same inconsistency risk as with memory/index.md.
*Introduced by*: 260207-bb1q-add-specs-index

### Always Load docs/memory/index.md
**Decision**: Added `docs/memory/index.md` to the "Always Load" layer alongside config.yaml and constitution.md.
**Why**: Gives every skill baseline awareness of the documentation landscape. The index is lightweight (a table of domains), so the context cost is minimal.
**Rejected**: Loading only when needed — would require each skill to independently decide, leading to inconsistency.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Always-Load `_cli-rk` Skill for rk Capabilities *(Superseded)*
**Decision**: Added `_cli-rk.md` as an optional always-load skill in `_preamble.md`, separate from `_cli-external.md`.
**Why**: rk iframe+proxy capabilities benefit every fab session (visual display of diagrams, plans, slide decks), not just operator sessions. Centralizing the visual display recipe in `_cli-rk.md` (rather than baking it into visual-explainer) gives any skill the superpower via separation of concerns.
**Rejected**: Adding rk to `_cli-external.md` and promoting to always-load — would bloat every session with operator-specific content (wt, idea, /loop). Also rejected decentralized approach (iframe logic in visual-explainer only) — forces other skills to duplicate logic or use visual-explainer as a middleman.
*Introduced by*: 260416-mgsm-add-cli-rk-skill
*Superseded by*: 260418-or0o-flatten-skill-helpers — `_cli-rk.md` content inlined into `_preamble.md` § Run-Kit (rk) Reference; the separate helper file is deleted. The silent-fail-when-rk-missing design is preserved verbatim in the inlined subsection.

### Flatten Helper Include Tree
**Decision**: Collapse the helper always-load set from `{_preamble, _cli-fab, _naming, _cli-rk}` to `{_preamble}` only. Inline `_naming` and `_cli-rk` into `_preamble`. Add a new per-skill `helpers:` frontmatter field listing the additional helpers each skill needs (`_generation`, `_review`, `_cli-fab`, `_cli-external`). Inline the 6 most-used `fab` commands into `_preamble` § Common fab Commands. Compress `_cli-fab` from 773 lines to ≤300.
**Why**: Two root causes. (1) Universal "also read" fanout from `_preamble` shipped ~1324 lines of helper content that 15 of 24 skills didn't use. (2) Agents silently skipped 2nd-layer "also read" directives — pointer-based loading was non-deterministic. Replacing the fanout with explicit, frontmatter-declared helpers is auditable, grep-able, and reliable (agents read frontmatter before body). Inlining the smallest helpers and the commonest commands eliminates the 2nd layer for most skills entirely.
**Rejected**: (a) Splitting `_preamble` further — deepens the tree, worsens skip-rate. (b) Relying on prompt caching — doesn't fix correctness when pointers are silently skipped. (c) Full inline of `_cli-fab` — adds ~500 lines to universal load. (d) Renaming `_`-prefix to visible names (backlog `[84bh]`) — addresses visibility but not fanout; structural fix supersedes it.
*Introduced by*: 260418-or0o-flatten-skill-helpers

### Standard Subagent Context as Centralized Template
**Decision**: Added a Standard Subagent Context subsection to `_preamble.md` § Subagent Dispatch, listing the 5 `fab/project/**` files that every subagent must read. Skills reference this template instead of maintaining ad-hoc file lists.
**Why**: Each skill that dispatched subagents maintained its own context list, creating silent quality gaps (forgotten files) and drift risk (new files not propagated). Centralizing in `_preamble.md` ensures all subagents — at any nesting depth — inherit project principles automatically.
**Rejected**: Including `docs/memory/index.md` and `docs/specs/index.md` in subagent context — these are navigation aids for the parent agent, not project principles needed by subagents.
*Introduced by*: 260318-dzze-standard-subagent-context

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260607-sx7a-reorg-memory-shape-rebalance | 2026-06-07 | Selective Domain Loading rewritten from a flat 2-hop walk to an **up-to-3-hop walk** (domain index → conditional sub-domain index → file), matching `_preamble.md` § Memory File Lookup and `SPEC-preamble.md`. An Affected Memory entry is now either flat (`{domain}/{name}`) or sub-domained (`{domain}/{sub-domain}/{name}`, used after a split); the sub-domain index hop is taken only for the 3-part form, so flat domains stay the degenerate 2-hop case (byte-identical behavior). Always Load layer description for `docs/memory/index.md` now notes a domain may contain sub-domains (surfaced via the domain index's `## Sub-Domains` table). New design decision "External Sub-Domain Addressing (Up-to-3-Hop Selective Load)". |
| 260418-or0o-flatten-skill-helpers | 2026-04-18 | Flattened helper include tree. Removed three "Also read" fanout directives from `_preamble.md`. Inlined `_naming.md` (76 lines) and `_cli-rk.md` (91 lines) into `_preamble` as `## Naming Conventions` and `## Run-Kit (rk) Reference` subsections; deleted source files and SPEC-_cli-rk.md. Added `## Skill Helper Declaration` subsection defining the per-skill `helpers:` frontmatter (values: `_generation`, `_review`, `_cli-fab`, `_cli-external`). Added `## Common fab Commands` subsection with headline table for the 6 most-used command families (`preflight`, `score`, `log command`, `change`, `resolve`, `status`). Compressed `_cli-fab.md` from 773 to ≤300 lines via condensed flag/syntax tables while preserving all canonical flag behavior. Set `helpers:` on 6 skills: `fab-new`/`fab-draft` → `[_generation]`, `fab-continue`/`fab-ff`/`fab-fff` → `[_generation, _review]`, `fab-operator` → `[_cli-fab, _cli-external]`. Other 19 skills load only `_preamble`. Superseded "Always-Load `_cli-rk`" design decision. Closed backlog item `[84bh]`. |
| 260416-mgsm-add-cli-rk-skill | 2026-04-16 | Added `_cli-rk.md` as optional always-load skill — run-kit iframe windows, proxy, visual display recipe. Separate from `_cli-external.md` (which remains operator-only). Silent fail when rk unavailable. Added design decision for centralized recipe approach. |
| 260402-gnx5-relocate-kit-to-system-cache | 2026-04-02 | Updated kit path references: `_preamble.md` is now at `$(fab kit-path)/skills/_preamble.md` (resolved from system cache). Template access via `$(fab kit-path)/templates/` instead of `fab/.kit/templates/`. Test-build guard removed from preamble (`kit.conf` eliminated). Skills deployed to `.claude/skills/` unchanged. |
| 260318-dzze-standard-subagent-context | 2026-03-18 | Added Standard Subagent Context section — defines the 5 `fab/project/**` files that every subagent prompt must include, distinct from the Always Load layer. Added design decision for centralized template approach. |
| 260303-6b7c-update-underscore-skill-references | 2026-03-04 | Standardized all skill top-of-file `_preamble.md` references to use `$(fab kit-path)/skills/_preamble.md` (no `./` prefix). No content changes to context-loading requirements — all references already used correct form. |
| 260221-5tj7-rename-context-to-preamble | 2026-02-21 | Renamed shared skill preamble file from `_context.md` to `_preamble.md`. Updated all references throughout — Overview, SRAD Protocol, and Next Steps Convention sections now reference `_preamble.md` |
| 260220-9ogw-add-fab-discuss | 2026-02-20 | Added `fab-discuss` special case note in Exception Skills section — the only skill whose primary output IS the always-load layer |
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Replaced stale `/fab-init` → `/fab-setup` in exception list (2 occurrences) |
| 260218-xkkc-add-code-review-5cs-quality | 2026-02-18 | Expanded Always Load layer from 6 to 7 files — added `fab/code-review.md` (optional, review policy) as item 5 after `code-quality.md`, grouping all `fab/` files before `docs/` files |
| 260218-bb93-restructure-config-yaml | 2026-02-18 | Expanded Always Load layer from 4 to 6 files — added `fab/context.md` (optional, free-form project context) and `fab/code-quality.md` (optional, coding standards) |
| 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions | 2026-02-16 | Added Next Steps Convention (State Table) section — documents the state-keyed suggestion derivation convention in `_preamble.md` replacing the old skill-keyed lookup table |
| 260215-v4n7-DEV-1025-rename-brief-to-intake | 2026-02-15 | Renamed `brief` stage/artifact to `intake` throughout — stage identifiers, artifact filenames, YAML keys, prose references |
| 260212-f9m3-enhance-srad-fuzzy | 2026-02-14 | SRAD protocol updated to fuzzy 0–100 dimension scoring with weighted mean aggregation, trapezoidal grade thresholds, optional Scores column, dynamic gate thresholds by change type |
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated always-load paths to `docs/memory/index.md` and `docs/specs/index.md`; updated selective domain loading to `docs/memory/{domain}/` |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_preflight.sh` → `lib/preflight.sh` in preflight script reference |
| 260213-w4k9-explicit-change-targeting | 2026-02-13 | Preflight script now accepts optional `$1` change-name override with case-insensitive substring matching; `fab/current` is not modified when override is used |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated docs/specs/index.md reference, removed plan.md from artifact loading |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Fixed stale `auto-guess` marker reference in SRAD Protocol section — replaced with `clarified` marker per updated `_preamble.md` |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added SRAD protocol section — framework is loaded via `_preamble.md` as part of Always Load layer |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-5mjv-preflight-grep-scripts | 2026-02-07 | Added preflight script integration — Change Context layer now uses `_preflight.sh` for validation and state resolution |
| 260207-bb1q-add-specs-index | 2026-02-07 | Added `docs/specs/index.md` as 4th file in Always Load layer |
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Added `docs/memory/index.md` to always-load, expanded selective loading to all skills on active changes |
