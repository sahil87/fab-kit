# Intake: `fab kit-path` prints a newline-terminated line

**Change**: 260916-8nd2-kit-path-trailing-newline
**Created**: 2026-09-16

## Origin

Promptless dispatch via `/fab-proceed` (create-new path, `{questioning-mode} = promptless-defer`)
from a conversation in which the user diagnosed the bug, approved the plan below, and invoked
`/fab-proceed`. The synthesized description handed to this intake:

> `fab kit-path` prints the resolved kit directory with NO trailing newline
> (`src/go/fab/cmd/fab/kitpath.go` line 20, `fmt.Fprint(cmd.OutOrStdout(), dir)`). Verified live:
> `fab kit-path | xxd` ends in `.../kit` with no `0a`, whereas `$(fab kit-path)/VERSION` DOES end in a
> newline. Consequence, observed on EVERY `/fab-setup migrations` run: the skill's pre-flight chains
> `fab kit-path && cat "$(fab kit-path)/VERSION" && fab log command "fab-setup" && fab migrations-status --json`;
> the first two outputs fuse into one line such as `/home/sahil/.fab-kit/versions/2.28.1/kit2.28.1`, the
> agent reads that as the kit path, tries to open `kit2.28.1/migrations/2.27.0-to-2.28.0.md`, gets
> "does not exist", runs an `ls`/`find` that fails, and only recovers after several wasted turns.
>
> Root cause is a deliberate spec, judged wrong: `src/kit/skills/_cli-fab.md` § `fab kit-path` states
> "No trailing newline or decoration." That contract buys nothing — `$(...)` strips trailing newlines
> anyway, and no code or script in the repo byte-compares raw `fab kit-path` output. Every other
> scalar-printing fab command ends in a newline.
>
> Decided changes: (1) `fmt.Fprint` → `fmt.Fprintln` in `kitpath.go`; (2) add `kitpath_test.go` in
> `src/go/fab/cmd/fab/` following the cobra-command test pattern, `t.Setenv(kitpath.KitPathEnv, …)`,
> asserting output equals the resolved dir plus exactly one `\n`; (3) replace the "No trailing newline
> or decoration." sentence in `_cli-fab.md`; (4) doc sweep for any other restatement of the
> no-trailing-newline claim (expected no-op); (5) `go test ./src/go/fab/cmd/fab/...` and `gofmt -l` on
> touched Go files. Constraints: do NOT change `fab-setup.md`'s pre-flight wording; no migration;
> not a breaking change; change type `fix`, light lane.

Verified again at intake time in this worktree (`fab 2.28.1`): `fab kit-path | xxd | tail -1` ends
`... 2f6b 6974  28.1/kit` — the last byte is `t`, no `0a`.

## Why

**Problem.** `fab kit-path` is the one `fab` command whose stdout does not end in a newline. On its own
that is invisible — the shell prompt simply lands on the same line. In a **chained** invocation it is
harmful: `fab kit-path && cat "$(fab kit-path)/VERSION"` prints
`/home/sahil/.fab-kit/versions/2.28.1/kit2.28.1` as a single line, and an agent reading that output has no
way to know where the path ends and the version begins. The `/fab-setup` skill's Pre-flight Check
(`src/kit/skills/fab-setup.md` § Pre-flight Check) tells the agent to run exactly that pair, so the fused
line is produced on every `/fab-setup migrations` run. The observed failure cascade is: agent records
`…/kit2.28.1` as the kit path → opens `…/kit2.28.1/migrations/2.27.0-to-2.28.0.md` → "does not exist" →
`ls`/`find` on the bogus path → several wasted turns before the agent re-derives the real path. The same
class of failure hits any human or agent who chains `fab kit-path` with another command or reads its
output from a terminal transcript.

**Why the current behavior exists, and why that reasoning does not hold.** The CLI reference
(`_cli-fab.md` § `fab kit-path`) documents "No trailing newline or decoration." as a deliberate contract,
presumably so `$(fab kit-path)/templates/` interpolation yields a clean path. But POSIX command
substitution strips **all** trailing newlines from the captured output, so `$(fab kit-path)` is
byte-identical with or without the newline. A repo-wide grep confirms nobody depends on the raw bytes:
every consumer is a `$(fab kit-path)/…` substitution in a skill, the user-level `fab-operator` pointer
template (`src/kit/templates/user-skill-fab-operator.md`, which tells the agent to *run* `fab kit-path`
and read the path — a reader that is **helped** by a newline), a comment in `scripts/sync-fkf.sh`, and Go
tests that assert the string `fab kit-path` appears in the rendered template. The contract therefore
protects nothing and costs real agent turns.

