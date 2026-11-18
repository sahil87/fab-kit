package agentmodels

import (
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

func TestModelsCommandValidRowsAndOverrides(t *testing.T) {
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"mytool": {
			ModelsCommand: `printf '[{"id":"m1","cli":"x","source":"fill"},{"id":""}]'`,
			Profiles:      map[string]config.ProviderProfile{"default": {Model: "fill-model"}},
		},
	}}
	rows, warnings := discoverWith(cfg, "mytool", defaultEnvironment())
	if len(warnings) != 0 {
		t.Fatal(warnings)
	}
	row := findRow(rows, "m1")
	if row == nil || row.CLI != "mytool" || row.Source != SourceTool || row.Label != "m1" {
		t.Fatalf("command row = %+v", row)
	}
	if len(rows) != 3 || findRow(rows, "fill-model") == nil {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestModelsCommandKeepsOnlyFirstDefault(t *testing.T) {
	cfg := commandTestConfig(`printf '[{"id":"first","default":true},{"id":"second","default":true}]'`)
	rows, warnings := discoverWith(cfg, "mytool", defaultEnvironment())
	if len(warnings) != 0 {
		t.Fatal(warnings)
	}
	if rows[1].ID != "first" || !rows[1].Default || rows[2].ID != "second" || rows[2].Default {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestModelsCommandDropsToolReportedReservedID(t *testing.T) {
	cfg := commandTestConfig(`printf '[{"id":"tool-default","label":"impostor"},{"id":"m1","default":true}]'`)
	rows, warnings := discoverWith(cfg, "mytool", defaultEnvironment())
	if len(warnings) != 0 {
		t.Fatal(warnings)
	}
	if rows[0].Source != SourceReserved || rows[0].Label != "tool default (m1)" {
		t.Fatalf("reserved row = %+v", rows[0])
	}
	for _, row := range rows[1:] {
		if row.ID == "tool-default" {
			t.Fatalf("tool-reported reserved ID survived: %+v", rows)
		}
	}
}

func TestModelsCommandFailures(t *testing.T) {
	for _, test := range []struct {
		name    string
		command string
		want    string
	}{
		{name: "non-zero", command: "exit 2", want: "exit status 2"},
		{name: "malformed", command: "printf nope", want: "invalid JSON"},
		{name: "null is not an array", command: "printf null", want: "expected a JSON array"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := commandTestConfig(test.command)
			rows, warnings := discoverWith(cfg, "mytool", defaultEnvironment())
			assertFailureFills(t, rows, warnings)
			if !strings.Contains(warnings[0], test.want) {
				t.Errorf("warning = %q, want %q", warnings[0], test.want)
			}
		})
	}
	t.Run("timeout", func(t *testing.T) {
		env := defaultEnvironment()
		env.timeout = 20 * time.Millisecond
		rows, warnings := discoverWith(commandTestConfig("sleep 1"), "mytool", env)
		assertFailureFills(t, rows, warnings)
		if !strings.Contains(warnings[0], "timed out after 20ms") {
			t.Errorf("warning = %q", warnings[0])
		}
	})
}

func commandTestConfig(command string) *config.Config {
	return &config.Config{Providers: map[string]config.ProviderConfig{
		"mytool": {
			ModelsCommand: command,
			Profiles:      map[string]config.ProviderProfile{"default": {Model: "fill-model"}},
		},
	}}
}
