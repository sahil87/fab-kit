---
type: memory
description: "The /docs-distill-memory skill — rewrites an existing memory domain's topic files to the FKF §3.2/§3.3 present-truth style, one domain per run, propose-then-apply (read-only until approval). Intent-first removal classifier: durable intent relocates into Design Decisions, intent-free narration recorded elsewhere is deleted, else relocate. Compresses over-cap descriptions, strips change-ids. Excludes generated index.md/log.md, curated log.seed.md, and removed-domains.md. Idempotent."
---
# Distill

**Domain**: memory-docs

## Overview

`/docs-distill-memory <domain>` rewrites an existing `docs/memory/` domain's topic files to the **FKF present-truth style** (`$(fab kit-path)/reference/fkf.md` §3.2/§3.3). The FKF present-truth rules govern what memory writers produce going forward; a corpus authored before them accumulates transition narration, superseded-state prose, and over-cap or change-id-carrying `description:` frontmatter. This skill is the remediation counterpart to the forward-looking writers — it is the only skill that rewrites **existing** body prose to the present-truth style across a whole domain. It is defined in `$(fab kit-path)/skills/docs-distill-memory.md` and auto-discovered by `fab sync`'s kit-skills-dir enumeration (no manifest registration).

> **Distinct from the sibling doc skills**: [hydrate](/memory-docs/hydrate.md) documents `/docs-hydrate-memory`, whose backfill mode is body-preserving (frontmatter only) and whose ingest/generate modes author *new* content. `/docs-reorg-memory` (see [templates](/memory-docs/templates.md) § Memory Tree Shape) reorganizes **structure** (splits/merges/moves + link rewrites), never body prose to a style. `/fab-continue` hydrate writes each change's delta as current truth but touches only the sections its change affects (see [execution-skills](/pipeline/execution-skills.md) § Hydrate Behavior). Distillation is the corpus-wide body-prose remediation none of those cover.

## Requirements

### One Domain Per Run, Propose-Then-Apply

The skill operates on exactly one memory domain per invocation, named by its `docs/memory/` folder (e.g. `/docs-distill-memory pipeline`). It runs **read-only analysis** first, emits a **per-file proposed-rewrite report** (per-file diffs/summaries with before/after snippets for the non-obvious edits and every relocation), and applies rewrites **only on explicit user approval** — the same report → confirm-and-apply posture as `/docs-reorg-memory`. Confirmation offers **apply all**, **cherry-pick** (specific files), or **skip**; nothing mutates until the user approves. A multi-domain invocation is rejected — run the skill once per domain so each domain's diffs are approved on their own. Memory files encode load-bearing behavioral contracts, so a human gates each domain; the skill is never an autonomous bulk rewriter.

### Present-Truth Rewrite Semantics (FKF §3.2/§3.3)

A rewrite transforms each topic file to the FKF present-truth style, citing the shipped extract `$(fab kit-path)/reference/fkf.md` (deployed skills reach the extract; the dev-repo `docs/specs/fkf.md` is absent in user repos). A rewrite:

- **Removes transition narration** — "renamed X→Y in {id}", "this inverts/supersedes {id}'s claim", "was `old.value`", "superseding the historical …", and similar retrospective prose.
- **Removes superseded-state descriptions** — the body carries only what IS; previous states belong to the per-folder generated `log.md`, git history, and archived change folders.
- **Keeps allowed provenance** — trailing `(change-id)` citations and the `*Introduced by*: {change-name}` field on Design Decisions. **Bare 4-char ids count the same as dated ids**: in trailing-citation position they stay; woven into narration they go with the narration (§3.3 defines allowed provenance by position/form, not id format).
- **Fixes `description:` frontmatter** — strips change-ids (§3.2 bans both a trailing `— xu0k`-style suffix and a `(d9rs)`-style citation) and compresses an over-cap value to the **≤500-character** single-line routing-signal shape, moving displaced routing-irrelevant detail into the body where it is not already present.

### Intent-First Removal Classifier (the critical constraint)

Token savings come from dropping narration, **never** rationale. Every removal candidate is classified **intent first**: does it carry durable intent — a deliberate-behavior defense, a "don't re-break this", a rejected alternative?

- **Durable intent → relocate into `## Design Decisions`** (`Why` / `Rejected`) as present-tense design intent, **regardless of where else it is recorded** — it is never deleted. This repo's history shows agents repeatedly "fixing" deliberate behavior (e.g. the Copilot poll-predicate); the distilled file must retain those defenses.
- **Intent-free narration recorded elsewhere → delete.** Deletion is safe only for narration whose content already lives in the per-folder `log.md`, git history, or an archived change folder.
- **When in doubt, relocate.** The safe default preserves; it does not delete.

### Generated-File Exclusions and the Tombstone Exemption

The skill never hand-edits generated files and excludes specific curated inputs from rewrite:

