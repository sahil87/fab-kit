# Intake: Node/TypeScript Project Template for Fab Kit

**Change:** 260305-qg80-node-typescript-template
**Type:** feat
**Issue:** fa-qg8
**Date:** 2026-03-05

---

## Problem Statement

Fab Kit scaffolds generic constitution and config files with no language-specific conventions. For Node.js/TypeScript projects — the most common stack in web development — teams must manually add strict typing rules, ESM conventions, testing standards, and linting directives. The Rust template (fa-wh2) established the pattern; this change extends it to the Node/TypeScript ecosystem.

## Proposed Solution

Add a Node/TypeScript-specific template layer so that `/fab-setup` auto-detects TS projects and merges TS conventions into the constitution and config.

---

## What Changes

### A. New File: `fab/.kit/templates/constitutions/typescript.md`

A TypeScript-specific constitution fragment containing:

- **Strict mode required.** `tsconfig.json` must include `"strict": true`. No `skipLibCheck` as a workaround for type errors.
- **No `any` type.** Use `unknown` for truly unknown types, then narrow with type guards. `any` silently disables type checking and defeats the purpose of TypeScript.
- **Explicit return types on exported functions.** Internal helpers may use inference, but all public API functions must declare their return type.
- **ESM imports only.** Use `import`/`export` syntax, never `require()`. Configure `"type": "module"` in package.json or `"module": "ESNext"` in tsconfig.
- **No default exports.** Named exports make refactoring safer, improve tree-shaking, and prevent import name mismatches.
- **Error handling via typed errors.** Never `throw` raw strings. Define error classes or use a Result pattern. `catch` blocks must type-narrow the error.
- **No mutation of function parameters.** Treat inputs as readonly. Use spread/destructuring to create new objects rather than modifying arguments.
- **Dependency discipline.** Justify every new dependency. Prefer Node built-ins (`node:fs`, `node:path`, `node:crypto`) over npm packages for standard operations. Pin exact versions in package.json.
- **No `console.log` in library code.** Use a structured logger (pino, winston) or accept a logger parameter. `console.log` is permitted only in CLI entry points and scripts.

### B. New File: `fab/.kit/templates/configs/typescript.yaml`

A TypeScript-specific config overlay:

```yaml
language: typescript
source_paths:
  - "src/"
  - "lib/"

checklist:
  extra_categories:
    - tsc_clean
    - eslint_pass
    - test_pass

stage_directives:
  spec:
    - "Reference package.json for dependency constraints"
    - "Reference tsconfig.json for module and target settings"
  review:
    - "Verify tsc --noEmit passes with zero errors"
    - "Verify eslint passes with no warnings"
    - "Verify test suite passes (npm test or equivalent)"
    - "Check for any usage that bypasses strict mode"
```

### C. Modified: `fab/.kit/skills/fab-setup.md`

Extend the language detection step (added by Rust template):

1. Existing: `Cargo.toml` → Rust
2. **New:** `package.json` + `tsconfig.json` both exist → **TypeScript project**
3. **New:** `package.json` exists without `tsconfig.json` → **Node.js project** (apply a subset — dependency discipline, ESM, no console.log — skip strict typing rules)
4. For TypeScript detection:
   - Merge `templates/constitutions/typescript.md` into `fab/project/constitution.md` under `## TypeScript Conventions` section
   - Merge `templates/configs/typescript.yaml` into `fab/project/config.yaml`
5. Print: `"Detected TypeScript project. Applied TypeScript conventions to constitution and config."`

### D. Modified: `fab/.kit/sync/2-sync-workspace.sh`

Add detection check (same pattern as Rust):

1. If `tsconfig.json` exists at project root, AND
2. `fab/project/constitution.md` does not contain `## TypeScript Conventions`
3. Print suggestion: `"TypeScript project detected but constitution lacks TypeScript conventions. Run /fab-setup --refresh to apply."`

---

## Impact Assessment

| Aspect | Detail |
|---|---|
| New files | 2 template files (~60-80 lines each) in `fab/.kit/templates/` |
| Modified files | `fab/.kit/skills/fab-setup.md`, `fab/.kit/sync/2-sync-workspace.sh` |
| Breaking changes | None. Additive only. |
| Risk | Low. Same pattern as Rust template. |
| Rollback | Remove template files and revert skill/sync modifications. |

---

## Assumptions

| # | Assumption | SRAD | Rationale |
|---|---|---|---|
| A1 | `tsconfig.json` at repo root is a reliable signal for TypeScript | S | Standard TS convention. Projects using TS always have tsconfig. |
| A2 | `package.json` without `tsconfig.json` indicates plain Node.js | R | Could also be a JS project using Babel/Flow, but Node.js conventions still apply. |
| A3 | TypeScript template subsumes Node.js conventions | S | TS projects are Node projects. All Node conventions apply, plus strict typing. |
| A4 | Polyglot detection deferred to later iteration | A | A project with both Cargo.toml and package.json could exist (Tauri, wasm-bindgen). Layering multiple templates is future work. |
| A5 | `## TypeScript Conventions` heading is the idempotency marker | S | Same pattern as Rust template. |

---

## Open Questions

1. Should we detect the package manager (npm/yarn/pnpm/bun) and adjust stage directives accordingly (e.g., `bun test` vs `npm test`)?
2. Should we detect monorepo tooling (turborepo, nx, lerna) and adjust `source_paths`?
3. For plain Node.js (no TypeScript), should we scaffold a separate constitution or just a reduced version of the TS one?
