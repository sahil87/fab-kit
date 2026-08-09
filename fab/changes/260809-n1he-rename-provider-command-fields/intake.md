# Intake: Rename Provider Command Fields

**Change**: 260809-n1he-rename-provider-command-fields
**Created**: 2026-08-09

## Origin

One-shot `/fab-new` invocation executing Change 4 of the config-overhaul plan:

> Config overhaul Change 4: rename providers config fields session_command/dispatch_command to interactive_command/headless_command with read-time alias, on-disk migration, and full kit-text sweep per fab/plans/sahil/config-overhaul.md Change 4.

All design decisions were resolved and user-confirmed in the 2026-08-08 `/fab-discuss` session recorded in `fab/plans/sahil/config-overhaul.md` (§ Change 4 and § Resolved decisions items 6–7). This intake transfers that plan section plus the current-repo state facts an implementing agent needs.

**Ordering context (verified against this branch, 2026-08-09)**: the plan's hard edge C3 → C4 is satisfied — Change 3 (`dispatch.mode` descent ladder, 260808-yilt) is merged and released in v2.18.1, as are Change 1 (env override layer, #553), Change 2 (six-verb `fab config` surface, #555), the config read-model redesign (#557), and the provider-roster change (#559: gemini removed; agy and kimi added). This change therefore lands on the post-C3 names and sweeps C3's text, exactly as the plan's recommended 3 → 4 order intends. Change 5 (source consolidation) is deliberately after this one.

## Why

1. **The names lie about the axis they split on.** `session_command` vs `dispatch_command` reads as tier-aligned (session = Tier-1 agents you talk to, dispatch = Tier-2 stage workers), but that is a false invariant: a Tier-2 **pane** stage worker runs the `session_command`. The fields actually split by **interaction mode** — "launch an interactive session a human can watch/steer" vs "run headless, prompt on stdin, exit when done" — and `interactive_command`/`headless_command` say exactly that.
2. **`dispatch_command` collides with two other meanings of "dispatch".** The field name, the `fab dispatch` verb family, and the `dispatch.*` config block (`dispatch.mode`, `dispatch.column_width`, `dispatch.reap_done`) are three different things sharing one word. Post-C3 the collision got worse: `dispatch.mode: native` resolves a stage to a rung that does *not* run the `dispatch_command`. `headless_command` breaks the collision.
3. **If we don't fix it now**, every new doc, skill restatement, and provider block written after C3 bakes the misleading names in deeper — and C5 (source consolidation) would consolidate around them. The rename is cheapest immediately after C3, whose sweep class this change's sweep deliberately shadows.
4. **Why alias + migration rather than a hard break**: user configs in both scopes (`fab/project/config.yaml`, `~/.fab-kit/config.yaml`) may carry `providers.<name>.session_command`/`.dispatch_command` overrides. The `agent.tiers` → `agent.profiles` rename (2.16.19-to-2.17.0) established the pattern: keep deprecated fields readable with new-spelling-wins resolution so a half-migrated config never breaks, and ship a migration that rewrites the disk.

## What Changes

### 1. `ProviderConfig` read-time alias (`src/go/fab/internal/config/config.go`)

Current struct (config.go:116–117):

```go
SessionCommand  string `yaml:"session_command"`
DispatchCommand string `yaml:"dispatch_command"`
```

Target shape — new canonical fields plus retained deprecated fields, mirroring the `Tiers`/`Profiles` precedent (config.go:187–201 declaration, config.go:813–838 read-path fallback):

```go
InteractiveCommand string `yaml:"interactive_command"`
HeadlessCommand    string `yaml:"headless_command"`
// Deprecated: pre-2.19 spellings, read-time alias only. Resolution prefers
// the new spelling PER FIELD, so a half-migrated config resolves everything.
SessionCommand  string `yaml:"session_command"`
DispatchCommand string `yaml:"dispatch_command"`
```

