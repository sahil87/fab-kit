# Intake: Status Query `--json`

**Change**: 260717-jx4w-status-query-json
**Created**: 2026-07-18

## Origin

One-shot `/fab-new jx4w` from the backlog. Raw backlog entry (`fab/backlog.md`):

> [jx4w] 2026-07-18: Toolkit principle №2 (stdout=data + `--json` on programmatically-consumed commands) — the `fab status` query subcommands (`confidence`, `plan`, `progress-map`, `get-issues`, `get-prs`, `get-summary`, `status.go:~122/175/202-205/221-225`) emit bespoke `key:value\n` lines that agents parse by hand and offer no `--json`. Deferred (systemic): add `--json` emitting a stable object schema across the status query surface. (The YAML emitters — `preflight`/`impact`/`score` — already produce a stable machine-parseable schema and are NOT flagged.) Deferred per change 260717-ptwh (principles audit) — whole-surface addition, not one-command. Any signature change updates `src/kit/skills/_cli-fab.md` + tests. shll v0.0.23.

Deferred out of the 260717-ptwh principles audit as a whole-surface change. No prior conversation in this session — intake decisions below come from the backlog text, the toolkit principles standard (`shll standards principles`, №2), and the existing code/precedent.

## Why

1. **Pain point**: Toolkit principle №2 (MUST) requires commands whose output is consumed programmatically to offer a machine-readable format, and the constitution's Toolkit Standards article binds this repo to that standard. The `fab status` read-only query subcommands emit bespoke formats — `key:value` lines (`plan`, `confidence`), `stage:state` pairs (`progress-map`, `display-stage`), and bare line-per-item lists (`get-issues`, `get-prs`, `all-stages`) — that agents and scripts parse by hand (e.g., `git-pr.md` reads `get-issues` line output; historical preflight consumed `progress-map` via `while IFS=: read -r`).
2. **Consequence if unfixed**: hand-rolled parsers break on any wording/format change with no versioning contract; every consumer re-invents extraction; the repo stays non-conformant with a MUST-grade standard it is constitutionally bound to.
3. **Why this approach**: additive per-subcommand `--json` flags following the two shipped in-repo precedents (`fab dispatch status --json`, `fab config reference --json`) — no breaking change to existing text output, no new top-level command, stable object schemas evolving additively per principle №2's stability rule. The YAML emitters (`fab preflight`, `fab impact`, `fab score`) already satisfy the principle and are out of scope.

## What Changes

### Scope: `--json` on the full `fab status` read-only query surface

Add a `--json` boolean flag to **nine** query subcommands in `src/go/fab/cmd/fab/status.go`. The backlog names six; three more (`current-stage`, `display-stage`, `all-stages`) are included for whole-surface uniformity — the backlog's own line refs (~175 = `display-stage`) and its "across the status query surface" framing support the superset.

**Excluded, deliberately**:
- `progress-line` — visual decoration for human status lines, not programmatic data (principle №2 scopes the MUST to programmatically-consumed output)
- `validate-status-file` — its contract is the exit code; it emits no data

### Per-subcommand JSON shapes

Keys are snake_case matching the `.status.yaml` field names. Lists that are ordered use JSON arrays (Go maps marshal alphabetically and would destroy pipeline stage order). Empty lists emit `[]`, never `null`.

| Subcommand | `--json` output shape |
|------------|----------------------|
| `confidence <change>` | `{"certain":2,"confident":3,"tentative":1,"unresolved":0,"score":4.2}` |
| `plan <change>` | `{"generated":true,"task_count":12,"acceptance_count":10,"acceptance_completed":3}` — same values as the text path, including the live-acceptance preference (`status.LiveAcceptance` over the cached counter); compute once, branch only on rendering |
| `progress-map <change>` | `[{"stage":"intake","state":"done"},{"stage":"apply","state":"active"},…]` — array to preserve stage order |
| `display-stage <change>` | `{"stage":"apply","state":"active"}` |
| `current-stage <change>` | `{"stage":"apply"}` |
| `all-stages` | `["intake","apply","review","hydrate","ship","review-pr"]` |
| `get-issues <change>` | `["DEV-988"]` (empty → `[]`) |
| `get-prs <change>` | `["https://github.com/…/pull/42"]` (empty → `[]`) |
| `get-summary <change>` | `{"summary":"…"}` — object (not a bare string) so fields can be added additively; empty summary → `{"summary":""}` |

