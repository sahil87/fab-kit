package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// setupConfigRepo creates a temp fab/ repo with the given project config.yaml
// content, chdirs into it (for resolve.FabRoot), and isolates HOME at an empty
// fake home so the system layer is absent unless the test writes one. Returns the
// repo root and the fake home dir.
func setupConfigRepo(t *testing.T, projectYAML string) (repo, home string) {
	t.Helper()
	repo = t.TempDir()
	projectDir := filepath.Join(repo, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(projectYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	home = t.TempDir()
	// chdirTestEnv restores cwd + env on cleanup; HOME isolates the system layer.
	env := map[string]string{"HOME": home}
	for _, key := range configscope.DottedKeys() {
		env["FAB_"+strings.ToUpper(strings.ReplaceAll(key, ".", "_"))] = ""
	}
	chdirTestEnv(t, repo, env)
	return repo, home
}

func writeSystemConfig(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".fab-kit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runConfig(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// TestConfigShow_PrintsEffectiveConfig: `fab config show` prints the fully
// composed effective config (defaults beneath project beneath system) as YAML
// and exits 0.
func TestConfigShow_PrintsEffectiveConfig(t *testing.T) {
	_, home := setupConfigRepo(t, `
providers:
  claude:
    interactive_command: project-session
`)
	writeSystemConfig(t, home, `
providers:
  codex:
    headless_command: codex exec
`)
	out, err := runConfig(t, "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	// project value present, system-only provider merged in.
	if !strings.Contains(out, "project-session") {
		t.Errorf("show output missing project value:\n%s", out)
	}
	if !strings.Contains(out, "codex exec") {
		t.Errorf("show output missing system-layer value:\n%s", out)
	}
}

func TestConfigShow_PrintsBuiltInDefaults(t *testing.T) {
	setupConfigRepo(t, "")
	out, err := runConfig(t, "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}

	var effective map[string]any
	if err := yaml.Unmarshal([]byte(out), &effective); err != nil {
		t.Fatalf("config show output is not YAML: %v\n%s", err, out)
	}
	dispatch, ok := effective["dispatch"].(map[string]any)
	if !ok {
		t.Fatalf("config show output missing the built-in dispatch map:\n%s", out)
	}
	if got := dispatch["mode"]; got != "native" {
		t.Fatalf("dispatch.mode = %#v, want the pure built-in default %q\n%s", got, "native", out)
	}
}

func TestConfigShow_ComposesDerivedDefaultsAgainstLiveKnobs(t *testing.T) {
	setupConfigRepo(t, "agent:\n  workers: codex\n")
	out, err := runConfig(t, "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}

	var effective map[string]any
	if err := yaml.Unmarshal([]byte(out), &effective); err != nil {
		t.Fatalf("config show output is not YAML: %v\n%s", err, out)
	}
	agentMap, ok := effective["agent"].(map[string]any)
	if !ok {
		t.Fatalf("config show output missing the agent map:\n%s", out)
	}
	profiles, ok := agentMap["profiles"].(map[string]any)
	if !ok {
		t.Fatalf("config show output missing derived agent profiles:\n%s", out)
	}
	doing, ok := profiles["doing"].(map[string]any)
	if !ok {
		t.Fatalf("config show output missing the derived doing profile:\n%s", out)
	}
	if got := doing["provider"]; got != "codex" {
		t.Fatalf("agent.profiles.doing.provider = %#v, want knob-composed default %q\n%s", got, "codex", out)
	}
}

// TestConfigShow_RejectsTooManyArgs: show accepts one key but rejects two.
func TestConfigShow_RejectsExtraArgs(t *testing.T) {
	setupConfigRepo(t, "providers:\n  claude:\n    interactive_command: x\n")
	if _, err := runConfig(t, "show", "agent.workers", "extra"); err == nil {
		t.Error("`config show` should reject more than one key")
	}
	if _, err := runConfig(t, "show", "--origin", "agent.workers", "extra"); err == nil {
		t.Error("`config show --origin` should reject more than one key")
	}
}

// TestConfigShow_KeyedValueOriginAndUnknown: a keyed show prints the raw effective
// value, and keyed --origin lists the key's FULL STACK — one line per tier that
// defines it, the winner marked. Unknown keys still fail naming the key.
func TestConfigShow_KeyedValueOriginAndUnknown(t *testing.T) {
	repo, _ := setupConfigRepo(t, "agent:\n    workers: codex\n")
	out, err := runConfig(t, "show", "agent.workers")
	if err != nil {
		t.Fatalf("keyed show: %v", err)
	}
	if out != "codex\n" {
		t.Fatalf("keyed scalar output = %q, want raw value", out)
	}
	out, err = runConfig(t, "show", "agent.workers", "--origin")
	if err != nil {
		t.Fatalf("keyed show --origin: %v", err)
	}
	projectPath := filepath.Join(repo, "fab", "project", "config.yaml")
	want := "agent.workers = codex   # project " + projectPath + "  (effective)\n" +
		"agent.workers = claude  # default  (shadowed)\n"
	if out != want {
		t.Fatalf("keyed origin output = %q,\nwant %q", out, want)
	}
	for _, key := range []string{"agent.workerz", "providers.#local.interactive_command"} {
		if _, err := runConfig(t, "show", key); err == nil || !strings.Contains(err.Error(), key) {
			t.Fatalf("unknown/structurally invalid keyed show %q should name the key, got %v", key, err)
		}
	}
}

// TestConfigShowKey_FullStackListsEveryDefiningTier: the visibility fix — with
// the same key set at three tiers, the keyed listing shows all three in
// precedence order so a shadowed override is visible instead of inferred.
func TestConfigShowKey_FullStackListsEveryDefiningTier(t *testing.T) {
	repo, home := setupConfigRepo(t, "agent:\n    workers: project-worker\n")
	writeSystemConfig(t, home, "agent:\n    workers: system-worker\n")
	t.Setenv("FAB_AGENT_WORKERS", "env-worker")

	out, err := runConfig(t, "show", "agent.workers", "--origin")
	if err != nil {
		t.Fatalf("keyed show --origin: %v", err)
	}
	wantLines := []string{
		"agent.workers = env-worker      # env $FAB_AGENT_WORKERS  (effective)",
		"agent.workers = system-worker   # system " + filepath.Join(home, ".fab-kit", "config.yaml") + "  (shadowed)",
		"agent.workers = project-worker  # project " + filepath.Join(repo, "fab", "project", "config.yaml") + "  (shadowed)",
		"agent.workers = claude          # default  (shadowed)",
	}
	if got := strings.Split(strings.TrimRight(out, "\n"), "\n"); len(got) != len(wantLines) {
		t.Fatalf("full-stack listing = %d lines, want %d:\n%s", len(got), len(wantLines), out)
	}
	for i, line := range wantLines {
		if got := strings.Split(strings.TrimRight(out, "\n"), "\n")[i]; got != line {
			t.Errorf("line %d = %q, want %q", i, got, line)
		}
	}
}

// TestConfigShow_KeyedListOriginIsCompact: a list leaf renders in compact flow
// style in the stack listing (one readable line per tier).
func TestConfigShow_KeyedListOriginIsCompact(t *testing.T) {
	repo, _ := setupConfigRepo(t, "source_paths:\n    - src/\n    - scripts/\n")
	out, err := runConfig(t, "show", "source_paths", "--origin")
	if err != nil {
		t.Fatalf("keyed list show --origin: %v", err)
	}
	want := "source_paths = [src/, scripts/]  # project " + filepath.Join(repo, "fab", "project", "config.yaml") + "  (effective)\n"
	if out != want {
		t.Fatalf("keyed list origin output = %q, want compact flow value %q", out, want)
	}
}

func TestConfigShow_KeyedMapUsesPerLeafOrigins(t *testing.T) {
	_, home := setupConfigRepo(t, "agent:\n    profiles:\n        review:\n            model: project-model\n")
	writeSystemConfig(t, home, "agent:\n    profiles:\n        review:\n            effort: high\n")
	out, err := runConfig(t, "show", "agent.profiles.review", "--origin")
	if err != nil {
		t.Fatalf("keyed map --origin: %v", err)
	}
	for _, want := range []string{"agent.profiles.review.model = project-model", "agent.profiles.review.effort = high"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestConfigSetUnset_ProjectRoundTripAndNoop(t *testing.T) {
	repo, _ := setupConfigRepo(t, "# owned\nagent:\n    workers: claude # inline\n")
	path := filepath.Join(repo, "fab", "project", "config.yaml")
	out, err := runConfig(t, "set", "agent.workers", "codex")
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	if !strings.Contains(out, "Set agent.workers") {
		t.Fatalf("missing set confirmation: %q", out)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "workers: codex # inline") || !strings.Contains(string(data), "# owned") {
		t.Fatalf("set did not preserve comments\n%s", data)
	}
	if _, err := runConfig(t, "unset", "agent.workers"); err != nil {
		t.Fatalf("config unset: %v", err)
	}
	data, _ = os.ReadFile(path)
	live := strings.SplitN(string(data), "# >>> fab reference", 2)[0]
	if strings.Contains(live, "workers:") || !strings.Contains(live, "# inline") {
		t.Fatalf("unset did not remove only the live override\n%s", data)
	}
	out, err = runConfig(t, "unset", "agent.workers")
	if err != nil || !strings.Contains(out, "nothing to unset") {
		t.Fatalf("absent unset must be exit-zero with notice: out=%q err=%v", out, err)
	}
}

func TestConfigSetShow_LiteralDottedKeyDoesNotShadowNestedPath(t *testing.T) {
	const literal = "agent.workers: literal # unrelated dotted key"
	for _, tc := range []struct {
		name     string
		original string
	}{
		{"instead of nested block", literal + "\n"},
		{"alongside nested block", literal + "\nagent:\n    workers: claude\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := setupConfigRepo(t, tc.original)
			if _, err := runConfig(t, "set", "agent.workers", "codex"); err != nil {
				t.Fatalf("config set: %v", err)
			}
			out, err := runConfig(t, "show", "agent.workers")
			if err != nil {
				t.Fatalf("config show: %v", err)
			}
			if out != "codex\n" {
				t.Fatalf("keyed show = %q, want the nested value", out)
			}

			data, err := os.ReadFile(filepath.Join(repo, "fab", "project", "config.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(data), literal) != 1 {
				t.Fatalf("set changed the unrelated literal dotted key\n%s", data)
			}
		})
	}
}

func TestConfigUnset_LiteralDottedKeyDoesNotMatchNestedPath(t *testing.T) {
	const literal = "agent.workers: literal # unrelated dotted key"
	for _, tc := range []struct {
		name     string
		original string
		wantNoop bool
	}{
		{"instead of nested block", literal + "\n", true},
		{"alongside nested block", literal + "\nagent:\n    workers: codex\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := setupConfigRepo(t, tc.original)
			out, err := runConfig(t, "unset", "agent.workers")
			if err != nil {
				t.Fatalf("config unset: %v", err)
			}
			if tc.wantNoop && !strings.Contains(out, "nothing to unset") {
				t.Fatalf("literal-only unset should be a no-op, got %q", out)
			}

			data, err := os.ReadFile(filepath.Join(repo, "fab", "project", "config.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(data), literal) != 1 {
				t.Fatalf("unset changed the unrelated literal dotted key\n%s", data)
			}
			if tc.wantNoop && string(data) != tc.original {
				t.Fatalf("literal-only unset changed bytes\n--- got ---\n%s\n--- want ---\n%s", data, tc.original)
			}

			shown, err := runConfig(t, "show", "agent.workers")
			if err != nil {
				t.Fatalf("config show after unset: %v", err)
			}
			if shown != "claude\n" {
				t.Fatalf("keyed show after unset = %q, want inherited default", shown)
			}
		})
	}
}

func TestConfigSet_OpaqueProviderNamesAreEffective(t *testing.T) {
	for _, name := range []string{"123", "true", "on", "-local", "测试"} {
		t.Run(name, func(t *testing.T) {
			setupConfigRepo(t, "")
			key := "providers." + name + ".interactive_command"
			if _, err := runConfig(t, "set", key, "tool"); err != nil {
				t.Fatalf("config set %q: %v", key, err)
			}
			out, err := runConfig(t, "show", key)
			if err != nil {
				t.Fatalf("config show %q: %v", key, err)
			}
			if out != "tool\n" {
				t.Fatalf("config show %q = %q, want %q", key, out, "tool\n")
			}
		})
	}
}

func TestConfigSetUnset_ValidationAndSystemScope(t *testing.T) {
	_, home := setupConfigRepo(t, "agent:\n    workers: claude\n")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"set", "agent.workerz", "codex"}, "agent.workerz"},
		{[]string{"set", "providers.#local.interactive_command", "tool"}, "dotted config-key grammar"},
		{[]string{"set", "agent.workers", "{bad: kind}"}, "single-line YAML scalar"},
		{[]string{"set", "agent.workers", "codex # note"}, "must not contain a YAML comment"},
		{[]string{"set", "source_paths", "[src/]"}, "scalar leaf"},
		{[]string{"set", "source_paths", "[src/]", "--system"}, "project scope"},
		{[]string{"set", "agent.workers", ""}, "fab config unset agent.workers"},
		{[]string{"set", "agent.workers", "   "}, "empty value"},
	} {
		if _, err := runConfig(t, tc.args...); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: want error containing %q, got %v", tc.args, tc.want, err)
		}
	}
	if _, err := runConfig(t, "set", "agent.workers", "codex", "--system"); err != nil {
		t.Fatalf("system set: %v", err)
	}
	systemPath := filepath.Join(home, ".fab-kit", "config.yaml")
	data, err := os.ReadFile(systemPath)
	if err != nil || !strings.Contains(string(data), "\n  workers: codex") {
		t.Fatalf("system set did not create the expected file: err=%v\n%s", err, data)
	}
	if _, err := runConfig(t, "unset", "agent.workers", "--system"); err != nil {
		t.Fatalf("system unset: %v", err)
	}
}

// TestConfigSet_EmptyValueRefusedWithoutWriting: the refusal is a guard, not a
// post-write complaint — the file is untouched and the message names `unset`. It
// tests the PARSED value, not the argv string, so the QUOTED-empty YAML forms and
// an explicit null are refused too: each writes a leaf empty-skip can never honor.
func TestConfigSet_EmptyValueRefusedWithoutWriting(t *testing.T) {
	for _, rawValue := range []string{"", "   ", `""`, "''", "null", "~"} {
		t.Run("value "+rawValue, func(t *testing.T) {
			repo, _ := setupConfigRepo(t, "agent:\n    workers: claude\n")
			path := filepath.Join(repo, "fab", "project", "config.yaml")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			out, err := runConfig(t, "set", "agent.workers", rawValue)
			if err == nil {
				t.Fatalf("an empty value must be refused, got output %q", out)
			}
			if !strings.Contains(err.Error(), "fab config unset agent.workers") {
				t.Errorf("the refusal must name the intended verb: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Errorf("a refused set must not touch the file\n--- before ---\n%s\n--- after ---\n%s", before, after)
			}
		})
	}
}

// TestConfigLoaderWarningsAreNotDuplicated: every `fab config` invocation resolves
// the read model ONCE. The surfaces used to load it twice (LoadLayers, then a
// second LoadPath behind the defaults projection), which printed each fail-open
// loader warning twice — a scope-pruned system key looked like two distinct
// problems. The defaults projection now consumes the already-loaded layers.
func TestConfigLoaderWarningsAreNotDuplicated(t *testing.T) {
	const warning = "ignoring project-scoped field"
	for _, args := range [][]string{
		{"show"},
		{"show", "--origin"},
		{"show", "agent.workers", "--origin"},
		{"set", "agent.workers", "codex"},
		{"unset", "agent.session"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, home := setupConfigRepo(t, "")
			// A project-scoped key in the system file is pruned with a warning on
			// every load of the cascade — the cheapest observable load counter.
			writeSystemConfig(t, home, "test_paths:\n  - '**/*_test.go'\n")

			var err error
			stderr := captureStderr(t, func() { _, err = runConfig(t, args...) })
			if err != nil {
				t.Fatalf("config %v: %v", args, err)
			}
			if got := strings.Count(stderr, warning); got != 1 {
				t.Errorf("loader warning printed %d times, want exactly 1 (one read-model load per invocation):\n%s", got, stderr)
			}
		})
	}
}

// captureStderr redirects os.Stderr — where internal/config writes its fail-open
// loader warnings — for the duration of fn, and returns what was written there.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prev := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = prev }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestConfigSet_ShadowWarning: a write a higher tier shadows still succeeds
// (exit 0) but says so, naming the tier that wins. Writing --system while only the
// PROJECT file defines the key is not shadowed — the system tier outranks it.
func TestConfigSet_ShadowWarning(t *testing.T) {
	t.Run("project write shadowed by system", func(t *testing.T) {
		_, home := setupConfigRepo(t, "")
		systemPath := filepath.Join(home, ".fab-kit", "config.yaml")
		writeSystemConfig(t, home, "agent:\n  workers: system-worker\n")

		out, err := runConfig(t, "set", "agent.workers", "codex")
		if err != nil {
			t.Fatalf("config set: %v", err)
		}
		if !strings.Contains(out, "Set agent.workers") {
			t.Errorf("the write itself must succeed: %q", out)
		}
		want := "fab: warning: agent.workers is shadowed by system " + systemPath
		if !strings.Contains(out, want) {
			t.Errorf("missing shadow warning %q in:\n%s", want, out)
		}
	})

	t.Run("project write shadowed by environment", func(t *testing.T) {
		setupConfigRepo(t, "")
		t.Setenv("FAB_AGENT_WORKERS", "env-worker")

		out, err := runConfig(t, "set", "agent.workers", "codex")
		if err != nil {
			t.Fatalf("config set: %v", err)
		}
		if !strings.Contains(out, "fab: warning: agent.workers is shadowed by env $FAB_AGENT_WORKERS") {
			t.Errorf("missing environment shadow warning:\n%s", out)
		}
	})

	t.Run("system write over a project value is not shadowed", func(t *testing.T) {
		setupConfigRepo(t, "agent:\n    workers: project-worker\n")

		out, err := runConfig(t, "set", "agent.workers", "codex", "--system")
		if err != nil {
			t.Fatalf("config set --system: %v", err)
		}
		if strings.Contains(out, "shadowed") {
			t.Errorf("the system tier outranks the project file — no warning expected:\n%s", out)
		}
	})

	t.Run("fail-open when the read model cannot be resolved", func(t *testing.T) {
		// A project file that does not parse makes LoadLayers fail. A --system write
		// does not touch it, so the write must still succeed — the notice is
		// best-effort and never fails a completed mutation.
		setupConfigRepo(t, "this: is: not: valid: yaml: [[[\n")

		out, err := runConfig(t, "set", "agent.workers", "codex", "--system")
		if err != nil {
			t.Fatalf("an unresolvable read model must not fail the write: %v", err)
		}
		if strings.Contains(out, "shadowed") {
			t.Errorf("no tier could be resolved, so no shadow warning is possible:\n%s", out)
		}
	})
}

// TestConfigShowOrigin_DrillDownIsKnobAware: the composed agent.profiles rows are
// derived from the depth knob and the provider's fills, so they must report the
// provider the knob actually selects — not the nil-config `claude` the registry
// default carries.
func TestConfigShowOrigin_DrillDownIsKnobAware(t *testing.T) {
	setupConfigRepo(t, "agent:\n    workers: codex\n")

	out, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	for _, want := range []string{
		"agent.profiles.doing.provider = codex",     // Tier-2 role: governed by agent.workers
		"agent.profiles.operator.provider = claude", // Tier-1 role: agent.session is unset
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing knob-aware drill-down row %q:\n%s", want, out)
		}
	}
}

// TestConfigShow_RoleProviderOverrideComposesThatProvidersFills: the chimera
// regression, end to end. A per-role provider override (system config
// `agent.profiles.operator.provider: codex`) used to compose with claude's
// operator fill on the model/effort leaves — a row no resolution path produces
// and one that disagrees with `fab agent operator -o yaml`. The derived fills
// must come from the overridden provider, while keyed --origin keeps the
// provider leaf's provenance split: the override (effective) over the
// knob-derived built-in (shadowed).
func TestConfigShow_RoleProviderOverrideComposesThatProvidersFills(t *testing.T) {
	_, home := setupConfigRepo(t, "")
	writeSystemConfig(t, home, "agent:\n    profiles:\n        operator:\n            provider: codex\n")

	// Derive codex's operator fill rather than restating model strings (a model
	// bump must not have to touch this test).
	knobbed, err := configref.DefaultsMapFor(&config.Config{Agent: config.AgentConfig{Session: "codex"}})
	if err != nil {
		t.Fatalf("DefaultsMapFor(session knob): %v", err)
	}
	oracle := knobbed["agent"].(map[string]any)["profiles"].(map[string]any)["operator"].(map[string]any)
	wantModel, _ := oracle["model"].(string)
	wantEffort, _ := oracle["effort"].(string)
	if wantModel == "" || wantEffort == "" {
		t.Fatalf("oracle broke: codex ships no operator fill (model %q, effort %q)", wantModel, wantEffort)
	}

	out, err := runConfig(t, "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	var effective map[string]any
	if err := yaml.Unmarshal([]byte(out), &effective); err != nil {
		t.Fatalf("config show output is not YAML: %v\n%s", err, out)
	}
	agentMap, ok := effective["agent"].(map[string]any)
	if !ok {
		t.Fatalf("config show output missing the agent map:\n%s", out)
	}
	profiles, ok := agentMap["profiles"].(map[string]any)
	if !ok {
		t.Fatalf("config show output missing derived agent profiles:\n%s", out)
	}
	operator, ok := profiles["operator"].(map[string]any)
	if !ok {
		t.Fatalf("config show output missing the composed operator profile:\n%s", out)
	}
	if got := operator["provider"]; got != "codex" {
		t.Errorf("agent.profiles.operator.provider = %#v, want the override codex\n%s", got, out)
	}
	if got := operator["model"]; got != wantModel {
		t.Errorf("agent.profiles.operator.model = %#v, want the overridden provider's fill %q (the chimera regression)\n%s", got, wantModel, out)
	}
	if got := operator["effort"]; got != wantEffort {
		t.Errorf("agent.profiles.operator.effort = %#v, want the overridden provider's fill %q\n%s", got, wantEffort, out)
	}

	providerStack, err := runConfig(t, "show", "agent.profiles.operator.provider", "--origin")
	if err != nil {
		t.Fatalf("keyed show --origin (provider): %v", err)
	}
	for _, want := range []string{
		"agent.profiles.operator.provider = codex",  // the system override wins…
		"agent.profiles.operator.provider = claude", // …shadowing the knob-derived built-in
	} {
		if !strings.Contains(providerStack, want) {
			t.Errorf("provider stack missing %q:\n%s", want, providerStack)
		}
	}

	modelStack, err := runConfig(t, "show", "agent.profiles.operator.model", "--origin")
	if err != nil {
		t.Fatalf("keyed show --origin (model): %v", err)
	}
	wantLine := "agent.profiles.operator.model = " + wantModel
	if !strings.Contains(modelStack, wantLine) || !strings.Contains(modelStack, "default  (effective)") {
		t.Errorf("model stack missing %q as the effective default line:\n%s", wantLine, modelStack)
	}
}

// TestConfigShowOrigin_AllEmptyMapDoesNotClaimTheNode: a tier whose subtree is
// entirely empty leaves defines NOTHING — the merge drops such a map wholesale, so
// provenance must too. With the emptiness test applied shallowly the system tier's
// `{claude: {interactive_command: null}}` looked like a defining map, hid the tier
// below it, and the listing reported a key no tier defines while the merge was
// resolving the project's value.
func TestConfigShowOrigin_AllEmptyMapDoesNotClaimTheNode(t *testing.T) {
	repo, home := setupConfigRepo(t, "providers: replaced-wholesale\n")
	writeSystemConfig(t, home, "providers:\n    claude:\n        interactive_command: null\n")

	out, err := runConfig(t, "show", "providers", "--origin")
	if err != nil {
		t.Fatalf("keyed show --origin: %v", err)
	}
	projectPath := filepath.Join(repo, "fab", "project", "config.yaml")
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.HasPrefix(first, "providers = replaced-wholesale ") ||
		!strings.HasSuffix(first, "# project "+projectPath+"  (effective)") {
		t.Errorf("first line = %q, want the project scalar as the effective tier", first)
	}
	if strings.Contains(out, "no tier defines this key") {
		t.Errorf("the project tier defines this key — the all-empty system map must not hide it:\n%s", out)
	}
}

// TestConfigShowOrigin_DefaultTierIsTheBuiltIn: the composed agent.profiles rows
// are resolved against the live config for the DEPTH KNOB (knob-awareness), which
// must not drag the user's own agent.profiles entry into the defaults tier. The
// stack listing is the visible symptom: a pinned model must appear on the project
// line only, with the built-in it shadows on the `default` line.
func TestConfigShowOrigin_DefaultTierIsTheBuiltIn(t *testing.T) {
	repo, _ := setupConfigRepo(t, "agent:\n    profiles:\n        review:\n            model: my-pinned-model\n")

	// Derive the expected built-in rather than restating a model string (a model
	// bump must not have to touch this test).
	builtin, err := configref.DefaultsMapFor(nil)
	if err != nil {
		t.Fatalf("DefaultsMapFor(nil): %v", err)
	}
	wantDefault := builtin["agent"].(map[string]any)["profiles"].(map[string]any)["review"].(map[string]any)["model"]

	out, err := runConfig(t, "show", "agent.profiles.review.model", "--origin")
	if err != nil {
		t.Fatalf("keyed show --origin: %v", err)
	}
	projectPath := filepath.Join(repo, "fab", "project", "config.yaml")
	got := strings.Split(strings.TrimRight(out, "\n"), "\n")
	want := []struct{ value, origin string }{
		{"my-pinned-model", "# project " + projectPath + "  (effective)"},
		{wantDefault.(string), "# default  (shadowed)"},
	}
	if len(got) != len(want) {
		t.Fatalf("stack listing = %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i, w := range want {
		// The column padding between the value and the `#` varies with the values,
		// so each line is matched by its two halves.
		if !strings.HasPrefix(got[i], "agent.profiles.review.model = "+w.value+" ") || !strings.HasSuffix(got[i], w.origin) {
			t.Errorf("line %d = %q, want value %q with origin %q", i, got[i], w.value, w.origin)
		}
	}
}

// TestConfigUnset_NoopNamesTheLiveTier: unsetting a key the target file does not
// carry stays an exit-zero no-op, and the notice says where the key IS live plus
// the command that would remove it.
func TestConfigUnset_NoopNamesTheLiveTier(t *testing.T) {
	t.Run("live in the system file", func(t *testing.T) {
		_, home := setupConfigRepo(t, "")
		systemPath := filepath.Join(home, ".fab-kit", "config.yaml")
		writeSystemConfig(t, home, "agent:\n  workers: system-worker\n")

		out, err := runConfig(t, "unset", "agent.workers")
		if err != nil {
			t.Fatalf("config unset: %v", err)
		}
		if !strings.Contains(out, "nothing to unset") {
			t.Errorf("the no-op notice must survive: %q", out)
		}
		if !strings.Contains(out, "live in system "+systemPath) ||
			!strings.Contains(out, "use: fab config unset agent.workers --system") {
			t.Errorf("missing live-tier notice:\n%s", out)
		}
	})

	t.Run("live in the environment", func(t *testing.T) {
		setupConfigRepo(t, "")
		t.Setenv("FAB_AGENT_WORKERS", "env-worker")

		out, err := runConfig(t, "unset", "agent.workers")
		if err != nil {
			t.Fatalf("config unset: %v", err)
		}
		if !strings.Contains(out, "live in env $FAB_AGENT_WORKERS") ||
			!strings.Contains(out, "unset cannot remove") {
			t.Errorf("missing environment live-tier notice:\n%s", out)
		}
	})

	t.Run("only the built-in default supplies it", func(t *testing.T) {
		setupConfigRepo(t, "")

		out, err := runConfig(t, "unset", "agent.workers")
		if err != nil {
			t.Fatalf("config unset: %v", err)
		}
		if strings.Contains(out, "live in") {
			t.Errorf("a default-only key needs no live-tier notice:\n%s", out)
		}
	})
}

func TestConfigSetUnset_ExactArgs(t *testing.T) {
	setupConfigRepo(t, "")
	for _, args := range [][]string{{"set", "agent.workers"}, {"set", "agent.workers", "codex", "extra"}, {"unset"}, {"unset", "agent.workers", "extra"}} {
		if _, err := runConfig(t, args...); err == nil {
			t.Errorf("%v should fail exact-argument validation", args)
		}
	}
}

// TestConfigShowOrigin_Provenance: --origin annotates each field with its
// provenance — project path for a project-set field, system path for a
// system-only field, and `default` for an unset field with a built-in default.
func TestConfigShowOrigin_Provenance(t *testing.T) {
	repo, home := setupConfigRepo(t, `
providers:
  claude:
    interactive_command: project-session
`)
	writeSystemConfig(t, home, `
agent:
  profiles:
    newbie:
      model: sys-model
`)
	out, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	projectPath := filepath.Join(repo, "fab", "project", "config.yaml")
	systemPath := filepath.Join(home, ".fab-kit", "config.yaml")

	assertOriginLine := func(keyValSubstr, wantOrigin string) {
		t.Helper()
		found := false
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, keyValSubstr) {
				found = true
				if !strings.Contains(line, "# "+wantOrigin) {
					t.Errorf("line %q: want origin %q\nfull:\n%s", line, wantOrigin, out)
				}
			}
		}
		if !found {
			t.Errorf("no --origin line contained %q:\n%s", keyValSubstr, out)
		}
	}

	// A project-set field shows the project path.
	assertOriginLine("providers.claude.interactive_command = project-session", projectPath)
	// A system-only field shows the system path (per-key map drill-down).
	assertOriginLine("agent.profiles.newbie.model = sys-model", systemPath)
	// An unset field with a built-in default shows `default`. The doing role's
	// model is such a field, and since 260806-j9nh it lives on the PROVIDER
	// (providers.claude.profiles.doing) rather than the agent side.
	assertOriginLine("providers.claude.profiles.doing.model =", "default")
	// The depth knobs are built-in defaults too. BOTH must appear: `agent` carries
	// three default-bearing registry rows (session, workers, profiles), so this
	// pins that the default subtree MERGES every matching row rather than stopping
	// at the first one.
	assertOriginLine("agent.session = claude", "default")
	assertOriginLine("agent.workers = claude", "default")
}

// TestConfigShowOrigin_TypoSurfacesAsDefault: a typo'd override (agent.profles)
// does not land, so the field the user MEANT to set shows origin `default` — the
// git-config-show-origin value the intake calls out. The misspelled key is
// simply an unknown key (ignored), so the doing role's model stays at its
// built-in default (which since 260806-j9nh lives on the claude provider).
func TestConfigShowOrigin_TypoSurfacesAsDefault(t *testing.T) {
	_, home := setupConfigRepo(t, `
agent:
  profles:          # typo — should have been "profiles"
    doing:
      model: i-meant-to-set-this
`)
	_ = home
	out, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	// The intended field is untouched — it shows the built-in default, alerting
	// the user their override did not take.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "providers.claude.profiles.doing.model =") {
			if !strings.Contains(line, "# default") {
				t.Errorf("typo'd override should leave the doing role's model at default, got: %q", line)
			}
			if strings.Contains(line, "i-meant-to-set-this") {
				t.Errorf("typo'd (misspelled) key must not take effect: %q", line)
			}
		}
	}
}

// TestConfigShowOrigin_HigherLayerScalarReplacesSubtree: when a higher-precedence
// layer replaces a map with a scalar, --origin must honor config.MergeLayers precedence and
// render the node as a single leaf at that layer's origin — NOT recurse into the
// lower layers' map keys. Here the project sets `providers: oops` (a scalar) while
// the built-in default provides a `providers` MAP; the effective value is the
// project scalar, so --origin must show `providers = oops # <project>` and emit no
// `providers.claude.*` leaves attributed to default. Regression for the flattenOrigin
// map-vs-scalar precedence bug.
func TestConfigShowOrigin_HigherLayerScalarReplacesSubtree(t *testing.T) {
	repo, _ := setupConfigRepo(t, "providers: oops\n")
	out, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	projectPath := filepath.Join(repo, "fab", "project", "config.yaml")

	sawProvidersLeaf := false
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "providers.") {
			t.Errorf("scalar project override must not recurse into default map keys, got: %q\nfull:\n%s", line, out)
		}
		if strings.Contains(line, "providers = oops") {
			sawProvidersLeaf = true
			if !strings.Contains(line, "# "+projectPath) {
				t.Errorf("providers leaf should show the project origin, got: %q", line)
			}
		}
	}
	if !sawProvidersLeaf {
		t.Errorf("expected a single `providers = oops` leaf at the project origin:\n%s", out)
	}
}

