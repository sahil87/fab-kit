# Tasks: Fix Pipeline Ship Timing

**Change**: 260221-6ljc-fix-pipeline-ship-timing
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add delay and log messages before sending ship command in `fab/.kit/scripts/pipeline/run.sh` — in `poll_change()`, `polling_fab_ff` case, after `hydrate:done` detected: log "waiting for Claude to finish turn...", sleep 8s, log "Sending /changes:ship pr"
- [x] T002 Split `tmux send-keys` into text + Enter with 0.5s gap in `fab/.kit/scripts/pipeline/run.sh` — replace single `tmux send-keys -t "$pane_id" "/changes:ship pr" Enter` with two calls: text first, sleep 0.5, then Enter separately

## Execution Order

- T001 and T002 modify the same code block — T001 first (adds delay/logs around the send), T002 second (splits the send-keys call itself). In practice these are a single edit.
