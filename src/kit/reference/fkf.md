# FKF — Fab Knowledge Format (v0.1) — Shipped Normative Extract

> **Single-sourcing note.** This file is the **shipped normative extract** of the dev-repo design
> doc `docs/specs/fkf.md` (in the fab-kit repository). It ships inside the kit so it is reachable in
> every user repo via `$(fab kit-path)/reference/fkf.md`, and it carries only the rules an agent
> must follow (the normative subset — §2/§3/§5/§6/§7/§8). The "why" and history (OKF lineage, prose
> rationale, Non-Scope, adoption/migration, glossary) live only in `docs/specs/fkf.md`.
>
> **When you change FKF normative rules, update BOTH files** — this extract and
> `docs/specs/fkf.md` — so they cannot silently diverge. The original `docs/specs/fkf.md` section
> numbers are preserved here so citations resolve identically against either file.

> **What this is.** FKF is the format fab-kit uses for the `docs/memory/` knowledge tree: a
> directory bundle of markdown files with YAML frontmatter, plus generated index and log files. FKF
> is a profile of OKF v0.1 (Open Knowledge Format) — every FKF bundle is a conforming OKF bundle,
> and FKF additionally requires and fixes a handful of things OKF leaves open. (Full OKF lineage and
> rationale: `docs/specs/fkf.md` §1.)
>
> **Scope: `docs/memory/` only.** FKF governs the post-implementation memory tree. It does **not**
> apply to `docs/specs/` — specs remain human-curated, frontmatter-free, and free-form per
> Constitution VI.

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

### 3.2 `description` (required, curated)

```yaml
description: "One-line summary used by the generated domain-index row."
```

OKF *recommends* `description`; FKF **requires** it, because it is **load-bearing**: the generated
domain index reads each file's row Description from this field, and the always-load context layer
routes on it. It is the one hand-curated frontmatter field — authored by every memory writer
(hydrate, `/docs-hydrate-memory`, `docs-reorg-memory`) and kept accurate on every edit.

Co-locating the description with the file (rather than in the index) is deliberate: editing a
description never touches the hot, churn-prone index row. It cannot be auto-derived from the
H1/Overview without loss (Overview prose contains literal `|` pipes that break index tables, and an
extracted first sentence degrades the routing signal).

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

The pipeline's hydrate step *writes into* `## Requirements` and `## Design Decisions`, and
review/intake *read from* them — so these headings remain the target shape and SHOULD be used
wherever the content warrants. But a small reference-pointer file need not invent a GIVEN/WHEN/THEN
scenario: the rule is *SHOULD-use-these-conventional-headings-where-they-apply*, not
*MUST-have-these-sections*.

> **No `## Changelog` section.** Per-file changelog tables are **removed** in FKF — change history
> lives in the per-folder generated `log.md` (§6). This is the single biggest FKF divergence from
> the pre-FKF memory format.

### 3.4 Optional frontmatter

FKF neither requires nor forbids the other OKF-recommended fields (`title`, `tags`,
`timestamp`, `resource`). `resource` (a URI to an underlying asset) is typically **absent** —
memory files document *behavior*, not addressable assets. Per OKF, consumers MUST preserve
unknown frontmatter keys on round-trip and MUST NOT reject a file for carrying them.

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
- Absence degrades gracefully: a change with no `summary` projects with the change slug in place of
  the descriptive line.

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
OKF — claiming bare `okf_version` would under-state what the bundle guarantees. `fab memory-index`
writes `fkf_version` into the root index on generation.

Minor versions add backward-compatible features; major versions may break. Per OKF, consumers
SHOULD attempt best-effort consumption rather than refusing an unknown version.
