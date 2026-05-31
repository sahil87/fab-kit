package main

import (
	"testing"
)

func TestFabKitCommands(t *testing.T) {
	// Verify fabKitCommands map contains the expected entries
	expected := []string{"init", "upgrade-repo", "sync"}
	for _, cmd := range expected {
		if !fabKitCommands[cmd] {
			t.Errorf("expected fabKitCommands to contain %q", cmd)
		}
	}

	// Verify workflow commands are not in the map (they belong to fab-go)
	workflow := []string{"status", "preflight", "resolve", "log", "change", "score"}
	for _, cmd := range workflow {
		if fabKitCommands[cmd] {
			t.Errorf("expected fabKitCommands to NOT contain %q (belongs to fab-go)", cmd)
		}
	}
}

func TestUpdateCmd_HasSkipBrewUpdateFlag(t *testing.T) {
	cmd := updateCmd()
	flag := cmd.Flags().Lookup("skip-brew-update")
	if flag == nil {
		t.Fatal("update command should register --skip-brew-update flag")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--skip-brew-update should be a bool flag, got %q", flag.Value.Type())
	}
	if flag.DefValue != "false" {
		t.Errorf("--skip-brew-update should default to false, got %q", flag.DefValue)
	}
}

func TestVersion(t *testing.T) {
	// The version variable should be set (defaults to "dev")
	if version == "" {
		t.Error("version should not be empty")
	}
}

func TestDisplayVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1.9.4", "v1.9.4"},
		{"v1.9.4", "v1.9.4"},
		{"dev", "dev"},
	}
	for _, tc := range cases {
		if got := displayVersion(tc.in); got != tc.want {
			t.Errorf("displayVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
