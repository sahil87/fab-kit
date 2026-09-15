# Plan: Make `fab status refresh` change argument optional + patch the dispatch-epilogue canon

**Change**: 260811-izj3-refresh-optional-change-arg
**Intake**: `intake.md`

## Requirements

### CLI: Optional Change Argument

#### R1: `fab status refresh` accepts an optional change argument with active-change fallback
`statusRefreshCmd` (`src/go/fab/cmd/fab/status.go`) SHALL declare `Use: "refresh [<change>]"` with `Args: cobra.MaximumNArgs(1)` and SHALL pass `optArg(args, 0)` to `withStatusLock`, so an omitted argument resolves the active change via the existing `resolve.ToAbsStatus(fabRoot, "")` → `resolveFromCurrent` symlink path — the identical resolution bare `fab preflight` uses. `internal/refresh` SHALL NOT be modified; this change is argument-parsing only.

- **GIVEN** a repo with an active change (`.fab-status.yaml` symlink present)
- **WHEN** the user runs bare `fab status refresh`
- **THEN** the command refreshes the active change exactly as `fab status refresh <change>` would
- **AND** exits zero

#### R2: Bare invocation without an active change fails with the existing resolution error
When neither an argument nor an active change exists, `fab status refresh` SHALL exit non-zero with `resolveFromCurrent`'s existing error message — no new error text, no silent no-op, consistent with every other bare-invocable command.

- **GIVEN** a repo with no active change (no `.fab-status.yaml` symlink)
- **WHEN** the user runs bare `fab status refresh`
- **THEN** the command exits non-zero with the existing no-active-change resolution error on stderr

#### R3: The explicit-argument form is unchanged
`fab status refresh <change>` SHALL behave byte-for-byte identically to before the change; every existing invocation is unaffected (purely additive).

- **GIVEN** any existing invocation `fab status refresh <change>`
- **WHEN** it runs after the change
- **THEN** resolution, refresh behavior, and exit codes are identical to the pre-change behavior

### Documentation: Epilogue Canon Spelling

#### R4: Worker-contract epilogue sites spell the change token
Every site in `src/kit/skills/_preamble.md` (5 sites) and `docs/specs/harness-adapters.md` (3 sites) that states the terminal refresh epilogue as a command a worker copies SHALL spell it `fab status refresh <change>` (the worker substitutes the 4-char change ID it was dispatched with), matching the form `docs/specs/hooks.md` already uses.

- **GIVEN** the 5 `_preamble.md` sites (~331, ~349, ~419, ~483, ~487) and 3 `harness-adapters.md` sites (~95, ~361, ~522)
- **WHEN** the canon patch lands
- **THEN** each spells `fab status refresh <change>` and no bare-literal epilogue command remains in either file

#### R5: Repo-wide sweep follows the owner-or-pointer rule
The sweep over `src/kit/skills/`, `docs/specs/`, and `docs/memory/` SHALL add the change token at every site that spells a runnable command a worker or prompt-composer would copy, and SHALL leave pure noun-phrase mechanism names ("the refresh epilogue", "terminal-refresh obligation", "recomputed by `fab status refresh`") unchanged — those are pointers, not restatements. `docs/specs/hooks.md` and the pointer-only sites (`src/kit/skills/_pipeline.md`, `src/kit/skills/fab-continue.md`, mechanism prose in `_intake.md`/`_generation.md`) SHALL NOT be edited.

- **GIVEN** a repo-wide grep for `fab status refresh`
- **WHEN** each match is classified as runnable-command vs. noun-phrase pointer
- **THEN** every runnable-command site carries `<change>` and every pointer site is untouched

#### R6: Command-surface documentation reflects the optional argument
`src/kit/skills/_cli-fab.md`'s `fab status` table `refresh` row SHALL show `refresh [<change>]` and note the omitted-arg active-change fallback (same resolution as bare `fab preflight`). Memory files that document the command surface with the literal `fab status refresh <change>` (`docs/memory/pipeline/hooks-may-enhance-never-own.md` ~21, `docs/memory/distribution/kit-architecture.md` ~290) SHALL note the now-optional argument; the five Affected Memory files' epilogue/worker-contract literals SHALL gain the change token per R5.

