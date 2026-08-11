# fab-ff
Fast-forwards apply → review → hydrate in one invocation — a thin wrapper binding `{driver}=fab-ff`, `{terminal}=hydrate` over the shared `_pipeline.md` bracket. Accepts `--force`; confidence-gated at intake.
## Flow
```
User invokes /fab-ff [change-name] [--force]
├─ Read: _preamble.md, helpers incl. _pipeline.md
└─ Execute the _pipeline.md bracket ({driver}=fab-ff, {terminal}=hydrate) — see SPEC-_pipeline.md
```
### Tools used
Read (`_preamble.md`, helpers); all other tool use lives in the bracket (see `SPEC-_pipeline.md`).
### Sub-agents
Per the bracket: `/fab-continue` Apply, Review (via `_review.md`), Hydrate.
