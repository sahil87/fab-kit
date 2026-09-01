package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configupgrade"
)

// setupInitRepo creates a temp repo rooted at a fab/ dir and chdirs into it, so
// resolve.FabRoot() resolves to <tmp>/fab. Returns the fab root path. HOME is
// isolated by the package TestMain.
func setupInitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	fabRoot := filepath.Join(repo, "fab")
	if err := os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdirTestEnv(t, repo, map[string]string{})
	return fabRoot
}

// TestConfigUpgradeCommand drives `fab config upgrade` end to end: it appends a
// managed fence to a legacy config.yaml, preserves the live keys, exits 0, and a
// second run is a byte-identical no-op reporting "already up to date".
func TestConfigUpgradeCommand(t *testing.T) {
	fabRoot := setupInitRepo(t)
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")
	legacy := "project:\n    name: t\n    description: d\n\nlegacy_mode: true\n"
	if err := os.WriteFile(cfgPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"upgrade"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("`config upgrade` returned an error: %v", err)
	}

	written, _ := os.ReadFile(cfgPath)
	got := string(written)
	if !strings.Contains(got, "# >>> fab reference") {
		t.Errorf("upgrade must append a managed fence, got:\n%s", got)
	}
	if !strings.Contains(got, "name: t") {
		t.Error("upgrade must preserve the live project field")
	}
	if !strings.Contains(got, "#   legacy_mode: true") {
		t.Error("upgrade must park the unknown live key below the fence")
	}
	if !strings.Contains(out.String(), "Upgraded") {
		t.Errorf("first run should report Upgraded, got: %q", out.String())
	}

	// Second run: no-op, byte-identical.
	cmd2 := configCmd()
	var out2 strings.Builder
	cmd2.SetOut(&out2)
	cmd2.SetErr(&out2)
	cmd2.SetArgs([]string{"upgrade"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("`config upgrade` (2nd) returned an error: %v", err)
	}
	written2, _ := os.ReadFile(cfgPath)
	if string(written2) != got {
		t.Error("second `config upgrade` churned the file (must be idempotent)")
	}
	if !strings.Contains(out2.String(), "already up to date") {
		t.Errorf("second run should report already up to date, got: %q", out2.String())
	}

	// Extra positional arg rejected (cobra.NoArgs).
	cmd3 := configCmd()
	var errBuf strings.Builder
	cmd3.SetOut(&errBuf)
	cmd3.SetErr(&errBuf)
	cmd3.SetArgs([]string{"upgrade", "extra"})
	if err := cmd3.Execute(); err == nil {
		t.Error("`config upgrade extra` should be rejected (cobra.NoArgs)")
	}
}

func TestConfigUpgradeSystem_CommandModesAndEffectiveValue(t *testing.T) {
	fabRoot := setupInitRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	systemPath := filepath.Join(home, ".fab-kit", "config.yaml")
	projectPath := filepath.Join(fabRoot, "project", "config.yaml")
	projectBefore := "project:\n    name: untouched\n# project sentinel\n"
	if err := os.WriteFile(projectPath, []byte(projectBefore), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(systemPath), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "# work laptop only\nagent:\n  workers: codex # keep exact\n"
	if err := os.WriteFile(systemPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	show := func() string {
		t.Helper()
		cmd := configCmd()
		var out strings.Builder
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{"show", "agent.workers"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("config show agent.workers: %v", err)
		}
		return out.String()
	}
	beforeShow := show()
	out, err := runConfigUpgrade(t, "--system")
	if err != nil {
		t.Fatalf("config upgrade --system: %v", err)
	}
	if !strings.Contains(out, "Upgraded "+systemPath) {
		t.Fatalf("system upgrade output = %q", out)
	}
	if projectAfter, _ := os.ReadFile(projectPath); string(projectAfter) != projectBefore {
		t.Fatal("system-only upgrade changed the project config")
	}
	afterShow := show()
	if afterShow != beforeShow {
		t.Fatalf("system adoption changed config show output: before=%q after=%q", beforeShow, afterShow)
	}
	written, err := os.ReadFile(systemPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "# work laptop only\nagent:\n  workers: codex # keep exact") {
		t.Fatalf("system upgrade did not preserve live bytes and comment:\n%s", written)
	}

	second, err := runConfigUpgrade(t, "--system")
	if err != nil || !strings.Contains(second, "already up to date") {
		t.Fatalf("second system upgrade = %q, %v", second, err)
	}
	cleanBytes, _ := os.ReadFile(systemPath)
	if string(cleanBytes) != string(written) {
		t.Fatal("second system upgrade changed clean bytes")
	}

	stale := strings.Replace(string(written), "(kit "+version+")", "(kit 0.0.1)", 1)
	if err := os.WriteFile(systemPath, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	checkOut, err := runConfigUpgrade(t, "--system", "--check")
	if err == nil || !strings.Contains(checkOut, "drifted") {
		t.Fatalf("system drift check = %q, %v; want non-zero drift", checkOut, err)
	}
	if after, _ := os.ReadFile(systemPath); string(after) != stale {
		t.Fatal("config upgrade --system --check wrote the file")
	}
}

func TestConfigUpgradeAll_ReportsBothLayersAndRejectsContradictoryModes(t *testing.T) {
	fabRoot := setupInitRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	projectPath := filepath.Join(fabRoot, "project", "config.yaml")
	systemPath := filepath.Join(home, ".fab-kit", "config.yaml")
	if err := os.WriteFile(projectPath, []byte("project:\n    name: t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(systemPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(systemPath, []byte("dispatch:\n  mode: pane\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runConfigUpgrade(t, "--all")
	if err != nil {
		t.Fatalf("config upgrade --all: %v", err)
	}
	if !strings.Contains(out, "project: Upgraded ") || !strings.Contains(out, "system: Upgraded ") {
		t.Fatalf("--all must report both labelled layers, got %q", out)
	}
	cleanOut, err := runConfigUpgrade(t, "--check", "--all")
	if err != nil {
		t.Fatalf("clean config upgrade --check --all: %v", err)
	}
	if !strings.Contains(cleanOut, "project: ") || !strings.Contains(cleanOut, "system: ") {
		t.Fatalf("clean --all check lacks per-layer labels: %q", cleanOut)
	}

	systemClean, _ := os.ReadFile(systemPath)
	staleSystem := strings.Replace(string(systemClean), "(kit "+version+")", "(kit 0.0.1)", 1)
	if err := os.WriteFile(systemPath, []byte(staleSystem), 0o644); err != nil {
		t.Fatal(err)
	}
	projectBefore, _ := os.ReadFile(projectPath)
	driftOut, err := runConfigUpgrade(t, "--all", "--check")
	if err == nil {
		t.Fatal("--check --all must fail when the system layer drifts")
	}
	if !strings.Contains(driftOut, "project: ") || !strings.Contains(driftOut, "system: ") || !strings.Contains(driftOut, "drifted") {
		t.Fatalf("drifted --all output lacks labelled results: %q", driftOut)
	}
	if projectAfter, _ := os.ReadFile(projectPath); string(projectAfter) != string(projectBefore) {
		t.Fatal("--check --all wrote the project file")
	}
	if systemAfter, _ := os.ReadFile(systemPath); string(systemAfter) != staleSystem {
		t.Fatal("--check --all wrote the system file")
	}

	for _, flags := range [][]string{{"--system", "--project"}, {"--system", "--all"}, {"--project", "--all"}} {
		if got, err := runConfigUpgrade(t, flags...); err == nil {
			t.Errorf("contradictory flags %v succeeded; output=%q", flags, got)
		}
	}
}

func TestConfigInitSystemAndUpgradeShareCanonicalRender(t *testing.T) {
	want, err := configupgrade.RenderSystemScaffold(version)
	if err != nil {
		t.Fatal(err)
	}
	setupInitRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := runConfigUpgrade(t, "--system"); err != nil {
		t.Fatalf("system upgrade on missing file: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(home, ".fab-kit", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatal("system init and upgrade no longer share the canonical scaffold renderer")
	}
	// A --check immediately after the shared render must report clean and exit 0
	// (acceptance: init --system then upgrade --system --check is a no-drift
	// no-op).
	checkOut, err := runConfigUpgrade(t, "--system", "--check")
	if err != nil {
		t.Fatalf("init --system then upgrade --system --check should exit 0: %v (%q)", err, checkOut)
	}
	if !strings.Contains(checkOut, "already up to date") {
		t.Errorf("--check after --system upgrade should report already up to date, got: %q", checkOut)
	}
}

// TestConfigInitProjectCommand drives `fab config init --project`: it generates a
// config.yaml with the seeded identity fields live and the managed fence, does NOT
// pin agent.profiles, and refuses to overwrite an existing file.
func TestConfigInitProjectCommand(t *testing.T) {
	fabRoot := setupInitRepo(t)
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")

	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"init", "--project",
		"--name", "MyProj", "--description", "a demo",
		"--source-path", "src/", "--test-path", "**/*_test.go"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("`config init --project` returned an error: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("expected a generated config.yaml: %v", err)
	}
	got := string(data)
	for _, want := range []string{"name: MyProj", "description: a demo", "source_paths:", "- src/", "test_paths:", "# >>> fab reference"} {
		if !strings.Contains(got, want) {
			t.Errorf("generated config missing %q:\n%s", want, got)
		}
	}
	// agent.profiles must NOT be pinned live (presence=intent) — the only `agent:`
	// mention allowed is the commented fence scaffold.
	for _, ln := range strings.Split(got, "\n") {
		if strings.HasPrefix(ln, "agent:") {
			t.Error("init --project must not pin agent.profiles live")
		}
	}

	// Refuses to overwrite.
	cmd2 := configCmd()
	var out2 strings.Builder
	cmd2.SetOut(&out2)
	cmd2.SetErr(&out2)
	cmd2.SetArgs([]string{"init", "--project", "--name", "Other"})
	if err := cmd2.Execute(); err == nil {
		t.Error("init --project must refuse to overwrite an existing config.yaml")
	}
}

// TestConfigInitBareDefaultsProjectAndBothFlagsRejected: bare init generates the
// project file, while passing both explicit modes still errors.
func TestConfigInitBareDefaultsProjectAndBothFlagsRejected(t *testing.T) {
	fabRoot := setupInitRepo(t)

	bare := configCmd()
	var b1 strings.Builder
	bare.SetOut(&b1)
	bare.SetErr(&b1)
	bare.SetArgs([]string{"init"})
	if err := bare.Execute(); err != nil {
		t.Fatalf("bare `config init` should select project mode: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fabRoot, "project", "config.yaml")); err != nil {
		t.Fatalf("bare init did not create project config: %v", err)
	}

	both := configCmd()
	var b2 strings.Builder
	both.SetOut(&b2)
	both.SetErr(&b2)
	both.SetArgs([]string{"init", "--system", "--project"})
	if err := both.Execute(); err == nil {
		t.Error("`config init --system --project` should error (mutually exclusive)")
	}
}
