# Stage Manager (stageman) Changelog

## 2026-02-12 - Renamed from workflow-lib to stageman

### Breaking Changes

- Renamed `workflow-lib.sh` → `stageman.sh`
- Renamed directory `src/workflow-lib/` → `src/stageman/`
- Updated all references and documentation

### Structure Changes

- **Main file**: `fab/.kit/scripts/stageman.sh` (was symlink, now main implementation)
- **Dev symlink**: `src/stageman/stageman.sh` → `../../fab/.kit/scripts/stageman.sh`
- Reversed symlink direction for cleaner distribution

### Rationale

"Stage Manager" (stageman) better describes the utility's purpose:
- Manages knowledge about stages and their states
- Query utility for workflow progression
- Single point of truth for stage/state definitions

## 2026-02-12 - Initial Creation

Created workflow query library with 20+ functions for stages, states, progression, and validation.
