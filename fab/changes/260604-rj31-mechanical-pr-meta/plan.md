# Plan: Mechanical PR Meta Block via `fab pr-meta`

**Change**: 260604-rj31-mechanical-pr-meta
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- Requirements absorbed from the intake design. RFC 2119 keywords; each has a
     GIVEN/WHEN/THEN scenario and a stable R# ID. Organized by domain. -->

### CLI: `fab pr-meta` command surface

#### R1: Subcommand registration and signature
The binary MUST expose a `fab pr-meta <change> --type <type> [--issues "<space-joined IDs>"]` subcommand, registered in `main.go` alongside the other subcommands, mirroring the cobra conventions of `fab impact`/`fab score` (positional change ref + flags). `--type` MUST be required.

- **GIVEN** the `fab` binary is built
- **WHEN** `fab pr-meta --help` is run
- **THEN** the command is listed with a `<change>` positional arg, a required `--type` flag, and an optional `--issues` flag
- **AND** invoking it without `--type` exits non-zero with an error

#### R2: Change resolution and self-contained data sourcing
The command MUST resolve `<change>` via the existing `resolve` package (4-char ID, folder substring, or full folder name) and read all remaining inputs itself: `.status.yaml` (via `statusfile`), `plan.md` task checkboxes, `fab/project/config.yaml` (`true_impact_exclude`, `test_paths`, `project.linear_workspace`), the impact math (via `internal/impact`), and git/`gh` context (branch, owner/repo, merge-base). The skill MUST pass only `<change>`, `--type`, and optional `--issues`.

- **GIVEN** an active change with a `.status.yaml` and `plan.md`
- **WHEN** `fab pr-meta <id> --type feat` is run
- **THEN** the command reads status, plan checkboxes, config, impact, and git/gh context without additional flags

### Rendering: the `## Meta` block

#### R3: Meta table
The output MUST begin with `## Meta`, a blank line, then a 5-column table (`ID | Type | Confidence | Plan | Review`) with cell-population rules: ID from `.status.yaml` `id` (`—` fallback); Type the passed `--type`; Confidence `{score}/5.0` (`—` when absent); Plan `{done}/{total} tasks, {acc_done}/{acc_total} acceptance` with a trailing ` ✓` when both pairs are complete and non-zero (`—` when no plan/tasks); Review derived from `progress.review` + `stage_metrics.review.iterations` (`✓ {N} cycle{s}` for done, `✗ {N} cycle{s}` for failed, `—` otherwise; drop count when iterations is 0).

- **GIVEN** a change with `id=rj31`, type `feat`, score 4.2, 8/8 tasks, 9/9 acceptance, review done after 2 cycles
- **WHEN** the Meta block is rendered
- **THEN** the row reads `| rj31 | feat | 4.2/5.0 | 8/8 tasks, 9/9 acceptance ✓ | ✓ 2 cycles |`
- **AND** a change with no plan and pending review renders `| rj31 | feat | 4.2/5.0 | — | — |`

#### R4: Pipeline line
The output MUST render a `**Pipeline**:` line listing the six stages in fixed order (`intake → apply → review → hydrate → ship → review-pr`) joined by ` → `, appending ` ✓` after each stage whose `progress.{stage}` is `done`. The `intake` label MUST hyperlink to the intake blob URL when `intake.md` exists; `apply` MUST hyperlink to the plan (or legacy `tasks.md`) blob URL when present; the other four are always plain text. Blob URLs use `https://github.com/{owner_repo}/blob/{branch}/fab/changes/{name}/{file}`.

- **GIVEN** intake+plan exist, owner/repo `o/r`, branch `b`, `intake`/`apply`/`review` done
- **WHEN** the Pipeline line is rendered
- **THEN** it reads `**Pipeline**: [intake](https://github.com/o/r/blob/b/fab/changes/{name}/intake.md) ✓ → [apply](.../plan.md) ✓ → review ✓ → hydrate → ship → review-pr`

#### R5: Issues line (optional)
When `--issues` is non-empty, the output MUST render a `**Issues**:` line positioned BELOW Pipeline and ABOVE Impact. When `project.linear_workspace` is configured, each ID renders as `[{ID}](https://linear.app/{workspace}/issue/{ID})` joined by `, `; otherwise bare IDs joined by `, `. When `--issues` is empty/absent, the line MUST be omitted.

- **GIVEN** `--issues "DEV-1 DEV-2"` and `linear_workspace: acme`
- **WHEN** rendered
- **THEN** the line reads `**Issues**: [DEV-1](https://linear.app/acme/issue/DEV-1), [DEV-2](https://linear.app/acme/issue/DEV-2)`
- **AND** with no `linear_workspace`, it reads `**Issues**: DEV-1, DEV-2`

