# Intake: shll.ai README-Extraction Conformance (§9-ACTIVE follow-up)

**Change**: 260608-yfg8-shll-readme-extraction-conformance
**Created**: 2026-06-08
**Status**: Draft

## Origin

> Task: conform this repo to shll.ai's README-extraction contract. shll.ai (the toolkit landing
> page) renders your tool's page by mechanically pulling a slice of your README.md and your
> docs/site/** tree on a daily schedule — nothing is hand-copied, and you push nothing. Read the
> contract and follow its §Producer conformance directive end-to-end:
> https://github.com/sahil87/shll.ai/blob/main/docs/specs/readme-extraction-contract.md
> (1) find this repo's row in the per-tool table; (2) Part 1 — restructure README.md; (3) Part 2 —
> add a docs/site/**/*.md tree (install.md / workflows.md); (4) run the Verify checklist. Ship as a
> single PR in this repo; do not touch shll.ai.

**Mode**: one-shot, agent-autonomous. The contract spec was read in full from the live shll.ai repo
(`gh api repos/sahil87/shll.ai/contents/docs/specs/readme-extraction-contract.md`, 785 lines). The
current `README.md`, `docs/site/`, `docs/internal/`, and `docs/img/` state were all read as ground
truth, so the gaps below are **observed against the current contract**, not hypothesized.