- **Per-field preference**: for each of the two fields independently, a non-empty new spelling wins; otherwise the deprecated spelling is read. A config with `interactive_command` on one provider and `dispatch_command` on another resolves both correctly during the migration window.
- The alias applies at the struct-decode/resolution seam, so it covers every layer of the four-tier cascade (env `FAB_PROVIDERS=...` values included) — anywhere YAML decodes into `ProviderConfig`.
- `ResolveProvider`'s per-field merge (agent.go:568–569 `mergeField` calls) and every consumer of the two fields (`resolve_agent.go`, `internal/dispatch` incl. `pane_mode.go`, `internal/spawn`, `cmd/fab/agent.go`, `batch_new.go`, `batch_switch.go`, `dispatch_start.go`, `operator.go`) move to the new canonical accessors.
- **No deprecation warning on old-spelling reads** — silent alias, parity with `agent.tiers`.

### 2. Built-in data and Go constants (`src/go/fab/internal/agent/`)

- `defaults.yaml`: rename the keys on all provider blocks (`claude.session_command` → `claude.interactive_command`, all four `dispatch_command` → `headless_command`) **and** sweep the comment prose — the header capability-grammar paragraph, the "agy and kimi deliberately carry NO session_command" block note, and each provider's per-block comments. The shipped defaults use only the new spellings (built-ins never need the alias).
- `agent.go`: rename the exported identifiers to match the new field names — `DefaultSessionCommand` → `DefaultInteractiveCommand`, `DefaultDispatchCommand` → `DefaultHeadlessCommand`, `DefaultCodexSessionCommand`/`DefaultCodexDispatchCommand`, `DefaultAgyDispatchCommand`, `DefaultKimiDispatchCommand` accordingly, plus doc comments (agent.go:118–184, 537–569). Docs cite these as `pkg.Symbol` qualifiers, so the identifier rename and the doc sweep must agree (grep-verify).
- Pinned tests update: `TestDefaultsFileIsWellFormed` (defaults_test.go), `TestDefaultRoleProfilesArePinned` / `TestNonClaudeProviderFillsArePinned` (agent_test.go), and the drift-guard tests that quote the field names (`TestMirrorDocsMatchDefaultProfiles`, `TestCLIFabReferenceListsDefaultRoles`, `TestDocTablesMatchAgentMaps`) as needed.

### 3. Registry and verb surface (`src/go/fab/internal/configref/`, `cmd/fab/config.go`)

