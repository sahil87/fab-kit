# Quality Checklist: Pipeline Orchestrator

**Change**: 260221-wy0e-pipeline-orchestrator
**Generated**: 2026-02-21
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 YAML Manifest Schema: Manifest parser accepts valid manifests with `base`, `changes[].id`, `changes[].depends_on`, optional `stage`
- [ ] CHK-002 Live Editing Contract: Orchestrator re-reads manifest from disk on every loop iteration; human-added entries preserved
- [ ] CHK-003 Main Dispatch Loop: run.sh runs indefinitely until SIGINT, dispatches serially, idle-polls when no work
- [ ] CHK-004 Resumability: Terminal stages skipped, intermediate stages re-dispatched, absent stages dispatched normally
- [ ] CHK-005 Topological Dispatch Order: Changes dispatched in dependency order; ties broken by manifest list order
- [ ] CHK-006 SIGINT Summary: Ctrl+C prints structured summary with Completed/Failed/Blocked/Skipped/Pending, exits 130
- [ ] CHK-007 Worktree Creation: Root nodes branch from HEAD, dependent nodes branch from parent via `git branch` + `wt-create`
- [ ] CHK-008 Worktree Lifecycle: Worktrees left in place after dispatch (success or failure)
- [ ] CHK-009 Artifact Provisioning: Change folder copied to worktree if not present
- [ ] CHK-010 Prerequisite Validation: intake.md, spec.md, confidence gate checked; invalid written on failure
- [ ] CHK-011 Pipeline Execution: `claude -p --dangerously-skip-permissions` for fab-switch and fab-ff
- [ ] CHK-012 Post-Pipeline Shipping: Claude invoked for commit/push/PR with contextual messages
- [ ] CHK-013 Stage Reporting: Terminal stage written to manifest via yq after dispatch
- [ ] CHK-014 Output: Full Claude output passthrough; `[pipeline]` prefixed status lines
- [ ] CHK-015 Example Scaffold: `fab/pipelines/example.yaml` fully commented-out with all required topics

## Behavioral Correctness

- [ ] CHK-016 Single-dependency constraint: Entries with >1 `depends_on` item rejected with error message
- [ ] CHK-017 Circular dependency detection: Circular deps caught and reported with involved IDs
- [ ] CHK-018 Infrastructure failure abort: wt-create/claude/git failures abort orchestrator with summary

## Scenario Coverage

- [ ] CHK-019 Happy path — linear chain: A → B → C all reach `done`, orchestrator continues polling
- [ ] CHK-020 Failed dependency blocks downstream: A fails, B and C never dispatched
- [ ] CHK-021 Resume after interruption: Intermediate stage re-dispatched in fresh worktree
- [ ] CHK-022 Idle polling picks up new work: Human-added entry discovered and dispatched
- [ ] CHK-023 Root node worktree: Branch created from HEAD (base)
- [ ] CHK-024 Dependent node worktree: Branch created from parent's pushed branch

## Edge Cases & Error Handling

- [ ] CHK-025 Missing required fields: Malformed manifest entry produces clear error
- [ ] CHK-026 Missing spec prerequisite: Change marked `invalid` with reason
- [ ] CHK-027 Confidence below gate: Change marked `invalid` with score and threshold
- [ ] CHK-028 fab-ff failure: Change marked `failed`, orchestrator continues to next change
- [ ] CHK-029 Manifest with no dispatchable changes: Orchestrator idle-polls without error

## Code Quality

- [ ] CHK-030 Pattern consistency: Scripts follow existing `fab/.kit/scripts/` conventions (set -euo pipefail, SCRIPT_DIR, yq usage)
- [ ] CHK-031 No unnecessary duplication: Reuses existing utilities (wt-create, calc-score.sh, stageman.sh, yq patterns)
- [ ] CHK-032 Readability: Functions focused, no god functions (>50 lines without clear reason)

## Documentation Accuracy

- [ ] CHK-033 example.yaml covers all topics listed in spec (base, depends_on, stage values, multi-level deps, prerequisites, live-editing, single-dep)

## Cross References

- [ ] CHK-034 Scripts consistent with spec requirements (stage values, branch naming, error messages match spec)
