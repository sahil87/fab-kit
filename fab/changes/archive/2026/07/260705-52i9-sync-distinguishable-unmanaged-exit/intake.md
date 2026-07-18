# Intake: fab sync: distinguishable "not a fab-managed repo" exit code

**Change**: 260705-52i9-sync-distinguishable-unmanaged-exit
**Created**: 2026-07-05

## Origin

> Backlog item `[52i9]` (added 2026-07-05, `fab/backlog.md:31`):
> "fab sync: emit a distinguishable 'not a fab-managed repo' outcome (distinct documented exit
> code, or an --if-managed no-op flag) instead of generic exit 1 — so callers (wt default init,
> hop, operators) can branch on 'not applicable' vs real sync failure without replicating the
> fab/project/config.yaml walk-up (error site: internal/sync.go ResolveConfig nil-check).
> Context: wt is shipping an interim wt-side marker probe that mirrors ResolveConfig to
> gracefully skip its default 'fab sync' init in non-fab repos; this exit-code contract would
> let fab stay authoritative long-term."

Created via `/fab-new 52i9 backlog` (one-shot, no prior conversational discussion of this
specific item — this intake reflects fresh code investigation, not a preceding design chat).

## Why

**Problem**: `fab sync` currently returns exit code 1 for two semantically different outcomes:
(a) "this directory isn't a fab-managed repo at all" (a benign non-applicability signal), and
(b) a genuine sync failure (a corrupt config, a failed scaffold write, a version-guard trip). A
caller like `wt`'s default init hook cannot tell these apart from the exit code alone, so it
cannot safely make "skip my fab-sync step" the default behavior — doing so today would also
silently swallow real failures.

**Consequence of not fixing**: `wt` (and any other external caller — hop, operator scripts) is
forced to duplicate fab's own `fab/project/config.yaml` walk-up logic client-side just to decide
whether to call `fab sync` at all. This is exactly what's happening today: the backlog item notes
wt is shipping an "interim wt-side marker probe that mirrors ResolveConfig" — a duplicated,
drift-prone reimplementation of logic that already lives in `fab-kit`. Every additional consumer
(hop, future operators) would need its own copy.

**Why this approach over alternatives**: The codebase already has a working precedent for
exactly this shape of problem — tiered non-1 exit codes returned via an in-handler `os.Exit(N)`
call, bypassing cobra's default error-wraps-to-1 behavior. `src/go/fab/cmd/fab/pane_window_name.go`
uses this for a `tmuxExitCode` (2 for pane-missing, 3 otherwise), and
`src/go/fab/cmd/fab/memory_index.go` uses `os.Exit(2)` for a tier-2 destructive-loss case, with an
explicit comment: *"main() exits 1 on any returned error, so a non-1 code must be set in-handler."*
Following this existing convention keeps the fix idiomatic and low-risk, and avoids introducing a
new flag-based API surface (`--if-managed`) when the simpler, already-proven mechanism (a
documented distinct exit code) fully satisfies the stated need: letting external callers branch
on "not applicable" vs. "real failure" without any new fab-side flag to learn or maintain.

## What Changes

### 1. New distinct exit code for "not a fab-managed repo"

`fab sync`'s cobra `RunE` currently returns a plain `fmt.Errorf("not in a fab-managed repo. Run
'fab init' to set one up")` when `ResolveConfig()` returns `(nil, nil)` (the non-error,
walked-to-filesystem-root case). This error return is indistinguishable, at the exit-code level,
from any other failure — both fall through to `main()`'s blanket `os.Exit(1)` on any `RunE`
error (`src/go/fab-kit/cmd/fab-kit/main.go`).

Change: when `ResolveConfig()` returns `(nil, nil)` inside `Sync()`
(`src/go/fab-kit/internal/sync.go`), the sync command SHALL print its existing "not in a
fab-managed repo" message to stderr and exit with a **new distinct, documented exit code**
(not 0, not 1) — e.g. `os.Exit(3)` — instead of returning a generic `error` that collapses to
exit 1 in `main()`. A genuine sync failure (corrupt config YAML, a failed scaffold write, a
version-guard trip) continues to return a real `error` and fall through to the existing exit 1
path — behavior for actual failures is unchanged.

Because `fab-kit`'s `main()` currently exits 1 uniformly for any `RunE` error
(`src/go/fab-kit/cmd/fab-kit/main.go:54-58`), this requires the same in-handler `os.Exit(N)`
treatment already used in the `fab` binary's `pane_window_name.go` / `memory_index.go` — call
`os.Exit(3)` directly inside the sync command handler (after printing the message) rather than
returning an `error` from `RunE`, mirroring the established pattern.

```go
// src/go/fab-kit/internal/sync.go — inside Sync(), where the nil-check lives today
cfg, err := ResolveConfig()
if err != nil {
    return err
}
if cfg == nil {
    fmt.Fprintln(os.Stderr, "not in a fab-managed repo. Run 'fab init' to set one up")
    os.Exit(3) // distinct from exit 1 (generic failure) — see docs/memory/distribution/exit-codes.md
}
```

The exact numeric code (proposed: `3`) SHALL be defined as a named constant (not a bare magic
number at each call site) so it can be referenced from documentation and from any future
consumer-side code in this repo.

### 2. Consolidate the duplicated nil-check (opportunistic, in-scope)

