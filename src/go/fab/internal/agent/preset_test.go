package agent

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// presetEnv isolates the system layer and every FAB_* registry variable, then
// writes the optional system and project config bodies. It returns the project
// config path. Mirrors the config package's isolateSystemConfig discipline.
func presetEnv(t *testing.T, system, project string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range configscope.DottedKeys() {
		t.Setenv("FAB_"+strings.ToUpper(strings.ReplaceAll(key, ".", "_")), "")
	}
	t.Setenv("FAB_PROVIDERS", "")
	if system != "" {
		dir := filepath.Join(home, ".fab-kit")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(system), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(project), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// captureWarnings runs fn with os.Stderr redirected (the loader resolves its warning
// stream at call time) and returns what was written.
func captureWarnings(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()
	fn()
	w.Close()
	data, _ := io.ReadAll(r)
	return string(data)
}

func resolveRoles(t *testing.T, cfg *config.Config) map[string]Profile {
	t.Helper()
	out := map[string]Profile{}
	for _, role := range RoleNames() {
		p, err := ResolveRole(cfg, role)
		if err != nil {
			t.Fatal(err)
		}
		out[role] = p
	}
	return out
}

func loadPreset(t *testing.T, path string) *config.Config {
	t.Helper()
	cfg, err := config.LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// TestPresetPrecedence is the table for the intake's required precedence cases.
func TestPresetPrecedence(t *testing.T) {
	codexWorkers := map[string]string{"doing": "codex", "review": "codex", "hydrate": "codex", "fast": "codex", "default": "claude", "operator": "claude"}
	tests := []struct {
		name    string
		system  string
		project string
		env     map[string]string
		wantCLI map[string]string
		warn    string
	}{
		{
			name:    "1 preset alone",
			project: "agent:\n  preset: codex-workers\n",
			wantCLI: codexWorkers,
		},
		{
			name:    "2 explicit field at the choosing layer beats the preset",
			project: "agent:\n  preset: codex-workers\n  workers: agy\n",
			wantCLI: map[string]string{"doing": "agy", "review": "agy", "hydrate": "agy", "fast": "agy", "default": "claude", "operator": "claude"},
		},
		{
			name:    "3 env-chosen preset beats system explicit fields",
			system:  "agent:\n  workers: claude\n",
			project: "{}\n",
			env:     map[string]string{"FAB_AGENT_PRESET": "codex-workers"},
			wantCLI: codexWorkers,
		},
		{
			name:    "4 system preset shadows project preset entirely",
			system:  "agent:\n  preset: codex-workers\n",
			project: "agent:\n  preset: codex-review\n",
			wantCLI: codexWorkers,
		},
		{
			name:    "5 unknown name warns and resolves as unset",
			project: "agent:\n  preset: nope\n",
			wantCLI: map[string]string{"doing": "claude", "review": "claude", "hydrate": "claude", "fast": "claude", "default": "claude", "operator": "claude"},
			warn:    `fab: warning: unknown agent preset "nope" (agent.preset from project); ignoring it`,
		},
		{
			name:    "6 reserved none at a higher layer cancels a lower preset",
			project: "agent:\n  preset: codex-workers\n",
			env:     map[string]string{"FAB_AGENT_PRESET": "none"},
			wantCLI: map[string]string{"doing": "claude", "review": "claude", "hydrate": "claude", "fast": "claude", "default": "claude", "operator": "claude"},
		},
		{
			name:    "7 empty env value is unset",
			project: "agent:\n  preset: codex-workers\n",
			env:     map[string]string{"FAB_AGENT_PRESET": ""},
			wantCLI: codexWorkers,
		},
		{
			name:    "8 explicit field in a layer above the chooser beats the preset",
			system:  "agent:\n  workers: agy\n",
			project: "agent:\n  preset: codex-workers\n",
			wantCLI: map[string]string{"doing": "agy", "review": "agy", "hydrate": "agy", "fast": "agy", "default": "claude", "operator": "claude"},
		},
		{
			name:    "9 project-defined preset",
			project: "agent_presets:\n  mine:\n    session: codex\nagent:\n  preset: mine\n",
			wantCLI: map[string]string{"doing": "claude", "review": "claude", "hydrate": "claude", "fast": "claude", "default": "codex", "operator": "codex"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := presetEnv(t, tt.system, tt.project)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			var cfg *config.Config
			stderr := captureWarnings(t, func() { cfg = loadPreset(t, path) })
			got := resolveRoles(t, cfg)
			for role, want := range tt.wantCLI {
				if got[role].CLI != want {
					t.Errorf("role %s cli = %q, want %q", role, got[role].CLI, want)
				}
			}
			if tt.warn == "" && stderr != "" {
				t.Errorf("unexpected stderr: %q", stderr)
			}
			if tt.warn != "" && !strings.Contains(stderr, tt.warn) {
				t.Errorf("stderr %q missing %q", stderr, tt.warn)
			}
		})
	}
}

// TestPresetUnknownEqualsUnset pins "resolution equals the unset case" byte for byte.
func TestPresetUnknownEqualsUnset(t *testing.T) {
	unset := resolveRoles(t, loadPreset(t, presetEnv(t, "", "{}\n")))
	var got map[string]Profile
	path := presetEnv(t, "", "agent:\n  preset: stale\n")
	captureWarnings(t, func() { got = resolveRoles(t, loadPreset(t, path)) })
	for role, want := range unset {
		if got[role] != want {
			t.Errorf("role %s: %+v != unset %+v", role, got[role], want)
		}
	}
}

// TestPresetCLISwapProvenance: codex-review re-derives review's model/effort from
// codex's own fills; source labels name the preset for cli and the codex fill for
// model/effort.
func TestPresetCLISwapProvenance(t *testing.T) {
	cfg := loadPreset(t, presetEnv(t, "", "agent:\n  preset: codex-review\n"))
	p, src, err := ResolveRoleWithSource(cfg, RoleReview, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	fill := defaultCLIs[cliCodex].Profiles[RoleReview]
	if p.CLI != cliCodex || p.Model != fill.Model || p.Effort != fill.Effort {
		t.Errorf("review = %+v, want codex with fills %+v", p, fill)
	}
	if src.CLI != "agent_presets.codex-review" {
		t.Errorf("source.cli = %q", src.CLI)
	}
	if src.Model != "clis.codex.profiles.review" || src.Effort != "clis.codex.profiles.review" {
		t.Errorf("model/effort sources = %q / %q", src.Model, src.Effort)
	}
	// Other roles untouched and still labelled by their knobs.
	_, doingSrc, _ := ResolveRoleWithSource(cfg, RoleDoing, Overrides{})
	if doingSrc.CLI != "built-in" {
		t.Errorf("doing source.cli = %q, want built-in", doingSrc.CLI)
	}
}

// TestPresetKnobSourceLabels: knob-supplied CLIs are labelled by the preset only when
// the preset actually supplied them; an explicit field at/above the choosing layer
// keeps its own label.
func TestPresetKnobSourceLabels(t *testing.T) {
	cfg := loadPreset(t, presetEnv(t, "", "agent:\n  preset: codex-workers\n"))
	_, src, _ := ResolveRoleWithSource(cfg, RoleDoing, Overrides{})
	if src.CLI != "agent_presets.codex-workers" {
		t.Errorf("preset-supplied workers label = %q", src.CLI)
	}
	cfg = loadPreset(t, presetEnv(t, "", "agent:\n  preset: codex-workers\n  workers: agy\n"))
	_, src, _ = ResolveRoleWithSource(cfg, RoleDoing, Overrides{})
	if src.CLI != "agent.workers" {
		t.Errorf("explicit workers label = %q, want agent.workers", src.CLI)
	}
}

// TestLoadWithPresetFlagEquivalence: an invocation-time preset behaves as the env
// layer's choice — equal to FAB_AGENT_PRESET, beaten by explicit env agent fields.
func TestLoadWithPresetFlagEquivalence(t *testing.T) {
	path := presetEnv(t, "agent:\n  workers: claude\n", "{}\n")
	viaFlag, err := config.LoadPathWithPreset(path, "codex-workers")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAB_AGENT_PRESET", "codex-workers")
	viaEnv := loadPreset(t, path)
	a, b := resolveRoles(t, viaFlag), resolveRoles(t, viaEnv)
	for role := range a {
		if a[role] != b[role] {
			t.Errorf("role %s: flag %+v != env %+v", role, a[role], b[role])
		}
	}
	if a[RoleDoing].CLI != "codex" {
		t.Errorf("flag preset did not win over system workers: %+v", a[RoleDoing])
	}

	// Explicit env agent field beats the flag-chosen preset.
	t.Setenv("FAB_AGENT_PRESET", "")
	t.Setenv("FAB_AGENT_WORKERS", "agy")
	viaFlag, _ = config.LoadPathWithPreset(path, "codex-workers")
	if got := resolveRoles(t, viaFlag)[RoleDoing].CLI; got != "agy" {
		t.Errorf("explicit env FAB_AGENT_WORKERS lost to the flag preset: %q", got)
	}

	// --preset none disables a configured preset.
	t.Setenv("FAB_AGENT_WORKERS", "")
	path = presetEnv(t, "", "agent:\n  preset: codex-workers\n")
	off, _ := config.LoadPathWithPreset(path, "none")
	if got := resolveRoles(t, off)[RoleDoing].CLI; got != "claude" {
		t.Errorf("--preset none left %q", got)
	}
}

// TestPresetNoneDefinitionIgnored: a definition named none is warned about and never
// selectable.
func TestPresetNoneDefinitionIgnored(t *testing.T) {
	path := presetEnv(t, "", "agent_presets:\n  none:\n    workers: codex\nagent:\n  preset: none\n")
	var cfg *config.Config
	stderr := captureWarnings(t, func() { cfg = loadPreset(t, path) })
	if !strings.Contains(stderr, `ignoring agent preset definition "none"`) {
		t.Errorf("stderr = %q", stderr)
	}
	if got := resolveRoles(t, cfg)[RoleDoing].CLI; got != "claude" {
		t.Errorf("none activated a preset: %q", got)
	}
	if _, ok := config.DefinedAgentPresets(cfg)["none"]; ok {
		t.Error("DefinedAgentPresets lists none")
	}
}

// TestLayersEffectiveMatchesLoader: `fab config show`'s tree (Layers.Effective) is the
// SAME spliced tree the loader resolves from.
func TestLayersEffectiveMatchesLoader(t *testing.T) {
	path := presetEnv(t, "", "agent:\n  preset: codex-workers\n")
	layers, err := config.LoadLayers(path)
	if err != nil {
		t.Fatal(err)
	}
	fromLayers, err := config.FromLayers(layers)
	if err != nil {
		t.Fatal(err)
	}
	if fromLayers.Agent.Workers != "codex" || fromLayers.Preset == nil || fromLayers.Preset.Name != "codex-workers" {
		t.Errorf("FromLayers = %+v preset %+v", fromLayers.Agent, fromLayers.Preset)
	}
	loaded := loadPreset(t, path)
	if loaded.Agent.Workers != fromLayers.Agent.Workers {
		t.Errorf("loader %q != layers %q", loaded.Agent.Workers, fromLayers.Agent.Workers)
	}
	// Raw layers stay unspliced for provenance.
	if _, ok := layers.Project["agent"].(map[string]any)["workers"]; ok {
		t.Error("raw project layer was mutated by the splice")
	}
}

// TestAgentPresetDefinitionSources: the highest contributing tier names each
// definition's source.
func TestAgentPresetDefinitionSources(t *testing.T) {
	path := presetEnv(t,
		"agent_presets:\n  codex-review:\n    description: tweaked\n",
		"agent_presets:\n  mine:\n    workers: agy\n")
	t.Setenv("FAB_AGENT_PRESETS", "{envmade: {session: codex}}")
	layers, err := config.LoadLayers(path)
	if err != nil {
		t.Fatal(err)
	}
	defs, src := config.AgentPresetDefinitions(layers)
	want := map[string]config.PresetSource{
		"claude":        config.PresetSourceBuiltIn,
		"codex-workers": config.PresetSourceBuiltIn,
		"codex-review":  config.PresetSourceSystem,
		"mine":          config.PresetSourceProject,
		"envmade":       config.PresetSourceEnv,
	}
	if len(defs) != len(want) {
		t.Errorf("defs = %v", defs)
	}
	for name, s := range want {
		if src[name] != s {
			t.Errorf("source[%s] = %q, want %q", name, src[name], s)
		}
	}
	if defs["codex-review"].Description != "tweaked" || defs["codex-review"].Profiles["review"].CLI != "codex" {
		t.Errorf("per-leaf merge lost: %+v", defs["codex-review"])
	}
}
