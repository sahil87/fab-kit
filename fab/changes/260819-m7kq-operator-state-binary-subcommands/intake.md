# Intake: Operator State Mutations Behind Binary Subcommands

**Change**: 260819-m7kq-operator-state-binary-subcommands
**Created**: 2026-08-19

## Origin

Backlog item `[m7kq]` (2026-08-13), invoked via `/fab-new` with the row's full text:

> Operator state mutations move behind binary subcommands (fab operator enroll/remove, watch add/rm/toggle, autopilot ops) so the server-keyed state-file schema cannot drift — motivating incident 2026-08-13: gpt-5.6-luna running /fab-operator hand-edited the YAML with an invented free-form 'note: trigger armed' field instead of a §7 watch entry. Today the model reads/writes the whole file per tick (fab-operator.md §4); prose-enforced schema fails on weaker instruction-followers. Follow-up to the operator provider-agnostic change (binary-rendered status frame + per-provider loop dictionary); same doctrine as fab score / tick-start: agents never compute (or hand-write) what the binary can own. Surfaces: fab operator Go subcommands + tests, _cli-fab.md § fab operator, fab-operator.md state-touching sections, runtime/operator memory at hydrate.

One-shot invocation; no prior conversation on this topic in the session. Note verified at intake: the referenced "operator provider-agnostic change (binary-rendered status frame + per-provider loop dictionary)" exists **neither** as an active change, an archived change, nor a backlog row — `fab-operator.md` §4 still has the model emitting the status frame itself. "Follow-up" is lineage (same motivating incident, same doctrine), not a dependency: this change proceeds standalone and touches only state-file mutation, not frame rendering (Assumptions row 11).

## Why

1. **The pain point**: the operator's server-keyed state file (`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`) holds the monitored set, autopilot queue, `branch_map`, and watches. Today the *model* reads and re-writes the whole YAML file on every tick and on every enrollment/removal/watch/autopilot action (`fab-operator.md` §4 Tick Behavior step 6 "Persist — write updated state"). The schema is enforced only by prose — the §4/§7 YAML examples in the skill file. On strong instruction-followers this mostly holds; on weaker ones it demonstrably does not: on 2026-08-13 a gpt-5.6-luna session running `/fab-operator` hand-edited the YAML with an invented free-form `note: trigger armed` field instead of writing a §7 watch entry. Prose-enforced schema fails exactly where the operator is supposed to become provider-agnostic.

2. **The consequence of not fixing**: every non-Claude (and every weaker-Claude) operator session is one hand-edit away from silently corrupting coordination state that must survive `/clear` and session restarts. Corrupted state poisons every later tick (the file is re-read each tick), and the failure is silent — nothing validates the file until behavior goes wrong. As the provider-agnostic direction lands (backlog `[tdqd]` et al.), the population of weaker instruction-followers driving `/fab-operator` grows, so drift probability rises over time.

3. **Why this approach**: it is the repo's established doctrine — *agents never compute (or hand-write) what the binary can own* — already applied to confidence scoring (`fab score`: agents never compute the score) and to this very file's tick bookkeeping (`fab operator tick-start` owns `tick_count`/`last_tick_at`, with atomic temp+rename writes). This change completes that migration for the remaining hand-written sections. A typed Go surface makes an invented field *impossible* rather than *prohibited*: the agent states intent through flags; the binary owns the schema, the timestamps, the list-cap pruning, and the atomic write. Rejected alternative: a schema validator (`fab operator lint`) that checks after hand-writes — it detects drift instead of preventing it, and still leaves timestamp/pruning computation on the agent.

## What Changes

### 1. New `fab operator` mutation subcommands (Go, `src/go/fab/cmd/fab/`)

All subcommands operate on the server-keyed state file via the existing `StatePath("")` derivation, honor the existing `operatorStatePathOverride` test seam, and write atomically via `atomicfile.WriteFile` — exactly the `tick-start` pattern. All timestamps (`enrolled_at`, `last_transition`, `last_checked`) are computed by the binary; no subcommand accepts a timestamp flag.

**Monitored set:**

