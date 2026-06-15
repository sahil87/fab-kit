---
type: memory
description: "`/docs-hydrate-memory` skill — argument routing, three modes (ingest + generate + backfill), hydration rules, mechanical index regen via `fab memory-index`, and the index-ownership model (`description:` frontmatter = single hand-curated field; stub before index — d9rs). Backfill adds frontmatter to existing files (pre-fab-kit trees), body-preserving + caller-aware regen; dispatched by `docs-reorg-memory` compatibility orchestration (5ewp). Refuse-before-regen guard at every regen site consulting `fab memory-index --check` exit 2 (destructive loss), born-compatible no-op; reorg detection now calls the `--check --json` primitive (glwc). All three modes author FKF frontmatter (`type: memory` + `description:`), stop writing per-file `## Changelog`, and use bundle-relative memory↔memory links; backfill also stamps `type: memory`, staying body-preserving (8fr5)"
---
# Hydrate

**Domain**: memory-docs

## Overview

`/docs-hydrate-memory [sources...|folders...|backfill]` is a standalone skill that operates in three modes: **ingest mode** (fetching URLs or reading `.md` files into `docs/memory/`), **generate mode** (scanning the codebase for undocumented areas and producing structured memory files), and **backfill mode** (re-scanning an existing tree to add missing `description:` frontmatter — the pre-fab-kit migration path, see Backfill Mode below). Mode is determined automatically by argument type — ingest/generate by argument type, backfill by the explicit `backfill` keyword or a `/docs-reorg-memory` dispatch. It requires `docs/memory/` to exist (created by `/fab-setup`). See [hydrate-generate](/memory-docs/hydrate-generate.md) for full generate mode requirements. Since d9rs the skill file carries an explicit `## Context Loading` override section (it skips the always-load layer — the skill-file override the `_preamble.md` §1 derivation rule keys on; see [_shared/context-loading](/_shared/context-loading.md)) and hosts the canonical **Index Ownership** model (its `### Index Ownership` section — see Index Ownership Model below).

