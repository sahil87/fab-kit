# Intake: Preserve config.yaml Comments on fab_version Stamping

**Change**: 260708-yogn-preserve-config-comments-fab-version
**Created**: 2026-07-08

## Origin

Promptless dispatch via `/fab-proceed` (Create-Intake Procedure, `{questioning-mode} = promptless-defer`), from a synthesized `/fab-discuss` conversation description dated 2026-07-08:

> `setFabVersion` (`src/go/fab-kit/internal/init.go:83`) stamps `fab_version` into `fab/project/config.yaml` by unmarshalling the whole file into a plain `map[string]interface{}` and re-marshalling with `yaml.Marshal`. This destroys every comment, alphabetizes all keys, normalizes indentation, and collapses comments-only mapping keys to `null`. Decision: fix via a targeted line splice that preserves the file byte-for-byte except the one line the function owns.

All design decisions below were made in that conversation; no questions were asked (promptless mode — any residual ambiguity is recorded in `## Assumptions`).

## Why

1. **The pain point.** `setFabVersion` owns exactly one scalar (`fab_version`) but rewrites the entire `fab/project/config.yaml` to update it: it unmarshals the whole file into `map[string]interface{}` and re-marshals with `yaml.Marshal`. That round-trip (a) strips every comment, (b) alphabetizes all keys, (c) normalizes indentation to 4-space, and (d) collapses any comments-only mapping key to `null` — e.g. a backfilled commented `agent:` template block becomes `agent: null`.

2. **The blast radius.** The function is called from BOTH `Init` (`src/go/fab-kit/internal/init.go:43`) and `Upgrade` (`src/go/fab-kit/internal/upgrade.go:120`), so **every `fab upgrade-repo` run mashes the user's config**. Observed 2026-07-03: a 2.13.1→2.13.3 upgrade wiped the providers comment template that migration `2.13.1-to-2.13.2` had backfilled — and that migration never re-applies because `fab/.kit-migration-version` is already past it, so the loss is permanent per-repo. Recurred 2026-07-08 on a 2.14.0 upgrade; a user's config had to be hand-restored. Migration files are innocent — none of the recent ones touch config.yaml.

3. **Why this approach.** A targeted line splice is minimal, byte-preserving for everything the function does not own, and trivially testable. The alternative — a `yaml.Node` round-trip (pattern exists in the sibling module at `src/go/fab/internal/statusfile/statusfile.go`) — is more general but still re-normalizes indentation/style details, and it is overkill for one guaranteed-top-level scalar. Additionally, `setFabVersion` is slated for deletion by a separate, already-designed `fab config upgrade` effort (to be executed by other agents); this change is the deliberate minimal **stopgap** so upgrades stop wiping user configs in the meantime.

## What Changes

### 1. Rewrite `setFabVersion` as a targeted line splice

File: `src/go/fab-kit/internal/init.go` (function `setFabVersion`, currently lines 82–108). Replace the `map[string]interface{}` unmarshal → `yaml.Marshal` → write pipeline with line-level splicing that preserves the file byte-for-byte except the single line the function owns. Exact behavior contract:

- **File missing** → create it (including parent dirs, as today via `os.MkdirAll`) containing just `fab_version: <version>`. This is the current new-file behavior; the existing `TestSetFabVersion_NewFile` keeps passing.
- **File exists with a top-level `fab_version:` line** — a line starting at column 0 with `fab_version:` (an indented or `#`-commented occurrence does NOT count) → replace that line's value in place, preserving everything else exactly (all comments, key order, indentation, blank lines, comments-only mapping blocks).
  - **Trailing same-line comment**: preserve it (`fab_version: 1.2.3  # pinned` → `fab_version: <new>  # pinned`) by replacing only the value token. If that proves fiddly during apply, replacing the whole line with `fab_version: <version>` is an acceptable fallback — the choice made MUST be noted in the plan/result.
  - **Quoted existing value** (`fab_version: "0.42.0"`): replacement MAY write the new value unquoted — `readFabVersion` (`src/go/fab-kit/internal/config.go:88`) unmarshals into a string field and parses both forms.
- **File exists without a top-level `fab_version:` line** → append `fab_version: <version>` as a new line at the end, ensuring the file ends with exactly one trailing newline.
- **Postconditions** (all cases): the result still parses as YAML, and `readFabVersion(path)` returns the new version.
- Read errors other than not-exist, and files that are not parseable YAML, should keep failing loudly as today (the function currently returns `cannot parse existing config.yaml` — the splice no longer needs a parse to write, but the postcondition tests enforce that output remains valid YAML; keep the error contract sensible and note any deliberate deviation).

Illustrative before/after for the recurring failure case:

```yaml
# before (user's config, post-migration backfill)
# Providers reference: run `fab config reference`
# agent:
#     tiers: ...
fab_version: 2.13.1
project:
    name: my-repo   # my main repo
```

```yaml
# after setFabVersion(path, "2.14.0") — ONLY the fab_version line changed
# Providers reference: run `fab config reference`
# agent:
#     tiers: ...
fab_version: 2.14.0
project:
    name: my-repo   # my main repo
```

