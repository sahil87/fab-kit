package main

import (
	"bytes"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
)

func TestFabKitArgs(t *testing.T) {
	// Verify the allowlist contains exactly the fab-kit commands
	expected := []string{"init", "upgrade-repo", "sync", "update"}
	for _, cmd := range expected {
		if !fabKitArgs[cmd] {
			t.Errorf("expected fabKitArgs to contain %q", cmd)
		}
	}

	// Verify workflow commands are not in the allowlist (they go to fab-go)
	workflow := []string{"status", "preflight", "resolve", "log", "change", "score"}
	for _, cmd := range workflow {
		if fabKitArgs[cmd] {
			t.Errorf("expected fabKitArgs to NOT contain %q (belongs to fab-go)", cmd)
		}
	}
}

func TestFabGoNoConfigArgs(t *testing.T) {
	// Verify pane is the sole exempt subcommand
	if len(fabGoNoConfigArgs) != 1 {
		t.Errorf("expected fabGoNoConfigArgs to contain exactly 1 entry, got %d", len(fabGoNoConfigArgs))
	}
	if !fabGoNoConfigArgs["pane"] {
		t.Error("expected fabGoNoConfigArgs to contain \"pane\"")
	}

	// Verify config-required commands are NOT in the allowlist — they MUST
	// continue to fail-closed outside a fab repo.
	configRequired := []string{"runtime", "status", "preflight", "change", "score"}
	for _, cmd := range configRequired {
		if fabGoNoConfigArgs[cmd] {
			t.Errorf("expected fabGoNoConfigArgs to NOT contain %q (must require config)", cmd)
		}
	}
}

func TestResolveFabVersion(t *testing.T) {
	cases := []struct {
		name          string
		cfg           *internal.ConfigResult
		arg0          string
		routerVersion string
		wantVersion   string
		wantExit      bool
	}{
		{
			name:          "in-repo exempt command uses project version",
			cfg:           &internal.ConfigResult{FabVersion: "1.3.7"},
			arg0:          "pane",
			routerVersion: "dev",
			wantVersion:   "1.3.7",
			wantExit:      false,
		},
		{
			name:          "in-repo non-exempt command uses project version",
			cfg:           &internal.ConfigResult{FabVersion: "1.3.7"},
			arg0:          "status",
			routerVersion: "dev",
			wantVersion:   "1.3.7",
			wantExit:      false,
		},
		{
			name:          "out-of-repo exempt command uses router version",
			cfg:           nil,
			arg0:          "pane",
			routerVersion: "1.3.7",
			wantVersion:   "1.3.7",
			wantExit:      false,
		},
		{
			name:          "out-of-repo non-exempt command signals exit",
			cfg:           nil,
			arg0:          "status",
			routerVersion: "1.3.7",
			wantVersion:   "",
			wantExit:      true,
		},
		{
			name:          "out-of-repo empty arg signals exit",
			cfg:           nil,
			arg0:          "",
			routerVersion: "1.3.7",
			wantVersion:   "",
			wantExit:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotVersion, gotExit := resolveFabVersion(tc.cfg, tc.arg0, tc.routerVersion)
			if gotVersion != tc.wantVersion {
				t.Errorf("version: want %q, got %q", tc.wantVersion, gotVersion)
			}
			if gotExit != tc.wantExit {
				t.Errorf("shouldExit: want %v, got %v", tc.wantExit, gotExit)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
}

func TestPrintVersion(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		var buf bytes.Buffer
		printVersion(&buf, "1.3.1", nil)
		got := buf.String()
		if got != "fab 1.3.1\n" {
			t.Errorf("expected %q, got %q", "fab 1.3.1\n", got)
		}
	})

	t.Run("matching versions", func(t *testing.T) {
		var buf bytes.Buffer
		printVersion(&buf, "1.3.1", &internal.ConfigResult{FabVersion: "1.3.1"})
		got := buf.String()
		want := "fab 1.3.1\nproject: 1.3.1\n"
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})

	t.Run("differing versions", func(t *testing.T) {
		var buf bytes.Buffer
		printVersion(&buf, "1.4.0", &internal.ConfigResult{FabVersion: "1.3.1"})
		got := buf.String()
		want := "fab 1.4.0\nproject: 1.3.1\n"
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})
}
