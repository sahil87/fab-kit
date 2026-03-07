# Quality Checklist: Add `/fab-operator1` Skill

**Change**: 260306-qkov-operator1-skill
**Generated**: 2026-03-07
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Skill file structure: `fab/.kit/skills/fab-operator1.md` exists with valid frontmatter (name, description)
- [x] CHK-002 Context loading: Skill loads always-load layer only, does NOT reference change-specific artifacts
- [x] CHK-003 Orientation on start: Skill displays pane map and status summary on invocation
- [x] CHK-004 State re-derivation: Skill instructs re-querying live state before every action
- [x] CHK-005 Seven use cases: All seven use cases documented in skill (broadcast, sequenced rebase, merge PRs, spawn worktree, status dashboard, unstick agent, notification surface)
- [x] CHK-006 Confirmation model: Three-tier risk model documented (read-only, recoverable, destructive)
- [x] CHK-007 Pre-send validation: Skill requires pane existence check + idle check before send-keys
- [x] CHK-008 Bounded retries: Retry limits documented per action type
- [x] CHK-009 Context discipline: Skill explicitly prohibits loading change artifacts
- [x] CHK-010 Not a lifecycle enforcer: Skill explicitly disclaims lifecycle enforcement
- [x] CHK-011 `fab send-keys` command: Subcommand exists with correct signature (`<change>` `<text>`)
- [x] CHK-012 `fab send-keys` pane resolution: Resolves change to pane via tmux + fab/current matching
- [x] CHK-013 `fab send-keys` tmux guard: Exits non-zero when $TMUX unset
- [x] CHK-014 `fab send-keys` pane validation: Validates pane exists before sending
- [x] CHK-015 `fab send-keys` no idle check: CLI does NOT enforce idle checks (separation of concerns)
- [x] CHK-016 Spec renamed: SPEC-fab-conductor.md renamed to SPEC-fab-operator1.md with references updated
- [x] CHK-017 _scripts.md updated: send-keys documented in command reference

## Behavioral Correctness
- [x] CHK-018 send-keys multi-pane: Sends to first matching pane with stderr warning when multiple panes match
- [x] CHK-019 send-keys no-pane: Exits non-zero with descriptive error when no pane found for change
- [x] CHK-020 send-keys change-not-found: Exits non-zero when change resolution fails

## Scenario Coverage
- [x] CHK-021 Scenario: Send text to known change's pane — `fab send-keys r3m7 "/fab-continue"` resolves and sends
- [x] CHK-022 Scenario: Change not found — exits with "No change matches" error
- [x] CHK-023 Scenario: No tmux session — exits with "Error: not inside a tmux session"
- [x] CHK-024 Scenario: No pane for change — exits with "No tmux pane found for change" error

## Edge Cases & Error Handling
- [x] CHK-025 Pane disappears between discovery and send — error reported, no crash
- [x] CHK-026 Multiple panes for same change — first pane used, warning emitted
- [x] CHK-027 Outside tmux degraded mode — operator skill notes tmux unavailable, status-only mode

## Code Quality
- [x] CHK-028 Pattern consistency: Go code follows existing patterns in panemap.go and runtime.go (cobra command structure, resolve package usage, error handling)
- [x] CHK-029 No unnecessary duplication: Reuses resolve.ToFolder, discoverPanes pattern, and readFabCurrent from existing code

## Documentation Accuracy
- [x] CHK-030 _scripts.md send-keys section matches actual command implementation
- [x] CHK-031 SPEC-fab-operator1.md references updated from "conductor" to "operator1"

## Cross References
- [x] CHK-032 main.go registers sendKeysCmd() alongside existing commands
- [x] CHK-033 Skill file cross-references correct CLI primitives (pane-map, status show, runtime, send-keys)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
