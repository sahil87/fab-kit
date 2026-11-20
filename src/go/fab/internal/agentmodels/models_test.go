package agentmodels

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

func findRow(rows []Row, id string) *Row {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

func TestMergeRowsPrecedencePositionAndBackfill(t *testing.T) {
	rows := mergeRows([]Row{
		{ID: "same", Label: "Alias label", Efforts: []string{"low"}, Source: SourceAlias},
		{ID: "same", Source: SourceTool},
		{ID: "same", Default: true, Source: SourceFill},
		{ID: "other", Source: SourceFill},
	}, "")
	if len(rows) != 2 || rows[0].ID != "same" {
		t.Fatalf("first occurrence did not keep its position: %+v", rows)
	}
	row := rows[0]
	if row.Source != SourceTool || row.Label != "Alias label" || !reflect.DeepEqual(row.Efforts, []string{"low"}) || !row.Default {
		t.Errorf("merged row = %+v", row)
	}
}

func TestMergeRowsAppendsUnlistedDefault(t *testing.T) {
	rows := mergeRows([]Row{{ID: "known", Source: SourceFill}}, "new-default")
	if len(rows) != 2 || rows[1].ID != "new-default" || !rows[1].Default || rows[1].Source != SourceTool {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestDiscoverFillsAlwaysPresentAndSorted(t *testing.T) {
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"custom": {Profiles: map[string]config.ProviderProfile{
			"default": {Model: "z-fill"},
			"doing":   {Model: "a-fill"},
			"review":  {Model: "z-fill"},
		}},
	}}
	rows, warnings := discoverWith(cfg, "custom", defaultEnvironment())
	if len(warnings) != 0 {
		t.Fatal(warnings)
	}
	if len(rows) != 3 || rows[0].ID != agent.ToolDefaultModel || rows[1].ID != "a-fill" || rows[2].ID != "z-fill" {
		t.Fatalf("rows = %+v", rows)
	}
	for _, row := range rows[1:] {
		if row.CLI != "custom" || row.Source != SourceFill {
			t.Errorf("row = %+v", row)
		}
	}
}

func TestDiscoverAgyAndUserProviderWithoutAdapter(t *testing.T) {
	rows, warnings := discoverWith(nil, "agy", defaultEnvironment())
	if len(warnings) != 0 || len(rows) == 0 {
		t.Fatalf("agy rows=%d warnings=%v", len(rows), warnings)
	}
	provider, _ := agent.ResolveProvider(nil, "agy")
	for id := range map[string]struct{}{provider.Profiles["default"].Model: {}, provider.Profiles["fast"].Model: {}} {
		if findRow(rows, id) == nil {
			t.Errorf("agy fill %q missing", id)
		}
	}
	if rows[0].Source != SourceReserved || rows[0].Label != "tool default" || rows[0].Efforts == nil || len(rows[0].Efforts) != 0 {
		t.Errorf("agy reserved row = %+v", rows[0])
	}
	for _, row := range rows[1:] {
		if row.Source != SourceFill {
			t.Errorf("agy row = %+v", row)
		}
	}

	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"custom": {Profiles: map[string]config.ProviderProfile{"default": {Model: "custom-model"}}},
	}}
	rows, warnings = discoverWith(cfg, "custom", defaultEnvironment())
	if len(warnings) != 0 || len(rows) != 2 || rows[1].Source != SourceFill {
		t.Fatalf("custom rows=%+v warnings=%v", rows, warnings)
	}
}

func TestDiscoverToolDefaultRowUsesDiscoveredDefault(t *testing.T) {
	rows, warnings := discoverWith(commandTestConfig(`printf '[{"id":"m1","efforts":["high"],"default":true}]'`), "mytool", defaultEnvironment())
	if len(warnings) != 0 {
		t.Fatal(warnings)
	}
	want := Row{CLI: "mytool", ID: agent.ToolDefaultModel, Label: "tool default (m1)", Efforts: []string{"high"}, Source: SourceReserved}
	if len(rows) == 0 {
		t.Fatal("no rows returned")
	}
	if !reflect.DeepEqual(rows[0], want) {
		t.Fatalf("first row = %+v, want %+v", rows[0], want)
	}
	if rows[1].ID != "m1" || !rows[1].Default {
		t.Errorf("real default row = %+v, want default preserved", rows[1])
	}
}

func TestDiscoverClaudeToolDefaultRowUsesKnownEfforts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	rows, warnings := discoverWith(nil, "claude", defaultEnvironment())
	if len(warnings) != 0 {
		t.Fatal(warnings)
	}
	row := rows[0]
	if row.Label != "tool default" || !reflect.DeepEqual(row.Efforts, claudeEfforts) {
		t.Errorf("claude reserved row = %+v", row)
	}
}

func TestRowEffortsMarshalAsArray(t *testing.T) {
	rows, _ := discoverWith(&config.Config{Providers: map[string]config.ProviderConfig{
		"custom": {Profiles: map[string]config.ProviderProfile{"default": {Model: "model"}}},
	}}, "custom", defaultEnvironment())
	data, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	efforts, ok := decoded[0]["efforts"].([]any)
	if !ok || efforts == nil || len(efforts) != 0 {
		t.Fatalf("efforts = %#v", decoded[0]["efforts"])
	}
}
