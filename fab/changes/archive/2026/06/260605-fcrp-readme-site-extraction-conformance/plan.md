# Plan: README Site-Extraction Conformance

**Change**: 260605-fcrp-readme-site-extraction-conformance
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md. This change makes fab-kit's README.md conformant to
     shll.ai's README-extraction contract (~/code/sahil87/shll.ai/docs/specs/readme-extraction-contract.md)
     so the public docs site shll.ai/fab-kit renders the pulled README slice correctly.
     Producer-side, THIS-REPO-ONLY. No shll.ai edits. -->

### Diagrams: dual-render mermaid + committed SVG

#### R1: Mermaid fences are preserved AND each is paired with an adjacent raw-URL SVG image
Both ` ```mermaid ` fences in `README.md` (the "6 Stages" flowchart and the "Stage Coverage by Command" `block-beta`) MUST remain in place (GitHub renders mermaid natively). Each fence MUST be immediately followed by a markdown image referencing a committed rendered SVG via an **absolute** `raw.githubusercontent.com` URL with mandatory descriptive alt text, because shll.ai's puller strips every ` ```mermaid ` fence and the renderer does not rewrite relative `src`.

- **GIVEN** the README's two mermaid fences
- **WHEN** the README is rendered on GitHub (mermaid native) and pulled+rendered on shll.ai (mermaid stripped)
- **THEN** GitHub shows the live mermaid and the SVG `<img>`; the site shows only the SVG `<img>` (which survives because its src is an absolute raw URL)
- **AND** each SVG image reference uses `https://raw.githubusercontent.com/sahil87/fab-kit/main/docs/img/<name>.svg` with alt text readable on both surfaces

#### R2: Two valid, hand-authored SVG assets are committed to `docs/img/`
`docs/img/pipeline-stages.svg` and `docs/img/stage-coverage.svg` MUST be real, openable SVG (valid XML, `<svg>` root), hand-authored to match their mermaid source labels and the existing fill colors (blue `#64b5f6` / orange `#ffb74d` / green `#81c784` / purple `#ce93d8` and the legend palette). No build step, no CI render workflow, no render script (Constitution I).

- **GIVEN** the two mermaid sources
- **WHEN** the SVGs are hand-authored once and committed
- **THEN** each file parses as valid XML with an `<svg>` root, is non-empty, and reproduces the node labels and color scheme of its mermaid source
- **AND** no build pipeline, CI workflow, or render script is introduced

### Links: repo-relative → absolute in the slice region

#### R3: Repo-relative doc links above the tail boundary become absolute GitHub-blob URLs
Every repo-relative doc link in the slice region (head boundary → `## Development` tail boundary) — the `docs/specs/*.md` links and `CONTRIBUTING.md` — MUST be rewritten to `https://github.com/sahil87/fab-kit/blob/main/<path>`. The README's relative base differs by surface (`/` on GitHub vs `/tools/fab-kit/readme/` on shll.ai) and the renderer does no base rewrite, so no relative string resolves in both.

- **GIVEN** a repo-relative link such as `[Glossary](docs/specs/glossary.md)` above the boundary
- **WHEN** the slice renders on shll.ai
- **THEN** the link is absolute (`https://github.com/sahil87/fab-kit/blob/main/docs/specs/glossary.md`) and resolves on both GitHub and the site
- **AND** no `](docs/`, `](CONTRIBUTING.md`, or `](fab/` repo-relative link remains above the boundary

#### R4: In-page anchors stay, except those pointing at moved/non-existent sections
In-page `#anchor` links above the boundary MUST remain as-is when their target heading survives in the slice. Anchors pointing at sections moved below `## Development` (`#stage-coverage-by-command`, `#learn-more`) or at a non-existent heading (`#standalone-cli-tools` — no such heading exists) MUST be removed or re-pointed.

- **GIVEN** the top "Contents:" nav, the "[Try it now]" line, and the H1 tagline
- **WHEN** the slice renders
- **THEN** no surviving anchor above the boundary points at a section that was moved below `## Development` or at a heading that does not exist
- **AND** anchors whose target survives in the slice are left unchanged

