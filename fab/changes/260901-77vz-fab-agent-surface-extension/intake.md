# Intake: fab agent Surface Extension

**Change**: 260901-77vz-fab-agent-surface-extension
**Created**: 2026-09-01

## Origin

> [77vz] 2026-09-01: fab agent unification Change 1/4: fab agent surface extension per fab/plans/sahil/26-09-01-fab-agent-unification.md § Change 1 -- stage selectors, role+--provider re-resolve, -t/--template, --headless, -o yaml. FULL lane. --print must stay byte-identical.

Backlog-ID invocation via `/fab-new 77vz` (one-shot). Two design sources carry the decisions:

1. **The plan doc** `fab/plans/sahil/26-09-01-fab-agent-unification.md` (commit `1b4f71b0`, written from a `/fab-discuss` design session 2026-09-01) — § Change 1 is this change; §§ Changes 2–4 are strictly-ordered successors (backlog `mp8d`/`u6es`/`0i4x`) and are OUT of scope here.
2. **The design sketch artifact** (linked from the plan header: https://claude.ai/code/artifact/9746628c-4f34-4aef-85ca-df2d74575b85, "The fab agent Command") — flag surface, worked examples, ranked design decisions. Where the sketch and the plan disagree, the **plan wins** (it was written after the sketch and records the final calls): the sketch's `--json` is superseded by the plan's `-o yaml` preference, and the sketch's bare-invocation picker is explicitly deferred out of Change 1.

All plan code anchors re-verified 2026-09-01 on this branch (`braided-wolverine`, main-equivalent at `1b4f71b0`): `cmd/fab/agent.go:79` (Use line), `internal/agent/agent.go` `stageRoles:304` / `ResolveRole:437` / `ResolveRoleWith:464` / `Resolve:498` / `ResolveProvider:609`, `internal/spawn/spawn.go` `WithProfile:94`, `_cli-fab.md:348`/`:1442`/`:1444`, `_cli-agents.md:63`.

Lane directive: **FULL lane** (backlog entry says so explicitly; Go + docs + sweep).

## Why

1. **The pain point**: launching/inspecting an agent is split across two commands with overlapping-but-different contracts — `fab agent` (roles, exec/print, `--model`/`--effort` only with `--provider`) and `fab resolve-agent` (stages or roles, ordered `model=`/`effort=`/`provider=`/`dispatch=` lines, bare `--model`/`--effort` legal). Users and skills must remember which command takes stages, which allows bare overrides, and which output form to parse. The resolution *engine* is already shared (`internal/agent` + `spawn.WithProfile`); the duplication is CLI-surface-only.
2. **If we don't fix it**: the two surfaces keep drifting (the documented "deliberate asymmetry" notes at `_cli-fab.md:348`/`:1442` exist only to explain the split), and the unification ladder (Changes 2–4: shared resolution struct, skill migration, labelled-rung choreography) has no surface to land on.
3. **Why this approach**: extend `fab agent` **additively** — every invocation legal today keeps byte-identical behavior; new capabilities occupy only today's error space. This makes Change 1 a safe rollback point and lets `fab resolve-agent` stay untouched (its `dispatch=`-line contract is frozen; parity comes later from Change 2's shared struct, retirement from Change 3's skill migration — never from mutating it).

## What Changes

All Go changes in `cmd/fab/agent.go` (+ tests), with a possible small hoist into `internal/agent` for selector-kind detection. `fab resolve-agent` is **not touched**.

### 1. Stage selectors on the positional

The `[role]` positional additionally accepts a **stage** name (`intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`), mapped through the fixed `stageRoles` map (`internal/agent/agent.go:304`) — reuse `internal/agent.Resolve(cfg, stage)`. Selector-kind detection: try the role set, then the stage set; the two colliding names (`review`, `hydrate`) are fixed points (`stageRoles[name] == name`, guarded by `TestStageRoleCollisionsAreFixedPoints`), so detection order is immaterial and the ambiguity is benign by construction.

```
$ fab agent apply          # stage apply → role doing → exec doing's session command
$ fab agent apply --print  # print it instead (bare command line, unchanged form)
```

Exec and `--print` output forms are **unchanged** — `--print` stays exactly the one bare command line (no stage→role annotation). The stage→role mapping is *reported* only on the structured `-o yaml` surface (`selector`/`kind`/`role` keys).

### 2. Selector + `--provider` becomes legal (re-resolve-fills semantics)

Today `[role]` and `--provider` are mutually exclusive (usage error, `agent.go:120-122`). That guard is removed: a selector (role or stage) combined with `--provider <name>` re-resolves the role's profile **from the named provider's own fills** — `agent.ResolveRoleWith(cfg, role, Overrides{Provider, ProviderSet: true})` (`agent.go:464`), which already implements exactly this (provider pin → refill from `providers.<p>.profiles.<role>` → `.default` → empty; an explicit `agent.profiles.<role>.model` pin still wins).

```
$ fab agent apply --provider kimi --print
kimi --auto -m k3        # model refilled from kimi's own fills, not patched into claude's command
```

**Bare `--provider` (no selector) keeps today's documented fill-BYPASS semantics untouched** — profile is exactly the passed flags, empty values follow `spawn.WithProfile`'s token-drop rule. The "spawn a provider whose model IDs you don't know" use case (`_cli-fab.md:1443`, `_cli-agents.md:67`) stays working verbatim. New capability where there was an error ⇒ zero behavior change.

### 3. `--model` / `--effort` become general post-refill overrides

Today they are a usage error without `--provider` (`agent.go:124-126`). They become **verbatim final overrides valid with any addressing form** — layered after role resolution / provider refill, exactly as `ResolveRoleWith`'s `o.ModelSet`/`o.EffortSet` already applies them and exactly as `fab resolve-agent` already allows them bare (this erases the documented asymmetry at `_cli-fab.md:348`/`:1442`):

```
$ fab agent review --model claude-sonnet-5 --effort max --print   # role profile, two fields overridden
$ fab agent --model claude-haiku-4-5 --print                      # default role + override
```

Bare-`--provider` mode's `--model`/`--effort` handling is unchanged (they were already legal there). No validation, no enum, no pair correction — verbatim pass-through throughout.

### 4. `-t, --template` — print the unsubstituted template

Prints the selected provider's command template with `{model}`/`{effort}` placeholders intact — a pipeline tap **before** the fill step, not a third sink. Implies print-mode (never execs). Combines with any selector, `--provider`, and `--headless` (they pick *which* template). Rejects `--model`/`--effort` with a usage error (they feed a step that never runs — reject, don't ignore):

