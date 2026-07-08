package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
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
	chdirTestEnv(t, repo, map[string]string{"HOME": home})
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

// TestConfigShow_RejectsExtraArgsAndWritesNoFile: show is a pure query — extra
// positional args are rejected by cobra.NoArgs, and it writes no file.
func TestConfigShow_RejectsExtraArgs(t *testing.T) {
	setupConfigRepo(t, "providers:\n  claude:\n    session_command: x\n")
	if _, err := runConfig(t, "show", "extra"); err == nil {
		t.Error("`config show extra` should be rejected (cobra.NoArgs)")
	}
	if _, err := runConfig(t, "show", "--origin", "extra"); err == nil {
		t.Error("`config show --origin extra` should be rejected (cobra.NoArgs)")
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
  tiers:
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
	assertOriginLine("agent.tiers.newbie.model = sys-model", systemPath)
	// An unset field with a built-in default shows `default` (the doing tier's
	// profile is a built-in default not overridden here).
	assertOriginLine("agent.tiers.doing.model =", "default")
}

// TestConfigShowOrigin_TypoSurfacesAsDefault: a typo'd override (agent.teirs)
// does not land, so the field the user MEANT to set shows origin `default` — the
// git-config-show-origin value the intake calls out. The misspelled key is
// simply an unknown key (ignored), so the real agent.tiers stays at its default.
func TestConfigShowOrigin_TypoSurfacesAsDefault(t *testing.T) {
	_, home := setupConfigRepo(t, `
agent:
  teirs:            # typo — should have been "tiers"
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
		if strings.Contains(line, "agent.tiers.doing.model =") {
			if !strings.Contains(line, "# default") {
				t.Errorf("typo'd override should leave agent.tiers.doing.model at default, got: %q", line)
			}
			if strings.Contains(line, "i-meant-to-set-this") {
				t.Errorf("typo'd (misspelled) key must not take effect: %q", line)
			}
		}
	}
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

	// The scaffold documents ONLY the system/both fields (agent.tiers, providers)
	// and none of the project-scoped fields.
	for _, want := range []string{"providers", "agent.tiers"} {
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
	// providers, no live tiers.
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
	if _, ok := cfg.GetAgentTier("doing"); ok {
		t.Error("scaffold must be fully commented — no live agent.tiers should parse")
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

// TestConfigInitBare_UsageError: bare `fab config init` (no --system) is a usage
// error — project bootstrap is /fab-setup's job.
func TestConfigInitBare_UsageError(t *testing.T) {
	setupConfigRepo(t, "providers:\n  claude:\n    session_command: x\n")
	if _, err := runConfig(t, "init"); err == nil {
		t.Error("bare `config init` (no --system) must be a usage error")
	}
}
