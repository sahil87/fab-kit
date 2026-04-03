package main

import (
	"testing"
)

func TestPaneSendCmd(t *testing.T) {
	t.Run("requires two arguments", func(t *testing.T) {
		cmd := paneSendCmd()
		cmd.SetArgs([]string{"%5"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing text argument, got nil")
		}
	})

	t.Run("requires at least pane argument", func(t *testing.T) {
		cmd := paneSendCmd()
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing arguments, got nil")
		}
	})

	t.Run("no-enter flag defaults to false", func(t *testing.T) {
		cmd := paneSendCmd()
		noEnter, _ := cmd.Flags().GetBool("no-enter")
		if noEnter {
			t.Error("expected no-enter to default to false")
		}
	})

	t.Run("force flag defaults to false", func(t *testing.T) {
		cmd := paneSendCmd()
		force, _ := cmd.Flags().GetBool("force")
		if force {
			t.Error("expected force to default to false")
		}
	})

	t.Run("flag existence", func(t *testing.T) {
		cmd := paneSendCmd()

		noEnterFlag := cmd.Flags().Lookup("no-enter")
		if noEnterFlag == nil {
			t.Error("expected 'no-enter' flag to exist")
		}

		forceFlag := cmd.Flags().Lookup("force")
		if forceFlag == nil {
			t.Error("expected 'force' flag to exist")
		}
	})
}
