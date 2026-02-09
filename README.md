# sddr
SDD Research

Research and documentation on Specification-Driven Development (SDD) tools and methodologies.

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
├── doc/
│   ├── speckit/      # Analysis of GitHub's Spec-Kit
│   ├── openspec/     # Analysis of Fission AI's OpenSpec
│   └── findings/     # Cross-cutting research findings
└── README.md
```

## Structure

The documentation and specs for this repo reside in `fab/specs` and `fab/docs`.

The `references/` folder contains docs from other libraries and projects, included purely for reference.

### [references/speckit/](references/speckit/)
Comprehensive analysis of **Spec-Kit** (https://github.com/github/spec-kit) - GitHub's SDD toolkit.
- Start with [README.md](references/speckit/README.md) for overview
- Key docs: philosophy, workflow, commands, templates, agents

### [references/openspec/](references/openspec/)
In-depth analysis of **OpenSpec** (https://github.com/Fission-AI/OpenSpec) - an AI-native spec-driven framework.
- Start with [README.md](references/openspec/README.md) for overview
- Key docs: overview, philosophy, cli-architecture, agent-integration
