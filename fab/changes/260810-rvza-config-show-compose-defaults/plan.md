# Plan: config-show-compose-defaults

**Change**: 260810-rvza-config-show-compose-defaults
**Intake**: `intake.md`

## Requirements

### CLI: `fab config show` bare output

#### R1: Bare `show` composes the built-in defaults tier
Bare `fab config show` (no key, no `--origin`) MUST print the fully composed effective config —
the built-in defaults tier merged BENEATH project, system, and environment — as YAML. The
composition MUST reuse the existing `readModelDefaults` → `configref.DefaultsMap`/`DefaultsMapFor`
projection and the shared `config.MergeLayers` rule, in the same tier order the keyed path already
uses (`defaults < project < system < env`), so bare `show`, keyed `show`, and `--origin` can never
disagree about a value.

- **GIVEN** a repo whose `fab/project/config.yaml` sets no `dispatch:` block
- **WHEN** the user runs `fab config show`
- **THEN** the output contains `dispatch:` with `mode: native` (a pure built-in default)
- **AND** the command exits 0 and writes no file

#### R2: Composition obeys the read model's empty-skip rule
The composed output MUST resolve each leaf to the highest tier that defines it NON-EMPTY, so a
file/environment value still wins over the built-in default and an empty override still falls
through to the default instead of shadowing it.

- **GIVEN** `agent.workers: codex` in the project file and `FAB_AGENT_WORKERS=""` in the environment
- **WHEN** the user runs `fab config show`
- **THEN** the printed `agent.workers` is `codex` (project wins over the built-in `claude`; the
  empty environment leaf falls through)

#### R3: The read model is still loaded exactly once per invocation
Bare `show` MUST obtain the defaults projection from the ALREADY-LOADED layers (`readModelDefaults`),
never by re-running the cascade, so each fail-open loader warning is still printed exactly once.

- **GIVEN** a system config carrying a project-scoped key (which the loader prunes with a
  `fab: warning:`)
- **WHEN** the user runs `fab config show`
- **THEN** the warning appears exactly once on stderr

#### R4: `--origin` and keyed semantics are untouched
`fab config show --origin`, `fab config show <key>`, and `fab config show <key> --origin` MUST keep
their current output byte-for-byte. `--origin` remains the provenance annotator; it is no longer the
only route to the composed values. No new flag ships (no `--sparse`/`--set-only`).

- **GIVEN** a key defined at three tiers
- **WHEN** the user runs `fab config show agent.workers --origin`
- **THEN** the full-stack listing is exactly as before (winner `(effective)`, the rest `(shadowed)`)

### Docs: help text and reference surfaces

#### R5: The command's own help text states what it now does
`configShowCmd`'s `Long` MUST drop the "built-in defaults are NOT materialized here / they apply at
point-of-use and are only surfaced explicitly by `--origin`" carve-out and state that bare output is
the fully composed config including built-in defaults; the `Example` block MUST describe `--origin`
as the provenance annotator rather than as the route to composed values.

- **GIVEN** the change is applied
- **WHEN** a reader runs `fab config show --help`
- **THEN** no sentence claims defaults are unmaterialized in bare output, and the opening sentence
  matches the actual behavior

#### R6: Reference docs and stale in-code claims are swept
Every non-memory occurrence of the "bare `show` skips/never consults the defaults tier" claim MUST
be updated: `src/kit/skills/_cli-fab.md` § fab config show, its mirror
`docs/specs/skills/SPEC-_cli-fab.md`, `docs/specs/config.md` § Six intent-grouped verbs, and the
in-code comments that state it (`src/go/fab/cmd/fab/config.go`,
`src/go/fab/internal/config/config.go`'s `Layers.Effective` doc, and the test comment in
`config_show_init_test.go`). `docs/memory/` is deliberately EXCLUDED — memory is hydrate's job.

- **GIVEN** the repo after this change
- **WHEN** one greps `src/kit/`, `docs/specs/`, and `src/go/` for the claim that bare `show` does not
  materialize built-in defaults
- **THEN** there are no remaining occurrences
- **AND** `docs/memory/` is untouched by apply

### Non-Goals

- No `--sparse` / `--set-only` companion flag (intake assumption 3 — YAGNI; additive later)
- No change to the loader's merged tree (`config.LoadPath` / `Layers.Effective` still carry no
  defaults tier — the `agent.profiles` poisoning and the `configref → agent → config` import cycle
  reasons in `docs/specs/config.md` § The defaults tier is materialized for the READ MODEL stand)
- No `docs/memory/` edits (hydrate owns them)

### Design Decisions

#### Compose at the render seam, not in the loader
**Decision**: Bare `show` composes `config.MergeLayers(defaults, layers.Project, layers.System, layers.Env)`
inside `renderShow`, using the `defaults` map the command already computes for the keyed/`--origin`
paths.
**Why**: It is the same four-tier expression `renderShowKey` uses, so the three surfaces share one
tier order and one emptiness rule. Composing in `internal/config` instead would close an import
cycle and poison `Config.Agent.Profiles` for the resolver.
**Rejected**: `MergeLayers(defaults, layers.Effective)` — cheaper-looking, but not identical: a
non-map leaf in a middle tier replaces a subtree differently under the two fold orders, which would
let bare `show` diverge from keyed `show` in exactly the edge case provenance exists to explain.
*Introduced by*: 260810-rvza-config-show-compose-defaults

