package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestDoctorCmd(porcelain bool) (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := doctorCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if porcelain {
		cmd.Flags().Set("porcelain", "true")
	}
	return cmd, &buf
}

func TestRunDoctorChecks_ReturnsResults(t *testing.T) {
	cmd, buf := newTestDoctorCmd(false)

	// Run checks (non-porcelain) — we don't call Execute because it calls os.Exit
	failures := runDoctorChecks(cmd, false)

	output := buf.String()

	// Should contain the header
	if !strings.Contains(output, "fab-doctor: checking prerequisites") {
		t.Error("expected header line in output")
	}

	// Should contain summary line with N/7
	if !strings.Contains(output, "/7 checks passed") {
		t.Errorf("expected summary with /7 in output, got:\n%s", output)
	}

	// Failures should be non-negative
	if failures < 0 {
		t.Errorf("failures = %d, want >= 0", failures)
	}
}

func TestRunDoctorChecks_Porcelain(t *testing.T) {
	cmd, buf := newTestDoctorCmd(true)

	failures := runDoctorChecks(cmd, true)

	output := buf.String()

	// Should NOT contain header or summary
	if strings.Contains(output, "fab-doctor: checking prerequisites") {
		t.Error("porcelain mode should not contain header")
	}
	if strings.Contains(output, "checks passed") {
		t.Error("porcelain mode should not contain summary")
	}

	// Should NOT contain tick marks
	if strings.Contains(output, "\u2713") {
		t.Error("porcelain mode should not contain pass marks")
	}

	// If there are failures, output should be non-empty
	if failures > 0 && output == "" {
		t.Error("porcelain mode with failures should produce output")
	}
}

func TestParseBashVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GNU bash, version 5.2.26(1)-release (aarch64-apple-darwin23.2.0)", "5.2.26"},
		{"GNU bash, version 3.2.57(1)-release (x86_64-apple-darwin21)", "3.2.57"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		got := parseBashVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseBashVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseYqVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"yq (https://github.com/mikefarah/yq/) version v4.44.1", "4.44.1"},
		{"yq version 4.44.1", "4.44.1"},
		{"yq version v4.30.8", "4.30.8"},
		{"", ""},
		{"some weird output", ""},
	}
	for _, tt := range tests {
		got := parseYqVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseYqVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseGhVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"gh version 2.45.0 (2024-03-04)", "2.45.0"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		got := parseGhVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseGhVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMajorVersion(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"4.44.1", 4},
		{"3.2.0", 3},
		{"", 0},
		{"abc", 0},
	}
	for _, tt := range tests {
		got := majorVersion(tt.input)
		if got != tt.want {
			t.Errorf("majorVersion(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFabKitCommands_IncludesDoctor(t *testing.T) {
	if !fabKitCommands["doctor"] {
		t.Error("fabKitCommands should include 'doctor'")
	}
}
