# Tasks: Create changeman.sh and Refactor fab-new Skill

**Change**: 260215-9yjx-DEV-1022-create-changeman-script
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/scripts/lib/changeman.sh` script skeleton — shebang (`#!/usr/bin/env bash`), `set -euo pipefail`, path resolution via `readlink -f` (derive `LIB_DIR`, `FAB_ROOT`, `STAGEMAN`), `--help` subcommand with usage text, CLI dispatch (`case` block routing `new` and `--help`)

## Phase 2: Core Implementation

- [x] T002 Implement `new` subcommand argument parsing in `fab/.kit/scripts/lib/changeman.sh` — parse `--slug`, `--change-id`, `--log-args` flags via `while` loop with `shift`; validate `--slug` is non-empty and matches `^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]$` (no trailing hyphen); validate `--change-id` matches `^[a-z0-9]{4}$` if provided; error to stderr with non-zero exit on validation failure
- [x] T003 Implement change creation logic in `fab/.kit/scripts/lib/changeman.sh` — date generation (`date +%y%m%d`), random ID generation (`LC_ALL=C tr -dc 'a-z0-9' < /dev/urandom | head -c4`), folder name construction (`{YYMMDD}-{XXXX}-{slug}`), collision detection against `fab/changes/` (fatal for provided IDs, retry up to 10 for random), directory creation via `mkdir`
- [x] T004 Implement `created_by` detection and `.status.yaml` initialization in `fab/.kit/scripts/lib/changeman.sh` — fallback chain: `gh api user --jq .login` → `git config user.name` → `"unknown"` (silent failures); read `fab/.kit/templates/status.yaml` and `sed` substitute `{NAME}`, `{CREATED}` (`date -Iseconds`), `{CREATED_BY}`; write to `fab/changes/{name}/.status.yaml`
- [x] T005 Implement stageman integration and stdout output in `fab/.kit/scripts/lib/changeman.sh` — call `$STAGEMAN set-state <status_file> intake active fab-new`; conditionally call `$STAGEMAN log-command <change_dir> "fab-new" "<log_args>"` when `--log-args` provided; print folder name (not full path) to stdout as final line

## Phase 3: Integration

- [x] T006 Refactor `fab/.kit/skills/fab-new.md` — consolidate Steps 3 (Create Change Directory) and 4 (Initialize `.status.yaml`) into a single Step 3 calling `changeman.sh new` with appropriate flags; update Step 1 to focus on slug generation only (AI task); renumber remaining steps; simplify error handling table (collapse collision/creation/template entries into single "changeman.sh failure" row)
- [x] T007 Smoke test — run `changeman.sh new --slug smoke-test-change` end-to-end; verify: directory created under `fab/changes/`, `.status.yaml` contains correct name/created_by/progress, `intake: active` in progress, `.history.jsonl` exists when `--log-args` provided; clean up test directory

---

## Execution Order

- T001 blocks T002–T005 (skeleton needed first)
- T002 blocks T003 (parsing needed before creation logic)
- T003 blocks T004 (directory must exist for .status.yaml write)
- T004 blocks T005 (status file must exist for stageman calls)
- T005 blocks T006 (changeman.sh must be complete before refactoring the skill)
- T007 runs after T005 (needs complete script)
- T006 and T007 are independent of each other
