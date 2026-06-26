# Intake: Detect & Fill test_paths at Setup

**Change**: 260626-5qf5-detect-fill-test-paths
**Created**: 2026-06-26

## Origin

> detect and fill test_paths at setup with migration and scaffold examples

Conversational origin. The discussion traced how `/git-pr`'s impact breakdown splits test vs. non-test code: it is **purely pathspec-driven** (`src/go/fab/internal/impact/impact.go` `Compute` runs a third `git diff --shortstat` pass with the `test_paths` globs as `:(glob)` includes combined with the `true_impact_exclude` patterns). `test_paths` is **language-specific and ships empty** — the scaffold has it commented out (`# test_paths: []`), and the Go config struct has no hardcoded default. When empty, the breakdown collapses to a single total line (no `impl`/`tests` split).

Key decisions reached in discussion:

1. **Rejected a case-insensitive `**/*test*` default.** A bare substring miscounts production code that contains "test" (`attestation.go`, `latest.go`, a `TestModeBanner` component) — inflating the test count and deflating `impl`, the opposite of the table's purpose. Case-insensitivity makes it worse (`Latest`, `Contestant`). The classification's reliability comes from *anchoring* to a language's test convention (`*_test.go` suffix, `test_*.py` prefix, `*.spec.ts` infix).
2. **Chosen approach: anchored language detection at setup, non-interactive.** Infer the right anchored pattern from on-disk marker files, fill `test_paths` automatically, surface a visible note rather than prompting. Zero-friction without the substring trap.
3. **Scaffold examples must persist as a standing comment** above the active key (annotated by ecosystem + convention name), so the user retains an editing reference even after a value is written.
4. **A migration backfills existing repos** — runs the same detection and fills `test_paths`, and refreshes the scaffold's comment block, for projects already on fab-kit.

## Why

**Problem.** The `/git-pr` impact breakdown is one of the most-read signals in a fab-kit PR body, but its test/impl split silently does nothing unless a project has hand-set `test_paths`. Since the key ships commented-out with no default and setup never asks for it, the overwhelming majority of fab-kit projects get the collapsed single-line breakdown — the richer `impl`/`tests` taxonomy is invisible. fab-kit itself only has the split because a human hand-edited the config.

**Consequence if unfixed.** The feature stays effectively opt-in-by-expert-knowledge. New projects never discover it; existing projects never gain it. The impact table under-delivers on its stated taxonomy (`raw / true / impl / tests / excluded`) for nearly everyone.

**Why this approach over alternatives.**
- *A hardcoded default glob* (e.g. `**/*test*`) was rejected — it can't be anchored to a language at config-write time without knowing the language, and an unanchored substring produces a confidently-wrong number (see Origin §1). A wrong-but-confident impact number is worse than an absent one.
- *Asking the user interactively* adds a setup prompt for a value most users won't have an opinion on. The user explicitly asked for non-interactive detection "if possible," and the marker-file signal is strong enough to auto-fill safely.
- *Detection at setup, anchored to the detected ecosystem*, gets zero-config correctness for recognized stacks while leaving the value trivially editable (and the migration re-runnable) if detection guesses wrong. Unrecognized stacks fall back to the standing examples — never a wrong value, just the (current) collapsed breakdown.

## What Changes

Four coordinated edits — a skill + scaffold + migration + their doc mirrors. No Go binary change (detection is prompt/skill logic, honoring Constitution I "Pure Prompt Play"; the `impact.go` consumer already handles any non-empty `test_paths` verbatim).

### 1. Scaffold — persistent example comment block

`src/kit/scaffold/fab/project/config.yaml`: replace the current `test_paths` block (lines 14–18) so the examples live as a **standing comment above the active key** (so they survive whether the key is filled or left empty), one example per line in YAML-list form, annotated by ecosystem and convention, plus the load-bearing `:(glob)` / anti-substring note:

