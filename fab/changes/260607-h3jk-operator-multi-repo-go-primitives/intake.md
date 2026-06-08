# Intake: Operator Multi-Repo Go Primitives

**Change**: 260607-h3jk-operator-multi-repo-go-primitives
**Created**: 2026-06-07
**Status**: Draft

## Origin

> feat: operator multi-repo Go primitives — Add the binary-side primitives that let `/fab-operator` coordinate fab agents across multiple repos and tmux sessions on one tmux server.

This is **change 1 of a 2-change split** (decided in a `/fab-discuss` session, 2026-06-07) that makes `/fab-operator` work across multiple repos and multiple tmux sessions on a single tmux server:

- **Change 1** (this change) — the **mechanism**: server-keyed XDG state file, per-repo `mainRoot` in `fab pane map`, and a `fab spawn-command --repo` helper. All Go, all in `src/go/fab/`.
- **Change 2** (`260607-oy0k-operator-multi-repo-skill`) — the **policy**: re-frame `fab-operator.md` and its specs around the `(session, repo, pane)` model. **Depends on this change** (same-repo dependency → cherry-picks under autopilot).

Splitting Go from skill+specs separates mechanism from policy: this change ships and is independently testable; change 2 layers the coordination behavior on top of the primitives introduced here.

**Design decisions** (locked in the discussion, recorded in memory `operator-multi-repo-design.md`):

1. **Isolation unit = tmux server.** One operator per server. The state file is keyed by the tmux **socket path** — not by repo (would lose cross-repo state) and not by a fixed global path (would force a machine-wide singleton). A fixed path was explicitly rejected for that reason.
2. **XDG resolution is uniform across Linux and macOS** — `$XDG_STATE_HOME` if set and absolute, else `$HOME/.local/state`. Deliberately NOT `~/Library/...` on macOS (terminal users expect `~/.local/state`). The Go stdlib has no `UserStateDir()`, so a small explicit helper is required.
3. **No migration** of old repo-rooted `.fab-operator.yaml` files — abandoned in place.
4. **Launch CWD unchanged** — `runOperator` keeps `gitRepoRoot()` only as the new-window working directory (minimal diff); only the *state path* decouples from the repo.

## Why

**Problem.** The operator's coordination state lives at `gitRepoRoot()/.fab-operator.yaml` — anchored to whichever repo the operator pane happens to sit in (`operator_tick_start.go:33,39`). This breaks multi-repo coordination: every `tick-start` writes to one repo regardless of where the work is, and the file is a single repo's branch namespace. Separately, `fab pane map` computes one `mainRoot` from the first parsable pane CWD and applies it to **all** rows (`panemap.go:85`), so worktree display paths for panes in *other* repos are computed relative to the wrong root — garbage relative paths. Finally, the operator can only read its own repo's `agent.spawn_command`, so it cannot spawn an agent into a different repo with that repo's spawn configuration.

**Consequence of not fixing.** Without these primitives, the skill (change 2) has nothing to build on: there is no server-keyed state location to document, no `repo` field to group the tick by, and no way to fetch a target repo's spawn command. The operator stays single-repo.

**Why this approach.** The Go primitives are already ~80% pane-centric (each pane resolves via its own CWD → its own git worktree → its own `fab/`), so the changes are surgical and additive:
- The state-file relocation is the one *foundational* change (everything else assumes coordination state has a stable, server-scoped home).
- The per-repo `mainRoot` fix is the one genuine *bug* (vs. missing feature) — it's a display-correctness fix independent of the rest.
- The `fab spawn-command --repo` helper is a thin wrapper over the existing `spawn.Command(path)` (which already accepts an explicit path), keeping the skill out of YAML parsing.

Keying on the **socket path** (not server PID) was chosen because the socket path survives a tmux-server restart (same `-L` label → same path), so a restarted operator resumes the same state file; PID would change and orphan the file.

## What Changes

All in `src/go/fab/`. No skill behavior changes here (change 2), except the Constitution-mandated `_cli-fab.md` update.

### A1 — Server-keyed state file `(foundational)`

Replace the repo-rooted `.fab-operator.yaml` location with an XDG-compliant, socket-keyed path. New helpers (in `operator.go`, or a small `internal/operator` package — apply-stage decision):

