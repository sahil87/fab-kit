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
- [x] [twpd] 2026-02-07:   Add a fab-preflight.sh script that consolidates the repeated pre-flight context loading that 
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
- [ ] [90g5] 2026-02-07: Add a constitution command the creates the constitution - base on SpecKit's constitution
- [x] [e1fp] 2026-02-07: architecture review AND checking all commands for simplification, and then autonomy - be more biased towards asking qns
- [ ] [sflf] 2026-02-07: create a fab-fff command that takes you all the way, given a proposal, to archive, this should absorb the fab-ff --auto mode (we no longer need it after this). This should be allowed only if the level of ambiguity is low - we should be able to determine this from the ambiguity score in .status.yaml 
- [ ] [jgt6] 2026-02-07: hydrate should distinguish between specs and docs. Ingestion = specs. Generation = docs
- [ ] [n4j0] 2026-02-08: Add a fab update script that updates the fab/dotkit folder from a central repo. 
- [ ] [eb7z] 2026-02-08: Using SRAD we should be able to come up with a level of ambiguity for every single proposal. If the level of ambiguity is high, then don't allow the user to run the FFF command. If it is low, then that command can be suggested. 
- [ ] [spcy] 2026-02-08: add a command fab-discuss that helps us to discuss anything (even an existing proposal), and if something comes of it - output a solid proposal.md (or improve the existing one) - like fab-new, with the differences being that fab-discuss doesn't switch to the change (you need to fab-switch to it) and it helps you understand if you are filling a gap (is the change even needed), walks you through making a solid proposal, asking clarifying questions. fab-new and fab-discuss should also try to fill the ambiquity score in .status.yaml (from the SRAD framework). When creating a task from fab-discuss, we want the ambiguity score to be low, so that after a long discussion, we would be able to directly run fab-fff.
- [ ] [dxcf] 2026-02-08: Add a command called fab-backfill that looks at the docs (fab/docs) and specs (fab/specs) and points out a max of top three areas that can be hydrated back from docs to specs. It should confirm with the user what it is planning to add back to specs because the language in these specs needs to be extremely concise and easy to understand for humans. We don't want the specs to bloat up in size.
- [ ] [v7qm] (BUG) 2026-02-07: fab-hydrate has broken template links (lines 97, 104, 124) pointing to old `doc/fab-spec/TEMPLATES.md` path after commit 9329bd5 moved files to `fab/specs/`
- [ ] [k3wf] (BUG) 2026-02-07: fab-continue and fab-ff duplicate spec/plan/tasks/checklist generation logic nearly verbatim — extract to a shared `_generation.md` partial
- [ ] [r8tn] (BUG) 2026-02-07: fab-ff invokes fab-clarify in auto mode but no mechanism (flag, context variable) is defined for how one skill signals mode to another
- [ ] [p2xe] (BUG) 2026-02-07: fab-continue stage guard checks stage name not progress value — if `stage: tasks` but `progress.tasks: active` (interrupted), the guard incorrectly blocks resumption
- [ ] [j5bh] (BUG) 2026-02-07: fab-apply, fab-review, fab-archive omit `fab/specs/index.md` from context loading, deviating from `_context.md` always-load protocol
- [ ] [m1gc] (BUG) 2026-02-07: fab-new collision handling says "append an additional random character" (making 5 chars) instead of "regenerate the 4-character component"
- [ ] [2jo9] 2026-02-08: add an index.md to the fab/changes/archive folder which gets updated on every fab-archive so its easy to search for changes later. The description here in this table should be longer, maybe one to two lines instead of just the name of the change folder itself. Change the folder name length requirement in fab-new from 2 to 4 words to maybe longer, something like 2 to 6 or 2 to 7 words, maybe.

