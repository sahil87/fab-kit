# Plan: Operator Note Narrative-State Verb Family

**Change**: 260823-s2gw-operator-note-narrative-state
**Intake**: `intake.md`

## Requirements

> Source of truth: `fab/plans/sahil/26-08-23-operator-offload-plan.md` § Phase A (post-four-agent-review decided design). Phases B1/B2/C2/C are OUT OF SCOPE (Changes 2–4 of the queue).

### Go binary: note schema & storage

#### R1: Note schema and persisted id counter
`src/go/fab/cmd/fab/operator_note.go` (NEW) SHALL define the note schema as a typed struct serialized into a new top-level `notes:` **list** in the server-keyed operator state file: `id` (string, `n<N>`), `kind` (enum `dependency_wait | phase_plan | coordination | correction`), `text` (free prose), `refs` (optional string list, omitted when empty), `created_at` / `updated_at` (RFC3339 UTC via `nowRFC3339()` — no verb accepts a timestamp), `resolved` (bool), `resolved_at` (nullable). Ids MUST be assigned by the binary from a persisted top-level `notes_seq` integer counter that only increments — **ids are never reused after prune**. `notes_seq` is a new top-level key that an old binary's tolerant read carries through unchanged.

- **GIVEN** an empty state file
- **WHEN** `fab operator note add --kind phase_plan "first"` then a second `add` run
- **THEN** the notes get ids `n1` and `n2`, `notes_seq` persists as `2`, and each note carries binary-set `created_at`/`updated_at` and `resolved: false`

- **GIVEN** resolved notes were pruned after the counter reached N
- **WHEN** a new note is added
- **THEN** its id is `n<N+1>` — never a pruned note's id

#### R2: Notes become the fifth owned section
`operator_state.go` SHALL gain `notes: []` in `emptyOperatorState()` (`operator_state.go:170`), and the owned-sections doc comment (`operator_state.go:15–23`, "the four OWNED sections (monitored, autopilot, branch_map, watches)") SHALL name **notes** as the fifth owned section. Note verbs re-marshal `notes` from the typed struct on mutation (invented in-section fields cannot survive), while unknown **top-level** keys (including a legacy hand-written `plan_queue:`) survive every verb's read-modify-write. All writes ride `mutateOperatorState` (`operator_state.go:127`) → `saveOperatorState` atomic temp+rename; concurrency stays last-writer-wins under one operator per server. No migration file — `notes`/`notes_seq` are additive keys, no existing data is restructured.

- **GIVEN** a state file containing an unknown top-level `plan_queue:` key
- **WHEN** any `note` verb mutates the file
- **THEN** `plan_queue:` round-trips unchanged

- **GIVEN** a missing state file
- **WHEN** `fab operator state` persists the skeleton
- **THEN** the skeleton includes `notes: []` alongside the existing four sections

### Go binary: note verbs

