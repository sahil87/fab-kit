# Intake: Add _cli-rk Skill

**Change**: 260416-mgsm-add-cli-rk-skill
**Created**: 2026-04-16
**Status**: Draft

## Origin

> Add `_cli-rk.md` skill to the always-load layer — a new internal skill file documenting run-kit (rk) capabilities: iframe windows, proxy URL pattern, and the centralized recipe for any skill to show HTML to the user via rk. Always loaded, fails silently if rk or visual-explainer plugin unavailable. Server URL captured at use-time via `rk context`. Update `_preamble.md` to include it as an optional always-load file. Update corresponding specs per constitution.

Discussion session preceded this draft. User wants every fab session to be auto-aware of rk's iframe+proxy capabilities, enabling the visual-explainer combo (generate HTML → serve → open iframe → user sees it in tmux). Two architecture options were evaluated (centralized recipe in `_cli-rk.md` vs. baked into visual-explainer only); centralized was chosen for separation of concerns and universal skill access.

## Why

Run-kit (`rk`) provides iframe windows and a proxy that let any tmux pane display web content. Combined with visual-explainer (which generates standalone HTML), this creates a powerful feedback loop: the agent generates a diagram, plan, or slide deck and the user sees it immediately in their tmux session — no context switch, no manual browser open.

Today, no fab skill knows about these capabilities. The agent can't proactively show visual output to the user because the iframe+proxy recipe isn't documented in the always-load layer. Adding `_cli-rk.md` to the always-load layer gives every fab session this superpower.

Without this change, agents must either be told about rk capabilities each session (via manual `rk context` pasting) or remain unaware of the visual display channel entirely.

## What Changes

### 1. New skill file: `src/kit/skills/_cli-rk.md`

A new internal skill file (not user-invocable, not model-invocable) documenting rk capabilities. Content covers:

- **Detection**: `command -v rk` check. All rk usage must fail silently if rk is not installed.
- **Iframe windows**: How to create a tmux window that displays a web page:
  ```sh
  tmux new-window -n <name>
  tmux set-option -w @rk_type iframe
  tmux set-option -w @rk_url <url>
  ```
- **Proxy URL pattern**: `/proxy/{port}/...` — access local services through the rk server.
- **Server URL discovery**: Run `rk context` at use-time to get the current server URL (do not hardcode).
- **Visual display recipe** (the centralized pattern): Any skill that wants to show HTML to the user follows this recipe:
  1. Generate HTML file to a known location
  2. Serve it (e.g., `python3 -m http.server <port>` or use an existing dev server)
  3. Open an rk iframe window pointing to the proxy URL
  4. Fail silently at any step if rk is unavailable
- **Visual-explainer integration note**: When the visual-explainer plugin is available, skills can delegate HTML generation to it and then use the recipe above to display the result. If visual-explainer is not available, skills should fail silently (no error, just skip the visual display).

The file follows the same frontmatter pattern as `_cli-external.md` and `_cli-fab.md`:
```yaml
---
name: _cli-rk
description: "Run-kit (rk) capabilities — iframe windows, proxy, visual display recipe. Always loaded; fails silently if rk unavailable."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
```

### 2. Update `_preamble.md` — always-load layer

Add `_cli-rk` to the always-load section (Section 1), after the existing `_cli-fab` and `_naming` entries:

```
Also read the **`_cli-rk`** skill (deployed to `.claude/skills/`) — run-kit iframe windows, proxy, and visual display recipe. Skip gracefully if the file is missing or rk is not available on the system.
```

This is an **optional** entry — skills skip it if the file is missing (projects that don't use rk) or if `rk` is not installed on the system.

### 3. Update specs per constitution

The constitution requires: "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file." This change needs:

- A new `docs/specs/skills/SPEC-_cli-rk.md` spec file for the new skill
- Update to `docs/specs/skills.md` if it maintains a list of internal skills

## Affected Memory

- `fab-workflow/context-loading`: (modify) Document `_cli-rk.md` as an always-load skill and the rk detection/silent-fail behavior

## Impact

- **All fab skills**: Gain awareness of rk capabilities via always-load layer. No behavioral change for projects without rk — the file is skipped silently.
- **`_preamble.md`**: One new optional entry in the always-load section.
- **`_cli-external.md`**: No changes. `_cli-rk.md` is a separate file — `_cli-external.md` remains operator-only.
- **Subagent context**: The standard subagent context list (5 files) does NOT change. `_cli-rk` is loaded via the always-load skill layer, not the project file layer.

## Open Questions

- Should `_cli-rk.md` document the `rk context` full output format, or just the subset needed for iframe+proxy? (Leaning toward subset — keep it focused.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Centralized recipe in `_cli-rk.md`, not in visual-explainer | Discussed — user chose Option A (centralized) over Option B (decentralized) | S:95 R:85 A:90 D:95 |
| 2 | Certain | Always-load, fail silently if rk unavailable | Discussed — user explicitly confirmed silent failure on missing rk | S:95 R:90 A:85 D:95 |
| 3 | Certain | Server URL captured at use-time via `rk context` | Discussed — user explicitly chose use-time over static config | S:95 R:90 A:90 D:95 |
| 4 | Certain | Fail silently if visual-explainer plugin unavailable | Discussed — user explicitly confirmed | S:90 R:90 A:85 D:95 |
| 5 | Certain | Separate file from `_cli-external.md` | Discussed — user chose `_cli-rk.md` over adding to `_cli-external.md` | S:95 R:85 A:90 D:95 |
| 6 | Confident | Spec file needed at `docs/specs/skills/SPEC-_cli-rk.md` | Constitution requires spec updates for skill changes; new skill = new spec file | S:70 R:80 A:85 D:75 |
| 7 | Confident | `_cli-rk.md` documents only the iframe/proxy/recipe subset of rk, not full `rk context` output | Keeps the file focused; agents run `rk context` at use-time for full details | S:70 R:85 A:80 D:70 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