```
fab operator enroll <change-id> --pane <pane-id> --repo <abs-path> --session <name> --branch <branch> \
    [--stage <stage>] [--agent <state>] [--stop-stage <stage>] [--spawned-by <watch>] [--depends-on <id,id,...>]
fab operator update <change-id> [--stage <stage>] [--agent <state>] [--stop-stage <stage>]
fab operator remove <change-id>
```

- `enroll` creates (or replaces) the monitored entry with the §4 schema fields, sets `enrolled_at` + `last_transition` to now, and **also records the `branch_map` entry** `{ branch, repo }` — enrollment is the documented moment `branch_map` gains its entry, so one command owns both writes.
- `update` mutates per-tick observed fields; when `--stage` changes the stored value, the binary touches `last_transition`.
- `remove` deletes the monitored entry and **retains** the `branch_map` entry (documented persistence policy — downstream dependency resolution needs it).
- The `»`/`›` window-name renames stay separate `fab pane window-name ensure-prefix|replace-prefix` calls — tmux side-effects remain composable primitives; these verbs touch only the state file.

**Watches:**

```
fab operator watch add <name> --source <linear|slack> --target-repo <abs-path> \
    [--query <json>] [--stop-stage <stage>] [--instructions <text>]
fab operator watch rm <name>
fab operator watch toggle <name> [--on|--off]
fab operator watch update <name> [--target-repo <path>] [--stop-stage <stage>] [--instructions <text>] [--query <json>]
fab operator watch checked <name> [--error <msg>]
fab operator watch seen <name> <item-id>
fab operator watch complete <name> <item-id>
```

- `add` initializes `enabled: true`, empty `known`/`completed`, `last_checked: null`, `last_error: null`. `--query` takes a JSON object string (nested lists/maps like `{ "status": ["Backlog", "Todo"] }` need it; the binary stores it as the YAML `query` map).
- `toggle` flips `enabled` (or forces with `--on`/`--off`) — the "pause/resume the Linear watch" path.
- `update` backs the conversational edits ("spawn into ~/code/bar instead", "also limit to 2 concurrent agents" — the skill appends to instructions and passes the merged text).
- `checked` sets `last_checked` to now and sets/clears `last_error` (present flag = set, absent = clear) — the per-tick query bookkeeping.
- `seen` appends to `known` and enforces the 200-entry cap (oldest pruned first) **in the binary** — the cap stops being an agent-counted rule.
- `complete` moves an item from `known` to `completed` (the stop-stage transition; keeps the union-dedupe invariant intact).

**Autopilot:**

```
fab operator autopilot start --queue <id,id,...>
fab operator autopilot pause
fab operator autopilot resume
fab operator autopilot advance [--skip]
fab operator autopilot stop
```

- `start` sets `{ queue, current: <first>, completed: [], state: running }`.
- `advance` moves `current` → `completed` (unless `--skip`) and promotes the next queue entry; on queue exhaustion sets `state: null` semantics per the §4 schema.
- `stop` clears the autopilot block (state `null`), `pause`/`resume` flip `state`.

**Branch map (explicit clear only):**

```
fab operator branch-map rm <change-id> | --all
```

Entries are *written* by `enroll`; the documented "persist until the user explicitly clears them" path gets this explicit verb.

**Read:**

```
fab operator state [--json]
```

Prints the state file (YAML verbatim by default, JSON with `--json`); creates the empty skeleton (`monitored: {}`, `autopilot: null`, `branch_map: {}`, `watches: {}`) when the file is missing — this replaces §2 Init step 1's "read the file; if missing it is created with empty …" hand-step, and removes the last reason for the agent to know the state-file path at all.

**Schema/IO posture**: mutation commands unmarshal the full file, apply typed edits to the section being mutated, and re-marshal. Unknown *top-level* keys are preserved (the `tick-start` read-modify-write posture — a legacy hand-drifted file never blocks the operator), but the four owned sections (`monitored`, `autopilot`, `branch_map`, `watches`) are rewritten from typed structs, so invented per-entry fields cannot survive a mutation of their section and can never be *introduced* (Assumptions row 5). Schema stays byte-compatible with the documented §4 shape; **no migration file is needed** (no field is renamed, moved, or retyped; the file is machine-local XDG state, not project data).

### 2. `fab-operator.md` state-touching sections (source: `src/kit/skills/fab-operator.md`)

The skill stops instructing the model to write YAML anywhere, replacing each write site with the matching verb:

- **§2 Init step 1**: read state via `fab operator state`; drop the "if missing, it is created with…" prose (binary-owned).
- **§4 Operator State File**: the schema block stays as *reference* documentation of what the binary maintains, with an explicit rule added: the file is mutated **only** through `fab operator` subcommands — the operator never hand-writes it (doctrine line, mirroring `fab score`'s "agents never compute it").
- **§4 Monitored Set / Branch Map**: enrollment → `fab operator enroll` (which also records `branch_map`); removal → `fab operator remove`; the adjacent `fab pane window-name` calls are unchanged.
- **§4 Tick Behavior**: step 1 reads via `fab operator state`; step 6 "Persist — write updated state" is **deleted/reworded** — persistence now happens at each action through its verb (per-tick observed-field changes ride `fab operator update`).
- **§6 Spawning step 7 + Autopilot**: enrollment and queue progression route through `enroll` / `autopilot start|advance|…`; autopilot state persistence line points at the verbs.
- **§7 Watches**: tick bookkeeping (steps 2/4) → `watch checked` / `watch seen` / `watch complete`; the Conversational Management table maps each utterance to its verb (`watch add` / `rm` / `toggle` / `update`).
- **§9 Key Properties**: "Uses the operator state file?" row notes mutations go through the binary.

Owner-or-pointer discipline: the *command contract* (signatures, flags, exit behavior) is owned by `_cli-fab.md` § fab operator; `fab-operator.md` names the verb at each policy site and does not restate flag tables.

### 3. `_cli-fab.md` § fab operator (source: `src/kit/skills/_cli-fab.md`)

Document every new subcommand alongside the existing `tick-start`/`time` entries: signature, one-paragraph behavior, the binary-owned invariants (timestamps, 200-cap pruning, branch_map persistence on remove, skeleton-on-missing for `state`), and the shared state-path/atomic-write mechanics (already documented under tick-start — point, don't restate).

### 4. Tests (`src/go/fab/cmd/fab/operator_test.go` or sibling files)

Per the constitution's CLI constraint (Go CLI change ⇒ tests + `_cli-fab.md`). Using `operatorStatePathOverride`, cover at minimum: enroll creates entry + branch_map + timestamps; re-enroll replaces; update touches `last_transition` only on stage change; remove keeps branch_map; watch add/rm/toggle/update round-trip; `seen` 200-cap oldest-first pruning; `complete` moves known→completed; `checked` set/clear of `last_error`; autopilot start/advance/skip/exhaustion/pause/resume/stop; `branch-map rm`/`--all`; `state` skeleton-on-missing and `--json`; unknown top-level key preserved across a mutation; invented per-entry field dropped when its section is rewritten.

### Non-goals

- **No binary-rendered status frame, no per-provider loop dictionary** — that is the (unshipped) sibling change; this change is state mutation only.
- **No change to `tick-start`/`time`** beyond sharing extracted load/save helpers.
- **No provider-neutrality sweep of fab-operator.md prose** — that is backlog `[tdqd]`.
- **No schema changes** — the file shape is byte-compatible; no migration.

## Affected Memory

- `runtime/operator`: (modify) state-file section gains the mutation-verb surface and the binary-owned-writes doctrine; enrollment/removal, watch-tick bookkeeping, and autopilot persistence descriptions updated to name the verbs
- `distribution/kit-architecture`: (modify) the "Operator State File" Go-mechanism section extends from path-derivation + tick-start to the full mutation-verb family

## Impact

- **Go**: `src/go/fab/cmd/fab/operator.go` (subcommand registration), new `operator_state_*.go` (or similar) command files, likely a small shared state load/save helper extracted from `operator_tick_start.go`; `operator_test.go` + new test files. No changes outside `cmd/fab` expected (state I/O is self-contained; `atomicfile` reused).
- **Kit skills**: `src/kit/skills/fab-operator.md` (§2, §4, §6, §7, §9), `src/kit/skills/_cli-fab.md` (§ fab operator).
- **Docs at hydrate**: `docs/memory/runtime/operator.md`, `docs/memory/distribution/kit-architecture.md`, regenerated memory indexes.
- **Scale**: ~15 small cobra subcommands sharing one read-modify-write helper; large but mechanical test surface; two skill files.
- **Compatibility**: state file shape unchanged; existing files (including tick-start's fields) keep working; no migration.

## Open Questions

- None blocking. (Verb naming details — e.g. `watch rm` vs `watch remove`, `seen`/`complete` — are apply-time decisions within the documented surface; the skill and `_cli-fab.md` are updated to whatever ships.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | **Full mediation**: every state-file mutation — including per-tick monitored-field updates and watch bookkeeping (`update`, `checked`, `seen`, `complete`) — gets a verb; the enumerated backlog list (enroll/remove, watch add/rm/toggle, autopilot ops) is read as examples, not the exhaustive surface | The stated goal ("schema cannot drift") is only true under full mediation — the motivating incident *was* a tick-time watch write; doctrine ("agents never hand-write what the binary can own") points the same way. Scope change mid-flight costs moderate rework across Go + two skill files | S:75 R:65 A:80 D:70 |
| 2 | Confident | Autopilot verb set = `start --queue` / `pause` / `resume` / `advance [--skip]` / `stop` | Backlog says only "autopilot ops", but the §4 schema fields (queue/current/completed/state) dictate exactly this lifecycle, incl. the documented skip/pause/resume interrupts; additive verbs are cheap to extend later | S:60 R:75 A:80 D:70 |
| 3 | Confident | Add read verb `fab operator state [--json]` with skeleton-on-missing | Not in the backlog list, but completes mediation: the agent never computes the state path (currently ambiguous in §2 Init) and init's "create if missing" hand-step becomes binary-owned; purely additive read command | S:55 R:85 A:75 D:75 |
| 4 | Confident | `»`/`›` window-name renames stay separate `fab pane window-name` primitives — not folded into `enroll`/`remove` | Existing composable-primitive pattern; rename-failure handling (log-and-continue) is operator policy layered on top; these verbs stay pure state-file operations. Trivially foldable later if wanted | S:60 R:80 A:85 D:80 |
| 5 | Confident | Tolerant-read/typed-write IO posture: unknown *top-level* keys preserved (tick-start precedent); the four owned sections rewritten from typed structs on mutation (drift inside them cannot survive); no strict-decode hard error | Strict decode would wedge the operator on any pre-existing hand-drifted file; tolerant top-level + typed sections prevents *new* drift while degrading gracefully on old. tick-start's in-file precedent makes tolerant the front-runner; posture is internal and switchable later | S:50 R:75 A:70 D:55 |
| 6 | Confident | `watch add/update --query` takes a JSON object string | Watch queries are nested (lists of statuses, maps); repeated `k=v` flags can't express them; binary stores as the YAML `query` map. A convenience alias could be added later without breaking anything | S:55 R:85 A:70 D:65 |
| 7 | Certain | All timestamps (`enrolled_at`, `last_transition`, `last_checked`) computed by the binary; no timestamp flags | Direct doctrine application; exact tick-start precedent (`last_tick_at`) | S:80 R:85 A:90 D:90 |
| 8 | Confident | `branch_map` written by `enroll`, retained by `remove`; explicit `branch-map rm <id>` (or `--all`) for the documented user-initiated clear | §4 documents enrollment as the write moment and persistence-after-removal as policy; "until the user explicitly clears" needs a verb under full mediation | S:70 R:80 A:85 D:75 |
| 9 | Certain | Schema stays byte-compatible; **no migration file** | No field renamed/moved/retyped; the migration anti-pattern covers restructuring, and the file is machine-local XDG state. tick-start's fields already interoperate | S:70 R:80 A:85 D:85 |
| 10 | Confident | `known` 200-cap (oldest-first) enforcement moves into `watch seen` | Cap is a mechanical counting rule — precisely what the doctrine says the binary owns; prose keeps only the policy statement | S:70 R:80 A:85 D:80 |
| 11 | Confident | Predecessor "provider-agnostic operator change" is unshipped and non-blocking; proceed standalone | Verified absent from active changes, archive, and backlog at intake; scopes are disjoint (state writes vs frame rendering); doctrine lineage only. If wrong, only sequencing adjusts | S:65 R:80 A:85 D:75 |

11 assumptions (2 certain, 9 confident, 0 tentative, 0 unresolved).
