# FKF — Fab Knowledge Format (v0.1)

> **What this is.** FKF is the format fab-kit uses for the `docs/memory/` knowledge tree:
> a directory bundle of markdown files with YAML frontmatter, plus generated index and log
> files. FKF is a **profile of [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)**
> (Open Knowledge Format, GoogleCloudPlatform/knowledge-catalog): every FKF bundle is a
> conforming OKF bundle, and FKF additionally *requires* and *fixes* a handful of things OKF
> leaves open. A generic OKF consumer can read an FKF bundle; fab's tooling enforces more than
> OKF requires.
>
> **Scope: `docs/memory/` only.** FKF governs the post-implementation memory tree. It does **not**
> apply to `docs/specs/` — specs remain human-curated, frontmatter-free, and free-form per
> a fab-kit design principle (specs are human-curated). See [§9 Non-Scope](#9-non-scope-docsspecs).

> **Shipped normative extract.** This file is the dev-repo design doc (rationale + history). The
> **normative subset** an agent must follow (§2/§3/§5/§6/§7/§8, original anchors preserved) is
> shipped to the kit cache as `src/kit/reference/fkf.md`, reachable in every user repo via
> `$(fab kit-path)/reference/fkf.md`; that is the file deployed skills cite. **Any change to FKF
> normative rules MUST update both files** so they cannot silently diverge.

---

## 1. Relationship to OKF

FKF is a **profile**: a base format plus a set of additional constraints. The split:

| Concern | OKF v0.1 says | FKF additionally requires |
|---------|---------------|---------------------------|
| Substrate | dir tree of markdown + YAML frontmatter | same |
| Directory organization | producer's choice | maps to fab's `{domain}/{sub-domain}/` (§4) |
| Required frontmatter | `type` (non-empty) | `type` **fixed to the constant `memory`** (§3.1) |
| `description` frontmatter | *recommended* | **required**, curated one-liner (§3.2) |
| Body | free-form; conventional headings | conventional headings *recommended, not mandated* (§3.3) |
| `index.md` | may be hand-written, auto-generated, or synthesized | **generated, single-writer, byte-stable, hand-edit forbidden** (§5) |
| `log.md` | optional, hand-appended, prose entries | **generated** from git + per-change summaries (C-lite, §6) |
| Cross-links | absolute (bundle-relative) recommended, or relative | **bundle-relative `/...` required** (§7) |
| Versioning | `okf_version` in root `index.md` | emit **`fkf_version`** instead (§8) |

FKF is **stricter** than OKF on indexes and links (it forbids what OKF merely discourages) and
**narrower** on `type` and frontmatter (it fixes/requires what OKF leaves to the producer). Both
directions keep FKF inside OKF conformance — OKF explicitly permits generated indexes, required
custom keys, and additional frontmatter.

---

## 2. Conformance

A `docs/memory/` tree conforms to **FKF v0.1** if all of the following hold:

1. Every non-reserved `.md` file carries a parseable YAML frontmatter block.
2. Every such block contains `type: memory` and a non-empty `description`.
3. Reserved filenames — `index.md` and `log.md` — follow their generated structures (§5, §6) and
   are written only by `fab memory-index`.
4. Cross-links between memory files use the bundle-relative form (§7).

Items 1–2 are the OKF conformance floor (specialized: `type` is fixed, `description` is promoted
to required). Items 3–4 are FKF's added strictness. As in OKF, consumers SHOULD degrade
gracefully — a missing optional body section, an unknown extra frontmatter key, or a stale
"Last Updated" cell does not make a file non-conforming.

---

## 3. Concept Documents (memory files)

A memory file = a YAML frontmatter block + a markdown body, at
`docs/memory/{domain}/{name}.md` or `docs/memory/{domain}/{sub-domain}/{name}.md`.

### 3.1 `type` (required, constant)

```yaml
type: memory
```

`type` is OKF's sole required field — its machine-routing discriminator. fab's memory files are
**homogeneous** (every file is "a documented area of system behavior"), so `type` carries no
distinguishing signal and is **fixed to the constant `memory`**. The value is stamped by tooling
(the memory-file template and every memory writer), never hand-curated — so "required" costs the
author nothing.

> **Why a constant, not sub-types.** A per-file kind (`requirements` vs `design-decision` vs
> `reference`) was rejected: a single memory file legitimately *mixes* requirements, design
> decisions, and history in one document, so a per-file type would misrepresent it. The
> organizing axis fab actually uses is the **domain** (the folder), not a `type`. If a genuinely
> distinct second kind of memory document ever appears, FKF v0.2 may widen the `type` vocabulary
> — driven by a real distinction, not anticipated up front.

### 3.2 `description` (required, curated)

```yaml
description: "One-line summary used by the generated domain-index row."
```

OKF *recommends* `description`; FKF **requires** it, because it is **load-bearing**: the generated
domain index reads each file's row Description from this field, and the always-load context layer
routes on it. It is the one hand-curated frontmatter field — authored by every memory writer
(hydrate, `/docs-hydrate-memory`, `docs-reorg-memory`) and kept accurate on every edit.

Co-locating the description with the file (rather than in the index) is deliberate — the
**Starlight lesson**: editing a description never touches the hot, churn-prone index row. It
cannot be auto-derived from the H1/Overview without loss (Overview prose contains literal `|`
pipes that break index tables, and an extracted first sentence degrades the routing signal).

### 3.3 Body (conventional headings, recommended — not mandated)

The body is standard markdown. FKF adopts OKF's posture: **conventional headings are recommended
where they apply, not required**. A file is conforming without any particular section. The
conventional structure:

```markdown
---
type: memory
description: "One-line summary used by the generated domain-index row."
---
# {File Name}

**Domain**: {domain}

## Overview
<!-- 1-2 sentences describing what this file covers. -->

## Requirements
### Requirement: {Name}
{RFC 2119 text: MUST / SHALL / SHOULD / MAY}

#### Scenario: {Name}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {expected outcome}

## Design Decisions
### {Decision Title}
**Decision**: {chosen approach}
**Why**: {rationale}
**Rejected**: {alternative and why it was worse}
*Introduced by*: {change-name}
```

> **Why recommended, not mandated.** The pipeline's hydrate step *writes into* `## Requirements`
> and `## Design Decisions`, and review/intake *read from* them — so these headings remain the
> target shape and SHOULD be used wherever the content warrants. But a small reference-pointer
> file should not be forced to invent a GIVEN/WHEN/THEN scenario. FKF relaxes the former
> *MUST-have-these-sections* rule to *SHOULD-use-these-conventional-headings-where-they-apply*,
> which keeps the hydrate-writes / review-reads contract working without imposing ceremony on
> files that don't need it.

> **No `## Changelog` section.** Per-file changelog tables are **removed** in FKF — change history
> lives in the per-folder generated `log.md` (§6). This is the single biggest FKF divergence from
> the pre-FKF memory format; see [§6](#6-log-files-logmd) and the migration note in [§10](#10-adoption--migration).

### 3.4 Optional frontmatter

FKF neither requires nor forbids the other OKF-recommended fields (`title`, `tags`,
`timestamp`, `resource`). `resource` (a URI to an underlying asset) is typically **absent** —
memory files document *behavior*, not addressable assets. Per OKF, consumers MUST preserve
unknown frontmatter keys on round-trip and MUST NOT reject a file for carrying them.

---

## 4. Bundle Organization (domains = directories)

The `docs/memory/` tree **is** an OKF bundle directory tree. fab's existing structure maps
directly:

```
docs/memory/                         # bundle root
├── index.md                         # root index (generated, domains-only)
├── {domain}/
│   ├── index.md                     # domain index (generated)
│   ├── log.md                       # domain log (generated, FKF)
│   ├── {topic}.md                   # memory file
│   └── {sub-domain}/                # split cluster (≥8 cohesive files)
│       ├── index.md                 # sub-domain index (generated)
│       ├── log.md                   # sub-domain log (generated, FKF)
│       └── {topic}.md
```

- A **domain** is a top-level folder under `docs/memory/`.
- A **sub-domain** is one level deeper (`{domain}/{sub-domain}/{topic}.md` — **depth 3, the max**),
  created reactively by `docs-reorg-memory` when an over-wide domain holds a real cluster of
  ≥8 cohesive files. Un-split domains stay flat.
- Reserved domains `_shared/` (cross-cutting) and `_unsorted/` (staging) are width-exempt.

**Shape bounds (SHOULD guidance, advisory — never enforced):** ~12 topic files per folder (soft
upper bound; `fab memory-index` warns over it), ~5 lower floor before a sub-domain earns its own
index, max depth 3. These surface as non-fatal `fab memory-index` warnings and the
`docs-reorg-memory` Shape Report. Acting on them (split/merge/flatten) is `docs-reorg-memory`'s
job; the index command only detects and warns.

---

## 5. Index Files (`index.md`) — generated

Every directory holding ≥1 non-index `.md` carries a generated `index.md`. **All index tiers are
generated artifacts written solely by `fab memory-index`** — agents never hand-edit index rows.
The render is a pure function of (folder contents + each file's `description:` frontmatter + git
dates), so the output is **byte-stable / idempotent**: two branches cannot produce conflicting
hand-edits to the same row, and any residual textual conflict auto-resolves by re-running
`fab memory-index` post-merge.

> FKF is stricter than OKF here: OKF permits hand-written, auto-generated, *or*
> consumer-synthesized indexes. FKF **forbids** hand-editing — generation is the single writer.

Three tiers:

- **Root** (`docs/memory/index.md`) — **domains-only**: `| Domain | Description |`. Each domain
  row's Description is read from that domain `index.md`'s `description:` frontmatter
  (round-tripped by the generator). No inlined per-file column (it silently drifts).
- **Domain** (`docs/memory/{domain}/index.md`) — file rows: `| File | Description | Last Updated |`.
  Description from each topic file's frontmatter; "Last Updated" git-stamped (degrades to `—` when
  uncommitted / in a worktree / shallow clone), never hand-stamped. Carries its own
  `description:` frontmatter (the source for the root row). When sub-domains exist, appends a
  `## Sub-Domains` table (`| Sub-Domain | Description |`) — emitted **only when sub-domains
  exist**, so a flat domain index is byte-identical to a sub-domain-free one.
- **Sub-domain** (`docs/memory/{domain}/{sub-domain}/index.md`) — same file-row contract as a
  domain index; carries its own `description:` frontmatter (the source for the parent's
  `## Sub-Domains` row).

The **one curated input** to index generation is the `description:` frontmatter on topic files
and on domain/sub-domain index files. Everything else in an index is derived.

> **Stub-before-index.** When a new domain/sub-domain is created, its `index.md` **stub**
> (carrying only `description:` frontmatter) is written **before** `fab memory-index` runs; the
> command fills the generated body and round-trips the description. This is the Index Ownership
> model — it avoids the contradiction of one step hand-editing an index the next step both
> generates and forbids editing.

**`fkf_version` on the root index** — see §8.

---

## 6. Log Files (`log.md`) — generated (C-lite)

Each domain and sub-domain folder carries a generated `log.md` recording that folder's change
history. **`log.md` is a generated artifact written solely by `fab memory-index`** (same
single-writer, byte-stable discipline as `index.md`). It replaces the per-file `## Changelog`
tables that FKF removes from memory files (§3.3).

### 6.1 The C-lite model

`log.md` is assembled from **two sources**, neither of which any agent hand-edits:

1. **Git history**, keyed to the folder — the *when*, the *which file*, and the change ID. This is
   a projection of `git log` (the same date source the index uses), so it is always accurate and
   never conflicts.
2. **A per-change one-line summary** — the *what*, written **once** into the change's own
   `.status.yaml` `summary:` field (§6.3). Because each change touches only *its own*
   `.status.yaml`, the summary has **zero conflict surface**.

The generator joins them: for each commit touching a file in the folder, it emits one entry
under that commit's date, carrying the file, the change's `summary`, and the change ID.

> **Why C-lite, not a hand-appended log or a slug-only projection.** A hand-appended `log.md`
> (OKF's literal convention) just *relocates* the changelog merge-conflict from N memory files
> into the folder's `log.md` — two same-day changes in one domain still collide. A pure git
> projection (slug only, no summary) is conflict-free but loses the *what-changed* signal an
> agent needs for archaeology ("where did `cssMarker` go?") and migration-trajectory questions.
> C-lite keeps the descriptive line **and** stays conflict-free, because the line lives in the
> per-change `.status.yaml`, not in the shared `log.md`. The cost is one curated line per change
> and generator plumbing in `fab memory-index`.

### 6.2 Format

```markdown
# Log — {domain}
<!-- Generated by `fab memory-index` from git history + per-change summaries. Do not hand-edit. -->

## 2026-06-13
- **Update** [migrations](/distribution/migrations.md) — surfaces the optional `agent.tiers`
  per-stage-model override as a fully-commented config reference block; additive, no schema change. (260613-l3ja)

## 2026-06-12
- **Update** [migrations](/distribution/migrations.md) — drops the dead `stage_directives:` block. (260612-c5tr)
- **Update** [migrations](/distribution/migrations.md) — path-cite conformance; no migration shipped. (260612-tb6f)
```

- Entries are **date-grouped, newest first**; ISO `YYYY-MM-DD` date headings (OKF convention).
- Each entry: an optional leading bold verb (`**Update**` / `**Creation**` / `**Deprecation**` —
  OKF-conventional, derived from the change's `change_type` / removal markers), a bundle-relative
  link to the file that changed, the change's `summary`, and the `(change-id)` in parens.
- The descriptive line is **one line per change per file** — deliberately not the paragraph-length
  prose the pre-FKF changelog rows carried. Durable, long-form *why* belongs in the memory file's
  `## Design Decisions` section (it is durable design intent, not dated history); `log.md` carries
  the dated *what*.

### 6.3 The `summary:` source field

The per-change summary line lives in the change's `.status.yaml`:

```yaml
summary: "surfaces the optional agent.tiers per-stage-model override as a commented config block"
```

- Written once during the change (authored at hydrate, or carried from the intake), via the fab
  CLI — single-change-touched, so conflict-free.
- Read by `fab memory-index` when generating `log.md`.
- Adding this field is a `.status.yaml` **schema change** → it MUST ship with a migration file in
  `src/kit/migrations/` (per the project's data-migration rule). Absence degrades gracefully: a
  change with no `summary` projects with the change slug in place of the descriptive line.

### 6.4 Freeze-on-write generation

`log.md` generation is **freeze-on-write**: the existing `log.md` is **authoritative and
write-once**. A pure projection of *live* git history is not deterministic — squash-merge rewrites
commit subjects and counts, and branch-deletion makes the original commits unreachable — so
re-projecting from scratch on every run produces a different result per contributor and across time
(merely touching `docs/memory/` would churn dozens of unrelated `log.md` files and keep `--check`
permanently red, a Constitution III violation in practice). Freeze-on-write inverts the model from
*"`log.md` is a pure function of git+status; regenerate freely"* to *"the existing log is
authoritative; never re-derive what's already written."* It generalizes the `log.seed.md` mechanism
(§6, the seed-merge): after first write the whole `log.md` behaves like the seed — a frozen,
git-independent store the generator reads but never rewrites.

The regeneration flow:

1. **Read** the existing `log.md` and parse it back into entries (the inverse of the §6.2 render —
   the same grammar `log.seed.md` uses).
2. **Treat existing entries as immutable** — never reworded, re-dated, or dropped.
3. **Project** current git history, but use the projection only to discover **new** entries to append.
4. **Append only** entries whose identity is not already recorded.
5. **Re-render** (§6.2) over the merged `existing ∪ appended ∪ seed` set.

**Append/dedup key = `(file-base, change-id)`.** An *attributable* projected entry (its commit
resolves to a registered change-id) is appended only when no existing entry already records that
`(file-base, change-id)` pair. Re-running, or re-projecting after a squash that *preserved* the
change token, is a no-op. The git commit hash (`%H`) is deliberately **not** the key: squash +
branch-delete makes the hash unreachable — the exact operation being defended against — whereas the
change-id survives in the change folder name and the registry, independent of git.

**Unattributable commits are frozen, not re-projected.** A commit with no registry change-id
(migrations, docs-reorgs, direct-`main` edits, or a squash-merge whose branch token was dropped) has
no key to append on. Unattributable entries **already present** in `log.md` stay verbatim (frozen);
a **new** unattributable commit is **not** projected into the log after first write. *Accepted
tradeoff*: future migration/reorg commits leave no log trace — those are tooling commits, not
memory-domain history. Without this rule, a squashed unattributable commit (whose subject text
changed) would look like a *new* entry and be appended alongside the frozen old line — additive churn.

**Bootstrap is not a special mode.** The first run on a folder with no `log.md` is simply the first
append into an empty log (plus the `log.seed.md` seed-merge). Unattributable commits *are* projected
at bootstrap (and frozen on first write); there is no `--first-generation` flag (it would invite a
re-run that re-introduces churn). Bootstrap and every later run share one code path.

**`--rebuild` — the destructive escape hatch.** `fab memory-index --rebuild` discards the
accumulated frozen state and re-projects every `log.md` from current git (the pre-freeze behavior,
made explicit and opt-in: it re-projects unattributable commits too). It can rewrite or drop frozen
lines, so it is **destructive** — for a corrupted frozen log or a deliberate re-baseline, never the
default path.

**`--check` semantics under freeze-on-write.** `--check` compares the committed `log.md` against the
freeze-on-write **merge** (not a from-scratch projection):

- **PASS** when the committed log is a valid **superset** of the merge — it may carry frozen lines
  the live history no longer shows (the case byte-equality false-fails today).
- **FAIL** (benign drift) when a projected attributable `(file-base, change-id)` entry is **missing**
  from the committed log (someone forgot to regenerate-and-commit — the report surfaces the gap).
- **FAIL** (benign drift) when a frozen line was **hand-edited** in a render-unstable way (the
  single-writer discipline was violated; a clean reword that round-trips through the §6.2 grammar is,
  by design, accepted as the new frozen truth).

All `log.md` `--check` drift remains **benign (tier 1)** — `log.md` introduces **no** destructive-loss
(tier 2) category; the three index-only detectors (description / tombstone / grouping) never run on a
`log.md` target.

**Migration.** Existing projects carry `log.md` files generated under the old pure-projection model;
they re-baseline onto freeze-on-write via a one-time `fab memory-index --rebuild` + commit, shipped as
a migration in `src/kit/migrations/` (the standard upgrade ordering — new binary first, then
`/fab-setup migrations` — applies, and the migration's pre-check verifies the binary understands
`--rebuild` before rewriting anything). That re-baseline commit is the last churn the repo sees from
the non-determinism; every run afterward is append-only stable.

---

## 7. Cross-links — bundle-relative

Links between memory files use the **bundle-relative absolute** form: a path beginning with `/`,
interpreted relative to the bundle root (`docs/memory/`).

```markdown
See [migrations](/distribution/migrations.md) and [configuration](/_shared/configuration.md).
```

> FKF picks OKF's *recommended* link form (over plain relative) for a concrete reason:
> `docs-reorg-memory` **moves files between domains** (splits/merges). Bundle-relative links
> survive a move — the reorg skill rewrites *far* fewer links — whereas plain relative links
> break on every move and must be rewritten in bulk. As in OKF, the relationship *type*
> (parent/child, references, depends-on) is conveyed by surrounding prose, not a typed link
> field, and consumers MUST tolerate broken links (a missing target is not malformed).

Links **out** of the bundle (to source files, specs, external URLs) use ordinary repo-relative
or absolute-URL forms as appropriate — the bundle-relative rule governs memory↔memory links.

---

## 8. Versioning

The bundle declares its FKF version in the **root `index.md` frontmatter** — the only `index.md`
permitted to carry frontmatter beyond the generator's own output:

```yaml
---
fkf_version: "0.1"
---
```

FKF emits **`fkf_version`**, not OKF's `okf_version`, because an FKF bundle is a *superset* of
OKF — claiming bare `okf_version` would under-state what the bundle guarantees. The FKF↔OKF
lineage lives in this spec (§1), not in a version field. `fab memory-index` writes
`fkf_version` into the root index on generation.

Minor versions add backward-compatible features; major versions may break. Per OKF, consumers
SHOULD attempt best-effort consumption rather than refusing an unknown version.

---

## 9. Non-Scope: `docs/specs/`

FKF governs `docs/memory/` only. `docs/specs/` is **out of scope** and unchanged:

- Specs remain **human-curated** and MUST NOT be auto-generated or overwritten by tooling
  (a fab-kit design principle).
- Specs carry **no frontmatter** and are deliberately flat and free-form.
- The `docs/specs/skills/SPEC-*.md` mirrors stay constitution-pinned (names derive mechanically
  from `src/kit/skills/` sources).

The one idea FKF's neighbours may borrow independently is **generated index files** — a
`fab specs-index` style generator for `docs/specs/index.md` would be a separate, optional
convenience and is **not** part of FKF. Adopting FKF frontmatter (`type`/`description`) on specs
would require a constitution amendment and is explicitly **not** proposed here.

---

## 10. Adoption / Migration

Moving the existing `docs/memory/` tree onto FKF is a data migration with these mechanical parts
(each tracked as its own pipeline change — FKF is the contract; these are its implementations):

1. **Add `type: memory`** frontmatter to every memory file (alongside the existing `description:`).
2. **Strip the `## Changelog` section** from every memory file (the per-file changelog tables) and
   **generate per-folder `log.md`** from git history + the new `summary:` field.
3. **Convert memory↔memory cross-links** from relative to bundle-relative (`/...`).
4. **Teach `fab memory-index`** to: stamp `type: memory` (template), emit `log.md` (C-lite
   projection joining git history + `.status.yaml` `summary`), write `fkf_version` into the root
   index, and validate/round-trip the FKF frontmatter.
5. **Add the `.status.yaml` `summary:` field** + its migration file (`src/kit/migrations/`).
6. **Update the doc skills** (`docs-hydrate-memory`, `docs-reorg-*`, and the `/fab-continue`
   hydrate path) to author FKF frontmatter, stop writing per-file changelogs, and rely on the
   generated `log.md` — with the corresponding `SPEC-*.md` mirror updates per the constitution.

Per OKF's permissive model, a partially-migrated tree still functions: a file missing `type` or
a folder missing `log.md` degrades gracefully rather than breaking consumers.

---

## 11. Glossary

| Term | Meaning |
|------|---------|
| **FKF** | Fab Knowledge Format — this spec; a profile of OKF v0.1 governing `docs/memory/`. |
| **OKF** | Open Knowledge Format v0.1 (GoogleCloudPlatform/knowledge-catalog) — the base FKF profiles. |
| **Bundle** | The `docs/memory/` directory tree, as an OKF/FKF knowledge bundle. |
| **Concept document / memory file** | A `{domain}[/{sub-domain}]/{topic}.md` file: FKF frontmatter + markdown body. |
| **Reserved filename** | `index.md` / `log.md` — generated, single-writer, not concept documents. |
| **C-lite** | The `log.md` generation model: git history (when/which/id) joined with a per-change `.status.yaml` `summary:` line (what), generated — descriptive *and* conflict-free. |
| **Stub-before-index** | Creating a new folder's `index.md` `description:`-only stub before `fab memory-index` runs (Index Ownership model). |
| **Bundle-relative link** | A memory↔memory link beginning with `/`, resolved from `docs/memory/`. |
