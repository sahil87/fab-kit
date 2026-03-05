## Rust Conventions

### No Unsafe Code Without Justification
Code MUST NOT contain `unsafe` blocks unless each is accompanied by an inline `// SAFETY:` comment explaining the invariant being upheld. Unjustified `unsafe` blocks SHALL be rejected during review.

### Error Handling Strategy
Library crates MUST use `thiserror` for structured, typed error definitions. Application-level code (binaries, CLI entry points) SHOULD use `anyhow` for ergonomic, context-rich error propagation. Mixing the two in the same crate is prohibited.

### No Unwrap in Library Code
`.unwrap()` MUST NOT appear in library code. It is permitted only in tests and binary entry points (`main.rs`, `build.rs`). Library code SHALL propagate errors with `?` or return `Result`. `.expect()` with a descriptive message is permitted for cases where the invariant is provably upheld (e.g., regex compilation of a literal).

### Pure Core Pattern
Domain logic crates MUST have zero I/O dependencies. All file system, network, and database access SHALL live in adapter crates that depend on the core, never the reverse. This enables deterministic testing of business logic without mocks.

### State Machine Coverage
Every state machine transition MUST have a dedicated unit test asserting the before-state, trigger, and after-state. Invalid transitions MUST also be tested to verify they produce the correct error. Untested transitions SHALL be flagged during review.

### Prefer Standard Library
Code SHOULD prefer `std` implementations over external crates when the standard library provides a reasonable solution. External crates are justified when they offer meaningful ergonomic or correctness advantages (e.g., `regex`, `serde`, `chrono`). Every dependency in `Cargo.toml` should be justifiable.

### Command Execution Discipline
Shell-outs via `std::process::Command` MUST check exit status, capture stderr, and return structured errors on failure. Fire-and-forget command execution is prohibited.

### No Hardcoded Thresholds
All timeouts, retry counts, intervals, and numeric thresholds MUST come from configuration (config file, environment variable, or builder pattern default). Magic numbers in source code SHALL be rejected during review.
