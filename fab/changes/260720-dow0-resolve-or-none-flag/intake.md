# Intake: fab current + fab resolve rework — absence as a first-class query result

**Change**: 260720-dow0-resolve-or-none-flag
**Created**: 2026-07-20

## Origin

Promptless dispatch (Create-Intake Procedure, `{questioning-mode} = promptless-defer`). The change description was synthesized verbatim from a design conversation with the user (fab-kit's owner); all decisions below were settled in that conversation and are treated as decided, not proposed. Raw input:

> **Title/scope**: `fab resolve` — treat "no active change" as a first-class query result instead of an error, via an opt-in `--or-none` flag, and migrate the five resolve-as-probe call sites to it.
>
> **Problem (root cause)**: Fab's state machine treats "no active change" as a first-class state (`initialized` in `_preamble.md`'s State Table), but `fab resolve` — documented as a pure query — can only fail on it: `src/go/fab/internal/resolve/resolve.go` returns a not-found error ("No active change. Run /fab-new…", resolve.go:192) and every error exits 1. Callers probing for the state must therefore read the error channel as a data channel, and Claude Code renders any non-zero Bash exit as an error (red) — so an expected, valid state renders as an alarm on every `/fab-discuss` run, `/fab-proceed` state detection, `_intake` backlog-ID probe, `fab-new` rename guard, and `fab-adopt` collision guard (five sanctioned resolve-as-probe sites). This is alarm fatigue by design and erodes the one thing the error channel is for.
>
> The classification machinery already exists internally: `internal/resolve/resolve.go:17-20` defines typed sentinels `ErrNotFound` / `ErrAmbiguous` (via `classifiedError` with `Unwrap`), distinct from infrastructure errors (fab/ dir not found, I/O failures). The CLI surface (`src/go/fab/cmd/fab/resolve.go`) flattens all of them to exit 1. The fix is exposing the existing classification at the CLI surface.

**Corrections (2026-07-21, post-pipeline, user-issued — supersede the interim design above where they conflict).** After the interim `--or-none` implementation completed the full pipeline (PR #515, draft, unmerged), the user reviewed the design and directed: *"I would directly jump to the final answer, rather than merging the current interim solution"*, and — on the breaking question — *"we can use a minor release version to indicate backward incompatibility"*. The final design (settled in that follow-up conversation):

1. **New command `fab current`** — a **total state query** over repo state: "what is the active change?" Every repo condition is a valid answer (active folder / none / ambiguous); it exits 0 for any state answer and non-zero for infrastructure errors only. This replaces the *bare-form* probes.
2. **Bare `fab resolve` is REMOVED** — the `<change>` argument becomes **required** (usage error otherwise, pointing at `fab current`). Same for the `fab change resolve` wrapper (kept, but ref-required). `fab resolve <ref>` is now purely a **dereference** — a partial function where not-found/ambiguous are loud errors. The user explicitly accepted the backward incompatibility, signaled by a minor release version per project policy.
3. **`--or-none` survives, narrowed to explicit refs only** — the design conversation established that `fab current` cannot cover the three *reference-lookup* probes (`_intake` backlog-ID, `fab-new` rename guard, `fab-adopt` collision guard), which ask "does *this ref* name a change?" where absence is expected data. With the bare form gone, the flag's semantics collapse to one line: explicit ref + `ErrNotFound` → `(none)`, exit 0. The bare-vs-override `ErrAmbiguous` special case ceases to exist structurally.
4. **Design principle (record in docs)**: an agent-facing query command is either a **total state query** (every state is an answer, exit 0) or a **partial dereference** (absence is an error), never both. `fab current` is the former; `fab resolve <ref>` the latter (with `--or-none` as the explicit opt-back into totality for expected-miss lookups); `fab preflight` remains a *gate*, deliberately neither.
5. **Repo-wide bare-call audit is in scope**: removing the bare form forces migration of every bare `fab resolve`/`fab change resolve` call site — including `git-branch.md`'s no-argument path (moves to `fab current` with the skill owning the STOP + guidance) and any bare uses in `git-pr.md`, `git-pr-review.md`, `fab-operator.md`. The git-branch:~158 branch-ownership probe (fab-new's rename-guard twin) migrates to the same `fab resolve --folder "$(git branch --show-current)" --or-none` form as its twin — the previously-recorded twin divergence dissolves.

