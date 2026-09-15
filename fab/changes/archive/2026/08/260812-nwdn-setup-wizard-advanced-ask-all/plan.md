# Plan: Setup Wizard Advanced Section Asks All Keys When Opted In

**Change**: 260812-nwdn-setup-wizard-advanced-ask-all
**Intake**: `intake.md`

## Requirements

### Wizard: Advanced Section Ask-All

#### R1: Opting in asks every advanced question
`askAdvanced` (`src/go/fab/cmd/fab/setup_wizard.go`) SHALL ask all four advanced questions (`agent.profiles.operator.provider`, `agent.profiles.review.provider`, `dispatch.column_width`, `dispatch.reap_done`) whenever the user answers yes at Q4, regardless of each key's winning tier. The per-question `tierDefault` skip loop, the `skipped`/`asked` bookkeeping, and the all-skipped note (`"No advanced overrides in effect — skipping %s. See: fab config explain <key>.\n\n"`) SHALL be removed outright. The `askAdvanced` doc comment SHALL state the new contract (opting in asks every advanced question; Enter keeps the current effective value so an all-Enter pass writes nothing). Answering no (or Enter) at Q4 still skips the section entirely; `--defaults` accepts Q4's default **N** and never enters the section (unchanged).

- **GIVEN** a fresh machine (no advanced key overridden at any tier)
- **WHEN** the user answers `y` at Q4
- **THEN** all four advanced questions are asked, in the listed order
- **AND** an all-Enter pass through them records zero changes ("nothing to change", no file written)

