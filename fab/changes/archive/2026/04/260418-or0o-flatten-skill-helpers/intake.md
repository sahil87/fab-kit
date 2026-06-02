# Intake: Flatten Skill Helper Include Tree

**Change**: 260418-or0o-flatten-skill-helpers
**Created**: 2026-04-18
**Status**: Draft

## Origin

Surfaced during a `/fab-discuss` session exploring context-loading behavior. User asked
how to optimize the tree of skill files loaded before a skill executes. Agent mapped the
full include tree across all 24 user-facing skills plus 7 helper skills.

> "The total no of skill layers need to be manageable. Need a balance. No point of
> slimming down skill files too much. Many times the agent decides to skip reading the
> 2nd layer includes. So the flatter the tree the better."

Subsumes open backlog item `[84bh]` (2026-04-02): *"Try using references instead of `_`
folders in skills - other agents don't read from `_` folders in skills."* The current
change addresses the same root concern — unreliable deeper-layer reads — with a concrete
structural fix rather than a renaming convention.

Interaction mode: conversational. Decisions were reached iteratively:
1. First proposed splitting `_preamble` into smaller section files → rejected by user
   (would deepen the tree, worsening the skip-rate problem).
2. Then proposed consolidating: inline small helpers, scope larger ones per-skill via
   frontmatter → accepted directionally.
3. User then required a full cross-skill audit before committing → agent mapped the
   complete tree (this intake encodes those findings).

## Why

**Problem 1 — universal fanout from `_preamble` that most skills don't need.** `_preamble/SKILL.md`
currently contains three "Also read" directives:

```markdown
Also read the **`_cli-fab`** skill ... — script invocation conventions ...
Also read the **`_naming`** skill ... — naming conventions ...
Also read the **`_cli-rk`** skill ... — run-kit iframe windows ... (optional)
```

Every skill inherits these pointers, pulling `_cli-fab` (773 lines), `_naming` (76), and
`_cli-rk` (91) even when the skill runs zero `fab` commands, creates no named artifacts,
and renders no panes. Measured impact: read-only skills like `/fab-discuss`,
`/fab-status`, `/fab-help`, `/fab-clarify`, `/fab-archive`, `/docs-hydrate-*`, `/git-*`,
`/internal-*` (about 15 of 24 skills) load ~1324 lines of helper content they don't use.

**Problem 2 — agents silently skip second-layer includes.** User's stated observation: when
`_preamble` points at further files via "also read" directives, the agent often loads
`_preamble` itself but fails to follow the pointer. This means the universal fanout isn't
even reliably happening — paying context cost when the load succeeds, paying correctness
cost when it silently doesn't.

**Problem 3 — `_cli-fab` is disproportionately heavy.** At 773 lines it is 45% of total
helper content, yet most skills use 2–5 commands from it (`preflight`, `score`, `log`,
`resolve`, `status`). Full reference manual, minimal use.

**If we don't fix it:**
- Every skill invocation pays for content it won't use — measurable context waste
  (~1300 lines × 24 skills of unnecessary load potential per session).
- Skill behavior becomes non-deterministic depending on whether the agent chose to follow
  the 2nd-layer pointers this turn.
- Adding new helpers makes the problem worse linearly, discouraging modularization even
  where it would help.

**Why this approach over alternatives:**

| Alternative considered | Rejected because |
|---|---|
| Split `_preamble` into more granular sections | Deepens tree → worsens skip-rate problem. Directly contradicts user's stated principle. |
| Rely on prompt caching, leave structure alone | Doesn't fix skip-rate, doesn't reduce context budget pressure, doesn't help first-turn latency. |
| Inline everything into every skill (full duplication) | Destroys DRY; every fab-CLI change edits 24 files. |
| Rename `_` → non-`_` folders (backlog 84bh premise) | Addresses visibility but not the structural fanout. A skill file called `cli-fab` still has the same universal-pull problem if every skill references it. |

Chosen approach: **per-skill explicit `helpers:` list in frontmatter** (replaces implicit
"also read" fanout) + **inline the smallest helpers** (`_naming`, `_cli-rk`) into
`_preamble` + **compress `_cli-fab`** and lift its common commands into `_preamble`.

## What Changes

### Change 1 — Remove "Also read" fanout from `_preamble`