## Why

1. **The pain point**: `fab resolve` is documented as a pure query ("Pure query — converts change reference to canonical output. No side effects." — `_preamble.md` § Common fab Commands), and fab's own state machine names "no active change" a first-class state (`initialized` in `_preamble.md`'s State Table). Yet the CLI can only *fail* on that state: `internal/resolve/resolve.go` returns `notFoundf("No active change. …")` and every error exits 1. Five sanctioned skill sites probe for absence and must read the error channel as a data channel; Claude Code renders any non-zero Bash exit as an error (red), so an expected state renders as an alarm on every probe.

2. **Why the two-command split (not just the interim flag)**: the interim `--or-none` overloaded one command with two questions — "what is the active change?" (a total state query where "none" is a valid answer) and "give me the path/ID for this ref" (a partial dereference where absence is a failure). The overload forced the ugly bare-vs-override `ErrAmbiguous` asymmetry and left the flag as a modal switch between semantics. Since PR #515 is an unmerged draft, there is no ecosystem cost to skipping the interim shape entirely — and migrating skill sites once (to the final form) beats migrating them twice.

3. **Why breaking is acceptable now**: user decision — backward incompatibility is signaled by a minor release version (project policy for this toolkit); the binary and kit version-lock per release, so the removal and the skill migration land atomically. External consumers are the user's own scripts/muscle memory, accepted explicitly.

4. **The classification machinery already exists**: `internal/resolve/resolve.go:17-44` defines typed sentinels `ErrNotFound` / `ErrAmbiguous` (via `classifiedError` with `Unwrap`, matched by `errors.Is`), distinct from infrastructure errors. Both `fab current` and the narrowed `--or-none` are CLI-surface mappings over those existing sentinels; `internal/resolve` itself needs no behavior change (`resolveFromCurrent` backs `fab current` verbatim, including the single-change fallback and its `(resolved from single active change)` stderr note).

### Rejected alternatives (from both design conversations)