#### R2: Sparse profile keys render a depth-correct inherit indication
The `agent.profiles` map is deliberately sparse — no profile key is stored by default — but the wizard's read model DERIVES a built-in-defaults row for each profile key from its depth knob (`readModelDefaults` composes against the live config), so `effectiveValue` reports the resolved provider at tier `tierDefault` rather than an empty value. A profile key whose winning tier is `tierDefault` (derived, never explicitly set — or genuinely empty) SHALL be presented as an explicit inherit indication over an **empty baseline**, naming the role's depth knob depth-correctly (`docs/specs/stage-models.md` role→depth partition): `agent.profiles.operator.provider` → `(inherit agent.session)`; `agent.profiles.review.provider` → `(inherit agent.workers)`. The indication SHALL appear in the `Default:` line (origin suppressed — the derived row's `default` origin would misread as a stored value), the `[current]` prompt bracket, and the diff-summary old side (e.g. `agent.profiles.operator.provider: (inherit agent.session) → codex`). Enter SHALL keep the inherit (recorded answer equals the empty baseline ⇒ no diff entry, no write). Typing a detected provider (name or option number) SHALL write the override through the existing surgical path — including when it equals the inherited depth value (an explicit pin is a real write, possible precisely because the baseline is empty rather than the derived provider). An explicitly overridden profile key (winning tier project/system/env) renders its stored value and origin as any other key. The two non-sparse keys (`dispatch.column_width` default `35`, `dispatch.reap_done` default `true`) need no rendering change.

- **GIVEN** `agent.profiles.operator.provider` unset at every tier
- **WHEN** its question renders
- **THEN** the default line and prompt bracket show `(inherit agent.session)`
- **AND** answering `codex` produces the diff line `agent.profiles.operator.provider: (inherit agent.session) → codex` and, on confirmation, a surgical write of exactly that key

#### R3: Provider options stay probe-filtered
The two profile questions SHALL keep offering the same `providerOptions()` list as Q1/Q2 — detected providers only, capability-annotated. No change to `detectedProviders`/`providerOptions` ("capability is detected, never asked" preserved).

- **GIVEN** only `claude` is on PATH
- **WHEN** the profile questions render
- **THEN** their option list is exactly the detected roster

#### R4: Tests conform to the new contract
`src/go/fab/cmd/fab/setup_test.go` SHALL be updated: `TestSetupWizard_AdvancedAllSkippedPrintsNote` and `TestSetupWizard_AdvancedOverriddenKeyIsAsked` rewritten to the ask-all contract, plus new coverage for the first-time profile write (R2 scenario). Stdin scripts account for four more consumed answer lines on opted-in runs.

- **GIVEN** the rewritten suite
- **WHEN** `go test ./cmd/fab/ -run TestSetupWizard` then the package run
- **THEN** all tests pass with no assertion of the removed skip behavior remaining

#### R5: Documentation states the new behavior (sibling sweep)
All prose restating the old skip rule SHALL be updated in the same change: `src/kit/skills/_cli-fab.md` (bare-`fab setup` wizard paragraph's "keys sitting at the built-in default and never overridden are skipped, with an all-skipped note naming them" clause → ask-all + inherit rendering) and `docs/memory/distribution/setup.md` (the **Advanced section** bullet, present-truth style; the "Advanced Section Reviews Existing Customizations (Skip Rule)" Design Decision superseded by the inverted decision). A repo-wide grep of "No advanced overrides", "at-default and never overridden", "reviews existing customizations", "all-skipped" SHALL find no remaining stale claims — excluding `fab/backlog.md`'s historical [stpw] entry (intent record, left untouched) and gitignored deployed copies (`.claude/skills/`, `.agents/skills/`).

- **GIVEN** the sweep is complete
- **WHEN** the grep runs repo-wide
- **THEN** every hit is a known non-target (backlog intent record, deployed copies, this change's own artifacts)

### Non-Goals

- Clearing an existing override via the wizard (needs a `config unset` write path — `writeOne` only Sets) — deferred
- C3's section entry points (`fab setup agents|dispatch`) and per-provider fill editing — stay deferred
- Any change to `fab setup check`, Q1–Q3, the diff-before-write flow, or `--defaults` semantics

### Design Decisions

#### Advanced Section Asks All Keys When Opted In
**Decision**: Answering yes at Q4 asks every advanced question regardless of winning tier; the `tierDefault` skip and the all-skipped note are removed. Sparse profile keys render a depth-correct inherit indication (`(inherit agent.session)` / `(inherit agent.workers)`) in place of an empty current value.
**Why**: Q4's opt-in (default N) already carries the consent the skip rule guarded, and on exactly the machines whose owners want to set an advanced knob for the first time (nothing overridden yet), the old rule made the section a dead end — the Q4 prompt asked whether to *configure* the options, then declined to configure them. Enter-keeps-current preserves the all-Enter zero-write invariant (Constitution III).
**Rejected**: Keeping the skip rule with a per-key "set it now?" confirm — four extra prompts expressing what Q4's `y` already expressed. Supersedes 260811-stpw's "Advanced Section Reviews Existing Customizations (Skip Rule)".
*Introduced by*: 260812-nwdn-setup-wizard-advanced-ask-all

## Tasks

### Phase 2: Core Implementation

- [x] T001 Invert the Q4 skip rule in `askAdvanced` (`src/go/fab/cmd/fab/setup_wizard.go`): remove the `tierDefault` skip loop, `skipped`/`asked` bookkeeping, and the all-skipped note; update the doc comment to the ask-all contract <!-- R1 -->
- [x] T002 Add depth-correct inherit rendering for the sparse profile keys in `src/go/fab/cmd/fab/setup_wizard.go`: `(inherit agent.session)` for operator, `(inherit agent.workers)` for review, shown in the `Default:` line, the `[current]` prompt bracket, and the diff-summary old side; Enter keeps the inherit (no write), explicit provider answers write <!-- R2 -->

### Phase 3: Integration & Edge Cases

- [x] T003 Rewrite `TestSetupWizard_AdvancedAllSkippedPrintsNote` (→ opted-in asks all four, inherit indication renders, all-Enter writes nothing) and `TestSetupWizard_AdvancedOverriddenKeyIsAsked` (overridden key asked with its value AND the other three asked, no note) in `src/go/fab/cmd/fab/setup_test.go`; add first-time-profile-write coverage (diff line + surgical write of exactly `agent.profiles.operator.provider`); run `go test ./cmd/fab/ -run TestSetupWizard`, then the package <!-- R4 -->

### Phase 4: Polish

- [x] T004 Docs sweep: update the `src/kit/skills/_cli-fab.md` wizard-paragraph clause and `docs/memory/distribution/setup.md` (Advanced-section bullet + supersede the Skip Rule Design Decision); grep repo-wide for "No advanced overrides", "at-default and never overridden", "reviews existing customizations", "all-skipped" and fix every non-exempt hit <!-- R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: On a fresh fixture, `y` at Q4 asks all four advanced questions; the skip loop, bookkeeping, and all-skipped note are gone from `setup_wizard.go`
- [x] A-002 R2: Profile questions render `(inherit agent.session)` / `(inherit agent.workers)` depth-correctly in the default line and prompt bracket when unset
- [x] A-003 R3: Profile questions offer exactly the probe-filtered `providerOptions()` roster

### Behavioral Correctness

- [x] A-004 R1: An all-Enter run through the opted-in advanced section prints "nothing to change" and writes no file (Constitution III)
- [x] A-005 R2: Answering a detected provider at an unset profile question produces the inherit-old-side diff line and a surgical write of exactly that key

### Scenario Coverage

- [x] A-006 R4: Both rewritten tests plus the new first-time-write test pass; `go test ./cmd/fab/ -run TestSetupWizard` and the package run are green

### Edge Cases & Error Handling

- [x] A-007 R2: An explicit provider answer equal to the inherited depth value still writes the override (explicit pin)

### Code Quality

- [x] A-008 Pattern consistency: new rendering code follows the existing `ask`/`effectiveValue`/`diffAndWrite` seam style and comment density
- [x] A-009 No unnecessary duplication: inherit mapping defined once, not per call site
- [x] A-010 Canonical sources only: no edits under `.claude/skills/` or `.agents/skills/`; skill prose edits land in `src/kit/skills/`
- [x] A-011 Owner-or-pointer: updated skill prose states the new rule only where it is owned (`_cli-fab.md` wizard paragraph), no new restatements introduced

### Documentation Accuracy

- [x] A-012 R5: `_cli-fab.md` and `docs/memory/distribution/setup.md` state the ask-all contract; the repo-wide grep finds only exempt hits (backlog [stpw] entry, deployed copies)

## Notes

- Known pre-existing failure, NOT this change's: `TestPaneReady_JSON/ready` (and siblings) flake on zsh-default machines — tracked as backlog `[zshf]`, found during 260812-kgam verification. The failing snippet is the `zsh-newuser-install` menu; unrelated to the setup wizard. `go test ./cmd/fab/ -run TestSetupWizard` and the rest of the package/module are green.
- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change inverts the Q4 skip rule and adds the inherit rendering without making existing code redundant; the dead code it created (the `tierDefault` skip loop, `skipped`/`asked` bookkeeping, and the all-skipped note in `askAdvanced`) was removed by the change itself.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Implementation seam for inherit rendering: a display-only mapping (question-level placeholder + diff-summary old-side rendering) that never changes the recorded answer/current values, so `diffAndWrite`'s equality comparison stays untouched | Intake explicitly leaves the seam to apply; keeping the answer model verbatim and mapping only at render sites is the smallest change that satisfies the observable contract | S:70 R:85 A:85 D:75 |
| 2 | Confident | `docs/memory/distribution/setup.md` is edited at apply (T004) as part of the sibling sweep, with hydrate verifying and regenerating indexes | code-quality.md § Sibling Sweeps requires updating every occurrence in the class before finishing apply, and the memory file documenting the behavior is explicitly in the class | S:75 R:85 A:85 D:80 |
| 3 | Confident | The intake's "effectiveValue returns empty for profile keys" premise corrected at apply: the read model derives profile rows from the depth knobs (`readModelDefaults`), so "never set" is detected via the `tierDefault` winning tier and mapped to an empty baseline + inherit display (`inheritAs` question field); the intake's observable contract (inherit indication, Enter-no-write, explicit-pin writes) is preserved verbatim | Discovered running the new tests — the derived row rendered `claude (origin: default)`; keying on `tierDefault` is the one signal that distinguishes derived-inherit from an explicit override, and an empty baseline is what makes the intake's explicit-pin clause implementable | S:80 R:85 A:85 D:75 |

3 assumptions (0 certain, 3 confident, 0 tentative).
