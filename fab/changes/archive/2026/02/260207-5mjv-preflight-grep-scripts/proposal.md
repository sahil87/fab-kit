# Proposal: Add fab-preflight.sh and update skills to consume it

**Change**: 260207-5mjv-preflight-grep-scripts
**Created**: 2026-02-07
**Status**: Draft

## Why

Every fab skill (ff, apply, review, archive, continue) repeats the same pre-flight sequence: read `fab/current`, validate the change directory exists, read `.status.yaml`, load `config.yaml`, `constitution.md`, and `fab/docs/index.md`. That's 5+ file reads as boilerplate before any real work begins. Consolidating this into a single script reduces duplication across skills and provides a single validation point. Updating existing skills to reference the script completes the picture — introducing a utility without wiring it in would leave the duplication in place.

## What Changes

- **New script `fab/.kit/scripts/fab-preflight.sh`**: Reads `fab/current`, validates the change directory and `.status.yaml` exist, and outputs structured YAML containing the change name, stage, full progress map, checklist counts, and branch name. Skills can consume this output instead of independently re-reading and re-validating.
- **Update `_context.md`**: Reference `fab-preflight.sh` in the "Change Context" loading layer so the convention is documented centrally.
- **Update skill files**: Update skills that perform pre-flight loading (ff, apply, review, archive, continue, clarify) to reference `fab-preflight.sh` instead of inline context loading instructions.

## Affected Docs

### New Docs
- `fab-workflow/preflight`: Documents the `fab-preflight.sh` script — purpose, output format, usage by skills

### Modified Docs
- `fab-workflow/context-loading`: Update to reference `fab-preflight.sh` as the recommended mechanism for the "Change Context" loading layer

### Removed Docs
<!-- None -->

## Impact

- **`fab/.kit/scripts/`**: One new shell script added
- **`fab/.kit/skills/_context.md`**: Updated to document preflight as the standard change-context mechanism
- **`fab/.kit/skills/fab-*.md`**: Skills that load change context updated to reference the script
- **Portability**: Script uses only POSIX-compatible tools (`grep`, `cat`, `sed`, `awk`) and follows the existing `set -euo pipefail` convention from other fab scripts. No new dependencies.

## Open Questions

- [DEFERRED] Should `fab-preflight.sh` also output paths to completed artifacts (proposal.md, spec.md, etc.) for convenience, or keep output minimal? (Assumed: minimal — skills already know the artifact naming convention.)
