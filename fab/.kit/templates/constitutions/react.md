## React Conventions

### Functional Components Only
Components MUST be written as function components. Class components SHALL NOT be used in new code. Existing class components SHOULD be migrated to function components during modification.

### Hooks Rules
Hooks MUST only be called at the top level of a component or custom hook — never inside loops, conditions, or nested functions. Custom hooks MUST start with the `use` prefix. Violating hooks rules SHALL be rejected during review.

### No Prop Drilling
Components more than two levels deep MUST NOT receive props solely to pass them further down. Use React Context, composition patterns, or state management libraries to avoid prop drilling. Intermediate components that only forward props SHALL be refactored.

### Proper Key Usage
List-rendered elements MUST use stable, unique keys derived from the data (e.g., IDs). Array indices MUST NOT be used as keys when the list can be reordered, filtered, or modified. Missing or unstable keys SHALL be flagged during review.

### Immutable State Updates
State MUST be updated immutably. Direct mutation of state objects or arrays is prohibited. Use spread operators, `Array.prototype.map`, `Array.prototype.filter`, or libraries like Immer for complex updates.

### Effect Cleanup
Effects that create subscriptions, timers, or event listeners MUST return a cleanup function. Missing cleanup in effects with side effects SHALL be flagged during review as a potential memory leak.

### Component Composition Over Configuration
Components SHOULD favor composition (children, render props, compound components) over configuration via numerous boolean props. Components with more than five boolean props SHOULD be refactored into composable parts.

### No Inline Object/Array Literals in JSX Props
Object or array literals MUST NOT be passed inline as JSX props in performance-sensitive components (those rendered in lists or on frequent re-renders). Use `useMemo`, `useCallback`, or module-level constants to maintain referential stability.
