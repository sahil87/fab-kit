# Intake: CLI Dispatch Skill Wiring (3d)

**Change**: 260702-aetz-cli-dispatch-skill-wiring
**Created**: 2026-07-02

## Origin

Created via promptless dispatch (no user interaction — `{questioning-mode} = promptless-defer`) from a synthesized description out of the driving conversation. This is change **3d — the final change** of the cross-harness dispatch series:

| # | Change | What shipped | Status |
|---|--------|-------------|--------|
| 3a | `260702-y022` | `fab status refresh` (pull-based state recompute) + artifact-write hook removal | shipped (#457) |
| 3b | `260702-24ec` | Per-tier `spawn_command` → the optional `spawn=` line on `fab resolve-agent` | shipped (#459) |
| 3c | `260702-6sgj` | `fab dispatch` runtime (start/status/logs/kill/clean) + `docs/specs/harness-adapters.md` (the contract spec) | shipped (#461) |
| 3d | **this change** | Wire `fab dispatch` into the skill-side dispatch seam | — |

> Wire the CLI dispatch adapter (`fab dispatch`) into the skill-side dispatch seam. The contract is **fixed** by `docs/specs/harness-adapters.md` — 3d is wiring-only: it implements against that spec and MUST NOT redefine it; if wiring reveals a contract flaw, the fix is an explicit spec amendment reviewed as a contract change, never a silent redefinition inside skill files. No Go changes expected — this change edits markdown skills (`src/kit/skills/*.md`), their SPEC mirrors, aggregate specs, and memory.

## Why

1. **The pain point**: 3b lets a tier opt into CLI dispatch (`agent.tiers.<tier>.spawn_command` → the `spawn=` line) and 3c ships the runtime that runs it (`fab dispatch`) — but the dispatch sites in the skills still consume only `model=`/`effort=` and always dispatch via the Claude Code Agent tool. `_preamble.md` says so explicitly: the `spawn=` line "is for the cross-harness dispatch follow-ups (3c/3d), not read here." Until the seam reads it, a configured `spawn_command` is dead config — it resolves, then is silently ignored.
2. **If not fixed**: the cross-harness capability the series exists for (e.g. a claude orchestrator handing `apply` to `codex exec`, or a stage running headless on a CI box) never activates; worse, a user who configures a tier `spawn_command` gets a silent no-op — the opt-in appears to work (the line resolves) while every stage still dispatches natively.
3. **Why this approach**: the contract was deliberately fixed once in `docs/specs/harness-adapters.md` (authored by 3c) precisely so this change can be pure wiring against a single authority — no protocol co-definition split across changes, no silent drift. The wiring branches at the existing single `fab resolve-agent <stage> --alias` call per dispatch site, so the native path stays byte-identical when `spawn=` is absent.

## What Changes

### 1. `_preamble.md` — the canonical CLI-adapter dispatch contract

`src/kit/skills/_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution is the canonical dispatch contract; the sites (`_pipeline.md`, `fab-continue.md`) reference it. Extend it with the CLI-adapter branch:

- **Branch on `spawn=` presence** from the single existing `fab resolve-agent <stage> --alias` call (line 313 today: "Dispatch-seam skills consume `model=`/`effort=` only; `spawn=` is for the cross-harness dispatch follow-ups (3c/3d), not read here" — this change makes the seam read it). Absence of `spawn=` ⇒ native Agent-tool dispatch, unchanged. There is **NO fallback** to `agent.spawn_command` (decided in 3b/3c). The choice is per-stage/per-tier: one pipeline run can mix native and CLI dispatches across stages.
- **The CLI-adapter dispatch procedure** (new canonical subsection, referenced by all sites):
  1. `fab dispatch start <change> <stage>` with the full stage prompt on **stdin** — the same block prompt the Agent tool would get, adapted per the dispatch-prompt obligations (§2 below). `start` resolves the tier `spawn_command` internally and launches it detached. No `--timeout` in v1 (orphan detection + `fab dispatch kill` exist).
  2. Poll `fab dispatch status <change> <stage>` with `sleep 30` between polls until a terminal state.
  3. Five-state handling:
     - `running` → keep polling.
     - `done` → read `.fab-dispatch/{4-char-id}/{stage}-result.yaml` as the block's returned result and proceed with the normal sequencer transition (finish/fail per verdict). A review verdict `fail` inside a `done` result is a **review outcome**, not a dispatch failure.
     - `failed` → infrastructure/worker failure (NOT a review-verdict fail): surface `fab dispatch logs <change> <stage> --tail N` and stop per the stage's failure path.
     - `failed (no-result)` → contract violation; **NEVER treat as done** — surface logs and stop.
     - `orphaned` → surface and stop with re-run guidance (`fab dispatch start` over a completed/orphaned attempt overwrites).
- **Model/effort seams under CLI dispatch**: the `spawn=` command ALWAYS embeds the FULL model ID and the substituted effort (via `internal/spawn` — even under `--alias`), so the Agent-tool seams (the `model` alias param + the imperative effort prompt line) do NOT apply on the CLI path — the profile rides the spawn command itself. Sites keep the single `--alias` call and branch; no second resolve call.
- **Compliance visibility extension**: each dispatch site MUST surface the resolved `spawn=` line alongside the existing `model=`/`effort=` surfacing, so a CLI dispatch (or a `spawn=` line resolved but not honored) is visible in orchestrator output rather than silent. This extends the existing compliance-visibility rule in § Per-Stage Model Resolution.
- **No cleanup wiring**: `.fab-dispatch/` is transient comms with NO automatic GC (fixed in 3c — archive-time deletion + explicit `fab dispatch clean` only). The wiring adds no cleanup calls after a `done` dispatch.

### 2. Dispatch-prompt obligations (bind BOTH adapters — spec-fixed rules, 3d writes the content)

Per `docs/specs/harness-adapters.md` § Dispatch-prompt obligations, every dispatched stage prompt — native Agent-tool AND CLI — MUST:

1. **Instruct the worker to write `{stage}-result.yaml`** — for the CLI adapter a real file at `.fab-dispatch/{4-char-change-id}/{stage}-result.yaml`; for the native adapter the structural equivalent is the returned result. The content schema is 3d's to define (§3 below); the spec fixes only the path (CLI) and the presence obligation (both).
2. **Carry the standard subagent context files** — `fab/project/config.yaml`, `fab/project/constitution.md`, optional `context.md`/`code-quality.md`/`code-review.md` (`_preamble.md` § Standard Subagent Context). Already true for native prompts; the CLI prompt content must carry the same instruction — a worker on a fresh harness has no other awareness of project principles.
3. **End with a post-stage `fab status refresh` epilogue** so the worker recomputes state from artifacts after finishing (the 3a pull-based recompute).

**Block-contract reconciliation** (forced by obligation 3): the universal block contract line "do NOT run `fab status` commands; return results only" (in `_pipeline.md` Behavior dispatch note and `fab-continue.md` Step 1 dispatch contract) must be refined to prohibit *transition* commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`) while REQUIRING the terminal `fab status refresh` — refresh is a pull-based recompute, not a transition; the orchestrator still owns all transitions. Exact phrasing decided at apply; the semantics are fixed as stated. Sweep every occurrence of the old line (it appears at multiple dispatch sites).

### 3. `{stage}-result.yaml` — minimal schema (3d-defined)

Minimal YAML mirroring the native block's per-stage return contract. Common envelope + stage-specific fields:

```yaml
# apply (mirrors: "returns completion status or failure with task ID and reason")
stage: apply
status: success            # success | failure  (worker-level outcome)
summary: "12/12 tasks complete, tests green"
# on failure only:
failed_task: T007
reason: "tests failing in internal/x after 3 attempts"
```

```yaml
# review (mirrors: "merged prioritized findings + pass/fail verdict")
stage: review
status: success            # the review RAN to completion (infrastructure outcome)
verdict: pass              # pass | fail  (the review verdict — distinct from status)
findings:
  must_fix: []             # list of strings, each self-contained (file/line refs inline)
  should_fix:
    - "src/x.md:41 — stale claim Y"
  nice_to_have: []
summary: "2 should-fix, verdict pass"
```

```yaml
# hydrate (mirrors: "returns completion status")
stage: hydrate
status: success
summary: "updated docs/memory/runtime/dispatch.md, regenerated indexes"
```

The `status` vs `verdict` split is load-bearing: a completed review with verdict `fail` is dispatch-state `done` (result present) — the orchestrator then takes the normal review-fail path. Dispatch-state `failed` is reserved for worker/infrastructure failure.

### 4. `_pipeline.md` — dispatch sites

Update every dispatch site to branch per the §1 contract (reference `_preamble.md`; don't restate the five-state machine):

- The **Behavior dispatch note** (the `_preamble.md` § Subagent Dispatch pointer + universal block contract line — carve-out per §2)
- The **Per-stage model resolution note** (surface `spawn=` alongside `model=`/`effort=`; branch on its presence)
- **Step 1 (apply)**, **Step 2 (review)**, **Step 3 (hydrate)** dispatches
- **Auto-Rework Loop items 3–4** (re-dispatch apply, fresh re-review)

### 5. `fab-continue.md` — dispatch sites

- **Normal Flow Step 1** sub-agent dispatch contract (the one-stage sequencer): branch on `spawn=`, surface it, five-state handling by reference, block-contract carve-out
- **Stage table rows** (`apply`, `review`, `hydrate` — `intake`/`ready` row's apply sequencer included)
- **Review Behavior nested-reviewer resolution** (the block resolving `fab resolve-agent review --alias` for its own nested sub-agents — on a CLI-dispatched review worker this resolution happens *inside* the worker where sub-agent support may be absent; see §6)

### 6. `_review.md` — nesting degradation

`review` is the one nesting stage (inward + outward reviewers + merge, § Shared Review Dispatch). Per the spec: on a harness WITH sub-agent support those run as parallel sub-agents; on a harness WITHOUT sub-agent support the worker runs the parts **sequentially inline in one context**. Only the concurrency degrades — the outcome contract (same merged findings + verdict) is identical. Placement (both, per graded assumption #10): a canonical degradation note in `_review.md` § Shared Review Dispatch, AND the degradation instruction carried in the review dispatch prompt on the CLI path (the worker may be a harness that never reads fab's skill files beyond the prompt).

### 7. SPEC mirrors, aggregate specs, and the stale-pointer sweep

Sweep class (`fab/project/code-quality.md` § Sibling & Mirror Sweeps — must-fix if missed):

- **SPEC mirrors** (same change, constitution-required): `docs/specs/skills/SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, `SPEC-_review.md` — one per edited skill
- **Aggregate specs** restating dispatch facts: `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md`
- **`docs/specs/stage-models.md`** stale forward pointers: line ~145 "*the dispatch that RUNS the command (`fab dispatch`) and the skill dispatch-seam wiring are separate follow-up changes (3c/3d)*" and line ~284 "*v1 emits the line only; the dispatch that RUNS it (`fab dispatch`) and the skill wiring are separate follow-ups (3c/3d)*" — both now shipped/landed; repoint to `harness-adapters.md` + this change
- **`docs/specs/harness-adapters.md`**: wiring-only conformance — do NOT edit it except to mark the 3d wiring as landed if its § Skill wiring section's tense warrants it; any *semantic* edit is a contract amendment and out of scope (would be an explicit, separately-reviewed spec change)
- **Grep the old claims repo-wide before finishing apply**: `"not read here"`, `"follow-ups (3c/3d)"`, `"3c/3d"`, `"3d"` — update every occurrence in the class (per-file Affected-Memory lists under-cover cross-cutting `_shared/` prose)

### 8. Memory updates

See Affected Memory. Notably `docs/memory/_shared/context-loading.md` line ~121 carries the same "Consuming the `spawn=` line into an actual cross-harness dispatch is 3c/3d's job — this change only *emits* it; the dispatch-seam skills that inject model/effort do not read `spawn=`" claim, and `docs/memory/runtime/dispatch.md` carries "its content is 3d's business" / "against which the 3d skill wiring conforms" forward references — both go stale the moment this wiring lands.

### Fixed decisions (not re-opened here)

- **No fallback** from a tier without `spawn_command` to `agent.spawn_command` (3b/3c decision)
- **No automatic GC** of `.fab-dispatch/`; cleanup = archive-time deletion + explicit `fab dispatch clean` (3c decision)
- **Contract authority** = `docs/specs/harness-adapters.md`; amendments are explicit spec changes only
- **claude-orchestrator v1**: the orchestrator side remains Claude Code; CLI dispatch is for the *worker* side

## Affected Memory

- `pipeline/execution-skills.md`: (modify) the sequencer/block dispatch contract — add the CLI-adapter branch, the `spawn=` surfacing, and the refined block-contract line (transition prohibition + required `fab status refresh` epilogue)
- `_shared/context-loading.md`: (modify) § Per-Stage Model Resolution — replace the stale "3c/3d's job … do not read `spawn=`" claim with the wired behavior (branch on `spawn=`; CLI path bypasses the alias/effort-prompt seams)
- `runtime/dispatch.md`: (modify) resolve the "content is 3d's business" forward reference — record the `{stage}-result.yaml` schema and that the skill wiring now consumes the five states
- `pipeline/hooks-may-enhance-never-own.md`: (modify) the dispatch protocol's worker-side `fab status refresh` epilogue is now written into the dispatch prompts — note the prompt epilogue (not a hook) as the protocol-owned step, per the spec's hooks-enhance-never-own rule

## Impact

- **Files edited (markdown only, no Go, no migrations, no template changes)**:
  - Skills: `src/kit/skills/_preamble.md`, `_pipeline.md`, `fab-continue.md`, `_review.md`
  - SPEC mirrors: `docs/specs/skills/SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, `SPEC-_review.md`
  - Aggregate specs: `docs/specs/skills.md`, `glossary.md`, `architecture.md`, `stage-models.md`
  - Memory: the four Affected Memory files (+ regenerated indexes via `fab memory-index`)
- **Runtime dependencies (all shipped)**: `fab resolve-agent` `spawn=` line (3b), `fab dispatch` family (3c), `fab status refresh` (3a)
- **Behavioral risk**: native-path regressions from the block-contract rewording — the carve-out must not loosen the "orchestrator owns all transitions" invariant; when no tier carries a `spawn_command`, dispatch behavior must remain functionally identical
- **Review risk**: sweep-class misses (SPEC mirrors, aggregate specs, stale 3c/3d pointers) — the project's most common rework cause; grep before finishing apply

## Open Questions

None asked — promptless dispatch. All decision points graded ≥ 20 composite (no Unresolved rows; the five explicitly-gradable points from the driving conversation are rows 7–11 below).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Implement strictly against `docs/specs/harness-adapters.md`; any contract flaw found during wiring → explicit spec amendment reviewed as a contract change, never a silent redefinition in skill files | Spec § Skill wiring + driving description both fix this | S:95 R:90 A:95 D:95 |
| 2 | Certain | Scope is wiring-only markdown: skills + SPEC mirrors + aggregate specs + memory; no Go changes, no migrations | Description states "No Go changes expected"; 3c shipped the full runtime | S:90 R:85 A:95 D:90 |
| 3 | Certain | Branch on `spawn=` presence from the single existing `fab resolve-agent <stage> --alias` call; absence ⇒ native dispatch unchanged; NO fallback to `agent.spawn_command`; per-stage native/CLI mixing allowed | Decided in 3b/3c and restated in the description | S:95 R:85 A:95 D:95 |
| 4 | Certain | On the CLI path the Agent-tool model/effort seams do not apply — the `spawn=` command always embeds the full model ID + substituted effort even under `--alias` | 3b's documented `--alias` semantics (`_cli-fab.md` § fab resolve-agent) | S:90 R:85 A:95 D:90 |
| 5 | Certain | Five-state handling: `done` → read result + normal transition; `failed` → surface logs, stop per stage failure path; `failed (no-result)` → never done, surface + stop; `orphaned` → surface + stop with re-run guidance | Spec five-state machine + description's explicit per-state handling | S:95 R:85 A:90 D:90 |
| 6 | Certain | No cleanup calls after a `done` dispatch — `.fab-dispatch/` cleanup stays archive-time + explicit `clean` only | Fixed in 3c; description says do not re-open | S:95 R:90 A:95 D:95 |
| 7 | Confident | Poll cadence: `sleep 30` between `fab dispatch status` polls (fixed, no backoff in v1) | Description delegates "a sane cadence"; stages run minutes-long, 30s is responsive without spam; trivially tunable prose | S:60 R:90 A:80 D:65 |
| 8 | Confident | No `--timeout` passed to `fab dispatch start` in v1 | Description leans no-timeout explicitly — orphan detection + `fab dispatch kill` cover the failure modes | S:65 R:90 A:75 D:75 |
| 9 | Confident | `{stage}-result.yaml` schema as proposed in § What Changes 3: common `stage`/`status`/`summary` envelope + `failed_task`/`reason` (apply failure) + `verdict`/`findings{must_fix,should_fix,nice_to_have}` (review); `status` (worker outcome) is distinct from `verdict` (review outcome) | Schema is explicitly 3d's to define; mirrors the native blocks' documented return contracts; minimal YAML per the description | S:70 R:70 A:80 D:70 |
| 10 | Confident | Nesting-degradation placement: BOTH a canonical note in `_review.md` § Shared Review Dispatch AND the degradation instruction carried in the CLI-path review dispatch prompt | Description offers "and/or"; a cross-harness worker may never read fab skill files, so the prompt must carry it; `_review.md` stays the canonical home | S:60 R:90 A:75 D:60 |
| 11 | Confident | Block-contract carve-out semantics: prohibit `fab status` *transition* commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`), REQUIRE the terminal `fab status refresh`; exact phrasing decided at apply | Semantics fixed by the description + spec obligation 3; only wording is open, and it is easily revised prose | S:70 R:80 A:85 D:75 |
| 12 | Certain | Canonical placement: the CLI-adapter dispatch procedure lives in `_preamble.md` § Subagent Dispatch (beside Per-Stage Model Resolution); `_pipeline.md`/`fab-continue.md` sites reference it rather than restating the five-state machine | Mirrors the existing canonical-contract pattern the seam already uses | S:75 R:85 A:90 D:85 |
| 13 | Certain | The dispatch-prompt obligations (result instruction, context files, `fab status refresh` epilogue) are written into BOTH adapters' prompts — native included | Spec § Dispatch-prompt obligations: "Whatever adapter dispatches a stage, the prompt handed to the worker MUST …" | S:90 R:80 A:90 D:90 |

13 assumptions (8 certain, 5 confident, 0 tentative, 0 unresolved).
