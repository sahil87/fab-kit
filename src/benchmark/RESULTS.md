# Benchmark Results

**Date**: 2026-03-05T09:35:56+05:30
**Machine**: Linux 6.17.0-14-generic aarch64
**CPU**: 
unknown

## Tool Versions

- yq: yq (https://github.com/mikefarah/yq/) version v4.52.4
- node: v24.13.1
- go: go version go1.26.0 linux/arm64
- rustc: rustc 1.93.1 (01f6ddf75 2026-02-11)
- hyperfine: hyperfine 1.20.0

## Binary Sizes

| Contender | Size |
|-----------|------|
| bash+yq (statusman.sh) | 44583 bytes (script) + 12648610 bytes (yq binary) |
| optimized-bash | 7049 bytes (script) |
| node | 3913 bytes (script) + 684K (node_modules) |
| go | 3519399 bytes (binary) |
| rust | 594544 bytes (binary) |

## Results

### Startup Overhead (--help)

| Contender | Mean | Stddev | Min | Max | Relative |
|-----------|------|--------|-----|-----|----------|
| bash+yq | 2.5 ms | 174.1 us | 2.2 ms | 3.5 ms | 1.0x |
| optimized-bash | 1.4 ms | 150.8 us | 1.2 ms | 2.9 ms | 1.8x |
| node | 12.6 ms | 514.5 us | 11.5 ms | 14.2 ms | 0.2x |
| go | 542.6 us | 97.2 us | 386.0 us | 2.8 ms | 4.6x |
| rust | 182.5 us | 42.3 us | 116.6 us | 828.5 us | 13.7x |

### progress-map (read)

| Contender | Mean | Stddev | Min | Max | Relative |
|-----------|------|--------|-----|-----|----------|
| bash+yq | 19.5 ms | 833.9 us | 18.3 ms | 25.0 ms | 1.0x |
| optimized-bash | 4.1 ms | 315.3 us | 3.6 ms | 7.7 ms | 4.8x |
| node | 14.2 ms | 572.7 us | 12.9 ms | 16.1 ms | 1.4x |
| go | 690.3 us | 97.5 us | 510.6 us | 1.9 ms | 28.2x |
| rust | 263.2 us | 48.4 us | 193.2 us | 691.3 us | 74.0x |

### set-change-type (write)

| Contender | Mean | Stddev | Min | Max | Relative |
|-----------|------|--------|-----|-----|----------|
| bash+yq | 6.8 ms | 351.4 us | 6.1 ms | 8.0 ms | 1.0x |
| optimized-bash | 3.5 ms | 329.3 us | 3.1 ms | 5.7 ms | 1.9x |
| node | 14.8 ms | 635.8 us | 13.7 ms | 17.5 ms | 0.5x |
| go | 802.0 us | 104.6 us | 597.4 us | 2.4 ms | 8.4x |
| rust | 331.8 us | 68.7 us | 239.1 us | 1.3 ms | 20.4x |

### finish (transition)

| Contender | Mean | Stddev | Min | Max | Relative |
|-----------|------|--------|-----|-----|----------|
| bash+yq | 39.4 ms | 1.7 ms | 36.6 ms | 45.1 ms | 1.0x |
| optimized-bash | 7.4 ms | 494.5 us | 6.4 ms | 9.8 ms | 5.3x |
| node | 15.4 ms | 611.8 us | 13.7 ms | 17.9 ms | 2.6x |
| go | 801.8 us | 94.7 us | 597.4 us | 1.9 ms | 49.1x |
| rust | 359.5 us | 86.5 us | 236.7 us | 1.9 ms | 109.5x |

## Summary

- **Startup**: fastest is **rust** (0.2 ms), 69x faster than slowest
- **progress-map (read)**: fastest is **rust** (0.3 ms), 74x faster than slowest
- **set-change-type (write)**: fastest is **rust** (0.3 ms), 45x faster than slowest
- **finish (transition)**: fastest is **rust** (0.4 ms), 109x faster than slowest

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
