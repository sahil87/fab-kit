# Intake: Ship FKF Normative Contract to Kit Cache

**Change**: 260616-frlo-ship-fkf-contract-to-kit-cache
**Created**: 2026-06-16

## Origin

> ship the FKF normative contract into the kit cache and repoint skill citations
>
> Surfaced in a `/fab-discuss` session: "Right now in the documentation skills we refer to the FKF
> specification. However in the repo that uses fabKit I don't think we are even shipping the FKF
> spec and the corresponding documents like log.md / log.seed.md (which the spec might refer to).
> Should these not get shipped along with fabKit?"

**Interaction mode**: Conversational. The discussion separated the question into three parts and
resolved each:

1. **log.md / log.seed.md** — confirmed these are *correctly not shipped*. `log.md` is a generated
   artifact produced in-repo by `fab memory-index` (same category as `index.md`); `log.seed.md` is
   an optional user-curated sidecar. The binary that creates them already ships. **Not a gap.**
2. **The FKF spec itself** — confirmed this *is* a real gap. `docs/specs/fkf.md` lives only in the
   fab-kit dev repo and never ships, yet deployed skills cite it as load-bearing authority.
3. **Where the contract should live** — the user proposed the kit cache (the `$(fab kit-path)`
   installation site, like templates) rather than a deployed skill helper or the user's
   `docs/specs/`. Agreed, with rationale (see What Changes §1).

**Key decisions reached during the discussion** (encoded as assumptions below):
- Citations are *user-reachable authority*, not dev-repo provenance (user-selected). The contract
  must therefore be reachable in a user's repo.
- Home = kit cache, read via `$(fab kit-path)/.../fkf.md` (user-selected over a deployed `_fkf.md`
  helper or `docs/specs/`).
- Content = extract only the *normative subset* (user-selected over shipping the spec verbatim).
- Packaging blast radius confirmed **zero** during intake: both `just install`
  (`rsync -a --delete src/kit/ → cache`) and the release path (`just dist-kit`:
  `cp -a src/kit/. dist/kit/` → archived whole) copy the entire `src/kit/` tree verbatim. A new
  file under `src/kit/` ships automatically — no Go change, no packaging-list edit.

## Why

**Problem.** Deployed doc skills cite `docs/specs/fkf.md §X` as the normative authority an agent
should follow when authoring memory files. There are **7 citations across 5 deployed skills**
(`fab-continue.md`, `docs-hydrate-memory.md`, `docs-reorg-memory.md`, `docs-reorg-specs.md`,
`_cli-fab.md`), and they are genuinely load-bearing — not provenance notes. Examples of the rules
they defer to:

- frontmatter shape: `type: memory` constant + curated `description:` (FKF §3.1–§3.2)
- no per-file `## Changelog` (FKF §3.3)
- stub-`index.md`-before-`fab memory-index` (FKF §5)
- per-folder `log.md` + the `.status.yaml` `summary:` source (FKF §6 / §6.3)
- bundle-relative memory↔memory cross-links (FKF §7)
- the root-index `fkf_version` frontmatter (FKF §8)

**But the cited file never ships.** `docs/specs/fkf.md` exists only in this dev repo. `fab init`'s
scaffold seeds a bare `docs/specs/index.md` and nothing else; no file under `src/kit/` carries the
FKF spec. So in a user's project, every one of those 7 citations is a **dangling reference** — a
user-side agent running `/fab-continue` (hydrate), `/docs-hydrate-memory`, or `/docs-reorg-memory`
is told to follow `docs/specs/fkf.md §7`, opens the path, and finds nothing.

**Consequence if unfixed.** The skill prose inlines *most* of each rule, so behavior degrades rather
than breaks outright — but the single-source-of-truth anchor is invisible, the agent cannot verify
the full rule, and any rule detail the prose omits is simply unreachable. This is precisely the
drift FKF was introduced to prevent, reappearing at the distribution boundary.

**Why this approach over alternatives.** Three homes were weighed during the discussion:

- *Ship into the user's `docs/specs/fkf.md`* — rejected: violates **Constitution VI** (specs are
  human-curated, must not be auto-generated/planted by tooling). It would also masquerade fab-kit's
  own internal spec as the user's design intent.
