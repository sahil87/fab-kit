# Plan: Templated Claude session_command

**Change**: 260703-gvxd-templated-claude-session-command
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md. The change is a form restructuring (change_type=refactor):
     make the claude provider's session_command a template with explicit {model}/{effort}
     placeholders instead of relying on spawn.WithProfile's implicit append mode, keeping
     resolved output byte-identical. -->

### Config & Constant: Templated claude session_command

#### R1: Built-in `DefaultSessionCommand` becomes a template
The built-in claude provider's session command constant `agent.DefaultSessionCommand` (`src/go/fab/internal/agent/agent.go`) SHALL carry explicit `{model}` / `{effort}` placeholders appended at the END, so that `spawn.WithProfile` resolves it via template substitution rather than append mode, yielding byte-identical resolved output.

- **GIVEN** the default config (no provider overrides), the `default` tier resolving to `{claude-fable-5, xhigh}`
- **WHEN** `fab agent default --print` composes the session command via `spawn.WithProfile(prov.SessionCommand, model, effort)`
- **THEN** the output is byte-identical to today: `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-fable-5 --effort xhigh`
- **AND** `fab agent operator --print` yields `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-sonnet-5 --effort medium`
- **AND** `spawn.DefaultSpawnCommand` (which aliases the constant) resolves identically through `WithProfile` as the no-`session_command` fallback

#### R2: Scaffold `session_command` becomes a template
The scaffold config `src/kit/scaffold/fab/project/config.yaml` `providers.claude.session_command` SHALL use the same templated form (single-quoted YAML string style preserved), and the surrounding comment block SHALL no longer present the live claude line as the plain/append-mode example.

- **GIVEN** a new project scaffolded by `fab init`
- **WHEN** the user reads `fab/project/config.yaml`
- **THEN** the claude `session_command` shows `{model}`/`{effort}` placeholders like every other command in the providers block
- **AND** the comment prose stays accurate: append mode still exists for plain commands, but the built-in default is no longer described as the append-mode example

#### R3: `fab config reference` output stays consistent with `fab init`
Because `configref.go` injects `agent.DefaultSessionCommand` into the rendered reference (`SessionCommand: agent.DefaultSessionCommand`, template line `session_command: '{{ .SessionCommand }}'`), the reference output SHALL automatically reflect the templated form once R1 lands, keeping `fab config reference` and `fab init` output in agreement (no live drift). Surrounding configref prose describing the claude block SHALL be checked for wording assuming the plain form.

- **GIVEN** the constant is templated (R1)
- **WHEN** `fab config reference` renders the providers block
- **THEN** the claude `session_command` line shows the templated form, matching the scaffold

#### R4: `spawn.WithProfile` append mode is unchanged
Append mode SHALL remain byte-for-byte untouched — it is load-bearing for existing user configs (the 2.12.1→2.13.0 migration moved `agent.spawn_command` values verbatim into `providers.claude.session_command`, some carrying user-pinned `--model`/`--effort` flags relying on append-last/last-wins).

- **GIVEN** a user config whose `session_command` is a plain command with no placeholder
- **WHEN** `WithProfile` composes it with a resolved profile
- **THEN** it appends `--model`/`--effort` at the END (last-wins; empty ⇒ omit) exactly as before — no behavior change

