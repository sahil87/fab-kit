# Tasks: Flatten Skill Helper Include Tree

**Change**: 260418-or0o-flatten-skill-helpers
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Read current `src/kit/skills/_naming.md` in full, strip YAML frontmatter, and stage its body content for insertion into `_preamble.md` as `## Naming Conventions`. Preserve all content verbatim.
- [x] T002 [P] Read current `src/kit/skills/_cli-rk.md` in full, strip YAML frontmatter, and stage its body content for insertion into `_preamble.md` as `## Run-Kit (rk) Reference`. Preserve the silent-fail-when-rk-missing behavior documentation verbatim.
- [x] T003 [P] Grep across `src/kit/skills/*.md` for runtime `fab <command>` invocations and compile a canonical list of the 5 most-used command families with example invocations, to populate `## Common fab Commands` in `_preamble.md`. Target families per spec: `status`, `change`, `score`, `preflight`, `log command`.

## Phase 2: Core Implementation

### 2a: Restructure `_preamble.md`

- [x] T004 Edit `src/kit/skills/_preamble.md` — delete the three "Also read the `_X`" paragraphs in §1 (Context Loading → Always Load). Replace with a single sentence pointing to the new `Skill Helper Declaration` subsection.
- [x] T005 Edit `src/kit/skills/_preamble.md` — add a new `## Skill Helper Declaration` subsection after the Context Loading section. Document the `helpers:` frontmatter field, its 4 allowed values (`_generation`, `_review`, `_cli-fab`, `_cli-external`), an example frontmatter block, the semantics (read each declared helper after `_preamble`, before the skill body), and the default (empty → only `_preamble`). Explicitly state that `_naming` and `_cli-rk` are not allowed values (they are inlined) and that `_preamble` is implicit.
- [x] T006 Edit `src/kit/skills/_preamble.md` — add a new `## Naming Conventions` subsection using the body content staged in T001.
- [x] T007 Edit `src/kit/skills/_preamble.md` — add a new `## Run-Kit (rk) Reference` subsection using the body content staged in T002.
- [x] T008 Edit `src/kit/skills/_preamble.md` — add a new `## Common fab Commands` subsection using the data compiled in T003. Format: a table with columns `Command | Purpose | Canonical form`. Cross-reference `_cli-fab` for exhaustive flag documentation.

### 2b: Delete inlined helper files

- [x] T009 Delete `src/kit/skills/_naming.md`.
- [x] T010 Delete `src/kit/skills/_cli-rk.md`.
- [x] T011 Delete `docs/specs/skills/SPEC-_cli-rk.md`.

### 2c: Audit and set `helpers:` frontmatter in every skill

- [x] T012 Edit `src/kit/skills/fab-new.md` — set frontmatter `helpers: [_generation]`.
- [x] T013 Edit `src/kit/skills/fab-draft.md` — set frontmatter `helpers: [_generation]`.
- [x] T014 Edit `src/kit/skills/fab-continue.md` — set frontmatter `helpers: [_generation, _review]`.
- [x] T015 Edit `src/kit/skills/fab-ff.md` — set frontmatter `helpers: [_generation, _review]`.
- [x] T016 Edit `src/kit/skills/fab-fff.md` — set frontmatter `helpers: [_generation, _review]`.
- [x] T017 Edit `src/kit/skills/fab-operator.md` — set frontmatter `helpers: [_cli-fab, _cli-external]`. Also update the "Required Reading" list (spec lines 41–43): remove the `_naming.md` line, keep the `_cli-fab.md` and `_cli-external.md` lines.
- [x] T018 [P] Audit the remaining 19 skills — `fab-discuss`, `fab-status`, `fab-help`, `fab-clarify`, `fab-archive`, `fab-setup`, `fab-switch`, `fab-proceed`, `docs-hydrate-memory`, `docs-hydrate-specs`, `docs-reorg-memory`, `docs-reorg-specs`, `internal-consistency-check`, `internal-retrospect`, `internal-skill-optimize`, `git-branch`, `git-pr`, `git-pr-review`. Confirm each has no `helpers:` field (or explicit empty list). If any already has a `helpers:` field, remove it or set to `[]`.

### 2d: Update inline `_naming` references

- [x] T019 [P] Edit `src/kit/skills/git-branch.md` — replace `> Branch naming conventions are defined in _naming.md.` with `> Branch naming conventions are defined in _preamble.md § Naming Conventions.`
- [x] T020 [P] Edit `src/kit/skills/git-pr.md` — same substitution as T019.

