# Intake: Six-Verb `fab config` Surface

**Change**: 260808-ihiv-config-six-verb-surface
**Created**: 2026-08-08

## Origin

> Implement Change 2 (the six-verb fab config surface: show, explain, set, unset, init, upgrade) of `fab/plans/sahil/26-08-08-config-overhaul.md` — read that full section plus the Orchestration and Resolved Decisions sections for scope, decisions, obligations, and rejected alternatives, including the 'Lessons from the discarded branch' day-one regression test cases (block-form orphaning, duplicated renderer, key-axis typos, value-type holes, quote holes, comment preservation). Changes 1, 3, and 6 are running in parallel in sibling worktrees off the same base commit; this change has zero dependency on them per the plan's dependency map — but it soft-couples with Change 1 (shared YAML value-parsing helper) and is a hard prerequisite for the later Change 5 (out of scope now). Shared-surface files (`_cli-fab.md` and its SPEC mirror, `docs/specs/config.md`, `_shared/configuration.md`) will also be touched by the other changes in different sections — stay strictly within this change's declared scope in those files.

One-shot `/fab-new` dispatch against a fully pre-resolved plan: every open decision in `fab/plans/sahil/26-08-08-config-overhaul.md` is **user-confirmed 2026-08-08** (see its § Resolved decisions). This change is a re-derivation of the discarded `260807-k0v3-consolidate-config-surface` design with one rename (`docs` → `explain`); the k0v3 branch (worktree nimble-ravine) is discarded as stale — **fresh implementation, no salvage** — but its 5 review-rework cycles survive as day-one regression tests (§ What Changes → Day-one regression tests).

## Why

