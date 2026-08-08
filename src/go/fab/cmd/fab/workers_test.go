package main

import "testing"

func TestWithWorkersEnv(t *testing.T) {
	command := "claude '/fab-new example'"
	if got := withWorkersEnv(command, "ignored", false); got != command {
		t.Errorf("omitted override changed command: %q", got)
	}

	got := withWorkersEnv(command, "co'dex; $(touch nope)", true)
	want := "FAB_AGENT_WORKERS='co'\\''dex; $(touch nope)' " + command
	if got != want {
		t.Errorf("quoted command = %q, want %q", got, want)
	}

	if got := withWorkersEnv(command, "", true); got != "FAB_AGENT_WORKERS='' "+command {
		t.Errorf("explicit empty override = %q", got)
	}
}
