# FKF — Fab Knowledge Format (v0.1) — Design Companion

> **What this is.** The **non-normative design companion** to the FKF standard: design rationale,
> OKF lineage, bundle organization, non-scope discussion, adoption/migration history, and the
> glossary. The **normative standard** — the rules an agent must follow — lives at
> [`docs/site/fkf.md`](../site/fkf.md), published at <https://shll.ai/fab-kit/fkf> and shipped
> verbatim into the kit cache as `$(fab kit-path)/reference/fkf.md` (the copy deployed skills
> cite). This file carries **zero normative text**: each former normative section below
> (§2/§3/§5/§6/§7/§8) is a pointer stub at its original heading, with that section's design
> rationale retained beneath it. Section numbering is original throughout — "FKF §N" citations
> resolve against the standard, and the stubs here redirect in one hop.
>
> **Editing FKF.** Normative rule changes are made in `docs/site/fkf.md`, then synced by
> `scripts/sync-fkf.sh` into `src/kit/reference/fkf.md`; the drift-guard test
> `src/go/fab/cmd/fab/fkf_sync_test.go` fails CI on any divergence. This companion changes only
> when the rationale or history changes.

---

## 1. Relationship to OKF

FKF is a **profile**: a base format plus a set of additional constraints. FKF profiles
[OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
(Open Knowledge Format, GoogleCloudPlatform/knowledge-catalog) — every FKF bundle is a conforming
OKF bundle; a generic OKF consumer can read an FKF bundle; fab's tooling enforces more than OKF
requires. The split:

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

*Normative — moved to the [FKF standard](../site/fkf.md) §2.*

---

## 3. Concept Documents (memory files)

*Normative — moved to the [FKF standard](../site/fkf.md) §3.* Retained design rationale:

> **Why `type` is a constant, not sub-types (§3.1).** A per-file kind (`requirements` vs
> `design-decision` vs `reference`) was rejected: a single memory file legitimately *mixes*
> requirements, design decisions, and history in one document, so a per-file type would
> misrepresent it. The organizing axis fab actually uses is the **domain** (the folder), not a
> `type`. If a genuinely distinct second kind of memory document ever appears, FKF v0.2 may widen
> the `type` vocabulary — driven by a real distinction, not anticipated up front.

> **Why the description lives on the file — the Starlight lesson (§3.2).** Co-locating the
> `description:` with the file (rather than in the index) is deliberate: editing a description
> never touches the hot, churn-prone index row. It cannot be auto-derived from the H1/Overview
> without loss (Overview prose contains literal `|` pipes that break index tables, and an
> extracted first sentence degrades the routing signal).

> **Why the length cap escalates to blocking (§3.2).** The blocking tier exists because the
> advisory-only posture demonstrably failed: 33×/50×-cap descriptions (16,519 and 24,906
> characters) shipped straight through the nag and bloated the always-load route tables. The
> rationale for a cap at all: a giant single-line description bloats the hot, same-line
> index-row merge surface and degrades the routing signal the description exists to provide.

> **Why conventional headings are recommended, not mandated (§3.3).** The pipeline's hydrate step
> *writes into* `## Requirements` and `## Design Decisions`, and review/intake *read from* them —
> so these headings remain the target shape and SHOULD be used wherever the content warrants. But
> a small reference-pointer file should not be forced to invent a GIVEN/WHEN/THEN scenario. FKF
> relaxes the former *MUST-have-these-sections* rule to
> *SHOULD-use-these-conventional-headings-where-they-apply*, which keeps the hydrate-writes /
> review-reads contract working without imposing ceremony on files that don't need it.

> **Why present-truth body style (§3.3).** Change-keyed delta narration duplicates what `log.md`
> (the dated *what*), git (the diff), and archived change folders already record; it accumulates
> monotonically (nothing ever consolidates a file back to current truth); it forces a reader to
> mentally replay a patch series to learn the current contract; and it wastes tokens on every
> always-load/lookup read. The body's job is the current contract, stated once.

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

**Present-truth debt meters (SHOULD guidance, advisory — never enforced).** Alongside the shape
bounds, `fab memory-index` emits per-topic-file advisory warnings that measure distillation debt and
staging hygiene — the standing meters an audit would otherwise have to run by hand. None affects the
exit code (unlike the §3.2 blocking escalations): **narration-marker density** (transition stems
`no longer`/`previously`/`renamed`/`supersed` plus registry-gated change-id tokens in the body that
fall **outside** the §3.3-sanctioned citation positions — a file reaching ~5 markers warns. A trailing
`(change-id)` citation and a change-id on an `*Introduced by*:` field line do NOT count: they are the
provenance distillation KEEPS, so a fully-distilled file clears the flag; a change-id woven into prose
still counts (a density signal for narrated ids)), **file size** (a topic file over ~400 lines or ~15KB is a split
candidate), **`_unsorted/` non-empty** (staging should trend to empty — any topic file present
warns), and **broken bundle-relative links** (a `](/...)` memory↔memory target absent on disk; the
author-side counterpart to §7's consumer-tolerates-broken-links posture — only `/`-prefixed targets,
and code-fenced / inline-code examples are skipped). Acting on them (distill / split / triage / fix)
is the doc skills' job (`docs-distill-memory`, `docs-reorg-memory`); the index command only detects
and warns, and (with `--check --json`) surfaces them on the additive `warnings` array.

---

## 5. Index Files (`index.md`) — generated

*Normative — moved to the [FKF standard](../site/fkf.md) §5.* Retained design rationale:

> **Why regenerate, never hand-merge (§5).** Hand-merging a generated file is the failure mode
> the merge policy exists to prevent — it is how a corrupted row gets carried from one branch onto
> another. `fab memory-index --check` at review-pr backstops staleness: a hand-merged or
> forgotten-regen index surfaces as drift there. This is the operational counterpart to the
> byte-stability guarantee — byte-stability makes the regenerate-wholesale resolution *always
> correct*, so there is never a reason to reconcile a generated file by hand.

---

## 6. Log Files (`log.md`) — generated (C-lite)

*Normative — moved to the [FKF standard](../site/fkf.md) §6.* Retained design rationale:

> **Why C-lite, not a hand-appended log or a slug-only projection (§6.1).** A hand-appended
> `log.md` (OKF's literal convention) just *relocates* the changelog merge-conflict from N memory
> files into the folder's `log.md` — two same-day changes in one domain still collide. A pure git
> projection (slug only, no summary) is conflict-free but loses the *what-changed* signal an
> agent needs for archaeology ("where did `cssMarker` go?") and migration-trajectory questions.
> C-lite keeps the descriptive line **and** stays conflict-free, because the line lives in the
> per-change `.status.yaml`, not in the shared `log.md`. The cost is one curated line per change
> and generator plumbing in `fab memory-index`.

> **Why freeze-on-write generation (§6.4).** A pure projection of *live* git history is not
> deterministic — squash-merge rewrites commit subjects and counts, and branch-deletion makes the
> original commits unreachable — so re-projecting from scratch on every run produces a different
> result per contributor and across time (merely touching `docs/memory/` would churn dozens of
> unrelated `log.md` files and keep `--check` permanently red, a Constitution III violation in
> practice). The existing `log.md` is therefore authoritative and write-once: the generator uses
> the git projection only to discover **new** entries to append, never to re-derive what's already
> written. The append/dedup key is `(file-base, change-id)`, deliberately **not** the git commit
> hash — squash + branch-delete makes the hash unreachable (the exact operation being defended
> against), whereas the change-id survives in the change folder name and the registry, independent
> of git. New unattributable commits are not re-projected after first write (*accepted tradeoff*:
> future migration/reorg commits leave no log trace — tooling commits, not memory-domain history).
> There is deliberately no `--first-generation` flag (it would invite a re-run that re-introduces
> churn); `--rebuild` is the explicit, destructive escape hatch for a corrupted frozen log or a
> deliberate re-baseline, never the default path. Shipped semantics:
> [pipeline/schemas.md](../memory/pipeline/schemas.md) § Freeze-on-Write `log.md` Generation.

---

## 7. Cross-links — bundle-relative

*Normative — moved to the [FKF standard](../site/fkf.md) §7.*

---

## 8. Versioning

*Normative — moved to the [FKF standard](../site/fkf.md) §8.*

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
