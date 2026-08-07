# Intake: Consolidate the fab config Subcommand Surface

**Change**: 260807-k0v3-consolidate-config-surface
**Created**: 2026-08-07

## Origin

<!-- Conversational — /fab-discuss session 2026-08-07. The user asked to review the
     `fab config` surface (init/reference/show/upgrade), then requested a clean-slate
     design ("Without restrictions, come up with the cleanest subcommand surface first"),
     explicitly waiving churn concerns ("I am not worried about churn"). The agent's
     proposed surface was accepted verbatim: "Yes, draft this as a single intake. Then
     fab-fff after that." -->

> Check the "fab config" command surface area. Right now we have init, reference, show etc. Can you consolidate it. I want to be able to very clearly check current config, print certain things vs apply changes. What are your suggestions?

Key decisions from the discussion (user-confirmed):

- Collapse the get-intent into `show [<key>]` — one inspect verb for state, whole-config and single-value
- Rename `reference` → `docs`, adding a per-key mode (`docs [<key>]` prints one field's block)
- Add fence-engine-backed `set <key> <value> [--system]` and `unset <key> [--system]` — surgical writes through the same `internal/configupgrade` engine `upgrade` uses (never a YAML round-trip)
- `unset` is first-class, not sugar: under presence=intent, removing a pinned field restores tier inheritance
- `init` defaults to project mode (`--system` stays as the flag)
- `upgrade` unchanged
- No `edit` verb; the reserved `validate` slot is dropped (set/unset's scope enforcement + unknown-key refusal + `show --origin` cover it)
- Spec framing: `set`/`unset` are surgical forms of `upgrade` — same engine, same invariants, not a new ownership model

## Why

1. **The pain point**: the current four subcommands (`show`, `reference`, `init`, `upgrade`) don't map one-to-one onto the three user intents — *check current config*, *print one specific thing*, *apply a change*. "Print one thing" today is `fab config show --origin | grep`; "apply a change" is hand-editing YAML followed by `fab config upgrade`. The show-vs-reference split (state vs schema docs) is real but not self-teaching from the names.

2. **The consequence of not fixing it**: agents and skills keep hand-editing `fab/project/config.yaml` — the exact operation class that produced the comment-mashing bug (the `setFabVersion` map round-trip stripped every user comment; fixed by line-splice in change yogn, PR #473). Without a first-class `set`, every future config-touching skill re-risks that bug. And scriptable per-key reads stay impossible without YAML parsing on the caller's side.

3. **Why this approach**: the fence engine (`internal/configupgrade`) now exists as the single comment-aware writer — surgical `set`/`unset` become thin verbs over proven machinery rather than a new ownership model. Collapsing get into `show [<key>]` (rather than a separate `get`, git's choice) is right for fab because there are no multi-value keys. Renaming `reference` → `docs` makes the read pair self-teaching: `show` = your state, `docs` = the schema.

## What Changes

Target surface (six verbs, three groups):

```
# Inspect — "what IS"
fab config show [<key>] [--origin]            # effective post-cascade config; with <key>, just that value

# Inspect — "what CAN be"
fab config docs [<key>] [--json]              # registry documentation (rename of `reference`); with <key>, that field's block

# Mutate — surgical, fence-aware
fab config set <key> <value> [--system]       # write one field through the configupgrade engine
fab config unset <key> [--system]             # remove an override, restoring inheritance/default

# Lifecycle — whole-file
fab config init [--system]                    # bootstrap; bare = project mode (--project flag retained)
fab config upgrade                            # mechanical reconcile against the registry (unchanged)
```

### `show [<key>]` — single-value mode

- `show` with no argument: unchanged (merged project-over-system YAML; `--origin` composes defaults and annotates provenance).
- `show <dotted.key>`: prints just that key's effective value, defaults composed (unlike bare `show`, which deliberately does not materialize defaults — the single-key form answers "what value is in effect", so it resolves the full cascade including built-ins). A scalar/list leaf prints as the raw value (scriptable, no `key =` prefix); a map-valued key (e.g. `agent.tiers.review`) prints the YAML subtree.
- `--origin` works at both depths: with `<key>` it appends the provenance annotation(s) for the matched leaf/leaves in the existing `key = value  # origin` format.
- An unknown key (no registry field matches and no configured value exists at that path) exits non-zero with an error naming the key.
- Implementation note: `renderShowOrigin`/`flattenOrigin` already produce dotted leaves with per-leaf origin; single-key mode is a prefix filter over that walk.

### `reference` → `docs [<key>]`

- `fab config docs` prints exactly what `fab config reference` prints today (the fully-commented reference config.yaml, byte-stable, generated from the field registry).
- `fab config docs <dotted.key>` prints only that field's rendered segment (description comment block + default), resolving the key to its registry row (`configref.Fields()`); keys documented inside another field's segment resolve to the owning segment. Unknown key exits non-zero.
- `--json` retained: bare emits the full field table (unchanged); with `<key>` it emits only the matching field row(s) as a JSON array.
- `reference` is retained as a cobra alias on the `docs` command (aliases do not appear in the group's command list, so the visible surface stays clean). This protects the `# Full reference of all available options: fab config reference` pointer comments seeded into user config.yaml files by migration 2.9.2-to-2.10.0 and keeps historical migration texts functional verbatim. All current kit/doc text migrates to `docs`.

### `set <key> <value> [--system]`

- Writes one field through the `internal/configupgrade` fence engine — never a YAML marshal round-trip. For a project-file write: if the field is live above the fence, splice the new value into the live line; if it is only a commented reference line inside the fence, materialize it live above the fence with the new value (presence=intent: setting a field pins it). The fence itself is regenerated per the engine's existing byte-stable rules.
- Refuses unknown keys: `<key>` must resolve to a registry field (or a leaf within a map-valued registry field). Error names the key and suggests `fab config docs`.
- Scope-enforced: writing a project-scoped field with `--system` (or into the system file generally) is an error naming the field's scope, per the `internal/configscope` taxonomy. This plus unknown-key refusal delivers the typo-linting the reserved `validate` slot was held for — `validate` is dropped.
- Deep dotted keys into map-valued fields are supported — the primary use case is pinning tiers/providers: `fab config set agent.tiers.review.model claude-opus-5`, `fab config set providers.codex.model gpt-5.3-codex`.
- `<value>` is parsed as a YAML scalar (so `true`, `3`, and quoted strings behave as YAML); a list value is accepted in YAML flow syntax (`'[a, b]'`).
- `--system` targets `~/.fab-kit/config.yaml`, creating it (with the existing scaffold header) when missing — git-config precedent: `set` never fails for lack of a file.
- Output: one confirmation line naming key, value, and file written.

### `unset <key> [--system]`

- Removes the field's live override from the target file through the same engine; the project file's fence regains/retains the commented reference line for that field (so the file continues to document the now-inherited default). Restores inheritance: system value, else built-in default — semantic under presence=intent, e.g. `unset agent.tiers.review.provider` restores the built-in review tier.
- Same unknown-key refusal and scope enforcement as `set`. Unsetting a key that is not currently set in the target file is a no-op with a notice (idempotent, exit 0).

### `init` — bare defaults to project mode

- Bare `fab config init` (today a usage error) becomes project mode: generates `fab/project/config.yaml` from the registry, identical to `--project`.
- `--project` is retained (explicit form; `fab init` in the fab-kit binary shells out to `fab config init --project` and must keep working). `--system` unchanged. `--system --project` together: still an error.
- Both modes still refuse to overwrite an existing target file.

### `upgrade` — unchanged

No behavior change. Spec framing only: `set`/`unset` are documented as surgical forms of the same reconcile engine.

### Group help regrouped

`fab config --help` lists subcommands in three cobra command groups — Inspect (`show`, `docs`), Modify (`set`, `unset`), Lifecycle (`init`, `upgrade`) — via `cobra.Group`/`GroupID`. The group `Long` text is rewritten around the intent model (what IS / what CAN be / apply changes).

## Affected Memory

- `_shared/configuration.md`: (modify) primary — documents the `fab config` group (reference/show/init/upgrade → the new six-verb surface, set/unset semantics, docs rename)
- `distribution/kit-architecture.md`: (modify) covers "the fab config group + loader cascade" — update the group's verb list; also falsified beyond the rename: the `cobra.NoArgs` claim for the reference/docs subcommand (now `cobra.MaximumNArgs(1)`) and the "leaves room for a future `fab config validate`" sentence (the slot is dropped)
- `distribution/setup.md`: (modify) light — `fab config init --project` mentions remain valid; note bare-init default where the flow is described
- `runtime/providers-and-tiers.md`: (modify) two instructions to run `fab config reference` (lines ~48, ~87) → `fab config docs`
- `distribution/migrations.md`: (modify) the schema-discovery mention of `fab config reference` (~line 137) → `fab config docs`; the quoted `# Full reference of all available options: fab config reference` config-comment example (~line 134) stays VERBATIM — that seeded comment is the artifact the `reference` alias protects

## Impact

- **Go**: `src/go/fab/cmd/fab/config.go` (all four commands + two new ones + groups); `src/go/fab/internal/configupgrade` (new surgical set/unset operations on the fence engine); `src/go/fab/internal/configref` (per-key segment/row lookup); `src/go/fab/internal/configscope` (consumed for set/unset enforcement, likely no change). `src/go/fab-kit/internal/init.go` untouched (`--project` retained).
- **Tests** (constitution: CLI change ⇒ tests): `config_test.go`, `config_show_init_test.go`, `config_upgrade_test.go` extended; new coverage for show-single-key, docs-per-key, set (live-splice, fence-materialize, scope refusal, unknown-key, system-file creation, comment preservation — regression for the yogn class), unset (inheritance restore, idempotence), init bare-default.
- **CLI reference** (constitution): `src/kit/skills/_cli-fab.md` § fab config rewritten; SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md`.
- **Spec**: `docs/specs/config.md` (the surface section + set/unset-as-surgical-upgrade framing + validate-slot removal); `docs/specs/index.md` description row.
- **Kit text sweep** for `fab config reference` → `docs`: `src/kit/skills/fab-setup.md` (+ its SPEC mirror), any other skill prose; `src/go/fab/cmd/fab/skill.md`; `docs/site/skill.md` if it restates the surface. Historical migration files under `src/kit/migrations/` are left verbatim (the `reference` alias keeps them functional).
- **Toolkit standards** (constitution § Toolkit Standards): this changes the CLI surface and help output — the plan MUST check the change against `shll standards` before finalizing command naming/help text.

## Open Questions

- (none — all decisions resolved in the /fab-discuss session or graded below)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Collapse get-intent into `show [<key>]`; `--origin` at both depths; unknown key exits non-zero | Discussed — user accepted the proposed surface verbatim | S:95 R:85 A:95 D:95 |
| 2 | Certain | Rename `reference` → `docs` with per-key mode; all repo text migrates | Discussed — user explicitly waived churn concerns | S:95 R:80 A:95 D:90 |
| 3 | Confident | Keep `reference` as a cobra alias on `docs`, invisible in the group command list | Not discussed; protects pointer comments seeded into user config files by migration 2.9.2-to-2.10.0 and keeps historical migration texts functional, at zero visible-surface cost | S:60 R:90 A:80 D:75 |
| 4 | Certain | `set`/`unset` write through the configupgrade fence engine, never a YAML round-trip | Discussed — comment-preservation is the point (yogn/#473 regression class) | S:90 R:70 A:95 D:90 |
| 5 | Certain | Scope enforcement + unknown-key refusal on `set`/`unset`; the reserved `validate` verb is dropped | Discussed — "quietly delivers the typo-linting validate was reserved for" | S:85 R:85 A:90 D:85 |
| 6 | Confident | `set` supports deep dotted keys into map-valued fields (agent.tiers.*, providers.*); value parsed as YAML scalar, lists via flow syntax | Pinning tier models is the primary use case; registry rows for map fields exist to resolve against | S:65 R:75 A:82 D:70 |
| 7 | Confident | `set --system` creates `~/.fab-kit/config.yaml` (scaffold header) when missing; `unset` of an unset key is an exit-0 no-op with notice | git-config precedent; idempotence per Constitution III | S:55 R:82 A:78 D:70 |
| 8 | Certain | Bare `fab config init` = project mode; `--project` flag retained (fab init shell-out unaffected); overwrite refusal unchanged | Discussed — "bootstrapping the repo config is the common case"; error→behavior is the safe breaking direction | S:88 R:85 A:92 D:88 |
| 9 | Confident | Group help via cobra command groups: Inspect / Modify / Lifecycle | Discussed as intent-grouping; cobra Groups is the idiomatic mechanism | S:75 R:95 A:90 D:85 |
| 10 | Confident | `show <key>` output shape: raw value for scalar/list leaves (no `key =` prefix), YAML subtree for map keys; single-key mode composes built-in defaults | Scriptability requires the bare value; bare `show`'s no-defaults rule answers a different question than "what is in effect for this key" | S:60 R:88 A:80 D:68 |
| 11 | Confident | `docs --json <key>` filters the JSON field table to the matching row(s) | Symmetry with the YAML per-key mode; the JSON table is the tooling surface | S:58 R:90 A:82 D:75 |

11 assumptions (5 certain, 6 confident, 0 tentative, 0 unresolved).
