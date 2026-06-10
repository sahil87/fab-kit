# Plan: Tear Down Push-Side shll.ai Integration

**Change**: 260603-mtf9-teardown-shll-push
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### Release CI: Remove the push-side shll.ai transport

#### R1: Delete the `Help-dump → shll.ai` release step
The `Help-dump → shll.ai` step in `.github/workflows/release.yml` (its leading comment
block plus the full step body — the `name:`, `env: SHLLAI_TOKEN`, and `run:` script) MUST
be removed in its entirety. This removes BOTH the fatal `help-dump > help/fab-kit.json` +
`jq` dump/validate self-check AND the auto-merging cross-repo PR transport (the `git clone`
of `sahil87/shll.ai`, the `help-dump/fab-kit-<version>` branch, `gh pr create`, and
`gh pr merge --auto --squash`). No other step consumes this step's outputs.

- **GIVEN** `.github/workflows/release.yml` with the `Help-dump → shll.ai` step between
  `Build all targets` and `Package kit archives`
- **WHEN** the step (and its leading comment block) is deleted
- **THEN** the workflow no longer references `shll`, `SHLLAI`, `help-dump`, or
  `help/fab-kit.json`
- **AND** no other step is affected (the step's only output, `help/fab-kit.json`, is
  consumed solely by the removed PR sub-step)

#### R2: Preserve step adjacency and YAML well-formedness
After removing the step, `Build all targets` MUST be immediately followed by
`Package kit archives`, and the workflow MUST remain valid, well-formed YAML (correct
2-space step indentation, no orphaned `env:`/`run:` fragments, no dangling comment block).

- **GIVEN** the `Help-dump → shll.ai` step removed from the steps list
- **WHEN** the YAML is parsed
- **THEN** it parses successfully as valid YAML with `jobs.release.steps` intact
- **AND** the step ordering is `... → Build all targets → Package kit archives → ...` with
  no gap, no leftover fragment, and consistent indentation

### Repo hygiene: Remove dead push-side residue

#### R3: Remove the dead `/help/` `.gitignore` entry
The `/help/` ignore entry in `.gitignore` (and its dedicated leading comment) MUST be
removed. With the producing step gone, nothing writes to `help/`, so the entry is dead.

- **GIVEN** `.gitignore` containing the `/help/` entry and its
  `# Transient CI help-dump artifact ...` comment
- **WHEN** the entry and its comment are removed
- **THEN** `.gitignore` no longer contains a `/help/` line
- **AND** the surrounding entries remain intact and the file is otherwise unchanged

### Non-Goals

- **The `help-dump` command itself** (`src/go/fab/cmd/fab/helpdump.go`,
  `src/go/fab/cmd/fab/helpdump_test.go`) — explicitly OUT OF SCOPE. This is the contract
  surface shll.ai's puller invokes and is a hard invariant from the shll.ai contract: do
  NOT touch it. The command stays covered by its existing Go unit tests.
- **`src/kit/skills/_cli-fab.md`** — the `help-dump` command docs survive; only the CI
  transport is removed. Do NOT modify.
- **Memory** (`docs/memory/fab-workflow/distribution.md`) — a HYDRATE-stage concern. Apply
  touches only code/config/workflow + the plan artifact, never memory.
- **`SHLLAI_TOKEN` GitHub repo-secret deletion** — an out-of-band manual GitHub-settings
  follow-up, not a file edit. Noted in Acceptance as a manual follow-up; apply does not
  attempt it.

### Design Decisions

1. **Remove the entire step, not just the PR transport**: delete the dump+validate
   self-check too — *Why*: the user explicitly chose "Remove entirely"; `help-dump` stays
   exercised by `helpdump_test.go`, so release-time validation is redundant. *Rejected*:
   keeping the dump+validate as a release smoke test (extra dead weight with no consumer
   now that shll.ai pulls).
