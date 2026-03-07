# Intake: Rust Binary Port

**Change**: 260307-bmp3-3-rust-binary-port
**Created**: 2026-03-07
**Status**: Draft

## Origin

> Big bang port of the Go binary (`src/fab-go/`) to Rust (`src/fab-rust/`). All 9 subcommands ported at once. The dispatcher (`fab/.kit/bin/fab`) already supports `fab-rust` with higher priority than `fab-go`. Add a mechanism to switch back to Go for comparison during the transition period. This is step 3 of the 4-part plan.

Discussion context: User chose big bang over incremental porting because the incremental approach doesn't work well — cobra/clap each want to own the full command tree, making per-subcommand delegation more complex than just porting everything. The Go codebase is small enough (~38 source files, 9 subcommands, deps: cobra + yaml.v3) to port in one shot. User wants an env var or file-based mechanism to switch back to Go for comparison. User confirmed `clap` derive for CLI help (equivalent to cobra's auto-help).

Depends on: `260307-b56y-2-tag-triggered-releases` (CI pipeline should be in place before adding a new binary).

## Why

The Go binary works, but Rust offers:
1. **Smaller binaries** — Rust with `strip` + `lto` produces ~2-3MB vs Go's ~6-8MB (Go embeds runtime + GC)
2. **Truly static binaries** — Rust with musl is the gold standard for static linking
3. **Better CLI framework** — `clap` derive gives auto-help, shell completions, man pages out of the box
4. **Ecosystem alignment** — growing Rust CLI ecosystem (`serde_yaml`, `clap`, `anyhow`/`thiserror`)

The binary is small and straightforward (YAML parsing, file I/O, symlink management, no async, no complex concurrency) — firmly in the "easy Rust" zone.

## What Changes

### New: `src/fab-rust/` Rust project

A new Rust binary crate at `src/fab-rust/` implementing all 9 subcommands:

| Subcommand | Go source | Description |
|------------|-----------|-------------|
| `resolve` | `cmd/fab/resolve.go` + `internal/resolve/` | Change reference resolution |
| `log` | `cmd/fab/log.go` + `internal/log/` | Append-only history logging |
| `status` | `cmd/fab/status.go` + `internal/status/` | Stage state machine + metadata |
| `preflight` | `cmd/fab/preflight.go` + `internal/preflight/` | Validation + structured YAML output |
| `change` | `cmd/fab/change.go` + `internal/change/` | Change lifecycle management |
| `score` | `cmd/fab/score.go` + `internal/score/` | Confidence scoring |
| `runtime` | `cmd/fab/runtime.go` | Runtime state management |
| `pane-map` | `cmd/fab/panemap.go` | Tmux pane-to-worktree mapping |
| `send-keys` | `cmd/fab/sendkeys.go` | Send text to a change's tmux pane |

Key Rust dependencies:
- `clap` (derive) — CLI argument parsing, auto-help, completions
- `serde` + `serde_yaml` — YAML serialization/deserialization
- `anyhow` — error handling
- Minimal additional deps — keep the dependency tree small

Project structure:
```
src/fab-rust/
├── Cargo.toml
├── Cargo.lock
└── src/
    ├── main.rs          # clap CLI definition, subcommand dispatch
    ├── resolve.rs       # resolve subcommand
    ├── log.rs           # log subcommand
    ├── status.rs        # status subcommand + stage machine
    ├── preflight.rs     # preflight subcommand
    ├── change.rs        # change subcommand (new, rename, switch, list, archive, restore)
    ├── score.rs         # confidence scoring
    ├── runtime.rs       # runtime state management
    ├── panemap.rs       # tmux pane mapping
    ├── sendkeys.rs      # tmux send-keys
    ├── config.rs        # shared config loading (config.yaml)
    ├── statusfile.rs    # .status.yaml read/write
    └── resolve_common.rs # shared change resolution logic
```

### Modified: `fab/.kit/bin/fab` dispatcher

The dispatcher already supports `fab-rust` > `fab-go` priority. Add a **backend override mechanism**:

