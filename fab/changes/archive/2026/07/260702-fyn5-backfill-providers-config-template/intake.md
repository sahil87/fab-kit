# Intake: Backfill Providers Config Template Migration

**Change**: 260702-fyn5-backfill-providers-config-template
**Created**: 2026-07-03

## Origin

> Ship a kit migration that brings the v2.13.1 providers config template (#467) to existing projects: append the commented codex/gemini starter blocks, claude's commented dispatch_command line, and the providers explanatory comment header to fab/project/config.yaml when absent — scaffold is copy-if-absent so existing configs never picked these up; no 2.13.0→2.13.1 migration exists today

Conversational origin (drafted via `/fab-draft` after live discussion). The gap was discovered while re-running `/fab-setup migrations` on fab-kit itself: the user noticed the codex/gemini lines were missing from `fab/project/config.yaml`. Diagnosis in-session:

- Commit `d7a87acb` (#467, shipped in v2.13.1) pre-filled three providers in the **scaffold** template (`src/kit/scaffold/fab/project/config.yaml`) — claude live, codex/gemini as commented starter blocks, plus an expanded explanatory header.
- Scaffold files are **copy-if-absent** (`fab sync`), so existing projects never receive template refreshes.
- The `2.12.1-to-2.13.0` migration predates #467 and writes only a bare `providers.claude.session_command`; **no migration file targets 2.13.0→2.13.1**, so no existing project ever gets the starter blocks.
- fab-kit's own `fab/project/config.yaml` was hand-patched in-session with the full template content (adapted to its 4-space indent); that working-tree edit is to be folded into this change.
- User then directed: "go ahead, draft it as a change."

## Why

1. **The pain point**: #467's entire purpose was discoverability — a user opening `config.yaml` should see how to wire codex/gemini providers and claude CLI dispatch without reading external docs. That benefit currently reaches only projects scaffolded on ≥ 2.13.1. Every existing project (including fab-kit itself, until the in-session hand-patch) has a bare `providers:` block with no header, no commented `dispatch_command`, and no codex/gemini starter template.
2. **The consequence of not fixing**: the installed base permanently diverges from the shipped template. Users on migrated configs don't discover multi-provider support or CLI dispatch; the scaffold improvement is effectively new-projects-only forever.
3. **Why this approach**: `fab/project/context.md` § Migrations mandates that restructuring of existing user config files ships as a migration file in `src/kit/migrations/` — not a subcommand or ad-hoc script (Constitution I, Pure Prompt Play). The catalog has two direct precedents for comment-only backfills: `2.9.2-to-2.10.0` (config-reference pointer line) and `2.11.0-to-2.12.0` (commented `spawn_command` reference note), both sentinel-guarded and value-preserving.

## What Changes

### New migration file: `src/kit/migrations/2.13.1-to-2.13.2.md`

Standard Summary / Pre-check / Changes / Verification shape. Config-only; no `.status.yaml` change, no binary capability pre-check (comments only — no new binary behavior is required for the comments to be valid).

**Pre-check**:

1. `fab/project/config.yaml` exists — else skip entirely (`Skipped: fab/project/config.yaml not present.`).
2. A top-level `providers:` key exists — else STOP with guidance: the config has not run the `2.12.1-to-2.13.0` migration (in the chained `/fab-setup migrations` flow this cannot happen — FROM-ascending order runs 2.12.1→2.13.0 first; only a direct-file invocation can hit it).
3. **Sentinel**: skip when the config already carries a `codex` or `gemini` provider key — live (`codex:` / `gemini:` mapping keys under `providers:`) or as the commented starter marker (`# codex:` / `# gemini:`). Print `Skipped: codex/gemini provider template already present.` (Comment-sentinel precedent: `2.2.0-to-2.3.0`, `2.11.0-to-2.12.0`.)

**Changes** (all comment-only; no live key is added, removed, or modified):

1. **Header refresh/insert.** If the #467 per-provider-notes paragraph (detection line: `# Per-provider notes (kept out of the blocks below so uncommenting a whole block`) is absent:
   - If the pre-#467 header is present (configs scaffolded on the 2.13.0 template — distinctive old line: `# dispatch; ABSENT → native Agent-tool dispatch). The two are NOT merged.`), replace that header paragraph with the v2.13.1 wording.
   - If no providers header exists at all (configs migrated by `2.12.1-to-2.13.0`), insert the full v2.13.1 header immediately above the `providers:` line.

   The v2.13.1 header (verbatim from the scaffold):

   ```yaml
   # providers — named agent invocation grammars. Each provider MAY carry a
   # session_command (opens an interactive session: fab operator / fab batch /
   # fab agent) and/or a dispatch_command (runs one headless stage task via fab
   # dispatch, which pipes the stage prompt to the command's STDIN; ABSENT →
   # native Agent-tool dispatch). The two are NOT merged. {model}/{effort}
   # placeholders are substituted from the resolved tier profile; a plain Claude
   # command has --model/--effort appended instead. Provider names are opaque,
   # user-chosen strings. See: fab config reference.
   #
   # claude is the built-in default (session_command shown live). codex and gemini
   # are a commented starter TEMPLATE — uncomment and adapt to add that provider.
   # Anything whose uncommenting changes default behavior ships commented: claude's
   # dispatch_command (flips claude native→headless CLI dispatch) and the opt-in
   # codex/gemini blocks.
   #
   # Per-provider notes (kept out of the blocks below so uncommenting a whole block
   # yields valid YAML — strip the leading '# ' from every line of a block):
   #   claude.dispatch_command — claude -p reads the prompt from stdin; uncommenting
   #     runs claude's stages as headless CLI processes instead of native sub-agents.
   #   codex — codex exec reads the prompt from stdin. Substitute a current model ID
   #     for {model} (e.g. gpt-5.3-codex); {model}/{effort} come from the tier.
   #   gemini — no {effort} (the gemini CLI has no reasoning-effort flag) and no -p:
   #     gemini's -p takes prompt TEXT (appended after stdin), whereas fab dispatch
   #     pipes the prompt to stdin, which gemini reads as the prompt in non-TTY mode.
   ```

2. **Claude commented `dispatch_command`.** When a `claude:` provider exists and carries no `dispatch_command` (live or commented): append the commented line directly after its `session_command`. When the old scaffold note `# no dispatch_command → claude's stages dispatch natively via the Agent tool` is present, replace it with this line. When no `claude:` provider exists (user renamed it / non-claude config, e.g. `UNNAMED_PROVIDER` from the 2.12.1→2.13.0 halt-and-ask path), skip this piece.

   ```yaml
       # dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'
   ```

3. **Codex/gemini starter blocks.** Append after the last entry of the `providers:` mapping (before the next top-level key):

   ```yaml
     # codex:
     #   session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
     #   dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
     # gemini:
     #   session_command: 'gemini -m {model}'
     #   dispatch_command: 'gemini -m {model}'   # no {effort} flag; no -p (fab dispatch pipes the prompt to stdin)
   ```

4. **Indent adaptation.** The scaffold is 2-space indented; go-yaml-written configs (e.g. fab-kit's own) are 4-space. Detect the file's mapping indent from the existing `providers:` block children and emit all commented lines so that stripping the leading `# ` from every line of a block yields valid YAML at the file's own indent. Worked example (4-space, proven by the in-session hand-patch):

   ```yaml
       # codex:
       #     session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
   ```

5. **Value preservation.** All live keys, values, and unrelated comments are preserved verbatim; `yq '.providers'` output (comments stripped) is semantically identical before and after.

**Verification** (in the migration file):

1. YAML still parses (`yq '.' fab/project/config.yaml`).
2. Live semantics unchanged: `yq '.providers'` (and `.agent`) identical to pre-migration values.
3. The per-provider-notes detection line, the commented claude `dispatch_command` (when a claude provider exists), and the `# codex:` / `# gemini:` markers are present.
4. Mechanically uncommenting a starter block in a temp copy (strip leading `# ` per line) yields valid YAML.
5. Re-run is a complete no-op (sentinel trips).

### `src/kit/VERSION` bump

`2.13.1` → `2.13.2` (patch — comment-only backfill, no binary change; patch-target precedent: `1.9.1-to-1.9.2`). FROM is the real current VERSION (`2.13.1`) per the `2.9.2-to-2.10.0` chaining precedent. Projects at local `2.13.0` reach it via a gap-skip to `2.13.1` then apply.

### fab-kit's own `fab/project/config.yaml`

Already hand-patched in-session with exactly this content (full header + commented claude dispatch_command + codex/gemini blocks, 4-space adapted); the uncommitted working-tree edit is folded into this change's branch — precedent: `2.12.1-to-2.13.0` updated fab-kit's own config in the same change. It also serves as the migration's worked example for indent adaptation.

### Explicitly NOT changing

- No Go/binary changes, no tests (markdown + comment-only YAML edits; `test_paths` untouched).
- No skill changes — `/fab-setup migrations` applies any migration file generically; no `_cli-fab.md` or SPEC mirror updates needed.
- No change to the `agent.tiers` comment block or any other scaffold section — #467's scaffold diff touched only the providers section.
- No live provider entries are written — a user who wants codex/gemini uncomments and adapts.

## Affected Memory

- `distribution/migrations.md`: (modify) add the `2.13.1-to-2.13.2` catalog entry (comment-only template backfill, sentinel-guarded, config-only, patch bump) following the existing per-migration section pattern.

## Impact

- `src/kit/migrations/2.13.1-to-2.13.2.md` (new) — the deliverable.
- `src/kit/VERSION` — `2.13.1` → `2.13.2`.
- `fab/project/config.yaml` (this repo) — already edited in the working tree; committed as part of this change.
- `docs/memory/distribution/migrations.md` — hydrate-stage catalog entry.
- User projects — affected only when they run `/fab-setup migrations` after upgrading; comments-only, zero behavior change (`yq` semantics identical).

## Open Questions

- None — the design falls out of the catalog precedents discussed in-session; remaining judgment calls are graded below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Vehicle is a kit migration file, not a script/subcommand | Mandated by context.md § Migrations + Constitution I (Pure Prompt Play); user explicitly asked for "a kit migration" | S:90 R:85 A:95 D:90 |
| 2 | Confident | Slot `2.13.1-to-2.13.2` — FROM = real current VERSION, TO = patch bump | FROM per `2.9.2-to-2.10.0` chaining precedent; patch because comment-only backfill with no binary change (`1.9.1-to-1.9.2` patch-target precedent). Release author can re-slot before release if the next release differs <!-- assumed: patch bump 2.13.2 — release author may prefer folding into a minor slot --> | S:60 R:85 A:70 D:55 |
| 3 | Confident | Sentinel: skip when any codex/gemini provider key (live or `# codex:`/`# gemini:` comment) already present | Comment-sentinel idempotency precedent (`2.2.0-to-2.3.0`, `2.11.0-to-2.12.0`); live-key check also protects users who already configured those providers | S:70 R:80 A:80 D:70 |
| 4 | Confident | Refresh the pre-#467 header wording and replace the old `# no dispatch_command →` claude note when detected | Configs scaffolded on the 2.13.0 template would otherwise mix old header + new blocks; #467's diff replaced both, and the per-provider notes are needed to understand the blocks | S:55 R:80 A:70 D:60 |
| 5 | Certain | Adapt commented blocks to the file's detected mapping indent so uncommenting yields valid YAML | Proven in-session on fab-kit's 4-space config; the scaffold's own header states the strip-`# `-to-uncomment contract | S:80 R:90 A:90 D:85 |
| 6 | Confident | No `claude:` provider present → skip the dispatch_command piece, still append codex/gemini blocks | Renamed/non-claude providers (e.g. `UNNAMED_PROVIDER`) can't take a claude-specific comment; starter blocks remain useful regardless | S:50 R:80 A:75 D:65 |
| 7 | Certain | Fold the existing working-tree edit to fab-kit's own config.yaml into this change | Precedent: `2.12.1-to-2.13.0` updated fab-kit's config in the same change; the edit is already made and matches the migration's output | S:65 R:90 A:85 D:80 |
| 8 | Confident | Pre-check requires an existing `providers:` block (else STOP with run-the-chain guidance) | Chained flow guarantees `2.12.1-to-2.13.0` runs first (FROM-ascending); only direct-file invocation can violate it | S:60 R:85 A:80 D:70 |

8 assumptions (3 certain, 5 confident, 0 tentative, 0 unresolved).