2. **Remove the `/help/` `.gitignore` entry as part of teardown**: *Why*: nothing writes
   `help/` once the step is gone; leaves no push-side residue. *Rejected*: leaving it
   (cosmetic dead config; the contract's spirit is to leave no residue).

## Tasks

### Phase 2: Core Implementation

- [x] T001 Delete the `Help-dump → shll.ai` step (leading comment block + `name:`/`env:`/`run:` body) from `.github/workflows/release.yml` so `Build all targets` is immediately followed by `Package kit archives` <!-- R1 -->
- [x] T002 Remove the `/help/` entry and its `# Transient CI help-dump artifact ...` comment from `.gitignore` <!-- R3 -->

### Phase 3: Integration & Edge Cases

- [x] T003 Verify `.github/workflows/release.yml` parses as valid YAML and step adjacency/indentation is correct (no orphaned `env:`/`run:` fragments) <!-- R2 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: The `Help-dump → shll.ai` step is fully removed from `.github/workflows/release.yml` — `grep -ni 'shll\|SHLLAI\|help-dump\|fab-kit.json' .github/workflows/release.yml` returns nothing
- [ ] A-002 R3: `.gitignore` no longer contains a `/help/` entry (nor its dedicated comment)

### Behavioral Correctness

- [ ] A-003 R2: `Build all targets` is immediately followed by `Package kit archives` in the steps list, with no intervening step or fragment

### Scenario Coverage

- [ ] A-004 R2: `.github/workflows/release.yml` parses as valid YAML (PyYAML/`yq` check passes) with `jobs.release.steps` intact and consistent indentation

### Removal Verification

- [ ] A-005 R1: No orphaned `env:`/`run:` fragments, dangling comment block, or dead `SHLLAI_TOKEN` reference remain anywhere in `.github/workflows/release.yml`

### Edge Cases & Error Handling

- [ ] A-006 R1: The `help-dump` contract surface still works — `git diff --stat src/go/fab/cmd/fab/helpdump.go src/go/fab/cmd/fab/helpdump_test.go` is empty AND `go test ./cmd/fab/ -run 'DumpDoc|BuildNode|HelpDumpCmd'` passes (release-time self-check removal does not regress the command)

### Code Quality

- [ ] A-007 Pattern consistency: Remaining workflow steps and `.gitignore` entries retain their existing style and indentation
- [ ] A-008 No unnecessary duplication: No new config or steps introduced; only dead push-side residue removed

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- **Manual follow-up (out-of-band, not a file edit)**: delete the `SHLLAI_TOKEN` GitHub
  repo secret from fab-kit's repo settings. A repo-wide grep confirmed it is referenced
  ONLY in the removed `Help-dump → shll.ai` step, so deletion is safe after this change
  merges. This happens in GitHub settings; the code change cannot perform it.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove the ENTIRE `Help-dump → shll.ai` step including the fatal dump+validate self-check, not just the PR transport. | Intake assumption #2 (Certain) + Design Decision 1 — user explicitly chose "Remove entirely"; `help-dump` stays covered by `helpdump_test.go`. | S:95 R:70 A:85 D:90 |
| 2 | Certain | Do NOT touch `helpdump.go` / `helpdump_test.go` / `_cli-fab.md` — contract surface and docs survive. | Hard invariant from the shll.ai contract restated in intake assumption #3 and the apply directive. | S:100 R:80 A:95 D:95 |
| 3 | Certain | The leading comment block (release.yml ~73-76) is part of the step and is removed with it. | The comment exclusively documents the step being removed; leaving it would be orphaned/misleading residue. Apply directive says "including its leading comment block." | S:95 R:85 A:90 D:95 |
| 4 | Confident | Remove the `/help/` `.gitignore` entry together with its dedicated `# Transient CI help-dump artifact ...` comment, not just the bare line. | Intake assumption #4 (Confident) — the comment exclusively annotates the `/help/` entry; removing the line but leaving the comment would orphan it. Cosmetic, trivially reversible. | S:75 R:90 A:88 D:85 |
| 5 | Confident | `SHLLAI_TOKEN` is referenced only by the removed step; its repo-secret deletion is a manual out-of-band follow-up, not an apply edit. | Intake assumption #5 (Confident) + verified by repo-wide grep at apply (all 5 `SHLLAI_TOKEN` references are inside the removed step). Secret deletion is a GitHub-settings task. | S:90 R:75 A:90 D:90 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
