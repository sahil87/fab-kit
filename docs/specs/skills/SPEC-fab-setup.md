# fab-setup

## Contents

- [Summary](#summary)
- [Flow](#flow)

## Summary

Bootstraps a new project or manages config/constitution/migrations. Creates `fab/project/` files and — via `fab sync` — `docs/memory/`, `docs/specs/`, deployed skill copies, and gitignore entries. Safe to re-run.

**Prose optimization** (260620-skop): skill content trimmed to remove re-explanation of partial-owned concepts and consolidate the seven Migrations Output Format blocks to one canonical block plus exact-string variant notes, and a `## Contents` TOC added; no behavioral change (Flow / Tools / Sub-agents unchanged).

**test_paths detection** (260626-5qf5): Config Create-Mode gains a **non-interactive** detection sub-step (step 2) that reads on-disk marker files (`go.mod`, `pyproject.toml`/`pytest.ini`, jest/vitest deps, `pom.xml`/`build.gradle`, `*.csproj`) and derives an anchored `test_paths` pattern, substituted as the new `{TEST_PATHS}` placeholder (step 4, alongside `{PROJECT_NAME}`/`{PROJECT_DESCRIPTION}`/`{SOURCE_PATHS}`) while preserving the scaffold's standing example comment block. It never prompts; Config Output surfaces the detected ecosystem+patterns (or "no test convention detected → left empty" for Rust/unrecognized stacks). The `2.7.1-to-2.8.0` migration backfills the same detection + comment refresh for existing repos.

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
│  │  (create mode when missing, raw template, OR missing
│  │   project.name/project.description — the canonical
│  │   fab init flow writes a fab_version-only config.yaml)
│  │  ├─ Read: README, package.json (project context)
│  │  ├─ Read: src/kit/scaffold/fab/project/config.yaml
│  │  ├─ (interactive: ask name, description, source_paths)
│  │  ├─ (NON-INTERACTIVE test_paths detection, 5qf5 — read on-disk
│  │  │   marker files → derive an anchored {TEST_PATHS} pattern from
│  │  │   the marker→ecosystem table: go.mod→**/*_test.go,
│  │  │   pyproject.toml/pytest.ini→**/test_*.py+**/*_test.py,
│  │  │   jest/vitest→**/*.spec|test.ts|js, pom.xml/build.gradle→
│  │  │   **/src/test/**, *.csproj→**/*Tests.cs+**/*Test.cs;
│  │  │   Rust/unrecognized → leave empty. NO prompt. Substituted as
│  │  │   the {TEST_PATHS} placeholder in step 4, preserving the
│  │  │   scaffold's standing example comment block; visible note in
│  │  │   Config Output: detected ecosystem+patterns, or "no convention
│  │  │   detected → empty")
│  │  └─ Write: fab/project/config.yaml
│  │     (preserves an existing fab_version key; on a fresh create
│  │      with no prior key, stamps the engine version from
│  │      $(fab kit-path)/VERSION — the scaffold template lacks it
│  │      and the router/fab sync error without it, c5tr)
│  │
│  ├─ Phase 1b: constitution.md
│  │  ├─ Read: src/kit/scaffold/fab/project/constitution.md
│  │  ├─ Read: project context (config, README, codebase)
│  │  ├─ (agent generates principles)
│  │  └─ Write: fab/project/constitution.md
│  │
│  └─ Phase 1c: fab sync (sync-first reorder, 260611-szxd f077 —
│     │          moved from last [old 1j] to immediately after 1a/1b,
│     │          since sync needs config.yaml's fab_version; outcome
│     │          identical via idempotency. Old hand-scaffolding steps
│     │          1c-1g/1i/1k are deleted — sync owns them all)
│     └─ Bash: fab sync
│        ├─ (copy-if-absent: context.md, code-quality.md, code-review.md,
│        │   docs/memory/index.md, docs/specs/index.md)
│        ├─ (directories: fab/changes/ + archive + .gitkeep)
│        ├─ (fab/.kit-migration-version — version logic per 1d note)
│        ├─ (skill deployment to .claude/skills/)
│        ├─ (.gitignore line-ensure merge: .fab-* — subsumes the old
│        │   literal .fab-status.yaml append)
│        └─ [non-zero exit] STOP — surface sync output (failure guard)
│
├── config: Config ──────────────────────────────────────
│  ├─ Read: fab/project/config.yaml
│  ├─ (interactive: menu → edit section; sections: project /
│  │   source_paths / checklist / context.md / code-quality.md /
│  │   code-review.md — the dead stage_directives editor was
│  │   removed in c5tr, nothing ever read the key)
│  └─ Edit: fab/project/config.yaml
│
├── constitution: Constitution ──────────────────────────
│  ├─ Read: fab/project/config.yaml, constitution.md
│  ├─ (interactive: amendment menu)
│  └─ Edit: fab/project/constitution.md
│
└── migrations: Migrations ─────────────────────────────
   ├─ Bash: fab migrations-status --json   (binary-owned discovery —
   │  │     incl. version read/parse/compare; 260611-szxd f080 deleted
   │  │     the skill-side existence checks, integer parsing, manual
   │  │     compare step, and Semver Comparison section. The binary
   │  │     exits non-zero with remediation hints on missing version
   │  │     files — skill surfaces stderr and stops)
   │  ├─ STOP if `overlaps` non-empty (report conflict)
   │  └─ `applicable` empty → branch on returned local/engine fields:
   │     equal → "Already up to date"; local ahead → "Local Version
   │     Ahead"; otherwise → "No Migrations Apply"
   │     (one-line semver rule restored at this branch in c5tr —
   │      compare MAJOR/MINOR/PATCH as integers, not lexicographically;
   │      f080 had deleted it with the rest of the Semver section)
   ├─ For each file in `applicable` (in order):
   │  ├─ Read: $(fab kit-path)/migrations/{file}
   │  ├─ (execute pre-checks, changes, verification)
   │  └─ Write: fab/.kit-migration-version (TO)
   └─ Finalize (version already at last TO; no-op case stamped by upgrade-repo)
```

Discovery (version read/parse/compare + scan/validate-non-overlap/sort + the
applicability walk) is owned by the `fab-kit` binary (`fab migrations-status`).
The skill consumes the `--json` result and still owns *application* of each
migration file (Pre-check/Changes/Verification + writing `TO`), per Pure Prompt Play — a fab-kit design principle.

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Scaffold templates, project files, migration files |
| Write | Project files, migration version |
| Edit | Config, constitution (`.gitignore` is owned by `fab sync`'s line-ensure merge — no direct edit since the 1k deletion) |
| Bash | `fab doctor`, `fab sync`, `fab log command`, `fab migrations-status --json` (migration discovery) |

### Sub-agents

None.
