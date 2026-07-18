# Intake: Slim _cli-external.md to Fab-Owned Content

**Change**: 260718-clix-slim-external-md
**Created**: 2026-07-18

## Origin

One-shot invocation: `/fab-new clix` (backlog ID). Raw backlog entry:

> [clix] 2026-07-18: Slim `src/kit/skills/_cli-external.md` to fab-owned content once toolkit `<tool> skill` bundles ship (contract: shll repo docs/site/standards/skill.md; per-repo adoption phased). WHAT MOVES OUT (tool-owned gists, delegated at use-time to `<tool> skill` — bare for wt/idea, command-v-gated fail-silent for rk/hop, exactly mirroring the existing help-dump delegation in § Reference Model): wt's create contract + flags/conflict table + probe-and-route recipe; idea's verb table + backlog format; hop's discovery gist; rk's notify fail-silent contract and the static context/iframe/proxy pointers. WHAT STAYS (fab-owned — no tool bundle may carry it): the absent-binary discipline (two install classes), all operator spawning choreography (run wt create in the TARGET repo's directory, `fab agent --print --repo`, tmux new-window with $SPAWN_CMD, known-change vs backlog-respawn routing incl. the /fab-new Step 11 branch semantics and the do-NOT-send-/git-branch rule), the operator's escalation usage of rk notify, the tmux section (third-party, no skill ever), and /loop (a Claude Code skill, not a binary). VERSION-SKEW FALLBACK (required): an installed tool may predate its `skill` subcommand — capability-probe `<tool> skill` and fall back to a retained gist (or a pointer to the shll.ai /<tool>/skill page) silently; operator context loading must never break on an older binary. DEPS: shll's skill standard published + at least `wt skill` shipped (operator-critical), seeded per the SEED RULE in shll's fab/backlog.md [agst] (bundles seeded from this file's tool-owned rows — those rows are the acceptance floor). Cross-repo pairing: this is the consumer-side half of shll [agst].

**Dependency verification (performed live at intake, 2026-07-18)**: the backlog gates this change on "shll's skill standard published + at least `wt skill` shipped". Both are satisfied, and exceeded:

- `shll standards` lists `skill` ("binary+repo — Agent skill bundle standard: docs/site/skill.md served by `<tool> skill`") — the standard is published.
- **All four** tools already serve their bundle: `wt skill`, `idea skill`, `rk skill`, and `hop skill` each exit 0 and print a markdown bundle. The minimum dep was `wt skill` only; the full set shipping means no per-tool deferral is needed — every tool-owned gist can delegate now.

## Why

1. **The pain point — dual maintenance of tool-owned knowledge.** `src/kit/skills/_cli-external.md` (340 lines, loaded by `/fab-operator` only) hand-authors a curated gist per external tool: wt's `create` contract + flags/conflict table + probe-and-route recipe, idea's verb table + backlog format, hop's discovery commands, rk's notify contract and context/iframe/proxy pointers. That content is *tool-owned* — it restates what each tool's own documentation says — and it drifts on every tool release. The file already acknowledges this for the *exhaustive* surface (its § Reference Model delegates full command/flag coverage to `<tool> help-dump` at use-time), but the curated gists themselves remained hand-maintained. The drift is not hypothetical: the wt 2af2 contract change (positional became new-branch-only, `--checkout` added) forced lockstep edits across `_cli-external.md`, `fab-operator.md`, and their SPEC mirrors.

2. **The unlock — the toolkit `skill` standard shipped.** Per `shll standards skill`, every toolkit CLI now exposes `<tool> skill`: a static, ≤150-line, agent-optimized usage briefing printed as raw markdown to stdout (exit 0, stderr empty), embedded in the binary and byte-identical to the tool repo's canonical `docs/site/skill.md`. It is **version-locked by construction** — the prose ships inside the same binary as the flags it describes, so it can never document a capability the installed binary lacks. The bundles were seeded from this very file's tool-owned rows (shll [agst] SEED RULE — those rows are the acceptance floor), so the tool-side content is a superset of what fab would be deleting.

3. **The consequence of not doing it**: fab-kit keeps a stale second copy of knowledge the installed binaries now serve authoritatively, recurring 2af2-style lockstep rework on every tool contract change, and the operator pays ~340 lines of context where the fab-owned core is a fraction of that.

4. **Why this approach over alternatives**: delegation-at-use-time exactly mirrors the file's own proven `help-dump` pattern (same § Reference Model, same two install classes, same fail-silent discipline) — no new convention is invented. The alternative (keep the gists, add a sync/drift-guard against tool repos) was implicitly rejected by the backlog design: it preserves the second copy and adds machinery, whereas the standard's whole point is that the binary itself is the offline, version-locked source.

## What Changes

All edits are markdown-only, in `src/kit/skills/` canonical sources (never `.claude/skills/` deployed copies) plus their SPEC mirrors. No Go code, no templates, no migrations.

### 1. `_cli-external.md` — the slim (primary edit)

**Moves out** (tool-owned; each section's gist is replaced by a use-time delegation instruction to `<tool> skill`, exactly mirroring the existing help-dump delegation wording in § Reference Model):

- **wt**: the Commands table (`list`/`list --path`/`create`/`delete`), the `wt create` Flags table (`--non-interactive`/`--worktree-name`/`--reuse`/`--base`/`--checkout`/positional + the exit-2 conflict semantics), and the generic probe-and-route recipe examples. The wt bundle documents these (verified: its Capabilities map carries `create`/`--checkout`/`--base`/`--reuse`, `list`, `delete`).
- **idea**: the verb table (bare/`add`/`list`/`show`/`done`/`reopen`/`edit`/`rm`), persistent flags (`--file`/`--main`), query matching, backlog format block, output formats.
- **hop**: the discovery gist (`ls`, `ls --trees`, `where`) and the what-it-is prose.
- **rk**: the `rk notify` contract (usage, fail-silent-by-contract bullet, delivery model) and the static pointers for `rk context` (server-URL discovery snippet, iframe windows `@rk_type`/`@rk_url` recipes, the `/proxy/{port}/` pattern, the Visual Display Recipe pointer + visual-explainer integration).

**Stays** (fab-owned — no tool bundle may carry it):

- § Reference Model itself, extended: the `help-dump` contract prose and the **absent-binary discipline** (two install classes: `wt`/`idea` assumed-present → invoke bare; `rk`/`hop` genuinely-optional → every invocation `command -v`-gated, fail silently).
- **All operator spawning choreography** in the wt section: run `wt create` in the TARGET repo's directory, `fab agent --print --repo <target-repo>` (never the operator's own config.yaml), tmux `new-window` with `$SPAWN_CMD`, and the known-change vs backlog-respawn routing **including** the `/fab-new` Step 11 branch semantics (disposable-branch rename) and the do-NOT-send-`/git-branch` rule. Note: the routing rule necessarily keeps stating *which wt form to use when* (existing branch → `--checkout <change-folder-name>`; missing → positional) because that decision is fab's; what it stops carrying is the generic wt flag/conflict reference behind it.
- The **operator's escalation usage of `rk notify`** (the gated send with the operator's message/title template — the fab-specific usage, not the tool contract).
- The **tmux section** unchanged (third-party — no skill bundle ever; `fab pane` internalization notes are fab-owned).
- The **/loop section** unchanged (a Claude Code skill, not a binary — no `skill` subcommand).

**New delegation instruction** (added to § Reference Model, sibling of the help-dump paragraph): for each owned tool's usage knowledge beyond the retained fab-owned content, run `<tool> skill` at use-time — bare for `wt`/`idea`, `command -v`-gated fail-silent for `rk`/`hop`:

```sh
wt skill                                          # wt/idea: assumed present, bare
idea skill
command -v rk  >/dev/null 2>&1 && rk skill        # rk/hop: gated, fail silently
command -v hop >/dev/null 2>&1 && hop skill
```

**Version-skew fallback (required)**: an installed tool may predate its `skill` subcommand. The delegation instruction MUST capability-probe (`<tool> skill` failing — non-zero exit or no output — is the probe) and fall back **silently** to the shll.ai bundle page pointer (`https://shll.ai/<tool>/skill`); operator context loading must never break or surface an error on an older binary. For `rk`/`hop` the probe composes with the existing `command -v` gate (absent binary → skip entirely; present-but-old binary → fallback pointer). The retained fab-owned choreography already covers the operator-critical wt semantics, so the fallback does not need to reproduce tool gists (see Assumption 4).

### 2. Cross-reference sweep (same-change, per the mirror rule)

- **`docs/specs/skills/SPEC-_cli-external.md`** — the Summary and the per-section Command Inventory table describe the gist model ("hand-authored gist per tool", the rk row's "full body the `_preamble.md` § Run-Kit pointer forwards to"); rewrite to the slimmed reality.
- **`src/kit/skills/_preamble.md` § Run-Kit (rk) Reference** — currently: "The full `rk` command reference — `rk context` (server-URL discovery, iframe windows, the proxy pattern, and the Visual Display Recipe) and `rk notify` (the operator's default notification send) — lives in `_cli-external.md` § rk (run-kit)." After the slim, the rk section carries the fab-owned escalation usage + a `rk skill` delegation, not the full command bodies — the pointer text must be updated to stay accurate (cross-reference rule). Mirror: `docs/specs/skills/SPEC-_preamble.md`.
- **`src/kit/skills/fab-operator.md`** — references `_cli-external.md § wt` for probe-and-route (lines citing the `--checkout`-vs-positional routing) and `§ rk (run-kit)` for the notify contract. The retained choreography keeps these referents alive but slimmer; each reference must be re-verified against the slimmed text and re-pointed where it cited moved-out material. Mirror: `docs/specs/skills/SPEC-fab-operator.md`.
- **`docs/specs/companions.md`** — documents the wt/idea integration; verify whether it restates tool-owned gist content and align its framing with the delegation model if so.
- Repo-wide grep for the moved-out phrases (e.g. "probe-and-route", "help-dump", "`_cli-external.md` § wt", "rk notify") to catch aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) restating per-tool facts — per code-quality.md § Sibling & Mirror Sweeps.

### 3. Non-goals

- No change to the two install classes or which tools belong to each.
- No change to `fab-operator.md`'s spawn procedure semantics (§6) — only reference accuracy.
- No removal of the `help-dump` delegation (it remains for the exhaustive command tree; `skill` covers usage knowledge — they are siblings per the standard).
- No shipping of any `skill` subcommand — that is the producer side, already done tool-side (shll [agst]).

## Affected Memory

- `runtime/operator.md`: (modify) — documents `_cli-external`'s content model ("as of 260616-os8z it documents a hand-authored per-tool **gist** … delegated to each tool's `help-dump` at use-time", line ~35) and the spawning-rules home (line ~88); both update to the skill-bundle delegation model + version-skew fallback.
- `distribution/kit-architecture.md`: (modify) — the `_cli-external.md` inventory row (line ~422) describes the file's contents; update to the slimmed fab-owned description.

## Impact

- **Files edited (apply)**: `src/kit/skills/_cli-external.md` (primary — 340 lines, expected to shrink substantially), `src/kit/skills/_preamble.md` (rk pointer text), `src/kit/skills/fab-operator.md` (reference accuracy), `docs/specs/skills/SPEC-_cli-external.md`, `docs/specs/skills/SPEC-_preamble.md`, `docs/specs/skills/SPEC-fab-operator.md`, `docs/specs/companions.md` (verify/align). Memory files at hydrate per Affected Memory.
- **No Go code, no tests**: markdown-only; nothing in `src/go/` changes. `true_impact_exclude` covers `fab/` and `docs/` but `src/kit/` edits count as source.
- **Runtime behavior change**: `/fab-operator` sessions (the sole `_cli-external` consumer) gain use-time `<tool> skill` calls where they previously read inlined gists; on old binaries the silent fallback keeps loading intact. Context cost drops for every operator session.
- **Cross-repo**: consumer-side half of shll [agst]; producer side verified shipped for all four tools. On completion, mark backlog `[clix]` done (archive-time).

## Open Questions

None — the backlog entry pre-decides scope, delegation form, retained set, and the fallback requirement; the stated dependencies were verified live at intake (all four bundles shipped, standard published).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope split — what moves out (wt create contract/flags/probe-and-route recipe; idea verbs/backlog format; hop discovery; rk notify contract + static context/iframe/proxy pointers) vs what stays (absent-binary discipline, operator spawning choreography incl. Step 11 semantics + no-`/git-branch` rule, escalation rk-notify usage, tmux, /loop) | Enumerated verbatim in backlog [clix] — no interpretation needed | S:95 R:70 A:90 D:95 |
| 2 | Certain | Delegation form: use-time `<tool> skill`, bare for wt/idea, `command -v`-gated fail-silent for rk/hop, mirroring the existing § Reference Model help-dump delegation | Specified in backlog; the mirrored pattern already exists in the file | S:95 R:80 A:95 D:95 |
| 3 | Certain | Dependencies satisfied — skill standard published, all four tools serve `skill` bundles | Verified live this session (`shll standards`, `wt/idea/rk/hop skill` all succeed); exceeds the `wt`-only minimum | S:90 R:90 A:100 D:95 |
| 4 | Confident | Version-skew fallback form = silent pointer to `https://shll.ai/<tool>/skill`, not retained duplicate gists | Backlog allows either ("a retained gist (or a pointer …)"); retaining gists would defeat the slim, and the retained fab-owned choreography already carries the operator-critical wt semantics; all four installed binaries already serve bundles, so the fallback is a degraded-mode edge | S:70 R:85 A:80 D:60 |
| 5 | Certain | Same-change sweep class: SPEC-_cli-external.md, _preamble.md § Run-Kit pointer + SPEC-_preamble.md, fab-operator.md references + SPEC-fab-operator.md, companions.md, aggregate-spec grep | Constitution mirror rule + cross-reference rule + code-quality.md § Sibling & Mirror Sweeps | S:85 R:75 A:95 D:90 |
| 6 | Confident | change_type = refactor — content restructuring/delegation with no new fab capability | Slimming + relocation of reference content; "slim" isn't a taxonomy keyword so inference needs verifying (Step 6) | S:60 R:90 A:80 D:70 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