Delete the three "Also read" paragraphs from `src/kit/skills/_preamble.md` §1 (Context
Loading → Always Load). They currently appear as lines 48, 50, 52 in the deployed copy:

```markdown
<!-- DELETE -->
Also read the **`_cli-fab`** skill (deployed to `.claude/skills/`) — script invocation
conventions (argument formats, stage transitions, error patterns). This is the
authoritative reference for calling `fab status`, `fab change`, `fab score`, and `fab
preflight`.

Also read the **`_naming`** skill (deployed to `.claude/skills/`) — naming conventions
for change folders, git branches, worktree directories, and operator spawning rules.

Also read the **`_cli-rk`** skill (deployed to `.claude/skills/`) — run-kit iframe
windows, proxy, and visual display recipe. *(optional — skip gracefully if the file is
missing or `rk` is not available on the system)*
```

Replace with a single sentence pointing to the new per-skill declaration convention:

```markdown
Additional helpers beyond this preamble are declared by each skill in its frontmatter
`helpers:` list (see Skill Helper Declaration below). `_preamble` loads nothing extra
by default.
```

### Change 2 — Add `helpers:` frontmatter convention

Add a new subsection to `_preamble.md` defining the convention:

```markdown
## Skill Helper Declaration

A skill may declare additional helper files it needs to load via frontmatter:

    ---
    name: fab-continue
    description: ...
    helpers: [_generation, _review]
    ---

Allowed values: `_generation`, `_review`, `_cli-fab`, `_cli-external`. (`_naming` and
`_cli-rk` are inlined into `_preamble`; `_preamble` itself is implicit — never list it.)

The agent MUST read `.claude/skills/{helper}/SKILL.md` for each declared helper after
reading `_preamble` and before executing the skill body. Skills that declare no
`helpers:` list load only `_preamble`.
```

Audit every skill and set `helpers:` correctly. Target mapping:

| Skill | `helpers:` |
|---|---|
| fab-discuss, fab-status, fab-help, fab-clarify, fab-archive, fab-setup, fab-switch, fab-proceed, fab-new (see note), docs-hydrate-memory, docs-hydrate-specs, docs-reorg-memory, docs-reorg-specs, internal-consistency-check, internal-retrospect, internal-skill-optimize, git-branch, git-pr, git-pr-review | *(none)* |
| fab-new, fab-draft | `[_generation]` |
| fab-continue, fab-ff, fab-fff | `[_generation, _review]` |
| fab-operator | `[_cli-fab, _cli-external]` |

Note on `fab-new` / `fab-draft`: they need `_generation` for intake generation procedure.
They don't need `_cli-fab` because the 5 common commands will be inlined into `_preamble`
(Change 4).

### Change 3 — Inline `_naming` and `_cli-rk` into `_preamble`

Move the full contents of `src/kit/skills/_naming.md` (76 lines, excluding frontmatter)
and `src/kit/skills/_cli-rk.md` (91 lines, excluding frontmatter) into new subsections of
`src/kit/skills/_preamble.md`:

```markdown
## Naming Conventions
{inlined body of _naming.md}

## Run-Kit (rk) Reference
{inlined body of _cli-rk.md}
```

After content is moved, delete `src/kit/skills/_naming.md` and `src/kit/skills/_cli-rk.md`.
Also delete the deployed copies `.claude/skills/_naming/SKILL.md` and
`.claude/skills/_cli-rk/SKILL.md` (they regenerate via `fab sync`, but stale directories
persist until explicit removal).

Update inline references in other skill files:
- `src/kit/skills/git-branch.md` line 9: `> Branch naming conventions are defined in _naming.md.` → `> Branch naming conventions are defined in _preamble.md § Naming Conventions.`
- `src/kit/skills/git-pr.md` line 9: same substitution.
- `src/kit/skills/fab-operator.md` lines 41–43: update the "Required Reading" list — remove `_naming.md` line (now inlined), keep `_cli-fab.md` and `_cli-external.md` lines.

### Change 4 — Compress `_cli-fab` and lift common commands into `_preamble`

Two sub-steps:

**4a — Inline the 5 most-used commands into `_preamble`**. Add a new subsection to
`_preamble.md`:

```markdown
## Common fab Commands

These 5 commands cover 90% of skill usage. See `_cli-fab` for the full reference.

| Command | Purpose | Common form |
|---|---|---|
| `fab preflight [change]` | Validate init + resolve active change; output YAML with id/name/stage/progress/checklist/confidence | `fab preflight` |
| `fab score [--check-gate] [--stage <stage>] <change>` | Compute confidence; `--check-gate` returns non-zero below threshold | `fab score --check-gate --stage intake <id>` |
| `fab log command "<skill>" [<id>]` | Best-effort command telemetry; failures ignored | `fab log command "fab-discuss" 2>/dev/null \|\| true` |
| `fab resolve [--folder]` | Resolve active change from `.fab-status.yaml`; `--folder` prints just the folder name | `fab resolve --folder 2>/dev/null` |
| `fab status ...` | Subcommands: `advance`, `finish`, `add-issue`, `set-change-type`, `set-checklist` | see `_cli-fab` for each |
```

**4b — Compress `_cli-fab` to ≤300 lines**. Target reductions:
- Convert prose blocks to syntax/flag tables where possible.
- Drop redundant worked examples when one suffices.
- Remove the 5 common commands covered in 4a (cross-reference `_preamble` instead).
- Drop historical rationale sections (those belong in `docs/memory/fab-workflow/`, not
  in the skill helper).

Do NOT remove content that is the canonical source of truth for specific flag behavior.
If unsure, move to `docs/memory/fab-workflow/` rather than delete.

### Change 5 — Update specs and memory

- `docs/specs/skills.md` — document the `helpers:` frontmatter field and the allowed
  values list.
- `docs/specs/skills/SPEC-_preamble.md` (if exists) or equivalent — reflect the new
  structure (inlined Naming + Run-Kit subsections, Common fab Commands table, removed
  "also read" directives).
- `docs/memory/fab-workflow/context-loading.md` — update references. Lines that
  currently state that `_cli-fab` / `_naming` / `_cli-rk` are loaded "via _preamble" must
  be revised to describe the new `helpers:` opt-in model.
- `docs/memory/fab-workflow/kit-architecture.md` — update helper-file inventory; remove
  `_naming` and `_cli-rk` entries.

### Change 6 — Close backlog item

Remove `[84bh]` from `fab/backlog.md` (its premise is addressed structurally by this
change).

## Affected Memory

- `fab-workflow/context-loading.md`: (modify) Update "Always Load" section to describe the
  new `helpers:` frontmatter mechanism. Remove references to `_cli-fab`, `_naming`,
  `_cli-rk` as universally loaded. Document per-skill opt-in.
- `fab-workflow/kit-architecture.md`: (modify) Update helper-file inventory — remove
  `_naming` and `_cli-rk` entries, note that their content is inlined into `_preamble`.
  Note `_cli-fab` compression.

## Impact

**Source files touched:**
- `src/kit/skills/_preamble.md` (major edits: delete 3 "also read" blocks, add 3
  subsections: `## Skill Helper Declaration`, `## Naming Conventions`, `## Run-Kit (rk)
  Reference`, `## Common fab Commands`).
- `src/kit/skills/_cli-fab.md` (compression pass).
- `src/kit/skills/_naming.md` (delete).
- `src/kit/skills/_cli-rk.md` (delete).
- All 24 user-facing skill files in `src/kit/skills/` (add `helpers:` frontmatter
  where applicable; 19 skills get no change to frontmatter, 5 skills get a short list).
- `src/kit/skills/git-branch.md`, `src/kit/skills/git-pr.md`, `src/kit/skills/fab-operator.md`
  (inline reference updates).
- `docs/specs/skills.md` (document convention).
- `docs/specs/skills/SPEC-_preamble.md` if exists (reflect structure changes).
- `docs/memory/fab-workflow/context-loading.md` (update).
- `docs/memory/fab-workflow/kit-architecture.md` (update).
- `fab/backlog.md` (remove 84bh).

**Deployment:** `.claude/skills/` is gitignored and regenerated via `fab sync`. After
source edits land, `fab sync` updates deployed copies. Test by running `fab sync` and
confirming `.claude/skills/_naming/` and `.claude/skills/_cli-rk/` are absent and
`_preamble` contains the inlined content.