**Why this is the sole outlier.** The other `fmt.Fprint(cmd.OutOrStdout(), …)` sites in
`src/go/fab/cmd/fab/` (`agent.go:281`, `config.go:215/313/320/1161/1200`, `resolve_agent.go:148`,
`pane_capture.go:68/96`, `operator_clock_sync.go:104`) print YAML renders, JSON, or whole file contents
that already carry their own trailing newline. `kitpath.go:20` is the only site printing a bare scalar
without one.

**If we don't fix it.** Every `/fab-setup migrations` run keeps paying the recovery cost, and the
fused-output trap silently awaits any new skill or script that chains `fab kit-path`. A defensive
rewording of the `fab-setup.md` pre-flight would only patch one caller; fixing the binary fixes every
caller, present and future.

**Why `Fprintln` over alternatives.** `fmt.Fprintln` is the minimal change (one identifier), produces
exactly one `\n`, and matches how every other scalar-printing command in the binary behaves. Keeping the
no-newline contract and rewording `fab-setup.md` instead was rejected: it leaves the outlier in place and
fixes only one of N call sites. Printing the path via `fmt.Fprintf("%s\n", dir)` is equivalent and gains
nothing.

## What Changes

### 1. `src/go/fab/cmd/fab/kitpath.go` — print a newline-terminated line

Single-line change inside `kitPathCmd()`'s `RunE`:

```go
// before
fmt.Fprint(cmd.OutOrStdout(), dir)

// after
fmt.Fprintln(cmd.OutOrStdout(), dir)
```

Everything else in the file is unchanged: `Use: "kit-path"`, `Args: cobra.NoArgs`, the
`kitpath.KitDir()` call, and the error wrapping
`cannot resolve kit path: %w\nRun 'fab sync' or 'fab upgrade-repo' to populate the cache`. Exit codes
are unchanged (0 on success, non-zero with the stderr error on failure). Output on success becomes the
absolute kit directory followed by exactly one `\n` and nothing else.

### 2. New test `src/go/fab/cmd/fab/kitpath_test.go`

A cobra-command test in the existing `package main` style of `src/go/fab/cmd/fab/*_test.go` (e.g.
`archive_test.go`: build the command via its constructor, `SetArgs`, `Execute`). `kitpath.KitDir()`
honors the per-process override `FAB_KIT_PATH` (constant `kitpath.KitPathEnv`, absolutized via
`filepath.Abs` and required to be an existing directory), so the test points it at a temp dir:

```go
package main

import (
	"bytes"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/kitpath"
)

// fab kit-path prints the resolved kit directory as one newline-terminated
// line (8nd2) — chained callers must never see the path fused with the next
// command's output.
func TestKitPathCmd_PrintsNewlineTerminatedLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(kitpath.KitPathEnv, dir)

	cmd := kitPathCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("kit-path: %v", err)
	}
	if got, want := buf.String(), dir+"\n"; got != want {
		t.Fatalf("output = %q, want %q (exactly one trailing newline)", got, want)
	}
}
```

Notes for the implementer:

- `t.TempDir()` already returns an absolute path, and `KitDir()` absolutizes with `filepath.Abs` (which
  does not resolve symlinks), so `dir` compares equal to what `KitDir()` returns — no `EvalSymlinks`
  dance is needed.
- The assertion is an exact byte compare of `dir + "\n"`, so it fails both on the current no-newline
  output and on any future double-newline regression.
- The error path (`FAB_KIT_PATH` pointing at a missing or non-directory path) is already covered by
  `src/go/fab/internal/kitpath/kitpath_test.go`; this test covers only the command's success-path output.
- Constitution Additional Constraints: a change to the `fab` Go binary MUST ship tests — this file is
  that test.

### 3. `src/kit/skills/_cli-fab.md` § `fab kit-path` — update the output contract

In the paragraph under the `fab kit-path` code block (currently around line 563), replace the sentence

> No trailing newline or decoration.

with

> Output is a single newline-terminated line — the path followed by exactly one `\n` — with no other
> decoration.

The rest of the paragraph is kept verbatim (the `FAB_KIT_PATH` precedence and fail-loud rules, exit
codes, the `$(fab kit-path)/templates/` usage note, the user-level operator skill note, and the
reader-path sentence). Constitution Additional Constraints: a `fab` CLI change MUST update the owning CLI
reference partial — `_cli-fab.md` owns the core family, which includes `kit-path`.

