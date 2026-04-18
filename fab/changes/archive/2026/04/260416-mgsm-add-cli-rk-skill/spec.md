# Spec: Add _cli-rk Skill

**Change**: 260416-mgsm-add-cli-rk-skill
**Created**: 2026-04-16
**Affected memory**: `docs/memory/fab-workflow/context-loading.md`

## Non-Goals

- Modifying `_cli-external.md` — rk is a separate file, not added to the operator-only reference
- Changing the standard subagent context (5 `fab/project/**` files) — `_cli-rk` is loaded via the always-load skill layer, not the project file layer
- Documenting the full `rk context` output — agents run `rk context` at use-time for full details

## Context Loading: _cli-rk Always-Load Skill

### Requirement: New `_cli-rk.md` Skill File

The kit SHALL include a new internal skill file at `src/kit/skills/_cli-rk.md` documenting run-kit (rk) capabilities. The file SHALL use the standard internal skill frontmatter (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`).

#### Scenario: File deployed via fab sync
- **GIVEN** `src/kit/skills/_cli-rk.md` exists in the kit
- **WHEN** `fab sync` runs
- **THEN** the file is deployed to `.claude/skills/_cli-rk/SKILL.md`

### Requirement: _cli-rk Content — Detection

The skill file SHALL document that all rk usage MUST be guarded by availability detection. The agent SHALL check `command -v rk` before using any rk command. If rk is not available, the agent SHALL skip all rk-related operations silently (no error, no warning to user).

#### Scenario: rk not installed
- **GIVEN** `rk` is not installed on the system
- **WHEN** an agent reads `_cli-rk.md` and attempts to use rk capabilities
- **THEN** the agent detects rk is unavailable via `command -v rk`
- **AND** skips all rk operations silently

#### Scenario: rk installed
- **GIVEN** `rk` is installed and available on PATH
- **WHEN** an agent reads `_cli-rk.md` and wants to use rk capabilities
- **THEN** the agent proceeds with rk operations

### Requirement: _cli-rk Content — Iframe Windows

The skill file SHALL document the iframe window creation pattern using tmux user options:

```sh
tmux new-window -n <name>
tmux set-option -w @rk_type iframe
tmux set-option -w @rk_url <url>
```

The skill file SHALL also document URL changes for existing iframe windows via `tmux set-option -w @rk_url <new-url>`.

#### Scenario: Agent creates iframe window
- **GIVEN** rk is available and the server is running
- **WHEN** an agent wants to display web content to the user
- **THEN** the agent creates a tmux window with `@rk_type iframe` and `@rk_url` set to the target URL

### Requirement: _cli-rk Content — Proxy URL Pattern

The skill file SHALL document the proxy URL pattern: `/proxy/{port}/...` for accessing local services through the rk server. The agent SHALL discover the server URL at use-time by running `rk context` and parsing the `Server URL` field — never hardcode the URL.

#### Scenario: Agent serves HTML and creates proxy URL
- **GIVEN** rk is available and a local service is running on port 8080
- **WHEN** an agent wants to display the service output in an iframe
- **THEN** the agent constructs the URL as `{server_url}/proxy/8080/`
- **AND** the server URL was obtained from `rk context` at use-time

### Requirement: _cli-rk Content — Visual Display Recipe

The skill file SHALL document a centralized recipe for any skill to show HTML content to the user via rk. The recipe SHALL consist of four steps:

1. Generate HTML file to a known location
2. Serve it (e.g., `python3 -m http.server <port>` in the file's directory)
3. Open an rk iframe window pointing to the proxy URL for that port
4. Fail silently at any step if rk is unavailable

This is the centralized pattern — any skill that reads `_cli-rk.md` gains this capability. Visual-explainer is not required.

#### Scenario: Skill uses visual display recipe
- **GIVEN** rk is available and the agent has generated an HTML file at `/tmp/diagram.html`
- **WHEN** the agent follows the visual display recipe
- **THEN** the agent starts a local server, creates an iframe window via the proxy, and the user sees the content
- **AND** if any step fails, the remaining steps are skipped silently

#### Scenario: Visual display without rk
- **GIVEN** rk is not installed
- **WHEN** the agent attempts to follow the visual display recipe
- **THEN** the agent skips the display step silently
- **AND** no error is shown to the user

### Requirement: _cli-rk Content — Visual-Explainer Integration

The skill file SHALL include a note that when the visual-explainer plugin is available, skills MAY delegate HTML generation to it and then use the visual display recipe to show the result. If visual-explainer is not available, skills SHALL fail silently (skip visual display entirely, no error).

#### Scenario: Visual-explainer available
- **GIVEN** rk is available and the visual-explainer plugin is installed
- **WHEN** a skill wants to show a diagram
- **THEN** the skill delegates HTML generation to visual-explainer and uses the rk iframe recipe to display it

#### Scenario: Visual-explainer not available
- **GIVEN** the visual-explainer plugin is not installed
- **WHEN** a skill wants to show a diagram
- **THEN** the skill skips the visual display silently

### Requirement: Always-Load Layer Integration

`_preamble.md` Section 1 (Always Load) SHALL include `_cli-rk` as an optional always-load skill, listed after the existing `_cli-fab` and `_naming` entries. The entry SHALL instruct agents to skip gracefully if the file is missing or rk is not available.

#### Scenario: Project with rk
- **GIVEN** `_cli-rk.md` is deployed to `.claude/skills/` and rk is available
- **WHEN** a skill loads the always-load layer per `_preamble.md`
- **THEN** the agent reads `_cli-rk` and gains rk capabilities

#### Scenario: Project without _cli-rk file
- **GIVEN** `_cli-rk.md` is not deployed (e.g., older kit version)
- **WHEN** a skill loads the always-load layer per `_preamble.md`
- **THEN** the agent skips `_cli-rk` without error and proceeds normally

### Requirement: Skill Spec File

A new spec file SHALL be created at `docs/specs/skills/SPEC-_cli-rk.md` following the existing pattern (summary, flow, tools used, sub-agents).

#### Scenario: Spec file exists
- **GIVEN** the change is complete
- **WHEN** a developer checks `docs/specs/skills/`
- **THEN** `SPEC-_cli-rk.md` exists and documents the skill's purpose, loading, and content structure

## Design Decisions

1. **Centralized recipe in `_cli-rk.md`, not in visual-explainer**
   - *Why*: Any skill gains the visual display superpower by reading `_cli-rk.md`. Separation of concerns — visual-explainer focuses on HTML generation, `_cli-rk.md` owns the display channel.
   - *Rejected*: Baking iframe logic into visual-explainer only — forces other skills to duplicate logic or use visual-explainer as a middleman.

2. **Separate file from `_cli-external.md`**
   - *Why*: `_cli-external.md` is operator-only (heavy tmux/wt docs). `_cli-rk.md` is always-load (lightweight, universal). Different loading scopes require different files.
   - *Rejected*: Adding rk to `_cli-external.md` and promoting to always-load — would bloat every session with operator-specific content.

3. **Use-time server URL discovery**
   - *Why*: The rk server URL can change between sessions. Running `rk context` at use-time ensures the URL is always fresh. Avoids stale static configuration.
   - *Rejected*: Static URL in config or environment variable — goes stale, requires manual maintenance.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Centralized recipe in `_cli-rk.md`, not in visual-explainer | Confirmed from intake #1 — user chose Option A (centralized) | S:95 R:85 A:90 D:95 |
| 2 | Certain | Always-load, fail silently if rk unavailable | Confirmed from intake #2 — user explicitly confirmed | S:95 R:90 A:85 D:95 |
| 3 | Certain | Server URL captured at use-time via `rk context` | Confirmed from intake #3 — user explicitly chose use-time | S:95 R:90 A:90 D:95 |
| 4 | Certain | Fail silently if visual-explainer plugin unavailable | Confirmed from intake #4 — user explicitly confirmed | S:90 R:90 A:85 D:95 |
| 5 | Certain | Separate file from `_cli-external.md` | Confirmed from intake #5 — user chose `_cli-rk.md` | S:95 R:85 A:90 D:95 |
| 6 | Certain | Spec file at `docs/specs/skills/SPEC-_cli-rk.md` | Constitution requires spec updates for skill changes | S:85 R:85 A:90 D:90 |
| 7 | Confident | Document only iframe/proxy/recipe subset of rk | Confirmed from intake #7 — focused content, agents run `rk context` at use-time for full details | S:75 R:85 A:80 D:75 |
| 8 | Confident | `_cli-rk` listed after `_cli-fab` and `_naming` in preamble | Natural ordering — CLI references grouped together, rk is newest addition | S:70 R:90 A:80 D:70 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
