# Spec: Rust vs Node Benchmark

**Change**: 260305-gt52-rust-vs-node-benchmark
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Production-quality implementations — these are benchmark-grade, not rewrite-grade
- Full statusman.sh parity — only the 3 specified operations
- Cross-platform benchmarking — single machine (dev box) only
- Decision enforcement — this change produces data, not a mandate

## Benchmark: Operations

### Requirement: Each contender SHALL implement exactly 3 operations

All 4 contenders (bash+yq baseline, optimized bash, Node, Rust) MUST implement the same 3 operations with identical CLI interfaces and output formats. This ensures apples-to-apples comparison.

**Operations**:
1. `progress-map <status_file>` — read `.status.yaml`, output `stage:state` pairs (one per line) for all 8 stages. Missing stages default to `pending`.
2. `set-change-type <status_file> <type>` — validate type is one of `feat|fix|refactor|docs|test|ci|chore`, read YAML, set `change_type` field, update `last_updated` timestamp, write atomically (tmpfile + mv).
3. `finish <status_file> <stage>` — read current state for stage, look up transition in `workflow.yaml` (finish: `[active, ready] → done`), update progress, set `last_updated`, apply metrics side-effect (set `completed_at`), auto-activate next pending stage (set to `active`, set metrics `started_at` + `iterations: 1`), write atomically.

All operations accept a direct `.status.yaml` file path (not a change name) to eliminate resolution overhead from the benchmark. The benchmark harness provides the path.

#### Scenario: Identical output across contenders
- **GIVEN** a `.status.yaml` fixture with `intake: ready, spec: pending, ...` and `change_type: chore`
- **WHEN** each contender runs `progress-map <fixture>`
- **THEN** all 4 produce identical output: `intake:ready\nspec:pending\ntasks:pending\napply:pending\nreview:pending\nhydrate:pending\nship:pending\nreview-pr:pending`

#### Scenario: Atomic write on set-change-type
- **GIVEN** a `.status.yaml` fixture
- **WHEN** a contender runs `set-change-type <fixture> feat`
- **THEN** the file contains `change_type: feat` and `last_updated:` is a recent ISO 8601 timestamp
- **AND** no partial writes are visible (atomic via tmpfile + mv)

#### Scenario: Finish with auto-activate
- **GIVEN** a `.status.yaml` with `intake: active, spec: pending`
- **WHEN** a contender runs `finish <fixture> intake`
- **THEN** `intake: done`, `spec: active`
- **AND** `stage_metrics.intake.completed_at` is set
- **AND** `stage_metrics.spec.started_at` is set and `iterations: 1`

### Requirement: Contenders SHALL accept `.status.yaml` path directly

To isolate YAML processing speed from resolution overhead, all contenders SHALL accept the status file path as a direct argument. The benchmark harness resolves the path and passes it.

#### Scenario: Direct path acceptance
- **GIVEN** a status file at `/tmp/bench/fixture/.status.yaml`
- **WHEN** a contender runs `progress-map /tmp/bench/fixture/.status.yaml`
- **THEN** it reads and processes that file directly (no fab/changes/ scanning)

## Benchmark: Contenders

### Requirement: Bash+yq baseline SHALL use the production statusman.sh logic

The baseline contender wraps the real `statusman.sh` functions. It does NOT re-implement anything — it calls the existing script with the fixture's `.status.yaml` path. This establishes the true production cost.

#### Scenario: Baseline invocation
- **GIVEN** the production `fab/.kit/scripts/lib/statusman.sh`
- **WHEN** the harness calls `statusman.sh progress-map <fixture-path>`
- **THEN** the output matches the expected format
- **AND** timing reflects real-world yq subprocess overhead

### Requirement: Optimized bash SHALL minimize subprocess spawns

The optimized bash contender MUST:
1. Batch `yq` reads — use single `yq eval` with compound expressions to extract multiple fields at once
2. Use `awk` for writes instead of `yq -i` — read the file, transform with awk, write to tmpfile, mv
3. Read `workflow.yaml` with `awk` (not `yq`) for transition lookup in `finish`
4. Maintain the same output format and atomic write semantics as baseline

#### Scenario: Batched read in progress-map
- **GIVEN** a `.status.yaml` fixture
- **WHEN** optimized bash runs `progress-map`
- **THEN** it invokes `yq` at most once (single compound expression extracting all 8 stages)

