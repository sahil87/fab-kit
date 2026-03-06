# fab-setup

## Summary

Bootstraps a new project or manages config/constitution/migrations. Creates `fab/project/` files, `docs/memory/`, `docs/specs/`, skill symlinks, and gitignore entries. Detects project language and applies templates. Safe to re-run.

## Flow

```
User invokes /fab-setup [subcommand]
│
├─ Pre-flight: verify fab/.kit/ and VERSION exist
├─ Bash: fab log command "fab-setup"
│
├── No argument: Bootstrap ─────────────────────────────
│  │
│  ├─ Phase 0: Bash: fab/.kit/scripts/fab-doctor.sh
│  │  └─ STOP if non-zero
│  │
│  ├─ Phase 1a: config.yaml
│  │  ├─ Read: README, package.json (project context)
│  │  ├─ Read: fab/.kit/scaffold/fab/project/config.yaml
│  │  ├─ (interactive: ask name, description, source_paths)
│  │  └─ Write: fab/project/config.yaml
│  │
│  ├─ Phase 1b: constitution.md
│  │  ├─ Read: fab/.kit/scaffold/fab/project/constitution.md
│  │  ├─ Read: project context (config, README, codebase)
│  │  ├─ (agent generates principles)
│  │  └─ Write: fab/project/constitution.md
│  │
│  ├─ Phase 1b-lang: Language Detection
│  │  ├─ Read: Cargo.toml / package.json / tsconfig.json / go.mod
│  │  ├─ Read: fab/.kit/templates/constitutions/{lang}.md
│  │  ├─ Read: fab/.kit/templates/configs/{lang}.yaml
│  │  └─ Edit: constitution.md + config.yaml (append/merge)
│  │
│  ├─ Phase 1b2-1b4: Optional project files
│  │  └─ Write: context.md, code-quality.md, code-review.md (from scaffold)
│  │
│  ├─ Phase 1c-1d: docs directories
│  │  └─ Write: docs/memory/index.md, docs/specs/index.md (from scaffold)
│  │
│  ├─ Phase 1f: Changes directory + sync
│  │  └─ Bash: fab/.kit/scripts/fab-sync.sh
│  │     └─ (creates directories, symlinks, migration version)
│  │
│  └─ Phase 1h: .gitignore
│     └─ Edit: .gitignore (append fab/current)
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
   ├─ Read: fab/.kit-migration-version, fab/.kit/VERSION
   ├─ Glob: fab/.kit/migrations/*.md
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
| Bash | `fab-doctor.sh`, `fab-sync.sh`, `fab log command` |
| Glob | Discover migration files |

### Sub-agents

None.
