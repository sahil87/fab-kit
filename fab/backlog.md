## Done

- [x] [eili] 2026-02-06: DEV-949 Branch creation
- [x] [hh1n] 2026-02-06: DEV-950 Ability to work on a particular stage individually (spec or tasks)
- [x] [uf7a] 2026-02-06: DEV-964 A way to go deep in a spec — iterative clarify/research loops to refine ambiguities repeatedly
- [x] [hccv] 2026-02-06: DEV-1003 discuss what happens in a monorepo → single fab/ at root, structured context sections in config.yaml
- [x] [iioo] 2026-02-06: DEV-1004 Add starting instruction (is simply copying the fab/.kit folder enough/is there a script to run?)
- [x] [cxpe] 2026-02-06: DEV-957 ~~Onboard command~~ → absorbed into `fab:init` (idempotent, accepts sources: Notion URLs, Linear URLs, local files)
- [x] [8dx2] 2026-02-06: DEV-956 After every command suggest the next possible commands in the flow → added Next Steps Convention with lookup table in SKILLS.md
- [x] [k7ho] 2026-02-06: DEV-951 Read the specs at doc/fab-spec/README.md. Implement it in this repo. (By creating distributable fab/.kit folder in this repo, symlinking .claude etc). Replicate the actual structure that other repos would be using.
- [x] [2t3g] 2026-02-07: DEV-952 Create a setup.sh in the .kit folder, so one can run fab/.kit/setup.sh to setup all the symlinks required properly
- [x] [88gc] 2026-02-07: DEV-953 add a fab-help skill also
- [x] [pn18] 2026-02-07: DEV-954 fab-init needs to be sync with the setup script. Or else the directory strucutres for both these commands will go out of sync
- [x] [waup] 2026-02-07: DEV-955 Standardize the scripts names (in fab/.kit/scripts/) : add a fab- prefix, Create a fab-help.sh script, and make /fab-help skill call the fab-help.sh script internally.
- [x] [9iyu] 2026-02-07: DEV-958 Separate our docs (fab/docs/) hydration from fab:init into a new fab:hydrate command. Ensure relevant context is always loaded smartly from fab/docs. Documentation should be properly indexed.
- [x] [twpd] 2026-02-07: DEV-959 Add a fab-preflight.sh script that consolidates the repeated pre-flight context loading that every skill performs at invocation. Also added fab-grep-all.sh for searching all fab-managed locations.
- [x] [g3nm] 2026-02-07: DEV-960 Add a "generate" mode to fab-hydrate alongside the existing "ingest" mode. Scans codebase, identifies undocumented areas, generates docs from code analysis into fab/docs/ with proper indexing.
- [x] [ny4x] 2026-02-07: DEV-961 Fix fab command naming — all fab commands listed as fab:xxx instead of fab-xxxx.
- [x] [hwnz] 2026-02-07: DEV-981 Separate out docs and specs. Specs: machine-generated. Docs: much shorter, for humans. Need to undergo review before merging.
- [x] [e1fp] 2026-02-07: DEV-965 architecture review AND checking all commands for simplification, and then autonomy - be more biased towards asking qns
- [x] [sflf] 2026-02-07: DEV-962 create a fab-fff command that takes you all the way, given a proposal, to archive. Only allowed if ambiguity is low (confidence gating).
- [x] [eb7z] 2026-02-08: DEV-963 Using SRAD, compute a level of ambiguity for every proposal. High ambiguity blocks fab-fff; low allows it.
- [x] [v7qm] (BUG) 2026-02-07: DEV-1005 fab-hydrate has broken template links pointing to old `doc/fab-spec/TEMPLATES.md` path
- [x] [spcy] 2026-02-08: DEV-966 add fab-discuss command for discussing ideas and refining proposals through conversation
- [x] [2jo9] 2026-02-08: DEV-969 add an index.md to the fab/changes/archive folder, updated on every fab-archive
- [x] [7fbf] 2026-02-09: DEV-983 FAB status should also show the confidence score
- [x] [s3d6] 2026-02-09: DEV-967 After `/fab-discuss` finalizes a new change, ask the user if they want to switch to it
- [x] [dxcf] 2026-02-08: DEV-968 Add fab-backfill command — looks at docs and specs, points out top three areas to hydrate back
- [x] [k3wf] (BUG) 2026-02-07: DEV-970 fab-continue and fab-ff duplicate generation logic — extract to shared `_generation.md` partial
- [x] [r8tn] (BUG) 2026-02-07: DEV-971 fab-ff invokes fab-clarify in auto mode but no mechanism defined for skill-to-skill mode signaling
- [x] [p2xe] (BUG) 2026-02-07: DEV-972 fab-continue stage guard checks stage name not progress value — blocks resumption incorrectly
- [x] [j5bh] (BUG) 2026-02-07: DEV-973 fab-apply, fab-review, fab-archive omit `fab/specs/index.md` from context loading
- [x] [m1gc] (BUG) 2026-02-07: DEV-974 fab-new collision handling says "append" instead of "regenerate"
- [x] [oa32] 2026-02-10: DEV-982 make prompt pantry opencode compatible
- [x] [7jfn] 2026-02-08: DEV-975 Rewrite README.md to lead with what Fab is
- [x] [wr07] 2026-02-08: DEV-976 Make fab/config.yaml self-documenting with inline comments explaining every field
- [x] [gqs1] 2026-02-08: DEV-977 Fix broken links in README.md and fab/specs/overview.md
- [x] [74eh] 2026-02-08: DEV-978 Standardize skill invocation syntax in specs and docs — `/fab-xxx` dash syntax
- [x] [xeti] 2026-02-08: DEV-979 Reduce overly broad permissions in `.claude/settings.local.json`
- [x] [wr10] 2026-02-10: DEV-980 Add Documentation Map to README with audience-specific reading paths
- [x] [mdfz] 2026-02-07: DEV-1006 Add specs index for pre-implementation intent — `fab/specs/` with proper indexing
- [x] [ogvt] 2026-02-08: DEV-1007 Branch integration in fab-switch
- [x] [n4j0] 2026-02-10: DEV-990 Add fab update script (`fab-update.sh`) for .kit distribution from central repo
- [x] [qurg] 2026-02-10: DEV-991 Document the SRAD framework — standalone `fab/specs/srad.md`
- [x] [6j7w] 2026-02-10: DEV-995 Simplify pipeline stages (7→6), removed "plan" stage
- [x] [gs42] 2026-02-10: DEV-996 Add attribution / owner for every change - in .status.yaml
- [x] [1rq4] 2026-02-12: DEV-1010 fab-discuss , fab-new should quickly create brief. Refinement of brief should be what we keep doing later in discuss (resolved by v5p2)
- [x] [maqp] 2026-02-12: DEV-1013 the two different modes of fab-discuss is confusing. Should have just one way of functioning. (deprecated - fab-discuss removed in v5p2)

