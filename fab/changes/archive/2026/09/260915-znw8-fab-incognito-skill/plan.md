# Plan: fab-incognito — Discussion-Priming Skill That Loads Process Knowledge, Not Project Memory

**Change**: 260915-znw8-fab-incognito-skill
**Intake**: `intake.md`

## Requirements

### Kit Skills: `fab-incognito` skill file

#### R1: New standalone skill `src/kit/skills/fab-incognito.md`
The kit SHALL ship a new user-invocable skill at canonical source `src/kit/skills/fab-incognito.md`, deployed by `fab sync` like every other skill. Its frontmatter MUST carry `name: fab-incognito`, a `description` (imperative summary + "Use when rethinking…" trigger clause + the read-blind-not-trace-free note), and `helpers: [_pipeline, _intake, _srad, _generation, _review]`. Its body MUST contain, in order: the `# /fab-incognito` heading, the standard `_preamble` opener blockquote, `## Purpose`, `## Arguments` (None), `## Context Loading`, `## Standing Session Rule`, `## Command Logging`, `## Behavior`, `## Orientation Summary`, `## Key Properties`.

- **GIVEN** a project with deployed skills under `.agents/skills/`
- **WHEN** the user invokes `/fab-incognito`
- **THEN** the agent loads `fab/project/config.yaml` + `constitution.md` (required) and `context.md`, `code-quality.md`, `code-review.md` (optional), the six process helpers (`_preamble` + the five declared), a skill catalog (frontmatter `name`/`description` of every `.agents/skills/*/SKILL.md`, helpers marked), and the kit version from `$(fab kit-path)/VERSION` (fallback `unknown`)
- **AND** it does NOT load `docs/memory/index.md`, `docs/specs/index.md`, any change artifact, and does NOT run preflight

#### R2: Context Loading override and lazy-load boundary
The skill's `## Context Loading` section MUST state the always-load override explicitly (it loads a reduced 5-file `fab/project/` set and skips both doc indexes, citing `_preamble.md` §1's "the skill file wins" derivation), MUST list the lazy loads (full skill bodies and the five `_cli-*` reference partials, opened only on demand) with the size rationale (~120 KB helpers loaded vs ~274 KB CLI partials and ~834 KB all deployed skills, against the ~36 KB always-load layer), and MUST resolve the active change via `fab resolve --folder --or-none` exactly as `/fab-discuss` does.

- **GIVEN** the discussion turns to a specific skill or command family
- **WHEN** the agent needs its detail
- **THEN** it opens exactly that `.agents/skills/<name>/SKILL.md` (or `_cli-*` partial) at that point, never up front

#### R3: Standing Session Rule
The skill MUST carry a `## Standing Session Rule` section stating, as a behavior obligation: for the remainder of the discussion the agent does NOT open any file under `docs/memory/` or `docs/specs/` (including both index files) and does NOT follow `_preamble.md` § Memory File Lookup; the single exception is a file the user names explicitly (open exactly that file, no index walk); claims about current behavior cite the implementing skill, not the memory that describes it; the rule binds the discussion only — a later user-invoked skill follows its own Context Loading section.

- **GIVEN** `/fab-incognito` has run and the user later asks how hydrate works
- **WHEN** the agent answers
- **THEN** it cites `fab-continue`'s Hydrate Behavior / `_generation`, and does not open `docs/memory/memory-docs/*`
- **GIVEN** the user then says "open docs/memory/pipeline/preflight.md"
- **WHEN** the agent complies
- **THEN** it reads exactly that file without opening `docs/memory/pipeline/index.md`

#### R4: Orientation Summary, logging, and Key Properties
The skill MUST log `fab log command "fab-incognito"` after loading (with one sentence naming the read-blind-not-trace-free contract), MUST end with the Orientation Summary block — project line, `Kit: {version}`, helpers loaded, `Skill catalog: {N} skills ({U} user-invocable, {H} helpers)`, project files loaded / not found, the verbatim line `Memory and specs: NOT loaded (incognito — will not be opened unless you name a file)`, the active-change line or `No active change`, and the ready signal `Ready to discuss the system, incognito. What would you like to rethink?` — and MUST NOT emit a `Next:` line. Its Key Properties table MUST carry the fab-discuss rows plus `Loads docs/memory/* / docs/specs/*? No`.

