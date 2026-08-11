# fab-fff
Runs the full pipeline apply → review → hydrate → ship → review-pr in one invocation — a wrapper over the shared `_pipeline.md` bracket that additionally owns the ship/review-pr steps, dispatched with explicit folder-name arguments.
## Flow
```
User invokes /fab-fff [change-name] [--force]
├─ Read: _preamble.md, helpers incl. _pipeline.md
├─ Execute the _pipeline.md bracket ({driver}=fab-fff, {terminal}=review-pr)
├─ Ship: dispatch /git-pr {name} (own ship transitions)
└─ Review-PR: dispatch /git-pr-review {name}
   ├─ [success / no-reviews] stage done; [failure] STOP with the error
   └─ [timeout] stage left active; report pending + re-run guidance
```
### Tools used
Read (`_preamble.md`, helpers); all other tool use lives in the bracket and the dispatched skills.
### Sub-agents
Bracket sub-agents per `SPEC-_pipeline.md` (/fab-continue Apply, Review, Hydrate) plus /git-pr and /git-pr-review.
