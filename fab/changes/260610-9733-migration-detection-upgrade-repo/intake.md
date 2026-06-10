# Intake: Mechanical migration detection in `fab upgrade-repo`

**Change**: 260610-9733-migration-detection-upgrade-repo
**Created**: 2026-06-10
**Status**: Draft

## Origin

> Make `fab upgrade-repo` mechanically detect whether any migrations are relevant, and consolidate the migration-discovery algorithm into the binary so the `/fab-setup migrations` skill can reuse it. In the no-migrations case, `fab upgrade-repo` should also update `fab/.kit-migration-version`. It should notify the user only if migrations are needed — and the reminder is easy to miss today, so it should be colored/bolded. Also check whether the `/fab-setup migrations` skill can reuse the new command.

Interaction mode: conversational, originating from a `/fab-discuss` session that walked the
current `upgrade.go` reminder logic, the `/fab-setup migrations` discovery algorithm, the
migration file format, and the existing `compareSemver`/`parseSemver` helpers. All major design
decisions were settled interactively (see Assumptions) before this intake was generated.

## Why

**The problem.** `fab upgrade-repo` (`src/go/fab-kit/internal/upgrade.go:92-99`) emits:

```
Run '/fab-setup migrations' to update project files (X -> Y)
```

whenever `fab/.kit-migration-version` (local) differs from the target version *string* — a naive
string inequality. Three concrete defects follow:

1. **No mechanical relevance check.** The discovery algorithm that actually decides whether a
   migration applies (`FROM <= local < TO` over files named `{FROM}-to-{TO}.md`) lives *only* in
   the `/fab-setup migrations` skill markdown. The Go binary has no access to it, so it cannot
   distinguish "migrations genuinely apply" from "the version number merely bumped." Live example:
   this repo's `.kit-migration-version` is `2.1.0`, the newest migration file is
   `1.9.7-to-1.10.0.md`, so **nothing applies** — yet the current code would still nag on upgrade.

