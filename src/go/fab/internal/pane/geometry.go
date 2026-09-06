package pane

import (
	"fmt"
	"strconv"
	"strings"
)

// This file holds the GEOMETRY PROBE of the shared pane layer: one
// `display-message` query that reads the window and pane dimensions a placement
// decision needs, plus its pure parse half.
//
// The probe exists for internal/dispatch's geometry floor (the pane-mode split
// demotes to a manually-sized window when the planned worker pane would land
// below dispatch.min_cols / dispatch.min_rows). It rides the same RunCmd /
// WithServer / StderrError helpers as every other tmux invocation in this
// package, and it is FAIL-OPEN by contract: a caller that cannot read the
// geometry proceeds with the unsized-split behavior rather than failing the
// dispatch — placement is cosmetic and must never fail an otherwise-launchable
// dispatch.

// Geometry is the measured size of the window a target pane lives in and of the
// target pane itself. The window pair prices a CARVING split (the worker column
// is a percent of the window's width and takes its full height); the pane pair
// prices a STACKING split (the worker inherits the column's width and half the
// sibling pane's height, tmux even-splitting an unsized `-v`).
type Geometry struct {
	WindowWidth  int
	WindowHeight int
	PaneWidth    int
	PaneHeight   int
}

// ProbeGeometry reads the geometry of the window containing target (a pane id)
// and of target itself, in a single tmux round-trip:
//
//	tmux [-L <server>] display-message -p -t <target> -F '#{window_width} #{window_height} #{pane_width} #{pane_height}'
//
// `-t` on a pane resolves the window formats to that pane's window, so one query
// serves both split shapes. Any failure — a dead server, a gone pane, or output
// that does not parse — is returned as an error for the caller to fail open on;
// nothing here decides policy.
func ProbeGeometry(server, target string) (Geometry, error) {
	out, stderr, err := RunCmd("tmux", WithServer(server,
		"display-message", "-p", "-t", target, "-F",
		"#{window_width} #{window_height} #{pane_width} #{pane_height}")...)
	if err != nil {
		return Geometry{}, StderrError(fmt.Errorf("tmux display-message (geometry): %w", err), stderr)
	}
	return parseGeometry(out)
}

// parseGeometry is the pure half of ProbeGeometry: it parses the four
// space-separated integers the format string requests. Extracted so the row
// grammar is unit-testable without a tmux server. Short rows, extra fields, and
// non-numeric fields are all errors — a partial reading would price the floor on
// fabricated dimensions, and the caller's fail-open posture needs a clean
// signal, not a guess.
func parseGeometry(out string) (Geometry, error) {
	fields := strings.Fields(out)
	if len(fields) != 4 {
		return Geometry{}, fmt.Errorf("geometry probe returned %d fields %q, want 4 (window_width window_height pane_width pane_height)", len(fields), strings.TrimSpace(out))
	}
	var nums [4]int
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil {
			return Geometry{}, fmt.Errorf("geometry probe field %d %q is not an integer: %w", i, f, err)
		}
		nums[i] = n
	}
	return Geometry{WindowWidth: nums[0], WindowHeight: nums[1], PaneWidth: nums[2], PaneHeight: nums[3]}, nil
}