### 4. Doc sweep — other restatements of the no-newline claim

Grep repo-wide, excluding `**/archive/**`, across `docs/memory/**`, `docs/specs/**`, `src/kit/**` for
restatements of the kit-path newline contract (`trailing newline`, `no newline`, `without a newline`,
`no trailing`, and `kit-path` lines mentioning `newline`/`decoration`/`bare`/`raw`).

Pre-check performed at intake time — the only kit-path hit is the `_cli-fab.md:563` sentence changed in
§3. Unrelated hits that MUST be left alone:

| File | Why unrelated |
|------|---------------|
| `src/kit/migrations/2.14.0-to-2.15.0.md:71,93` | about writing `fab/.fab-version` with a trailing newline |
| `docs/memory/pipeline/schemas.md:103` | about `json.Encoder.Encode`'s trailing newline |
| `docs/memory/distribution/kit-architecture.md:185` | about `fab/.fab-version`'s `bare semver + \n` shape |
| `docs/memory/distribution/migrations.md:22` | about `VERSION` / `.kit-migration-version` content |
| `docs/memory/distribution/setup.md:162`, `pipeline/execution-skills.md:121`, `pipeline/planning-skills.md:190`, `runtime/operator.md:665`, `runtime/dispatch.md:153` | "trailing" used in other senses (`/`, reset, pointer argument) |

The four memory files named as possible touch points were checked: `distribution/distribution.md`
(lines 45, 57 — lists `kit-path` among config-free commands), `distribution/setup.md`,
`distribution/migrations.md` reference `$(fab kit-path)/…` paths only and do **not** restate the output
contract — no edit. `distribution/kit-architecture.md:318` describes the command
(`print the absolute resolved kit content directory that skills interpolate as $(fab kit-path)/...`)
without a newline claim; hydrate adds the newline-terminated wording there (see Affected Memory) so
memory matches the CLI reference. Apply re-runs the grep to confirm the sweep is still a no-op at
implementation time.

### 5. Verification

- `go test ./src/go/fab/cmd/fab/...` (scoped to the touched package) MUST pass, including the new test.
  Widen to `go test ./src/go/fab/...` only if something outside the package is touched.
- `gofmt -l src/go/fab/cmd/fab/kitpath.go src/go/fab/cmd/fab/kitpath_test.go` MUST print nothing
  (project lesson: worker-written Go once failed CI on gofmt).
- Live check with a dev binary (`go build -o /tmp/fab-go-dev ./src/go/fab/cmd/fab` or `fab-kit-dev` if
  present): `FAB_KIT_PATH=$PWD/src/kit /tmp/fab-go-dev kit-path | xxd | tail -1` ends in `0a`, and
  `FAB_KIT_PATH=$PWD/src/kit /tmp/fab-go-dev kit-path && echo NEXT` shows `NEXT` on its own line.

### Non-goals

- **No change to `src/kit/skills/fab-setup.md`'s Pre-flight Check wording.** The instruction ("Run
  `fab kit-path` and check that it exits 0"; "Check that `$(fab kit-path)/VERSION` exists") is correct
  once the binary prints a newline. Deployed skills and the `fab-go` binary are version-pinned together
  (`fab/.fab-version` selects both the kit and the routed binary), so a fixed `fab-setup.md` never runs
  against an unfixed binary — a defensive rewording would buy nothing.
- **No migration file.** No user data (config, `.status.yaml`, archive layout) is restructured.
- **No change to the `FAB_KIT_PATH` resolution semantics** in `internal/kitpath` or to any other
  `fmt.Fprint` site in `src/go/fab/cmd/fab/`.

## Affected Memory

