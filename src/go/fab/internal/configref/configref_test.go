package configref

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configvalue"
)

// TestFieldKeysMatchConfigscopeDottedKeys is the cycle-free registry parity
// lint. internal/config cannot import this package (configref → agent → config),
// so its generic environment walk consumes configscope.DottedKeys instead. This
// guard makes the leaf enumeration equivalent to the canonical registry rather
// than allowing a copied list to drift.
func TestFieldKeysMatchConfigscopeDottedKeys(t *testing.T) {
	want, err := FieldKeys()
	if err != nil {
		t.Fatalf("FieldKeys: %v", err)
	}
	got := configscope.DottedKeys()
	if !slices.Equal(got, want) {
		t.Fatalf("configscope dotted keys do not match configref registry:\n configscope: %v\n configref:   %v", got, want)
	}
}

func TestResolveKey(t *testing.T) {
	tests := []struct {
		key       string
		wantOK    bool
		wantField string
		wantOwner string
		wantKind  configvalue.Kind
	}{
		{key: "agent.workers", wantOK: true, wantField: "agent.workers", wantOwner: "agent.session", wantKind: configvalue.KindString},
		{key: "agent.profiles.review.model", wantOK: true, wantField: "agent.profiles", wantOwner: "agent.session", wantKind: configvalue.KindString},
		{key: "providers.codex.dispatch_command", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "providers.codex.profiles.review.effort", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "stage_hooks.apply.pre", wantOK: true, wantField: "stage_hooks", wantOwner: "stage_hooks", wantKind: configvalue.KindString},
		{key: "source_paths", wantOK: true, wantField: "source_paths", wantOwner: "source_paths", wantKind: configvalue.KindSequence},
		{key: "dispatch.column_width", wantOK: true, wantField: "dispatch.column_width", wantOwner: "dispatch.watchable", wantKind: configvalue.KindInt},
		{key: "project", wantOK: true, wantField: "project.name", wantOwner: "project.name", wantKind: configvalue.KindMapping},
		{key: "agent", wantOK: true, wantField: "agent.session", wantOwner: "agent.session", wantKind: configvalue.KindMapping},
		{key: "agent.profiles.review", wantOK: true, wantField: "agent.profiles", wantOwner: "agent.session", wantKind: configvalue.KindMapping},
		{key: "providers.codex", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindMapping},
		{key: "providers.codex.profiles", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindMapping},
		{key: "providers.codex.profiles.default", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindMapping},
		{key: "stage_hooks.apply", wantOK: true, wantField: "stage_hooks", wantOwner: "stage_hooks", wantKind: configvalue.KindMapping},
		{key: "agent.profile.review.model"},
		{key: "dispatch.colum_width"},
		{key: "agent.profiles.unknown.model"},
		{key: "providers.codex.profiles.unknown.model"},
		{key: "stage_hooks.unknown.pre"},
		{key: `"agent".workers`},
		{key: "providers.123.session_command"},
		{key: "providers.claude.v2.session_command"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok, err := ResolveKey(tt.key)
			if err != nil {
				t.Fatalf("ResolveKey: %v", err)
			}
			if ok != tt.wantOK {
				t.Fatalf("ResolveKey(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.Field.Key != tt.wantField || got.Owner.Key != tt.wantOwner || got.Kind != tt.wantKind {
				t.Fatalf("ResolveKey(%q) = field %q owner %q kind %q; want %q %q %q", tt.key, got.Field.Key, got.Owner.Key, got.Kind, tt.wantField, tt.wantOwner, tt.wantKind)
			}
		})
	}
}

func TestKeyedRenderUsesOwningSegment(t *testing.T) {
	agentText, err := RenderKey("agent.profiles.review.model")
	if err != nil {
		t.Fatalf("RenderKey: %v", err)
	}
	if !strings.Contains(agentText, "agent:") || !strings.Contains(agentText, "workers:") {
		t.Fatalf("agent descendant did not resolve to the owning agent segment:\n%s", agentText)
	}

	jsonText, err := RenderJSONKey("agent.workers")
	if err != nil {
		t.Fatalf("RenderJSONKey: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonText), &rows); err != nil {
		t.Fatalf("keyed JSON is invalid: %v", err)
	}
	var keys []string
	for _, row := range rows {
		keys = append(keys, row["key"].(string))
	}
	want := []string{"agent.session", "agent.workers", "agent.profiles"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("keyed JSON keys = %v, want %v", keys, want)
	}
}

func TestRegistryRowsDeclareValueKinds(t *testing.T) {
	fields, err := Fields()
	if err != nil {
		t.Fatalf("Fields: %v", err)
	}
	for _, field := range fields {
		if !configvalue.Valid(field.Kind) {
			t.Errorf("field %q has invalid kind %q", field.Key, field.Kind)
		}
	}
}
