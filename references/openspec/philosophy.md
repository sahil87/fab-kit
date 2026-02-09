# OpenSpec Philosophy

## Core Principles

OpenSpec is built on four foundational principles that distinguish it from traditional specification systems:

### 1. Fluid not Rigid

**Traditional:** Phase gates lock you into planning → implementation → done
**OpenSpec:** Work on artifacts in any order that makes sense

There are no gates or checkpoints. Dependencies guide what's *possible*, but humans decide what's *sensible*. You can:
- Start with design if the technical approach is the unknown
- Jump straight to tasks if requirements are crystal clear
- Go back and refine the proposal after learning from implementation

### 2. Iterative not Waterfall

**Traditional:** Get it right the first time
**OpenSpec:** Learn as you build, refine as you go

Requirements change. Understanding deepens during implementation. OpenSpec embraces this reality:
- Artifacts can be updated at any point
- Delta specs track changes explicitly
- Archive merges capture the final state

The system assumes you'll discover things along the way and provides mechanisms to capture that learning.

### 3. Easy not Complex

**Traditional:** Heavyweight frameworks with extensive ceremony
**OpenSpec:** Lightweight setup, minimal ceremony

Initialize in seconds, start working immediately:
```bash
openspec init       # That's it
```

Customize only when needed. Sensible defaults cover most cases. Compare to frameworks that require:
- Configuration files
- Project structure changes
- Team training
- Tool integration setup

### 4. Brownfield-First

**Traditional:** Greenfield-focused, changes as afterthoughts
**OpenSpec:** Changes to existing systems are first-class

Most software work modifies existing systems. OpenSpec's delta-based approach makes this explicit:

```markdown
## ADDED Requirements
New functionality being introduced

## MODIFIED Requirements
Existing behavior being changed

## REMOVED Requirements
Functionality being deprecated
```

This isn't just formatting - it's a fundamental design choice that affects parsing, validation, and merge operations.

## The Spec-Driven Development Philosophy

### Power Inversion

**Traditional:** Code is king, specs are scaffolding
**SDD:** Specifications are primary, code serves specifications

Specifications aren't just documentation - they're the authoritative description of system behavior. Code is an implementation detail that can be regenerated from specs.

### Specifications as Lingua Franca

Specs serve as the common language between:
- Humans defining requirements
- AI assistants implementing features
- Validation systems checking correctness
- Documentation systems explaining behavior

### Executable Specifications

Specs must be precise, complete, and unambiguous enough to:
- Generate working implementations
- Validate existing code
- Detect regressions
- Guide refactoring

This requires structured formats (Requirements + Scenarios) with formal keywords (RFC 2119).

### Continuous Refinement

Consistency validation happens continuously, not as a one-time gate:
- Each artifact creation triggers validation
- Archive operations verify specs merge correctly
- Implementation can feed back to spec refinement

## Design Values

### Human-AI Collaboration

OpenSpec assumes a human-AI partnership:
- Humans make strategic decisions (scope, approach, priorities)
- AI handles tactical execution (implementation, validation)
- Specs bridge the communication gap

Neither side works alone. The framework facilitates their collaboration.

### Explicit Over Implicit

Everything is explicit:
- Changes are named and scoped
- Dependencies are declared in schemas
- Progress is tracked via artifact existence
- Delta changes are marked as ADDED/MODIFIED/REMOVED

No implicit state. No magic. What you see is what exists.

### Convention Over Configuration

Defaults work for most cases:
- Default schema: `spec-driven` (proposal → specs → design → tasks)
- Default directory: `openspec/`
- Default tool detection: automatic

Override only when necessary.

### Tool Agnosticism

OpenSpec doesn't favor any AI tool. The same commands and workflows work with:
- Claude Code
- Cursor
- GitHub Copilot
- Windsurf
- And 18+ others

Tool-specific adapters handle formatting differences.

## Anti-Patterns OpenSpec Avoids

### Spec Rot
**Problem:** Specs become stale immediately after writing
**Solution:** Specs are living documents updated through changes

### AI Hallucination
**Problem:** AI makes assumptions without understanding requirements
**Solution:** Explicit specs provide authoritative requirements

### Parallel Conflicts
**Problem:** Multiple changes step on each other
**Solution:** Each change is isolated in its own directory

### Lost Context
**Problem:** Why was this decision made?
**Solution:** Archived changes preserve proposal, design, and discussion

### Merge Hell
**Problem:** Combining text changes is error-prone
**Solution:** Semantic ADDED/MODIFIED/REMOVED instead of text diffs

## The OPSX Workflow Philosophy

OPSX commands embody these principles:

| Command | Philosophy |
|---------|------------|
| `/opsx:explore` | Thinking before doing |
| `/opsx:new` | Explicit scope boundaries |
| `/opsx:continue` | Incremental progress |
| `/opsx:ff` | Skip ceremony when appropriate |
| `/opsx:apply` | Specs guide implementation |
| `/opsx:verify` | Continuous validation |
| `/opsx:archive` | Preserve context |

The workflow is:
1. **Defined** - Clear commands with specific purposes
2. **Flexible** - Use what makes sense, skip what doesn't
3. **Traceable** - Every action has visible results
4. **Reversible** - Nothing is permanent until archived
