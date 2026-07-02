# Plan: _cli-external.md Volatility Split + hop

**Change**: 260616-os8z-cli-external-volatility-split
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md § What Changes (parts 1–7) + § Absent-binary discipline.
     This is a docs/skill-content restructure of one helper file
     (src/kit/skills/_cli-external.md). No Go code, no SPEC mirror (uliv exclusion). -->

### Reference Helper: Volatility Split & Reference Model

#### R1: Frontmatter description names hop and the gist/help-dump model
The `description:` frontmatter of `src/kit/skills/_cli-external.md` MUST list `hop` (multi-repo navigator) alongside the existing tools AND state the two-tier model (hand-authored gist of operator-critical commands/flags + integration semantics; the exhaustive surface delegated to each tool's `help-dump` at use-time).

- **GIVEN** the helper file's YAML frontmatter
- **WHEN** an agent reads the `description:` line
- **THEN** it names `hop` in the tool list
- **AND** it states the gist + `help-dump`-delegation model
- **AND** it preserves "Loaded by operator skills only."

#### R2: A Reference Model preamble section establishes the volatility split and names the help-dump contract
The file MUST carry a `## Reference Model` section near the top (after the intro blockquote) that (a) states the two-tier model (gist vs. exhaustive), (b) names the hidden-but-stable `help-dump` JSON contract explicitly, and (c) instructs the agent to run `<tool> help-dump` (or `<tool> <cmd> --help`) at use-time as authoritative for anything not in the gist.

- **GIVEN** the restructured file
- **WHEN** an agent reads the `## Reference Model` section
- **THEN** it learns the file documents a curated gist, not an exhaustive reference
- **AND** it learns each owned tool (and `fab`) exposes a hidden `help-dump` emitting JSON with `tool`, `version`, `captured_at`, `schema_version` (currently `1`), and a recursive `root` → `commands[]` tree (each node: `name`, `path`, `short`, `usage`, `text`)
- **AND** it is instructed to treat `help-dump` output, not this file, as authoritative for the exhaustive surface
- **AND** the `help-dump` instruction is scoped to the four owned binaries (`wt`/`idea`/`rk`/`hop`), explicitly NOT tmux (no `help-dump`) or `/loop` (a skill, not a binary)

#### R3: The two-install-class absent-binary discipline is encoded inseparably from the delegation
The `## Reference Model` section MUST encode the two install classes and the asymmetric fail-silent rule: `wt`/`idea` are assumed-present (brew `depends_on` of fab-kit) and MAY be invoked bare; `rk`/`hop` are genuinely-optional (separate formulas) and every invocation including `help-dump` MUST be `command -v`-gated and fail silently (never surface `command not found`). The gate MUST NOT be generalized to `wt`/`idea`.

- **GIVEN** the `## Reference Model` section
- **WHEN** an agent prepares to run a `help-dump` (or any invocation)
- **THEN** for `rk`/`hop` it uses the gated form `command -v <tool> >/dev/null 2>&1 && <tool> help-dump` and skips silently when absent
- **AND** for `wt`/`idea` the bare form is acceptable (guaranteed present)
- **AND** the section shows a concrete example contrasting the gated (rk/hop) and bare (wt/idea) forms

#### R4: The wt section is trimmed to the operator-used gist while preserving integration semantics verbatim
The `wt` section MUST keep the operator-driven commands (`list`, `list --path`, `create`, `delete`), the integration-critical `wt create` flags (`--non-interactive`, `--worktree-name`, `--reuse`, `--base`, positional `[branch]`), and the two worked examples (known-change / autopilot-respawn). It MUST drop `init`, `open`, `shell-init`, `update` from the inlined tables (reachable via `wt help-dump`). It MUST preserve verbatim the Repo-targeted spawning note and the entire Operator Spawning Rules subsection.

- **GIVEN** the restructured `wt` section
- **WHEN** an agent reads it
- **THEN** the Commands table lists only `list`, `list --path`, `create`, `delete`
- **AND** the `wt create` Flags table lists `--non-interactive`, `--worktree-name`, `--reuse`, `--base`, `[branch]`
- **AND** the two worked examples remain
- **AND** the Repo-targeted spawning blockquote and the Operator Spawning Rules subsection are unchanged in substance
- **AND** `init`/`open`/`shell-init`/`update` do not appear as inlined table rows
- **AND** a one-line pointer notes the exhaustive `wt` surface is reachable via `wt help-dump`

#### R5: The idea section is trimmed to the operator-used verbs while preserving integration semantics
The `idea` section MUST keep the operator-used verbs (bare shorthand, `add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`) and MUST drop `fmt`, `prune`, `shell-init`, `update` from the inlined table. It MUST preserve the `--main` worktree-resolution behavior, the `--file`/`IDEAS_FILE` priority, the query-matching rule, and the backlog/output format blocks.

- **GIVEN** the restructured `idea` section
- **WHEN** an agent reads it
- **THEN** the subcommand table lists only bare, `add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`
- **AND** `fmt`/`prune`/`shell-init`/`update` do not appear as inlined rows
- **AND** the `--main`/`--file`/`IDEAS_FILE` persistent flags, the resolution prose, the query-matching rule, and the backlog/output format blocks remain
- **AND** a one-line pointer notes the exhaustive `idea` surface is reachable via `idea help-dump`

#### R6: The rk section framing is tightened with a help-dump pointer and an unchanged fail-silent contract
The `rk` section MUST keep `rk notify` (full body), `rk context` (server-URL discovery), iframe windows, the proxy pattern, and the Visual Display Recipe delegation. It MUST add a one-line pointer that the full `rk` surface (`daemon`, `doctor`, `serve`, `reaper`, `riff`, `init-conf`, `status`, `update`, …) is available via `rk help-dump`, itself subject to the existing `command -v rk` gate. The fail-silent contract and the `_preamble.md` cross-reference MUST NOT change.

- **GIVEN** the restructured `rk` section
- **WHEN** an agent reads it
- **THEN** `rk notify`, `rk context`, iframe windows, proxy, and Visual Display Recipe content are intact
- **AND** a one-line `rk help-dump` pointer for the full surface is present, gated on `command -v rk`
- **AND** the fail-silent contract sentence and the `_preamble.md § Run-Kit (rk) Reference` cross-reference are unchanged

#### R7: A new hop section documents the discovery front-end, fail-silent and gated
The file MUST add a `## hop (Multi-Repo Navigator)` section that (a) opens with the fail-silent note mirroring the `rk` section (genuinely-optional binary; every invocation including `help-dump` is `command -v hop`-gated; never surface `command not found`), (b) frames `hop` as the discovery front-end to the same repo/worktree space `wt` operates on (locates repos via `hop.yaml`), (c) documents the discovery gist (`hop ls`, `hop ls --trees` fanning out to `wt list --json`, `hop <name> where` and `hop <name>/<wt> where`), (d) explains why it matters (resolving a sibling repo's absolute main-worktree root for repo-targeted spawning), and (e) carries a delegation pointer to `hop help-dump` for the full surface.

- **GIVEN** the restructured file
- **WHEN** an agent reads the `## hop (Multi-Repo Navigator)` section
- **THEN** the first content establishes the `command -v hop` fail-silent discipline
- **AND** `hop` is framed as the discovery front-end to the wt-managed repo/worktree space
- **AND** `hop ls`, `hop ls --trees` (fans out to `wt list --json`), and `hop <name> where` / `hop <name>/<wt> where` are documented
- **AND** the section connects `hop where` to resolving sibling-repo roots for the wt repo-targeted spawning note
- **AND** a `hop help-dump` delegation pointer covers the full surface (`add`, `clone`, `rm`, `config`, `shell-init`, `update`, batch verbs, `--all`)

#### R8: tmux and /loop sections are substantively unchanged and excluded from the help-dump instruction
The `tmux` and `/loop` sections MUST remain substantively unchanged. The `## Reference Model` `help-dump` instruction MUST make clear it scopes to the four owned binaries, not tmux or `/loop`.

- **GIVEN** the restructured file
- **WHEN** an agent reads the tmux and /loop sections
- **THEN** their content is unchanged in substance
- **AND** the `help-dump` delegation in `## Reference Model` does not claim to apply to tmux or `/loop`

### Non-Goals

- Editing the deployed `.claude/skills/_cli-external/SKILL.md` copy — it is regenerated by `fab sync`; only `src/kit/skills/_cli-external.md` is edited.
- Any Go code, CLI, or test changes — `help-dump` already ships on all binaries.
- Creating a SPEC mirror — `_cli-external.md` is excluded by the `uliv` policy.
- Editing `docs/memory/runtime/operator.md` — its tool-set enumeration update is a HYDRATE-stage concern, not apply.
- Rewriting the operator §6 spawn procedure to mandate `hop` — deferred (Open Questions; assumption 9).
- Extending `_preamble.md`'s rk rule to name `hop` — `hop`'s gate is stated self-contained in `_cli-external.md` (assumption 14).

### Design Decisions

1. **Volatility split (gist + runtime `help-dump` delegation)** over a conformance check — *Why*: deletes the drift class instead of detecting it; mirrors the existing `rk context` delegation pattern; fits Pure Prompt Play (Constitution I). *Rejected*: a `fab doctor --external` conformance check (user explicitly rejected as belt-and-suspenders), and keep-tables-plus-distrust-note (user chose the committed trim).
2. **Two install classes, asymmetric gating** — *Why*: `wt`/`idea` are brew `depends_on` of fab-kit (guaranteed on PATH); `rk`/`hop` are separate optional formulas. Gating all four would be over-cautious uniformity. *Rejected*: gate all four; gate none.
3. **`hop`'s fail-silent gate lives in `_cli-external.md`, not `_preamble.md`** — *Why*: `hop` is operator-only like the rest of this helper; `_preamble.md`'s rk gate is inline-carried by every skill for a universal-carry reason `hop` lacks. *Rejected*: extend `_preamble.md § Run-Kit (rk) Reference` to a generic external-tool rule.

## Tasks

### Phase 1: Setup

- [x] T001 Confirm canonical source vs. deployed copy and ground the gist against installed binaries — list `src/kit/skills/_cli-external.md` (flat source) vs. `.claude/skills/_cli-external/SKILL.md` (deployed dir, do NOT edit); run `wt help-dump`, `idea help-dump`, `rk help-dump`, `hop help-dump`, `hop --help` to capture real flag/usage/command names and the live `schema_version` <!-- R2 -->

### Phase 2: Core Implementation

- [x] T002 Update the `description:` frontmatter of `src/kit/skills/_cli-external.md` to add `hop (multi-repo navigator)` to the tool list and state the gist + `help-dump`-delegation model, preserving "Loaded by operator skills only." <!-- R1 -->
- [x] T003 Add a `## Reference Model` section after the intro blockquote: state the gist-vs-exhaustive two-tier model, name the hidden `help-dump` JSON contract (`tool`/`version`/`captured_at`/`schema_version:1`/recursive `root`→`commands[]` with `name`/`path`/`short`/`usage`/`text`), instruct run-`<tool> help-dump`-at-use-time, encode the two-install-class gate (rk/hop gated + fail-silent; wt/idea bare) with a contrasting example, and scope the instruction to the four owned binaries (not tmux/`/loop`) <!-- R2 R3 R8 -->
- [x] T004 Trim the `wt` section: reduce the Commands table to `list`/`list --path`/`create`/`delete`; keep the `wt create` Flags table (`--non-interactive`/`--worktree-name`/`--reuse`/`--base`/`[branch]`) and the two worked examples; add a one-line `wt help-dump` pointer for the dropped commands (`init`/`open`/`shell-init`/`update`); preserve the Repo-targeted spawning blockquote and the Operator Spawning Rules subsection verbatim <!-- R4 -->
- [x] T005 Trim the `idea` section: reduce the subcommand table to bare/`add`/`list`/`show`/`done`/`reopen`/`edit`/`rm`; add a one-line `idea help-dump` pointer for the dropped verbs (`fmt`/`prune`/`shell-init`/`update`); preserve the Persistent Flags table, `--main`/`--file`/`IDEAS_FILE` resolution prose, query-matching rule, and backlog/output format blocks <!-- R5 -->
- [x] T006 Tighten the `rk` section framing: add a one-line `rk help-dump` pointer (full surface, gated on `command -v rk`) without altering `rk notify`/`rk context`/iframe/proxy/Visual Display Recipe content, the fail-silent contract sentence, or the `_preamble.md` cross-reference <!-- R6 -->
- [x] T007 Add a new `## hop (Multi-Repo Navigator)` section (placed after the `idea` section / before `tmux`, mirroring discovery-tool grouping): open with the `command -v hop` fail-silent note mirroring `rk`; frame `hop` as the discovery front-end to the wt-managed repo/worktree space (via `hop.yaml`); document `hop ls`, `hop ls --trees` (fans out to `wt list --json`), `hop <name> where` / `hop <name>/<wt> where`; explain the sibling-repo-root resolution tie-in to the wt Repo-targeted spawning note; add a `hop help-dump` delegation pointer for the full surface <!-- R7 -->

### Phase 3: Integration & Edge Cases

- [x] T008 Verify tmux and `/loop` sections are substantively unchanged and that the `## Reference Model` instruction explicitly excludes them from the `help-dump` scope <!-- R8 -->

### Phase 4: Polish

- [x] T009 Final consistency pass: confirm all 7 restructure parts present, markdown well-formed (tables, code fences, blockquotes), table/blockquote idioms match the file's existing style, the two-install-class gate is encoded correctly (rk/hop gated, wt/idea bare, no over-generalization), and no edit touched the deployed copy, operator.md, Go code, or specs <!-- R2 R3 R4 R5 R6 R7 R8 -->

## Execution Order

- T001 precedes all edits (grounds the gist).
- T002–T007 edit distinct sections of the same file; execute sequentially (single-file edits) in document order to keep the file coherent: frontmatter (T002) → Reference Model (T003) → wt (T004) → idea (T005) → rk (T006) → hop (T007).
- T008 and T009 are verification, after all edits.

## Acceptance

### Functional Completeness

- [x] A-001 R1: The `description:` frontmatter names `hop (multi-repo navigator)`, states the gist + `help-dump`-delegation model, and preserves "Loaded by operator skills only."
- [x] A-002 R2: A `## Reference Model` section exists after the intro blockquote, states the two-tier model, names the `help-dump` JSON contract (tool/version/captured_at/schema_version:1/recursive root→commands tree with name/path/short/usage/text), and instructs run-`help-dump`-at-use-time as authoritative.
- [x] A-003 R3: The `## Reference Model` section encodes the two install classes — rk/hop gated + fail-silent (`command -v <tool> && <tool> help-dump`), wt/idea bare — with a contrasting example, and does not generalize the gate to wt/idea.
- [x] A-004 R4: The `wt` Commands table lists only list/list --path/create/delete; the create Flags table keeps --non-interactive/--worktree-name/--reuse/--base/[branch]; both worked examples remain; a `wt help-dump` pointer covers init/open/shell-init/update; the Repo-targeted spawning note and Operator Spawning Rules subsection are preserved verbatim.
- [x] A-005 R5: The `idea` subcommand table lists only bare/add/list/show/done/reopen/edit/rm; a `idea help-dump` pointer covers fmt/prune/shell-init/update; the Persistent Flags, --main/--file/IDEAS_FILE prose, query-matching rule, and backlog/output format blocks are preserved.
- [x] A-006 R6: The `rk` section retains notify/context/iframe/proxy/Visual Display Recipe content unchanged, adds a `command -v rk`-gated `rk help-dump` pointer, and leaves the fail-silent contract and `_preamble.md` cross-reference unchanged.
- [x] A-007 R7: A `## hop (Multi-Repo Navigator)` section exists, opens with the `command -v hop` fail-silent note mirroring rk, frames hop as the discovery front-end to the wt repo/worktree space, documents ls/ls --trees (fans out to wt list --json)/where, ties where to sibling-repo-root resolution, and carries a `hop help-dump` delegation pointer.

### Behavioral Correctness

- [x] A-008 R8: tmux and /loop sections are substantively unchanged and the `## Reference Model` `help-dump` instruction explicitly scopes to the four owned binaries, excluding tmux and /loop.

### Scenario Coverage

- [x] A-009 R3: An agent following the `## Reference Model` example runs the gated form for rk/hop and the bare form for wt/idea — verified by the example contrasting the two forms.
- [x] A-010 R2: The named `help-dump` contract matches the live binaries (schema_version 1; tool/version/captured_at present, captured_at may be empty) — grounded against the installed tools in T001.

### Edge Cases & Error Handling

- [x] A-011 R3 R7: A naked `help-dump` on a missing rk/hop binary is never produced — every rk/hop invocation in the file (including the help-dump pointers) is `command -v`-gated, so absence fails silently with no `command not found`.

### Code Quality

- [x] A-012 Pattern consistency: New sections (Reference Model, hop) follow the file's existing idioms — `##` section headers, markdown tables for command/flag references, blockquote notes for integration semantics, fenced code blocks for shell snippets; the hop fail-silent opener mirrors the rk section's wording.
- [x] A-013 No unnecessary duplication: The `help-dump` contract and gate are stated once in `## Reference Model` and referenced (not re-specified in full) by each tool section; the rk fail-silent rule is not duplicated from `_preamble.md` but cross-referenced.
- [x] A-014 Documentation accuracy: All command names, flags, usage strings, and the `help-dump` schema match the installed binaries verified in T001 (no invented flags; `wt --base`/`--reuse`/`--worktree-name`/`--non-interactive`, idea verbs, hop ls/--trees/where confirmed live).
- [x] A-015 Cross-references: The `_preamble.md § Run-Kit (rk) Reference` cross-reference, the `_cli-fab.md` references (spawn-command, pane commands), and the `wt list --json` ↔ `hop ls --trees` seam are accurate and intact.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- This is a `docs` change: "tests" = internal-consistency verification of the restructured markdown (all parts present, well-formed, gate correctly encoded). No Go test suite.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Edit only `src/kit/skills/_cli-external.md` (flat source), never the deployed `.claude/skills/_cli-external/SKILL.md` dir copy | Constitution V / context.md: src/kit is canonical, .claude/skills regenerated by `fab sync`; confirmed deployed copy is a SKILL.md under a dir | S:95 R:90 A:100 D:95 |
| 2 | Certain | No SPEC mirror, no Go code, no tests | `uliv` exclusion policy (verified in specs-index.md); `help-dump` already ships on all binaries; scope is prose | S:95 R:90 A:100 D:95 |
| 3 | Certain | Two install classes with asymmetric gating: gate+fail-silent rk/hop only; wt/idea bare | Verified: wt/idea are brew depends_on, rk/hop are separate optional formulas; intake assumptions 12–13 | S:95 R:85 A:95 D:90 |
| 4 | Confident | Place the new `## hop` section after `idea` and before `tmux` (discovery tools grouped before terminal/UI tooling) | Intake says "add a hop section" without fixing placement; grouping it with the other repo/worktree discovery tools (wt/idea) reads best; reversible | S:75 R:90 A:75 D:70 |
| 5 | Confident | Add a short per-tool `help-dump` pointer line to wt/idea/rk (not just rely on the global Reference Model instruction) | Intake parts 3/4/5 each call for delegating the dropped surface; an in-section pointer makes the delegation discoverable at the point of trimming; matches the file's per-section style | S:80 R:85 A:80 D:75 |
| 6 | Confident | Describe the `help-dump` schema with `captured_at` as a present-but-may-be-empty field | Verified live: idea/rk/hop emit captured_at (hop's empty), wt omits it; the intake contract lists captured_at — document it as part of the contract, noting it may be absent/empty | S:80 R:85 A:85 D:75 |
| 7 | Tentative | Do NOT touch `docs/memory/runtime/operator.md` in apply — its enumeration update is a hydrate-stage concern | Intake § Affected Memory + dispatch instructions assign operator.md to hydrate; assumption 9 leans defer; reversible | S:70 R:65 A:70 D:65 |
| 8 | Tentative | State `hop`'s fail-silent gate self-contained in `_cli-external.md`; do NOT extend `_preamble.md`'s rk rule | Intake assumption 14 / Open Questions lean this way; hop is operator-only like the rest of the helper; reversible follow-up if a universal-carry need emerges | S:70 R:65 A:70 D:65 |

8 assumptions (3 certain, 3 confident, 2 tentative).
