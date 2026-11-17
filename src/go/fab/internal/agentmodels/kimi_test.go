package agentmodels

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestKimiModelTablesAndDefault(t *testing.T) {
	home := t.TempDir()
	const secret = "credential-must-not-leak"
	writeTestFile(t, filepath.Join(home, ".kimi-code", "config.toml"), `default_model = "openrouter/nex"

[models."openrouter/nex"]
display_name = "Nex"
support_efforts = [ "low", "high" ]

[models.simple]
support_efforts = []

[providers.openrouter]
api_key = "`+secret+`"

[[hooks]]
command = "also-ignored"
`)
	t.Setenv("HOME", home)
	rows, defaultID, err := discoverKimi(defaultEnvironment())
	if err != nil || defaultID != "openrouter/nex" {
		t.Fatalf("defaultID=%q err=%v", defaultID, err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].Label != "Nex" || !reflect.DeepEqual(rows[0].Efforts, []string{"low", "high"}) {
		t.Errorf("quoted model = %+v", rows[0])
	}
	if rows[1].ID != "simple" || rows[1].Efforts == nil {
		t.Errorf("bare model = %+v", rows[1])
	}
	if strings.Contains(fmt.Sprint(rows, defaultID, err), secret) {
		t.Fatal("provider credential surfaced")
	}
}

func TestKimiIgnoresUnsupportedValuesInOtherTables(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".kimi-code", "config.toml"), `default_model = "real"
[providers.example]
ports = [1, 2]
args = [
  "--flag",
  "[not-a-header]",
]
description = """
multi-line value
[also-not-a-header]
"""
[models.real]
display_name = "Real"
`)
	t.Setenv("HOME", home)
	rows, defaultID, err := discoverKimi(defaultEnvironment())
	if err != nil || defaultID != "real" {
		t.Fatalf("defaultID=%q err=%v", defaultID, err)
	}
	if len(rows) != 1 || rows[0].ID != "real" || rows[0].Label != "Real" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestKimiMissingFileSilent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	rows, defaultID, err := discoverKimi(defaultEnvironment())
	if err != nil || defaultID != "" || len(rows) != 0 {
		t.Fatalf("rows=%v defaultID=%q err=%v", rows, defaultID, err)
	}
}

func TestKimiMalformedModelLineDoesNotLeakSecrets(t *testing.T) {
	home := t.TempDir()
	const secret = "credential-must-not-leak"
	writeTestFile(t, filepath.Join(home, ".kimi-code", "config.toml"), `[models."safe"]
display_name = "unterminated
[providers.openrouter]
api_key = "`+secret+`"
`)
	t.Setenv("HOME", home)
	_, _, err := discoverKimi(defaultEnvironment())
	if err == nil {
		t.Fatal("expected malformed model line to warn")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "unterminated") {
		t.Fatalf("error copied config content: %v", err)
	}
}
