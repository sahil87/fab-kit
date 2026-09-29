# Intake: Bump Claude Default Model Profiles to Sonnet 5.5

**Change**: 260929-9dwn-bump-claude-defaults-sonnet-5-5
**Created**: 2026-09-29

## Origin

Created by `/fab-proceed`'s promptless create-intake dispatch (`{questioning-mode} = promptless-defer`) from a description the team lead synthesized after the fab-kit owner reviewed the proposal in conversation. No questions were asked; the scope arrived already decided and is recorded as such below.

> Claude Sonnet 5.5 has been released. Bump fab-kit's built-in Claude agent default profiles for the two roles still pinned to Sonnet 5 — `operator` and `fast` — from `claude-sonnet-5` to `claude-sonnet-5-5`. This is the Sonnet counterpart to change 260923-vvdv (PR #684), which moved `default`/`doing`/`review`/`hydrate` from `claude-opus-5` to `claude-opus-5-5`. After this change all six Claude role fills sit on the 5.5 generation. The built-in defaults are the bottom tier of the config cascade and are what every fresh install gets. Leaving `operator` and `fast` on the superseded Sonnet 5 means the highest-volume coordinator role and the mechanical/ship role run on an older model than necessary, for no cost saving. Efforts do NOT change — `operator` and `fast` stay at `effort: medium`. No re-tiering: whether Sonnet 5.5 is now strong enough for `review` or `hydrate` to come DOWN from Opus is a separate change. This is a pure model-ID bump.

Interaction mode: one-shot dispatch. Decisions carried in from the conversation: the two target rows, unchanged efforts, no re-tiering, no spec prose rewrite, and the four-site edit inventory (§ What Changes). One item the brief flagged as unverified — Sonnet 5.5's built-in default reasoning effort — was **resolved during intake** rather than deferred; see § What Changes 5 and Assumption 7.

## Why

**The pain point.** `src/go/fab/defaults.yaml` is the single `go:embed`'d data file that decides which model every fab-dispatched Claude worker runs on when a project has not overridden a role. It is the bottom tier of the four-tier config cascade (environment > system > project > built-in defaults), so it is exactly what a fresh `fab` install gets with no config of its own. After 260923-vvdv four of its six Claude role fills moved to `claude-opus-5-5`; the remaining two — `operator` and `fast` — still pin `claude-sonnet-5`, whose successor `claude-sonnet-5-5` shipped in the same generation wave.

**The consequence of not doing it.** Nothing breaks: Sonnet 5 is still served, and Sonnet 5.5 is the same price ($2 input / $10 output per MTok), so unlike the Opus bump there is no overspend to stop. The cost is **capability left on the table and a split-generation default table**. `operator` is the highest-volume role in the kit — it drives the coordination loop that reads panes, classifies agent state, and routes commands — and `fast` carries `ship` and `review-pr`, where faithful PR-description comprehension is the whole job (the documented reason Haiku was excluded from the defaults). Both are roles where a same-price capability upgrade is free margin. Leaving them behind also drifts `defaults.yaml` — the spec's own "single file to edit when a new model ships" — one generation behind the CLI it drives, and leaves the six-row table straddling two model generations for no stated reason.

**Why this approach.** This is the event `defaults.yaml` was designed for. The repo already carries the full lockstep machinery: a pinned-profile Go test, a drift-guarded doc mirror, and an append-only digest catalog that fails the suite until the new rendering is recorded. Bumping the two values and updating those three guards in the same commit is the whole change; the config cascade carries every user override untouched. Two adjacent options were considered and are deliberately **not** taken here:

- **Re-tier `review` / `hydrate` down from Opus to Sonnet 5.5** — rejected as a separate change. It is a cost/quality judgment about the critic and the memory writer, not a model-ID bump, and it would change what the pipeline's quality-bearing stages run on. Raised in conversation and explicitly deferred.
- **Change the efforts** — rejected. `operator` and `fast` stay at `medium`. See § What Changes 5: `medium` is also what the Sonnet 5.5 migration guidance recommends as the starting point for agentic and multi-step tool-use work, so the shipped value is already the recommended one.

## What Changes

### 1. `src/go/fab/defaults.yaml` — the two Sonnet fills