**Critical context — this is a follow-up, not a cold start.** A prior change
`260605-fcrp-readme-site-extraction-conformance` (merged as **PR #375**, commit `889f6e51`) already
conformed this README to an **earlier version** of the contract — the **§9-RESERVED era**. The
contract has since materially changed (changes `x0br` 2026-06-07 + the 2026-06-08 producer-directive
and reserved-slug edits). This change closes the *new* gaps that PR #375 could not have addressed
because the contract did not yet specify them. It does **not** redo PR #375's correct work.

## Why

shll.ai/fab-kit is fab-kit's public documentation site. Its per-tool pages render a **deduced,
curated slice** of *this* repo's `README.md` (rendered at `/tools/fab-kit/readme`) plus each file in
an optional `docs/site/**/*.md` tree (rendered at `/tools/fab-kit/<path>`), pulled daily by shll.ai's
scheduled refresh and rendered at build time. The tool repo is **canonical** — shll.ai never
hand-edits prose. Any structural defect in our README, and any stale or mis-shaped `docs/site/`
content, shows up verbatim on the public site, and the only place to fix it is **here**.

**What PR #375 already did right (do NOT touch):**

1. Rendered both inline ` ```mermaid ` diagrams to committed SVGs — `docs/img/pipeline-stages.svg`
   (6-stage pipeline) and `docs/img/stage-coverage.svg` (command×stage matrix) — referenced by
   absolute `raw.githubusercontent.com` URLs immediately after the mermaid fences. §5/§6 strip the
   fences on pull; the SVGs survive. ✓ verified present.
2. Head order is conformant: `# Fab Kit` (markdown H1) → canonical toolkit blockquote
   `> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.`
   (correct `https://shll.ai`, not `ai.shll.in`) → contiguous badge row → prose. No frontmatter,
   no HTML comment, no `<h1>` above the H1. ✓ verified.
3. All images are absolute; all links in the README **body above the `## Development` tail** are
   absolute `https://github.com/sahil87/fab-kit/blob/main/...` URLs. ✓ verified.
4. The §2 tail correctly drops everything at/below `## Development` (line 461) — the deep-dive link
   index, the stage-coverage matrix, the companions table, the `CONTRIBUTING.md` link. The 8
   remaining relative links (`docs/specs/*.md`, `CONTRIBUTING.md`) all live **below** that tail
   boundary, so they are cut on pull and never 404 on the site. ✓ verified.

**The gaps this change must close — created by the contract evolving since PR #375:**

1. **`docs/site/README.md` is stale and now actively wrong.** PR #375 created it as a placeholder
   declaring **"§9 pull path is RESERVED / NOT YET IMPLEMENTED … Nothing reads `docs/site/` yet."**
   That was true on 2026-06-05. Change `x0br` (2026-06-07) **flipped §9 RESERVED → ACTIVE**:
   `docs/site/**` is now pulled (tarball fetch) and rendered (dynamic route, one page per file).
   Two problems result:
   - The placeholder's central claim is now **false**.
   - Worse, because `docs/site/**` is now pulled, a file literally named `docs/site/README.md`
     renders as a **live public page** at `/tools/fab-kit/README` whose entire content tells visitors
     the feature doesn't exist. This is a self-contradicting public page that must be removed/replaced.
2. **`docs/internal/README.md` references a removed concept.** PR #375 created `docs/internal/` as
   the "maintainer-only, not-pulled" folder. The 2026-06-08 contract edit **removed the
   `docs/internal/` concept entirely** — the pull surface is now exactly `README.md` + `docs/site/**`,
   so *everything else* is un-pulled by default and maintainer notes need no blessed folder. The
   folder is harmless to the site (never pulled) but is now a vestigial artifact pointing at a dead
   contract concept; it should be removed so the repo matches the current model.
3. **No `docs/site/` depth pages exist (Part 2 — encouraged).** The directive explicitly invites
   `docs/site/install.md` and `docs/site/workflows.md`. The README's `## Development` tail — which is
   *dropped from the site* — currently holds exactly the kind of depth (deep-dive index, stage
   coverage, companions) that the site has no home for. `docs/site/` is that home: it renders as
   first-class pages under `/tools/fab-kit/<path>` and is the canonical, mechanically-synced way to
   give the site depth beyond the README slice.

**Consequence if we don't fix it:** the public fab-kit site keeps a live page asserting its own
docs-site feature is unimplemented, the repo carries a folder for a deleted contract concept, and we
forgo the now-live ability to publish install/workflow depth that the README tail can't surface.

## What Changes

### 1. Replace the stale `docs/site/README.md` placeholder

`docs/site/README.md` must stop being a "§9 is RESERVED" placeholder. Two viable shapes; this change
takes the second (see Assumptions #2):

- **Option A (rejected):** keep a `docs/site/README.md` but rewrite it to describe the now-ACTIVE
  model. Rejected because it would still render as a public `/tools/fab-kit/README` page — a
  meta-page about the docs tree is not user-facing content and clutters the tool's page set.
- **Option B (chosen):** **remove** `docs/site/README.md`. A `docs/site/`-explainer is a
  maintainer/design note, and per the current contract maintainer notes live *anywhere outside*
  `docs/site/` (they need no special folder). The directory becomes home to real, user-facing site
  pages only (install.md, workflows.md — below). If a maintainer-facing note about the tree is still
  wanted, it goes outside `docs/site/` (this intake + plan already record the rationale).

### 2. Remove the vestigial `docs/internal/` folder

`docs/internal/README.md` (and the `docs/internal/` directory) is removed. The `docs/internal/`
concept was deleted from the contract on 2026-06-08; the pull surface is exactly `README.md` +
`docs/site/**`, so a dedicated "not-pulled" folder is no longer part of the model. No README links
point into `docs/internal/` (verified — the README's only relative links are the `docs/specs/*`
deep-dive index below the tail), so removal is link-safe.

### 3. Add `docs/site/install.md` (Part 2)

A tool-specific install guide rendered at `/tools/fab-kit/install`. Content is drawn from the
README's existing `## Prerequisites` + `## Quick Start › 1. Install` material (Homebrew tap, the
`yq`/`jq`/`gh`/`direnv` companions, `gh auth login`, the direnv hook, shell completion, the
"developing fab-kit" Go/just tools, and the new-project / existing-repo / upgrade flows). This is
the deeper, tool-specific install detail the contract explicitly says *belongs on the site*
(§2: "Install is INCLUDED"; the directive: install depth belongs in `docs/site/install.md`).

Conformance rules applied to this page (closed-set, §9.1):
- **All images absolute** (none expected on this page, but any added must be `https://…`).
- **External links absolute-by-author** — every link leaving the rendered set (Homebrew, direnv,
  gh, yq, jq, Go, just, and any link back into the repo's `docs/specs/` or source) is written as a
  full `https://…` URL. No relative `docs/...` or `../` link may remain.
- **Intra-`docs/site/` links** (e.g. install → workflows) written relative *inside* `docs/site/`
  (`[workflows](./workflows.md)` → rewritten by the site to `/tools/fab-kit/workflows`), satisfying
  closure (no `..` escape).
- **Not named** `overview` / `readme` / `commands` (reserved). `install` is explicitly allowed.

### 4. Add `docs/site/workflows.md` (Part 2)

A workflows / usage guide rendered at `/tools/fab-kit/workflows`. Content is drawn from the README's
`## Quick Start › 2. Your first change` + `3. Going parallel` (the per-stage command walk, `/fab-ff`
vs `/fab-fff`, worktree parallelism) plus the conceptual `## Why Fab Kit` framing (the assembly line,
shared memory, the review loop, SRAD) — distilled into a task-oriented "how to drive the pipeline"
page. Same four closed-set rules as install.md (all images absolute; external links absolute;
intra-set links relative; not a reserved slug).

### 5. Link the README into the new `docs/site/` pages (§9.1 rule 4)

In the README's `## Development › Learn More` (or a suitable body location), add natural repo-relative
links to the new pages: `[Install guide](docs/site/install.md)` and
`[Workflows guide](docs/site/workflows.md)`. The site **auto-rewrites** `docs/site/<p>.md` →
`/tools/fab-kit/<p>` (§link-resolution rule 2), so these render correctly on the site **and** work as
plain repo links on GitHub. They MUST be written as **plain inline links** — never behind a
badge/thumbnail (`[![alt](img)](docs/site/x.md)`) and never as a reference-style definition
(`[id]: docs/site/x.md`), since those two shapes are known-unhandled by the consumer and would 404.

> **Resolved (user, 2026-06-08):** these links go in the README **body** (above the tail), so the
> site `readme` page cross-links to its sibling install/workflows pages. <!-- clarified: README→docs/site links in body, not tail-only — user chose body for site cross-nav -->
> They are written as plain inline links (never behind a badge or as reference-style definitions).

### 6. README body conformance re-verification (mostly a no-op)

Re-run the directive's Verify checklist against the README. Expected: already conformant from PR #375
(head order, canonical blockquote, absolute images, rendered mermaid SVGs, absolute body links, tail
boundary). Any residual defect found during verification is fixed; none is currently expected. No
`#gh-dark-mode-only` / `#gh-light-mode-only` fragments exist (verified) — nothing to strip.

### Out of scope

- **Any change to shll.ai.** Its pull + render wiring is live for all 7 slugs and pulls
  automatically; this change touches only this repo's content structure.
- **Re-rendering the mermaid SVGs** — PR #375's committed SVGs are correct and current.
- **The help-dump / `help/fab-kit.json` producer** (backlog `xob7`) — a separate, already-shipped
  contract (`help-dump-contract.md`), not this README-prose contract.
- **`docs/specs/` and `docs/memory/`** — these keep their fab meanings and are explicitly **not**
  pulled by shll.ai; they are untouched.

## Affected Memory

<!-- This is a producer-side docs-conformance change against an external consumer's contract.
     fab-kit's own memory tracks fab workflow/pipeline/distribution behavior, not the shape of the
     public docs site as dictated by shll.ai. The one durable, non-obvious fact worth recording is
     the shll.ai pull contract surface (README slice + docs/site/** tree, the reserved slugs, the
     §9-ACTIVE flip) so a future agent does not re-derive it or re-strand content. Candidate domain:
     distribution (how fab-kit content reaches users / the public site). Marked tentative — hydrate
     decides final placement. -->

- `distribution/shll-ai-readme-contract`: (new) The shll.ai README-extraction pull contract as it
  applies to fab-kit — pull surface is exactly `README.md` (deduced slice: head-skip + tail-denylist
  + mermaid/gh-theme strips) + `docs/site/**` (closed-set tree, one page per file); reserved slugs
  `overview`/`readme`/`commands`; `install`/`workflows` belong to the repo; images all-absolute;
  external links absolute-by-author. Records the §9-RESERVED→ACTIVE flip so the placeholder mistake
  is not reintroduced. *(Final domain/name decided at hydrate.)*

## Impact

- **Files added:** `docs/site/install.md`, `docs/site/workflows.md`.
- **Files removed:** `docs/site/README.md` (stale placeholder), `docs/internal/README.md` + the
  `docs/internal/` directory (dead concept).
- **Files modified:** `README.md` (add the two `docs/site/` links; re-verify conformance — otherwise
  expected near-no-op).
- **No code, no CI, no build, no push to any external repo.** Pure markdown/doc restructuring in this
  repo. Constitution IV (markdown-only artifacts) and the "no images generated by tooling" rule are
  respected — the only images are PR #375's pre-existing committed SVGs, referenced absolutely.
- **External dependency:** correctness is defined by shll.ai's `extract-readme.ts` (the contract's
  machine anchor). We conform to the prose directive; the site validates on its daily pull (no
  blocking gate — divergence is report-only `::warning::`).
- **Verification:** the directive's Verify checklist (head shape; no relative link/image targets that
  escape the rendered set; no gh-theme fragments; no reserved-slug `docs/site/` page names). Optional
  local self-check (`extract-readme-cli.mjs`) requires the shll.ai repo checkout — available at
  `~/code/sahil87/shll.ai` per the prior change's note; run if convenient, not required.

## Open Questions

- ~~Should the README → `docs/site/` links sit in the README **body** or only in the `## Development`
  tail?~~ **Resolved (user): body.**
- Should a maintainer-facing note explaining the `docs/site/` tree's purpose live somewhere outside
  `docs/site/` (e.g. a short `CONTRIBUTING.md` mention), or is the intake/plan record sufficient?
  (Default: intake/plan record is sufficient; no new maintainer file.)
- How much README prose should `install.md` / `workflows.md` *duplicate* vs. *deepen*? The site pulls
  both the README slice and the docs/site pages, so verbatim duplication is wasteful. (Default:
  docs/site pages **deepen** with tool-specific detail and a task-oriented framing, the README slice
  stays the concise overview — the coexistence model the contract describes for install sections.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Repo row = `fab-kit` slug, `content/fab-kit/` collector, `/tools/fab-kit/` URL space, reserved slugs `overview`/`readme`/`commands` | Read directly from the contract's §Producer conformance directive per-tool table | S:98 R:90 A:98 D:98 |
| 2 | Confident | Remove `docs/site/README.md` rather than rewrite it (Option B) | A `docs/site/`-explainer renders as a public `/tools/fab-kit/README` page, which is not user-facing content; the contract says maintainer notes live *outside* `docs/site/`. Reversible (re-add a file). One clear front-runner | S:80 R:85 A:80 D:75 |
| 3 | Confident | Remove the `docs/internal/` folder | The `docs/internal/` concept was explicitly deleted from the contract on 2026-06-08; pull surface is now exactly README + docs/site/**. No README links point into it (verified), so removal is link-safe | S:85 R:80 A:85 D:80 |
| 4 | Confident | Use `docs/site/install.md` + `docs/site/workflows.md` as the Part 2 pages | The task names these two files explicitly; both are allowed (non-reserved) slugs per the 2026-06-08 reserved-set shrink | S:95 R:75 A:90 D:85 |
| 5 | Certain | README → docs/site links go in the README **body** (above the tail), as plain inline links | Clarified — user chose body so the site `readme` page cross-links to its sibling install/workflows pages | S:95 R:85 A:90 D:90 |
| 6 | Confident | docs/site pages **deepen** (tool-specific detail) rather than duplicate the README slice verbatim | Clarified — user chose deepen-not-duplicate; the site pulls both surfaces so duplication is wasteful, matching the contract's install coexistence model. Some overlap acceptable | S:85 R:80 A:85 D:80 |
| 7 | Certain | Do NOT touch shll.ai, the committed mermaid SVGs, `help/fab-kit.json`, or `docs/specs/`/`docs/memory/` | Explicit task instruction (single PR in this repo) + contract out-of-scope boundary + PR #375 already rendered the SVGs correctly | S:98 R:85 A:95 D:95 |
| 8 | Confident | README body is already conformant from PR #375; step 6 is a re-verify, expected near-no-op | Verified head order, canonical blockquote, absolute images/body-links, rendered SVGs, tail boundary, no gh-theme fragments | S:85 R:75 A:90 D:80 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
