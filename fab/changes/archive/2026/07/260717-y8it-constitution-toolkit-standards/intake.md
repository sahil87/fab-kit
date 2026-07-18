# Intake: Bind Constitution to sahil87 Toolkit Standards

**Change**: 260717-y8it-constitution-toolkit-standards
**Created**: 2026-07-18

## Origin

One-shot `/fab-new` invocation with a fully-specified task description (no prior conversation; all decisions below were prescribed in the input or derived from the constitution file's own structure):

> Task: Amend this repo's fab constitution to bind it to the sahil87 toolkit standards. This repo is part of the sahil87 toolkit. The toolkit publishes binding, producer-facing standards — CLI design principles plus mechanical contracts (machine-readable help output, README/docs-site structure, and others over time). They are canonically authored in the sahil87/shll repository's docs/site/standards/ tree, rendered on https://shll.ai, and readable offline via the `shll standards` command. This change adds a constitution article so every future pipeline run in this repo loads and enforces the obligation.
>
> Make this change:
> 1. In fab/project/constitution.md, add a new article under Additional Constraints (create the section if this constitution lacks it, matching the file's existing structure): [### Toolkit Standards — full article text reproduced verbatim in What Changes below]
> 2. Bump the constitution's Last Amended date (and version, per this file's own governance line).
> 3. Deliberate constraint: do NOT copy standard names, counts, or per-standard URLs into the constitution — `shll standards` is the enumeration, and the article must stay correct as standards evolve.
>
> Ship per this repo's normal flow (docs-type fab change → PR). Nothing else is in scope — no conformance fixes in this change.

## Why

1. **Problem**: fab-kit is part of the sahil87 toolkit, whose binding producer-facing standards (CLI design principles plus mechanical contracts — machine-readable help output, README/docs-site structure, and others over time) are canonically authored in the sahil87/shll repository's `docs/site/standards/` tree, rendered on https://shll.ai, and enumerable offline via `shll standards`. Nothing in this repo's governance currently references those standards, so no pipeline run has any loaded obligation to check surface changes (CLI, help output, README.md, docs/site/) against them — conformance depends entirely on a human remembering out-of-band.

2. **Consequence if unfixed**: every future change to a governed surface can silently drift from the toolkit standards, and drift is only caught (if ever) by manual audits. Since `fab/project/constitution.md` is in the always-load context layer that every pipeline stage and dispatched subagent reads, the constitution is precisely the seam where a standing obligation becomes self-enforcing.

3. **Why this approach**: reference-by-enumeration rather than enumeration-by-copy. The article names `shll standards` as the live enumeration and the sahil87/shll `docs/site/standards/` tree as the canonical source, deliberately copying **no** standard names, counts, or per-standard URLs. Standards added or revised upstream then bind this repo immediately, with no re-amendment needed — the article can never go stale against the evolving standard set. (The rejected alternative — listing the current standards in the article — would be more immediately scannable but rots the moment shll adds or renames a standard.)

## What Changes

### New `### Toolkit Standards` article in `fab/project/constitution.md`

The constitution already has an `## Additional Constraints` section (currently a flat bullet list, no `###` subsections). Add the following article under it — after the existing bullets, before `## Governance` — exactly as prescribed (em-dashes normalized to the file's `—` convention; content otherwise verbatim from the task input):

```markdown
### Toolkit Standards

This tool is part of the sahil87 toolkit and MUST conform to the toolkit's published standards. The standards are enumerated by running `shll standards` — each entry names what it governs; read one with `shll standards <name>`. Before changing the CLI surface, help output, README.md, or docs/site/, the change MUST be checked against the standards governing that surface. If shll is unavailable, the canonical sources are the sahil87/shll repository's docs/site/standards/ tree (rendered on https://shll.ai). Standards added or revised there bind this repo without further amendment to this constitution.
```

Deliberate constraint (from the task, binding on apply and review): the article MUST NOT name individual standards, state how many exist, or link per-standard URLs — `shll standards` is the enumeration. The only permitted references are the command (`shll standards` / `shll standards <name>`), the repository tree (sahil87/shll `docs/site/standards/`), and the rendered site (https://shll.ai).

### Governance line bump

Current governance line (constitution.md line 38):

```markdown
**Version**: 1.3.0 | **Ratified**: 2026-02-06 | **Last Amended**: 2026-06-01
```

Update to:

```markdown
**Version**: 1.4.0 | **Ratified**: 2026-02-06 | **Last Amended**: 2026-07-18
```

Minor version bump (1.3.0 → 1.4.0) because this amendment **adds a new normative MUST-rule** — unlike the two prior recorded amendments, whose comments each note "no new normative MUST-rule was added". Ratified date unchanged.

### Dated amendment comment

Following the file's established amendment-record pattern (two existing dated HTML comments below the governance line), append a third:

```markdown
<!-- 2026-07-18 (260717-y8it): Added the `### Toolkit Standards` article under Additional
     Constraints — binds this repo to the sahil87 toolkit's published standards via the
     `shll standards` enumeration (canonical source: sahil87/shll docs/site/standards/,
     rendered on https://shll.ai). Deliberately enumerates nothing (no standard names,
     counts, or per-standard URLs) so the article stays correct as standards evolve.
     New normative MUST-rule added → minor version bump 1.3.0 → 1.4.0. -->
```

### Explicitly out of scope

- No conformance fixes to any governed surface (CLI, help output, README.md, docs/site/) — this change only installs the obligation.
- No changes to kit sources (`src/kit/`), the Go binary, tests, or specs — `fab/project/constitution.md` is project config, not kit content, so the constitution's own skill-file/CLI mirror constraints do not apply.
- No copying of the standards' content or enumeration into this repo.

## Affected Memory

None — this amendment is project governance content, not kit behavior. `_shared/configuration.md` documents the constitution *mechanism* (governance line, lifecycle), which is unchanged; no `docs/memory/` file records this repo's individual constitution articles.

## Impact

- **Files**: exactly one content file — `fab/project/constitution.md` (one new article, governance line bump, one amendment comment). Plus this change's own pipeline artifacts.
- **Behavioral**: every future pipeline run (all stages and dispatched subagents load the constitution in the always-load layer) picks up a standing MUST to check CLI-surface / help-output / README.md / docs/site/ changes against the toolkit standards via `shll standards`.
- **Risk**: minimal — pure markdown docs change, trivially revertible, no code paths touched.
- **Change type**: docs (explicit in the task; overrides the keyword-inferred `feat`).

## Open Questions

None — the task prescribes the article text, placement, versioning instruction, and scope boundary.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Article text, heading (`### Toolkit Standards`), and placement under the existing `## Additional Constraints` section are taken verbatim from the task | Fully prescribed in the input; the section already exists, so the create-if-lacking branch is moot | S:95 R:90 A:100 D:100 |
| 2 | Certain | No standard names, counts, or per-standard URLs appear in the article — only the `shll standards` command, the sahil87/shll `docs/site/standards/` tree, and https://shll.ai | Explicit deliberate constraint in the task | S:95 R:85 A:100 D:100 |
| 3 | Certain | A dated amendment HTML comment is appended below the governance line | The file's own structure answers this — two prior amendments each carry a dated `<!-- YYYY-MM-DD (change-id): ... -->` comment | S:60 R:95 A:90 D:85 |
| 4 | Certain | change_type is `docs` (explicit `fab status set-change-type` override of the keyword-inferred `feat`); scope excludes all conformance fixes | Task states "docs-type fab change" and "no conformance fixes in this change" verbatim | S:90 R:90 A:100 D:95 |
| 5 | Confident | Version bumps 1.3.0 → 1.4.0 (minor) | Task says bump "per this file's own governance line" without naming the increment; adding a new normative MUST article is a minor bump under semver-style constitution versioning, and prior no-new-rule amendments explicitly flagged themselves as such | S:70 R:90 A:75 D:70 |
| 6 | Confident | The task input's ASCII `--` is rendered as em-dash `—` in the article; content otherwise verbatim | The `--` is a plain-text artifact of the prompt; the constitution uses `—` throughout, and matching house typography changes no meaning | S:50 R:95 A:85 D:70 |
| 7 | Confident | Affected Memory is empty — hydrate records nothing in `docs/memory/` | Project-governance content, not kit behavior; `_shared/configuration.md` covers the constitution mechanism only, which is untouched | S:65 R:85 A:80 D:75 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
