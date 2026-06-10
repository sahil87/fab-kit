# Intake: README Site-Extraction Conformance

**Change**: 260605-fcrp-readme-site-extraction-conformance
**Created**: 2026-06-05
**Status**: Draft

## Origin

> we need to reorganize README.md and other related docs in accordance with
> `<shll.ai>/docs/specs/readme-extraction-contract.md` (shll.ai is at `~/code/sahil87/shll.ai`).
> shll.ai/fab-kit becomes the online documentation website for fab-kit. It takes its content
> from this repo.

**Mode**: conversational — three scoping decisions were resolved with the user before intake
generation (see Assumptions). The contract spec and the *live already-pulled slice* on shll.ai
(`content/fab-kit/README.md`) were both read as ground truth, so the gaps below are observed, not
hypothesized.

## Why

shll.ai/fab-kit is fab-kit's public documentation site. Its per-tool README page renders a
**deduced, curated slice** of *this* repo's `README.md`, pulled daily by shll.ai's
`scheduled-readme-refresh.yml` and rendered at build time by `ReadmeSlice.astro`. The tool repo is
**canonical** — shll.ai never hand-edits the prose. That means any structural defect in our README
shows up verbatim on the public site, and the only place to fix it is *here*.

The consumer side is **already wired and live**: `content/fab-kit/README.md` exists on shll.ai, the
refresh job maps the `fab-kit:fab-kit` slug, and `/tools/fab-kit/{overview,readme,commands,install,workflows}`
pages exist. So this is a pure **producer-side conformance** task (exactly the "forward, per-repo,
gradual" activity the contract's out-of-scope boundary describes) — not a shll.ai change.

Inspecting the live pulled slice surfaces three defects that are **currently shipping on the public
site**:

1. **Both diagrams have silently vanished.** The README has two ` ```mermaid ` fences (the 6-stage
   pipeline flowchart and the stage-coverage diagram). §5/§6 of the contract **strip inline mermaid
   on pull** because Astro Starlight does not render mermaid. The live slice confirms it:
   `grep -c '```mermaid' content/fab-kit/README.md` → `0`. The prose still says "moves through six
   stages:" and then — nothing. The site shows an empty gap where each diagram should be.

2. **~13 repo-relative links 404 on the site.** Links like `[Glossary](docs/specs/glossary.md)` and
   `[Contributing](CONTRIBUTING.md)` resolve on GitHub but break on shll.ai. Confirmed by reading
   `ReadmeSlice.astro`: it renders the slice with `createMarkdownProcessor({})` and performs **no
   relative-path rewriting** — a relative href is emitted as-is and dead-ends on the site.

3. **No tail boundary — the entire README flows to the site.** §2 ends the slice at the first
   denylisted heading (`Contributing`/`Development`/`Building`/`License`/`Acknowledgements`). Our
   README has **none** of these, so the slice runs to EOF, dragging GitHub-oriented tail content
   (the 130-row "Stage Coverage by Command" matrix, "Companion tools", "Learn More" link farm) onto
   the public page where it reads as noise.

If we don't fix this: the flagship tool's own doc site looks broken (missing diagrams, dead links)
while every *other* tool's README is conformed — fab-kit would be the worst-looking page on its own
author's site. The cost is low and entirely local to this repo.

**Why this approach.** The contract's §5 explicitly prescribes the diagram fix we chose ("commit a
rendered SVG for the site; keep mermaid for GitHub"). The link and tail fixes are mechanical
applications of §1/§2/§3. We are conforming *to* the published contract, not inventing structure.

## What Changes

### 1. Diagrams: render to SVG, reference by absolute URL, keep mermaid (§5)

Both ` ```mermaid ` fences stay in `README.md` (GitHub renders them natively, zoomable). In
**addition**, each diagram is committed as a rendered **SVG** and referenced immediately adjacent to
its fence so the site — which strips the fence — still shows the diagram.

- New committed assets: `docs/img/pipeline-stages.svg` and `docs/img/stage-coverage.svg` (or
  similar). SVG is chosen per §4 for dark-theme control.
  <!-- clarified: docs/img/ confirmed as the asset dir -->
- **Why SVGs are still required** (a clarify-session correction): it was suggested mermaid might
  already be supported in shll.ai's Astro theme — but that is **false**. There is no
  `mermaid`/`rehype-mermaid` dependency in the site's `package.json` or `astro.config.mjs`, and the
  canonical puller (`extract-readme.ts`) still strips every ` ```mermaid ` block on pull (the live
  `content/fab-kit/README.md` slice contains 0 fences). Mermaid renders on **GitHub** (why we keep
  the fence) but not on shll.ai. The SVG is the only thing that survives onto the site.
- **SVGs are hand-authored once** (export the two diagrams from the mermaid source via mermaid.live
  or a local `mmdc` run), committed to `docs/img/`, with no automation. The ` ```mermaid ` fence in
  the README stays as the canonical diagram source; the SVG is a one-time render for the site. Only
  two low-churn diagrams, so this is Constitution-I-pure (no CI step, no build pipeline). Accepted
  cost: a future mermaid edit requires a manual SVG re-export, and the two can drift — mitigated by
  the fence being the source of truth. <!-- clarified: hand-author SVGs once -->
- **Image references MUST be absolute raw URLs**, not repo-relative. The renderer does not rewrite
  `src`, so `![alt](docs/img/x.svg)` would 404 on the site. Use
  `![6-stage pipeline](https://raw.githubusercontent.com/sahil87/fab-kit/main/docs/img/pipeline-stages.svg)`.
  This resolves on both GitHub and the site (§3: "referenced by repo URL — alt text travels,
  single-source"). Alt text is mandatory and authored to read correctly on both surfaces.

Pattern per diagram:

````markdown
```mermaid
flowchart TD
  ...
```

![6-stage pipeline: intake → apply → review → hydrate → ship → review-PR](https://raw.githubusercontent.com/sahil87/fab-kit/main/docs/img/pipeline-stages.svg)
````

On GitHub the reader sees the live mermaid (the raw-URL `<img>` is a redundant duplicate — acceptable,
or hidden behind a `<picture>`/comment if it reads poorly <!-- assumed: dual-render acceptable; refine in plan -->).
On the site the mermaid is stripped and only the SVG survives.

### 2. Repo-relative links → absolute (§ renderer behavior)

Every repo-relative href in the **slice region** (head boundary → tail boundary) is rewritten to an
absolute URL so it survives on the site. ~13 links today:
`docs/specs/{glossary,companions,assembly-line,srad,operator,overview,user-flow,skills}.md` and
`CONTRIBUTING.md`.

**Resolved → absolute GitHub-blob URLs** (`https://github.com/sahil87/fab-kit/blob/main/docs/specs/glossary.md`).
<!-- clarified: GitHub-blob chosen; fully-relative links verified non-viable -->

**Why not fully-relative links** (a clarify-session question): a single relative href *cannot*
resolve correctly in both locations, because the README sits at a different base in each:
- On GitHub the README is at the repo root, so `docs/specs/glossary.md` → `.../blob/main/docs/specs/glossary.md` ✓
- On shll.ai the slice renders at route `/tools/fab-kit/readme`, and `createMarkdownProcessor({})`
  emits the href verbatim (no base rewrite), so `docs/specs/glossary.md` → `shll.ai/tools/fab-kit/readme/docs/specs/glossary.md` ✗ (404)

The relative base differs (`/` vs `/tools/fab-kit/readme/`), so no relative string is simultaneously
correct. Links must be **absolute**. GitHub-blob is chosen over shll.ai-page URLs because every target
provably exists in this repo (no anchor→page URL map to build or keep fresh), matching §3's
single-source principle for images. Tradeoff accepted: a site reader following a deep doc link bounces
to GitHub — mild, and most `docs/specs/*.md` files have no 1:1 site page anyway.

**In-page anchor links** (22, e.g. `[Quick Start](#quick-start)`) are left as-is: they target
headings that survive in the slice, and Starlight re-slugs rendered headings consistently, so they
resolve on the readme page. (Anchors pointing at sections that get fenced into the tail — see §3 —
must be re-pointed or removed.)

### 3. Tail boundary: fence GitHub-only content behind a denylisted heading (§2)

**Resolved → cut at `## Development`.** Move "Stage Coverage by Command", "Companion tools", and
"Learn More" **below** a `## Development` heading (a §2 denylisted heading). The slice ends there.
Everything above stays in the slice (the diagrams, "Why Fab Kit", the 5 Cs, Command Quick Reference,
Quick Start). Relocated content may additionally point into `CONTRIBUTING.md` / `docs/` pages.
<!-- clarified: tail cut at ## Development confirmed -->

The slice that ships to the site should end on genuinely user-facing prose ("Why Fab Kit", the 5 Cs,
Command Quick Reference) and stop before GitHub-native chrome.

### 4. docs/ reorganization for the §9 audience axis (forward-looking)

**Resolved → full reorg now.** Per the user's decision and §9's intended model, reorganize `docs/`
along the **audience axis**, including migrating real content (not just scaffolding):

- **`docs/site/`** — site-only prose that should not live in the README but *should* reach shll.ai.
  Content is migrated here now (e.g. extended narrative / examples pulled out of the README to keep
  the slice tight).
- **`docs/internal/`** — maintainer/design notes that must **never** reach the site (distinct from
  `docs/specs/`, which fab already defines as pre-implementation design intent — Constitution VI).
- `docs/img/` — rendered diagram assets (from change 1).
- `docs/memory/`, `docs/specs/` — unchanged (existing fab semantics).

> **Flagged tradeoff (accepted knowingly, kept visible).** §9 (`docs/site/` pull path) is marked
> **"RESERVED / NOT YET IMPLEMENTED"** in the contract — shll.ai's puller fetches *only* `README.md`
> today; nothing reads `docs/site/`. So content migrated into `docs/site/` **will not appear on the
> site** until a future shll.ai change ships the §9 pull+render path. The user chose "Full reorg now"
> with this understood — the structure lands ahead of its consumer. **Risk to manage in plan:** do
> not move README content that users currently rely on *into* `docs/site/` such that it disappears
> from *both* GitHub-README-readers *and* the (not-yet-pulling) site — i.e. anything moved out of the
> README slice must still be reachable (a GitHub-blob link to `docs/site/*.md` works on GitHub even
> while the site can't pull it). <!-- clarified: full reorg incl. content migration — §9-unimplemented risk noted -->

### Out of scope

- **No shll.ai changes.** The consumer (puller, renderer, pages) is complete and live. We only
  change this repo.
- **No `extract-readme.ts` changes.** The extraction mechanics are canonical on the shll.ai side.
- The §7 divergence reporter is report-only; we are not gating on it. (Our README's command examples
  should still match `help/fab-kit.json` to avoid a `::warning::`, but that is not the focus.)

## Affected Memory

- `fab-workflow/distribution`: (modify) — fab-kit's public-docs surface (shll.ai/fab-kit) and the
  producer-side README contract obligation. <!-- assumed: distribution is the closest existing domain for "how fab-kit is published"; confirm at hydrate -->
- `conventions/readme-extraction`: (new) — fab-kit's own copy of the README-structure rules it must
  keep conformant (head/tail/diagram/link conventions). Mirrors shll.ai's consumer-side memory of
  the same contract. <!-- assumed: a new conventions domain; verify domain exists at hydrate -->

## Impact

- **`README.md`** — primary surface. Diagram references, link rewrites, tail-boundary heading.
- **`docs/img/*.svg`** (new) — committed rendered diagrams.
- **`docs/site/`, `docs/internal/`** (new dirs) — forward audience-axis structure.
- **`CONTRIBUTING.md`** — possible destination for relocated GitHub-only tail content.
- **`.github/workflows/`** — possibly a new mermaid→SVG render step (repo currently has only
  `release.yml`); or a local script. Constitution I tension (no build steps) — a CI render step that
  commits SVGs is acceptable since the SVG, not a build pipeline, is the artifact.
- **shll.ai** — *no changes*; verification only (confirm the next pull renders diagrams + working
  links on `/tools/fab-kit/readme`).

## Open Questions

*Resolved in the 2026-06-05 clarify session: link target (→ GitHub-blob absolute), tail cut line
(→ `## Development`), asset dir (→ `docs/img/`), reorg depth (→ full reorg incl. content migration).
The "mermaid already supported" premise was refuted — SVGs remain required.*

SVG render mechanism (#11) resolved → **hand-author the two SVGs once**, no automation.

Remaining (editorial, refine in plan — not gating):

- Does the dual render (live mermaid + adjacent SVG `<img>`) read acceptably on GitHub, or should the
  mermaid be wrapped (e.g. in a `<details>`) so only one shows per surface? Editorial; refine in plan.
- Exactly *which* README content (if any) migrates into `docs/site/` vs. stays in the slice — the
  reorg is full, but the specific content moves are an editorial call for plan.

## Clarifications

### Session 2026-06-05

| # | Question | Resolution |
|---|----------|------------|
| 8 | Link target: GitHub-blob vs shll.ai-page vs mixed — or can links stay fully relative? | Fully-relative **rejected** (verified: README base differs by location, no relative string resolves in both). → **GitHub-blob absolute** URLs. |
| 9 | docs/ reorg depth: scaffold-only / skip / full? | **Full reorg now** — create `docs/site/` + `docs/internal/` + `docs/img/` and migrate real content. §9-unimplemented risk noted and accepted. |
| 11 | SVG render mechanism — "isn't mermaid already supported in shll.ai's theme?" | Premise **refuted** by ground truth (no mermaid dep; `extract-readme.ts` strips fences; live slice has 0). SVGs required. Mechanism resolved → **hand-author the two SVGs once** (no CI/script). |
| 7 | Asset dir? | **Confirmed** — `docs/img/`, referenced by `raw.githubusercontent.com` absolute URLs. |
| 10 | Tail cut line? | **Confirmed** — cut at `## Development`. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | This is a producer-side, this-repo-only change; shll.ai's puller/renderer/pages are untouched. | Contract's out-of-scope boundary states README conformance is per-repo forward work; consumer is verified live (`content/fab-kit/` exists, slug mapped, pages present). | S:95 R:80 A:95 D:95 |
| 2 | Certain | Change type is `docs`. | Reorganizing README + docs; keyword "readme"/"docs" → docs per type heuristic. | S:90 R:70 A:95 D:90 |
| 3 | Confident | Diagrams: keep ` ```mermaid ` AND commit rendered SVGs referenced adjacently (§5 recommended pattern). | User chose "Render SVG + keep mermaid" over SVG-only and drop-diagrams; matches contract §5 verbatim. | S:90 R:60 A:80 D:85 |
| 4 | Confident | Scope = repo-relative link fixes + tail-boundary heading + diagram fix (full conformance). | User chose "Links + tail boundary + diagrams" over the two narrower options. | S:90 R:55 A:80 D:80 |
| 5 | Confident | Image refs and rewritten links MUST be absolute URLs, not repo-relative. | Read `ReadmeSlice.astro`: `createMarkdownProcessor({})` does no relative-path rewrite → relative href/src 404s on site. Observed, not assumed. | S:85 R:70 A:90 D:85 |
| 6 | Confident | In-page `#anchor` links stay as-is (except those pointing into the fenced-off tail). | Anchors target slice-surviving headings; Starlight re-slugs consistently. Low risk, easily reversed. | S:75 R:80 A:75 D:80 |
| 7 | Certain | Rendered SVGs live in `docs/img/`; referenced via `raw.githubusercontent.com/sahil87/fab-kit/main/docs/img/*.svg`. | Clarified — user confirmed. | S:95 R:60 A:55 D:55 |
| 8 | Confident | Repo-relative doc links rewrite to **GitHub-blob absolute** URLs (`github.com/sahil87/fab-kit/blob/main/...`). Fully-relative links were considered and rejected. | Clarified — user asked whether links could stay fully relative; verified NO: the README's relative base differs by location (`/` on GitHub vs `/tools/fab-kit/readme/` on shll.ai) and `createMarkdownProcessor({})` does no base rewrite, so one relative string can't resolve in both. Absolute required; GitHub-blob chosen (every target provably exists, no URL map to maintain). | S:80 R:55 A:75 D:60 |
| 9 | Certain | **Full docs/ reorg now**: create `docs/site/` AND migrate site-only prose into it, plus `docs/internal/` for maintainer notes and `docs/img/` for SVGs. | Clarified — user chose "Full reorg now" over scaffold-only/skip. NOTE: §9 (`docs/site/` pull path) is RESERVED/UNIMPLEMENTED on shll.ai — content placed there is not pulled until shll.ai ships §9. User accepted this forward investment knowingly. | S:95 R:35 A:55 D:45 |
| 10 | Certain | Tail cut at `## Development`: GitHub-only content (Stage Coverage table, Companion tools, Learn More) moves below a `## Development` denylisted heading; the slice ends there. | Clarified — user confirmed. | S:95 R:65 A:55 D:50 |
| 11 | Certain | SVGs are **hand-authored once** (export the 2 diagrams via mermaid.live / local `mmdc`), committed to `docs/img/`, no automation. Mermaid fences stay in README as the GitHub source of truth. | Clarified — user chose hand-author-once over CI/script. Only 2 low-churn diagrams; Constitution-I-pure (no CI dep, no build step). Accepted drift risk: mermaid↔SVG can diverge on future edits, mitigated by the fence remaining canonical. | S:95 R:55 A:60 D:50 |

11 assumptions (6 certain, 5 confident, 0 tentative, 0 unresolved).