Under `providers.claude.profiles` (lines 122–128), the two `claude-sonnet-5` rows become `claude-sonnet-5-5`. Every `effort` value is unchanged on every row. The four Opus rows are untouched.

| Role | Current | New |
|------|---------|-----|
| `default` | `claude-opus-5-5` / `high` | unchanged |
| `operator` | `claude-sonnet-5` / `medium` | `claude-sonnet-5-5` / `medium` |
| `doing` | `claude-opus-5-5` / `high` | unchanged |
| `review` | `claude-opus-5-5` / `high` | unchanged |
| `hydrate` | `claude-opus-5-5` / `high` | unchanged |
| `fast` | `claude-sonnet-5` / `medium` | `claude-sonnet-5-5` / `medium` |

Resulting block (the file's existing column alignment is preserved — the longer model ID does not change the `{ model: …` column start, which is set by the longest role key `operator:`):

```yaml
    profiles:
      default:  { model: claude-opus-5-5, effort: high }
      operator: { model: claude-sonnet-5-5, effort: medium }
      doing:    { model: claude-opus-5-5, effort: high }
      review:   { model: claude-opus-5-5, effort: high }
      hydrate:  { model: claude-opus-5-5, effort: high }
      fast:     { model: claude-sonnet-5-5, effort: medium }
```

No other key in `defaults.yaml` changes. The `codex`, `kimi`, and `agy` provider blocks are out of scope, as are both `claude` command templates (they already substitute `{model}` and `{effort}`).

### 2. `src/go/fab/internal/agent/agent_test.go` — the deliberate-change pin

`TestDefaultRoleProfilesArePinned` (lines 63–83) is the one place in Go that asserts the shipped default values as literals. Its own comment states the contract: "When you bump a default: edit defaults.yaml, then update this table to match." Two entries change:

```go
RoleOperator: {Provider: "claude", Model: "claude-sonnet-5-5", Effort: "medium"},
RoleFast:     {Provider: "claude", Model: "claude-sonnet-5-5", Effort: "medium"},
```

`RoleDefault` / `RoleDoing` / `RoleReview` / `RoleHydrate` stay on `claude-opus-5-5` / `high`.

**Every other `claude-sonnet-5` literal in `src/go` is an arbitrary fixture or override value, not a claim about the shipped default, and stays verbatim.** Grep-verified in this worktree:

- `internal/agent/agent_test.go:311, 317` — an explicit cross-role-fallback fixture (`"doing"` profile supplied by the test).
- `internal/agent/agent_test.go:796–797` — an `Overrides{Model: …}` invocation-override case.
- `internal/agent/agent_test.go:891, 937` — an explicit `agent.profiles.fast` config fixture.
- `internal/agent/agent_test.go:1295` — the `ModelAlias` prefix table (see § 5 below).
- `internal/config/config_test.go:363, 377–378` — an explicit `doing:` profile parsed from a YAML fixture.
- `cmd/fab/agent_surface_test.go:123–140` — explicit `--model claude-sonnet-5` flag overrides.
- `cmd/fab/resolve_agent_test.go:320–326, 497` — an explicit `fast:` config fixture and a codex-provider dispatch fixture.

Apply MUST re-grep before finishing rather than trusting this list.

### 3. `docs/specs/stage-models.md` § Default role profiles — the drift-guarded mirror

The table under `### Default role profiles` (lines 126–134) is the verified mirror of what `defaults.yaml` composes; `TestDocTablesMatchAgentMaps` parses it under that heading and fails on any disagreement. Two cells change:

| Role | Provider | Model | Effort |
|------|----------|-------|--------|
| `operator` | `claude` | `claude-sonnet-5-5` | `medium` |
| `fast` | `claude` | `claude-sonnet-5-5` | `medium` |

The table and the YAML MUST land in the same commit.

**No prose rewrite.** The "Why these defaults" paragraph that follows names `operator` as "Sonnet/`medium`" and `fast` as "the mechanical floor on Sonnet/`medium`" — by **family**, with no version digits — so every sentence in it stays true verbatim. The one version-bearing sentence in that paragraph is about the **Opus** rows ("Opus 5.5's built-in default effort is `medium`, and fab passes `--effort` on every CLI arm"), which is unaffected by this change. Constitution VI: this is a human-curated spec table being kept truthful, not generated content.

### 4. `src/go/fab/internal/configupgrade/configupgrade.go` — append one digest

`knownGeneratedSystemParagraphDigests` (line 119) is the **append-only, byte-exact** identity catalog for every historical rendering of a generated config-fence paragraph. The `providers:` advert paragraph is rendered live from the embedded defaults, so changing the claude fills changes that paragraph's bytes and therefore its digest. `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` fails until the new identity is appended, and its failure message prints the missing digest verbatim — apply reads the value from the test output rather than computing it by hand.

Append a NEW entry at the end of the map, following the existing commenting style (a dated/change-tagged comment naming what moved, then the digest). The existing entry immediately above it — the vvdv `claude-opus-5-5` rendering at lines 176–177 — and **every** older entry MUST be retained; the catalog's whole contract is that a missed historical identity causes R10a to treat a generated paragraph as user content:

```go
	// Current providers advert: the claude operator/fast fills moved to
	// claude-sonnet-5-5 (260929-9dwn).
	"<digest printed by the failing test>": {},
```

Apply MUST also reword the **preceding** comment so the catalog stays readable: the vvdv entry is currently labelled "Current providers advert", and it is no longer current once this entry lands. Change that label to name its era instead (e.g. "Providers advert with the claude Opus rows on claude-opus-5-5 and Sonnet rows still on claude-sonnet-5 (260923-vvdv)"), exactly as the earlier `260908-wcib through v2.28.3` and `column_width 40 / min_cols 60 (PR #654)` entries were re-labelled when they were superseded. This is comment text only — no digest value is altered or removed.

### 5. Sonnet 5.5 effort: the brief's open item, resolved — not deferred

The dispatch brief flagged one unverified fact: Sonnet 5.5's built-in default reasoning effort, which matters only for the YAML comment at `src/go/fab/defaults.yaml:118–120`. It was **verified during intake** against the bundled Claude API reference, and the answer is the opposite of the branch the brief guessed at:

> **Sonnet 5.5's effort default is `high`, not `medium`.** The effort default is `high` on every current model *except* Claude Opus 5.5, whose default is `medium`. Sonnet 5 also defaulted to `high`. Additionally, **Sonnet 5.5's effort levels are recalibrated relative to Sonnet 5**, with `medium` named as the recommended starting point for agentic coding and multi-step tool use, and `low` for chat.

Three consequences:

1. **The functional bump is correct and unchanged.** The explicit `effort: medium` on the two Sonnet rows is a deliberate step *down* from a `high` default — it was load-bearing before this change (against Sonnet 5's `high`) and remains load-bearing after it (against Sonnet 5.5's `high`). Nothing about the effort column's behavior changes, and fab passes `--effort` on every CLI arm regardless.
2. **`medium` is now the documented recommendation for these two roles' workloads**, which is a fact that strengthens the existing spec rationale rather than contradicting it. No spec edit is required.
3. **The YAML comment at lines 118–120 is currently Opus-only and now reads as incomplete.** It explains why `effort: high` must not be dropped from the Opus rows, and says nothing about the Sonnet rows — whose explicit `medium` is load-bearing in the *opposite direction*. Apply SHOULD widen it by one clause so the effort column's two directions are both visible to the next person who touches the file. Suggested shape (wording is apply's to settle):

   ```yaml
    # The explicit effort is load-bearing in BOTH directions: Opus 5.5 defaults
    # to `medium`, so dropping `effort: high` would step every Opus dispatch
    # down; Sonnet 5.5 defaults to `high`, so dropping `effort: medium` would
    # step the Sonnet rows UP (both command templates above pass --effort).
   ```

   **Sibling-sweep obligation if apply takes this.** The "explicit effort is load-bearing" claim has exactly three homes, and `fab/project/code-quality.md` § Sibling Sweeps makes updating one of a class without the others a must-fix. The three are: this YAML comment (`defaults.yaml:118–120`), the spec sentence (`docs/specs/stage-models.md:152–154`), and the memory Design Decisions **Why** (`docs/memory/runtime/providers-and-profiles.md:633`). If the comment is widened, the spec sentence gains the matching Sonnet clause in the same commit and the memory DD is updated at hydrate. If apply instead leaves the comment Opus-scoped, all three stay as they are — the existing claim is still true, just partial. **Do not widen one and not the others.**