func TestConfigShow_EnvironmentEffectiveAndOrigins(t *testing.T) {
	setupConfigRepo(t, `
agent:
  session: claude
  workers: claude
`)
	t.Setenv("FAB_AGENT_SESSION", "agy")
	t.Setenv("FAB_AGENT_WORKERS", "codex")

	plain, err := runConfig(t, "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(plain, "session: agy") || !strings.Contains(plain, "workers: codex") {
		t.Errorf("plain show must include env-effective values:\n%s", plain)
	}

	origin, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	for keyValue, variable := range map[string]string{
		"agent.session = agy":   "$FAB_AGENT_SESSION",
		"agent.workers = codex": "$FAB_AGENT_WORKERS",
	} {
		found := false
		for _, line := range strings.Split(origin, "\n") {
			if strings.Contains(line, keyValue) {
				found = true
				if !strings.Contains(line, "# "+variable) {
					t.Errorf("origin line %q does not name %s", line, variable)
				}
			}
		}
		if !found {
			t.Errorf("origin output missing %q:\n%s", keyValue, origin)
		}
	}
}

func TestConfigShowOrigin_EnvironmentMapUsesRowVariable(t *testing.T) {
	setupConfigRepo(t, "agent:\n  profiles:\n    review: {model: project-model}\n")
	t.Setenv("FAB_AGENT_PROFILES", "{review: {provider: codex, effort: high}}")

	out, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	for _, leaf := range []string{
		"agent.profiles.review.provider = codex",
		"agent.profiles.review.effort = high",
	} {
		found := false
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, leaf) {
				found = true
				if !strings.Contains(line, "# $FAB_AGENT_PROFILES") {
					t.Errorf("map-valued env leaf has wrong origin: %q", line)
				}
			}
		}
		if !found {
			t.Errorf("origin output missing %q:\n%s", leaf, out)
		}
	}
	if !strings.Contains(out, "agent.profiles.review.model = project-model") {
		t.Errorf("env map must preserve non-conflicting project leaf:\n%s", out)
	}
}

