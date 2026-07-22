# Reuse awareness — stop apply/review reinventing existing code

> Backlog detail doc — written 2026-07-22 after a `/fab-discuss` session on how the apply
> and review stages can know what already exists in a codebase, so agents stop rebuilding
> utilities the project already has. The work splits into two independent parts:
> **Part 1** is prose-only (skill/template/config changes — shippable on current fab-kit,
> no Go changes) and **Part 2** is `fab codemap`, a Go subsystem with a plugin architecture
> for per-language support. Part 2 supersedes Part 1's inventory *infrastructure* when it
> lands; Part 1's *process layer* (ledger, acceptance gate) survives both.

## Problem

`code-quality.md` names "duplicating existing utilities" as an anti-pattern, but no
mechanism gives apply or review an actual inventory to check against. Duplicate-detection
tools (jscpd and friends) are inherently **post-hoc** — they compare code that exists to
code that exists. Pre-coding detection is a different problem: comparing an *intent*
("I'm about to write a frontmatter parser") against existing code, before any code is
written.

Duplication happens at three granularities, and each is catchable exactly where the
information first exists:

| Granularity | Example | Detection point |
|---|---|---|
| Feature-level | the whole change already exists / is 80% built | intake |
| Function-level | the plan proposes a helper the codebase already has | apply entry (plan generation) |
| Line-level | literal/near-literal clones written during apply | review (+ jscpd-class tools) |

## Design decisions (resolved in the session)

1. **Function-level detection belongs at apply entry, not intake.** Intake deliberately
   captures the what/why; the concrete "I'll write function X" decisions crystallize when
   `plan.md`'s `## Tasks` are generated. Plan-generation time is still pre-coding —
   that's the seam. Pushing it into intake would force design work intake is supposed to
   defer and add noise to the one human-gated stage.
