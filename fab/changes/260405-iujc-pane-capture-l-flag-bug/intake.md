# Intake: Pane Capture -l Flag Bug

**Change**: 260405-iujc-pane-capture-l-flag-bug
**Created**: 2026-04-05
**Status**: Draft

## Origin

> debug: fab pane capture -l flag fails with "unknown flag" — check if _cli-fab skill documents -l incorrectly or if fab-go has a bug

User-reported: running `fab pane capture <pane> -l N` fails with `unknown flag: -l`. The `_cli-fab` skill documents `-l` as a valid short flag (type `int`, default `50`, description: "Number of lines to capture"). Either the Go implementation never registered `-l` as a short form, or it was registered but was later removed without updating the doc.

## Why

The `_cli-fab` skill is the authoritative CLI reference loaded by all skills and agents. When its documented interface diverges from the actual binary behavior, every operator and automation that calls `fab pane capture -l` breaks silently or fails with a confusing error. The divergence also erodes trust in the documentation generally.

If not fixed, the operator and any skill that calls `fab pane capture -l` (e.g., `/fab-operator`) will be broken or will need to omit the flag entirely, accepting the 50-line default.

## What Changes

### Investigation

Identify the root cause by reading `fab-go`'s pane capture command implementation:
- Check `src/cmd/pane.go` (or equivalent) for the `capture` subcommand's flag definitions
- Determine whether `-l` was ever registered as a short flag or only `--lines` (or another long form) exists

### Fix path A — Go binary bug

If `-l` is absent from the Go implementation but documented in `_cli-fab`:
- Add `-l` as the short form for the lines flag in `fab-go`'s `pane capture` command
- Add or update test coverage for the short flag
- No doc change needed

### Fix path B — documentation error

If the Go implementation uses a different flag name (e.g., `--lines`, `--count`) or has no short form:
- Update `src/kit/skills/_cli-fab.md` to reflect the actual flag name
- Update the `.claude/skills/_cli-fab/SKILL.md` deployed copy accordingly (via `fab sync` or direct edit)
- No Go change needed

### Constitution constraints

Per `fab/project/constitution.md`:
- Changes to the `fab` CLI binary **MUST** include corresponding test updates
- Changes to CLI command signatures **MUST** update `src/kit/skills/_cli-fab.md`

## Affected Memory

- `fab-workflow/kit-architecture.md`: (modify) Update pane capture flag documentation if CLI interface changes

## Impact

- `src/kit/skills/_cli-fab.md` — canonical skill source (may change)
- `fab-go` pane capture command implementation (may change)
- `fab-go` pane capture tests (may change if Go binary is fixed)
- `.claude/skills/_cli-fab/SKILL.md` — deployed copy (reflects whichever fix path is taken)

## Open Questions

- Does `fab-go` define `-l` as a short flag for pane capture, or only a long form?
- If the long form exists, what is its name (`--lines`, `--count`, or something else)?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Root cause is either missing short-flag registration in Go or incorrect doc — binary choice | Description explicitly frames it as one-or-the-other; both paths are straightforward fixes once investigated | S:65 R:70 A:55 D:60 |
| 2 | Certain | If Go binary changes, tests must be updated | Mandated by constitution: "Changes to the fab CLI MUST include corresponding test updates" | S:90 R:90 A:90 D:90 |
| 3 | Certain | `src/kit/skills/_cli-fab.md` must be updated if the CLI interface changes | Mandated by constitution: "MUST update src/kit/skills/_cli-fab.md with any new or changed command signatures" | S:90 R:90 A:90 D:90 |

3 assumptions (2 certain, 1 confident, 0 tentative, 0 unresolved).