#### Scenario: Awk-based write in set-change-type
- **GIVEN** a `.status.yaml` fixture
- **WHEN** optimized bash runs `set-change-type <fixture> feat`
- **THEN** no `yq -i` is invoked — the write uses `awk` piped to a tmpfile

### Requirement: Node contender SHALL use js-yaml (pure JS)

The Node contender MUST:
1. Use `js-yaml` package for YAML parsing and serialization (no native bindings)
2. Be a single ESM file (`statusman.mjs`) or small module
3. Parse CLI args from `process.argv`
4. Use `fs.writeFileSync` to tmpfile + `fs.renameSync` for atomic writes

#### Scenario: Node startup
- **GIVEN** `node` and `js-yaml` installed in `src/benchmark/statusman-node/`
- **WHEN** the harness calls `node statusman.mjs progress-map <fixture>`
- **THEN** it produces correct output

### Requirement: Rust contender SHALL compile to a single release binary

The Rust contender MUST:
1. Use `serde` + `serde_yaml` for YAML deserialization/serialization
2. Use `clap` for CLI argument parsing (or manual arg parsing to minimize binary size)
3. Compile with `--release` for benchmarking
4. Use `tempfile` crate or manual tmpfile + rename for atomic writes

#### Scenario: Rust binary
- **GIVEN** `cargo build --release` has been run in `src/benchmark/statusman-rust/`
- **WHEN** the harness calls `./target/release/statusman progress-map <fixture>`
- **THEN** it produces correct output from a single statically-linked binary

## Benchmark: Harness

### Requirement: Harness SHALL use hyperfine for all measurements

The benchmark harness (`src/benchmark/bench.sh`) MUST use `hyperfine` with:
- `--warmup 3` for warm benchmarks
- `--min-runs 50` for statistical significance
- `--export-json` for machine-readable results
- `--prepare` to reset fixture state between write/transition operations (cp from pristine copy)

The harness MUST NOT measure cold-start with cache-clearing (`/proc/sys/vm/drop_caches`) — this requires root and is fragile. Instead, measure startup overhead via a `--help` or no-op invocation.

#### Scenario: Warm benchmark
- **GIVEN** 4 contenders and a fixture
- **WHEN** the harness runs warm benchmarks for `progress-map`
- **THEN** `hyperfine --warmup 3 --min-runs 50` runs each contender
- **AND** JSON results are exported to `src/benchmark/results/progress-map.json`

#### Scenario: Fixture reset for write operations
- **GIVEN** a write operation (`set-change-type` or `finish`) that modifies the fixture
- **WHEN** the harness benchmarks it
- **THEN** `--prepare 'cp pristine.yaml fixture.yaml'` resets state between runs

#### Scenario: Startup overhead measurement
- **GIVEN** 4 contenders
- **WHEN** the harness measures startup overhead
- **THEN** it benchmarks `<contender> --help` (or a no-op subcommand) to isolate process startup cost

### Requirement: Harness SHALL produce RESULTS.md

After all benchmarks complete, the harness SHALL generate `src/benchmark/RESULTS.md` containing:
1. **Environment**: machine info (`uname -a`, CPU, RAM), tool versions (`yq`, `node`, `rustc`, `hyperfine`)
2. **Startup overhead**: mean time for `--help` per contender
3. **Operation results**: table per operation with mean, stddev, min, max, relative speed vs baseline
4. **Summary**: overall ranking and key takeaways
5. **Raw data**: paths to JSON files for reproducibility

#### Scenario: Results table format
- **GIVEN** completed benchmark JSON files
- **WHEN** the harness generates RESULTS.md
- **THEN** each operation table looks like:

```
### progress-map
| Contender      | Mean     | Stddev  | Min      | Max      | Relative |
|----------------|----------|---------|----------|----------|----------|
| bash+yq        | 120.3 ms | 5.2 ms | 112.1 ms | 135.7 ms | 1.00x    |
| optimized bash | 45.1 ms  | 3.1 ms | 40.2 ms  | 52.3 ms  | 2.67x    |
| node           | 35.2 ms  | 2.8 ms | 31.5 ms  | 42.1 ms  | 3.42x    |
| rust            | 3.1 ms  | 0.4 ms | 2.6 ms   | 4.2 ms   | 38.8x   |
```
(numbers are illustrative, not predicted)