```go
// stateDir returns the XDG state base dir, spec-compliant and uniform on
// Linux and macOS. Honors XDG_STATE_HOME only when set AND absolute.
func stateDir() (string, error) {
    if s := os.Getenv("XDG_STATE_HOME"); s != "" && filepath.IsAbs(s) {
        return s, nil
    }
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".local", "state"), nil
}

// serverSlug derives a filesystem-safe slug from the tmux socket path.
// Falls back to "default" if tmux can't be queried.
func serverSlug(server string) string {
    out, err := exec.Command("tmux", pane.WithServer(server, "display-message", "-p", "#{socket_path}")...).Output()
    if err != nil {
        return "default"
    }
    return slugify(strings.TrimSpace(string(out)))
}

// StatePath = $XDG_STATE_HOME/fab/operator/<server-slug>.yaml, parent MkdirAll'd.
func StatePath(server string) (string, error) {
    base, err := stateDir()
    if err != nil {
        return "", err
    }
    dir := filepath.Join(base, "fab", "operator")
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return "", err
    }
    return filepath.Join(dir, serverSlug(server)+".yaml"), nil
}
```

- `runOperatorTickStart` (`operator_tick_start.go`): use `StatePath()` instead of `gitRepoRoot()` + `.fab-operator.yaml`.
- Rename the test seam `operatorRepoRootOverride` → `operatorStatePathOverride` (now a full file path, not a dir — cleaner injection).
- `runOperator` (`operator.go:44,59`): **unchanged** — keeps `gitRepoRoot()` only as the new-window launch CWD.
- **No migration.** Old repo-rooted files are abandoned in place.

The slug derivation for a socket path like `/tmp/tmux-1000/default` must be filesystem-safe (e.g. replace `/` with `-`, strip leading separator) — exact slugify rule is an apply-time detail, but it must be deterministic and collision-free for distinct socket paths.

### A2 — Per-repo `mainRoot` in `fab pane map` `(bug fix)`

`panemap.go:82-87` computes one `mainRoot` from all pane CWDs and passes it to every `resolvePane` call. With panes from multiple repos, `WorktreeDisplayPath(wtRoot, mainRoot)` (`internal/pane/pane.go:200`) computes relative paths from the wrong repo's parent.

Fix:
- Compute `mainRoot` **per distinct repo**, cached by the pane's `GitWorktreeRoot`. Each pane's display path uses *its own* repo's main root.
- Add a **`repo`** field (absolute main-worktree root) to both `paneRow` (`panemap.go:43-53`) and `paneJSON` (`panemap.go:309-319`), so the skill (change 2) can group rows by repo without re-deriving. The table output MAY add a `Repo` column or leave the table as-is and expose `repo` only in `--json` (apply-time presentation decision; `--json` field is required either way).

### A4 — `fab spawn-command --repo <path>` helper `(new command)`

Add a `spawnCommandCmd()` cobra command:
- `fab spawn-command --repo <path>` reads `agent.spawn_command` from `<path>/fab/project/config.yaml` via the existing `spawn.Command(configPath)` and prints it to stdout.
- If `--repo` is omitted, default to the current repo (search upward via `resolve.FabRoot()`, consistent with other commands).
- This lets the operator skill fetch the **target** repo's spawn command instead of its own.

### `_cli-fab.md` update (Constitution-mandated)

`fab/project/constitution.md`: *"Changes to the `fab` CLI (Go binary) MUST … update `src/kit/skills/_cli-fab.md` with any new or changed command signatures."* Document:
- the new `fab spawn-command [--repo <path>]` command, and
- the `fab operator tick-start` state-path change (now server-keyed XDG, not repo-rooted).

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) operator state file location — now server-keyed XDG (`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`) instead of repo-rooted `.fab-operator.yaml`.
- `fab-workflow/execution-skills`: (modify) `fab pane map` now exposes a `repo` field and computes per-repo display paths; new `fab spawn-command` helper.

(Hydrated at this change's hydrate stage. The operator's *behavioral* contract — addressing tuple, two-tier deps — is change 2's memory impact, not this one's.)

## Impact