1. **The verb pain**: `fab config` today has four subcommands (`reference`, `show`, `init`, `upgrade`) that don't map onto the three user intents — *check state* (`show`), *learn the schema* (`reference`), *apply a change* (nothing). "Apply a change" is hand-editing YAML — the exact operation class that produced the comment-mashing bug (`setFabVersion`, patched by yogn #473, retired by the fence engine). `set`/`unset` close the last hand-editing driver.
2. **If not fixed**: every config mutation remains a hand edit against a fence-managed file, inviting the mangling class the fence engine was built to end; agents and users keep guessing key names with no typo linting (the reserved `validate` slot never shipped); and Change 5 of the plan (fence scope annotations rendering `fab config set --system` pointers) has no verb to point at — this change is its hard prerequisite.
3. **Why this approach**: surveyed against git 2.46+ (`get/set/unset/list/edit`), npm (`get/set/delete/list/edit/fix`), gh, gcloud, kubectl — the `get/set/unset/list` core is standard; `explain` follows the `kubectl explain` precedent for per-key schema docs. Rejected (user-confirmed): `edit` ($EDITOR over a fence-managed file invites mangling), `fix`/`doctor` (`upgrade` already hoists/parks/regenerates — extend it, don't duplicate; the `--check` drift mode is Change 5's), and the reserved `validate` slot is **dropped** (`set`'s unknown-key refusal + scope enforcement deliver the typo linting it was held for).

## What Changes

Target surface — six intent-grouped verbs (three cobra command groups):

```
# Inspect — "what IS"
fab config show [<key>] [--origin]        # effective post-cascade config; with <key>, just that value

# Inspect — "what CAN be"
fab config explain [<key>] [--json]       # registry documentation (rename of `reference`); with <key>, that field's block

# Modify — surgical, fence-aware
fab config set <key> <value> [--system]   # write one field through the configupgrade engine
fab config unset <key> [--system]         # remove an override, restoring inheritance/default

# Lifecycle — whole-file
fab config init [--system]                # bootstrap; bare = project mode (--project flag retained)
fab config upgrade                        # mechanical reconcile against the registry (unchanged)
```

### `show <dotted.key>` (new positional arg)

- Prints the **effective value with built-in defaults composed** — unlike bare `show`, which deliberately doesn't materialize defaults (the single-key form answers "what is in effect"). Bare `show` behavior is unchanged.
- Scalar/list leaves print **raw** (scriptable — no `key =` prefix); map-valued keys print the YAML subtree.
- `--origin` works at both depths (bare listing unchanged; with `<key>`, provenance for that key).
- Unknown key ⇒ **non-zero exit naming the key**.
- Implementation note: `cmd/fab/config.go` `configShowCmd()` is `cobra.NoArgs` today — becomes `cobra.MaximumNArgs(1)`. The default-composition machinery already exists in `renderShowOrigin`/`defaultSubtree`/`flattenOrigin` (cmd/fab/config.go) — reuse it, don't duplicate.

### `explain [<key>]` (rename of `reference`)

- Bare form prints **exactly what `fab config reference` prints today** (`configref.Render()`).
- With `<key>`: only that field's rendered `Segment`. Keys documented inside another field's segment (rows whose `Segment == ""`, e.g. `project.name` inside the `project` block; leaves like `providers.claude.session_command` inside the `providers` segment) **resolve to the owning segment**.
- `--json` bare = full field table (`RenderJSON()`, unchanged); with `<key>` = matching row(s).
- **`reference` is retained as an invisible cobra alias** (`Aliases: ["reference"]` — cobra aliases don't appear in help). This protects the `# Full reference of all available options: fab config reference` pointer comments seeded into existing user configs by migration 2.9.2-to-2.10.0, and keeps historical migration texts functional. All current kit/doc text migrates to `explain`; **historical migration files stay verbatim** (the alias keeps them working).

### `set <key> <value> [--system]` (new)

- Writes **through the `internal/configupgrade` fence engine** — never a YAML marshal round-trip (the yogn/#473 comment-mashing class is the whole motivation).
- Live field ⇒ splice the new value into the existing live block; fence-only field ⇒ **materialize live above the fence** (presence=intent: setting pins the override; the fence stops advertising it — `renderFence` already omits live keys).
- **Refuses unknown keys** (error suggests `fab config explain`); **scope-enforced** via `internal/configscope` (`set --system <project-scoped-key>` refuses; this + unknown-key refusal delivers the typo linting the dropped `validate` slot was reserved for).
- **Deep dotted keys into map-valued fields supported**: `set agent.profiles.review.model …`, `set providers.codex.profiles.default.model …`. Setting a leaf inside an existing live map block must splice only that leaf — never rewrite or orphan the block's sibling keys (day-one regression test).
- `<value>` parsed as a **YAML scalar; lists in flow syntax** (`[a, b]`) — the same value-parsing semantics Change 1's env layer specifies (see Assumptions #4 on the soft-coupling).
- `--system` targets `~/.fab-kit/config.yaml`, **creating it with the scaffold header when missing** (git-config precedent — `git config --global` creates `~/.gitconfig`).

### `unset <key> [--system]` (new)

- Removes the live override through the same engine; **the fence regains the commented reference line** on the next render (automatic: `renderFence` re-advertises non-live advertise:true fields).
- Restores inheritance (system value, else built-in) — the semantic meaning of removal under presence=intent.
- Unsetting an unset key = **exit-0 no-op with notice** (idempotent).
- **Deliberately ungated on value type** — unset is the repair path; it must work on any live key the registry knows, however malformed the value.

### `init` (bare becomes project mode)

- Bare `fab config init` — today a **usage error** — becomes `--project` mode (error→behavior is the safe breaking direction; nothing depended on the error).
- `--project` flag retained (fab-kit's `fab init` shells out to `fab config init --project`); `--system` unchanged; **both together still an error**; overwrite refusal unchanged (both modes).

### `upgrade` (unchanged behavior)

- No semantic change. In-scope only: help/description text updated where it names sibling verbs, and the group's "leaves room for a future `fab config validate`" comment/prose is **removed** (validate is dropped).

### Day-one regression tests (lessons from the discarded k0v3 branch)

Every k0v3 review cycle found a real "silent-wrong-at-exit-0" escape in the surgical writers. Write these as regression tests **before** the writer code:

1. **Block-form orphaning** — setting a leaf (`agent.profiles.review.model`) inside a live map block must not orphan/drop the block's sibling keys.
2. **Duplicated renderer** — the segment text `set` materializes into the live area and the fence renderer's segment MUST be single-sourced (one renderer, `configref` Segment / `configupgrade`); a second copy drifts.
3. **Key-axis typos** — a valid-*looking* dotted path that matches no registry row (`agent.profile.review.model`, `dispatch.colum_width`) must refuse, not write.
4. **Value-type holes** — a map value where a scalar belongs (`set agent.workers '{a: b}'`) must refuse via the registry's expected-kind signal, not write silently.
5. **Quote holes** — the comment scanner must not be fooled by leading-quote-only values; quoted keys are refused.
6. **Comment preservation end-to-end** — the yogn class: user comments (above, inline, interior) survive every `set`/`unset` round-trip. Include idempotence (set same value twice = byte-identical) per the engine's golden/freeze test discipline.

### Documentation & sweep obligations (this change's declared scope — do not exceed)

- `src/kit/skills/_cli-fab.md` § fab config **rewrite** (six verbs, new signatures, alias note) + `docs/specs/skills/SPEC-_cli-fab.md` mirror + Go tests (constitution: CLI change ⇒ `_cli-fab.md` + tests).
- `docs/specs/config.md`: surface section (four verbs → six), set/unset-as-surgical-upgrade framing, validate-slot removal note. **Stay out of** the cascade-layers section (C1's) and mode/rename sections (C3/C4's).
- `docs/specs/index.md`: update the `config` row's description.
- Kit-text sweep `reference` → `explain`: `src/kit/skills/fab-setup.md` (+ SPEC mirror) if it names the command, `docs/site/skill.md` (lines naming `fab config {reference,show,init,upgrade}` and `fab config reference`) — edit the canonical `docs/site/skill.md` and run `scripts/sync-skill.sh` to refresh the embedded copy `src/go/fab/cmd/fab/skill.md` (drift-guard test `TestSkillEmbedMatchesCanonical` fails otherwise; the file has a ≤150-line budget under the shll `skill` standard).
- Memory sweep: `_shared/configuration.md` (primary — § Schema Discovery, § visibility commands, § single-writer prose; also its now-stale `cobra.NoArgs` claims on `show`), `distribution/kit-architecture.md` (the stale `cobra.NoArgs` claim on the command inventory rows and the "leaves room for a future `fab config validate`" sentence), `runtime/providers-and-profiles.md` (any `fab config reference` pointers), `distribution/migrations.md` — **the quoted seeded pointer comment (`# Full reference of all available options: fab config reference`) stays VERBATIM** there; it documents what the migration seeded, which is exactly why the alias exists.
- Historical migration files (`src/kit/migrations/2.9.2-to-2.10.0.md`, `2.11.0-to-2.12.0.md`, `2.12.1-to-2.13.0.md`, `2.13.1-to-2.13.2.md`): **left verbatim**.
- **shll standards check** before finalizing naming/help text (constitution § Toolkit Standards): `shll standards principles` (CLI principles), `help-dump` (cobra tree shape changes with new subcommands — contract is generated, no action expected), `skill` (docs/site/skill.md budget/shape).
- The fence anchor text (`# >>> fab reference (kit …) >>>`) and the fence header comment are **NOT renamed** — they are byte-exact file-format anchors in every user's config.yaml, not command references (see Assumptions #6).

### Out of scope (other changes in the plan — do not pre-empt)

- C1: env override layer, `FAB_*` mapping, `show --origin` env origin, launch flags.
- C3: `dispatch.mode`, watchable dissolution, provider capability flags.
- C4: `session_command`/`dispatch_command` renames.
- C5: `upgrade --check`, fence scope annotations + `set --system` pointers, stub retirement, one-values-file, origin drill-down fix.
- C6: `FAB_KIT_PATH`.

## Affected Memory

- `_shared/configuration.md`: (modify) primary — schema-discovery surface renamed to `explain` (+ alias), `show <key>` form, new `set`/`unset` surgical-writer verbs under the single-writer story, stale `cobra.NoArgs` claims corrected
- `distribution/kit-architecture.md`: (modify) command-inventory rows — `explain` rename, new verbs, drop the stale "leaves room for a future `fab config validate`" sentence and stale `cobra.NoArgs` claims
- `runtime/providers-and-profiles.md`: (modify) `fab config reference` pointers → `fab config explain`
- `distribution/migrations.md`: (modify) note the alias protecting the seeded pointer comment; the quoted seeded comment itself stays verbatim

## Impact

- **Go (fab-go binary)**: `src/go/fab/cmd/fab/config.go` (group rework: `show` positional, `explain` + alias, new `set`/`unset`, `init` bare-default) + `config_test.go`/`config_show_init_test.go`/`config_upgrade_test.go`; `src/go/fab/internal/configupgrade/` (the set/unset surgical write path — new exported entry points beside `Upgrade`; regression tests **first**); `src/go/fab/internal/configref/` (segment-by-key lookup, expected-kind signal for value-type checking); `internal/configscope` consumed for scope enforcement (likely no change to the leaf package itself); a small YAML scalar/flow value-parsing helper placed where Change 1 can reuse it.
- **Kit text**: `src/kit/skills/_cli-fab.md`, `src/kit/skills/fab-setup.md` (if it names the command); canonical `docs/site/skill.md` + synced embed `src/go/fab/cmd/fab/skill.md`.
- **Specs**: `docs/specs/config.md`, `docs/specs/index.md`, `docs/specs/skills/SPEC-_cli-fab.md` (+ SPEC-fab-setup if fab-setup changes).
- **Memory**: the four files above.
- **No migration**: no user-data restructuring — existing config files are untouched; `reference` keeps working via the alias. New verbs are additive; `init` bare goes error→behavior.
- **Parallel-worktree discipline**: C1/C3/C6 run in siblings off the same base commit; on the shared files (`_cli-fab.md` + SPEC mirror, `docs/specs/config.md`, `_shared/configuration.md`) touch **only** this change's sections — merge conflicts should stay textual/mechanical per the plan's conflict-surface table.

## Open Questions

*(none — all design decisions were user-confirmed in the plan's § Resolved decisions; residual implementation choices are graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Six-verb surface exactly as specified (show `<key>`, explain+alias, set, unset, init bare-default, upgrade unchanged); no edit/fix/doctor/validate | Plan § Change 2 + § Resolved decisions 2–3, all user-confirmed 2026-08-08 | S:95 R:70 A:95 D:95 |
| 2 | Certain | Fresh implementation; k0v3 branch not salvaged; its review findings become day-one regression tests written before the writer code | Plan header + § Resolved decisions 1, user-confirmed | S:95 R:80 A:95 D:90 |
| 3 | Certain | `reference` kept as invisible cobra alias; historical migration files verbatim; seeded pointer comments keep working | Plan § Change 2 carried-forward detail, user-confirmed | S:90 R:75 A:90 D:90 |
| 4 | Confident | The YAML scalar/flow value-parsing helper is implemented **in this change** in a reusable location (e.g. a small internal package or exported func) with C1-compatible semantics (scalars + flow lists/maps); whichever of C1/C2 merges second rebases onto the other's helper | Plan § Orchestration names this the C1↔C2 soft-coupling and prescribes the mechanical rebase; C2 cannot depend on unmerged parallel work | S:75 R:70 A:70 D:65 |
| 5 | Confident | `set`/`unset --system` operate on `~/.fab-kit/config.yaml` with the same comment-preserving splice discipline but **no managed fence** (the system file has none today); when the file is missing, `set --system` creates it with the existing `systemScaffoldHeader`-style header plus the one live key | Plan states only "creates ~/.fab-kit/config.yaml (scaffold header) when missing — git-config precedent"; the system file format (all-commented scaffold, fence-less) is verified in `runConfigInitSystem`/`renderSystemScaffold`; adding a fence there is C5-adjacent scope creep | S:70 R:70 A:75 D:60 |
| 6 | Confident | The managed-fence anchor text (`# >>> fab reference (kit …) >>>`) and in-file fence prose are NOT part of the `reference`→`explain` sweep | Anchors are byte-exact splice markers present in every user's config.yaml (`beginLineRe`/`endLineRe`); renaming them is a file-format change needing migration + engine changes — clearly outside "kit/doc text migrates to explain"; "fab reference" there names the reference *fence*, not the command | S:70 R:65 A:85 D:75 |
| 7 | Confident | Value-type checking (`value-type holes` test) derives the expected kind from the registry: a non-nil `Field.Default`'s Go/JSON type, else scalar-vs-map plausibility from the key's position (a leaf inside a map-valued row vs a scalar row); refusal message names the expected kind | Plan names "the expected-kind registry signal" without specifying derivation; registry `Default any` is the only typed signal available (verified in configref.go); exact mechanics are apply-time detail, easily revised | S:60 R:75 A:70 D:60 |
| 8 | Confident | `explain --json <key>` emits the matching row(s) as a JSON array (same object shape as bare `--json`), plural when a key resolves to an owning segment covering several rows | Plan says "matching row(s)"; array-of-same-shape is the only consistent reading and matches the existing deterministic `RenderJSON` discipline | S:65 R:80 A:75 D:70 |
| 9 | Confident | `show <key>` composes built-in defaults via the existing `defaultSubtree`/`flattenOrigin` machinery in cmd/fab/config.go rather than a new composition path | The machinery already exists and single-sourcing is the code-quality bar (anti-pattern: duplicating utilities); verified present | S:70 R:80 A:85 D:80 |
| 10 | Confident | Change ships with **no migration file** | Additive verbs; no user-data restructuring; `init` bare goes error→behavior; alias preserves old command text — nothing matches the constitution's migration trigger | S:75 R:70 A:85 D:80 |
| 11 | Certain | Scope discipline on shared files: only this change's sections of `_cli-fab.md` (+ SPEC), `docs/specs/config.md`, `_shared/configuration.md` are edited; C1/C3/C4/C5/C6 content is untouched | Explicit in the dispatch instruction and plan § Orchestration | S:95 R:75 A:90 D:90 |

11 assumptions (4 certain, 7 confident, 0 tentative, 0 unresolved).
