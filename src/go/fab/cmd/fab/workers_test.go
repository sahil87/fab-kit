package main

import (
	"reflect"
	"testing"
)

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

func TestEnvWithWorkers(t *testing.T) {
	t.Run("replaces an inherited entry rather than duplicating it", func(t *testing.T) {
		got := envWithWorkers([]string{"PATH=/bin", "FAB_AGENT_WORKERS=parent", "HOME=/root"}, "codex")
		want := []string{"PATH=/bin", "HOME=/root", "FAB_AGENT_WORKERS=codex"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("env = %#v, want %#v", got, want)
		}
	})

	t.Run("appends when the parent carried no entry", func(t *testing.T) {
		got := envWithWorkers([]string{"PATH=/bin"}, "codex")
		want := []string{"PATH=/bin", "FAB_AGENT_WORKERS=codex"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("env = %#v, want %#v", got, want)
		}
	})

	t.Run("an explicit empty override still replaces the inherited value", func(t *testing.T) {
		got := envWithWorkers([]string{"FAB_AGENT_WORKERS=parent"}, "")
		want := []string{"FAB_AGENT_WORKERS="}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("env = %#v, want %#v", got, want)
		}
	})

	t.Run("a variable that merely shares the prefix is preserved", func(t *testing.T) {
		got := envWithWorkers([]string{"FAB_AGENT_WORKERS_EXTRA=keep"}, "codex")
		want := []string{"FAB_AGENT_WORKERS_EXTRA=keep", "FAB_AGENT_WORKERS=codex"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("env = %#v, want %#v", got, want)
		}
	})
}