- **New code**: `stateDir`/`serverSlug`/`StatePath`/`slugify` helpers; `spawnCommandCmd()` and its file (e.g. `spawn_command.go`).
- **Modified**: `src/go/fab/cmd/fab/operator_tick_start.go` (state path + renamed seam), `src/go/fab/cmd/fab/panemap.go` (per-repo mainRoot + `repo` field), `src/go/fab/cmd/fab/operator.go` (wire `spawnCommandCmd` into the root command; launch CWD unchanged).
- **Tests**: adapt `operator_test.go` to the renamed seam; new `stateDir` table test (env set / unset / relative-ignored); `serverSlug` slugify test; multi-repo fixture in `panemap_test.go` (per-repo display paths + `repo` JSON field); `fab spawn-command` test.
- **Docs**: `src/kit/skills/_cli-fab.md`.
- **No templates, no migrations.**
- **Two Go modules**: per recent commit #378 ("gate PR merges on tests passing across both Go modules"), CI runs both modules — ensure new tests pass in the relevant module.

## Open Questions

_(All resolved during clarify — see `## Clarifications`.)_

- **A1 placement** — RESOLVED: keep the new helpers in `cmd/fab/operator.go` alongside `gitRepoRoot` (minimal diff, no new package). <!-- clarified: helpers in operator.go, not internal/operator -->
- **A2 table column** — RESOLVED: expose `repo` only in `fab pane map --json` (the skill consumes JSON); leave the human table as-is. The per-repo `mainRoot` fix still corrects the Worktree column's display paths. <!-- clarified: repo in --json only, no human-table column -->

## Clarifications

### Session 2026-06-08

| # | Action | Detail |
|---|--------|--------|
| 10 | Confirmed | Helpers stay in `cmd/fab/operator.go` (no `internal/operator` package) |
| 11 | Confirmed | `repo` exposed in `fab pane map --json` only; no human-table column |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | State file keyed by tmux socket path under `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` | Discussed — user confirmed; rejected fixed global path (forces machine-wide singleton) and repo-rooting (loses cross-repo state) | S:97 R:55 A:88 D:93 |
| 2 | Certain | XDG resolution: `$XDG_STATE_HOME` if set+absolute, else `$HOME/.local/state`; uniform Linux + macOS (NOT `~/Library`) | Discussed in detail — user asked about both platforms; spec-compliant fallback, terminal-user convention on mac | S:96 R:70 A:92 D:95 |
| 3 | Certain | No migration of old repo-rooted `.fab-operator.yaml`; abandon in place | Discussed — user explicitly said no migration needed | S:98 R:80 A:90 D:98 |
| 4 | Certain | Socket path (not server PID) as the key | Discussed — socket path survives server restart (same `-L` → same path); PID would orphan the file | S:92 R:60 A:88 D:90 |
| 5 | Certain | `runOperator` launch CWD unchanged — `gitRepoRoot()` stays only as new-window dir | Discussed — user/agent agreed leave launch CWD as-is for minimal diff; state path is the only thing that decouples | S:90 R:75 A:88 D:90 |
| 6 | Certain | Ship as Go-primitives change separate from skill+specs; change 2 depends on this | Discussed — user selected "Split Go / skill" | S:97 R:75 A:92 D:95 |
| 7 | Confident | Add `repo` field to `fab pane map --json` (and `paneRow`); fix `mainRoot` to be per-repo | Verified bug in `panemap.go:85`; the `repo` field is what change 2's tick groups by. Only sensible fix | S:85 R:60 A:88 D:82 |
| 8 | Confident | `fab spawn-command --repo` wraps existing `spawn.Command(path)`; defaults to current repo when `--repo` omitted | Verified `spawn.Command` takes an explicit path; default-to-current matches other commands' `resolve.FabRoot()` pattern | S:82 R:65 A:85 D:80 |
| 9 | Confident | `serverSlug` falls back to `"default"` when tmux can't be queried | Operator must still function if `#{socket_path}` query fails; a fixed fallback is the safe degrade | S:78 R:70 A:80 D:75 |
| 10 | Certain | Keep new helpers in `cmd/fab/operator.go` rather than a new `internal/operator` package | Clarified — user confirmed | S:95 R:78 A:60 D:52 |
| 11 | Certain | Expose `repo` only in `--json`, not as a new human-table column | Clarified — user confirmed | S:95 R:80 A:58 D:50 |

11 assumptions (8 certain, 3 confident, 0 tentative, 0 unresolved).