## Backlog

### Commands & Scripts

- [ ] [90g5] 2026-02-07: DEV-988 Add a constitution command that creates the constitution
- [ ] [jgt6] 2026-02-07: DEV-989 hydrate should distinguish between specs and docs. Ingestion = specs. Generation = docs
- [ ] [s1u9] 2026-02-09: DEV-984 Add `fab/.kit/scripts/fab-status-update.sh` — helper script for `.status.yaml` field updates
- [ ] [s2t4] 2026-02-09: DEV-985 Add `fab/.kit/scripts/fab-task-complete.sh` — marks tasks done in `tasks.md`
- [ ] [s4c8] 2026-02-09: DEV-986 Extract a `fab-changelog-insert.sh` script for archive hydration

### Documentation

- [ ] [wr01] 2026-02-08: DEV-992 Write an end-to-end "Your First Change" tutorial

### Cleanup & Hardening

- [ ] [wr04] 2026-02-08: DEV-987 Harden all shell scripts with proper error handling and safe variable expansion
- [ ] [alat] 2026-02-10: DEV-993 Scores don't change after clarify - clarification should ideally increase score
- [ ] [29xv] 2026-02-10: DEV-994 Scoring formula needs to be relooked at - scores are generally too high. Check historical scores from archive. Do an analysis of the reason, the methodology, and the best way get relevant scores that give a strong signal. Also feels strange: Scores don't change after clarify - clarification should ideally increase score
- [ ] [q33r] 2026-02-11: DEV-997 `execution-skills.md` covers `/fab-apply`, `/fab-review`, `/fab-archive` but not `/fab-status`. Either add a section or create a separate `informational-skills.md` doc.
- [ ] [ff2a] 2026-02-11: DEV-998 `fab-status.md` behavior section lists rendered fields but this list drifts as fields are added — it already doesn't mention `created_by`. Either maintain the enumeration or generalize to "renders all `.status.yaml` fields."
- [ ] [swy8] 2026-02-11: DEV-999 `/fab-new` has a "Key points" section after its `.status.yaml` yaml block explaining field semantics. `/fab-discuss` has no equivalent. Add a matching "Key points" section.
- [ ] [qnzx] 2026-02-11: DEV-1000 No skill in the pipeline checks or closes backlog items. `/fab-archive` should scan `fab/backlog.md` for related items and offer to mark them done.
- [x] [c9vt] 2026-02-12: DEV-1001 `fab/backlog.md` is committed to git but is a personal scratchpad. Fix: gitignore, symlink from main worktree, update worktree-init.sh.
- [ ] [bk1n] 2026-02-11: DEV-1002 Modify fab-ff. fab-ff takes you all the way to archive - but can stop at clarifications. fab-fff takes you to archive, but with auto-clarification (doesn't stop, a bit unsafe)
- [x] [emcb] 2026-02-12: DEV-1008 Clarify fab-setup responsibilities and initialize fab/design folder (merged with DEV-1009)
- [ ] [ni3o] 2026-02-12: DEV-1011 Capture more metrics like: time taken for every stage, tokens used per stage
- [x] [0r8e] 2026-02-12: DEV-1012 Format for capturing created_by is wrong. Try using github id. Its ok the assume availabilit of the gh command line, with the git fallback
- [ ] [a4bd] 2026-02-12: DEV-1014 rename a few more commands to fab-continue. fab-continue should be able to take it to the end (archive)
- [ ] [ipoe] 2026-02-12: DEV-1015 Whats the point of saving checklists - and only checklists - in a separate folder in the changes folder. Check if it can be moved to the root of changes folder (as a sibiling to brief.md)
- [ ] [egqa] 2026-02-12: DEV-1016 Add a fab-switch variant takes you back to the main branch - ie. the state with no active change. Check if such a flow already exists.
- [ ] [pr1u] 2026-02-12: BUG During fab-init, fab/changes/archive is created, but without a .gitkeep. Add a .gitkeep in archive folder. In fab-init, Next steps: /fab-new <description> — Start a new change from a description, /fab-hydrate <sources> — Hydrate docs from external sources. Here, give suggent for just "/fab-hydrate" also - the variant that hydrates docs from code analysis
- [ ] [akhp] 2026-02-12: Rename /fab-backfill to /fab-hydrate-design
- [ ] [y07z] 2026-02-12: Absorb fab-apply, fab-review and fab-archive also into fab-continue. Add a form in fab-continue to continue any specific stage - eg typing fab-continue spec should redo the move from brief to spec stage. This should improve DX as not developers only need to remember fewer commands - fab-continue and fab-clarify mainly
