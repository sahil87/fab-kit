# Tasks: Separate Doc Hydration from Init, Add Smart Context Loading, and Index fab/docs

**Change**: 260207-q7m3-separate-hydrate-smart-context
**Plan**: `plan.md`
**Spec**: `spec.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/skills/fab-hydrate.md` with frontmatter (name, description), purpose section, and reference to `_context.md`. Populate with the full hydration skill by moving Phase 2 content from `fab/.kit/skills/fab-init.md` — include pre-flight check (`fab/docs/` must exist), arguments (`[sources...]`), fetch/read logic, domain analysis, doc creation/merge, and index maintenance (both `fab/docs/index.md` and `fab/docs/{domain}/index.md`). Add idempotency guarantees, error handling table, and output format. Ensure the skill references `doc/fab-spec/TEMPLATES.md` for index and doc formats.

## Phase 2: Core Implementation

- [x] T002 Update `fab/.kit/skills/fab-init.md`: Remove Phase 2 (Source Hydration) entirely. Remove `[sources...]` from the title and Arguments section. Add a guard: if arguments are passed, output "Did you mean /fab:hydrate? /fab:init no longer accepts source arguments." Remove the "With Sources" output section. Update first-run output to suggest `/fab:hydrate` alongside `/fab:new`. Update description frontmatter to remove "optionally hydrate docs" wording. Update the Next line to include `/fab:hydrate`.
- [x] T003 [P] Update `fab/.kit/skills/_context.md`: (1) In "Always Load" section, add `fab/docs/index.md` as a third bullet alongside config.yaml and constitution.md. Add `/fab:hydrate` to the exceptions list. (2) Expand "Centralized Doc Lookup" from "when writing or validating specs" to "when operating on an active change" — update the heading, description, and applicability. (3) Ensure the layer description says agents read domain indexes and individual docs selectively based on the proposal's Affected Docs section.
- [x] T004 [P] Update `fab/.kit/skills/fab-archive.md`: In Step 3c (Existing Doc), after updating the domain index, also update the top-level `fab/docs/index.md` doc-list column to reflect any newly added docs in the domain. In Step 3b (New Doc), ensure the top-level index row includes the full doc list for the new domain. This ensures the top-level index stays accurate as docs accumulate.

## Phase 3: Integration & Edge Cases

- [x] T005 Update `doc/fab-spec/SKILLS.md`: (1) Add a new `/fab:hydrate [sources...]` section between `/fab:init` and `/fab:new` — include purpose, prerequisite, arguments, creates, examples, and behavior (mirroring the structure of the init section but scoped to hydration). (2) Update `/fab:init` section: remove `[sources...]` from heading and arguments, remove source hydration from behavior and examples, update description to "structural bootstrap only", add redirect behavior note. (3) Update Context Loading Convention section: add `fab/docs/index.md` to "Always loaded", expand "Centralized doc lookup" to "loaded by skills operating on an active change" (not just spec-writing). (4) Add `/fab:hydrate` to the exceptions list in "Always loaded".
- [x] T006 [P] Update `doc/fab-spec/README.md`: (1) Add `/fab:hydrate [sources...]` row to Quick Reference table with purpose "Ingest external docs into fab/docs/" and creates "Updated fab/docs/ with indexes". (2) Update `/fab:init` row — remove `[sources...]`, change creates to "config.yaml, constitution.md, docs/, skill symlinks (idempotent)". (3) Update "Hydrating Docs from Existing Sources" section to reference `/fab:hydrate` instead of `/fab:init`. Update code examples to use `/fab:hydrate`.
- [x] T007 [P] Update `doc/fab-spec/ARCHITECTURE.md`: (1) Update "The Bootstrap Sequence" to add step 4: "User optionally runs /fab:hydrate → ingests external docs into fab/docs/" and renumber step 4 to step 5. (2) Update the paragraph after the sequence to remove hydration from `/fab:init` re-run description. (3) Update the `fab-setup.sh` description to not mention hydration. (4) Update "Why Two Phases?" section if it mentions source hydration in the init step.

## Phase 4: Polish

- [x] T008 Verify consistency: Read all modified files and check that (1) no references to `/fab:init [sources...]` remain in any skill or doc file, (2) `/fab:hydrate` is consistently described, (3) the `_context.md` exceptions list includes all four exceptions (init, switch, status, hydrate), (4) the fab-setup.sh glob pattern `fab/.kit/skills/fab-*.md` will pick up `fab-hydrate.md` automatically (no script changes needed).

---

## Execution Order

- T001 must complete before T002 (hydrate skill must exist before init can reference it)
- T002, T003, T004 can run in parallel after T001
- T005 depends on T002 and T003 (SKILLS.md must reflect the actual skill file changes)
- T006 and T007 can run in parallel with T005
- T008 runs last (final consistency check across all files)
