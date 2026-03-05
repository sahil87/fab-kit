# Tasks: Add Conventions Section to config.yaml

**Change**: 260213-r3m7-add-conventions-section
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 Add `conventions:` section with commented-out example keys to `fab/config.yaml` — place after `source_paths:` and before `stages:`, include section header comment, per-key inline comments, and example values for `branch_naming`, `pr_title`, and `backlog`

## Phase 2: Documentation

- [x] T002 Update `fab/docs/fab-workflow/configuration.md` — add `conventions` subsection under `config.yaml` Schema documenting the section purpose, key definitions (name, type, description), and relationship to `naming`/`git` sections

---

## Execution Order

- T001 and T002 are independent (config file vs. centralized doc) but T001 first establishes the canonical format that T002 documents
