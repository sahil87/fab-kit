# Plan: shll.ai README-Extraction Conformance (§9-ACTIVE follow-up)

**Change**: 260608-yfg8-shll-readme-extraction-conformance
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- change_type: docs. Requirements derive from shll.ai's README-extraction
     contract as it applies to fab-kit (the producer). The contract changed
     since PR #375 (§9 RESERVED→ACTIVE; docs/internal/ concept removed); these
     requirements close the new gaps only. -->

### Docs Site: Stale Placeholder Removal

#### R1: The stale `docs/site/README.md` placeholder MUST be removed
The repo SHALL NOT carry a `docs/site/README.md` file. With §9 now ACTIVE, `docs/site/**`
is pulled and rendered one-page-per-file, so a file named `README.md` would render as a live
public page at `/tools/fab-kit/README` whose content falsely claims the docs-site feature is
unimplemented. The file MUST be deleted and MUST NOT be replaced by any other README-named file
inside `docs/site/`. `README` is also a reserved site-owned slug.

- **GIVEN** §9 of the contract is now ACTIVE (`docs/site/**` is pulled and rendered)
- **WHEN** the repo is conformed
- **THEN** `docs/site/README.md` does not exist
- **AND** no replacement README-named page is created inside `docs/site/`

### Docs Layout: Vestigial Folder Removal

#### R2: The vestigial `docs/internal/` folder MUST be removed
The `docs/internal/` concept was deleted from the contract on 2026-06-08; the pull surface is
now exactly `README.md` + `docs/site/**`, so everything else is un-pulled by default and a
dedicated "not-pulled" folder is no longer part of the model. `docs/internal/README.md` and the
`docs/internal/` directory SHALL be removed. Removal MUST be link-safe: no link in `README.md`
or anywhere in `docs/site/` may point into `docs/internal/`.

- **GIVEN** the `docs/internal/` concept was removed from the contract
- **WHEN** no link in `README.md` or `docs/site/**` points into `docs/internal/` (verified by grep)
- **THEN** `docs/internal/README.md` and the `docs/internal/` directory are removed

### Docs Site: Install Depth Page

#### R3: A `docs/site/install.md` install guide MUST be created
A tool-specific install guide SHALL exist at `docs/site/install.md`, rendered at
`/tools/fab-kit/install`. It MUST begin with a single `# Install` H1 on line 1 (no frontmatter,
no HTML comment above it). It SHALL DEEPEN rather than duplicate the README — carrying the
tool-specific install depth: the Homebrew tap (`brew tap sahil87/tap` + `brew install fab-kit`),
the companion utilities (`yq jq gh direnv`), `gh auth login`, the direnv shell hook, optional
shell completion (`eval "$(fab shell-init zsh)"` / `fab completion <shell>`), the "Developing
Fab Kit" extras (`brew install go just`), the three install flows (new project, onboarding an
existing repo, upgrading), and `fab doctor` for troubleshooting. It MUST obey the closed-set
conformance rules (R6).

- **GIVEN** §9 is ACTIVE and `install` is an allowed (non-reserved) slug
- **WHEN** the page is authored from the README `## Prerequisites` + `## Quick Start › 1. Install`
- **THEN** `docs/site/install.md` exists, starts with a single `# Install` H1, and carries the install depth
- **AND** it cross-links to the workflows page via an intra-set relative link

### Docs Site: Workflows Depth Page

#### R4: A `docs/site/workflows.md` workflows guide MUST be created
A task-oriented workflows/usage guide SHALL exist at `docs/site/workflows.md`, rendered at
`/tools/fab-kit/workflows`. It MUST begin with a single `# Workflows` H1 on line 1. It SHALL
DEEPEN rather than duplicate the README — a "how to drive the pipeline" walkthrough covering the
per-stage command sequence (`/fab-new` → `/fab-continue` ×3 → `/git-pr` → `/git-pr-review` →
`/fab-archive`), `/fab-ff` vs `/fab-fff` vs `/fab-proceed`, the apply⇄review auto-rework loop,
`/fab-status` checkpointing, and going parallel with `wt create` + worktrees — framed with the
conceptual "why" (assembly line, shared memory, SRAD confidence gate). It MUST obey R6.

- **GIVEN** §9 is ACTIVE and `workflows` is an allowed (non-reserved) slug
- **WHEN** the page is authored from README `## Quick Start › 2/3` + `## Why Fab Kit`
- **THEN** `docs/site/workflows.md` exists, starts with a single `# Workflows` H1, and carries the usage depth
- **AND** it cross-links to the install page via an intra-set relative link

### README Body: Cross-Links Into New Pages

