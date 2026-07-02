# Intake: Gitignore-aware sync dedup

**Change**: 260625-mqiq-gitignore-aware-sync-dedup
**Created**: 2026-06-25

## Origin

> Make `fab sync`'s `.gitignore` dedup recognize equivalent ignore forms and never clobber a downstream negation.
>
> `fab sync` merges the scaffold fragment `src/kit/scaffold/fragment-.gitignore` into the project's `.gitignore` via `lineEnsureMerge` in `src/go/fab-kit/internal/scaffold.go`. The fragment ships the canonical entry `/.claude` (alongside `/.agents`, `/.cursor`, `/.opencode`, `/.codex`, `/.gemini`). The "already present?" dedup check is literal string equality, so sync only treats `.claude` as already-ignored when the `.gitignore` contains the exact string `/.claude`. Semantically-equivalent forms — `/.claude/`, `/.claude/*`, `.claude`, `.claude/`, `.claude/*` — do not match, so sync concludes the entry is missing and appends `/.claude`. This silently clobbers a user's downstream negation (the user-reported bug).

Promptless dispatch (`/fab-proceed` create-new): one-shot, no conversation. Synthesized from a detailed bug report with an explicit fix direction and named guardrails.

## Why

1. **The problem.** `lineEnsureMerge` (`src/go/fab-kit/internal/scaffold.go`, function at lines ~266–348) decides whether a fragment line is "already present" in the destination using literal string equality at scaffold.go:~313–318:

   ```go
   for _, dl := range destLines {
       if strings.TrimRight(dl, "\r") == entry {
           found = true
           break
       }
   }
   ```

   So sync treats `.claude` as already-ignored only when the `.gitignore` contains the exact string `/.claude`. The semantically-equivalent forms `/.claude/`, `/.claude/*`, `.claude`, `.claude/`, `.claude/*` do not match, so sync concludes the entry is missing and **appends `/.claude`**.

