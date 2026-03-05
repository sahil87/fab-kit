# Quality Checklist: Create changeman.sh and Refactor fab-new Skill

**Change**: 260215-9yjx-DEV-1022-create-changeman-script
**Generated**: 2026-02-15
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Script Conventions: changeman.sh uses `#!/usr/bin/env bash`, `set -euo pipefail`, `readlink -f` path resolution, derives FAB_ROOT and STAGEMAN correctly
- [x] CHK-002 CLI Interface: `new` subcommand accepts `--slug`, `--change-id`, `--log-args`; `--help` prints usage; unknown subcommands error
- [x] CHK-003 Date Generation: date prefix is YYMMDD format via `date +%y%m%d`
- [x] CHK-004 Change ID Generation: random ID is 4 chars from `[a-z0-9]` via `/dev/urandom`; provided ID validated as `^[a-z0-9]{4}$`
- [x] CHK-005 Slug Validation: non-empty, alphanumeric+hyphen pattern, no trailing hyphen; uppercase allowed (issue IDs)
- [x] CHK-006 Folder Name Construction: format is `{YYMMDD}-{XXXX}-{slug}` with correct hyphen joining
- [x] CHK-007 Collision Detection: provided ID collision is fatal; random ID collision retries up to 10 times
- [x] CHK-008 Directory Creation: uses `mkdir` (not `mkdir -p`)
- [x] CHK-009 created_by Detection: fallback chain `gh api user` → `git config user.name` → `"unknown"`, silent failures
- [x] CHK-010 .status.yaml Initialization: template read via sed, all three placeholders (`{NAME}`, `{CREATED}`, `{CREATED_BY}`) substituted
- [x] CHK-011 Stageman Integration: calls `set-state intake active fab-new`; conditionally calls `log-command` with `--log-args`
- [x] CHK-012 Output Contract: stdout contains only folder name (one line); errors to stderr with `ERROR:` prefix; exit 0 on success
- [x] CHK-013 fab-new Refactoring: Steps 3+4 collapsed into single changeman.sh call; error table simplified

## Behavioral Correctness

- [x] CHK-014 fab-new produces identical end state: same directory structure, .status.yaml content, and .history.jsonl entry as the current inline implementation
- [x] CHK-015 fab-new step renumbering is consistent (no gaps, no duplicate step numbers)

## Scenario Coverage

- [x] CHK-016 Minimal invocation: `changeman.sh new --slug add-oauth` creates directory with random ID
- [x] CHK-017 Full invocation: `changeman.sh new --slug DEV-988-add-oauth --change-id a7k2 --log-args "desc"` creates correct directory with all integrations
- [x] CHK-018 Missing --slug flag produces error
- [x] CHK-019 Invalid change-id (wrong length, uppercase) produces error
- [x] CHK-020 Provided ID collision produces fatal error (no retry)
- [x] CHK-021 Random ID collision triggers retry

## Edge Cases & Error Handling

- [x] CHK-022 Empty slug rejected with clear error message
- [x] CHK-023 Slug with invalid characters rejected
- [x] CHK-024 Stageman call failure propagates (script exits non-zero)
- [x] CHK-025 No cleanup on partial failure (directory left on disk)

## Code Quality

- [x] CHK-026 Pattern consistency: changeman.sh follows stageman.sh conventions (path resolution, error format, CLI dispatch)
- [x] CHK-027 No unnecessary duplication: reuses STAGEMAN variable, no inline yq/stageman logic

## Documentation Accuracy

- [x] CHK-028 --help output accurately describes all flags and subcommands
- [x] CHK-029 fab-new.md error table matches changeman.sh error behavior

## Cross References

- [x] CHK-030 changeman.sh references to stageman.sh use correct CLI subcommand names
- [x] CHK-031 fab-new.md references changeman.sh with correct flag names

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