#### R5: No migration ships
No migration file SHALL be added; shipped migrations (including the #468 `260702-fyn5` backfill writing the plain form) stay frozen historical artifacts. Existing plain-form configs keep working via append mode; the form drift is cosmetic since both forms resolve identically.

- **GIVEN** an existing project whose config carries the plain-form claude `session_command`
- **WHEN** the user upgrades to the version shipping this change
- **THEN** nothing rewrites their config; append mode continues to resolve their command identically

### Documentation: Sweep the mirror class

#### R6: Live docs/specs/skills/memory reflect the templated form; historical artifacts stay verbatim
Every live occurrence describing the scaffold/default going forward SHALL be updated to the templated form or reworded mechanism prose; historical/compat occurrences (shipped migrations, completed change folders, this repo's own config, append-mode compat documentation) SHALL be kept verbatim. Skill edits under `src/kit/skills/` SHALL carry their `docs/specs/skills/SPEC-*.md` mirror updates in the same change (Constitution Additional Constraints; code-quality.md § Sibling & Mirror Sweeps). Canonical sources under `src/kit/` only — never `.claude/skills/`.

- **GIVEN** the grep-classified occurrence list in intake §4
- **WHEN** apply sweeps the mirror class
- **THEN** each live occurrence uses the templated form / reworded prose, and each historical occurrence is left untouched
- **AND** no `.claude/skills/` deployed copy is edited

### Non-Goals

- Changing `spawn.WithProfile` logic (append or template resolution) — untouched (R4).
- Shipping a migration or rewriting existing user configs (R5).
- Updating this repo's own `fab/project/config.yaml:58` (plain form) — it is an existing user config covered by append mode (intake §3, Assumption 5).
- Adding new capability — template mode already shipped in `260702-6tmi` (PR #456); this only changes which form the default/scaffold uses.

### Design Decisions

1. **Placeholders at the END of the command**: template substitution then produces byte-identical resolved commands to today's append mode — *Why*: append mode appends `--model … --effort …` last; putting the placeholders last makes the substituted output match position-for-position — *Rejected*: placeholders mid-command (would change resolved byte order and break the acceptance criterion).
2. **Include the Go constant, not scaffold-only**: `fab config reference` renders from `agent.DefaultSessionCommand` — *Why*: a scaffold-only change would make `fab init` output contradict `fab config reference` output (live drift, not just doc drift) — *Rejected*: scaffold-only (explicitly rejected in intake §2 / Assumption 2).

## Tasks

### Phase 1: Core source change (byte-identical resolution)

- [x] T001 Update `agent.DefaultSessionCommand` in `src/go/fab/internal/agent/agent.go:49` to the templated form `` `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}` `` and update its doc comment (lines 45-48) to note the placeholders are substituted by `spawn.WithProfile`'s template mode. <!-- R1 -->
- [x] T002 Verify `spawn.DefaultSpawnCommand` (`src/go/fab/internal/spawn/spawn.go:13`) doc comment still reads correctly given the constant is now a template; adjust wording only if it asserted the plain/append form. <!-- R1 -->
- [x] T003 Check `src/go/fab/internal/configref/configref.go` surrounding prose/comments (~lines 148-190) for wording that assumes the plain claude form; reword the mechanism sentence so it no longer implies the live claude line demonstrates append mode. <!-- R3 -->

### Phase 2: Kit content (scaffold + skills, canonical `src/kit/` only)

- [x] T004 Update `src/kit/scaffold/fab/project/config.yaml:66` `session_command` to the templated form (single-quoted style) and reword the surrounding comment block (lines ~40-63) so the mechanism sentence stays true but the live claude line is no longer presented as the plain/append example. <!-- R2 -->
- [x] T005 [P] Update `src/kit/skills/_cli-fab.md` passages that present claude as the append-mode case: §fab config reference (~line 317), §fab operator (~line 828 composition-of-a-fully-defaulted-launch prose), §fab agent (~line 868), §fab batch (~lines 882-883) — mechanism prose ("appends for plain / substitutes for template") stays true, but the built-in default now takes the substitution path. <!-- R6 -->
- [x] T006 [P] Update `src/kit/skills/fab-operator.md` lines ~452 and ~702 which quote the plain default command verbatim → templated form (adjusting for the profile-resolved context each describes). <!-- R6 -->
- [x] T007 [P] Update `src/kit/skills/_preamble.md` WithProfile grammar-forgiving passage (~line 325): mechanism description stays true; update the example framing so the built-in default is no longer the plain/append example. <!-- R6 -->

### Phase 3: SPEC mirrors + other specs + memory

- [x] T008 [P] Update `docs/specs/skills/SPEC-fab-operator.md` lines 36 and 164 (quote the plain default fallback constant verbatim → templated form, since `spawn.DefaultSpawnCommand` now equals the templated constant). <!-- R6 -->
- [x] T009 [P] Sweep `docs/specs/skills/SPEC-_cli-fab.md` (line 39 mechanism summary "substituted/appended" stays true; check whole file for any plain-form assumption). <!-- R6 -->
- [x] T010 [P] Update `docs/specs/skills/SPEC-_preamble.md` (mirror of `_preamble.md`; align any WithProfile example framing with T007). <!-- R6 -->
- [x] T011 [P] Update `docs/specs/stage-models.md:149` (plain default in the `providers:` config snippet → templated form) and §Skill wiring append-mode prose (~lines 319-320: mechanism stays true, reword built-in-default framing). <!-- R6 --> <!-- rework cycle 2: review must-fix — the cycle-1 rescope is STILL wrong: "all-or-nothing on the empty-value drop" binds all-or-nothing to the drop, but resolveTemplate's drop is per-placeholder (independent modelNeedsDrop/effortNeedsDrop, spawn.go:135-136); all-or-nothing canonically means any-placeholder-disables-append. APPLY THE REVIEWER'S CLAUSE VERBATIM, replacing the clause "all-or-nothing on the empty-value drop (an empty value drops the placeholder's token and a preceding `-`-flag)" with: "all-or-nothing (any placeholder disables the append entirely); an empty value drops the placeholder's token and a preceding `-`-flag". Do NOT reword creatively beyond splicing this clause grammatically into the sentence. Acceptance A-018. -->
- [x] T012 [P] Update `docs/specs/architecture.md:233` (plain default in the `providers:` config snippet → templated form; note this snippet uses the unquoted YAML style). <!-- R6 -->
- [x] T013 [P] Update memory: `docs/memory/runtime/providers-and-tiers.md:27`, `docs/memory/_shared/configuration.md:59,65`, `docs/memory/runtime/operator.md:291`, `docs/memory/distribution/kit-architecture.md:122,327` — quoted constant/default values → templated form; re-check surrounding append-mode prose stays accurate. (Note: hydrate owns the authoritative memory update; apply corrects the literal quotes now to keep the sweep class consistent per code-quality.md.) <!-- R6 --> <!-- rework cycle 2: review should-fix — docs/memory/_shared/context-loading.md:129 was missed by the sweep: it retains the old append-first framing ("for a non-templated Claude session_command it appends…, and for a templated… it substitutes") — align it with the substitution-first framing (built-in default is now templated), matching operator.md:293. Optional nice-to-have: docs/memory/runtime/operator.md:3 frontmatter description still frames WithProfile append-first; fold in the substitution-first framing while touching the class. -->

### Phase 4: Tests + verification

- [x] T014 Run `go test ./internal/agent/... ./internal/spawn/... ./internal/configref/... ./internal/config/... ./internal/dispatch/... ./cmd/fab/...` and confirm green. Raw-constant assertions compare against the `DefaultSessionCommand`/`DefaultSpawnCommand` symbol (so they update automatically); resolved-output assertions for default tiers MUST pass unchanged (byte-identical resolution is the acceptance check — a broken resolved-output test is a real regression, not a fixture to bend, per Constitution VII). Fix only genuine raw-constant literal assertions if any exist. <!-- R1 --> <!-- rework: review should-fix + nice-to-have — src/go/fab/cmd/fab/agent_test.go:95,106: TestAgentPrintBuiltinFallback's doc comment ("profile appended") and error message ("want the default-tier profile appended") still describe the built-in default as append-path; reword to substitution. Also strengthen the test from two Contains-assertions to exact-equality on the full resolved command (pins the fallback path's byte-identity in-repo). Then re-run the affected-package tests. -->
- [x] T015 Acceptance verification via changed source (installed binary won't reflect Go changes): build/`go run` the changed source and confirm `fab agent default --print` and `fab agent operator --print` are byte-identical to the pre-change baseline. <!-- R1 -->

## Execution Order

- T001 is the linchpin (the constant); T002/T003 depend on reading it. T004-T013 are independent doc/kit edits ([P] within phase). T014/T015 run last (need T001).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `agent.DefaultSessionCommand` is the templated form with `{model} {effort}` at the END and its doc comment describes template-mode substitution.
- [x] A-002 R2: The scaffold `session_command` is the templated single-quoted form and its comment block no longer presents the live claude line as the plain/append example.
- [x] A-003 R3: `fab config reference` renders the templated claude `session_command`, matching `fab init` scaffold output (no live drift); configref prose no longer implies the plain form.
- [x] A-004 R4: `spawn.WithProfile` append mode is byte-for-byte unchanged (no logic edits to spawn.go).
- [x] A-005 R5: No migration file added; shipped migrations and completed change folders untouched.
- [x] A-006 R6: Every live docs/specs/skills/memory occurrence uses the templated form / reworded prose; every historical/compat occurrence is verbatim; SPEC mirrors updated for every edited skill. <!-- review note: SPEC-_cli-fab.md / SPEC-_preamble.md verified content-level in-sync with no edit needed (neither quotes the plain form nor the changed passages; SPEC-_cli-fab.md:39 "substituted/appended" stays true) -->

### Behavioral Correctness

- [x] A-007 R1: `fab agent default --print` → `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-fable-5 --effort xhigh` (byte-identical to pre-change baseline).
- [x] A-008 R1: `fab agent operator --print` → `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-sonnet-5 --effort medium` (byte-identical to pre-change baseline).
- [x] A-009 R4: A plain-form user `session_command` still resolves via append mode identically (covered by existing `TestWithProfile` append tests staying green).

### Scenario Coverage

- [x] A-010 R1: `go test` is green on all affected packages (`internal/agent`, `internal/spawn`, `internal/configref`, `internal/config`, `internal/dispatch`, `cmd/fab`). <!-- review note: internal/configref has no test files; remaining five packages pass with -count=1 -->

### Edge Cases & Error Handling

- [x] A-011 R1: The no-`session_command` fallback (`spawn.DefaultSpawnCommand`) resolves through `WithProfile` template mode to the same byte-identical output (covered by `spawn.Command` fallback tests + acceptance prints).

### Code Quality

- [x] A-012 Pattern consistency: New config/comment text follows the surrounding templated-command style (single-quoted YAML in scaffold; unquoted where architecture.md uses unquoted).
- [x] A-013 No unnecessary duplication: The constant is the single source; scaffold/reference/specs quote it rather than reintroduce a second literal.
- [x] A-014 Canonical source only: No edit under `.claude/skills/` (gitignored deployed copies); all kit edits under `src/kit/` (code-quality.md anti-pattern).
- [x] A-015 SPEC-mirror sync: Every edited `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in this change (whole mirror class swept). <!-- review note: mirror class swept; SPEC-fab-operator.md updated; SPEC-_cli-fab.md and SPEC-_preamble.md verified to contain nothing restating the changed passages, so no edit was required -->
- [x] A-016 CLI ⇒ docs: `_cli-fab.md` reflects the change (no signature change here, but the append/substitute prose is corrected).

### documentation_accuracy

- [x] A-017: No live doc/spec/memory retains the plain-form claude default as the going-forward canonical value; historical text is correctly preserved verbatim (grep-classified per intake §4).

### cross_references

- [x] A-018: Cross-references between the constant, scaffold, reference, specs, and memory remain internally consistent (all live copies agree on the templated form; append-mode prose remains true for plain user commands). <!-- review cycle 3: MET — the cycle-2 rework applied the prescribed clause verbatim at stage-models.md:321-323: "all-or-nothing (any placeholder disables the append entirely); an empty value drops the placeholder's token and a preceding `-`-flag". Verified against spawn.go ground truth (all-or-nothing = any-placeholder-disables-append, WithProfile doc lines 56-58 + isTemplate gate; empty-value drop is per-placeholder, independent modelNeedsDrop/effortNeedsDrop lines 135-136 — re-confirmed empirically: WithProfile(DefaultSpawnCommand, "", "xhigh") drops only the model tokens and substitutes effort) and consistent with all siblings (_preamble.md:325, _cli-fab.md:828, configuration.md:81, operator.md:293, SPEC-fab-operator.md:36). context-loading.md:129 substitution-first reframe and operator.md:3 frontmatter description also verified accurate (append mode still correctly described for plain-form configs). -->

## Notes

- Check items as you review: `- [x]`
- The installed `fab` binary (2.13.3) will NOT reflect the Go source change — acceptance byte-identity is verified against the changed source via `go run`, not the installed binary.
- Pre-change baseline (captured at apply entry via the installed binary against the repo's already-templated-equivalent resolution):
  - `fab agent default --print` → `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-fable-5 --effort xhigh`
  - `fab agent operator --print` → `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-sonnet-5 --effort medium`

## Deletion Candidates

- `src/go/fab/internal/spawn/spawn.go:78-88` (`WithProfile` append-mode branch) — no longer exercised by any built-in/default path (the templated default resolves via `resolveTemplate`); **deliberately retained per R4** as load-bearing compat for user plain-form configs carried forward by the 2.12.1→2.13.0 migration (this repo's own `fab/project/config.yaml:58` still exercises it) — deletable only after a future migration templatizes those configs, not in this change.

No other code, config, or tests became redundant: the change swaps one constant's value and sweeps documentation; all consumers (`spawn.Command`, `WithProfile`, `configref`, `fab agent`/`operator`/`batch`) remain live, and no test became dead (append-mode tests still cover the retained compat path).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Placeholders appended at the END of both the constant and scaffold, single-quoted YAML style kept in scaffold — byte-identical resolution to today's append mode | Intake Assumption 1 (Certain); verified empirically against baseline | S:90 R:85 A:95 D:95 |
| 2 | Certain | The Go constant is included (not scaffold-only) — `fab config reference` renders from it, so scaffold-only would create live drift | Intake Assumption 2 (Certain); configref.go:92 injects the constant | S:95 R:80 A:90 D:90 |
| 3 | Certain | `spawn.WithProfile` append mode stays untouched (load-bearing for existing user configs) | Intake Assumption 3 (Certain) | S:90 R:70 A:90 D:90 |
| 4 | Certain | No migration ships; shipped migrations + completed change folders stay frozen verbatim | Intake Assumption 4 (Certain) | S:90 R:75 A:85 D:85 |
| 5 | Confident | This repo's own `fab/project/config.yaml:58` (plain form) stays untouched | Intake Assumption 5 (Confident); existing user config covered by append mode | S:60 R:90 A:75 D:70 |
| 6 | Confident | Grep-sweep classification: live occurrences → templated/reworded; historical → verbatim | Intake Assumption 6 (Confident); per-occurrence judgment applied at apply | S:75 R:85 A:80 D:70 |
| 7 | Certain | `docs/memory/distribution/kit-architecture.md` (122, 327) is in the sweep class | Intake Assumption 7 (Certain) | S:80 R:90 A:90 D:85 |
| 8 | Confident | Apply corrects the memory literal quotes now (consistency with the sweep class), while hydrate remains the authoritative memory writer; this avoids leaving the sweep class half-done at apply | Constitution II (docs source of truth is hydrate's) balanced against code-quality.md § Sibling & Mirror Sweeps ("update every occurrence in the class" at apply); the literals are exact-value quotes, not new prose | S:70 R:80 A:75 D:70 |
| 9 | Confident | Go tests need no raw-constant literal edits — every assertion compares against the `DefaultSessionCommand`/`DefaultSpawnCommand` symbol or uses independent test-local fixtures | Verified by reading the 9 flagged test files; the literal `claude --dangerously-skip-permissions` in tests is a fixture without `-n`/placeholders, independent of the constant | S:75 R:85 A:85 D:80 |

9 assumptions (5 certain, 4 confident, 0 tentative).