2. **Why this is worse than a redundant duplicate.** Users who want to re-include a subdirectory of `.claude` must write the directory-contents form `/.claude/*` followed by a negation such as `!/.claude/commands/`. This is forced by git's rule: *you cannot re-include a path if a parent directory is excluded.* `/.claude` (or `/.claude/`) excludes the directory itself, so git never descends into it and `!/.claude/commands/` is dead; `/.claude/*` excludes the contents but not the directory, so the negation works. When sync appends `/.claude` after the user's `/.claude/*` + `!/.claude/commands/` block, the later `/.claude` line wins (git's last-match rule) and **silently re-excludes the whole directory, defeating the user's negation.** This is the user-reported bug — not a cosmetic duplicate, but a silent behavioral regression in the user's working tree.

3. **What happens if we don't fix it.** Every `fab sync` after a user has set up a `.claude/*` + negation block re-clobbers their re-inclusion, so committed agent files (e.g. shared `/.claude/commands/`) silently fall out of version control. The user must re-fix `.gitignore` after every sync, with no signal that sync caused it.

4. **Why this approach over alternatives.** A gitignore-aware "already covered" predicate is the minimal, targeted fix: it stops the spurious append without changing what the scaffold emits into a fresh file. Alternatives rejected: (a) changing the shipped canonical entry to `/.claude/*` — out of scope and changes the default behavior for every fresh project; (b) full gitignore-spec parsing — over-engineered for a six-entry directory fragment; (c) suppressing the append whenever any `.claude`-ish line exists — too broad and would mask legitimately-different patterns.

## What Changes

### 1. Gitignore-aware "already covered" dedup in `lineEnsureMerge`

Replace the literal `==` dedup (scaffold.go:~314) with a predicate `gitignoreCovers(existingLine, entry)` that, for a directory-style entry like `/.claude`, treats the variant set as already covering the entry:

```
{ /.claude, /.claude/, /.claude/*, .claude, .claude/, .claude/* }
```

That is: leading slash optional, optional trailing `/` or `/*`. If any destination line matches a variant of the fragment entry, the entry is considered already covered and is **not** appended.

Concretely, for an entry `E` (e.g. `/.claude`), normalize by stripping a leading `/` and any trailing `/` or `/*` to a core token (`.claude`); a destination line `D` covers `E` when `D` normalizes to the same core token under the same rules. Keep the existing `strings.TrimRight(dl, "\r")` carriage-return handling.

<!-- clarified: 2026-06-25 — confirmed directory-token-only: a deeper nested path like `/.claude/commands/` does NOT normalize to the core token and therefore does NOT count as covering `/.claude` (such a file still gets `/.claude` appended). Conservative — avoids false "already covered" matches. -->
Normalization is anchored at the directory token only: a deeper nested path like `/.claude/commands/` does **not** reduce to the core token `.claude`, so it does not count as covering `/.claude`.

### 2. Guardrail A — scope semantic matching to `.gitignore` only

`lineEnsureMerge` is also used for `.envrc` (fragment `src/kit/scaffold/fragment-.envrc`, e.g. `export IDEAS_FILE=fab/backlog.md`), which must keep **strict literal equality** — environment-variable export lines are not gitignore patterns and have no variant set. Gate the semantic comparison on the destination being a `.gitignore`: use the `label` parameter (the third argument to `lineEnsureMerge`, already `".gitignore"` / `".envrc"` at the call sites) or the destination basename. When the label/basename is not `.gitignore`, fall back to the existing literal `==` check unchanged.

### 3. Guardrail B — negation is a hard stop

<!-- clarified: 2026-06-25 — confirmed "suppress on ANY negation": if any `!.../.claude/...` line is present, never append a broader `.claude` ignore, regardless of whether a `/.claude/*` exclusion precedes it. A lone negation is a git no-op, so suppressing never breaks a correct file and protects the user even if they add the exclusion later. -->

If the destination already contains a negation under the path (a line matching `!/.claude/...` or `!.claude/...` for the entry's core token), **never** append a broader ignore that would override it — even beyond the variant set. This holds whether or not a broader `/.claude/*` exclusion precedes the negation: the mere presence of a `!.../.claude/...` line suppresses the append. This protects the exact user-reported scenario: a `/.claude/*` + `!/.claude/commands/` block must never gain a trailing `/.claude` from sync. The negation check is the binding guardrail; the variant-set "covered" check is the common case. (A present negation implies the user has deliberately structured their ignores; appending any broader `.claude` ignore is the harmful action to prevent.)

### 4. Tests (test-alongside, Constitution VII + code-quality)

A `.go` behavior change ships tests in the same change. Current coverage in `src/go/fab-kit/internal/scaffold_test.go` (`TestLineEnsureMerge_CreateNew`, `_AppendNew`, `_SkipComments`, `_PropagatesWriteError`) and `src/go/fab-kit/internal/sync_integration_test.go` only exercises exact-match dedup. Add fixtures for:

- **Variant coverage** — a `.gitignore` already containing `/.claude/`, `/.claude/*`, `.claude`, `.claude/`, or `.claude/*` does NOT gain an appended `/.claude` (one case per variant, or table-driven).
- **The negation case** — a `.gitignore` containing `/.claude/*` followed by `!/.claude/commands/` is left unchanged by sync (no trailing `/.claude` appended); the negation survives.
- **`.envrc` strict equality preserved** — semantic matching must NOT leak to `.envrc`; an `.envrc` whose lines differ literally still appends (regression guard for Guardrail A).
- **Genuine-miss still appends** — a `.gitignore` with none of the variants still gets `/.claude` appended (the original happy path is unchanged).

Scope the Go test run to the affected package (`src/go/fab-kit/internal`) first; widen only if cross-cutting.

## Affected Memory

- `distribution/setup.md`: (modify, conditional) The scaffold-source table (lines ~95–96) describes the `.gitignore`/`.envrc` entries as a "Line-ensuring merge from `{cache}/kit/scaffold/fragment-.gitignore`", and line ~103 (jznd) describes `lineEnsureMerge`'s failure-surfacing behavior. If the dedup semantics ("line-ensuring merge") are documented as plain equality anywhere, the prose needs a note that `.gitignore` dedup is now gitignore-aware (variant-set coverage + negation hard-stop) while `.envrc` stays literal. During hydrate, verify whether this prose actually states the equality semantics; if it only names the merge at a high level, no edit is required — implementation-only behavior refinements don't always need a memory edit.

<!-- assumed: setup.md is the only memory file with scaffold/gitignore-merge prose; kit-architecture.md was grepped and carries no fragment/dedup prose. Hydrate confirms scope. -->

## Impact

- `src/go/fab-kit/internal/scaffold.go` — `lineEnsureMerge` dedup logic; replace the literal `==` at ~line 314 with the gitignore-aware predicate, gated on the `.gitignore` label/basename, plus the negation hard-stop. Likely a new unexported helper (e.g. `gitignoreCovers`).
- `src/go/fab-kit/internal/scaffold_test.go` — add variant-coverage, negation, and `.envrc`-strict-equality regression cases alongside the existing `TestLineEnsureMerge_*` tests.
- `src/go/fab-kit/internal/sync_integration_test.go` — extend the end-to-end sync fixture (already exercises a `.claude/` fragment form at line ~75) to cover the negation-survival case through a full `fab sync`.
- **Not touched**: `src/kit/skills/_cli-fab.md` — this is NOT a command-signature change (no new/changed `fab` subcommand or flag), so the CLI⇒docs constraint does not trigger. No SPEC-mirror sweep (no `src/kit/skills/*.md` edit). No migration (no user-data restructuring — `.gitignore` content is the user's, not a managed config/`.status.yaml`/archive artifact).
- **Scaffold default unchanged**: `src/kit/scaffold/fragment-.gitignore` keeps its canonical `/.claude` entry; the fix is the dedup recognizing existing equivalent forms, not changing what we emit into a fresh file.

## Open Questions

_Both resolved via `/fab-clarify` on 2026-06-25 — see § Clarifications._

- ~~Exact normalization boundary: should `gitignoreCovers` treat a deeper path like `/.claude/commands/` as covering `/.claude`?~~ **Resolved: No** — directory-token-only; a deeper path does not count as covering.
- ~~Should the negation hard-stop suppress the append when a negation exists but no broader exclusion precedes it?~~ **Resolved: Yes** — suppress on any `!.../.claude/...` negation, regardless of a preceding exclusion.

## Clarifications

### Session 2026-06-25

| Q | Question | Answer |
|---|----------|--------|
| 1 | Negation hard-stop scope — when `!.../.claude/...` is present but no broader `/.claude/*` exclusion precedes it, still suppress the append? | **Suppress on any negation** — the presence of any `!.../.claude/...` line suppresses a broader `.claude` append, regardless of a preceding exclusion (resolves Unresolved row 9) |
| 2 | Should `gitignoreCovers` treat a deeper path like `/.claude/commands/` as covering `/.claude`? | **Directory token only** — a deeper path does NOT count as covering; such a file still gets `/.claude` appended (resolves Tentative row 8) |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace literal `==` dedup with a gitignore-aware coverage predicate in `lineEnsureMerge` | The report names the exact location (scaffold.go:~314), the bug, and the fix direction; codebase confirms the literal check | S:95 R:80 A:95 D:90 |
| 2 | Certain | Scope semantic matching to `.gitignore`; `.envrc` keeps strict equality (Guardrail A) | Explicit guardrail in the report; the `label` arg already distinguishes `.gitignore` vs `.envrc` at both call sites | S:95 R:85 A:95 D:90 |
| 3 | Certain | Negation is a hard stop: never append a broader ignore when `!.../.claude/...` is present (Guardrail B) | Explicit guardrail; it is the binding protection for the user-reported scenario | S:90 R:80 A:90 D:85 |
| 4 | Certain | Variant set is `{ /.claude, /.claude/, /.claude/*, .claude, .claude/, .claude/* }` (leading-slash optional, trailing `/` or `/*`) | Enumerated verbatim in the report | S:95 R:80 A:95 D:95 |
| 5 | Certain | Ship Go tests in the same change (Constitution VII, test-alongside): variant coverage, negation survival, `.envrc` strict-equality regression, genuine-miss-still-appends | Constitution + code-quality require it; existing `TestLineEnsureMerge_*` give the pattern | S:90 R:85 A:100 D:90 |
| 6 | Certain | Scaffold default stays `/.claude`; `_cli-fab.md` and SPEC mirrors untouched; no migration | Report marks all three as non-goals; no command signature or skill file changes | S:95 R:80 A:95 D:90 |
| 7 | Confident | `distribution/setup.md` prose only needs editing if it states the dedup as plain equality; confirm during hydrate | Grep shows setup.md names the "line-ensuring merge" at a high level; kit-architecture.md carries no dedup prose. Memory edit may be a no-op | S:70 R:85 A:75 D:75 |
| 8 | Certain | Normalization is anchored at the directory token only — a deeper nested path like `/.claude/commands/` does NOT count as covering `/.claude` | Clarified — user confirmed directory-token-only (deeper paths still get `/.claude` appended). Reversible code predicate, straightforward to implement, ambiguity removed by the user's answer | S:95 R:80 A:90 D:90 |
| 9 | Certain | Suppress the append whenever any `!.../.claude/...` negation is present — regardless of whether a broader `/.claude/*` exclusion precedes it | Clarified — user decided "suppress on any negation": a lone negation is a git no-op, so suppressing never breaks a correct file and protects intent even if the exclusion is added later. Reversible, straightforward, fully disambiguated | S:95 R:80 A:90 D:85 |

9 assumptions (8 certain, 1 confident, 0 tentative, 0 unresolved).
