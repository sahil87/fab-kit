## Node/TypeScript Conventions

### Strict TypeScript Configuration
All projects MUST enable `strict: true` in `tsconfig.json`. The `any` type is prohibited — use `unknown` with type guards or explicit generic constraints instead. All functions MUST have explicit return type annotations.

### ESM Imports Only
Code MUST use ES module syntax (`import`/`export`) exclusively. CommonJS `require()` calls are prohibited in source code. The `tsconfig.json` MUST target ESM output (`"module": "ESNext"` or `"module": "NodeNext"`).

### No Implicit Dependencies
Every runtime dependency MUST be declared in `package.json`. Peer dependencies MUST specify version ranges. Dev dependencies used in CI (linters, test runners) MUST be pinned to exact versions to ensure reproducible builds.

### Error Handling Strategy
Code MUST NOT use bare `try/catch` blocks that swallow errors silently. Caught exceptions MUST be logged, re-thrown, or converted to typed error objects. Promise rejections MUST be handled — unhandled rejection listeners are not a substitute for proper error propagation.

### No Side Effects at Import Time
Module-level code MUST NOT perform I/O, modify global state, or trigger network requests on import. All initialization MUST happen in explicitly called functions or factory methods. This enables tree-shaking and deterministic test setup.

### Immutable by Default
Data structures SHOULD use `readonly` properties and `ReadonlyArray<T>` where mutation is not required. Functions SHOULD return new objects rather than mutating inputs. Mutable state MUST be confined to clearly scoped boundaries (e.g., class internals, reducer functions).

### No Magic Strings or Numbers
All configuration values, timeout durations, retry counts, and environment variable names MUST be defined as named constants. Inline string literals for keys, routes, or identifiers are prohibited — use a constants module or enum.

### Consistent Async Patterns
Code MUST NOT mix callbacks and Promises in the same module. All async operations MUST use `async`/`await` syntax. Callback-based APIs MUST be wrapped with `util.promisify` or equivalent before use.
