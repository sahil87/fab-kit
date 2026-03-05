# Benchmark Results

**Date**: 2026-03-05T09:14:53+05:30
**Machine**: Linux 6.17.0-14-generic aarch64
**CPU**: 
unknown

## Tool Versions

- yq: yq (https://github.com/mikefarah/yq/) version v4.52.4
- node: v24.13.1
- rustc: rustc 1.93.1 (01f6ddf75 2026-02-11)
- hyperfine: hyperfine 1.20.0

## Binary Sizes

| Contender | Size |
|-----------|------|
| bash+yq (statusman.sh) | 44583 bytes (script) + 12648610 bytes (yq binary) |
| optimized-bash | 7103 bytes (script) |
| node | 3955 bytes (script) + 684K (node_modules) |
| rust | 594544 bytes (binary) |

## Results

### Startup Overhead (--help)

| Contender | Mean | Stddev | Min | Max | Relative |
|-----------|------|--------|-----|-----|----------|
| bash+yq | 2.5 ms | 179.5 us | 2.2 ms | 3.5 ms | 1.0x |
| optimized-bash | 1.4 ms | 121.4 us | 1.2 ms | 1.9 ms | 1.8x |
| node | 12.4 ms | 619.8 us | 11.3 ms | 15.6 ms | 0.2x |
| rust | 196.7 us | 49.1 us | 128.3 us | 1.3 ms | 12.5x |

### progress-map (read)

| Contender | Mean | Stddev | Min | Max | Relative |
|-----------|------|--------|-----|-----|----------|
| bash+yq | 19.2 ms | 579.5 us | 18.0 ms | 21.3 ms | 1.0x |
| optimized-bash | 4.0 ms | 287.2 us | 3.4 ms | 5.7 ms | 4.8x |
| node | 14.0 ms | 650.0 us | 13.0 ms | 16.7 ms | 1.4x |
| rust | 258.6 us | 45.9 us | 189.7 us | 836.8 us | 74.3x |

### set-change-type (write)

| Contender | Mean | Stddev | Min | Max | Relative |
|-----------|------|--------|-----|-----|----------|
| bash+yq | 6.7 ms | 346.5 us | 5.9 ms | 8.7 ms | 1.0x |
| optimized-bash | 3.4 ms | 252.3 us | 3.0 ms | 4.8 ms | 2.0x |
| node | 14.5 ms | 685.1 us | 13.4 ms | 17.6 ms | 0.5x |
| rust | 301.7 us | 55.3 us | 222.3 us | 1.4 ms | 22.1x |

### finish (transition)

| Contender | Mean | Stddev | Min | Max | Relative |
|-----------|------|--------|-----|-----|----------|
| bash+yq | 37.8 ms | 861.5 us | 35.5 ms | 40.1 ms | 1.0x |
| optimized-bash | 7.2 ms | 507.0 us | 6.4 ms | 9.9 ms | 5.3x |
| node | 14.6 ms | 541.6 us | 13.4 ms | 16.7 ms | 2.6x |
| rust | 307.3 us | 56.3 us | 226.4 us | 1.7 ms | 123.0x |

## Summary

- **Startup**: fastest is **rust** (0.2 ms), 63x faster than slowest
- **progress-map (read)**: fastest is **rust** (0.3 ms), 74x faster than slowest
- **set-change-type (write)**: fastest is **rust** (0.3 ms), 48x faster than slowest
- **finish (transition)**: fastest is **rust** (0.3 ms), 123x faster than slowest

**Overall ranking** (wins across operations):

1. **rust**: 4 fastest

**Key takeaways**:

- Rust is the clear performance winner across all operations (sub-millisecond)
- Optimized bash is a viable middle ground (~2-5x faster than baseline, no new dependencies)
- Node is slower than baseline for simple operations due to V8 startup overhead (~13ms floor)
- The baseline bash+yq finish operation (38ms) shows the cumulative cost of repeated yq subprocess spawns

## Raw Data

JSON files in `src/benchmark/results/`:

- `finish.json`
- `progress-map.json`
- `set-change-type.json`
- `startup.json`