### 6. `ModelAlias` — no change required, and the optional test case

`agent.ModelAlias` matches on the family prefix `claude-sonnet-` (`src/go/fab/internal/agent/agent.go:399–404`), so `claude-sonnet-5-5` already resolves to the native Agent-tool alias `sonnet` with no Go logic change. The `TestModelAlias` case map (`agent_test.go:1295`) already carries `"claude-sonnet-5": "sonnet"` plus a dated-variant case (`claude-haiku-4-5-20251001`) that exercises exactly the longer-suffix prefix behavior this bump relies on.

Adding a `"claude-sonnet-5-5": "sonnet"` case is therefore **optional coverage, not a correctness requirement**, and it would be redundant with two cases already present. Default: **do not add it.** Apply MAY add it if it judges the explicitness worth one line; either choice passes.

### 7. Verified non-sites

- **`claude-sonnet-5` outside `src/go`**: `grep` over the tracked tree finds it only in `docs/specs/stage-models.md` (§ 3 above), the two memory files in § Affected Memory, prior change folders and `fab/changes/archive/**` (history — never edited), and `fab/plans/sahil/26-07-08-config-upgrade.md` (a historical plan document). **No `src/kit/**` file mentions `claude-sonnet-5`** — unlike the Opus bump, which had two `_cli-fab.md` example lines to update. Nothing under `src/kit/` changes, so no `fab sync` / deployed-copy concern arises and the Constitution V citation rule is not engaged.
- **The `# >>> fab reference` fence** in `fab/project/config.yaml` and the rendered config reference derive their fill lines live from `ResolveProvider` (`configref.go`), so they carry no hand-edited model string and update themselves on the next `fab config upgrade`. `TestConfigReferenceDocumentsProviderFill` (cmd/fab) derives its expectations the same way and needs no edit.
- **No migration file.** Config `presence=intent`: a user who copied the old rows above their fence pinned `claude-sonnet-5` on purpose, and that pin must keep working. This matches the vvdv precedent.
- **No Go logic change and no CLI surface change**, so the Constitution's CLI⇒docs constraint (`_cli-fab*.md` updates) is not triggered. The Go change still ships tests — the pin and the digest catalog ARE the test surface (Constitution VII).
- **Sonnet 5.5's API-level breaking changes do not reach fab.** The Sonnet 5 → 5.5 breaking changes are all Messages-API request-shape concerns (`thinking: {type: "disabled"}` now 400s, forced `tool_choice: any|tool` now 400s, preserved-thinking binding, the `computer_toolset_20260801` requirement, advisor-pairing restrictions). fab never calls the Messages API — it composes CLI command strings (`claude --model {model} --effort {effort}`) and hands them to `fab dispatch` or the operator launcher. None of those parameters exist in fab's surface, so the bump carries no API-compatibility work.
- **The installed Claude Code CLI accepts the ID.** The 2.1.284 binary contains the string `claude-sonnet-5-5` (24 occurrences), so `--model claude-sonnet-5-5` is valid on both the pane and headless arms. Live serving is not exercised at intake; the first dispatch after apply is the smoke test and a one-line revert covers failure.

