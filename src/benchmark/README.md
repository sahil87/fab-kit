# statusman.sh Benchmark: Bash vs Optimized Bash vs Node vs Rust

Compares 4 implementations of core `statusman.sh` operations to inform the runtime choice for kit script rewrites.

## Prerequisites

- `yq` v4 (Go version)
- `node` (v18+)
- `cargo` / `rustc` — install via [rustup](https://rustup.rs/) if not present:
  ```bash
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
  source "$HOME/.cargo/env"
  ```
- `hyperfine` — install via cargo if not available:
  ```bash
  cargo install hyperfine
  ```

## Setup

```bash
# Install Node deps
cd src/benchmark/statusman-node && npm install && cd -

# Build Rust contender
cd src/benchmark/statusman-rust && cargo build --release && cd -
```

## Running

```bash
# From repo root:
bash src/benchmark/bench.sh
```

Runs all benchmarks via `hyperfine` and generates `RESULTS.md`. Requires the Rust binary and Node deps to be pre-built.

## What's benchmarked

3 operations across 4 contenders:

| Operation | What it tests |
|-----------|---------------|
| `progress-map` | YAML read — parse all 8 stage progress values |
| `set-change-type` | YAML write — validate, modify, atomic write |
| `finish` | Complex transition — read, lookup, multi-field update, auto-activate next stage |

## Contenders

1. **bash+yq** — production `statusman.sh` (baseline)
2. **optimized-bash** — batched yq reads + awk writes
3. **node** — `js-yaml` (pure JS)
4. **rust** — `serde_yaml` (compiled binary)

See `RESULTS.md` for the latest numbers.
