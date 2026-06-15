# Intake: Add `.status.yaml` `summary:` field + migration

**Change**: 260615-5943-status-summary-field
**Created**: 2026-06-15

## Origin

Backlog item `[5943]` (2026-06-15), the **FOUNDATION** change of the four-part FKF
("Fab folder log") bundle. Initiated via `/fab-new 5943` (one-shot, from the backlog
description). No prior conversation discussion — the design is fully specified by the
backlog text and confirmed by the spec at `docs/specs/fkf.md` §6.3.

> [5943] FKF Change 1/4 (feat, FOUNDATION — no deps): Add .status.yaml `summary:` field + migration.
> The C-lite log.md generator (FKF Change 2) reads this per-change one-line summary, so it MUST exist
> first. Tasks: (a) add `Summary string \`yaml:"summary,omitempty"\`` to StatusFile struct in
> src/go/fab/internal/statusfile/statusfile.go — decode in Load(), encode in syncToRaw() via insertKey();
> (b) add `fab status set-summary <change> <text>` + `get-summary` CLI verbs (the conflict-free write path
> — each change touches only its own .status.yaml); (c) add `summary: ""` to template
> src/kit/templates/status.yaml; (d) migration src/kit/migrations/2.4.1-to-2.5.0.md adding the field to
> in-flight changes under fab/changes/ (skip archive/**, idempotent); bump src/kit/VERSION to 2.5.0.
> Docs: _cli-fab.md § fab status, SPEC-fab-status.md mirror, docs/specs/templates.md, memory
> pipeline/change-lifecycle.md + schemas.md. Graceful absence: a change with no summary projects its slug
> in the log. Spec: docs/specs/fkf.md §6.3. Context PR #419.

## Why

1. **What problem this solves.** The FKF bundle replaces hand-appended per-domain
   changelogs with a **C-lite generated `log.md`**: `fab memory-index` joins git
   history (which files a commit touched) with a per-change one-line *summary* to
   emit a conflict-free, descriptive memory changelog. That summary line has to
   live *somewhere* per-change, and the chosen home — per the §6.3 design — is the
   change's own `.status.yaml`. This change creates that home.

2. **Why it must ship first (FOUNDATION, no deps).** FKF Change 2 (the `log.md`
   generator) *reads* `summary:` while walking changes. If the field does not yet
   exist in the schema, the template, and in-flight changes, the generator has
   nothing to read. This change is the structural prerequisite for the whole
   bundle; it carries no dependency of its own.

3. **Why `.status.yaml` and not a shared `log.md` line.** Each change touches only
   *its own* `.status.yaml`, so a summary written there has **zero merge-conflict
   surface** — the entire point of C-lite over OKF's hand-appended `log.md` (which
   just relocates the same-day collision from N memory files into one `log.md`) and
   over a pure slug-only git projection (conflict-free but loses the *what-changed*
   signal an agent needs for archaeology). C-lite keeps the descriptive line **and**
   stays conflict-free. (§6.1 rationale.)

4. **Consequence of not doing it.** FKF Change 2 cannot be built; the bundle stalls.

## What Changes

A `.status.yaml` **schema change** (new optional string field) plus its full
plumbing: struct, round-trip serialization, CLI verbs, template, migration, version
bump, and the doc/memory/spec mirrors. Modeled end-to-end on the existing optional
string field `change_type_source` (the closest sibling — same `omitempty`,
drop-when-empty, insert-when-absent shape).

### 1. `StatusFile` struct field (`src/go/fab/internal/statusfile/statusfile.go`)

Add a `Summary` field to the `StatusFile` struct (currently ends at `LastUpdated`,
lines 103–124). Use `omitempty` so an absent/empty summary serializes to nothing:

```go
type StatusFile struct {
    ID         string `yaml:"id"`
    // ... existing fields ...
    PRs         []string    `yaml:"prs"`
    TrueImpact  *TrueImpact `yaml:"true_impact,omitempty"`
    Summary     string      `yaml:"summary,omitempty"`   // NEW — per-change one-line log summary (C-lite source, §6.3)
    LastUpdated string      `yaml:"last_updated"`

    raw *yaml.Node
}
```

Field placement note: in the struct the `yaml` decode tags drive `Load()`, so
declaration order does not affect decode. For **write** ordering see §2 (the
`syncToRaw` insert position is what controls where the key lands in the file).

### 2. Round-trip serialization (`Load()` + `syncToRaw()` in the same file)

`Load()` (line 131) decodes via struct tags — the new tag is picked up automatically;
**no manual decode edit is required** beyond the struct tag. `syncToRaw()` (line 414)
is the field-preserving writer and **does** need an explicit case + insert clause,
mirroring `change_type_source` exactly:

- **In the `for` loop's `switch key` block** — add a `case "summary":` that drops the
  key when empty (so an empty summary round-trips to *absent*, matching `omitempty`)
  and otherwise sets the value:

  ```go
  case "summary":
      // Empty == no summary: drop the key rather than emit an empty scalar,
      // so an absent field stays absent (back-compat round-trip).
      if sf.Summary == "" {
          dropKeyAt(root, i)
          i -= 2
      } else {
          val.Value = sf.Summary
      }
  ```

- **In the post-loop "insert if absent" block** — add, mirroring the
  `change_type_source` insert (lines 474–476), so a summary set on a sparse legacy
  document that lacks the key gets created on write:

  ```go
  if !seen["summary"] && sf.Summary != "" {
      insertKey(root, "summary", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: sf.Summary})
  }
  ```

  `insertKey` (line 525) places the key immediately before `last_updated`, which is
  the desired position (`summary` between `true_impact`/`prs` and `last_updated`).

### 3. CLI verbs — `set-summary` + `get-summary` (`src/go/fab/cmd/fab/status.go` + `src/go/fab/internal/status/`)

Register two new subcommands on `statusCmd()` (line 21 `AddCommand` block):

- **`fab status set-summary <change> <text>`** — the conflict-free write path. Mirror
  `statusSetChangeTypeCmd` (line 327): `cobra.ExactArgs(2)`, body goes through
  `withStatusLock`, delegating to a new `status.SetSummary(st, statusPath, args[1])`
  package function (parallel to `status.SetChangeType`).

  ```go
  func statusSetSummaryCmd() *cobra.Command {
      return &cobra.Command{
          Use:   "set-summary <change> <text>",
          Short: "Set the per-change log summary",
          Args:  cobra.ExactArgs(2),
          RunE: func(cmd *cobra.Command, args []string) error {
              return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
                  return status.SetSummary(st, statusPath, args[1])
              })
          },
      }
  }
  ```

- **`fab status get-summary <change>`** — read-only. Mirror `statusGetIssuesCmd`
  (line 477): `cobra.ExactArgs(1)`, load via the lock-free `loadStatus` reader, print
  `st.Summary` to stdout. Empty summary prints an empty line (graceful — callers like
  the generator fall back to the slug; see §6.3 "graceful absence").

  Add both to the `AddCommand(...)` list. The new `status.SetSummary` helper sets
  `st.Summary = text` then `Save`s (look at `status.SetChangeType` for the exact
  save/last_updated-touch pattern to copy).

### 4. Template (`src/kit/templates/status.yaml`)

Add `summary: ""` to the template so freshly-created changes carry the key. Place it
to match the §2 write order — between `prs: []` and the `# true_impact` comment /
`last_updated`. (An empty value is fine; the writer drops it on the next mutation per
the `omitempty` round-trip, but seeding it in the template documents the field for
humans reading a fresh change.)

```yaml
prs: []
summary: ""
# true_impact: lazily created on first apply-finish (no placeholder here).
last_updated: {CREATED}
```

### 5. Migration (`src/kit/migrations/2.4.2-to-2.5.0.md`)

> **Filename deviation from the backlog**: the backlog text says `2.4.1-to-2.5.0.md`,
> but the live `src/kit/VERSION` is already **2.4.2** (the backlog line predates the
> 2.4.2 release). The migration MUST be named for the *current* version → target, i.e.
> **`2.4.2-to-2.5.0.md`** (see Assumptions row 1). Bump `src/kit/VERSION` 2.4.2 → 2.5.0.

A `.status.yaml` **schema change requires a migration** (project data-migration rule).
The migration adds `summary: ""` (or the omitempty-absent equivalent) to in-flight
changes. Follow the existing migration format (`2.2.0-to-2.3.0.md` is the latest
reference — Summary / Pre-check / Changes sections):

- **Pre-check**: skip if `fab/changes/` is absent.
- **Scope**: iterate `fab/changes/*/.status.yaml` only — **skip `fab/changes/archive/**`**
  (the single-level glob already excludes it; state it explicitly).
- **Idempotent**: skip any `.status.yaml` that already has a `summary:` key (re-running
  is a no-op). Because the field is `omitempty`, a defensible alternative is to add
  *nothing* to in-flight files and rely on graceful-absence — but the backlog explicitly
  says "adding the field to in-flight changes," so the migration writes `summary: ""`
  (see Assumptions row 3).
- Insert position mirrors the template: before `last_updated`.

### 6. Documentation, spec, and memory mirrors

Schema-change ripple — update every place the `.status.yaml` schema or `fab status`
verbs are documented:

- **`.claude/skills/_cli-fab/SKILL.md`** § fab status — add `set-summary` / `get-summary`
  to the verb list.
- **`docs/specs/skills/SPEC-fab-status.md`** — mirror the same two verbs (keep in sync
  with `_cli-fab`).
- **`docs/specs/templates.md`** — document the new `summary:` template field.
- **`docs/memory/pipeline/change-lifecycle.md`** — note where/when `summary` is authored
  (per §6.3: "written once during the change — authored at hydrate, or carried from the
  intake"). This change only *creates the field*; it does not wire authoring into any
  stage (that is out of scope — see Non-Goal below).
- **`docs/memory/pipeline/schemas.md`** — add `summary` to the `.status.yaml` schema doc.

### Non-Goals

- **No authoring wiring.** This change creates the field + write/read verbs only. It does
  NOT make hydrate (or intake, or any stage) automatically *populate* `summary`. §6.3 says
  the line is "authored at hydrate, or carried from the intake" — that wiring belongs to a
  later FKF change, not the FOUNDATION. (Assumptions row 2.)
- **No `log.md` generator.** That is FKF Change 2, which *consumes* this field.
- **No `fab memory-index` changes.** The generator read happens in Change 2.

## Affected Memory

- `pipeline/change-lifecycle.md`: (modify) note the new `summary:` field and where it is authored (hydrate / carried from intake)
- `pipeline/schemas.md`: (modify) add `summary` (optional string) to the documented `.status.yaml` schema

## Impact

**Code:**
- `src/go/fab/internal/statusfile/statusfile.go` — struct field + `syncToRaw` case + insert clause
- `src/go/fab/internal/status/` — new `SetSummary` helper (mirror `SetChangeType`)
- `src/go/fab/cmd/fab/status.go` — two new subcommands + `AddCommand` registration

**Kit:**
- `src/kit/templates/status.yaml` — seed `summary: ""`
- `src/kit/migrations/2.4.2-to-2.5.0.md` — new migration
- `src/kit/VERSION` — 2.4.2 → 2.5.0

**Docs / spec / memory:** `_cli-fab.md`, `SPEC-fab-status.md`, `templates.md`,
`change-lifecycle.md`, `schemas.md` (all listed above).

**Tests:** `src/go/fab/internal/statusfile/*_test.go` (round-trip of `summary` set/empty/absent),
`src/go/fab/internal/status/status_test.go` (`SetSummary`), and `src/go/fab/cmd/fab/status_test.go`
or equivalent for the CLI verbs (follow the existing `set-change-type` / `get-issues` test patterns).

**Dependencies / systems:** No external deps. Downstream consumer is FKF Change 2 (the
`log.md` generator), which is a separate change and is unblocked by this one.

## Open Questions

- None blocking. The version-target, authoring-scope, and migration-write-content questions
  are all resolved as graded assumptions below (rows 1–3) with clear codebase/spec signal.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Migration is named `2.4.2-to-2.5.0.md` (not the backlog's `2.4.1-to-2.5.0.md`); VERSION bumps 2.4.2 → 2.5.0 | Live `src/kit/VERSION` is 2.4.2; the backlog line predates the 2.4.2 release. Migrations must be named current→target. Strong codebase signal, one obvious interpretation, trivially reversible. | S:80 R:90 A:95 D:85 |
| 2 | Confident | Scope is field + read/write verbs only; no stage auto-populates `summary` (authoring wiring deferred to a later FKF change) | Backlog scopes Change 1 as "Add field + migration"; §6.3 names hydrate/intake as the *eventual* author but FKF Change 1 is FOUNDATION with no-deps. Adding stage wiring would over-reach the ticket. | S:75 R:80 A:80 D:75 |
| 3 | Confident | Migration writes `summary: ""` into in-flight `fab/changes/*/.status.yaml` (rather than relying solely on omitempty graceful-absence) | Backlog explicitly says "adding the field to in-flight changes under fab/changes/". Idempotent + skips archive/**. omitempty would make a no-write defensible, but the ticket is explicit. | S:80 R:85 A:85 D:70 |
| 4 | Certain | Model the field on `change_type_source` (omitempty optional string, drop-when-empty in syncToRaw, insert-when-absent) | The codebase has an exact existing pattern (lines 112, 435–443, 474–476); template rule says follow existing patterns. Deterministic. | S:90 R:90 A:100 D:95 |
| 5 | Certain | `set-summary` mirrors `set-change-type` (withStatusLock + status.SetSummary); `get-summary` mirrors `get-issues` (loadStatus reader) | Direct sibling commands already exist (status.go lines 327, 477); copying their shape is the obvious, deterministic choice. | S:90 R:90 A:100 D:95 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).
