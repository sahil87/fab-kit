# Intake: Fix Stale Shell-Script References After Go Binary Conversion

**Change**: 260311-i7it-fix-stale-shell-script-refs
**Created**: 2026-03-11
**Status**: Draft

## Origin

> Internal consistency check (`/internal-consistency-check`) revealed 16 inconsistencies (8 critical, 6 minor) across specs and memory following the conversion of `wt` and `idea` from shell scripts to Go binaries. The conversion landed in changes `260310-qbiq-go-wt-binary` and `260310-pl72-port-idea-to-go`, and the shell-to-Go migration for core lib scripts landed in `260305-u8t9-clean-break-go-only`. Documentation was not fully updated to match.

## Why

1. **Agents get wrong context**: Memory files (`kit-scripts.md`, `kit-architecture.md`) describe deleted shell scripts as current implementation. Agents consulting these files will generate specs and plans based on a shell-script architecture that no longer exists — leading to incorrect file paths, wrong command signatures, and invalid assumptions about `yq` dependencies.

2. **Users get wrong instructions**: `packages.md` tells users that wt/idea are "plain shell scripts" at `fab/.kit/packages/`. Users following this will look for shell files in an empty directory. `naming.md` has copy-paste errors pointing `idea` at `fab/.kit/packages/wt/`.

3. **Confidence erosion**: Stale docs undermine trust in the documentation-as-source-of-truth principle (Constitution §II). If agents can't trust memory/specs, the entire Fab workflow's value proposition degrades.

## What Changes

### 1. Rewrite `docs/specs/packages.md`

The spec currently describes wt and idea as "plain shell scripts" in `fab/.kit/packages/`. Rewrite to reflect:

- Both are **Go binaries** at `fab/.kit/bin/wt` and `fab/.kit/bin/idea`
- `wt` uses subcommands (`wt create`, `wt list`, etc.) — not hyphenated separate executables (`wt-create`, `wt-list`)
- `wt pr` was **dropped** (replaced by `/git-pr`)
- `idea` also available as `fab idea` via the dispatcher
- The `idea` shell package at `fab/.kit/packages/idea/bin/idea` is retained for rollback safety but the Go binary is the preferred path
- `fab/.kit/packages/` directory structure: only `idea` shell package remains (for rollback), `wt` package was removed entirely
- Remove references to `lib/wt-common.sh` (deleted shared shell library)
- Update PATH setup explanation: `env-packages.sh` adds `$KIT_DIR/bin` to PATH (making `fab`, `wt`, `idea` available), then iterates `packages/*/bin` for any remaining shell packages

### 2. Fix `docs/specs/naming.md`

- **Line 42**: Change `wt-create` (`fab/.kit/packages/wt/bin/wt-create`) → `wt create` (subcommand of `fab/.kit/bin/wt`)
- **Line 66**: Change `idea` command (`fab/.kit/packages/wt/`) → `idea` command (`fab/.kit/bin/idea` — backlog management)

### 3. Rewrite `docs/memory/fab-workflow/kit-scripts.md`

The entire file documents 7 deleted shell scripts (resolve.sh, statusman.sh, changeman.sh, archiveman.sh, logman.sh, calc-score.sh, preflight.sh) removed in change `260305-u8t9-clean-break-go-only`. Specific stale sections:

- **Lines 1-5**: Title "Kit Scripts Reference" and intro referencing shell scripts
- **Lines 17-26**: Call graph showing inter-script dependencies (monolithic Go binary has no inter-script deps)
- **Lines 28-48**: Argument resolution for deleted scripts
- **Lines 50-72**: Stage state machine with yq-based transitions and shell atomicity
- **Lines 74-109**: History logging referencing `logman.sh`

Rewrite as a Go binary command reference documenting `fab resolve`, `fab status`, `fab log`, `fab change`, `fab score` — or delete if `_scripts.md` already covers this adequately (evaluate overlap with `fab/.kit/skills/_scripts.md` before deciding).