**No dependency changes.** No migration required — the `helpers:` field is additive
metadata; skills without it still work (they just get the new default of "load only
`_preamble`"). The real behavioral change is that the "also read" fanout from `_preamble`
stops happening, which means any skill that genuinely needed `_cli-fab` must declare it.
Missing declarations would surface as runtime failures (skill calls a fab command the
agent doesn't know the syntax for). Audit against the mapping in Change 2 before merge.

**No constitution violation.** Constitution XI (changes to skill files must update
corresponding SPEC docs) — honored via Change 5.

## Open Questions

- Should the `helpers:` allowed-values list be enforced (e.g., by `fab sync` rejecting
  unknown helpers) or purely conventional? Enforcement adds safety but requires a
  binary-side change. Convention-only keeps the change in the markdown layer.
- Is there an existing test or validation that audits skill frontmatter? If so, it needs
  an update to recognize the new `helpers:` field. If not, this may be a candidate for a
  follow-up (not blocking).
- `_generation` is currently referenced inline as `_generation.md` (the file path) by
  planning skills, not via frontmatter. Should those inline references remain (they're
  procedure-doc references within body text, not helper loads), or should they also move
  to the `helpers:` list? Recommendation: keep body references; frontmatter lists the
  auto-load set.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove "also read" fanout directives from `_preamble` | Discussed — user explicitly agreed in conversation that the universal fanout is the root problem. Confirmed via cross-skill audit showing 15/24 skills don't use those helpers. | S:95 R:80 A:90 D:95 |
| 2 | Certain | Leave `_generation`, `_review`, `_cli-external` alone | Cross-skill audit showed these are already inline-referenced only by the exact skills that need them (5 planning skills + 1 operator). No change warranted. | S:95 R:90 A:95 D:95 |
| 3 | Certain | `_naming` and `_cli-rk` are small enough to inline | Measured: 76 + 91 = 167 lines total. Inlining flattens tree without meaningful bloat to `_preamble`. | S:95 R:85 A:95 D:95 |
| 4 | Confident | Per-skill `helpers:` frontmatter list is the opt-in mechanism | Discussed — user agreed directionally. Alternative (inline reference in skill body) works but is less discoverable and harder to audit/enforce. Frontmatter is conventional for skill metadata. | S:80 R:70 A:85 D:75 |
| 5 | Confident | Compress `_cli-fab` to ≤300 lines | User agreed; 773 lines is disproportionate to use. Exact target line count is approximate — may land slightly over if canonical-reference content demands it. | S:75 R:60 A:80 D:70 |
| 6 | Confident | Inline top-5 common `fab` commands into `_preamble` | Discussed. Choice of which 5 (`preflight`, `score`, `log`, `resolve`, `status`) derived from grep across `src/kit/skills/` for `fab ` calls; these are the most frequent. | S:80 R:75 A:85 D:80 |
| 7 | Certain | Change type is `refactor` | Deterministic per fab-new Step 6 keyword rule: intake contains "refactor"/"restructure"/"consolidate" — taxonomy match is unambiguous. | S:95 R:95 A:95 D:95 |
| 8 | Confident | Allowed `helpers:` values are the 4 listed (`_generation`, `_review`, `_cli-fab`, `_cli-external`) | Full cross-skill audit shows these are the only four helpers referenced by any skill. Easy to extend later if a new helper emerges. | S:85 R:85 A:80 D:80 |
| 9 | Confident | Backlog item 84bh is fully subsumed | Backlog item's root concern is unreliable 2nd-layer loading (`"other agents don't read from _ folders"`); this change eliminates the 2nd layer entirely for 15/24 skills and makes the rest explicit via frontmatter. Naming convention is orthogonal and can be a separate follow-up if desired. | S:80 R:85 A:80 D:75 |
| 10 | Confident | No binary-level enforcement of `helpers:` values needed | Convention-only keeps change scope in markdown layer. `fab sync` can be extended later if enforcement becomes valuable. Pure markdown change has highest reversibility. | S:80 R:90 A:75 D:80 |
| 11 | Certain | Per-skill SPEC files: `SPEC-preamble.md` exists (no underscore); `SPEC-_cli-rk.md` exists (will be deleted alongside `_cli-rk`). No `SPEC-_naming.md`, no `SPEC-_cli-fab.md`. | Verified via `ls docs/specs/skills/`. Spec updates target: modify `SPEC-preamble.md`; delete `SPEC-_cli-rk.md`; create `SPEC-_cli-fab.md` if helper file conventions warrant. | S:95 R:85 A:95 D:90 |

11 assumptions (5 certain, 6 confident, 0 tentative, 0 unresolved).