#### R5: The README body MUST link to both new pages as plain inline links
The README **body** (ABOVE the `## Development` tail heading) SHALL contain plain inline links to
the two new pages, written as the natural repo-relative path: `[Install guide](docs/site/install.md)`
and `[Workflows guide](docs/site/workflows.md)`. The site auto-rewrites `docs/site/<p>.md` →
`/tools/fab-kit/<p>`. These links MUST be plain inline `[text](docs/site/x.md)` — NEVER behind a
badge/image (`[![alt](img)](docs/site/x.md)`) and NEVER reference-style (`[id]: docs/site/x.md`),
because those two shapes are known-unhandled by the consumer and would 404 on the site.

- **GIVEN** the site readme page should cross-link to its sibling install/workflows pages
- **WHEN** the links are placed in the README body above the `## Development` heading
- **THEN** the README body contains `docs/site/install.md` and `docs/site/workflows.md` plain inline links
- **AND** neither link is behind an image nor written reference-style

### Closed-Set Conformance For docs/site Pages

#### R6: Both new docs/site pages MUST satisfy the closed-set conformance rules
Every page under `docs/site/**` SHALL conform to the consumer's closed-set rules: (a) all images
absolute `https://…`; (b) every link leaving the rendered docs/site set (external sites AND any
link back into the repo's own `docs/specs/*`, source, or `CONTRIBUTING.md`) MUST be a full
`https://…` URL — repo-internal absolute links use `https://github.com/sahil87/fab-kit/blob/main/<path>`;
(c) intra-docs/site links (install ↔ workflows) written relative INSIDE docs/site (`./workflows.md`,
`./install.md`); (d) no reserved-slug filenames (`overview`/`readme`/`commands`); (e) each page starts
with a single `#` H1, no frontmatter, no HTML comment above it. No relative `../`, `./`, or `docs/...`
link may escape the docs/site set; no relative image anywhere.

- **GIVEN** the consumer renders docs/site/** as a closed set with strict link/image rules
- **WHEN** the verification greps run against the new pages
- **THEN** no escaping relative link and no relative image is present in either page
- **AND** intra-set cross-links resolve within docs/site, external/repo links are absolute https

### README Body: Conformance Re-Verification

#### R7: The README body MUST remain conformant (no-op re-verify)
The README body conformance established by PR #375 SHALL be preserved: no
`#gh-dark-mode-only`/`#gh-light-mode-only` fragments; head order, absolute images, rendered mermaid
SVGs, absolute body links, and the `## Development` tail boundary are unchanged except for the R5
link insertion. Any residual defect found during verification is fixed; none is expected.

- **GIVEN** PR #375 already conformed the README body
- **WHEN** the verification greps run against `README.md`
- **THEN** no gh-theme fragments exist and the only body change is the R5 inline links above the tail

### Non-Goals

- Any change to shll.ai — its pull/render wiring is live; this change touches only this repo.
- Re-rendering the mermaid SVGs — PR #375's committed SVGs are correct and reused by absolute URL only.
- `help/fab-kit.json` producer — a separate, already-shipped contract.
- `docs/specs/` and `docs/memory/` — keep their fab meanings, explicitly not pulled, untouched (memory updates belong to hydrate).
- Adding a maintainer explainer about the docs/site tree — the intake/plan record is sufficient; maintainer notes live outside docs/site/.

### Design Decisions

1. **Remove `docs/site/README.md` rather than rewrite it (Option B)**: a docs/site explainer would
   render as a public `/tools/fab-kit/README` page (and `README` is a reserved slug) — *Why*: not
   user-facing content; the contract says maintainer notes live outside docs/site/ — *Rejected*:
   rewriting it to describe the ACTIVE model (still renders as a public meta-page).
2. **Place README→docs/site links in the body, not the tail**: *Why*: the user chose body so the
   site `readme` page cross-links to its sibling install/workflows pages — *Rejected*: tail-only
   placement (the tail is dropped on pull, so the site readme page would have no cross-nav).
3. **docs/site pages deepen, not duplicate**: *Why*: the site pulls both the README slice and the
   docs/site pages, so verbatim duplication is wasteful — *Rejected*: copy README prose verbatim.

## Tasks

### Phase 1: Removals + Verification

- [x] T001 Verify no link in `README.md` or `docs/site/**` points into `docs/internal/` (grep); confirm none before removal <!-- R2 -->
- [x] T002 Remove the stale placeholder `docs/site/README.md` via `git rm` (no README-named replacement inside docs/site/) <!-- R1 -->
- [x] T003 Remove `docs/internal/README.md` and the `docs/internal/` directory via `git rm` <!-- R2 -->

### Phase 2: New docs/site Pages

- [x] T004 [P] Create `docs/site/install.md` — single `# Install` H1; install depth from README `## Prerequisites` + `## Quick Start › 1. Install`; closed-set links (external/repo absolute https, intra-set `./workflows.md` relative) <!-- R3 -->
- [x] T005 [P] Create `docs/site/workflows.md` — single `# Workflows` H1; task-oriented pipeline walkthrough + conceptual framing from README `## Quick Start › 2/3` + `## Why Fab Kit`; closed-set links (external/repo absolute https, intra-set `./install.md` relative) <!-- R4 -->

### Phase 3: README Linking

- [x] T006 Add plain inline `[Install guide](docs/site/install.md)` and `[Workflows guide](docs/site/workflows.md)` links into the README body ABOVE the `## Development` tail (intro nav line / Quick Start) — not behind images, not reference-style <!-- R5 -->

### Phase 4: Verification

- [x] T007 Run the 6-check verification suite: `ls docs/site/`; `ls docs/internal/`; escaping-relative grep over docs/site + README; gh-theme grep; README body link shape check; H1/no-docs-site-README check on both pages. Fix any closure violation to absolute https <!-- R6 R7 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `docs/site/README.md` no longer exists and no README-named page was created inside `docs/site/`
- [x] A-002 R2: `docs/internal/README.md` and the `docs/internal/` directory are removed
- [x] A-003 R3: `docs/site/install.md` exists, starts with a single `# Install` H1, and carries the tool-specific install depth (Homebrew tap, companions, gh auth, direnv hook, completion, dev extras, three flows, fab doctor)
- [x] A-004 R4: `docs/site/workflows.md` exists, starts with a single `# Workflows` H1, and carries the pipeline walkthrough + conceptual framing
- [x] A-005 R5: README body contains both `docs/site/install.md` and `docs/site/workflows.md` links above the `## Development` heading

### Behavioral Correctness

- [x] A-006 R5: Both README→docs/site links are plain inline `[text](docs/site/x.md)`, NOT behind an image and NOT reference-style
- [x] A-007 R2: Removal verified link-safe — no link in README or docs/site pointed into `docs/internal/`

### Scenario Coverage

- [x] A-008 R6: Each docs/site page cross-links to the other via an intra-set relative link (`./workflows.md` / `./install.md`)
- [x] A-009 R6: The escaping-relative grep over `docs/site/ README.md` yields only intra-set `./x.md` links or README→`docs/site/<p>.md` links; no `docs/specs/`, `docs/internal/`, `CONTRIBUTING.md`, or other escaping relative target inside docs/site pages

### Edge Cases & Error Handling

- [x] A-010 R6: No relative image anywhere in docs/site/**; any referenced image is absolute `https://…`
- [x] A-011 R3 R4: Neither page is named a reserved slug (`overview`/`readme`/`commands`) and neither references `docs/site/README`

### Code Quality

- [x] A-012 Pattern consistency: New docs/site pages follow the README/docs voice and standard CommonMark (Constitution: markdown-only)
- [x] A-013 No unnecessary duplication: Pages deepen rather than verbatim-duplicate the README slice
- [x] A-014 R7: No `#gh-dark-mode-only`/`#gh-light-mode-only` fragments in `README.md` or `docs/site/`; README body otherwise unchanged except the R5 links

### Documentation Accuracy

<!-- config.yaml checklist.extra_categories: documentation_accuracy -->

- [x] A-015 R3 R4: Install/workflows page content accurately reflects the current README commands and flows (no stale or invented commands)

### Cross-References

<!-- config.yaml checklist.extra_categories: cross_references -->

- [x] A-016 R5 R6: Cross-references resolve correctly — README→docs/site links use the rewrite-compatible `docs/site/<p>.md` form; intra-set links use `./<p>.md`; repo-back links use absolute `github.com/.../blob/main/...`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- This is `change_type: docs` — no `## Deletion Candidates` section (review parsimony/deletion passes skipped)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove (not rewrite) `docs/site/README.md`; no README-named replacement inside docs/site/ | Contract + intake Option B: a docs/site explainer renders as a public reserved-slug page; maintainer notes live outside docs/site/ | S:95 R:85 A:95 D:90 |
| 2 | Certain | Remove `docs/internal/` folder; link-safety verified by grep (none point in) | Contract deleted the concept 2026-06-08; pull surface is exactly README + docs/site/**; grep confirms no inbound links | S:95 R:80 A:95 D:90 |
| 3 | Confident | `docs/memory/distribution/*` references to `docs/internal/`/RESERVED docs/site are left untouched | Intake Assumption #7 + Constitution II: docs/memory is not pulled and is owned by the hydrate stage, not apply; updating it here is out of scope | S:88 R:80 A:85 D:85 |
| 4 | Certain | README→docs/site links go in the body (above tail) as plain inline links | User-resolved in intake; plain inline shape required by consumer (badge/reference-style are unhandled) | S:98 R:85 A:95 D:95 |
| 5 | Confident | docs/site pages deepen (tool-specific detail) rather than duplicate README verbatim | Intake Assumption #6 + contract install-coexistence model; some overlap acceptable | S:85 R:80 A:85 D:80 |
| 6 | Confident | Place the two inline links in the existing intro nav line (line ~11) — least-disruptive natural spot | Intake suggests intro nav line or Quick Start as natural spots; intro nav already lists Try it now / concepts / glossary, so install/workflows guides fit there | S:85 R:90 A:85 D:80 |

6 assumptions (3 certain, 3 confident, 0 tentative).
