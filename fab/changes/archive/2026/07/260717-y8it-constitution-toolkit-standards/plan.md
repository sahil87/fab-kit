# Plan: Bind Constitution to sahil87 Toolkit Standards

**Change**: 260717-y8it-constitution-toolkit-standards
**Intake**: `intake.md`

## Requirements

<!-- Requirements derived from intake.md. This is a docs-type change touching exactly
     one project-config file (fab/project/constitution.md). No code, no specs, no kit. -->

### Constitution: Toolkit Standards Article

#### R1: New `### Toolkit Standards` article under Additional Constraints
The constitution MUST gain a `### Toolkit Standards` article under the existing `## Additional Constraints` section — placed after the section's existing bullet list and before `## Governance` — reproducing the intake's prescribed article text verbatim (em-dashes in the file's `—` convention). The article MUST bind this repo to the sahil87 toolkit's published standards via a standing MUST-conform rule and MUST route enumeration through the `shll standards` command, naming the sahil87/shll `docs/site/standards/` tree and https://shll.ai as the canonical/rendered fallbacks.

- **GIVEN** the constitution with a flat-bullet `## Additional Constraints` section and no `###` subsections
- **WHEN** the amendment is applied
- **THEN** a `### Toolkit Standards` article appears after the last bullet and before `## Governance`
- **AND** its body matches the intake's prescribed text word-for-word, using `—` for dashes

#### R2: Reference-by-enumeration only — no per-standard content
The article MUST NOT name any individual standard, state how many standards exist, or link any per-standard URL. The only permitted references are the command (`shll standards` / `shll standards <name>`), the repository tree (sahil87/shll `docs/site/standards/`), and the rendered site (https://shll.ai). This keeps the article correct as the upstream standard set evolves.

- **GIVEN** the prescribed article text
- **WHEN** the article is inserted
- **THEN** it contains no standard names, no counts, and no per-standard URLs
- **AND** the only external references present are `shll standards`, `sahil87/shll docs/site/standards/`, and https://shll.ai

#### R3: Governance line version + Last-Amended bump
The governance line MUST be updated from `**Version**: 1.3.0 | **Ratified**: 2026-02-06 | **Last Amended**: 2026-06-01` to `**Version**: 1.4.0 | **Ratified**: 2026-02-06 | **Last Amended**: 2026-07-18`. The version bump is minor (1.3.0 → 1.4.0) because the amendment adds a new normative MUST-rule; the Ratified date is unchanged.

- **GIVEN** the current governance line at version 1.3.0, last amended 2026-06-01
- **WHEN** the amendment is applied
- **THEN** the line reads version 1.4.0, ratified 2026-02-06, last amended 2026-07-18

#### R4: Dated amendment record comment
A third dated amendment HTML comment MUST be appended below the governance line, following the file's two existing `<!-- YYYY-MM-DD (change-id): ... -->` amendment records, carrying the intake's prescribed comment text (change id `260717-y8it`, dated 2026-07-18, noting the new MUST-rule and the 1.3.0 → 1.4.0 bump).

- **GIVEN** two existing dated amendment comments below the governance line
- **WHEN** the amendment is applied
- **THEN** a third dated comment for `260717-y8it` follows them, matching the established pattern

### Non-Goals

- No conformance fixes to any governed surface (CLI, help output, README.md, docs/site/) — this change only installs the obligation.
- No changes to kit sources (`src/kit/`), the Go binary, tests, or `docs/specs/` — `fab/project/constitution.md` is project config, not kit content, so the constitution's own skill-file/CLI mirror constraints do not apply and no SPEC mirror exists.
- No copying of the standards' content or enumeration into this repo.

### Design Decisions

1. **Reference-by-enumeration, not enumeration-by-copy**: the article names `shll standards` as the live enumeration and the sahil87/shll tree as canonical source — *Why*: standards added/revised upstream bind this repo immediately with no re-amendment, so the article can never go stale — *Rejected*: listing the current standards inline (more scannable but rots the moment shll adds or renames a standard).
2. **Minor version bump 1.3.0 → 1.4.0**: *Why*: this amendment adds a new normative MUST-rule, unlike the two prior amendments (each flagged "no new normative MUST-rule was added") — *Rejected*: patch bump (would understate that a new binding rule was introduced).

## Tasks

### Phase 1: Amend the constitution

- [x] T001 Insert the `### Toolkit Standards` article into `fab/project/constitution.md` after the last bullet of `## Additional Constraints` (current line 34) and before `## Governance` (current line 36), reproducing the intake's prescribed article text verbatim with `—` dashes <!-- R1 --> <!-- R2 -->
- [x] T002 Update the governance line in `fab/project/constitution.md` from `1.3.0 ... Last Amended: 2026-06-01` to `1.4.0 ... Last Amended: 2026-07-18` (Ratified unchanged) <!-- R3 -->
- [x] T003 Append the third dated amendment HTML comment for `260717-y8it` below the governance line in `fab/project/constitution.md`, after the two existing amendment comments, per the intake's prescribed comment text <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: The `### Toolkit Standards` article exists under `## Additional Constraints`, after the bullets and before `## Governance`, with body text matching the intake verbatim
- [x] A-002 R3: The governance line reads `**Version**: 1.4.0 | **Ratified**: 2026-02-06 | **Last Amended**: 2026-07-18`
- [x] A-003 R4: A third dated amendment comment for `260717-y8it` (2026-07-18) follows the two existing amendment comments below the governance line

### Behavioral Correctness

- [x] A-004 R2: The article contains no individual standard names, no counts, and no per-standard URLs — only `shll standards` / `shll standards <name>`, `sahil87/shll docs/site/standards/`, and https://shll.ai

### Scenario Coverage

- [x] A-005 R1: Section ordering is preserved — Core Principles → Additional Constraints (bullets, then the new article) → Governance (line, then the three amendment comments)

### Code Quality

- [x] A-006 Pattern consistency: The article's heading level (`###`), amendment-comment format (`<!-- YYYY-MM-DD (change-id): ... -->`), and dash convention (`—`) match the surrounding file
- [x] A-007 No unnecessary duplication: Existing constitution content (Core Principles, other Additional Constraints bullets, prior amendment comments) is preserved unchanged; no content is duplicated
- [x] A-008 Markdown-only artifact: The change is standard CommonMark markdown, diffable and readable (Constitution IV)
- [x] A-009 Canonical source only: The edit touches only `fab/project/constitution.md` (project config), not `.claude/skills/` or `src/kit/`; no SPEC mirror applies

### documentation_accuracy

- [x] A-010 The article text is reproduced verbatim from the intake with no paraphrase, no added or dropped clauses, and dashes normalized to `—`; the version/date bump matches the intake exactly

### cross_references

- [x] A-011 All references in the article resolve to the intended targets — `shll standards` (command), `sahil87/shll docs/site/standards/` (repo tree), https://shll.ai (rendered site) — and no per-standard link is introduced

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Article text, heading (`### Toolkit Standards`), and placement (after the `## Additional Constraints` bullets, before `## Governance`) are taken verbatim from the intake's What Changes block | Fully prescribed; the section already exists so the create-if-lacking branch is moot | S:95 R:90 A:100 D:100 |
| 2 | Certain | No standard names, counts, or per-standard URLs — only `shll standards`, sahil87/shll `docs/site/standards/`, and https://shll.ai | Explicit deliberate binding constraint in the intake | S:95 R:85 A:100 D:100 |
| 3 | Certain | Governance line becomes `1.4.0 | 2026-02-06 | 2026-07-18` and a third dated amendment comment is appended below it, matching the two existing comments' pattern | Prescribed verbatim in the intake; the file's own structure confirms the amendment-comment pattern | S:90 R:90 A:95 D:95 |
| 4 | Certain | Scope is exactly one content file (`fab/project/constitution.md`); no conformance fixes, no kit/Go/spec/test changes, no SPEC mirror | Intake states docs-type change, no conformance fixes, and that constitution is project config not kit content | S:90 R:90 A:100 D:95 |
| 5 | Confident | The article's prescribed text is already em-dash-normalized in the intake's code block, so it is reproduced as-is (no further `--` conversion needed) | The intake's What Changes block presents the final normalized form; matching house typography changes no meaning | S:70 R:95 A:85 D:80 |

5 assumptions (4 certain, 1 confident, 0 tentative).
