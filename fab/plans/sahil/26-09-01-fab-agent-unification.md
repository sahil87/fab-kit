# fab agent unification — one launcher surface, resolve-agent retirement ladder

> Plan doc — written 2026-09-01 from a /fab-discuss design session. Design sketch
> (flag surface + worked examples, using the real `defaults.yaml` grammars):
> https://claude.ai/code/artifact/9746628c-4f34-4aef-85ca-df2d74575b85
> Code anchors verified 2026-09-01 against `cmd/fab/agent.go`,
> `cmd/fab/resolve_agent.go`, `internal/agent/agent.go`, `internal/spawn/spawn.go`
> on branch `scarlet-hedgehog` (main-equivalent) — re-verify file/line claims
> before implementing. Three strictly ordered changes + one optional follow-up;
> each ships independently and each is a rollback point.

## Goal

Make `fab agent` the single human-and-machine launcher surface — one selector
grammar (role or stage), print/exec/template modes, and a structured `-o yaml`
output — then migrate skill prose off `fab resolve-agent`, retiring it to a
deprecated alias. The resolution *engine* is already shared internally; this
plan unifies the CLI surface without ever mutating `resolve-agent`'s frozen
line-output contract.

## Current state (verified 2026-09-01)

- **`fab agent`** (`cmd/fab/agent.go:79` — `Use: "agent [role] [-- <agent-args>...]"`):
  exec-by-default via `sh -c` (no TTY guard — document-don't-validate), `--print`,
  `--repo`, `--workers`, `--` passthrough. Two mutually exclusive addressing modes:
  `[role]` positional (fills apply) XOR `--provider` (deliberate fill BYPASS —
  profile is exactly the passed flags; `--model`/`--effort` valid only alongside
  `--provider`). Runs outside a fab project (config-only cascade). Documented at
  `_cli-fab.md` § fab agent (~line 1415).
- **`fab agent --print` is already machine-consumed** — it is NOT purely
  human-facing: the operator spawn path (`_cli-agents.md:46` "never hand-assembled…
  Ask `fab agent` to compose it", `:52-58`) and cross-repo spawning
  (`_cli-external.md:166`, `fab agent --print --repo <target>`) treat its output
  as a contract. Byte-stability of `--print` is a standing constraint on every
  change below.
- **`fab resolve-agent`** (`cmd/fab/resolve_agent.go`): pure query; ordered
  byte-stable lines `model=` / `effort=` / `provider=` / `dispatch=` (each
  optional line omitted when empty). `--alias` maps Claude IDs to Agent-tool
  short aliases (`model=` line only; `dispatch=` always embeds the full ID).
  `dispatch=` presence/absence IS the branch discriminator for the CLI-adapter
  path, and the line is **unlabelled** — orchestrators cannot tell pane from
  headless from the line, hence `_preamble.md`'s "attempt `start` first and let
  its refusal be the discriminator" choreography. Bare `--model`/`--effort` are
  valid here (documented deliberate asymmetry with `fab agent`,
  `_cli-fab.md:348`).
- **Shared engine already exists**: both commands resolve through
  `internal/agent` (`ResolveRole` :437, `ResolveRoleWith` :464, `Resolve(stage)`
  :498, `ResolveProvider` :609, `Profile` :224) and compose via
  `spawn.WithProfile` (`internal/spawn/spawn.go:94` — template mode vs append
  mode; empty-value token-drop). The duplication is CLI-surface-only.
- **Stage→role map**: `internal/agent/agent.go:304` (`stageRoles`). The two
  name collisions (`review`, `hydrate`) are fixed points, guarded by
  `TestStageRoleCollisionsAreFixedPoints` — stage-as-selector on `fab agent` is
  benign by construction.
- **Template substitution sites** (consumers, unchanged by this plan):
  `cmd/fab/agent.go:168,266`, `cmd/fab/resolve_agent.go:158`,
  `cmd/fab/dispatch_start.go:393`, `cmd/fab/pane_open.go:92`,
  `cmd/fab/operator.go:185`, `cmd/fab/batch.go:34`. Raw templates live in
  `providers.<name>.interactive_command`/`headless_command`
  (`src/go/fab/defaults.yaml:110-172`); today the raw form is shown ONLY by
  `fab config show` — no resolution-side command prints it.

## Change 1 — `fab agent` surface extension (additive)

Rewrite `fab agent`'s surface to the sketch, shaped so nothing existing changes
behavior.

**New capabilities:**

1. **Stage selectors**: the positional accepts a stage as well as a role,
   mapped through `stageRoles`. Output/`-o yaml` reports both
   (`stage apply → role doing`). Reuse `internal/agent.Resolve(cfg, stage)`;
   selector kind detection = try role set, then stage set (collisions are fixed
   points, so order is immaterial).
2. **`role + --provider` becomes legal** (today: usage error) with
   re-resolve-fills semantics — the role's profile refills from the named
   provider's fills (`ResolveRoleWith` + provider pin). **Bare `--provider`
   (no role) keeps today's documented fill-bypass semantics untouched**
   (`_cli-fab.md:1443`, `_cli-agents.md:67` — "spawn a provider whose model IDs
   you don't know" stays working verbatim). New capability where there was an
   error ⇒ zero behavior change.
3. **`-t, --template`**: print the provider's command template unsubstituted —
   a pipeline tap BEFORE the fill step, not a third sink. Implies print-mode;
   combines with `--provider`/`--headless` (they pick which template); rejects
   `--model`/`--effort` with a usage error (they feed a step that never runs —
   reject, don't ignore).
4. **`--headless`**: resolve `headless_command` instead of
   `interactive_command` (print/template modes; exec of a headless command is
   a usage error). Missing capability = hard error naming the config key —
   never a silent descent (matches `fab dispatch open`'s posture).
5. **`-o yaml`**: structured output of the full resolution (schema fixed in
   Change 2 — this change may ship it minimal or defer the flag entirely to
   Change 2; implementer's call, note it in the intake).

**Constraints (must-not-break):**

- `--print` output stays **byte-identical** for every invocation legal today
  (operator spawn seam). Golden tests before refactor.
- Bare `fab agent` keeps today's behavior: exec the default role. The sketch's
  picker is OUT of this change (see Open decisions).
- `--` passthrough, `--repo`, `--workers`, project-free cascade, exec-via-sh -c,
  no-TTY-guard posture: all unchanged.

**Files**: `cmd/fab/agent.go` (+ tests), possibly a small hoist in
`internal/agent` for selector-kind detection; docs: `_cli-fab.md` § fab agent
(surface + the asymmetry note at :348 gains the role+provider case),
`_cli-agents.md` §§ around :46-67 (new template/stage forms available to the
operator), `docs/specs/stage-models.md` (mention the launcher's new selector
grammar). Sibling sweep: grep `fab agent` repo-wide — the "mutually exclusive"
claim about role/`--provider` appears in at least `_cli-fab.md:1418`/`:348` and
`_cli-agents.md:63` and must be updated everywhere in the same change.

## Change 2 — parity by construction (NO resolve-agent surface change)

Do **not** modify `fab resolve-agent`'s flags, arguments, or output. Instead:

1. **Extract one resolution-result struct** (working name
   `agent.Resolution`): `selector`, `kind` (role|stage), `role`, `provider`,
   `model` (full ID), `model_alias` (Agent-tool alias, empty for non-Claude),
   `effort`, `template`, `fill_mode` (template|append), `command`
   (substituted), `dispatch` (nil, or `{rung: pane|headless, command}`), and
   `source` provenance (which config tier / fill rung supplied each value).
   Both commands' renderers become projections of this struct:
   `resolve-agent`'s ordered lines, and `fab agent -o yaml`'s full serialization.
2. **`-o yaml` schema lands here** (if deferred from Change 1). Key semantics:
   `dispatch:` key absent ⇔ native rung (preserves today's branch-on-presence
   rule for migrating consumers), but its `rung:` field is **labelled** —
   resolve-agent's `dispatch=` line cannot say which rung produced it; the YAML
   can, and Change 4 cashes that in. `model_alias` is always emitted for Claude
   IDs, making a separate `--alias` flag unnecessary on the YAML surface.
3. **Parity test matrix** asserting resolve-agent's lines ≡ the line-projection
   of the struct across: templated vs plain (append-mode) commands, empty
   fills (inherit signal), `--alias` on/off, non-Claude ID pass-through
   (`gpt-5` verbatim), dispatch present (pane rung, headless rung) and absent,
   dated Claude variants (`claude-haiku-4-5-20251001` → `haiku`). Plus golden
   tests pinning resolve-agent's exact bytes across the refactor.

**Acceptance**: `fab resolve-agent`'s output is byte-identical before/after for
the full matrix; the struct is the only place resolution results are composed
(no second composition site survives).

**Files**: `internal/agent` (struct + projections), `cmd/fab/resolve_agent.go`
and `cmd/fab/agent.go` (become renderers), tests. Docs: `_cli-fab.md` § fab
agent gains the `-o yaml` schema; § fab resolve-agent gains one line noting the
shared engine (no contract change).

## Change 3 — skill migration to `fab agent -o yaml`

The big sweep. Migrate every skill-prose dispatch site from
`fab resolve-agent <stage> --alias` to `fab agent <stage> -o yaml`, **1:1 in
semantics**: consumers keep branching on `dispatch:` key presence exactly as
they branched on the `dispatch=` line; the Agent-tool model parameter reads
`model_alias` instead of relying on `--alias`. No choreography changes in this
change (that's Change 4).

**Sweep list** (grep `resolve-agent` repo-wide; verified occurrences as of
2026-09-01 — re-grep at implementation):

- `src/kit/skills/_preamble.md` — § Per-Stage Model Resolution (the seam
  tables, override table, surfacing rule), § CLI-Adapter Dispatch (branch
  table), § Worker Continuation (profile-fixity mentions ×2)
- `src/kit/skills/_pipeline.md`, `fab-continue.md`, `fab-adopt.md` — dispatch
  sites and their surfacing obligations
- `src/kit/skills/_cli-fab.md` — § fab resolve-agent gains a deprecation
  banner pointing at § fab agent; the operator-launcher exception ("resolves
  without `--alias`") rewords against the YAML surface
- `docs/specs/stage-models.md` § Skill wiring (owner of the resolver design)
- `docs/memory/runtime/*` + `docs/memory/pipeline/*` files describing dispatch
  resolution (find via `fab memory-index` grep, not assumption)
- Aggregate specs `skills.md` / `glossary.md` / `architecture.md` — the known
  sibling-sweep class; grep the phrase classes (`resolve-agent`, `--alias`,
  `dispatch=`, "model= line"), not just the token
- `*_test.go` comments referencing resolve-agent output forms

**resolve-agent disposition**: keep it working and untouched, add the
deprecation note in docs only. Actual deletion (or thin-alias conversion) is a
LATER change after a release window — out of this plan's scope. Skills and
binary version together via kit releases, so the cutover is atomic per version;
the alias exists for out-of-band users and muscle memory.

**Risk note for the operator**: this is the sweep-heaviest change class in this
repo (top recurring rework cause — `fab/project/code-quality.md` § Sibling
Sweeps). Budget a full contrastive-phrase sweep before review, and expect
review to treat missed occurrences as must-fix.

## Change 4 (optional follow-up, backlog if not queued) — labelled-rung choreography

With `dispatch.rung` labelled in the YAML, `_preamble.md` § CLI-Adapter
Dispatch step 1's "the `dispatch=` line is unlabelled … attempt `start` first
and let its answer be the discriminator" collapses to a branch on `rung:` —
pane goes straight to `open → gate → deliver`, headless straight to `start`.
Keep `start`'s pane-refusal as defense-in-depth; the prose stops teaching it as
the discovery mechanism. Pure prose/choreography change, no Go. Separated from
Change 3 so the mechanical migration and the semantic simplification are
independently reviewable (and revertible).

## Sizing / lane hints

| Change | Shape | Lane hint |
|--------|-------|-----------|
| 1 | Go surface + tests + 3-4 doc files | FULL (Go + docs + sweep) |
| 2 | Internal refactor + golden/parity tests, minimal docs | FULL (touches both commands' composition paths) |
| 3 | Skill/spec/memory prose sweep, zero Go | FULL — sweep-heavy, review-critical |
| 4 | `_preamble` choreography prose only | LIGHT |

Strict order 1 → 2 → 3 (→ 4). Do not stack 3 on an unmerged 2 — 3's prose cites
2's shipped YAML schema.

## Open decisions (resolve at each change's intake)

1. **Output flag spelling**: user preference is `-o yaml`; repo precedent is
   boolean `--json` flags (`fab pane map --json`, `fab dispatch status --json`)
   while preflight emits bare YAML. Pick one at Change 1/2 intake and record it;
   `-o yaml|json` is the forward-compatible spelling.
2. **Picker**: bare `fab agent` today execs the default role (documented).
   Adding the sketch's interactive picker either breaks that (bare → picker) or
   ships behind a flag/`fab agent ?`. Deferred out of Change 1; take it up only
   on explicit request.
3. **Change 1 ships `-o yaml` minimal vs deferring it wholly to Change 2** —
   implementer's call; the schema authority is Change 2 either way.

## Non-goals

- **No mutation of `fab resolve-agent`'s surface or output, ever, in this
  plan** — parity comes from the shared struct, deprecation is docs-only,
  deletion is a separate post-window change.
- No change to the operator launcher's resolution path (`operator.go:185`,
  `WithProfile` direct) or to `fab pane open` / `fab batch` / `fab dispatch
  start` composition.
- No provider grammar validation — verbatim pass-through philosophy holds
  everywhere, including the new `-t` and `-o yaml` surfaces.
- No change to `spawn.WithProfile` semantics (template/append modes,
  empty-value token-drop).
- Bare `--provider` fill-bypass semantics are load-bearing and stay.
