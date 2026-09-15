# Intake: Root-Agnostic Docs-Index Generator

**Change**: 260908-flt3-root-agnostic-docs-index
**Created**: 2026-09-08

## Origin

One-shot promptless dispatch (`/fab-proceed` create-new lane). The user's brief, verbatim:

> # Generalize `fab memory-index` into a root-agnostic doc-index generator
>
> ## The ask
>
> `fab memory-index` is hardcoded to `docs/memory`. I want to run the same generator over `docs/specs` (and potentially other doc roots) without forking it. Generalize the command to take a configurable root, and rename it to match its new scope.
>
> Two things to decide and implement:
>
> 1. **Root-agnostic generation** — a configured/flagged root instead of the hardcoded `docs/memory`.
> 2. **Rename** — `memory-index` no longer describes what it does. Something like `fab docs-index` / `fab index`, with `memory-index` kept as a deprecated alias for at least one minor version so existing scripts and skills don't break. Grep the kit for internal callers (skills, hydrate/reorg guards, stage hooks, CI) and update them in the same change.
>
> ## Why this is not a trivial parameterization
>
> The specs tree violates assumptions memory's generator is allowed to make. Measured on a real repo (~294 spec files, ~338 memory files):
>
> |  | docs/memory | docs/specs |
> |--|-------------|------------|
> | Max folder depth | 3 | 7 |
> | Depth distribution | 29 dirs @2, 16 @3 | 15 @2, 19 @3, 22 @4, 32 @5, 11 @6, 2 @7 |
> | Files with `description:` | 249 / 338 | 2 / 294 |
> | Folder landing file | `index.md` only | `index.md` (37) and `README.md` (9) |
> | Versioned subfolders | none | `v1/ v2/ … v5/` + `archive/` throughout |
>
> So at minimum:
>
> - **Depth**: the domain → sub-domain → file model (depth 3) does not hold. The generator needs to recurse to arbitrary depth, or take a configurable depth bound. The existing soft-warn at depth 3 must not become a hard failure for a root that is legitimately deeper.
> - **Landing-file name**: must accept `README.md` as a folder's index where that is the existing convention, or make the index filename configurable per root. Do not silently create `index.md` next to an existing `README.md`.
> - **Missing descriptions**: with 2/294 populated, a first run must degrade gracefully — emit a row with the H1 and an empty/placeholder description and warn, never fail, and never invent a description.
> - **Superseded subtrees must render differently from live ones.** This is the most important requirement and the easiest to under-specify, so it is spelled out in its own section below.
>
> ## Superseded subtrees (the part to get right)
>
> A folder has a **status** — current or superseded — and the generator must render the two differently. Superseded material gets a pointer and a count, not per-file rows. It is not a display preference: roughly half the words in our specs tree are superseded, so enumerating them buries the live content.
>
> A real folder from our repo, `1-categorization/` — 9 live files, 33 archived:
>
> ```
> 1-categorization/
> ├── v5/          9 files   <- current
> └── archive/
>     ├── v4/      8 files   <- superseded
>     ├── v3/      8 files
>     ├── v2/     12 files
>     └── v1/      5 files
> ```
>
> Enumerating all 42 is the failure mode: the 9 live rows are lost among 33 dead ones, and a reader cannot tell whether `archive/v2/4-resolver.md`, `archive/v2/4-resolver-v2.md`, or something under `v5/` is the current resolver doc. Three of those files describe the same subsystem and two are wrong.
>
> **Wanted — two-level rendering.** At the parent index, the whole superseded subtree is one row:
>
> ```
> | [v5/](v5/index.md) | Current — cards, cells, identity, stamping, resolver | 9 files |
> | archive/           | 4 superseded versions (v1–v4), 33 files              |         |
> ```
>
> and at the `archive/` index itself (if one is generated), one row per version:
>
> ```
> | archive/v4/ | Superseded                                    | 8 files  |
> | archive/v3/ | Superseded                                    | 8 files  |
> | archive/v2/ | Superseded                                    | 12 files |
> | archive/v1/ | Superseded — May findings, original proposal  | 5 files  |
> ```
>
> So the live index stays about live content, and the history stays one click away and still navigable. Descriptions are not read from files inside a superseded subtree — only the count and the folder name are used.
>
> Name the config knob for the status, not for the rendering side-effect (`superseded:`, not `collapse:`). It then mechanizes a rule the repo has already written down in its own `versioning.md`: superseded versions are kept, and point forward to their successor.
>
> **Check before implementing**: `archive/**` may not be the only marker for superseded material. In our tree there is also at least one `[archived]-`-prefixed filename and a `Z`-prefix folder convention in the naming docs. If superseded content is marked three different ways, the pattern list has to accept all three, or the index silently misses some. Prefer a list of glob patterns over a single hardcoded `archive/`.
>
> ## Preserve, exactly
>
> Everything that makes `memory-index` safe today must survive:
>
> - **Byte-stable / idempotent output**, a pure function of content (no git dates), so output is branch-independent and does not generate merge conflicts.
> - `--check` with the graded exit code: 0 clean / 1 benign drift / 2 destructive loss, plus the blocking-vs-advisory warning split where blocking findings floor the exit at 1 without becoming tier-2.
> - `--json` loss report with the `tier` / `drift` / `losses` / `malformed` / `warnings` keys.
> - **Refuse-before-regen guards** that currently key on exit 2.
> - **Only index tables are rewritten.** Prose outside the generated block is human-owned and must never be touched. This is the property that makes generation acceptable in a hand-curated folder — call it out in the docs.
>
> `log.md` / FKF freeze-on-write is memory-specific. Make it opt-in per root rather than assuming it; a specs root should generate indexes and no `log.md`.
>
> ## Config shape (suggestion, open to better)
>
> ```yaml
> docs_index:
>   roots:
>     - path: docs/memory
>       index_file: index.md
>       log: true              # FKF log.md freeze-on-write
>       max_depth: 3           # keeps today's soft-warn behavior
>     - path: docs/specs
>       index_file: index.md
>       also_accept: [README.md]
>       log: false
>       max_depth: 8
>       superseded: ["**/archive/**"]   # pointer + count, never per-file rows
> ```
>
> `fab docs-index` with no argument processes every configured root; `fab docs-index docs/specs` (or `--root`) processes one. Keep it working with zero config for a repo that only has `docs/memory`, so this is not a breaking change.
>
> ## Deliverables
>
> 1. The generalized command + rename, with `memory-index` as a deprecated alias.
> 2. Config schema + `fab config explain` entry for the new key.
> 3. Internal callers in the kit updated (skills, guards, hooks).
> 4. A migration note: what an existing repo has to do (ideally nothing).
> 5. Tests covering a deep tree (depth 7), a `README.md`-landing folder, a root with almost no `description:` frontmatter, a superseded subtree rendering as a pointer + count at the parent and per-version rows at its own index, and idempotency (run twice, no diff).