## Tasks

### Phase 1: Core Implementation

- [x] T001 In `src/go/fab/cmd/fab/config.go` `configShowCmd` RunE: compute `readModelDefaults(layers)` unconditionally (delete the `withOrigin || len(args) == 1` guard and its stale comment) <!-- R1 R3 -->
- [x] T002 In `src/go/fab/cmd/fab/config.go` `renderShow`: compose the defaults tier beneath project/system/env via `config.MergeLayers(defaults, layers.Project, layers.System, layers.Env)` for the non-`--origin` path and marshal that; leave `renderShowOrigin` untouched <!-- R1 R2 R4 -->

### Phase 2: Help Text & Docs Sweep

- [x] T003 Rewrite `configShowCmd`'s `Long` and `Example` in `src/go/fab/cmd/fab/config.go` — drop the "defaults are NOT materialized here" carve-out; describe `--origin` as provenance-only <!-- R5 -->
- [x] T004 [P] Update `src/kit/skills/_cli-fab.md` § fab config show — the mode table's bare row and the trailing "Bare output remains unchanged" claim <!-- R6 -->
- [x] T005 [P] Update the SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md`'s `fab config` row (constitution: a `src/kit/skills/*.md` change carries its SPEC mirror) <!-- R6 -->
- [x] T006 [P] Update `docs/specs/config.md` § Six intent-grouped verbs' `show` entry (bare output composes the defaults tier; `--origin` is provenance) <!-- R6 -->
- [x] T007 [P] Update the stale in-code claim in `src/go/fab/internal/config/config.go` (`Layers.Effective` doc comment) so it describes the loader tree without asserting what bare `show` prints <!-- R6 -->
- [x] T008 Grep `src/kit/`, `docs/specs/`, `src/go/` repo-wide for residual "bare show skips/never consults the defaults tier"-class claims and fix any missed occurrence; confirm `docs/memory/` is untouched <!-- R6 -->

### Phase 3: Tests

- [x] T009 In `src/go/fab/cmd/fab/config_show_init_test.go`: add a bare-`show` case asserting a pure-default field (`dispatch.mode: native`) appears on a bare project, and that a project value still wins over its built-in default <!-- R1 R2 -->
- [x] T010 Update `TestConfigShow_PrintsEffectiveConfig` and the `TestConfigLoaderWarningsAreNotDuplicated` comment for the composed bare output / unconditional projection; verify no other test pins sparse bare-show output <!-- R1 R3 -->
- [x] T011 Run `go test ./src/go/fab/cmd/fab/ ./src/go/fab/internal/config/ ./src/go/fab/internal/configref/` and `gofmt -l` on every touched `.go` file; fix any diff <!-- R1 R2 R3 R4 -->

## Execution Order

- T001 blocks T002 (same function's caller supplies `defaults`)
- T004 blocks T005 (the SPEC mirror restates the skill row)
- T009–T010 follow T002; T011 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: Bare `fab config show` prints the composed config with the built-in defaults tier merged beneath project/system/environment, reusing `readModelDefaults` + `config.MergeLayers` — `config.go:391-404`; verified end-to-end against a dev binary in this repo (system `dispatch.mode: pane` + `agent.workers: kimi`, defaults `column_width: 35`/`reap_done: true`, project `source_paths`/`project.*` all present in one document)
- [x] A-002 R2: Composed output honors empty-skip — a non-empty file/env leaf outranks the default, an empty one falls through to it (`TestConfigShow_ComposedOutputHonorsEmptySkip`, `TestConfigShowOrigin_EnvironmentNullFallsThrough`'s plain-`show` arm)
- [x] A-003 R3: Bare `show` resolves the read model once (defaults projected from the already-loaded layers), so a fail-open loader warning prints exactly once — `readModelDefaults(layers)` takes the loaded layers; `TestConfigLoaderWarningsAreNotDuplicated`'s `{"show"}` case is now a real (previously vacuous) assertion of this
- [x] A-004 R4: `--origin`, keyed `show`, and keyed `--origin` output is unchanged and no new flag was added — `renderShowOrigin`/`renderShowKey` untouched, `renderShow` only reorders the `withOrigin` branch, and `--origin` remains the sole registered flag
- [x] A-005 R5: `configShowCmd`'s `Long`/`Example` describe the composed bare output and `--origin` as provenance-only, with the carve-out sentence gone (`config.go:204-228`)
- [x] A-006 R6: `_cli-fab.md`, `SPEC-_cli-fab.md`, `docs/specs/config.md`, and the stale in-code comments all state the new behavior

### Behavioral Correctness

- [x] A-007 R1: A test asserts `dispatch.mode: native` (a pure built-in) appears in bare `show` on a bare project — the pain point from the intake is closed (`TestConfigShow_ComposesDefaultsTier`, `TestConfigShow_BareRepoReportsTheBuiltIns`; both assert `config.DefaultDispatchMode` rather than a literal)
- [x] A-008 R4: Existing `--origin`/keyed tests pass unmodified, proving the provenance surfaces were not disturbed — the test diff adds cases and rewords two comments; no existing `--origin`/keyed assertion changed

### Scenario Coverage

- [x] A-009 R2: A test covers project-over-default precedence in bare composed output (`TestConfigShow_ComposesDefaultsTier` — `workers: codex` present, `workers: claude` absent)
- [x] A-010 R3: `TestConfigLoaderWarningsAreNotDuplicated`'s bare-`show` case still asserts exactly one warning with the projection now unconditional

### Edge Cases & Error Handling

- [x] A-011 R1: A repo with no project/system config no longer under-reports — bare `show` prints the built-in defaults rather than an empty-config note (`TestConfigShow_BareRepoReportsTheBuiltIns`)
- [x] A-012 R2: An explicit environment `null` still falls through to the file value in bare composed output (`TestConfigShowOrigin_EnvironmentNullFallsThrough` plain arm: `FAB_AGENT_WORKERS=null` + project `project-worker` ⇒ `workers: project-worker`)

### Code Quality

- [x] A-013 Pattern consistency: New code follows the file's existing naming, comment, and tier-order conventions (`defaults < project < system < env`, as in `renderShowKey`) — the composed expression is `renderShowKey`'s four-layer form with the subtree isolation dropped
- [x] A-014 No unnecessary duplication: The defaults projection and the merge rule are reused, not reimplemented (intake assumption 4)
- [x] A-015 Canonical source only: kit edits land in `src/kit/skills/`, never in the gitignored `.claude/skills/` deployed copy — no `.claude/` path in the changed-file list
- [x] A-016 SPEC-mirror sync: the `src/kit/skills/_cli-fab.md` edit ships with its `docs/specs/skills/SPEC-_cli-fab.md` mirror
- [x] A-017 CLI ⇒ docs + tests: the command-behavior change updates `_cli-fab.md` and ships Go test updates (three new bare-`show` cases; see review finding on `TestConfigShow_PrintsEffectiveConfig`'s now-vacuous system-tier assertion)
- [x] A-018 Sibling sweep: the "bare show skips the defaults tier" claim class was grepped repo-wide and updated everywhere outside `docs/memory/` — re-verified at review over `src/kit/`, `docs/specs/` (incl. `glossary.md`/`skills.md`/`architecture.md`), `src/go/`, and `README.md`; the residual `point-of-use`/`no defaults tier` claims all describe the LOADER tree and remain true
- [x] A-019 No migration needed: the change is a pure query with no user-data restructuring, so no `src/kit/migrations/` file is required
- [x] A-020 Tests green and `gofmt` clean on every touched `.go` file — `go test ./...` for the whole `src/go/fab` module is green, `go vet` clean, `gofmt -l` empty on all three touched files

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/cmd/fab/config.go:396-400` (the `len(composed) == 0` guard and its `# (no effective config — the built-in defaults projection is empty)` note) — the change made the whole branch unreachable in practice: the registry projection always defines rows, and `configref.DefaultsMap`/`DefaultsMapFor` return an *error* (never an empty map) on registry drift, which `renderShow`'s caller already surfaces. Plan assumption 2 deliberately keeps it as a fail-visible fallback; recorded here as the honest answer to the prompt, not a recommendation to delete.
- `src/go/fab/internal/config/config.go:452-457` `Layers.Effective` — NOT a deletion candidate, listed to close the question: `renderShow` no longer reads it, but `readModelDefaults` still feeds it to `config.FromMap` (and `LoadPath` unmarshals it), so the field keeps two live consumers.
- Nothing else — the change replaces one render expression and rewords prose; it introduced no new symbol, flag, or file that shadows existing code.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Compose with the explicit four-layer `MergeLayers(defaults, Project, System, Env)` rather than `MergeLayers(defaults, layers.Effective)` | The four-layer form is byte-identical to what `renderShowKey` already does; the two-layer fold differs when a middle tier holds a non-map over a lower map | S:85 R:90 A:90 D:85 |
| 2 | Confident | Keep an empty-composed guard in `renderShow` as a defensive note instead of printing a bare `{}` | The registry projection is never empty in practice, so the branch is a fail-visible fallback, not a live path; deleting it would make a projection regression print `{}` silently | S:60 R:90 A:75 D:65 |
| 3 | Confident | Reword rather than delete `Layers.Effective`'s doc comment | The loader tree genuinely still carries no defaults tier; only its claim about what bare `show` prints is now stale | S:75 R:90 A:85 D:80 |
| 4 | Confident | Sweep the SPEC mirror `SPEC-_cli-fab.md` even though the changed phrase is one clause in a long row | Constitution + code-review policy read the skill⇒SPEC mirror rule strictly | S:80 R:85 A:85 D:80 |

4 assumptions (1 certain, 3 confident, 0 tentative).