- **GIVEN** `_cli-fab.md`'s refresh row and the two command-surface memory sites
- **WHEN** the documentation updates land
- **THEN** the signature reads `refresh [<change>]` with the fallback noted, and memory's command-surface statements match the shipped CLI

### Testing

#### R7: The optional-argument behavior is covered by `cmd/fab` tests
`src/go/fab/cmd/fab/refresh_selfheal_test.go` (or a sibling `cmd/fab` test file) SHALL gain tests proving: (a) bare `fab status refresh` with an active `.fab-status.yaml` symlink refreshes that change; (b) bare invocation with no active change exits non-zero with the resolution error. The existing with-arg test (`TestRefreshCmd_HealsStaleStatus`) SHALL remain passing and unmodified in behavior (verify coverage, don't duplicate).

- **GIVEN** the self-heal test fixture in `refresh_selfheal_test.go`
- **WHEN** `go test ./cmd/fab/...` runs from `src/go/fab`
- **THEN** all three behaviors (bare+active, bare+no-active, with-arg) pass

### Non-Goals

- No changes to `internal/refresh` or `internal/resolve` — argument parsing only.
- No `docs/specs/skills/SPEC-*.md` mirror updates — the skill edits are prose-only (constitution v1.5.0 narrowed mirror rule).
- No edits to `docs/specs/hooks.md` — it already spells the with-arg form correctly.
- No migration — no user-data restructuring; the change is purely additive.

## Tasks

### Phase 1: Setup

- [x] T001 Read `src/go/fab/cmd/fab/status.go` (`statusRefreshCmd`, `withStatusLock`, `optArg`) and `src/go/fab/internal/resolve/resolve.go` (`ToFolder`/`ToAbsStatus`/`resolveFromCurrent`) to confirm the empty-override fallback path; extract patterns (cobra command style, `optArg` usage, error propagation) <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 Change `statusRefreshCmd` in `src/go/fab/cmd/fab/status.go` to `Use: "refresh [<change>]"`, `Args: cobra.MaximumNArgs(1)`, `withStatusLock(optArg(args, 0), ...)`; update the cobra `Short`/help text only if it states the argument as required <!-- R1 -->
- [x] T003 Extend `src/go/fab/cmd/fab/refresh_selfheal_test.go`: (a) bare refresh with an active `.fab-status.yaml` symlink heals that change; (b) bare refresh with no active change exits non-zero with the resolution error; verify existing `TestRefreshCmd_HealsStaleStatus` still covers the with-arg form <!-- R7 -->
- [x] T004 [P] Patch the 5 epilogue sites in `src/kit/skills/_preamble.md` (~331, ~349, ~419, ~483, ~487) to spell `fab status refresh <change>` <!-- R4 -->
- [x] T005 [P] Patch the 3 sites in `docs/specs/harness-adapters.md` (~95, ~361, ~522) and the Dispatch-adapter entry in `docs/specs/glossary.md` (~32) to spell `fab status refresh <change>` <!-- R4 -->
- [x] T006 [P] Update the `refresh` row in `src/kit/skills/_cli-fab.md` (~124) to `refresh [<change>]` with the active-change fallback note <!-- R6 -->

### Phase 3: Integration & Edge Cases

- [x] T007 Run the repo-wide sweep: grep `fab status refresh` across `src/kit/skills/`, `docs/specs/`, `docs/memory/`; classify each match per the owner-or-pointer rule and fix every runnable-command site not already covered by T004–T006 (memory sites: `docs/memory/_shared/context-loading.md` ~118/~133, `docs/memory/pipeline/execution-skills.md` ~101/~108/~470, `docs/memory/runtime/dispatch.md` ~599, `docs/memory/pipeline/hooks-may-enhance-never-own.md` ~15); confirm pointer-only sites stay untouched <!-- R5 -->
- [x] T008 Note the now-optional argument at the two command-surface memory sites: `docs/memory/pipeline/hooks-may-enhance-never-own.md` (~21) and `docs/memory/distribution/kit-architecture.md` (~290) <!-- R6 -->
- [x] T009 Run `go test ./cmd/fab/...` from `src/go/fab` and fix any failures; also run `gofmt`/`go vet` on the touched package if the project tooling does so <!-- R1 -->
- [x] T010 Run the Toolkit Standards check for the CLI-surface change: `shll standards`, identify entries governing CLI surface/help output, and check the new `refresh [<change>]` signature against them (fall back to sahil87/shll `docs/site/standards/` if `shll` is unavailable) <!-- R1 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Bare `fab status refresh` with an active change refreshes that change and exits zero
- [x] A-002 R2: Bare `fab status refresh` with no active change exits non-zero with the existing resolution error
- [x] A-003 R3: `fab status refresh <change>` behaves identically to before (existing tests unchanged and green)
- [x] A-004 R4: All 5 `_preamble.md` epilogue sites and all 3 `harness-adapters.md` sites spell `fab status refresh <change>`
- [x] A-005 R6: `_cli-fab.md` refresh row shows `refresh [<change>]` with the active-change fallback noted; the two command-surface memory sites note the optional argument
- [x] A-006 R7: `go test ./cmd/fab/...` passes from `src/go/fab`, covering bare+active, bare+no-active, and with-arg refresh

### Behavioral Correctness

- [x] A-007 R1: The bare form resolves via the same `resolveFromCurrent` symlink path as bare `fab preflight` (no new resolution code)

### Scenario Coverage

- [x] A-008 R5: A final grep for bare `fab status refresh` shows no remaining runnable-command sites without the change token; pointer/noun-phrase sites are untouched (`hooks.md`, `_pipeline.md`, `fab-continue.md`, `_intake.md`, `_generation.md`, `SPEC-_preamble.md`)

### Edge Cases & Error Handling

- [x] A-009 R2: The no-active-change error text is the pre-existing `resolveFromCurrent` message — no new error wording introduced

### Code Quality

- [x] A-010 Pattern consistency: the CLI edit follows the surrounding cobra command style and reuses the existing `optArg` helper (no new helpers, no duplicated resolution logic)
- [x] A-011 No unnecessary duplication: the fallback reuses `withStatusLock`/`resolve.ToAbsStatus` rather than new resolution code
- [x] A-012: No file under `.claude/skills/` is edited — only canonical sources in `src/kit/skills/`
- [x] A-013: No `docs/specs/skills/SPEC-*.md` mirror is edited (prose-only skill edits, constitution v1.5.0)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Epilogue placeholder spelled `fab status refresh <change>`; a short substitution note ("worker substitutes its change ID") added only where the prose doesn't already make it obvious | Intake assumption 4 leaves exact wording to the agent; matching `hooks.md`'s existing form keeps the canon uniform | S:65 R:85 A:80 D:75 |
| 2 | Confident | The two `docs/memory/pipeline/execution-skills.md` carve-out/continuation restatements (~101, ~108) get the token alongside ~470 — all three state the command a worker copies | Sweep rule (runnable-command sites get the token); intake listed ~470 explicitly, the sibling restatements are the same class | S:70 R:85 A:80 D:70 |
| 3 | Confident | Mechanism-prose matches (`recomputed by fab status refresh`, migration history, `docs/specs/index.md`/`templates.md`/`change-types.md`/`architecture.md`) stay bare — they name the mechanism, not the epilogue command | Owner-or-pointer sweep rule from intake assumption 5 | S:70 R:85 A:80 D:70 |
| 4 | Certain | `statusRefreshCmd`'s `Short` text needs no edit — it describes the recompute, never states the argument as required | Verified by reading `status.go:405-424` at plan time | S:90 R:90 A:90 D:85 |

4 assumptions (2 certain, 2 confident, 0 tentative).
