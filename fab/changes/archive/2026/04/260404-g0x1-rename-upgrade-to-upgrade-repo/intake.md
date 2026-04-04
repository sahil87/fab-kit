# Intake: Rename upgrade to upgrade-repo

**Change**: 260404-g0x1-rename-upgrade-to-upgrade-repo
**Created**: 2026-04-05
**Status**: Draft

## Origin

> PR review feedback on README.md flagged the `fab upgrade` vs `fab update` confusion. Discussion explored renaming options: `fab use`, `fab pin`, `fab self-update`, `fab repo-upgrade`, `fab upgrade-repo`. User chose `fab upgrade-repo` — verb-first, target-second pattern, immediately distinct from `fab update`.

Key decisions from conversation:
- `fab update` stays unchanged — it's a pattern the user follows across other CLI tools (self-updates the binary via Homebrew)
- `fab upgrade-repo` chosen over `fab use` because users don't always know the latest version — they want "give me the latest for this repo"
- `fab upgrade-repo` chosen over `fab repo-upgrade` for verb-first convention

## Why

1. **Problem**: `fab upgrade` and `fab update` are easily confused by newcomers. Both sound like "get a newer thing" but target different things — `upgrade` changes the project's pinned kit version in `config.yaml`, while `update` updates the fab-kit binary itself via Homebrew.
2. **Consequence**: A newcomer who runs `fab update` expecting to upgrade their project's kit version gets the binary updated instead (and vice versa). Silent wrong-target confusion.
3. **Approach**: Rename `fab upgrade` to `fab upgrade-repo` — the `-repo` suffix makes the target explicit. The verb-first pattern (`upgrade-repo` not `repo-upgrade`) follows CLI convention.

## What Changes

### CLI subcommand rename

Rename the `upgrade` subcommand to `upgrade-repo` in the Go source code:

- `src/go/fab-kit/internal/upgrade.go` — the `Upgrade()` function (logic unchanged, only the command registration)
- Command registration in `src/go/fab-kit/cmd/` (wherever `upgrade` is wired to its handler)
- Help text and usage strings

The function signature and behavior remain identical — this is purely a rename of the CLI entry point.

### Documentation updates

All references to `fab upgrade` become `fab upgrade-repo`:

- `README.md` — "Updating from a previous version" section (line ~130)
- `docs/memory/fab-workflow/distribution.md`
- `docs/memory/fab-workflow/kit-architecture.md`
- `docs/memory/fab-workflow/migrations.md`
- `docs/memory/fab-workflow/configuration.md`
- `src/kit/skills/fab-setup.md` — any references to the upgrade command

### Existing change artifacts (no update needed)

Files in `fab/changes/` that reference `fab upgrade` are historical artifacts of past changes and should NOT be updated — they reflect what was true at the time.

## Affected Memory

- `fab-workflow/distribution`: (modify) update `fab upgrade` references to `fab upgrade-repo`
- `fab-workflow/kit-architecture`: (modify) update `fab upgrade` references to `fab upgrade-repo`
- `fab-workflow/migrations`: (modify) update `fab upgrade` references to `fab upgrade-repo`
- `fab-workflow/configuration`: (modify) update `fab upgrade` references to `fab upgrade-repo`

## Impact

- **CLI**: Users who have `fab upgrade` in scripts or muscle memory will get an unknown-command error. Low blast radius — this is an infrequently used command.
- **Docs**: All user-facing documentation referencing `fab upgrade` must be updated.
- **Skills**: `fab-setup.md` references the upgrade command in migration guidance.
- **Tests**: Any tests exercising the `upgrade` subcommand need updating.
- **CONTRIBUTING.md**: May reference the upgrade workflow.

## Open Questions

(None — all decisions resolved in discussion.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename to `upgrade-repo`, not other candidates | Discussed — user explicitly chose `upgrade-repo` over `use`, `pin`, `repo-upgrade` | S:95 R:90 A:95 D:95 |
| 2 | Certain | `fab update` remains unchanged | Discussed — user confirmed it's a cross-tool pattern they follow | S:95 R:90 A:95 D:95 |
| 3 | Certain | Historical change artifacts not updated | Constitution principle: change artifacts are transient records of what was true at the time | S:90 R:95 A:90 D:95 |
| 4 | Confident | No backward-compatibility alias for `fab upgrade` | Low usage frequency, clean break preferred over maintaining aliases | S:60 R:75 A:70 D:70 |
| 5 | Confident | Go function name `Upgrade()` stays as-is | Internal naming doesn't face users; renaming internals adds churn without user benefit | S:65 R:85 A:80 D:75 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
