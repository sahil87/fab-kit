# Intake: FKF Change 2/4 — `fab memory-index` emits per-folder `log.md` + stamps FKF frontmatter

**Change**: 260615-bmzo-memory-index-log-fkf-frontmatter
**Created**: 2026-06-15

## Origin

<!-- How was this change initiated? Include the user's raw input/prompt, the interaction
     mode (one-shot vs. conversational), and key decisions from the conversation. -->

> /fab-new bmzo

One-shot invocation against backlog ID `[bmzo]`. This is the **KEYSTONE** change of the four-part
FKF (Fab Knowledge Format) adoption tracked in PR #419's spec (`docs/specs/fkf.md`):

- **Change 1** `[5943]` (FOUNDATION) — add the `.status.yaml` `summary:` field + migration.
  **Already merged**: `internal/statusfile/statusfile.go` carries `Summary string \`yaml:"summary,omitempty"\``,
  `fab status set-summary`/`get-summary` exist, and `src/kit/VERSION` is at `2.5.0` with the
  migration `src/kit/migrations/2.4.2-to-2.5.0.md`. The dependency is satisfied.
- **Change 2** `[bmzo]` (this change, KEYSTONE) — teach `fab memory-index` to emit per-folder
  `log.md` and stamp FKF frontmatter. *"Once in, the format physically exists."*
- **Change 3** `[8fr5]` (depends on 2) — doc-skill prose authors FKF frontmatter + calls
  `set-summary` instead of writing per-file `## Changelog` tables.
- **Change 4** `[oovf]` (depends on 2 + 3, CUTOVER) — migrate the existing `docs/memory/` tree to FKF.

This change makes the FKF artifacts *physically exist*; it does **not** migrate the current tree
(that is Change 4) or change skill authoring prose (that is Change 3). The design is fully
specified in `docs/specs/fkf.md` §5, §6, §8 — this intake reproduces the load-bearing details.

## Why

<!-- 1. problem, 2. consequence of inaction, 3. why this approach. -->

1. **Problem.** FKF (`docs/specs/fkf.md`) defines two *generated* artifacts that do not yet exist
   in the binary: per-folder `log.md` files (C-lite change history) and FKF frontmatter
   (`type: memory` on memory files, `fkf_version: "0.1"` on the root index). Today `fab memory-index`
   generates only `index.md` tiers. Per-file `## Changelog` tables — which FKF replaces with `log.md`
   — are a recurring merge-conflict source: two same-day changes touching one memory file collide on
   the changelog rows.

2. **Consequence of inaction.** Change 3 (skill prose) and Change 4 (tree migration) are **blocked**:
   skills cannot truthfully say "history lives in generated `log.md`" until the generator emits it,
   and the tree cannot be migrated to a format the tooling does not produce. The whole FKF adoption
   stalls without this keystone.