### Tail boundary: fence GitHub-only content

#### R5: A single top-level `## Development` heading marks the tail boundary
A single top-level `## Development` heading (a §2 denylisted heading) MUST be introduced near the end of the README. "Stage Coverage by Command", "Companion tools", and "Learn More" MUST move below it (as `###` subsections). Everything above the boundary is the site slice and MUST end on genuinely user-facing prose.

- **GIVEN** the contract's §2 denylist (`Contributing`/`Development`/`Building`/`License`/`Acknowledgements`)
- **WHEN** the puller computes the tail boundary
- **THEN** the slice ends at `## Development`; the Stage Coverage matrix, Companion tools, and Learn More ship only on GitHub
- **AND** exactly one top-level `## Development` heading exists (the existing `### Developing Fab Kit` subsection under Prerequisites is left intact and is NOT promoted)

### docs/ reorganization (audience axis, §9)

#### R6: `docs/img/`, `docs/internal/`, and `docs/site/` exist with explainer READMEs
`docs/img/` (SVG assets), `docs/internal/` (maintainer/design notes that MUST never reach the site), and `docs/site/` (site-only prose along the audience axis) MUST exist. `docs/internal/` and `docs/site/` MUST each carry a short `README.md` explaining their purpose. Because §9 (`docs/site/` pull path) is RESERVED/UNIMPLEMENTED on shll.ai, no README content that users currently rely on may be migrated into `docs/site/` such that it disappears from both the GitHub README and the (non-pulling) site — the safe floor is structure + explainer READMEs only.

- **GIVEN** the §9 audience axis (user-facing / GitHub-native / maintainer-facing) and its RESERVED status
- **WHEN** the docs/ reorg lands
- **THEN** the three directories exist; `docs/internal/README.md` and `docs/site/README.md` explain their audience-axis role and `docs/internal/`'s distinction from `docs/specs/`
- **AND** no load-bearing README content is stranded in the non-pulling `docs/site/`

### Non-Goals

- No changes under `~/code/sahil87/shll.ai` — the consumer (puller, renderer, pages) is complete and live.
- No `extract-readme.ts` changes — extraction mechanics are canonical on the shll.ai side.
- No CI workflow / render script for the SVGs (Constitution I) — hand-authored once.
- Not gating on the §7 divergence reporter (report-only).
- No content migration into `docs/site/` in this change (§9 unimplemented; would strand content) — structure + explainer only.

### Design Decisions

