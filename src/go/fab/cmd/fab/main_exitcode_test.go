package main

import (
	"io"
	"testing"
)

// TestRunExitCode pins the binary-wide exit-code convention (toolkit principle
// №4): 0 = success, 1 = operational failure, 2 = usage error. Usage errors are
// caught at parse/validation time before any RunE begins; operational errors
// originate from inside a RunE. Classification rides on execution phase, not on
// message-string matching — this table drives run() end-to-end through the live
// cobra tree so the flag-parse, arg-count, unknown-subcommand, and
// flags-group seams are all exercised. Mirrors pane_exitcode_test.go's shape.
func TestRunExitCode(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		// Usage errors → 2 (all four classes exit 1 today).
		{"unknown flag", []string{"score", "--nope"}, 2},
		{"arg-count violation", []string{"score"}, 2},
		{"unknown subcommand", []string{"nonsense"}, 2},
		{"flags-group conflict", []string{"resolve", "--status", "--folder"}, 2},

		// Operational failure → 1: valid syntax, the error surfaces from inside
		// RunE (a change that cannot exist by name).
		{"operational nonexistent change", []string{"resolve", "definitely-not-a-real-change-xyz"}, 1},

		// Success → 0.
		{"success (version)", []string{"--version"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := run(tc.args, io.Discard); got != tc.want {
				t.Errorf("run(%v) = %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}
