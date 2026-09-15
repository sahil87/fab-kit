# Intake: Code-Reorg Structure Sweep

**Change**: 260824-ffny-code-reorg-structure-sweep
**Created**: 2026-08-24

## Origin

Synthesized from a completed `/code-reorg src` analysis run on 2026-08-24. The skill produced three evidence-backed structural proposals; the user approved all three verbatim ("go for all") and asked for them as one combined change. Dispatched promptless via `/fab-proceed`-style create-intake (no questions asked; would-be questions recorded as deferred Unresolved rows — none arose, the approved report fully specifies the work).

> Ship the three approved `/code-reorg src` proposals as one structure-only refactor: (1) normalize 4 concatenated multi-word filenames in `src/go/fab/cmd/fab/`, (2) move the `src/benchmark/` historical decision record to `docs/findings/`, (3) rename Go package `internal/hooklib` → `internal/artifact`. Zero behavior change intended; present-truth doc sweeps included.

## Why

1. **Pain point**: three structural inconsistencies in `src/`:
   - 4 of 39 multi-word files in `src/go/fab/cmd/fab/` use concatenated names (`panemap.go`, `fabhelp.go`, `helpdump.go`, `shellinit.go`) while the other 35 use underscore separation (`memory_index.go`, `resolve_agent.go`, …); `panemap.go` additionally breaks the `pane_*.go` prefix used by all 9 of its `fab pane` sibling subcommand files.
   - `src/benchmark/` sits inside the source tree but holds only a self-described "Historical decision record" (README.md + RESULTS.md — the benchmark code itself was deleted in change 260612-tb6f/F47). `fab/project/context.md` documents `src/` as canonical sources (`src/kit/` + `src/go/`) only; `docs/findings/` already holds exactly this genre of historical report.
   - The package name `internal/hooklib` describes deleted machinery: the `fab hook` family was removed in kit 2.14.0 (`docs/specs/hooks.md`) and change 260819-hk7p (PR #602, merged) deleted the orphaned payload parsers. What remains is pure artifact-content analysis, and `docs/memory/distribution/kit-architecture.md` even records "the package keeps its legacy name for now" — an acknowledged debt.
2. **Consequence of not fixing**: naming conventions erode by example (new files copy the concatenated style), the source tree misadvertises `src/benchmark/` as live code, and every reader of `internal/hooklib` must re-derive that "hooklib" no longer means hooks.
3. **Why this approach**: pure `git mv` renames + a package rename with exactly two importers is the minimal, evidence-backed normalization; the user chose the MOVE variant for `src/benchmark/` (a 2026-06 binary-review finding had suggested deletion as an alternative — explicitly not chosen, the decision record stays).

## What Changes

### Part 1 — Normalize 4 concatenated filenames in `src/go/fab/cmd/fab/`

8 files total, each rename via `git mv`, all `package main`, zero import changes:

| Old | New | Implements |
|-----|-----|------------|
| `src/go/fab/cmd/fab/panemap.go` | `pane_map.go` | `fab pane map` (joins the `pane_*.go` prefix of its 9 siblings) |
| `src/go/fab/cmd/fab/panemap_test.go` | `pane_map_test.go` | |
| `src/go/fab/cmd/fab/fabhelp.go` | `fab_help.go` | `fab fab-help` |
| `src/go/fab/cmd/fab/fabhelp_test.go` | `fab_help_test.go` | |
| `src/go/fab/cmd/fab/helpdump.go` | `help_dump.go` | `fab help-dump` |
| `src/go/fab/cmd/fab/helpdump_test.go` | `help_dump_test.go` | |
| `src/go/fab/cmd/fab/shellinit.go` | `shell_init.go` | `fab shell-init` |
| `src/go/fab/cmd/fab/shellinit_test.go` | `shell_init_test.go` | |

Verified: none of the new names end in a Go implicit build-constraint suffix (no GOOS/GOARCH token; `_test.go` twins stay `_test.go`).

**Present-truth doc sweep for old filenames** (verified by grep on 2026-08-24):
- `docs/specs/skills.md:214` — `fabhelp.go` (the `skillToGroupMap` instruction)
- `docs/memory/pipeline/code-reorg.md:17` — `fabhelp.go`
- `docs/memory/distribution/kit-architecture.md` — multiple lines, NOT just one: line 309 (`shellinit.go`), line 311 (`helpdump.go`), line 313 (`panemap.go`, several occurrences), line 317 (`panemap`, `fabhelp` as test names), line 357 (`panemap.go`, twice)

Historical `log.md`/`log.seed.md` entries and `docs/specs/findings/*` are dated records — leave untouched. No CLI command signatures change, so `src/kit/skills/_cli-fab.md` needs no update (verified: zero old-filename mentions in `src/kit/`).

### Part 2 — Move `src/benchmark/` decision record to `docs/findings/`

- `git mv src/benchmark/README.md docs/findings/statusman-benchmark/README.md`
- `git mv src/benchmark/RESULTS.md docs/findings/statusman-benchmark/RESULTS.md`
- `src/benchmark/` ends up empty and gone (git tracks files, not dirs)

The README→RESULTS.md relative link survives (both files move together). User approved the MOVE variant; deletion (suggested by the 2026-06 binary-review finding as an alternative) was not chosen.

**Present-truth doc sweep**: `docs/memory/distribution/kit-architecture.md:516` says `src/benchmark/` retains README + RESULTS "as the historical decision record — this section is the surviving summary" — repoint to `docs/findings/statusman-benchmark/`. Add a row to `docs/findings/index.md` for the relocated record (the index table enumerates findings with Status + Summary; the new row should mark it as a closed/historical decision record, pointing at `statusman-benchmark/README.md`).

### Part 3 — Rename Go package `internal/hooklib` → `internal/artifact`

Current contents: `src/go/fab/internal/hooklib/artifact.go` + `artifact_test.go` only — artifact-content analysis (`InferChangeType`, `HasSectionHeading`, `CountSectionItemsBounded`, `CountCompletedSectionItemsBounded`, plus the `SectionTasks`/`SectionAcceptance` constants).

- `git mv src/go/fab/internal/hooklib src/go/fab/internal/artifact`
- Change the package clause `package hooklib` → `package artifact` in both files
- Update the two importers (verified as the complete set on 2026-08-24 — `grep -rln hooklib src/go/fab --include='*.go'`):
  - `src/go/fab/internal/refresh/refresh.go` — import path + `hooklib.` selectors (lines 19, 80, 143–144, 156, 162–163)
  - `src/go/fab/internal/status/acceptance.go` — import path + `hooklib.` selectors (lines 7, 28, 31–32) + the line-14 comment "existing hooklib counters"
- Also sweep the code comment at `src/go/fab/internal/prmeta/prmeta.go:538` ("canonical bound in hooklib.HasSectionHeading / scanSectionItems") — comment-only mention, no import
- Import path becomes `github.com/sahil87/fab-kit/src/go/fab/internal/artifact`

**Present-truth doc sweep for `hooklib`** (verified by grep; one mention each unless noted):
- `docs/memory/pipeline/schemas.md`
- `docs/specs/change-types.md`
- `docs/memory/pipeline/hooks-may-enhance-never-own.md`
- `docs/memory/distribution/kit-architecture.md` (multiple mentions in the architecture and testing paragraphs, including the now-superseded claim "the package keeps its legacy name for now (ioku)" — rewrite to present truth, don't merely token-swap)
- `docs/memory/runtime/runtime-agents.md`

Historical mentions stay: `docs/memory/*/log.md`, `log.seed.md`, `docs/specs/findings/*`.

## Affected Memory

- `pipeline/code-reorg`: (modify) `fabhelp.go` filename reference at line 17 → `fab_help.go`
- `pipeline/schemas`: (modify) one `hooklib` mention → `artifact`
- `pipeline/hooks-may-enhance-never-own`: (modify) one `hooklib` mention → `artifact`
- `distribution/kit-architecture`: (modify) old filenames (lines 309/311/313/317/357), the `src/benchmark/` reference (line 516), and the `hooklib` package prose including the "keeps its legacy name for now" claim
- `runtime/runtime-agents`: (modify) one `hooklib` mention → `artifact`

Memory indexes will need regeneration (`fab memory-index`) after the sweeps — index descriptions may embed swept phrases, and `fab memory-index --check` flags drift at review-pr otherwise.

## Impact

- **Code**: `src/go/fab/cmd/fab/` (8 file renames, content unchanged), `src/go/fab/internal/hooklib/` → `internal/artifact/` (2 files moved, package clause changed), `src/go/fab/internal/refresh/refresh.go` + `src/go/fab/internal/status/acceptance.go` (import path + selector updates), `src/go/fab/internal/prmeta/prmeta.go` (one comment). Zero behavior change; zero exported-API change outside the package path.
- **Non-code moves**: `src/benchmark/{README,RESULTS}.md` → `docs/findings/statusman-benchmark/`.
- **Docs**: `docs/specs/skills.md`, `docs/specs/change-types.md`, 5 memory files (see Affected Memory), `docs/findings/index.md` (new row).
- **Tests**: run `go test ./...` scoped to `src/go/fab` (at minimum `cmd/fab`, `internal/artifact`, `internal/refresh`, `internal/status`) plus a `gofmt -l` check — CI has tripped on gofmt before. Existing tests move with their files; no test logic changes.
- **In-flight exposure**: verified clean — PRs #602/#612/#614 (which touched these files) are all MERGED; no open branch or active fab change touches the affected files.
- **No migration needed**: nothing here restructures user data (no config, `.status.yaml`, or archive-layout change); the renamed package is internal to the binary.
- **No `_cli-fab.md` update**: no CLI command signature changes (constitution CLI constraint checked — signatures untouched).

## Open Questions

- None — the approved `/code-reorg` report fully specifies the work; no would-be-asked decisions arose under promptless-defer.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship all three proposals as one combined `refactor` change | Discussed — user approved all three as presented and asked for one combined change ("go for all") | S:95 R:80 A:95 D:95 |
| 2 | Certain | Part 1 rename set: the 4 file pairs above, underscore-separated, via `git mv`, no import/package changes | 35-of-39 directory convention + `pane_*.go` sibling prefix; all `package main`; build-constraint suffixes checked | S:90 R:95 A:95 D:90 |
| 3 | Confident | Part 2 destination is `docs/findings/statusman-benchmark/` (MOVE, not delete) | Discussed — user approved the MOVE variant; deletion alternative from the 2026-06 binary-review finding explicitly not chosen | S:70 R:90 A:75 D:55 |
| 4 | Confident | Part 3 new package name is `internal/artifact` (matches its sole file `artifact.go` and its artifact-analysis content) | Package-scoped rename with exactly 2 importers verified by grep; elevated blast acknowledged but fully enumerated | S:80 R:80 A:85 D:70 |
| 5 | Certain | Doc-sweep scope is present-truth only — historical `log.md`/`log.seed.md`/`docs/specs/findings/*` stay untouched | FKF present-truth rule; dated records are deliberately frozen | S:85 R:90 A:90 D:85 |
| 6 | Confident | Add a `docs/findings/index.md` row for the relocated benchmark record, marked as a closed/historical decision record | Origin said "may need a row"; the index enumerates findings, and an unlisted folder would be invisible — exact status label decided at apply | S:55 R:95 A:70 D:60 |
| 7 | Certain | Sweep breadth = all grep-found present-truth occurrences, not only the lines the origin enumerated (adds kit-architecture 309/311/313/317, `prmeta.go:538` comment, `acceptance.go:14` comment) | code-quality.md Sibling Sweeps rule: grep the old claim repo-wide up front; verified by grep on 2026-08-24 | S:75 R:90 A:90 D:80 |
| 8 | Certain | Test scope: `go test ./...` in `src/go/fab` + `gofmt` check; no new tests needed (structure-only, existing tests move with files) | Stated in the approved constraints; code-quality.md scoped-test rule; CI gofmt history | S:90 R:95 A:95 D:95 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