- *Deployed `.claude/skills/_fkf.md` helper* — workable, but the deployed copy can drift from the
  binary if the user doesn't re-sync, and it would be an always-loaded helper for a point-of-use
  citation.
- *Kit cache, read via `$(fab kit-path)`* — **chosen.** It is version-pinned to the installed binary
  (the contract always matches the `fab memory-index` behavior that produces `log.md`/`fkf_version`),
  it stays out of the user's curated `docs/specs/` (Constitution VI clean), and it reuses the proven
  existing pattern where skills already read `$(fab kit-path)/templates/intake.md` and
  `.../templates/plan.md` on demand at point of use.

## What Changes

### 1. New shipped FKF contract under `src/kit/` (read via `$(fab kit-path)`)

Add a new normative FKF contract file inside the kit tree so it ships to the cache and is reachable
in every user repo via `$(fab kit-path)`.

- **Location**: `src/kit/reference/fkf.md` (a new `reference/` sibling to `templates/`,
  `migrations/`, `scaffold/`, `skills/`, `bin/`). A new dedicated dir is used over dropping the
  file into `templates/` because the FKF contract is *reference material to read*, not an *artifact
  template to instantiate* — mixing it into `templates/` would muddy that directory's single
  purpose. <!-- clarified: reference/ dir confirmed over templates/ — user-selected in clarify session -->
- **Read path in skills**: `$(fab kit-path)/reference/fkf.md` — exactly mirroring how
  `_generation.md` / `_intake.md` already read `$(fab kit-path)/templates/intake.md`.
- **Content = the normative subset only.** Extract the rules an agent must follow:
  - §2 Conformance
  - §3 Concept documents (frontmatter `type`/`description`, no-`## Changelog` body rule, §3.4
    optional frontmatter)
  - §5 Index files (generated; stub-before-index)
  - §6 Log files (C-lite model, format, the `summary:` source field)
  - §7 Cross-links (bundle-relative)
  - §8 Versioning (`fkf_version`)
  Leave in `docs/specs/fkf.md` (dev-repo design doc): §1 OKF relationship/lineage, §4 prose
  rationale, §9 Non-Scope (`docs/specs/`), §10 Adoption/Migration history, §11 Glossary — the
  "why"/history, not the "what an agent must do."
- **Section numbering**: the shipped contract MUST preserve the §-anchors the citations use
  (§3.1, §3.2, §3.3, §5, §6, §6.3, §7, §8) so the repointed citations resolve to the right section.
  <!-- assumed: preserve original § numbers in the extracted subset — Confident -->

### 2. Single-sourcing discipline note (anti-drift)

The shipped contract and `docs/specs/fkf.md` now both state the normative rules — a drift risk.
Add an explicit single-sourcing note so they cannot silently diverge:

- A short header note in `src/kit/reference/fkf.md` declaring it the **shipped extract** of
  `docs/specs/fkf.md` (the dev-repo design doc), with a one-line "when you change FKF normative
  rules, update both" instruction.
- A reciprocal pointer in `docs/specs/fkf.md` naming `src/kit/reference/fkf.md` as the shipped
  normative extract.
- Decision: keep two files (extract + design doc) rather than collapse — the extract is what ships
  and stays tight; the design doc keeps the rationale/history. The note is the chosen mitigation for
  the duplication this introduces.

### 3. Repoint the 7 skill citations

Change every load-bearing `docs/specs/fkf.md §X` citation in the deployed skills to the
`$(fab kit-path)/reference/fkf.md §X` form. Skills (canonical source `src/kit/skills/`):

- `src/kit/skills/fab-continue.md` — 1 citation (hydrate behavior, line ~195)
- `src/kit/skills/docs-hydrate-memory.md` — 3 citations (§3.1–§3.2 twice, §3.1 backfill)
- `src/kit/skills/docs-reorg-memory.md` — citations to §7 / §3 (FKF-aware moves)
- `src/kit/skills/docs-reorg-specs.md` — §9 / Constitution VI references (the "no FKF frontmatter on
  specs" note — verify whether these should point at the shipped extract or stay pointed at the
  dev-repo design doc, since §9 Non-Scope is *not* in the shipped subset)
