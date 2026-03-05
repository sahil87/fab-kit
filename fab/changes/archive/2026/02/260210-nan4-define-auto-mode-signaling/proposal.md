# Proposal: Define auto-mode signaling mechanism for skill-to-skill invocation

**Change**: 260210-nan4-define-auto-mode-signaling
**Created**: 2026-02-10
**Status**: Draft

## Why

`fab-ff` invokes `fab-clarify` in "auto mode" between stage generations, but no mechanism is defined for how one skill signals mode to another. `fab-clarify.md` says mode is determined by "call context" — that `fab-ff` calling it internally triggers auto mode — but there is no concrete protocol for how the calling skill communicates this. This leaves the behavior entirely implicit and dependent on agent interpretation, which is unreliable. A defined signaling mechanism makes the contract explicit and testable.

## What Changes

- Define an explicit "call context" protocol in `_context.md` for skill-to-skill invocation, specifying how the calling skill signals mode (e.g., via a context variable or explicit instruction prefix)
- Update `fab-clarify.md` to document how it detects auto mode using the defined protocol
- Update `fab-ff.md` to use the defined protocol when invoking fab-clarify
- Update `fab-fff.md` if it also invokes fab-clarify internally

## Affected Docs

### New Docs
- None

### Modified Docs
- `fab-workflow/planning-skills`: Update to document the skill-to-skill invocation protocol
- `fab-workflow/clarify`: Update to document how auto mode is signaled

### Removed Docs
- None

## Impact

- `fab/.kit/skills/_context.md` — new section defining the skill invocation protocol
- `fab/.kit/skills/fab-clarify.md` — Mode Selection section updated with concrete detection logic
- `fab/.kit/skills/fab-ff.md` — auto-clarify invocations updated to use the protocol
- `fab/.kit/skills/fab-fff.md` — same updates if it invokes fab-clarify

## Open Questions

- None — the gap is clearly defined (missing protocol) and the solution space is narrow (define the protocol in `_context.md` where other shared conventions already live).

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Define protocol in `_context.md` rather than a separate file | `_context.md` already holds shared conventions (SRAD, confidence scoring, next steps) — this is the canonical location |
| 2 | Confident | Use an explicit instruction prefix pattern rather than a flag or env variable | Skills are markdown-only (constitution: Pure Prompt Play) — no flags or variables available; the calling skill must include explicit text in its prompt to the called skill |

2 assumptions made (2 confident, 0 tentative).
