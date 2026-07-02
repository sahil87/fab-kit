# Intake: Offline-first `upgrade-repo` (default to systemVersion)

**Change**: 260613-1hmj-upgrade-repo-offline-default
**Created**: 2026-06-13

## Origin

> Surfaced from a `/fab-discuss` session debugging a live failure:
>
> ```
> ❯ fab upgrade-repo
> Resolving latest version...
> ERROR: cannot resolve latest version: GitHub API returned HTTP 403
> ```
>
> Root cause diagnosed interactively: the `403` is GitHub **unauthenticated API rate
> limiting** (60 req/hr per IP, `X-RateLimit-Remaining: 0`), not an auth/permissions
> failure — GitHub returns `403` (not `429`) when the anonymous limit is exhausted.
> The user then asked: *"why does `fab upgrade-repo` need to make a GitHub API call?
> Why not upgrade the repo to the latest system-available version of fab-kit?"*
>
> **Interaction mode**: conversational. The source was read together (`upgrade.go`,
> `download.go`, `cache.go`, `main.go`) before any decision. Decisions reached and
> locked via an explicit `AskUserQuestion`:
> 1. **Resolution model**: default no-arg `upgrade-repo` to `systemVersion` (the running
>    binary's embedded version, offline); add a `--latest` flag to opt into the GitHub
>    API check. (Chosen over "systemVersion only, no flag" and over "keep the API, just
>    fix the error message".)
> 2. **Workflow**: run this through the full fab pipeline (not a direct edit).
>
> The user's original instinct ("latest *cached* version") was explored and rejected in
> favor of `systemVersion` — see ## Why and assumption #2.

## Why

**Problem.** `fab upgrade-repo` with no version argument resolves its target by calling
GitHub's REST API (`internal.LatestVersion()` → `GET /repos/sahil87/fab-kit/releases/latest`,
`src/go/fab-kit/internal/download.go:219`). Unauthenticated, GitHub allows only 60 requests
per hour per IP. On a shared host (e.g. a GCP box) that budget is trivially exhausted, and
the command hard-fails with a misleading `cannot resolve latest version: GitHub API returned
HTTP 403`. The binary makes the call anonymously — neither `GH_TOKEN` nor `GITHUB_TOKEN` is
read — even when a perfectly good authenticated `gh` token exists on the machine.

**Consequence if unfixed.** The most common upgrade path (`fab upgrade-repo` with no arg)
is network-dependent and flaky for a reason unrelated to the user's intent. They are not
trying to discover what is newest upstream — they have just `brew upgrade`d the `fab` binary
and want their repo's kit content to match it. Forcing a network round-trip (and a fragile,
rate-limited one) to answer a question the binary already knows the answer to is wrong, and
the error text actively misleads ("403" reads as auth failure, not rate limiting).

**Why this approach over alternatives.**
- The running `fab-kit` binary already carries its own version, injected at build time
  (`var version = "dev"` in `main.go:12`, overridden via `-ldflags` by the brew formula).
  It is **already threaded into `Upgrade()`** as the `systemVersion` parameter
  (`main.go:81` → `upgrade.go:20`) — currently used only for the `runSync` version guard
  (`upgrade.go:88`), never for resolution. The authoritative "system version" is in hand,
  offline, today. Defaulting to it makes the common path offline-first and eliminates the
  403 entirely.
- This aligns with the distribution model (Constitution V): the binary is installed via
  `brew install fab-kit`; the repo's `fab/` + `.claude/skills/` kit content is *synced to
  match the binary*. Reconciling the repo to the installed binary's version is the natural
  meaning of "upgrade this repo".
