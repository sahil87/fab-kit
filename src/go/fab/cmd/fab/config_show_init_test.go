package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
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

// TestConfigShow_PrintsEffectiveConfig: `fab config show` prints the merged
// effective config (project over system) as YAML and exits 0.
func TestConfigShow_PrintsEffectiveConfig(t *testing.T) {
	_, home := setupConfigRepo(t, `
providers:
  claude:
    session_command: project-session
`)
	writeSystemConfig(t, home, `
providers:
  codex:
    dispatch_command: codex exec
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

// TestConfigShow_RejectsTooManyArgs: show accepts one key but rejects two.
func TestConfigShow_RejectsExtraArgs(t *testing.T) {
	setupConfigRepo(t, "providers:\n  claude:\n    session_command: x\n")
	if _, err := runConfig(t, "show", "agent.workers", "extra"); err == nil {
		t.Error("`config show` should reject more than one key")
	}
	if _, err := runConfig(t, "show", "--origin", "agent.workers", "extra"); err == nil {
		t.Error("`config show --origin` should reject more than one key")
	}
}

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
	wantOrigin := filepath.Join(repo, "fab", "project", "config.yaml")
	if out != "codex  # "+wantOrigin+"\n" {
		t.Fatalf("keyed origin output = %q", out)
	}
	for _, key := range []string{"agent.workerz", "providers.#local.session_command"} {
		if _, err := runConfig(t, "show", key); err == nil || !strings.Contains(err.Error(), key) {
			t.Fatalf("unknown/structurally invalid keyed show %q should name the key, got %v", key, err)
		}
	}
}

func TestConfigShow_KeyedListOriginIsCompact(t *testing.T) {
	repo, _ := setupConfigRepo(t, "source_paths:\n    - src/\n    - scripts/\n")
	out, err := runConfig(t, "show", "source_paths", "--origin")
	if err != nil {
		t.Fatalf("keyed list show --origin: %v", err)
	}
	want := "[src/, scripts/]  # " + filepath.Join(repo, "fab", "project", "config.yaml") + "\n"
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
			key := "providers." + name + ".session_command"
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
		{[]string{"set", "providers.#local.session_command", "tool"}, "dotted config-key grammar"},
		{[]string{"set", "agent.workers", "{bad: kind}"}, "single-line YAML scalar"},
		{[]string{"set", "agent.workers", "codex # note"}, "must not contain a YAML comment"},
		{[]string{"set", "source_paths", "[src/]"}, "scalar leaf"},
		{[]string{"set", "source_paths", "[src/]", "--system"}, "project scope"},
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
    session_command: project-session
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
	assertOriginLine("providers.claude.session_command = project-session", projectPath)
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
// layer replaces a map with a scalar, --origin must honor deepMerge precedence and
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
	t.Setenv("FAB_AGENT_SESSION", "gemini")
	t.Setenv("FAB_AGENT_WORKERS", "codex")

	plain, err := runConfig(t, "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(plain, "session: gemini") || !strings.Contains(plain, "workers: codex") {
		t.Errorf("plain show must include env-effective values:\n%s", plain)
	}

	origin, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	for keyValue, variable := range map[string]string{
		"agent.session = gemini": "$FAB_AGENT_SESSION",
		"agent.workers = codex":  "$FAB_AGENT_WORKERS",
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

func TestConfigShowOrigin_EnvironmentNullRemainsPresent(t *testing.T) {
	setupConfigRepo(t, "agent:\n  workers: project-worker\n")
	t.Setenv("FAB_AGENT_WORKERS", "null")

	plain, err := runConfig(t, "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(plain, "workers: null") {
		t.Fatalf("plain show must preserve the explicit null override:\n%s", plain)
	}

	origin, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatalf("config show --origin: %v", err)
	}
	for _, line := range strings.Split(origin, "\n") {
		if strings.Contains(line, "agent.workers =") {
			if !strings.Contains(line, "agent.workers = null") || !strings.Contains(line, "# $FAB_AGENT_WORKERS") {
				t.Fatalf("explicit null must retain its env value and origin, got %q", line)
			}
			return
		}
	}
	t.Fatalf("origin output missing agent.workers:\n%s", origin)
}

func TestConfigShowKey_EnvironmentNullWins(t *testing.T) {
	setupConfigRepo(t, "agent:\n  workers: project-worker\n")
	t.Setenv("FAB_AGENT_WORKERS", "null")

	plain, err := runConfig(t, "show", "agent.workers")
	if err != nil {
		t.Fatalf("config show agent.workers: %v", err)
	}
	if plain != "null\n" {
		t.Fatalf("keyed show ignored the explicit environment null: %q", plain)
	}

	origin, err := runConfig(t, "show", "agent.workers", "--origin")
	if err != nil {
		t.Fatalf("config show agent.workers --origin: %v", err)
	}
	if origin != "null  # $FAB_AGENT_WORKERS\n" {
		t.Fatalf("keyed show lost explicit-null provenance: %q", origin)
	}
}

func TestConfigShowKey_UsesNearestReplacedAncestorOrigin(t *testing.T) {
	setupConfigRepo(t, "providers:\n  codex:\n    session_command: project-command\n")
	t.Setenv("FAB_PROVIDERS", "null")

	plain, err := runConfig(t, "show", "providers.codex.session_command")
	if err != nil {
		t.Fatalf("config show descendant: %v", err)
	}
	if plain != "null\n" {
		t.Fatalf("ancestor replacement did not null the descendant: %q", plain)
	}

	origin, err := runConfig(t, "show", "providers.codex.session_command", "--origin")
	if err != nil {
		t.Fatalf("config show descendant --origin: %v", err)
	}
	if origin != "null  # $FAB_PROVIDERS\n" {
		t.Fatalf("descendant lost the nearest replaced ancestor's provenance: %q", origin)
	}
}

func TestConfigShowKey_EmptyCollectionKeepsWinningOrigin(t *testing.T) {
	t.Run("environment mapping", func(t *testing.T) {
		setupConfigRepo(t, "")
		t.Setenv("FAB_PROVIDERS", "{custom: {}}")

		plain, err := runConfig(t, "show", "providers.custom")
		if err != nil {
			t.Fatalf("config show empty mapping: %v", err)
		}
		if plain != "{}\n" {
			t.Fatalf("empty mapping output = %q", plain)
		}

		origin, err := runConfig(t, "show", "providers.custom", "--origin")
		if err != nil {
			t.Fatalf("config show empty mapping --origin: %v", err)
		}
		if origin != "{}  # $FAB_PROVIDERS\n" {
			t.Fatalf("empty mapping lost its effective value or environment origin: %q", origin)
		}
	})

	t.Run("project sequence", func(t *testing.T) {
		repo, _ := setupConfigRepo(t, "source_paths: []\n")

		origin, err := runConfig(t, "show", "source_paths", "--origin")
		if err != nil {
			t.Fatalf("config show empty sequence --origin: %v", err)
		}
		want := "[]  # " + filepath.Join(repo, "fab", "project", "config.yaml") + "\n"
		if origin != want {
			t.Fatalf("empty sequence lost its compact value or project origin: got %q, want %q", origin, want)
		}
	})
}

// TestConfigInitSystem_WritesScaffoldAndRefusesOverwrite: `fab config init
// --system` writes the ~/.fab-kit/config.yaml scaffold (only system/both fields,
// all commented → parses as inert), and a second run refuses to overwrite.
func TestConfigInitSystem_WritesScaffoldAndRefusesOverwrite(t *testing.T) {
	_, home := setupConfigRepo(t, "providers:\n  claude:\n    session_command: x\n")

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
	for _, absent := range []string{"source_paths:", "test_paths:", "true_impact_exclude:", "fab_version:", "branch_prefix:"} {
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
