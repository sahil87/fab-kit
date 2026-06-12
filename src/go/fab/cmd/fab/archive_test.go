package main

import "testing"

// fab change archive with no argument must be a usage error (non-zero exit),
// not exit-0 help text (k4ge).
func TestChangeArchiveCmd_NoArgIsUsageError(t *testing.T) {
	cmd := changeArchiveCmd()
	cmd.SetArgs([]string{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected a usage error when no argument is given")
	}
}
