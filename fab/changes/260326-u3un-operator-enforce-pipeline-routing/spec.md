# Spec: Operator Enforce Pipeline Routing

**Change**: 260326-u3un-operator-enforce-pipeline-routing
**Created**: 2026-03-26
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md` (modify)

## Operator: Pipeline-First Routing Principle

### Requirement: Pipeline-First Routing Principle in §1

The operator7 skill §1 (Principles) SHALL include a new principle — **"Pipeline-first routing"** — stating that the operator MUST route all new work through the fab pipeline (`/fab-new` then `/fab-fff`, `/fab-ff`, or `/fab-continue`), and MUST NOT dispatch raw inline implementation instructions to agent panes.

The principle SHALL explicitly list the exemption for operational maintenance commands (merge PR, archive, delete worktree, rebase, `/git-branch`, `/fab-switch`) which are coordination-level actions, not pipeline work.

#### Scenario: Operator routes new backlog item
- **GIVEN** a user asks the operator to work on a backlog item
- **WHEN** the operator processes the request
- **THEN** the operator spawns an agent with `/fab-new <id>` (or the structured flow from §6)
- **AND** the operator does NOT send raw implementation instructions like "fix the bug in auth.ts"

#### Scenario: Operator routes raw text request
- **GIVEN** a user says "fix login after password reset"
- **WHEN** the operator processes the request
- **THEN** the operator creates a backlog entry via `idea add` and proceeds through the structured flow (which invokes `/fab-new`)
- **AND** the operator does NOT send "fix the login code by changing the reset handler" directly to an agent

#### Scenario: Operational maintenance remains direct
- **GIVEN** a monitored change has a merged PR
- **WHEN** the operator needs to archive the change
- **THEN** the operator executes `/fab-archive` directly (coordination-level action)
- **AND** this does NOT violate the pipeline-first routing principle

### Requirement: Reinforcing Note in §6 "Working a Change"

The §6 "Working a Change" subsection SHALL include a highlighted note at the top reinforcing that all three work paths (backlog/Linear, raw text, existing change) MUST go through the pipeline. The note SHALL explicitly prohibit sending raw implementation instructions or `/fab-continue` without a prior `/fab-new` for new work.

#### Scenario: §6 note visible before work paths
- **GIVEN** an operator reads §6 "Working a Change"
- **WHEN** they encounter the three work path descriptions
- **THEN** a note appears before the paths stating the pipeline-first routing rule
- **AND** the note references the §1 principle

## Non-Goals

- Creating a new `SPEC-fab-operator7.md` file — no existing spec exists for operators 6 or 7; this change doesn't warrant creating one
- Modifying operator6 — the principle applies to operator7 only (operator6 is superseded)
- Adding enforcement tooling — this is guidance for the LLM operator, not a programmatic gate

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add "Pipeline-first routing" to §1 Principles | §1 is the authoritative section for operator behavioral invariants — confirmed by reading the file | S:85 R:90 A:90 D:90 |
| 2 | Confident | Add reinforcing note at top of §6 "Working a Change" | Dual placement is low-cost and prevents drift when operators read only §6 for mechanics. Could be §1-only but belt-and-suspenders preferred | S:75 R:90 A:80 D:70 |
| 3 | Certain | Exempt operational maintenance commands | §1 "Coordinate, don't execute" already carves out maintenance — this aligns with the existing exemption | S:90 R:95 A:90 D:90 |
| 4 | Certain | No SPEC-fab-operator7.md needed | No existing spec for operators 6 or 7. Constitution says update corresponding specs — none exists to update | S:85 R:95 A:85 D:90 |
| 5 | Confident | Use blockquote format for §6 note | Blockquotes (`>`) are the established pattern in fab skills for callouts and notes — matches existing style | S:70 R:95 A:80 D:65 |
| 6 | Certain | Include `/fab-continue` prohibition for new work | `/fab-continue` advances an existing change — using it without `/fab-new` skips intake generation entirely | S:85 R:85 A:90 D:85 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