- `src/kit/skills/_cli-fab.md` — its FKF reference block (documents the generated half: `log.md`,
  `type: memory` round-trip, `fkf_version`)

**Nuance — §9 references (resolved).** `docs-reorg-specs.md` cites FKF **§9 (Non-Scope: `docs/specs/`)**,
which is rationale, *not* part of the normative subset shipping in `reference/fkf.md`. **Decision: rewrite
that citation to stand on Constitution VI alone** and drop the §9 anchor. Constitution VI (specs are
human-curated, not auto-generated) is the rule's real authority and already ships in every user repo
(`fab/project/constitution.md`), so the "no FKF frontmatter on specs" note loses nothing. No §9 stub
ships in `reference/fkf.md` — the normative subset stays §2/§3/§5/§6/§7/§8.
<!-- clarified: §9 citation rewritten onto Constitution VI (no §9 stub) — user-selected in clarify session -->

### 4. SPEC-mirror updates (Constitution requirement)

The constitution requires: changes to `src/kit/skills/*.md` MUST update the corresponding
`docs/specs/skills/SPEC-*.md`. Mirrors confirmed to exist for the four `/skill` files (the
`_cli-fab.md` reference helper is correctly excluded from SPEC mirrors per the naming policy):

- `docs/specs/skills/SPEC-fab-continue.md`
- `docs/specs/skills/SPEC-docs-hydrate-memory.md`
- `docs/specs/skills/SPEC-docs-reorg-memory.md`
- `docs/specs/skills/SPEC-docs-reorg-specs.md`

Each gets the citation-target update reflected (the §X reference now points at the shipped contract).

### 5. Memory update — kit cache layout

