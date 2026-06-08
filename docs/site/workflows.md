# Workflows

This guide is a task-oriented walkthrough of how to drive a change through Fab Kit's pipeline —
from idea to merged PR — plus the conceptual "why" behind the assembly-line model. If you have not
installed Fab Kit yet, start with the [Install guide](./install.md).

Fab Kit skills are slash commands you type into an AI agent's chat (`/fab-*` in Claude Code,
`$fab-*` in Codex), not the terminal. Open a session in your project directory first:

- **Claude Code:** run `claude` in the terminal
- **Codex:** run `codex` in the terminal
- **Cursor / Windsurf:** open the project and use the chat panel

## The mental model

Three ideas make the workflow worth the structure:

- **An assembly line, not a queue.** Without Fab you describe a task, wait while the AI works,
  review, and repeat — more waiting than working. With Fab you batch self-contained changes (each
  in its own folder and git worktree) and create the next batch while the AI executes the current
  one. Both you and the agent stay busy.
- **Shared memory that compounds.** Every completed change hydrates its learnings into
  `docs/memory/` — a domain-organized knowledge base committed to git. Future changes load those
  files as context, so the agent starts each change with real knowledge of your system instead of
  guessing. Team knowledge, not per-session notes.
- **A confidence gate, not blind autonomy.** Before any code is written, the SRAD framework scores
  how safe each planning decision is to assume (Signal · Reversibility · Agent competence ·
  Disambiguation). Grades aggregate into a confidence score that gates the fast-forward commands:
  if ambiguity is too high, the pipeline refuses to run and tells you what to clarify. For the full
  framework, see the [SRAD spec](https://github.com/sahil87/fab-kit/blob/main/docs/specs/srad.md).

## Driving a single change, stage by stage

A change moves through six stages: intake → apply → review → hydrate → ship → review-PR. The
manual, one-stage-at-a-time path looks like this — run each in your agent and wait for it to finish
before the next:

```
# Creation — creates the change folder, writes intake.md, activates the change, creates the git branch
/fab-new Add a loading spinner to the submit button

# Apply — generates plan.md (requirements + tasks + acceptance) and implements the code, checking off tasks as it goes
/fab-continue
# Review — a sub-agent validates the implementation against the plan's requirements + your constitution
/fab-continue
# Hydrate — saves learnings into docs/memory/
/fab-continue

# Ship — commit, push, and open a GitHub PR
/git-pr
# Review-PR — triage and fix PR review comments from humans or bots
/git-pr-review

# Archive — move the completed change folder out of active changes
/fab-archive
```

`/fab-continue` always advances exactly one stage from wherever the change currently is, which is
why it appears three times above (apply, then review, then hydrate). At any point, run
`/fab-status` to see where you are — current change, branch, stage, plan progress, and the
suggested next command. Every stage produces a persistent artifact, so you can interrupt anything
and resume later by re-running the same command; all pipeline skills are idempotent.

### The apply ⇄ review loop

Review runs as a **sub-agent in a fresh context** and returns prioritized findings (must-fix,
should-fix, nice-to-have). The applying agent triages them and loops back to the right stage:

| Review finds | Priority | Loops back to | What happens |
|--------------|----------|---------------|--------------|
| Requirement mismatch, failing tests | Must-fix | → apply | Unchecks the failed tasks in `plan.md` and re-runs them |
| Missing or wrong tasks | Must-fix | → apply | Regenerates `plan.md` and re-applies |
| Requirements were wrong | Must-fix | → apply | Updates `plan.md`'s requirements, regenerates tasks |
| Code-quality issue | Should-fix | → apply | Addressed when clear and low-effort |
| Style suggestion | Nice-to-have | — | May be skipped |

## Fast-forwarding: ff vs fff vs proceed

When you do not want to step through stages by hand, the fast-forward commands chain them together
behind the single intake confidence gate, auto-looping between apply and review (each re-review
spawns a fresh sub-agent, up to 3 cycles) before escalating to you:

| Command | Covers | Use when |
|---------|--------|----------|
| `/fab-ff` | apply → review → hydrate | You want the change implemented and reviewed, but will ship the PR yourself. Falls back to interactive rework after exhausting auto-retries. |
| `/fab-fff` | apply → review → hydrate → ship → review-PR | You want the full path through a raised PR and its review, hands-off. |
| `/fab-proceed` | detects state, runs any setup steps (new / switch / branch), then delegates to `/fab-fff` | You are not sure what state the change is in and want one command to do the right thing. |

All three are gated by the intake confidence score: low ambiguity runs unattended; high ambiguity
stops and tells you what to clarify (resolve it with `/fab-clarify`, which refines the current
artifact without advancing). A typical `/fab-fff` run uses a few agent turns per stage, with the
review sub-agent in its own context.

## Going parallel

The payoff of self-contained change folders is parallelism. While the AI works one change, start
another in an isolated [git worktree](https://git-scm.com/docs/git-worktree):

```bash
# In your terminal:
wt create                # creates an isolated worktree with a random name
```

```
# In a new agent session inside that worktree:
/fab-new Add an error toast for failed submissions
```

Because each change is its own folder with its own intake, plan, and status — and runs in its own
worktree — parallel sessions never step on each other. `/fab-new` auto-activates the change, so you
can start working immediately. Use `/fab-draft` to queue a change without switching to it. For the
batch scripts and the full numbers behind parallel development, see
[The Assembly Line](https://github.com/sahil87/fab-kit/blob/main/docs/specs/assembly-line.md).

## Where to go next

- New to the terminology? See the
  [Glossary](https://github.com/sahil87/fab-kit/blob/main/docs/specs/glossary.md).
- Want every skill's detailed behavior? See the
  [full command reference](https://github.com/sahil87/fab-kit/blob/main/docs/specs/skills.md).
- Setting up or upgrading? See the [Install guide](./install.md).
