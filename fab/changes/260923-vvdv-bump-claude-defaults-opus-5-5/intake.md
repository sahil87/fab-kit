# Intake: Bump Claude Default Model Profiles to Opus 5.5

**Change**: 260923-vvdv-bump-claude-defaults-opus-5-5
**Created**: 2026-09-23

## Origin

Created by `/fab-proceed`'s promptless create-intake dispatch (`{questioning-mode} = promptless-defer`) from a description the team lead synthesized after the fab-kit owner reviewed and approved the exact proposal in conversation. No questions were asked; every decision below arrived already made and is recorded as such.

> Bump the kit-shipped Claude default model profiles from Claude Opus 5 to Claude Opus 5.5. Claude Opus 5.5 (`claude-opus-5-5`) has been released — a straight successor to Claude Opus 5 in the same tier at a lower price ($4/$20 per MTok versus $5/$25), same 1M context, same 128K max output, same tokenizer and feature set. The kit still pins `claude-opus-5` in four of the six Claude role fills. There is no reason to keep the old pin. Keep `effort: high` explicit on the Opus rows; Sonnet rows stay; the kit `default` becomes Opus 5.5, not Fable; codex/kimi/agy fills are out of scope.

Interaction mode: one-shot dispatch. Decisions carried in from the conversation (see § What Changes and § Assumptions): the four target rows and their unchanged effort, the two untouched Sonnet rows, Opus 5.5 rather than Fable as the kit default, Claude-only scope, no `review` effort bump, and the verified facts that the installed Claude Code CLI (2.1.280) contains the `claude-opus-5-5` model ID and that `agent.ModelAlias` already resolves it to `opus`.

## Why

**The pain point.** `src/go/fab/defaults.yaml` is the single embedded data file that decides which model every fab-dispatched Claude worker runs on when a project has not overridden a role. Four of its six Claude role fills (`default`, `doing`, `review`, `hydrate`) still pin `claude-opus-5`. Claude Opus 5.5 (`claude-opus-5-5`) is its direct successor in the same Opus tier: same 1M context window, same 128K max output, same tokenizer, same feature set, and a lower list price ($4 input / $20 output per MTok against $5 / $25). Every kit user on the defaults is therefore paying roughly 20% more per token than they need to for the same tier, on every apply, review, and hydrate dispatch, plus every `fab batch` worker session on the `default` role.