```yaml
# Glob/pathspec patterns identifying test files. Used by the true-impact
# breakdown to attribute lines to tests vs. implementation (impl = total − tests).
# Language-specific — no kit default. Patterns are :(glob) magic pathspecs, so
# `**` matches across directories and `*` does NOT match `/`. Anchor to your
# language's test convention rather than a bare substring (a substring like
# `*test*` miscounts production code such as `attestation.go` or `latest.go`).
# When absent/empty, the breakdown collapses to a single total line.
#
# Examples (uncomment/adapt the line for your stack):
#   - "**/*_test.go"                  # Go   — `_test.go` suffix
#   - "**/test_*.py"                  # Python (pytest) — `test_` prefix
#   - "**/*.spec.ts"                  # JS/TS (Jest/Vitest) — `.spec` infix
#   - "**/*.test.ts"                  # JS/TS — `.test` infix
#   - "**/src/test/**"                # Java/Kotlin (Maven/Gradle) — test source root
# test_paths: []
```

When create-mode setup detects and fills a value, it MUST preserve this comment block intact and replace only the `# test_paths: []` line (so the examples persist as an editing reference). The placeholder substitution introduces a new `{TEST_PATHS}` token (see §2 for the create-mode flow).

### 2. fab-setup skill — non-interactive detection in create mode

`src/kit/skills/fab-setup.md`, **Config Create-Mode** (current steps at lines 157–166). Detection slots between reading root files (step 1) and writing the config:

- **Step 2** (currently "Ask the user: project name, description, source paths") gains a **detection sub-step**: after reading root files, apply the language→pattern table below to on-disk marker files and derive `{TEST_PATHS}` automatically. Do NOT prompt for it — the user explicitly wants this non-interactive.
- **Step 4** (placeholder substitution) gains `{TEST_PATHS}` alongside `{PROJECT_NAME}`, `{PROJECT_DESCRIPTION}`, `{SOURCE_PATHS}`.
- **Detection writes the value AND surfaces a note.** When detection fills `test_paths`, the create-mode output (step 7 / Config Output) adds a visible line, e.g.:
  `Detected {ecosystem} — set test_paths to {patterns}. Edit via /fab-setup config source_paths if wrong.`
  When no ecosystem is recognized, leave the key commented (`# test_paths: []`), the breakdown collapses to one line (today's behavior), and note: `No test convention detected — test_paths left empty (impact breakdown will show a single total). Set it later if desired.`

**Detection table** (marker file on disk → anchored pattern). Multi-marker repos take the union:

| Detected marker | Ecosystem | `test_paths` |
|---|---|---|
| `go.mod` | Go | `**/*_test.go` |
| `pytest.ini` / `pyproject.toml` / `setup.cfg` | Python (pytest) | `**/test_*.py`, `**/*_test.py` |
| `package.json` with jest/vitest dep, or `*.spec.ts`/`*.test.ts`/`*.spec.js`/`*.test.js` present | JS/TS | `**/*.spec.ts`, `**/*.test.ts`, `**/*.spec.js`, `**/*.test.js` |
| `pom.xml` / `build.gradle` | Java/Kotlin (Maven/Gradle) | `**/src/test/**` |
| `*.csproj` referencing a test SDK | .NET | `**/*Tests.cs`, `**/*Test.cs` |
| `Cargo.toml` | Rust | *(none — Rust tests are inline `#[cfg(test)]`; not glob-addressable)* → leave empty, note why |
| *(no marker / unrecognized)* | — | leave empty; standing examples remain the reference |

The Rust row is deliberate: a substring match would "work" there yet be doubly wrong (matches `attestation.rs`, misses the actual inline tests) — so the honest behavior is to leave it empty.

### 3. Migration — backfill existing repos

New file `src/kit/migrations/2.7.1-to-2.8.0.md` (next minor; bump `src/kit/VERSION` 2.7.1 → 2.8.0 as part of this change). Follows the established migration shape (cf. `2.2.0-to-2.3.0.md`): markdown instruction file, idempotent, applied by `/fab-setup migrations`. Two effects:

- **Refresh the scaffold comment block** in the user's existing `fab/project/config.yaml` to match §1 (the annotated example comment + `:(glob)`/anti-substring note), so existing repos get the editing reference even if they keep `test_paths` empty.
- **Detect and fill `test_paths`** by running the §2 detection table against the user's repo — *only when the key is currently absent or empty*.

**Pre-check / idempotency (Constitution III):**
1. If `fab/project/config.yaml` is absent → `Skipped: fab/project/config.yaml not present.`
2. If `test_paths` is already present and **non-empty** → do NOT overwrite the user's value; still refresh the comment block. (A user who hand-set patterns keeps them.)
3. Use a sentinel comment marker (e.g. the `# Examples (uncomment/adapt the line for your stack):` line) to detect whether the refreshed comment block is already present, so re-running is a no-op on the comment refresh.
4. If detection recognizes no ecosystem → leave `test_paths` empty, refresh the comment only, and report it.

Report lines mirror the create-mode notes in §2 (detected ecosystem + patterns, or "no convention detected").

### 4. Doc mirrors (sweep class)

Constitution-required and review-must-fix:

- `docs/specs/skills/SPEC-fab-setup.md` — mirror the create-mode detection sub-step + `{TEST_PATHS}` placeholder (Constitution Additional Constraints: skill change MUST carry SPEC mirror).
- `docs/memory/distribution/setup.md` — document that create-mode now detects/fills `test_paths`.
- `docs/memory/distribution/migrations.md` — note the new `2.7.1-to-2.8.0` migration if that file enumerates migrations (verify at apply).
- Re-run `fab memory-index` after any memory write (byte-stable index).

## Affected Memory

- `distribution/setup`: (modify) create-mode now auto-detects and fills `test_paths` from on-disk marker files (non-interactive), with a visible note; unrecognized stacks leave it empty.
- `distribution/migrations`: (modify) record the new `2.7.1-to-2.8.0` backfill migration (detection + comment refresh, idempotent).

## Impact

- **Skill**: `src/kit/skills/fab-setup.md` (create-mode steps 2 + 4 + 7 / Config Output).
- **Scaffold**: `src/kit/scaffold/fab/project/config.yaml` (`test_paths` comment block + `{TEST_PATHS}` placeholder).
- **Migration**: new `src/kit/migrations/2.7.1-to-2.8.0.md`; `src/kit/VERSION` bump 2.7.1 → 2.8.0.
- **SPEC mirror**: `docs/specs/skills/SPEC-fab-setup.md`.
- **Memory**: `docs/memory/distribution/setup.md`, `docs/memory/distribution/migrations.md`; regen indexes.
- **No Go change**: `impact.go` already consumes any non-empty `test_paths` verbatim; detection is skill/migration prompt logic per Constitution I. (Therefore no `_cli-fab.md` / Go-test obligation is triggered.)
- **Downstream effect**: new projects (and migrated existing ones) get the `impl`/`tests` impact split automatically for recognized stacks; the impact table delivers its full taxonomy without expert hand-config.

## Open Questions

- Confirm at apply whether `docs/memory/distribution/migrations.md` enumerates individual migrations (→ needs the new entry) or only describes the migration *mechanism* (→ no per-migration edit needed). Resolve by reading the file at apply; does not block planning.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Detection at setup is non-interactive (auto-fill + visible note), not a prompt | User explicitly asked "non-interactively if possible"; marker-file signal is strong; value is trivially editable and the migration is re-runnable | S:90 R:80 A:80 D:80 |
| 2 | Confident | Anchored language→pattern detection table (Go/Python/JS-TS/Java-Kotlin/.NET; Rust & unknown → empty) | Direction approved in discussion; anchoring to convention is what makes the classification reliable; rejected the unanchored substring | S:80 R:75 A:75 D:75 |
| 3 | Certain | Migration must skip repos with a non-empty `test_paths` and be sentinel-guarded for the comment refresh | Constitution III (idempotent operations); established migration pattern (`2.2.0-to-2.3.0` pre-check shape) | S:85 R:85 A:95 D:90 |
| 4 | Confident | New migration file `2.7.1-to-2.8.0.md` + `VERSION` bump to 2.8.0 (next minor) | Current VERSION is 2.7.1; additive config-only change is a minor bump per the existing migration cadence | S:75 R:80 A:85 D:80 |
| 5 | Confident | No Go binary change — detection is skill/migration prompt logic | Constitution I (Pure Prompt Play: workflow logic in markdown/scripts); `impact.go` already handles any non-empty `test_paths` | S:80 R:75 A:90 D:85 |
| 6 | Tentative | Unrecognized/inline-test stacks (e.g. Rust) leave `test_paths` empty rather than guessing | Rust tests aren't glob-addressable; a guess would be doubly wrong; empty = today's safe collapsed breakdown | S:70 R:80 A:70 D:55 |

6 assumptions (1 certain, 4 confident, 1 tentative, 0 unresolved).