3. **Why C-lite (the chosen model), not alternatives.** Per `fkf.md` §6.1 and backlog DECISION b:
   - A **hand-appended `log.md`** (OKF's literal convention) merely *relocates* the changelog
     merge-conflict from N memory files into one folder `log.md` — same-day changes still collide.
   - A **pure git projection** (slug only, no summary) is conflict-free but loses the *what-changed*
     signal needed for archaeology and migration-trajectory questions.
   - **C-lite** joins git history (when/which-file/change-id — conflict-free, the `when` source the
     index already uses) with the per-change `.status.yaml` `summary:` (the *what* — written once,
     into the change's own file, so zero conflict surface). It keeps the descriptive line **and**
     stays conflict-free. The cost is one curated summary line per change (Change 1's field) plus
     generator plumbing (this change).

## What Changes

<!-- The primary input for plan generation. Subsections per change area. -->

All work lives in the **existing** package `src/go/fab/internal/memoryindex/` (`memoryindex.go`,
`loss.go`, `indexparse.go`) and its cmd `src/go/fab/cmd/fab/memory_index.go`. No new package is
created — the backlog's "Package src/go/fab/internal/memoryindex/" names the home, which already
exists from prior memory-index work. The render/gather split (pure `Render*` + I/O `Gather`,
mirroring `internal/prmeta`) is preserved: new generators are pure functions over gathered inputs.

### (a) C-lite `log.md` generator — one `log.md` per domain and sub-domain folder

For each domain folder **and** each sub-domain folder (the same folders that get a generated
`index.md`), emit a `log.md` recording that folder's change history.

**Two sources, joined (neither hand-edited):**

1. **Git history keyed to the folder** — reuse the **existing batched** `loadGitDates` pass. Today
   `loadGitDates` runs `git -c core.quotepath=off log --date=short --format=%x00%ad --name-only -- docs/memory`
   and keeps only the *newest* date per path (`parseGitDates`, first-seen wins). The log generator
   needs **every commit** touching each file (not just the newest date) plus each commit's **change
   ID**. This means either extending the existing pass to also capture per-commit (date, sha,
   files-touched) tuples, or adding a second projection over the same `git log` output. Reuse the
   one batched pass — do **not** reintroduce per-file `git log` spawns (pw3k F34 explicitly collapsed
   those).

   **Commit → change-id mapping (RESOLVED — join via change-folder/branch history).** The 4-char
   change ID is embedded in the change folder name (`{YYMMDD}-{XXXX}-{slug}`) **and** in the git
   branch (branch == folder name, per `_preamble` Naming Conventions). The authoritative join:
   enumerate the change registry — every folder under `fab/changes/*` **and** `fab/changes/archive/**`
   gives the canonical set of `(change-id, folder-name)` pairs — then correlate each memory-touching
   commit to its change via git's branch/merge graph (the commits reachable on / merged from a
   change's branch). This is *authoritative* because the change owns its own identity (the folder is
   the registry), rather than relying on commit-message text. Note: `.status.yaml` and
   `.history.jsonl` do **not** currently record commit SHAs — so the join uses git's own
   commit↔branch association keyed by the folder/branch name, not a stored-SHA lookup. **Fallback**
   when a memory commit cannot be attributed to any known change (a direct edit on `main`, a pre-FKF
   historical commit): degrade gracefully — omit the `(change-id)` token (or use a short SHA) and use
   the slug/`—` for the descriptive line, per the FKF graceful-degradation rule. The exact graph-walk
   command shape is an apply-time implementation detail.

2. **Per-change one-line summary** — read each change's `.status.yaml` `summary:` field (Change 1's
   field, already shipped). Graceful absence: a change with no `summary` projects the **change slug**
   in place of the descriptive line (per `fkf.md` §6.3 and the backlog: "a change with no summary
   projects its slug in the log").

**Exact output format** (`fkf.md` §6.2 — reproduce verbatim):

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

Format rules (§6.2):
- Entries are **date-grouped, newest first**; ISO `YYYY-MM-DD` date headings.
- Each entry = **optional leading bold verb** (`**Update**` / `**Creation**` / `**Deprecation**`,
  derived from the change's `change_type` / removal markers) + a **bundle-relative** link to the
  changed file (`/{domain}/{file}.md` — beginning with `/`, resolved from `docs/memory/`, per §7) +
  the change's `summary` (or slug fallback) + the `(change-id)` in parens.
- **One line per change per file** — deliberately not paragraph-length prose.
- The generated header comment (`<!-- Generated by ... Do not hand-edit. -->`) marks `log.md` as a
  single-writer generated artifact, same discipline as `index.md`.

The **verb derivation** (CONFIRMED) maps each per-commit file change to the bold verb that leads its
`log.md` entry: a *newly added* file → `**Creation**`; a *removed/deleted* file → `**Deprecation**`;
otherwise (edited in place) → `**Update**`.
<!-- clarified: verb mapping confirmed by user; new→Creation, removed→Deprecation, else→Update -->

**How the verb is used** (per the user's question at clarify): the verb is the **leading bold token**
of each entry — e.g. `- **Update** [migrations](/distribution/migrations.md) — …`. The generator
picks it per (file, commit) tuple. The primary signal is **git's per-commit name-status** (`A` added /
`D` deleted / `M` modified — available from a `git log --name-status` projection over the same batched
pass), with the change's `change_type` as a secondary hint. Because §6.2 makes the verb **optional**,
the safe fallback when the signal is ambiguous is to **omit the verb** (or emit `**Update**`) — never a
hard error. The exact name-status flag→verb resolution is an apply-time implementation detail.

### (b) Stamp `type: memory` frontmatter

`type: memory` is the FKF constant frontmatter on every memory (concept) file (`fkf.md` §3.1).
This change teaches the generator the `type: memory` **mechanism** — it does NOT inject `type:` into
topic files (that boundary is RESOLVED at clarify, assumption #10):
- **Template** — wherever the generator scaffolds/round-trips frontmatter. Today `RenderDomain`
  round-trips `description:` into the generated `index.md` frontmatter. `type: memory` is
  carried/round-tripped the same way for the file tier the generator owns; the memory-file template
  gains the field.
- **Round-trip in `RenderDomain`/`RenderRoot`** — the backlog names these two functions explicitly.
  The generated index frontmatter is the generator's own. The generator **preserves** `type: memory`
  when it is present on a topic file but does **not author or bulk-stamp** it.
  <!-- clarified: generator = mechanism only; bulk-stamp → Change 4 migration, authoring → Change 3 skills -->
- **Boundary (CONFIRMED).** §10 step 4 ("teach `fab memory-index` to stamp `type: memory` (template)")
  is *this* change's mechanism. §10 step 1 ("add `type: memory` to every memory file") is **Change 4's
  migration** (bulk-stamp the existing ~20 files). Authoring `type:` on *new* files is **Change 3's
  skill prose**. Reaching into topic files to inject frontmatter from the generator would break the
  single-writer separation (the generator owns `index.md`/`log.md`; humans/skills own topic files).

### (c) Write `fkf_version: "0.1"` into the root `index.md` frontmatter

Per `fkf.md` §8: the **root** `docs/memory/index.md` is the **only** `index.md` permitted frontmatter
beyond the generator's output, and it carries `fkf_version: "0.1"`:

```yaml
---
fkf_version: "0.1"
---
```

`RenderRoot` currently emits **no** frontmatter (it starts with `# Memory Index`). This change makes
`RenderRoot` prepend the `fkf_version` frontmatter block. This is a **byte-stable change to the root
index output** — every regen now writes the frontmatter; a tree without it is benign drift on
`--check` (tier 1), not destructive loss.

### (d) Extend `--check`/`--json` loss tiers to cover `log.md` + new frontmatter

The `loss.go` classifier (`TierClean`/`TierBenignDrift`/`TierDestructiveLoss` → exit 0/1/2) must keep
the **refuse-before-regen** guards working once `log.md` and FKF frontmatter join the generated
surface. Today `Classify` walks `index.md` targets only. After this change:
- `log.md` files become **generated targets** — their rendered-vs-existing drift must be classified
  too, so a stale `log.md` reports tier-1 drift and CI/hydrate guards still fire.
- A **destructive-loss** category may be warranted for `log.md` (e.g. a hand-curated pre-FKF log
  whose entries the C-lite projection cannot reproduce — analogous to the existing `description`/
  `tombstone`/`grouping` categories). **Open Question OQ4**: which (if any) new tier-2 categories
  `log.md` and frontmatter introduce, vs. treating all `log.md`/frontmatter drift as benign (tier 1).
- The `--json` shape (`{"tier", "drift", "losses":[{"category","path","detail"}]}`) extends only if a
  new category is added; otherwise it is unchanged (additive).

A **born-FKF tree is provably never tier 2** (the existing invariant for index files extends: native
`log.md`/frontmatter is exactly what the generator produces).

### (e) Documentation (constitution-required, same PR)

Per Constitution (Go change → `_cli-fab.md` update + SPEC mirror):
- **`src/kit/skills/_cli-fab.md` § fab memory-index** (~L455–545) — document `log.md` generation,
  the `type: memory`/`fkf_version` frontmatter stamping, and any new loss category / exit-code
  semantics.
- **`docs/specs/fkf.md`** is the authoritative spec — already written (§5, §6, §8); reconcile only if
  implementation reveals a spec gap (specs are human-curated, MUST NOT be auto-overwritten —
  Constitution VI).
- Memory updates — see Affected Memory.

## Affected Memory

<!-- Spec-level behavior changes. -->

- `pipeline/change-lifecycle.md`: (modify) `fab memory-index` now emits per-folder `log.md` (C-lite)
  and stamps FKF frontmatter — the generated-artifact surface grows beyond `index.md`. FKF is already
  referenced here; extend with the `log.md` + frontmatter behavior.
- `memory-docs/templates.md`: (modify) the memory-file template gains `type: memory` frontmatter and
  drops the per-file `## Changelog` expectation (history now lives in generated `log.md`). Document
  the `log.md` / `fkf_version` generated shape.
- `pipeline/schemas.md`: (modify) FKF is referenced here; record the `log.md` C-lite schema, the
  `summary:` source field linkage, and the extended `--check` loss tiers.
- `distribution/migrations.md` (or wherever migrations are tracked): (modify) only if a new migration
  ships — see Open Question OQ5. This change is generator plumbing; the *tree* migration is Change 4.

## Impact

<!-- Affected code areas, APIs, dependencies, systems. -->

- **Go (`src/go/fab/`)**:
  - `internal/memoryindex/memoryindex.go` — new `RenderLog` pure function + `LogData`/`LogEntry`
    structs; extend `Gather` (or add a sibling gather) to project per-commit (date, change-id,
    files) tuples and join `.status.yaml` `summary:`; `RenderRoot` prepends `fkf_version` frontmatter;
    `type: memory` round-trip.
  - `internal/memoryindex/loss.go` + `indexparse.go` — extend `Classify`/`CheckTarget` to cover
    `log.md` targets (+ any new loss category); a `log.md`/frontmatter parser if a tier-2 detector
    needs to read existing content.
  - `cmd/fab/memory_index.go` — add `log.md` targets to the write/`--check` target list; wire the
    root-frontmatter and `log.md` outputs through the existing byte-stable write loop.
  - **Tests** (Constitution: Go change MUST add tests) — `memoryindex_test.go`, `loss_test.go`,
    `golden_test.go`, `memory_index_test.go`: golden `log.md` output, verb derivation, summary/slug
    fallback, `fkf_version` frontmatter, byte-stability/idempotence, new loss-tier classification.
- **Dependency**: Change 1 `[5943]` `.status.yaml` `summary:` field — **satisfied** (merged).
- **Reads only**: `.status.yaml` `summary:` (per change), git history, `docs/memory/` tree. **Writes**:
  `docs/memory/{domain}[/{sub}]/log.md`, `docs/memory/index.md` (frontmatter), generated index files.
- **Out of scope** (Change 3/4): authoring prose in `docs-hydrate-memory`/`fab-continue`/`docs-reorg-*`;
  bulk-stamping/migrating the existing 20 memory files; stripping existing `## Changelog` tables.
- **Backward compat**: a partially-migrated tree still functions — a folder missing `log.md` or a file
  missing `type:` degrades gracefully (`fkf.md` §10, OKF permissive model).

## Open Questions

<!-- SRAD handles prioritization at apply entry. Just list the questions. -->

<!-- OQ2 (verb table) RESOLVED at clarify — see What Changes (a) verb derivation; assumption #8 confirmed. -->

<!-- OQ3 (generator vs. skill-authored frontmatter) RESOLVED at clarify — generator provides the
     mechanism only; bulk-stamp → Change 4, authoring → Change 3. See What Changes (b); assumption #10 confirmed. -->

- **OQ4 — new tier-2 loss categories.** *(Confirmed-default: treat as benign/tier-1 unless a concrete
  case surfaces at apply — assumption #9.)* Do `log.md` and FKF frontmatter introduce new
  destructive-loss categories (a hand-curated pre-FKF log the C-lite projection can't reproduce, a
  curated `type:`/`fkf_version` wiped), or is all `log.md`/frontmatter drift benign (tier 1)?
- **OQ5 — migration needed?** *(Confirmed-default: no new migration; regen seeds `log.md` —
  assumption #11.)* Change 1 already shipped `2.4.2-to-2.5.0.md` and bumped VERSION to 2.5.0. Does this
  generator change need its own migration (e.g. to seed `log.md`), or is regeneration via
  `fab memory-index` sufficient (the tree migration being Change 4's job)?

## Clarifications

### Session 2026-06-15

| # | Q / Action | Detail |
|---|------------|--------|
| 8 | Confirmed (with question) | Verb map confirmed (new→Creation, removed→Deprecation, else→Update). User asked *how the verb is used*: it is the leading bold token of each `log.md` entry, picked per-commit from `git log --name-status` (A/D/M) with `change_type` as a secondary hint; optional, so omit/`**Update**` is the fallback. Folded into What Changes (a). |
| 9 | Confirmed | `log.md` becomes a `--check` drift target; treat its drift as benign (tier 1) unless a concrete destructive case surfaces at apply (OQ4 stays as the open sub-question). |
| 10 | Confirmed (after explanation) | Generator-stamped vs. skill-authored `type: memory`: explained the single-writer boundary (generator owns index/log + the template/round-trip *mechanism*; topic-file authoring → Change 3, bulk-stamp → Change 4). User agreed with the assumption as written. |
| 11 | Confirmed | No new migration ships; `fab memory-index` regen seeds `log.md`; tree migration is Change 4. |

## Assumptions

<!-- STATE TRANSFER: sole continuity mechanism between intake-stage and apply-entry agents.
     All four SRAD grades recorded. Scores column required for every row. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Extend the existing `internal/memoryindex/` package + `cmd/fab/memory_index.go`; preserve the pure-`Render*` / I/O-`Gather` split. No new package. | Package exists; backlog names it; render/gather split is the established pattern (mirrors `internal/prmeta`). Codebase-deterministic. | S:95 R:80 A:95 D:90 |
| 2 | Certain | `log.md` output format is exactly `fkf.md` §6.2 (date-grouped newest-first, `# Log — {domain}` + generated-comment header, `- **Verb** [file](/bundle-rel) — summary (change-id)`). | Spec reproduces the format verbatim with a worked example. No interpretation latitude. | S:95 R:75 A:95 D:95 |
| 3 | Certain | `fkf_version: "0.1"` frontmatter goes on the **root** `index.md` only; `RenderRoot` prepends it. | `fkf.md` §8 is explicit: root index is the only index permitted frontmatter, value `"0.1"`. | S:95 R:85 A:95 D:95 |
| 4 | Certain | Change 1 dependency (`.status.yaml` `summary:` field) is satisfied — already merged (field, CLI verbs, VERSION 2.5.0, migration all present). | Verified by inspection of `statusfile.go`, `fab status set-summary`, `VERSION`, `src/kit/migrations/2.4.2-to-2.5.0.md`. | S:100 R:90 A:100 D:100 |
| 5 | Confident | Reuse the single batched `loadGitDates`/`git log` pass (extended to capture per-commit tuples) — no per-file `git log` spawns. | pw3k F34 explicitly collapsed per-file spawns into one batched pass; reintroducing them regresses that. Strong codebase signal; easily revisited. | S:75 R:70 A:85 D:75 |
| 6 | Confident | Summary-absent fallback projects the change **slug** in place of the descriptive line. | `fkf.md` §6.3 + backlog state this explicitly ("projects its slug in the log"). One obvious behavior. | S:85 R:80 A:90 D:90 |
| 7 | Confident | `type: memory` is round-tripped/stamped via the same frontmatter mechanism `RenderDomain` uses for `description:`. | §3.1 (constant frontmatter) + existing `RenderDomain` frontmatter round-trip is the established pattern. | S:70 R:70 A:80 D:70 |
| 8 | Confident | Verb mapping: new file → `**Creation**`, removed → `**Deprecation**`, else `**Update**`; signal = git per-commit name-status (A/D/M) with `change_type` as secondary hint; verb is optional, so omit/`**Update**` is the safe fallback. The verb is the bold leading word on each `log.md` entry (§6.2). | Clarified — user confirmed the mapping; verb-usage explained (it is the leading bold token per log entry, derived per-commit from `git log --name-status`). | S:95 R:65 A:55 D:45 |
| 9 | Confident | `log.md` files become `--check` targets classified for drift; treat `log.md`/frontmatter drift as **benign (tier 1)** unless a concrete destructive case is identified during apply. | Clarified — user confirmed. Whether a new tier-2 category is warranted stays open (OQ4); additive and reversible at apply. | S:95 R:60 A:55 D:50 |
| 10 | Confident | Generator provides the `type: memory` template/round-trip **mechanism** only (preserves the field when present; the memory-file template gains it); bulk-stamping the existing files is deferred to Change 4 (migration), authoring on new files to Change 3 (skills). The generator does NOT inject `type:` into topic files — that would break the single-writer separation (it owns index/log; humans/skills own topic files). | Clarified — user confirmed after explanation. §10 step 4 (teach the generator) is the mechanism; §10 step 1 (bulk add to every file) is Change 4's migration. | S:95 R:55 A:60 D:55 |
| 11 | Confident | No new migration file ships with this change; `fab memory-index` regeneration seeds `log.md`. The tree migration is Change 4 `[oovf]`. | Clarified — user confirmed. Change 1 already shipped the 2.5.0 migration + VERSION bump; this is generator plumbing. | S:95 R:65 A:60 D:55 |
| 12 | Confident | Commit → change-id mapping joins via the **change-folder/branch registry**: enumerate `fab/changes/*` (+ `archive/**`) for the `(change-id, folder)` set, correlate each memory commit to its change via git's branch/merge graph (branch == folder name); unattributable commits degrade gracefully (omit `(change-id)`, slug/`—` line). | Asked at Step 8 — user chose "join via .status.yaml history" (authoritative: the change owns its identity, vs. commit-message parsing). Grounded in the branch==change-id convention; `.status.yaml`/`.history.jsonl` carry no SHA, so the join is via git's own commit↔branch graph. Exact command shape is apply-time. | S:80 R:60 A:75 D:75 |

12 assumptions (4 certain, 8 confident, 0 tentative, 0 unresolved).
