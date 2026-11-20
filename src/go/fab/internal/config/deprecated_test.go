package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The deprecated `providers:` / `.provider` spellings are read as aliases of
// `clis:` / `.cli` (261120-f2ly). These tests pin the per-field fold, the
// cascade behaviour, the env alias and the key-mapping helpers.

func loadWith(t *testing.T, project, system string) *Config {
	t.Helper()
	home := isolateSystemConfig(t)
	if system != "" {
		writeSystemConfig(t, home, system)
	}
	cfg, err := Load(writeProjectConfig(t, project))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func TestDeprecatedProvidersTableReadsAsCLIs(t *testing.T) {
	warnings := captureWarnings(t)
	old := loadWith(t, `
providers:
  codex:
    interactive_command: codex
    profiles:
      review: {model: x, effort: high}
agent:
  profiles:
    review: {provider: codex}
`, "")
	cur := loadWith(t, `
clis:
  codex:
    interactive_command: codex
    profiles:
      review: {model: x, effort: high}
agent:
  profiles:
    review: {cli: codex}
`, "")
	if !reflect.DeepEqual(old, cur) {
		t.Errorf("old spelling decodes differently from the new one:\nold=%+v\nnew=%+v", old, cur)
	}
	if got, ok := old.GetCLI("codex"); !ok || got.Profiles["review"].Model != "x" {
		t.Errorf("clis.codex via providers: = %+v ok=%v", got, ok)
	}
	if p, _ := old.GetAgentProfile("review"); p.CLI != "codex" {
		t.Errorf("agent.profiles.review.cli via .provider = %q, want codex", p.CLI)
	}
	if w := warnings(); w != "" {
		t.Errorf("ordinary reads of the old spelling must be silent, got %q", w)
	}
}

func TestDeprecatedLegacyTiersProviderLeafFolds(t *testing.T) {
	cfg := loadWith(t, "agent:\n  tiers:\n    doing: {provider: kimi, model: m}\n", "")
	if p, ok := cfg.GetAgentProfile("doing"); !ok || p.CLI != "kimi" || p.Model != "m" {
		t.Errorf("agent.tiers.doing.provider = %+v ok=%v, want cli kimi / model m", p, ok)
	}
}

func TestDeprecatedMixedSpellingsMergePerLeafNewWins(t *testing.T) {
	cfg := loadWith(t, `
clis:
  codex:
    profiles:
      review: {model: new}
providers:
  codex:
    headless_command: old-exec
    profiles:
      review: {model: old, effort: high}
agent:
  profiles:
    review: {cli: claude, provider: codex, effort: low}
`, "")
	got, ok := cfg.GetCLI("codex")
	if !ok {
		t.Fatal("clis.codex missing")
	}
	if rp := got.Profiles["review"]; rp.Model != "new" || rp.Effort != "high" {
		t.Errorf("profiles.review = %+v, want {model: new, effort: high}", rp)
	}
	if got.HeadlessCommand != "old-exec" {
		t.Errorf("headless_command = %q, want the providers-only leaf carried", got.HeadlessCommand)
	}
	if p, _ := cfg.GetAgentProfile("review"); p.CLI != "claude" || p.Effort != "low" {
		t.Errorf("review profile = %+v, want cli claude (new wins) / effort low", p)
	}
}

func TestDeprecatedEmptyNewLeafFallsBackToOld(t *testing.T) {
	cfg := loadWith(t, `
clis:
  codex: {headless_command: ""}
providers:
  codex: {headless_command: old-exec}
agent:
  profiles:
    review: {cli: "", provider: codex}
`, "")
	if got, _ := cfg.GetCLI("codex"); got.HeadlessCommand != "old-exec" {
		t.Errorf("empty clis leaf must fall through to providers, got %q", got.HeadlessCommand)
	}
	if p, _ := cfg.GetAgentProfile("review"); p.CLI != "codex" {
		t.Errorf("empty .cli must fall through to .provider, got %q", p.CLI)
	}
}

func TestDeprecatedNativeFalsePresenceSurvivesOldSpelling(t *testing.T) {
	cfg := loadWith(t, "providers:\n  claude:\n    native: false\n", "")
	got, ok := cfg.GetCLI("claude")
	if !ok || !got.NativeSet || got.Native {
		t.Errorf("providers.claude.native: false = %+v ok=%v, want NativeSet && !Native", got, ok)
	}
}

func TestDeprecatedAcrossCascadeTiersMergesLikeSameSpelling(t *testing.T) {
	// System file still on `providers:`, project already on `clis:` — resolves as
	// two same-spelling layers would: system outranks project per leaf.
	mixed := loadWith(t,
		"clis:\n  codex:\n    headless_command: project-exec\n    interactive_command: project-tui\n",
		"providers:\n  codex:\n    headless_command: system-exec\n")
	same := loadWith(t,
		"clis:\n  codex:\n    headless_command: project-exec\n    interactive_command: project-tui\n",
		"clis:\n  codex:\n    headless_command: system-exec\n")
	if !reflect.DeepEqual(mixed, same) {
		t.Errorf("half-migrated cascade differs:\nmixed=%+v\nsame=%+v", mixed, same)
	}
	if got, _ := mixed.GetCLI("codex"); got.HeadlessCommand != "system-exec" || got.InteractiveCommand != "project-tui" {
		t.Errorf("codex = %+v, want system headless over project interactive", got)
	}
}

func TestDeprecatedLayersExposeOnlyTheNewShape(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, "providers:\n  a: {headless_command: h}\n")
	fabRoot := writeProjectConfig(t, "providers:\n  b: {headless_command: h}\nagent:\n  profiles:\n    doing: {provider: b}\n")
	layers, err := LoadLayers(filepath.Join(fabRoot, "project", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for name, layer := range map[string]map[string]any{"project": layers.Project, "system": layers.System, "effective": layers.Effective} {
		if _, has := layer["providers"]; has {
			t.Errorf("%s layer still carries providers: %v", name, layer)
		}
		if _, has := layer["clis"]; !has {
			t.Errorf("%s layer lost its table: %v", name, layer)
		}
	}
	doing := layers.Project["agent"].(map[string]any)["profiles"].(map[string]any)["doing"].(map[string]any)
	if doing["cli"] != "b" || doing["provider"] != nil {
		t.Errorf("project agent.profiles.doing = %v, want cli: b only", doing)
	}
}

func TestNormalizeDeprecatedDoesNotMutateInput(t *testing.T) {
	in := map[string]any{
		"providers": map[string]any{"codex": map[string]any{"native": true}},
		"agent":     map[string]any{"profiles": map[string]any{"doing": map[string]any{"provider": "codex"}}},
	}
	out := NormalizeDeprecated(in)
	if _, has := in["providers"]; !has {
		t.Error("input lost its providers key")
	}
	if _, has := in["agent"].(map[string]any)["profiles"].(map[string]any)["doing"].(map[string]any)["provider"]; !has {
		t.Error("input lost its .provider leaf")
	}
	if _, has := out["providers"]; has {
		t.Error("output still carries providers")
	}
	if same := NormalizeDeprecated(map[string]any{"clis": map[string]any{}}); same == nil {
		t.Error("a layer with no deprecated key must come back as-is")
	}
	if NormalizeDeprecated(nil) != nil {
		t.Error("nil layer must stay nil")
	}
}

func TestEnvDeprecatedProvidersAlias(t *testing.T) {
	isolateSystemConfig(t)
	warnings := captureWarnings(t)
	fabRoot := writeProjectConfig(t, "agent:\n  workers: claude\n")
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")

	t.Run("alias alone", func(t *testing.T) {
		t.Setenv("FAB_PROVIDERS", `{custom: {headless_command: "t"}}`)
		layers, err := LoadLayers(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if got := layers.EnvOrigins["clis"]; got != "FAB_PROVIDERS" {
			t.Errorf("origin = %q, want FAB_PROVIDERS", got)
		}
		cfg, _ := Load(fabRoot)
		if got, ok := cfg.GetCLI("custom"); !ok || got.HeadlessCommand != "t" {
			t.Errorf("clis.custom via FAB_PROVIDERS = %+v ok=%v", got, ok)
		}
	})

	t.Run("canonical wins per field when both are set", func(t *testing.T) {
		t.Setenv("FAB_PROVIDERS", `{custom: {headless_command: "old", interactive_command: "old-tui"}}`)
		t.Setenv("FAB_CLIS", `{custom: {headless_command: "new"}}`)
		layers, err := LoadLayers(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if got := layers.EnvOrigins["clis"]; got != "FAB_CLIS" {
			t.Errorf("origin = %q, want FAB_CLIS", got)
		}
		cfg, _ := Load(fabRoot)
		got, _ := cfg.GetCLI("custom")
		if got.HeadlessCommand != "new" || got.InteractiveCommand != "old-tui" {
			t.Errorf("custom = %+v, want headless new (FAB_CLIS) + interactive old-tui (alias fills the gap)", got)
		}
	})

	t.Run("malformed alias warns naming the variable and is skipped", func(t *testing.T) {
		t.Setenv("FAB_PROVIDERS", "oops")
		if _, err := Load(fabRoot); err != nil {
			t.Fatal(err)
		}
		if w := warnings(); w == "" || !strings.Contains(w, "$FAB_PROVIDERS") {
			t.Errorf("warning = %q, want one naming $FAB_PROVIDERS", w)
		}
	})
}

func TestCanonicalKeyAndTwin(t *testing.T) {
	for _, tt := range []struct {
		key, canonical string
		deprecated     bool
		twin           string
	}{
		{"providers", "clis", true, "clis"},
		{"providers.codex.profiles.review.model", "clis.codex.profiles.review.model", true, "clis.codex.profiles.review.model"},
		{"agent.profiles.review.provider", "agent.profiles.review.cli", true, "agent.profiles.review.cli"},
		{"agent.tiers.review.provider", "agent.profiles.review.cli", true, "agent.profiles.review.cli"},
		{"clis.codex.native", "clis.codex.native", false, "providers.codex.native"},
		{"agent.profiles.review.cli", "agent.profiles.review.cli", false, "agent.profiles.review.provider"},
		{"agent.profiles.review.model", "agent.profiles.review.model", false, ""},
		{"agent.session", "agent.session", false, ""},
		{"agent.profiles.review", "agent.profiles.review", false, ""},
	} {
		got, dep := CanonicalKey(tt.key)
		if got != tt.canonical || dep != tt.deprecated {
			t.Errorf("CanonicalKey(%q) = %q,%v want %q,%v", tt.key, got, dep, tt.canonical, tt.deprecated)
		}
		if twin := TwinKey(tt.key); twin != tt.twin {
			t.Errorf("TwinKey(%q) = %q, want %q", tt.key, twin, tt.twin)
		}
	}
}

func TestDeprecatedUsesListsLiveOldKeysSorted(t *testing.T) {
	layer := map[string]any{
		"providers": map[string]any{"a": map[string]any{"native": true}},
		"agent": map[string]any{"profiles": map[string]any{
			"review": map[string]any{"provider": "codex"},
			"doing":  map[string]any{"cli": "codex"},
			"fast":   map[string]any{"provider": ""},
		}},
	}
	want := []DeprecatedUse{
		{Old: "agent.profiles.review.provider", New: "agent.profiles.review.cli"},
		{Old: "providers", New: "clis"},
	}
	if got := DeprecatedUses(layer); !reflect.DeepEqual(got, want) {
		t.Errorf("DeprecatedUses = %v, want %v", got, want)
	}
	if got := DeprecatedUses(map[string]any{"clis": map[string]any{}}); got != nil {
		t.Errorf("clean layer reported %v", got)
	}
}
