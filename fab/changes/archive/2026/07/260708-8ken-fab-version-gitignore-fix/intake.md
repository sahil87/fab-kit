# Intake: Un-gitignore fab/.fab-version

**Change**: 260708-8ken-fab-version-gitignore-fix
**Created**: 2026-07-08

## Origin

User-reported field failure (screenshot of `wt create` on the fab-kit repo, 2026-07-08 21:42) +
same-session diagnosis, immediately after Change 3 of the config-upgrade effort (j0qm, PR #476,
released v2.15.1) shipped. User approved the three-item fix proposal verbatim and directed: work on
a new branch created on top of latest origin/main.

> `wt create` → "Init (fab sync)" → `ERROR: no fab version found in fab/.fab-version or
> config.yaml. Run 'fab init' to set one` — while the main worktree's `cat fab/.fab-version`
> happily prints `2.15.1`.

**Diagnosis (verified in-session)**: `fab/.fab-version` exists on disk but is **gitignored** —
`git check-ignore -v fab/.fab-version` → `.gitignore:29:.fab-*`. A gitignore pattern containing no
slash matches at ANY directory depth, so the `.fab-*` line (added 2026-03-11 for the root runtime
files `.fab-status.yaml`/`.fab-backend`/`.fab-runtime.yaml`) also swallows `fab/.fab-version`. The
scaffold ships the same pattern to every user project (`src/kit/scaffold/fragment-.gitignore:2`).
`fab/.kit-migration-version` does not match the pattern, which is why the "sibling precedent"
never surfaced this. Design decision 1 (config-upgrade effort) says `.fab-version` is
**committed** — silently defeated everywhere.

## Why

1. **Pain point**: post-migration, a repo's version source is `fab/.fab-version` only (the
   config.yaml `fab_version:` key is parked/removed by `fab config upgrade` — main's config.yaml
   carries only the parked comment `#   fab_version: 2.13.6` at line 65). Because the file can
   never be committed, every fresh checkout — new worktree, new clone, CI — has NO version source,
   and fab-kit's `readFabVersion` both-absent path fail-louds: `wt init`/`fab sync` die exactly as
   in the screenshot.
2. **Consequence if unfixed**: every fab-managed repo that upgrades to 2.15.x and migrates loses
   the ability to spin worktrees/clones without manually re-running `fab init`; teammates and CI
   see a bricked repo. The failure is silent until the first fresh checkout (`git status` shows
   nothing — ignored ≠ untracked).