`docs/memory/distribution/kit-architecture.md` documents the kit-cache structure (the "underscore
file ecosystem" / which dirs live under `src/kit/`). Adding `reference/` as a new shipped content
dir is a spec-level fact and must be recorded there. The packaging invariant ("the whole `src/kit/`
tree is copied verbatim, so new content ships automatically") is worth stating explicitly here too,
since it is the reason no Go/packaging change is needed.

### Non-Goals

- **No migration.** Existing user repos need none: `fab sync` re-deploys the repointed skills on
  upgrade, and the contract is read from the version-pinned cache that `fab upgrade-repo` already
  populates. Confirm during the plan, but the expectation is *no* `src/kit/migrations/` file.
- **No Go/binary change.** Packaging copies `src/kit/` whole (verified at intake). `fab memory-index`
  behavior is unchanged — this change ships documentation the binary's behavior already implies.
- **Not touching `log.md` / `log.seed.md` shipping.** Confirmed out of scope — they are correctly
  generated/curated in-repo, not kit content.
- **Not removing `docs/specs/fkf.md`.** It stays as the dev-repo design doc (rationale + history).

## Affected Memory

- `distribution/kit-architecture`: (modify) record the new `src/kit/reference/` shipped content dir
  and the "src/kit/ copied verbatim → new content ships automatically" packaging invariant.

<!-- The skill-behavior changes are reflected in the docs/specs/skills/SPEC-*.md mirrors (curated,
     not memory). No other memory domain changes: this is a documentation-distribution fix, not a
     behavioral change to the pipeline, runtime, or memory-docs authoring rules. -->

## Impact

- **New file**: `src/kit/reference/fkf.md` (shipped via the verbatim `src/kit/` copy).
- **Edited (skills, canonical source `src/kit/skills/`)**: `fab-continue.md`, `docs-hydrate-memory.md`,
  `docs-reorg-memory.md`, `docs-reorg-specs.md`, `_cli-fab.md` — citation repointing only; behavior
  unchanged.
- **Edited (specs)**: `docs/specs/fkf.md` (reciprocal single-sourcing pointer) and the 4 SPEC mirrors.
- **Edited (memory)**: `docs/memory/distribution/kit-architecture.md`.
- **Deploy mechanics**: after editing `src/kit/skills/*.md`, `fab sync` redeploys to `.claude/skills/`
  (never hand-edit deployed copies — Constitution V).
- **No code, no tests, no packaging, no migration** (verified at intake). This is content-only.
- **Verification**: after `fab sync`, confirm no remaining `docs/specs/fkf.md` citation in deployed
  skills that is meant to be user-reachable, and that `$(fab kit-path)/reference/fkf.md` resolves to a
  readable file in a fresh install.

## Open Questions

_Both prior open questions resolved in the 2026-06-16 clarify session:_

- ~~Exact extraction boundary for the §9 (Non-Scope) citation in `docs-reorg-specs.md`~~ → **Resolved**:
  rewrite the citation onto Constitution VI; drop the §9 anchor; no §9 stub ships.
- ~~Directory name: `reference/` vs. another kit-cache home~~ → **Resolved**: `src/kit/reference/fkf.md`.

## Clarifications

### Session 2026-06-16

| Q | Answer |
|---|--------|
| Where in the kit tree should the shipped FKF contract live? (`reference/` vs `templates/` vs kit `docs/`) | New `src/kit/reference/fkf.md` dir — keeps `templates/` for instantiable artifacts; contract is reference-to-read. (Assumption #9 → Confident) |
| How should `docs-reorg-specs.md`'s FKF §9 (Non-Scope) citation resolve, given §9 is not in the shipped normative subset? | Rewrite onto Constitution VI alone; drop the §9 anchor; no §9 stub ships. (Assumption #10 → Confident) |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | log.md / log.seed.md are out of scope — they are generated/curated in-repo, not kit content | Established in discussion; log.md is `fab memory-index` output (like index.md), log.seed.md is a user sidecar; the generating binary already ships | S:95 R:90 A:95 D:95 |
| 2 | Certain | No Go/packaging/enumeration change needed — a new `src/kit/` file ships automatically | Verified at intake: `just install` rsyncs `src/kit/` whole; `just dist-kit` does `cp -a src/kit/.` then archives the whole tree | S:90 R:85 A:100 D:95 |
| 3 | Certain | Citations are user-reachable authority (not provenance), so the contract must ship to user repos | User-selected explicitly in discussion | S:100 R:80 A:90 D:100 |
| 4 | Confident | Home = kit cache via `$(fab kit-path)/reference/fkf.md`, not a deployed `_fkf.md` helper nor `docs/specs/` | User-selected; version-pinned to binary, Constitution VI clean, reuses the `$(fab kit-path)/templates/` read pattern | S:90 R:70 A:85 D:85 |
| 5 | Confident | Content = normative subset (§2/§3/§5/§6/§7/§8); leave §1/§4/§9/§10/§11 rationale in docs/specs/fkf.md | User-selected "extract the normative subset"; section headings inspected at intake to draw the boundary | S:85 R:75 A:80 D:80 |
| 6 | Confident | Shipped extract preserves original §-anchors so repointed citations resolve | Citations reference specific § numbers; preserving them is the low-risk default | S:80 R:80 A:90 D:85 |
| 7 | Confident | No migration for existing user repos | `fab sync` redeploys skills on upgrade; contract read from version-pinned cache; matches the "skills re-deploy, no data restructure" pattern | S:75 R:70 A:85 D:80 |
| 8 | Confident | SPEC mirrors for the 4 `/skill` files must be updated; `_cli-fab.md` excluded | Constitution skill↔SPEC rule; mirror existence + `_cli-fab` exclusion verified at intake | S:90 R:75 A:95 D:90 |
| 9 | Confident | New `reference/` dir (`src/kit/reference/fkf.md`) rather than dropping fkf.md into `templates/` | Clarified — user confirmed; `templates/` is for instantiable artifacts, reference-to-read differs (recomputed composite 72.75 → Confident) | S:95 R:65 A:70 D:60 |
| 10 | Confident | §9 (Non-Scope) citation in docs-reorg-specs.md rewritten onto Constitution VI; no §9 stub ships | Clarified — user confirmed; Constitution VI is the rule's real authority and already ships (recomputed composite 72.0 → Confident) | S:95 R:70 A:65 D:55 |

10 assumptions (3 certain, 7 confident, 0 tentative, 0 unresolved).