## Why

`fab memory-index` (`src/go/fab/cmd/fab/memory_index.go` + `src/go/fab/internal/memoryindex/`, ~5,100 lines with tests) is hardcoded to `docs/memory` at every layer: the root path (`filepath.Join(repoRoot, "docs", "memory")`), the 3-tier domain → sub-domain → file model, `MaxDepth = 3`, the `index.md`-only landing convention, and the always-on FKF `log.md` freeze-on-write projection. Users with large spec trees (measured: ~294 files, depth up to 7, 2/294 with `description:` frontmatter, mixed `index.md`/`README.md` landings, roughly half the content in superseded `archive/` subtrees) get none of the generator's benefits — byte-stable navigable indexes, drift detection, merge-conflict immunity — without forking the generator.

If not fixed, spec trees stay hand-indexed (staleness, merge conflicts on hot index rows — exactly the failure mode memory-index was built to kill) or users fork the generator (divergence). Generalizing the existing generator, rather than writing a second one, preserves the hardened safety machinery — the graded `--check` exit codes, blocking-vs-advisory warning split, destructive-loss classifier, freeze-on-write logs — that took multiple changes to get right.

The rename is a consequence, not vanity: a command named `memory-index` that indexes arbitrary roots misleads. The deprecated alias keeps every existing script, skill, and muscle memory working for at least one minor version.