3. **Approach**: negation lines (`!fab/.fab-version`) at the three seams — this repo, the shipped
   fragment, and a migration for already-shipped repos — plus a fail-open stamp-path warning so
   this class (written-but-ignored version file) can never go quiet again. Chosen over renaming
   the file (churn on a just-shipped design) or narrowing `.fab-*` to `/.fab-*` (would change
   coverage for nested worktree roots and the fragment's existing dedup semantics; higher risk).

## What Changes

### 1. This repo's own `.gitignore` + commit the file

- Add `!fab/.fab-version` immediately after the `.fab-*` line (`.gitignore:29`).
- Commit `fab/.fab-version` containing `2.15.1\n` (already written to the working tree this
  session; matches the deployed kit on main). Both ride in this change's PR.

### 2. Scaffold fragment gains the negation (all future + syncing projects)

Add `!fab/.fab-version` to `src/kit/scaffold/fragment-.gitignore` directly under `.fab-*`.
Mechanics verified against the merge code (`src/go/fab-kit/internal/scaffold.go`):

- The fragment is applied by `lineEnsureMerge` via `scaffoldTreeWalk` (`scaffold.go:100-162`),
  called from `sync.go:89` — i.e. on **every `fab sync`**, not only at init. Existing repos
  therefore self-heal their `.gitignore` on their next `fab upgrade-repo`/`fab sync`.
- `gitignoreIsDirectoryToken` (`scaffold.go:296`) is false both for `.fab-*` (contains `*`) and
  for `!fab/.fab-version` (no leading `/`, no trailing `/`), so BOTH use strict literal dedup:
  the negation is appended once if absent, and the Guardrail-B negation hard-stop
  (`gitignoreHasNegation`, `scaffold.go:324`) is never consulted for either line — adding the
  negation cannot suppress the `.fab-*` ensure. Pin this with fragment-merge tests (append-once,
  idempotent re-sync, no hard-stop interaction, ordering under `.fab-*`).

### 3. Migration for already-shipped repos: verify + commit

New `src/kit/migrations/2.15.1-to-2.15.2.md` (constitution: user-data restructure — it edits the
user's `.gitignore` and commits a file). Because the fragment merge self-heals the negation at
sync time (see 2), the migration's job is verification + the commit the binary cannot do:

- **Pre-check** (sentinel/idempotency): skip when `git check-ignore -q fab/.fab-version` fails
  (not ignored) AND `git ls-files --error-unmatch fab/.fab-version` succeeds (already committed).
  Skip silently when not a git repo or `fab/.fab-version` absent (pre-2.15 repos hit this
  migration only after the 2.14.0-to-2.15.0 one).
- **Changes**: append `!fab/.fab-version` after the `.fab-*` line (or at end) of `.gitignore` if
  still ignored (covers repos whose sync predates the fixed fragment); `git add .gitignore
  fab/.fab-version` and commit.
- **Verification**: `git check-ignore fab/.fab-version` exits non-zero;
  `git ls-files --error-unmatch fab/.fab-version` exits 0; re-run is a no-op.

### 4. Stamp-path hardening (fail-open warning)

`stampFabVersion` (`src/go/fab-kit/internal/init.go:92-98`, callers `Init` + `Upgrade` at
`upgrade.go:44`) — after a successful write, when inside a git work tree and
`git check-ignore -q fab/.fab-version` reports ignored, print a fail-open warning to stderr
(e.g. `fab: warning: fab/.fab-version is gitignored — commit it so worktrees/clones/CI see the
version (add '!fab/.fab-version' to .gitignore)`). Never an error; silent when git is absent or
not a repo (the rk fail-silent discipline precedent). Tests: warning fires on an ignored path,
silent when negated, silent outside a git repo.

### 5. Version + docs obligations

- Bump `src/kit/VERSION` `2.15.1` → `2.15.2` (patch — pure fix); migration name matches.
- `docs/specs/config.md`: the `.fab-version` "committed" prose gains the gitignore caveat +
  negation requirement (the design said "committed"; now the mechanism guarantees it).
- No CLI signature change ⇒ no `_cli-fab.md` command change; add the warning line to its
  `upgrade-repo`/init output description only if that section enumerates output (apply verifies).
- Go tests alongside every code change (constitution VII); skill↔SPEC mirror sweep only if a
  `src/kit/skills/*.md` file is touched (none expected).
- Memory updates at hydrate (see Affected Memory).

## Affected Memory

- `distribution/migrations`: (modify) new `2.15.1-to-2.15.2` entry (verify+commit shape, fragment
  self-heal rationale)
- `distribution/kit-architecture`: (modify) fragment gains the negation line; stampFabVersion
  check-ignore warning
- `distribution/setup`: (modify) fragment-merge notes (mqiq) — negation lines are non-directory
  tokens under strict literal dedup
- `_shared/configuration`: (modify) `.fab-version` "committed" caveat — the `.fab-*` ignore class
  and its negation

## Impact

- **Repo root**: `.gitignore` (+1 line), `fab/.fab-version` (new committed file, `2.15.1\n`)
- **Kit content** (`src/kit/`): `scaffold/fragment-.gitignore` (+1 line), new
  `migrations/2.15.1-to-2.15.2.md`, `VERSION` bump
- **fab-kit Go** (`src/go/fab-kit/`): `internal/init.go` (stamp warning helper),
  `internal/scaffold_test.go` (fragment negation merge tests), init/upgrade warning tests
- **Docs**: `docs/specs/config.md` caveat; memory at hydrate
- **User-facing**: repos self-heal on next sync; the migration commits the file; fresh
  worktrees/clones/CI regain a version source

## Open Questions

*(none — fix approach user-approved in this conversation; merge-mechanics risk verified against
`scaffold.go` in-session)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix = negation lines at three seams + stamp warning + commit the file (no rename, no `.fab-*` narrowing) | User approved the numbered proposal verbatim this session | S:95 R:75 A:90 D:90 |
| 2 | Certain | Fragment merge runs on every sync, so existing repos self-heal; migration's job reduces to verify + commit | Verified: `sync.go:89` → `scaffoldTreeWalk` → `lineEnsureMerge` unconditionally per sync | S:90 R:80 A:95 D:90 |
| 3 | Certain | `!fab/.fab-version` in the fragment is safe under the merge: strict literal dedup (not a directory token), Guardrail-B hard-stop not consulted for it or for `.fab-*` | Verified against `gitignoreIsDirectoryToken`/`gitignoreHasNegation` (`scaffold.go:296,324`); tests will pin it | S:85 R:80 A:90 D:85 |
| 4 | Confident | Stamp warning lives in/next to `stampFabVersion`, stderr `fab: warning:`, git-presence-gated, never an error | Fail-open discipline precedent (upgrade-repo config step, rk detection); exact wording/helper shape decided at apply | S:70 R:85 A:85 D:75 |
| 5 | Confident | Version/migration = patch `2.15.1-to-2.15.2` | Pure fix, no schema change; migration naming rule FROM=released TO=next | S:75 R:85 A:85 D:80 |
| 6 | Certain | This repo's `fab/.fab-version` committed as `2.15.1\n` in this PR | Matches main's deployed kit (screenshot + `fab --version`); file already staged-able once negation lands | S:90 R:85 A:95 D:90 |
| 7 | Confident | Migration Pre-check also silently skips non-git repos and repos without `fab/.fab-version` | Migration ordering guarantees 2.14.0-to-2.15.0 ran first; non-git repos cannot commit anyway | S:65 R:85 A:80 D:75 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
