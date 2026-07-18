package main

import (
	"strings"
	"testing"
)

// exampleTargetPaths are the user-facing, multi-flag commands the toolkit
// principle №3 audit (change 260717-b91h, deferred from 260717-ptwh) requires
// to carry runnable `Example:` invocations in their `-h` output. The paths are
// argv fragments relative to the root command (excluding the "fab" root name),
// as cobra's Find consumes them.
var exampleTargetPaths = [][]string{
	{"batch", "archive"},
	{"batch", "switch"},
	{"config", "init"},
	{"config", "show"},
	{"resolve"},
	{"score"},
	{"dispatch", "start"},
}

// TestUserFacingCommandsCarryExamples walks the real assembled command tree
// (newRootCmd) and asserts that each of the 7 audit-named commands populates
// cobra's Example field. Cobra renders a non-empty Example under an "Examples:"
// heading in `-h`, and it byte-flows into the shll.ai command reference via
// help-dump — so a non-empty Example is the pinned contract surface. The walk
// uses cobra's own Find (the same resolution `-h` uses), so a renamed or
// re-parented command fails to resolve and trips the test rather than silently
// passing.
func TestUserFacingCommandsCarryExamples(t *testing.T) {
	root := newRootCmd()

	for _, path := range exampleTargetPaths {
		cmd, _, err := root.Find(path)
		if err != nil {
			t.Errorf("could not resolve command %q against the assembled tree: %v", strings.Join(path, " "), err)
			continue
		}
		// Find resolves to the nearest matching command; verify it landed on
		// the intended leaf (a bad path resolves to an ancestor, not an error).
		if cmd.Name() != path[len(path)-1] {
			t.Errorf("command path %q resolved to %q, not the intended leaf", strings.Join(path, " "), cmd.CommandPath())
			continue
		}
		if strings.TrimSpace(cmd.Example) == "" {
			t.Errorf("command %q has an empty Example field — populate it so `-h` shows runnable invocations (toolkit principle №3)", cmd.CommandPath())
		}
	}
}

// TestExampleBlocksAreTwoSpaceIndented asserts the cobra-convention formatting
// on the populated Example blocks: every non-blank line begins with a two-space
// indent. This is the secondary formatting check the intake flags as optional —
// it keeps the rendered "Examples:" section aligned under cobra's stock
// template. (Purely a formatting guard; the non-empty assertion above is the
// primary contract.)
func TestExampleBlocksAreTwoSpaceIndented(t *testing.T) {
	root := newRootCmd()

	for _, path := range exampleTargetPaths {
		cmd, _, err := root.Find(path)
		if err != nil || cmd.Name() != path[len(path)-1] {
			// Resolution failures are already reported by the test above;
			// skip here to avoid duplicate noise.
			continue
		}
		for _, line := range strings.Split(cmd.Example, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if !strings.HasPrefix(line, "  ") {
				t.Errorf("command %q Example line is not two-space indented: %q", cmd.CommandPath(), line)
			}
		}
	}
}