- `distribution/kit-architecture`: (modify) § fab-go command list — the `fab kit-path` bullet (around
  line 318) gains "prints one newline-terminated line"; add a `## Design Decisions` entry in the
  four-field shape — **Decision**: `fab kit-path` prints the path plus exactly one `\n`. **Why**: `$(...)`
  substitution strips trailing newlines, so the old no-newline contract protected nothing, while chained
  invocations (`fab kit-path && cat "$(fab kit-path)/VERSION"` in `/fab-setup`'s pre-flight) fused the path
  with the next command's output and misled agents into a bogus `…/kit2.28.1` path. **Rejected**: keeping
  the no-newline contract and rewording `fab-setup.md` (patches one caller of N; leaves the binary's sole
  scalar-output outlier in place). *Introduced by*: 260916-8nd2-kit-path-trailing-newline.

No other memory file restates the output contract (verified by the § 4 sweep):
`distribution/distribution`, `distribution/setup`, `distribution/migrations` are unaffected.

## Impact

**Files touched (apply)**

| File | Change |
|------|--------|
| `src/go/fab/cmd/fab/kitpath.go` | `fmt.Fprint` → `fmt.Fprintln` (1 line) |
| `src/go/fab/cmd/fab/kitpath_test.go` | new, ~25 lines |
| `src/kit/skills/_cli-fab.md` | 1 sentence in § `fab kit-path` |

**Files touched (hydrate)**: `docs/memory/distribution/kit-architecture.md` (one bullet + one DD entry).

**Behavior change**: `fab kit-path` stdout gains one trailing `\n`. Exit codes, stderr, and the
`FAB_KIT_PATH` override are unchanged.

**Compatibility — not a breaking change for practical purposes.** Every in-repo consumer uses
`$(fab kit-path)/…` command substitution, which strips trailing newlines, so those consumers see
identical bytes. Consumers that *read* the output (a human at a terminal, an agent following the
user-level `fab-operator` pointer or `fab-setup`'s pre-flight) are the beneficiaries. No Go code or
script in the repo execs `fab kit-path` and byte-compares raw stdout (grep at intake: the only references
are skill substitutions, the `user-skill-fab-operator.md` template prose, a `scripts/sync-fkf.sh` comment,
and tests asserting the literal string `fab kit-path` appears in the rendered template). An out-of-tree
script doing `[ "$(fab kit-path | wc -c)" = … ]` or comparing raw piped bytes would notice; none is known.

**Release**: released `fab 2.28.1` has the bug; the fix lands with the next release. Until then a
project pinned to 2.28.1 keeps the old behavior (kit and binary are pinned together).

**Scope**: light lane — three apply edits, one hydrate touch, no migration, no spec change
(`docs/specs/*` carries no kit-path output contract).

## Open Questions

None — the user approved the decided changes and non-goals before dispatch.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix is `fmt.Fprint` → `fmt.Fprintln` in `kitpath.go:20`; no other Go change | Discussed — user approved the plan by invoking `/fab-proceed`; single-identifier change; matches every other scalar-printing command | S:95 R:95 A:95 D:95 |
| 2 | Certain | New `src/go/fab/cmd/fab/kitpath_test.go`: cobra constructor + `SetOut`/`SetArgs`/`Execute`, `t.Setenv(kitpath.KitPathEnv, t.TempDir())`, exact compare `dir + "\n"` | Discussed — test shape specified in the description; Constitution requires tests for Go CLI changes; existing `*_test.go` pattern gives one obvious form | S:90 R:90 A:90 D:85 |
| 3 | Certain | `_cli-fab.md` § `fab kit-path`: replace "No trailing newline or decoration." with the single-newline-terminated-line sentence; rest of paragraph kept | Discussed — user decided; Constitution names `_cli-fab.md` as the owning reference for core commands | S:90 R:95 A:90 D:85 |
| 4 | Certain | Doc sweep is a no-op beyond `_cli-fab.md`; hydrate touches only `kit-architecture.md` (bullet + DD entry) | Intake-time grep found no other restatement; the four named memory files reference `$(fab kit-path)/…` paths only | S:80 R:90 A:80 D:75 |
| 5 | Certain | No migration file | Discussed — no user data restructured; context.md § Migrations trigger not met | S:90 R:95 A:95 D:95 |
| 6 | Certain | Change type `fix`, light lane, recorded as non-breaking in Impact | Discussed — user said "likely `fix`"; `$(...)` consumers unaffected; 3 apply edits | S:85 R:90 A:85 D:80 |
| 7 | Certain | Verification = `go test ./src/go/fab/cmd/fab/...` + `gofmt -l` on touched Go files + dev-binary `xxd` check | Discussed — user listed the commands; code-quality.md § Test Strategy asks for scoped package tests | S:90 R:95 A:95 D:95 |
| 8 | Certain | `fab-setup.md` pre-flight wording left unchanged; no defensive tweak recorded as a deferred decision | Discussed — user constrained this and offered a defer if judged worthwhile; judged not worthwhile because kit and binary are version-pinned together, so a fixed skill never runs against an unfixed binary | S:85 R:95 A:85 D:80 |
| 9 | Certain | Error-path coverage is not added to the new test | `internal/kitpath/kitpath_test.go` already covers missing/non-directory `FAB_KIT_PATH`; the command test asserts only the success-path bytes | S:70 R:95 A:85 D:75 |

9 assumptions (9 certain, 0 confident, 0 tentative, 0 unresolved).