// TestConfigShowOrigin_EnvironmentNullFallsThrough: an explicit environment null
// is EMPTY, so it neither wins nor blocks — the project value stays effective and
// the environment contributes no origin.
func TestConfigShowOrigin_EnvironmentNullFallsThrough(t *testing.T) {
	setupConfigRepo(t, "agent:\n  workers: project-worker\n")
	t.Setenv("FAB_AGENT_WORKERS", "null")

	plain, err := runConfig(t, "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(plain, "workers: project-worker") {
		t.Fatalf("an empty env override must fall through to the project value:\n%s", plain)
	}

	origin, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	for _, line := range strings.Split(origin, "\n") {
		if strings.Contains(line, "agent.workers =") {
			if !strings.Contains(line, "agent.workers = project-worker") || strings.Contains(line, "$FAB_AGENT_WORKERS") {
				t.Fatalf("an empty env override must not shadow or claim provenance, got %q", line)
			}
			return
		}
	}
	t.Fatalf("origin output missing agent.workers:\n%s", origin)
}

// TestConfigShowKey_EmptyValuesFallThrough: the read model's empty-skip rule, on
// the keyed surface — a null environment value and an empty project list both
// fall through instead of resolving as the effective value.
func TestConfigShowKey_EmptyValuesFallThrough(t *testing.T) {
	t.Run("environment null", func(t *testing.T) {
		setupConfigRepo(t, "agent:\n  workers: project-worker\n")
		t.Setenv("FAB_AGENT_WORKERS", "null")

		plain, err := runConfig(t, "show", "agent.workers")
		if err != nil {
			t.Fatalf("config show agent.workers: %v", err)
		}
		if plain != "project-worker\n" {
			t.Fatalf("keyed show = %q, want the project value (the env null falls through)", plain)
		}

		origin, err := runConfig(t, "show", "agent.workers", "--origin")
		if err != nil {
			t.Fatalf("config show agent.workers --origin: %v", err)
		}
		if strings.Contains(origin, "FAB_AGENT_WORKERS") {
			t.Fatalf("an empty env value must define no tier in the stack listing: %q", origin)
		}
		if !strings.Contains(origin, "agent.workers = project-worker") || !strings.Contains(origin, "(effective)") {
			t.Fatalf("keyed stack lost the effective project value: %q", origin)
		}
	})

	t.Run("empty project sequence", func(t *testing.T) {
		setupConfigRepo(t, "source_paths: []\n")

		plain, err := runConfig(t, "show", "source_paths")
		if err != nil {
			t.Fatalf("config show source_paths: %v", err)
		}
		if plain != "null\n" {
			t.Fatalf("empty sequence = %q, want null (it defines nothing, and no lower tier does either)", plain)
		}

		origin, err := runConfig(t, "show", "source_paths", "--origin")
		if err != nil {
			t.Fatalf("config show source_paths --origin: %v", err)
		}
		if !strings.Contains(origin, "no tier defines this key") {
			t.Fatalf("empty sequence should report that no tier defines the key: %q", origin)
		}
	})

	t.Run("empty environment mapping", func(t *testing.T) {
		setupConfigRepo(t, "")
		t.Setenv("FAB_PROVIDERS", "{custom: {}}")

		plain, err := runConfig(t, "show", "providers.custom")
		if err != nil {
			t.Fatalf("config show empty mapping: %v", err)
		}
		if plain != "null\n" {
			t.Fatalf("empty mapping output = %q, want null (an all-empty mapping defines nothing)", plain)
		}
	})
}

