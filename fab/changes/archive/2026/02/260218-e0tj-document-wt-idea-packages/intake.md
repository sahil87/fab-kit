# Intake: Document wt and idea packages

**Change**: 260218-e0tj-document-wt-idea-packages
**Created**: 2026-02-18
**Status**: Draft

## Origin

> Discussion about whether fab-help.sh should cover wt-* and idea commands, and whether dedicated docs/specs pages should exist. Agreed on two actions: (1) add a compact "Packages" footer section to fab-help.sh that lists wt-* and idea without mixing into skill categories, and (2) create a single docs/specs/packages.md covering concepts and workflows, not per-command reference (since inline help already covers that).

## Why

The wt-* commands (wt-create, wt-list, wt-open, wt-delete, wt-init) and the idea command ship as packages within fab-kit, but they are invisible to users who discover the toolkit through `/fab-help`. Currently:

1. **Discoverability gap**: A user running `/fab-help` sees only fab pipeline skills. They have no indication that worktree management and backlog utilities exist — unless they happen to read the README or browse the directory structure.
2. **Conceptual gap**: There's no documentation explaining how these packages integrate with the fab pipeline (e.g., wt-create enables the assembly-line pattern described in docs/specs/assembly-line.md; idea feeds `/fab-new` via backlog IDs). The README mentions them in a table but doesn't connect the dots.
3. **No dedicated spec**: docs/specs/ thoroughly covers the fab pipeline but has zero coverage of packages. This is the only major fab-kit capability without a spec page.

If left unaddressed, users will underuse these tools or reinvent equivalent workflows manually.

## What Changes

### 1. fab-help.sh — Add "Packages" footer section

Add a new `PACKAGES` section at the end of `fab/.kit/scripts/fab-help.sh` (after the `TYPICAL FLOW` section) that lists the bundled packages:

```
PACKAGES

  wt-create, wt-list, wt-open, wt-delete, wt-init   Git worktree management
  idea                                                 Per-repo backlog (fab/backlog.md)

  Run <command> help for details.
```

Key decisions:
- **Static block, not dynamic scanning** — packages are stable and few; dynamic discovery adds complexity for no benefit
- **Not mixed into skill categories** — packages are standalone CLI tools, not Claude Code skills invoked via `/`
- **One-liner descriptions** — pointing users to `<command> help` for details avoids duplication
- **Placed after TYPICAL FLOW** — visually separated from the skills section, clearly supplementary

### 2. docs/specs/packages.md — New spec page

Create a single `docs/specs/packages.md` covering both packages at the concept/workflow level:

Structure:
- **Overview** — what packages are, how they relate to the fab pipeline
- **wt (Worktree Management)** — concept, the 5 commands with one-liner descriptions, integration with fab (assembly-line pattern, `/fab-new` → `wt-create` flow), common workflows (exploratory worktree, branch-based worktree, cleanup)
- **idea (Backlog Management)** — concept, CRUD commands with one-liner descriptions, integration with fab (`idea` → `fab/backlog.md` → `/fab-new` via backlog ID), common workflows (capture idea, triage and start work, mark done)
- **Package architecture** — where packages live (`fab/.kit/packages/{name}/`), bin/ and lib/ convention, how they're distributed

What this page is NOT:
- Not a per-command reference (that's what `<command> help` is for)
- Not a tutorial (the README handles onboarding)
- Not auto-generated (consistent with constitution principle VI — specs are human-curated)

### 3. docs/specs/index.md — Add packages.md entry

Add a row to the specs index table for the new page.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Add packages section covering the package directory structure and distribution model

## Impact

- `fab/.kit/scripts/fab-help.sh` — new static section appended
- `docs/specs/packages.md` — new file
- `docs/specs/index.md` — new row in table
- `docs/memory/fab-workflow/kit-architecture.md` — updated during hydrate

## Open Questions

None — scope and approach were agreed in the preceding discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Static block in fab-help.sh, not dynamic scanning | Packages are stable (2 packages, unlikely to change frequently); dynamic scanning adds complexity for negligible benefit | S:90 R:90 A:85 D:90 |
| 2 | Certain | Packages section placed after TYPICAL FLOW | Visually separates packages from skills; follows the script's existing top-down structure (overview → commands → flow → packages) | S:85 R:95 A:90 D:85 |
| 3 | Certain | Single packages.md, not per-package spec pages | Agreed in discussion; avoids sync burden since inline help is the reference; consistent with the "concepts + workflows" approach | S:95 R:85 A:90 D:90 |
| 4 | Confident | Include package architecture section in packages.md | Useful for contributors to understand where packages live and how they're structured; not explicitly discussed but natural fit | S:60 R:90 A:75 D:80 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