### 2e: Compress `_cli-fab.md`

- [x] T021 Read `src/kit/skills/_cli-fab.md` in full. Identify: (a) redundant prose that can convert to tables, (b) the 5 common commands already inlined in `_preamble` (remove from `_cli-fab` with a cross-reference note), (c) historical rationale sections to relocate to `docs/memory/fab-workflow/`, (d) canonical flag documentation to preserve verbatim.
- [x] T022 Rewrite `src/kit/skills/_cli-fab.md` to ≤300 lines. Use condensed syntax/flag tables. Preserve all canonical flag behavior documentation. Add cross-reference at top: "See `_preamble.md § Common fab Commands` for the 5 most-used commands. This file documents the remaining commands and flag details."
- [x] T023 Relocate any historical rationale removed from `_cli-fab.md` into `docs/memory/fab-workflow/` under an appropriate existing memory file (most likely `kit-architecture.md` or a dedicated `cli-rationale.md`). Preserve attribution to the originating change if documented.

## Phase 3: Integration & Edge Cases

### 3a: Deployed directory cleanup

- [x] T024 Run `fab sync` to regenerate `.claude/skills/` from source. Verify `.claude/skills/_naming/` and `.claude/skills/_cli-rk/` no longer exist. If either persists (stale directories not removed by sync), manually delete via `rm -rf .claude/skills/_naming .claude/skills/_cli-rk`. This is a local-workspace-only cleanup — `.claude/skills/` is gitignored.

### 3b: Memory and spec updates

- [x] T025 Edit `docs/memory/fab-workflow/context-loading.md` — in `### Always Load Layer`, remove the 3-bullet list of `_cli-fab`/`_naming`/`_cli-rk` internal skill references. Replace with a single sentence: "The only universal helper beyond the 7 project files is `_preamble.md`. Additional helpers are declared per-skill via `helpers:` frontmatter — see the `Skill Helper Declaration (Opt-In)` subsection below." Then add a new `### Skill Helper Declaration (Opt-In)` subsection documenting the 4 allowed values. Mark the existing `### Always-Load \`_cli-rk\` Skill for rk Capabilities` design decision as superseded (with a pointer to this change). Add a new design decision `### Flatten Helper Include Tree` explaining the rationale (fanout caused unreliable loads + wasted context; per-skill opt-in replaces universal inheritance). Append a changelog entry.
- [x] T026 Edit `docs/memory/fab-workflow/kit-architecture.md` — in the skills directory tree (around line 22), remove the `_naming.md` entry. Update the always-load/selective mapping table (around lines 367–374) so `_preamble.md` is the only always-load helper; move `_cli-fab.md` to "Selective (via `helpers:` frontmatter)". Remove the `_naming.md` line from that table. Update any references to `_cli-fab.md` as "Always-load (via preamble)" → "Selective (via `helpers:` frontmatter)". Append a changelog entry.
- [x] T027 Edit `docs/specs/skills/SPEC-preamble.md` — document the four new subsections added to `_preamble.md`: `## Skill Helper Declaration`, `## Naming Conventions`, `## Run-Kit (rk) Reference`, `## Common fab Commands`. For each, include a brief description and cross-reference to `_preamble.md` itself as the canonical source.
- [x] T028 Edit `docs/specs/skills.md` — add a subsection documenting the `helpers:` frontmatter field, its 4 allowed values, and the default behavior. Place near the frontmatter conventions section if one exists, or in a new `## Skill Helpers` top-level section.

### 3c: Backlog cleanup

- [x] T029 Edit `fab/backlog.md` — remove line `[ ] [84bh] 2026-04-02: Try using references instead of _ folders in skills - other agents don't read from _ folders in skills`. This item is subsumed by the current change.

## Phase 5: Rework Cycle 1 (post-review findings)

