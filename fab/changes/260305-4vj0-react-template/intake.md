# Intake: React Project Template for Fab Kit

**Change:** 260305-4vj0-react-template
**Type:** feat
**Issue:** fa-4vj
**Date:** 2026-03-05

---

## Problem Statement

React projects have specific conventions around component architecture, state management, hooks usage, and testing patterns that go beyond general TypeScript rules. The Node/TypeScript template (fa-qg8) covers language-level strictness, but React projects need an additional layer of framework-specific guidance. Without it, agents produce inconsistent component patterns, misuse hooks, and skip accessibility considerations.

## Proposed Solution

Add a React-specific template that layers on top of the TypeScript template. Detection is via `react` in `package.json` dependencies. This is the first framework-level template, establishing the pattern for future framework templates (Next.js, Vue, etc.).

---

## What Changes

### A. New File: `fab/.kit/templates/constitutions/react.md`

A React-specific constitution fragment:

- **Functional components only.** No class components. Use function declarations (`function MyComponent()`) not arrow expressions for top-level components — this gives better stack traces and React DevTools names.
- **Hooks rules are non-negotiable.** No conditional hooks. No hooks in loops. No hooks in nested functions. ESLint `react-hooks/rules-of-hooks` must be enabled and passing.
- **No prop drilling beyond 2 levels.** If a prop passes through more than 2 intermediate components that don't use it, use context, composition (children/render props), or a state manager.
- **Component files export one component.** A file named `UserCard.tsx` exports `UserCard`. Co-located helpers/hooks are fine, but only one component per file boundary.
- **Explicit `key` prop usage.** Every mapped list must use a stable, unique key — never array index unless the list is static and never reorders. Keys must be explained in a comment if the choice isn't obvious.
- **No inline styles for layout.** Use CSS modules, Tailwind, or styled-components. Inline styles are permitted only for truly dynamic values (calculated positions, theme-derived colors).
- **Custom hooks for shared logic.** Any stateful logic used by 2+ components must be extracted into a `use*` hook. Hooks live in a `hooks/` directory or co-located with the feature.
- **Accessible by default.** All interactive elements must have accessible names (aria-label, aria-labelledby, or visible text). All images must have alt text. Form inputs must have associated labels. `eslint-plugin-jsx-a11y` must be enabled.
- **No `useEffect` for derived state.** If a value can be computed from props or other state, compute it during render (useMemo if expensive). `useEffect` is for synchronization with external systems, not state derivation.
- **Test components by behavior, not implementation.** Use React Testing Library. Assert on what the user sees/does (text content, clicks, form submissions), never on component internals (state values, hook calls).

### B. New File: `fab/.kit/templates/configs/react.yaml`

A React-specific config overlay:

```yaml
framework: react
source_paths:
  - "src/components/"
  - "src/hooks/"
  - "src/pages/"
  - "src/app/"

checklist:
  extra_categories:
    - build_clean
    - component_tests
    - a11y_lint_pass

stage_directives:
  spec:
    - "Identify which components are affected and their prop interfaces"
    - "Note any shared state or context changes"
  apply:
    - "Write component tests alongside component implementation"
    - "Ensure all interactive elements are keyboard accessible"
  review:
    - "Verify build passes (npm run build or equivalent)"
    - "Verify component tests pass"
    - "Verify eslint-plugin-jsx-a11y passes"
    - "Check for unnecessary re-renders (missing memo/useMemo/useCallback)"
```

### C. Modified: `fab/.kit/skills/fab-setup.md`

Extend framework detection (runs after language detection):

1. If TypeScript/Node detected AND `package.json` contains `"react"` in `dependencies`:
   - Apply React template on top of TypeScript template
   - Merge `templates/constitutions/react.md` under `## React Conventions` section
   - Merge `templates/configs/react.yaml` into config (additive to TS values)
2. Print: `"Detected React project. Applied React conventions to constitution and config."`

Detection is simple: read `package.json`, check if `dependencies` or `devDependencies` contains `react`. No JSON parsing needed — string match on `"react":` within the file is sufficient and avoids jq/node dependency.

### D. Modified: `fab/.kit/sync/2-sync-workspace.sh`

Add framework detection check:

1. If `package.json` contains `"react"` AND
2. `fab/project/constitution.md` does not contain `## React Conventions`
3. Print: `"React project detected but constitution lacks React conventions. Run /fab-setup --refresh to apply."`

---

## Impact Assessment

| Aspect | Detail |
|---|---|
| New files | 2 template files (~60-80 lines each) |
| Modified files | `fab/.kit/skills/fab-setup.md`, `fab/.kit/sync/2-sync-workspace.sh` |
| Breaking changes | None. Additive. Layers on existing TS template. |
| Risk | Low. Same pattern. React detection is simple string match. |
| Rollback | Remove template files and revert modifications. |

---

## Assumptions

| # | Assumption | SRAD | Rationale |
|---|---|---|---|
| A1 | `"react"` in package.json dependencies is a reliable React signal | S | Every React project lists react as a dependency. |
| A2 | React template layers on TypeScript template (not standalone) | R | Nearly all React projects use TypeScript. Pure JS React is increasingly rare. If detected without TS, apply only React conventions. |
| A3 | Next.js, Remix, Gatsby are React under the hood but may need their own templates later | A | These frameworks have additional conventions (file-based routing, server components). React template covers the shared base. |
| A4 | `## React Conventions` heading is the idempotency marker | S | Same pattern as Rust and TypeScript templates. |
| A5 | String match `"react":` in package.json is sufficient — no JSON parsing | R | Avoids requiring jq or node in the detection script. False positive risk is negligible (no non-React package would have `"react":` as a dependency key). |

---

## Open Questions

1. Should we detect Next.js (`"next"` in dependencies) and apply additional SSR/RSC conventions?
2. Should the React template assume Tailwind if `tailwind.config.*` exists, and add Tailwind-specific directives?
3. Should we detect the test runner (vitest/jest) and adjust the `test_pass` directive command?
