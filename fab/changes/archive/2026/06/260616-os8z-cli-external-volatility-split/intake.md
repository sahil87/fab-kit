# Intake: _cli-external.md Volatility Split + hop

**Change**: 260616-os8z-cli-external-volatility-split
**Created**: 2026-06-16

## Origin

Discussion session (`/fab-discuss`) exploring how to keep `src/kit/skills/_cli-external.md` — the external-CLI reference helper loaded only by operator skills — from going stale. The conversation established the problem, narrowed it, and designed the fix collaboratively.

> User raw input to `/fab-new`: "Restructure src/kit/skills/_cli-external.md around a volatility split: a hand-authored 'gist' tier (what each tool is, operator-critical commands/flags, integration semantics) plus a runtime-help-dump delegation for the exhaustive command/flag surface. Add a Reference-model preamble naming the hidden but stable `help-dump` JSON contract (tool/version/commands tree, schema_version:1) and instructing agents to run `<tool> help-dump` at use-time as authoritative for anything not in the gist. Trim the wt/idea/rk tables to the operator-used gist (keeping integration-critical flags like wt --non-interactive/--worktree-name/--reuse/--base), delegating the rest. Add a new `hop` section (multi-repo navigator) focused on repo/worktree discovery — `hop ls` and `hop ls --trees` (fans out to `wt list --json`) — framed as the discovery front-end to the same repo/worktree space wt operates on. Update the file's frontmatter description to mention hop. This is a docs/skill-content restructure of one helper file; no Go code changes."

**Interaction mode**: conversational. The design was reached over a multi-turn discussion, not a one-shot prompt. Key path:

1. Identified two drift classes — **fab-caller drift** (fab skills change how they call the tools) vs. **external-tool drift** (the tools evolve in their own release cycles). User chose **external-tool drift** as the priority.
2. Empirically confirmed the drift is **live, not hypothetical**: the doc is missing real commands across every tool (`wt` missing `init`/`open`; `idea` missing `fmt`/`prune`; `hop` entirely undocumented) — measured against the installed binaries.
3. Established that the user **owns all four tools** (`rk`, `wt`, `idea`, `hop`) and that **all expose a hidden but stable `help-dump`** subcommand emitting uniform JSON.
4. Considered three mechanism options — a written conformance obligation, a runnable `fab doctor --external` conformance check, and single-sourcing the volatile tables out of the doc. **User chose the third** (volatility split + runtime `help-dump` delegation) and **explicitly rejected the conformance check** as belt-and-suspenders that would re-introduce the maintenance the design eliminates.
5. Chose the **committed** trim target (trim tables to the operator-used gist) over the conservative one (keep tables, just add a distrust note).

## Why

**Problem.** `_cli-external.md` inlines exhaustive command/flag tables for four tools fab-kit does not version-lock against. The tools (`wt`, `idea`, `rk`, `hop`) ship on their own release cadences, so every inlined table is a standing drift liability with **no signal** until something breaks at runtime. The drift is already real and measured: the doc omits commands that exist in the installed binaries today, and `hop` — a relevant multi-repo navigator — isn't documented at all.

**Consequence if not fixed.** The doc keeps decaying silently. An operator agent trusts a stale table, guesses a flag that no longer exists, or never learns about a tool (`hop`) that would let it discover sibling repos. Each tool release widens the gap. A conformance check could *detect* the drift, but it leaves the stale tables in place and adds machinery to police content that doesn't need to exist.

**Why this approach over alternatives.** The tools already carry a machine-readable source of truth: a hidden, stable, version-stamped `help-dump` JSON (uniform `tool`/`version`/`captured_at`/`schema_version:1`/recursive `root.commands` tree across all five binaries — fab included). Rather than *mirror* that surface (and police the mirror), the doc should **delegate** it: keep only the parts that don't drift (what each tool *is*, the few operator-critical commands/flags, and the integration semantics), and instruct the agent to run `<tool> help-dump` at use-time for anything exhaustive. This *deletes the drift class* instead of detecting it — there is nothing left to go stale. It is the same delegation move already proven in this file for `rk`'s Visual Display Recipe ("read `rk context` at use-time"), generalized to the whole file, and it fits the **Pure Prompt Play** principle (the agent reads + runs a tool; no generated artifact to maintain). No Go code, no CI check, no cross-repo discipline.

## What Changes