- [x] T033 Edit `docs/memory/fab-workflow/kit-architecture.md` line 9 — update the banner note about `_cli-fab.md` from "the canonical CLI reference, loaded by every skill via `_preamble.md`" to reflect the new selective/opt-in model. Suggested replacement: "(the canonical CLI reference — loaded selectively via a skill's `helpers: [_cli-fab]` frontmatter; the most-used command families are inlined into `_preamble.md § Common fab Commands`.)"
- [x] T034 Edit `docs/memory/fab-workflow/kit-architecture.md` line 24 — update the directory-tree inline annotation `_cli-fab.md          # Fab CLI command reference (always-load, renamed from _scripts.md)` to remove the stale "(always-load, renamed from _scripts.md)" comment. Replace with something like `_cli-fab.md          # Fab CLI command reference (selective via helpers: [_cli-fab])`.
- [x] T035 Reconcile the 5-vs-6 command count drift across artifacts. The actual `_preamble.md § Common fab Commands` table lists 6 families (`preflight`, `score`, `log command`, `change`, `resolve`, `status`). Update every artifact that currently says "5" or "top-5" to say "6" or "top-6" instead, for consistency with reality: (a) `src/kit/skills/_cli-fab.md` line 11; (b) `docs/memory/fab-workflow/context-loading.md` line 152 (the changelog entry); (c) `fab/changes/260418-or0o-flatten-skill-helpers/spec.md` § "Common fab commands inlined into `_preamble`" and the supporting scenario. Do NOT drop a row from the `_preamble` table — keep 6.
- [x] T036 Edit `src/kit/skills/fab-operator.md` — remove the body "Also read:" paragraph at lines 41-43 (or wherever it currently sits after T017 edits). The frontmatter `helpers: [_cli-fab, _cli-external]` already declares these. Replacement: either delete the paragraph entirely, or reword to "Helpers declared in frontmatter: `_cli-fab` (fab command reference) and `_cli-external` (wt, idea, tmux, /loop reference). See `_preamble.md § Naming Conventions` for change/branch/worktree naming rules." The goal is zero body-level "Also read" directives in any skill post-change, matching the new convention.
- [x] T037 Edit `src/kit/skills/_preamble.md` inlined `## Naming Conventions` subsection — restore `### Known change (already exists)` and `### New change (from backlog)` as H3 subheadings (currently flattened to bold paragraph labels at lines 155 and 163). Verbatim preservation of the original `_naming.md` heading structure.
- [x] T038 [P] Add a one-line `helpers:` note to each per-skill SPEC where the corresponding skill got a new `helpers:` frontmatter: `docs/specs/skills/SPEC-fab-new.md`, `SPEC-fab-draft.md`, `SPEC-fab-continue.md`, `SPEC-fab-ff.md`, `SPEC-fab-fff.md`. Each note should record the declared helpers value (e.g., "Declares `helpers: [_generation, _review]` in frontmatter per the skill-helpers convention documented in `docs/specs/skills.md § Skill Helpers`."). Place consistently — top of the SPEC near the summary/metadata.

## Phase 4: Polish

- [x] T030 Verify `src/kit/skills/_cli-fab.md` is ≤300 lines: `wc -l src/kit/skills/_cli-fab.md`. If over by more than 10%, trim further. If under 300 but canonical content needs more space, accept slight overshoot but document in the change's final commit message.
- [x] T031 Verify `src/kit/skills/_preamble.md` contains all four new subsections (`## Skill Helper Declaration`, `## Naming Conventions`, `## Run-Kit (rk) Reference`, `## Common fab Commands`) and zero occurrences of "Also read the `_cli-fab`", "Also read the `_naming`", or "Also read the `_cli-rk`".
- [x] T032 Audit: grep across `src/kit/skills/*.md` for any remaining references to `_naming.md` or `_cli-rk.md`. Expected matches: zero (all inlined or updated to point at `_preamble.md § Naming Conventions`).

---

## Execution Order

- **Phase 1 (T001–T003)** runs first, all parallelizable — pure reads/staging, no file writes.
- **Phase 2a (T004–T008)** depends on T001–T003 output. Within 2a, tasks are sequential because they all edit `_preamble.md`.
- **Phase 2b (T009–T011)** depends on T004–T008 completing (content must be inlined into `_preamble.md` before source files are deleted).
- **Phase 2c (T012–T018)** is independent of 2a/2b and can run alongside Phase 2a (different files). T018 is parallelizable within itself (19 independent files).
- **Phase 2d (T019–T020)** depends on T006 (Naming Conventions subsection exists in `_preamble.md`). T019 and T020 are parallel.
- **Phase 2e (T021–T023)** is independent of other Phase 2 work (different file: `_cli-fab.md`). T021 → T022 → T023 sequential.
- **Phase 3a (T024)** depends on all Phase 2 work completing (sync regenerates deployed tree).
- **Phase 3b (T025–T028)** depends on Phase 2 completing (docs describe the new structure). Within 3b, all tasks edit different files and can run in parallel.
- **Phase 3c (T029)** independent, can run anytime.
- **Phase 4 (T030–T032)** is the final verification pass after all Phase 2/3 work.