- **`index.md` (root/domain/sub-domain tiers) and `log.md` are the generated pair** — written solely by `fab memory-index` (FKF §5/§6). The skill regenerates them after applying rewrites; it never edits their rows.
- **`log.seed.md` is a curated read-only SEED INPUT, not a generated file** — `fab memory-index` *reads* it during the seed-merge but never *writes* it (like `description:` frontmatter, it is a gathered input; the generator stays the sole writer of `log.md`). It is nonetheless **excluded from distillation**: its body *is* a citation-carrying seed ledger of pre-FKF history in the §6.2 entry format, not topic-file prose — the same exclusion posture as a ledger. Skip it entirely.
- **`_shared/removed-domains.md` is EXEMPT** from rewrite — the §3.3 tombstone carve-out: its body *is* removal records, a citation-carrying tombstone ledger, not transition narration. (fab-kit's own tree has no such file; the exemption matters in user projects, where `/docs-reorg-memory` authors it.)

### Regeneration with the Refuse-Before-Regen Guard

After applying rewrites, the skill regenerates the generated files via `fab memory-index` — never by hand-editing rows. Before regenerating it consults `fab memory-index --check` (the same guard [hydrate](/memory-docs/hydrate.md) § Refuse-Before-Regen Guard carries; exit tiers documented in `_cli-fab` § fab memory-index): on **exit 0/1** it regenerates; on **exit 2** (destructive loss) it **refuses** and surfaces the pointer `→ run /docs-reorg-memory to remediate …`. This guard is a **no-op on born-compatible fab-kit trees** — they are always exit 0/1, never 2 — so it never fires here; it is defense-in-depth for a pre-fab-kit tree reaching this skill (not dead code). Regeneration derives the **index tiers** from folder contents + each file's `description:` frontmatter (content-only, no dates) and each **`log.md`** from the C-lite join of git history + per-change `.status.yaml` `summary:` fields (freeze-on-write, append-only; any `log.seed.md` merged beneath).

### Reduced Context Loading Override

The skill file's `## Context Loading` section is the skill-file override the `_preamble.md` §1 always-load contract keys on: it does **not** load the always-load layer and requires **no active change, config, or constitution** (see [_shared/context-loading](/_shared/context-loading.md) § Exception Skills). It reads only the memory landscape (`docs/memory/index.md` + the target domain's `index.md` and any sub-domain index), every topic file in the target domain, and `$(fab kit-path)/reference/fkf.md` (the normative extract, so each rewrite cites the deployed rule). It declares no `helpers:`, reaching `_cli-fab` § fab memory-index by in-body pointer instead — the `/docs-reorg-memory` pointer style.

### Idempotent Re-Runs

Re-running the skill on an already-distilled domain finds nothing to rewrite and reports "no rewrites proposed — {domain} is already distilled", mutating no file. `fab memory-index` regeneration is byte-stable, so a no-op re-run produces no index diff (Constitution III).

### fab-help Group Registration

`/docs-distill-memory` is registered in the "Maintenance" `/fab-help` group (with `/docs-reorg-memory`, `/docs-reorg-specs`, `/docs-hydrate-specs`) via `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go`, and listed on the hardcoded `Maintain docs:` TYPICAL FLOW line. `/fab-help` discovers the command itself by scanning deployed skill frontmatter; only its *grouping* comes from the map (an unmapped skill falls into the "Other" bucket). The registration carries no command-signature change, so `_cli-fab.md` is unaffected.

## Design Decisions

### Separate Discoverable Skill, Not a `docs-reorg-memory` Mode

**Decision**: `/docs-distill-memory` is a separate user-invocable skill, not a mode of `/docs-reorg-memory`.
**Why**: A mode is not discoverable — a distinct command surfaces in `/fab-help` and the command inventories on its own.
**Rejected**: A `docs-reorg-memory` mode — the corpus-style remediation would be buried and undiscoverable.
*Introduced by*: 260717-dgp8-docs-distill-memory-skill

### Per-Domain Human Gate Over Autonomous Bulk Rewrite

**Decision**: The skill operates one domain per run with per-file diff review, applying only on explicit approval — never an autonomous full-corpus rewrite.
**Why**: Memory files encode load-bearing behavioral contracts; a human approves per domain seeing per-file diffs, so a mis-classified rewrite cannot silently corrupt a contract at scale.
**Rejected**: One-run full-corpus autonomous rewrite — too risky for contract docs.
*Introduced by*: 260717-dgp8-docs-distill-memory-skill

### Intent-First Classification, Relocate-When-In-Doubt

**Decision**: Removal candidates are classified by durable intent before anything else — intent is relocated into Design Decisions regardless of where else it is recorded; only intent-free narration recorded elsewhere is deleted; when in doubt, relocate.
**Why**: The failure mode this skill must not cause is stripping a deliberate-behavior defense (agents in this repo repeatedly "fix" deliberate behavior). A relocate-biased classifier keeps rationale as a design *fact* while still dropping the narration that costs tokens on every context load.
**Rejected**: Zero-provenance stripping (loses the `(id)` citations that cheaply defend deliberate behavior); deleting any narration recorded elsewhere without an intent check (drops rationale that happens to also appear in a log).
*Introduced by*: 260717-dgp8-docs-distill-memory-skill

### Cite the Shipped FKF Extract, Not the Dev-Repo Spec

**Decision**: The skill cites `$(fab kit-path)/reference/fkf.md` §3.2/§3.3, not `docs/specs/fkf.md`.
**Why**: The skill ships to user repos where only the kit extract is reachable; deployed sibling skills cite it the same way.
**Rejected**: Citing the dev-repo `docs/specs/fkf.md` — absent in user repos.
*Introduced by*: 260717-dgp8-docs-distill-memory-skill
