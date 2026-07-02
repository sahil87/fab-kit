# Intake: Agent Config v3 — Providers & Role Tiers

**Change**: 260702-tykw-agent-providers-role-tiers
**Created**: 2026-07-02

## Origin

> agent config v3 — providers + role tiers: split provider config out of agent config (new `providers:` section with per-provider `session_command`/`dispatch_command`; claude explicit, native dispatch = absent dispatch_command), replace the thinking/doing/fast tier vocabulary with five role tiers (default, operator, doing, review, fast — thinking dissolves since intake never dispatches; review split out for author/critic separation), tiers become {provider, model, effort} with per-field inheritance from `default` (provider written explicitly on all lines as style), retire the `review_tools` config block (copilot/codex/claude toggles move to a code-review.md § Review Tools section; fix the configref.go vs _review.md contradiction), rename tier spawn_command → dispatch_command and agent.spawn_command → providers.*.session_command, add `fab agent [tier] [--print] [--repo <path>]` to launch (or print) the resolved session command in the current shell — retiring `fab spawn-command`. Operator tier default sonnet/medium. Ships with migration(s); likely splits into a series at planning.

Conversational origin: designed end-to-end in a `/fab-discuss` session (2026-07-02) that walked the agent-config surface key by key. The design went through several explicit user decisions and counter-proposals (recorded in the Assumptions table): the user proposed extracting a providers section ("WAY more understandable"), proposed the role-tier roster, chose the doing profile (opus/xhigh), asked for provider written on every tier line, ratified no-inference-from-model-strings and document-don't-validate, and proposed `fab agent --print` as the replacement for `fab spawn-command`. Two agent counter-proposals were accepted by the user continuing on them: two command fields per provider instead of one merged `command` (session vs dispatch grammars differ), and dropping the `thinking` tier (5 tiers, not 6) because intake never dispatches.

## Why

Three problems, all surfaced by walking fab-kit's own `fab/project/config.yaml`:

1. **The agent config conflates provider mechanics with role/budget policy, and the current names actively confuse.** Two different fields share the name `spawn_command` (`agent.spawn_command` opens interactive *sessions*; `agent.tiers.*.spawn_command` runs one headless *stage task*) — fab-kit's own config needed a 7-line comment (added in PR #465) just to explain their independence. The tier vocabulary (`thinking`/`doing`/`fast`) names cognitive modes whose referents are hidden: `thinking` nominally governs intake + review, but intake never dispatches (it is pre-boundary, foreground), so in every real dispatch `thinking` *means* review — a fact even the kit's author had to re-derive. Provider grammar (`codex exec -m {model} …`) is embedded inline per tier, so a second provider means repeating command templates.
2. **Dead and self-contradictory review-tool config.** fab-kit's `review_tools: {claude: true, codex: true, copilot: true}` block is a no-op (absent key defaults everything to true). Worse, the repo contradicts itself about the codex/claude keys: `configref.go:143-147` (the generated config reference) says they are "legacy keys… the pre-ship review-stage cascade is not configurable here," while `_review.md:127-142` documents the outward-reviewer cascade as controlled by exactly those keys. One side is wrong today. The toggles are review *policy* (which external critic reviews the diff) and belong in the discoverable prose file `code-review.md`, not in an obscure config block.
3. **No way to launch the configured agent, and the printable command is profile-less.** There is no command that starts the default agent in the current shell (the workaround is `eval "$(fab spawn-command)"`). Worse, `fab spawn-command` prints the raw session command with *no tier profile injected* — templated `{model}`/`{effort}` placeholders are stripped (leak prevention), so operator-spawned workers launch without any model/effort from the tier system.

If we don't fix it: every config reader keeps paying the two-spawn_command / hidden-referent decoding tax; the configref/`_review.md` contradiction ships to every project via `fab config reference`; multi-provider setups (the whole point of the harness-adapters work) stay awkward to configure; and worker sessions keep launching profile-less.

Why this approach: extraction (providers) + renaming to roles (tiers) attacks the confusion at its source rather than adding more comments. The role-tier vocabulary gives every tier a concrete referent a user can point at ("the operator", "the reviewer") — directly serving the stated goal of saving people from the confusion of many pipeline stages. Rejected alternatives are recorded per-decision in Assumptions (e.g., merging session/dispatch into one `command` — the grammars are different subcommands of the binary and one template cannot express both; folding `agent.spawn_command` in as a `default` tier command — implies fallback semantics that were deliberately rejected in the 3a–3d series).

