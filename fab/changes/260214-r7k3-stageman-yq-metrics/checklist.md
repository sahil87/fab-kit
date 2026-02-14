# Quality Checklist: Stage Metrics, History Tracking, and stageman yq Migration

**Change**: 260214-r7k3-stageman-yq-metrics
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Status metrics schema: `.status.yaml` supports `stage_metrics.<stage>.{started_at,completed_at,driver,iterations}`.
- [x] CHK-002 Stage completion metadata: `completed_at` is set on `done` without clobbering prior stage metrics fields.
- [x] CHK-003 Iteration counting: each transition to `active` increments `iterations` exactly once.
- [x] CHK-004 Command history format: `.history.jsonl` is append-only JSONL with required `ts`, `event`, and `outcome` fields.
- [x] CHK-005 Outcome coverage: failed command invocations are logged with `outcome: error`.
- [x] CHK-006 Confidence/review history events: `log_confidence` and `log_review` emit required event payload fields.
- [x] CHK-007 yq status reads: stageman accessors read status via `yq`, not regex parsing.
- [x] CHK-008 yq status writes: stageman writers mutate via `yq`, preserving unrelated fields.
- [x] CHK-009 yq-backed validation: invalid states are rejected through `validate_status_file`.
- [x] CHK-010 Missing-yq behavior: stageman/calc-score fail fast with clear install guidance and no awk fallback.
- [x] CHK-011 calc-score integration: score writes use stageman API and append confidence history.
- [x] CHK-012 Templates/schemas updated: status template and workflow schema reflect `stage_metrics` support.
- [x] CHK-013 Skill integration: planning/execution skills pass driver identity and log command events.
- [x] CHK-014 Governance alignment: constitution explicitly permits single-binary runtime tooling like `yq`.

## Behavioral Correctness

- [x] CHK-015 Existing stage progression behavior remains unchanged except added metrics/history side-effects.
- [x] CHK-016 `last_updated` refresh behavior remains consistent across all status mutations.

## Scenario Coverage

- [x] CHK-017 Transition scenarios verify `started_at`/`completed_at` behavior for `set-state` and `transition` paths.
- [x] CHK-018 Rework scenario verifies `iterations` increments when a stage re-enters `active`.
- [x] CHK-019 History scenarios verify both success and error command outcomes are appended, not overwritten.

## Edge Cases & Error Handling

- [x] CHK-020 Missing `yq` exits non-zero before status read/write and returns installation guidance.
- [x] CHK-021 Invalid checklist/confidence/stage inputs leave `.status.yaml` unchanged.

## Documentation Accuracy

- [x] CHK-022 Developer docs for stageman/calc-score match implemented APIs, dependency requirements, and behavior.

## Cross References

- [x] CHK-023 Memory docs (`kit-architecture`, `change-lifecycle`, `planning-skills`, `execution-skills`) reflect shipped behavior and script paths.

## Notes

- Mark non-applicable checks as `- [x] CHK-xxx **N/A**: reason`.
