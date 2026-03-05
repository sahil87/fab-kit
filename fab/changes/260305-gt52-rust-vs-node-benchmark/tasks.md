# Tasks: Rust vs Node Benchmark

**Change**: 260305-gt52-rust-vs-node-benchmark
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create directory structure: `src/benchmark/`, `fixtures/`, `statusman-bash-opt/`, `statusman-node/`, `statusman-rust/`, `results/`
- [x] T002 [P] Create fixtures: copy a representative `.status.yaml` to `src/benchmark/fixtures/status.yaml` and `fab/.kit/schemas/workflow.yaml` to `src/benchmark/fixtures/workflow.yaml`
- [x] T003 [P] Scaffold Rust project: `cargo init src/benchmark/statusman-rust` with `serde`, `serde_yaml`, `clap` dependencies in `Cargo.toml`
- [x] T004 [P] Scaffold Node project: `src/benchmark/statusman-node/package.json` with `js-yaml` dependency, create empty `statusman.mjs`

## Phase 2: Core Implementation

- [x] T005 Implement optimized bash `src/benchmark/statusman-bash-opt/statusman.sh` — `progress-map` (single yq eval), `set-change-type` (awk write), `finish` (awk-based transition + write)
- [x] T006 [P] Implement Node `src/benchmark/statusman-node/statusman.mjs` — `progress-map`, `set-change-type`, `finish` with `js-yaml` and atomic writes
- [x] T007 [P] Implement Rust `src/benchmark/statusman-rust/src/main.rs` — `progress-map`, `set-change-type`, `finish` with `serde_yaml` and atomic writes. Build with `cargo build --release`

## Phase 3: Integration & Edge Cases

- [x] T008 Validate all 4 contenders produce identical output for `progress-map` against the fixture
- [x] T009 Validate all 4 contenders produce correct `.status.yaml` after `set-change-type` and `finish`
- [x] T010 Write benchmark harness `src/benchmark/bench.sh` — run hyperfine for startup, progress-map, set-change-type, finish across all 4 contenders with fixture reset
- [x] T011 Run benchmarks and generate `src/benchmark/RESULTS.md` from hyperfine JSON output

## Phase 4: Polish

- [x] T012 Add a brief README to `src/benchmark/` explaining how to run the benchmarks and prerequisites

---

## Execution Order

- T001 blocks all others (directory structure)
- T002, T003, T004 are parallel after T001
- T005 depends on T002 (fixtures)
- T006 depends on T002 + T004 (fixtures + node scaffold)
- T007 depends on T002 + T003 (fixtures + rust scaffold)
- T005, T006, T007 are parallel with each other
- T008, T009 depend on T005 + T006 + T007 (all implementations)
- T010 depends on T008, T009 (validated implementations)
- T011 depends on T010 (harness)
- T012 is independent after T010