(Today's implementation instead emits an alphabetized, comment-free file.)

### 2. Tests (same change, per constitution Test Integrity + test-alongside)

In `src/go/fab-kit/internal/init_test.go`, keep the existing `TestSetFabVersion_NewFile` / `TestSetFabVersion_ExistingFile` green and add preservation coverage:

- Comments preserved: header comments, inline (same-line) comments, and comment-only blocks survive the call byte-for-byte.
- Comments-only mapping key (e.g. a fully commented `agent:` template block) is NOT collapsed to `agent: null`.
- Key order preserved (non-alphabetical input stays non-alphabetical).
- Indentation untouched (e.g. 2-space nested mapping stays 2-space).
- Replace-if-present: top-level `fab_version:` value updated in place.
- Append-if-missing: file without `fab_version:` gains it as the last line, exactly one trailing newline.
- New-file creation (existing test).
- Quoted-value replacement (`fab_version: "0.42.0"` → new version; readback works).
- Readback via `readFabVersion` returns the new version in every case.

Run `go test ./internal/...` from `src/go/fab-kit/` before considering apply done (scope-first per code-quality.md; `TestUpgrade_*` in `upgrade_test.go` also exercises the seam via `readFabVersion` readbacks and must stay green).

### Non-goals / scope constraints

- Go-only change in `src/go/fab-kit/internal/` (function `setFabVersion` + tests). No CLI command signature changes → no `src/kit/skills/_cli-fab.md` update needed. No skill file changes → no `docs/specs/skills/SPEC-*.md` mirrors.
- No migration file: this change restructures no user data — it stops the binary from destroying it. Nothing in `fab/` layout or `.status.yaml` changes shape.
- Does NOT attempt the general comment-preserving config writer — that is the separate, already-designed `fab config upgrade` effort which will delete this function entirely.

## Affected Memory

- `distribution/distribution.md`: (modify) note that `fab init`/`fab upgrade-repo` version stamping is now byte-preserving (targeted `fab_version` line splice; no longer rewrites/normalizes config.yaml)
- `distribution/kit-architecture.md`: (modify) fab-kit lifecycle note — `setFabVersion` is a line splice (stopgap until the `fab config upgrade` effort deletes it), version threading from init/upgrade unchanged

## Impact

- **Code**: `src/go/fab-kit/internal/init.go` (`setFabVersion`, ~25 lines rewritten), `src/go/fab-kit/internal/init_test.go` (new preservation tests). Call sites `init.go:43` and `upgrade.go:120` unchanged.
- **Behavior**: `fab init` and `fab upgrade-repo` stop destroying user config.yaml comments/ordering/indentation. New-file output changes from a marshalled one-key map to the literal line `fab_version: <version>` (equivalent content).
- **Dependencies**: none added; the `gopkg.in/yaml.v3` import in `init.go` may become unused in this file (remove if so — it stays used elsewhere in the package, e.g. `config.go`).
- **Systems**: no CLI signature change, no kit content change, no migration, no `.status.yaml` schema change.
- **Tests**: `go test ./internal/...` in `src/go/fab-kit/` is the gate; `upgrade_test.go`'s readback assertions double as integration coverage of the splice.

## Open Questions

- None — all design decisions were resolved in the originating discussion; residual micro-decisions are graded in `## Assumptions` below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix via targeted line splice in `setFabVersion`, byte-preserving except the one owned line | Discussed — user chose splice over the `yaml.Node` round-trip (rejected as still-normalizing and overkill for one top-level scalar; function is slated for deletion by the separate `fab config upgrade` effort) | S:95 R:70 A:90 D:90 |
| 2 | Certain | File-missing case keeps current behavior: create file containing just `fab_version: <version>` | Discussed — explicitly required so `TestSetFabVersion_NewFile` keeps passing | S:95 R:90 A:95 D:95 |
| 3 | Confident | Trailing same-line comment: attempt value-token replacement preserving the comment; whole-line replacement is an acceptable fallback, choice noted | Discussed — description names both options and pre-authorizes the fallback; which lands is an apply-time detail | S:85 R:85 A:75 D:60 |
| 4 | Certain | Quoted existing values may be rewritten unquoted | Discussed — `readFabVersion` parses both forms (string-field unmarshal) | S:90 R:90 A:85 D:80 |
| 5 | Certain | "Top-level `fab_version:`" means a line starting at column 0 with `fab_version:`; indented/commented occurrences don't match | Discussed — stated verbatim in the description; matches YAML top-level semantics for this file | S:90 R:85 A:90 D:85 |
| 6 | Certain | Missing-key case appends `fab_version: <version>` at EOF with exactly one trailing newline | Discussed — stated verbatim in the description | S:90 R:85 A:90 D:85 |
| 7 | Certain | Scope is Go-only (`src/go/fab-kit/internal/`): no `_cli-fab.md` update (no signature change), no SPEC mirrors (no skill files), tests ship in-change | Constitution Additional Constraints + code-quality.md give a clear answer for this scope | S:90 R:80 A:95 D:90 |
| 8 | Confident | No migration file ships with this change | The migration rule covers restructuring user data; this change restructures nothing — it stops the binary from mangling existing files. No `fab/`-layout or `.status.yaml` shape change | S:70 R:80 A:80 D:75 |
| 9 | Confident | Affected memory limited to `distribution/` notes (distribution.md + kit-architecture.md one-line behavior updates at hydrate) | Domain index maps `fab init`/`upgrade-repo`/fab-kit lifecycle to these two files; change is behavioral (user-visible preservation), so memory notes are warranted but small | S:60 R:90 A:80 D:70 |
| 10 | Confident | Line-ending handling: treat content as LF-delimited (Go/POSIX norm here); untouched lines keep their original bytes regardless, so risk is confined to the one owned line | Not discussed — but the splice design preserves all untouched bytes by construction, and the repo/toolchain is LF-only | S:35 R:90 A:75 D:70 |

10 assumptions (6 certain, 4 confident, 0 tentative, 0 unresolved).
