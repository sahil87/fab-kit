# Intake: Publish FKF Standard to docs/site

**Change**: 260720-g538-publish-fkf-standard-site
**Created**: 2026-07-20

## Origin

Conversational (`/fab-discuss` session, 2026-07-20). The user asked where the FKF standard is maintained and whether it can be published to shll.ai. Investigation established the publishing mechanism (shll.ai pulls every tool repo's `docs/site/**` tree daily and renders each page at `/<tool>/<path>` — the `readme-extraction` toolkit standard, checked live via `shll standards readme-extraction`). The user then drove two decisions explicitly:

> 1. Do we need 3 copies? … I am ok with docs/site/fkf.md being the authoritative copy.
> 2. Is [1] copied from [2] or [2] copied from [1]? … a build step can ensure it gets copied … or something like a lint rule can ensure it stays in sync?

Agreed design (user approved "yes, and after that go ahead with fab-fff"):

- `docs/site/fkf.md` — **authoritative** normative standard, published at `https://shll.ai/fab-kit/fkf`
- `src/kit/reference/fkf.md` — **byte-copy** of it (shipped to the kit cache; deployed skills keep citing `$(fab kit-path)/reference/fkf.md`)
- `docs/specs/fkf.md` — slims to **rationale + history + pointer**, zero normative text (the `srad.md` / `srad-scoring-rationale-v1-to-v2.md` pattern)
- Sync direction [2]→[1] enforced by a **copy script + Go drift-guard test in CI**, not a release/build step

## Why

1. **Pain point**: FKF's normative rules currently live in two hand-synced variants — `docs/specs/fkf.md` (600-line design doc: rationale + history + normative sections) and `src/kit/reference/fkf.md` (373-line curated extract of §2/§3/§5/§6/§7/§8 with reworded framing). Both headers carry a "when you change FKF normative rules, update BOTH files" duty with **no mechanical enforcement** — divergence is silent until someone notices. Neither file is published: `docs/specs/` is explicitly invisible to shll.ai ("Only `README.md` and `docs/site/**` are ever pulled").
2. **Consequence of not fixing**: the standard stays unpublishable (no stable public URL for a format other repos — loom, run-kit — already consume via their `docs/memory/` trees), and the unenforced two-file sync duty eventually diverges.
3. **Why this approach**: publishing via `docs/site/` costs zero infrastructure — shll.ai auto-mounts the page (`docsSiteSidebarItems` in the site's astro config picks it up; no shll.ai-side change). Making the published page authoritative and the shipped copy a **byte-copy** turns the sync duty from "keep two prose variants aligned" into "run a one-line script", mechanically enforced. The repo has an **exact precedent**: `docs/site/skill.md` is canonical, `scripts/sync-skill.sh` copies it into the Go package, and a drift-guard test (`src/go/fab/cmd/fab/skill_test.go`) keeps the copy byte-honest on every `go test`. Rationale/history moves nowhere — it stays in `docs/specs/fkf.md`, which simply stops duplicating rules (mirroring how `srad.md` carries the contract and `srad-scoring-rationale-v1-to-v2.md` carries the why).

## What Changes

### 1. New `docs/site/fkf.md` — the authoritative standard

Create `docs/site/fkf.md` whose body is the current normative extract content (`src/kit/reference/fkf.md` sections §2 Conformance / §3 Concept Documents / §5 Index Files / §6 Log Files / §7 Cross-links / §8 Versioning — **original section numbers preserved**, exactly as the extract does today, so every existing citation like "FKF §3.2" / "§5" resolves identically) with a **rewritten shared header** replacing the extract's current "Single-sourcing note". The new header must read correctly in BOTH homes (published page and kit cache), stating:

- What FKF is (keep the current "What this is" + "Scope: `docs/memory/` only" blockquotes, which already carry the one-paragraph OKF framing)
- **This file is the canonical, authoritative FKF standard**, maintained at `docs/site/fkf.md` in the fab-kit repo, published at `https://shll.ai/fab-kit/fkf`, and shipped verbatim into the kit cache as `$(fab kit-path)/reference/fkf.md`
- Design rationale, OKF lineage, non-scope discussion, adoption/migration history, and glossary live in the non-normative companion — linked **absolutely**: `https://github.com/sahil87/fab-kit/blob/main/docs/specs/fkf.md`
- Section numbering is preserved from the original design doc (gaps are intentional)
- The edit workflow: edit `docs/site/fkf.md`, run `scripts/sync-fkf.sh`; a CI test fails on divergence

**Closed-set link audit** (readme-extraction standard, docs/site tree rules): every link in the page that leaves `docs/site/**` must be absolute `https://…` (the OKF spec link already is; the companion-doc link per above; audit any other relative targets in the extract body). No relative images (there are none today).

### 2. `src/kit/reference/fkf.md` — becomes a byte-copy

Replace its content with the byte-identical content of `docs/site/fkf.md` (same header, same body). Its path and section anchors do not change, so all deployed-skill citations (`docs-distill-memory.md`, `docs-hydrate-memory.md`, `docs-reorg-memory.md`, `fab-continue.md`, `git-pr.md`, `git-pr-review.md`, `_cli-fab.md` — all cite `$(fab kit-path)/reference/fkf.md` §N) remain valid **unchanged**. No skill file edits are required for citations.

### 3. New `scripts/sync-fkf.sh`

Mirror `scripts/sync-skill.sh` exactly (repo-root cd, `set -euo pipefail`, `cp -f`, one echo):

```bash
SRC="docs/site/fkf.md"
DEST="src/kit/reference/fkf.md"
cp -f "$SRC" "$DEST"
echo "synced FKF standard: $DEST"
```

Header comment explains: canonical file is `docs/site/fkf.md`; the committed copy ships in the kit; the drift-guard test keeps it byte-honest.

### 4. Go drift-guard test

A test asserting byte-equality of `docs/site/fkf.md` and `src/kit/reference/fkf.md`. Unlike `skill.md` (which is `go:embed`ed and guarded in `cmd/fab/skill_test.go`), `reference/fkf.md` is **not embedded** — both files live outside the module root (`src/go/fab/`), so the test reads both via repo-relative paths from the test file's location (precedent: `src/go/fab/internal/score/changetypes_doc_test.go` reads repo docs the same way). Place the test in the `fab` module (e.g., a small `fkf_sync_test.go` near the doc-conformance test precedent, or `cmd/fab` — implementer's choice following the existing pattern). Failure message must name the fix: `run scripts/sync-fkf.sh`. Runs automatically in CI (`ci.yml` runs `go test`).

### 5. `docs/specs/fkf.md` — slims to non-normative rationale + pointer

- **Header rewritten**: this is the non-normative design companion; the normative standard lives at `docs/site/fkf.md` (published at `https://shll.ai/fab-kit/fkf`, shipped as `$(fab kit-path)/reference/fkf.md`). The old "Shipped normative extract / update BOTH files" note is replaced by the new sync contract (edit `docs/site/fkf.md` → `scripts/sync-fkf.sh` → CI test enforces).
- **Keeps** (original numbering, with intentional gaps): §1 Relationship to OKF (full profile table + prose), §4 Bundle Organization, §9 Non-Scope, §10 Adoption/Migration, §11 Glossary, and all design-history/rationale prose.
- **Removes** the normative sections §2/§3/§5/§6/§7/§8, each replaced by a one-line pointer stub (e.g., `## 2. Conformance — moved` → see the standard) so inbound anchors don't silently 404 on GitHub and readers are redirected. Zero normative text remains.
- The existing citation `docs-hydrate-memory.md:192` → "`docs/specs/fkf.md` §10 item 2" stays valid (§10 remains).

### 6. Reference sweep (Sibling & Mirror Sweeps discipline)

Grep repo-wide for `specs/fkf` and `reference/fkf` and update every description of the **arrangement** (not the citations, which stay valid):

- `docs/specs/index.md` — fkf row description: now "design rationale + history companion; normative standard at docs/site/fkf.md → shll.ai/fab-kit/fkf"
- `docs/memory/distribution/kit-architecture.md:62` — `reference/fkf.md` provenance comment ("Shipped FKF normative extract (§2/§3/§5/§6/§7/§8)…" → byte-copy of `docs/site/fkf.md`, synced by `scripts/sync-fkf.sh`, drift-guarded) — this is the hydrate-stage memory edit
- Any other file restating "extract of docs/specs/fkf.md" / "update BOTH files" (grep confirms: the two fkf.md headers themselves; `docs-distill-memory.md:33`'s parenthetical "(deployed skills reach the extract; the dev-repo docs/specs/fkf.md is absent in user repos)" stays true and needs no edit — verify during apply)

If any `src/kit/skills/*.md` file does end up edited, its `docs/specs/skills/SPEC-*.md` mirror MUST be updated in the same change (constitution).

### 7. README cross-link

Add the FKF standard to the README's docs-hub links (line 11's link row and/or the docs section), written naturally as `docs/site/fkf.md` (the site rewrites it to `/fab-kit/fkf`; GitHub resolves it) — per readme-extraction standard rule 8 ("the README is the hub").

## Affected Memory

- `distribution/kit-architecture.md`: (modify) — update the `reference/fkf.md` provenance line (§ kit tree diagram, line ~62) and note the `scripts/sync-fkf.sh` + drift-guard mechanism alongside the existing skill.md sync entry if one exists
- `memory-docs/templates.md`: (modify) — only if it restates the extract arrangement; verify during hydrate, likely no-op

## Impact

- **New files**: `docs/site/fkf.md`, `scripts/sync-fkf.sh`, one Go test file (`src/go/fab/...`)
- **Rewritten**: `src/kit/reference/fkf.md` (byte-copy), `docs/specs/fkf.md` (slimmed ~600 → ~250 lines)
- **Touched**: `README.md`, `docs/specs/index.md`, `docs/memory/distribution/kit-architecture.md` (hydrate)
- **No changes**: fab CLI surface (no `_cli-fab.md` update needed), skill behavior, shll.ai repo (auto-pull), kit packaging/release (reference/ already ships)
- **Tests**: the new drift-guard test IS the test update; run `go test ./...` scoped to the affected package plus a full-module pass
- **Publication timing**: the page appears on shll.ai after the site's next scheduled refresh (daily) or a manual `refresh-readme.yml` dispatch — no action required from this repo

## Open Questions

*(none — decisions resolved in the originating discussion)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `docs/site/fkf.md` is authoritative; `src/kit/reference/fkf.md` is a byte-copy; `docs/specs/fkf.md` becomes non-normative rationale + pointer | Discussed — user explicitly approved ("I am ok with docs/site/fkf.md being the authoritative copy") | S:95 R:70 A:95 D:95 |
| 2 | Certain | Sync = copy script + Go drift-guard CI test, direction [2]→[1]; no release/build-step writing into src/ | Discussed — user asked "build step or lint rule?", recommendation (test-style enforcement) approved; exact repo precedent `scripts/sync-skill.sh` + `skill_test.go` | S:90 R:85 A:95 D:90 |
| 3 | Confident | Published standard keeps the current extract's section set (§2/§3/§5/§6/§7/§8) with original numbering; §1/§4/§9–§11 stay in the rationale doc | Extract set exists to keep agent-loaded content lean; numbering preservation keeps every existing "FKF §N" citation valid; changing the normative set is out of scope | S:60 R:75 A:80 D:70 |
| 4 | Confident | One shared header framing readable in both homes replaces the extract's "Single-sourcing note" | Byte-copy requires it; content specified in What Changes §1 | S:55 R:85 A:85 D:75 |
| 5 | Confident | New standalone `scripts/sync-fkf.sh` mirroring `sync-skill.sh` (not a generalized multi-file sync script) | Matches the one-file-per-script precedent; trivial to generalize later | S:50 R:90 A:75 D:60 |
| 6 | Confident | Drift-guard test reads both files via repo-relative paths (no embed), placed per the `changetypes_doc_test.go` precedent | `reference/fkf.md` is not embedded in the binary — equality of two repo files is all that's needed | S:55 R:85 A:80 D:70 |
| 7 | Confident | README gains a `docs/site/fkf.md` cross-link | readme-extraction standard rule 8: README is the hub, cross-links its docs/site pages | S:50 R:95 A:85 D:75 |
| 8 | Certain | All links leaving `docs/site/**` in the new page are absolute `https://…` | readme-extraction standard closed-set rules 1–2 (verified live via `shll standards readme-extraction`) | S:70 R:90 A:90 D:85 |
| 9 | Confident | Removed normative sections in `docs/specs/fkf.md` leave one-line pointer stubs at their original heading anchors | Prevents silent 404s for inbound GitHub anchors; cheap; keeps the "zero normative text" property | S:45 R:90 A:80 D:65 |

9 assumptions (3 certain, 6 confident, 0 tentative, 0 unresolved).
