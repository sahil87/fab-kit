---
type: memory
description: "The /docs-hydrate-memory skill — argument routing, three modes (ingest/generate/backfill), the writer contract, index-ownership model. Ingest/generate rewrite each topic section to current truth (no change-id headings, rationale in four-field Design Decisions) then self-check before regen; backfill is body-preserving and self-check-exempt. Refuse-before-regen guard keys on fab memory-index --check exit 2, distinct from the exit-1 blocking class (malformed/change-id/over-cap description)."
---
# Hydrate

**Domain**: memory-docs

## Overview

`/docs-hydrate-memory [sources...|folders...|backfill]` is a standalone skill that operates in three modes: **ingest mode** (fetching URLs or reading `.md` files into `docs/memory/`), **generate mode** (scanning the codebase for undocumented areas and producing structured memory files), and **backfill mode** (re-scanning an existing tree to add missing `description:` frontmatter — the pre-fab-kit migration path, see Backfill Mode below). Mode is determined automatically by argument type — ingest/generate by argument type, backfill by the explicit `backfill` keyword or a `/docs-reorg-memory` dispatch. It requires `docs/memory/` to exist (created by `/fab-setup`). See [hydrate-generate](/memory-docs/hydrate-generate.md) for full generate mode requirements. The skill file carries an explicit `## Context Loading` override section (d9rs) (it skips the always-load layer — the skill-file override the `_preamble.md` §1 derivation rule keys on; see [_shared/context-loading](/_shared/context-loading.md)) and hosts the canonical **Index Ownership** model (its `### Index Ownership` section — see Index Ownership Model below).

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
- Creates or merges memory files in `docs/memory/{domain}/`, authoring each file's **FKF frontmatter** — both `type: memory` (the constant FKF type, [fkf.md](../../specs/fkf.md) §3.1) and a curated `description:` one-liner (§3.2 — a single-line scalar **capped at 500 characters/runes** on the quote-stripped value and **free of change-ids** (neither a trailing `— xu0k`-style suffix nor a `(d9rs)`-style citation); it is a routing signal, so detail and provenance citations belong in the file body, never the description). A created file carries Overview / Requirements / Design Decisions sections — **no `## Changelog`** (FKF §3.3 removes per-file changelog tables; change history lives in the generated per-folder `log.md`, §6). The body states **current truth in present tense** — no transition narration ("renamed X→Y in {id}", "supersedes {id}'s claim", "was `old.value`"); superseded statements are removed, not narrated (§3.3). **Headings carry no change-ids** — a heading names its topic (`## Dispatch States`), never a change (`### Dispatch States (xu0k)`); change-ids stay citation-only in body text. Any *why* / rejected alternative goes into a `## Design Decisions` entry in the four-field shape (**Decision** / **Why** / **Rejected** / *Introduced by*), never inline narration; the changelog-bullet shape (`- **{change-id} — retired X**`) is banned inside `## Design Decisions` (that is `log.md`'s job, §6). Any memory↔memory cross-link in the body uses the **bundle-relative `/...` form** (§7, resolved from `docs/memory/`); links out of the bundle (sources, specs, URLs) stay repo-relative/absolute-URL
- On a merge into an existing file, after any body edit **re-checks the `description:` still routes** — one line, ≤500 chars, change-id-free (a body edit can leave it stale, FKF §3.2)
- For a new domain (or sub-domain), creates the `index.md` **stub** — only the `description:` frontmatter one-liner — **before** `fab memory-index` runs (see Index Ownership Model below)
- **Post-hydrate self-check** (a numbered Step 3.5, before the index regen): re-reads every file created or merged *this run* and strips any transition phrasing just introduced — no "renamed / now / previously / no longer / was `old.value`" narration, no change-keyed delta paragraph left below an older paragraph on the same topic, no change-ids in headings — and confirms each touched `description:` still routes. A self-review of this run's own writes, **not** a corpus sweep (draining pre-existing debt across the tree is `/docs-distill-memory`'s job)
- Regenerates all indexes (root + every domain + every sub-domain) mechanically via `fab memory-index` — never by hand-editing rows
- Multiple sources are processed in a single pass; the self-check then `fab memory-index` run once at the end

### Generate Mode Behavior

When arguments route to generate mode (no arguments or folder paths), the skill scans the codebase for undocumented areas, presents an interactive gap report, and generates structured memory files. Generated bodies follow the same writer rules ingest applies at creation — present-truth prose, **no change-ids in headings**, any *why* / rejected alternative as a four-field `## Design Decisions` entry (never inline narration, never a changelog bullet), change-id-free `description:` — and generate mode runs the same **post-hydrate self-check** (Step 3.5, on the files generated this run) before the index regen. See [hydrate-generate](/memory-docs/hydrate-generate.md) for full requirements.