```sh
# Backend override: FAB_BACKEND env var or .fab-backend file
backend_override="${FAB_BACKEND:-}"
if [ -z "$backend_override" ] && [ -f "$SCRIPT_DIR/../../.fab-backend" ]; then
  backend_override=$(cat "$SCRIPT_DIR/../../.fab-backend" | tr -d '[:space:]')
fi

if [ "$backend_override" = "go" ] && [ -x "$SCRIPT_DIR/fab-go" ]; then
  exec "$SCRIPT_DIR/fab-go" "$@"
elif [ "$backend_override" = "rust" ] && [ -x "$SCRIPT_DIR/fab-rust" ]; then
  exec "$SCRIPT_DIR/fab-rust" "$@"
fi

# Default priority: rust > go (unchanged)
```

Override via:
- `FAB_BACKEND=go fab resolve` — per-command override
- `echo "go" > .fab-backend` — persistent project-level override (`.fab-backend` at repo root, gitignored)

The `.fab-backend` file lives at the repo root (two levels up from `fab/.kit/bin/`), is gitignored, and contains just `go` or `rust`.

### Modified: `justfile` (from change 1)

Add Rust build recipe for local dev:

```just
# Build Rust binary for the current platform (local dev)
build-rust:
    cargo build --manifest-path src/fab-rust/Cargo.toml --release
    cp target/release/fab-rust fab/.kit/bin/fab-rust
```

### Existing: Go parity tests

The Go parity tests at `src/fab-go/test/parity/` verify behavior against shell script baselines. The Rust port MUST pass equivalent tests. Strategy:
- Port the parity test suite to Rust integration tests (`src/fab-rust/tests/`)
- OR run the existing Go parity tests against the Rust binary (by temporarily symlinking `fab-rust` as `fab-go`)
- OR create a shared test harness that runs against whichever binary is present

### New: `.gitignore` entry

Add `.fab-backend` to `.gitignore` — this is a local developer preference, not committed.

## Affected Memory

- `fab-workflow/distribution`: (modify) Document the Rust backend, backend override mechanism (FAB_BACKEND env var, .fab-backend file), and the transition period where both binaries are shipped
- `fab-workflow/kit-architecture`: (modify) Document the Rust source at `src/fab-rust/`, its dependencies, and project structure

## Impact

- **`src/fab-rust/`**: New Rust crate — all 9 subcommands
- **`fab/.kit/bin/fab`**: Small modification — add backend override check (~8 lines)
- **`fab/.kit/bin/fab-rust`**: New binary (built locally or by CI)
- **`.gitignore`**: Add `.fab-backend`
- **`justfile`**: Add `build-rust` recipe
- **Go binary**: Unchanged — continues to work as fallback
- **Parity tests**: Need a strategy for testing Rust binary against expected behavior

## Open Questions

- Should the Rust binary be named `fab` directly (replacing the dispatcher) long-term, or keep the dispatcher pattern permanently? The dispatcher adds ~5ms startup overhead but provides clean backend switching.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Big bang port — all 9 subcommands at once | Discussed — incremental doesn't work well because each CLI framework wants to own the full command tree | S:95 R:70 A:90 D:95 |
| 2 | Certain | Use `clap` derive for CLI parsing | Discussed — equivalent to cobra's auto-help, also gives completions and man pages | S:90 R:85 A:90 D:90 |
| 3 | Certain | Backend override via FAB_BACKEND env var + .fab-backend file | Discussed — user wants mechanism to switch back to Go for comparison | S:85 R:90 A:85 D:80 |
| 4 | Certain | Dispatcher priority remains rust > go by default | Already implemented in current fab dispatcher | S:95 R:85 A:95 D:95 |
| 5 | Confident | Use `serde_yaml` for YAML handling | Standard Rust YAML library, mature enough for this use case | S:70 R:80 A:85 D:75 |
| 6 | Confident | Use `anyhow` for error handling | Standard pattern for CLI tools — simple, good error messages | S:70 R:85 A:80 D:80 |
| 7 | Confident | Flat module structure (one file per subcommand + shared modules) | Matches Go's flat structure, codebase is small enough | S:75 R:90 A:80 D:75 |
| 8 | Confident | .fab-backend file at repo root, gitignored | User suggested file-based override; repo root is the natural location | S:80 R:90 A:80 D:75 |
| 9 | Tentative | Port Go parity tests to Rust integration tests rather than shared harness | Rust-native tests are simpler to maintain; shared harness adds complexity. But the existing Go tests could be reused | S:50 R:75 A:60 D:50 |

9 assumptions (4 certain, 4 confident, 1 tentative, 0 unresolved).