**Known repo tension — RESOLVED (2026-09-08 clarification)**: Constitution VI states specs "MUST NOT be auto-generated or overwritten by tooling", `docs/memory/memory-docs/specs-index.md` records "no specs-index generator" as a deliberate design stance, and `src/kit/skills/docs-reorg-specs.md` explicitly instructs "Do not 'fix the asymmetry' by adding a specs backfill… no generated-index model for specs." The user resolved this: **Constitution VI is amended in this change** to carve out index/navigation files — auto-generated index tables are navigation aids, not spec content, so generating them in `docs/specs` does not violate the specs-are-human-curated principle (prose outside the generated block stays human-owned). **fab-kit's own repo configures a `docs/specs` root in this change** (full dogfooding), and the recorded "no specs-index generator" stance (specs-index.md memory, docs-reorg-specs.md) is superseded by this decision and updated in this change. <!-- clarified: Constitution VI amendment + dogfood specs root + supersede recorded stance — user decision, 2026-09-08 session --> See Assumptions #1–#2 (resolved).

## What Changes

### CLI: rename to `fab docs-index`, keep `memory-index` as a deprecated alias

- New command `fab docs-index` (chosen over `fab index` — matches the brief's own suggested config key `docs_index`; assumption #4). Signature:
  - `fab docs-index` — processes every configured root (zero config ⇒ the implicit `docs/memory` root with today's exact behavior).
  - `fab docs-index <root-path>` (positional, e.g. `fab docs-index docs/specs`) — processes one configured root. Unconfigured path argument is an error naming the config key.
  - Flags carried over unchanged: `--check`, `--json`, `--rebuild` (`--rebuild` remains meaningful only for `log:`-enabled roots).
- `fab memory-index` becomes a deprecated alias: same flags, runs the new code against the `docs/memory` root only, prints a one-line deprecation notice to stderr (never stdout — `--json` consumers parse stdout). Kept for at least one minor version; removal is a future change (Non-Goal here).
- Exit-code contract preserved verbatim: `--check` 0 clean / 1 benign drift / 2 destructive loss; blocking findings floor the exit at 1 without becoming tier-2; `--json` keeps the `tier`/`drift`/`losses`/`malformed`/`warnings` keys (additive keys allowed, none removed or renamed).

### Config: `docs_index.roots` (project scope)

New config key, registered with `fab config explain` metadata (`default`/`description`/`scope`/`advertise` per `docs/specs/config.md`), shaped per the brief's suggestion:

```yaml
docs_index:
  roots:
    - path: docs/memory
      index_file: index.md
      log: true              # FKF log.md freeze-on-write
      max_depth: 3           # keeps today's soft-warn behavior
    - path: docs/specs
      index_file: index.md
      also_accept: [README.md]
      log: false
      max_depth: 8
      superseded: ["**/archive/**"]   # pointer + count, never per-file rows
```

Per-root fields:

- `path` (required) — repo-root-relative doc root.
- `index_file` (default `index.md`) — the generated landing filename.
- `also_accept` (default empty) — alternative landing filenames (e.g. `README.md`). A folder whose landing file is an `also_accept` name keeps it: the generator MUST NOT silently create `index_file` next to an existing accepted landing file.
- `log` (default `false`; the implicit zero-config memory root defaults it `true`) — opt-in FKF `log.md` freeze-on-write. `log.md`/C-lite/`log.seed.md` machinery is untouched and remains memory-shaped; a specs root generates indexes and no `log.md`.
- `max_depth` (default `3`) — the soft-warn depth bound, per root. A deeper tree warns (advisory, stderr) and still generates — never a hard failure.
- `superseded` (default empty) — list of glob patterns marking superseded material (see below).

**Zero-config default**: when `docs_index` is absent, behavior is byte-identical to today — one implicit root `{path: docs/memory, index_file: index.md, log: true, max_depth: 3}`. Explicitly not a breaking change; existing repos do nothing (deliverable 4's migration note documents exactly this).

### Generation: arbitrary depth, configurable landing files, graceful missing descriptions

- **Depth**: recursion generalizes from the fixed domain/sub-domain/file model to arbitrary nesting. Every folder with content gets a generated landing file (subject to the superseded rules below); the parent index links child folders. The `max_depth` soft-warn replaces the hardcoded `MaxDepth = 3` (which becomes the memory root's per-root value). Existing depth-3 memory trees render byte-identically.
- **Landing files**: for a folder whose existing landing file matches `also_accept` (e.g. `README.md`), the generated index table is written into that file inside a **marker-delimited generated block**; prose outside the block is human-owned and never touched. For `index_file` landings the whole-file generation model is preserved (today's memory `index.md` bytes unchanged). <!-- clarified: landing-file model user-confirmed 2026-09-08 (bulk confirm, Assumption #7) — generated block inside also_accept landings, whole-file generation for index_file landings -->
- **Missing descriptions**: a file without `description:` frontmatter emits a row with its H1 and an empty/placeholder description (`—`, today's rendering) plus an advisory warning — never a generation failure, and the generator never invents a description. This is today's behavior, restated as a contract because sparse-frontmatter roots (2/294) make it the common case, not the edge case.
- **Only index tables are rewritten** — called out explicitly in the command's docs (`_cli-fab.md` § and the long help): generated blocks/files are the tool's; everything else is human-owned.

### Superseded subtrees: two-level rendering keyed on folder status

A folder (or path) matching any `superseded:` glob has status **superseded**; everything else is **current**. Rendering:

1. **At the parent index**: the entire superseded subtree is ONE row — folder name, a summary description (`{N} superseded versions ({v1}–{vN}), {M} files` when children are versioned folders; otherwise `Superseded — {M} files`), and no per-file rows. No link when no index is generated inside; link to the superseded folder's own index when one is (see #2).
2. **At the superseded folder's own index**: one row per immediate child folder — `Superseded` (plus the child's own `description:` when its landing-file stub carries one), and a file count. Per-file rows are never emitted inside a superseded subtree.
3. **Descriptions are not read from files inside a superseded subtree** — only counts and folder names. (Cheap by construction: the walk can skip file reads under superseded paths.)

Worked example (from the brief, target rendering at the parent):

```
| [v5/](v5/index.md) | Current — cards, cells, identity, stamping, resolver | 9 files |
| archive/           | 4 superseded versions (v1–v4), 33 files              |         |
```

and at `archive/`'s own index:

```
| archive/v4/ | Superseded                                    | 8 files  |
| archive/v3/ | Superseded                                    | 8 files  |
| archive/v2/ | Superseded                                    | 12 files |
| archive/v1/ | Superseded — May findings, original proposal  | 5 files  |
```

- The knob is named for the **status** (`superseded:`), never the rendering side-effect (`collapse:`).
- **Pattern list, not a hardcoded `archive/`**: the config takes glob patterns. Patterns may match folders (`**/archive/**`) or files (`**/[archived]-*` — the brief flags at least one `[archived]-`-prefixed filename and a `Z`-prefix folder convention in the target repo). A superseded *file* inside a current folder is excluded from per-file rows and folded into a `{K} superseded files` note on its folder's row. <!-- clarified: file-level pattern matches fold into the parent folder's superseded count — no per-file rows anywhere; user confirmed 2026-09-08 -->
- Whether an index is generated *inside* the superseded folder at all is per rendering rule 2 above (yes, one level, per-version rows) <!-- assumed: generate the superseded folder's own index one level deep — the brief shows it parenthetically ("if one is generated") and the wanted-rendering example includes it -->

### Safety machinery: preserved and generalized

- **Byte-stable / idempotent**: output remains a pure function of content (no git dates) across all roots; run-twice-no-diff is a test.
- **`--check` graded exits + blocking/advisory split + `--json` report**: preserved verbatim, now scoped per root and aggregated (worst exit wins across roots). Refuse-before-regen guards in `/docs-hydrate-memory`, `/docs-reorg-memory`, `/fab-continue` hydrate, `/docs-distill-memory` continue to key on exit 2 unchanged.
- **Destructive-loss classifier**: the tier-2 detectors (curated-description wipe, tombstone drop, custom-grouping flatten) apply to every root.
- **First-run adoption = seed-import (resolved 2026-09-08, Assumption #2)**: on the first generation over a root with pre-existing hand-curated landing files, the generator reads the existing curated index rows/descriptions and imports them into the generated output — seeding frontmatter/placeholder descriptions from curated rows where files lack `description:` — so nothing curated is lost and the tier-2 destructive-loss guard does not trip. No adopt/force flag, no refuse-until-backfill: an existing repo has to do nothing. <!-- clarified: seed-import first-run adoption — user decision, 2026-09-08 session -->
- **FKF specifics stay memory-scoped**: `fkf_version` root frontmatter, `log.md`, `log.seed.md`, C-lite joins, `_shared/`/`_unsorted/` reserved-domain exemptions, and the FKF §3.2 description escalations remain tied to the `log: true` / memory-shaped root. <!-- clarified: blocking description-escalation classes (change-id in description, >1000-rune cap) stay memory-root-only; advisory warnings (width, depth, description length) generalize — user confirmed 2026-09-08 -->

### Constitution VI amendment + fab-kit dogfooding (resolved 2026-09-08)

- **Constitution VI amended in this change**: carve out index/navigation files from the "specs MUST NOT be auto-generated" rule — auto-generated index tables are navigation aids, not spec content; prose outside the generated block stays human-owned, so the specs-are-human-curated principle is preserved. Amendment follows the repo's governance convention (dated HTML-comment note; narrows/rewords a normative MUST rule ⇒ minor version bump 1.6.0 → 1.7.0 per the xy7a/rehi precedent).
- **fab-kit's own repo configures a `docs/specs` root** (full dogfooding): `fab/project/config.yaml` gains a `docs_index.roots` entry for `docs/specs` (`index_file: index.md`, `log: false`; exact `max_depth`/`superseded` values chosen at apply — the tree today is 24 files, depth 2, no archive subtrees). First generation runs against the existing hand-curated `docs/specs/index.md` (22/24 files lack `description:`), exercising the seed-import path end to end.
- **Recorded stance superseded and updated in this change**: `docs/memory/memory-docs/specs-index.md` ("no specs-index generator", the no-symmetry rationale) and `src/kit/skills/docs-reorg-specs.md` (the no-symmetry note and "no generated-index model for specs" rows, plus its hand-rewrite-the-index step) are rewritten to the new model — the specs *index* is generated; spec *content* stays human-curated.

### Kit skills migrated to the new command (explicit scope item — user addendum)

**Every kit skill that invokes `fab memory-index` moves to the new command in this same change. The deprecated alias exists for EXTERNAL scripts only; the kit's own skills MUST NOT rely on it.** Grep-verified caller surface (occurrence counts of the `memory-index` literal per file — the plan covers each file):

| Skill file (`src/kit/skills/`) | Hits | What invokes/references it |
|---|---|---|
| `docs-reorg-memory.md` | 23 | reorg orchestration: `--check` guard, backfill dispatch, single end-of-run regen |
| `docs-hydrate-memory.md` | 19 | index-ownership model, refuse-before-regen guard, ingest/generate/backfill Step-4 regen, backfill caller-awareness |
| `docs-distill-memory.md` | 16 | survey's `--check --json` machine surface (canonical signal source), regen step, older-binary fallbacks |
| `_cli-fab.md` | 12 | `## fab memory-index` command-reference section (rewrite to `## fab docs-index` + alias note; CLI ⇒ docs constraint) |
| `fab-continue.md` | 5 | hydrate steps 5–6: regen + refuse-guard + shape guidance |
| `git-pr.md` | 4 | ship-time index/log regen step + never-hand-merge note |
| `git-pr-review.md` | 1 | never-hand-merge note on rebase conflicts (the review-pr drift-check path) |
| `docs-reorg-specs.md` | 1 | the "no specs-index generator" asymmetry note — superseded; rewritten per resolved Assumption #1 (generated index, human-curated content; index step re-pointed at the generator) |

- Older-binary fallbacks inside these skills (e.g. distill's "`fab memory-index --check --json` unavailable" row) are updated to probe the NEW command and fall back through the alias for a fab binary that predates the rename — the one sanctioned in-kit alias use, and only inside version-skew fallback prose.
- **Templates/reference/scaffold**: `src/kit/templates/memory.md` (cap-warning comment), `src/kit/reference/fkf.md` (§2, §3.2, §5, §6 command mentions), `src/kit/scaffold/docs/memory/index.md` (generated-by header).
- **Go binary docs**: `src/go/fab/cmd/fab/skill.md`.
- **Generated-by headers**: the root/domain index header line `> **Generated by \`fab memory-index\`**` is generator output — the renderer emits the new name (memory-root indexes churn one line on first regen; benign drift, not destructive).
- **Migrations**: historical migration files (`2.4.2-to-2.5.0.md`, `2.5.5-to-2.6.0.md`, `2.6.6-to-2.7.0.md`) are frozen history — NOT rewritten. New migration note added per deliverable 4.
- **CI**: no `.github` workflow references `memory-index` (grep-verified) — nothing to update there.

### Migration note

A `src/kit/migrations/` note (next minor version range) documenting: existing repos need to do **nothing** — zero config keeps today's behavior byte-for-byte; `fab memory-index` still works (deprecated alias); repos wanting additional roots add `docs_index.roots`. First regen after upgrade rewrites the one generated-by header line per index file (benign drift).

### Tests (deliverable 5)

Per the brief, covering at least: a depth-7 tree; a `README.md`-landing folder (generated block inserted, surrounding prose untouched); a root with almost no `description:` frontmatter (placeholder rows + warnings, exit 0 on write path); a superseded subtree rendering pointer+count at the parent and per-version rows at its own index; idempotency (run twice, no diff); the zero-config memory root rendering byte-identically to today (golden); the deprecated alias delegating with a stderr notice; `--check` exit tiers per root. Go changes ship tests in the same change (Constitution / code-review policy).

## Affected Memory

- `memory-docs/docs-index`: (new) the generalized generator — roots config, per-root knobs, superseded-status rendering, landing-file model, what stayed memory-specific
- `memory-docs/hydrate`: (modify) command rename in index-ownership model, refuse-before-regen guard, backfill caller-awareness
- `memory-docs/templates`: (modify) index-generation and cap-warning claims naming `fab memory-index`
- `memory-docs/distill`: (modify) survey machine-surface (`--check --json`) and regeneration references
- `memory-docs/specs-index`: (modify) the "no specs-index generator" stance is superseded (resolved Assumption #1) — rewrite to the generated-index-for-navigation model; spec content stays human-curated
- `distribution/kit-architecture`: (modify) command surface + internal package map (`internal/memoryindex` rename/generalization)
- `distribution/migrations`: (modify) shipped migration catalog gains the rename/config note

## Impact

- **Go**: `src/go/fab/internal/memoryindex/` (1,437-line core + log/loss/seed/indexparse + 4 test files, ~5,100 lines total) — generalize walk/render/classify to per-root config; `src/go/fab/cmd/fab/memory_index.go` (+test) — new `docs-index` command + alias; `main.go` registration; config schema + `fab config explain` metadata for `docs_index` (project scope); possibly `defaults.yaml` if the implicit root is expressed there.
- **Kit content**: 8 skill files, `templates/memory.md`, `reference/fkf.md`, scaffold header, new migration file, `_cli-fab.md`, `cmd/fab/skill.md` (CLI ⇒ docs + tests constraint applies).
- **Specs (human-curated, updated in-change per repo practice)**: `docs/specs/config.md` (new key), possibly `docs/specs/fkf.md`; `fab/project/constitution.md` — Constitution VI amendment (index/navigation carve-out, minor version bump; resolved Assumption #1); `fab/project/config.yaml` — fab-kit's own `docs_index.roots` `docs/specs` entry (dogfooding); `docs/specs/index.md` — first generated regeneration via seed-import.
- **Behavioral surface**: memory-root output byte-identical except the generated-by header rename; new behavior only under new config. PR tier: MINOR (new command + config key + deprecation).
- **Sibling-sweep classes** (code-quality.md): the `fab memory-index` literal appears in dozens of prose spots including user-facing string literals (`remediationPointer`, error texts, `--check` messages) — the behavior-claim sweep must include string literals and `*_test.go` comments.

## Open Questions

- ~~Does generating index files inside `docs/specs` require a Constitution VI amendment, and should fab-kit's own repo configure a `docs/specs` root?~~ **Resolved (2026-09-08)**: yes to both — Constitution VI amended in this change (index/navigation carve-out: generated index tables are navigation aids, not spec content), and fab-kit's own repo configures a `docs/specs` root (full dogfooding); the recorded "no specs-index generator" stance is superseded and its docs updated in this change.
- ~~What should the first run over a root with pre-existing hand-curated `index.md`/`README.md` files do?~~ **Resolved (2026-09-08)**: seed-import — the generator reads the existing curated index rows/descriptions and imports them into the generated output (seeding frontmatter/placeholder descriptions from curated rows where files lack `description:`), so nothing curated is lost and the tier-2 guard does not trip; no adopt/force step, no refuse-until-backfill — an existing repo has to do nothing.
- ~~Exact new command name: `fab docs-index` assumed (matches the suggested `docs_index` config key) over `fab index` — confirm.~~ **Resolved (2026-09-08 bulk confirm)**: `fab docs-index`, with `memory-index` as a deprecated alias emitting a one-line stderr notice.
- Internal Go package naming: rename `internal/memoryindex` → `internal/docsindex` (clean but churns imports/history) or keep the package name with a generalized API?

## Clarifications

### Session 2026-09-08 (auto — user-provided decisions via dispatcher)

| # | Action | Detail |
|---|--------|--------|
| 1 | Resolved | Amend Constitution VI in this change (index/navigation carve-out — generated index tables are navigation aids, not spec content); configure a `docs/specs` root in fab-kit's own repo (full dogfooding); supersede + update the "no specs-index generator" stance (specs-index.md memory, docs-reorg-specs.md) |
| 2 | Resolved | Seed-import: first generation over a root with hand-curated landings imports curated rows/descriptions into the generated output (seeding frontmatter/placeholder descriptions where `description:` is missing) — nothing curated lost, tier-2 guard does not trip, existing repos do nothing |
| 7 | Reinforced | S raised 40→70: decision #1's "prose outside the generated block stays human-owned" endorses the generated-block landing model; the index_file whole-file half remains inferred |

Scoring note: rows 1–2 were promptless-dispatch deferrals whose S/R/A/D were blocking placeholders (per `_srad.md` § Critical Rule, promptless-dispatch carve-out) grading the *unasked question*, not the substantive decision. On resolution all four dimensions were re-scored for the decided content (S:95 per the clarify rule; R/A/D re-scored because keeping question-placeholder lows would misgrade a fully user-decided item and permanently defeat the clarify-unblocks-the-gate design).

### Session 2026-09-08 (bulk confirm)

The user answered a bulk-confirm round directly (second session, same day): all seven Confident rows confirmed exactly as written, and Tentative row #6 resolved. Per the clarify rule, S → 95 with R/A/D unchanged; grades recomputed from the new composites (all eight rows land Confident — composites 56–76.5, below the ≥80 Certain band).

| # | Action | Detail |
|---|--------|--------|
| 4 | Confirmed | — |
| 6 | Confirmed | "fold into folder count" — a `superseded:` pattern matching an individual file folds into its parent folder's superseded count; no per-file rows anywhere |
| 7 | Confirmed | — |
| 12 | Confirmed | — |
| 13 | Confirmed | — |
| 14 | Confirmed | — |
| 15 | Confirmed | — |
| 16 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Constitution VI amended in this change: index/navigation files carved out — generated index tables are navigation aids, not spec content (prose outside the generated block stays human-owned); fab-kit's own repo configures a `docs/specs` root (full dogfooding); the recorded "no specs-index generator" stance (specs-index.md memory, docs-reorg-specs.md) is superseded and updated in this change | Clarified — user decided (2026-09-08 session); deferred-placeholder dims re-scored for the resolved decision | S:95 R:60 A:90 D:90 |
| 2 | Certain | First-run adoption = seed-import: the generator reads pre-existing curated index rows/descriptions and imports them into the generated output (seeding frontmatter/placeholder descriptions where files lack `description:`), so nothing curated is lost and the tier-2 guard does not trip; no adopt/force flag, no refuse-until-backfill — existing repos do nothing | Clarified — user decided (2026-09-08 session); deferred-placeholder dims re-scored for the resolved decision | S:95 R:75 A:85 D:90 |
| 3 | Certain | Zero-config default = implicit `docs/memory` root with today's exact behavior (log on, max_depth 3, index.md); not a breaking change; existing repos do nothing | Brief mandates it explicitly ("Keep it working with zero config… not a breaking change") | S:90 R:85 A:90 D:90 |
| 4 | Confident | New command named `fab docs-index` (not `fab index`); `memory-index` kept as deprecated alias ≥1 minor version with a one-line stderr notice; alias removal is a future change | Clarified — user confirmed (2026-09-08 bulk confirm) | S:95 R:40 A:60 D:65 |
| 5 | Certain | Superseded rendering: pointer + count at parent (one row for the whole subtree), per-version rows at the superseded folder's own index, descriptions never read inside superseded subtrees; knob named `superseded:` for status, not rendering | Brief specifies it with worked examples and names it "the part to get right" | S:90 R:75 A:85 D:90 |
| 6 | Confident | `superseded:` is a list of glob patterns (not hardcoded `archive/`); patterns may match folders or files; a pattern matching an individual file folds into its parent folder's superseded count — no per-file rows anywhere | Clarified — user confirmed (2026-09-08 bulk confirm): fold into folder count | S:95 R:55 A:50 D:45 |
| 7 | Confident | Landing-file model: fenced generated block only inside `also_accept` landings (README.md — human prose preserved outside the block); `index_file` landings keep today's whole-file generation (memory root byte-stable) | Clarified — user confirmed (2026-09-08 bulk confirm); both halves now user-decided | S:95 R:40 A:50 D:50 |
| 8 | Certain | Missing `description:` degrades gracefully: H1 + placeholder row + advisory warning; generation never fails on it and never invents a description | Brief mandates verbatim; matches today's `—` rendering | S:85 R:85 A:90 D:90 |
| 9 | Certain | Depth: recurse to arbitrary depth; per-root `max_depth` is a soft-warn bound only (advisory stderr, never a hard failure); memory root keeps 3 | Brief mandates ("must not become a hard failure"); config shape carries max_depth per root | S:85 R:80 A:85 D:85 |
| 10 | Certain | `log.md`/FKF freeze-on-write is per-root opt-in (`log:`); memory root keeps it on, specs roots default off — a specs root generates indexes and no log.md | Brief decides it explicitly ("Make it opt-in per root rather than assuming it") | S:90 R:80 A:85 D:90 |
| 11 | Certain | Preserve exactly: byte-stability/idempotency, `--check` 0/1/2 graded exits, blocking-floor-at-1 vs advisory split, `--json` `tier`/`drift`/`losses`/`malformed`/`warnings` keys, exit-2 refuse-before-regen guards | Brief's "Preserve, exactly" section enumerates each; existing tests pin them | S:90 R:70 A:90 D:90 |
| 12 | Confident | Config shape adopted as suggested (`docs_index.roots` with path/index_file/also_accept/log/max_depth/superseded), project scope, registered with `fab config explain` metadata; field names may be tuned only if implementation reveals a conflict | Clarified — user confirmed (2026-09-08 bulk confirm) | S:95 R:70 A:65 D:70 |
| 13 | Confident | Single-root invocation via positional arg (`fab docs-index docs/specs`); `--root` not added | Clarified — user confirmed (2026-09-08 bulk confirm) | S:95 R:80 A:65 D:70 |
| 14 | Confident | FKF §3.2 blocking description escalations (change-id in description, >1000-rune cap) stay scoped to the memory-shaped (`log: true`) root; shape/size/length advisory warnings generalize to all roots | Clarified — user confirmed (2026-09-08 bulk confirm) | S:95 R:60 A:55 D:55 |
| 15 | Confident | Migration deliverable = a migration note documenting nothing-to-do (zero-config back-compat + alias); no user-data restructuring occurs, so no restructuring migration logic | Clarified — user confirmed (2026-09-08 bulk confirm) | S:95 R:70 A:70 D:70 |
| 16 | Confident | Historical migration files' `memory-index` mentions are frozen history and not rewritten; generated-by header lines in existing indexes churn once (benign drift) on first regen with the new name | Clarified — user confirmed (2026-09-08 bulk confirm) | S:95 R:70 A:55 D:40 |
| 17 | Certain | All kit skills invoking `fab memory-index` migrate to the new command in this change; the deprecated alias is for external scripts only — kit skills never rely on it (except inside older-binary version-skew fallback prose) | User addendum mandates it verbatim; caller surface grep-enumerated in What Changes | S:90 R:80 A:90 D:90 |

17 assumptions (9 certain, 8 confident, 0 tentative, 0 unresolved).