> **Distinct from pipeline hydrate**: This file documents the standalone `/docs-hydrate-memory` skill. The `/fab-continue` pipeline hydrate stage (which advances a change from `review: done` to `hydrate: done` and updates `docs/memory/` from the change's spec/plan) is documented in [execution-skills](/pipeline/execution-skills.md) under "Hydrate Behavior (via `/fab-continue`)". The pipeline hydrate stage reads `## Deletion Candidates` from `plan.md` informationally (per execution-skills Hydrate Behavior Step 3) — see that file for the authoritative behavior.

## Requirements

### Standalone Hydrate Skill

The system provides `/docs-hydrate-memory [sources...|folders...]` as an independent skill containing hydration and generation logic. It is defined in `$(fab kit-path)/skills/docs-hydrate-memory.md` and is auto-discovered by `sync/2-sync-workspace.sh`'s `*.md` glob pattern.

### Argument-Driven Mode Selection

The skill determines its operating mode from argument type:

| Argument type | Detection rule | Mode |
|---|---|---|
| `backfill` keyword | First argument is the literal `backfill`, OR the invocation is a `/docs-reorg-memory` dispatch naming the operation | Backfill (re-scan existing tree for files missing `description:`) |
| No arguments | Argument list is empty | Generate (scan from project root) |
| URL | Matches `notion.so`, `notion.site`, `linear.app`, or `http(s)://` | Ingest |
| Markdown file | Path ends with `.md` | Ingest |
| Folder | Path resolves to an existing directory | Generate |

Backfill is checked first — it is reached only by the explicit `backfill` keyword or a reorg dispatch, so it never collides with bare ingest/generate routing. Mixed-mode invocations (e.g., a URL and a folder) SHALL be rejected with an error.

### Ingest Mode Behavior

When arguments route to ingest mode:

- Fetches/reads each source independently
- Analyzes content and maps to domains
- Creates or merges memory files in `docs/memory/{domain}/`, authoring each file's **FKF frontmatter** — both `type: memory` (the constant FKF type, [fkf.md](../../specs/fkf.md) §3.1) and a curated `description:` one-liner (§3.2). A created file carries Overview / Requirements / Design Decisions sections — **no `## Changelog`** (FKF §3.3 removes per-file changelog tables; change history lives in the generated per-folder `log.md`, §6). Any memory↔memory cross-link in the body uses the **bundle-relative `/...` form** (§7, resolved from `docs/memory/`); links out of the bundle (sources, specs, URLs) stay repo-relative/absolute-URL (8fr5)
- For a new domain (or sub-domain), creates the `index.md` **stub** — only the `description:` frontmatter one-liner — **before** `fab memory-index` runs (see Index Ownership Model below)
- Regenerates all indexes (root + every domain + every sub-domain) mechanically via `fab memory-index` — never by hand-editing rows
- Multiple sources are processed in a single pass; `fab memory-index` runs once at the end

### Generate Mode Behavior

When arguments route to generate mode (no arguments or folder paths), the skill scans the codebase for undocumented areas, presents an interactive gap report, and generates structured memory files. See [hydrate-generate](/memory-docs/hydrate-generate.md) for full requirements.

### Backfill Mode Behavior (5ewp)

Backfill migrates an **existing** hand-curated `docs/memory/` tree (typically pre-fab-kit) to the convention `fab memory-index` depends on: each topic file leads with a `description:` frontmatter line. Without it, the generator — which reads descriptions exclusively from frontmatter (`frontmatter.Field(path, "description")`) — renders `—` for every row, wiping curated descriptions on the first regen. Backfill is the one-time fix. It is invoked directly (`/docs-hydrate-memory backfill`) or dispatched by `/docs-reorg-memory` as the second step of its compatibility orchestration (see [templates](/memory-docs/templates.md) § Memory Tree Shape).

- **Pure frontmatter operation, body-preserving**: backfill only prepends/edits the leading FKF frontmatter (`type: memory` + `description:` — it stamps `type: memory` alongside the synthesized `description:` so a backfilled file is FKF-conforming, [fkf.md](../../specs/fkf.md) §2 item 2; 8fr5) and creates missing `description:`-only index stubs. It NEVER touches a file's body (preserved byte-for-byte) — in particular it does **not** strip any existing `## Changelog` section a pre-fab-kit file still carries (stopping new changelog writes was 8fr5; the bulk strip of fab-kit's own per-file tables landed separately in the **oovf** cutover, FKF change 4/4, which also seeded their history into per-folder `log.seed.md` — but backfill itself stays strictly body-preserving and strips nothing). It also does NOT detect/relocate tombstone rows, flatten custom groupings, or move files — those structural concerns belong to `/docs-reorg-memory` (the restructure/author seam: reorg detects + relocates the one mechanical file; backfill synthesizes per-file descriptions).
- **Independent re-scan, no caller manifest**: backfill walks `docs/memory/` itself to find every topic file (non-`index.md` `.md`) lacking a `description:` field — it does not receive a file list from its caller. This holds for both forms: the direct-user invocation and the reorg dispatch (reorg's prompt names the operation — "backfill this tree" — not the files; assumption #9). A file with no frontmatter, or frontmatter without a `description:` key, counts as missing (the same `frontmatter.Field` semantics the generator uses). The walk is the loose, idempotent seam between the two independently-invocable skills.
- **Synthesis source**: for each discovered file, synthesize a one-line `description:` from the file's own content (Overview / first section / H1). Where an existing curated index row maps file-by-file to the file, **prefer the curated row text** — it is higher quality than re-synthesis.
- **Idempotent skip**: files that already carry a `description:` are skipped — backfill never overwrites an existing one, so a second pass over an already-converted tree is a no-op (no frontmatter rewrites, no body changes, byte-stable index — Constitution III).
- **Stub-before-index** (Index Ownership Model below): backfill creates any missing domain/sub-domain `description:`-only `index.md` stub the same way ingest/generate do, so `fab memory-index` has the domain description to read.
- **Caller-aware regen deferral**: backfill learns its caller from the dispatch prompt. When dispatched by reorg, it does NOT run `fab memory-index` (reorg runs it once at the end of its orchestration — the single regen for the whole run). When invoked directly by a user, it runs `fab memory-index` as the final step like the other modes. The direct-user form does NOT detect/relocate tombstones (assumption #11) — that stays reorg-only.

> **Reorg orchestration seam (5ewp; mechanical detection glwc)**: `/docs-reorg-memory` is the single front door for the once-per-repo "make an existing tree fab-kit-compatible" task. It detects the compatibility gap (missing frontmatter, tombstone rows, custom groupings) **mechanically by calling `fab memory-index --check --json`** (glwc — the older-binary prose-eyeballing fallback is retained), surfaces it in its approve-before-mutate findings report, and on approval orchestrates: relocate confirmed tombstones → `docs/memory/_shared/removed-domains.md` (the one mechanical file reorg authors) → dispatch this skill's backfill mode as a general-purpose sub-agent (no manifest; defer-regen signal) → rebalance + a single `fab memory-index`. Per-file *synthesis* lives here in backfill; reorg's job stays structural. See [templates](/memory-docs/templates.md) § Memory Tree Shape for reorg's side of the seam.

### Refuse-Before-Regen Guard (glwc)

The skill carries a **refuse-before-regen guard** at every `fab memory-index` regeneration site (ingest/generate/backfill regen steps): before regenerating, consult `fab memory-index --check`; on **exit 2** (destructive loss — a curated description would regenerate to `—`, a tombstone row would drop, or a custom grouping would flatten), **refuse to regenerate** and surface the pointer `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.` (`/docs-reorg-memory` is the orchestrator for all three tier-2 categories — it relocates tombstone rows itself and dispatches *this* skill's backfill mode for descriptions; backfill alone does not relocate tombstones). This is the **primary pre-fab-kit-tree entry point**, so the guard protects the *first* regen of a legacy tree — the exact silent-loss scenario, reachable by any path that does not go through reorg.

- The guard is a **no-op on born-compatible fab-kit trees**: they are provably always exit 0/1, never 2, so the guard never fires (do NOT mistake it for dead code or remove it). It only ever fires on a pre-fab-kit tree reached via ingest/generate before the tree has been backfilled.
- The loss logic lives entirely in Go (`fab memory-index --check`, tiered exit codes 0/1/2) — this guard is a one-line exit-code consult, not duplicated detection. The same primitive is consulted at the other two regen sites: `/docs-reorg-memory` (via its `--check --json` compatibility detection above) and `/fab-continue`'s pipeline hydrate stage (defense-in-depth — see [execution-skills](/pipeline/execution-skills.md) § Hydrate Behavior).
- Backfill mode itself never destroys content (it only adds frontmatter), so once backfill has run, *its own* terminal regen finds the guard already a no-op.

### Prerequisite

`/docs-hydrate-memory` requires `docs/memory/` to exist. If missing, it aborts with: "docs/memory/ not found. Run /fab-setup first to create the memory directory."

### Idempotent Hydration

Safe to run repeatedly with the same sources:
- New requirements from the source are added
- Existing requirements are updated if source content changed
- Manually-added content in memory files is preserved
- No duplication of requirements on re-hydration

### Index Ownership Model (defined once — d9rs)

The skill file's `### Index Ownership` section states the ownership model **once**, and every index-touching skill follows it:

- Index files (`index.md` at the root, domain, and sub-domain tiers) are **generated artifacts** — `fab memory-index` is their single writer. Generated content (file rows, `## Sub-Domains` tables, "Last Updated" cells) is never hand-edited.
- The **one hand-curated field** is the `description:` frontmatter — on topic files and on domain/sub-domain index files alike.
- When a new domain or sub-domain is created, its `index.md` **stub** — only the `description:` frontmatter one-liner, nothing else — is created **BEFORE** `fab memory-index` runs; the command fills in the generated body and round-trips the description.

Both modes of this skill follow the model, and d9rs propagated it to the other index-touching surfaces: `docs-reorg-memory` Step 5.3/5.4 (the former contradiction — Step 5.3 hand-editing a sub-domain index that Step 5.4 both generated and forbade editing — is resolved via the same stub-before-index pattern; see [templates](/memory-docs/templates.md) § Memory Tree Shape) and `/fab-continue`'s hydrate step, whose index-regeneration tier wording now names all three tiers (root, domain, sub-domain).

### Index Maintenance

Every hydration operation regenerates the navigable indexes **mechanically** via `fab memory-index` — the skill never hand-edits index rows:
- **Top-level** (`docs/memory/index.md`): domains-only — `| Domain | Description |`. The legacy inlined per-file "Memory Files" column was dropped (tciy); per-domain descriptions come from each domain `index.md`'s `description:` frontmatter (round-tripped by the generator).
- **Domain-level** (`docs/memory/{domain}/index.md`): file rows — `| File | Description | Last Updated |`. Each row's Description is read from the topic file's `description:` frontmatter; "Last Updated" is git-stamped from ONE batched `git -c core.quotepath=off log --date=short --format=%x00%ad --name-only -- docs/memory` pass (newest-first; the first date seen per path wins, keyed relative to the git top-level — output-equivalent to the old per-file `git log -1 --date=short` defaults, which is retained solely as the fallback when the batched call fails), degrading to `—` for uncommitted files; never hand-stamped (pw3k).
- **Sub-domain-level** (`docs/memory/{domain}/{sub-domain}/index.md`): same file-row contract as a domain index, generated for every sub-domain directory holding ≥1 non-index `.md` (sx7a; the skill's tier descriptions name all three tiers since d9rs).
- The command is the single writer of all index levels — output is byte-stable / idempotent, so re-running produces no diff and any post-merge conflict auto-resolves by re-running `fab memory-index`.
- Memory writers MUST author the `description:` frontmatter on every new/modified topic file so the regenerated index has content.
- Formats follow `docs/specs/templates.md`

## Design Decisions

### Extract Hydration from Init into Standalone Skill
**Decision**: Move Phase 2 verbatim from `fab-init.md` into `fab-hydrate.md`, then remove it from init.
**Why**: Preserves tested hydration logic; single source of truth. Clean separation — init = structure, hydrate = content.
**Rejected**: Rewriting hydration from scratch in the new skill — risks introducing bugs and inconsistencies.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Hydrate Requires docs/memory/ to Exist
**Decision**: `/docs-hydrate-memory` checks for `docs/memory/` and aborts if missing, directing user to run `/fab-setup` first.
**Why**: Keeps the dependency clear — init creates structure, hydrate populates it.
**Rejected**: Auto-creating `docs/memory/` in hydrate — would blur the separation of concerns.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Memory Index Maintenance is a Mechanical `fab memory-index` Call
**Decision**: The hydrate skill regenerates `docs/memory/` indexes by invoking the deterministic `fab memory-index` Go subcommand, not by hand-editing index rows in skill instructions.
**Why**: Hand-maintained per-row index cells (`description` + `Last Updated`) were the dominant merge-conflict and drift source — they get rewritten on nearly every memory edit. A generated, byte-stable index removes the hand-edit entirely, so two branches can never produce conflicting hand-edits to the same row, and any residual textual conflict auto-resolves by re-running the command. The render is a pure function of folder contents + `description:` frontmatter + git dates, mirroring the established `internal/prmeta` Render/Gather pattern.
**Rejected**: Markdown skill instructions for index updates (the prior approach) — they silently drift (the old root roster listed 18 files when 20+ existed; hand-stamped dates were already wrong). A bespoke bash table-parser was also rejected earlier as brittle; the deterministic Go helper is admitted by the constitution (cf. `prmeta`/`impact`/`score`) and is fully unit-testable.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context (original inline-instruction design); *Superseded by*: 260607-tciy-memory-tree-shape-rebalance (mechanical `fab memory-index`)
