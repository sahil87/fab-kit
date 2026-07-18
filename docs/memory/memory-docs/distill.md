---
type: memory
description: "The /docs-distill-memory [<domain>] skill — rewrites a memory domain's topic files to the FKF §3.2/§3.3 present-truth style: strips narration, superseded state, change-id heading suffixes, byte-identical duplicates; rewrites DD changelog bullets; relocates TODOs to the backlog. One domain per run, propose-then-apply. <domain> optional: named forces a full read; omitted surveys via one fab memory-index --check --json call and auto-picks. Intent-first classifier keeps durable intent. Idempotent."
---
# Distill

**Domain**: memory-docs

## Overview

`/docs-distill-memory [<domain>]` rewrites an existing `docs/memory/` domain's topic files to the **FKF present-truth style** (`$(fab kit-path)/reference/fkf.md` §3.2/§3.3). The FKF present-truth rules govern what memory writers produce going forward; a corpus authored before them accumulates transition narration, superseded-state prose, and over-cap or change-id-carrying `description:` frontmatter. This skill is the remediation counterpart to the forward-looking writers — it is the only skill that rewrites **existing** body prose to the present-truth style across a whole domain. The `<domain>` argument is optional: named explicitly it forces a full read of that domain; omitted it runs a heuristic **survey** that flags candidate domains and auto-picks the first (see § No-Arg Survey Mode). It is defined in `$(fab kit-path)/skills/docs-distill-memory.md` and auto-discovered by `fab sync`'s kit-skills-dir enumeration (no manifest registration).

> **Distinct from the sibling doc skills**: [hydrate](/memory-docs/hydrate.md) documents `/docs-hydrate-memory`, whose backfill mode is body-preserving (frontmatter only) and whose ingest/generate modes author *new* content. `/docs-reorg-memory` (see [templates](/memory-docs/templates.md) § Memory Tree Shape) reorganizes **structure** (splits/merges/moves + link rewrites), never body prose to a style. `/fab-continue` hydrate writes each change's delta as current truth but touches only the sections its change affects (see [execution-skills](/pipeline/execution-skills.md) § Hydrate Behavior). Distillation is the corpus-wide body-prose remediation none of those cover.

## Requirements

### One Domain Per Run, Propose-Then-Apply

The skill reads-in-full and rewrites exactly one memory domain per run, named by its `docs/memory/` folder (e.g. `/docs-distill-memory pipeline`) or auto-picked by the no-arg survey (§ No-Arg Survey Mode). "One domain per run" is a property of the **analysis+apply unit, not the invocation**: exactly one domain is read-in-full and rewritten per run whether it was named explicitly or auto-picked. It runs **read-only analysis** first, emits a **per-file proposed-rewrite report** (per-file diffs/summaries with before/after snippets for the non-obvious edits and every relocation), and applies rewrites **only on explicit user approval** — the same report → confirm-and-apply posture as `/docs-reorg-memory`. Confirmation offers **apply all**, **cherry-pick** (specific files), or **skip**; nothing mutates until the user approves. A multi-domain invocation is rejected — run the skill once per domain so each domain's diffs are approved on their own. Memory files encode load-bearing behavioral contracts, so a human gates each domain; the skill is never an autonomous bulk rewriter.

### Optional `<domain>` — Named Override or Auto-Picked

The `<domain>` argument is **optional**. Named explicitly, it is the **override**: the skill skips the survey heuristics and forces a full read of that domain. Omitted, the skill runs **survey mode** (§ No-Arg Survey Mode) — a heuristic scan that flags candidate domains and auto-picks the first. Domain resolution is case-insensitive substring match against folder names; an ambiguous name (matches >1 folder) or an unknown name aborts with the available-domains list. A multi-domain invocation aborts (one domain per run). A no-`<domain>` invocation is **survey mode, not an abort** — the only aborts are the ambiguous, unknown, and multiple-domains cases.

### No-Arg Survey Mode

A no-arg invocation runs a **heuristic survey** across all domains before any full read. The survey is a cheap scan — it ranks and picks, it does not classify exhaustively (the full read still runs once a domain is selected). It reports per domain in the order of `docs/memory/index.md`'s domain table (deterministic, matches the user-facing landscape) and counts **flagged files** per domain.

The survey's signal source is **one `fab memory-index --check --json` invocation** — the canonical machine surface (`_cli-fab` § fab memory-index), not an agent-side grep of frontmatter and bodies. It runs the check once and aggregates per-domain flagged-file counts from four finding kinds — the same §3.2/§3.3 defect classes distillation fixes:

1. `malformed[]` kind **`description-change-id`** — a `description:` carrying a registry-gated change-id (§3.2 ban, enforced/blocking).
2. `malformed[]` kind **`description-over-cap`** — a `description:` over the 1000-rune blocking cap (§3.2).
3. `warnings[]` kind **`description-length`** — a `description:` in the 501–1000 advisory band, over the 500-char soft cap (§3.2).
4. `warnings[]` kind **`narration-density`** — a topic file whose body carries ≥5 narration markers (§3.3 distillation-debt meter).