- **Rejected — "latest cached version"** (the user's first instinct): the version cache
  (`~/.fab-kit/versions/` and `~/.fab-kit/local-versions/`) is populated lazily and can hold
  a *stale* download or an *unreleased* `local-versions/` dev build (which `CachedKitDir`
  actively prefers, `cache.go:40-46`). Silently upgrading to "newest thing in the cache"
  is ambiguous and surprising, and would need a new cache-enumeration helper that does not
  exist. `systemVersion` is unambiguous and authoritative — no enumeration, no precedence
  puzzle.
- **Rejected — "keep the API default, only fix the error/add token reading"**: a smaller
  change, but it leaves the common path network-dependent. It is folded in partially: the
  error message improvement still has value on the now-opt-in `--latest` path (see
  ## What Changes §4, a SHOULD).

## What Changes

### 1. Resolution precedence in `internal.Upgrade` (`src/go/fab-kit/internal/upgrade.go`)

The `Upgrade` signature gains a `useLatest bool` parameter, and the no-arg resolution branch
(currently `upgrade.go:48-55`) is rewritten to this precedence (first match wins):

```
fab upgrade-repo            → systemVersion        (offline; the running binary's version)
fab upgrade-repo <version>  → <version>            (explicit arg — unchanged)
fab upgrade-repo --latest   → LatestVersion()      (opt-in GitHub API — the OLD default)
```

Sketch (replacing the current `if targetVersion == "" { ... LatestVersion() ... }` block):

```go
// Resolve target version.
//   - explicit arg wins
//   - --latest queries GitHub (opt-in network call)
//   - default: the running binary's own version (offline, authoritative)
if targetVersion == "" {
    switch {
    case useLatest:
        fmt.Println("Resolving latest version...")
        latest, err := LatestVersion()
        if err != nil {
            return fmt.Errorf("cannot resolve latest version: %w", err)
        }
        targetVersion = latest
    case systemVersion != "" && systemVersion != "dev":
        targetVersion = systemVersion
    default:
        // A dev/just-built shim (version == "dev") or an unstamped binary has no
        // real release tag to sync to — fall back to the network so it can still
        // resolve a published release.
        fmt.Println("Resolving latest version...")
        latest, err := LatestVersion()
        if err != nil {
            return fmt.Errorf("cannot resolve latest version: %w", err)
        }
        targetVersion = latest
    }
}
```

The `"dev"` guard is load-bearing: a `just build` shim reports `version == "dev"`, which is
not a real release tag — without the fallback, `upgrade-repo` would try to sync a nonexistent
`vdev`. Everything downstream of resolution (the `currentVersion == targetVersion`
short-circuit, `EnsureCached`, `runSync`, the F18 stamp-after-sync ordering, migration
detection) is **unchanged**.

> Note: `EnsureCached` (`cache.go:69`) still lazily downloads the resolved target if it is
> not cached. The *resolution* becomes offline; the *fetch* stays as-is. In practice the
> systemVersion is almost always already cached, because the brew binary fetched its own
> kit on install/sync.

### 2. CLI flag wiring (`src/go/fab-kit/cmd/fab-kit/main.go`)

`upgradeCmd()` (currently `main.go:71-84`) gains a `--latest` boolean flag, threaded into
the new parameter:

```go
func upgradeCmd() *cobra.Command {
    var useLatest bool
    cmd := &cobra.Command{
        Use:   "upgrade-repo [version]",
        Short: "Upgrade the repo's kit to the installed binary's version (or --latest / an explicit version)",
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            targetVersion := ""
            if len(args) > 0 {
                targetVersion = args[0]
            }
            return internal.Upgrade(version, targetVersion, useLatest)
        },
    }
    cmd.Flags().BoolVar(&useLatest, "latest", false, "Resolve the newest published release from GitHub instead of the installed binary version")
    return cmd
}
```

Mutual-exclusivity note: passing both an explicit `<version>` arg and `--latest` should
behave sensibly. Since the explicit arg wins (`targetVersion != ""` skips the whole
resolution switch), `--latest` is simply ignored when a version is given. Acceptable; the
review stage should confirm this is the desired precedence (it is — explicit intent beats
a discovery flag).

### 3. Tests (`src/go/fab-kit/internal/upgrade_test.go`)

The existing 9 tests call `Upgrade(systemVersion, targetVersion)` — all need the new third
arg (mechanical `, false`). New behavioral cases required (Constitution: CLI changes MUST
include test updates; Test Integrity):

- **Default resolves to systemVersion, no network**: `Upgrade("2.3.1", "", false)` resolves
  target to `2.3.1` and never calls `LatestVersion()`. Assert by pointing `githubAPIURL` at
  an `httptest` server that fails the test if hit (the existing test harness already
  redirects `githubAPIURL`/`githubDownloadURL` — `download_test.go`).
- **`--latest` calls the API**: `Upgrade("2.3.1", "", true)` hits the stubbed `releases/latest`
  endpoint and resolves to whatever tag it returns.
- **`dev` binary falls back to the API**: `Upgrade("dev", "", false)` calls `LatestVersion()`
  (the fallback branch).
- **Explicit arg ignores `--latest`**: `Upgrade("2.3.1", "2.2.0", true)` resolves to `2.2.0`
  and does not hit the API.

### 4. `--latest`-path error message (SHOULD, fold in if low-effort)

On the now-opt-in `--latest` path, when `LatestVersion()` gets an HTTP `403` with response
header `X-RateLimit-Remaining: 0`, the error SHOULD name rate-limiting rather than the bare
status code, e.g. `GitHub API rate limit exceeded (unauthenticated: 60/hr); set GH_TOKEN or
retry after <reset>`. This requires `LatestVersion()` (`download.go:219-253`) to inspect the
`403` response headers. Reading `GH_TOKEN`/`GITHUB_TOKEN` into the request is a candidate but
**out of scope** for this change unless trivially clean — flagged as a follow-up. Keep the
core change (1–3) shippable independently of this.

### 5. Docs (`src/kit/skills/_cli-fab.md`)

The `upgrade-repo` row (`_cli-fab.md:29`) MUST document the new resolution precedence
(default = installed binary version; `--latest` = GitHub; explicit arg wins). Constitution:
"Changes to the `fab` CLI (Go binary) MUST ... update `src/kit/skills/_cli-fab.md` with any
new or changed command signatures." Check `src/kit/skills/fab-setup.md` (references
`upgrade-repo` at lines 34, 302, 387) for any instruction affected by the default change —
those references are about the migration-stamp no-op case and cache-population guidance, not
resolution, so likely no edit, but verify.

## Affected Memory

- `distribution/{file}`: (modify) The distribution domain documents the three-binary
  architecture and `/fab-setup`/upgrade flow. The default-resolution change to `upgrade-repo`
  is a spec-level behavior change to how a repo reconciles its kit version, so the relevant
  distribution memory file (covering `fab upgrade-repo` / version resolution) is updated at
  hydrate. Exact file to be confirmed against `docs/memory/distribution/index.md` during the
  Memory File Lookup at apply/hydrate.

## Impact

- **Code**: `src/go/fab-kit/internal/upgrade.go` (resolution logic + signature),
  `src/go/fab-kit/cmd/fab-kit/main.go` (`--latest` flag + threading),
  `src/go/fab-kit/internal/upgrade_test.go` (signature update on 9 callers + 4 new cases).
  Optionally `src/go/fab-kit/internal/download.go` (`LatestVersion` error message, §4 SHOULD).
- **CLI surface**: `fab upgrade-repo` no-arg semantics change (network → offline); new
  `--latest` flag. This is a user-visible behavior change to a documented command.
- **Docs**: `src/kit/skills/_cli-fab.md` (required); `src/kit/skills/fab-setup.md` (verify,
  likely no change).
- **Spec**: No `docs/specs/skills/SPEC-*` update is required for the `_cli-fab.md` edit.
  <!-- clarified: confirmed against the spec layout — there is no SPEC-_cli-fab.md; _cli-fab/_cli-external/_naming are CLI reference partials deliberately without SPEC files (unlike _generation/_preamble/_review/_srad/_pipeline). The constitution's SPEC-update rule fires only when a /fab-* skill file changes. -->
  The constitution's rule (`src/kit/skills/*.md` → matching `SPEC-*`) triggers only when a
  `/fab-*` skill file changes; `_cli-fab.md` has no SPEC. **Conditional**: `upgrade-repo` is
  also documented in `SPEC-fab-setup.md` (because `/fab-setup` orchestrates the upgrade/migration
  flow) — so IF apply ends up editing `src/kit/skills/fab-setup.md`, that SPEC must be updated
  too. Per §5, no `fab-setup.md` edit is expected (its `upgrade-repo` references concern the
  migration-stamp no-op and cache population, not resolution), so confirm `fab-setup.md` is
  untouched and no SPEC update is needed.
- **Migrations**: none. No user data restructms; this is binary behavior + docs only.
- **Backward compatibility**: anyone scripting `fab upgrade-repo` expecting "go to newest
  upstream" must switch to `fab upgrade-repo --latest`. Worth a note in the change/PR. The
  inverse (offline default) is strictly more robust for the dominant use case.

## Open Questions

- None blocking. The two design decisions (resolution model, pipeline workflow) were settled
  in the originating conversation. The §4 error-message improvement and `GH_TOKEN` reading
  are scoped as optional/follow-up, not open blockers.

## Clarifications

### Session 2026-06-13

| # | Question | Resolution |
|---|----------|------------|
| 8 | Does this CLI-binary change (documented in `_cli-fab.md`) require a `docs/specs/skills/SPEC-*` update? | No. Verified against the spec layout: there is no `SPEC-_cli-fab.md` — `_cli-fab` is a CLI reference partial deliberately without a SPEC (unlike `_generation`/`_preamble`/`_review`/`_srad`/`_pipeline`). The constitution's SPEC-update rule fires only when a `/fab-*` skill file changes. Conditional: if apply edits `fab-setup.md`, `SPEC-fab-setup.md` must be updated too — but no such edit is expected. → re-graded Tentative → Certain. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No-arg `upgrade-repo` defaults to `systemVersion`; add `--latest` flag for the GitHub-API path | User chose this explicitly via AskUserQuestion over two stated alternatives; `systemVersion` already threaded into `Upgrade()`, so the change is localized and authoritative | S:98 R:80 A:95 D:95 |
| 2 | Certain | Reject "latest cached version" as the default | User's original instinct, explicitly discussed and ruled out; cache can hold stale/unreleased builds (`CachedKitDir` prefers `local-versions`), and no enumeration helper exists | S:95 R:80 A:90 D:90 |
| 3 | Certain | Run through the full fab pipeline, not a direct edit | User chose "Run the fab pipeline" via AskUserQuestion | S:100 R:90 A:95 D:100 |
| 4 | Confident | Keep a network fallback when `systemVersion == "dev"` or empty | A `just build` shim has no real release tag; syncing to `vdev` would fail. Standard pattern, one obvious interpretation, fully reversible | S:75 R:85 A:90 D:85 |
| 5 | Confident | Explicit `<version>` arg takes precedence over `--latest` (the flag is ignored when an arg is given) | Explicit intent should beat a discovery flag; matches the "first match wins" precedence and avoids an error path for a benign combination | S:70 R:85 A:80 D:80 |
| 6 | Confident | Error-message improvement (name rate-limiting on 403) and `GH_TOKEN` reading are SHOULD/follow-up, not required for this change | Keeps the core change shippable independently; the rate-limit text only matters on the now-opt-in `--latest` path. Discussed and scoped as secondary | S:80 R:90 A:75 D:75 |
| 7 | Confident | Affected memory is a `distribution` domain file (exact file confirmed at hydrate via the domain index) | Behavior change to version resolution is spec-level and lives in distribution per the memory index; exact file is a lookup, not a judgment call | S:65 R:85 A:80 D:70 |
| 8 | Certain | No `docs/specs/skills/SPEC-*` update needed (CLI-binary change documented in `_cli-fab.md`, which has no SPEC), conditional on `fab-setup.md` being untouched | Clarified — user confirmed; verified against spec layout: no `SPEC-_cli-fab.md` exists (`_cli-fab` is a CLI reference partial deliberately without a SPEC); SPEC rule fires only on `/fab-*` skill changes | S:95 R:70 A:90 D:90 |

8 assumptions (4 certain, 4 confident, 0 tentative).
