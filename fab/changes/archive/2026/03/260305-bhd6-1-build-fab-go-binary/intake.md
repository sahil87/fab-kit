# Intake: Build fab Go Binary

**Change**: 260305-bhd6-1-build-fab-go-binary
**Created**: 2026-03-05
**Status**: Draft

## Origin

> 1-build-fab-binary: Build the `fab` Go binary — scaffold Go module at src/go/fab/ with cobra CLI, then port all 8 lib/ shell scripts (statusman.sh, resolve.sh, logman.sh, preflight.sh, changeman.sh, calc-score.sh, archiveman.sh) into subcommands of a single `fab` binary. Shared internal/yaml package for .status.yaml struct. Tasks follow dependency graph: scaffold first, then status+resolve+log in parallel, then preflight, then change+score in parallel, then archive.

This change follows a benchmarking effort (260305-gt52-rust-vs-node-benchmark) that evaluated Rust, Go, Node, and optimized bash as replacements for the shell+yq scripts. The benchmark demonstrated Go as the recommended target: 8-49x faster than bash+yq baseline, sub-millisecond for all operations, trivial cross-compilation, and constitution-compliant as a single-binary utility.

The decision was made in a `/fab-discuss` session where the full dependency graph of all lib/ scripts was analyzed. Key finding: `statusman.sh` is the sole YAML bottleneck (43 yq invocations), but the compound win comes from collapsing the entire call chain (preflight → resolve → statusman → changeman → logman) into in-process function calls within a single binary.

## Why

1. **Performance**: Every skill invocation pays a ~25ms tax through the preflight → statusman → resolve call chain. In agent workflows running dozens of skills per session, this accumulates. The Go binary eliminates subprocess spawning entirely — the full preflight check drops from ~25ms to <1ms.

2. **Compound call chain elimination**: The benchmark only tested `statusman.sh` in isolation. In production, scripts chain through 3-5 subprocesses per operation. A single binary collapses all inter-script communication to in-process function calls.

3. **yq dependency elimination**: `statusman.sh` requires `yq` v4 — a 12.6 MB Go binary that must be installed separately. The `fab` binary internalizes YAML parsing, removing this external dependency for users.

4. **Constitution alignment**: Principle I allows "single-binary utilities" — `fab` is exactly that pattern, consistent with how `yq` and `gh` are used in the ecosystem.

## What Changes

### Go Module Scaffold (`src/go/fab/`)

New Go module at `src/go/fab/` with the following structure:

```
src/go/fab/
├── cmd/
│   └── fab/
│       └── main.go              # cobra root command, registers all subcommands
├── internal/
│   ├── statusfile/              # .status.yaml: parse, modify, write
│   │   ├── statusfile.go        # StatusFile struct + Load/Save
│   │   └── statusfile_test.go
│   ├── resolve/                 # change folder resolution
│   ├── log/                     # JSON-line append to .history.jsonl
│   ├── status/                  # statusman subcommands
│   ├── preflight/               # validation + aggregation
│   ├── change/                  # create, rename, list, switch
│   ├── score/                   # confidence scoring
│   └── archive/                 # archive/restore
├── go.mod
└── go.sum
```

Uses `cobra` for CLI framework. `internal/statusfile/` is the shared foundation — one `StatusFile` struct parsed once, passed by pointer to all operations. This is the main advantage over bash where each yq call re-parses the YAML.

Dependencies: `cobra` (CLI), `gopkg.in/yaml.v3` (YAML parsing). No CGo — pure Go for clean cross-compilation.

### Port: statusman.sh → `fab status` (43 yq calls)

Port the complete statusman.sh state machine to `fab status <subcommand>`. All subcommands from the existing script:

- `fab status finish <change> <stage> [driver]`
- `fab status start <change> <stage> [driver] [from] [reason]`
- `fab status advance <change> <stage> [driver]`
- `fab status reset <change> <stage> [driver] [from] [reason]`
- `fab status skip <change> <stage> [driver]`
- `fab status fail <change> <stage> [driver] [rework]`
- `fab status set-change-type <change> <type>`
- `fab status set-checklist <change> <field> <value>`
- `fab status set-confidence <change> <counts...> <score> [--indicative]`
- `fab status set-confidence-fuzzy <change> <counts...> <score> <dims...> [--indicative]`
- `fab status progress-line <change>`
- `fab status current-stage <change>`
- `fab status progress-map <change>`
- `fab status add-issue <change> <issue-id>`
- `fab status add-pr <change> <pr-url>`

Must produce **identical stdout/stderr output** to the bash version for parity testing. Stage transition side effects (auto-activate next stage, auto-log review results) must be preserved.

