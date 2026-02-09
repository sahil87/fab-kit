# Spec-Kit Scripts

## Script Infrastructure Overview

Spec-Kit provides automation scripts in both **Bash** (POSIX shell) and **PowerShell** to support cross-platform development. The CLI auto-selects based on OS unless explicitly specified.

## Script Locations

```
.specify/scripts/
├── bash/
│   ├── common.sh                # Shared utility functions
│   ├── create-new-feature.sh    # Feature branch creation
│   ├── setup-plan.sh            # Plan directory setup
│   ├── check-prerequisites.sh   # Validation and path detection
│   └── update-agent-context.sh  # Agent file updates
└── powershell/
    ├── common.ps1
    ├── create-new-feature.ps1
    ├── setup-plan.ps1
    ├── check-prerequisites.ps1
    └── update-agent-context.ps1
```

---

## Script Summary

| Script | Purpose | Used By |
|--------|---------|---------|
| `create-new-feature` | Create feature branch and spec directory | `/speckit.specify` |
| `setup-plan` | Initialize plan directory and copy templates | `/speckit.plan` |
| `check-prerequisites` | Validate paths and detect available docs | `/speckit.tasks`, `/speckit.implement`, `/speckit.clarify`, `/speckit.analyze` |
| `update-agent-context` | Update agent-specific context files | `/speckit.plan` |
| `common` | Shared utility functions | All scripts |

---

## `create-new-feature.sh`

**Purpose**: Create a new feature branch and initialize spec directory structure.

**Usage**:
```bash
./create-new-feature.sh [--json] [--short-name <name>] [--number N] "<feature_description>"
```

**Options**:
| Option | Description |
|--------|-------------|
| `--json` | Output in JSON format |
| `--short-name <name>` | Custom branch name suffix (2-4 words) |
| `--number N` | Override auto-detected branch number |
| `--help, -h` | Show help message |

**Execution Flow**:

1. **Parse arguments** - Extract JSON mode, short name, number, description
2. **Find repo root** - Search for `.git` or `.specify` directory
3. **Determine branch number**:
   - Fetch all remote branches
   - Get highest number from branches (local + remote)
   - Get highest number from specs directories
   - Use max + 1
4. **Generate branch name**:
   - Filter stop words (a, the, to, for, etc.)
   - Keep words >= 3 chars or uppercase acronyms
   - Use first 3-4 meaningful words
   - Format: `###-short-name`
5. **Create branch** (if git repo)
6. **Create directory structure**:
   ```
   specs/###-feature-name/
   └── spec.md (copied from template)
   ```
7. **Output** JSON or plain text with `BRANCH_NAME`, `SPEC_FILE`, `FEATURE_NUM`

**Branch Name Generation Examples**:
```
"I want to add user authentication" → "001-user-authentication"
"Implement OAuth2 integration for API" → "002-oauth2-integration-api"
"Create a dashboard for analytics" → "003-dashboard-analytics"
```

**Stop Words Filtered**:
```
i, a, an, the, to, for, of, in, on, at, by, with, from, is, are,
was, were, be, been, being, have, has, had, do, does, did, will,
would, should, could, can, may, might, must, shall, this, that,
these, those, my, your, our, their, want, need, add, get, set
```

---

## `setup-plan.sh`

**Purpose**: Initialize the plan directory and copy templates for the implementation plan phase.

**Usage**:
```bash
./setup-plan.sh [--json]
```

**Execution Flow**:

1. **Detect feature** from git branch or `SPECIFY_FEATURE` env var
2. **Create directories**:
   ```
   specs/###-feature-name/
   ├── contracts/
   └── checklists/
   ```
3. **Copy templates**:
   - `plan-template.md` → `plan.md`
4. **Output** JSON with:
   - `FEATURE_SPEC` - path to spec.md
   - `IMPL_PLAN` - path to plan.md
   - `SPECS_DIR` - specs directory
   - `BRANCH` - branch name

---

## `check-prerequisites.sh`

**Purpose**: Validate that required files exist and return available document paths.

**Usage**:
```bash
./check-prerequisites.sh [--json] [--paths-only] [--require-tasks] [--include-tasks]
```

**Options**:
| Option | Description |
|--------|-------------|
| `--json` | Output in JSON format |
| `--paths-only` | Minimal output (only paths, no validation) |
| `--require-tasks` | Fail if tasks.md doesn't exist |
| `--include-tasks` | Include tasks content in output |