2. **Lexical/agentic search over vector retrieval.** Industry evidence (Claude Code
   itself ships no embedding index; agentic grep outperformed RAG in Anthropic's testing)
   says code search wants exact identifiers + iterative refinement. The misleading-name
   problem ("it's called `extractHeader`") is solved by *intent descriptions* — curated
   one-liners in Part 1, LLM-generated cards in Part 2 — which make lexical search over
   the descriptions sufficient. Embeddings are a back-pocket escalation, not a foundation.
3. **Evidence over discipline.** "Please search before creating" is soft and skippable.
   The reuse ledger makes search evidence a *required plan artifact* that review can
   verify — converting discipline into a checkable mechanism.
4. **Part 2 is plugin-shaped from commit one; the marketplace comes later.** The plugin
   seam (manifest + WASM grammar + query file) is first-cut work by necessity — the
   alternative (CGo bindings, hardcoded language switch) is a worse architecture that
   would need ripping out. Distribution machinery (install command, registry) is
   genuinely deferrable.
5. **WASM grammars via wazero keep the core binary pure Go.** Native tree-sitter bindings
   mean CGo, which poisons cross-compiled Homebrew bottles and single-static-binary
   builds. Runtime-loaded WASM grammars (wazero is pure Go) make the plugin architecture
   *simplify* the core build rather than complicate it.
6. **CLI, not MCP.** Fab agents already shell out to `fab score`/`fab resolve`/`fab
   preflight`; `fab codemap search` joins that grammar. No server lifecycle, no
   per-project client config, works identically for claude/codex/gemini workers.

---

## Part 1 — Prose-only (current fab-kit: skills + templates + config)

Everything routes through one artifact: a **curated utilities inventory** in
`docs/memory/`. Every other piece consults it. All changes are markdown/prompt/config —
constitution-clean, no Go.

### Phase 1 — the artifact

- **Curated utilities inventory**: a memory domain (or per-domain `utilities.md` files)
  listing reusable functions with **one-line intent descriptions** — what problem each
  solves, not signatures. Context-budget-critical: one line per utility, ruthlessly
  curated; the moment it becomes API documentation it stops being loadable wholesale and
  the scheme reverts to on-demand search.
- **Backfill pass**: a `docs-hydrate-memory`-style sweep over `source_paths` generates
  the initial inventory. Without this, hydrate-maintained coverage starts at ~0% and
  agents trust an inventory that lists 10% of the codebase.
- **Hydrate extension**: an explicit rule in hydrate behavior — "record new reusable
  helpers introduced; remove deleted ones." Must be a concrete skill instruction, not a
  hope.
- **Search-before-create discipline**: ~3 lines in apply behavior — before writing any
  new helper, grep for existing implementations by concept keywords. Zero maintenance;
  it is the *fallback for inventory gaps*, and what makes the ledger's "nothing found"
  claim meaningful.

### Phase 2 — the consumers

- **Inventory-primed planning**: load the inventory before plan/task generation
  (Affected-Memory-style walk, or always-load if it stays small). Prevention-by-context:
  an agent that already saw `resolveChange()` won't propose writing one. Highest-leverage
  consumer.
- **Conditional reuse ledger in `plan.md`**: for every *new* function/helper a task
  proposes, record either "reuses `existing.Foo`" or "no existing fit — searched X, Y, Z;
  closest is `Bar` but lacks Q." **Conditional** = generated only when the plan proposes
  new symbols; an unconditional section degrades into boilerplate "N/A" that trains
  everyone to ignore it.

### Phase 3 — the verifiers (cheap, config-mostly)

- **`reuse` acceptance category** via the existing `checklist.extra_categories` config +
  a review checklist entry ("ledger complete; no new helper duplicates an existing
  utility"). It verifies the ledger — cut the ledger, cut this too.
- **Optional intake one-liner** ("closest existing capability: X") inside the existing
  Affected Memory walk output — only if Phase 1–2 experience shows whole-change
  redundancy actually occurs. Deliberately deferred from v1: rare payoff, and intake is
  where generation noise costs most.
- **Optional jscpd post-apply `stage_hooks` hook** as the line-level canary.

### Known failure mode

Single point of failure by design: five mechanisms, one source of truth. If the inventory
under-covers, priming silently doesn't prime, the ledger's "nothing found" is
unfalsifiable, and review verifies against an incomplete list — all layers degrade
*together and silently*. Mitigations: periodic backfill re-sweeps; write the maintenance
expectation into the inventory's own index. (Part 2 dissolves this failure mode — its
index is complete by construction.)

### Surfaces

`src/kit/skills/` (hydrate + apply behavior in `fab-continue`/`_generation`, plan
template, `docs-hydrate-memory`) with the full SPEC-mirror sweep per constitution;
project config for the acceptance category. Phases 1+2 can be one change; 3 is
separable.

---

## Part 2 — `fab codemap` (Go subsystem + language plugins)

A derived code index inside the existing `fab` binary. Solves Part 1's three structural
weaknesses — coverage (complete by construction), staleness (incremental reindex, even
mid-apply so task 3's new functions are visible when planning task 7), curation labor
(automated) — at the cost of fab's biggest Go surface to date.

### Core commands

- **`fab codemap build`** — walks `source_paths` (config already scopes it), parses via
  tree-sitter, emits one record per exported symbol: name, kind, signature, doc comment,
  file:line, content hash. Incremental by file hash. Store: SQLite or JSONL under a
  gitignored `.fab-cache/` — derived data, never committed.
- **`fab codemap annotate`** — batches un-carded symbols (≈50/prompt) through the
  configured provider's `dispatch_command` (the `providers:` table fab already has) to
  generate **one-line intent cards**, cached by content hash. No API key, no embedding
  service, no new dependency — fab borrows the agent CLI the user already runs the
  pipeline with. The pivotal move: semantic quality of curation with the coverage and
  freshness of generation.
- **`fab codemap search <query>`** — BM25-ish lexical matching over name + signature +
  doc + card, top-k as markdown. The cards solve the misleading-name problem
  (`extractHeader`'s card *says* "parses YAML frontmatter"), so lexical is sufficient.
  No vectors in v1; an embeddings flag only if lexical over cards demonstrably misses.
- **`fab codemap check`** — diffs the branch, extracts *new* symbols, searches each
  against the pre-existing index, reports suspected near-duplicates (name/signature/card
  similarity; optionally shells to jscpd for literal clones if installed). Runs
  mid-apply, not just at review.

### Plugin architecture (per-language support, marketplace-ready)

**Tier 1 — declarative grammar packs (default, ~90% case).** A plugin is pure data:

```
fab-codemap-go/
  manifest.yaml     # name, language, extensions, grammar ABI version, schema version
  grammar.wasm      # tree-sitter grammar compiled to WASM
  tags.scm          # tree-sitter query: what is a symbol, where its doc comment lives
```

Core embeds **wazero** and owns the pipeline: extension → pack, parse via WASM grammar,
run `tags.scm` captures, emit the JSONL schema. Reuse win: upstream tree-sitter already
ships `tags.scm` code-navigation queries for most mainstream languages — per-language
work is mostly packaging. Security: data-only packs execute inside the WASM sandbox —
near-zero marketplace attack surface.

**Tier 2 — exec extractors (escape hatch).** An executable
(`fab-codemap-<lang>`, PATH or plugin-dir discovery, git-subcommand pattern) that emits
the same symbol JSONL on stdout. Covers what tree-sitter can't: LSP-grade accuracy
(`gopls`), proprietary DSLs. Requires trust (checksums, install-time warning).

**The contract core owns**: the symbol JSONL schema (`schema_version`) + the manifest
format (grammar ABI pinning; core refuses incompatible packs loudly — cheap on day one,
painful to retrofit). Languages are plugins; extraction *semantics* are core.

**Distribution** (maps onto existing fab machinery):
`~/.fab-kit/plugins/codemap/<name>/<version>/` (sibling of the versions cache);
registry-as-git-repo of manifests (brew-tap model), artifacts fetched by pinned checksum;
`codemap.languages: [go, python]` in project config so `build` can prompt for missing
packs. A rendered index on shll.ai is the marketplace v1; a web UI is cosmetic later.

**First cut**: the seam (extractor interface, manifest format, schema, wazero runtime,
dispatch) + 2–3 packs **embedded via `go:embed` in the exact plugin layout** as
"preinstalled plugins" — every code path exercised from day one.
`fab codemap plugin install` (fetch the same bundle from a URL instead of the embed FS)
is v2; the marketplace registry is v3. No rework at any step.

### Pipeline wiring

- **Freshness**: `build` + `annotate` at apply entry (`stage_hooks.apply.pre` or baked
  into apply behavior). Incremental → seconds.
- **Plan entry**: planner runs `codemap search` per planned area (top-k cards as
  priming); the reuse ledger records the actual search command + output — falsifiable
  evidence.
- **Apply**: search-before-create becomes one concrete line: "run `fab codemap search`
  before writing any new helper."
- **Review**: runs `codemap check`, verifies the ledger, gated by the `reuse` acceptance
  category.
- **Hydrate**: optionally promotes genuinely load-bearing utilities into a small curated
  "greatest hits" memory file — machines query the index; humans read the map.

### What Part 2 supersedes vs. what survives

| Part 1 piece | Fate under Part 2 |
|---|---|
| Curated inventory as infrastructure | **Superseded** — derived index is complete, always fresh |
| Backfill pass | **Superseded** — first `build && annotate` *is* the backfill |
| Search-before-create (grep) | **Subsumed** — becomes `fab codemap search` |
| Intake adjacent-capabilities note | **Subsumed** — redundancy surfaces at plan-time search |
| Reuse ledger | **Survives** — records the judgment call no index makes |
| `reuse` acceptance category | **Survives** — review gate over the ledger + `check` report |
| Hydrate "greatest hits" promotion | **Survives** — the human-readable map |

### Phasing

1. **v1**: `build` + `search` over raw signatures/docs (no cards yet) + plan/apply wiring
   + reuse ledger. Most duplication is findable lexically already.
2. **v2**: `annotate` via the provider seam (the quality jump) + `plugin install`.
3. **v3**: `check` + review wiring; marketplace registry; embeddings flag only on
   evidence.

### Caveats

- WASM parsing is ~2–5× slower than native tree-sitter — irrelevant for incremental,
  hash-gated indexing; only the first full build on a huge repo notices.
- Upstream `tags.scm` quality varies per language; some packs need query fixes (exactly
  the contribution shape a marketplace absorbs well).
- First-annotate cost on big repos is real but one-time; batch aggressively.
- **Constitutional**: this departs from Pure Prompt Play (Constitution I) enough to
  amend it explicitly ("the fab binary may embed analysis tooling; workflow *logic*
  remains markdown") rather than lawyer around it. All CLI-change obligations apply in
  force: tests, `_cli-fab.md`, SPEC mirrors.
- Registry governance parked, not solved: one blessed repo, PRs welcome (brew-tap
  answer) is enough for a long time.

---

## Sequencing across the parts

Part 1 is shippable now and cheap; Part 2 is the destination. They are not wasted
against each other: Part 1's process layer (ledger, acceptance category, hydrate
promotion) is exactly the wiring Part 2 plugs into — building Part 1 first means Part 2
lands as a drop-in replacement for the inventory's *source*, not a redesign of the
pipeline seams.

## Success metric

Review findings of the "duplicates existing utility" class trend to zero. If a jscpd
post-apply hook is wired, it is the canary: if it keeps firing, the pre-coding layers
aren't working — escalate (cards quality, then embeddings) rather than adding more
process.
