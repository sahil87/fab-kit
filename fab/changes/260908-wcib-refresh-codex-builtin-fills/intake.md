# Intake: Refresh Codex Built-in Fills

**Change**: 260908-wcib-refresh-codex-builtin-fills
**Created**: 2026-09-08

## Origin

> Refresh the built-in codex per-role model fills in `src/go/fab/defaults.yaml` (`providers.codex.profiles`) to use the current codex model catalog, with GPT-6 Astra as the codex `default` and every role carrying an explicit model (no silent inheritance of `default`'s model). Target: `default`/`doing` → `gpt-6-astra @ high`; `review` → `gpt-5.6-sol @ xhigh`; `hydrate` → `gpt-5.6-sol @ high` (new explicit row); `operator` → `gpt-5.6-luna @ medium`; `fast` → `gpt-5.6-luna @ low`. Data + test + spec-mirror change; `change_type` is `feat`.

**Interaction mode**: promptless dispatch (`{questioning-mode} = promptless-defer`). The description was synthesized by the team lead from a prior discussion with the user and handed to the intake agent; no questions were asked during intake. Every decision below was either made in that discussion (recorded as Certain/Confident with the rationale it was made with) or is a routine judgment call the intake agent made and recorded in `## Assumptions`. No decision landed Unresolved.

**Verification performed at intake** (2026-09-08, this worktree):

- The installed codex CLI is `codex-cli 0.153.4`. Its model catalog (`~/.codex/models_cache.json`) lists, in priority order: `gpt-6-astra` (1, "Our most capable model for complex, demanding work", default effort `medium`, supports `low`…`ultra`), `gpt-reserve` (3, hidden), `gpt-5.6-sol` (6, "Reliable agentic workhorse for everyday tasks", default effort `low`, supports `low`…`ultra`), `gpt-5.6-terra` (7, "Balanced agentic coding model for everyday work"), `gpt-5.6-luna` (8, "Fast and affordable agentic coding model", supports `low`…`max`, no `ultra`), `gpt-5.5` (12, previous generation), `gpt-5.4-mini` (23), `gpt-5.3-codex-spark` (26), `codex-auto-review` (43, hidden). Every slug in the target table is present with every effort level the table uses.
- The user's own `~/.codex/config.toml` already sets `model = "gpt-6-astra"`.
- Current resolution (`fab agent <role> --provider codex -o yaml`, six roles): `default` sol/high, `doing` sol/xhigh, `review` sol/xhigh, `hydrate` sol/high (inherited), `operator` luna/medium, `fast` luna/low.
- `TestMirrorDocsMatchDefaultProfiles` and `TestCLIFabReferenceListsDefaultRoles` **do not exist** as functions anymore (only comment/doc references survive at `src/go/fab/internal/agent/agent_test.go:59-60,98`, `docs/specs/stage-models.md` § Drift guard, `docs/memory/runtime/providers-and-profiles.md:225`). The spec's inline codex sample is shape-only with `...` elided IDs and is not test-guarded; `defaults.yaml`'s own header says the claude table is "the ONE doc mirror left". The brief's statement that the sample is checked by that test is therefore stale, and this intake corrects it.

## Why

1. **The pain point.** The shipped codex fills were pinned on 2026-08-07 (`260806-ywkx`) from the then-installed `codex-cli 0.146.0`, when `gpt-5.6-sol` was the priority-1 "latest frontier agentic coding model". The installed catalog has moved: `gpt-6-astra` is now priority 1 and codex itself now describes sol as a "reliable agentic workhorse for everyday tasks" whose default effort is `low`. A user who points `agent.workers: codex` at the pipeline today gets the second-tier model as author, critic, and memory writer, which is exactly the role-flattening the fills exist to prevent. Separately, the current table leaves `doing`/`review` model-less (effort-only rows) and `hydrate` absent, so a future edit to `default`'s model silently repoints three roles at once; there is no spot where a reader can see what `doing` resolves to without knowing the per-field merge rule.

2. **The consequence of not fixing it.** `docs/specs/stage-models.md` § Refreshing the non-claude fills makes refresh-at-kit-release-cadence the whole justification for shipping fills in the binary ("fab-kit releases every few days, so users see refreshed suggestions at kit cadence rather than CLI cadence"). A catalog that has moved without the fills following it is the rot argument the ywkx lineage note claims to have retired. Left alone, every codex-worker run is one tier below what the user has already chosen for their own interactive sessions.

3. **Why this table.**
   - `default` → astra @ `high`: the role covers intake, bare `fab agent` sessions, and `fab batch` workers, all judgment-heavy. Astra's own default effort is `medium` (chat-tuned); fab pins `high`, matching the claude fills' convention (every claude frontier-model role runs `high`).
   - `doing` → astra @ `high`, model pinned explicitly: apply is "execution that must not err" and the stage where rework originates, so the top model belongs there (spec § apply↔review coupling). `high` rather than the current `xhigh` follows the spec's stated sweet-spot rationale for the claude fills (`xhigh` buys marginal gains at disproportionate latency/cost). Pinning the model line closes the silent-inheritance path where a change to `default`'s model leaks into `doing`.
   - `review` → sol @ `xhigh`: keep the critic on a **different model than the author** (astra writes, sol reads) so the reviewer does not share the author's blind spots. The 2026-08-10 four-provider comparison (`docs/findings/four-providers-rvza-ttff.html`) recorded that reviewer strictness tracks the worker model across all four arms; sol@xhigh's strictness is proven in real runs (the all-codex arm of `kimi-vs-codex-rvza-ttff`).
   - `hydrate` → sol @ `high`, now an explicit row: memory writing is sweep-heavy prose where sol performed well (the ttff run). Previously inherited from `default`; making it explicit is what lets the table be read without knowing the merge rule.
   - `operator` and `fast` → luna (`medium` / `low`): unchanged; protocol-following and mechanical work.

4. **Alternatives rejected.**
   - `gpt-5.6-terra` ("Balanced agentic coding model"): no evidence in this repo, so not shipped as a default. A/B candidate only.
   - `gpt-5.5` (previous generation), `gpt-5.4-mini` (codex marks it deprecated in favour of luna), `gpt-5.3-codex-spark` (not in the API), `gpt-reserve` and `codex-auto-review` (hidden visibility): excluded.
   - Keeping `doing`/`review` as effort-only rows: rejected because the silent inheritance is the defect, not a convenience.
   - Effort `ultra` on any role: **never shipped**. Codex describes `ultra` as "Maximum reasoning with automatic task delegation"; a pane or headless worker that spawns its own sub-agents would break fab's single-worker dispatch contract and the `{stage}-result.yaml` protocol (`docs/specs/harness-adapters.md`). `max` stays an explicit-override-only level (the claude fills top out at `high`; nothing has shown headroom above `xhigh`).
   - A migration: not needed. Built-in fills live in the binary and are never seeded into user config, so an upgrade refreshes them and no project pins rot in place (spec § Refreshing the non-claude fills).

5. **Known limitation (not a resolved fact).** Codex per-model pricing was **not** available offline. The tier ladder (astra > sol > luna) is inferred from codex's own descriptions and priority order, not from cost data. Record this in the spec/memory rationale as an inference; do not present it as measured.

## What Changes

### 1. `src/go/fab/defaults.yaml` — the data diff (the SINGLE source)

Replace the codex `profiles:` block. Before:

```yaml
    # Sparse: a role absent here resolves the `default` entry, so hydrate gets
    # the frontier model at `high` without a row of its own. The codex CLI takes
    # a concrete model SLUG (no alias mechanism), so these are pinned IDs and
    # are the rows to bump when the catalog moves.
    profiles:
      default:  { model: gpt-5.6-sol,  effort: high }
      operator: { model: gpt-5.6-luna, effort: medium }
      doing:    { effort: xhigh }        # model inherits `default`
      review:   { effort: xhigh }
      fast:     { model: gpt-5.6-luna, effort: low }
```

After (values exact; comment wording is the author's, but it MUST carry the four points listed below):

```yaml
    profiles:
      default:  { model: gpt-6-astra,  effort: high }
      doing:    { model: gpt-6-astra,  effort: high }
      review:   { model: gpt-5.6-sol,  effort: xhigh }
      hydrate:  { model: gpt-5.6-sol,  effort: high }
      operator: { model: gpt-5.6-luna, effort: medium }
      fast:     { model: gpt-5.6-luna, effort: low }
```

The replacement comment MUST state: (a) codex's `-m` takes a concrete catalog slug (no alias mechanism), so these are pinned IDs and the rows to bump when the catalog moves — verify against the installed binary's catalog (`~/.codex/models_cache.json`), never from memory; (b) the map is **dense on purpose**: every role pins its own model so a `default` bump never silently repoints another role (the pre-wcib effort-only rows inherited `default`'s model); (c) the author/critic split: `doing` and `review` deliberately run **different** models; (d) the effort ceiling: never ship `ultra` (automatic task delegation breaks the single-worker dispatch contract), and `max` is an explicit-override level only. Keep the two `interactive_command`/`headless_command` lines and their comment untouched. Also revise the file-header sentence at line ~74 ("claude, codex and agy ship fills…") only if it makes a sparseness claim; the header's bump procedure (lines 16-31) stays but its drift-guard list is already accurate for the tests that exist.

No Go file holds a copy of these values; `DefaultCodex*` package vars read the parsed entries.

### 2. Tests

- **`src/go/fab/internal/agent/agent_test.go` — `TestNonClaudeProviderFillsArePinned`** (line 99). This is the deliberate-change pin. Update the codex table to the six rows above (`RoleDefault`, `RoleDoing`, `RoleReview`, `RoleHydrate`, `RoleOperator`, `RoleFast`, each with `Model` and `Effort`). Rewrite the preceding doc comment (lines 91-98): the tables are sparse **where `defaults.yaml` is** — agy's still is, kimi's is empty, codex's is now dense by policy — and the "guarded separately by `TestMirrorDocsMatchDefaultProfiles`" sentence must name the tests that actually exist (`TestDocTablesMatchAgentMaps` for the claude table; `TestConfigReferenceDocumentsProviderFill` for the rendered reference). Apply the same correction to the identical stale sentence at lines 59-60 above `TestDefaultRoleProfilesArePinned`.
- **`src/go/fab/internal/agent/defaults_test.go`** — `TestDefaultsFileIsWellFormed` (line 45) already asserts claude's six roles each carry model + effort and its siblings assert each filled built-in's `default` names a model. Add one structural assertion (in this test or a sibling) that **every codex role fill names a model** — the "dense by policy" invariant, so a future bump cannot reintroduce an effort-only codex row unreviewed. agy stays exempt (model-only sparse map is its documented shape) and kimi stays asserted empty. Update the comment at lines 73-77 ("claude, codex and agy carry per-role fills") only if it claims codex is sparse (it does not today; verify).
- **`src/go/fab/cmd/fab/config_test.go`** — `TestConfigReferenceDocumentsProviderFill` (line 874) derives expectations from `ResolveProvider`, so it needs no value edits; fix its doc comment at line 871 ("codex's and agy's sparse maps") to "agy's sparse map and codex's dense one" or equivalent.
- **Do NOT edit** `src/go/fab/internal/configupgrade/configupgrade_test.go`'s `legacyV2198ToV2220ProvidersAdvert` (lines ~1131-1160, plus its uses at ~1236 and ~1275). It is a byte-for-byte snapshot of the released v2.19.8–v2.22.0 reference output and is deliberately independent of the current registry ("cannot exercise history" otherwise). The current rendering already differs from it (claude's permission flags changed); this change widens the difference, which is the drift that test family simulates. Run that package to confirm it stays green.
- Go comments that restate the sparse shape and go stale: `src/go/fab/internal/agent/agent.go:237` ("codex — … its own SPARSE per-role fills"), `src/go/fab/internal/configref/configref.go:73-74` ("codex's and agy's fill maps are SPARSE") and `:261` ("a sparse fill map (codex, agy)"). Reword each so agy is the sparse example and codex is dense.

### 3. `docs/specs/stage-models.md` — the spec mirror (§ Built-in providers)

- **Inline YAML sample (lines 260-268)**: change the codex block from the sparse shape to the dense one, keeping the spec's rule that model IDs are elided (`...`) outside the claude table:

  ```yaml
    codex:
      interactive_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
      headless_command: 'codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
      profiles:                                 # dense — every role pins its own model (no inheritance from `default`)
        default:  { model: ..., effort: ... }
        doing:    { model: ..., effort: ... }
        review:   { model: ..., effort: ... }   # a DIFFERENT model from doing — the critic never shares the author's blind spots
        hydrate:  { model: ..., effort: ... }
        operator: { model: ..., effort: ... }
        fast:     { model: ..., effort: ... }
  ```

- **Prose at lines 281-285** ("The maps are SPARSE for the non-claude providers… codex's `hydrate` lands on codex's `default`… codex's `doing`/`review` rows — effort only — take their model from `default` too"): rewrite. agy's map is the sparse one (its non-`fast` roles take `default`); codex's map is dense by policy, and the per-field merge rule still exists for user overrides but no shipped codex row relies on it.
- **Consequence bullet at lines 336-338** ("`agent.workers: codex` resolves `xhigh` for apply/review and codex's cheaper `fast` model at `low` for ship"): rewrite role-relatively without literal IDs, e.g. apply runs codex's top catalog model, review runs a different model at codex's highest non-delegating effort, ship takes the cheaper `fast` model at `low`.
- **Add a short "Why these codex fills" paragraph** in § Built-in providers (role-relative, no literal IDs, mirroring the claude "Why these defaults" paragraph in § Default role profiles): top catalog model on `default`/`doing`; a different model as the critic on `review` with the strictness evidence from `docs/findings`; the sweep-prose model on `hydrate`; the cheap model on `operator`/`fast`; the `ultra` exclusion and `max`-override-only rule; the cost-ladder-is-inferred limitation. Leave § Refreshing the non-claude fills (lines 356-391) as the governing rule; no change expected there beyond, at most, one clause pointing at the catalog file as the verification source.
- **Upgrade note (no migration)** — add beside § Upgrade note — the hydrate split (line 855): a project carrying the deprecated flat `providers.codex.model` (an alias for `profiles.default`) previously saw that value reach `doing`, `review`, and `hydrate` because those shipped rows carried no model; after this change every codex row pins a model, so the flat value reaches only the `default` role. The modern spelling `providers.codex.profiles.<role>.model` is the fix; nothing restructured, so an upgrade note, not a migration (same precedent as the hydrate split).
- **§ Drift guard (line ~903)**: replace the `TestMirrorDocsMatchDefaultProfiles` sentence with what is true — the inline sample is shape-only and unguarded by design; the rendered reference's fill lines are guarded by `TestConfigReferenceDocumentsProviderFill`.
- Extend the decision-lineage note (line 378) with `260908-wcib` as the first catalog refresh under the ywkx policy, one clause.

### 4. Rendered config reference (user-facing string literals)

`src/go/fab/internal/configref/configref.go` lines 785-789 render: "claude, codex and agy carry per-role fills, and every role still resolves a model suited to it; the non-claude maps are SPARSE: a role absent from one takes that provider's `default` entry." Reword so only agy is called sparse (codex dense). The fill lines themselves are interpolated from `ResolveProvider` and need no edit. Re-run `fab config explain providers` from the locally built binary and eyeball the codex block (six rows, `RoleNames()` order).

### 5. Skill prose (`src/kit/skills/`, canonical — never `.claude/skills/`)

- `_cli-fab.md:349` — "Three of the four built-ins carry per-role fills (claude exhaustively, codex and agy sparsely)" → codex now exhaustive too; agy sparse.
- `_cli-fab.md:340-344` and `:364-369` — the illustrative `fab resolve-agent apply` outputs under codex show `effort=xhigh`. After this change apply (`doing`) resolves `high`. Change both to `high` (or to a `<effort>` placeholder); the surrounding "values illustrative" caveat stays.
- `_cli-agents.md:187` — "fab ships per-role codex fills (`default`/`doing`/`review`/`fast`)" is already stale (omits `operator`) and becomes more so; say all six roles. In the same Model-discovery cell, add `~/.codex/models_cache.json` (the installed CLI's cached catalog, which carries slug, priority, description, and supported reasoning levels) as the verified source alongside the existing `codex --version` / `--help` probe. This is a recipe, not a catalog, so it stays within the dictionary's no-model-IDs rule.
- No command signature changes, so no `_cli-fab.md` § fab agent signature edits and no `docs/specs/skills.md` flow edits.

### 6. Other spec restatements (sibling sweep)

- `docs/specs/glossary.md:89` — `providers` entry says "codex with both command grammars, no native capability, and sparse fills" → "and a dense fill map (all six roles)"; agy keeps "sparse model-only fills".
- `docs/specs/architecture.md:259` ("Codex's -m takes a concrete model SLUG, so its shipped fills are pinned IDs") stays true; check the sample block below it (line ~275 onward) for a codex `profiles:` shape and align if it shows effort-only rows.
- `docs/specs/config.md:101` is shape-neutral; verify only.

### 7. Sweep procedure (code-quality.md § Sibling Sweeps, must run before apply finishes)

Grep repo-wide, excluding `fab/changes/`, `.claude/`, and `docs/findings/`, and update every occurrence in the class:

```
gpt-5.6-sol                      # expect: defaults.yaml + agent_test.go pin table only (legacy fixture excluded)
model inherits                   # expect: zero after the change
inherits \`default\`              # expect: zero for codex
effort only|effort-only          # codex references → zero; agy/none references untouched
SPARSE|sparse                    # codex references → zero; agy references stay
codex and agy sparsely|sparse maps
xhigh for apply|apply/review.*xhigh
```

Exclusions from the sweep (historical, must not be edited): `src/kit/migrations/2.16.19-to-2.17.0.md` (describes the 2.16.19 state), `configupgrade_test.go`'s legacy advert fixture, everything under `fab/changes/` and `docs/findings/`, and `docs/memory/**/log.md` / `log.seed.md` (generated/frozen).

### 8. `change_type`

`feat` — a behavior change in shipped defaults (a codex-configured project resolves different models after upgrading). Set explicitly with `fab status set-change-type` so `fab status refresh` cannot re-infer `chore`/`docs` from the data-diff wording.

### Non-goals

- No change to agy or kimi fills, to any command grammar, or to the claude table.
- No effort-enum validation and no model/provider compatibility check (provider neutrality stands).
- No migration file.
- No A/B of `gpt-5.6-terra`; that is a follow-up idea if anyone wants it.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) built-in table row for `codex` (`profiles` column: sparse → all six roles); the "claude, codex and agy ship per-role fills… non-claude maps are sparse" paragraph (line ~119); the `fab resolve-agent apply --provider codex` scenario (line ~308, "inherited from codex's `default`, since the `doing` row carries effort only"); the flat-alias sentence at line ~415 ("a flat `model` on codex reaches every role whose shipped row sets no model of its own" — now only `default`); the drift-guard test list at line ~225 (drop the two non-existent test names); the Design Decision "Codex's Fills Are Catalog Slugs; agy's Embed Effort in the ID; kimi Ships None" (line ~531: Decision → dense map with the author/critic split and the `ultra` exclusion; Why → adds the silent-inheritance and different-model-critic rationale plus the cost-ladder-inferred limitation; Rejected → adds effort-only rows, `ultra`, terra; *Updated by* → `260908-wcib`).
- `_shared/configuration`: (modify) line ~173 ("codex's effort-only rows each render as written") → codex renders six model+effort rows; the omitempty note still applies to agy's model-only rows.
- `runtime/agent-primitives`: (modify) only if the `_cli-agents.md` codex discovery-recipe cell changes per § 5 — one clause at line ~91 noting the cached-catalog file as a discovery source.

## Impact

- **Code**: `src/go/fab/defaults.yaml` (data), `src/go/fab/internal/agent/agent_test.go`, `src/go/fab/internal/agent/defaults_test.go`, `src/go/fab/internal/agent/agent.go` (comment), `src/go/fab/internal/configref/configref.go` (comments + one rendered prose string), `src/go/fab/cmd/fab/config_test.go` (comment).
- **Specs**: `docs/specs/stage-models.md`, `docs/specs/glossary.md`, possibly `docs/specs/architecture.md`.
- **Skills (canonical)**: `src/kit/skills/_cli-fab.md`, `src/kit/skills/_cli-agents.md`.
- **Memory**: per Affected Memory (hydrate).
- **Behavior**: only projects (or sessions) that name `codex` on a knob, role override, or `--provider` flag see a difference; the shipped `claude` defaults are byte-unchanged. Within codex: `default` and `doing` move to astra, `doing` effort drops `xhigh` → `high`, `hydrate` becomes explicit (same resolved value as before for a stock config), `review` keeps sol@xhigh, `operator`/`fast` unchanged. Deprecated flat `providers.codex.model` overrides now reach only `default` (documented as an upgrade note).
- **No** CLI signature change, **no** migration, **no** `_cli-fab.md` command reference change.
- **Verification** (scope down first, then widen):
  1. From the Go module root (`src/go/fab`): `go test ./internal/agent/ ./internal/configref/ ./internal/configupgrade/ ./cmd/fab/`, then `go test ./...`.
  2. Build the local binary (`go build -o /tmp/fab-wcib ./cmd/fab` or the repo's usual build path) and run `<local-binary> agent <role> --provider codex -o yaml` for `default`, `doing`, `review`, `hydrate`, `operator`, `fast`; expect exactly the six target profiles. **Do not use the `fab` shim for this** — it routes to the binary matching the project's kit pin, not the working tree.
  3. `<local-binary> config explain providers` shows the codex block with six rows in `RoleNames()` order and the reworded prose.
  4. `gofmt -l` clean on touched Go files; `fab memory-index --check` after hydrate.

## Open Questions

- Codex per-model pricing was not retrievable offline, so whether astra on `default`/`doing` changes per-run cost materially versus sol is unknown. Not blocking — the user chose the table with this known; recorded as an inference in the rationale prose rather than as fact.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Target codex fill table is exactly the six rows in § What Changes 1: default/doing `gpt-6-astra`@high, review `gpt-5.6-sol`@xhigh, hydrate `gpt-5.6-sol`@high, operator `gpt-5.6-luna`@medium, fast `gpt-5.6-luna`@low | Discussed and decided; every slug and effort re-verified at intake against the installed codex-cli 0.153.4 catalog (`~/.codex/models_cache.json`, 2026-09-08) per spec § Refreshing the non-claude fills | S:95 R:85 A:95 D:95 |
| 2 | Certain | `default` and `doing` pin `high`, not astra's own `medium` default | Discussed — fab pins effort per role (claude convention); `medium` is chat-tuned, and both roles are judgment-heavy/execution-critical | S:90 R:85 A:85 D:85 |
| 3 | Certain | `review` runs a different model (sol@xhigh) from the author (astra) | Discussed — critic must not share the author's blind spots; `docs/findings` four-provider comparison shows reviewer strictness tracks the worker model; sol@xhigh proven in real runs | S:90 R:85 A:80 D:85 |
| 4 | Certain | `hydrate` becomes an explicit row (sol@high) instead of inheriting `default` | Discussed — sweep-heavy prose where sol performed well (ttff); explicit row makes the table readable without the merge rule | S:90 R:90 A:90 D:90 |
| 5 | Certain | `operator` and `fast` stay on luna at medium/low | Discussed — unchanged; protocol-following and mechanical work | S:95 R:90 A:95 D:95 |
| 6 | Certain | Excluded: terra, gpt-5.5, gpt-5.4-mini, spark, gpt-reserve, codex-auto-review; `ultra` is never shipped on any role; `max` is explicit-override only | Discussed — terra has no repo evidence; the others are previous-gen, deprecated, non-API, or hidden; `ultra`'s automatic task delegation breaks the single-worker dispatch contract | S:90 R:90 A:85 D:85 |
| 7 | Certain | `change_type` is `feat`, set explicitly via `fab status set-change-type` | Discussed — shipped-default behavior change; explicit set survives `fab status refresh` re-inference (recurring trap on data/docs-shaped diffs) | S:90 R:95 A:90 D:90 |
| 8 | Certain | No migration | Spec § Refreshing the non-claude fills: fills live in the binary, never seeded into user config, so an upgrade refreshes them and nothing restructures | S:90 R:90 A:95 D:95 |
| 9 | Certain | `configupgrade_test.go`'s `legacyV2198ToV2220ProvidersAdvert` fixture and `src/kit/migrations/2.16.19-to-2.17.0.md` are historical and are NOT edited | The fixture's own comment pins it as byte-for-byte released output kept independent of the current registry; the migration describes the 2.16.19 state | S:80 R:90 A:95 D:90 |
| 10 | Confident | The spec's inline codex sample is updated to the dense shape with `...` elided IDs, plus a role-relative "why these codex fills" paragraph with no literal IDs; the brief's claim that `TestMirrorDocsMatchDefaultProfiles` guards the sample is corrected (the test no longer exists; the sample is shape-only) | Verified by grep at intake — no such function in `src/go`; spec § Built-in providers states IDs are spelled only in the claude table (ywkx T023 de-literal rule) | S:60 R:85 A:85 D:80 |
| 11 | Confident | Sibling sweep of the "codex is sparse / effort-only / model inherits default" class across Go comments, the rendered `configref` prose string, `_cli-fab.md`, `_cli-agents.md`, `glossary.md`, `stage-models.md`, and memory — rewritten so agy is the sparse example and codex is dense | code-quality.md § Sibling Sweeps; fab-recurring-lessons: behavior-claim sweeps must include user-facing string literals; agy's map genuinely stays sparse so the concept survives with a different exemplar | S:65 R:80 A:85 D:75 |
| 12 | Confident | The two non-existent drift-guard test names (`TestMirrorDocsMatchDefaultProfiles`, `TestCLIFabReferenceListsDefaultRoles`) are corrected in passing where they sit in blocks this change already edits (agent_test.go:59-60/98, stage-models.md § Drift guard, providers-and-profiles.md:225) | Pre-existing drift, but the sentences sit directly above the pin table being edited and in the spec section being edited; leaving a false test name next to a fresh edit would fail review's documentation_accuracy check | S:30 R:90 A:75 D:65 |
| 13 | Confident | The flat-alias consequence (`providers.codex.model` now reaches only `default`) is documented as an upgrade note in stage-models.md, not a migration | Nothing restructures and the flat pair is already deprecated (alias for `profiles.default`); same precedent as the hydrate-split upgrade note | S:45 R:85 A:80 D:75 |
| 14 | Confident | Add one structural test assertion that every codex role fill names a model (dense-by-policy), beside the existing claude check in `defaults_test.go`; agy exempt (model-only sparse), kimi still asserted empty | Brief frames "no silent inheritance" as policy, not a snapshot; test-alongside strategy; the pin table alone would let a future effort-only row through if both sides were edited together | S:45 R:90 A:65 D:60 |
| 15 | Confident | `_cli-agents.md`'s codex Model-discovery recipe gains `~/.codex/models_cache.json` as the verified catalog source | This is what the ywkx and wcib refreshes actually read; a recipe (where to look), not a catalog (what is there), so it stays within the dictionary's no-model-IDs rule | S:25 R:90 A:40 D:40 |
| 16 | Confident | `_cli-fab.md`'s illustrative `fab resolve-agent apply` outputs under codex change `effort=xhigh` → `high` | Behavior claim in a skill file; apply → `doing` now resolves `high`; the "values illustrative" caveat stays | S:50 R:90 A:85 D:80 |
| 17 | Confident | Ship astra on `default`/`doing` without per-model cost data; record the tier ladder as inferred from codex's descriptions and priority order | User decided with this limitation known; pricing not retrievable offline; recorded in § Why 5 and in the spec/memory rationale as a limitation | S:70 R:80 A:30 D:50 |
| 18 | Certain | Resolution verification runs the locally built binary, not the `fab` shim | The shim routes to the binary matching the project's kit pin (2.23.14 here), not the working tree, so it cannot observe the edited defaults | S:60 R:95 A:90 D:90 |

18 assumptions (10 certain, 8 confident, 0 tentative, 0 unresolved).