A single-file restructure of `src/kit/skills/_cli-external.md`. No Go code, no other skills, no SPEC mirror (`_cli-external.md` is excluded from SPEC mirrors by the `uliv` exclusion policy — pure-reference partials carry no SPEC).

### 1. Frontmatter `description:` update

Extend the `description:` to mention `hop` and the reference model. Current value:

```
description: "External CLI tool reference — wt (worktree manager), idea (backlog manager), tmux, rk (run-kit: context/iframe/proxy/visual-display + notify), and /loop. Loaded by operator skills only."
```

New value (add `hop` to the tool list and note the gist+`help-dump`-delegation model), e.g.:

```
description: "External CLI tool reference — wt (worktree manager), idea (backlog manager), hop (multi-repo navigator), tmux, rk (run-kit: context/iframe/proxy/visual-display + notify), and /loop. Hand-authored gist (operator-critical commands/flags + integration semantics) per tool; the exhaustive command/flag surface is delegated to each tool's `help-dump` at use-time. Loaded by operator skills only."
```

### 2. New "Reference model" preamble section (top of file, after the existing intro blockquote)

A short, load-bearing section establishing the volatility split and naming the `help-dump` contract. It MUST:

- State the **two-tier model**: the file documents a hand-authored **gist** per tool (what it is, the operator-critical commands and integration-critical flags, and the integration semantics); the **exhaustive** command/flag surface is NOT inlined.
- **Name the hidden `help-dump` contract explicitly** — this is the discoverability fix. Each tool (`wt`, `idea`, `rk`, `hop`, and `fab` itself) exposes a hidden `help-dump` subcommand emitting stable JSON: `tool`, `version`, `captured_at`, `schema_version` (currently `1`), and a recursive `root` → `commands[]` tree (each node: `name`, `path`, `short`, `usage`, `text`). It is hidden (not in `--help`) but stable.
- **Instruct the agent**: for any flag or subcommand not covered in the gist below, run `<tool> help-dump` (or `<tool> <cmd> --help`) at use-time and treat *that*, not this file, as authoritative for the exhaustive surface. The inlined gist tables are deliberately a curated subset.
- **Carry the detection gate inseparably from the instruction** (see § "Absent-binary discipline" below): a naked `<tool> help-dump` on a missing binary fails *loudly* (`command not found`, exit 127) — verified empirically. The delegation instruction MUST therefore be stated as gated: run `command -v <tool> >/dev/null 2>&1 && <tool> help-dump`, skipping silently when absent. The gate is not optional polish — it is what makes the delegation safe.

### Absent-binary discipline (two install classes)

The four owned binaries fall into **two classes** by install guarantee, and the fail-silent rule applies asymmetrically:

- **Assumed-present** — `wt`, `idea`. These are Homebrew `depends_on` of `fab-kit` (they land together via `brew install fab-kit`; the four-binaries-on-PATH guarantee is documented in `distribution/distribution.md`). The doc may continue to treat them as present and is NOT required to `command -v`-gate every `wt`/`idea` invocation. (This matches today's behavior — both sections already say "Installed system-wide via `brew install fab-kit`".)
- **Genuinely-optional** — `rk`, `hop`. `rk` is a separate run-kit project (already governed by the `_preamble.md` § Run-Kit fail-silent rule). `hop` is a separate sibling formula the user owns but `fab-kit` does NOT pull as a dependency — so, like `rk`, it can legitimately be absent. **Every `rk` and `hop` invocation (including `help-dump`) MUST be `command -v`-gated and fail silently** — never surface `command not found` or any error/warning when the tool is absent. This generalizes the existing `rk`-only discipline to `hop` (the new tool), but deliberately does NOT extend it to `wt`/`idea`.

The §2 `help-dump` delegation instruction MUST encode this split: for `rk`/`hop` the gated form is mandatory; for `wt`/`idea` the bare form is acceptable (they're guaranteed present). The new `## hop` section (§6) opens with the same fail-silent note the `rk` section carries, pointing at the `_preamble.md` discipline.

Example shape (illustrative, author may refine wording):

```markdown
## Reference Model

This file documents a hand-authored **gist** per tool — what it is, the commands and
flags the operator's correctness depends on, and the integration semantics. It is
deliberately **not** an exhaustive command reference.

Every tool below (and `fab` itself) exposes a **hidden but stable** `help-dump`
subcommand that emits the full command tree as JSON:

​```json
{ "tool": "wt", "version": "v0.0.16", "captured_at": "...", "schema_version": 1,
  "root": { "name": "wt", "commands": [ { "name": "create", "usage": "...", "text": "..." }, ... ] } }
​```

For any flag or subcommand not in the gist, run `<tool> help-dump` (or
`<tool> <cmd> --help`) at use-time and treat that as authoritative — not this file.

`wt`/`idea` are guaranteed present (brew `depends_on` of fab-kit). `rk`/`hop`
are optional — gate them and fail silently (never surface `command not found`):

​```sh
command -v hop >/dev/null 2>&1 && hop help-dump   # rk/hop: gated
wt help-dump                                       # wt/idea: assumed present
​```
```

### 3. Trim `wt` tables to the operator-used gist

Keep the operator-critical surface; delegate the rest.

- **Keep** the `wt create` flags that are **integration-critical** (operator correctness depends on them): `--non-interactive`, `--worktree-name`, `--reuse`, `--base`, and the positional `[branch]`.
- **Keep** `list`, `list --path`, `create`, `delete` in the command gist (operator uses them) and the two worked examples (known-change / autopilot-respawn).
- **Drop** from the inlined tables the commands the operator does not drive (`init`, `open`, `shell-init`, `update`) — these are reachable via `wt help-dump`.
- **Preserve verbatim** the integration prose: the **Repo-targeted spawning** note (`wt create` runs in the target repo's directory; `fab spawn-command --repo <target-repo>`) and the entire **Operator Spawning Rules** subsection (known-change vs. new-change strategy). These are hand-owned integration semantics, not reference, and `runtime/operator.md` points here for them.

### 4. Trim `idea` tables to the operator-used gist

- **Keep** the verbs the operator/agents actually use: `add` (+ bare shorthand), `list`, `show`, `done`, `reopen`, `edit`, `rm`. **Drop** `fmt`, `prune`, `shell-init`, `update` from the inlined table (reachable via `idea help-dump`).
- **Preserve** the integration semantics: the **`--main` worktree resolution** behavior, the `--file`/`IDEAS_FILE` priority, the query-matching rule, and the backlog/output format blocks. These are hand-owned.

### 5. Trim `rk` to the operator-used gist

`rk` is already largely delegation-shaped (the Visual Display Recipe defers to `rk context`). Tighten the framing:

- **Keep** `rk notify` (operator default notification channel — full body, since `_preamble.md` points here for it), `rk context` (server-URL discovery), iframe windows, the proxy pattern, and the Visual Display Recipe delegation.
- Add a one-line pointer that the full `rk` command surface (`daemon`, `doctor`, `serve`, `reaper`, `riff`, `init-conf`, `status`, `update`, …) is available via `rk help-dump` — the operator only uses the subset above. This `rk help-dump` pointer is itself subject to the `command -v rk` gate (the existing `rk` fail-silent rule already covers it).
- No change to the fail-silent contract or the `_preamble.md` cross-reference.

### 6. New `hop` section (multi-repo navigator)

Add a `## hop (Multi-Repo Navigator)` section. Framing: **`hop` is the discovery front-end to the same repo/worktree space `wt` operates on** — it locates repos (and their worktrees) from a `hop.yaml` registry.

The section MUST open with the **fail-silent note** (mirroring the `rk` section): `hop` is a genuinely-optional binary (separate sibling formula, not a `fab-kit` dependency), so every `hop` invocation — including `hop help-dump` — is `command -v hop`-gated and skips silently when absent, per the absent-binary discipline above. Never surface `command not found`.

Gist content (operator-relevant subset):

- **What it is**: `hop <selection> <action>` — locate, open, and operate on repos registered in `hop.yaml`. Selection = a repo name (substring → fzf on ambiguity), `repo/worktree`, a group, or `--all`. Action = a builtin verb (`cd`/`open`/`where`), a batch verb (`pull`/`push`/`sync`), or any PATH binary.
- **Primary operator capability — repo/worktree discovery**:
  - `hop ls` — list all registered repos as aligned name/path columns (the most useful command for an agent discovering where sibling repos live on disk).
  - `hop ls --trees` — list repos **with worktree summaries**, fanning out to `wt list --json` per repo. This is the explicit `hop`↔`wt` integration seam: `hop` enumerates repos, `wt` enumerates each repo's worktrees.
  - `hop <name> where` — echo the absolute path of a matching repo (or `hop <name>/<wt> where` for a specific worktree, resolved via `wt list --json`).
- **Why it matters to the operator**: multi-repo coordination needs to resolve the absolute main-worktree root of a *sibling* repo (e.g., to spawn an agent into it — see the Repo-targeted spawning note in the `wt` section). `hop` is how an agent discovers those locations rather than hardcoding paths.
- **Delegation pointer**: the full `hop` surface (`add`, `clone`, `rm`, `config`, `shell-init`, `update`, batch verbs, `--all` fan-out) is available via `hop help-dump`; the gist covers only discovery.

### 7. tmux and /loop sections

No substantive change — tmux is third-party/frozen and `/loop` is already gist-shaped. Optionally add the one-line `help-dump` framing note only where it reduces ambiguity (tmux has no `help-dump`; `/loop` is a skill, not a binary — so the `help-dump` instruction in §2 scopes to the four owned binaries, not these two). The §2 preamble wording MUST make clear `help-dump` applies to the four owned binaries (`wt`/`idea`/`rk`/`hop`), not to tmux or `/loop`.

## Affected Memory

- `runtime/operator.md`: (modify) The operator's external-tool helper is `_cli-external.md`; operator.md enumerates the tool set (`wt`, `idea`, `tmux`, `/loop`) in two places (the §2 Context-Loading paragraph naming the `_cli-external` helper, and the `description:` frontmatter), and homes the spawning rules there. Adding `hop` and the reference-model preamble changes what that helper documents — update operator.md's tool-set enumerations to include `hop`, and note (if warranted) that the gist+`help-dump`-delegation model now governs `_cli-external.md`'s reference content. The normative spawn procedure (operator §6) and the spawning-rules home (§ wt) are unchanged in substance — only the enumeration of which tools `_cli-external` covers changes.

Not affected:
- `distribution/distribution.md` — `hop` is NOT a Homebrew `depends_on` of `fab-kit` (only `wt`/`idea` are pulled transitively). The distribution model ("four binaries on PATH") stays accurate. A doc restructure does not change distribution. Deliberately out of scope to avoid over-scoping.
- No SPEC mirror — `_cli-external.md` is excluded from SPEC mirrors by the `uliv` exclusion policy.

## Impact

- **Files touched**: `src/kit/skills/_cli-external.md` (the restructure), `docs/memory/runtime/operator.md` (enumeration update at hydrate). The canonical source is `src/kit/skills/` — never edit the deployed `.claude/skills/` copy (it is regenerated by `fab sync`).
- **No Go code**: no `fab` CLI changes, no tests. (`help-dump` already exists on all five binaries; this change consumes it from prose, it does not add it.)
- **Constitution constraint check**: the rule "Changes to skill files MUST update the corresponding SPEC-*.md" does not bind here — `_cli-external.md` is explicitly exempt (uliv). The "Changes to the fab CLI MUST update `_cli-fab.md`" rule does not apply (no CLI change).
- **Consumers**: only `/fab-operator` loads `_cli-external.md` (operator-only helper). Blast radius is one skill's reference helper plus one memory enumeration.
- **Risk — over-trimming**: the gist/exhaustive line is "does the operator's correctness depend on this command/flag?" not "is it commonly used." `wt --base` looks minor but is load-bearing for sequenced autopilot — it stays. The author MUST apply the integration-critical test per flag, not a frequency heuristic.

## Open Questions

- Should `hop` be wired into the operator's actual spawn/discovery workflow (e.g., operator §6 uses `hop where`/`hop ls` to resolve sibling-repo roots), or only documented in `_cli-external.md` for completeness this round? (Leaning: document now, defer any operator §6 workflow change to a separate change — keeps this restructure to the helper file + the enumeration update.)
- Exact wording of the `## Reference Model` preamble — illustrative shape given; author refines at apply.
- Where should the generalized fail-silent rule for `hop` physically live — extend `_preamble.md` § Run-Kit (rk) Reference to name `hop` too (making it a § "External-tool detection" rule covering both optional binaries), or state it self-contained in `_cli-external.md`'s § Absent-binary discipline? (Leaning: state it in `_cli-external.md` — `hop` is operator-only like the rest of that helper, and `_preamble.md`'s rk rule exists there only because the rk gate is carried inline by *every* skill; `hop` has no such universal-carry need. Keep `_preamble.md` rk-scoped; document `hop`'s gate in the operator-only helper.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Restructure only `src/kit/skills/_cli-external.md` (the canonical source), not the deployed `.claude/skills/` copy | Constitution V / context.md: `src/kit/` is canonical; `.claude/skills/` is regenerated by `fab sync` | S:95 R:90 A:100 D:95 |
| 2 | Certain | No SPEC mirror update required | `uliv` exclusion policy: `_cli-fab.md`/`_cli-external.md` carry no SPEC (pure-reference partials) — confirmed in specs-index.md | S:90 R:85 A:100 D:95 |
| 3 | Certain | No Go code / no CLI change / no tests | Scope is a prose restructure; `help-dump` already ships on all binaries — confirmed empirically this session | S:95 R:90 A:100 D:95 |
| 4 | Confident | Volatility split (gist + runtime `help-dump` delegation), conformance check explicitly rejected | User chose this option in discussion and explicitly rejected belt-and-suspenders; mirrors the existing `rk context` delegation pattern | S:90 R:75 A:85 D:90 |
| 5 | Confident | `runtime/operator.md` is the affected memory; `distribution/distribution.md` is not | operator.md documents `_cli-external` as its helper and enumerates the tool set; `hop` is not a fab-kit Homebrew dep so distribution is unchanged — verified by grep this session | S:85 R:70 A:90 D:85 |
| 6 | Confident | Committed trim target (trim tables to operator-used gist), not conservative (keep tables + distrust note) | User explicitly chose "committed" — stale tables next to a distrust note are worse than either extreme | S:90 R:65 A:80 D:85 |
| 7 | Confident | Keep `wt --non-interactive/--worktree-name/--reuse/--base` and the spawning-rules prose; drop `init`/`open`/`shell-init`/`update` from inlined tables | User named these flags as integration-critical; the gist/exhaustive line is "operator correctness depends on it," and operator.md points here for the spawning rules | S:85 R:70 A:85 D:80 |
| 8 | Confident | `hop` gist focuses on discovery (`hop ls`, `hop ls --trees`, `hop where`); rest delegated to `hop help-dump` | User: "hop allows the agent to discover locations of other repos — probably `hop ls`"; `hop ls --trees` is the explicit `hop`↔`wt` seam | S:85 R:75 A:80 D:80 |
| 9 | Tentative | Document `hop` in `_cli-external.md` and update operator.md's enumeration, but do NOT rewrite the operator §6 spawn procedure to mandate `hop` this round | User specified discovery documentation, not a workflow rewrite; wiring `hop` into operator §6 is a separable, reversible follow-up — flagged in Open Questions | S:70 R:60 A:65 D:65 |
| 10 | Tentative | `help-dump` delegation instruction scopes to the four owned binaries (`wt`/`idea`/`rk`/`hop`); tmux (no `help-dump`) and `/loop` (a skill) are excluded from that instruction | tmux is third-party with no `help-dump`; `/loop` is a Claude Code skill not a binary — both fall outside the contract, so the preamble must scope explicitly | S:75 R:70 A:75 D:70 |
| 11 | Certain | A naked `<tool> help-dump` on a missing binary fails loudly (exit 127, `command not found`), so the delegation instruction MUST be `command -v`-gated to be safe | Verified empirically this session: `nonexistent-tool help-dump` → exit 127; `command -v` gate returns cleanly/silently when absent | S:95 R:90 A:100 D:95 |
| 12 | Certain | Two install classes: `wt`/`idea` assumed-present (brew `depends_on` of fab-kit), `rk`/`hop` genuinely-optional (separate formulas) | Verified: distribution.md documents the four-binary `depends_on` guarantee; `hop` is not a fab-kit dep; `rk` already has its own fail-silent rule | S:95 R:85 A:95 D:90 |
| 13 | Confident | Gate + fail-silent only `rk`/`hop`; leave `wt`/`idea` un-gated (assumed present) — do NOT generalize the gate to all four | User chose "gate only the genuinely-optional tools (rk + hop)"; matches the actual install guarantees rather than over-cautious uniformity | S:90 R:75 A:90 D:90 |
| 14 | Tentative | The generalized fail-silent rule for `hop` lives self-contained in `_cli-external.md` (§ Absent-binary discipline + `hop` section opener), NOT by extending `_preamble.md`'s rk rule | `hop` is operator-only like the rest of the helper; `_preamble.md`'s rk gate is inline-carried by every skill for a reason `hop` lacks — keep preamble rk-scoped. Reversible; flagged in Open Questions | S:70 R:65 A:70 D:65 |

14 assumptions (5 certain, 6 confident, 3 tentative, 0 unresolved). Run /fab-clarify to review.