#### R6: Impact line — three-row form, single-line form, and omission
The output MUST render the `**Impact**:` line(s) using the impact math. When a `tests` pair is present, it MUST render the three-row impl/tests/total form, deriving `impl` as the per-component clamped residual (`max(0, total − tests)`), using Unicode minus `−` (U+2212), and annotating the total row `← excludes {COMMA_LIST_CODE}` where `{COMMA_LIST_CODE}` is the actual `true_impact_exclude` values each wrapped in single backticks and comma-joined (annotation omitted when excludes empty). When no `tests` pair, it MUST render the single-line `total`/`raw` form, with the `(excluding ...)` clause only when excludes are configured. The Impact line(s) MUST be omitted entirely when the total pass is `+0/−0`, the merge-base cannot be resolved, or impact computation fails. `total` is the `excluding` pass when present, else the raw pass.

- **GIVEN** total `+87/−38 (net 49)`, tests `+40/−0 (net 40)`, excludes `[fab/, docs/]`
- **WHEN** rendered
- **THEN** the impl row reads `  impl:  +47/−38  (net +9)`, tests `  tests: +40/−0  (net +40)`, total `  total: +87/−38  (net +49)  ← excludes ` + "`fab/`, `docs/`"
- **AND** with no tests pair and excludes `[fab/]`, raw `+142/−38`, it reads ``**Impact**: +87/−38 code (excluding `fab/`) · +142/−38 total``
- **AND** when total is `+0/−0` or merge-base is missing, no Impact line is emitted

### Behavior: graceful degradation and exit codes

#### R7: Non-fab / failure exit
The command MUST exit non-zero (emitting nothing on stdout) when the change cannot be resolved or `.status.yaml` is absent, so the skill omits the Meta block exactly as today when `{has_fab}` is false. Unreachable `gh` (no owner/repo) MUST degrade to plain-text stage labels in the Pipeline line, never a hard error; a failed/missing merge-base MUST drop only the Impact line.

- **GIVEN** the CWD has no resolvable change
- **WHEN** `fab pr-meta nonexistent --type feat` is run
- **THEN** the command exits non-zero and prints nothing to stdout
- **AND** when `gh repo view` fails but the change resolves, the Pipeline stages render as plain text and the rest of the block still renders

### Documentation (constitution-mandated)

#### R8: `git-pr.md` delegation
`src/kit/skills/git-pr.md` Step 3c MUST be collapsed so the Meta block is produced by calling `fab pr-meta <change> --type <type> --issues "<issues>"` and pasting its stdout verbatim (omitting the block on non-zero exit / empty stdout). Type resolution (Step 0b), issue gathering (Step 1), and the agent-generated `## Summary` / `## Changes` MUST remain. The progress token MUST stay `✓ body — meta + summary + changes`.

- **GIVEN** the updated `git-pr.md`
- **WHEN** Step 3c is read
- **THEN** it instructs calling `fab pr-meta` and pasting the result, with no inlined Meta table/Pipeline/Impact formatting prose

#### R9: `_cli-fab.md` signature
`src/kit/skills/_cli-fab.md` MUST document the `fab pr-meta` command signature and behavior (per Constitution: CLI changes update `_cli-fab.md`).

- **GIVEN** the updated `_cli-fab.md`
- **WHEN** searched for `pr-meta`
- **THEN** a section documents the signature, flags, self-contained sourcing, output contract, and exit codes

#### R10: `SPEC-git-pr.md` delegation
`docs/specs/skills/SPEC-git-pr.md` MUST reflect the `fab pr-meta` delegation in place of the inlined Meta rendering (per Constitution: skill-file changes update the matching `SPEC-*.md`).

- **GIVEN** the updated `SPEC-git-pr.md`
- **WHEN** the Step 3c flow and Tools-used rows are read
- **THEN** they describe calling `fab pr-meta` rather than assembling the Meta block from `fab impact` + prose

### Design Decisions

