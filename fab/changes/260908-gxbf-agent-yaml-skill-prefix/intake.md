# Intake: Fold skill prefix into `fab agent` resolution; retire `fab skill-prompt`

**Change**: 260908-gxbf-agent-yaml-skill-prefix
**Created**: 2026-09-08

## Origin

Conversational, via `/fab-proceed` (promptless create-intake dispatch). The user reviewed PR #649 (change `260908-tfzn-fix-agent-skill-prefix`, branch of the same name, draft, base `main`) together with the agent and asked: "is this the best way, is it too roundabout?" The agent's assessment — keep #649's Go renderer and launcher wiring, but replace the new `fab skill-prompt` subcommand with a `skill_prefix` key on the existing `fab agent -o yaml` resolver and collapse the three-step shell recipe in `_cli-agents.md` § Skill Prompts — was accepted by the user asking for a **stacked PR** implementing it.

> Stacking constraint: this change's branch MUST be cut from the current HEAD (`9103fafa`, the head of `260908-tfzn-fix-agent-skill-prefix`), and its PR base MUST be `260908-tfzn-fix-agent-skill-prefix`, **not** `main`. PR #649 is unmerged and unreleased, so nothing this change deletes has ever shipped — no migration, no deprecation shim.

Key decisions from the conversation are reproduced in § What Changes and § Assumptions; the rejected alternatives are in § Why.

## Why

**The bug #649 fixed is real and its Go half is right.** Codex invokes skills with `$skill-name`; every other harness fab knows uses `/skill-name`. Before #649, `fab batch new`, `fab batch switch` and the built-in (rk-absent) `fab operator` launcher hardcoded `/`, so selecting Codex launched the right executable with the wrong initial prompt. #649 added `internal/agent.SkillPrompt(provider, skill, arguments)` (`src/go/fab/internal/agent/skill_prompt.go`: exact provider `codex` ⇒ `$`, every other/unknown/empty provider ⇒ `/`, arguments verbatim after one space) and threaded the launched provider alongside the composed command (`defaultRoleSpawnCommand` in `batch.go` and `operatorSpawnCommand` in `operator.go` both return `(command, provider)`; the Claude executable fallback sets provider `claude`). `TestSkillPromptLaunchers` proves every launcher's final argv carries the right prefix for `codex`/`claude`/`agy`/`kimi`/`custom`/`missing`. All of that stays.

**The roundabout half is the `fab skill-prompt` subcommand.** It exists only so the markdown operator (`fab-operator.md`) can obtain a one-character prefix, and it does so by:

1. Adding a second resolver. `fab skill-prompt --repo <path>` calls `loadRepoConfig` + `agent.ResolveRole(cfg, agent.RoleDefault)` to find the default-role provider — the same question `fab agent default -o yaml --repo <path>` already answers in its `provider:` key. Two resolvers with different failure semantics (the `--repo` mode skips dispatch-capability checks; `fab agent -o yaml` runs the descent ladder) is exactly the owner-or-pointer drift surface `fab/project/code-quality.md` names.
2. Justifying `--repo` with a case the operator cannot hit. #649's plan argued `fab agent -o yaml` errors when a provider has only `interactive_command` and `$TMUX` is unset (no reachable dispatch rung — verified: that error exists). But `/fab-operator` requires `$TMUX` to run at all, and inside tmux the same `-o yaml` query succeeds for such a provider (verified: prints `provider: custom`, `dispatch: rung: pane`). The escape hatch guards against an environment the only caller never runs in.
3. Making the operator do three shell steps for a binary decision. `_cli-agents.md` § Skill Prompts (added by #649) has the operator run `fab skill-prompt --shell-quote …`, then `eval "fab_skill_prompt=$fab_skill_token"`, then `rk mux send`, plus a paragraph on why the closing quote preserves trailing argument newlines — to choose between `$` and `/` when the operator has *already* identified the receiving harness in order to pass `--provider`. Once you know the provider, the rule is one line; the CLI hop adds nothing but surface.

**If we leave it:** a public `fab` subcommand, its 177-line test file, a `_cli-fab.md` section, a help-bundle bullet mirrored across two files under a drift guard, three memory files and three spec files all describe a command whose entire job is `provider == "codex" ? "$" : "/"`. Every future provider-resolution change has to be made twice.

**Rejected alternatives (from the conversation):**

- *Keep `fab skill-prompt` with fewer flags (drop `--repo`).* Still a second entry point and still a shell hop for a one-character decision.
- *A provider capability config field (e.g. `providers.<name>.skill_prefix`).* The user decided in #649 that there is no config knob; the rule is `codex ⇒ $`, else `/`, hard-coded.
- *A prose-only rule with no Go owner.* The Go launchers genuinely need the renderer, so `agent.SkillPrompt`/`SkillPrefix` remains the single owner; prose points at it.

## What Changes

### 1. Remove the `fab skill-prompt` subcommand (Go)

Delete outright:

- `src/go/fab/cmd/fab/skill_prompt.go` (86 lines — `skillPromptCmd()`, the `skillPromptName` regex, `--provider`/`--repo`/`--shell-quote`/`--json` flags).
- `src/go/fab/cmd/fab/skill_prompt_test.go` (177 lines) — **except** `TestSkillPromptLaunchers` (lines ~100–177), which exercises `batch new`/`batch switch`/`operator` launchers end-to-end and must survive. Move it to a launcher-appropriate test file (e.g. `batch_test.go` or a new `launcher_prompt_test.go`); drop `TestSkillPromptCLI`, `TestSkillPromptUsageErrors`, `TestSkillPromptRepoProvider`, `TestSkillPromptShellRoundTrip` (the last one tests the `eval` round-trip recipe this change deletes).
- The `skillPromptCmd(),` registration at `src/go/fab/cmd/fab/main.go:47`.
- The `Long` help text reference in `src/go/fab/cmd/fab/operator.go:34` — `fab-operator skill prompt (see 'fab skill-prompt').` — reword to point at `fab agent -o yaml`'s `skill_prefix` or simply drop the parenthetical.

After this, `fab skill-prompt` is an unknown command (cobra's default error). No alias, no stub.

### 2. Expose the prefix through the existing resolver (Go)

**`src/go/fab/internal/agent/skill_prompt.go`** — extract the single owner:

```go
// SkillPrefix returns the explicit-skill-invocation prefix the receiving
// provider understands: "$" for the exact provider name "codex", "/" for
// every other name, including custom, unknown, and empty.
func SkillPrefix(provider string) string {
	if provider == "codex" {
		return "$"
	}
	return "/"
}

// SkillPrompt renders <prefix><skill>[ <arguments>] for the receiving provider.
// Arguments are prompt text, not shell syntax, and are preserved verbatim.
func SkillPrompt(provider, skill, arguments string) string {
	prompt := SkillPrefix(provider) + skill
	if arguments != "" {
		prompt += " " + arguments
	}
	return prompt
}
```

Extend `src/go/fab/internal/agent/skill_prompt_test.go` with a `TestSkillPrefix` table (`codex` ⇒ `$`; `claude`, `agy`, `kimi`, `custom`, `""` ⇒ `/`).

**`src/go/fab/internal/agent/resolution.go`** — add one field to `Resolution`, **after** `Dispatch` so the documented ordering rule ("the original seven keys keep their order and the remaining keys follow") holds and existing YAML fixtures change only by appending a line:

```go
type Resolution struct {
	Selector string `yaml:"selector"`
	Kind     string `yaml:"kind"`
	Role     string `yaml:"role"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Effort   string `yaml:"effort"`
	Command  string `yaml:"command"`

	ModelAlias  string              `yaml:"model_alias"`
	Template    string              `yaml:"template"`
	FillMode    string              `yaml:"fill_mode"`
	Source      Source              `yaml:"source"`
	Dispatch    *DispatchResolution `yaml:"dispatch,omitempty"`
	SkillPrefix string              `yaml:"skill_prefix"`
}
```

Note `dispatch` is `omitempty`, so `skill_prefix` is the last key when dispatch is present and directly after `source:` when it is absent. `skill_prefix` is **never** omitted — it always has a value (`"$"` or `"/"`), and the bare-provider form (`fab agent --provider oracle -o yaml`) gets one too.

**`src/go/fab/cmd/fab/resolution.go`** — populate it where `agent.Resolution{…}` is built (around line 77): `SkillPrefix: agent.SkillPrefix(profile.Provider)`. The `provider` used is the resolved one (post-fallback, post-override), so `fab agent apply --provider codex -o yaml` yields `skill_prefix: "$"`.

**`Resolution.Lines()` (the `fab resolve-agent` byte-stable line protocol) is NOT extended.** `resolve-agent` is deprecated-compat and its output is asserted byte-for-byte in `resolve_agent_test.go`; the prefix rides YAML only.

**Tests to update (struct-parity, from `260901-mp8d-agent-resolution-struct-parity`):** every `want:` YAML literal in `src/go/fab/cmd/fab/agent_surface_test.go` (the four `-o yaml` cases starting ~line 330: `oracle` native ⇒ append `skill_prefix: /`; headless ⇒ `skill_prefix: /` after the `dispatch:` block; pane ⇒ same; bare provider ⇒ `skill_prefix: /`). Add at least one case whose resolved provider is `codex` (e.g. `agent: {session: codex}` with `fab agent default -o yaml`, or `--provider codex`) asserting `skill_prefix: $` — YAML-quoting note: `yaml.v3` emits `$` bare (`skill_prefix: $`) and `/` bare (`skill_prefix: /`); confirm the exact rendering by running the test rather than guessing, and match the fixture to what the marshaller actually emits. `src/go/fab/internal/agent/resolution_test.go` `TestResolutionLines` is unaffected (Lines ignores the new field) — add an assertion that `Resolution{Provider: "codex", SkillPrefix: "$"}.Lines(false)` still emits no `skill_prefix` line, guarding the protocol.

Run `go test ./...` under `src/go/fab` (`cd src/go/fab && go test ./...`).

### 3. `_cli-fab.md` (canonical CLI reference — constitution MUST)

`src/kit/skills/_cli-fab.md`:

- **Delete** the `## fab skill-prompt` section (lines ~1480–1506) and its Contents entry (`- fab skill-prompt`, line 39).
- **§ fab agent** `-o yaml` key table (line ~1445): append a row after `dispatch`:

  | Key | Semantics |
  |-----|-----------|
  | `skill_prefix` | Explicit-skill-invocation prefix for the resolved provider: `$` for `codex`, `/` for every other provider (built-in, custom, or unknown). Always present; the last key. Skill consumers compose `<skill_prefix><skill>[ <args>]` — `_cli-agents.md` § Skill Prompts owns receiver selection and quoting. |

  Adjust the sentence "The original seven keys keep their order and the remaining keys follow" only if wording needs it (it already covers appended keys).
- **§ fab batch** (line ~1518): replace "Initial skill prompts use § fab skill-prompt with the actual launched provider (including Claude when command resolution falls back to the built-in)." with "Initial skill prompts are rendered by `internal/agent.SkillPrompt` for the actual launched provider (`$` for `codex`, `/` otherwise — including Claude when command resolution falls back to the built-in)."
- **§ fab operator** built-in launcher paragraph (line ~1252–1258): if it mentions `fab skill-prompt`, reword the same way.

### 4. `_cli-agents.md` § Skill Prompts — collapse to a short owner section

Replace the whole `#### Skill Prompts` block (`src/kit/skills/_cli-agents.md` lines ~99–122) with an owner section of roughly this shape (prose may be tightened; the rule content is fixed):

> #### Skill Prompts
>
> An explicit skill invocation is `<prefix><skill>[ <arguments>]`, where the prefix belongs to the **receiving** harness: `codex` ⇒ `$`, every other or unknown provider ⇒ `/`. `internal/agent.SkillPrefix` owns the rule; `fab agent … -o yaml` exposes it as `skill_prefix`. Applies to initial prompts and to skill commands routed into existing panes.
>
> 1. **Identify the receiver.**
>    - *Fresh default-role spawn* (composed with `fab agent --print --repo <target-repo>`): read `skill_prefix` from the same repo's resolution — `fab agent default -o yaml --repo <target-repo>` — alongside `command`. One query supplies both the session command and the prefix; do not resolve the provider a second way.
>    - *Other fresh roles*: the provider already resolved for that launch (`fab agent <role> -o yaml`'s `skill_prefix`).
>    - *Existing pane*: the live agent process per § Peek's process-tree command (`rk mux process <pane>`); the interactive harness executable — including a child beneath a wrapper shell — is the receiver. Never identify it from prompt text, a nested tool worker, the operator's own provider, a model ID, or the repo's current default. Unknown ⇒ `/`.
> 2. **Compose and quote once.** Render `<prefix><skill>[ <arguments>]` with arguments verbatim, then shell-quote the whole prompt as **one token** exactly like any other initial prompt (§ Spawn Composition's spawn-embedding and `rk mux send` rules apply unchanged; the pre-send gate still applies to existing-pane sends). Never embed a literal `$`-prefixed skill inside an outer double-quoted shell string.
> 3. **Keep message types distinct.** Only explicit skill invocations take a prefix. Ordinary prompts, answers, keys, `Read <path> and execute it.` pointers, and native TUI controls such as `/loop` or `/clear` use their own contracts and are not transformed.

Deleted with the old block: the `fab_skill_token=$(fab skill-prompt …)` / `eval "fab_skill_prompt=$fab_skill_token"` recipe, both code fences, and the trailing-newline-preservation paragraph. Keep the Peek process-tree pointer as a pointer (owner-or-pointer).

### 5. `fab-operator.md` — fix the two places that name the retired renderer

`src/kit/skills/fab-operator.md`:

- Line 120 (`**Skill routing:** … uses _cli-agents.md § Skill Prompts …`) — keep as is; it is a pointer.
- Step 6 (line ~543): replace "Use the same target repo in the renderer's `--repo` mode per `_cli-agents.md` § Skill Prompts." with "Read `skill_prefix` from the same target repo's `fab agent default -o yaml --repo <target-repo>` per `_cli-agents.md` § Skill Prompts (one resolution supplies both `command` and the prefix)."
- Step 7 (lines ~544–550): the tmux example must stop referencing `$fab_skill_token`. Rewrite as:

  ```sh
  tmux new-window -t '<session>:' -P -F '#{session_name} #{pane_id}' -n "»<wt>" -c <worktree-path> "$spawn_cmd $skill_prompt_quoted; exec \"\$SHELL\""
  ```

  with the parenthetical reading "… `$spawn_cmd` is the target repo's command from step 6, and `$skill_prompt_quoted` is the composed `<skill_prefix><skill> <args>` prompt shell-quoted as one token per `_cli-agents.md` § Skill Prompts". Everything else in step 7 (session-name escaping, `-P -F`, missing-`-t` error) is untouched.
- Lines ~636–638 (Working a Change items 1–3) — "render it per `_cli-agents.md` § Skill Prompts" stays; these are pointers.

Owner-or-pointer: `fab-operator.md` points, `_cli-agents.md` owns, `internal/agent.SkillPrefix` is the Go owner.

### 6. Help bundle, specs, memory — sibling sweep

Grep `skill-prompt`, `skill_prompt`, `fab_skill_token`, `fab_skill_prompt`, `SkillPrompt`, and `renderer` repo-wide (excluding `.claude/` and `fab/changes/`) before finishing apply. Known hits to fix:

- **Help bundle** — `docs/site/skill.md:55` (canonical) and `src/go/fab/cmd/fab/skill.md:55` (go:embed copy; `skill_test.go` drift-guards byte-equality; `scripts/sync-skill.sh` copies canonical → embed): delete the `- **Skill prompts** — fab skill-prompt …` bullet (three lines). Edit the canonical file and re-run `scripts/sync-skill.sh`.
- **Specs** — `docs/specs/glossary.md:104` (remove `fab skill-prompt` from the `fab` CLI row's orchestration list); `docs/specs/architecture.md:566` (rewrite: "Initial skill prompts are rendered by `internal/agent.SkillPrompt` for the receiving provider — `$` for Codex, `/` for every other or unknown provider; `fab agent -o yaml` exposes the same rule as `skill_prefix`; launch commands quote the complete prompt as one shell argument. The fallback operator launcher shares the renderer."); `docs/specs/skills.md:1095` (drop "via `fab skill-prompt`"; say the prefix comes from `skill_prefix` / receiver identity per `_cli-agents.md` § Skill Prompts); `docs/specs/stage-models.md:572–574` (YAML key list — add `skill_prefix`). Specs are human-curated pre-implementation intent (Constitution VI) — edit the sentences that name the retired command; do not restructure.
- **Memory** (hydrate rewrites these; the intake lists the load-bearing edits so apply's sweep and hydrate agree):
  - `docs/memory/runtime/agent-primitives.md` § Requirement: Skill Prompt Rendering (lines ~45–57): drop the `fab skill-prompt` paragraph and the "capture `--shell-quote` output … decode … closing quote preserves trailing newlines" prose; state `SkillPrefix`/`SkillPrompt` ownership and the `skill_prefix` YAML exposure; keep the receiver-identity rules. Design Decision "Shared Skill Rendering Uses the Receiving Provider" (line ~154): amend **Decision** to "…its prefix is exposed as `skill_prefix` on `fab agent -o yaml`; no separate CLI" and add this change to *Introduced by*, or add a new DD entry recording the retirement with **Rejected**: a standalone `fab skill-prompt` CLI (second resolver, shell hop).
  - `docs/memory/runtime/operator.md` line ~97 (the tmux example with `$fab_skill_token`) and line ~99 ("The shared renderer formats each invocation…") — align with § 5 above.
  - `docs/memory/runtime/providers-and-profiles.md` line ~370 (YAML key list — add `skill_prefix`) and lines ~383 (delete "For fresh default-role launches, `fab skill-prompt --repo <path>` resolves… Its explicit `--provider` mode…"; keep the launcher-retains-provider sentence).
  - `docs/memory/runtime/index.md` descriptions for `agent-primitives` and `operator` already say "receiver-aware skill prompts" — still true; leave unless hydrate's `fab memory-index` regenerates them from frontmatter.
- **Always-load / dispatch-site prose that enumerates YAML keys** (`_preamble.md:305`, `_pipeline.md:95`, `fab-continue.md:69`, `fab-fff.md:62`, `docs/memory/_shared/context-loading.md:125`) says "at minimum `provider`, `model`, `model_alias`, `effort`, and `dispatch:` presence" — that is a *minimum surfacing* list, not the schema; `skill_prefix` does not need to be added there. Do not touch them.

### 7. Explicit non-goals

- **`src/go/fab/internal/change/change.go` `defaultCommand`** (line ~462) still emits slash-form `/fab-continue`, `/git-pr`, `/git-pr-review` in the human-facing `Next:` line of `fab status`. **Leave it.** It is a suggestion to the human reading the terminal, not a wire payload, and fixing it would require plumbing provider resolution into `fab status`. Recorded here as a **follow-up candidate**, not scope.
- No change to run-kit or the delegated `rk operator` kickoff.
- No provider config keys (no `skill_prefix` in `providers.<name>`).
- No change to the launcher behaviour #649 fixed (`batch.go`, `batch_new.go`, `batch_switch.go`, `operator.go` wiring stays; only the `Long` text in `operator.go` is touched).
- No migration, no deprecation shim, no alias for `fab skill-prompt` — it never shipped.

## Affected Memory

- `runtime/agent-primitives`: (modify) § Requirement: Skill Prompt Rendering — drop the `fab skill-prompt` CLI contract and the shell-quote/`eval` recipe; state `SkillPrefix`/`SkillPrompt` ownership and `skill_prefix` exposure; amend/add the Design Decision recording the CLI's retirement.
- `runtime/operator`: (modify) spawn step 7 example (`$fab_skill_token` ⇒ one-token quoted prompt from `skill_prefix`) and the "shared renderer" sentence in the work-paths paragraph.
- `runtime/providers-and-profiles`: (modify) `fab agent -o yaml` key list gains `skill_prefix`; remove the `fab skill-prompt --repo`/`--provider` sentences from the launcher paragraph.

## Impact

- **Go** (`src/go/fab`): `cmd/fab/skill_prompt.go` (delete), `cmd/fab/skill_prompt_test.go` (delete; relocate `TestSkillPromptLaunchers`), `cmd/fab/main.go` (unregister), `cmd/fab/operator.go` (help text), `cmd/fab/resolution.go` (populate), `cmd/fab/agent_surface_test.go` (YAML fixtures + codex case), `internal/agent/skill_prompt.go` (+`SkillPrefix`), `internal/agent/skill_prompt_test.go` (+table), `internal/agent/resolution.go` (+field), `internal/agent/resolution_test.go` (Lines guard). Net negative LOC. `go test ./...` must pass.
- **Kit skills** (canonical `src/kit/skills/` only): `_cli-fab.md`, `_cli-agents.md`, `fab-operator.md`.
- **Help bundle**: `docs/site/skill.md` + `src/go/fab/cmd/fab/skill.md` via `scripts/sync-skill.sh` (drift-guarded).
- **Specs**: `glossary.md`, `architecture.md`, `skills.md`, `stage-models.md`.
- **Memory**: the three files above.
- **Behavioural surface**: `fab skill-prompt` removed (never released); `fab agent -o yaml` gains one always-present trailing key. `fab resolve-agent` output unchanged. Launcher argv unchanged.
- **Change type**: `refactor` — no user-visible launcher behaviour changes; a resolver key moves ownership of an already-shipped-in-branch decision.
- **Git**: branch from HEAD `9103fafa`; PR base `260908-tfzn-fix-agent-skill-prefix`.

## Open Questions

None. The conversation settled the approach, the ownership split, the non-goals and the stacking. The one item left to apply — which file `TestSkillPromptLaunchers` moves into — is a naming choice recorded as an assumption (row 11), not a question for the user.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove `fab skill-prompt` entirely (Go command, tests, `main.go` registration, `_cli-fab.md` section, help bundle, specs, memory) with no shim or migration | Discussed — user asked for a stacked PR implementing exactly this; PR #649 is unmerged so the command never shipped | S:95 R:90 A:95 D:95 |
| 2 | Certain | Keep #649's Go renderer and launcher wiring (`agent.SkillPrompt`, `(command, provider)` returns, Claude-fallback provider, `TestSkillPromptLaunchers`) | Discussed — agent's assessment "the Go half is correct and minimal — KEEP IT", accepted by the user | S:95 R:90 A:95 D:95 |
| 3 | Certain | Expose the prefix as a `skill_prefix` key on `fab agent -o yaml`, appended after existing keys; `"$"` for `codex`, `"/"` otherwise; owner `agent.SkillPrefix(provider)` used by `SkillPrompt` | Discussed — the agreed replacement for the second resolver; ordering follows the documented "original seven first, remaining follow" rule | S:90 R:85 A:95 D:90 |
| 4 | Certain | Collapse `_cli-agents.md` § Skill Prompts to receiver identity (process tree for existing panes, `skill_prefix` from `fab agent default -o yaml --repo` for fresh default-role spawns, unknown ⇒ `/`) + one-token quoting; delete the `fab_skill_token`/`eval` recipe and trailing-newline prose; keep the message-types-distinct sentence | Discussed — item 3 of the agreed approach, verbatim | S:95 R:85 A:95 D:95 |
| 5 | Certain | `fab-operator.md` keeps its pointers; only step 6/7 text and the tmux example change to reference `skill_prefix` and a one-token quoted prompt | Discussed — owner-or-pointer per code-quality.md; fab-operator points, `_cli-agents.md` owns | S:90 R:90 A:95 D:90 |
| 6 | Certain | `change.go` `defaultCommand` slash-form `Next:` suggestions are a non-goal, recorded as a follow-up candidate | Discussed — explicit user-accepted decision, not an open question | S:95 R:95 A:95 D:95 |
| 7 | Certain | Change type `refactor` | Launcher behaviour is unchanged; the change relocates ownership of an in-branch decision and removes surface — matches `docs/specs/change-types.md` `refactor`; `fix` keywords are absent from the intent | S:85 R:95 A:90 D:85 |
| 8 | Certain | `Resolution.Lines()` (`fab resolve-agent`) is NOT extended with the prefix; `skill_prefix` rides YAML only, and `skill_prefix` is never `omitempty` | `resolve-agent` is deprecated-compat with byte-asserted output; the YAML consumer is the operator; always-present avoids a "missing means what?" branch in prose | S:75 R:85 A:85 D:80 |
| 9 | Certain | Place the new field after `Dispatch` in the struct so it is the final YAML key whether or not `dispatch:` is present | Keeps every existing fixture a one-line append; the documented rule only constrains the first seven keys | S:70 R:90 A:85 D:80 |
| 10 | Certain | The always-load/dispatch-site "surface at minimum provider/model/model_alias/effort/dispatch presence" sentences are not updated for `skill_prefix` | They state a minimum surfacing set for stage dispatch, not the schema; stage workers do not compose skill prompts | S:70 R:90 A:85 D:80 |
| 11 | Certain | Relocate `TestSkillPromptLaunchers` (not delete it) when `skill_prompt_test.go` goes; exact destination file chosen at apply | The test guards the launcher fix this change keeps; the file name is a low-stakes naming choice — deferred to apply, not the user | S:70 R:95 A:90 D:75 |

11 assumptions (11 certain, 0 confident, 0 tentative, 0 unresolved).