**Aggregation:** a file with **multiple findings counts once** (dedupe by `path`); a **sub-domain file rolls up to its domain** — the first path segment under `docs/memory/`. The survey **re-applies the distillation exclusion set to the JSON finding paths** — it drops any finding whose path is an `index.md` or `_shared/removed-domains.md` before counting. The primitive scans neither exhaustively: it inspects `index.md` stubs for the three description-tier kinds and treats `_shared/removed-domains.md` as an ordinary topic file (its citation-dense rows trip `narration-density`), so their findings would otherwise be miscounted against a distilled domain (`log.md` / `log.seed.md` never appear — the walker skips them). Re-applying the exclusion set keeps a fully-distilled tree surveying clean and leaves the auto-pick semantics unchanged. The **check's exit code does NOT gate the survey** — it consumes the report, it is not a regen guard, so exit 1 (benign drift) and exit 2 (destructive loss) still produce a survey (the JSON is emitted on all `--check` exits). A **missing `type: memory` is NOT a survey signal** — the full read stamps it once a domain is selected, so it does not affect ranking.

**Older-binary fallback.** When `fab memory-index --check --json` is unavailable, or its output lacks the `warnings` key (an older binary that predates the machine surface), the survey **falls back to the legacy agent-side grep heuristics verbatim** and **warns the user to upgrade `fab`** (mirroring `/docs-reorg-memory`'s Step 1 older-binary fallback posture) — the three §3.2/§3.3 classes: a `description:` over the 500-character cap, change-ids in `description:` (a `— xu0k`-style suffix or a `(d9rs)`-style citation), and narration markers in bodies (a grep for the transition-narration patterns `renamed` / `supersed` / `` was ` `` / `superseding the historical` / `inverts`, seeded from the full-read classification's pattern list and extensible). The fallback applies the same exclusion set (skip `index.md`, `log.md`, `log.seed.md`, `_shared/removed-domains.md`) and recurses into sub-domains like the full read.

The survey then reports per-domain status, **auto-selects the first flagged domain** in domain-table order, announces the pick, and proceeds into the one-domain flow (full read → per-file report → approval gate, all unchanged). When nothing is flagged anywhere, it reports the terminal **"all domains distilled (survey heuristic)"** case and stops without reading or mutating anything.

The survey is **heuristic**: a domain can pass the cheap scan while still carrying superseded-state prose. That is fine for ranking/picking, but the only silent-skip risk is the terminal all-clean case, so survey output **states the caveat** (`Survey is heuristic; run /docs-distill-memory <domain> to force a full read of a specific domain.`), mandatory on the all-clean case.

### Dynamic `Next:` Line

The skill's closing `Next:` line lists the **surveyed remaining candidate domains** — in `docs/memory/index.md` domain-table order, each with its flagged-file count — or reports "all domains distilled" when none remain. On a **no-arg** invocation the completion line reuses the Step 0 survey result minus the completed domain; a domain the user **skipped** or only **partially cherry-picked** stays listed while it still carries flagged files (the line reports surveyed truth). On an **explicit-`<domain>`** invocation (no upfront survey ran), the completion step runs the survey to populate the line.

### Present-Truth Rewrite Semantics (FKF §3.2/§3.3)

A rewrite transforms each topic file to the FKF present-truth style, citing the shipped extract `$(fab kit-path)/reference/fkf.md` (deployed skills reach the extract; the dev-repo `docs/specs/fkf.md` is absent in user repos). A rewrite:

- **Removes transition narration** — "renamed X→Y in {id}", "this inverts/supersedes {id}'s claim", "was `old.value`", "superseding the historical …", and similar retrospective prose.
- **Removes superseded-state descriptions** — the body carries only what IS; previous states belong to the per-folder generated `log.md`, git history, and archived change folders.
- **Keeps allowed provenance** — trailing `(change-id)` citations and the `*Introduced by*: {change-name}` field on Design Decisions. **Bare 4-char ids count the same as dated ids**: in trailing-citation position they stay; woven into narration they go with the narration (§3.3 defines allowed provenance by position/form, not id format).
- **Fixes `description:` frontmatter** — strips change-ids (§3.2 bans both a trailing `— xu0k`-style suffix and a `(d9rs)`-style citation) and compresses an over-cap value to the **≤500-character** single-line routing-signal shape, moving displaced routing-irrelevant detail into the body where it is not already present.

### Structural Removal Classes (§3.3)

Beyond the narration/superseded-state/description rewrites above, the skill handles four structural defect classes, each cited to `$(fab kit-path)/reference/fkf.md` §3.3 and each identified in Step 1, reported in Step 2, and applied in Step 4:

