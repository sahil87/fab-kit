# Intake: User-Level fab-operator Skill Shim — the Operator Resolves From a Neutral cwd

**Change**: 260913-ga0v-operator-skill-user-level-shim
**Created**: 2026-09-13

## Origin

Backlog item `[ga0v]` (2026-09-13), created from a `/fab-discuss` assessment and dispatched promptless (`{questioning-mode} = promptless-defer`: no questions asked; would-be questions land in `## Assumptions`). The user accepted the assessment's recommendation and said: "track the user-level shim task as your next task", to be started from origin/main after 1nxw (#676) merged — done; base is `cbbaf8b7`.

> **Backlog `[ga0v]`** (abridged — the full entry is in `fab/backlog.md`): User-level fab-operator skill shim so the operator works from a neutral cwd (home dir). PROBLEM: `/fab-operator` is resolved by the agent harness from the window's cwd; `fab sync` deploys project-level only. `rk operator`'s interactive path opens the window at the git root of the caller's cwd, but the `-L <server>` path (used by the operator-tick cron entry's `if_absent: respawn` → `rk operator -L {server}`) opens it in the HOME directory, where `/fab-operator` does not resolve — the kickoff is typed into the agent as an unknown command and rk still exits 0. GOAL: `fab sync` (or a `fab setup` step) writes a user-level SHIM, not a copy: `~/.agents/skills/fab-operator/SKILL.md` unconditionally and `~/.claude/skills/fab-operator/SKILL.md` gated on the `claude` CLI — the same two-tier rule fab applies at project level (t513/yd9s) and `shll setup agent` applies user-level. The shim's body is a pointer: read `$(fab kit-path)/skills/fab-operator.md` and follow it, resolving `_preamble` and the `helpers:` list from `$(fab kit-path)/skills/`. PREREQ PROSE CHANGE: `_preamble` § Skill Helper Declaration gains the fallback `.agents/skills/` when present, else `$(fab kit-path)/skills/`. TRADEOFF (accepted): a user-level shim follows the installed fab, not a project's pinned kit — right for the operator, which spans repos and has no owning project by design (2sdj). RELATED: iyb4 (#675), 1nxw (#676).

**Terminology.** This intake calls the new file the **user-level operator skill** (or "the pointer skill"). "Shim" survives in the title and slug because the user named it so, but fab's docs already use "the shim" for the `fab` router binary (`_cli-fab.md` § Calling Convention: "the shim's default route forwards it") and `fab sync --shim` is an existing flag with an unrelated meaning. Go identifiers, CLI output, and doc prose in this change use "user-level operator skill" / `deployUserOperatorSkill`, never "shim".

**Verified during intake (2026-09-13), beyond the backlog text:**

- Claude Code's official docs (code.claude.com/docs/en/skills.md) state the precedence for a same-named skill: **enterprise > personal (`~/.claude/skills/`) > project (`.claude/skills/`)** — the user-level copy wins even inside a project. Claude Code walks `.claude/skills/` from the cwd up to the repo root and does **not** read `.agents/skills/` (matches memory: yd9s verified 2.1.263). The repo documents no project-vs-user precedence anywhere (`docs/memory/distribution/kit-architecture.md` § Agent Skill Deployment only mentions same-name shadowing between per-brand and generic dirs). Consequence: the backlog's acceptance line "a project cwd still resolves the project copy first" is not achievable for Claude Code; the pointer skill must itself defer to the project copy when one exists (§ What Changes 1, rule 1). Codex's `.agents/skills` vs `~/.agents/skills` precedence is undocumented in the repo; rule 1 makes it moot.
- `fab sync` (`src/go/fab-kit/internal/sync.go` `Sync`, `internal/skills.go` `deploySkills`) lives in the **`src/go/fab-kit` Go module**; `fab setup`, `fab setup check` (`src/go/fab/cmd/fab/setup.go`, `setup_wizard.go`, `src/go/fab/internal/setupcheck/`) and `fab kit-path` (`src/go/fab/cmd/fab/kitpath.go`) live in the **separate `src/go/fab` module** (fab-go). Neither module imports the other, and Go's `internal/` rule forbids importing `src/go/fab-kit/internal` from `src/go/fab`. One shared Go helper called from both `fab sync` and `fab setup` is therefore impossible without duplicating it or introducing a cross-module dependency (§ What Changes 2 resolves this).
- `fab kit-path` is context-dependent: inside a project it resolves the **pinned** kit (this worktree: `~/.fab-kit/versions/2.26.3/kit`, pin `fab/.fab-version` = 2.26.3 while the binary is 2.26.4); outside any project it resolves the installed fab's kit (`~/.fab-kit/versions/2.26.4/kit`). Both hold `skills/*.md` flat copies of every skill and helper.
- `fab init` and `fab upgrade-repo` both call `Sync` (`init.go:74`, `upgrade.go:144`), so a writer inside `Sync` runs on every project bootstrap, every upgrade, and every plain sync.
- `~/.claude/skills/` currently holds only `shll-toolkit` (written by `shll setup agent`, `sahil87/shll` `src/cmd/shll/agent_setup.go`: `~/.agents/skills/` unconditional, `~/.claude/skills/` gated on `claude` on PATH — the precedent for the two-tier user-level rule).
- No Go doc test pins the standard preamble blockquote (`Read the `_preamble` skill first …`); `docs/specs/skills.md:208` mandates it as the uniform opening line of every skill.
- `internal/setupcheck` carries the header invariant "Probes read config, run exec.LookPath, and read at most one file (the kit cache's VERSION)" and a "nothing here writes" rule; `Input` already exposes `LookPath` and `KitDir` seams.

## Why

**Problem.** The operator is, by design, a per-tmux-server, cross-repo singleton with no owning project (memory `runtime/operator.md` DD *Operator Launch Is Git-Optional, `fab/`-Optional* — 2sdj: "its natural launch point is a neutral parent directory"). But its skill exists only where `fab sync` put it: inside synced projects. `rk operator -L <server>` — the exact command the operator-tick cron entry uses to respawn a missing operator (`if_absent: respawn`, verified in `rk cron list --json`) — opens the window in `$HOME`, boots the provider, and types `/fab-operator` (or `$fab-operator` for codex). From `$HOME` the harness finds no such skill; the kickoff is consumed as an unknown command, rk exits 0, and the "operator" is a bare agent session with no instructions. Today the operator works only because the user launches it from inside a fab project (live pane cwd: `/home/sahil/code/sahil87/fab-kit`).

**Consequence of not fixing.** The whole clock design (bjrk: fab seeds the cron entry; tnmm: skip-if-busy delivery; qvek: mute/lease) assumes the respawn path produces a working operator. It does not — the respawned window silently lacks the skill, so every "operator died overnight" case ends in an idle agent that never ticks. The launch precondition "run it from a fab project" is also undocumented and contradicts the 2sdj design.

**Why a pointer, not a copy.** `fab-operator.md` line 9 reads `_preamble` from `.agents/skills/`, and `_preamble.md` § Skill Helper Declaration makes the agent read `.agents/skills/{helper}/SKILL.md` for each of the operator's four helpers (`_cli-fab-operator`, `_cli-fab-pane`, `_cli-agents`, `_cli-external`). None of those exist under `~`. A full user-level copy of the operator skill plus its helpers is ~1,500 lines of prose that drifts on every release — the exact anti-pattern `fab/project/code-quality.md` names (owner-or-pointer). The kit cache is machine-level and complete (`$(fab kit-path)/skills/*.md`), `fab agent operator -o yaml` resolves project-free from `~` through the config cascade, and the operator's `fab/project/*` loads are already optional (§2 Context Loading). So a ~15-line pointer that says "read the kit copy, resolve helpers from the kit copy" is sufficient, drifts nothing, and follows the same `$(fab kit-path)/…` citation style Constitution V already allows.

**Why the pointer must defer to the project copy.** Because Claude Code lets the personal skill win over a same-named project skill, the user-level file will be what `/fab-operator` runs even inside a project. Rule 1 of the pointer (read the project's `.agents/skills/fab-operator/SKILL.md` when inside a fab project) keeps the project's deployed copy governing; and even if rule 1 were skipped, `fab kit-path` inside a project resolves the project's pinned kit, so the content would be the same. Behaviour is therefore identical inside projects, and the change is additive outside them.

**Alternatives rejected (in the discussion and this intake):**

1. Full user-level copy of operator + helpers — drift on every release (above).
2. Making `rk operator -L` open the window inside a fab project — contradicts 2sdj; the operator has no owning project. No run-kit change.
3. A writer only in the `fab operator` launcher — misses the cron respawn path (rk calls `fab agent operator --print`, not `fab operator`), and the launcher is in the `src/go/fab` module while the deployment machinery is in `src/go/fab-kit`.
4. A repo-free `fab setup` writer alongside `fab sync` (the discussion's recommendation) — blocked by the module boundary. The `fab` router (`src/go/fab-kit/cmd/fab/main.go`) sends the `LifecycleCommandSet()` names (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`) to the `fab-kit` binary and everything else to the version-pinned `fab-go` binary; the only cross-binary call today is `upgrade-repo` exec'ing the pinned fab-go's `fab config upgrade` fail-open (`internal/upgrade.go` `runConfigUpgrade`). The three ways to have both writers were weighed and rejected:
   - **(a) writer in the `fab` module**, called by the `fab setup` wizard and by `fab operator` pre-launch, with `fab-kit sync` exec'ing the pinned fab-go for it — the pinned fab-go may predate the verb (a second fail-open reminder path), a two-file write would ride a subprocess, and it adds a new fab-go verb (CLI surface + docs + `help-dump` tree) for what `Sync` can do in-process with the `kitDir` and `claudeAvailable` it already holds.
   - **(b) writer in `fab-kit`, `fab setup` shelling out to `fab-kit`** — `fab-kit sync` exits 3 outside a managed repo and has no pinned kit to read there, so the shell-out would need a new lifecycle verb, which means the router allowlist (`LifecycleCommands`), both binaries' help, and `_cli-fab.md` § Calling Convention all change for a ~15-line file.
   - **(c) a duplicated writer in both modules** — the exact anti-pattern `fab/project/code-quality.md` names; two renderers of the same template drift.
   Chosen: `fab sync` is the sole writer (every `init`, `upgrade-repo`, and plain sync on the machine refreshes the file); `fab setup check` reports presence/staleness with the fix hint (§ What Changes 2–3). A repo-free writer can be added later if a machine with fab installed but no fab project ever needs the operator — reversible, and today that machine has nothing for the operator to coordinate.
5. `fab sync` outside a managed repo writing the skill before exiting 3 — mixes a side effect into the "not applicable here" exit contract and has no pinned kit to read from.
6. The pointer prose as a Go string constant — workflow prose belongs in markdown (Constitution I); a kit template keeps it reviewable and inside the portability guard's walk.

## What Changes

### 1. The user-level operator skill — content and paths

Two files, same content:

| Tier | Path | Written when |
|------|------|--------------|
| Agents dir (portable) | `$HOME/.agents/skills/fab-operator/SKILL.md` | every `fab sync` (unconditional) |
| Claude Code | `$HOME/.claude/skills/fab-operator/SKILL.md` | `claudeAvailable` — the same shared gate `deploySkills` uses (`agentAvailable("claude")`, `FAB_AGENTS` override honoured) |

No `.opencode` user tier (OpenCode reads `~/.agents/skills`). No `.gitignore` manifest at user level — the files are outside any repo, there is exactly one fixed path per tier, and nothing is pruned: sibling skills (`~/.claude/skills/shll-toolkit/`, anything the user put there) are never read, listed, or touched. When the Claude gate is closed an existing `~/.claude/skills/fab-operator/SKILL.md` is preserved, never deleted (mirrors the project-tier rule "preserves existing Claude content"). Idempotency is **rewrite-if-differs**: read the existing file, write only when the bytes differ.

The content is rendered from a new kit template, `src/kit/templates/user-skill-fab-operator.md`, with one placeholder `{DESCRIPTION}` filled from the `description:` frontmatter line of `<kitDir>/skills/fab-operator.md` (so harness discovery text matches the real skill). Proposed template (wording may be tightened at apply; the two numbered rules are normative):

```markdown
---
name: fab-operator
description: {DESCRIPTION}
---
# /fab-operator — user-level pointer

> Generated by `fab sync` — do not edit; the next sync rewrites this file. It carries no
> operator logic. It exists so `/fab-operator` resolves from any directory, including the
> home-directory window `rk operator -L <server>` opens for the operator-tick respawn.

1. **Inside a fab project** (a `fab/project/config.yaml` exists at or above the current
   directory): read `.agents/skills/fab-operator/SKILL.md` at that project's root and follow
   it — the project's deployed copy governs, including its helpers under `.agents/skills/`.
   If that file is missing, continue with 2.
2. **Otherwise**: run `fab kit-path`, read `<kit-path>/skills/fab-operator.md`, and follow it.
   Wherever that skill or `_preamble` says to read `.agents/skills/{x}/SKILL.md`, read
   `<kit-path>/skills/{x}.md` instead — the flat kit copy of the same file.
```

Frontmatter carries `name` and `description` only — no `helpers:` (the real skill declares them and rule 2 says where to load them from), no `user-invocable`/`disable-model-invocation` flags (the kit's `fab-operator.md` has none). Rule 1's predicate is deliberately "inside a fab project", not "a `.agents/skills/` directory exists up the tree": from `$HOME` the latter would find `~/.agents/skills/` and the pointer would resolve to itself.

The template cites only `$(fab kit-path)/…`, `fab/project/config.yaml`, and `.agents/skills/…` — all allowed by Constitution V; the portability guard (`src/go/fab-kit/cmd/fab/kit_portability_test.go`) walks `src/kit/**` and covers it automatically.

### 2. The writer — `fab sync` (fab-kit module), sole writer

New helper in `src/go/fab-kit/internal/skills.go`:

```go
// deployUserOperatorSkill writes the machine-level fab-operator pointer skill
// (~/.agents/skills always; ~/.claude/skills when claude is available) so the
// operator resolves from a directory with no fab project. Rewrite-if-differs;
// never prunes or lists sibling skills; no manifest. A kit without the template
// (a pin predating this feature) is a silent no-op.
func deployUserOperatorSkill(kitDir string, claudeAvailable bool) error
```

Behaviour:

1. Resolve `$HOME` via `os.UserHomeDir()` (honours `$HOME` on unix — tests pin it with `t.Setenv("HOME", tmp)`, the `cache_test.go` precedent). Failure → returned error.
2. If `<kitDir>/templates/user-skill-fab-operator.md` does not exist → return `nil` and write nothing. This is required: `Sync` runs against the **project's pinned kit** (`kitDir`), and a newer binary syncing a project pinned to an older kit must not fail.
3. Read `<kitDir>/skills/fab-operator.md`, extract the `description:` frontmatter value (line-prefix parse within the leading `---` block; missing → error). Render the template.
4. For each fired tier: `MkdirAll(dir, 0755)`; if the existing file's bytes equal the rendered bytes → skip; else write `0644`. Print one line in the existing tally style, e.g. `User-level skill (fab-operator): written` / `up to date` / `Skipping ~/.claude tier: claude not found in PATH`.
5. Return the joined write errors.

Call site: `Sync` (`sync.go`), inside `if !projectOnly`, immediately after `deployErr = deploySkills(...)`, with the result joined into `deployErr` (`errors.Join`) so a write failure fails the sync like any other deploy failure (the jznd fail-loud contract) but does not abort the remaining repair steps. `shimOnly`/`projectOnly` semantics are unchanged (the user-level skill rides the fab-kit-owned steps 1–5, not the project scripts step). Because `init.go` and `upgrade.go` call `runSync`, the file is written on first `fab init` on a machine and refreshed on every `fab upgrade-repo`.

`fab setup` (the fab-go wizard) writes **nothing** for this feature. `fab operator` (fab-go) writes nothing.

### 3. `fab setup check` reports it (fab-go module, read-only)

`src/go/fab/internal/setupcheck/probes.go` gains `ProbeUserSkills(homeDir string, lookPath LookPathFunc, kitDir string) []Finding` (check name `user-skills`), wired in `run.go` after the environment probe; `Input` gains `HomeDir string` (`""` → `os.UserHomeDir()`), mirroring the `KitDir` seam. Findings:

| Condition | Severity | Detail (shape) |
|-----------|----------|----------------|
| `~/.agents/skills/fab-operator/SKILL.md` present | OK | `user-level fab-operator skill: ~/.agents/skills` |
| … absent | Warn | `user-level fab-operator skill missing — run 'fab sync' in any fab project (the rk operator -L respawn window opens in $HOME)` |
| `claude` on PATH, `~/.claude/skills/fab-operator/SKILL.md` present | OK | as above for `~/.claude/skills` |
| `claude` on PATH, … absent | Warn | same hint |
| `claude` not on PATH | Info | `~/.claude tier not expected (claude not on PATH)` |
| a present file whose `description:` line differs from `<kitDir>/skills/fab-operator.md`'s | Warn | `user-level fab-operator skill is stale — re-run 'fab sync'` |

Warn, never Fail: a missing user-level skill breaks nothing inside projects, so the doctor's exit code (1 only on failures) is unaffected. Staleness compares only the `description:` line — the body is a version-independent pointer, and comparing the description avoids duplicating the template renderer across modules. The package header comment's "read at most one file" sentence is updated (it now also reads the user-level skill files and the kit's `fab-operator.md` frontmatter); the **no-writes invariant is untouched**. `renderSetupCheck` needs no structural change (one more check group).

### 4. Prose fallback that makes the pointer work (deployed kit, `src/kit/skills/`)

- **`_preamble.md` § Skill Helper Declaration → Semantics** (owner of the rule). One clause added: the agent MUST read `.agents/skills/{helper}/SKILL.md` for each declared helper — **or, when that deployed file does not exist (a skill launched outside any `fab sync`-deployed project, e.g. the operator from a neutral directory via its user-level pointer skill), `$(fab kit-path)/skills/{helper}.md`, the flat kit copy of the same file.** The predicate is file-existence, which is correct from `$HOME` too (`~/.agents/skills/_cli-fab-operator/SKILL.md` does not exist → kit path).
- **`fab-operator.md` §2 Startup → Context Loading**: one pointer sentence after the helpers paragraph — launched from a directory with no deployed `.agents/skills/` (how `/fab-operator` resolves there is the user-level pointer skill `fab sync` writes to `~/.agents/skills/` and `~/.claude/skills/`), the helpers load from `$(fab kit-path)/skills/{helper}.md` per `_preamble.md` § Skill Helper Declaration, and the optional `fab/project/*` loads skip. Owner-or-pointer: the rule stays in `_preamble`.
- **Line 9 of `fab-operator.md` is NOT changed**, nor line 12 of `_preamble.md`: the opening blockquote is the uniform standard line of every skill (`docs/specs/skills.md:208`); the pointer skill's rule 2 already redirects the `_preamble` read, so a per-skill deviation would be a second statement of the same rule.
- **`_preamble.md` § Path Convention**: unchanged — it governs how skill authors spell paths; `$(fab kit-path)/…` substitution is already its convention for kit assets.
- **`_cli-fab-operator.md` § fab operator, "Launch cwd" paragraph**: one sentence — from a neutral directory (including the `$HOME` window `rk operator -L <server>` opens) `/fab-operator` resolves through the user-level pointer skill `fab sync` writes.

### 5. CLI reference and setup skill (Constitution Additional Constraints — CLI ⇒ docs)

- **`_cli-fab.md` § Workspace Command Exit Semantics, `sync` row**: add the user-level deploy — `~/.agents/skills/fab-operator/SKILL.md` always, `~/.claude/skills/fab-operator/SKILL.md` under the same Claude gate; pointer to `$(fab kit-path)/skills/fab-operator.md`; rewrite-if-differs; no manifest; a pinned kit without the template is a no-op; write failure counts as a deployment failure.
- **`_cli-fab.md` § fab setup**, the `fab setup check` probe list: new bullet **User-level operator skill** (the § 3 table in one sentence; Warn/Info semantics; exit code unaffected).
- **`_cli-fab.md` § fab kit-path**: append the pointer skill to the "Used by" list (one clause).
- **`fab-setup.md` § 1c `fab sync`** bullet list: new bullet **User-level operator skill** (paths, gate, pointer). The `{fab sync report …}` line gains `user-level fab-operator skill`.

### 6. Governance and project docs (`fab/project/`)

- **`constitution.md` Principle V**: after "Both deployed copies are ignored via the generated per-target `.gitignore` manifests `fab sync` writes." add one sentence: *The machine additionally carries one user-level skill — the `fab-operator` pointer `fab sync` writes to `~/.agents/skills/fab-operator/SKILL.md` (always) and `~/.claude/skills/fab-operator/SKILL.md` (when `claude` is available) — so the cross-repo operator resolves from a directory with no project; it points at `$(fab kit-path)/skills/` and carries no copied prose.* The deployed-content parenthetical needs no change (the template lives in `src/kit/templates/`, already listed).
- **`constitution.md` Additional Constraints, canonical-source bullet**: one clause — the user-level `fab-operator` pointer under `~/` is likewise generated by `fab sync` and never edited directly.
- **Governance**: wording-only, no MUST added or narrowed → **no version bump** (udwv / jjg0 / t513 / yd9s / si4k precedent); append the dated HTML comment `<!-- 2026-09-13 (260913-ga0v): … -->`; update **Last Amended** to the apply date.
- **`context.md` § Distribution** and **§ Skills**: one sentence each naming the machine-level tier and that `src/kit/templates/user-skill-fab-operator.md` is its canonical source.
- **`code-quality.md` § Anti-Patterns (project-specific), "Editing deployed skills directly"**: add the user-level files to the never-edit list (same sibling class).

### 7. Tests (Constitution Additional Constraints — Go changes ship tests)

`src/go/fab-kit/internal/skills_test.go` (HOME pinned per test with `t.Setenv("HOME", t.TempDir())`; a fixture kit with `skills/fab-operator.md` carrying a known `description:` and `templates/user-skill-fab-operator.md`):

- `TestDeployUserOperatorSkill_BothTiersWhenClaude` — both files exist; each contains `name: fab-operator`, the fixture description verbatim, and the literal `fab kit-path`; no `.gitignore` in either directory.
- `TestDeployUserOperatorSkill_AgentsTierOnlyWithoutClaude` — only `~/.agents/skills/…`; `~/.claude` is not created.
- `TestDeployUserOperatorSkill_ClaudeGateClosedPreservesExisting` — a pre-existing `~/.claude/skills/fab-operator/SKILL.md` survives a closed-gate run byte-for-byte.
- `TestDeployUserOperatorSkill_Idempotent` — second run writes nothing (mtime unchanged or the `up to date` line; content equal).
- `TestDeployUserOperatorSkill_NeverTouchesSiblings` — pre-seeded `~/.claude/skills/shll-toolkit/SKILL.md` and `~/.agents/skills/other/SKILL.md` are untouched.
- `TestDeployUserOperatorSkill_KitWithoutTemplateIsNoop` — old-kit fixture → `nil`, nothing written.
- `TestDeployUserOperatorSkill_MissingDescriptionErrors`.
- A `Sync`-level wiring assertion where the existing harness allows (`sync_integration_test.go` pattern); otherwise the call-site is covered by reading `Sync` in review.

`src/go/fab/internal/setupcheck/setupcheck_test.go`: table test for `ProbeUserSkills` (present / absent / stale / claude-gated Info) using `HomeDir` + `LookPath` seams and a fixture kit. `src/go/fab/cmd/fab/setup_test.go`: the rendered `setup check` output includes the new check and the exit code stays 0 on a Warn-only report.

Guards: `cd src/go/fab-kit && go test ./cmd/fab/` (portability guard over the new template) and `go test ./internal/`; `cd src/go/fab && go test ./internal/setupcheck/ ./cmd/fab/ -run 'Setup|UserSkill'`. Run `gofmt` on worker-written Go before ship.

### 8. Memory (hydrate) and human-curated docs

Memory edits are hydrate's; the sites are enumerated under § Affected Memory. Human-curated sweep sites included in apply (owner-or-pointer; the deploy-target class per `code-quality.md` § Sibling Sweeps):

- `docs/specs/architecture.md` § deploy-target table (lines ~495–501): one row **User (machine)** — `~/.agents/skills/fab-operator/SKILL.md` always, `~/.claude/skills/fab-operator/SKILL.md` when `claude` is available — pointer to `$(fab kit-path)/skills/fab-operator.md`; and `docs/specs/overview.md:8` gains a clause. These are the change's own design intent, not tooling-generated content (Constitution VI).
- `README.md:283`, `docs/site/install.md:122`, `docs/site/skill.md:113`: one clause each that `fab sync` also writes the machine-level `fab-operator` pointer skill. Within existing bullets — no structural change, so the `readme-extraction` toolkit standard is not engaged.
- `fab/backlog.md` `[ga0v]` is ticked at archive time by `/fab-archive` (not in apply).

### Not in scope

- No run-kit change; rk's cwd rule for `-L` and its typed kickoff stay as they are.
- No per-project shim; no user-level deploy of any other skill or helper (only `fab-operator` is ever launched from a neutral cwd).
- No writer in `fab setup`, `fab operator`, or a repo-free `fab sync` path (module boundary; see Why).
- No migration: no existing user data is restructured; the file appears on the next sync/upgrade.
- No Constitution version bump; no change to the `fab sync --shim` flag or the router's allowlist.

## Affected Memory

- `distribution/kit-architecture`: (modify) § Agent Skill Deployment — new paragraph *User-level operator skill* (two paths, shared Claude gate, pointer body rendered from `templates/user-skill-fab-operator.md`, rewrite-if-differs, no manifest, siblings untouched, pre-feature pin = no-op, failures join `deployErr`); § Directory Structure `templates/` entry lists the new template; DD *Agent Skill Deployment Strategy* gains `*Updated by*`; new DD *The User-Level Operator Skill Is a Pointer, Not a Copy* (Decision / Why: drift + Claude Code personal-over-project precedence + module boundary → sync is the sole writer / Rejected: full copy, rk cwd change, launcher writer, `fab setup` writer, Go string template / *Introduced by* 260913-ga0v) — the owner of the version-skew tradeoff (the pointer is written from the syncing project's pinned kit; only the `description:` can vary; `fab kit-path` at run time resolves the installed fab outside a project and the pin inside one).
- `distribution/setup`: (modify) § Setup-State Doctor probe list gains the `user-skills` probe (Warn/Info semantics, description-line staleness, exit unaffected); § Delegation Pattern table row "Skill deployment" gains the user tier; any restatement of "reads at most one file".
- `distribution/distribution`: (modify) § Sync Staleness Detection / § Update sentences that enumerate what `fab sync` and `fab upgrade-repo` refresh (lines ~194, ~213) gain the user-level skill.
- `runtime/operator`: (modify) § Launch preconditions paragraph (the neutral-cwd / `$HOME` window now resolves `/fab-operator` through the user-level pointer skill; the cron `rk operator -L` respawn produces a working operator); DD *Operator Launch Is Git-Optional, `fab/`-Optional, and Operator-Role-Resolved* gains `*Updated by*: 260913-ga0v` with a pointer to the kit-architecture DD (no second DD — deployment is kit-architecture's domain).
- `_shared/context-loading`: (modify) § Skill Helper Declaration (Opt-In), the "MUST read `.agents/skills/{helper}/SKILL.md`" sentence gains the kit-path fallback clause (restates the preamble rule → sibling sweep).

Not affected (checked): `memory-docs/templates.md:112` and `pipeline/planning-skills.md:106,127` describe internal-helper source-vs-deployed layout, not the deploy-target set; `runtime/agent-primitives.md:26` is a scenario step inside a project. `docs/memory/*/index.md` regenerate via `fab docs-index` only if a `description:` changes.

## Impact

- **Go (fab-kit module)**: `src/go/fab-kit/internal/skills.go` (+`deployUserOperatorSkill`, description parse), `sync.go` (call site, one line + error join), `skills_test.go` (7 tests). **Go (fab module)**: `src/go/fab/internal/setupcheck/probes.go` (+`ProbeUserSkills`), `run.go` (`Input.HomeDir`, wiring), `setupcheck.go` (header comment), `setupcheck_test.go`, `src/go/fab/cmd/fab/setup_test.go`. Two modules, two `go test` runs.
- **Kit**: new `src/kit/templates/user-skill-fab-operator.md`; `src/kit/skills/_preamble.md`, `fab-operator.md`, `_cli-fab-operator.md`, `_cli-fab.md`, `fab-setup.md` (one clause/sentence/bullet each).
- **Governance**: `fab/project/constitution.md` (two sentences + dated comment, no bump), `context.md`, `code-quality.md`.
- **Human-curated docs**: `docs/specs/architecture.md`, `docs/specs/overview.md`, `README.md`, `docs/site/install.md`, `docs/site/skill.md` (one row/clause each).
- **Memory**: five files at hydrate (above).
- **Behaviour change for users**: after the next `fab sync`/`fab init`/`fab upgrade-repo` on a machine, `/fab-operator` resolves from any directory; `fab setup check` gains one check group. Inside projects nothing observable changes (rule 1). `fab sync` prints one extra tally line.
- **Deployed content**: every touched `src/kit/**` file deploys into customer repos — cite only kit skills, `fab` commands, `fab/project/` files, `$(fab kit-path)/…`, and host convention paths (Constitution V; the Go guard fails otherwise).
- **Lane**: Go in two modules + template + five skill files + governance + specs → FULL lane (well over 5 tasks).
- **Manual acceptance** (post-ship, on this machine): (a) `fab sync` in this repo twice — second run reports `up to date`, both `~/.agents/skills/fab-operator/SKILL.md` and `~/.claude/skills/fab-operator/SKILL.md` exist, `~/.claude/skills/shll-toolkit/` untouched; (b) from `~` inside tmux, `rk operator` and `rk operator -L <server>` each land a window whose typed `/fab-operator` resolves, and the agent reads `_preamble` + the four helpers from `$(fab kit-path)/skills/`; (c) from a project cwd, `/fab-operator` follows rule 1 into the project's `.agents/skills/` copy; (d) `fab setup check` from `~` shows the `user-skills` group OK, and Warn after deleting one tier's file.

## Open Questions

- None blocking. Whether `docs/specs/operator.md` should record this as a version row is a human curation call outside the change (Constitution VI).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship a user-level **pointer** skill (frontmatter `name: fab-operator` + the kit skill's `description:`; body = the two numbered rules), never a copy of operator prose or helpers | Discussed — the user accepted the recommendation; code-quality.md owner-or-pointer; the full-copy alternative rejected in conversation | S:95 R:80 A:90 D:95 |
| 2 | Certain | Two tiers under the project-level rule: `~/.agents/skills/fab-operator/SKILL.md` always, `~/.claude/skills/fab-operator/SKILL.md` on the shared `claudeAvailable` gate; no `.opencode` tier; no manifest/`.gitignore`; rewrite-if-differs; siblings never touched; a closed Claude gate preserves an existing file | Discussed — t513/yd9s project rule and the `shll setup agent` precedent; project-tier "preserve existing Claude content" contract | S:90 R:85 A:90 D:90 |
| 3 | Confident | `fab sync` (fab-kit module, `deployUserOperatorSkill` in `internal/skills.go`, called from `Sync` after `deploySkills`, errors joined into `deployErr`) is the **sole** writer; `fab setup` and `fab operator` write nothing; `fab setup check` reports — deviating from the discussion's "shared helper called from both `fab sync` and `fab setup`" | Verified: `sync` lives in `src/go/fab-kit`, `setup`/`operator` in `src/go/fab`; the two modules do not import each other and Go `internal/` cannot cross the boundary. Options weighed in § Why alternative 4: (a) writer in the `fab` module exec'd from `fab-kit sync` via the pinned fab-go (new fab-go verb, subprocess for a two-file write, second fail-open path); (b) writer in `fab-kit` with `fab setup` shelling out (needs a new lifecycle verb → router allowlist, both helps, `_cli-fab.md`; `sync` exits 3 outside a repo); (c) duplicated writer (code-quality anti-pattern). `init`/`upgrade-repo` call `Sync`, so coverage is every bootstrap and upgrade; a repo-free writer is a reversible later addition | S:70 R:75 A:80 D:60 |
| 4 | Certain | Pointer rule 1 defers to the project's `.agents/skills/fab-operator/SKILL.md` when inside a fab project (`fab/project/config.yaml` up the tree), rule 2 reads `$(fab kit-path)/skills/…`; the backlog's acceptance "a project cwd resolves the project copy first" is restated as "project semantics are preserved" | Verified in Claude Code's docs: personal skills outrank project skills for a same-named `/name`; the predicate "inside a fab project" avoids the self-reference `~/.agents/skills/` would cause from `$HOME`; inside a project `fab kit-path` is the pin anyway | S:75 R:80 A:85 D:80 |
| 5 | Confident | The pointer's prose is a kit template `src/kit/templates/user-skill-fab-operator.md` with a `{DESCRIPTION}` placeholder filled from `<kitDir>/skills/fab-operator.md`; a pinned kit lacking the template is a silent no-op | Not discussed; Constitution I (prose in markdown), `templates/` already holds binary-filled templates (`status.yaml`), the portability guard walks `src/kit/**`; `Sync` reads the project's pinned kit so an older pin must not fail; alternatives: Go string constant (prose in Go), a new `src/kit/user-skills/` dir (new kit dir → Constitution V list + directory docs for one file), `skills/` (auto-deploys project-level) | S:40 R:80 A:70 D:55 |
| 6 | Certain | `_preamble.md` § Skill Helper Declaration (owner) gains one file-exists-else-`$(fab kit-path)/skills/{helper}.md` clause; `fab-operator.md` §2 Context Loading gets one pointer sentence; the standard line-9 blockquote and `_preamble.md` line 12 stay verbatim in every skill | Discussed (fallback clause requested); the blockquote is the uniform standard opening line (`docs/specs/skills.md:208`) and rule 2 of the pointer already redirects the `_preamble` read — owner-or-pointer forbids a second statement | S:80 R:85 A:85 D:70 |
| 7 | Confident | `fab setup check` gains a `user-skills` probe (`ProbeUserSkills`, `Input.HomeDir` seam): OK/Warn per tier, Info when `claude` is absent, Warn on a stale `description:` line; never Fail; no writes; header comment updated | Discussed ("`fab setup check` should report presence/staleness"); Warn keeps the exit contract untouched; description-only staleness avoids duplicating the renderer across modules | S:65 R:85 A:80 D:65 |
| 8 | Certain | Version skew accepted and recorded as a DD: the pointer is written from the syncing project's pinned kit (only `description:` can vary between pins); the body resolves `fab kit-path` at run time — the installed fab outside a project, the pin inside one | Discussed — "accepted tradeoff … right for the operator, which spans repos and has no owning project (2sdj)"; verified kit-path behaviour in both contexts | S:85 R:80 A:85 D:85 |
| 9 | Certain | Constitution V + the canonical-source constraint gain one sentence/clause each, dated comment, **no version bump**, Last Amended updated; `context.md` § Distribution/§ Skills and `code-quality.md` anti-pattern gain one clause each | Discussed ("wording-only, no new MUST → no version bump, per the udwv/jjg0/t513 precedent"); the deployed-content parenthetical already lists `src/kit/templates/` | S:80 R:90 A:85 D:80 |
| 10 | Certain | CLI ⇒ docs: `_cli-fab.md` `sync` row + § fab setup probe list + § fab kit-path clause; `_cli-fab-operator.md` § fab operator launch-cwd sentence; `fab-setup.md` § 1c bullet | Constitution Additional Constraints; `_cli-fab.md` owns `sync`/`setup`/`kit-path`, `_cli-fab-operator.md` owns `fab operator` | S:75 R:90 A:90 D:85 |
| 11 | Certain | Tests as enumerated in § 7, HOME pinned via `t.Setenv`, portability guard and both modules' packages green, `gofmt` before ship | Constitution VII + code-quality test-alongside; `cache_test.go` HOME precedent; the l0bf lesson (worker-written Go not formatted) | S:80 R:90 A:90 D:90 |
| 12 | Confident | Sweep the deploy-target class in human-curated docs: `docs/specs/architecture.md` table row, `docs/specs/overview.md` clause, `README.md` / `docs/site/install.md` / `docs/site/skill.md` one clause each; `docs/specs/operator.md` untouched | code-quality.md § Sibling Sweeps (aggregate specs restating per-skill facts); Constitution VI allows a change's own design intent in specs; one clause inside existing bullets does not engage the `readme-extraction` standard's structure rules | S:50 R:90 A:70 D:70 |
| 13 | Confident | Vocabulary: "user-level operator skill" / `deployUserOperatorSkill` in code, CLI output, and docs; "shim" only in the title/slug with a disambiguation | Not discussed; "the shim" already names the `fab` router binary in `_cli-fab.md` and `fab sync --shim` is an existing flag — a second meaning would confuse every reader of the sync docs | S:40 R:95 A:80 D:75 |
| 14 | Confident | Sync output: one tally-style line per run (`written` / `up to date` / Claude-tier skip); an unresolvable `$HOME` or a write failure joins `deployErr` and fails the sync after the remaining repair steps | Not discussed; matches `deploySkills`' per-target tally and the jznd fail-loud deployment contract | S:45 R:90 A:75 D:60 |
| 15 | Certain | Non-goals hold: no run-kit change, no per-project shim, no other user-level skill, no `fab setup`/`fab operator`/repo-free `fab sync` writer, no migration, no router/allowlist change | Discussed — the user's non-goal list, plus the module-boundary finding | S:90 R:90 A:90 D:90 |
| 16 | Confident | Codex/OpenCode project-vs-user precedence for `.agents/skills` is undocumented in the repo and left undocumented; pointer rule 1 makes it immaterial | Verified by repo grep (only per-brand-vs-generic shadowing is documented); inside a project both resolutions yield the pinned kit's content | S:50 R:90 A:60 D:70 |
| 17 | Confident | `_preamble.md` § Path Convention is left unchanged | The convention governs how skill authors spell paths; `$(fab kit-path)/…` is already its form for kit assets; the operator's own Context Loading tolerates a project-less cwd | S:40 R:95 A:70 D:60 |

17 assumptions (9 certain, 8 confident, 0 tentative, 0 unresolved).
