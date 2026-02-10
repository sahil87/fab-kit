# fab

## What is Fab?

Fab (fabricate) is a Specification-Driven Development (SDD) workflow kit that runs entirely as AI agent prompts — no CLI installation, no system dependencies. It gives structure to the work developers already do (define, design, build, review, document) by providing named stages, markdown templates, and skill definitions that any AI agent (Claude Code, Cursor, Windsurf, etc.) can execute.

The core ideas:

1. **Pure prompt play** — The entire engine lives in `fab/.kit/` as markdown skill files and templates. You copy the directory into your project and go. No package manager, no binary, no runtime.
2. **Docs as source of truth** — Centralized docs in `fab/docs/` are the authoritative record of what the system does and why. Code changes flow *into* docs (via hydration at archive time), not the other way around.
3. **Change folders as the unit of work** — Each change gets its own folder under `fab/changes/` containing a proposal, spec, plan, tasks, and quality checklist. Git integration is optional and informational — Fab never touches branches, commits, or pushes.
4. **7 stages, 5 user-facing commands** — Internally there are 7 stages (proposal, specs, plan, tasks, apply, review, archive), but the user mostly interacts through `/fab-new`, `/fab-continue` (or `/fab-ff` to fast-forward), `/fab-apply`, `/fab-review`, and `/fab-archive`.
5. **Hybrid lineage** — Cherry-picks from two earlier systems: SpecKit's customizable folder structure, intuitive navigation, and pure-prompt approach, combined with OpenSpec's fast-forward workflow and centralized doc hydration on completion.

The design philosophy leans toward discipline without rigidity — it enforces a structured planning-before-coding workflow with quality checklists and spec validation, but keeps everything lightweight, git-optional, and easy to customize.

## Get Started

Copy the fab/.kit folder to your repo, and run:

```bash
fab-setup.sh #this should already by in your PATH because of .envrc
#Or else, run
fab/.kit/scripts/fab-setup.sh
```



## Repository Structure

```
sddr/
├── references/
│   ├── speckit/      # Analysis of GitHub's Spec-Kit
│   └── openspec/     # Analysis of Fission AI's OpenSpec
├── fab/              # Fab workflow kit
└── README.md
```

## Documentation

Start here: **[fab/specs/index.md](fab/specs/index.md)** — the specs index covers Fab's design, architecture, skills, and templates.

For post-implementation docs (what the system actually does), see [fab/docs/](fab/docs/).

## References

The `references/` folder contains docs from other libraries and projects, included purely for reference.

### [references/speckit/](references/speckit/)
Comprehensive analysis of **Spec-Kit** (https://github.com/github/spec-kit) - GitHub's SDD toolkit.
- Start with [README.md](references/speckit/README.md) for overview
- Key docs: philosophy, workflow, commands, templates, agents

### [references/openspec/](references/openspec/)
In-depth analysis of **OpenSpec** (https://github.com/Fission-AI/OpenSpec) - an AI-native spec-driven framework.
- Start with [README.md](references/openspec/README.md) for overview
- Key docs: overview, philosophy, cli-architecture, agent-integration
