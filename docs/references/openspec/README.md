# OpenSpec Documentation

This directory contains an in-depth analysis of the OpenSpec framework - an AI-native, spec-driven development system.

## Documentation Index

| File | Description |
|------|-------------|
| [overview.md](overview.md) | High-level architecture and purpose |
| [philosophy.md](philosophy.md) | Core principles and design philosophy |
| [conventions.md](conventions.md) | Naming, formatting, and structural conventions |
| [cli-architecture.md](cli-architecture.md) | CLI commands and structure |
| [agent-integration.md](agent-integration.md) | AI agent support and skill system |
| [artifact-system.md](artifact-system.md) | Artifact types, workflow, and dependency graph |
| [schemas.md](schemas.md) | Workflow schema system and customization |
| [scripts.md](scripts.md) | Build, release, and utility scripts |
| [configuration.md](configuration.md) | Project and global configuration |
| [spec-format.md](spec-format.md) | Specification document format and delta specs |
| [tool-adapters.md](tool-adapters.md) | Multi-tool adapter system (Claude, Cursor, etc.) |

## Quick Reference

**Repository:** https://github.com/Fission-AI/OpenSpec
**Package:** `@fission-ai/openspec`
**Version:** 1.1.1
**License:** MIT

## Key Directories in OpenSpec

```
OpenSpec/
├── src/                    # TypeScript source (~20K lines)
│   ├── cli/               # CLI entry point
│   ├── commands/          # Command implementations
│   ├── core/              # Core logic (parsers, validation, etc.)
│   └── utils/             # Utilities
├── schemas/               # Workflow schema definitions
├── docs/                  # User documentation
├── test/                  # Test suite
└── openspec/              # Self-hosted specs (dogfooding)
```

## What is OpenSpec?

OpenSpec is two things:

1. **A CLI tool** (`openspec`) for managing specs and changes
2. **An AI workflow framework** providing `/opsx:*` slash commands for AI assistants

It bridges the gap between AI coding assistants and structured software development using a delta-based specification approach.
