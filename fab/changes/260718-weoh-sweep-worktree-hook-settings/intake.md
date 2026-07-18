# Intake: Sweep Worktree Hook Settings

**Change**: 260718-weoh-sweep-worktree-hook-settings
**Created**: 2026-07-18

## Origin

> Migration sweeping every worktree's .claude/settings.local.json for removed `fab hook` entries — the 2.10.1-to-2.11.0 and 2.13.6-to-2.14.0 migrations only cleaned the checkout they ran in, while the committed fab/.fab-version gate means they never re-run in sibling checkouts; stale `fab hook *` hooks in the main checkout error on every Write/Edit in ALL worktree sessions (Claude Code resolves settings through worktrees to the main repo root)

Conversational origin (`/fab-discuss` session, 2026-07-18): the user kept seeing `PostToolUse:Write hook error — ERROR: unknown command "hook" for "fab"` in a freshly created worktree. Investigation traced the full mechanism (documented under Why) and hand-cleaned fab-kit's own repo (84 worktrees + the main checkout). The user then approved filing the kit-side fix so other fab-kit users get the same cleanup via a migration.

## Why

1. **The pain point.** The `fab hook` command family was removed outright in 2.14.0 (agent-state divestment, ioku, PR #472) with no deprecation shim. Any hook entry still registered in a `.claude/settings.local.json` now errors every time it fires: Claude Code invokes `fab hook <x>`, cobra prints `ERROR: unknown command "hook" for "fab"`, and the user sees a non-blocking `PostToolUse:{Write,Edit}` (or SessionStart/Stop/UserPromptSubmit) hook-error warning on every single file write / session event. Harmless but relentless noise.

2. **Why the existing migrations don't cover it.** Two shipped migrations remove these entries — `2.10.1-to-2.11.0.md` (the two `fab hook artifact-write` PostToolUse entries) and `2.13.6-to-2.14.0.md` §1 (the three session-scoped entries) — but both edit **only the checkout in which `/fab-setup migrations` runs**. Migration applicability is gated on `fab/.kit-migration-version`, which is a **committed, repo-wide** file: once the migration runs in one checkout and the version bump merges, no sibling checkout ever re-runs it. Meanwhile `.claude/settings.local.json` is **gitignored, per-checkout** state — pre-2.14.0, `fab sync`'s `syncHooks` step minted the 5 hook entries into every checkout it ran in (and `wt create` → `wt init` runs `fab sync` in every new worktree). Result: every worktree synced before 2.14.0 carries its own stale copy that no migration will ever touch. In fab-kit's own repo that was **84 worktrees plus the main checkout** (142 `artifact-write` actions, 84 each of `session-start`/`stop`/`user-prompt`), hand-cleaned 2026-07-18.

3. **Why the main checkout is the live poison.** Claude Code resolves project settings **through worktrees to the main repository root** (official docs, code.claude.com/docs/en/settings.md: `.claude/settings.local.json` is "read at the root of the git repository, resolved through worktrees to the main checkout, so one file covers sessions started in any subdirectory or worktree"). So a stale main checkout poisons **every** worktree session — including worktrees created *after* 2.14.0 whose own settings files are clean. (Pre-v2.1.211 Claude Code versions read the starting directory's file directly, so the per-worktree copies matter too; hooks are re-read live by a file watcher, so cleanup takes effect without a session restart.)

4. **If we don't fix it.** Every fab-kit user whose repo predates 2.14.0 and who uses worktrees keeps seeing hook errors on every Write/Edit forever — with no signal that the fix is a settings cleanup. The two shipped migrations will never reach the stale copies.

5. **Why this approach.** A new migration file is the constitutionally mandated shape: restructuring user-owned data (`.claude/settings.local.json`) MUST ship as a `src/kit/migrations/` file applied by `/fab-setup migrations`, never an ad-hoc script. The worktree-sweep pattern has direct precedent: `2.13.6-to-2.14.0.md` §2 already enumerates worktrees via `git worktree list --porcelain` to delete `.fab-runtime.yaml` everywhere — this change applies the same discipline to the settings edits its own §1 (and 2.11.0's) missed.

## What Changes

### 1. New migration `src/kit/migrations/2.15.7-to-2.15.8.md`

Standard structure (Summary / Pre-check / Changes / Verification). Behavior:

**Target set** — a hook action is stale when its `command` either:
- starts with the prefix `fab hook ` (prefix match, NOT an enumeration of the four known subcommands — the entire command family is gone, so *any* `fab hook <x>` errors), or
- matches the legacy script-shim forms carried by the prior migrations' target sets: `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-<script>.sh` / `bash fab/.kit/hooks/on-<script>.sh` (the scripts no longer exist, so these also fail if fired).

**Pre-check**:
1. If not inside a git repository: handle only the current directory (step semantics below), skip the sibling sweep — mirroring `2.13.6-to-2.14.0` §2.
2. Enumerate all worktrees via `git worktree list --porcelain` (the main checkout is the first entry, so the sweep inherently covers it).
3. Sentinel: if no `<worktree>/.claude/settings.local.json` contains a target entry, skip: `Skipped: no stale fab hook entries found in any worktree.` Re-running is a complete no-op.

**Changes** — for each worktree path whose `.claude/settings.local.json` exists and carries at least one target entry:
1. Parse the file as JSON. In every event array under `hooks` (any event — `PostToolUse`, `SessionStart`, `Stop`, `UserPromptSubmit`, etc.), drop every `{"type": "command", "command": ...}` action whose command matches the target set. An entry mixing a target action with an unrelated custom command keeps the custom command (the preserve-non-fab-hooks discipline from `2.10.1-to-2.11.0` §1 / `0.46.0-to-1.1.0` §1). Remove an entry when its `hooks[]` becomes empty; leave an emptied event as an empty array (or omit the key); never delete the `hooks` object.
2. Preserve every non-hook top-level key (`permissions`, `model`, …) verbatim.
3. Atomic write (temp file in the same directory + rename).
4. Print one line per cleaned worktree: `Removed stale fab hook entries from <worktree-path>/.claude/settings.local.json.`

Example of the stale shape being removed (as found in the wild):

```json
{
  "hooks": {
    "PostToolUse": [
      { "matcher": "Write", "hooks": [ { "type": "command", "command": "fab hook artifact-write" } ] },
      { "matcher": "Edit",  "hooks": [ { "type": "command", "command": "fab hook artifact-write" } ] }
    ],
    "SessionStart":     [ { "matcher": "", "hooks": [ { "type": "command", "command": "fab hook session-start" } ] } ],
    "Stop":             [ { "matcher": "", "hooks": [ { "type": "command", "command": "fab hook stop" } ] } ],
    "UserPromptSubmit": [ { "matcher": "", "hooks": [ { "type": "command", "command": "fab hook user-prompt" } ] } ]
  }
}
```

**Verification**:
1. No `.claude/settings.local.json` in any worktree contains a command starting with `fab hook ` or a legacy `fab/.kit/hooks/on-*.sh` shim, under any event.
2. Non-fab hooks and non-hook top-level keys preserved verbatim.
3. Re-run trips the sentinel and is a complete no-op.

No binary pre-check (pure prompt logic — no new binary capability required), no `.status.yaml` change, no `fab/` data change, no commit (unlike `2.15.1-to-2.15.2` — nothing here lands in git; the edited files are gitignored).

### 2. `src/kit/VERSION` bump

Bump `2.15.7` → `2.15.8` (patch — a pure fix with no schema or binary change; patch-target precedent: `1.9.1-to-1.9.2`, `2.13.1-to-2.13.2`, `2.15.1-to-2.15.2`). FROM is the real current released VERSION (`2.15.7`) per the chaining precedent; if another in-flight change claims the `2.15.7-to-2.15.8` slot first, re-slot per the `2.11.0-to-2.12.0` "slot note" precedent (FROM = real current VERSION at apply time).

### 3. Stale spec claims updated

Two spec files currently state that hook-entry cleanup "is done by the `2.13.6-to-2.14.0` migration" — true only for the single checkout it ran in. Extend both sentences to also name the new worktree-sweep migration:

- `docs/specs/architecture.md` (§ the `fab init`/`fab sync` paragraph, ~line 448)
- `docs/specs/skills.md` (§ Delegation pattern, ~line 157)

No skill files (`src/kit/skills/*.md`) change, so no `SPEC-*.md` mirror obligations are triggered.

### 4. Migration-authoring lesson recorded (hydrate)

`docs/memory/distribution/migrations.md` gains the new migration's catalog section plus a Design Decision capturing the general rule: **a migration touching per-checkout gitignored state (e.g. `.claude/settings.local.json`) MUST sweep all worktrees** — the version gate is committed/repo-wide, so a current-checkout-only edit permanently strands sibling checkouts. (This is the hydrate-stage output; listed here so apply scopes the memory edit into the plan.)

## Affected Memory

- `distribution/migrations`: (modify) add the `2.15.7-to-2.15.8` catalog section (worktree settings sweep) and the per-checkout-state ⇒ worktree-sweep Design Decision

## Impact

- `src/kit/migrations/2.15.7-to-2.15.8.md` — new file (the whole behavioral surface)
- `src/kit/VERSION` — `2.15.7` → `2.15.8`
- `docs/specs/architecture.md`, `docs/specs/skills.md` — one-sentence factual extensions
- `docs/memory/distribution/migrations.md` — hydrate
- No Go code, no `.status.yaml` schema, no skill files, no scaffold/fragment changes. fab-kit's own repo was already hand-cleaned (2026-07-18), so the migration no-ops here — its re-run-is-a-no-op Verification step doubles as the local validation path.

## Open Questions

*(none — all decisions resolved during the originating discussion)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove by prefix match `fab hook ` (any subcommand), not by enumerating the four known commands | The whole command family was removed in 2.14.0 — any `fab hook <x>` is a cobra unknown-command error; enumeration could strand an unlisted variant | S:80 R:85 A:90 D:85 |
| 2 | Certain | Include the legacy `bash …fab/.kit/hooks/on-*.sh` script-shim forms in the target set | Both prior settings migrations carried them; the scripts no longer exist so they also error if fired | S:75 R:85 A:85 D:80 |
| 3 | Certain | Sweep via `git worktree list --porcelain`, covering the main checkout (first entry); non-git dirs handle current directory only | Direct precedent in `2.13.6-to-2.14.0` §2; main checkout is the live poison per the docs-confirmed settings resolution | S:85 R:85 A:90 D:90 |
| 4 | Certain | No binary capability pre-check | Pure prompt/JSON-edit logic; no new flag or command needed (unlike the `2.5.5-to-2.6.0`/`2.6.6-to-2.7.0` re-baselines) | S:80 R:90 A:90 D:85 |
| 5 | Confident | Version slot `2.15.7-to-2.15.8` (patch bump, FROM = current released VERSION) | Pure-fix patch precedent (`2.15.1-to-2.15.2`); slot re-checked at apply time per the `2.11.0-to-2.12.0` slot-note precedent | S:70 R:90 A:75 D:70 |
| 6 | Confident | Update the two stale spec sentences (architecture.md, skills.md) in this change | They assert cleanup is "done by" the 2.13.6-to-2.14.0 migration — incomplete once this migration ships; specs are human-curated, so the edit is a minimal factual extension | S:65 R:85 A:75 D:70 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
