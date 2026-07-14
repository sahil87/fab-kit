# Intake: Stable window_id in pane map JSON

**Change**: 260713-ueuy-pane-map-window-id
**Created**: 2026-07-13

## Origin

One-shot invocation of `/fab-new` with a fully-specified task description (no prior conversation; all design decisions were embedded in the input and are encoded as assumptions below). Raw input:

> Task: add a stable `window_id` field to `fab pane map --json` output.
>
> Repo: fab-kit. All changes in src/go/fab/cmd/fab/panemap.go + panemap_test.go (plus docs if a pane-map JSON contract is documented — grep docs/ for "pane map" / "window_index" and update in the same commit).
>
> Motivation (context, no action needed): run-kit joins pane-map output to its tmux snapshot to attach change/stage/display_state to windows. The JSON identifies a window only by (session, window_index), and run-kit caches the map for 5s — so after a `tmux swap-window` reorder, the stale index join misattributes one window's fab status to whichever window now occupies that index, until the cache expires (visible as a sidebar status dot lagging a window swap). `window_id` (@N) is stable for a window's lifetime and travels with the window across swap-window/move-window, so it removes the positional coupling. `window_index` MUST remain in the output (backward compat).
>
> Changes:
> 1. tmuxPaneFormat (panemap.go:171): append `\t#{window_id}` at the END, after #{@rk_agent_state}. Rationale: @rk_agent_state can be empty and is currently the trailing field (see the trailing-tab comment on parsePaneLines); making never-empty window_id the trailing field turns the possibly-empty field into a middle field, simplifying that invariant. Update the format-string doc comment accordingly.
> 2. parsePaneLines: SplitN(line, "\t", 7). len==7 → agentState=parts[5], windowID=parts[6]. Preserve the existing tolerance: len==6 → agentState present, no windowID; len==5 → neither; len<5 → skip. Keep the newline-only trim.
> 3. paneEntry: add `windowID string`. paneRow: add `windowID string`. resolvePane: copy p.windowID through in BOTH branches (non-git panes get it too — a window id exists regardless of git/fab context).
> 4. paneJSON: add `WindowID *string `json:"window_id"`` immediately after WindowIndex; emit via toNullable (null when empty/unparsed, matching the file's nullable conventions). printPaneJSON populates it.
> 5. Table output (printPaneTable): NO new column — this is a JSON-contract change only; the human table keeps WinIdx.
>
> Tests (panemap_test.go, follow existing table-driven style):
> - parsePaneLines: 7-field line; 7-field line with EMPTY agent-state middle field ("...\t3\t\t@5"); legacy 6- and 5-field lines still parse with empty windowID; <5 fields skipped.
> - JSON output: window_id present and correct; null when windowID is empty.
> - resolvePane threads windowID in both the git and non-git branches.
>
> Constraints:
> - Do not rename/remove any existing JSON key, reorder existing fields, or change the table format. Additive only.
> - Do not change CLI flags.
> - Run the repo's Go tests per its justfile/README conventions; follow the repo's changelog/version-bump convention for a minor additive change.
>
> Acceptance: `fab pane map --json --all-sessions` on a live server emits `"window_id": "@N"` matching `tmux list-panes -a -F '#{pane_id} #{window_id}'` for every pane; old fields byte-identical otherwise; all tests green.

## Why

**Problem**: run-kit's sidebar joins `fab pane map --json` rows to its own tmux window snapshot to attach `change`/`stage`/`display_state` to windows. The JSON identifies a window only positionally, by `(session, window_index)`, and run-kit caches the pane map for 5 seconds. After a `tmux swap-window` reorder, the cached rows still carry the *old* indexes, so the join attributes one window's fab status to whichever window now occupies that index — a visible bug (a sidebar status dot lagging a window swap) that persists until the cache expires.

**If not fixed**: every window reorder produces up to 5 seconds of misattributed status in any pane-map consumer that joins by index. The race is inherent to positional identity — no cache tuning fixes it, only shortens it.

**Why this approach**: tmux's `#{window_id}` (`@N`) is a server-assigned identifier that is stable for a window's lifetime and travels with the window across `swap-window`/`move-window`. Exposing it lets consumers join on stable identity instead of position, removing the coupling entirely. The alternative (run-kit invalidating its cache on window events) treats the symptom in one consumer; exposing the stable ID fixes the contract for all consumers. `window_index` stays in the output — the change is strictly additive (matching the `repo` → `display_state` → `pr_url`/`pr_number` additive-field precedent in this file).

## What Changes

All Go changes in `src/go/fab/cmd/fab/panemap.go` + `panemap_test.go`. Verified against current source (lines cited from HEAD of `main`).

### 1. `tmuxPaneFormat` — append `#{window_id}` as the new trailing field

`panemap.go:171` currently:

```go
const tmuxPaneFormat = "#{pane_id}\t#{window_name}\t#{pane_current_path}\t#{session_name}\t#{window_index}\t#{@rk_agent_state}"
```

Append `\t#{window_id}` at the END (after `#{@rk_agent_state}`), making it a seven-field format. Rationale (from the input, verified against the code): `#{@rk_agent_state}` can be empty and is currently the trailing field — the doc comments on `tmuxPaneFormat` (lines 165–170) and `parsePaneLines` (lines 229–235) both explain the trailing-tab/newline-only-trim invariant that protects a trailing empty field. Making the never-empty `window_id` the trailing field turns the possibly-empty field into a middle field, simplifying that invariant. Update **both** doc comments accordingly ("six tab-separated fields" in the `parsePaneLines` comment becomes seven; the trailing-field narrative moves from "@rk_agent_state is trailing and may leave the line ending in a tab" to "@rk_agent_state is a middle field; window_id trails and is never empty").

### 2. `parsePaneLines` — seven-field parse with legacy tolerance

Currently (`panemap.go:243-251`): `strings.SplitN(line, "\t", 6)`, `len < 5` skip, `len == 6` → `agentState = strings.TrimSpace(parts[5])`.

New behavior: `strings.SplitN(line, "\t", 7)`.

- `len == 7` → `agentState = parts[5]` (TrimSpace as today), `windowID = parts[6]`
- `len == 6` → agentState present (parts[5]), no windowID (legacy six-field line)
- `len == 5` → neither (legacy five-field line)
- `len < 5` → skip the line
- Keep the per-line newline-only trim (`strings.Trim(line, "\r\n")`, never TrimSpace) — with `@rk_agent_state` now a middle field, an empty agent state produces `\t\t` mid-line (e.g. `...\t3\t\t@5`), which SplitN preserves regardless, but the newline-only trim remains load-bearing for legacy 6-field lines whose empty agent state leaves a trailing tab.

### 3. Thread `windowID` through `paneEntry` / `paneRow` / `resolvePane`

- `paneEntry` (`panemap.go:35-42`): add `windowID string` (raw `#{window_id}` value, e.g. `"@5"`; `""` when absent from a legacy line). Update the struct doc comment.
- `paneRow` (`panemap.go:45-58`): add `windowID string`.
- `resolvePane` (`panemap.go:306-385`): copy `p.windowID` through in **BOTH** branches — the non-git early-return branch (`panemap.go:309-329`) and the git branch (`panemap.go:371-384`). A window ID exists regardless of git/fab context (same axis-independence argument the agent-state fields already document in the non-git branch comment).

### 4. `paneJSON` — nullable `window_id` immediately after `WindowIndex`

`paneJSON` (`panemap.go:404-418`): add the field **immediately after** `WindowIndex`:

```go
WindowID *string `json:"window_id"`
```

`printPaneJSON` (`panemap.go:474-501`) populates it via the existing `toNullable(r.windowID)` helper — `null` when empty/unparsed (legacy input or missing field), matching the file's nullable conventions (`repo`/`change`/`stage`/`display_state`/`pr_url`). Placement after `WindowIndex` puts the stable identifier next to the positional one in emitted JSON.

### 5. Table output — deliberately unchanged

`printPaneTable` gets **no** new column. This is a JSON-contract change only; the human table keeps `WinIdx`. (Same `--json`-only pattern as `repo`, `display_state`, `pr_url`/`pr_number`.)

### 6. Docs — CLI reference in the same change; memory at hydrate

Grep of docs/ + src/kit/ for the pane-map JSON contract found exactly three files naming `window_index`:

- **`src/kit/skills/_cli-fab.md:420`** (the `fab pane map` `--json` flag row, which enumerates the snake_case JSON field list) — MUST be updated in this change per the constitution ("Changes to the `fab` CLI … MUST update `src/kit/skills/_cli-fab.md`"). Add `window_id` to the field list plus a one-clause contract note (`string|null`; tmux `@N` window ID, stable across swap-window/move-window; `null` when unavailable; `--json` only, no table column).
- **`docs/memory/runtime/pane-commands.md:40`** (the "JSON fields" paragraph) — memory file; updated at **hydrate** per the pipeline (lands in the same PR). Add `window_id` to the field list and a short field paragraph following the `display_state`/`pr_url` precedent (nullability contract, run-kit join motivation).
- **`docs/memory/distribution/kit-architecture.md:355`** — mentions the pane-map tmux format string ("extends the tmux format string with `#{session_name}` and `#{window_index}`"); sweep at hydrate for the one-phrase update.

`docs/specs/skills/SPEC-_cli-fab.md` (mirror of `_cli-fab.md`) summarizes `fab pane` at a level that does not restate JSON field names — verify at apply that no claim there changes; per code-review.md reviewers read the mirror rule strictly, so confirm rather than assume.

### 7. Tests (`panemap_test.go`, existing table-driven style)

Extend the existing suites (`TestParsePaneLines` at line 325, `TestPrintPaneJSON` at 616, and the `TestResolvePane*` pattern):

- **parsePaneLines**: a 7-field line (windowID populated); a 7-field line with an EMPTY agent-state middle field (`"...\t3\t\t@5"` → `agentState == ""`, `windowID == "@5"`); legacy 6-field and 5-field lines still parse with `windowID == ""`; `<5`-field lines skipped.
- **JSON output**: `window_id` present and correct when set; `null` when `windowID` is empty.
- **resolvePane**: threads `windowID` in both the git and non-git branches.

Run per repo convention: `cd src/go/fab && go test ./cmd/fab/ -count=1` scoped first, then `just test` (both modules) before finishing apply.

## Affected Memory

- `runtime/pane-commands`: (modify) add `window_id` to the `fab pane map --json` field list + a field paragraph (nullability contract, stable-identity join motivation, additive-shape precedent)
- `distribution/kit-architecture`: (modify) one-phrase sweep — the pane-map format-string mention ("extends the tmux format string with `#{session_name}` and `#{window_index}`") gains `#{window_id}`

## Impact

- **Code**: `src/go/fab/cmd/fab/panemap.go` (format string, parser, structs, resolvePane, paneJSON, printPaneJSON), `src/go/fab/cmd/fab/panemap_test.go`. No other Go packages — `tmuxPaneFormat`/`parsePaneLines`/`resolvePane` are package-private to `cmd/fab`.
- **Skills/docs**: `src/kit/skills/_cli-fab.md` (apply); `docs/memory/runtime/pane-commands.md`, `docs/memory/distribution/kit-architecture.md` (hydrate). SPEC mirror verify-only.
- **Contract**: strictly additive JSON — no key renamed/removed/reordered, table byte-identical, no CLI flag changes. Existing consumers ignore unknown keys (established precedent: `repo` h3jk, `display_state`, `pr_url`/`pr_number` r7ju).
- **External consumer**: run-kit's sidebar join opts in separately in its own repo — out of scope here.
- **Versioning**: no CHANGELOG file in this repo; version bumps are separate release commits via `just release <bump>` (`scripts/release.sh`). This ships as a normal feature PR; the next minor release picks it up.

## Open Questions

*None — the input specified the design, placement, parse tolerance, nullability, test matrix, and constraints; all decision points scored Certain/Confident.*

## Assumptions

<!-- STATE TRANSFER: This table is the sole continuity mechanism between the intake-stage
     agent and the apply-entry agent (which co-generates plan.md). Pipeline stages may execute
     in separate agent contexts with no shared memory — this table is what gives downstream
     agents visibility into what was decided, assumed, or left open. Every row must be substantive. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `#{window_id}` appended as the new TRAILING field of `tmuxPaneFormat`, after `#{@rk_agent_state}`; both format-string and parser doc comments updated to the seven-field / never-empty-trailing narrative | Specified verbatim in the input with rationale; verified against panemap.go:165-171 — the trailing-tab invariant exists exactly as described | S:95 R:85 A:95 D:95 |
| 2 | Certain | `parsePaneLines` uses `SplitN(line, "\t", 7)` with graded tolerance: 7→both, 6→agentState only, 5→neither, <5→skip; newline-only trim kept | Specified verbatim; mirrors the existing len==6/len==5 tolerance at panemap.go:243-251 | S:95 R:85 A:90 D:90 |
| 3 | Certain | `window_id` emitted as `WindowID *string` immediately after `WindowIndex` in `paneJSON`, via `toNullable` (null when empty); no table column; `window_index` retained | Specified verbatim; matches the file's nullable convention and the repo/display_state/pr_url `--json`-only precedent | S:95 R:90 A:95 D:95 |
| 4 | Certain | `resolvePane` threads `p.windowID` through BOTH branches — non-git panes get it too | Specified verbatim; consistent with the existing axis-independence comment in the non-git branch (agent state resolves regardless of fab context) | S:95 R:90 A:95 D:95 |
| 5 | Certain | `windowID` stored verbatim as a raw string (e.g. `"@5"`) — no `@N` validation, no int parse; "unparsed" simply means empty | Input says "null when empty/unparsed"; the field is an opaque tmux identifier, same treatment as the raw agentState option string | S:75 R:90 A:90 D:85 |
| 6 | Certain | Tests extend the existing table-driven suites in `panemap_test.go` per the input's enumerated matrix; run scoped `go test ./cmd/fab/` first, then `just test` | Test matrix specified verbatim; runner convention verified in justfile + code-quality.md (scope down, then widen) | S:90 R:95 A:95 D:95 |
| 7 | Certain | No CHANGELOG edit and no version-bump commit in this PR — releases are separate `release: vX.Y.Z` commits via `just release <bump>`; a minor additive change simply rides the next release | Verified: no CHANGELOG file exists; scripts/release.sh + recent history (`release: v2.15.3`) show release commits are minted separately | S:55 R:95 A:90 D:85 |
| 8 | Confident | "Update docs in the same commit" is satisfied pipeline-style: `_cli-fab.md` (constitution-bound CLI reference) edits at apply; the two memory files edit at hydrate — same PR, not literally the same commit | Repo convention: memory files are hydrate-stage artifacts; net effect (docs land with the change's PR) matches the input's intent | S:65 R:90 A:85 D:70 |
| 9 | Confident | `docs/specs/skills/SPEC-_cli-fab.md` needs no content change (its `fab pane` row doesn't restate JSON field names) — but apply verifies the mirror class rather than assuming | Verified by grep (no `window_index`/field-list restatement in the SPEC); residual uncertainty only because code-review.md says reviewers read the mirror rule strictly | S:60 R:95 A:75 D:75 |

9 assumptions (7 certain, 2 confident, 0 tentative, 0 unresolved).
