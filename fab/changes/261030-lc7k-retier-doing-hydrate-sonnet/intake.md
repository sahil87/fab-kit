# Intake: Re-tier Claude `doing` and `hydrate` Fills to Sonnet 5.5

**Change**: 261030-lc7k-retier-doing-hydrate-sonnet
**Created**: 2026-10-30

## Origin

Created by a promptless create-intake dispatch (`{questioning-mode} = promptless-defer`) from a description the orchestrating agent synthesized from a conversation with the fab-kit owner. No questions were asked; every decision below arrived already made and is recorded as such.

> Change the kit's built-in Claude provider fills for the `doing` and `hydrate` roles from `claude-opus-5-5` / effort `high` to `claude-sonnet-5-5` / effort `high`. Kit-wide default (user explicitly chose "kit default for everyone" over a machine-local `fab config set --system` override). `default` and `review` stay `claude-opus-5-5` / `high`; `operator` and `fast` stay `claude-sonnet-5-5` / `medium`. Other providers (codex, agy, kimi) untouched. Effort `high` stays EXPLICIT on both rows even though Sonnet 5.5's built-in default is already `high` — house rule: fab never relies on a model's own default effort. Hydrate (memory writing from a finished diff) is well within Sonnet's range; the agent pushed back on `doing` (apply writes all the code; moving it to Sonnet leaves review as the only Opus step and may raise must-fix findings / rework cycles — an unmeasured guess). The user held on both after hearing the objection. Record this as a design decision with the tradeoff, not as an open question.

Interaction mode: conversational, then one-shot dispatch. Decisions carried in from the conversation: the two target rows and their values; the four untouched Claude rows; kit-wide default rather than a machine-local system-tier override; explicit `effort: high` kept on both rows; Claude-only scope; and the `doing` re-tier accepted **over** the agent's objection (the objection and the user's hold are both part of the record — see § Why and § Assumptions).

This is the re-tiering decision that 260929-9dwn explicitly deferred ("Re-tier `review` / `hydrate` down from Opus to Sonnet 5.5 — rejected as a separate change"). It moves `hydrate` (which 9dwn named) and `doing` (which 9dwn did not); `review` stays on Opus.

## Why

**The pain point.** `src/go/fab/defaults.yaml` is the single `go:embed`'d data file that decides which model every fab-dispatched Claude worker runs on when a project has not overridden a role — the bottom tier of the config cascade, i.e. what every fresh install gets. Today four of its six Claude fills pin `claude-opus-5-5`, including `doing` (the `apply` stage, plus the retired manual `review-pr` stage) and `hydrate` (the `hydrate` stage). Those two roles carry the pipeline's highest output volume — apply writes the whole diff, hydrate rewrites memory files — so they dominate per-change token spend. The owner judged that both are within Sonnet 5.5's range and wants the kit default for everyone to reflect that.

**The consequence of not doing it.** Every un-overridden project keeps paying Opus rates on its two highest-volume stages. The owner could get the cheaper behavior for themselves alone with a system-tier override, but explicitly rejected that: the point is the kit-wide default, so every project that adopted fab-kit without per-role overrides gets the cheaper split.