### Non-goals

- **No re-tiering.** Whether Sonnet 5.5 is strong enough for `review` or `hydrate` to move down from Opus is deliberately deferred to a separate change. This change moves no role between model families.
- **No effort changes.** `operator` and `fast` stay at `medium`; the Opus rows stay at `high`.
- **No prose rewrite of `docs/specs/stage-models.md`'s "Why these defaults"** beyond the conditional one-clause sibling sweep in § 5, which only fires if apply widens the YAML comment.
- **No change to codex, kimi, or agy fills or command templates.**
- **No wholesale sweep of `claude-sonnet-5` in the tree.** The remaining occurrences are arbitrary test fixtures, historical plan/change documents, and illustrative "write model IDs versioned" examples — none is a default-fill claim.
- **No migration file, no Go logic change, no new CLI surface.**

## Affected Memory

- `runtime/providers-and-profiles`: (modify) bump the present-truth "versioned ID" example at ~line 211 from `claude-sonnet-5` to `claude-sonnet-5-5`; add a Design Decisions entry (four-field shape) recording the Sonnet 5.5 bump — all six Claude fills now on the 5.5 generation, efforts unchanged, re-tiering explicitly deferred, same-price upgrade. If apply widened the `defaults.yaml` effort comment (§ What Changes 5), also extend the existing "explicit effort is load-bearing" **Why** at ~line 633 with the Sonnet direction.
- `_shared/configuration`: (modify) the "Documented style: reach for a knob first" paragraph at ~line 232 cites `(claude-sonnet-5, claude-opus-5-5)` as the versioned-ID example; bump the Sonnet half to `claude-sonnet-5-5` so both halves show the shipped generation. The point being made (versioned IDs vs. bare family IDs, which miss `ModelAlias`'s trailing-hyphen prefix match) is unchanged.

## Impact

- **Code**: `src/go/fab/defaults.yaml` (2 value edits, plus an optional comment widening), `src/go/fab/internal/agent/agent_test.go` (2 literal edits, plus an optional alias case), `src/go/fab/internal/configupgrade/configupgrade.go` (1 appended digest + 1 re-labelled preceding comment). All Go files touched MUST be `gofmt`-clean — the `ci-gate` check runs gofmt, and a prior change in this repo failed CI on exactly that.
- **Specs**: `docs/specs/stage-models.md` (2 table cells; plus 1 conditional sentence only if § 5's sweep fires).
- **Kit skills**: none. No `src/kit/**` file mentions `claude-sonnet-5`.
- **Memory**: two files per § Affected Memory, plus hydrate's log entries.
- **Tests to run** (scoped first per `fab/project/code-quality.md` § Test Strategy, widening only if something surprises): from `src/go/fab`, `go test ./internal/agent/... ./internal/configupgrade/... ./cmd/fab/...`. The three tests that observe this change are `TestDefaultRoleProfilesArePinned`, `TestDocTablesMatchAgentMaps`, and `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer`. Expect the digest test to fail on the first run **by design** — its failure message supplies the value to append.
- **Ordering note**: the digest in § What Changes 4 depends on the final bytes of `defaults.yaml`. Edit the YAML first, then run the catalog test to harvest the digest. If the effort comment (§ 5) is also being widened, do that **before** harvesting — the comment is inside the rendered advert paragraph, so widening it after appending would invalidate the digest just recorded and require a second harvest.
- **Runtime effect**: once a release carrying this ships and users upgrade, every un-overridden `operator` and `fast` dispatch moves to Sonnet 5.5 at `medium`. Because Sonnet 5.5's effort levels are **recalibrated**, `medium` is not behaviorally identical to `medium` on Sonnet 5 even though the config value is unchanged — `medium` is, however, the documented recommended starting point for exactly these workloads (agentic / multi-step tool use). Projects with per-role overrides see no change.
- **Cost**: neutral. Sonnet 5.5 is the same price as Sonnet 5 ($2 input / $10 output per MTok). This is a capability-currency bump, not a savings one — which is the one substantive way it differs from its Opus sibling 260923-vvdv.

## Open Questions

- None blocking. The one item the dispatch brief flagged as unverified (Sonnet 5.5's built-in default effort) was resolved during intake — see § What Changes 5 and Assumption 7. The single remaining judgment call, whether to widen the `defaults.yaml` effort comment and sweep its two sibling sites, is recorded as Assumption 8 with a default and is safe for apply to settle.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `operator` and `fast` move to `claude-sonnet-5-5`; every `effort` value on every row is unchanged | Discussed — the brief states the two target rows and that efforts do not change; `medium` stays on both | S:95 R:90 A:95 D:95 |
| 2 | Certain | The four Opus rows stay on `claude-opus-5-5` / `high` | Discussed — 260923-vvdv already moved them; this change touches only the Sonnet half | S:95 R:90 A:95 D:95 |
| 3 | Certain | No re-tiering: `review` and `hydrate` stay on Opus | Discussed — raised and explicitly deferred to a separate change as a cost/quality judgment, not a model-ID bump | S:90 R:70 A:90 D:90 |
| 4 | Certain | Four edit sites: `defaults.yaml`, the `TestDefaultRoleProfilesArePinned` table, the `stage-models.md` mirror table, and an appended `configupgrade` digest | Brief-supplied inventory, re-verified by inspection in this worktree (line numbers confirmed at each site) | S:90 R:85 A:95 D:90 |
| 5 | Certain | No `src/kit/**` edit; no migration; no Go logic or CLI-surface change | Grep-verified: zero `claude-sonnet-5` occurrences under `src/kit/`. `presence=intent` makes a user's copied pin deliberate (vvdv precedent); `ModelAlias` prefix-matches the new ID | S:85 R:90 A:95 D:90 |
| 6 | Certain | Change type is `chore`, set explicitly via `fab status set-change-type` | `change-types.md` puts "update configs / bump dependencies" under `chore`; keyword inference would misfire on the `docs`/`test` path mentions throughout this intake. Same treatment as vvdv | S:85 R:95 A:90 D:85 |
| 7 | Certain | Sonnet 5.5's built-in default effort is `high` (not `medium`), and its effort levels are recalibrated from Sonnet 5 with `medium` recommended for agentic / multi-step tool-use work | Verified at intake against the bundled Claude API reference, resolving the brief's one flagged unknown. Confirms the bump is correct as specified and that `medium` is the right value for both roles | S:85 R:90 A:85 D:90 |
| 8 | Confident | Widen the `defaults.yaml` effort comment by one clause to cover the Sonnet direction, and sweep its two sibling sites (`stage-models.md:152–154`, `runtime/providers-and-profiles.md:633`) in lockstep — or leave all three untouched. Apply picks one; it MUST NOT update a subset | Follows from Assumption 7: the comment is now visibly Opus-only while the Sonnet rows' `medium` is load-bearing in the opposite direction. Prose-only and trivially reversible, but the three-site class makes a partial update a must-fix under code-quality.md § Sibling Sweeps. The brief's "no spec prose rewrite" Non-Goal is preserved either way — the conditional clause is an addition, not a rewrite | S:70 R:95 A:75 D:55 |
| 9 | Certain | Edit `defaults.yaml` (including any comment widening) BEFORE harvesting the digest from the failing catalog test | The digest covers the rendered advert paragraph's bytes, which include the comment; harvesting first would require a second harvest. Mechanical ordering, self-correcting if missed (the test simply fails again) | S:75 R:95 A:85 D:80 |
| 10 | Certain | Do NOT add a `"claude-sonnet-5-5": "sonnet"` case to `TestModelAlias`; apply MAY add it | `modelAliasPrefixes` matches `claude-sonnet-`, and the existing `claude-sonnet-5` and dated-variant (`claude-haiku-4-5-20251001`) cases already cover the prefix behavior. Optional coverage, not a correctness requirement — the brief says so and inspection confirms it | S:80 R:95 A:85 D:70 |
| 11 | Certain | Test scope is `./internal/agent/... ./internal/configupgrade/... ./cmd/fab/...` from `src/go/fab`; the digest test is EXPECTED to fail on the first run and supplies the value to append | code-quality.md § Test Strategy says scope to affected packages first. The catalog is append-only and self-enforcing by design, so its first-run failure is the mechanism, not a defect | S:80 R:90 A:90 D:80 |
| 12 | Certain | Memory scope is the two present-truth "versioned ID" examples plus a Design Decisions entry; archive folders, prior change artifacts, `fab/plans/`, and log files stay verbatim | Present-truth vs. history distinction in the FKF memory style; logs are append-only. Mirrors the vvdv memory scope decision | S:75 R:90 A:80 D:75 |
| 13 | Certain | Re-label the superseded vvdv digest comment (currently "Current providers advert") when appending the new one | The catalog's existing entries show this exact pattern — `260908-wcib through v2.28.3` and `column_width 40 / min_cols 60 (PR #654)` were both re-labelled when superseded. Comment text only; no digest value is touched | S:70 R:95 A:85 D:80 |
| 14 | Certain | Sonnet 5.5's Messages-API breaking changes (disabled thinking, forced `tool_choice`, preserved thinking, computer toolset, advisor pairings) require no fab work | fab composes CLI command strings and never calls the Messages API; none of those parameters exists in its surface. Verified against both `claude` command templates in `defaults.yaml` | S:75 R:90 A:85 D:85 |

14 assumptions (13 certain, 1 confident, 0 tentative, 0 unresolved).