### Port: resolve.sh → `fab resolve`

Port the change resolver:

- `fab resolve [--id|--folder|--dir|--status] [<change>]`

Supports 4-char ID, folder name substring (case-insensitive), and full folder name. Reads `fab/current` when no argument given. Must handle collision detection (multiple matches → error).

### Port: logman.sh → `fab log`

Port the append-only JSON logger:

- `fab log command <cmd> [change] [args]`
- `fab log confidence <change> <score> <delta> <trigger>`
- `fab log review <change> <result> [rework]`
- `fab log transition <change> <stage> <action> [from] [reason] [driver]`

Appends JSON lines to `fab/changes/{name}/.history.jsonl`. Silent exit on resolution failure for `command` without explicit change.

### Port: preflight.sh → `fab preflight`

Port the validation entry point:

- `fab preflight [<change-name>]`

This is where the compound win materializes. Today preflight shells out to resolve.sh and statusman.sh multiple times. In Go, these become in-process function calls to `internal/resolve` and `internal/statusfile`. Must produce identical structured YAML output on stdout.

### Port: changeman.sh → `fab change`

Port the change lifecycle manager:

- `fab change new --slug <slug> [--change-id <4char>] [--log-args <desc>]`
- `fab change rename --folder <current-folder> --slug <new-slug>`
- `fab change resolve [<override>]`
- `fab change switch <name> | --blank`
- `fab change list [--archive]`

Handles `fab/current` pointer, date generation, random ID generation, collision detection, `created_by` detection (git config + gh api), `.status.yaml` initialization from template, and logman integration.

### Port: calc-score.sh → `fab score`

Port the confidence scorer:

- `fab score [--check-gate] [--stage <stage>] <change>`

Parses Assumptions tables from intake.md or spec.md using the same regex patterns. Computes SRAD composite scores. Writes confidence block to `.status.yaml` via internal statusfile package. Gate check mode compares against per-type thresholds.

### Port: archiveman.sh → `fab archive`

Port the archive manager:

- `fab archive <change> --description "..."`
- `fab archive restore <change> [--switch]`
- `fab archive list`

File operations: move directories, update `fab/changes/archive/index.md`, manage `fab/current` pointer via internal change package.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document Go binary as part of the kit architecture, update script architecture section
- `fab-workflow/distribution`: (modify) Document per-platform archives and binary inclusion in kit.tar.gz

## Impact

- **Source**: New `src/go/fab/` directory with Go module (~1,500-2,000 lines estimated)
- **Scripts**: All 8 lib/ scripts (`statusman.sh`, `resolve.sh`, `logman.sh`, `preflight.sh`, `changeman.sh`, `calc-score.sh`, `archiveman.sh`, `frontmatter.sh`) — shell scripts remain unchanged in this change, shim + switchover is a separate change
- **Dependencies**: New Go toolchain requirement for building from source (not for end users — they get pre-built binaries)
- **Kit distribution**: No changes in this change — release integration is a separate change
- **Existing tests**: `src/packages/tests/` (bats tests) — unaffected, they test shell scripts

## Open Questions

- None — all design decisions were resolved in the preceding discussion session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go as the implementation language | Discussed — benchmark data (8-49x over bash+yq), trivial cross-compilation, constitution-compliant | S:95 R:85 A:95 D:95 |
| 2 | Certain | Single `fab` binary with subcommands | Discussed — eliminates inter-script subprocess overhead, matches yq/gh pattern | S:90 R:80 A:90 D:90 |
| 3 | Certain | Module location at `src/go/fab/` | Discussed — follows existing `src/` convention, keeps Go source separate from kit distribution | S:85 R:90 A:90 D:90 |
| 4 | Certain | Cobra for CLI framework | Standard Go CLI library, well-maintained, supports nested subcommands | S:80 R:90 A:90 D:95 |
| 5 | Certain | `internal/statusfile/` as shared YAML package | Discussed — single StatusFile struct parsed once, passed by pointer | S:90 R:85 A:90 D:90 |
| 6 | Certain | Identical CLI interface to bash scripts | Required for parity testing and shim switchover (separate change) | S:85 R:70 A:90 D:95 |
| 7 | Confident | `gopkg.in/yaml.v3` for YAML parsing | Standard Go YAML library, proven in yq itself. Alternatives (goccy/go-yaml) exist but yaml.v3 is more widely used | S:75 R:90 A:85 D:70 |
| 8 | Confident | No CGo dependencies | Discussed — pure Go enables trivial cross-compilation without target-specific toolchains | S:80 R:75 A:85 D:85 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
