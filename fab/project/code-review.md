# Code Review

<!-- Optional review policy consumed by the validation sub-agent during review.
     Projects opt in by creating this file. All sections are independently optional.
     Delete or leave empty any section that doesn't apply to your project.

     This file guides the REVIEWING agent (critic). For the WRITING agent (author),
     see code-quality.md. Different cognitive modes, different concerns. -->

## Severity Definitions

<!-- How findings are prioritized. The review sub-agent classifies each finding
     into one of these tiers. Override the defaults below to match your project's
     quality bar. -->

- **Must-fix**: Spec mismatches, failing tests, checklist violations — always addressed during rework
- **Should-fix**: Code quality issues, pattern inconsistencies — addressed when clear and low-effort
- **Nice-to-have**: Style suggestions, minor improvements — may be skipped

## Review Scope

<!-- What the review sub-agent inspects. Adjust to exclude generated code,
     vendor directories, or other paths that shouldn't be reviewed. -->

- Changed files only (files touched during apply)
- Skip generated code and vendor directories
- Skip binary files and assets

## False Positive Policy

<!-- How to suppress or override findings the reviewer flags incorrectly.
     Use inline comments in source code to mark intentional deviations. -->

- Inline `<!-- review-ignore: {reason} -->` in markdown files
- Inline `// review-ignore: {reason}` or `# review-ignore: {reason}` in code files
- Suppressed findings are noted in the review report but not counted as failures

## Rework Budget

<!-- Max auto-rework cycles before escalating to the user.
     Applies to /fab-fff and /fab-ff auto-rework loops. -->

- Max cycles: 3
- After 2 consecutive "fix code" attempts on the same issue, escalate to "revise plan" or "revise requirements"

## Project-Specific Review Rules

<!-- These are must-fix unless noted. They encode the constitution's Additional
     Constraints and the recurring rework causes for this repo. -->

- **SPEC-mirror sync** — a `src/kit/skills/*.md` change MUST carry the matching `docs/specs/skills/SPEC-*.md` update. Treat the *whole* mirror class as in-scope, not just files with the literal changed phrase (reviewers read this strictly). See code-quality.md § Sibling & Mirror Sweeps.
- **CLI ⇒ docs + tests** — a change to the `fab` Go binary's command signatures MUST update `src/kit/skills/_cli-fab.md` and include corresponding test updates.
- **Canonical source only** — flag any edit under `.claude/skills/` (gitignored deployed copies); kit changes belong in `src/kit/`.
- **Migrations for user-data restructuring** — changes touching config/`.status.yaml`/archive layout MUST ship a `src/kit/migrations/` file, not an ad-hoc script.
- **Go changes ship tests** — a `.go` change without accompanying test updates is a must-fix gap; tests conform to the spec, never the reverse (Constitution VII).
- **Markdown-only artifacts** — no binary/proprietary formats; standard CommonMark.