// TestConfigShowKey_UsesNearestReplacedAncestorOrigin: a higher tier replacing a
// map ancestor with a scalar nulls the descendant, and the keyed listing reports
// the REPLACING tier — the provenance a user needs to understand the null.
func TestConfigShowKey_UsesNearestReplacedAncestorOrigin(t *testing.T) {
	repo, _ := setupConfigRepo(t, "providers: oops\n")

	plain, err := runConfig(t, "show", "providers.claude.interactive_command")
	if err != nil {
		t.Fatalf("config show descendant: %v", err)
	}
	if plain != "null\n" {
		t.Fatalf("ancestor replacement did not null the descendant: %q", plain)
	}

	origin, err := runConfig(t, "show", "providers.claude.interactive_command", "--origin")
	if err != nil {
		t.Fatalf("config show descendant --origin: %v", err)
	}
	want := "providers.claude.interactive_command = null  # project " + filepath.Join(repo, "fab", "project", "config.yaml") + "  (effective)\n"
	if origin != want {
		t.Fatalf("descendant lost the nearest replaced ancestor's provenance: got %q, want %q", origin, want)
	}
}

// TestConfigInitSystem_WritesScaffoldAndRefusesOverwrite: `fab config init
// --system` writes the ~/.fab-kit/config.yaml scaffold (only system/both fields,
// all commented → parses as inert), and a second run refuses to overwrite.
func TestConfigInitSystem_WritesScaffoldAndRefusesOverwrite(t *testing.T) {
	_, home := setupConfigRepo(t, "providers:\n  claude:\n    interactive_command: x\n")

	out, err := runConfig(t, "init", "--system")
	if err != nil {
		t.Fatalf("config init --system: %v", err)
	}
	if !strings.Contains(out, "Wrote system config scaffold") {
		t.Errorf("expected a write confirmation, got: %q", out)
	}
	path := filepath.Join(home, ".fab-kit", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("scaffold not written: %v", err)
	}
	scaffold := string(data)

	// The scaffold documents ONLY the system/both fields (agent.profiles, providers)
	// and none of the project-scoped fields.
	for _, want := range []string{"providers", "agent.profiles"} {
		if !strings.Contains(scaffold, want) {
			t.Errorf("scaffold must document the system-overridable field %q", want)
		}
	}
	for _, absent := range []string{"source_paths:", "test_paths:", "true_impact_exclude:", "fab_version:"} {
		if strings.Contains(scaffold, absent) {
			t.Errorf("scaffold must NOT document project-scoped field %q (only system/both)", absent)
		}
	}

	// The scaffold is fully commented → parses as an inert (empty) config: no live
	// providers, no live role profiles.
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadPath(tmp)
	if err != nil {
		t.Fatalf("scaffold must parse cleanly (all-commented YAML): %v", err)
	}
	if _, ok := cfg.GetProvider("claude"); ok {
		t.Error("scaffold must be fully commented — no live providers should parse")
	}
	if _, ok := cfg.GetAgentProfile("doing"); ok {
		t.Error("scaffold must be fully commented — no live agent.profiles should parse")
	}

	// A second run refuses to overwrite (non-zero exit, message names the path).
	out2, err := runConfig(t, "init", "--system")
	if err == nil {
		t.Fatal("second `init --system` must refuse to overwrite (non-zero exit)")
	}
	_ = out2
	if !strings.Contains(err.Error(), path) {
		t.Errorf("refusal message should name the path %q, got: %v", path, err)
	}
	// The existing file is not truncated/rewritten.
	after, _ := os.ReadFile(path)
	if string(after) != scaffold {
		t.Error("refused overwrite must leave the existing file byte-identical")
	}
}
