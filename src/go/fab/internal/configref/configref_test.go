package configref

import (
	"encoding/json"
	"regexp"
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
		wantErr   bool
	}{
		{key: "agent.workers", wantOK: true, wantField: "agent.workers", wantOwner: "agent.session", wantKind: configvalue.KindString},
		{key: "agent.profiles.review.model", wantOK: true, wantField: "agent.profiles", wantOwner: "agent.session", wantKind: configvalue.KindString},
		{key: "providers.codex.headless_command", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "providers.codex.profiles.review.effort", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "stage_hooks.apply.pre", wantOK: true, wantField: "stage_hooks", wantOwner: "stage_hooks", wantKind: configvalue.KindString},
		{key: "source_paths", wantOK: true, wantField: "source_paths", wantOwner: "source_paths", wantKind: configvalue.KindSequence},
		{key: "dispatch.column_width", wantOK: true, wantField: "dispatch.column_width", wantOwner: "dispatch.mode", wantKind: configvalue.KindInt},
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
		{key: "providers.123.interactive_command", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "providers.true.interactive_command", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "providers.on.interactive_command", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "providers.-local.interactive_command", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "providers.测试.interactive_command", wantOK: true, wantField: "providers", wantOwner: "providers", wantKind: configvalue.KindString},
		{key: "providers.#local.interactive_command", wantErr: true},
		{key: "providers.local:dev.interactive_command", wantErr: true},
		{key: "providers. local.interactive_command", wantErr: true},
		{key: "providers.local\nname.interactive_command", wantErr: true},
		{key: "providers.claude.v2.interactive_command"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok, err := ResolveKey(tt.key)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "dotted config-key grammar") {
					t.Fatalf("ResolveKey(%q) error = %v, want dotted-key-grammar refusal", tt.key, err)
				}
				return
			}
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

func TestKeyedRenderAcceptsOpaqueProviderNames(t *testing.T) {
	for _, name := range []string{"123", "true", "on", "-local", "测试"} {
		key := "providers." + name + ".interactive_command"
		if _, err := RenderKey(key); err != nil {
			t.Errorf("RenderKey(%q): %v", key, err)
		}
	}
}

func TestRenderedRegistryGuidanceUsesExplain(t *testing.T) {
	text, err := Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(text, "fab config reference") {
		t.Fatalf("current registry guidance advertises the invisible alias:\n%s", text)
	}
	for _, want := range []string{"fab config explain", "fab config explain --json"} {
		if !strings.Contains(text, want) {
			t.Errorf("registry guidance is missing %q", want)
		}
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

// TestShortSegmentsCarryScopeAnnotationsAndRealExplainKeys is the 260809-wll4
// R6/R7 registry contract for the file-bound short form: every ShortSegment
// carries its row's [project|system|both] scope tag, exactly one
// `fab config explain <key>` pointer whose key is a REAL registry key (a pointer
// at an unresolvable key would strand the reader), and — for preference-class
// (scope both) rows — the `fab config set --system` machine-wide pointer. The
// lint pins tag/pointer presence at construction; this guard pins the pointer
// TARGETS (ResolveKey calls Fields, so the lint itself cannot).
func TestShortSegmentsCarryScopeAnnotationsAndRealExplainKeys(t *testing.T) {
	fields, err := Fields()
	if err != nil {
		t.Fatalf("Fields: %v", err)
	}
	explainRe := regexp.MustCompile(`fab config explain (\S+)`)
	short := 0
	for _, f := range fields {
		if f.ShortSegment == "" {
			continue
		}
		short++
		if tag := "[" + string(f.Scope) + "]"; !strings.Contains(f.ShortSegment, tag) {
			t.Errorf("field %q ShortSegment is missing its scope tag %q", f.Key, tag)
		}
		matches := explainRe.FindAllStringSubmatch(f.ShortSegment, -1)
		if len(matches) != 1 {
			t.Errorf("field %q ShortSegment carries %d explain pointers, want exactly 1", f.Key, len(matches))
		}
		for _, m := range matches {
			if _, ok, err := ResolveKey(m[1]); err != nil || !ok {
				t.Errorf("field %q ShortSegment points at %q, which does not resolve to a registry key (ok=%v err=%v)", f.Key, m[1], ok, err)
			}
		}
		if f.Scope == configscope.ScopeBoth && !strings.Contains(f.ShortSegment, "fab config set --system ") {
			t.Errorf("field %q is scope both — its ShortSegment must point at `fab config set --system <key> <value>`", f.Key)
		}
	}
	if short == 0 {
		t.Fatal("no ShortSegments — the file-bound renderers would render nothing")
	}
}