2. **No-migrations case never advances `.kit-migration-version`.** Because only the skill ever
   writes that file, and the skill only runs when the user is prompted, the local version drifts
   permanently behind the engine version even when there is nothing to migrate. The skill's own
   algorithm already specifies the correct behavior ("no match and no later migrations → set to
   engine version") — `upgrade-repo` just never does it.

3. **The reminder is plain text** in the middle of sync output, so users miss it (explicitly
   reported by the user: "even now, many times I miss that message").

**Consequence if unfixed.** Spurious nags train users to ignore the message (defect 3 compounds
defect 1); `.kit-migration-version` drift means `/fab-status` drift detection and future migration
discovery both operate on a stale baseline; and the discovery algorithm remains duplicated between
prose (skill) and — once we add detection — code, inviting the two to diverge.

**Why this approach.** Put the discovery algorithm in the binary *once*, expose it as a queryable
command (`fab migrations-status`), and have **both** `upgrade-repo` and the `/fab-setup migrations`
skill consume it. This makes the binary the single source of truth for discovery, removes the
brittle "LLM scans a directory and parses semver" steps from the skill, and gives `upgrade-repo`
the mechanical signal it needs to (a) suppress spurious nags, (b) self-stamp the no-op case, and
(c) only then bother styling a reminder.

## What Changes

### 1. New discovery logic — `src/go/fab-kit/internal/migrations.go` (new file)

A self-contained implementation of the discovery algorithm currently described only in
`fab-setup.md` Migrations Step 2–3. Reuses the existing `parseSemver` / `compareSemver` helpers in
`src/go/fab-kit/internal/sync.go` (same package — no new dependency).

```go
// MigrationRange is one parsed migration file: {From}-to-{To}.md.
type MigrationRange struct {
    From string // semver, e.g. "1.9.7"
    To   string // semver, e.g. "1.10.0"
    File string // base filename, e.g. "1.9.7-to-1.10.0.md"
}

// DiscoverResult is the full outcome of a discovery pass.
type DiscoverResult struct {
    Local      string           // fab/.kit-migration-version
    Engine     string           // target/engine VERSION
    Applicable []MigrationRange // ordered list to apply, FROM ascending
    GapSkips   []string         // human-readable "no migration for X -> Y, skipping"
    Overlaps   []string         // pairs of overlapping filenames (non-empty => error)
}
```

Functions:

- `parseMigrationFilename(name string) (MigrationRange, bool)` — matches `{FROM}-to-{TO}.md`,
  parses both as semver; returns `false` for non-matching names (e.g. `.gitkeep`, `README.md`).
- `DiscoverMigrations(migrationsDir, local, engine string) (DiscoverResult, error)` — scans the
  directory, parses filenames, validates **non-overlapping ranges** (`A.From < B.To && B.From <
  A.To` ⇒ overlap), sorts by FROM ascending, then walks the discovery loop:
  1. find first migration where `FROM <= current < TO` → append to `Applicable`, set `current = TO`, repeat
  2. else if a later migration exists with `FROM > current` → record a gap-skip, advance `current` to that FROM, repeat
  3. else → done
- A convenience predicate: `len(result.Applicable) > 0` ⇒ migrations needed. (No separate
  `NeedsMigration` field required; callers check the slice.)

Overlap is reported, not silently resolved — a malformed migration set must not be guessed at.

### 2. New command — `fab migrations-status [--json]`

Wired into `src/go/fab-kit/cmd/fab-kit/main.go` (and reachable through the `fab` shim, which routes
unknown subcommands to `fab-kit` — to be verified during apply). Resolves the repo's
`fab/.kit-migration-version` and the engine `$(fab kit-path)/VERSION`, scans the engine's
`migrations/` directory, and runs `DiscoverMigrations`.

- **Human output** (default): local version, engine version, the ordered applicable list (or "no
  migrations apply"), any gap-skips, and any overlap error.
- **`--json` output** (for the skill): the `DiscoverResult` serialized — `local`, `engine`,
  `applicable` (array of `{from,to,file}`), `gap_skips`, `overlaps`. The skill parses this instead
  of re-deriving discovery in prose.

Exit code: `0` on a clean query (including "nothing applies"); non-zero only on a real error
(missing version file, overlap detected — TBD whether overlap is non-zero exit or just populated
`overlaps` field; resolved as a Confident assumption below).

### 3. `fab upgrade-repo` behavior change — `src/go/fab-kit/internal/upgrade.go:92-99`

Replace the string-inequality reminder block. After sync completes, run `DiscoverMigrations`
against the **target** version's cached `migrations/` dir and the current `.kit-migration-version`:

- **Overlap detected** → print a warning naming the conflicting files + "Run '/fab-setup
  migrations' to resolve." Do **not** stamp `.kit-migration-version`. (Refuse to guess on a
  malformed set.)
- **`Applicable` non-empty** → print the reminder, styled **bold + yellow when `os.Stdout` is a
  character device**, plain text when piped/redirected. Do **not** stamp — `/fab-setup migrations`
  owns the write after it actually applies each file.
- **`Applicable` empty (and no overlap)** → **stamp `.kit-migration-version` to the target version
  silently** (no migration line printed). Mirrors the skill's "no match, no later migrations → set
  to engine version" rule and stops the drift described in Why #2.
- **`.kit-migration-version` missing** → preserve the existing init-guidance behavior (per
  `docs/memory/distribution/migrations.md` "Version Drift Detection").

TTY detection is dependency-free:

```go
func isTTY(f *os.File) bool {
    info, err := f.Stat()
    if err != nil {
        return false
    }
    return info.Mode()&os.ModeCharDevice != 0
}
```

Styling helper emits `\033[1;33m…\033[0m` (bold yellow) only when `isTTY(os.Stdout)`; otherwise the
raw string. No `golang.org/x/term`, no `go-isatty` — consistent with Constitution I (minimal,
single-binary, no new runtime deps).

### 4. Skill update — `src/kit/skills/fab-setup.md` (Migrations Step 2–3)

Replace the manual "scan `migrations/` → parse FROM/TO → validate non-overlap → sort" prose (Step
2) and the inlined discovery loop (Step 3) with:

> Run `fab migrations-status --json`. It returns the ordered `applicable` list, any `gap_skips`,
> and any `overlaps`. If `overlaps` is non-empty, STOP and report the conflict. Otherwise, apply
> each file in `applicable` in order (see [Applying a Migration]).

The **"Applying a Migration"** section is unchanged — the LLM still reads each migration file and
executes its Pre-check / Changes / Verification steps, and still writes `TO` to
`.kit-migration-version` after each. Only *discovery* moves into the binary; *application* stays in
the skill (Constitution I — migration instruction files remain pure-prompt and LLM-applied).

### 5. Constitution-required companions

- **`src/kit/skills/_cli-fab.md`** — add the `fab migrations-status [--json]` signature (constitution:
  "Changes to the `fab` CLI ... MUST update `src/kit/skills/_cli-fab.md`").
- **`docs/specs/skills/SPEC-fab-setup.md`** — reflect the skill's Step 2–3 change (constitution:
  "Changes to skill files ... MUST update the corresponding `docs/specs/skills/SPEC-*.md`").
- **`src/go/fab-kit/internal/migrations_test.go`** — table-driven tests for `parseMigrationFilename`
  and `DiscoverMigrations` (applicable chain, gap-skip, overlap, no-op/empty, local==engine,
  local-ahead). Constitution: "Changes to the `fab` CLI ... MUST include corresponding test updates."
- **`docs/memory/distribution/migrations.md`** — changelog entry; the discovery algorithm now has a
  binary home, and `upgrade-repo`'s "Version Drift Detection" bullet gains the mechanical-detection
  + self-stamp behavior.

## Affected Memory

- `distribution/migrations`: (modify) Document `fab migrations-status`, the binary-owned discovery
  algorithm, `upgrade-repo`'s mechanical detection + silent self-stamp on the no-op case, and the
  TTY-gated styled reminder. Update the "Version Drift Detection" and "`/fab-setup migrations`
  Discovery algorithm" subsections.