**Why this approach.** `defaults.yaml` is the designed seam for exactly this curated per-release judgment (the spec's § Model upgrade path: "The per-role fill table is fab's curated judgment per release, not a fixed effort-per-role-rank rule"). The cascade preserves every user override untouched, so a project that wants Opus apply back sets one `agent.profiles.doing` (or `providers.claude.profiles.doing`) row.

**The tradeoff, recorded as a design decision (not an open question).**

- **`hydrate` → Sonnet**: low-risk. Hydrate writes memory from a *finished, reviewed* diff — summarization and present-truth prose, well within Sonnet's range. Not contested.
- **`doing` → Sonnet**: contested, accepted by the owner. The agent objected: apply writes all the code, and moving it to Sonnet leaves `review` as the only Opus step in the pipeline. The spec's own § apply↔review coupling argues that a cheaper author under a sharper critic drives *more* rework cycles (capped at 3 per `fab/project/code-review.md`), and three extra review rounds can cost more than running apply on the stronger model once. Whether that actually happens with Sonnet 5.5 is **unmeasured** — it is a plausible guess, not data. The owner heard the objection and held. The change therefore ships the re-tier and rewrites the spec's coupling section so it states the accepted tradeoff honestly instead of arguing the opposite of what ships.

**Alternatives rejected in conversation.**

- **Machine-local override** (`fab config set --system …` putting the two rows in `~/.fab-kit/config.yaml`) — rejected by the owner: they want the kit default for everyone, not a personal setting.
- **Re-tier `hydrate` only, keep `doing` on Opus** — the agent's counter-proposal; rejected by the owner after hearing the rework-coupling argument.
- **Drop the explicit `effort: high` on the two Sonnet rows** because it now equals Sonnet 5.5's built-in default — rejected: house rule that fab never relies on a model's own default effort (the next Sonnet generation may default differently; see 260923-vvdv's "Explicit `effort` on Every Claude Fill Is Load-Bearing, in Both Directions").

## What Changes

### 1. `src/go/fab/defaults.yaml` — two fills + the explanatory comment

Under `providers.claude.profiles` (currently lines 124–130), `doing` and `hydrate` change model; effort stays `high` on both. All other rows are untouched.

| Role | Current | New |
|------|---------|-----|
| `default` | `claude-opus-5-5` / `high` | unchanged |
| `operator` | `claude-sonnet-5-5` / `medium` | unchanged |
| `doing` | `claude-opus-5-5` / `high` | **`claude-sonnet-5-5` / `high`** |
| `review` | `claude-opus-5-5` / `high` | unchanged |
| `hydrate` | `claude-opus-5-5` / `high` | **`claude-sonnet-5-5` / `high`** |
| `fast` | `claude-sonnet-5-5` / `medium` | unchanged |

Resulting block (keep the file's existing column alignment):

```yaml
    profiles:
      default:  { model: claude-opus-5-5, effort: high }
      operator: { model: claude-sonnet-5-5, effort: medium }
      doing:    { model: claude-sonnet-5-5, effort: high }
      review:   { model: claude-opus-5-5, effort: high }
      hydrate:  { model: claude-sonnet-5-5, effort: high }
      fast:     { model: claude-sonnet-5-5, effort: medium }
```

**The comment above `profiles:` (currently lines 117–123) MUST be updated** — it currently says:

```yaml
    # The six role fills. `default` doubles as this provider's cross-role
    # fallback: a role absent here resolves the `default` entry, then empty.
    # The explicit effort is load-bearing in BOTH directions, because fab never
    # relies on a model's own default: Opus 5.5 defaults to `medium`, so dropping
    # `effort: high` would silently step every Opus dispatch DOWN a level; Sonnet
    # 5.5 defaults to `high`, so dropping `effort: medium` would silently step the
    # Sonnet rows UP one (both command templates above pass --effort).
```

After this change "the Sonnet rows" are no longer all `effort: medium` (`doing`/`hydrate` are Sonnet at `high`), so the last sentence becomes false. Replace with wording that names the medium rows specifically and covers the high Sonnet rows. Suggested text (apply may tighten it, but must keep all three facts):

```yaml
    # The six role fills. `default` doubles as this provider's cross-role
    # fallback: a role absent here resolves the `default` entry, then empty.
    # Every effort is explicit because fab never relies on a model's own default
    # (both command templates above pass --effort): Opus 5.5 defaults to `medium`,
    # so dropping `effort: high` would silently step the Opus rows DOWN a level;
    # Sonnet 5.5 defaults to `high`, so dropping `effort: medium` would step
    # operator/fast UP one. doing/hydrate's `effort: high` equals Sonnet 5.5's
    # default today and stays explicit anyway — the next Sonnet may not match.
```

No other key in `defaults.yaml` changes. The codex, agy, and kimi blocks are out of scope.

### 2. `docs/specs/stage-models.md` — mirror table + the prose that now contradicts it

Constitution VI: specs are human-curated design intent. These edits record the owner's decision (made in conversation) — they are not tool-generated — and they are required because the current prose argues the opposite of what ships.

**2a. § Default role profiles table (lines 127–134)** — the drift-guarded mirror (`TestDocTablesMatchAgentMaps` in `src/go/fab/internal/agent/stagemodels_doc_test.go` parses it under `### Default role profiles`). The `doing` and `hydrate` Model cells become `claude-sonnet-5-5`; Effort cells stay `high`. Must land in the same commit as the YAML.

**2b. "Why these defaults." paragraph (lines 139–158)** — currently says "`doing` runs Opus at `high` — Opus is the strongest coding/agentic model … a strong author minimizes rework cycles", "`hydrate` runs Opus/`high` — knowledge work and memory writing are named Opus strengths", and the `default` sentence refers to "the same coding/agentic strength that earns `doing` its Opus". Rewrite so that:

- `doing` runs Sonnet/`high` — apply is the highest-output-volume stage; the kit default accepts the rework-coupling risk (see § apply↔review coupling) in exchange for Sonnet-rate apply; a project that wants an Opus author sets `agent.profiles.doing`.
- `hydrate` runs Sonnet/`high` — it writes memory from a finished, reviewed diff, which is summarization within Sonnet's range.
- `review` stays Opus/`high` — now the pipeline's only Opus stage; the critic carries the judgment the cheaper author may miss.
- `default` stays Opus/`high` — re-ground its justification on direct interactive work and intake scoring (it can no longer lean on "the strength that earns `doing` its Opus").
- The explicit-effort sentence is generalized the same way as the YAML comment: Opus 5.5 defaults to `medium` (explicit `high` holds the Opus rows), Sonnet 5.5 defaults to `high` (explicit `medium` holds `operator`/`fast`), and `doing`/`hydrate`'s `high` matches Sonnet 5.5's default today but stays explicit because fab never relies on a model's own default.

**2c. § apply↔review coupling: why apply is `doing`, not cheaper (line ~845)** — currently concludes "This is why `apply` stays in `doing` on a top-end author rather than dropping to a cheap model … the coupling argues louder", and says "`doing` and `review` are **separate roles that currently resolve to the same profile**". Both are false after this change. Rewrite so it:

- Keeps the coupling analysis itself (a cheaper author under a sharper critic can drive more rework cycles, capped at 3; "cheaper apply = cheaper pipeline" is not strictly true) — the mechanism is still real.
- States the shipped decision: the kit default now runs `doing` on Sonnet and `review` on Opus, accepting the coupling risk; the rework-rate effect is **unmeasured**, and a project that sees elevated rework sets `agent.profiles.doing` back to an Opus model.
- Replaces "currently resolve to the same profile" with the fact that the two roles now resolve to *different* models — the role separation is what makes that a one-row config change for projects that disagree. (Keep the point that author/critic separation is enforced by fresh context and adversarial framing, not model diversity.)
- The section heading may be retitled (e.g. "apply↔review coupling") since "why apply is `doing`, not cheaper" no longer describes the shipped default. If retitled, grep for links to the old anchor (`#applyreview-coupling-why-apply-is-doing-not-cheaper` or similar) across `docs/` and `src/kit/` and update them; the "Why these defaults" paragraph links to it as "§ apply↔review coupling".

**2d. § Model upgrade path (line ~866)** — "the frontier model wherever the named strengths are code review, agentic execution, memory writing, and direct interactive work, Sonnet at the mechanical floor and for the highest-volume coordinator" no longer matches. Update to: the frontier model for code review and direct interactive work; Sonnet for the high-volume author and memory writer (`doing`, `hydrate`), the mechanical floor, and the coordinator. The rest of that section (one-place bump, curated judgment, pinned roles opt out) stays.

No other spec sections change. The effort-support table (line ~656: Sonnet supports `low/medium/high/max`) already permits `high` on Sonnet.

### 3. `src/go/fab/internal/agent/agent_test.go` — `TestDefaultRoleProfilesArePinned`

Lines 65–70. `RoleDoing` and `RoleHydrate` become `{Provider: "claude", Model: "claude-sonnet-5-5", Effort: "high"}`; the other four rows stay. Verified in this worktree: no other `claude-opus-5-5` literal exists under `src/go` tests (the oracle/override fixtures in `cmd/fab/*_test.go` use other IDs), so this is the only test pin. Apply MUST re-grep `src/go` for `claude-opus-5-5` / `claude-sonnet-5-5` before finishing rather than trusting this.

### 4. `src/kit/skills/_cli-fab.md` — the `fab resolve-agent apply` examples

Verified: both example blocks call `fab resolve-agent apply`, and `apply` resolves through the `doing` role, so they illustrate a changed role and MUST be updated:

- Line 326: `model=claude-opus-5-5` → `model=claude-sonnet-5-5` (first block, `$ fab resolve-agent apply`).
- Line 331: `model=opus` → `model=sonnet` (`$ fab resolve-agent apply --alias` — the alias follows the model; the brief's site list did not name this line, it was found at intake).
- Line 374: `model=claude-opus-5-5` → `model=claude-sonnet-5-5` (`$ fab resolve-agent apply --effort medium`; `effort=medium` stays — it is the override being illustrated).
- The `--provider codex` example in between uses `<codex-model-id>` placeholders and is unchanged.

This is deployed kit content (deployed via `fab sync`; do not edit `.agents/`/`.claude/` copies). No command signature changes. It cites no fab-kit-only path.

### 5. `src/go/fab/internal/configupgrade/configupgrade.go` — append one digest (APPEND-ONLY)

`knownGeneratedSystemParagraphDigests` (line 119) is the append-only byte-exact catalog of every historical rendering of a generated `--system` fence paragraph. The `providers` advert renders the claude fills, so its digest changes when the fills change, and `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` (`configupgrade_test.go:1168`) fails until the new digest is appended.

- Re-label the existing last entry's comment (lines 179–182, digest `ab93e3cc8204e7179344b8497fbfa089a83c3943ce4c72b0821b5cadfc084d59`), currently "Current providers advert: the claude operator/fast fills moved to claude-sonnet-5-5, putting all six claude fills on the 5.5 generation (260929-9dwn).", to a historical label in the catalog's existing style, e.g. `// Providers advert with all six claude fills on the 5.5 generation — default/doing/review/hydrate on claude-opus-5-5, operator/fast on claude-sonnet-5-5 (260929-9dwn).` Do NOT remove or alter the digest itself.
- Append a new final entry: `// Current providers advert: the claude doing/hydrate fills moved to claude-sonnet-5-5 (effort high), leaving default/review on claude-opus-5-5 (261030-lc7k).` followed by `"<digest printed by the failing test>": {},`.
- **Ordering**: edit `defaults.yaml` (values AND comment) first, then run the catalog test to harvest the new digest — any later byte change to the rendered paragraph would invalidate it.

### 6. Memory (hydrate stage)

- `docs/memory/runtime/providers-and-profiles.md`:
  - Design Decision "The Kit Default Is the Cheapest Top-Tier Opus, Never Fable" (line ~625) states the fills pin `default`, `doing`, `review`, `hydrate` to `claude-opus-5-5` — update the present-truth Decision text to the new split (`default`/`review` on Opus; `doing`/`hydrate` on Sonnet) without rewriting its Why/Rejected history.
  - Design Decision "The Claude Fills Track One Model Generation Across All Six Roles" (line ~631) names "the four judgment-dense roles (`default`, `doing`, `review`, `hydrate`) on `claude-opus-5-5`" and lists "Re-tiering `review` or `hydrate` down to Sonnet" as Rejected (as a separate decision). Update its Decision text to the new family split; the generation-tracking point itself still holds (all six remain on 5.5).
  - Design Decision "Explicit `effort` on Every Claude Fill Is Load-Bearing, in Both Directions" (line ~637) — extend to note `doing`/`hydrate`'s `high` equals Sonnet 5.5's default and stays explicit by the same rule.
  - Add a new Design Decision (four-field shape: Decision / Why / Rejected / *Introduced by*) recording: `doing` and `hydrate` re-tiered to `claude-sonnet-5-5`/`high` kit-wide; Why = highest-volume stages, hydrate within Sonnet's range, owner chose kit-wide over personal; the accepted tradeoff = review is now the only Opus stage and the apply↔review coupling may raise rework (unmeasured); Rejected = machine-local system-tier override, hydrate-only re-tier, dropping the now-default-equal explicit `high`.
  - Any present-truth prose elsewhere in the file asserting `doing`/`hydrate` run Opus — hydrate re-greps.
- `docs/memory/_shared/configuration.md` line ~232 cites `claude-sonnet-5-5`, `claude-opus-5-5` only as examples of the versioned-ID style — both IDs remain shipped, so it likely stays; hydrate confirms.
- Logs (`docs/memory/**/log.md`) and `docs/findings/*` are history and stay verbatim; hydrate appends its own log entry.

### Non-goals

- No change to `default`, `review`, `operator`, or `fast` fills or efforts.
- No change to codex, agy, or kimi fills or any command template.
- No Go logic change; no new CLI surface.
- No migration file: built-in defaults are not user data, and config `presence=intent` means a user who copied old rows above their fence keeps them deliberately.
- No measurement harness for the rework-rate effect (a possible follow-up, not part of this change).

## Affected Memory

- `runtime/providers-and-profiles`: (modify) update the three existing Claude-fill Design Decisions' present-truth Decision text to the new Opus/Sonnet split; add a Design Decision for the `doing`/`hydrate` re-tier with the accepted apply↔review coupling tradeoff and rejected alternatives
- `_shared/configuration`: (modify) verify-only — the versioned-ID style examples (`claude-sonnet-5-5`, `claude-opus-5-5`) remain valid; touch only if a present-truth sentence asserts `doing`/`hydrate` run Opus

## Impact

- **Code**: `src/go/fab/defaults.yaml` (2 value edits + comment rewrite), `src/go/fab/internal/agent/agent_test.go` (2 literal edits), `src/go/fab/internal/configupgrade/configupgrade.go` (1 comment relabel + 1 appended digest). Go files MUST be gofmt-clean (`ci-gate` runs gofmt).
- **Specs**: `docs/specs/stage-models.md` — mirror table (2 cells), "Why these defaults" paragraph, § apply↔review coupling, § Model upgrade path sentence.
- **Kit skills**: `src/kit/skills/_cli-fab.md` (3 example lines: 326, 331, 374).
- **Memory**: per § Affected Memory, plus hydrate's log entries.
- **Tests to run** (scoped first per `fab/project/code-quality.md`): from `src/go/fab`, `go test ./internal/agent/... ./internal/configupgrade/... ./cmd/fab/...`. Observing tests: `TestDefaultRoleProfilesArePinned`, `TestDocTablesMatchAgentMaps`, `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` (expected to fail once and supply the digest), `TestConfigReferenceDocumentsProviderFill` (derives from `ResolveProvider`, needs no edit).
- **Runtime effect** (after a release carrying this and a user upgrade): un-overridden `apply` (and manual `review-pr`) dispatches run `claude-sonnet-5-5`/`high`; `hydrate` dispatches run `claude-sonnet-5-5`/`high`. On the native Agent-tool arm `fab agent apply -o yaml` emits `model_alias: sonnet`. `intake` (`default`), `review`, `ship` (`fast`), and the operator are unchanged. Projects with per-role overrides see no change.
- **Risk**: possible increase in review must-fix findings / rework cycles on apply (unmeasured; accepted by owner). Mitigation is a one-row per-project override.

## Open Questions

- None. Every decision arrived pre-made from the owner conversation, including the contested `doing` re-tier (recorded as an accepted tradeoff, not an open question).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `doing` and `hydrate` claude fills become `{ model: claude-sonnet-5-5, effort: high }` | Discussed — owner chose these exact values | S:95 R:90 A:95 D:95 |
| 2 | Certain | `default`/`review` stay `claude-opus-5-5`/`high`; `operator`/`fast` stay `claude-sonnet-5-5`/`medium`; codex/agy/kimi untouched | Discussed — explicit scope from the owner | S:95 R:90 A:95 D:95 |
| 3 | Certain | Kit-wide built-in default, not a machine-local `--system` override | Discussed — owner explicitly chose "kit default for everyone" | S:95 R:85 A:95 D:95 |
| 4 | Certain | `doing` re-tier ships despite the rework-coupling objection; recorded as an accepted tradeoff (review becomes the only Opus stage; rework effect unmeasured) | Discussed — agent objected, owner held after hearing it; brief says record as design decision, not open question | S:90 R:80 A:90 D:90 |
| 5 | Certain | Explicit `effort: high` stays on both Sonnet rows even though it equals Sonnet 5.5's built-in default | Discussed — house rule that fab never relies on a model's own default effort (vvdv Design Decision) | S:95 R:90 A:95 D:95 |
| 6 | Certain | The `defaults.yaml` comment above `profiles:` is rewritten so it no longer calls all Sonnet rows `effort: medium` | Brief requires the comment stay accurate; current last sentence becomes false | S:90 R:95 A:90 D:85 |
| 7 | Certain | `_cli-fab.md` lines 326, 331 (`model=opus`→`model=sonnet`), and 374 are updated | Verified at intake: all three examples run `fab resolve-agent apply`, which resolves the changed `doing` role | S:85 R:95 A:90 D:90 |
| 8 | Certain | Append a new current digest to `knownGeneratedSystemParagraphDigests` and relabel the `ab93e3cc…` comment as historical; never remove an entry | Catalog is append-only and self-enforcing via `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer`; matches the wcib/vvdv/9dwn relabel pattern | S:90 R:90 A:90 D:90 |
| 9 | Certain | Edit `defaults.yaml` (values and comment) BEFORE harvesting the digest from the failing catalog test | The digest covers the rendered advert's final bytes | S:85 R:95 A:90 D:90 |
| 10 | Certain | No migration file; no Go logic or CLI change | Built-in defaults are not user data; templates already substitute `{model}`/`{effort}`; `ModelAlias` prefix-maps `claude-sonnet-` → `sonnet` | S:85 R:90 A:90 D:90 |
| 11 | Confident | Spec prose beyond the mirror table is rewritten: "Why these defaults", § apply↔review coupling (incl. the "currently resolve to the same profile" claim), and § Model upgrade path's role-strength sentence | Verified at intake that all three now contradict the shipped table; Constitution VI — these edits record the owner's decision, not generated content | S:75 R:90 A:80 D:70 |
| 12 | Confident | The § apply↔review coupling section keeps the coupling mechanism, states the accepted risk and the per-project override, and may be retitled (with any inbound anchor links fixed) | The mechanism is still true; only the conclusion flipped. Exact wording is apply's call within those constraints | S:70 R:95 A:75 D:65 |
| 13 | Confident | `TestDefaultRoleProfilesArePinned` is the only test pin to change; no other `claude-opus-5-5` test literal exists under `src/go` | Grep-verified in this worktree; apply re-greps before finishing | S:75 R:90 A:85 D:75 |
| 14 | Confident | Memory scope: update present-truth Decision text of the three existing Claude-fill Design Decisions in `runtime/providers-and-profiles.md` and add one new Decision; `_shared/configuration.md` is verify-only | Present-truth vs. history per FKF style; the versioned-ID examples still name shipped IDs | S:70 R:90 A:80 D:70 |
| 15 | Certain | Change type is `chore` (config-default refresh), pinned explicitly if inference differs | Same shape as 260923-vvdv / 260929-9dwn | S:80 R:95 A:90 D:85 |

15 assumptions (11 certain, 4 confident, 0 tentative, 0 unresolved).