- **GIVEN** `.agents/skills/` is missing or empty
- **WHEN** the catalog scan runs
- **THEN** the skill STOPs with `Deployed skills not found — run fab sync.`
- **GIVEN** `fab resolve --folder --or-none` prints `(none)`
- **WHEN** the summary renders
- **THEN** it shows `No active change` and treats this as expected, not an error

#### R5: Portability (Constitution V)
The new skill MUST cite only kit skills, `fab` commands, `fab/project/` files, `$(fab kit-path)/…` assets, and the host convention paths `docs/memory/index.md` / `docs/specs/index.md`; it MUST NOT cite fab-kit's `docs/specs/*.md`, `docs/memory/*.md` (beyond the allow-list), `docs/site/*`, or `src/go/*`. The Go guard `TestKitContentCitesNoRepoLocalPaths` MUST stay green.

- **GIVEN** the new skill file exists
- **WHEN** `go test ./src/go/fab-kit/cmd/fab/ -run KitContent` runs
- **THEN** it passes

### CLI: `/fab-help` grouping

#### R6: `fab fab-help` lists the skill under "Start & Navigate"
`skillToGroupMap` in `src/go/fab/cmd/fab/fab_help.go` MUST map `"fab-incognito"` to `"Start & Navigate"` next to `fab-discuss`, and `expectedMapped` in `TestFabHelp_GroupMapping` MUST include `"fab-incognito"`. This is a skill-list constant, not a command-signature change — no `_cli-fab*.md` edit.

- **GIVEN** the skill is deployed
- **WHEN** `fab fab-help` runs
- **THEN** `/fab-incognito` renders under "Start & Navigate", not "Other"

### Docs and sibling sweep

#### R7: Sibling cross-references and sweep
Every surface that enumerates or singles out `/fab-discuss` as the discussion primer MUST gain the sibling: `src/kit/skills/fab-discuss.md` (one "see also" line in `## Purpose`), `src/kit/skills/_preamble.md` (§ Always Load derived-exception examples; § Next Steps Convention opt-out examples), `docs/specs/skills.md` (new `## /fab-incognito` section beside `## /fab-discuss` with Purpose/Context/Key properties/Output/Flow/Tools/Sub-agents; the two Next-line opt-out mentions), `docs/specs/glossary.md` (row), `docs/specs/overview.md` (quick-reference row), `docs/specs/user-flow.md` (the `/fab-discuss` entry-edge label gains `/fab-incognito`), `README.md` (onboarding sentence, command table row, stage-coverage Cyan legend row + entry-point sentence, one-line note under the quick-reference matrix — no new matrix column, no SVG change). `code-dedupe.md`/`code-reorg.md` analogies, findings, logs, `config.md`, `planning-skills.md` DD, `CONTRIBUTING.md` stay untouched.

- **GIVEN** apply is finishing
- **WHEN** `grep -rn 'fab-discuss' src/kit/skills docs/specs README.md` is re-run
- **THEN** every enumeration/singling-out site either names `/fab-incognito` alongside or is on the deliberate-skip list above

### Non-Goals

- No change to `/fab-discuss` loading or output beyond the one cross-reference line
- No new `fab` verb/flag/config key; no migration
- No enforcement mechanism for the Standing Session Rule (prompt-level obligation, Constitution I)
- No new stage-coverage matrix column or SVG edit
- Memory edits happen at hydrate (`_shared/context-loading`, `pipeline/preflight`), not in apply

### Design Decisions