- `pipeline/*` (fab-setup skill behavior): (modify, via spec) The `/fab-setup migrations` discovery
  steps now delegate to the binary. Captured in `docs/specs/skills/SPEC-fab-setup.md` rather than a
  memory file, since the skill's *application* behavior is unchanged.

## Impact

- **Code**: `src/go/fab-kit/internal/migrations.go` (new), `src/go/fab-kit/internal/upgrade.go`
  (reminder block rewrite + self-stamp), `src/go/fab-kit/cmd/fab-kit/main.go` (command wiring),
  `src/go/fab-kit/internal/migrations_test.go` (new). Reuses `parseSemver`/`compareSemver` from
  `sync.go`.
- **Shim**: verify `fab` (the `src/go/fab` shim) routes `migrations-status` through to `fab-kit`
  (most subcommands pass through; confirm during apply).
- **Skill**: `src/kit/skills/fab-setup.md` (discovery steps), `src/kit/skills/_cli-fab.md` (signature).
- **Specs/Docs**: `docs/specs/skills/SPEC-fab-setup.md`, `docs/memory/distribution/migrations.md`.
- **No data migration of its own** — this change ships no `{FROM}-to-{TO}.md` file; it changes tool
  behavior, not project-file format. (`true_impact_exclude` already excludes `fab/` and `docs/`.)
- **Backward compatibility**: `migrations-status` is additive; the skill change is behavior-preserving
  (same files discovered, just by the binary); the self-stamp only advances the local version when
  it is provably safe (nothing applies).

## Open Questions

- None blocking. All design points were resolved during the `/fab-discuss` session (see Assumptions).

## Clarifications

### Session 2026-06-10 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 4 | Confirmed | — |
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |
| 9 | Confirmed | Resolved to recommended default — exit 0 on clean query incl. overlap; overlap surfaced via `overlaps` field |

## Assumptions

<!-- All Certain/Confident assumptions below were settled interactively during the originating
     /fab-discuss session, then re-confirmed via explicit AskUserQuestion choices. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No-migrations case: stamp `.kit-migration-version` to target **silently** (no migration line). | User explicitly chose "Stamp to target, silent" via AskUserQuestion. Mirrors the skill's existing no-op rule. | S:98 R:80 A:90 D:95 |
| 2 | Certain | Reminder styling: **bold + yellow, TTY-gated** (ANSI when `os.Stdout` is a char device, plain when piped). | User explicitly chose "Bold + color, TTY-gated". Standard CLI convention; no garbled codes in logs. | S:98 R:85 A:90 D:95 |
| 3 | Certain | Overlap during `upgrade-repo`: **warn with detail + do NOT stamp**; defer resolution to `/fab-setup migrations`. | User explicitly chose "Warn, don't stamp". Refusing to guess on a malformed migration set is the safe default. | S:98 R:75 A:85 D:90 |
| 4 | Certain | TTY detection is dependency-free via `Stat().Mode()&os.ModeCharDevice`, not `golang.org/x/term`/`go-isatty`. | Clarified — user confirmed. Codebase has zero color/isatty deps today; Constitution I favors minimal single-binary deps. | S:95 R:80 A:85 D:75 |
| 5 | Certain | Discovery logic reuses existing `parseSemver`/`compareSemver` in `sync.go` (same package), not a new semver lib. | Clarified — user confirmed. Helpers already exist in-package and match the skill's integer-triplet comparison exactly. | S:95 R:80 A:90 D:80 |
| 6 | Certain | Expose discovery as a **queryable command** (`fab migrations-status`) consumed by both `upgrade-repo` and the skill, rather than a private helper used only by `upgrade-repo`. | Clarified — user confirmed. A shared command makes the binary the single source of truth and de-duplicates the algorithm currently living in skill prose. | S:95 R:65 A:85 D:80 |
| 7 | Certain | The `/fab-setup migrations` skill keeps **applying** migrations (Pre-check/Changes/Verification + writing `TO`); only **discovery** moves to the binary. | Clarified — user confirmed. Migration files are pure-prompt LLM instructions (Constitution I); application cannot move to the binary without violating that principle. | S:95 R:75 A:90 D:85 |
| 8 | Certain | `--json` shape: `{local, engine, applicable:[{from,to,file}], gap_skips, overlaps}`. | Clarified — user confirmed. Minimal machine-readable projection of `DiscoverResult` sufficient for the skill to apply files and detect overlap. | S:95 R:75 A:80 D:70 |
| 9 | Certain | `migrations-status` exit code: `0` for a clean query incl. "nothing applies" AND incl. overlap (overlap is surfaced via the `overlaps` field, not exit code); non-zero reserved for a genuine error (missing VERSION file, unreadable migrations dir). | Clarified — user confirmed bulk; resolved to recommended default. Both callers read the structured `overlaps` field, so exit 0 keeps the query semantics simple and uniform. | S:95 R:80 A:65 D:75 |

9 assumptions (9 certain, 0 confident, 0 tentative, 0 unresolved).
