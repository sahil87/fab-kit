# Intake: Rust vs Node Benchmark

**Change**: 260305-gt52-rust-vs-node-benchmark
**Created**: 2026-03-05
**Status**: Draft

## Origin

> Evaluate rust vs node for speed for moving the tool scripts. Take any one command (say statusman.sh) and create it node and rust version. Compare performance.

One-shot request from backlog. The project has previously discussed Rust rewrites (e.g., the stageman CLI-only migration in `260215-lqm5` was explicitly designed to prepare for a Rust rewrite). This change produces the data to inform whether Rust or Node is the right target runtime.

## Why

1. **Problem**: The kit's shell scripts (`statusman.sh`, `changeman.sh`, `resolve.sh`, etc.) rely heavily on `yq` for YAML manipulation. Each `yq` invocation spawns a Go binary, parses the YAML file from scratch, and writes it back. A single `statusman.sh finish` call can invoke `yq` 10+ times. This adds up to noticeable latency in the agent workflow — every skill invocation pays this tax.

2. **Consequence**: If we don't measure, we pick a runtime on vibes. Rust and Node have very different tradeoffs: Rust produces a single static binary (aligns with constitution's "single-binary utilities" principle) but has higher implementation cost. Node is faster to write but introduces a runtime dependency and `node_modules`.

3. **Approach**: Implement a representative subset of `statusman.sh` in four variants — current bash+yq (baseline), optimized bash (batched yq + awk writes), Node, and Rust — run controlled benchmarks with `hyperfine` on the same machine, and compare startup time, typical operation latency, and cold-start behavior. The benchmark results will inform the rewrite target for all kit scripts.

## What Changes

### Benchmark subject: `statusman.sh` subset

The benchmark implementations cover a representative operation mix:

1. **Read operation** — `progress-map <change>`: Parse `.status.yaml`, read all stage progress values, output `stage:state` pairs. Tests YAML parsing speed.
2. **Write operation** — `set-change-type <change> <type>`: Validate type, read YAML, set field, write atomically (tmp + mv). Tests YAML read-modify-write speed.
3. **Transition operation** — `finish <change> <stage>`: Read YAML, look up transition in `workflow.yaml`, update progress, auto-activate next stage, write atomically. Tests complex multi-read-write operation.

These three cover the spectrum: pure read, simple write, and complex state machine logic.

### Optimized bash implementation (`src/benchmark/statusman-bash-opt/`)

- Fork of the relevant statusman.sh functions, rewritten to minimize subprocess spawns
- Batch `yq` reads into single `yq eval` calls with compound expressions (read all fields at once instead of one-per-call)
- Use `awk` for writes instead of `yq -i` (as `set_confidence_block_fuzzy` already does at line 542 of the original)
- Same CLI interface and output format as the original
- Goal: measure how much of the bash+yq overhead is eliminable without leaving bash

### Node implementation (`src/benchmark/statusman-node/`)

- Single `statusman.mjs` file (or small module)
- Uses `js-yaml` for YAML parsing (no native bindings — fair comparison)
- CLI interface matching the bash version's subcommands
- Resolve logic: find change folder by ID/substring match in `fab/changes/`

### Rust implementation (`src/benchmark/statusman-rust/`)

- Cargo project with `serde_yaml` for YAML
- CLI via `clap` or manual arg parsing
- Same subcommand interface as bash/node versions
- Compiled binary for benchmarking (release mode)

### Benchmark harness (`src/benchmark/bench.sh`)

Shell script that:
1. Sets up a test fixture (copies a real `.status.yaml` and `workflow.yaml` to a temp dir)
2. Runs each of the 4 implementations against each of the 3 operations using `hyperfine` (required dependency)
3. Uses `hyperfine --warmup 3 --min-runs 50` for warm benchmarks, `--prepare 'sync; echo 3 | sudo tee /proc/sys/vm/drop_caches || true'` for cold-start
4. Produces `hyperfine` JSON export for post-processing
5. Generates `RESULTS.md` summary from the JSON data

### Expected output

A `RESULTS.md` file in the benchmark directory summarizing:
- Startup overhead (empty/help invocation)
- Read operation latency
- Write operation latency
- Complex transition latency
- Cold start vs warm comparison
- Binary size (Rust) vs dependency footprint (Node)

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Record the benchmark findings and runtime decision if one emerges

## Impact

- **No production code changes** — this is a research spike
- New files in `src/benchmark/` only
- Requires `rustc`/`cargo` and `node` on the benchmark machine
- Results inform future architectural decisions for all kit scripts

## Open Questions

- Is the 3-operation subset representative enough, or should we add `progress-line` (string formatting) too?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Benchmark goes in `src/benchmark/`, not in `fab/.kit/` | Constitution: `.kit/` is distributed to users; benchmarks are internal tooling | S:90 R:95 A:95 D:95 |
| 2 | Certain | Use `statusman.sh` as the benchmark subject | Explicitly specified in the request; it's the most complex script with the most `yq` calls | S:95 R:90 A:90 D:95 |
| 3 | Confident | 3-operation subset (read/write/transition) is sufficient | Covers the YAML operation spectrum; adding more would increase implementation effort without proportional insight | S:70 R:85 A:75 D:70 |
| 4 | Confident | Node version uses `js-yaml` (pure JS) not native bindings | Fair comparison — native bindings would blur the Rust/Node boundary | S:65 R:90 A:80 D:75 |
| 5 | Confident | Rust version uses `serde_yaml` + `clap` | De facto standard crates for YAML and CLI in Rust ecosystem | S:70 R:85 A:85 D:80 |
| 6 | Certain | `hyperfine` is a hard requirement for benchmarking | Clarified — user confirmed; single binary, handles warmup and statistical analysis | S:95 R:90 A:90 D:95 |
| 7 | Confident | Results captured in `RESULTS.md` alongside the benchmark code | Keeps findings co-located with the code that produced them | S:70 R:90 A:80 D:85 |
| 8 | Certain | 4 contenders: bash+yq baseline, optimized bash, Node, Rust | Discussed — user agreed to add optimized bash as 4th contender to test whether bash-level optimization is sufficient | S:95 R:90 A:90 D:95 |

8 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