- **Ship the interim `--or-none`-only shape (PR #515 as-was)** — rejected by the user: it's a retrofit; jumping to the final design avoids double-migration. Superseded, not merged.
- **Skill-side `|| echo "(none)"` incantation** — repeats at every call site and swallows infrastructure errors ("fab/ directory not found" would masquerade as "no active change").
- **Distinct exit code for not-found (e.g. exit 2)** — the harness paints any non-zero red; also collides with the toolkit convention exit 2 = usage error.
- **Soft default on bare `fab resolve` (exit 0 + `(none)` without any flag)** — converts visible failures into silent `(none)` string propagation at dereference call sites; rejected in favor of removing the bare form outright so the two questions get two commands.
- **Covering the ref-probe sites with `fab current <ref>`** — muddles the state query back into a lookup; the ref probes stay on `fab resolve <ref> --or-none`.
- **Deleting the `fab change resolve` wrapper entirely** — more breaking surface for no design gain; kept as a thin ref-required alias sharing `runResolve`.

## What Changes

### 1. CLI: new `fab current` command (Go)

New `src/go/fab/cmd/fab/current.go`:

- **`fab current`** — resolves the active change exactly as bare resolve did (`resolve.FabRoot()` + `resolve.ToFolder(fabRoot, "")`, i.e. `resolveFromCurrent`: `.fab-status.yaml` pointer → single-change fallback → none/ambiguous), then maps outcomes to **state answers**:
  - Resolved → print the **folder name**, exit 0 (the single-change fallback's `(resolved from single active change)` stderr note is preserved — it comes from `internal/resolve` untouched).
  - `errors.Is(err, resolve.ErrNotFound)` → print `(none)`, exit 0. The sentinel error's guidance text ("No active change. Run /fab-new … or /fab-switch …") is emitted on **stderr** as an informational note — actionable hints survive without the error exit.
  - `errors.Is(err, resolve.ErrAmbiguous)` → print `(none)`, exit 0, with the ambiguous guidance ("multiple changes exist — use /fab-switch") on stderr.
  - Infrastructure errors (no `fab/` dir, I/O) → non-zero, unchanged error path.
- **`--json`** — emits a single JSON object instead: `{"active": "<folder>"}` when resolved; `{"active": null, "reason": "none"}` / `{"active": null, "reason": "ambiguous", "candidates": ["<folder>", …]}` otherwise (candidates listed for ambiguous only; stderr notes suppressed in JSON mode — the reason field carries the state). Exit semantics identical to plain mode.
- No output-mode flags (`--id`/`--dir`/`--status`/`--pane`) on v1 — the probes need the folder; parsimony. Deref of the current change composes as `fab resolve <folder-from-current> --dir` if ever needed.
- The `(none)` token stays a shared named constant (`noneToken`) — now used by both `fab current` and `--or-none`.

### 2. CLI: `fab resolve` bare form removed; `--or-none` narrowed (Go)

In `src/go/fab/cmd/fab/resolve.go` and `change.go`:

- **`fab resolve` requires the `<change>` argument** (`cobra.ExactArgs(1)`): bare invocation is a usage error whose message points at the replacement — e.g. `fab resolve requires a <change> argument; for the active change, use: fab current`. Follows the binary-wide usage-error convention (exit 2 tier where the router applies it; cobra arg validation otherwise — match the existing pattern in the codebase).
- **`fab change resolve` likewise requires `<change>`** — kept as a thin ref-required wrapper sharing `runResolve` (single-implementation invariant preserved; still flag-free per the documented `_preamble.md` invariant).
- **`--or-none` narrowed**: with the bare form gone, its mapping is exactly: explicit ref + `errors.Is(err, resolve.ErrNotFound)` → print `(none)`, exit 0. `ErrAmbiguous` with a ref stays a loud error (unchanged). Infrastructure errors stay non-zero. The interim bare+`ErrAmbiguous`→`(none)` branch is **deleted**.
- `--pane` composition unchanged: the sentinel mapping applies to change-resolution only; post-resolution pane-lookup failures stay non-zero.

### 3. Go tests (test-alongside)

Extend `src/go/fab/cmd/fab/resolve_test.go` and add `current_test.go`:

- `fab current`: active-pointer → folder + exit 0; no changes → `(none)` + exit 0 (+ stderr guidance); multiple-changes-no-pointer → `(none)` + exit 0; single-change fallback → folder + exit 0; dangling pointer → falls through per `resolveFromCurrent`; `--json` for each state (field names + `candidates` on ambiguous only); infra error → non-zero.
- `fab resolve`: bare invocation → usage error naming `fab current`; explicit ref not-found + `--or-none` → `(none)` exit 0; explicit ref ambiguous + `--or-none` → error; explicit ref not-found without flag → error (regression); output-mode composition (`--folder`/`--id` + `--or-none`).
- `fab change resolve`: bare → usage error; ref-required path works; flag still rejected.
- Skill-embed drift guard (`TestSkillEmbedMatchesCanonical`) re-synced.

### 4. Skill call-site migration (canonical sources under `src/kit/skills/` only)

| Site | Interim (#515 as-committed) | Final |
|------|------------------------------|-------|
| `fab-discuss.md` Context Loading step 1 | `fab resolve --folder --or-none` | `fab current` — stdout `(none)` ⇒ "No active change" |
| `fab-proceed.md` State Detection Step 1 | `fab resolve --folder --or-none` | `fab current` — branch on `(none)` vs folder name |
| `_intake.md` backlog-ID probe | `fab resolve --id {id} --or-none` | **unchanged** (explicit-ref lookup; equality compare preserved) |
| `fab-new.md` rename guard | `fab resolve --folder "$(git branch --show-current)" --or-none` | **unchanged** (explicit-ref lookup) |
| `fab-adopt.md` collision guard | `fab resolve --folder "$(git branch --show-current)" --or-none` | **unchanged** (explicit-ref lookup) |
| `git-branch.md` no-argument path | bare `fab change resolve` — stderr + STOP | `fab current` — on `(none)`, the **skill** displays the no-active-change guidance and STOPs (hard stop stays, now skill-owned) |
| `git-branch.md` branch-ownership probe (~:158) | bare-ish `fab change resolve "$(git branch --show-current)" 2>/dev/null` | `fab resolve --folder "$(git branch --show-current)" --or-none` — now byte-matches its fab-new twin; the recorded twin divergence dissolves |
| any bare `fab resolve`/`fab change resolve` in `git-pr.md`, `git-pr-review.md`, `fab-operator.md`, other kit sources | (audit) | migrate per kind: active-change probes → `fab current`; explicit-ref uses → ref-required `fab resolve` (grep repo-wide at apply; the bare form no longer exists) |

### 5. Reference docs (CLI change ⇒ doc obligations)

- `src/kit/skills/_cli-fab.md`: new § fab current (contract, states, `--json`, exit semantics); § fab resolve rewritten (ref required, narrowed `--or-none`, wrapper ref-required, pointer to `fab current` for the active change; record the **total-query vs partial-dereference** design principle and preflight's gate role).
- `src/kit/skills/_preamble.md` § Common fab Commands: add a `fab current` row (the canonical probe form); update the `fab resolve` row (`fab resolve <change> [--or-none]`, ref required) and the `fab change` row's resolve note.
- The State-derivation prose in `_preamble.md` (initialized = "no active change") gains the `fab current` probe as its detection command where a command is named.

### 6. SPEC-mirror + aggregate sweep (whole class up front)

All touched skill sources get their `docs/specs/skills/SPEC-*.md` mirrors updated in the same change (at minimum: fab-discuss, fab-proceed, _intake, fab-new, fab-adopt, git-branch, _cli-fab, _preamble, plus any audit-migrated skill). Aggregates: `docs/specs/skills.md` (restates fab-proceed step 1), `docs/specs/glossary.md` (new `fab current` term; resolve entry), `docs/specs/architecture.md` if it names bare resolve. The shll skill bundle `docs/site/skill.md` Resolution bullet re-swept with the embedded copy re-synced (byte-identical embed, ≤150-line budget). Re-grep `fab resolve` (excluding `resolve-agent`) and `fab change resolve` repo-wide before finishing apply.

### 7. Versioning / no migration

- **Breaking change shipped in a minor release** — explicit user policy for this toolkit ("we can use a minor release version to indicate backward incompatibility"). The PR body MUST carry a prominent `BREAKING` note: bare `fab resolve` / `fab change resolve` removed → use `fab current`.
- Binary + kit version-lock per release: the removal and the skill migration land atomically; no cross-version skew window.
- No user data (config, `.status.yaml`, archive layout) is restructured — **no `src/kit/migrations/` file**. Deployed skills refresh via the normal `fab sync` on upgrade.

## Affected Memory

- `pipeline/change-lifecycle`: (modify) the resolution-pattern section is rewritten to the final model — `fab current` total state query (+ `--json`), ref-required `fab resolve <ref>` dereference, narrowed `--or-none`, the total-query/partial-dereference/gate triad (current / resolve / preflight)
- `distribution/kit-architecture`: (modify) subcommand inventory gains `fab current`; `fab resolve` row updated (ref required, narrowed flag, wrapper ref-required); exit-code convention note (state answers exit 0 on `fab current`); breaking-in-minor-release policy noted
- `pipeline/planning-skills`: (modify) `_intake` backlog-ID probe unchanged from interim (`--or-none`), fab-new rename-guard twin note updated (git-branch twin now byte-matches — divergence gone)
- `pipeline/execution-skills`: (modify) `/fab-proceed` state detection + `/fab-discuss` orientation move to `fab current`; git-branch no-arg path now `fab current` + skill-owned STOP; audit results for git-pr/git-pr-review/fab-operator recorded

## Impact

- **Go**: `src/go/fab/cmd/fab/current.go` (new), `resolve.go` (arg-required + narrowed flag), `change.go` (wrapper arg-required), `resolve_test.go` + `current_test.go`, `skill.md` embed re-sync. `internal/resolve` untouched. Scope test runs to the `cmd/fab` package first (module root `src/go/fab`).
- **Kit skills**: fab-discuss, fab-proceed, git-branch (+ _intake/fab-new/fab-adopt already in final form from the interim commit), _cli-fab, _preamble, plus any bare-call audit hits (git-pr, git-pr-review, fab-operator).
- **Specs**: the touched-skill SPEC mirrors + skills.md + glossary + shll bundle.
- **Memory**: the same 4 files re-hydrated to the final truth.
- **Behavioral risk**: the bare-form removal is deliberately breaking (user-accepted, minor-release-signaled); every in-repo call site migrates in this change (atomic via version-lock). The explicit-ref surface is regression-covered; `fab current` is additive.
- **Untouched by design**: `fab preflight` (the gate), `internal/resolve` (sentinels already correct), `.fab-dispatch`/pane internals (use the internal package, not the CLI).

## Open Questions

None — the follow-up conversation settled the breaking question (minor-release signaling) and the two-command split; remaining details are recorded as Confident assumptions below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Two-command final design: `fab current` (total state query, exit 0 for all state answers) + ref-required `fab resolve <ref>` (partial dereference); bare forms removed from both spellings | User-directed ("jump to the final answer"); breaking accepted via minor-release signaling (user-issued policy) | S:95 R:70 A:90 D:95 |
| 2 | Certain | `--or-none` survives narrowed to explicit refs (`ErrNotFound` → `(none)` exit 0; ambiguous stays error); the three ref-probe sites keep their interim form unchanged | Settled in the follow-up conversation — `fab current` cannot answer "does this ref name a change?"; the flag is the explicit opt-back into totality for expected-miss lookups | S:90 R:80 A:90 D:90 |
| 3 | Certain | Token stays exactly `(none)` (shared `noneToken` const, used by both commands) | Carried over from the settled interim design — `none` is a legal 4-char ID; empty output is illegible and `$(…)`-hazardous | S:95 R:80 A:90 D:95 |
| 4 | Certain | Repo-wide bare-call audit + migration in this change: active-change probes → `fab current`; git-branch no-arg path → `fab current` with skill-owned STOP; git-branch :158 probe → `--or-none` form matching its fab-new twin | Forced by the removal (bare form no longer exists); correction item 5 names the sites; twin divergence dissolving is strictly simplifying | S:85 R:75 A:90 D:85 |
| 5 | Certain | `fab current` plain-mode contract: folder or `(none)` on stdout; sentinel guidance moves to stderr as informational notes; single-change fallback + its stderr note preserved via untouched `resolveFromCurrent` | Derived: keeps actionable hints without the error exit; behavior parity for the migrated probes (fallback semantics identical to bare resolve today) | S:75 R:85 A:85 D:75 |
| 6 | Confident | `fab current --json` schema: object with `active` (folder string, or null), `reason` ("none" or "ambiguous", only when active is null), `candidates` (array, ambiguous only); stderr notes suppressed in JSON mode | Field names agent-decided (no competing JSON convention in cmd/fab); shape settled in the design conversation's greenfield sketch | S:70 R:90 A:80 D:70 |
| 7 | Certain | `fab change resolve` kept as thin ref-required wrapper (not deleted); stays flag-free | Smallest breaking surface achieving the design goal; preserves the `_preamble.md` "query flags live on top-level only" invariant and `runResolve` single-implementation | S:75 R:85 A:85 D:75 |
| 8 | Certain | No output-mode flags on `fab current` v1 (folder + `--json` only) | Parsimony — the probes need the folder; deref composes via `fab resolve <folder> --dir`; adding modes later is additive | S:70 R:95 A:85 D:75 |
| 9 | Certain | Sweep obligations: Go tests alongside; `_cli-fab.md` + `_preamble.md`; all touched-skill SPEC mirrors + `skills.md` + `glossary.md` + shll bundle swept up front; no migration file; PR body carries the BREAKING note | Constitution Additional Constraints + code-quality.md § Sibling & Mirror Sweeps + correction item 2's release-signaling decision | S:90 R:85 A:95 D:95 |

9 assumptions (8 certain, 1 confident, 0 tentative, 0 unresolved).