1. **Extract rendering into `internal/prmeta`**: pure render functions take structured inputs and return markdown; a `Gather` orchestrator does the I/O (file reads, git/gh shelling). — *Why*: mirrors `internal/impact`/`internal/score`; lets the byte-for-byte render contract be unit-tested without git/gh fixtures (assumption #11). — *Rejected*: inlining all logic in `cmd/fab/pr_meta.go` (harder to test the pure rendering matrix).
2. **`linear_workspace` read directly from config.yaml in prmeta**: the shared `config.Config` struct has no `project.linear_workspace` field; rather than widening the shared struct (used by impact/hooks), `prmeta` parses the one nested field it needs locally. — *Why*: keeps the shared config surface minimal; the field is prmeta-specific. — *Rejected*: adding `Project.LinearWorkspace` to `internal/config` (broader blast radius for a single consumer).
3. **Keep the legacy `tasks.md` Plan fallback for one release**: parse `tasks.md` checkboxes when `plan.md` is absent (assumption #10). — *Why*: matches current skill behavior; cheap to carry.

### Non-Goals

- Rendering `## Summary` / `## Changes` (stays agent prose — intake assumption #2).
- Changing the Meta block content/format — this is mechanization, not redesign (assumption #7); output is byte-for-byte the current Step 3c spec.
- Re-deriving PR `--type` in Go (stays skill-resolved — assumption #8).
- Removing or altering the `fab impact` subcommand (remains public for other consumers).

## Tasks

### Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/prmeta/prmeta.go` with the package skeleton: input structs (`Data` holding status fields, plan counts, impact result, config values, git/gh context, type, issues) and the rendering entry point `Render(d Data) string`. <!-- R3 -->

### Phase 2: Core Implementation (pure rendering)

- [x] T002 Implement the Meta table renderer (header + cell population: ID, Type, Confidence, Plan with ` ✓` completion suffix, Review `✓/✗ {N} cycle{s}`) in `internal/prmeta/prmeta.go`. <!-- R3 -->
- [x] T003 Implement the Pipeline line renderer (six fixed stages, ` ✓` per done stage, hyperlinked intake/apply labels via blob URLs, plain text otherwise) in `internal/prmeta/prmeta.go`. <!-- R4 -->
- [x] T004 Implement the Issues line renderer (Linear-linked when workspace set, bare IDs otherwise, omitted when empty) in `internal/prmeta/prmeta.go`. <!-- R5 -->
- [x] T005 Implement the Impact line renderer (three-row form with per-component `max(0,…)` impl clamp + Unicode minus + backtick-wrapped excludes annotation; single-line form; omission on `+0/−0`/missing impact) in `internal/prmeta/prmeta.go`. <!-- R6 -->

### Phase 3: Integration & data gathering

- [x] T006 Implement `Gather(fabRoot, changeArg, prType, issues string) (Data, bool, error)` in `internal/prmeta/prmeta.go`: resolve the change; read `.status.yaml` (id, confidence, plan counts, progress, review iterations); parse `plan.md` (or legacy `tasks.md`) task checkboxes; read config (`true_impact_exclude`, `test_paths`, `project.linear_workspace`); compute impact via `impact.ComputeForRepo` against the internal merge-base; resolve branch + owner/repo via git/`gh`. Returns ok=false (no error) when there is no fab context so the caller exits non-zero silently. <!-- R2 R7 -->
- [x] T007 Implement git/gh context helpers in `internal/prmeta/prmeta.go`: `git branch --show-current`, merge-base against `origin/main`/`origin/master`, `gh repo view --json nameWithOwner`. `gh` failure degrades to empty owner/repo (plain-text labels); merge-base failure drops the Impact line. <!-- R7 -->
- [x] T008 Add `src/go/fab/cmd/fab/pr_meta.go` with `prMetaCmd()` cobra command (positional `<change>`, required `--type`, optional `--issues`) that calls `prmeta.Gather` + `prmeta.Render`, prints to stdout, and exits non-zero when not in a fab context. Register it in `src/go/fab/cmd/fab/main.go`. <!-- R1 R2 R7 -->

### Phase 4: Documentation

- [x] T009 Collapse `src/kit/skills/git-pr.md` Step 3c body-generation prose to delegate to `fab pr-meta` (call command, paste verbatim, omit on non-zero/empty), preserving type resolution, issue gathering, Summary, Changes, and the `✓ body — meta + summary + changes` token. <!-- R8 -->
- [x] T010 Add the `fab pr-meta` section to `src/kit/skills/_cli-fab.md` (signature, flags, self-contained sourcing, output contract, exit codes); update the command-coverage list line. <!-- R9 -->
- [x] T011 Update `docs/specs/skills/SPEC-git-pr.md` Step 3c flow and Tools-used row to describe the `fab pr-meta` delegation. <!-- R10 -->

### Phase 5: Tests

- [x] T012 [P] Add `src/go/fab/internal/prmeta/prmeta_test.go` — table-driven golden tests for `Render` across the matrix (has/no plan, has/no tests pair, has/no issues, Linear vs bare, review done/failed/pending, empty `true_impact_exclude`, missing merge-base/Impact omission, `+0/−0` omission, plain-text vs hyperlinked stages). <!-- R3 R4 R5 R6 -->
- [x] T013 [P] Add `src/go/fab/cmd/fab/pr_meta_test.go` — command-level tests (required `--type`, non-zero exit when no fab context). <!-- R1 R7 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab pr-meta <change> --type <type> [--issues ...]` is registered in `main.go`, `--type` is required, and the command appears in `--help`.
- [x] A-002 R2: The command resolves the change and self-sources status, plan, config, impact, and git/gh context with no inputs beyond `<change>`/`--type`/`--issues`.
- [x] A-003 R3: The Meta table renders all five columns with correct cell-population and the ` ✓` Plan completion suffix and `✓/✗ {N} cycle{s}` Review cell.
- [x] A-004 R4: The Pipeline line renders six fixed-order stages with ` ✓` per done stage and hyperlinked intake/apply labels.
- [x] A-005 R5: The Issues line renders (Linear-linked or bare) only when `--issues` is non-empty.
- [x] A-006 R6: The Impact line renders the three-row form (with clamped impl, Unicode minus, backtick excludes) when a tests pair exists and the single-line form otherwise.
- [x] A-007 R8: `git-pr.md` Step 3c delegates to `fab pr-meta` with the inlined Meta prose removed.
- [x] A-008 R9: `_cli-fab.md` documents the `fab pr-meta` signature and behavior.
- [x] A-009 R10: `SPEC-git-pr.md` reflects the delegation.

### Behavioral Correctness

- [x] A-010 R6: `impl` is the per-component `max(0, total − tests)` residual — no negative component is ever rendered, and `impl` is derived at render time (not stored).
- [x] A-011 R6: The excludes annotation is built from the actual `true_impact_exclude` config values, each backtick-wrapped — never hardcoded — and is omitted when excludes are empty.

### Scenario Coverage

- [x] A-012 R3 R4 R5 R6: Table-driven golden tests in `prmeta_test.go` cover the rendering matrix and pass.
- [x] A-013 R1 R7: Command-level tests in `pr_meta_test.go` cover required `--type` and non-zero exit without fab context.

### Edge Cases & Error Handling

- [x] A-014 R7: Non-resolvable change / absent `.status.yaml` → non-zero exit with empty stdout.
- [x] A-015 R7: `gh` failure degrades to plain-text Pipeline labels; missing/failed merge-base drops only the Impact line; both leave the rest of the block intact.
- [x] A-016 R6: A `+0/−0` total omits the Impact line entirely (no `+0/−0` rendered).

### Code Quality

- [x] A-017 Pattern consistency: new Go follows the cobra/command and internal-package patterns of `impact.go`/`score.go` (naming, error handling, package layout).
- [x] A-018 No unnecessary duplication: reuses `internal/impact`, `internal/statusfile`, `internal/resolve`, `internal/config`; merge-base/impact math are not reimplemented.
- [x] A-019 Documentation accuracy: `_cli-fab.md`, `git-pr.md`, and `SPEC-git-pr.md` accurately describe the shipped command (signature, flags, output, exit codes).
- [x] A-020 Cross-references: the `fab impact` "Consumers" note and any cross-links remain consistent after `git-pr.md` stops calling `fab impact` directly.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Render logic extracted into `internal/prmeta`; `cmd/fab/pr_meta.go` is a thin cobra wrapper | Mirrors `internal/impact`/`internal/score` precedent; enables pure-render unit tests (intake assumption #11) | S:90 R:85 A:90 D:85 |
| 2 | Confident | `prmeta` parses `project.linear_workspace` locally rather than widening the shared `internal/config.Config` struct | Shared Config struct lacks the field and backs impact/hooks; a local parse keeps the shared surface minimal for a single consumer | S:70 R:80 A:80 D:75 |
| 3 | Certain | Output is byte-for-byte the current `git-pr.md` Step 3c contract (table, Pipeline, Issues, Impact spacing/glyphs) | Forced by the change goal (mechanize to reduce drift); any content change contradicts the requirement (intake assumption #7) | S:95 R:88 A:95 D:96 |
| 4 | Confident | Keep the legacy `tasks.md` Plan fallback for one release | Matches current skill behavior; cheap, trivially reversible (intake assumption #10) | S:72 R:82 A:78 D:75 |
| 5 | Confident | `gh`/merge-base failures degrade gracefully (plain labels / drop Impact), never hard-fail | Matches today's `{has_fab}=false` path and Constitution I/III; `/git-pr` must stay autonomous (intake assumption #9) | S:80 R:75 A:88 D:80 |
| 6 | Confident | `{COMMA_LIST_CODE}` is per-element backtick-wrapped + comma-joined in BOTH Impact forms, with no extra outer backticks | The single-line template (git-pr.md:273) wraps the already-backtick-wrapped token in additional backticks, which would emit broken markdown (`` ``fab/`...` ``). The intake's stated expected output is `` (excluding `fab/`, `docs/`, `sites/_playground/`) `` — per-element wrap only. Mechanizing to the intended (drift-free) output resolves the source-prose inconsistency | S:78 R:82 A:85 D:82 |

6 assumptions (2 certain, 4 confident, 0 tentative).
</content>
</invoke>
