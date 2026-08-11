# fab-setup
Bootstraps a new project or manages config/constitution/migrations. Creates `fab/project/` files and — via `fab sync` — `docs/memory/`, `docs/specs/`, deployed skill copies, and gitignore entries. Safe to re-run.
## Flow
```
User invokes /fab-setup [subcommand]
├─ Pre-flight: src/kit/ + VERSION exist; Bash: fab log command "fab-setup"
├─── No argument: Bootstrap
│  ├─ Bash: fab doctor → [non-zero] STOP
│  ├─ config.yaml: fab init --project seeds identity; refine in place
│  ├─ Read scaffold + project context → Write constitution.md
│  └─ Bash: fab sync (files, skill deploy, .gitignore) → [non-zero] STOP
├─── config: Read → interactive menu → Edit: fab/project/config.yaml
├─── constitution: Read → amendment menu → Edit: constitution.md
└─── migrations
   ├─ Bash: fab migrations-status --json (binary discovers; skill applies each file)
   ├─ [overlaps non-empty] STOP
   └─ Per applicable file: Read migration → execute → Write: fab/.kit-migration-version
```
### Tools used
Read (scaffolds, project files, migration files), Write (project files, migration version), Edit (config, constitution), Bash (`fab doctor`, `fab sync`, `fab log command`, `fab migrations-status --json`).
### Sub-agents
None.