### 4. Remove stale sections from `docs/memory/fab-workflow/kit-architecture.md`

Lines 126-166 contain detailed sections for 6 deleted `lib/` shell scripts:

- `lib/statusman.sh` (lines ~126-137) → now `fab status`
- `lib/logman.sh` (lines ~139-146) → now `fab log`
- `lib/calc-score.sh` (lines ~143-146) → now `fab score`
- `lib/changeman.sh` (lines ~147-156) → now `fab change`
- `lib/archiveman.sh` (lines ~158-166) → now `fab change archive/restore`
- References to `preflight.sh` → now `fab preflight`

Remove these sections and replace with a brief note that these operations are now handled by Go binary subcommands (with cross-reference to `_scripts.md` for command details).

Also fix **line 186** (`env-packages.sh` description) to clarify that `wt` is a binary in `bin/`, not discoverable via `packages/*/bin` iteration.

### 5. Terminology alignment

Across all touched files, replace "shell script" / "package" terminology with "Go binary" / "compiled binary" when referring to `wt` and `idea`. The term "package" should only be used for the legacy `idea` shell package retained for rollback.

## Affected Memory

- `fab-workflow/kit-scripts`: (remove) Delete — entirely obsolete, `_scripts.md` is the canonical command reference
- `fab-workflow/kit-architecture`: (modify) Add cross-reference to `fab/.kit/skills/_scripts.md` as the canonical CLI reference
- `fab-workflow/kit-architecture`: (modify) Remove stale lib/ script sections, fix env-packages description

## Impact

- `docs/specs/packages.md` — full rewrite
- `docs/specs/naming.md` — two line fixes
- `docs/memory/fab-workflow/kit-scripts.md` — delete, add canonical-source cross-ref in `kit-architecture.md`
- `docs/memory/fab-workflow/kit-architecture.md` — section removal + fixes
- No code changes — purely documentation
- No template changes
- No skill file changes

## Open Questions

- ~~Should `kit-scripts.md` be rewritten as a Go binary reference or deleted entirely?~~ **Resolved**: Delete it. `fab/.kit/skills/_scripts.md` is the canonical CLI reference — add a cross-reference from `kit-architecture.md`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | wt is a Go binary at `fab/.kit/bin/wt` | Discussed — verified by consistency check, confirmed as Mach-O executable | S:95 R:90 A:95 D:95 |
| 2 | Certain | idea is a Go binary at `fab/.kit/bin/idea` | Discussed — verified by consistency check, confirmed as Mach-O executable | S:95 R:90 A:95 D:95 |
| 3 | Certain | 7 lib/ shell scripts were deleted in 260305-u8t9 | Discussed — changelog confirms deletion, `lib/` contains only env-packages.sh and frontmatter.sh | S:95 R:90 A:95 D:95 |
| 4 | Certain | `wt pr` subcommand was dropped | Discussed — memory changelog states "wt pr dropped entirely (overlaps /git-pr)" | S:90 R:85 A:90 D:95 |
| 5 | Certain | idea shell package retained at packages/idea/ for rollback | Discussed — memory changelog confirms "shell package retained for rollback safety" | S:90 R:85 A:90 D:90 |
| 6 | Certain | wt uses subcommands not hyphenated executables | Discussed — Go binary verified: `wt create` not `wt-create` | S:95 R:90 A:95 D:95 |
| 7 | Confident | packages.md needs full rewrite, not incremental fixes | Discussed — structural mismatch too deep for line edits (location, invocation model, architecture all wrong) | S:80 R:75 A:80 D:75 |
| 8 | Certain | kit-scripts.md should be deleted, not rewritten | Clarified — user confirmed `_scripts.md` is the canonical source; add cross-ref from kit-architecture.md | S:95 R:80 A:95 D:95 |
<!-- clarified: kit-scripts.md deletion — user confirmed _scripts.md is canonical, cross-ref from kit-architecture.md -->

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
