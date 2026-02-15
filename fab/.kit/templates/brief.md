# Brief: {CHANGE_NAME}

**Change**: {YYMMDD-XXXX-slug}
**Created**: {DATE}
**Status**: Draft

## Origin

<!-- The user's raw input/prompt that initiated this change. Preserves original language
     and intent for downstream spec generation. -->

> {USER_INPUT}

## Why

<!-- Explain the motivation. What problem does this solve? Why now? 1-3 sentences. -->

## What Changes

<!-- Be specific about new capabilities, modifications, or removals. Use bullets. -->

## Affected Memory

<!-- Which memory files will be created, modified, or removed by this change.
     Use kebab-case identifiers matching docs/memory/ paths. Mark each with (new), (modify), or (remove).
     Only list if spec-level behavior changes — implementation-only changes don't need memory updates. -->

- `{domain}/{file-name}`: ({new|modify|remove}) {brief description}

## Impact

<!-- Affected code areas, APIs, dependencies, systems. Helps scope the spec. -->

## Open Questions

<!-- Clarifying questions the agent couldn't resolve from context alone.
     SRAD handles prioritization at spec generation time — no need for explicit
     blocking/deferred labels here. Just list the questions. -->

- {question}

## Assumptions

<!-- STATE TRANSFER: This table is the sole continuity mechanism between the brief-stage
     agent and the spec-stage agent. Pipeline stages may execute in separate agent contexts
     with no shared memory — this table is what gives downstream agents visibility into
     what was decided, assumed, or left open. Every row must be substantive.

     All four SRAD grades (Certain, Confident, Tentative, Unresolved) are recorded.
     Scores column is required for every row.
     Unresolved rows must include status context in Rationale (e.g., "Asked — user undecided"). -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | {Certain|Confident|Tentative|Unresolved} | {decision summary} | {why this grade} | S:nn R:nn A:nn D:nn |

{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved).
