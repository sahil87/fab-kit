# Intake: Setup Wizard Advanced Section Asks All Keys When Opted In

**Change**: 260812-nwdn-setup-wizard-advanced-ask-all
**Created**: 2026-08-12

## Origin

Promptless dispatch via `/fab-proceed`, synthesized from the live 2026-08-12 discussion.

> `fab setup` wizard — make the Q4 advanced section actually ask its questions when opted in (invert the review-only skip rule), so a user can SET first-time advanced overrides (especially `agent.profiles.operator.provider`) through the wizard.

The user ran the shipped wizard (PR #592 / change 260811-stpw, released in 2.19.7), answered `y` at Q4 ("Configure advanced options (agent.profiles.operator/review, dispatch.column_width, dispatch.reap_done)?"), and got only:

```
No advanced overrides in effect — skipping agent.profiles.operator.provider, agent.profiles.review.provider, dispatch.column_width, dispatch.reap_done. See: fab config explain <key>.
```

— no questions asked. Root cause: `askAdvanced` in `src/go/fab/cmd/fab/setup_wizard.go` (lines 379–415) skips any advanced key whose winning tier is the built-in defaults (`tier == tierDefault`), so the section can only REVIEW existing overrides, never CREATE one. This was a deliberate intake decision in 260811-stpw (its Assumption 6, "read literally" from the backlog entry), with first-time setup consciously punted to `fab config set` and a deferred C3 ("per-provider fill editing, section entry points `fab setup agents|dispatch`"). The user now wants first-time advanced setup in the wizard; this change is narrower than C3 (no section entry points, no per-provider fill editing).

## Why

1. **The pain point**: the Q4 opt-in reads as "yes, walk me through the advanced knobs" — but on any machine that never customized them (i.e., exactly the machine whose owner wants to set them for the first time), answering `y` produces a skip note and nothing else. The one knob users most plausibly want to set interactively (`agent.profiles.operator.provider` — "coordinate from another CLI") is unreachable through the wizard.
2. **The consequence of not fixing**: first-time advanced configuration stays a `fab config set` incantation away; the wizard's advanced section is a dead end on fresh machines, and the Q4 prompt actively misleads (it asks whether to *configure* the options, then declines to configure them).
3. **Why this approach**: the old skip rule's worry (walking every knob on every run) is already handled by Q4 being opt-in with default **N** — only users who explicitly say `y` see the four questions. Enter-keeps-current still guarantees an all-Enter run writes nothing, preserving idempotency (Constitution III). The rejected alternative — keeping the skip rule but adding a per-key "set it now?" confirm — would be four extra prompts expressing what Q4's `y` already expressed.

## What Changes

### 1. Invert the Q4 skip rule (`askAdvanced`)

`src/go/fab/cmd/fab/setup_wizard.go` — `askAdvanced` (lines 379–415). Answering `y` at Q4 asks ALL FOUR advanced questions regardless of tier:

- `agent.profiles.operator.provider`
- `agent.profiles.review.provider`
- `dispatch.column_width`
- `dispatch.reap_done`

Remove the per-question `tierDefault` skip loop (`if _, _, tier := w.effectiveValue(q.key); tier == tierDefault { skipped = ...; continue }`) and the all-skipped note (`"No advanced overrides in effect — skipping %s. See: fab config explain <key>.\n\n"`) — with no skip, the note's condition is unreachable dead code. The `skipped`/`asked` bookkeeping goes with it. Update the `askAdvanced` doc comment (currently "The section REVIEWS existing customizations…") to state the new contract: opting in asks every advanced question; Enter keeps the current effective value so an all-Enter pass writes nothing.

Answering `n` (or Enter — the default) at Q4 still skips the section entirely; `--defaults` accepts Q4's default **N**, so `--defaults` runs never enter the advanced section (unchanged).

### 2. Inherit rendering for the sparse profile keys

`agent.profiles.operator/review.provider` has no built-in default — the `agent.profiles` map is deliberately sparse (see `src/go/fab/internal/config/config.go`), so `effectiveValue` returns `("", "", "")` and the prompt would render `Default:  (origin: )` and `agent.profiles.operator.provider []:`. Render the empty case as an explicit inherit indication naming the role's depth knob, **depth-correct per key** (per `docs/specs/stage-models.md` § role→depth partition — operator is a Tier-1/session role, review a Tier-2/workers role):

- `agent.profiles.operator.provider` → `(inherit agent.session)`
- `agent.profiles.review.provider` → `(inherit agent.workers)`

The indication appears wherever the empty current value would render — the `Default: … (origin: …)` line and the `[current]` prompt bracket. Enter keeps the inherit: the recorded answer equals the empty current, so `diffAndWrite` sees no change and writes nothing. Typing a detected provider name (or its option number) writes the override through the existing surgical write path (`configupgrade.SetSystem`/`Set` via `diffAndWrite`) — **including when the typed provider equals the inherited depth value** (an explicit pin is a real write; the wizard does not second-guess it). In the diff summary, render the old side of a first-time profile write with the inherit indication rather than an empty string (e.g. `agent.profiles.operator.provider: (inherit agent.session) → codex`).

The two numeric/bool advanced keys (`dispatch.column_width`, `dispatch.reap_done`) have real built-in defaults (`35`, `true`), so their prompts already render a concrete current value and need no rendering change — Enter keeps the built-in default and writes nothing.

The rendering seam (special-casing in `ask`, in `effectiveValue`, or a per-question display hook) is an apply-time implementation decision; the observable contract above is what binds.

### 3. Provider options stay probe-filtered

The two profile questions keep offering the same `providerOptions()` list as Q1/Q2 — detected providers only, annotated with capabilities. "Capability is detected, never asked" is preserved; no change to `detectedProviders`/`providerOptions`.

### 4. Tests (`src/go/fab/cmd/fab/setup_test.go`)

The two tests encoding the old rule must be rewritten:

- `TestSetupWizard_AdvancedAllSkippedPrintsNote` (~line 298) — currently asserts the "No advanced overrides in effect" note on a fresh machine and that at-default questions are NOT asked. Replace with the inverted contract: on a fresh machine, opting in (`y` at Q4) asks all four questions; the profile questions render the inherit indication; an all-Enter pass through the advanced section records zero changes ("nothing to change") and writes no file.
- `TestSetupWizard_AdvancedOverriddenKeyIsAsked` (~line 375) — currently asserts only the overridden key is asked while the other three skip. Rewrite: the overridden key is asked with its current value + origin as default AND the other three are asked too (at their defaults / inherit indication); no skip note ever prints.

New coverage: answering a detected provider for the unset `agent.profiles.operator.provider` produces the diff-summary line and the surgical write to the target tier (assert the written YAML carries `agent.profiles.operator.provider` and nothing else changed). Note the stdin scripts change shape — with all four advanced questions asked, an opted-in run consumes four more answer lines (e.g. `"\n\n\ny\n\n\n\n\n"` for all-Enter) plus the write confirmation where a change exists.

Run scope: `go test ./cmd/fab/ -run TestSetupWizard` first, then the package.

### 5. Docs sweep (up-front, sibling class — this repo's #1 rework cause)

The old skip rule is restated in three places that must all change:

- `src/kit/skills/_cli-fab.md` (~line 876) — the bare-`fab setup` wizard paragraph's clause "keys sitting at the built-in default and never overridden are skipped, with an all-skipped note naming them" → the new ask-all contract (opting in asks all four; sparse profile keys render an inherit indication).
- `docs/memory/distribution/setup.md` (~line 25) — the **Advanced section** bullet ("skipping any key whose winning tier is the built-in defaults (at-default and never overridden — the section reviews existing customizations). When every advanced question skips, a one-line note names the skipped keys…") → new behavior, present-truth style.
- `docs/memory/distribution/setup.md` (~line 233) — the Design Decision "**Advanced Section Reviews Existing Customizations (Skip Rule)**" — supersede/rewrite it to the new decision per present-truth style (the new entry records the inversion, why — Q4 opt-in already carries the consent the skip rule guarded, and the wizard must be able to CREATE a first-time override — and the rejected per-key-confirm alternative; *Introduced by* this change).

Survive as-is (verified — they only name the knob set, no skip-rule claim): `docs/specs/skills.md` (~line 248) and `src/kit/skills/fab-setup.md` (~line 130).

Grep the old claim repo-wide before finishing apply — "No advanced overrides", "at-default and never overridden", "reviews existing customizations", "all-skipped" — including `*_test.go` comments and contrastive phrases. Known non-target hits to leave alone: `fab/backlog.md` line ~33 (the historical [stpw] backlog entry recording that change's original design ask — a record, not a present-truth behavior claim) and the gitignored deployed copies (`.claude/skills/`, `.agents/skills/` — regenerated by `fab sync`, never edited; Constitution V / code-quality anti-pattern).

### Constraints that bind

- CLI behavior change ⇒ `_cli-fab.md` update + test updates ship in the same change (Constitution Additional Constraints).
- Edit canonical `src/kit/skills/` sources, never deployed copies.
- No migration needed — behavior-only, no user-data restructuring.
- `fab setup check` (the read-only doctor) is untouched.

## Affected Memory

- `distribution/setup`: (modify) Advanced-section bullet rewritten to the ask-all contract + inherit rendering; the "Advanced Section Reviews Existing Customizations (Skip Rule)" Design Decision superseded by the inverted decision.

## Impact

- `src/go/fab/cmd/fab/setup_wizard.go` — `askAdvanced` (skip loop + note removal, doc comment), empty-current inherit rendering for the two sparse profile keys (prompt `Default:` line, `[current]` bracket, diff-summary old side).
- `src/go/fab/cmd/fab/setup_test.go` — two tests rewritten, new first-time-write coverage.
- `src/kit/skills/_cli-fab.md` — wizard paragraph clause.
- `docs/memory/distribution/setup.md` — bullet + Design Decision (via hydrate, per Affected Memory).
- No config schema, CLI flag, or migration changes; `fab setup check`, Q1–Q3, the diff-before-write flow, and `--defaults` semantics unchanged. User-visible behavior change is wizard-interactive only ⇒ likely PATCH/MINOR release note territory, judged at ship.

## Open Questions

None — the originating discussion resolved every decision point (see Assumptions).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Invert the Q4 skip rule: opting in asks all four advanced questions regardless of winning tier; the `tierDefault` skip and all-skipped note are removed outright | Discussed — explicit user decision reversing 260811-stpw's Assumption 6; Q4's opt-in (default N) already carries the consent the skip rule guarded | S:95 R:85 A:90 D:95 |
| 2 | Certain | Idempotency holds via Enter-keeps-current: an all-Enter run (including through the opted-in advanced section) records zero changes and writes nothing | Discussed — Constitution III named explicitly; existing `ask`/`diffAndWrite` mechanics already guarantee it | S:90 R:90 A:95 D:90 |
| 3 | Confident | Empty current for the sparse profile keys renders a depth-correct inherit indication — `(inherit agent.session)` for operator, `(inherit agent.workers)` for review — in the `Default:` line and the `[current]` prompt bracket; Enter keeps the inherit (no write) | Discussed ("render as explicit inherit indication, e.g. `(inherit agent.workers)`"); the per-key depth mapping refines the example using `docs/specs/stage-models.md`'s role→depth partition (operator = Tier-1/session, review = Tier-2/workers) | S:65 R:85 A:80 D:70 |
| 4 | Certain | Provider options for the two profile questions stay probe-filtered (same `providerOptions()` as Q1/Q2) — capability is detected, never asked | Discussed — decision 3, preserves the wizard's core invariant | S:90 R:90 A:95 D:95 |
| 5 | Certain | Out of scope: clearing an existing override via the wizard (would need a `config unset` write path — `writeOne` only calls `configupgrade.Set`/`SetSystem` today); C3's section entry points and per-provider fill editing stay deferred | Discussed — decisions 4 and scope framing; keeps the change small | S:90 R:95 A:90 D:90 |
| 6 | Confident | Typing a provider at an inherit-state profile question writes an explicit override even when it equals the inherited depth value (answer ≠ empty current ⇒ diff + surgical write) | Follows from discussed decision 2 ("choosing a detected provider writes the override") plus existing `diffAndWrite` semantics; an explicit pin is a legitimate intent | S:65 R:80 A:80 D:70 |
| 7 | Confident | Diff-summary old side for a first-time profile write renders the inherit indication (e.g. `(inherit agent.session) → codex`), not an empty string | Not explicitly discussed; small readability call consistent with decision 2's rendering rule, trivially reversible | S:50 R:90 A:70 D:55 |
| 8 | Confident | Change type `feat` — the wizard gains a capability it never had (first-time advanced setting); the shipped skip rule was as-designed (260811-stpw), so this is not a regression fix per docs/specs/change-types.md | Taxonomy: `fix` = bug/regression; the old behavior matched its intake | S:70 R:90 A:80 D:70 |
| 9 | Certain | Docs sweep set as listed in What Changes §5: `_cli-fab.md` clause + `setup.md` bullet + superseded Design Decision, repo-wide grep of the old-claim phrases; `docs/specs/skills.md` and `src/kit/skills/fab-setup.md` survive (verified — they only name the knob set) | Discussed — sweep list given up front; survivors verified against the working tree during intake | S:85 R:80 A:90 D:85 |
| 10 | Confident | The historical [stpw] backlog entry (`fab/backlog.md` ~line 33) is left untouched by the sweep — it records that change's original design ask, not a present-truth behavior claim | Backlog entries are intent records; rewriting history would falsify what stpw was asked to build | S:55 R:85 A:75 D:60 |
| 11 | Certain | No migration ships — behavior-only change, no user-data restructuring (config schema, `.status.yaml`, archive layout all untouched) | context.md § Migrations trigger conditions plainly not met | S:90 R:90 A:95 D:95 |

11 assumptions (6 certain, 5 confident, 0 tentative, 0 unresolved).
