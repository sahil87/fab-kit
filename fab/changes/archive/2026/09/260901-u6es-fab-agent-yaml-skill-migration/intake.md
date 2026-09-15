# Intake: fab agent YAML Skill Migration

**Change**: 260901-u6es-fab-agent-yaml-skill-migration
**Created**: 2026-09-01

## Origin

> /fab-fff u6es — backlog row: `[u6es] 2026-09-01: fab agent unification Change 3/4: skill/spec/memory prose sweep migrating dispatch sites to fab agent <stage> -o yaml per fab/plans/sahil/26-09-01-fab-agent-unification.md § Change 3 -- sweep-heavy (sibling-sweep class), FULL lane, review-critical. depends on Change 2 (cites its shipped YAML schema).`

Sources of record, in authority order:

1. **The plan doc** `fab/plans/sahil/26-09-01-fab-agent-unification.md` § Change 3 (written 2026-09-01 from a /fab-discuss design session). §§ Changes 1–2 (`77vz` PR #635, `mp8d` PR #636) are **both MERGED to main** (Change 2 merged 2026-09-01 15:10 UTC as `47f03d73`, this branch's base) — the dependency is satisfied. § Change 4 (backlog `0i4x`, labelled-rung choreography) is a strictly-ordered successor and OUT of scope here.
2. **The shipped `-o yaml` schema** — authoritative in `src/kit/skills/_cli-fab.md` § fab agent (the `-o, --output yaml` bullet's key table), backed by the parity/golden tests from #636 (`cmd/fab/agent_surface_test.go`, `cmd/fab/resolve_agent_test.go`, `internal/agent/resolution.go`).

## Why

The resolution *engine* is already shared (`internal/agent`, unified by Changes 1–2), but the skill prose still teaches two CLI surfaces: `fab agent` for humans/operators and `fab resolve-agent <stage> --alias` for pipeline dispatch sites. Migrating the dispatch sites onto `fab agent <stage> -o yaml` completes the single-launcher-surface goal:

1. **One surface to document and evolve.** Every future resolution capability lands once, on `fab agent`, instead of twice.
2. **Structured output beats line-scraping.** The YAML carries `model_alias` natively (no `--alias` flag), labelled `dispatch.rung` (cashed in by Change 4), and `source` provenance — the ordered-lines contract cannot grow any of these without breaking byte-stability.
3. **Unblocks retirement.** `fab resolve-agent` stays working and untouched (frozen contract), but once no skill prose instructs running it, a later post-release-window change can convert it to a thin alias or delete it.

If we don't do this, Change 4 has no consumer surface to branch on (`rung:` exists only in the YAML), and the deprecation can never start.

## What Changes

**Zero behavior change by design.** This is a 1:1 semantic migration of *instructions in prose*: every consumer keeps the exact same choreography, reading the same facts from a different surface. `fab resolve-agent`'s flags, arguments, and byte-stable output are NOT touched (plan-doc non-goal, load-bearing). `fab agent`'s Go surface is NOT extended — Change 2 already shipped everything this sweep cites.

### The 1:1 mapping (applies at every dispatch site)

| Today (resolve-agent lines) | After (fab agent YAML) |
|-----------------------------|------------------------|
| Run `fab resolve-agent <stage> --alias` | Run `fab agent <stage> -o yaml` |
| Branch on `dispatch=` **line presence** | Branch on `dispatch:` **key presence** (key absent ⇔ native rung — same rule) |
| Agent-tool `model` parameter ← the `model=` alias (via `--alias`) | ← the `model_alias` key (always emitted for Claude IDs; empty for non-Claude — then use `model` or inherit) |
| Effort instruction ← the `effort=` line | ← the `effort` key |
| Execute the `dispatch=` value's command — never | Execute the `dispatch.command` value — still never; `fab dispatch` remains the executor |
| Surface the resolved `model=/effort=/provider=/dispatch=` lines | Surface the resolved YAML (at minimum `provider`/`model`/`model_alias`/`effort` and `dispatch:` presence); an all-empty resolution is still a flag-don't-dispatch signal |

**Choreography is deliberately unchanged** (Change 4's job, not this one): the YAML's `dispatch.rung` is labelled, but consumers do NOT branch on it yet — `_preamble.md` § CLI-Adapter Dispatch keeps the "attempt `start` first and let its refusal discriminate" discovery, the pane readiness gate, the wait/recovery machine, reap timing, worker continuation, and profile fixity all verbatim. Where prose currently *explains* the start-probe by "the `dispatch=` line is unlabelled", reword honestly: the label now exists in the YAML but the choreography does not consume it until Change 4 — do not delete the rationale, re-anchor it.

Reference resolution from this repo (dispatch.mode=pane, workers=codex), for workers to sanity-check the mapping:

```yaml
# fab agent apply -o yaml   (run from source: cd src/go/fab && go run ./cmd/fab agent apply -o yaml)
selector: apply
kind: stage
role: doing
provider: codex
model: gpt-5.6-sol
effort: xhigh
command: codex --dangerously-bypass-approvals-and-sandbox -m gpt-5.6-sol -c model_reasoning_effort=xhigh
model_alias: ""
template: codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}
fill_mode: template
source:
    provider: agent.workers
    model: providers.codex.profiles.default
    effort: providers.codex.profiles.doing
dispatch:
    rung: pane
    command: codex --dangerously-bypass-approvals-and-sandbox -m gpt-5.6-sol -c model_reasoning_effort=xhigh
```

> **Verification note**: the installed Homebrew `fab` 2.23.9 predates #636 and rejects `-o` — verify any `-o yaml` claim against the worktree source (`go run ./cmd/fab …` from `src/go/fab`), not the installed binary (the bottle-predates-source trap).

### Skill dispatch sites (`src/kit/skills/`)

Verified occurrence counts 2026-09-01 (re-grep at apply — counts are for sweep-completeness checking, not a contract):

- **`_preamble.md`** (9) — § Per-Stage Model Resolution: the seam table ("using `fab resolve-agent <stage> --alias`"), the user-override table (rides "the same single `fab resolve-agent <stage> --alias` call"), the surfacing rule (`model=`/`effort=`/... lines → YAML keys), the `--alias` sentence ("emits the Agent-tool-valid short alias directly" → `model_alias` is native to the YAML); § Worker Continuation: profile-fixity mentions ×2 ("`fab resolve-agent apply --alias` is NOT re-run on the resume path"); § CLI-Adapter Dispatch: the branch table (`dispatch=` absent/present → `dispatch:` key absent/present), the "unlabelled" rationale re-anchor described above; § pane-mode bullets referencing `fab resolve-agent` in error guidance.
- **`_pipeline.md`** (4) — § Stage Dispatch Procedure items 1 and 5 (the per-stage resolve + surface instruction), § Light Lane ("no dispatch, no `fab resolve-agent`" mentions), Step 3 hydrate framing.
- **`fab-continue.md`** (8) — its stage-dispatch instructions per stage (same resolve + surface pattern).
- **`fab-fff.md`** (3) — Steps 4/5 ship + review-pr resolve instructions and the Behavior-note mention.
- **`fab-proceed.md`** (3) — its delegation prose citing the resolver.
- **`_cli-agents.md`** (2) — operator-facing references to the resolver surface.
- **`_cli-fab.md`** (22) — two distinct treatments: (a) § fab resolve-agent gains a **deprecation banner** pointing at § fab agent (the command's own reference documentation otherwise STAYS — it documents a working command; do not delete its ladder/fill-precedence content if § fab agent doesn't already own it — relocate-then-point where needed); (b) every *instructional* mention elsewhere in the file (e.g., § fab dispatch's "re-run `fab resolve-agent`" guidance, the operator-launcher exception "resolves without `--alias`") rewords against the YAML surface.

`fab-adopt.md` currently greps clean but is named by the plan's sweep list — re-check at apply (it consumes `_pipeline.md`'s loop by pointer, so it may genuinely carry no token).

### Specs (`docs/specs/`)

- **`stage-models.md`** (28) — the design owner. § Skill wiring rewrites onto `fab agent <stage> -o yaml`; the resolver sections keep documenting the shared engine but present the YAML surface as the skill-consumed one and resolve-agent as the deprecated alias surface.
- **`harness-adapters.md`** (7) — the dispatch-contract references to the branch discriminator and resolve step.
- **`skills.md`** (5), **`glossary.md`** (4), **`architecture.md`** (2), **`config.md`** (2), **`index.md`** (1) — aggregate restatements (the known sibling-sweep class).

### Memory (`docs/memory/`) — present-truth updates

Content files only; `log.md`/`log.seed.md` rows are **history and stay verbatim** (FKF present-truth applies to claims about the present, not to dated log entries). Counts as of 2026-09-01: `runtime/providers-and-profiles.md` (40 — the big one), `_shared/context-loading.md` (12), `_shared/configuration.md` (9), `pipeline/execution-skills.md` (8), `runtime/dispatch.md` (3), `distribution/kit-architecture.md` (3), `runtime/agent-primitives.md` (2), `pipeline/issue-linking.md` (2), `distribution/migrations.md` (1), `runtime/index.md` (1), `runtime/operator.md` (1). Regenerate indexes (`fab memory-index`) after edits.

### User-facing Go string literals (bounded — the one Go touch)

Per the standing sweep lesson (behavior-claim sweeps must include user-facing STRING LITERALS), two Go strings instruct callers to use the old surface and migrate with the prose:

1. `cmd/fab/dispatch_start.go:532` — the native-mode error: "re-run `fab resolve-agent %s --alias` and dispatch natively when the `dispatch=` line is absent" → reword to `fab agent %s -o yaml` / "when the `dispatch:` key is absent". Update any test pinning this string (`dispatch_start_test.go`, `dispatch_restart_test.go`).
2. `internal/configref/configref.go:770` — the generated config-fence line "# per stage or role by `fab resolve-agent <stage|role>`" → `fab agent <stage|role> -o yaml`. Update pinned tests (`config_show_init_test.go`, `noproject_config_test.go` if they assert the fence text). Note this changes `fab config upgrade` fence output — comment-only, no migration file needed (fences regenerate on upgrade by design).

Everything else in Go stays: engine comments about resolve-agent (`internal/dispatch/*`, `internal/agent/*`, `setupcheck/probes.go`), the parity/golden tests (they PIN the frozen contract — untouched), and `cmd/fab/skill.md`'s CLI enumeration (it lists commands that exist; at most add `fab agent` alongside — judgment at apply). `*_test.go` **comments** that describe *skills consuming* resolve-agent output forms migrate; test assertions never do.

### Contrastive-phrase sweep classes

Grep beyond the bare token before finishing apply (the top rework cause in this repo): `resolve-agent`, `resolve_agent`, `--alias`, `dispatch=`, "`model=` line", "model=/effort=", "line presence", "byte-stable lines" (where it claims skills consume them), "ordered lines". Sweep `src/kit/skills/`, `docs/specs/`, `docs/memory/` (content files), and user-facing Go string literals. Historical artifacts (`fab/changes/archive/`, log files, `docs/findings/`, plan docs under `fab/plans/`) are records and stay.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) resolver surface presented as `fab agent <stage> -o yaml`; resolve-agent repositioned as deprecated alias; fill/ladder facts unchanged
- `runtime/dispatch`: (modify) dispatch-site resolve step + branch discriminator wording
- `runtime/agent-primitives`: (modify) launcher-surface mentions
- `runtime/operator`: (modify) operator-launcher exception wording vs the YAML surface
- `runtime/index`: (modify) regenerated index line
- `pipeline/execution-skills`: (modify) stage-dispatch procedure references
- `pipeline/issue-linking`: (modify) incidental resolver mentions
- `_shared/context-loading`: (modify) always-load / dispatch-seam references
- `_shared/configuration`: (modify) config-surface references to the resolver
- `distribution/kit-architecture`: (modify) surface enumeration mention
- `distribution/migrations`: (modify) incidental mention

## Impact

- ~7 skill files, ~7 spec files, ~11 memory content files, 2 Go string literals + their pinned tests. No `fab resolve-agent` surface/output change (golden tests must stay green untouched). No choreography change. No new config, states, or schema.
- Kit + binary version together via kit releases, so the prose cutover is atomic per version; `fab resolve-agent` keeps working for out-of-band users and muscle memory.
- Review posture: sweep-heavy — review treats missed occurrences as must-fix (`documentation_accuracy` + `cross_references`). Budget the contrastive-phrase sweep before finishing apply, not after review flags it.
- Go test runs scoped to `cmd/fab` + `internal/configref` (the string-literal touches); the full-suite parity/golden tests are the no-regression guard.

## Open Questions

*(none — the plan doc § Change 3 plus the shipped schema settle the design; residual judgment calls are graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Schema authority is the shipped `-o yaml` key table in `_cli-fab.md` § fab agent (from merged #636); no schema invention here | Plan doc: "3's prose cites 2's shipped YAML schema"; #636 merged as 47f03d73 | S:95 R:90 A:95 D:95 |
| 2 | Certain | `fab resolve-agent` stays working and byte-identical; deprecation is a docs-only banner; deletion is a later post-window change | Plan-doc non-goal, stated twice; golden tests pin it | S:95 R:85 A:95 D:95 |
| 3 | Certain | Choreography unchanged: consumers branch on `dispatch:` key presence only; `rung:` is not consumed (Change 4's job); start-probe discovery, gate, reap, continuation, profile fixity all verbatim | Plan doc: "1:1 in semantics … No choreography changes in this change" | S:90 R:80 A:90 D:90 |
| 4 | Confident | The two user-facing Go string literals (dispatch_start.go:532 error hint, configref.go:770 config-fence line) migrate with the prose, plus their pinned tests — despite the plan's "zero Go" shape hint | Standing repo lesson: behavior-claim sweeps must include user-facing string literals; both strings instruct callers to run the old surface; shape hints describe effort, not scope law | S:70 R:75 A:80 D:70 |
| 5 | Certain | `log.md`/`log.seed.md` rows, archived changes, `docs/findings/`, and `fab/plans/` are historical records — excluded from the sweep | FKF present-truth governs claims about the present; rewriting dated history is falsification | S:75 R:85 A:85 D:80 |
| 6 | Confident | The surfacing rule rewords to "surface the resolved YAML — at minimum `provider`/`model`/`model_alias`/`effort` and `dispatch:` presence"; all-empty still flags | Direct transliteration of the existing rule's intent; exact field list is editorial | S:65 R:85 A:80 D:70 |
| 7 | Confident | `_cli-fab.md` § fab resolve-agent's reference content survives under the banner (relocate-then-point only where § fab agent already owns the fact) | Owner-or-pointer rule; the command still works and needs docs until deletion | S:65 R:80 A:80 D:70 |
| 8 | Confident | `cmd/fab/skill.md`'s CLI enumeration keeps its resolve-agent mention (it enumerates existing commands); at most gains `fab agent` alongside — final call at apply after reading its context | Enumeration vs instruction is contextual; low blast radius either way | S:50 R:80 A:55 D:45 |
| 9 | Certain | change_type is `refactor` (1:1 semantic migration, no new capability, no fix) | Matches mp8d precedent (#636 shipped as refactor); override explicitly if inference differs | S:70 R:90 A:80 D:75 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