## What Changes

### 1. New top-level `providers:` section

Provider invocation grammar moves out of the tiers into a named table. Each provider MAY carry two command fields:

```yaml
providers:
    claude:
        session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")"'
        # no dispatch_command → this provider's stages dispatch natively via the Agent tool
    codex:
        session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
        dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
```

Semantics:

- **`session_command`** — opens an interactive agent session (the current `agent.spawn_command` semantics, relocated). Consumed by `fab operator`, `fab batch`, and the new `fab agent`.
- **`dispatch_command`** — runs one headless stage task via `fab dispatch` (the current per-tier `spawn_command` semantics, relocated and renamed). **Absent `dispatch_command` = native Agent-tool dispatch** (the existing absence signal, unchanged). In practice native dispatch only works for claude-family models (the Agent tool's model enum) — but fab does not validate this; a misconfig fails loudly at dispatch time, per the existing no-validation contract.
- The two fields are **not merged into one `command`**: session and dispatch are different invocations of the same binary (claude: `claude … -n <name>` interactive vs `claude -p` headless; codex: `codex` TUI vs `codex exec`), and no single template expresses both.
- `{model}`/`{effort}` placeholder substitution keeps its existing one-source-many-sinks contract (`internal/spawn`, reused): the tier declares the values, the provider command declares where they land. `WithProfile`'s grammar-forgiving append (Claude-style flags appended when no placeholder present) carries over for `session_command`.
- **The claude provider is explicit and shipped as the built-in default** (not "nothing needed"): fab-kit's built-in provider table contains `claude` with the default session command (`claude --dangerously-skip-permissions`, profile appended) and no dispatch_command (native). A project overrides or extends via its own `providers:` block, per-field merged over the built-in.
- **Provider names are opaque user-chosen strings.** fab never infers a provider from a model string (`claude-*` → claude would require a provider registry, which the no-validation/provider-neutrality contract refuses). The documented rule for the one footgun: *override `model` cross-provider ⇒ override `provider` too* — fab documents this in the config reference, it does not validate it.
- **No fallback between the two command fields**, preserving the 3a–3d decision: absence of `dispatch_command` signals native dispatch, never "use session_command".

### 2. Tier vocabulary: `thinking`/`doing`/`fast` → five role tiers

`agent.tiers` keys become **`default`, `operator`, `doing`, `review`, `fast`** — roles with concrete referents, replacing cognitive modes. Tier values become `{provider, model, effort}` (commands live in `providers:`, not on tiers).

| Tier | Governs | Built-in default profile |
|------|---------|--------------------------|
| `default` | Spawned worker sessions (`fab batch`), `fab agent` with no tier arg, `fab agent --print`; intake (advisory only — it runs foreground in the user's own session, which fab cannot re-model); **per-field fallback for every other tier** | claude / `claude-fable-5` / `xhigh` |
| `operator` | The operator coordinator session (`fab operator` launcher — which today *borrows* the doing tier; this names that seam properly) | claude / `claude-sonnet-5` / `medium` |
| `doing` | apply, review-pr, hydrate (unchanged membership) | claude / `claude-opus-4-8` / `xhigh` |
| `review` | review (split out — author/critic separation: people want a different agent checking the work than doing it) | claude / `claude-fable-5` / `xhigh` |
| `fast` | ship (unchanged) | claude / `claude-sonnet-5` / `low` |

- **`thinking` is removed, not split.** With review split out, `thinking`'s only remaining stage would be intake — which never dispatches (pre-boundary). Intake rides `default`, honestly: it runs wherever the interactive session runs.
- **The stage→tier mapping stays fab-owned and fixed** (updated in `internal/agent`'s `stageTiers` map + the drift-guarded spec tables): intake→default (advisory), apply→doing, review→review, hydrate→doing, ship→fast, review-pr→doing.
- **Per-field inheritance from `default`**: any tier field left unset (provider, model, effort) inherits from the project's `default` tier, then from fab-kit's built-ins. Inheriting `{provider, model, effort}` is safe precisely because commands moved to `providers:` — the dangerous cross-semantics command inheritance can no longer happen.
- **Documented style: write `provider:` explicitly on every tier line** even though inheritance makes it optional — consistency and per-line readability (the config scaffold and all examples show it explicit; inheritance is the safety net, not the style).
- **Model IDs are written versioned** (`claude-sonnet-5`, `claude-opus-4-8`): bare family IDs like `claude-sonnet` are invalid at the API (404) and also miss `ModelAlias`'s trailing-hyphen prefix match (`claude-sonnet-` at `agent.go:113`), so they fail both dispatch seams. Note for docs: on the *native* seam the version digits are collapsed to the family alias anyway (the harness picks its current model for `sonnet`); versions bind on the CLI-dispatch and session seams.
- **Review resolves once** carries over: the review tier's single resolved profile applies to the inward + outward reviewer sub-agents and the merge.
- fab-kit's own target config after this change:

```yaml
providers:
    claude:
        session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")"'

agent:
    tiers:
        default:  { provider: claude, model: claude-fable-5,  effort: xhigh }
        operator: { provider: claude, model: claude-sonnet-5, effort: medium }
        doing:    { provider: claude, model: claude-opus-4-8, effort: xhigh }
        review:   { provider: claude, model: claude-fable-5,  effort: xhigh }
        fast:     { provider: claude, model: claude-sonnet-5, effort: low }
```

### 3. Retire the `review_tools` config block

- The key is removed from the config schema, `configref.go`, `_cli-fab.md`, and all docs.
- Its two live semantics move to a new **`fab/project/code-review.md` § Review Tools** section (prose, read by the agents that already load code-review.md):
  - the outward-reviewer **Codex → Claude cascade** toggles (`_review.md` updates to read the section instead of config; this also resolves the `configref.go` vs `_review.md` contradiction — the new prose home becomes the single truth),
  - the **Copilot request toggle** for `/git-pr-review` Phase 2 (the skill's config check re-points at the same section).
- Absent section / absent file = all enabled (today's absent-key default, unchanged). The `--tool` flag's force-override behavior on `/git-pr-review` is unchanged.
- Migration: drop `review_tools` from user configs; seed a `code-review.md` § Review Tools entry **only** when a key was explicitly `false` (an all-true block is a no-op and is silently deleted — fab-kit's own block is exactly this case).

### 4. New command: `fab agent` (retires `fab spawn-command`)

```
fab agent [tier] [--print] [--repo <path>]
```

- Resolves the tier profile (`default` when omitted; any of the five tier names accepted), composes `providers.<provider>.session_command` with `{model}`/`{effort}` substituted (or Claude-style flags appended for a non-templated command), and **execs it in the current shell** — `fab agent` starts the default-tier agent right here; `fab agent operator` starts the coordinator profile.
- `--print` prints the fully-resolved command instead of executing — **this replaces `fab spawn-command`**, with a semantic upgrade: output is profile-resolved (model/effort substituted), not placeholder-stripped as today, so callers that spawn from the printed command finally get the tier profile.
- `--repo <path>` reads the target repo's config (the operator's fetch-another-repo's-command use case, carried over from `fab spawn-command --repo`).
- `fab spawn-command` is removed in the same release (no deprecation alias): its only CLI consumer is the operator skill, which ships in the same kit and is updated in the same change. `fab batch` and the operator launcher use the internal `spawn` package, not the CLI command.

### 5. Resolver and dispatch plumbing

- `internal/agent`: `defaultTiers`/`stageTiers` maps rewritten for the five-tier vocabulary; tier values gain `provider`; resolution gains the provider→commands lookup and default-tier per-field inheritance. Drift-guard tests (`TestDocTablesMatchAgentMaps`) and the spec tables they parse update together.
- `fab resolve-agent` keeps its stage-name contract for the dispatch seam; it additionally needs to serve tier-level resolution for `fab agent`/the operator launcher (exact CLI surface is an apply-entry design decision — see Assumptions #14). The `spawn=` output line is expected to become `dispatch=` to match the field rename. <!-- assumed: resolve-agent gains tier-name acceptance and renames spawn= → dispatch=; exact output contract (provider=/session= lines, --alias interaction) left to plan generation -->
- `fab dispatch start` resolves the stage's tier → provider → `dispatch_command` (error when absent, no fallback — message updated for the new key path).
- The operator launcher (`fab operator`) resolves the **operator** tier instead of borrowing doing (`fab resolve-agent apply`), via the same internal resolution. **`fab operator` is NOT reduced to an alias of `fab agent operator`** — it keeps its distinct responsibilities (tmux singleton window management, launching with `/fab-operator` as the initial prompt, `tick-start`/`time` subcommands); only the session-command *composition* is shared. Conceptually: `fab operator` ≈ singleton tmux window running `$(fab agent --print operator) '/fab-operator'`.
- `fab batch new`/`switch` and the operator worker-spawn path compose from `providers.<default.provider>.session_command` + the default tier profile (workers finally spawn with a profile; the placeholder-stripping print path disappears with `fab spawn-command`).
- `configref.go` scaffold rewritten: `providers:` block (claude live, codex commented), five-tier `agent.tiers` with explicit providers, `review_tools` and `agent.spawn_command` removed.

### 6. Migrations (constitution-required)

One migration file (or one per concern, planner's call) restructuring user configs:

- `agent.spawn_command` → `providers.claude.session_command` (verbatim value move; provider name `claude` assumed for existing configs — today's spawn commands are claude invocations or already-templated non-claude commands, in which case the user is told to name the provider).
- `agent.tiers.{thinking,doing,fast}` → five-tier shape: `doing`/`fast` overrides carry over field-by-field; a `thinking` override maps to `review` (its only dispatched stage); per-tier `spawn_command` values become `providers.<name>.dispatch_command` with the tier pointing at the provider.
- `review_tools` removed per § 3.
- fab-kit's own `fab/project/config.yaml` updated to the target shape above.

## Affected Memory

- `_shared/configuration.md`: (modify) config.yaml schema — `providers:` section, five role tiers with `{provider, model, effort}` + inheritance, `review_tools` and `agent.spawn_command` removal
- `runtime/operator.md`: (modify) operator launcher resolves the `operator` tier (no longer borrows doing); worker spawns carry the default-tier profile; `fab spawn-command` retirement
- `runtime/dispatch.md`: (modify) `fab dispatch` resolves `providers.*.dispatch_command`; error message/key-path changes
- `pipeline/execution-skills.md`: (modify) outward-cascade and `/git-pr-review` Copilot toggles now read `code-review.md` § Review Tools (review_tools removed)
- `runtime/providers-and-tiers.md`: (new) the providers/tiers model — provider table, role tiers, inheritance, `fab agent`, the session-vs-dispatch command split

## Impact

- **Go** (`src/go/fab/`): `internal/agent` (maps, resolution, provider lookup), `internal/spawn` (session composition seams), `internal/configref` (scaffold rewrite), `cmd`: new `agent` command, `spawn-command` removal, `resolve-agent` output changes, `dispatch start` key-path change, `operator` launcher tier change. Tests throughout (constitution: Go changes ship tests; drift guards must stay green).
- **Kit skills** (`src/kit/skills/`): `_preamble.md` § Per-Stage Model Resolution (+ tier/provider vocabulary), `_cli-fab.md` (new/changed/removed command signatures — constitution MUST), `_review.md` (cascade config source), `git-pr-review.md` (copilot toggle source), `fab-operator.md`/`_cli-external.md` (spawn-command → fab agent --print), `fab-setup.md` (config sections).
- **Specs** (mirror-class sweep, constitution MUST): `docs/specs/stage-models.md` (major rewrite — tier tables are drift-guarded against the Go maps), `harness-adapters.md`, `architecture.md`, `glossary.md`, `skills.md`, per-skill `docs/specs/skills/SPEC-*.md` for every touched skill.
- **Scaffold/templates**: `src/kit/scaffold` config, `code-review.md` scaffold gains the § Review Tools comment block.
- **Migrations**: `src/kit/migrations/` file(s) per § 6.
- **Scale**: comparable to the 3a–3d dispatch series; expected to split into a small ordered series at planning (e.g., schema+resolver → commands → review_tools/docs), with this intake as the shared design source.
- **Out of scope**: the sibling discussion outcome on `checklist.extra_categories` → `code-quality.md` prose (separate change if pursued); any change to the six-stage pipeline or the SRAD/scoring layer.

## Open Questions

- Exact `fab resolve-agent` CLI surface after the rename: does it accept tier names positionally alongside stage names, or via `--tier`? Does the output gain a `provider=` line? How does `--alias` interact with a non-claude provider's native-dispatch misconfig?
- Migration edge: a project whose existing `agent.spawn_command` is a non-claude template (e.g. a codex command) — auto-create a `codex`-named provider, or halt and ask the user to name it?
- Should `fab agent` (exec mode) refuse to run when not attached to a TTY, or is exec-and-let-the-CLI-fail acceptable?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Extract a top-level `providers:` section; tiers reference providers by name | User-proposed and ratified in discussion ("WAY more understandable"); direct quote of the design goal | S:95 R:70 A:90 D:95 |
| 2 | Confident | Two command fields per provider (`session_command` + `dispatch_command`), not one merged `command` | User proposed the merge; countered with grammar evidence (claude `-n` vs `-p`, `codex` vs `codex exec` — one template cannot express both); user proceeded on the two-field design | S:70 R:70 A:85 D:80 |
| 3 | Confident | Drop the `thinking` tier entirely (five tiers, not six) | User's roster included `thinking`; counter-proposal to drop it (intake never dispatches — with review split out, thinking governs nothing) was made explicitly and the user continued on the 5-tier design without objection | S:65 R:60 A:90 D:80 |
| 4 | Certain | Tier roster `default/operator/doing/review/fast`; stage→tier mapping stays fab-owned and fixed | Roster is user-specified; fixed mapping is the standing design (stage-models.md) the user did not contest | S:90 R:65 A:85 D:90 |
| 5 | Certain | `doing` profile = claude-opus-4-8 / xhigh | User explicit: "opus xhigh for apply, review-pr, hydrate" | S:100 R:90 A:95 D:100 |
| 6 | Confident | `operator` profile = claude-sonnet-5 / medium | User asked "sonnet or opus?"; sonnet/medium recommended (highest-volume agent; pattern-matching work; escalation discipline makes cheaper safe) and unobjected | S:70 R:95 A:75 D:70 |
| 7 | Confident | `default` = claude-fable-5/xhigh, `review` = claude-fable-5/xhigh, `fast` = claude-sonnet-5/low | User named the models ("fabel" for default-context and review, "sonnet" for fast); efforts partially assumed (xhigh matches the spec's Fable upgrade curve; low carries over for fast) | S:70 R:95 A:75 D:70 |
| 8 | Certain | `provider:` written explicitly on every tier line (documented style); per-field inheritance from `default` remains the mechanics | User explicit: "mention provider on all lines — even though its optional — to make it consistent and easy to read" | S:95 R:95 A:95 D:95 |
| 9 | Certain | No provider inference from model strings; cross-provider misconfig is documented, never validated | User ratified ("1 and 2 are ok"); consistent with the standing no-validation/provider-neutrality contract | S:90 R:85 A:95 D:90 |
| 10 | Confident | `review_tools` retires into `code-review.md` § Review Tools (cascade + copilot toggles); absent = enabled | User said "Yes, I would remove it"; the code-review.md home was recommended (policy file already read by both consumers) and accepted implicitly; also resolves the configref/_review contradiction | S:75 R:65 A:80 D:75 |
| 11 | Certain | New `fab agent [tier] [--print] [--repo]`; execs resolved session command in current shell; retires `fab spawn-command` | User requested the capability, and proposed `--print` replacing spawn-command themselves | S:90 R:80 A:90 D:90 |
| 12 | Certain | Model IDs written versioned (`claude-sonnet-5`); bare family IDs unsupported | Verified: API accepts only catalog IDs; `ModelAlias` prefix table requires the trailing hyphen (`agent.go:113`) | S:85 R:90 A:95 D:90 |
| 13 | Confident | Tiers stay nested under `agent.tiers` (no flattening to top-level `tiers:`) | User's phrasing "the agent section could use the tiers…" implies keeping `agent:`; limits config churn | S:60 R:90 A:70 D:65 |
| 14 | Tentative | `fab resolve-agent` gains tier-name acceptance; `spawn=` output line renamed `dispatch=` | Design sketch only — the exact CLI/output contract was not discussed; easily settled at plan generation | S:35 R:60 A:50 D:40 |
| 15 | Confident | `fab spawn-command` removed outright, no one-release deprecation alias | Only CLI consumer is the operator skill, shipped and updated in the same kit; user's "--print could take its place" implies removal is fine; trivially reversible (alias can be added later) | S:60 R:80 A:65 D:60 |
| 16 | Confident | Single intake now; expected to split into an ordered series at planning | Stated in the discussion and in the change description; matches the 3a–3d precedent for this scale | S:70 R:85 A:75 D:70 |
| 17 | Certain | Review-resolves-once carries over: one review-tier profile for inward + outward + merge | Unchanged standing semantics (stage-models.md); nothing in this change alters review's internal structure | S:65 R:85 A:85 D:80 |

17 assumptions (8 certain, 8 confident, 1 tentative, 0 unresolved).