```
$ fab agent apply -t
claude --permission-mode bypassPermissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}
$ fab agent apply --provider kimi -t
kimi --auto -m {model}
$ fab agent apply --model k3 -t
error: --model has no effect with --template (substitution never runs)
```

Today the raw template is shown only by `fab config show` — this gives the resolution surface its own window (debugging a mangled command, authoring a provider entry, downstream substitution).

### 5. `--headless` — resolve `headless_command` instead of `interactive_command`

Valid in the print-family modes only (`--print`, `-t`, `-o yaml`); **exec of a headless command is a usage error**. A provider with no `headless_command` is a hard error naming the config key (`providers.<name>.headless_command`) — never a silent descent to another command slot (matches `fab dispatch open`'s posture).

```
$ fab agent doing --headless --print
claude -p --permission-mode bypassPermissions --model claude-opus-5
```

### 6. `-o yaml` — structured resolution output (minimal in this change)

`-o, --output <format>` accepting exactly `yaml` (anything else is a usage error; `json` is addable later — the `-o <format>` spelling is the forward-compatible one, resolving the plan's open decision 1 per the recorded user preference over the sketch's `--json` and the repo's boolean `--json` precedent). Implies print-mode. Ships **minimal**: keys `selector`, `kind` (`role|stage`), `role`, `provider`, `model`, `effort`, `command` — resolving the plan's open decision 3 as "ship minimal in Change 1" (the backlog entry lists `-o yaml` in this change's scope). Change 2 (`mp8d`) remains the schema authority and extends additively (`model_alias`, `template`, `fill_mode`, `source`, `dispatch`); no `dispatch:` key is emitted in Change 1 (consistent with Change 2's absence ⇔ native rule, since this surface resolves session commands, not dispatch).

```
$ fab agent apply -o yaml
selector: apply
kind: stage
role: doing
provider: claude
model: claude-opus-5
effort: ""
command: claude --permission-mode bypassPermissions -n "$(basename "$(pwd)")" --model claude-opus-5
```

### 7. `-p` shorthand for `--print`

Additive alias from the sketch's synopsis; the long form and its output are untouched.

### Constraints (must-not-break)

- **`--print` output stays byte-identical for every invocation legal today.** It is machine-consumed: the operator spawn path (`_cli-agents.md:46-58`) and cross-repo spawning (`_cli-external.md:166`) treat it as a contract. **Golden tests pinning today's `--print` bytes are written BEFORE any refactor.**
- Bare `fab agent` keeps today's behavior: exec the default role. The sketch's interactive picker is OUT (plan open decision 2 — take up only on explicit request).
- Unchanged: `--` passthrough (shell-quoted append), `--repo`, `--workers` (env replace), project-free config cascade, exec via `sh -c`, no-TTY-guard posture, the shared `unknownProviderError` phrasing, `roleSessionCommand`'s empty-command → actionable-error contract.
- `fab resolve-agent`: zero changes to flags, arguments, or output (plan non-goal — frozen surface).
- No provider grammar validation anywhere, including the new `-t` and `-o yaml` surfaces; no change to `spawn.WithProfile` semantics.

### Documentation sweep (same change)

- `src/kit/skills/_cli-fab.md` § fab agent (~:1415-1466): new selector grammar, the now-legal combinations, `-t`/`--headless`/`-o yaml`/`-p`; the asymmetry notes at `:348` and `:1442` (bare `--model`/`--effort` asymmetry is ERASED by capability 3 — rewrite both sides) and the mutual-exclusion bullet at `:1444` (now describes only… nothing — replaced by re-resolve semantics).
- `src/kit/skills/_cli-agents.md` §§ ~:46-67: the `:63` "mutually exclusive" + "--model/--effort only valid alongside --provider" claims must be rewritten; new template/stage forms available to the operator.
- `docs/specs/stage-models.md`: mention the launcher's new selector grammar (human-curated spec — surgical edit).
- **Sibling sweep obligation**: grep `fab agent` and the phrase classes ("mutually exclusive", "require --provider", "asymmetry", "spawn-command") repo-wide — aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) and memory files restating the surface are in the sweep class even where this intake's Affected Memory list misses them (`fab/project/code-quality.md` § Sibling Sweeps; top rework cause).

## Affected Memory

- `runtime/providers-and-profiles`: (modify) — documents the `fab agent` surface, the role/`--provider` mutual exclusion, the `--model`/`--effort`-require-`--provider` rule, and the `fab resolve-agent` asymmetry; all four claims change.
- `runtime/agent-primitives`: (modify) — the `_cli-agents` helper memory mirrors the `:63` mutual-exclusion/override claims and the spawn-composition guidance; gains the stage/template forms.

(Index regen via `fab memory-index` at hydrate; the apply-time sweep may pull additional memory files that restate the surface — e.g. `runtime/operator.md`'s `fab agent --print` composition claims are expected to survive unchanged since `--print` bytes are frozen, but verify.)

## Impact

- **Go**: `cmd/fab/agent.go` (surface rework: Args validator, guard removal, new flags, selector-kind detection, `-o yaml` renderer) + `cmd/fab/agent_test.go` (golden `--print` tests first, then coverage for every new capability and every new usage error); possibly a small `internal/agent` hoist for selector-kind detection (role-set/stage-set probe). Existing consumers of `roleSessionCommand` (operator launcher) untouched.
- **Skills/docs**: `_cli-fab.md`, `_cli-agents.md`, `docs/specs/stage-models.md` (+ sweep-discovered occurrences). Constitution constraint: any `fab` CLI change MUST update `_cli-fab.md` and include test updates — both are in scope above.
- **Memory**: the two files listed in Affected Memory.
- **Downstream**: Changes 2–4 of the ladder build on this surface; nothing here blocks them, and nothing here may touch `fab resolve-agent` or the operator/pane/batch/dispatch-start composition paths (plan non-goals).
- **Release**: MINOR (new command surface).

## Open Questions

- None — the plan plus the design sketch resolve every decision; the two implementer-call decisions (open decisions 1 and 3) are resolved above and graded in Assumptions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Stage selectors reuse `internal/agent.Resolve` + `stageRoles`; detection = try role set then stage set; `review`/`hydrate` collisions are fixed points so order is immaterial | Plan § Change 1 item 1 specifies mechanism verbatim; guard test exists | S:90 R:85 A:95 D:95 |
| 2 | Certain | Selector+`--provider` = re-resolve-fills via `ResolveRoleWith` provider pin; bare `--provider` fill-bypass untouched | Plan § Change 1 item 2 explicit; engine already implements; today's error space only | S:90 R:80 A:95 D:90 |
| 3 | Confident | `--model`/`--effort` become post-refill overrides valid with ANY addressing form (not only with `--provider`), erasing the documented asymmetry with resolve-agent | Sketch § Overrides + worked example show it bare with a role; `ResolveRoleWith` already applies `ModelSet`/`EffortSet` after refill; Change 3's within-Claude override migration needs it; plan's capability list omits it (hence not Certain) | S:65 R:80 A:85 D:70 |
| 4 | Certain | `-t/--template` implies print, combines with selectors/`--provider`/`--headless`, rejects `--model`/`--effort` with a usage error | Plan item 3 + sketch worked examples give exact behavior incl. error text shape | S:90 R:85 A:90 D:90 |
| 5 | Certain | `--headless` valid only in print-family modes; exec = usage error; missing `headless_command` hard-errors naming the config key | Plan item 4 explicit (matches `fab dispatch open` posture) | S:90 R:85 A:90 D:90 |
| 6 | Confident | Ship `-o yaml` MINIMAL in Change 1 (keys: selector, kind, role, provider, model, effort, command; no dispatch/model_alias); Change 2 stays schema authority, extends additively | Plan open decision 3 delegates to implementer; backlog entry lists `-o yaml` in this change's scope; additive extension is safe | S:60 R:75 A:70 D:55 |
| 7 | Confident | Output-flag spelling `-o, --output <format>`, sole accepted value `yaml` (usage error otherwise); not boolean `--json` | Plan open decision 1 records user preference `-o yaml` and names it the forward-compatible spelling, superseding the sketch's `--json`; repo's `--json` precedent noted and rejected | S:70 R:80 A:75 D:65 |
| 8 | Confident | Add `-p` shorthand for `--print` | Sketch synopsis shows `-p/--print`; purely additive alias; plan silent | S:55 R:90 A:80 D:70 |
| 9 | Certain | Bare `fab agent` unchanged (exec default role); interactive picker OUT of this change | Plan constraint + open decision 2 defer it explicitly | S:95 R:90 A:95 D:95 |
| 10 | Confident | Selector matching stays case-sensitive (today's behavior); sketch's "case-insensitive" note deferred with the picker | Plan silent; smallest-surface reading of "nothing existing changes behavior"; trivially reversible | S:40 R:85 A:70 D:55 |
| 11 | Certain | `--print` byte-identity pinned by golden tests written before refactor; `--repo`/`--workers`/`--` passthrough/`sh -c` exec/no-TTY-guard all unchanged | Plan constraints section verbatim; operator seam is machine-consumed | S:95 R:60 A:95 D:95 |

11 assumptions (6 certain, 5 confident, 0 tentative, 0 unresolved).
