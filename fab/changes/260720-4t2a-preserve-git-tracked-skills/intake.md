# Intake: Preserve Git-Tracked Skills

**Change**: 260720-4t2a-preserve-git-tracked-skills
**Created**: 2026-07-20

## Origin

> cleanStaleSkills deletes any .claude/skills (and sibling agent dirs) entry not in fab-kit's own
> canonical skill list, including custom skills a consuming repo commits to its own git history.
> Fix: check git-tracking before removing; preserve tracked entries, still clean up genuinely
> stale/untracked ones.

One-shot input (via `fab change new --log-args`), but backed by a full conversational
investigation that preceded this intake: a real-world repro was traced end-to-end (a Terraform
infra monorepo's `wt create` — which runs `fab sync` as its worktree-init script — silently
deleted a custom, git-committed Claude Code skill, `archive-dormant-repos`, on every new
worktree). Root cause was confirmed by disassembling the installed `fab-kit` binary (`strings`
revealed `internal.cleanStaleSkills`) and cross-checked against `git ls-tree`/`.gitignore` in the
consuming repo to rule out a git-config issue on the consumer's side.

A fix was implemented and PR'd (sahil87/fab-kit#510) before this intake was created. The
maintainer (`sahil-noon`) responded with two comments: "Run this through the fab-kit pipeline"
and "in other repos we already do this via the untouched `.claude/commands/` folder" (pointing at
`sahil87/shll`'s `.claude/commands/` as the existing convention). This intake exists to (a) run
the already-implemented fix through the proper pipeline per that request, and (b) record the
resolution of the `.claude/commands/` alternative, which was investigated and found insufficient
— see Why.

## Why

**Problem**: `cleanStaleSkills` (`src/go/fab-kit/internal/skills.go`) treats "not in fab-kit's own
canonical skill list" as synonymous with "stale, safe to delete." It has no notion of a skill a
consuming project authored and committed itself. Every `fab sync` — including the sync `wt
create` runs automatically as its init script — silently deletes such a skill from disk, with no
warning, surfacing only as a mysterious `deleted:` line in `git status` in the consuming repo.

**Consequence if unfixed**: any fab-kit consumer that commits a custom Claude Code / Codex /
Gemini / OpenCode skill loses it on every sync. The failure is silent and recurring (not a
one-time surprise) — every new worktree re-triggers it — and the fix required tracing through the
compiled binary's symbol table to even diagnose, since nothing in the consuming repo's own
config or `.gitignore` was wrong.

**Why this approach over the `.claude/commands/` alternative**: the maintainer's suggested
escape hatch (`.claude/commands/`, confirmed via the `sahil87/shll` example — genuinely untouched
by `cleanStaleSkills`, none of its four agent configs reference that path) only substitutes for
skills that need **zero** encapsulation and **zero** auto-invocation:

1. **Encapsulation** — `.claude/commands/` is flat (one file per command); `.claude/skills/<name>/`
   is a directory, so a skill can ship supporting resources (references, scripts, templates)
   alongside `SKILL.md`, not squeezed into a single file.
2. **Auto-invocation** — Claude Code's model-driven skill discovery only scans
   `.claude/skills/*/SKILL.md`. Files under `.claude/commands/` are explicit-invocation only.
   `archive-dormant-repos` happens to already be explicit-only (`disable-model-invocation`,
   added upstream in fab-kit `d5fdcf2`) — for that *one* skill, `.claude/commands/` would
   incidentally work. But `cleanStaleSkills`'s bug isn't specific to that skill: any git-tracked
   skill relying on auto-invocation would be deleted identically, with no escape hatch available
   for it. Fixing `cleanStaleSkills` itself is the general fix; redirecting individual skills to
   `.claude/commands/` is a per-skill workaround that doesn't scale to the auto-invocation case.

## What Changes

### `cleanStaleSkills` (`src/go/fab-kit/internal/skills.go`)

New helper:

```go
func isGitTracked(repoRoot, relPath string) bool {
	cmd := exec.Command("git", "ls-files", "--error-unmatch", "--", relPath)
	cmd.Dir = repoRoot
	return cmd.Run() == nil
}
```

`cleanStaleSkills` (both the directory-format and flat-format branches) now skips deletion when
`isGitTracked` returns true for the candidate path, computed via `filepath.Rel(repoRoot, ...)`.
Fails open (git absent, or `repoRoot` not a git work tree → `false`) so the historical cleanup
behavior for non-git contexts is unchanged; the loosening only ever applies to entries git can
positively confirm as tracked. Untracked/stale entries are still removed exactly as before.

### Tests (`src/go/fab-kit/internal/skills_test.go`)

Two new tests mirroring the existing `TestCleanStaleSkills_Directory` / `_Flat` style:
`TestCleanStaleSkills_Directory_PreservesGitTrackedCustomSkill` and
`TestCleanStaleSkills_Flat_PreservesGitTrackedCustomFile` — each asserts a git-tracked custom
entry survives while an untracked stale entry in the same directory is still removed. Verified
red→green: reverted the fix with the new tests in place (both failed for the expected reason —
custom skill deleted), then restored the fix (both passed).

### End-to-end verification (already performed, not part of this pipeline run)

Built the actual `fab-kit` binary from the fix branch and ran `fab-kit sync` twice against a temp
repo shaped like the real reproduction case (git-tracked custom skill + one canonical stock
skill): first run preserved the custom skill and deployed the canonical one; second run (with an
added *untracked* leftover skill) removed the untracked stale entry while still preserving the
git-tracked custom skill.

## Affected Memory

- `distribution/kit-architecture.md`: (modify) `cleanStaleSkills`'s behavior description (the
  "clean stale entries" line in the `fab-kit sync` 6-step pipeline writeup) needs a clause noting
  that git-tracked entries are now preserved rather than swept as stale.

