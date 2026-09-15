# Intake: Per-Change Base Branch — Record `base_branch` in `.status.yaml` and Read It on Every Base-Relative Surface

**Change**: 260915-vy27-per-change-base-branch
**Created**: 2026-09-15

## Origin

Conversational — created by `/fab-proceed`'s create-new dispatch (`_intake` Create-Intake Procedure, `{questioning-mode} = promptless-defer`; no questions asked, open points deferred as Unresolved rows). The user accepted the one-change recommendation by invoking `/fab-proceed`.

> **Trigger** (report from another agent session): "The PR's Meta Impact block is measured against main, not gatekeeper-sim, because `fab pr-meta` has no base flag" — a stacked PR whose Impact numbers were inflated by the parent branch's diff.
>
> **Audit finding**: fab has no concept of a per-change base branch. No `.status.yaml` field records it, and no command accepts one except `fab impact <base> <head>`, which no skill wires. Every consumer derives the base from the repo default branch; the Go side goes further and hardcodes `origin/main` then `origin/master`, ignoring `origin/HEAD`. Five sites share the root cause (two Go, three skills). Not affected: `/git-pr-review` (works from `gh pr view`, which knows the PR's real base) and the operator's Dependency Resolution step 0 (resolves the default branch properly).
>
> **Decided approach**: one change rather than five patches — record a `base_branch` field in `.status.yaml`, set at branch creation (default: the resolved repo default branch; the operator's `stacked-prs` mode sets it explicitly to the dependency's branch), have `fab pr-meta`, `WriteTrueImpact`, the `_review.md` review dispatch, `/fab-adopt`, and `/git-pr` (`gh pr create --base`) read it, unify the two Go merge-base helpers into one shared helper that honors `origin/HEAD`, ship a migration for the schema addition, and fail open to the resolved default branch when the field is absent.
>
> **Rejected**: threading a `--base` flag through the four surfaces separately — each caller must then remember to pass it, and the `stacked-prs` operator path (`gh pr edit <pr> --base <dep-branch>` after creation, "/git-pr itself is unchanged and mode-unaware") already shows the drift that causes.

## Why

**Problem.** A change's diff, Impact numbers, and review scope are all "everything since the merge-base with the base branch". fab has no per-change notion of that base — every surface assumes the repo default branch. For a stacked change (branch created off a dependency's branch, PR targeting it), that assumption is wrong on every surface at once:

- `fab pr-meta` (PR `## Meta` Impact block) counts the parent branch's lines as this change's impact.
- `WriteTrueImpact` at `fab status finish` writes an inflated `true_impact` block into `.status.yaml`; `fab change list --show-stats` displays it.
- The review worker (`_review.md`) diffs `<merge-base(default)>...HEAD`, so it re-reviews the parent's already-reviewed diff and judges plan conformance against the wrong delta.
- `/fab-adopt` reconstructs intake/plan from a diff that absorbs the parent's work.
- `/git-pr` creates the PR against the repo default; the operator has to retarget it afterwards with `gh pr edit --base`, which fixes only the PR target and none of the four surfaces above.

A second, independent defect rides along: the two Go helpers hardcode `origin/main` → `origin/master` and ignore `origin/HEAD`. On a repo whose default branch is `develop` (or anything else), `fab pr-meta` silently emits no Impact block and `fab status finish` silently skips `true_impact` — while the skills resolve the default branch correctly via the g8st chain (`origin/HEAD` → `gh repo view` → main/master probe). Go and skills disagree about what "the base" is.

