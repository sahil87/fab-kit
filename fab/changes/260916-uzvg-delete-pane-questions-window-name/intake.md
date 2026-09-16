# Intake: Delete the dead `fab pane questions` and `fab pane window-name` verbs

**Change**: 260916-uzvg-delete-pane-questions-window-name
**Created**: 2026-09-16

## Origin

Backlog item **[uzvg]** in `fab/backlog.md` (line 48), quoted verbatim:

> - [ ] [uzvg] 2026-09-11: Delete the fab pane questions and fab pane window-name Go verbs (src/go/fab/cmd/fab/pane_questions.go, pane_window_name.go + tests) and their _cli-fab-pane.md sections — the operator skill stopped consuming them in 260911-4a8m (detection rides rk mux capture --classify, window marks ride rk tab mark/note); no other Go consumer exists (dispatch.go references window-name in a comment only). Retained in 4a8m to bound scope; sweep docs/memory/runtime/pane-commands.md and agent-primitives.md when removing.

**Interaction mode**: promptless dispatch from `/fab-proceed` (create-new path, `{questioning-mode} = promptless-defer`). The description was synthesized from the live conversation of 2026-09-16, in which the user gave an explicit go-ahead to run this as a tracked fab change. No questions were asked; every would-be question is recorded in `## Assumptions`.

**Key decisions carried across the boundary** (from the conversation, verified against the tree in this worktree on 2026-09-16):

- Delete, do not replace: no substitute verbs, no run-kit changes, no change to the pane-family exit-code convention (`2` = pane missing / `3` = other tmux failure) used by the surviving verbs.
- The "rk-absent fallback" rationale that once justified keeping `fab pane questions` no longer applies: since 4a8m the operator reads `rk mux capture --classify` directly and is spawned by `rk operator`, so run-kit is a hard dependency on that path.
- Scope is deletion + the owning CLI reference partial + the docs/memory sweep. Historical files (`docs/memory/**/log*.md`, `docs/specs/findings/*`) stay as written.
- Expected to be a small, low-risk change suited to the LIGHT lane.

## Why