**Detected Documents**:
| Document | Path |
|----------|------|
| `spec.md` | `specs/[feature]/spec.md` |
| `plan.md` | `specs/[feature]/plan.md` |
| `tasks.md` | `specs/[feature]/tasks.md` |
| `data-model.md` | `specs/[feature]/data-model.md` |
| `research.md` | `specs/[feature]/research.md` |
| `quickstart.md` | `specs/[feature]/quickstart.md` |
| `contracts/` | `specs/[feature]/contracts/` |
| `checklists/` | `specs/[feature]/checklists/` |

**Output JSON Structure**:
```json
{
  "FEATURE_DIR": "/path/to/specs/001-feature",
  "FEATURE_SPEC": "/path/to/specs/001-feature/spec.md",
  "IMPL_PLAN": "/path/to/specs/001-feature/plan.md",
  "TASKS": "/path/to/specs/001-feature/tasks.md",
  "AVAILABLE_DOCS": ["spec.md", "plan.md", "data-model.md"]
}
```

---

## `update-agent-context.sh`

**Purpose**: Update the agent-specific context file with technology information from the current plan.

**Usage**:
```bash
./update-agent-context.sh [agent_type]
```

**Supported Agents**:
| Agent Type | Context File |
|------------|--------------|
| `claude` | `.claude/CLAUDE.md` or project `CLAUDE.md` |
| `gemini` | `.gemini/context.md` |
| `copilot` | `.github/copilot-instructions.md` |
| `cursor-agent` | `.cursor/rules/specify-rules.md` |
| `windsurf` | `.windsurf/rules/specify-rules.md` |
| (default) | Auto-detect based on existing files |

**Execution Flow**:

1. **Detect agent** from argument or existing files
2. **Read plan.md** to extract new technologies
3. **Update context file**:
   - Preserve content outside markers
   - Update section between markers
4. **Markers used**:
   ```markdown
   <!-- SPECIFY_CONTEXT_START -->
   [Generated content]
   <!-- SPECIFY_CONTEXT_END -->
   ```

---

## `common.sh` / `common.ps1`

**Purpose**: Shared utility functions used by all scripts.

**Key Functions**:

| Function | Description |
|----------|-------------|
| `find_repo_root` | Find repository root by searching for markers |
| `get_feature_from_branch` | Extract feature name from git branch |
| `get_highest_from_specs` | Find highest feature number in specs directory |
| `get_highest_from_branches` | Find highest feature number in git branches |
| `clean_branch_name` | Sanitize branch name (lowercase, hyphens) |
| `output_json` | Format output as JSON |

---

## Non-Git Repository Support

Scripts handle repositories initialized with `--no-git`:

1. **Repo root detection** falls back to searching for `.specify` directory
2. **Feature detection** uses `SPECIFY_FEATURE` environment variable
3. **Branch operations** are skipped with warning messages

**Setting Feature Manually**:
```bash
# Before running /speckit.plan or follow-up commands
export SPECIFY_FEATURE="001-my-feature"
```

---

## Error Handling

Scripts follow these conventions:

1. **Exit on error** - `set -e` in bash
2. **Clear error messages** - Printed to stderr
3. **JSON error output** - Structured errors when `--json` is used
4. **Graceful degradation** - Non-git repos continue with warnings

---

## Script Invocation from Commands

Commands reference scripts via `{SCRIPT}` placeholder:

```yaml
# In command frontmatter
scripts:
  sh: scripts/bash/create-new-feature.sh --json "{ARGS}"
  ps: scripts/powershell/create-new-feature.ps1 -Json "{ARGS}"
```

The AI agent:
1. Detects the current shell environment
2. Substitutes `{SCRIPT}` with appropriate path
3. Substitutes `{ARGS}` with user arguments
4. Executes the script and parses output

---

## GitHub Branch Length Validation

`create-new-feature.sh` enforces GitHub's 244-byte branch name limit:

```bash
MAX_BRANCH_LENGTH=244
if [ ${#BRANCH_NAME} -gt $MAX_BRANCH_LENGTH ]; then
    # Truncate suffix at word boundary
    # Remove trailing hyphen
    # Warn user about truncation
fi
```
