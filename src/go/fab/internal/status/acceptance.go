package status

import (
	"os"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/hooklib"
)

// LiveAcceptance derives acceptance progress from the change's plan.md
// `## Acceptance` checkboxes at read time, rather than trusting the
// hook-maintained `.status.yaml` counter (which goes stale on any
// hook-bypassing mutation — sed edits, direct file edits). It composes the
// existing hooklib counters so checkbox-parsing logic lives in one place.
//
// Returns (done, total, ok). ok is false — and done/total are 0 — when
// plan.md is absent/unreadable or has no `## Acceptance` heading; callers
// SHOULD then fall back to the persisted `.status.yaml` counter (the
// write-time cache). The counter remains authoritative only as that fallback;
// when ok is true the live count is the source of truth.
func LiveAcceptance(changeDir string) (done, total int, ok bool) {
	planPath := filepath.Join(changeDir, "plan.md")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return 0, 0, false
	}
	content := string(data)
	if !hooklib.HasSectionHeading(content, hooklib.SectionAcceptance) {
		return 0, 0, false
	}
	total = hooklib.CountSectionItemsBounded(content, hooklib.SectionAcceptance)
	done = hooklib.CountCompletedSectionItemsBounded(content, hooklib.SectionAcceptance)
	return done, total, true
}