**The consequence of not doing it.** The pin does not break anything today — Opus 5 remains served — so the cost is silent and continuous rather than a failure. It compounds across every project that adopted fab-kit without a per-role override, and it drifts the "single file to edit when a new model ships" (the spec's own description of `defaults.yaml`) one generation behind the CLI it drives.

**Why this approach.** The spec designed `defaults.yaml` exactly for this event: bump the fills, keep the drift-guarded mirror table and the pinned-profile test in lockstep, and let the config cascade carry every user override untouched. Two alternatives were considered and rejected in conversation:

- **Make the kit `default` Fable (`claude-fable-5-1`)** — rejected. Fable is about 2.5x the price of Opus 5.5 and carries additional safety gating. It is the right choice for some users, and the config cascade already lets them make it (the owner's own machine overrides `default` to `claude-fable-5-1` at the system tier and that keeps working with no change here). The kit default should be the cheapest model in the top coding tier, which is now Opus 5.5.
- **Raise `review` effort to `xhigh`** to match the codex critic row — rejected as out of scope. That is a separate judgment about the critic's cost/benefit, not part of a model bump; the spec's "Why these defaults" already argues `high` is the sweet spot for the Claude rows.

## What Changes

### 1. `src/go/fab/defaults.yaml` — the four Opus fills

Under `providers.claude.profiles` (currently lines 120–125), the four `claude-opus-5` rows become `claude-opus-5-5`. Effort stays exactly as-is on every row. The two Sonnet rows are untouched: there is no newer Sonnet, and Haiku is deliberately absent from the defaults (no effort parameter; `ship` needs faithful PR-description comprehension — see `_shared/configuration.md`).

| Role | Current | New |
|------|---------|-----|
| `default` | `claude-opus-5` / `high` | `claude-opus-5-5` / `high` |
| `operator` | `claude-sonnet-5` / `medium` | unchanged |
| `doing` | `claude-opus-5` / `high` | `claude-opus-5-5` / `high` |
| `review` | `claude-opus-5` / `high` | `claude-opus-5-5` / `high` |
| `hydrate` | `claude-opus-5` / `high` | `claude-opus-5-5` / `high` |
| `fast` | `claude-sonnet-5` / `medium` | unchanged |

Resulting block (alignment preserved in the file's existing column style):

```yaml
    profiles:
      default:  { model: claude-opus-5-5, effort: high }
      operator: { model: claude-sonnet-5, effort: medium }
      doing:    { model: claude-opus-5-5, effort: high }
      review:   { model: claude-opus-5-5, effort: high }
      hydrate:  { model: claude-opus-5-5, effort: high }
      fast:     { model: claude-sonnet-5, effort: medium }
```

**Explicit effort is now load-bearing.** Opus 5.5's built-in default effort is `medium`, one level below Opus 5's `high`. fab never relies on the model's built-in default: both claude command templates in the same file carry `--effort {effort}` (`interactive_command` and `headless_command`), and the operator launcher substitutes the profile's effort through `spawn.WithProfile`. So kit behavior is unchanged by the bump — but the explicit `effort: high` on the Opus rows can no longer be treated as redundant with the model's own default, and MUST NOT be dropped. Apply SHOULD record that in a short YAML comment beside the profiles block (one or two lines), so the next person to touch the file sees why the effort column matters.

No other key in `defaults.yaml` changes. The codex, kimi, and agy provider blocks are out of scope.

### 2. `docs/specs/stage-models.md` § Default role profiles — the drift-guarded mirror

The table at lines 128–133 is the verified mirror of what `defaults.yaml` composes; `TestDocTablesMatchAgentMaps` (`src/go/fab/internal/agent/stagemodels_doc_test.go`) parses it under the `### Default role profiles` heading and fails on any disagreement. The four `claude-opus-5` cells become `claude-opus-5-5`; the `operator` and `fast` rows and every `Effort` cell stay. The table and the YAML MUST land in the same commit.

The "Why these defaults" paragraph that follows names "Opus" and "Sonnet" generically, with no version — it stays as-is except for one added sentence noting that Opus 5.5's built-in default effort is `medium`, which is why the explicit `high` on the Opus rows is load-bearing (fab passes `--effort` on every CLI arm). Constitution VI: this is a human-curated spec table being kept truthful, not generated content.

### 3. Go test pins — only the rows that encode the shipped default

Grep in this worktree shows exactly one place that asserts the shipped Claude defaults as literals: `TestDefaultRoleProfilesArePinned` in `src/go/fab/internal/agent/agent_test.go` (lines 65–70). Its comment says "When you bump a default: edit defaults.yaml, then update this table to match." The four `RoleDefault`/`RoleDoing`/`RoleReview`/`RoleHydrate` entries become `claude-opus-5-5`; `RoleOperator`/`RoleFast` stay.

Every other `claude-opus-5` / `claude-sonnet-5` occurrence in `src/go` tests is an **arbitrary fixture value**, not an assertion about the shipped default, and stays verbatim:

- `src/go/fab/cmd/fab/agent_surface_test.go:336–348` — an explicit `providers.oracle.profiles.doing` fixture with an `oracle tui -m claude-opus-5 -e high` command.
- `src/go/fab/cmd/fab/resolve_agent_test.go:124–138, 707–711` — explicit `provider: oracle` profile fixtures and an explicit `--model claude-opus-5 --effort xhigh` override.
- Sonnet fixtures throughout (`agent_test.go:311–937`, `config_test.go:363–378`, `agent_surface_test.go:123–140`, `resolve_agent_test.go:320–497`) — explicit config or flag values.

The other test files named in the dispatch brief (`configupgrade_test.go`, `freeze_test.go`, `config_test.go`, `resolution_test.go`, `spawn_test.go`, `cmd/fab/agent_test.go`) contain **no** `claude-opus-5` literal (verified: zero matches); `spawn_test.go` uses `claude-opus-4-8` as a fixture and stays. `TestConfigReferenceDocumentsProviderFill` (cmd/fab) derives its expected fill lines from `ResolveProvider`, so it needs no edit and will pass once the YAML changes. Apply MUST re-grep before finishing rather than trusting this list.

### 4. `src/kit/skills/_cli-fab.md` — two example output lines

Lines 325 and 373 show `fab resolve-agent apply` example output with `model=claude-opus-5`. Both become `model=claude-opus-5-5`. The `--alias` example on the following block prints `model=opus` and is unchanged (the prefix map `claude-opus-` → `opus` already covers the new ID). No command signature changes, so the Constitution's CLI⇒docs constraint is satisfied by this text update alone. This is deployed kit content; it cites no fab-kit-only path.

### 5. Memory (hydrate) — present-truth citations only

- `docs/memory/runtime/providers-and-profiles.md` (~line 211) and `docs/memory/_shared/configuration.md` (~line 232) cite `claude-sonnet-5` / `claude-opus-5` as examples of the "versioned ID" style. Where the example illustrates the shipped default, hydrate bumps `claude-opus-5` → `claude-opus-5-5`; the point being made (versioned IDs vs. bare family IDs) is unchanged.
- `docs/memory/_shared/context-loading.md` mentions `claude-opus-4-8` only as a historical example inside Design Decisions and stays.
- `docs/memory/*/log.md`, `log.seed.md`, and `docs/findings/*` are history and stay verbatim; hydrate appends its own log entry as usual.
- A Design Decisions entry in `runtime/providers-and-profiles.md` (four-field shape) records: kit default = Opus 5.5 not Fable (price + gating; Fable is a per-user cascade override), explicit `effort: high` is load-bearing under Opus 5.5's `medium` built-in default, Sonnet rows unchanged (no newer Sonnet), Claude-only scope.

### 6. Verified non-sites

- `grep -rn claude-opus-5 src/go` outside tests hits only `defaults.yaml`. The `# >>> fab reference` fence (`fab config explain` / `fab config upgrade`) renders fill lines live from the embedded defaults (`configref.go`), so the project's `fab/project/config.yaml` fence updates itself on the next `fab config upgrade` and carries no hand-edited model string.
- `agent.ModelAlias` (`src/go/fab/internal/agent/agent.go:399–419`) matches on the `claude-opus-` prefix, so `claude-opus-5-5` → `opus` with no Go logic change; `agent_test.go:1295`'s alias table already covers the prefix behavior.
- No CHANGELOG file exists in the repo; releases are cut by separate `release: vX.Y.Z` commits.
- The installed Claude Code CLI (2.1.280) contains the string `claude-opus-5-5` (14 occurrences in the binary), so `--model claude-opus-5-5` is accepted on both the pane and headless arms.

### Non-goals

- No change to codex, kimi, or agy fills or command templates.
- No `review` effort change (`high` stays; `xhigh` is a separate decision).
- No Fable anywhere in the kit defaults.
- No migration file: config `presence=intent` means a user who copied the old rows above their fence keeps `claude-opus-5` on purpose, which is correct behavior.
- No Go logic change; no new CLI surface.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) bump the present-truth `claude-opus-5` example to `claude-opus-5-5`; add a Design Decisions entry for the Opus 5.5 bump (default = Opus 5.5 not Fable; explicit `effort: high` load-bearing; Sonnet unchanged; Claude-only)
- `_shared/configuration`: (modify) bump the "versioned ID" example `claude-opus-5` → `claude-opus-5-5` at the `agent.profiles` style paragraph; no schema change

## Impact

- **Code**: `src/go/fab/defaults.yaml` (4 value edits + a short comment), `src/go/fab/internal/agent/agent_test.go` (4 literal edits). Go files touched MUST be gofmt-clean (the `ci-gate` check runs gofmt).
- **Specs**: `docs/specs/stage-models.md` (4 table cells + one sentence of rationale).
- **Kit skills**: `src/kit/skills/_cli-fab.md` (2 example lines). Deploy is via `fab sync`; no deployed-copy edits.
- **Memory**: two files per § Affected Memory, plus hydrate's log entries.
- **Tests to run** (scoped first, then widen only if something surprises): `go test ./internal/agent/... ./internal/config/... ./internal/configupgrade/... ./cmd/fab/...` from `src/go/fab`. `TestDefaultRoleProfilesArePinned`, `TestDocTablesMatchAgentMaps`, and `TestConfigReferenceDocumentsProviderFill` are the three that observe this change.
- **Runtime effect**: every Claude dispatch on an un-overridden `default`/`doing`/`review`/`hydrate` role moves to Opus 5.5 at `high` once a release carrying this ships and the user upgrades. Projects with per-role overrides (including the owner's system-tier Fable `default`) see no change.
- **Cost**: ~20% lower per-token spend on the four affected roles for kit-default users.

## Open Questions

- None. Every decision arrived pre-made from the owner-approved proposal; see § Assumptions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The four Opus rows (`default`, `doing`, `review`, `hydrate`) change to `claude-opus-5-5` with `effort: high` kept explicit | Discussed — owner approved the exact table; effort explicitness is required because Opus 5.5's built-in default is `medium` and fab passes `--effort` on every CLI arm | S:95 R:85 A:95 D:95 |
| 2 | Certain | `operator` and `fast` stay on `claude-sonnet-5` / `medium` | Discussed — no newer Sonnet exists; Haiku is deliberately excluded from the defaults | S:90 R:90 A:95 D:95 |
| 3 | Certain | The kit `default` role becomes Opus 5.5, not Fable | Discussed — Fable is ~2.5x the price with extra gating; per-user override via the config cascade (the owner's system-tier `claude-fable-5-1` keeps working untouched) | S:90 R:85 A:90 D:90 |
| 4 | Certain | Codex, kimi, and agy fills and templates are out of scope | Discussed — Claude-only bump | S:90 R:90 A:95 D:95 |
| 5 | Certain | No migration file ships | Config `presence=intent`: a user who copied the old rows above the fence keeps their pin deliberately; the fence itself regenerates live from embedded defaults | S:80 R:85 A:90 D:90 |
| 6 | Certain | Change type is `chore`, set explicitly | `change-types.md` lists "update configs / bump dependencies" under `chore`; the keyword inference would misfire on `docs`/`test` path mentions in this intake, so the type is pinned with `fab status set-change-type` | S:80 R:95 A:90 D:85 |
| 7 | Confident | Test sweep is limited to the six-row pin in `TestDefaultRoleProfilesArePinned`; every other `claude-opus-5` in tests is an arbitrary fixture value and stays | Grep-verified in this worktree; apply re-greps before finishing | S:70 R:90 A:80 D:70 |
| 8 | Confident | Memory scope: bump only the present-truth "versioned ID" examples in `runtime/providers-and-profiles.md` and `_shared/configuration.md`; historical `claude-opus-4-8` examples, logs, and `docs/findings` stay | Present-truth vs. history distinction in the FKF memory style; log files are append-only | S:70 R:90 A:80 D:70 |
| 9 | Confident | Record the "explicit effort is load-bearing" rationale as a short YAML comment in `defaults.yaml` and one sentence in the spec's "Why these defaults" | Owner-stated constraint ("must not be dropped") deserves a durable home next to the value it protects; small, reversible prose | S:70 R:95 A:80 D:65 |
| 10 | Confident | `claude-opus-5-5` is accepted by the installed CLI and served on the owner's account | The 2.1.280 binary contains the model ID string (14 hits) and the released-model reference lists it; live serving is not exercised at intake — the first pane/headless dispatch after apply is the smoke test, and a one-line revert covers failure | S:80 R:90 A:45 D:70 |
| 11 | Certain | No change to the claude `interactive_command` / `headless_command` templates or any Go logic | Both templates already substitute `{model}` and `{effort}`; `ModelAlias` prefix-matches the new ID | S:85 R:90 A:90 D:90 |
| 12 | Certain | `review` effort stays `high`; `xhigh` is not part of this change | Discussed — explicitly rejected as a separate decision | S:90 R:95 A:90 D:90 |

12 assumptions (8 certain, 4 confident, 0 tentative, 0 unresolved).