### Flag + emit mechanics (follow `fab dispatch status --json` precedent)

- Per-subcommand `cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")`
- Named `xxxJSON` struct types with `json:` tags (see `dispatchStatusJSON` in `src/go/fab/cmd/fab/dispatch_status.go`)
- Emit via `json.NewEncoder(cmd.OutOrStdout())` with `enc.SetIndent("", "  ")` — indented, trailing newline via `Encode`
- **No `schema_version` field** — stability is the additive-evolution rule (new fields optional, per principle №2); matches both existing fab `--json` surfaces, neither of which carries a version field

### Back-compat: text output byte-identical

The default (no-flag) output of every touched subcommand stays **byte-identical**. `--json` is purely additive. Existing consumers (`git-pr.md`'s `get-issues` line parse, any hand parsers) keep working unchanged; migrating them to `--json` is a follow-up, not this change.

### Docs + tests (constitution-required)

- `src/kit/skills/_cli-fab.md` § fab status — note `--json` (and its shape) on each of the nine query rows
- `docs/specs/skills/SPEC-_cli-fab.md` — SPEC mirror updated in the same change (mirror-sweep class; on a CLI-signature change treat all of the skill's SPEC mirrors as the sweep class)
- `src/go/fab/cmd/fab/status_test.go` — a `--json` test case per subcommand (shape, empty-list `[]`, empty-summary, live-acceptance parity between text and JSON paths)
- `fab help-dump` reflects the new flags by construction (generated by walking the command tree); its fidelity tests need no fixture regeneration
- Apply-time sweep: grep the repo for other files restating the affected line formats (e.g., aggregate specs) and update any in the class

## Affected Memory

- `pipeline/schemas.md`: (modify) the workflow-schema authority documenting the `fab status` CLI surface (it already describes `get-summary`'s print behavior and `status plan`'s live-acceptance read path) — add the `--json` query contract and shapes

## Impact

- `src/go/fab/cmd/fab/status.go` — nine subcommand constructors gain the flag + JSON render branch; `loadStatus` reader path unchanged (queries stay lock-free)
- `src/go/fab/cmd/fab/status_test.go` — new `--json` cases
- `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md` — doc + mirror
- `docs/memory/pipeline/schemas.md` — hydrate target
- No skill behavior changes; no `.status.yaml` schema change; no migration (no user-data restructuring)

## Open Questions

None — the backlog entry, principle №2, and the two in-repo `--json` precedents resolve all design decisions; remaining choices are recorded as graded assumptions below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Cover nine subcommands (backlog's six + `current-stage`/`display-stage`/`all-stages`); exclude `progress-line` and `validate-status-file` | Backlog says "across the status query surface" and its line refs include `display-stage`; principle №2 scopes the MUST to programmatic output; later additions are additive | S:70 R:85 A:80 D:65 |
| 2 | Confident | JSON shapes: snake_case keys matching `.status.yaml` fields; `progress-map` as ordered array of `{stage,state}`; bare arrays for `get-issues`/`get-prs`/`all-stages`; `get-summary` wrapped in an object; empty lists `[]` never `null` | Field names copied from the status file schema; Go map marshaling would alphabetize stage order, so array is the only order-preserving option; object wrapper keeps summary additively extensible | S:60 R:70 A:85 D:60 |
| 3 | Certain | Emit mechanics: per-subcommand `--json` bool flag, named `xxxJSON` structs, `json.NewEncoder` + two-space `SetIndent` | Determined by the shipped `fab dispatch status --json` precedent (`dispatchStatusJSON`) — follow existing project patterns | S:65 R:90 A:95 D:90 |
| 4 | Confident | No `schema_version` field; stability = additive-only evolution (new fields optional) | Matches both existing fab `--json` surfaces; principle №2's versioning obligation is satisfied by the additive rule; a version field can be added later without breaking | S:55 R:60 A:80 D:70 |
| 5 | Certain | Default text output stays byte-identical; `--json` is purely additive | Back-compat with `git-pr.md`'s `get-issues` consumer and any hand parsers; the backlog asks to *add* `--json`, not to change existing output | S:80 R:70 A:95 D:95 |
| 6 | Confident | Consumer migration (skills switching to `--json`) is out of scope | Backlog scopes to offering `--json`; line output keeps working; migration can land any time later as an independent change | S:65 R:90 A:75 D:70 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).