1. **Keep mermaid + commit SVG, reference SVG by absolute raw URL**: dual-render — *Why*: contract §5 mandates a rendered image for the site while GitHub renders mermaid natively; the renderer does not rewrite `src`, so the image URL must be absolute. — *Rejected*: SVG-only (loses GitHub's zoomable native mermaid); drop diagrams (site shows empty gaps, the current defect).
2. **GitHub-blob absolute URLs for doc links** — *Why*: every target provably exists in this repo, no anchor→page URL map to maintain; matches §3 single-source. — *Rejected*: fully-relative (no string resolves in both bases); shll.ai-page URLs (requires a per-target page map that can rot).
3. **Tail cut at `## Development`** — *Why*: §2 denylisted heading; cleanly fences the GitHub-native chrome (130-row matrix, companion-tool table, link farm) below the slice. — *Rejected*: `## Contributing` (content is dev-setup oriented but "Development" reads better for this tail); no boundary (entire README floods the site — the current defect).
4. **`docs/site/` is structure + explainer only (no content migration)** — *Why*: §9 is RESERVED on shll.ai; content placed there is pulled by nothing today, so migrating load-bearing README prose into it would strand that prose on both surfaces. — *Rejected*: full content migration now (intake flagged this exact risk; the safe floor is forward structure without stranding content).
5. **Drop the broken `#standalone-cli-tools` anchors** — *Why*: no "Standalone CLI Tools" heading exists in the README (already broken on GitHub and site today); the morally-related "Companion tools" section moves below the boundary, so a slice-internal anchor cannot point at it. — *Rejected*: re-point to `#companion-tools` (that section leaves the slice); leave broken (fails R4).

## Tasks

### Phase 1: Setup

- [x] T001 [P] Create `docs/img/`, `docs/internal/`, `docs/site/` directories <!-- R6 -->
- [x] T002 [P] Hand-author `docs/img/pipeline-stages.svg` (6-stage flowchart: 1 Intake → Execution[2 Apply→3 Review] → Completion[4 Hydrate] → Shipping[5 Ship→6 Review-PR], matching mermaid labels + blue/orange/green/purple fills) <!-- R2 -->
- [x] T003 [P] Hand-author `docs/img/stage-coverage.svg` (stage×command matrix with the 5-color legend matching the block-beta source) <!-- R2 -->
- [x] T004 [P] Create `docs/internal/README.md` explaining it holds maintainer/design notes that must never reach the site (distinct from `docs/specs/`) <!-- R6 -->
- [x] T005 [P] Create `docs/site/README.md` explaining it holds site-only prose along the audience axis; note §9 RESERVED status <!-- R6 -->

### Phase 2: README edits

- [x] T006 Add the raw-URL SVG image immediately after the "6 Stages" mermaid fence in `README.md` <!-- R1 -->
- [x] T007 Rewrite all repo-relative `docs/specs/*.md` and `CONTRIBUTING.md` links **above** the tail boundary to `https://github.com/sahil87/fab-kit/blob/main/<path>` absolute URLs in `README.md` <!-- R3 -->
- [x] T008 Introduce a single top-level `## Development` heading after `## Command Quick Reference`; move "Stage Coverage by Command", "Companion tools", and "Learn More" below it as `###` subsections in `README.md` <!-- R5 -->
- [x] T009 Add the raw-URL SVG image immediately after the "Stage Coverage by Command" mermaid fence (now under `## Development`) in `README.md` <!-- R1 -->
- [x] T010 Fix anchors above the boundary: drop/re-point `#stage-coverage-by-command`, `#learn-more` (moved sections) and the broken `#standalone-cli-tools` (no such heading) in the H1 tagline and the "Contents:" nav line in `README.md` <!-- R4 -->

### Phase 3: Verification

- [x] T011 Re-grep `README.md` to confirm checks (a)–(e): both mermaid fences present; each has an adjacent raw-URL SVG; no `](docs/`/`](CONTRIBUTING.md` above the boundary; single `## Development` tail heading with moved sections below; no above-boundary anchor points at a moved/missing section <!-- R1 R3 R4 R5 -->
- [x] T012 Validate both SVGs parse as well-formed XML with an `<svg>` root and are non-empty <!-- R2 -->

## Execution Order

- Phase 1 tasks are all `[P]` (independent files/dirs).
- T006–T010 are sequential edits to the same file (`README.md`) — must run in order to keep line context coherent.
- Phase 3 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: Both ` ```mermaid ` fences remain in `README.md`, each immediately followed by a markdown image whose src is an absolute `raw.githubusercontent.com/sahil87/fab-kit/main/docs/img/*.svg` URL with descriptive alt text
- [x] A-002 R2: `docs/img/pipeline-stages.svg` and `docs/img/stage-coverage.svg` exist, parse as valid XML with `<svg>` root, are non-empty, and reproduce their mermaid source labels + color scheme
- [x] A-003 R3: No `](docs/`, `](CONTRIBUTING.md`, or `](fab/` repo-relative link remains above the `## Development` boundary; all such links are absolute GitHub-blob URLs
- [x] A-004 R4: No surviving anchor above the boundary points at a section moved below `## Development` or at a non-existent heading; surviving-target anchors are unchanged
- [x] A-005 R5: Exactly one top-level `## Development` heading exists; "Stage Coverage by Command", "Companion tools", and "Learn More" are below it; the slice above ends on user-facing prose
- [x] A-006 R6: `docs/img/`, `docs/internal/`, `docs/site/` exist; `docs/internal/README.md` and `docs/site/README.md` explain their audience-axis role; no load-bearing README content is stranded in `docs/site/`

### Behavioral Correctness

- [x] A-007 R1: On a simulated pull (mermaid fences stripped), each diagram location still has a surviving SVG `<img>` with an absolute src — diagrams do not vanish on the site
- [x] A-008 R5: A tail-boundary scan ending at the first denylisted heading yields a slice that excludes the Stage Coverage matrix, Companion tools, and Learn More

### Scenario Coverage

- [x] A-009 R3: Following a rewritten doc link (e.g. Glossary) resolves to a real file under `https://github.com/sahil87/fab-kit/blob/main/docs/specs/glossary.md`

### Code Quality

- [x] A-010 Pattern consistency: README edits follow existing tone/structure; SVG color/label scheme matches the mermaid sources
- [x] A-011 No unnecessary duplication: the mermaid fence remains the single canonical diagram source; the SVG is the one-time render (no third copy)

### Documentation Accuracy

- [x] A-012 R3 R4: Rewritten link targets all exist in the repo; no dead links introduced above the boundary

### Cross-References

- [x] A-013 R5: Cross-references from moved sections (e.g. Companion tools → companions.md, Learn More entries) still point at valid GitHub-blob targets

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Producer-side, this-repo-only; no shll.ai edits. | Intake assumption 1 + ground-truth directive; consumer verified live. | S:95 R:80 A:95 D:95 |
| 2 | Confident | Keep mermaid + commit adjacent raw-URL SVG per diagram (§5). | Intake assumption 3; contract §5 verbatim. | S:90 R:60 A:80 D:85 |
| 3 | Confident | Repo-relative doc links above boundary → GitHub-blob absolute URLs. | Intake assumption 8; fully-relative verified non-viable. | S:85 R:55 A:80 D:75 |
| 4 | Certain | Tail cut at `## Development`; move Stage Coverage + Companion tools + Learn More below. | Intake assumption 10 (user-confirmed). | S:95 R:65 A:60 D:55 |
| 5 | Certain | SVGs hand-authored once into `docs/img/`, no automation. | Intake assumption 11; Constitution I. | S:95 R:55 A:60 D:55 |
| 6 | Confident | `docs/site/` gets structure + explainer README only — NO content migration this change. | Intake §4 explicitly flags §9-unimplemented risk: migrating load-bearing prose into the non-pulling `docs/site/` would strand it on both surfaces. Intake says "exercise judgment: structure + explainer is the safe floor." Chose the safe floor. | S:80 R:55 A:75 D:65 |
| 7 | Confident | Drop the broken `#standalone-cli-tools` anchors (H1 tagline ×2 + Contents nav) rather than re-point. | No "Standalone CLI Tools" heading exists (already broken today); the related "Companion tools" section moves below the boundary so no slice-internal anchor can target it. Plain text reads fine. | S:80 R:75 A:80 D:70 |
| 8 | Confident | Second SVG (stage-coverage) is committed and referenced adjacent to its fence even though that fence ends up below the `## Development` boundary (GitHub-only). | Ground-truth directive mandates an adjacent SVG for BOTH fences; consistency + future-proofing if the boundary moves. Low cost. | S:85 R:80 A:75 D:70 |
| 9 | Confident | Existing `### Developing Fab Kit` (under Prerequisites) is left as a `###` and NOT promoted; the new `## Development` is a distinct, single top-level tail heading. | Ground-truth caution: introduce a single top-level `## Development` near the END; the existing subsection is dev-prerequisites prose that belongs in the slice. A `###` does not trigger the §2 denylist conflict (matching is on heading text but the first denylisted heading after head terminates — a `### Developing Fab Kit` heading text is "Developing Fab Kit", not "Development", so it does not match the denylist). | S:75 R:55 A:70 D:60 |

9 assumptions (3 certain, 6 confident, 0 tentative, 0 unresolved).
