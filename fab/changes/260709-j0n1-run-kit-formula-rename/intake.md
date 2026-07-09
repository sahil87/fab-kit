# Intake: Reflect the run-kit rk→run-kit Formula Rename

**Change**: 260709-j0n1-run-kit-formula-rename
**Created**: 2026-07-09

## Origin

One-shot `/fab-new` invocation:

> the run-kit formula/binary was renamed from rk to run-kit (homebrew-tap: formula_renames.json maps rk->run-kit, Formula/rk.rb deleted, Formula/run-kit.rb 3.0.0 with rk kept as a symlink alias; run-kit repo v3.0.0 already made this switch, e.g. its own updater now targets sahil87/tap/run-kit). Find and update any references to the old rk formula/binary name, docs, config, or fixtures in this repo (fab-kit) that need to reflect the rename, then run through the full fab pipeline including opening a PR.

Key facts stated by the user (treated as ground truth for this change):

- The Homebrew formula was renamed `rk` → `run-kit` at run-kit v3.0.0 (`Formula/rk.rb` deleted, `Formula/run-kit.rb` created at 3.0.0, `formula_renames.json` maps `rk` → `run-kit` so the old name still resolves in brew).
- The `rk` **binary name is kept as a symlink alias** — every `rk …` shell invocation continues to work.
- The run-kit repo itself already switched (its updater targets `sahil87/tap/run-kit`).

An intake-time repo-wide sweep (grep for `tap/rk`, `Formula/rk`, `rk.rb`, `brew install rk`, `rk formula`, plus a broad `\brk\b` file inventory across `src/`, `docs/`, `scripts/`, `.github/`, `README.md`, Go sources, and testdata) established the actual blast radius — see What Changes.

## Why

1. **The pain point**: fab-kit's operator-facing external-CLI reference (`src/kit/skills/_cli-external.md`) documents rk's install identity — "`rk` … a separate sibling formula" — which is now imprecise: the formula is named `run-kit` (`sahil87/tap/run-kit`), and `rk` survives only as a binary alias. Under Constitution II (Docs Are Source of Truth) and the docs-accuracy checklist category, agents reading this reference could give a user stale install guidance (e.g., naming the formula `rk`).
2. **If we don't fix it**: the drift is silent — `brew` still resolves the old name via `formula_renames.json` and the `rk` alias keeps every invocation working — so nothing breaks loudly; the docs just become progressively wrong about what run-kit *is*, and the next agent that documents or scripts against the formula name propagates the stale identity.
3. **Why this approach**: a minimal docs-accuracy edit at the one live claim site (plus its constitution-mandated SPEC mirror), with an explicit *verified-unchanged* inventory for everything else. The sweep proved fab-kit never targets the formula by name (no `sahil87/tap/rk`, no `brew install rk`, no fixture/config references — zero hits), so a rename-style mass edit would be wrong: the correct change is small and surgical.

## What Changes

### 1. `src/kit/skills/_cli-external.md` — the only live install-identity claim (2 edit sites)

**Site A — § Reference Model** (currently lines 81–85):

Current text:

```markdown
- **Genuinely-optional — `rk`, `hop`.** Each is a separate sibling formula the user
  may or may not have installed (`rk` is run-kit; `hop` is the multi-repo
  navigator). **Every `rk`/`hop` invocation — including `help-dump` — MUST be
  `command -v`-gated and fail silently** (never surface `command not found` or any
  error/warning when the tool is absent). Do NOT generalize this gate to `wt`/`idea`.
```

New text (only the parenthetical changes — the gate rule is untouched):

```markdown
- **Genuinely-optional — `rk`, `hop`.** Each is a separate sibling formula the user
  may or may not have installed (`rk` is run-kit — formula `sahil87/tap/run-kit`
  since run-kit v3.0.0, with `rk` kept as a binary alias; `hop` is the multi-repo
  navigator). **Every `rk`/`hop` invocation — including `help-dump` — MUST be
  `command -v`-gated and fail silently** (never surface `command not found` or any
  error/warning when the tool is absent). Do NOT generalize this gate to `wt`/`idea`.
```

**Site B — § rk (run-kit) intro** (first paragraph of the section):

Add one sentence after the existing opening sentence ("run-kit is the tmux session manager with a web UI that hosts the operator's session."):

```markdown
Since run-kit v3.0.0 the Homebrew formula and primary binary are named `run-kit`
(`sahil87/tap/run-kit`); `rk` is kept as a symlink alias and remains the invocation
form used throughout fab skills.
```

This makes the alias status explicit exactly where the command reference lives, and pins *why* fab skills keep writing `rk` (deliberate, not stale).

### 2. `docs/specs/skills/SPEC-_cli-external.md` — constitution-mandated mirror

Constitution ("Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file"): reflect the edit in the SPEC's rk row/prose — the § Inventory row `rk (run-kit)` (and/or the overview prose) gains the same one-line rename fact (formula `sahil87/tap/run-kit` since v3.0.0, `rk` kept as alias/invocation form). Keep it to a minimal factual addition; the SPEC's mirror rule note (a change to either side must keep the pointer accurate) is the trigger.