**The pain point.** Two `fab pane` subcommands — `questions` (a question-detection sweep over candidate panes) and `window-name` (`ensure-prefix` / `replace-prefix` guarded window renames) — exist only because the operator skill once consumed them. Change `260911-4a8m-operator-generic-tracked-items` (merged as #663) moved both jobs onto run-kit: pending-prompt classification rides `rk mux capture <pane> --lines 40 --classify --json` and window marks ride `rk tab mark @<window_id> …` / `rk tab note @<window_id> …` (see `src/kit/skills/fab-operator.md` tick step 3/4 and § Window marks). 4a8m deliberately left the two verbs in place to bound its own scope and recorded their deletion as a declared follow-up (`docs/memory/runtime/pane-commands.md` § Consumers and the *Updated by* trailer on the "Capture/Process/Kill Demoted" design decision).

**Verified consumer census (2026-09-16, this worktree):**

| Surface | Result |
|---------|--------|
| Kit skills (`src/kit/`) | Zero invocations. The only mentions are the two reference sections in `src/kit/skills/_cli-fab-pane.md` (`### window-name`, `### questions`) plus that file's frontmatter `description`, the `fab pane <…>` usage line and the pane-family exit-code sentence. `_cli-fab-operator.md:30` mentions a "window-name match" — that is the `fab operator` singleton *matcher*, unrelated. |
| Go code (`src/go/fab`) | Zero code consumers outside the four files themselves. `paneWindowNameCmd()` / `paneQuestionsCmd()` are referenced only by the `cmd.AddCommand(...)` list in `src/go/fab/cmd/fab/pane.go`. `tmuxExitCode` (defined in `pane_window_name.go`) is referenced elsewhere only in a comment (`pane.go:11`). `internal/pane.ReadWindowName` has exactly one caller — `pane_window_name.go` — and no test in `internal/pane/pane_test.go`; it becomes dead with the verb. All other Go mentions are comments (listed under What Changes). |
| run-kit | No consumers. Its promptscan package carries comments saying it mirrors the fab classifier; two closed run-kit backlog rows (hzih, a7g2) reference the verb historically. No run-kit change is needed. |
| `docs/specs/` | No spec lists either verb as current surface (`operator.md`, `harness-adapters.md`, `architecture.md`, `glossary.md` checked). Only `docs/specs/findings/*` (dated review reports) mention them. |
| Present-truth docs | `docs/memory/runtime/pane-commands.md` (19 mentions), `docs/memory/runtime/index.md` (generated row), `docs/memory/distribution/kit-architecture.md:308`, `docs/site/skill.md:53`. `docs/memory/runtime/agent-primitives.md` — named by the backlog row — has **zero** mentions of either verb (its § Peek already describes classification as `rk mux capture --classify`). |

**Consequence of not doing it.** ~1,100 lines of Go (incl. tests) plus two reference sections keep describing a skill-facing verb no skill invokes, and the operator documentation carries a stale pointer (`dispatch.go:199` still says the operator marks windows through "its own idempotent `fab pane window-name ensure-prefix` primitive"). Each future `fab pane` change pays the maintenance and review cost of two verbs with no caller, and the toolkit's version-locked `fab skill` bundle continues to advertise `window-name`.

**Why plain deletion over alternatives.** Hiding the verbs in cobra was already rejected once (pane-commands.md DD "Capture/Process/Kill Demoted") as a CLI surface change with tests + standards audit for zero gain; that reasoning applied while a consumer might still appear. Pushing `questions` to `rk mux` has already happened in substance (`--classify`). Keeping `window-name` as a "dispatch-facing primitive" has no dispatch caller either — `internal/dispatch.WindowName` composes a name string and never renames a window. Nothing remains to preserve.

## What Changes

### 1. Go — delete the two verbs and their dead helper

**Delete these four files** (about 1,100 lines total):

```
src/go/fab/cmd/fab/pane_questions.go        (315 lines)
src/go/fab/cmd/fab/pane_questions_test.go   (341 lines)
src/go/fab/cmd/fab/pane_window_name.go      (173 lines)
src/go/fab/cmd/fab/pane_window_name_test.go (269 lines)
```

Everything they define is file-local except the two constructors registered in `pane.go`: `paneWindowNameCmd`, `paneWindowNameEnsurePrefixCmd`, `paneWindowNameReplacePrefixCmd`, `runEnsurePrefix`, `runReplacePrefix`, `renameWindow`, `renameArgs`, `tmuxExitCode`, `printTmuxErr`, `windowNameResult`, `emitResult`, `paneQuestionsCmd`, `questionsCaptureLines`, `paneCaptureFn`, `questionsRowsFn`, `questionMatch`, `questionSkip`, `questionsResult`, `runPaneQuestions`, `printQuestionsHuman`, `isBlankCapture`, `turnBoundaryPattern`, `hasTurnBoundary`, `nonEmptyTrailingSplit`, the `indicator*` constants, the `*Pattern` regex vars, `isQuestionMarkLine`, `matchOtherClasses`, `scanIndicators`. `collectPaneRows` (used by `questionsRowsFn`) is defined in `pane_map.go` and keeps its `map` callers — it stays.

**Unregister** in `src/go/fab/cmd/fab/pane.go` — remove `paneWindowNameCmd(),` and `paneQuestionsCmd(),` from the `cmd.AddCommand(...)` list, and update the `Long` help string:

```go
// before
Long:  "Tmux pane operations: map, capture, process, window-name, open, ready, deliver, kill, questions",
// after
Long:  "Tmux pane operations: map, capture, process, open, ready, deliver, kill",
```

**Delete the dead helper** `ReadWindowName(paneID, server string) (string, []byte, error)` in `src/go/fab/internal/pane/pane.go` (lines ~391–400) — its only caller is the deleted verb and it has no test. Reword the `RunCmd` doc comment at `pane.go:71` ("Generalizes the ReadWindowName capture pattern…") to describe the pattern without naming the deleted function (e.g. "the single subprocess-capture pattern for any child command (tmux, git, wt)…").

**Comment sweep — reword only where the deleted verb is named as the owner of the convention; the convention itself (exit `2` = pane missing, `3` = any other tmux failure) is unchanged:**

| File:line | Current text | Action |
|-----------|--------------|--------|
| `src/go/fab/cmd/fab/pane.go:11` | "exit-code scheme shared with window-name's tmuxExitCode: 2 = pane missing, 3 = …" | Reword: "the pane-family exit-code scheme: 2 = pane missing, 3 = any other tmux failure" (drop the `tmuxExitCode` reference — the symbol is gone). |
| `src/go/fab/internal/pane/pane.go:109` | "Shared by ValidatePane and the window-name verbs' exit-code mapping." | Reword: "Shared by ValidatePane and the pane-family exit-code mapping." |
| `src/go/fab/internal/dispatch/dispatch.go:199` | "An operator that genuinely enrolls the window still adds the marker through its own idempotent `fab pane window-name ensure-prefix` primitive." | Rewrite: "An operator that genuinely enrolls the window marks it itself via `rk tab mark` / `rk tab note` (the mark/note contracts are run-kit's)." Keep the surrounding rationale about `»`/`›` not being pre-applied by dispatch. |
| `src/go/fab/cmd/fab/pane_capture.go:52`, `pane_exitcode_test.go:15` | "…with window-name: 2 = pane missing, 3 = other tmux failure" | Reword to name the convention ("the pane-family scheme") rather than `window-name` as its owner. |
| `src/go/fab/cmd/fab/pane_ready.go:62`, `pane_ready_test.go:86` | "(the window-name precedent)" for the always-one-JSON-object rule | Reword to "(the pane-family `--json` precedent — one object always)" or equivalent; do not leave a pointer to a verb that no longer exists. |
| `src/go/fab/internal/pane/pane_test.go:276,301` | "a window-name target resolves…" / "window-name target rejected (ID-exactness)" | **Leave** — these describe tmux *target grammar* (a window name used as a `-t` target), not the deleted verb. |
| `src/go/fab/cmd/fab/operator.go:115`, `operator_test.go:287` | "exact, server-wide window-name match(er)" | **Leave** — the `fab operator` singleton matcher (`findWindowExact`), unrelated. |

**Verification (Go):** from `src/go/fab`: `gofmt -l ./...` (empty), `go build ./...`, `go vet ./...`, `go test ./cmd/fab/... ./internal/pane/... ./internal/dispatch/...` first, then `go test ./...`. `pane_exitcode_test.go` must stay green unmodified in behaviour (its comment may be reworded). A worker-written Go edit is not automatically formatted — run `gofmt` before finishing (CI failed once on this in l0bf).

### 2. CLI reference partial — `src/kit/skills/_cli-fab-pane.md` (Constitution: owns `fab pane`)

- Frontmatter `description`: change `map/capture/process/window-name/open/ready/deliver/kill/questions` → `map/capture/process/open/ready/deliver/kill`.
- Usage line (~line 22): `fab pane <map|capture|process|window-name|open|ready|deliver|kill|questions> [flags...]` → `fab pane <map|capture|process|open|ready|deliver|kill> [flags...]`.
- Pane-family exit-codes sentence (~line 26): drop `window-name` from the parenthetical family list; the clause "`map` and `questions` alone use plain `ERROR:`-formatted exit 1" becomes "`map` alone uses plain `ERROR:`-formatted exit 1 …".
- **Delete** the whole `### window-name — fab pane window-name <ensure-prefix|replace-prefix> …` section (~lines 75–86: intro sentence, verb table, exit-codes paragraph, output paragraph).
- **Delete** the whole `### questions — fab pane questions [--all-sessions] [--panes <id>...] …` section (~lines 104–124: intro, flag table, discovery paragraph, JSON-field table, output/exit-code paragraph) up to the `---` that precedes `## fab dispatch`.
- After editing, `grep -nE "questions|window-name|ensure-prefix|replace-prefix" src/kit/skills/_cli-fab-pane.md` must return nothing (the § "Dispatch-internal verbs" paragraph and the § agent-state paragraph are checked by that grep too). The deployed copies under `.agents/skills/` / `.claude/skills/` are regenerated by `fab sync`, never edited.

### 3. Toolkit skill bundle — `docs/site/skill.md` + embedded copy

`docs/site/skill.md:53` reads `fab pane {map,capture,send,process,window-name}` (the `send` there is already stale — retired by 260815-4i0n). Update to the surviving set:

```
- **Panes / operator** — `fab pane {map,capture,process,open,ready,deliver,kill}` inspects and
  drives tmux panes; `fab operator` launches the coordination tab.
```

Then run `scripts/sync-skill.sh` so the `//go:embed skill.md` copy in `src/go/fab/cmd/fab/skill.md` matches — `src/go/fab/cmd/fab/skill_test.go` is a byte-identity drift guard that fails `go test` otherwise. The bundle is governed by the toolkit `skill` standard (`shll standards skill`: static-only, ≤150 lines, byte-identical to `fab skill` output) — this edit removes text and stays within it. `help-dump` output adjusts itself once the subcommands are unregistered (no hand-maintained artefact).

### 4. Memory sweep (present truth) — `docs/memory/`

**`docs/memory/runtime/pane-commands.md`** (the owning memory file; 19 mentions):

- Frontmatter `description`: verb set → `{map,capture,process,open,ready,deliver,kill}`; delete the clause "questions and window-name stay shipped with no operator consumer (detection: `rk …`)" and instead say the two were deleted in this change (detection rides `rk mux capture --classify`, marks ride `rk tab mark`/`note`).
- `## Overview`: "grouping nine tmux-pane operations. Six query, manipulate, or sweep…" → seven operations; drop the `questions` sweep and the `window-name` rewrite from the enumerations; "This doc covers the eight subcommands" → seven.
- `### Parent Command`: "nine subcommands (`map`, `capture`, `process`, `window-name`, `open`, `ready`, `deliver`, `kill`, `questions`)" → "seven subcommands (`map`, `capture`, `process`, `open`, `ready`, `deliver`, `kill`)"; help "listing the nine" → seven.
- **Delete** `### Subcommand: fab pane window-name` with its `#### Verb: ensure-prefix`, `#### Verb: replace-prefix`, `#### Exit codes (both verbs)` subsections (~lines 104–128).
- **Delete** `### Subcommand: fab pane questions` (~lines 169–172) **and** its trailing `#### Output modes` and `#### Consumers` subsections (~lines 180–186 — both describe `window-name` output/consumers and are orphaned once the verb goes). **Keep** the intervening `### Usage-Error Coexistence with the Binary-Wide Exit-2 Convention (swon)` section — it governs the surviving verbs; just drop `window-name` from its verb list ("across `capture`/`process`/`ready`/`deliver`/`kill`").
- `### --server / -L Flag`: "visible on all eight subcommands' `--help`" → seven.
- `### Shared Pane Package (internal/pane)`: `IsPaneMissing` bullet — drop "and the `window-name` verbs' exit-code mapping (`tmuxExitCode` … unified onto it)"; **remove** the `ReadWindowName(...)` bullet; leave the `ValidatePane` bullet's "window-name / target-grammar args" wording (tmux target grammar, not the verb).
- `## Design Decisions` → "Capture/Process/Kill Demoted to Dispatch-Internal…": append to the trailer `; *Updated by*: 260916-uzvg-delete-pane-questions-window-name (\`questions\` and \`window-name\` deleted — the named exception is closed)`. Hydrate should additionally add a new four-field DD entry recording the deletion (Decision / Why / Rejected / *Introduced by*), per FKF §3.3 — rationale as in § Why above.
- Any remaining `questions` / `window-name` / `ensure-prefix` mention must be reworded or removed; `grep -nE "pane questions|window-name|ensure-prefix|replace-prefix|ReadWindowName" docs/memory/runtime/pane-commands.md` should return only the DD history trailers.

**`docs/memory/runtime/agent-primitives.md`** — named by the backlog row; **verify-only**: it has zero mentions of either verb today and § Peek already routes classification through `rk mux capture --classify`. Re-run the grep at hydrate; no edit is expected.

**`docs/memory/distribution/kit-architecture.md:308`** — sibling aggregate: "parent command grouping eight pane-related subcommands (`map`, `capture`, `process`, `window-name`, `open`, `ready`, `deliver`, `kill`)" → "seven pane-related subcommands (`map`, `capture`, `process`, `open`, `ready`, `deliver`, `kill`)".

**`docs/memory/runtime/index.md`** — generated (`fab docs-index`, seeded from `pane-commands.md`'s frontmatter description). Do **not** hand-edit; regenerate after the description edit with a `fab` built from this branch (the released binary may lag — l0bf's stale-binary lesson), and confirm the row no longer names the verbs.

**Left untouched (history, not present truth):** `docs/memory/runtime/log.md`, `log.seed.md`, `docs/memory/pipeline/log.md`, `log.seed.md`, `docs/specs/findings/binary-review-2026-06-12.md`, `docs/specs/findings/skills-review-2026-06-11.md`, `docs/wiki/operator-tick-anatomy.html` (a dated design study of 2026-09-11 that already describes the rk replacement), and backlog row `[2ne8]` (a future-daemon idea that names `fab pane questions` as an input — user-owned backlog prose; the ship report should note it now needs re-reading).

### 5. Backlog

`[uzvg]` is marked done at ship by the standard pipeline behaviour — not now.

### Non-goals

- No replacement verbs, aliases, or cobra-hidden stubs.
- No run-kit changes (its promptscan comments and closed backlog rows are its own history).
- No change to the pane-family exit-code convention (`2`/`3`) or to `pane_exitcode_test.go`'s assertions.
- No edits to the operator skill's `rk`-based flows (they already do not reference these verbs).
- No migration: nothing under `fab/`, `.status.yaml`, or the archive layout changes shape.
- Not fixing `docs/site/skill.md`'s other content beyond the one verb list (the stale `send` is corrected only because it sits in the same token list).

## Affected Memory

- `runtime/pane-commands`: (modify) drop the `window-name` and `questions` subcommand sections and the orphaned `#### Output modes` / `#### Consumers` subsections; verb counts nine→seven; remove the `ReadWindowName` bullet and the `window-name` clause on `IsPaneMissing`; frontmatter description; *Updated by* trailer on the demotion DD plus a new deletion DD
- `runtime/index`: (modify) regenerated by `fab docs-index` from the new `pane-commands` description — not hand-edited
- `distribution/kit-architecture`: (modify) `fab pane` bullet — seven subcommands, no `window-name`

## Impact

**Code**: `src/go/fab/cmd/fab/` (4 files deleted, `pane.go` registration + help + comment, comment rewording in `pane_capture.go`, `pane_ready.go`, `pane_ready_test.go`, `pane_exitcode_test.go`), `src/go/fab/internal/pane/pane.go` (delete `ReadWindowName`, two comment rewordings), `src/go/fab/internal/dispatch/dispatch.go` (one comment), `src/go/fab/cmd/fab/skill.md` (synced embed). Net ≈ −1,120 lines Go.

**Kit / docs**: `src/kit/skills/_cli-fab-pane.md`, `docs/site/skill.md`, `docs/memory/runtime/pane-commands.md`, `docs/memory/runtime/index.md` (regenerated), `docs/memory/distribution/kit-architecture.md`.

**CLI surface**: `fab pane window-name` and `fab pane questions` cease to exist — a breaking removal for any external caller, of which the census found none (skills, Go, run-kit). `help-dump` reflects the removal automatically. The `skill` bundle drift test enforces the `docs/site/skill.md` ↔ embed sync.

**Tests**: two test files deleted with their subjects; `pane_exitcode_test.go`, `pane_ready_test.go`, `internal/pane/pane_test.go` keep passing with comment-only edits; `skill_test.go` passes once `scripts/sync-skill.sh` has run. Run scoped packages first, then `go test ./...` in `src/go/fab`.

**Risk**: low — pure removal with a verified-zero consumer census; the only behavioural coupling is the shared exit-code convention, which is untouched. Constitution checks: CLI change ⇒ owning partial `_cli-fab-pane.md` updated + tests updated (deleted alongside); toolkit `skill` standard for `docs/site/skill.md`; `help-dump` standard satisfied structurally; deployed content cites no fab-kit-only paths (the partial only loses text).

## Open Questions

- None. Every decision that could have been a question is graded in `## Assumptions`; no row required deferral.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete all four Go files and unregister both constructors in `pane.go`; no cobra hiding, no stubs | Backlog row + conversation name exactly these files; symbol census shows no cross-file code consumers | S:95 R:70 A:95 D:95 |
| 2 | Certain | Also delete `internal/pane.ReadWindowName` and reword the `RunCmd` comment that cites it | Verified: sole caller is `pane_window_name.go`, no test covers it — it is dead the moment the verb goes | S:60 R:90 A:95 D:85 |
| 3 | Certain | Exit-code convention (2/3) unchanged; only comments naming `window-name`/`tmuxExitCode` as its owner are reworded; tmux target-grammar and operator-matcher mentions stay | Conversation stated this explicitly; the surviving verbs implement the scheme via `paneValidationExitCode`/`IsPaneMissing` | S:90 R:95 A:85 D:75 |
| 4 | Certain | Rewrite `dispatch.go:199` to say the operator marks via `rk tab mark`/`rk tab note` | Conversation decision; matches `fab-operator.md` § Window marks | S:95 R:95 A:90 D:90 |
| 5 | Confident | Set `change_type` explicitly to `chore` (dead-code cleanup); do not rely on inference (the intake's `rename-window` prose would trip the `refactor` keyword) | change-types.md defines `chore` as "clean up dead code"; removal of a CLI verb is not a behaviour-preserving refactor | S:70 R:95 A:75 D:70 |
| 6 | Confident | Update `docs/site/skill.md`'s `fab pane {…}` list to the seven survivors (dropping already-stale `send` in passing) and re-run `scripts/sync-skill.sh` | Toolkit `skill` standard: bundle must not document a capability the binary lacks; `skill_test.go` enforces the sync | S:55 R:90 A:85 D:80 |
| 7 | Certain | Sweep `docs/memory/distribution/kit-architecture.md:308` (eight→seven subcommands) | code-quality.md sibling-sweep rule for aggregate docs; found by repo-wide grep | S:60 R:95 A:90 D:90 |
| 8 | Certain | Regenerate `docs/memory/runtime/index.md` via `fab docs-index` (branch-built binary), never hand-edit | The file is marked generated; l0bf stale-binary lesson | S:70 R:95 A:95 D:95 |
| 9 | Certain | Leave history untouched: `log*.md`, `docs/specs/findings/*`, `docs/wiki/operator-tick-anatomy.html` | Conversation decision for logs/findings; the wiki page is a dated (2026-09-11) design study that already narrates the rk replacement | S:80 R:90 A:85 D:85 |
| 10 | Confident | Leave backlog row `[2ne8]` (mentions `fab pane questions` as a future daemon input) untouched; mention it in the ship report | Backlog prose is user-owned; the row's idea survives with `rk mux capture --classify` as the input, but rewording it is the user's call | S:40 R:95 A:60 D:55 |
| 11 | Certain | `agent-primitives.md` is verify-only (no edit) despite the backlog row naming it | Grep on 2026-09-16 found zero mentions; § Peek already documents the rk path | S:75 R:95 A:95 D:90 |
| 12 | Confident | Hydrate appends an *Updated by* trailer to the existing demotion DD and adds a new four-field deletion DD in `pane-commands.md` | FKF DD entries are append-history; the existing DD's `Rejected` clause names the `questions` exception this change closes | S:50 R:90 A:80 D:65 |
| 13 | Certain | No run-kit change; run-kit mentions are comments and closed backlog rows | Conversation-verified; out of scope by the backlog row | S:95 R:90 A:90 D:95 |
| 14 | Certain | No migration file — no user data (`fab/`, `.status.yaml`, archive layout) changes shape | context.md § Migrations trigger does not apply to a binary/doc removal | S:85 R:95 A:100 D:100 |

14 assumptions (10 certain, 4 confident, 0 tentative, 0 unresolved).
