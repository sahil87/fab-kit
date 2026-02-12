# Constitution Governance

**Domain**: fab-workflow

## Overview

`fab/constitution.md` holds the project's principles and constraints — the architectural DNA that governs how specifications become code. This doc covers the amendment workflow and versioning model managed by `/fab-init-constitution`.

## Amendment Workflow

### Creating a Constitution

On first run (or when `constitution.md` doesn't exist), `/fab-init-constitution` generates a constitution from project context: `config.yaml`, README, codebase patterns, and conversation.

### Amending an Existing Constitution

Run `/fab-init-constitution` when the file exists to enter update mode:

1. Current constitution is displayed
2. Amendment menu offers: add principle, modify principle, remove principle, add/modify constraint, update governance
3. Multiple amendments can be made per session
4. Version is bumped automatically based on change severity

## Semantic Versioning

Constitution versions follow `MAJOR.MINOR.PATCH`:

| Change type | Bump | Example |
|-------------|------|---------|
| Remove or fundamentally change a principle | MAJOR | `1.2.0 → 2.0.0` |
| Add a new principle or constraint | MINOR | `1.2.0 → 1.3.0` |
| Clarify wording without changing meaning | PATCH | `1.2.0 → 1.2.1` |

When multiple amendments are made in one session, the highest-severity bump takes precedence (MAJOR > MINOR > PATCH).

## Structural Rules

The constitution maintains a consistent structure:

- Level-1 heading: `# {Project Name} Constitution`
- `## Core Principles` with Roman numeral headings (`### I.`, `### II.`, etc.)
- `## Additional Constraints`
- `## Governance` with version, ratified date, and last amended date

When principles are removed, remaining principles are re-numbered sequentially.

## Audit Trail

Amendment summaries are included in the command output. The constitution file itself does not contain a changelog — git history serves as the authoritative record, and the version number provides semantic signal.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260212-h9k3-fab-init-family | 2026-02-12 | Initial creation — constitutional amendment workflow with `/fab-init-constitution` |
