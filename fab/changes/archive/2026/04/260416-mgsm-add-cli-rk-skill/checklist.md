# Quality Checklist: Add _cli-rk Skill

**Change**: 260416-mgsm-add-cli-rk-skill
**Generated**: 2026-04-16
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 _cli-rk.md exists at `src/kit/skills/_cli-rk.md` with correct frontmatter (name, description, user-invocable: false, disable-model-invocation: true, metadata.internal: true)
- [ ] CHK-002 _cli-rk.md documents detection pattern (`command -v rk`) with silent failure
- [ ] CHK-003 _cli-rk.md documents iframe window creation (tmux `@rk_type iframe`, `@rk_url`)
- [ ] CHK-004 _cli-rk.md documents proxy URL pattern (`/proxy/{port}/...`)
- [ ] CHK-005 _cli-rk.md documents server URL discovery via `rk context` at use-time
- [ ] CHK-006 _cli-rk.md documents visual display recipe (4-step centralized pattern)
- [ ] CHK-007 _cli-rk.md documents visual-explainer integration note with silent failure
- [ ] CHK-008 _preamble.md includes `_cli-rk` in always-load section after `_naming`
- [ ] CHK-009 Spec file exists at `docs/specs/skills/SPEC-_cli-rk.md`

## Scenario Coverage

- [ ] CHK-010 Silent fail: rk not installed — no errors surfaced
- [ ] CHK-011 Silent fail: visual-explainer not available — no errors surfaced
- [ ] CHK-012 Preamble entry marked optional — skills skip gracefully if file missing

## Code Quality

- [ ] CHK-013 Pattern consistency: Frontmatter matches `_cli-external.md` and `_cli-fab.md` patterns
- [ ] CHK-014 Pattern consistency: Preamble entry follows same format as `_cli-fab` and `_naming` entries
- [ ] CHK-015 No unnecessary duplication: rk content not duplicated from `_cli-external.md`

## Documentation Accuracy

- [ ] CHK-016 Iframe commands match `rk context` output (tmux set-option syntax)
- [ ] CHK-017 Proxy URL pattern matches rk server implementation

## Cross References

- [ ] CHK-018 Spec file `SPEC-_cli-rk.md` follows pattern from existing skill specs
- [ ] CHK-019 `docs/specs/skills.md` updated if it lists internal skills

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