The `if cfg == nil { return fmt.Errorf("not in a fab-managed repo...") }` check is currently
copy-pasted three times with no shared helper:
- `src/go/fab-kit/internal/sync.go` (the fix target)
- `src/go/fab-kit/internal/migrations_status.go`
- `src/go/fab-kit/internal/upgrade.go` (a more elaborate variant tolerating a config.yaml with a
  missing `fab_version` field)

Since this change touches the exact line the other two duplicate, extract a small shared helper
(e.g. `requireManagedRepo() (*Config, error)` or similar) that `sync.go` and
`migrations_status.go` both call, returning the distinct-exit-code behavior consistently. Do
**not** change `upgrade.go`'s more elaborate variant in this change — its config-with-missing-
version case is a different semantic (partially-managed, not unmanaged) and is out of scope here;
note it as a candidate for a future follow-up rather than conflating it with this fix.

### 3. Documentation

- Update `docs/memory/distribution/kit-architecture.md` (currently states sync "exits non-zero
  when any deployment/scaffolding write fails or when the version guard trips") to document the
  new distinct exit code and the two-outcome split (not-managed vs. real failure).
- Update `docs/memory/distribution/distribution.md` (existing "not in a fab-managed repo" framing
  around lines 31/48/53/107) to reference the new exit code contract.
- The exit code value and its meaning SHOULD live in one canonical documented location (new or
  existing memory file) that external consumers like `wt` can be pointed to, so `wt` can retire
  its interim client-side `ResolveConfig`-mirroring probe in favor of checking this exit code
  directly.

## Affected Memory

- `distribution/kit-architecture`: (modify) document the new distinct "not a fab-managed repo"
  exit code for `fab sync`, replacing the current generic "exits non-zero" framing
- `distribution/distribution`: (modify) update sync/router behavior description to reference the
  new exit-code contract
- `distribution/migrations`: (modify) `migrations.md`'s claim that `migrations-status` exits
  "non-zero only on a genuine error" is stale — exit 3 is now a non-error precondition signal
  (added during rework cycle 2 review; missed in the original Affected Memory list)
- `distribution/exit-codes`: (new, if no existing exit-code convention file exists) canonical
  documentation of fab's non-1 exit code conventions (the `pane_window_name`/`memory_index`
  precedent plus this new sync code), so future commands needing a distinct exit code have one
  place to register it — confirm during hydrate whether this should be a new file or a section
  within `kit-architecture.md`

## Impact

- **Code**: `src/go/fab-kit/internal/sync.go` (primary fix), `src/go/fab-kit/internal/config.go`
  (read-only reference — `ResolveConfig`/`resolveConfigFrom`, no change expected),
  `src/go/fab-kit/cmd/fab-kit/main.go` (verify `RunE`/`os.Exit` interaction accommodates an
  in-handler exit), `src/go/fab-kit/internal/migrations_status.go` (opportunistic consolidation,
  see What Changes #2)
- **Not touched**: `upgrade.go`'s separate, more elaborate nil/missing-version variant (explicitly
  out of scope, see #2)
- **External consumers** (not modified by this change, but the intended beneficiaries): `wt`
  (run-kit's tool, sibling Homebrew-tap project) — out of repo, no code change here, but this
  change is what lets `wt` eventually retire its interim client-side probe
- **Tests**: new/updated Go tests around `Sync()`'s behavior in a non-fab-managed directory,
  asserting the new exit code is returned (existing test coverage for `ResolveConfig` itself is
  presumed already adequate per the investigation — verify during apply)
- **Docs**: `docs/memory/distribution/kit-architecture.md`, `docs/memory/distribution/distribution.md`

## Open Questions

- Confirm the exact numeric exit code to standardize on (this intake proposes `3` as unused by
  existing `fab-kit` binary exit paths — needs a repo-wide grep for existing `fab-kit` exit codes
  before finalizing, since `fab-kit` is a separate binary from `fab` and may not share the `fab`
  binary's 2/3 usage in `pane_window_name.go`).
- Confirm whether `docs/memory/distribution/exit-codes.md` should be a new standalone memory file
  or a section appended to the existing `kit-architecture.md` — deferred to hydrate.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Solve via a distinct documented exit code, not a new `--if-managed` flag | Codebase already has a proven, idiomatic precedent for exactly this shape (tiered `os.Exit(N)` in `pane_window_name.go`/`memory_index.go`); a flag would add new API surface with no corresponding existing pattern, and the backlog item itself frames the exit-code route as what lets "fab stay authoritative long-term" | S:60 R:70 A:80 D:65 |
| 2 | Tentative | Use exit code `3` specifically | No existing `fab-kit` (separate binary from `fab`) exit-code registry found during investigation; `3` avoids collision with the `fab` binary's own 2/3 usage but that binary's codes don't necessarily apply to `fab-kit`. Needs a final grep-verify during apply before locking in. | S:40 R:75 A:50 D:45 |
| 3 | Confident | Also consolidate the duplicated `cfg == nil` check in `migrations_status.go` (not `upgrade.go`) | Same exact duplicated logic sits one file away and is trivial to fold in while touching this code path; `upgrade.go`'s variant is semantically different (missing-version vs. fully-unmanaged) and safer left alone | S:65 R:70 A:75 D:60 |
| 4 | Confident | Real sync failures (corrupt config, failed writes, version-guard trips) keep returning a normal `error` → exit 1, unchanged | Backlog item only asks to distinguish the "not managed" case; broadening the change to re-tier all failure exit codes would be scope creep beyond what was requested | S:70 R:75 A:80 D:75 |

4 assumptions (0 certain, 3 confident, 1 tentative, 0 unresolved).
