## Done

- [x] [akhp] 2026-02-12: Rename /fab-backfill to /fab-hydrate-design (implemented by 260212-akhp-rename-fab-backfill)
- [x] [a4bd] 2026-02-12: DEV-1014 rename a few more commands to fab-continue. fab-continue should be able to take it to the end (archive). Absorb fab-apply, fab-review and fab-archive also into fab-continue. Add a form in fab-continue to continue any specific stage - eg typing fab-continue spec should redo the move from brief to spec stage. This should improve DX as not developers only need to remember fewer commands - fab-continue and fab-clarify mainly

## Backlog

### Commands & Scripts

- [ ] [jgt6] 2026-02-07: DEV-989 hydrate should distinguish between specs and docs. Ingestion = specs. Generation = docs
- [ ] [s1u9] 2026-02-09: DEV-984 Add `fab/.kit/scripts/fab-status-update.sh` — helper script for `.status.yaml` field updates
- [ ] [s2t4] 2026-02-09: DEV-985 Add `fab/.kit/scripts/fab-task-complete.sh` — marks tasks done in `tasks.md`
- [ ] [s4c8] 2026-02-09: DEV-986 Extract a `fab-changelog-insert.sh` script for archive hydration

### Documentation

- [ ] [wr01] 2026-02-08: DEV-992 Write an end-to-end "Your First Change" tutorial

### Cleanup & Hardening

- [ ] [wr04] 2026-02-08: DEV-987 Harden all shell scripts with proper error handling and safe variable expansion
- [ ] [q33r] 2026-02-11: DEV-997 `execution-skills.md` covers `/fab-continue` (apply/review/archive) but not `/fab-status`. Either add a section or create a separate `informational-skills.md` doc.
- [ ] [ff2a] 2026-02-11: DEV-998 `fab-status.md` behavior section lists rendered fields but this list drifts as fields are added — it already doesn't mention `created_by`. Either maintain the enumeration or generalize to "renders all `.status.yaml` fields."
- [ ] [swy8] 2026-02-11: DEV-999 `/fab-new` has a "Key points" section after its `.status.yaml` yaml block explaining field semantics. `/fab-discuss` has no equivalent. Add a matching "Key points" section.
- [ ] [qnzx] 2026-02-11: DEV-1000 No skill in the pipeline checks or closes backlog items. `/fab-archive` should scan `fab/backlog.md` for related items and offer to mark them done.
- [ ] [ni3o] 2026-02-12: DEV-1011 Capture more metrics like: confidence level at each stage, time taken for every stage, tokens used per stage, time that we switched to a stage. this is 2D data. Was continue used or ff or fff used to move forard. Was clarify done. Confidence score before and after it - would need a table.
- [ ] [pr1u] 2026-02-12: DEV-1017 BUG During fab-init, fab/changes/archive is created, but without a .gitkeep. Add a .gitkeep in archive folder. In fab-init, Next steps: /fab-new <description> — Start a new change from a description, /fab-hydrate <sources> — Hydrate docs from external sources. Here, give suggent for just "/fab-hydrate" also - the variant that hydrates docs from code analysis