#### Standalone Sibling Skill, Not a fab-discuss Flag
**Decision**: `fab-incognito` is its own skill file with its own `## Context Loading` override.
**Why**: the two skills have opposite load profiles (doc indexes vs kit process partials), opposite ready signals, and incognito carries a rule that persists beyond the invocation; a skill file is the place `_preamble.md` §1 tells readers to look for the override, and a named skill is discoverable in `/fab-help`.
**Rejected**: `/fab-discuss --incognito` (hides the affordance behind a flag and makes fab-discuss's Context Loading conditional — user rejected); names `fab-rethink` / `fab-fresh` / `fab-kit-discuss`.
*Introduced by*: 260915-znw8-fab-incognito-skill

#### Helpers Loaded in Full, Skill Bodies and CLI Partials Lazy
**Decision**: the six process helpers (~120 KB) are the up-front payload; skill bodies and `_cli-*` partials (~274 KB) open on demand; the catalog is frontmatter-only.
**Why**: "clean context" is a misnomer — incognito is a different anchor, not less context — so it must not consume the window with ~834 KB up front while still carrying the whole process.
**Rejected**: loading every deployed skill body at start.
*Introduced by*: 260915-znw8-fab-incognito-skill

## Tasks

### Phase 1: Core Implementation

- [x] T001 Write `src/kit/skills/fab-incognito.md` — frontmatter (`name`, `description`, `helpers: [_pipeline, _intake, _srad, _generation, _review]`), `_preamble` opener, `## Purpose`, `## Arguments`, `## Context Loading` (5-file `fab/project/` override, helpers, catalog scan with `Deployed skills not found — run fab sync.` STOP, `cat "$(fab kit-path)/VERSION"` with `unknown` fallback, lazy-load list with size rationale, active-change resolve), `## Standing Session Rule`, `## Command Logging`, `## Behavior`, `## Orientation Summary`, `## Key Properties`; cite only allow-listed paths <!-- R1, R2, R3, R4, R5 -->
- [x] T002 [P] Add `"fab-incognito": "Start & Navigate"` to `skillToGroupMap` in `src/go/fab/cmd/fab/fab_help.go` beside `fab-discuss`; add `"fab-incognito"` to `expectedMapped` in `src/go/fab/cmd/fab/fab_help_test.go`; run `go test ./src/go/fab/cmd/fab/ -run FabHelp` <!-- R6 -->
- [x] T003 [P] Kit sibling edits: one see-also line in `src/kit/skills/fab-discuss.md` `## Purpose`; `src/kit/skills/_preamble.md` § Always Load derived-exception examples gain `/fab-incognito` (reduced 5-file `fab/project/` set, both doc indexes skipped) and § Next Steps Convention opt-out examples gain `/fab-incognito`'s ready signal <!-- R7 -->

### Phase 2: Docs Sweep & Verification

- [x] T004 Docs sweep: `docs/specs/skills.md` (new `## /fab-incognito` section after `## /fab-discuss`; Next-line opt-out mentions at § Next Steps Convention and New Skill Checklist item 4), `docs/specs/glossary.md` row, `docs/specs/overview.md` quick-reference row, `docs/specs/user-flow.md` entry-edge label, `README.md` (onboarding sentence ~L148, command table ~L444, Cyan legend row + entry-point sentence ~L493–497, one-line note under the quick-reference matrix ~L627) <!-- R7 -->
- [x] T005 Verify: `fab sync` deploys `.agents/skills/fab-incognito/SKILL.md`; `fab fab-help` shows `/fab-incognito` under "Start & Navigate"; `go test ./src/go/fab-kit/cmd/fab/ -run KitContent` and `go test ./src/go/fab/cmd/fab/ -run FabHelp` pass; `gofmt -l src/go` is empty; re-grep `fab-discuss` across `src/kit/skills docs/specs README.md` and confirm every enumeration site is covered or on the skip list <!-- R5, R6, R7 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/fab-incognito.md` exists with the required frontmatter (`name`, `description`, `helpers` = the five allowed partials) and all ten body sections in order
- [x] A-002 R2: `## Context Loading` states the 5-file `fab/project/` override, names both skipped doc indexes, lists the lazy loads with the size rationale, and resolves the active change via `fab resolve --folder --or-none`
- [x] A-003 R3: `## Standing Session Rule` exists as its own section and carries all four clauses (no memory/specs incl. indexes; no Memory File Lookup; user-named-file exception without index walk; discussion-only scope, later skills follow their own Context Loading)
- [x] A-004 R4: `## Orientation Summary` block contains the `Kit:` line, helper list, catalog counts, the verbatim `Memory and specs: NOT loaded (incognito — will not be opened unless you name a file)` line, the active-change line, and the ready signal; `## Command Logging` runs `fab log command "fab-incognito"`; Key Properties carries the `Loads docs/memory/* / docs/specs/*? No` row and `Outputs Next: line? No`
- [x] A-005 R6: `skillToGroupMap` maps `fab-incognito` to `Start & Navigate` and `expectedMapped` includes it
- [x] A-006 R7: every sweep surface listed in R7 names `/fab-incognito` (fab-discuss.md, _preamble.md ×2, skills.md section + 2 mentions, glossary.md, overview.md, user-flow.md, README.md ×4)

### Behavioral Correctness

- [x] A-007 R6: `fab fab-help` output (built from this tree) lists `/fab-incognito` under "Start & Navigate", not "Other"
- [x] A-008 R1: `fab sync` deploys `.agents/skills/fab-incognito/SKILL.md` (and `.claude/skills/fab-incognito/SKILL.md` where `claude` is available) with frontmatter intact

### Scenario Coverage

- [x] A-009 R3: the skill text makes the "cite the implementing skill, not the memory" instruction explicit so the hydrate-question scenario resolves without opening memory
- [x] A-010 R2: the skill instructs that `_cli-*` partials and skill bodies open only when the discussion turns to that skill/command family

### Edge Cases & Error Handling

- [x] A-011 R4: missing/empty `.agents/skills/` ⇒ STOP with `Deployed skills not found — run fab sync.`
- [x] A-012 R4: missing `$(fab kit-path)/VERSION` ⇒ `Kit: unknown`, no error
- [x] A-013 R4: `(none)` from `fab resolve --folder --or-none` ⇒ `No active change`, non-zero exit ⇒ surfaced as an error per the preamble failure rule

### Code Quality

- [x] A-014 Pattern consistency: the new skill follows the fab-discuss/fab-help header pattern (frontmatter, `_preamble` opener, section order, Key Properties table shape)
- [x] A-015 No unnecessary duplication: the skill points at `_preamble.md` §1 for the "skill file wins" derivation and at `/fab-discuss` for nothing it restates; it owns the incognito rule and states it once (owner-or-pointer)
- [x] A-016 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`; deployed copies come from `fab sync`
- [x] A-017 Deployed content cites no fab-kit-only paths: `TestKitContentCitesNoRepoLocalPaths` passes (R5)
- [x] A-018 Go changes ship tests and are gofmt-clean: `TestFabHelp_GroupMapping` extended, `go test ./src/go/fab/cmd/fab/ -run FabHelp` passes, `gofmt -l src/go` empty
- [x] A-019 No CLI reference edit needed: `_cli-fab.md` untouched (skill-list constant, not a command signature) — and no migration (no user-data restructuring)
- [x] A-020 Sibling sweep done as a class: the R7 skip list is honored (code-dedupe/code-reorg analogies, findings, logs, config.md, planning-skills DD, CONTRIBUTING.md untouched)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `docs/specs/user-flow.md`: extend the existing `WT -->|"/fab-discuss"| IDEA` edge label to name `/fab-incognito` too, rather than adding a new edge | Intake said "do not invent a new flow edge"; both skills are worktree→idea entry primers, so one shared label is accurate and minimal | S:70 R:90 A:80 D:75 |
| 2 | Confident | README stage-coverage: Cyan legend row and entry-point sentence gain `/fab-incognito`; one line under the quick-reference matrix says it covers the same `context` cell; no new column, no SVG edit | Intake row 17 (legend note, not a column); the Cyan row is the legend | S:60 R:85 A:75 D:65 |
| 3 | Confident | Five tasks ⇒ light lane; apply and hydrate run inline, review dispatched | `_pipeline.md` Step 1 fork rule (≤ 5); intake predicted light | S:90 R:95 A:95 D:90 |
| 4 | Certain | `CONTRIBUTING.md` row describing `planning-skills.md` is untouched | It describes a memory file's contents, which this change does not add fab-incognito to | S:85 R:95 A:90 D:85 |
| 5 | Confident | `docs/specs/skills.md` New Skill Checklist item 4's opt-out example gains `fab-incognito` alongside `fab-discuss`/`fab-operator` | It enumerates opt-out examples; sibling sweep class per code-quality.md | S:70 R:90 A:80 D:75 |

5 assumptions (1 certain, 4 confident, 0 tentative).