#### R3: `fab operator note add`
`fab operator note add --kind <k> [--ref <r>]... <text>` SHALL create a note and print its id to stdout. Unknown `--kind` or text over the cap MUST exit 1 with a one-line error (the operator-verb enum convention — exit 2 stays the pane-family pane-missing code). The text cap is a named constant of **500** characters (the plan's "~500", read as 500). Repeated `--ref` flags accumulate into `refs`. Duplicate `add` of similar text is allowed — dedupe is operator judgment.

- **GIVEN** an empty state file
- **WHEN** `fab operator note add --kind coordination --ref s2gw --ref fab-kit "merge sequence pos 1/4"`
- **THEN** stdout prints the id (e.g. `n1`) and the stored note carries kind, refs, and text verbatim

- **GIVEN** `--kind lesson` (not in the enum — deliberately absent) or a 501-char text
- **WHEN** `add` runs
- **THEN** exit code is 1 and no state is written

#### R4: `fab operator note resolve` — idempotent + bounded resolved history
`fab operator note resolve <id>` SHALL set `resolved: true` + `resolved_at` (now). Resolving an already-resolved note is **idempotent**: exit 0, no field changes. Unknown id exits 1. After marking, the verb SHALL prune **resolved** notes past a cap of **50**, oldest-first (the `knownCap = 200` pattern, `operator_state.go:72`). **Open notes are never pruned or auto-expired** — notes are decisions, not a dedupe cache.

- **GIVEN** 51 resolved notes and 30 open notes
- **WHEN** the 51st resolve lands
- **THEN** the oldest resolved note is pruned, all 30 open notes survive, and `notes_seq` is untouched

- **GIVEN** a resolved note `n3`
- **WHEN** `note resolve n3` runs again
- **THEN** exit 0 and `resolved_at` is unchanged

#### R5: `fab operator note update`
`fab operator note update <id> <text>` SHALL replace the note's text in place (evolving phase-plan notes update rather than accreting near-duplicates) and refresh `updated_at` (age and staleness render from `updated_at`). The 500-char cap applies (exit 1 over-cap); unknown id exits 1. `created_at`, `id`, `kind`, `refs`, and resolution fields are untouched.

- **GIVEN** note `n2` with text "phase 1 of 3"
- **WHEN** `note update n2 "phase 2 of 3"`
- **THEN** the text is replaced, `updated_at` moves forward, `created_at` is unchanged

#### R6: `fab operator note list`
`fab operator note list [--open|--all] [--json]` SHALL list notes — `--open` is the default (open notes only); `--all` includes resolved. Human output renders per note: id, kind, age from `updated_at`, and the text's first line; a note whose `updated_at` age exceeds **14 days** carries a display-only staleness flag (e.g. `⚠ 21d`). When more than **25** notes are open, the verb warns (the "do I even need a note?" nudge made mechanical) on **stderr** so stdout stays clean. `--json` emits the (filtered) notes as JSON with no warning/decoration on stdout. Age rendering uses a day-capable formatter (`FormatIdleDuration` stops at hours, so notes need `d`).

- **GIVEN** one open note updated 21 days ago and one resolved note
- **WHEN** `note list` runs
- **THEN** only the open note prints, flagged `⚠ 21d`; `note list --all` includes the resolved one

- **GIVEN** 26 open notes
- **WHEN** `note list` runs
- **THEN** a warning naming the count appears on stderr and stdout stays the plain list

### Go binary: `fab operator state` header

#### R7: OPEN NOTES header + `--all` + resolved filtering
`fab operator state` SHALL print an **OPEN NOTES header block** (one line per open note: id · kind · age · first line of text) before the state dump — **human output only**: absent from `--json`, and comment-prefixed (`# `) in default mode so stdout stays parseable YAML for yq consumers. The header is omitted when there are no open notes. Default output (both YAML and `--json`) SHALL **exclude resolved notes** from the `notes:` list; a new `--all` flag includes them. When no filtering applies (no resolved notes) the existing raw-verbatim print is preserved; when filtering applies, the state is re-marshaled with the filtered list (the file on disk is never rewritten by a read). Skeleton-on-missing behavior is unchanged apart from R2's `notes: []`.

- **GIVEN** a state file with one open and one resolved note
- **WHEN** `fab operator state` runs
- **THEN** stdout begins with `# OPEN NOTES` comment lines, the YAML body parses cleanly under yq, and the resolved note is absent; `--all` includes it

- **GIVEN** the same file
- **WHEN** `fab operator state --json` runs
- **THEN** the output is pure JSON — no header lines — and excludes the resolved note (unless `--all`)

### Kit skills

#### R8: `fab-operator.md` § Notes + doctrine table + restore list
`src/kit/skills/fab-operator.md` SHALL gain a `### Notes` subsection under §4 (after `### Branch Map`) documenting the four verbs, the note lifecycle (add → update → resolve; bounded resolved history; open notes never expire), and the **doctrine table verbatim in content** from the intake (note vs watch vs not-operator-state; the deliberate absence of a `lesson` kind as the guard against a reflexive scratchpad). §2 Init step 2's restore list ("monitored set, autopilot queue, and branch_map") SHALL include **notes**. The §4 schema reference block SHALL gain `notes_seq` + a `notes:` example entry; the §4 never-hand-write verb enumeration and the §9 Key Properties state-file row SHALL name the `note` verbs/notes.

- **GIVEN** the updated skill
- **WHEN** grepping for the restore-list phrase and the verb enumeration
- **THEN** every occurrence names notes; the doctrine table appears once, under § Notes

#### R9: `_cli-fab.md` signatures (constitution-mandated)
`src/kit/skills/_cli-fab.md` SHALL gain a `### fab operator note` block (between `### fab operator enroll / update / remove` and `### fab operator watch`) with the four verb signatures and their semantics (exit 1 validation, cap 500, resolved-prune 50, `--open` default, stderr warn >25, staleness flag), SHALL update `### fab operator state` to `[--all] [--json]` with the header/filtering contract, and SHALL update the § Shared state-verb mechanics line (`_cli-fab.md:1233`): five owned sections, and the timestamp enumeration gains `created_at`/`updated_at`/`resolved_at`.

- **GIVEN** the updated `_cli-fab.md`
- **WHEN** an agent reads § fab operator
- **THEN** every new/changed command signature is present and consistent with the Go implementation

### Non-Goals

- No `lesson` note kind — durable process lessons route via `idea` → a fab change into docs/memory (deliberate guard).
- No auto-expiry for open notes; no `depends_on` survival teaching; no migration code for `plan_queue:` (one-time manual conversion).
- No changes to `operator_tick_start.go`, `panemap.go`, `pane_questions.go`, or `_cli-agents.md` — Phases B1/B2/C2/C are Changes 2–4 of the queue.
- Hydrate for this change touches only the Phase-A part of `docs/memory/runtime/operator.md` (verb enumeration + notes-vs-watches decision); the sweep of memory restatements (`operator.md:79/:495`, `kit-architecture.md:98` four-owned-sections/verb-family lines) is hydrate's, not apply's.

### Design Decisions

#### Notes stored as a list, pruned by list order
**Decision**: `notes:` is a YAML list in creation order (append at add); resolve-time pruning drops the earliest resolved entries in list order.
**Why**: matches the skeleton `notes: []`, keeps "oldest-first" trivially well-defined without sorting timestamps, and renders naturally in `state`/`list`.
**Rejected**: a map keyed by id (like `watches`) — loses creation order, making oldest-first pruning depend on timestamp parsing.
*Introduced by*: 260823-s2gw-operator-note-narrative-state

#### `state` default mode re-marshals only when filtering applies
**Decision**: `fab operator state` prints the raw file verbatim when no resolved notes exist (today's behavior byte-preserved); it re-marshals the parsed state with a filtered `notes:` list only when resolved notes must be excluded (or `--all` is absent with resolved notes present).
**Why**: preserves the "pure read never rewrites" property and byte-stability for existing files; yaml.v3 re-marshal is already the mutation-path format so filtered output stays consistent.
**Rejected**: always re-marshaling — needlessly reformats untouched files; filtering the raw text line-wise — fragile YAML surgery.
*Introduced by*: 260823-s2gw-operator-note-narrative-state

#### Day-capable note age formatter
**Decision**: a small `formatNoteAge` helper (s/m/h/d, floor division) local to `operator_note.go`, used by `list` and the `state` header; staleness threshold a named constant (14 days), display-only.
**Why**: `pane.FormatIdleDuration` caps at hours; note ages span weeks (`⚠ 21d` is the plan's example rendering).
**Rejected**: extending `FormatIdleDuration` — it formats pane idle-durations where a `d` unit is out of contract for its existing callers.
*Introduced by*: 260823-s2gw-operator-note-narrative-state

## Tasks

### Phase 1: Setup

- [x] T001 Update `src/go/fab/cmd/fab/operator_state.go`: add `"notes": []interface{}{}` to `emptyOperatorState()`; extend the owned-sections doc comment (lines 15–23) to name notes as the fifth owned section <!-- R2 -->

### Phase 2: Core Implementation

- [x] T002 Create `src/go/fab/cmd/fab/operator_note.go` (follow `operator_watch.go` structure): `noteEntry` struct per R1, kind enum, named constants (`noteTextCap = 500`, `resolvedNotesCap = 50`, staleness threshold 14d), `notes_seq` read/increment helper tolerant of YAML int decoding, notes-section read/write via `operatorSection`, `formatNoteAge` helper (s/m/h/d) <!-- R1 -->
- [x] T003 Implement `fab operator note add --kind <k> [--ref <r>]... <text>` in `operator_note.go`: validate kind + cap (exit 1), assign `n<notes_seq+1>`, persist counter, set timestamps, print id <!-- R3 -->
- [x] T004 Implement `fab operator note resolve <id>`: idempotent, unknown id exit 1, set `resolved`/`resolved_at`, prune resolved past 50 oldest-first (open notes never pruned) <!-- R4 -->
- [x] T005 Implement `fab operator note update <id> <text>`: in-place text replace, cap validation, refresh `updated_at`, unknown id exit 1 <!-- R5 -->
- [x] T006 Implement `fab operator note list [--open|--all] [--json]`: `--open` default, id · kind · age · first line, `⚠ <age>` past 14d, stderr warning above 25 open, `--json` clean output <!-- R6 -->
- [x] T007 Register `operatorNoteCmd()` in `src/go/fab/cmd/fab/operator.go`'s `AddCommand` list <!-- R1 -->
- [x] T008 Modify `runOperatorState`/`operatorStateCmd` in `src/go/fab/cmd/fab/operator_state.go`: add `--all` flag; default excludes resolved notes (re-marshal only when filtering applies, raw-verbatim otherwise); `# `-prefixed OPEN NOTES header (id · kind · age · first line) in human mode only, omitted when no open notes; `--json` header-free with the same filtering <!-- R7 -->

### Phase 3: Integration & Edge Cases

- [x] T009 Create `src/go/fab/cmd/fab/operator_note_test.go` using the `withOperatorState`/`runOperatorCmd`/`readStateFile` helpers: ids from persisted `notes_seq` never reused after prune; binary-set timestamps; kind + text-cap validation exit 1; resolve idempotency + unknown-id exit 1; resolved-cap prune at 50; open notes never pruned; list shapes (`--open` default, `--all`, `--json`) + staleness flag + >25 stderr warning; unknown top-level keys (`plan_queue:`) survive every verb <!-- R1 -->
- [x] T010 Extend state tests (in `operator_note_test.go`): `state` header human-mode-only (absent from `--json`), stdout parses as YAML under the header comments, default excludes resolved / `--all` includes, skeleton gains `notes: []` <!-- R7 -->
- [x] T011 Run `gofmt -l` on touched Go files and `go test ./src/go/fab/cmd/fab/` (widen only if cross-cutting) <!-- R1 -->

### Phase 4: Polish

- [x] T012 Update `src/kit/skills/fab-operator.md`: new `### Notes` under §4 after `### Branch Map` (verbs, lifecycle, doctrine table verbatim); §2 Init step 2 restore list gains notes; §4 schema block gains `notes_seq` + `notes:` example; §4 never-hand-write verb enumeration gains the `note` verbs; §9 state-file row gains notes <!-- R8 -->
- [x] T013 Update `src/kit/skills/_cli-fab.md`: new `### fab operator note` block (after enroll/update/remove); `### fab operator state` signature `[--all] [--json]` + header/filtering contract; § Shared state-verb mechanics — five owned sections + timestamps list gains the note fields <!-- R9 -->
- [x] T014 Sibling sweep: grep repo-wide (excluding `.claude/` and `docs/memory/` — memory is hydrate's) for `monitored set, autopilot queue`, `four owned sections`/`four OWNED sections`, and the state-verb enumeration phrase; update every `src/` occurrence missed by T001/T012/T013 <!-- R8 -->

## Execution Order

- T002 blocks T003–T006 (shared schema/helpers); T007 needs T002; T008 needs T002 (header uses note helpers)
- T009/T010 need T003–T008; T011 last in Phase 3
- T012–T014 are independent of each other, after code settles

## Acceptance

### Functional Completeness

- [x] A-001 R1: `operator_note.go` exists with the full schema; ids assigned `n<N>` from a persisted `notes_seq` that never reuses ids after prune
- [x] A-002 R2: skeleton includes `notes: []`; owned-sections comment names five sections; unknown top-level keys survive every note verb
- [x] A-003 R3: `note add` prints the id; unknown kind and >500-char text exit 1 writing nothing
- [x] A-004 R4: `note resolve` is idempotent, prunes resolved past 50 oldest-first, never touches open notes
- [x] A-005 R5: `note update` replaces text in place and refreshes `updated_at` only
- [x] A-006 R6: `note list` defaults to open notes with age + `⚠` staleness past 14d, warns >25 open on stderr, `--json` is clean
- [x] A-007 R7: `state` prints a `# `-prefixed OPEN NOTES header in human mode only; default output excludes resolved notes; `--all` includes; stdout stays yq-parseable
- [x] A-008 R8: `fab-operator.md` carries § Notes with the doctrine table (no `lesson` kind), the updated restore list, schema block, verb enumeration, and §9 row
- [x] A-009 R9: `_cli-fab.md` carries the four note-verb signatures, the updated `state` signature, and the five-owned-sections mechanics line

### Behavioral Correctness

- [x] A-010 R7: an existing state file with no notes prints byte-identically to today (raw verbatim, no header)
- [x] A-011 R4: resolving an already-resolved id exits 0 without changing `resolved_at`

### Scenario Coverage

- [x] A-012 R1: `operator_note_test.go` covers every § Tests item from the intake (ids-never-reused, timestamps, validation exits, idempotency, prune, list shapes, staleness, header modes, skeleton, unknown-key survival)
- [x] A-013 R1: `gofmt` clean and `go test ./src/go/fab/cmd/fab/` passes

### Edge Cases & Error Handling

- [x] A-014 R2: a legacy hand-written `plan_queue:` top-level key round-trips unchanged through every note verb (tolerant read; no migration code ships)
- [x] A-015 R3: duplicate `note add` with identical text succeeds (dedupe is judgment, aided by the list warning)

### Code Quality

- [x] A-016 Pattern consistency: `operator_note.go` mirrors `operator_watch.go`'s cobra/typed-struct/`mutateOperatorState` structure; tests reuse the existing helpers
- [x] A-017 No unnecessary duplication: reuses `operatorSection`, `nowRFC3339`, `mutateOperatorState`; no second state-IO path
- [x] A-018 Magic numbers named: 500/50/25/14d are named constants
- [x] A-019 Canonical source only: skill edits land in `src/kit/skills/`, never `.claude/skills/`
- [x] A-020 CLI ⇒ docs + tests: every new/changed command signature is reflected in `_cli-fab.md` and covered by tests (Constitution Additional Constraints)
- [x] A-021 Owner-or-pointer: the doctrine table lives once (skill § Notes); other sites point, never restate

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- **For hydrate**: beyond the intake's Affected Memory row (`runtime/operator`), the sweep class includes `docs/memory/runtime/operator.md:79/:495` (verb-family enumeration, four-owned-sections decision) and `docs/memory/distribution/kit-architecture.md:98` (mutation-verb family list) — update the note verbs/fifth section there during hydrate.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Text cap counts characters as Unicode runes (not bytes) | "chars" in the plan; runes are the natural Go reading; trivially adjustable constant | S:60 R:90 A:80 D:70 |
| 2 | Confident | `notes:` is a list in creation order; oldest-first prune = earliest resolved entries in list order | Skeleton `notes: []` implies a list; creation order makes oldest-first well-defined | S:70 R:80 A:80 D:75 |
| 3 | Confident | `state` prints raw bytes verbatim when nothing is filtered; re-marshals only when excluding resolved notes | Preserves today's byte-stable behavior for existing files; filtering requires a parse anyway | S:60 R:85 A:85 D:70 |
| 4 | Certain | New local `formatNoteAge` helper with a `d` unit; `FormatIdleDuration` untouched | Existing helper caps at hours by contract with its pane callers; plan renders `21d` | S:80 R:90 A:90 D:85 |
| 5 | Confident | The >25-open-notes warning goes to stderr | Keeps stdout machine-clean, consistent with the state header's yq-parseable rule | S:55 R:90 A:80 D:70 |
| 6 | Confident | Placement: `### Notes` after `### Branch Map` in `fab-operator.md` §4; `### fab operator note` between enroll/update/remove and watch in `_cli-fab.md` | Notes are a state-file section like monitored/branch_map; watches keep their own §7 | S:55 R:90 A:75 D:65 |
| 7 | Certain | Second `resolve` of the same id: exit 0, no field changes (including `resolved_at`) | "Idempotent" per plan/intake — same inputs, same result | S:85 R:90 A:90 D:85 |
| 8 | Confident | `note update` is allowed on resolved notes (refreshes `updated_at` as usual) | No stated restriction; forbidding it adds a rule the plan doesn't carry | S:50 R:85 A:75 D:65 |
| 9 | Confident | The OPEN NOTES header is omitted entirely when no open notes exist | An empty decorative header adds noise; matches "open notes" semantics | S:55 R:90 A:85 D:75 |
| 10 | Confident | Header first line is `# OPEN NOTES (N)` (open-note count) followed by one `# id kind age first-line` line per open note; a stale note's `⚠ <age>` flag replaces its plain age in that line | The plan fixes the line content but not the banner shape; the count aids the skim consumer; one rendering shared by `note list` and the header keeps them consistent | S:50 R:90 A:75 D:60 |
| 11 | Confident | `note list --open` is accepted as an explicit no-op spelling of the default (no `--open`/`--all` conflict error) | R6 names `--open` as the default; erroring on the explicit default would punish the documented spelling | S:50 R:90 A:80 D:65 |

11 assumptions (2 certain, 9 confident, 0 tentative).