## Benchmark: File Layout

### Requirement: All benchmark code SHALL live under `src/benchmark/`

```
src/benchmark/
├── bench.sh                        # Harness script
├── fixtures/                       # Test data
│   ├── status.yaml                 # Pristine .status.yaml (copy of real one)
│   └── workflow.yaml               # Copy of fab/.kit/schemas/workflow.yaml
├── statusman-bash-opt/
│   └── statusman.sh                # Optimized bash implementation
├── statusman-node/
│   ├── package.json
│   └── statusman.mjs               # Node implementation
├── statusman-rust/
│   ├── Cargo.toml
│   └── src/
│       └── main.rs                 # Rust implementation
├── results/                        # hyperfine JSON output
│   ├── startup.json
│   ├── progress-map.json
│   ├── set-change-type.json
│   └── finish.json
└── RESULTS.md                      # Generated summary
```

#### Scenario: Self-contained benchmark
- **GIVEN** the `src/benchmark/` directory
- **WHEN** a developer runs `bash src/benchmark/bench.sh`
- **THEN** it builds the Rust contender, installs Node deps, runs all benchmarks, and produces `RESULTS.md`
- **AND** no files outside `src/benchmark/` are modified

## Design Decisions

1. **Direct path instead of resolution**: Contenders accept `.status.yaml` path directly rather than change names
   - *Why*: Isolates YAML processing speed from filesystem scanning. Resolution overhead is identical across contenders (they'd all scan the same directory), so measuring it adds noise without signal.
   - *Rejected*: Full resolve + statusman interface — adds complexity to each implementation without measuring anything useful.

2. **No cold-start cache clearing**: Use `--help` timing instead of `/proc/sys/vm/drop_caches`
   - *Why*: Cache clearing requires root, is OS-specific, and measures OS page cache behavior rather than application performance. Startup overhead via `--help` cleanly isolates process startup cost.
   - *Rejected*: `sudo` cache clearing — fragile, root-dependent, conflates OS caching with app performance.

3. **Fixture copy for write operations**: Use `--prepare 'cp pristine.yaml fixture.yaml'` in hyperfine
   - *Why*: Write operations modify the fixture. Resetting between runs ensures each iteration starts from identical state. `cp` is fast enough to not distort measurements.
   - *Rejected*: In-memory-only benchmarks — wouldn't measure real filesystem I/O which is part of the production cost.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Benchmark goes in `src/benchmark/` | Confirmed from intake #1; user explicitly agreed | S:95 R:95 A:95 D:95 |
| 2 | Certain | `statusman.sh` as benchmark subject | Confirmed from intake #2 | S:95 R:90 A:90 D:95 |
| 3 | Confident | 3-operation subset sufficient | Confirmed from intake #3; covers read/write/transition spectrum | S:70 R:85 A:75 D:70 |
| 4 | Confident | Node uses `js-yaml` (pure JS) | Confirmed from intake #4 | S:65 R:90 A:80 D:75 |
| 5 | Confident | Rust uses `serde_yaml` + `clap` | Confirmed from intake #5 | S:70 R:85 A:85 D:80 |
| 6 | Certain | `hyperfine` is a hard requirement | Confirmed from intake #6; user clarified | S:95 R:90 A:90 D:95 |
| 7 | Confident | Results in co-located `RESULTS.md` | Confirmed from intake #7 | S:70 R:90 A:80 D:85 |
| 8 | Certain | 4 contenders including optimized bash | Confirmed from intake #8; user agreed in discussion | S:95 R:90 A:90 D:95 |
| 9 | Certain | Direct `.status.yaml` path (no resolution) | Eliminates resolution overhead from measurements; all contenders would have identical resolution cost anyway | S:85 R:95 A:90 D:90 |
| 10 | Confident | No cold-start cache clearing; use --help timing | Avoids root requirement and OS-specific behavior; startup overhead via --help is cleaner | S:60 R:90 A:75 D:70 |
| 11 | Certain | Fixture reset via cp in --prepare | Standard hyperfine pattern for stateful benchmarks; cp overhead is negligible vs operation cost | S:85 R:95 A:90 D:90 |

11 assumptions (6 certain, 5 confident, 0 tentative, 0 unresolved).
