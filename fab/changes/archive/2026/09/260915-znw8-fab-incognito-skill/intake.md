# Intake: fab-incognito — Discussion-Priming Skill That Loads Process Knowledge, Not Project Memory

**Change**: 260915-znw8-fab-incognito-skill
**Created**: 2026-09-15

## Origin

Conversational — a `/fab-discuss`-style redesign conversation, dispatched promptless via `/fab-proceed`
(create-new path, `{questioning-mode} = promptless-defer`). The user's synthesized request:

> Add a new standalone user-invocable skill, `fab-incognito`, at canonical source
> `src/kit/skills/fab-incognito.md` (deployed by `fab sync` like every other skill). It is a
> discussion-priming skill, sibling to `fab-discuss`, that loads the fab **process knowledge** (how a fab
> task is supposed to be performed — the kit's helper partials and skill catalog) but deliberately does
> NOT load the host project's `docs/memory/*` or `docs/specs/*`. Purpose: when redesigning the existing
> system, present-truth memory anchors the agent to defend the status quo and pattern-match proposals onto
> the current structure; incognito anchors the discussion on the kit's operative behavior (the skills)
> instead of the documented narrative (memory/specs).

Key decisions reached in the conversation (all user-confirmed; encoded in `## Assumptions`):

1. **Standalone skill, named `fab-incognito`.** The assistant recommended a flag on fab-discuss
   (`/fab-discuss --incognito`); the user rejected it and chose a standalone skill. Candidate names
   `fab-rethink`, `fab-fresh`, `fab-kit-discuss` were rejected. The name evokes browser incognito — a fresh
   session that ignores stored state; "no memory" doubles as "skip `docs/memory`".
2. **Read-blind, not trace-free.** The skill does not *read* project memory/specs, but it still logs its
   invocation (`fab log command "fab-incognito"`) and later skills in the session may write artifacts.
3. **Standing session rule** — after loading, the agent MUST NOT open `docs/memory/*` or `docs/specs/*`
   (including the two index files) for the remainder of the discussion unless the user names a specific
   file. This rule is the real lever: fab-discuss already loads only the two index files, so the load-time
   difference alone is small; what "gets in the way" is the agent following Memory File Lookup mid-discussion.
4. **What is loaded**: the `fab/project/` layer, the six process helper partials in full, a skill catalog
   (frontmatter `name` + `description` of every deployed skill), and the kit version. Skill bodies and the
   CLI reference partials are lazy-loaded on demand.
5. **Active change**: resolved like fab-discuss (`fab resolve --folder --or-none`), no artifacts, no preflight.
6. **Orientation output** mirrors fab-discuss's shape, adds `Kit:`, helpers, catalog count, and an explicit
   "Memory and specs: NOT loaded" line; ends with a ready signal, not a `Next:` line.

Alternatives rejected: flag on fab-discuss (user chose standalone); loading every skill body up front
(~834 KB deployed total — "clean context" is a misnomer; the skill is a *different anchor*, not less context,
so it must not eat everything up front); names `fab-rethink` / `fab-fresh` / `fab-kit-discuss`.

## Why

**The pain point.** `/fab-discuss` is the session entry point for exploratory conversation. It loads the
always-load layer (`_preamble.md` §1): `fab/project/*` plus `docs/memory/index.md` and `docs/specs/index.md`.
From there the agent follows `_preamble.md` § Memory File Lookup whenever a topic touches a domain — it opens
the domain index, then the memory file, and grounds its reasoning in *present truth*: what shipped, why, and
which design decisions were made. That is exactly right for incremental work. It is exactly wrong for
redesign: present-truth memory anchors the agent to the status quo. Proposals get pattern-matched onto the
current structure, existing Design Decisions are cited as reasons not to change, and the conversation
converges on "how the system already works" instead of "how the process should work".

**The observation behind the fix.** The fab *process* — how a task is supposed to be performed — is not in
`docs/memory/`. It is in the deployed skill set: the six helper partials (`_preamble`, `_pipeline`,
`_intake`, `_srad`, `_generation`, `_review`) and the user-invocable skills that compose them. Memory and
specs are the *narrative about* the system; the skills are its *operative behavior*. A redesign discussion
should be anchored on the latter.

**Why not just tell the agent "don't read memory"?** Because the instruction has to survive the whole
session and override a written convention. `_preamble.md` §1 says the always-load layer applies "unless the
skill's own Context Loading section says otherwise — the skill file wins", and § Memory File Lookup is
invoked by habit in every skill. Only a skill file can declare the override durably, in the place the
preamble itself points to. A one-off chat instruction is forgotten at the next context compaction; a skill's
Context Loading section is re-readable and cites the preamble's own escape hatch.

**Why a standalone skill rather than a fab-discuss flag.** The user decided this. The supporting argument:
the two skills have opposite load profiles (fab-discuss loads the doc indexes and *nothing* about the kit;
incognito loads ~120 KB of kit process partials and *nothing* from the docs), opposite ready signals, and a
behavioral rule that persists beyond the invocation. A flag would make fab-discuss's Context Loading section
conditional — the very section the preamble tells readers to consult for the override — and would hide the
"no memory" affordance behind an argument nobody discovers. A sibling skill with its own name is
discoverable in `/fab-help` and self-describing in its frontmatter.

**Consequence of not doing it.** Every redesign conversation starts by re-explaining the process from the
skills and repeatedly telling the agent not to open memory. The conversation's context fills with
present-truth narrative that defends what exists; redesign proposals stay incremental.

## What Changes

### 1. New skill `src/kit/skills/fab-incognito.md`

A new user-invocable skill, canonical source `src/kit/skills/fab-incognito.md`, deployed by `fab sync` to
`.agents/skills/fab-incognito/SKILL.md` (and `.claude/skills/fab-incognito/SKILL.md` when `claude` is
available). Follows the fab-discuss / fab-help header pattern.

**Frontmatter** — `name`, `description`, and a `helpers:` list declaring the five loadable process partials
(`_preamble` is implicit and never listed; the five below are all allowed values per `_preamble.md` § Skill
Helper Declaration):

```yaml
---
name: fab-incognito
description: "Prime the agent with fab process knowledge for a redesign discussion — loads the kit's helper partials and a skill catalog, deliberately skips docs/memory and docs/specs, and holds a standing rule not to open them for the rest of the session. Use when rethinking how the fab process should work rather than how the current system does work. Read-blind, not trace-free: the invocation is still logged."
helpers: [_pipeline, _intake, _srad, _generation, _review]
---
```

**Body sections** (mirroring fab-discuss's section set):

- `# /fab-incognito` heading, then the standard opener line: ``Read the `_preamble` skill first (deployed to `.agents/skills/` via `fab sync`). Then follow its instructions before proceeding.``
- `## Purpose` — one paragraph: prime for a *redesign* discussion by loading the kit's operative behavior
  (process partials + skill catalog) instead of the project's documented narrative (memory/specs). State the
  incognito metaphor and its inversion in one sentence: browser incognito means "don't record history";
  this skill means "don't *read* history" — the invocation is logged and later skills may still write
  artifacts (read-blind, not trace-free). Read-only, no artifact generation, no stage advancement.
- `## Arguments` — None.
- `## Context Loading` — the **override** section (this is what makes `_preamble.md` §1's "the skill file
  wins" derivation work). Explicit text:

  1. **Always-load override**: load `fab/project/config.yaml` and `fab/project/constitution.md`
     (required — error if missing) and `fab/project/context.md`, `fab/project/code-quality.md`,
     `fab/project/code-review.md` (optional — skip gracefully). Do **NOT** load `docs/memory/index.md` or
     `docs/specs/index.md`. This skill declares the override the preamble's §1 provides for; it loads a
     reduced 5-file `fab/project/` set.
  2. **Process helpers**: the frontmatter `helpers:` list loads `_pipeline`, `_intake`, `_srad`,
     `_generation`, `_review` in full (plus `_preamble`, loaded universally) — six helper partials,
     roughly 120 KB together. These are the fab process knowledge and are the primary payload.
  3. **Skill catalog**: for every deployed `.agents/skills/*/SKILL.md`, read only the frontmatter `name`
     and `description` (not the body). Include helper partials (`_*`) and `internal-*` skills, marking
     `user-invocable: false` entries as helpers. Suggested mechanism (verify during apply; any equivalent
     that reads only frontmatter is fine):

     ```bash
     for f in .agents/skills/*/SKILL.md; do
       awk '/^---$/{c++; next} c==1 && /^(name|description):/' "$f"
     done
     ```

     If `.agents/skills/` is missing or empty, STOP with: `Deployed skills not found — run fab sync.`
  4. **Kit version**: `cat "$(fab kit-path)/VERSION"` — the same file `fab fab-help` reads
     (`readKitVersion`). Verified during intake: the file exists and prints `2.26.3` in this repo while
     the binary is `2.28.0`, which is precisely the lag the user wants surfaced (in fab-kit's own repo the
     deployed skills are the *released* kit, not HEAD). Fall back to `unknown` if the file is missing, as
     `fab fab-help` does.
  5. **Lazy loads (on demand, never up front)**: full skill bodies (`.agents/skills/<name>/SKILL.md`) and
     the CLI reference partials `_cli-fab`, `_cli-fab-pane`, `_cli-fab-operator`, `_cli-agents`,
     `_cli-external` (~274 KB together) — open one only when the discussion turns to that skill or command
     family. State the size rationale: loading every deployed skill is ~834 KB versus the ~36 KB
     always-load layer; incognito is a *different anchor*, not less context, so it must not consume the
     window up front.
  6. **Active change**: run `fab resolve --folder --or-none`. `(none)` ⇒ "No active change" (expected, not
     an error; a non-zero exit is a real error — surface it per `_preamble.md`'s failure rule). If a folder
     prints, read `fab/changes/{name}/.status.yaml` and derive the stage from its `progress` map (the stage
     holding `active` or `ready`, `failed` for a parked review/review-pr; all `done`/`skipped` ⇒ complete).
     Do **not** load change artifacts (intake, plan). Do **not** run preflight.

- `## Standing Session Rule` — its own section, stated as an output/behavior obligation, not a loading
  note. Verbatim intent:

  > For the remainder of this discussion, do **NOT** open any file under `docs/memory/` or `docs/specs/`
  > — including `docs/memory/index.md` and `docs/specs/index.md` — and do **NOT** follow `_preamble.md`
  > § Memory File Lookup. The single exception: the user names a specific file; open exactly that file,
  > without walking its domain index. Reason about the process from the loaded helpers and the skill
  > catalog; when a claim about current behavior is needed, cite the skill that implements it, not the
  > memory that describes it. This rule binds the *discussion*. A skill the user invokes later
  > (`/fab-new`, `/fab-proceed`, …) follows its own Context Loading section — the skill file wins — and
  > is not constrained by this rule.

- `## Command Logging` — after context loading:

  ```bash
  fab log command "fab-incognito"
  ```

  With one sentence naming the read-blind-not-trace-free contract: logging is deliberate; incognito
  governs what is *read*, not what is *recorded*.
- `## Behavior` — numbered: (1) load the `fab/project/` layer (skip optional files gracefully), (2) load
  the six helpers via `helpers:`, (3) build the skill catalog, (4) read the kit version, (5) resolve the
  active change, (6) log the command, (7) output the Orientation Summary, (8) hold the Standing Session
  Rule for the rest of the conversation.
- `## Orientation Summary` — the output block:

  ```
  Project: {name} — {description}
  Kit: {version}   (deployed skills read from .agents/skills/)

  Process helpers loaded:
    _preamble, _pipeline, _intake, _srad, _generation, _review

  Skill catalog: {N} skills ({U} user-invocable, {H} helpers)

  Project files: {loaded list} / not found: {missing optional list}

  Memory and specs: NOT loaded (incognito — will not be opened unless you name a file)

  Active change: {name} (stage: {stage})  — or "No active change"

  Ready to discuss the system, incognito. What would you like to rethink?
  ```

  Bullets under it: counts come from the catalog scan; `Kit:` comes from `$(fab kit-path)/VERSION`; the
  memory/specs line is mandatory and verbatim; ends with the ready signal, not a `Next:` line (a
  documented opt-out per `_preamble.md` § Next Steps Convention — the skill file wins, like
  `/fab-discuss`'s ready signal).
- `## Key Properties` table:

  | Property | Value |
  |----------|-------|
  | Arguments | None |
  | Requires active change? | No |
  | Runs preflight? | No |
  | Read-only? | Yes — modifies no files |
  | Idempotent? | Yes |
  | Advances stage? | No |
  | Outputs `Next:` line? | No — ends with the incognito ready signal |
  | Loads `docs/memory/*` / `docs/specs/*`? | **No** — and holds the Standing Session Rule afterwards |

**Portability (Constitution V)**: the skill cites only kit skills, `fab` commands, `fab/project/` files,
`$(fab kit-path)/…`, and the host convention paths `docs/memory/index.md` / `docs/specs/index.md` (both on
the guard's allow-list); glob spellings `docs/memory/*` / `docs/specs/*` carry no `.md` suffix and are not
matched by the guard regex. It restates the Memory File Lookup override in its own words rather than
pointing at a fab-kit doc (owner-or-pointer: this skill *owns* the incognito rule; `_preamble.md` §1 already
owns the "skill file wins" derivation and is cited, not restated).

### 2. `/fab-help` grouping — `src/go/fab/cmd/fab/fab_help.go` + test

`fab-help.md` holds no command list; `fab fab-help` scans deployed skill frontmatter, so the new skill
appears automatically — but under the fallback `Other` group unless mapped. `skillToGroupMap` is a
hardcoded skill-list constant, so this is the one permitted Go touch (the description's carve-out): add

```go
"fab-incognito":       "Start & Navigate",
```

next to `"fab-discuss"`, and add `"fab-incognito"` to `expectedMapped` in
`TestFabHelp_GroupMapping` (`fab_help_test.go`). Not a CLI command-signature change ⇒ no `_cli-fab*.md`
update. Run `go test ./src/go/fab/cmd/fab/ -run 'FabHelp'` scoped first.

### 3. Sibling cross-reference — `src/kit/skills/fab-discuss.md`

One line in `## Purpose` (or a short `> See also` note under it): "For a *redesign* discussion that should
not be anchored on present-truth memory, use `/fab-incognito` — it loads the kit's process helpers and skill
catalog instead of the doc indexes." Nothing else in fab-discuss changes.

### 4. Sibling sweep — every surface that enumerates or singles out `fab-discuss`

Per `fab/project/code-quality.md` § Sibling Sweeps, the whole class is swept up front (grep
`fab-discuss` repo-wide at apply; the list below is the intake-time snapshot):

| Surface | Edit |
|---------|------|
| `docs/specs/skills.md` | New `## /fab-incognito` per-skill section next to `## /fab-discuss` (line ~1040), same shape: purpose, flow skeleton (`fab log command "fab-incognito"`, `fab resolve --folder --or-none`, `cat "$(fab kit-path)/VERSION"`, catalog scan), Key properties. Also the Next-line opt-out examples at lines ~181/~210 gain `/fab-incognito`. |
| `docs/specs/glossary.md` | New row beside `/fab-discuss` (line ~76): standalone read-blind-not-trace-free discussion primer; skill catalog; Standing Session Rule. |
| `docs/specs/overview.md` | Quick-reference row beside `/fab-discuss` (line ~103): `Prime agent with fab process knowledge, skipping memory/specs \| — (read-only)`. |
| `docs/specs/user-flow.md` | Command map: add `/fab-incognito` beside `/fab-discuss` where commands are enumerated (line ~42 is a flow edge — add only if the map lists entry commands; do not invent a new flow edge). |
| `README.md` | Command table row (line ~444) beside `/fab-discuss`; the "run `/fab-discuss` to orient" onboarding sentence (line ~148) gains an "or `/fab-incognito` to rethink the process" clause. The stage-coverage matrix (mermaid at ~515–629) and `docs/img/stage-coverage.svg` are **not** given a new column — incognito covers the same "context" cell as fab-discuss; add a one-line legend note instead. |
| `src/kit/skills/_preamble.md` | § Always Load's derived-exception examples: `/fab-incognito` joins `/fab-setup` / `/docs-hydrate-memory` / `/fab-operator` as a named override example (it loads a reduced 5-file `fab/project/` set and skips both doc indexes). § Next Steps Convention's opt-out example list (line ~215) gains `/fab-incognito`'s ready signal. |
| `src/kit/skills/code-dedupe.md`, `code-reorg.md` | The "like `/fab-discuss`'s ready signal" analogies — leave as-is (one example suffices; adding a second is noise). Listed so the sweep consciously skips them. |
| `docs/specs/config.md`, `docs/specs/findings/*`, `docs/memory/**/log*.md`, `planning-skills.md` line ~329 | Historical / session-log mentions — untouched. |

### 5. Memory (hydrate)

See `## Affected Memory`. No new domain; the new skill is documented where the always-load exception set
and the no-preflight skill list already live.

### Non-goals

- No change to fab-discuss's own loading or output beyond the one cross-reference line.
- No new `fab` CLI verb, flag, or config key; no migration (no user-data restructuring).
- No mechanism to *enforce* the Standing Session Rule (hooks, file-access denial) — it is a prompt-level
  obligation, consistent with Constitution I.
- Not a "clean context" mode: the skill deliberately loads ~120 KB of process partials.

## Affected Memory

- `_shared/context-loading`: (modify) the always-load exception set gains `/fab-incognito` (reduced
  `fab/project/` set, both doc indexes skipped, six helpers via `helpers:`, skill catalog, Standing Session
  Rule that suspends Memory File Lookup for the session); the "special case: `/fab-discuss` is *not* an
  exception … the only skill whose entire purpose is to surface that layer" paragraph (line ~202) is
  reworded now that a sibling exists whose purpose is to surface the *helper* layer instead.
- `pipeline/preflight`: (modify) the list of skills that operate without an active change and do not run
  preflight (line ~69) gains `/fab-incognito`.

## Impact

**Files touched**

- New: `src/kit/skills/fab-incognito.md`
- Modified (kit): `src/kit/skills/fab-discuss.md` (1 line), `src/kit/skills/_preamble.md` (2 example
  lists)
- Modified (Go, skill-list constant only): `src/go/fab/cmd/fab/fab_help.go`,
  `src/go/fab/cmd/fab/fab_help_test.go`
- Modified (docs): `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/overview.md`,
  `docs/specs/user-flow.md`, `README.md`
- Modified (memory, at hydrate): `docs/memory/_shared/context-loading.md`,
  `docs/memory/pipeline/preflight.md`
- Not modified: `.agents/skills/`, `.claude/skills/` (regenerated by `fab sync`); gitignore manifests
  (directory-derived); `_cli-fab*.md` (no command-signature change); `docs/img/stage-coverage.svg`.

**Tests / guards**

- `TestFabHelp_GroupMapping` (`src/go/fab/cmd/fab/fab_help_test.go`) — extended with the new name.
- `TestKitContentCitesNoRepoLocalPaths` (`src/go/fab-kit/cmd/fab/kit_portability_test.go`) — must stay
  green: the new skill cites only allow-listed convention paths and `.md`-less globs.
- After apply, `fab sync` must deploy the new skill and `fab fab-help` must list it under
  "Start & Navigate" (not "Other").

**Constitution touchpoints**: I (prompt play — Go touch limited to the existing skill-list constant),
III (idempotent, read-only), V (portable citations; canonical source in `src/kit/`; deployed copies never
edited), Additional Constraints (no CLI signature change ⇒ no `_cli-fab.md` update).

**Scale**: one new ~150-line skill file, ~15 one-to-five-line edits across 11 files, 2 memory files at
hydrate. Light lane expected.

## Open Questions

- None. Every decision is recorded in `## Assumptions`; no rows were deferred.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Standalone skill `fab-incognito` at `src/kit/skills/fab-incognito.md`, not a `--incognito` flag on fab-discuss | Discussed — assistant recommended the flag, user rejected it and chose standalone; opposite load profiles and a persisting behavioral rule justify a separate skill file | S:95 R:70 A:90 D:95 |
| 2 | Certain | Name `fab-incognito`; `fab-rethink`, `fab-fresh`, `fab-kit-discuss` rejected | Discussed — user chose the browser-incognito metaphor ("fresh session, ignores stored state"; "no memory" doubles as skip `docs/memory`) | S:95 R:85 A:80 D:95 |
| 3 | Certain | Semantics are read-blind, not trace-free: invocation logged via `fab log command "fab-incognito"`; later skills may write artifacts; the skill text says so plainly | Discussed — user drew the distinction explicitly (browser incognito = don't record history; this = don't read history) | S:90 R:90 A:90 D:90 |
| 4 | Certain | Standing Session Rule: after loading, never open `docs/memory/*` or `docs/specs/*` (incl. both index files) or follow Memory File Lookup for the rest of the discussion, unless the user names a specific file — stated as an output/behavior obligation in its own section | Discussed — user identified this rule as "the real lever" since fab-discuss already loads only the two indexes | S:95 R:90 A:90 D:90 |
| 5 | Certain | Loads `fab/project/config.yaml` + `constitution.md` (required) and `context.md`, `code-quality.md`, `code-review.md` (optional); skips both doc indexes; declares the override in its own `## Context Loading` per `_preamble.md` §1 "the skill file wins" | Discussed — constitution MUST rules are the guardrails wanted during redesign; the preamble's derived-exception mechanism is the documented hook | S:95 R:90 A:95 D:95 |
| 6 | Certain | The six process helpers `_preamble`, `_pipeline`, `_intake`, `_srad`, `_generation`, `_review` are loaded in full from `.agents/skills/<name>/SKILL.md`; measured 120,077 bytes together | Discussed (~120 KB claim) — verified by `wc -c` on the deployed copies | S:95 R:85 A:90 D:90 |
| 7 | Certain | The five loadable helpers are declared in frontmatter `helpers: [_pipeline, _intake, _srad, _generation, _review]` (`_preamble` implicit) rather than as in-body reads | All five are allowed `helpers:` values; unconditional pre-body loads are exactly what frontmatter declares per `_preamble.md` § Skill Helper Declaration; in-body reads are for stage-conditional loads | S:70 R:95 A:85 D:75 |
| 8 | Certain | Skill bodies and the CLI partials `_cli-fab`, `_cli-fab-pane`, `_cli-fab-operator`, `_cli-agents`, `_cli-external` (274,220 bytes together) are lazy-loaded on demand, never up front; the skill states the size rationale (~834 KB all deployed vs ~36 KB always-load) | Discussed — "clean context is a misnomer; different anchor, not less context"; sizes verified by `wc -c` | S:95 R:90 A:90 D:90 |
| 9 | Confident | Skill catalog = frontmatter `name` + `description` of every deployed `.agents/skills/*/SKILL.md`, including `_*` helper partials and `internal-*` skills, with `user-invocable: false` entries marked as helpers; a shell frontmatter scan is the suggested mechanism, missing `.agents/skills/` ⇒ STOP "run fab sync" | Discussed ("every deployed skill, not bodies"); inclusion of partials/internal skills and the missing-dir error are the agent's fill — partials are the process knowledge the catalog exists to surface | S:70 R:90 A:80 D:70 |
| 10 | Certain | Kit version read via `cat "$(fab kit-path)/VERSION"` (fallback `unknown`), printed as `Kit: {version}` | User asked to verify during apply; verified at intake — the file exists (prints `2.26.3` vs binary `2.28.0`), and it is the same file `fab fab-help`'s `readKitVersion` reads | S:85 R:95 A:90 D:80 |
| 11 | Certain | Active change via `fab resolve --folder --or-none`; `(none)` ⇒ "No active change" (not an error; non-zero exit is); stage derived from `.status.yaml` `progress`; no artifacts loaded; no preflight | Discussed — mirrors fab-discuss verbatim; verified `(none)` exits 0 in this worktree | S:95 R:90 A:95 D:95 |
| 12 | Certain | Orientation Summary shape: project line, `Kit:`, helpers list, `Skill catalog: {N} skills`, project files loaded/not found, verbatim `Memory and specs: NOT loaded (incognito — will not be opened unless you name a file)`, active-change line, ready signal `Ready to discuss the system, incognito. What would you like to rethink?`; no `Next:` line | Discussed — user specified every line; the Next-line opt-out is the documented "skill file wins" path | S:90 R:95 A:90 D:85 |
| 13 | Certain | Key Properties: Arguments none; requires active change no; preflight no; read-only yes; idempotent yes; advances stage no; `Next:` no; plus an explicit "Loads docs/memory / docs/specs? No" row | Discussed — user enumerated the table; the extra row is the skill's defining property | S:95 R:95 A:95 D:95 |
| 14 | Confident | Frontmatter `description` is an imperative summary in fab-discuss's style followed by a "Use when rethinking how the fab process should work…" trigger clause | User asked for a "Use when …" trigger "like sibling skills"; survey shows fab-discuss/fab-help use imperative summaries and only fab-operator opens with "Use when" — blending both honors the trigger request without breaking the sibling pattern | S:60 R:95 A:75 D:65 |
| 15 | Certain | `fab-help.md` needs no edit (no markdown command list); the Go skill-list constant `skillToGroupMap` in `src/go/fab/cmd/fab/fab_help.go` gains `"fab-incognito": "Start & Navigate"` and `TestFabHelp_GroupMapping`'s `expectedMapped` gains the name; no `_cli-fab.md` update (not a command-signature change) | Verified — `fab fab-help` scans frontmatter and renders unmapped skills under `Other`; the description's Go carve-out ("unless a skill-list constant exists") applies; same group as fab-discuss | S:85 R:95 A:90 D:85 |
| 16 | Certain | Sibling sweep class (grep `fab-discuss`): `docs/specs/skills.md`, `glossary.md`, `overview.md`, `user-flow.md`, `README.md`, `_preamble.md` (Always Load examples + Next-line opt-out list), `fab-discuss.md` cross-ref; `code-dedupe.md`/`code-reorg.md` analogies, findings, logs, `config.md`, `planning-skills.md` DD deliberately untouched | Enumerated by repo-wide grep at intake per `code-quality.md` § Sibling Sweeps; apply re-greps before finishing | S:80 R:90 A:85 D:80 |
| 17 | Confident | README stage-coverage mermaid matrix and `docs/img/stage-coverage.svg` do not get a new column; a one-line legend note mentions `/fab-incognito` as fab-discuss's sibling covering the same "context" cell | Agent judgment — the matrix encodes stage coverage, which is identical to fab-discuss's; an SVG column is high-effort, low-signal and reversible later | S:50 R:85 A:65 D:55 |
| 18 | Confident | Memory hydrate targets `_shared/context-loading` (always-load exception set; reword the "fab-discuss is not an exception" special case) and `pipeline/preflight` (no-preflight skill list); no new memory file | Derived from grep of the domain indexes and existing `fab-discuss` mentions — no dedicated fab-discuss memory file exists, so the sibling is documented where the exception set already lives; hydrate may still elect a new file | S:55 R:90 A:60 D:55 |
| 19 | Confident | The Standing Session Rule binds the discussion only; a later user-invoked skill follows its own Context Loading section ("the skill file wins") and is not constrained | User scoped the rule to "the remainder of the discussion"; other skills' loading is owned by their own files — stated explicitly to prevent the rule bleeding into `/fab-new`'s Memory File Lookup | S:65 R:90 A:80 D:70 |
| 20 | Certain | Constitution V portability: the skill cites `docs/memory/index.md` / `docs/specs/index.md` (allow-listed host convention paths) and `.md`-less globs `docs/memory/*` / `docs/specs/*` (unmatched by the guard regex); never fab-kit's own docs or `src/go/*`; canonical source `src/kit/`, deployed copies untouched | Verified against `kit_portability_test.go` (allow-list lines 16–19, regex line 23 requires `.md`) | S:90 R:95 A:95 D:90 |

20 assumptions (15 certain, 5 confident, 0 tentative, 0 unresolved).
