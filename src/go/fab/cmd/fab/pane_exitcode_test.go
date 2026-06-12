package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// TestPaneValidationExitCode pins the pane-family exit-code scheme shared
// with window-name: 2 = pane missing, 3 = any other tmux failure (F30).
func TestPaneValidationExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"pane not found", &pane.PaneNotFoundError{Pane: "%5"}, 2},
		{"wrapped pane not found", fmt.Errorf("validate: %w", &pane.PaneNotFoundError{Pane: "%5"}), 2},
		{"other tmux failure", errors.New("tmux display-message: exit status 1: no server running"), 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := paneValidationExitCode(tc.err); got != tc.want {
				t.Errorf("paneValidationExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
