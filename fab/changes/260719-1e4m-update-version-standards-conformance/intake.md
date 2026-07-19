# Intake: Update & Version Standards Conformance

**Change**: 260719-1e4m-update-version-standards-conformance
**Created**: 2026-07-20

## Origin

One-shot `/fab-new` invocation. Raw input:

> Bring this repo into conformance with the shll toolkit 'update' and 'version' standards (docs/site/standards/update.md and version.md in the shll repo, or https://shll.ai/standards). Audit the update and --version subcommands against every MUST/SHOULD in both standards, fix any gaps found, and add/update tests pinning the fixed behavior. If the audit finds the repo is already fully conformant with no code changes needed, skip /git-pr entirely — do not open an empty PR.

Key intake-time decisions: both standards were re-fetched live via `shll standards update` and `shll standards version` (the standards evolve; local memory of them goes stale), and the full clause-by-clause audit was performed **at intake time** as gap analysis. Findings are baked into this intake, so apply executes a known fix list rather than re-auditing. The audit found real gaps, so the "fully conformant → skip /git-pr" branch does NOT apply — this change proceeds through the normal pipeline including PR creation.

## Why

The constitution's **Toolkit Standards** article (Additional Constraints, added 1.4.0) binds this repo to the shll toolkit's published standards: the CLI surface MUST conform, and standards revised upstream bind without further amendment. The `update` standard's **brew-handling safety** clause was added from an observed 2026-07-19 incident: a wrapper's **120-second hard kill landed mid-keg-swap** (between `brew unlink` and `brew link`) during a stalled GitHub API call inside `brew upgrade`, corrupting the keg and leaving a broken binary (`zsh: permission denied: <tool>`). fab-kit's `runWithTimeout` (`src/go/fab-kit/internal/update.go:129-143`) is exactly that wrapper shape: a 120s hard timeout on `brew upgrade` (30s on `brew update`) terminated via `cmd.Process.Kill()` — Go's `Process.Kill` sends **SIGKILL**, which brew cannot trap. Every `fab-kit update` on a machine with a slow network moment risks corrupting its own install.

If unfixed: (1) install corruption on slow networks — the exact incident the standard documents; (2) `shll update` composes each tool's `update`, so fab-kit's non-conformances degrade the whole-toolkit upgrade run; (3) the non-brew-install path exits non-zero, making `shll update` report a false failure for the run on dev-build machines.

The `--version` side is already conformant in behavior (`fab-kit version v2.16.5`, exit 0, stdout, local, instant) but lacks the minimal pinning test the standard's verify checklist calls for.

## What Changes

### 1. Audit results (performed at intake, 2026-07-20, against live `shll standards` output)

Scope note: the shll roster tool is **`fab-kit`** — the standards' probe/delegation target is the `fab-kit` binary (`fab-kit update`, `fab-kit --version`), which is also reachable as `fab update` via the router's lifecycle allowlist.

**update standard** (`src/go/fab-kit/internal/update.go`, `cmd/fab-kit/main.go`):

| Clause | Level | Verdict |
|---|---|---|
| Expose `update` subcommand, in-place upgrade, works standalone | MUST | ✅ pass |
| `update --help` contains literal substring `--skip-brew-update` | MUST | ✅ pass (verified against built binary) |
| Honor `--skip-brew-update` (skip internal `brew update`) | MUST | ✅ pass (pinned by `TestUpdateSkipBrewUpdateGating`) |
| Exit 0 on success incl. already-up-to-date | MUST | ✅ pass (`latest == currentVersion` → nil) |
| Exit non-zero only on genuine failure | MUST | ⚠️ gap — non-brew install exits 1 (see fix 3) |
| No SIGKILL to a package-manager subprocess mid-transaction | MUST | ❌ **FAIL** — `runWithTimeout` calls `cmd.Process.Kill()` (SIGKILL) on both `brew update` and `brew upgrade` |
| No short hard timeout on `brew upgrade` | MUST | ❌ **FAIL** — 120s hard timeout (the standard's incident cites a 120-second hard kill) |
| Any bound generous + graceful (SIGTERM + grace, never SIGKILL) | SHOULD | ❌ fail — subsumed by the two rows above |
| Self-update only when brew-installed, `/Cellar/` gate via `os.Executable()` symlink resolution | SHOULD | ✅ pass (`isBrewInstalled`) |
| Non-brew install degrades with clear message instead of erroring | SHOULD | ⚠️ gap — message printed, but `ErrNotBrewInstalled` propagates to `os.Exit(1)` |
| One name, four places (repo / roster / formula leaf / binary = `fab-kit`) | MUST | ✅ pass |
| `v{semver}` release tags | MUST | ✅ pass (e.g. `v2.16.5`) |
| Rename ships `formula_renames.json` | MUST | ✅ n/a — no rename |

**version standard** (`cmd/fab-kit/main.go` — cobra `Version: displayVersion(version)`):

| Clause | Level | Verdict |
|---|---|---|
| `--version` supported, exit 0, version to stdout | MUST | ✅ pass |
| Respond within 2s, no network I/O on the version path | MUST | ✅ pass (pure local cobra template) |
| Version token on first non-empty line, no banner above | MUST | ✅ pass — `fab-kit version v2.16.5` is the RECOMMENDED canonical shape |
| Binary name on PATH equals tool name | MUST | ✅ pass |
| Minimal test pinning exit 0 + first-line shape | verify checklist | ⚠️ gap — `displayVersion` has unit tests, but nothing pins the actual root-command `--version` output shape (see fix 4) |

### 2. Fix: brew-handling safety — delete `runWithTimeout`, run brew unbounded

Remove the timeout wrapper entirely. In `internal/update.go`:

- `brew update --quiet` and `brew upgrade fab-kit` run via plain `cmd.Run()` with inherited stdout/stderr (as today) and **no bound, no kill path**.
- Delete `runWithTimeout` outright (its only two callers are these brew invocations).

This satisfies the verify checklist literally ("no code path sends `SIGKILL` to `brew`, and no short hard timeout caps `brew upgrade`") by deletion rather than by adding SIGTERM-escalation machinery. brew inherits the terminal; a user can Ctrl-C (SIGINT — brew traps it and unwinds). The standard explicitly suggests "not reaching for a timeout at all" as the cleaner alternative. Rejected alternative: a generous (tens-of-minutes) bound with SIGTERM + grace — conformant too, but adds signal-handling code and a hard-to-test escalation path for no concrete benefit; a hung brew is visible (output streams to the user) in both interactive `update` and the `versionGuard` auto-update path.

### 3. Fix: non-brew install exits 0

The `update` command maps the sentinel to success at the command layer — `cmd/fab-kit/main.go` `updateCmd()`:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    err := internal.Update(version, skipBrewUpdate)
    if errors.Is(err, internal.ErrNotBrewInstalled) {
        return nil // degrade with the already-printed message — not brew's to upgrade (update standard)
    }
    return err
},
```

`internal.Update` keeps returning `ErrNotBrewInstalled` unchanged — `versionGuard` (`internal/sync.go`) depends on the sentinel to compose its "auto-update did not succeed" error text and must keep treating not-brew as a guard failure (a too-old non-brew binary must still block sync). The message Update prints ("was not installed via Homebrew… Update manually, or reinstall with: brew install sahil87/tap/fab-kit") already satisfies the standard's "clear message" requirement. Behavior change is `fab-kit update` / `fab update` exit code only: 1 → 0 on non-brew installs.

### 4. Tests (pinning the fixed behavior)

In `cmd/fab-kit/main_test.go`:

- **Version shape** (version standard verify checklist): execute the root command with `--version` (injected version, e.g. `2.16.5`), assert exit success and that the **first line** of stdout matches `^fab-kit version v\d+(\.\d+)*$`.
- **Help contract**: execute `update --help`, assert output contains the literal substring `--skip-brew-update` (frozen textual contract — substring presence, not regex).
- **Non-brew exit 0**: with `isBrewInstalled` overridden to false, assert `updateCmd` RunE returns nil (exit 0) while `internal.Update` still returns the sentinel (existing `TestUpdate_NotBrewInstalledReturnsSentinel` stays as-is, guarding the versionGuard contract).

In `internal/update_test.go`:

- **Already-up-to-date exits 0**: fake brew reporting stable == currentVersion → `Update` returns nil and the brew log contains no `upgrade` invocation.
- Existing `TestUpdateSkipBrewUpdateGating` continues to pin flag honoring; it needs no change (the fake-brew harness is unaffected by the timeout-wrapper deletion).

The *absence* of the kill path is enforced structurally (`runWithTimeout` deleted — nothing left to call `Process.Kill`); review verifies no `Process.Kill`/`exec` timeout reappears on the brew paths.

### 5. Docs

- `src/kit/skills/_cli-fab.md` — the fail-loud exit-contract row for `update` (currently: "Exits non-zero … when the binary is not brew-installed") must be rewritten to the new semantics: exit 0 with the degrade message on non-brew installs; non-zero only on genuine brew/upgrade failure. Constitution: CLI-behavior changes MUST update `_cli-fab.md`.
- No `src/kit/skills/*.md` skill-behavior changes → no `docs/specs/skills/SPEC-*.md` mirrors in scope. Sweep check at apply: grep repo-wide for prose claiming `update` exits non-zero on non-brew installs (behavior-claim sweeps must include user-facing string literals).
- `docs/memory/distribution/distribution.md` records toolkit-standards conformance "audited at shll v0.0.23" — hydrate updates it with this audit and the two behavior changes.

## Affected Memory

- `distribution/distribution.md`: (modify) update the update-mechanism / toolkit-standards-conformance sections — brew calls now unbounded (no timeout wrapper, no SIGKILL path), `update` exits 0 on non-brew installs, conformance re-audited against the live update + version standards

## Impact

- `src/go/fab-kit/internal/update.go` — delete `runWithTimeout`, unbounded brew calls
- `src/go/fab-kit/internal/update_test.go` — new already-up-to-date test
- `src/go/fab-kit/cmd/fab-kit/main.go` — `updateCmd` sentinel→exit-0 mapping
- `src/go/fab-kit/cmd/fab-kit/main_test.go` — version-shape, help-contract, non-brew-exit-0 tests
- `src/kit/skills/_cli-fab.md` — `update` exit-contract row
- Callers unaffected: `versionGuard` (sync.go) keeps its sentinel-based contract; router `fab update` inherits the new exit code via exec
- No migration (no user-data restructuring); no skill files; no SPEC mirrors

## Open Questions

None — the intake-time audit resolved all decision points; see Assumptions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Audit verdict: two MUST failures (SIGKILL + 120s hard timeout on `brew upgrade`, both in `runWithTimeout`) and one SHOULD gap (non-brew install exits 1); every other clause of both standards passes; version standard needs only a pinning test | Grounded in direct code read of update.go/main.go + live `shll standards` output + built-binary help/version checks at intake | S:85 R:90 A:95 D:90 |
| 2 | Confident | Fix the brew-safety MUSTs by deleting `runWithTimeout` and running brew unbounded, rather than a generous SIGTERM+grace bound | Standard sanctions both; verify checklist ("no code path sends SIGKILL") is satisfied cleanest by deletion; removes code instead of adding an escalation path; Ctrl-C (SIGINT) remains available interactively | S:60 R:85 A:85 D:70 |
| 3 | Confident | Non-brew install: map `ErrNotBrewInstalled` → exit 0 at the command layer only; `internal.Update` keeps returning the sentinel for `versionGuard` | Standard says degrade "instead of erroring" + exit non-zero only on genuine failure; `shll update` delegation would otherwise read false failures; versionGuard's too-old-blocks-sync contract preserved via the internal sentinel | S:60 R:80 A:75 D:65 |
| 4 | Confident | Do NOT set `HOMEBREW_NO_GITHUB_API=1` on brew subprocesses | The standard offers it only for tools that must bound the call; with the bound removed the stalled-API risk is a slow update, not a corrupted keg; keeping brew behavior stock | S:50 R:90 A:75 D:70 |
| 5 | Confident | Router `fab --version` output (`fab 2.16.5` + `project:` line) is out of scope and left unchanged | Version standard scope is the seven binaries by tool name — for this repo the roster binary `fab-kit`; the router's first line still carries a parseable bare token regardless | S:65 R:85 A:80 D:75 |
| 6 | Certain | `--version` behavior needs no code change; only the verify-checklist pinning test is added | Live check: `fab-kit --version` → `fab-kit version v2.16.5`, exit 0, stdout, instant, pure-local cobra template — the standard's RECOMMENDED canonical shape | S:80 R:90 A:90 D:85 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).