## Impact

- `src/go/fab-kit/internal/skills.go` — `cleanStaleSkills`, new `isGitTracked`
- `src/go/fab-kit/internal/skills_test.go` — two new tests
- `docs/memory/distribution/kit-architecture.md` — description update (hydrate stage)
- No config schema change, no new dependency, no CLI surface change

## Open Questions

(none — the one open question from the maintainer's review, whether `.claude/commands/` obviates
this fix, was resolved in-thread on PR #510: it doesn't, for the auto-invocation case; see Why.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Root cause is `cleanStaleSkills` deleting any skill-dir entry absent from fab-kit's own canonical list, with no git-tracking check | Confirmed by reading the actual source (`skills.go`) and cross-checked against the compiled binary's symbol table (`strings` → `internal.cleanStaleSkills`); reproduced live via `wt create` | S:95 R:90 A:95 D:95 |
| 2 | Certain | Fix is `isGitTracked` via `git ls-files --error-unmatch`, fail-open on git-absent/non-repo | Matches an existing pattern already in this file's package (`warnIfFabVersionIgnored`'s `git check-ignore -q` fail-open shape); red→green tested; E2E tested with the real binary | S:90 R:85 A:95 D:90 |
| 3 | Certain | Keep the `cleanStaleSkills` fix rather than redirecting `archive-dormant-repos` to `.claude/commands/` | Explicit maintainer discussion on PR #510 — `.claude/commands/` verified (via the `sahil87/shll` example) to be untouched by sync, but structurally incapable of auto-invocation; user (contributor) explicitly decided to keep the general fix for that reason | S:90 R:80 A:85 D:90 |
| 4 | Confident | Update `distribution/kit-architecture.md`'s `cleanStaleSkills` description during hydrate, no other memory files affected | No config/CLI-surface change, so no other documented behavior is touched; scoped to the one sentence describing the cleanup step | S:80 R:85 A:75 D:80 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
