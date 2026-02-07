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
- [ ] [9iyu] 2026-02-07: Understand the spec from doc/fab-spec/* . Separate our docs (fab/docs/) hydartion from fab:init - these two can be made separate (vai a new fab:hydrate command). Right now I think fab:init had this dual responsibility. That should no longer be the case. Now that we have a proper documentation in place at fab/docs/ after running fab:hydrate the following change should be safe: ensure the relevant context is always loaded smartly (from fab/docs) before any fab: command. (And hence) the documentation in fab/docs should be properly indexed (contain proper index.md files referencing other files) so its easy agents to get to and load the relevant sections to context fast. 
- [ ] [ny4x] 2026-02-07: In the next command suggestion we are giving, all fab commands are listed as fab:xxx instead of fab-xxxx. This needs to be updated.
- [ ] [twpd] 2026-02-07:   Add a fab-preflight.sh script that consolidates the repeated pre-flight context loading that 
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
- [ ] [g3nm] 2026-02-07: Add a "generate" mode to fab-hydrate alongside the existing "ingest" mode. Currently fab-hydrate only handles external source ingestion (Notion URLs, Linear URLs, local files). The new mode should scan the codebase, identify undocumented areas (APIs, modules, patterns, architecture), and generate docs from code analysis into fab/docs/ with proper indexing. For large codebases with many undocumented sections, it should offer interactive scoping — presenting discovered gaps and letting the user prioritize what to document first (similar to Codex's "/init" command).
- [ ] [hwnz] 2026-02-07: Separate out docs and specs. Specs: Maching generated, how everything works. Docs: much shorted, for humans. Need to undergo review before mergin