- **Change-id heading suffixes** — a heading carrying a change-id token (`### Dispatch States (xu0k)`, `## Foo — 260718-mxgu`, `## xu0k — dispatch states`) has the **token stripped, keeping the heading text** (§3.3: a heading names its topic, never a change). Token recognition is **registry-gated** (the same posture the mxgu change-id checks use): a full `YYMMDD-XXXX-slug` token always matches; a bare 4-char id matches **only** when registry-plausible (present under `fab/changes/*` / `archive/**`) — the Step 3 human gate covers residual false positives. If the stripped token carried provenance worth keeping, it is re-added as a **trailing `(change-id)` citation in the section body** (allowed provenance), never left in the heading.
- **Literal duplicate headings/blocks** — a **byte-identical** duplicated heading pair or block within one file has the **later duplicate removed** (§3.3: a body states current truth once). A merely *similar* (non-byte-identical) block is a **near-duplicate** — **flagged in the Step 2 report for manual review, never auto-removed**. Content judgment stays with the human gate; cross-file duplication is `/docs-reorg-memory`'s duplicate-coverage pass, not this skill's.
- **Design-Decisions changelog bullets** — a `- **{change-id} — retired X**`-shaped bullet inside a `## Design Decisions` section (the shape §3.3 bans there) is **rewritten to the four-field entry** (**Decision** / **Why** / **Rejected** / *Introduced by* — the change-id moves into *Introduced by* or a trailing citation) when it encodes a durable decision, or **removed** under the deletion-safety rule when it is pure change history already recorded in `log.md`/git. It **never fabricates rationale**: when `Why`/`Rejected` content is not derivable from the bullet or surrounding context, the rewritten entry carries only the fields that exist (Decision + *Introduced by*).
- **Embedded operational TODOs → relocated to `fab/backlog.md`, never deleted** — a TODO, "still needs X", or next-step checklist item in a memory body is **relocated** out of the body into `fab/backlog.md` (§3.3: follow-up work items belong in the project backlog or change folder, not a memory body). Relocation removes the TODO from the body and appends a standard backlog entry `- [ ] [{fresh-4char-id}] {YYYY-MM-DD}: {TODO text} (relocated from docs/memory/{domain}/{file}.md by /docs-distill-memory)` under the backlog's `## Open` section, generating a fresh 4-char id (not colliding with a registered change or existing backlog id) and today's date. When `fab/backlog.md` does not exist (user repos) it is created with a minimal `# Backlog` header first. `fab/backlog.md` is the **one** file outside `docs/memory/` this skill writes. Relocation honors the Step 3 per-file approval unit — a file the user skips or cherry-picks away keeps its TODOs, so no orphaned relocation is written.

The Step 2 report and completion line carry matching counters for the four classes (e.g. `strip change-id heading suffixes: N`, `dedupe byte-identical blocks: N (near-duplicates flagged: M)`, `rewrite DD changelog bullets: N`, `RELOCATE TODOs → fab/backlog.md: N`). Each class runs the same intent-first classifier below — the DD-bullet rewrite applies the intent test, and the TODO class is a relocation, never a deletion.

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

The skill file's `## Context Loading` section is the skill-file override the `_preamble.md` §1 always-load contract keys on: it does **not** load the always-load layer and requires **no active change, config, or constitution** (see [_shared/context-loading](/_shared/context-loading.md) § Exception Skills). Once a target domain is resolved, it reads only the memory landscape (`docs/memory/index.md` + the target domain's `index.md` and any sub-domain index), every topic file in the target domain, and `$(fab kit-path)/reference/fkf.md` (the normative extract, so each rewrite cites the deployed rule). **Survey mode consumes the machine surface, not a full read**: on a no-arg invocation, before a target domain exists, the survey runs one `fab memory-index --check --json` call and aggregates its `malformed[]`/`warnings[]` findings per domain (re-applying the distillation exclusion set to the finding paths) — it reads no topic file. Only the older-binary fallback (no `--json` / no `warnings` key) scans domains read-only via the agent-side grep. Either way the full read is confined to the single domain the survey selects. It declares no `helpers:`, reaching `_cli-fab` § fab memory-index by in-body pointer instead — the `/docs-reorg-memory` pointer style.

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

### No-Arg Survey Over a Multi-Domain Loop or a Persistent State Marker

**Decision**: The optional `<domain>` uses a no-arg heuristic **survey** that auto-picks the first flagged domain into the single one-domain flow, keeping the per-run unit at one domain.
**Why**: Distillation is a one-time corpus sweep (the forward-looking writers already emit present truth), so the natural workflow is "run until nothing's left". The auto-pick gives a one-keystroke loop without hand-tracking remaining domains, while the survey stays read-only and re-runnable so a fully-distilled tree surveys clean every time. The one-domain-per-run guardrail was enforced at the invocation seam, but its rationale (per-domain approval granularity + rewrite-quality/context budget) is a property of the analysis+apply unit — forcing the user to name the domain bought no safety the Step 3 approval gate did not already provide.
**Rejected**: (a) A multi-domain invocation expanding to sequential per-domain runs with per-domain approval gates — a single session chewing through all domains erodes rewrite quality as context fills, and the auto-pick already gives the one-keystroke loop; the multiple-domains abort stays. (b) A persistent distilled-state marker/tracking file — distillation is a one-time remediation sweep, and extra state violates the docs-are-source-of-truth ethos (Constitution II); survey-scanning each time is cheaper and stateless.
*Introduced by*: 260718-ukpf-distill-noarg-survey
