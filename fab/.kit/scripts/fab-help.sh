#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/fab-help.sh — Print Fab Kit help overview
#
# Run from anywhere: fab/.kit/scripts/fab-help.sh
# Safe to re-run (read-only, no side effects).

kit_dir="$(cd "$(dirname "$0")/.." && pwd)"

if [ -f "$kit_dir/VERSION" ]; then
  version=$(cat "$kit_dir/VERSION")
else
  version="unknown"
fi

cat <<EOF
Fab Kit v${version} — Specification-Driven Development

WORKFLOW

  /fab-new ─→ /fab-continue (or /fab-ff) ─→ /fab-archive
               ↕ /fab-clarify

  Planning stages: spec → tasks
  Execution stages: apply → review → hydrate

COMMANDS

  Start & Navigate
    /fab-new <desc>         Start a new change from a description
    /fab-switch [name]      Switch active change (lists all if no name)
    /fab-status             Show current change state at a glance

  Planning
    /fab-continue [stage]   Advance to the next planning stage (or reset to stage)
    /fab-ff                 Fast-forward through all remaining planning stages
    /fab-clarify            Refine the current stage artifact without advancing

  Execution
    /fab-apply              Implement tasks from tasks.md in dependency order
    /fab-review             Validate implementation against specs and checklists

  Completion
    /fab-archive            Archive change — move to archive, mark backlog done

  Maintenance
    /fab-hydrate-specs [domain]   Identify doc→spec gaps and propose additions

  Setup
    /fab-init               Bootstrap fab/ directory structure (safe to re-run)
    /fab-hydrate <sources>  Ingest external sources into fab/memory/
    /fab-help               Show this help

TYPICAL FLOW

  Quick change:  /fab-new → /fab-ff → /fab-archive
  Careful change: /fab-new → /fab-continue (repeat) → /fab-archive
EOF