### Backfill Mode Behavior

Backfill (5ewp) migrates an **existing** hand-curated `docs/memory/` tree (typically pre-fab-kit) to the convention `fab memory-index` depends on: each topic file leads with a `description:` frontmatter line. Without it, the generator — which reads descriptions exclusively from frontmatter (`frontmatter.Field(path, "description")`) — renders `—` for every row, wiping curated descriptions on the first regen. Backfill is the one-time fix. It is invoked directly (`/docs-hydrate-memory backfill`) or dispatched by `/docs-reorg-memory` as the second step of its compatibility orchestration (see [templates](/memory-docs/templates.md) § Memory Tree Shape).

- **Pure frontmatter operation, body-preserving**: backfill only prepends/edits the leading FKF frontmatter (`type: memory` + `description:` — it stamps `type: memory` alongside the synthesized `description:` so a backfilled file is FKF-conforming, [fkf.md](../../specs/fkf.md) §2 item 2) (8fr5) and creates missing `description:`-only index stubs. It NEVER touches a file's body (preserved byte-for-byte) — in particular it does **not** strip any existing `## Changelog` section a pre-fab-kit file still carries. It also does NOT detect/relocate tombstone rows, flatten custom groupings, or move files — those structural concerns belong to `/docs-reorg-memory` (the restructure/author seam: reorg detects + relocates the one mechanical file; backfill synthesizes per-file descriptions).
- **Exempt from the §3.3 body rules and the post-hydrate self-check**: the ingest/generate merge-time body rules (heading change-id ban, rationale → Design Decisions, changelog-bullet ban) and the Step 3.5 self-check are body-editing operations, so they do **not** apply to backfill — its body-preserving contract forbids body edits. Backfill applies only the change-id-free `description:` rule of §3.2, which it already does.
- **Independent re-scan, no caller manifest**: backfill walks `docs/memory/` itself to find every topic file (non-`index.md` `.md`) lacking a `description:` field — it does not receive a file list from its caller. This holds for both forms: the direct-user invocation and the reorg dispatch (reorg's prompt names the operation — "backfill this tree" — not the files; assumption #9). A file with no frontmatter, or frontmatter without a `description:` key, counts as missing (the same `frontmatter.Field` semantics the generator uses). The walk is the loose, idempotent seam between the two independently-invocable skills.
- **Synthesis source**: for each discovered file, synthesize a one-line `description:` from the file's own content (Overview / first section / H1), keeping it a curated one-liner **within the 500-character cap and free of change-ids** (FKF §3.2 — a routing signal, not a body summary or provenance record). Where an existing curated index row maps file-by-file to the file, **prefer the curated row text** — it is higher quality than re-synthesis (strip any change-ids from a reused row).
- **Idempotent skip**: files that already carry a `description:` are skipped — backfill never overwrites an existing one, so a second pass over an already-converted tree is a no-op (no frontmatter rewrites, no body changes, byte-stable index — Constitution III).
- **Stub-before-index** (Index Ownership Model below): backfill creates any missing domain/sub-domain `description:`-only `index.md` stub the same way ingest/generate do, so `fab memory-index` has the domain description to read.
- **Caller-aware regen deferral**: backfill learns its caller from the dispatch prompt. When dispatched by reorg, it does NOT run `fab memory-index` (reorg runs it once at the end of its orchestration — the single regen for the whole run). When invoked directly by a user, it runs `fab memory-index` as the final step like the other modes. The direct-user form does NOT detect/relocate tombstones (assumption #11) — that stays reorg-only.

> **Reorg orchestration seam (5ewp)**: `/docs-reorg-memory` is the single front door for the once-per-repo "make an existing tree fab-kit-compatible" task. It detects the compatibility gap (missing frontmatter, tombstone rows, custom groupings) **mechanically by calling `fab memory-index --check --json`** (glwc) (the older-binary prose-eyeballing fallback is retained), surfaces it in its approve-before-mutate findings report, and on approval orchestrates: relocate confirmed tombstones → `docs/memory/_shared/removed-domains.md` (the one mechanical file reorg authors) → dispatch this skill's backfill mode as a general-purpose sub-agent (no manifest; defer-regen signal) → rebalance + a single `fab memory-index`. Per-file *synthesis* lives here in backfill; reorg's job stays structural. See [templates](/memory-docs/templates.md) § Memory Tree Shape for reorg's side of the seam.

### Refuse-Before-Regen Guard

The skill carries a **refuse-before-regen guard** (glwc) at every `fab memory-index` regeneration site (ingest/generate/backfill regen steps): before regenerating, consult `fab memory-index --check`; on **exit 2** (destructive loss — a curated description would regenerate to `—`, a tombstone row would drop, or a custom grouping would flatten), **refuse to regenerate** and surface the pointer `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.` (`/docs-reorg-memory` is the orchestrator for all three tier-2 categories — it relocates tombstone rows itself and dispatches *this* skill's backfill mode for descriptions; backfill alone does not relocate tombstones). This is the **primary pre-fab-kit-tree entry point**, so the guard protects the *first* regen of a legacy tree — the exact silent-loss scenario, reachable by any path that does not go through reorg.

- The guard is a **no-op on born-compatible fab-kit trees**: they are provably always exit 0/1, never 2, so the guard never fires (do NOT mistake it for dead code or remove it). It only ever fires on a pre-fab-kit tree reached via ingest/generate before the tree has been backfilled.
- The loss logic lives entirely in Go (`fab memory-index --check`, tiered exit codes 0/1/2) — this guard is a one-line exit-code consult, not duplicated detection. The same primitive is consulted at the other two regen sites: `/docs-reorg-memory` (via its `--check --json` compatibility detection above) and `/fab-continue`'s pipeline hydrate stage (defense-in-depth — see [execution-skills](/pipeline/execution-skills.md) § Hydrate Behavior).
- **The blocking content class is a SEPARATE signal the guard does NOT key on.** Four `description:`/frontmatter signatures floor `fab memory-index --check` at exit ≥ 1 **independent of drift**: an unclosed frontmatter fence, a `description:` that starts with a quote but fails quote-stripping (e.g. the glued-fence `"…text…"---`), a **registry-gated change-id** in `description:` (the enforced FKF §3.2 ban), and a **gross over-cap** `description:` over 1000 characters. None is a tier-2 destructive loss — so the refuse-before-regen guard (which fires only on **exit == 2**) does not treat any of them as a reason to refuse, and each routes to a **fix-the-file remediation, not the `→ /docs-reorg-memory` pointer** (a reorg repairs neither source corruption nor a bad routing signal). The finding still surfaces as a blocking `--check` failure at review-pr / CI (exit ≥ 1). The advisory warnings — the 501–1000-char `description:` trim nag, plus the per-topic-file debt meters (narration density, file size, `_unsorted/` non-empty, broken bundle-relative links) — are likewise **non-blocking** (they never affect `--check`). See [pipeline/schemas.md](/pipeline/schemas.md) § Blocking Content Class for the exit-code/tier contract.
- Backfill mode itself never destroys content (it only adds frontmatter), so once backfill has run, *its own* terminal regen finds the guard already a no-op.

### Prerequisite

`/docs-hydrate-memory` requires `docs/memory/` to exist. If missing, it aborts with: "docs/memory/ not found. Run /fab-setup first to create the memory directory."

### Idempotent Hydration (merge as current truth)

Safe to run repeatedly with the same sources. When a target file already exists, the merge is a **present-truth rewrite keyed on the topic/section**, not a change-keyed append (FKF §3.3):
- New requirements from the source are added
- The section already documenting a topic is **rewritten to state current truth** — superseded statements are removed, not narrated (no "renamed X→Y in {id}", no "was `old.value`"); body provenance stays citation-only (a trailing `(change-id)` / the `*Introduced by*` field), and **headings carry no change-ids** (a heading names its topic, never a change)
- Any *why* / rejected alternative lands in a `## Design Decisions` entry in the four-field shape (**Decision** / **Why** / **Rejected** / *Introduced by*), never inline narration; the changelog-bullet shape (`- **{change-id} — retired X**`) is banned inside `## Design Decisions`
- After any body edit, the `description:` is re-checked to confirm it still routes (one line, ≤500 chars, change-id-free, §3.2)
- Manually-added content in memory files is preserved
- No duplication of requirements on re-hydration — a re-run rewrites the same section to the same current truth

### Index Ownership Model (defined once)

The skill file's `### Index Ownership` section states the ownership model **once** (d9rs), and every index-touching skill follows it:

- Index files (`index.md` at the root, domain, and sub-domain tiers) are **generated artifacts** — `fab memory-index` is their single writer. Generated content (file rows, `## Sub-Domains` tables) is never hand-edited.
- The **one hand-curated field** is the `description:` frontmatter — on topic files and on domain/sub-domain index files alike.
- When a new domain or sub-domain is created, its `index.md` **stub** — only the `description:` frontmatter one-liner, nothing else — is created **BEFORE** `fab memory-index` runs; the command fills in the generated body and round-trips the description.

Both modes of this skill follow the model, and every other index-touching surface follows it too (d9rs): `docs-reorg-memory` Step 5.3/5.4 use the same stub-before-index pattern (Step 5.3 creates the stub, Step 5.4 generates — no hand-editing of generated content; see [templates](/memory-docs/templates.md) § Memory Tree Shape), and `/fab-continue`'s hydrate step's index-regeneration tier wording names all three tiers (root, domain, sub-domain).

### Index Maintenance

Every hydration operation regenerates the navigable indexes **mechanically** via `fab memory-index` — the skill never hand-edits index rows:
- **Top-level** (`docs/memory/index.md`): domains-only — `| Domain | Description |`. There is no inlined per-file "Memory Files" column (tciy); per-domain descriptions come from each domain `index.md`'s `description:` frontmatter (round-tripped by the generator).
- **Domain-level** (`docs/memory/{domain}/index.md`): file rows — `| File | Description |`. Each row's Description is read from the topic file's `description:` frontmatter; the index carries no dates (content-only, branch-independent) (ugde). Recency-at-a-glance lives in the per-folder `log.md`, which consumes the batched `git log` pass (see Per-folder `log.md` below).
- **Sub-domain-level** (`docs/memory/{domain}/{sub-domain}/index.md`): same file-row contract as a domain index, generated for every sub-domain directory holding ≥1 non-index `.md` (sx7a); the skill's tier descriptions name all three tiers (d9rs).
- **Per-folder `log.md`** (bmzo): the same `fab memory-index` call also (re)generates each folder's C-lite change log. The index tiers are a pure function of folder contents + frontmatter (content-only, no git dates) (ugde); the `log.md` files regenerate **append-only** under the **freeze-on-write** model (tayp), and `log.md` is the sole consumer of the batched `git log` pass for its dates — the existing `log.md` is authoritative and write-once, so regeneration reads it back and appends only new `(file-base, change-id)` entries rather than re-projecting live git from scratch (and a new unattributable commit is frozen, not re-projected). This keeps a hydration run from churning unrelated `log.md` entries on a history rewrite. See [templates](/memory-docs/templates.md) § Generated `log.md` and [pipeline/schemas.md](/pipeline/schemas.md) § Freeze-on-Write `log.md` Generation for the full contract.
- The command is the single writer of all index levels **and `log.md` files** — both are byte-stable / idempotent, so re-running produces no diff and any post-merge conflict auto-resolves by re-running `fab memory-index`.
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
**Why**: Hand-maintained per-row index cells (`description`, and formerly a `Last Updated` date) were the dominant merge-conflict and drift source — they get rewritten on nearly every memory edit. A generated, byte-stable index removes the hand-edit entirely, so two branches can never produce conflicting hand-edits to the same row, and any residual textual conflict auto-resolves by re-running the command. The render is a pure function of folder contents + `description:` frontmatter (content-only (ugde) — a `git log` projection is HEAD/branch-relative and so not idempotent, so there is no date column), mirroring the established `internal/prmeta` Render/Gather pattern.
**Rejected**: Markdown skill instructions for index updates (the prior approach) — they silently drift (the old root roster listed 18 files when 20+ existed; the former hand-stamped dates were already wrong). A bespoke bash table-parser was also rejected earlier as brittle; the deterministic Go helper is admitted by the constitution (cf. `prmeta`/`impact`/`score`) and is fully unit-testable.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context (original inline-instruction design); *Updated by*: 260607-tciy-memory-tree-shape-rebalance (mechanical `fab memory-index`)

### Backfill Is Strictly Body-Preserving — Changelog Stripping Lives Elsewhere
**Decision**: Backfill only prepends/edits leading FKF frontmatter; it never strips a `## Changelog` section, even from a pre-fab-kit file that still carries one.
**Why**: Backfill is the frontmatter-only migration seam. Stripping bodies is a separate, riskier operation that belongs to the bulk FKF cutover (which also seeds the stripped history into per-folder `log.seed.md`), not to a mechanical frontmatter pass.
**Rejected**: Having backfill strip changelogs too — couples two independent migrations and risks losing history that has no seed yet.
*Introduced by*: 260614-8fr5