**Consequence of not fixing.** Stacked-PR work (the operator's `stacked-prs` merge mode) ships PRs whose Meta block lies about impact, persists wrong `true_impact` data into change history, and burns review cycles on the parent's diff. Non-`main`/`master` repos get no Impact data at all. Each surface that grows later will re-derive the base its own way and drift again.

**Why this approach.** A single recorded fact (`base_branch` in `.status.yaml`) read by every consumer is the owner-not-pointer shape this repo already prefers (code-quality.md § Anti-Patterns: stating an owned rule and pointing at it; duplicating utilities). Per-surface `--base` flags were rejected: the caller has to know and remember the base at every call site, and the operator's post-hoc `gh pr edit --base` retarget is exactly the drift that produces. Unifying the two Go helpers removes a near-duplicate utility (code-quality.md anti-pattern) and fixes the `origin/HEAD` bug once. Fail-open to the resolved default branch means non-stacked work sees zero behavior change and existing changes need no data rewrite.

## What Changes

### 1. `.status.yaml` schema — new optional `base_branch` field

Add a top-level optional scalar `base_branch` to `.status.yaml`:

```yaml
id: vy27
name: 260915-vy27-per-change-base-branch
created: 2026-09-15T…
created_by: sahil87
change_type: feat
base_branch: main            # NEW — optional; absent ⇒ consumers resolve the repo default branch
issues: []
progress:
  …
```

- Go: `statusfile.StatusFile` gains `BaseBranch string \`yaml:"base_branch,omitempty"\`` (drop-when-empty round-trip, modeled on `summary` / `change_type_source`), with the matching `syncToRaw` read case and write-time `insertKey` so the key is inserted into an existing document that lacks it (mz4q F07 sparse-key insertion — without this the field would be silently dropped on save).
- Template: `$(fab kit-path)/templates/status.yaml` carries **no placeholder** — a comment line in the style of the existing `# true_impact: lazily created …` line (`# base_branch: written when the change's branch is created (no placeholder here).`). `fab change new` runs before any branch exists, so seeding a value there would be a guess.
- Value: the value stored is the branch **name** as the remote knows it (e.g. `main`, `260914-abcd-parent-change`); consumers prefix `origin/` when they need the remote-tracking ref. *(Whether to store a plain name or a remote ref is a deferred decision — see Assumptions; plain name is the working default because `gh pr create --base` / `gh pr edit --base` take a plain name and every skill's `default_branch` variable already holds one.)*
- `fab status refresh` does **not** touch `base_branch` — it is not artifact-derived (hooks-may-enhance-never-own: refresh recomputes only `change_type`, `confidence`, and `plan.*` from `intake.md`/`plan.md`).
- `validate-status-file` accepts the new optional key.

### 2. Write path — `fab status set-base-branch` / `get-base-branch`

Skills never hand-edit `.status.yaml` (flock-serialized writes, `fab status` is the sole writer), so the field needs a verb pair modeled on `set-summary` / `get-summary`:

| Subcommand | Usage | Notes |
|------------|-------|-------|
| `set-base-branch` | `set-base-branch <change> <branch>` | Writes `base_branch` under the status flock (`withStatusLock` → `status.SetBaseBranch`). Non-empty required; existence on the remote is **not** validated (the operator may record a dependency branch that has not been pushed yet). |
| `get-base-branch` | `get-base-branch <change> [--json]` | Prints the recorded value, **empty line when absent** (graceful absence — callers fall back to the default-branch chain). `--json` → `{"base_branch":"main"}` (object-wrapped, additive; absent → `{"base_branch":""}`), joining the read-only `--json` query surface. |

`_cli-fab.md` § fab status table gains both rows; the `--json` paragraph's enumeration of read-only query subcommands grows from nine to ten (`get-base-branch`). Go tests: cobra `Use` prefix checks in `cmd/fab/status_test.go` (same pattern as `set-summary`), a mutator test in `internal/status/mutators_test.go`, a statusfile round-trip (absent → set → save → reload; absent key inserted; empty value dropped) in `internal/statusfile/statusfile_test.go`, and a `status_json_test.go` case for `get-base-branch --json`.

### 3. Who writes it — branch creation (`/git-branch` Step 4, `/fab-new` Step 11) and the operator

`base_branch` is written **when the change's branch is created**, by the twin branch-creation tables in `git-branch.md` Step 4 and `fab-new.md` Step 11 (kept in sync per their existing `<!-- Keep these cases in sync -->` comments):

- On every **create or rename** action (`git checkout -b` / `git branch -m`) — and, for completeness, on the `checkout --track origin/{name}` remote-only case — the skill records the base:
  ```bash
  # default: the branch we are branching FROM, if it is the repo default branch; otherwise the resolved default branch
  base_branch="{--base argument if given}"
  [ -n "$base_branch" ] || base_branch=$(git symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|^origin/||')
  [ -n "$base_branch" ] || base_branch=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name 2>/dev/null)
  [ -n "$base_branch" ] || base_branch=$(git rev-parse --verify -q refs/remotes/origin/main >/dev/null && echo main || echo master)
  fab status set-base-branch "{id}" "$base_branch"
  ```
  The three-line chain is the existing g8st default-branch convention already inlined in `git-pr.md`, `fab-adopt.md`, and `fab-operator.md` (memory: it is deliberately inlined per consumer, not a `_` helper).
- **Already-on-target / checked-out (existing local branch) cases**: write the field only if absent (idempotent — Constitution III; a re-run never clobbers an operator-set value).
- `git-branch.md` § Key Properties row **"Modifies `.status.yaml`? No"** becomes **"Yes — writes `base_branch` via `fab status set-base-branch` (only when absent, or when `--base` is given)"**; `docs/specs/glossary.md` (`/git-branch` "never modifies fab state") and `docs/specs/skills.md` § `/git-branch` Key properties are swept to match.
- **Operator `stacked-prs` mode** (`fab-operator.md` § Same-repo resolution (`stacked-prs` mode)): after the §6 spawn sequence's worktree/branch step creates the dependent's branch off the dependency's branch, the operator records `fab status set-base-branch <change> <dep-branch>` (in the target worktree). Because `/git-pr` now passes `--base` (§ 6 below), the sentence *"After `/git-pr` creates the dependent's PR, the operator retargets its base to the dependency's branch: `gh pr edit <pr> --base <dep-branch>` (`/git-pr` itself is unchanged and mode-unaware)"* is **removed** — the retarget-after-create is redundant. The merge-all choreography's retarget-verify / `rebase --onto` steps under Ordered Merge are **unchanged** (they re-point a stacked PR at the default branch after its dependency merges — a different moment).
- Changes that never get a branch (e.g. `/fab-draft` left unactivated) simply have no `base_branch`; every consumer falls back (§ 4).

*How a standalone `/git-branch` invocation learns a non-default base (an optional `--base <branch>` argument, vs. only the operator writing the field via the verb) is deferred — see Assumptions. The `{--base argument if given}` slot above is the working shape if the argument is adopted.*

### 4. Shared Go merge-base helper honoring `origin/HEAD` and the per-change base

Replace the two near-duplicate helpers — `prmeta.mergeBase(repoDir) string` (`src/go/fab/internal/prmeta/prmeta.go` ~line 621) and `status.resolveMergeBase(repoDir) (string, error)` (`src/go/fab/internal/status/true_impact.go` ~line 76) — with **one** exported helper. Working placement: `src/go/fab/internal/impact` (both callers already import it, so no new package or import-cycle risk; apply may choose a small `internal/gitbase` package instead if `impact` reads as the wrong home).

```go
// ResolveBaseRef returns the remote-tracking ref to diff against for a change.
// Order: the per-change base (origin/<baseBranch>) when baseBranch is non-empty
// AND the ref resolves; else origin/HEAD's target; else origin/main; else origin/master.
// Returns "" when nothing resolves.
func ResolveBaseRef(repoDir, baseBranch string) string

// MergeBase returns `git merge-base <baseRef> HEAD` (trimmed), or "" / an error when
// the base ref is empty or the merge-base cannot be computed.
func MergeBase(repoDir, baseRef string) (string, error)
```

- `origin/HEAD` is read via `git symbolic-ref --short refs/remotes/origin/HEAD` (yielding e.g. `origin/develop`); a missing `origin/HEAD` (fresh clones without `git remote set-head`) falls through to the literal probes, so no repo that works today stops working.
- **Fail-open on a vanished base**: when `base_branch` is set but `origin/<base_branch>` does not resolve (typical after the dependency's PR merged and its branch was deleted), the helper falls through to the default-branch chain exactly as if the field were absent. The PR/true_impact numbers then measure against the default branch — which, after the dependency merged, is the correct base.
- `prmeta.Gather` and `status.WriteTrueImpact` both already hold the loaded `*sf.StatusFile`, so they pass `statusFile.BaseBranch` — no new I/O. `WriteTrueImpact` keeps its best-effort contract (stderr warning + `nil` on failure); `pr-meta` keeps dropping only the Impact block on a missing merge-base.
- `fab impact <base> <head>` is **unchanged** (it never resolved a merge-base itself; callers pass refs).
- Tests (Constitution VII, code-quality.md test-alongside): table-driven unit tests for `ResolveBaseRef` covering (a) per-change base present and resolvable, (b) per-change base set but ref missing → default chain, (c) `origin/HEAD` → `develop` with no `origin/main` (the bug: today yields no Impact / no true_impact), (d) no `origin/HEAD`, `origin/main` present, (e) only `origin/master`, (f) nothing → `""`. Existing `true_impact_test.go` (`setupGitRepo` pins `origin/main`) and `prmeta_test.go` gain a stacked-branch case asserting the Impact counts exclude the parent's commits. The old private helpers are deleted, not kept as wrappers.

### 5. Skill consumers read the field — one shared snippet

Each skill consumer resolves the base with the same three-step snippet (recorded field → default-branch chain → merge-base):

```bash
base_branch=$(fab status get-base-branch "{id}" 2>/dev/null)
if [ -z "$base_branch" ]; then
  base_branch=$(git symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|^origin/||')
  [ -n "$base_branch" ] || base_branch=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name 2>/dev/null)
  [ -n "$base_branch" ] || base_branch=$(git rev-parse --verify -q refs/remotes/origin/main >/dev/null && echo main || echo master)
fi
git rev-parse --verify -q "refs/remotes/origin/$base_branch" >/dev/null || base_branch={resolved default, same chain}   # vanished stacked base ⇒ fail open
base=$(git merge-base HEAD "origin/$base_branch")
git diff "$base"...HEAD
```

- **`src/kit/skills/_review.md`** (§ Review Agent Dispatch, "Context the worker operates on", ~line 55): replace *"compute the merge-base against the default branch (`git merge-base HEAD origin/main` or the resolved default)"* with the snippet above (the sequencer passes `{id}`; the worker runs it). The changed-file list uses the same `<base>`.
- **`src/kit/skills/fab-adopt.md`** Step 0 (~lines 50–67): the guard chain already resolves `default_branch`; step 4's `base=$(git merge-base HEAD "origin/$default_branch")` becomes the snippet. Note the ordering subtlety: at Step 0 the change does not exist yet (it is created in Steps 1+2), so `get-base-branch` has nothing to read on a first adopt — the adopted PR's real base is available from `gh pr view --json baseRefName` and is the better source when a PR exists; use it, else the default chain. After `fab change new` in Step 1, adopt records it: `fab status set-base-branch {name} "$base_branch"`. `_generation.md` § Intake-from-Diff's "default-branch merge-base" wording (~line 158) is swept to "base-branch merge-base".
- **`src/kit/skills/git-pr.md`**: Step 3c row 4 `gh pr create --draft --title … --body …` gains `--base "$base_branch"` (resolved by the snippet; the existing Step 2 `default_branch` chain at lines 84–86 is where `base_branch` is derived — the default-branch guard "cannot ship from the default branch" keeps using `default_branch`, since being on the *base* branch of a stack is not the same guard). The `--fill` fallback also passes `--base`. Step 3d (Meta retrofit) is unaffected. `/git-pr-review` is unaffected (works from `gh pr view`).

### 6. Migration + CLI reference + sweep

- **Migration** `src/kit/migrations/2.27.0-to-2.28.0.md` (exact range set at release per the range-based model): an **announce** migration — Summary describes the new optional `base_branch` field and the two verbs; **Changes: none** — the field is absent on every existing change and every consumer falls back to the resolved default branch, so there is nothing to rewrite (Pre-check: `fab status get-base-branch <any-change>` prints an empty line; Verification: `fab status validate-status-file <change>` exits 0 on an untouched change; `set-base-branch` then `get-base-branch` round-trips). `docs/memory/distribution/migrations.md` catalog gains the entry.
- **CLI reference** (`_cli-fab.md`, Constitution Additional Constraints): § fab status table (+2 rows, `--json` enumeration 9→10); § fab pr-meta "Impact math" bullet — *"against the merge-base of HEAD vs `origin/main` (falling back to `origin/master`)"* → *"against the merge-base of HEAD vs the change's `base_branch` when recorded (else `origin/HEAD`'s target, else `origin/main`, else `origin/master`)"*; § fab impact "Consumers" paragraph names the shared helper's resolution order.
- **Sibling sweep** (code-quality.md § Sibling Sweeps — up front, not after review): grep `origin/main` / `origin/master` / `merge-base` / `retarget` / `mode-unaware` across `src/kit/skills/*.md`, `docs/memory/**`, `docs/specs/**` and update every stale claim. Known members: `fab-operator.md` stacked-prs paragraph (line ~582) and the "Why `origin/{default_branch}` as base" paragraph; `_generation.md` ~158; `docs/memory/pipeline/schemas.md` (`true_impact` write path, `pr-meta`), `docs/memory/pipeline/change-lifecycle.md` (`.status.yaml` fields, `/git-branch` key properties, default-branch convention), `docs/memory/pipeline/execution-skills.md` (`/git-pr` 3c, review dispatch diff, `/fab-adopt` Step 0), `docs/memory/runtime/operator.md` (stacked-prs resolution + g8st DD), `docs/specs/templates.md` (`.status.yaml` field notes), `docs/specs/skills.md` (`/git-branch`, `/git-pr`, `/fab-adopt`), `docs/specs/architecture.md` § git integration, `docs/specs/glossary.md` (`/git-branch`), `docs/specs/operator.md` (stacked-prs).
- **Constitution V**: every deployed edit (`src/kit/**`) cites only kit skills, `fab` commands, or host convention paths — never `src/go/*` or fab-kit's `docs/*` (the Go portability guard `src/go/fab-kit/cmd/fab/kit_portability_test.go` enforces it).

### Non-goals

- Dependency-branch **drift** after a dependent PR exists (a dep's rework moving its branch) — out of scope, as today; conflicts surface at merge-all.
- Changing `/git-pr-review` or the operator's Dependency Resolution step 0 — both already use the real base.
- Rewriting historical `true_impact` blocks on archived or in-flight changes.
- A `fab impact` change — it stays a pure `<base> <head>` calculator.

## Affected Memory

- `pipeline/schemas`: (modify) `.status.yaml` gains the optional `base_branch` field (omitempty, sparse-key insertion); `true_impact` write path and `fab pr-meta` Impact math resolve the base via the shared helper (`base_branch` → `origin/HEAD` → `origin/main` → `origin/master`); `fab status set-base-branch` / `get-base-branch [--json]` join the query surface.
- `pipeline/change-lifecycle`: (modify) `.status.yaml` field list; `/git-branch` (and `/fab-new` Step 11) now write `base_branch` at branch creation — the "never modifies `.status.yaml`" property is retired; the g8st default-branch convention is extended: the per-change base is consulted first.
- `pipeline/execution-skills`: (modify) `/git-pr` Step 3c passes `--base`; review dispatch diff base is the recorded base; `/fab-adopt` Step 0 diff base prefers the PR's `baseRefName` and records it.
- `runtime/operator`: (modify) `stacked-prs` same-repo resolution records the dependency branch via `fab status set-base-branch`; the post-create `gh pr edit --base` retarget is removed; g8st Design Decision gains an *Updated by* note.
- `distribution/migrations`: (modify) catalog entry for the announce migration (`base_branch` field + verbs; no file edits).

## Impact

**Go** (`src/go/fab/`):
- `internal/statusfile/statusfile.go` (+ `statusfile_test.go`, `golden_test.go` if the golden fixture enumerates keys) — `BaseBranch` field, read/insert cases.
- `internal/status/` — `SetBaseBranch` mutator (`status.go` or a mutators file, + `mutators_test.go`); `true_impact.go` drops `resolveMergeBase`, calls the shared helper with `statusFile.BaseBranch` (+ `true_impact_test.go`).
- `internal/prmeta/prmeta.go` — drops `mergeBase`, calls the shared helper (+ `prmeta_test.go` stacked case).
- `internal/impact/` (working placement) — `ResolveBaseRef`, `MergeBase` + table tests.
- `cmd/fab/status.go` — `statusSetBaseBranchCmd`, `statusGetBaseBranchCmd` (+ `status_test.go`, `status_json_test.go`).
- Run `go test ./src/go/fab/internal/statusfile/... ./src/go/fab/internal/status/... ./src/go/fab/internal/prmeta/... ./src/go/fab/internal/impact/... ./src/go/fab/cmd/fab/...` (scope first, widen if cross-cutting); `gofmt` before ship (a prior change's CI failed on unformatted worker-written Go).

**Kit** (`src/kit/`):
- `skills/_cli-fab.md` (§ fab status, § fab pr-meta, § fab impact), `skills/git-branch.md` (Step 4, Key Properties), `skills/fab-new.md` (Step 11 twin), `skills/git-pr.md` (Step 2/3c), `skills/_review.md` (§ Review Agent Dispatch), `skills/fab-adopt.md` (Step 0/1), `skills/_generation.md` (~158 wording), `skills/fab-operator.md` (stacked-prs paragraphs).
- `templates/status.yaml` (comment line only).
- `migrations/2.27.0-to-2.28.0.md` (new).

**Docs**: the memory files above; spec sweep of `docs/specs/templates.md`, `skills.md`, `architecture.md`, `glossary.md`, `operator.md`, `config.md` (only if it enumerates `.status.yaml` keys).

**Behavioral scope**: zero change for any change without `base_branch` on a `main`/`master` repo; on non-`main`/`master` repos `fab pr-meta` and `true_impact` start producing Impact data (bug fix); stacked changes get correct Impact, `true_impact`, review diff, adopt diff, and PR target.

## Open Questions

- Does `base_branch` store a plain branch name (`main`) or a remote-tracking ref (`origin/main`)? Working default: plain name.
- Should `fab pr-meta` and `fab status finish` also accept an explicit `--base` override on top of the status field, or is the field the only input?
- How does a standalone `/git-branch` invocation learn a non-default base — an optional `--base <branch>` argument, or only the operator writing the field via `fab status set-base-branch` after the branch exists?
- Does `fab change list --show-stats` need anything beyond inheriting the corrected `true_impact` numbers (e.g. a base column for stacked changes)?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | One change: a `base_branch` field in `.status.yaml` is the single recorded fact every base-relative surface reads (`fab pr-meta`, `WriteTrueImpact`, `_review.md` dispatch, `/fab-adopt`, `/git-pr --base`); per-surface `--base` flags rejected | Discussed — user accepted the recommendation by invoking `/fab-proceed`; the rejected alternative and its drift evidence (`gh pr edit --base` retarget) were named in the conversation | S:95 R:70 A:90 D:95 |
| 2 | Certain | Unify `prmeta.mergeBase` and `status.resolveMergeBase` into one shared exported helper whose order is per-change base → `origin/HEAD` target → `origin/main` → `origin/master`; old private helpers deleted | Discussed — explicitly decided; code-quality.md anti-pattern (duplicating utilities) and the `origin/HEAD` bug are both in the audit | S:90 R:85 A:95 D:90 |
| 3 | Certain | Fail-open: an absent `base_branch` resolves to the repo default branch — zero behavior change for non-stacked work; no data rewrite of existing changes | Discussed — stated as a constraint of the approach; matches Constitution III idempotency posture | S:90 R:80 A:90 D:90 |
| 4 | Certain | Ship an announce-type migration under `src/kit/migrations/` (Summary + Pre-check + Changes: none + Verification) — the schema addition is optional and fail-open, so no file edits are prescribed | context.md § Migrations and code-quality.md make the migration mandatory for `.status.yaml` schema changes; the 2.23.17-to-2.24.0 / 2.25.1-to-2.26.0 precedents establish the announce/no-edit shape | S:85 R:90 A:95 D:85 |
| 5 | Certain | `fab status refresh` never writes or clears `base_branch`; `fab impact <base> <head>` is unchanged | hooks-may-enhance-never-own: refresh recomputes only artifact-derived fields; `fab impact` never resolved a merge-base and the CLI ref says callers pass refs | S:70 R:85 A:95 D:90 |
| 6 | Certain | CLI reference (`_cli-fab.md` § fab status / § fab pr-meta / § fab impact) and Go tests ship in the same change; deployed edits cite no `src/go/*` or fab-kit `docs/*` paths | Constitution Additional Constraints (CLI ⇒ docs + tests) and Constitution V (portability guard test) determine this | S:85 R:90 A:95 D:95 |
| 7 | Confident | Write path is a `fab status set-base-branch <change> <branch>` / `get-base-branch <change> [--json]` verb pair modeled on `set-summary`/`get-summary` (flock write; empty line on absence; object-wrapped JSON) | `.status.yaml` is written only through `fab status` under flock; the summary pair is the exact precedent; both deferred options for `/git-branch` presuppose such a verb | S:70 R:80 A:85 D:75 |
| 8 | Confident | `base_branch` is written at branch creation by the `git-branch.md` Step 4 / `fab-new.md` Step 11 twins (create/rename/track cases; write-if-absent on the no-op cases), defaulting to the g8st default-branch chain; the status template carries only a comment line; branch-less changes leave it absent | Discussed — "set at branch creation (git-branch / fab-new tail), defaulting to the resolved repo default branch"; `fab change new` runs before any branch exists, so the template cannot seed a real value; idempotent write-if-absent protects an operator-set value | S:80 R:75 A:80 D:70 |
| 9 | Confident | Operator `stacked-prs` mode records the dependency branch via `fab status set-base-branch` at the spawn sequence's branch step and **drops** the post-create `gh pr edit --base` retarget (now redundant since `/git-pr` passes `--base`); Ordered Merge's retarget-verify / `rebase --onto` steps stay | Discussed — operator sets the field explicitly in stacked-prs mode; keeping a redundant retarget is the owner-and-pointer drift the change exists to remove; the merge-all retarget serves a different moment (after the dep merges) | S:70 R:85 A:80 D:70 |
| 10 | Confident | Skill consumers share one snippet: `fab status get-base-branch` → g8st chain when empty → verify `origin/$base_branch` exists (else fall back) → `git merge-base HEAD origin/$base_branch`; `/git-pr` passes `--base "$base_branch"` on both the normal and `--fill` create paths; the "cannot ship from the default branch" guard keeps using `default_branch` | The chain is already inlined per consumer by decision (g8st DD rejected a `_` helper); `gh pr create --base` takes a plain branch name; the default-branch guard has a different purpose than base selection | S:75 R:85 A:80 D:75 |
| 11 | Confident | Vanished-base fail-open: when `base_branch` is set but `origin/<base_branch>` no longer resolves (dependency merged, branch deleted), Go and skill consumers fall through to the default-branch chain instead of erroring | After the dependency merges into the default branch, the default branch *is* the correct base; erroring would break `fab status finish` / `pr-meta` on every completed stack; mirrors the absent-field posture | S:50 R:80 A:75 D:65 |
| 12 | Confident | `/fab-adopt` Step 0 prefers the adopted PR's `gh pr view --json baseRefName` as the base when a PR exists (else the default chain) and records it with `set-base-branch` after `fab change new` in Step 1 | At Step 0 no change exists yet so the field cannot be read; the PR's real base is authoritative and already fetched by the same `gh pr view` call the guard uses | S:60 R:80 A:80 D:70 |
| 13 | Confident | Shared Go helper lives in `internal/impact` (both callers import it; no new package, no import cycle); apply may relocate to a small dedicated package if `impact` reads as the wrong home | Placement only — trivially reversible; `internal/impact` is the sole package both `prmeta` and `status` already depend on for this computation | S:60 R:85 A:65 D:45 |
| 14 | Unresolved | Storage format of `base_branch`: plain branch name (`main`) vs remote-tracking ref (`origin/main`); working default plain name | Deferred — promptless dispatch | S:35 R:55 A:80 D:70 |
| 15 | Unresolved | Whether `fab pr-meta` and `fab status finish` also grow an explicit `--base` override on top of the status field | Deferred — promptless dispatch | S:35 R:80 A:65 D:50 |
| 16 | Unresolved | How a standalone `/git-branch` learns a non-default base: optional `--base <branch>` argument vs only the operator writing the field via `fab status set-base-branch` | Deferred — promptless dispatch | S:35 R:70 A:55 D:45 |
| 17 | Unresolved | Whether `fab change list --show-stats` needs any change beyond inheriting corrected `true_impact` numbers (e.g. a base column) | Deferred — promptless dispatch | S:40 R:90 A:80 D:75 |

17 assumptions (6 certain, 7 confident, 0 tentative, 4 unresolved). Run /fab-clarify to review.
