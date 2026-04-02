# fab-setup

## Summary

Bootstraps a new project or manages config/constitution/migrations. Creates `fab/project/` files, `docs/memory/`, `docs/specs/`, skill symlinks, and gitignore entries. Safe to re-run.

## Flow

```
User invokes /fab-setup [subcommand]
│
├─ Pre-flight: verify src/kit/ and VERSION exist
├─ Bash: fab log command "fab-setup"
│
├── No argument: Bootstrap ─────────────────────────────
│  │
│  ├─ Phase 0: Bash: fab doctor
│  │  └─ STOP if non-zero
│  │
│  ├─ Phase 1a: config.yaml
│  │  ├─ Read: README, package.json (project context)
│  │  ├─ Read: src/kit/scaffold/fab/project/config.yaml
│  │  ├─ (interactive: ask name, description, source_paths)
│  │  └─ Write: fab/project/config.yaml
│  │
│  ├─ Phase 1b: constitution.md
│  │  ├─ Read: src/kit/scaffold/fab/project/constitution.md
│  │  ├─ Read: project context (config, README, codebase)
│  │  ├─ (agent generates principles)
│  │  └─ Write: fab/project/constitution.md
│  │
│  ├─ Phase 1c-1e: Optional project files
│  │  └─ Write: context.md, code-quality.md, code-review.md (from scaffold)
│  │
│  ├─ Phase 1f-1g: docs directories
│  │  └─ Write: docs/memory/index.md, docs/specs/index.md (from scaffold)
│  │
│  ├─ Phase 1i: Changes directory + sync
│  │  └─ Bash: src/kit/scripts/fab-sync.sh
│  │     └─ (creates directories, symlinks, migration version)
│  │
│  └─ Phase 1k: .gitignore
│     └─ Edit: .gitignore (append .fab-status.yaml)
│
├── config: Config ──────────────────────────────────────
│  ├─ Read: fab/project/config.yaml
│  ├─ (interactive: menu → edit section)
│  └─ Edit: fab/project/config.yaml
│
├── constitution: Constitution ──────────────────────────
│  ├─ Read: fab/project/config.yaml, constitution.md
│  ├─ (interactive: amendment menu)
│  └─ Edit: fab/project/constitution.md
│
└── migrations: Migrations ─────────────────────────────
   ├─ Read: fab/.kit-migration-version, src/kit/VERSION
   ├─ Glob: src/kit/migrations/*.md
   ├─ For each applicable migration:
   │  ├─ Read: migration file
   │  ├─ (execute pre-checks, changes, verification)
   │  └─ Write: fab/.kit-migration-version
   └─ Write: fab/.kit-migration-version (finalize)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Scaffold templates, project files, migration files |
| Write | Project files, migration version |
| Edit | Config, constitution, gitignore |
| Bash | `fab doctor`, `fab-sync.sh`, `fab log command` |
| Glob | Discover migration files |

### Sub-agents

None.