### 3. Verified-unchanged inventory (explicit non-goals — the sweep found these need NO edit)

- **`rk` command invocations everywhere** (`command -v rk`, `rk notify`, `rk context`, `rk agent-setup`, `rk help-dump` in `_preamble.md`, `_cli-external.md`, `fab-operator.md`, SPECs, memory): the `rk` alias is kept deliberately — invocations stay `rk`. Do NOT rewrite to `run-kit`.
- **`src/kit/skills/_preamble.md` § Run-Kit (rk) Reference**: documents only the detection/fail-silent rule (`command -v rk`) and a pointer to `_cli-external.md`; makes no formula-identity claim. No edit (and therefore no SPEC-_preamble.md edit).
- **`@rk_agent_state` tmux pane option** (Go sources `src/go/fab/internal/pane/`, pane tests, `_cli-fab.md`, memory `runtime/runtime-agents.md`, migration `2.13.6-to-2.14.0.md`): a run-kit-owned data convention; the rename covers formula/binary only. Unchanged.
- **Historical references**: `rk v2.3.2` release attributions (SPEC-fab-operator.md, `docs/memory/runtime/operator.md`), archived change artifacts (`fab/changes/archive/**`), memory `log.md`/`log.seed.md` entries. Historical facts are not rewritten.
- **Formula-path references**: zero hits for `sahil87/tap/rk`, `Formula/rk.rb`, `brew install rk` anywhere in fab-kit (README, docs/site, `.github/formula-template.rb`, `src/go/fab-kit/internal/update.go` — fab-kit's updater targets only the `fab-kit` formula). Nothing to update.
- **Config & fixtures**: `fab/project/config.yaml` and Go testdata contain no rk formula references. Nothing to update.

## Affected Memory

- None — no spec-level fab-kit behavior changes. The rk install-identity gist lives in the skill reference (`_cli-external.md`), not in `docs/memory/`; memory references to rk are invocation-level (`rk notify` contract, `@rk_agent_state` convention) or historical, all verified unchanged by the rename. Hydrate should verify this holds against the final diff and regenerate indexes only if anything moved.

## Impact

- **Files edited**: `src/kit/skills/_cli-external.md` (2 small prose edits), `docs/specs/skills/SPEC-_cli-external.md` (1 mirror addition). Docs/skill-content only — no Go code, no config, no fixtures, no templates, no migrations.
- **Behavior**: zero runtime behavior change; no command, flag, or contract changes. Deployed skill copies in `.claude/skills/` refresh via `fab sync` on release (gitignored, never edited directly).
- **Change type**: docs.

## Open Questions

- None — the input pinned the ground truth (rename facts, alias kept) and the sweep bounded the blast radius.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Blast radius is exactly `_cli-external.md` + its SPEC mirror — no formula-path references exist anywhere in fab-kit | Repo-wide greps for `tap/rk`, `Formula/rk`, `brew install rk`, and a `\brk\b` file inventory across src/docs/scripts/.github/Go/testdata returned zero formula-path hits; the § Reference Model parenthetical is the only live install-identity claim | S:90 R:90 A:95 D:90 |
| 2 | Confident | Keep `rk` as the invocation form in all skills/docs — do not rewrite invocations to `run-kit` | User stated the alias is kept deliberately (symlink in Formula/run-kit.rb); the terse form is entrenched across skills/SPECs/memory and run-kit itself preserves it; easily revisited if run-kit ever drops the alias | S:80 R:90 A:75 D:70 |
| 3 | Certain | `@rk_agent_state` pane-option name and all `rk agent-setup` references unchanged | run-kit-owned schema; the user-described rename covers formula/binary naming only, and the alias keeps `rk agent-setup` valid | S:85 R:90 A:85 D:85 |
| 4 | Certain | Historical references stay verbatim — `rk v2.3.2` attributions, archived changes, log entries | Historical facts record what was true at ship time; rewriting them is falsification, per the repo's established swept-prose-vs-historical-rows convention (4rtx precedent) | S:80 R:95 A:90 D:85 |
| 5 | Certain | `docs/specs/skills/SPEC-_cli-external.md` must be updated alongside the skill edit | Constitution: skill-file changes MUST update the corresponding SPEC-*.md; the SPEC's own mirror rule restates it | S:95 R:90 A:100 D:95 |
| 6 | Confident | No `_preamble.md` edit (and no SPEC-_preamble.md edit) | Its § Run-Kit section carries only the detection/fail-silent rule + pointer, no formula-identity claim; adding the rename note there would duplicate `_cli-external.md`'s single-claim-site role | S:75 R:90 A:80 D:65 |
| 7 | Certain | Affected Memory: none — hydrate verifies and regenerates indexes only if needed | Behavior unchanged; memory's rk references are invocation-level or historical; install-identity of external tools is `_cli-external.md`'s domain | S:75 R:85 A:85 D:75 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
