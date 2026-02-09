# OpenSpec Overview

## What is OpenSpec?

OpenSpec is an AI-native, spec-driven development system that bridges the gap between AI coding assistants and structured software development. It's a lightweight, fluent framework that helps teams agree on **what** to build before any code is written.

## Core Problem It Solves

Traditional development often suffers from:
- AI assistants making assumptions without understanding requirements
- Specification documents that become stale immediately
- No structured way to track what's changing vs. what exists
- Difficulty coordinating between human decisions and AI implementation

OpenSpec addresses this through **delta-based specifications** - explicit tracking of ADDED, MODIFIED, and REMOVED requirements relative to existing system behavior.

## Two Components

### 1. CLI Tool (`openspec`)

```bash
openspec init                    # Initialize in project
openspec new change add-auth     # Start new change
openspec list                    # Show active changes
openspec validate                # Validate specs
openspec archive                 # Complete and archive change
```

### 2. AI Workflow Framework

Provides `/opsx:*` slash commands that AI assistants can execute:

| Command | Purpose |
|---------|---------|
| `/opsx:new` | Start a new change |
| `/opsx:continue` | Create next artifact |
| `/opsx:ff` | Fast-forward all planning |
| `/opsx:apply` | Implement from specs |
| `/opsx:verify` | Validate completion |
| `/opsx:archive` | Complete and finalize |
| `/opsx:explore` | Investigation mode (no implementation) |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        OpenSpec                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   CLI       │    │   Core      │    │   Adapters  │     │
│  │   Layer     │ -> │   Logic     │ -> │   (22+      │     │
│  │             │    │             │    │    tools)   │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
│        │                   │                   │            │
│        v                   v                   v            │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │ Commander.js│    │ Artifact    │    │ Claude      │     │
│  │ Commands    │    │ Graph       │    │ Cursor      │     │
│  │             │    │ Schema      │    │ Windsurf    │     │
│  │             │    │ Parsers     │    │ Copilot     │     │
│  │             │    │ Validation  │    │ ...         │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Key Concepts

### Specs (`openspec/specs/`)
Source of truth describing current system behavior. Organized by domain.

### Changes (`openspec/changes/`)
Proposed modifications. Each change = one folder with complete context:
- `proposal.md` - Intent and scope
- `specs/*.md` - Delta specs (what's changing)
- `design.md` - Technical approach
- `tasks.md` - Implementation checklist

### Artifacts
Structured documents created during a change workflow. Default schema:
```
proposal → specs → design → tasks → implement
```

### Delta Specs
Track changes explicitly:
- `## ADDED Requirements` - New functionality
- `## MODIFIED Requirements` - Changed behavior
- `## REMOVED Requirements` - Deprecated functionality

## Supported AI Tools

OpenSpec generates tool-specific skill files for 22+ AI assistants:

- Claude Code (`.claude/skills/`)
- Cursor (`.cursor/rules/`)
- Windsurf, Cline, Continue
- GitHub Copilot, Amazon Q
- Gemini CLI, Qwen Code
- And 13+ more

## Directory Structure

```
project/
├── openspec/
│   ├── config.yaml           # Project configuration
│   ├── specs/                # Source of truth
│   │   ├── auth/
│   │   ├── payments/
│   │   └── ...
│   └── changes/
│       ├── add-2fa/          # Active change
│       │   ├── .openspec.yaml
│       │   ├── proposal.md
│       │   ├── specs/
│       │   ├── design.md
│       │   └── tasks.md
│       └── archive/          # Completed changes
├── .claude/                  # Claude Code skills
│   └── skills/
└── .cursor/                  # Cursor skills
    └── rules/
```

## Data Flow

```
1. INIT
   openspec init → detect tools → generate skills → create structure

2. NEW CHANGE
   /opsx:new → create change dir → write .openspec.yaml

3. ARTIFACT CREATION
   /opsx:continue → load schema → generate instructions → AI creates artifact

4. IMPLEMENTATION
   /opsx:apply → read tasks.md → implement → check off tasks

5. COMPLETION
   /opsx:archive → validate → merge deltas → move to archive/
```

## Statistics

- **Lines of Code:** ~20,000 TypeScript
- **Supported AI Tools:** 22+
- **CLI Commands:** 30+ subcommands
- **Documentation Files:** 11 guides
- **Test Coverage:** Unit, integration, E2E
