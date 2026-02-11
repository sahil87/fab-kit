## Done

- [x] [eili] 2026-02-06: Branch creation
- [x] [hh1n] 2026-02-06: Ability to work on a particular stage individually (spec or plan or tasks)
- [x] [uf7a] 2026-02-06: A way to go deep in a spec — iterative clarify/research loops to refine ambiguities repeatedly
- [x] [hccv] 2026-02-06: discuss what happens in a monorepo → single fab/ at root, structured context sections in config.yaml
- [x] [iioo] 2026-02-06: Add starting instruction (is simply copying the fab/.kit folder enough/is there a script to run?)
- [x] [cxpe] 2026-02-06: ~~Onboard command~~ → absorbed into `fab:init` (idempotent, accepts sources: Notion URLs, Linear URLs, local files)
- [x] [8dx2] 2026-02-06: After every command suggest the next possible commands in the flow → added Next Steps Convention with lookup table in SKILLS.md
- [x] [k7ho] 2026-02-06: Read the specs at doc/fab-spec/README.md. Implement it in this repo. (By creating distributable fab/.kit folder in this repo, symlinking .claude etc). Replicate the actual structure that other repos would be using.
- [x] [2t3g] 2026-02-07: Create a setup.sh in the .kit folder, so one can run fab/.kit/setup.sh to setup all the symlinks required properly
- [x] [88gc] 2026-02-07: add a fab-help skill also
- [x] [pn18] 2026-02-07: fab-init needs to be sync with the setup script. Or else the directory strucutres for both these commands will go out of sync
- [x] [waup] 2026-02-07: Understand the spec from doc/fab-spec/* . Then look at current implementation in fab/* . Our task: Standardize the scripts names (in fab/.kit/scripts/) : add a fab- prefix, Create a fab-help.sh script, and make /fab-help skill call the fab-help.sh script internally.
- [x] [9iyu] 2026-02-07: Understand the spec from doc/fab-spec/* . Separate our docs (fab/docs/) hydartion from fab:init - these two can be made separate (vai a new fab:hydrate command). Right now I think fab:init had this dual responsibility. That should no longer be the case. Now that we have a proper documentation in place at fab/docs/ after running fab:hydrate the following change should be safe: ensure the relevant context is always loaded smartly (from fab/docs) before any fab: command. (And hence) the documentation in fab/docs should be properly indexed (contain proper index.md files referencing other files) so its easy agents to get to and load the relevant sections to context fast.
- [x] [twpd] 2026-02-07: Add a fab-preflight.sh script that consolidates the repeated pre-flight context loading that
  every skill performs at invocation. Currently, every skill (ff, apply, review, archive)
  independently reads fab/current, .status.yaml, config.yaml, constitution.md, and
  fab/docs/index.md — 5 file reads as boilerplate before any real work starts. The script should
  read fab/current, validate the change directory and .status.yaml exist, and output structured
  YAML with the change name, stage, progress map, checklist counts, and branch name. Skills can
  consume this output instead of each re-reading and re-validating independently. Additionally, add
   a fab-grep-all.sh <pattern> utility that searches all fab-managed file locations
  (fab/.kit/skills/, fab/.kit/scripts/, doc/fab-spec/) so that consistency/verification tasks have
  a single reliable search scope instead of ad-hoc greps that miss locations (as happened when
  fab-help.sh was missed during the hydrate change).
- [x] [g3nm] 2026-02-07: Add a "generate" mode to fab-hydrate alongside the existing "ingest" mode. Currently fab-hydrate only handles external source ingestion (Notion URLs, Linear URLs, local files). The new mode should scan the codebase, identify undocumented areas (APIs, modules, patterns, architecture), and generate docs from code analysis into fab/docs/ with proper indexing. For large codebases with many undocumented sections, it should offer interactive scoping — presenting discovered gaps and letting the user prioritize what to document first (similar to Codex's "/init" command).
- [x] [ny4x] 2026-02-07: In the next command suggestion we are giving, all fab commands are listed as fab:xxx instead of fab-xxxx. This needs to be updated.
- [x] [hwnz] 2026-02-07: Separate out docs and specs. Specs: Maching generated, how everything works. Docs: much shorted, for humans. Need to undergo review before mergin
- [x] [e1fp] 2026-02-07: architecture review AND checking all commands for simplification, and then autonomy - be more biased towards asking qns
- [x] [sflf] 2026-02-07: create a fab-fff command that takes you all the way, given a proposal, to archive, this should absorb the fab-ff --auto mode (we no longer need it after this). This should be allowed only if the level of ambiguity is low - we should be able to determine this from the ambiguity score in .status.yaml
- [x] [eb7z] 2026-02-08: Using SRAD we should be able to come up with a level of ambiguity for every single proposal. If the level of ambiguity is high, then don't allow the user to run the FFF command. If it is low, then that command can be suggested.
- [x] [v7qm] (BUG) 2026-02-07: fab-hydrate has broken template links (lines 97, 104, 124) pointing to old `doc/fab-spec/TEMPLATES.md` path after commit 9329bd5 moved files to `fab/specs/`
- [x] [spcy] 2026-02-08: add a command fab-discuss that helps us to discuss anything (even an existing proposal), and if something comes of it - output a solid proposal.md (or improve the existing one) - like fab-new, with the differences being that fab-discuss doesn't switch to the change (you need to fab-switch to it) and it helps you understand if you are filling a gap (is the change even needed), walks you through making a solid proposal, asking clarifying questions. fab-new and fab-discuss should also try to fill the ambiquity score in .status.yaml (from the SRAD framework). When creating a task from fab-discuss, we want the ambiguity score to be low, so that after a long discussion, we would be able to directly run fab-fff.
- [x] [2jo9] 2026-02-08: add an index.md to the fab/changes/archive folder which gets updated on every fab-archive so its easy to search for changes later. The description here in this table should be longer, maybe one to two lines instead of just the name of the change folder itself. Change the folder name length requirement in fab-new from just 2-4 words to maybe longer, something like 2-7 words, maybe.
- [x] [7fbf] 2026-02-09: FAB status should also show the confidence score
- [x] [s3d6] 2026-02-09: After `/fab-discuss` finalizes a new change, ask the user if they want to switch to it (#24)
- [x] [dxcf] 2026-02-08: Add a command called fab-backfill that looks at the docs (fab/docs) and specs (fab/specs) and points out a max of top three areas that can be hydrated back from docs to specs (#22)
- [x] [k3wf] (BUG) 2026-02-07: fab-continue and fab-ff duplicate spec/plan/tasks/checklist generation logic nearly verbatim — extract to a shared `_generation.md` partial (#28)
- [x] [r8tn] (BUG) 2026-02-07: fab-ff invokes fab-clarify in auto mode but no mechanism (flag, context variable) is defined for how one skill signals mode to another (#29)
- [x] [p2xe] (BUG) 2026-02-07: fab-continue stage guard checks stage name not progress value — if `stage: tasks` but `progress.tasks: active` (interrupted), the guard incorrectly blocks resumption (#25)
- [x] [j5bh] (BUG) 2026-02-07: fab-apply, fab-review, fab-archive omit `fab/specs/index.md` from context loading, deviating from `_context.md` always-load protocol (#26)
- [x] [m1gc] (BUG) 2026-02-07: fab-new collision handling says "append an additional random character" (making 5 chars) instead of "regenerate the 4-character component" (#27)
- [x] [oa32] 2026-02-10: make prompt pantry opencode compatible (#30)
- [x] [7jfn] 2026-02-08: Rewrite README.md to lead with what Fab is
- [x] [wr07] 2026-02-08: Make fab/config.yaml self-documenting with inline comments explaining every field
- [x] [gqs1] 2026-02-08: Fix broken links in README.md and fab/specs/overview.md
- [x] [74eh] 2026-02-08: Standardize skill invocation syntax in specs and docs — `/fab-xxx` dash syntax
- [x] [xeti] 2026-02-08: Reduce overly broad permissions in `.claude/settings.local.json`
- [x] [wr10] 2026-02-10: Add Documentation Map to README with audience-specific reading paths, grouped inventory, glossary links, and index file orientation notes

## Backlog

### Commands & Scripts

- [ ] [90g5] 2026-02-07: Add a constitution command that creates the constitution - base on SpecKit's constitution
- [ ] [jgt6] 2026-02-07: hydrate should distinguish between specs and docs. Ingestion = specs. Generation = docs
- [ ] [n4j0] 2026-02-08: Add a fab update script that updates the fab/dotkit folder from a central repo.
- [ ] [s1u9] 2026-02-09: Add `fab/.kit/scripts/fab-status-update.sh` — a helper script that takes `key=value` pairs and updates `.status.yaml` fields mechanically. Every stage transition currently requires a manual read-edit-write cycle on `.status.yaml` (6 times during a single fab-fff run). The script should handle nested YAML paths (e.g. `progress.tasks=done`, `checklist.generated=true`) and always update `last_updated` automatically.
- [ ] [s2t4] 2026-02-09: Add `fab/.kit/scripts/fab-task-complete.sh` — takes a task ID (e.g. `T001`) and marks it `[x]` in the active change's `tasks.md`. Currently every task completion during fab-apply requires a separate edit call to flip `- [ ]` to `- [x]`. The script should read `fab/current`, locate the task line by ID, and toggle the checkbox.
- [ ] [s4c8] 2026-02-09: Extract a `fab-changelog-insert.sh` script that automates the repetitive changelog row insertion during `/fab-archive` hydration. Currently each archive run requires 3-6 manual Edit calls to prepend changelog rows to centralized docs and update "Last Updated" dates in the domain index. The script should take `change-name`, `date`, and a list of `doc-path:summary` pairs, and for each: insert a row after the changelog table header, and update the corresponding "Last Updated" column in `fab/docs/{domain}/index.md`. Alternatively, add more explicit mechanical guidance to the `/fab-archive` skill prompt to make this pattern less error-prone for agents.

### Documentation

- [ ] [qurg] 2026-02-08: Document the SRAD framework. SRAD is referenced by fab-fff confidence gating and backlog item `[eb7z]` as implemented, but the framework itself — what the acronym stands for, how ambiguity scores are calculated, what thresholds gate fab-fff, how scores are stored in `.status.yaml` — is defined nowhere in the codebase. Without this, the confidence gating logic is opaque to anyone who didn't build it. Add a section to `fab/specs/skills.md` or a standalone `fab/specs/srad.md` covering: the acronym expansion, the scoring dimensions, how each dimension is evaluated, the threshold that gates fab-fff, and an example of a high-ambiguity vs low-ambiguity proposal.
- [ ] [wr01] 2026-02-08: Write an end-to-end "Your First Change" tutorial. Currently there is no guided path from clone to completed archive entry. Create a doc (e.g. fab/docs/fab-workflow/tutorial.md) that walks a new user through: cloning the repo, running worktree-init scripts, running fab-new with a simple example change, progressing through each stage (proposal → spec → plan → tasks → apply → review → archive), and verifying the result. Reference archived changes as "what good looks like". Should be readable in under 10 minutes and result in one complete archived change. Index it from fab/docs/index.md and the README.
### Cleanup & Hardening

- [ ] [wr04] 2026-02-08: Harden all shell scripts in fab/worktree-init/ with proper error handling and safe variable expansion. Currently 1-direnv.sh, 2-claude-settings.sh, and 3-fab-setup.sh lack `set -euo pipefail`, have unquoted variable expansions vulnerable to word splitting on paths with spaces, and silently continue on failure. For each script: add `set -euo pipefail` at the top, quote all variable expansions (`"$var"` not `$var`), add meaningful error messages on failure, validate prerequisites exist before operating on them. Also audit any other .sh files under fab/.kit/scripts/ for the same issues.
- [ ] [alat] 2026-02-10: Scores don't change after clarify - clarification should ideally increase state
- [ ] [29xv] 2026-02-10: Scoring formula needs to be relooked at - scores are generaly too high
- [ ] [6j7w] 2026-02-10: Go over the proposal -> specs -> plan -> tasks" "thinking" workflow. Do we need these many? Any rewording? Should state names change?
- [x] [gs42] 2026-02-10: Add attribution / owner for every change - in .status.yaml

## 2026-02-11

### fab-status has no centralized doc section
- [ ] [gs42] 2026-02-11: `execution-skills.md` covers `/fab-apply`, `/fab-review`, `/fab-archive` but not `/fab-status`. Status display behavior is only documented in the skill file and script. Future changes to `/fab-status` have no centralized doc section to hydrate into. Either add a `/fab-status` section to `execution-skills.md` or create a separate `informational-skills.md` doc.

### fab-status.md skill description enumerates fields that drift
- [ ] [gs42] 2026-02-11: `fab-status.md` behavior section lists rendered fields ("version header, change name, branch, stage number, progress table, checklist counts, confidence score") but this list drifts as fields are added — it already doesn't mention `created_by`. Either maintain the enumeration or generalize to "renders all `.status.yaml` fields."

### fab-discuss missing Key Points section for .status.yaml
- [ ] [gs42] 2026-02-11: `/fab-new` has a "Key points" section after its `.status.yaml` yaml block explaining field semantics (e.g. `created_by` population, fallback behavior). `/fab-discuss` has no equivalent — just the bare yaml block. Add a matching "Key points" section so field-level instructions have a home when new fields are added.

### Backlog items not closed by fab-archive
- [ ] [gs42] 2026-02-11: No skill in the pipeline checks or closes backlog items. When a change completes that matches a backlog entry, `/fab-archive` (or `/fab-fff`) should scan `fab/backlog.md` for related items and offer to mark them done. Currently this is manual and easy to forget.

### Make fab/backlog.md personal and worktree-safe
- [ ] [gs42] 2026-02-11: `fab/backlog.md` is committed to git but is a personal scratchpad — not suitable for shared repos. Fix: (1) add `fab/backlog.md` to `.gitignore`, (2) move the canonical file to the main worktree, (3) update `worktree-init.sh` to symlink `fab/backlog.md` → main worktree's copy (same pattern as `.envrc`), (4) update `/retrospect` skill to work with the symlinked file. Need to `git rm --cached fab/backlog.md` to stop tracking it without deleting the file.
