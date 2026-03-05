# Tasks: Orchestrator Idle Hooks

**Change**: 260305-bs5x-orchestrator-idle-hooks
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/hooks/` directory with `.gitkeep`

## Phase 2: Core Implementation

- [x] T002 [P] Create `fab/.kit/hooks/on-stop.sh` — Stop hook that writes `agent.idle_since` timestamp to `.status.yaml`. Must: resolve repo root via `git rev-parse --show-toplevel`, read `fab/current`, resolve change dir via `fab/.kit/bin/fab resolve --dir`, write `agent.idle_since` via `yq -i`, exit 0 on all error paths (no fab/current, no change dir, no .status.yaml, no yq, no fab dispatcher)
<!-- clarified: added explicit error paths to T003 matching spec "Hook Scripts Must Never Block" requirement (no fab/current, no change dir, no .status.yaml, no yq, no fab dispatcher) — was implied by "same resolution pattern" but now explicit for parity with T002 -->
- [x] T003 [P] Create `fab/.kit/hooks/on-session-start.sh` — SessionStart hook that clears the `agent` block from `.status.yaml`. Must: same resolution pattern as T002, delete `agent` block via `yq -i 'del(.agent)'`, exit 0 on all error paths (no fab/current, no change dir, no .status.yaml, no yq, no fab dispatcher), idempotent when no `agent` block exists
<!-- clarified: added hook entry format requirement ("type":"command") from spec "Hook Entry Format" section — was missing from task description -->
- [x] T004 Create `fab/.kit/sync/5-sync-hooks.sh` — Sync script that registers hooks from `fab/.kit/hooks/` into `.claude/settings.local.json`. Must: discover `on-*.sh` files (ignore non-matching like `helper.sh`), map filenames to Claude Code events (hardcoded: `on-session-start.sh`→`SessionStart`, `on-stop.sh`→`Stop`), build entries as `{"type":"command","command":"bash fab/.kit/hooks/{filename}"}`, merge into `hooks.*` arrays via `jq`, idempotent (no duplicates via command-field comparison, preserves user hooks), handle missing jq (warn and skip), handle missing settings file (create), handle no hook scripts (silent skip)

## Phase 3: Integration & Edge Cases

<!-- clarified: added missing test case for T005 — fab dispatcher not available exits 0 — from spec "fab Dispatcher Not Available" scenario -->
- [x] T005 Add BATS tests for `on-stop.sh` — cover: active change writes timestamp, no fab/current exits 0, missing change dir exits 0, missing .status.yaml exits 0, yq not available exits 0, fab dispatcher not available exits 0. Test file: `src/hooks/test-on-stop.bats`
<!-- clarified: added missing test cases for T006 — yq not available and fab dispatcher not available — matching spec "Hook Scripts Must Never Block" requirement that applies to both hooks -->
- [x] T006 Add BATS tests for `on-session-start.sh` — cover: active change clears agent block, no agent block is idempotent, no fab/current exits 0, missing change dir exits 0, missing .status.yaml exits 0, yq not available exits 0, fab dispatcher not available exits 0. Test file: `src/hooks/test-on-session-start.bats`
<!-- clarified: added missing test case for T007 — unknown hook script (helper.sh) ignored — from spec "Unknown Hook Script" scenario -->
- [x] T007 Add BATS tests for `5-sync-hooks.sh` — cover: first sync creates hooks, idempotent re-sync, preserves user hooks, no jq warns and skips, no hook scripts is silent, missing settings.local.json creates file, unknown hook script (helper.sh) is ignored. Test file: `src/sync/test-5-sync-hooks.bats`

---

## Execution Order

- T001 blocks T002, T003, T004
- T002 and T003 are independent ([P])
- T004 depends on T001 (needs hooks dir to exist for discovery)
- T005 depends on T002, T006 depends on T003, T007 depends on T004