- The `providers` registry row's rendered segment (configref.go:513 area) and `defaultsmap.go` interpolation follow the new key names.
- The **dynamic dotted-key matcher** for `set`/`unset`/`explain`/`show` (configref.go:969–1007: `len(parts) == 3 && parts[0] == "providers"` sub-field whitelist) accepts the **new spellings only**. An old-spelling dotted key (`fab config set providers.codex.dispatch_command …`) refuses as unknown-key with the standard `fab config explain` pointer — the alias is a file-read affordance, not a write-surface one; the migration is the fix for old keys on disk.
- **`renamed_from` metadata is set anyway** for `--json` consumers, as documented on the `agent.profiles` row (configref.go:170–176, 197–208): these are *nested* keys, so the upgrade engine's top-level `renamed_from` mechanical carry cannot fire — the metadata is informational, and the on-disk rewrite is the migration's job.
- Fence text in user configs (`fab/project/config.yaml` managed fence — e.g. the `dispatch.mode` paragraph's "pane needs tmux + session_command") regenerates from the updated registry segments via `fab config upgrade`; this repo's own `fab/project/config.yaml` fence is refreshed as part of the change.

### 4. Migration (`src/kit/migrations/`)

- New migration file rewriting **both scopes** on disk: in `fab/project/config.yaml` and `~/.fab-kit/config.yaml`, rename `providers.<name>.session_command` → `interactive_command` and `providers.<name>.dispatch_command` → `headless_command` for every provider block, preserving values and comments (no marshal round-trip — the yogn/#473 class).
- Named against the next release: `2.18.1-to-2.19.0.md` (current version 2.18.1; a rename shipping a migration is a minor bump per the `agent.tiers` precedent 2.16.19-to-2.17.0). Finalize the exact version at release.
- **Historical migration files stay verbatim** — `2.12.1-to-2.13.0.md`, `2.13.1-to-2.13.2.md`, `2.16.19-to-2.17.0.md` mention the old names as then-current facts and are instructions for older upgrades; rewriting them would corrupt the historical record (same rule C2 applied). Likewise archived/completed change artifacts under `fab/changes/` are transient records and are never swept.

### 5. Full kit-text sweep (grep both old names repo-wide, incl. user-facing string literals — the ioku lesson)

Current hit counts (2026-08-09, this branch) for `session_command|dispatch_command`:

| Surface | Files (hits) |
|---|---|
| Kit skills | `_cli-fab.md` (20), `_cli-agents.md` (16), `_preamble.md` (4), `fab-operator.md` (2) |
| SPEC mirrors | `SPEC-_cli-fab.md`, `SPEC-_cli-agents.md`, `SPEC-_preamble.md`, `SPEC-fab-operator.md` |
| Specs | `stage-models.md` (36), `architecture.md` (19), `harness-adapters.md` (14), `config.md` (5), `glossary.md` (3), `skills.md` (1), `index.md` (row descriptions) |
| Memory | `_shared/configuration.md` (17), `runtime/providers-and-profiles.md` (29), `runtime/dispatch.md` (17), plus `_shared/context-loading.md`, `distribution/kit-architecture.md`, `distribution/migrations.md`, `pipeline/execution-skills.md`, `runtime/agent-primitives.md`, `runtime/operator.md` |
| Go sources/tests | the `src/go/fab/` files in Impact below |
| This repo's own config | `fab/project/config.yaml` fence (regenerate) |

- Memory index descriptions: any file whose `description:` frontmatter mentions the old names gets its frontmatter edited; regenerate indexes with `fab memory-index` (index files are generated — never hand-edit).
- The sweep also owns C3's text (the ladder prose that says "pane needs tmux + session_command" etc. across `_preamble.md` § CLI-Adapter Dispatch, specs, memory) and the post-plan provider roster: the agy/kimi "NO session_command" rationale blocks become "NO interactive_command" (the plan predates #559's gemini→agy/kimi swap; the sweep binds to the *current* roster, not the plan's examples).
- `fab/plans/sahil/config-overhaul.md` itself is a historical planning record — not swept (its Change 1 comment "(fields renamed interactive_command/headless_command by Change 4)" already anticipates this change).
- **Out of scope**: `dispatch.mode`'s **value** `headless` and the `fab dispatch` command family keep their names — the rename disambiguates the *field* from them; it does not touch them.

## Affected Memory

- `_shared/configuration.md`: (modify) § providers — field names in the providers-table description, cascade examples, and dispatch prose
- `runtime/providers-and-profiles.md`: (modify) heaviest consumer (29 hits) — capability grammar, per-provider blocks, resolution seams
- `runtime/dispatch.md`: (modify) rung-prerequisite prose (pane = interactive command; headless = headless command)
- `_shared/context-loading.md`: (modify) always-load-layer description of config.yaml's providers table
- `distribution/kit-architecture.md`: (modify) registry/defaults references to the field names
- `distribution/migrations.md`: (modify) add the new migration to the catalog; any *quoted seeded comments* documented there stay VERBATIM
- `pipeline/execution-skills.md`: (modify) dispatch-related field-name mentions
- `runtime/agent-primitives.md`: (modify) field-name mentions
- `runtime/operator.md`: (modify) operator launcher's `WithProfile` substitution prose (templated `session_command` → `interactive_command`)
- `runtime/index.md` / `_shared/index.md` / `distribution/index.md` / `pipeline/index.md`: (modify) regenerated via `fab memory-index` after frontmatter description edits — never hand-edited

## Impact

- **Go**: `internal/config/config.go` (+ `config_test.go`), `internal/agent/agent.go`, `defaults.yaml`, `agent_test.go`, `defaults_test.go`, `internal/configref/configref.go`, `defaultsmap.go`, `configref_test.go`, `internal/configupgrade` (if segment rendering touches the names), `internal/dispatch/dispatch.go`, `pane_mode.go`, `dispatch_test.go`, `internal/spawn/spawn.go` + test, `cmd/fab/` (`agent.go`, `batch_new.go`, `batch_switch.go`, `config.go`, `dispatch_start.go`, `operator.go`, `resolve_agent.go` + their tests). Constitution: CLI-surface changes require test updates and an `_cli-fab.md` update.
- **Kit content**: 4 skill files + 4 SPEC mirrors (constitution: skill edits update SPEC mirrors; canonical sources under `src/kit/skills/`, never `.claude/skills/`).
- **Docs**: 7 spec files, ~10 memory files + regenerated indexes.
- **Migration**: one new file; users' configs in both scopes rewritten on next `/fab-setup migrations`; pre-migration configs keep working via the alias.
- **Behavior**: zero functional change — resolution, the descent ladder, and all dispatch behavior are name-invariant. The one user-visible surface change: `fab config set/unset/explain/show` dotted keys accept the new spellings and refuse the old.
- **Cross-change**: C5 (source consolidation) follows this change and inherits the new names; the plan's C4 → C5 soft ordering is preserved.

## Open Questions

None — all design decisions were resolved and user-confirmed in `fab/plans/sahil/config-overhaul.md` (2026-08-08 session), and the ordering precondition (C3 merged) is verified against this branch.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename splits by interaction mode: `interactive_command`/`headless_command`; tier-aligned naming rejected (pane workers are Tier-2 and run the interactive command) | Plan § Resolved decisions item 7, user-confirmed 2026-08-08 | S:95 R:60 A:95 D:95 |
| 2 | Certain | Read-time alias keeps both deprecated fields on `ProviderConfig`; resolution prefers the new spelling per field (half-migrated configs fully resolve) | Plan § Change 4 bullet 1; `agent.tiers` → `agent.profiles` precedent in config.go | S:90 R:70 A:95 D:90 |
| 3 | Certain | Migration rewrites both scopes on disk; `renamed_from` metadata set even though the top-level mechanical carry cannot fire on nested keys | Plan § Change 4 bullet 2, explicit | S:90 R:65 A:90 D:90 |
| 4 | Certain | Historical migration files and archived change artifacts stay verbatim; the config-overhaul plan doc is not swept | C2's identical, user-confirmed rule; migrations are instructions for older upgrades | S:80 R:80 A:90 D:90 |
| 5 | Confident | Sweep binds to the current provider roster (claude/codex/agy/kimi per merged #559), not the plan's pre-roster examples; agy/kimi "NO session_command" prose becomes "NO interactive_command" | Plan predates the roster change; repo state is authoritative | S:75 R:70 A:85 D:75 |
| 6 | Confident | Go identifiers renamed to match (`SessionCommand` → `InteractiveCommand`, `DefaultSessionCommand` → `DefaultInteractiveCommand`, etc.); deprecated alias fields keep the old struct names | Docs cite `pkg.Symbol` qualifiers (recorded grep-verify lesson); leaving stale identifiers would re-create the naming lie in Go | S:60 R:80 A:80 D:70 |
| 7 | Confident | `fab config set/unset/explain` dotted keys accept only new spellings; old spellings refuse as unknown-key with the `explain` pointer | Alias is read-time by plan wording; C2's unknown-key refusal is the established write-surface contract | S:55 R:75 A:70 D:65 |
| 8 | Confident | Migration file named `2.18.1-to-2.19.0.md` (minor bump), exact target version finalized at release | `agent.tiers` rename shipped as 2.16.19-to-2.17.0 (minor); current version 2.18.1 | S:60 R:85 A:75 D:75 |
| 9 | Confident | No deprecation warning when the alias reads an old spelling — silent, parity with `agent.tiers` | Precedent reads `Tiers` silently; the migration, not nagging, is the closure mechanism | S:55 R:80 A:70 D:65 |

9 assumptions (4 certain, 5 confident, 0 tentative, 0 unresolved).
