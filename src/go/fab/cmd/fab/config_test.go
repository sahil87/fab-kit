package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
)

// TestConfigReferenceRoundTrips is the VALIDITY contract: the emitted reference
// parses cleanly via the same internal/config loader real project configs use.
// A malformed reference (bad indentation, an un-quoted value) would fail here.
func TestConfigReferenceRoundTrips(t *testing.T) {
	// Isolate HOME so config.LoadPath does not merge the developer's real
	// ~/.fab-kit/config.yaml over the reference (the cascade, lpb5, reads the
	// system layer at every LoadPath). We assert the REFERENCE's own live keys.
	t.Setenv("HOME", t.TempDir())

	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadPath(path)
	if err != nil {
		t.Fatalf("reference config.yaml did not round-trip into Config: %v", err)
	}

	// The live baseline keys populate their Config fields (sanity that the
	// live/commented split landed as intended — not just that it parsed).
	prov, ok := cfg.GetProvider("claude")
	if !ok || prov.InteractiveCommand == "" {
		t.Error("providers.claude.interactive_command should be a live key with a value in the reference")
	}
	if prov.HeadlessCommand != agent.DefaultHeadlessCommand || !prov.Native {
		t.Errorf("providers.claude capabilities = %+v, want live dispatch command and native=true", prov)
	}
	// codex, agy and kimi are Go BUILT-IN providers whose reference blocks render
	// LIVE like claude's — all four blocks uniform, so a fence hoist of any one
	// yields exactly the built-in the block restates (a live copy PINS those
	// fills against kit-release refreshes; presence=intent still holds because
	// the reference file is documentation, not a project config).
	for _, name := range []string{"codex", "agy", "kimi"} {
		want, _ := agent.ResolveProvider(nil, name)
		got, ok := cfg.GetProvider(name)
		if !ok {
			t.Errorf("providers.%s must be live in the reference (all four built-ins render uniformly)", name)
			continue
		}
		if got.InteractiveCommand != want.InteractiveCommand || got.HeadlessCommand != want.HeadlessCommand {
			t.Errorf("providers.%s = {%q, %q}, want the built-in {%q, %q}",
				name, got.InteractiveCommand, got.HeadlessCommand, want.InteractiveCommand, want.HeadlessCommand)
		}
	}
	if len(cfg.TestPaths) == 0 {
		t.Error("test_paths should be a live key with a value in the reference")
	}
	if len(cfg.TrueImpactExclude) == 0 {
		t.Error("true_impact_exclude should be a live key with a value in the reference")
	}
	// The two advertised depth knobs are shown LIVE at their built-in value — they
	// are the whole advertised agent surface (260806-j9nh), so they must parse.
	if got := cfg.GetAgentSession(); got != "claude" {
		t.Errorf("agent.session must be live in the reference at its built-in value, got %q", got)
	}
	if got := cfg.GetAgentWorkers(); got != "claude" {
		t.Errorf("agent.workers must be live in the reference at its built-in value, got %q", got)
	}
	// agent.profiles is the DEMOTED machinery beneath them: documented in the
	// segment's pointer lines, but never live (a live per-role profile would pin
	// today's models into every project's config — presence=intent).
	if _, ok := cfg.GetAgentProfile("doing"); ok {
		t.Error("agent.profiles must be commented-out in the reference (parsed as live)")
	}
	// The opt-in override block must stay commented-out (uncommenting = opting in).
	if len(cfg.StageHooks) != 0 {
		t.Error("stage_hooks must be commented-out in the reference (parsed as live)")
	}
}

// TestConfigReferenceCoversBinaryKeys is the BINARY-KEY coverage contract: every
// yaml-tagged key path reachable from config.Config (recursively — nested
// structs and map value types) must appear in the reference (commented or live).
// Adding a new binary-consumed key to Config then forces a reference update at
// test time. Injected default *values* need no drift test (no second copy), but
// key *presence* is guarded here.
func TestConfigReferenceCoversBinaryKeys(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}

	segments := yamlKeySegments(reflect.TypeOf(config.Config{}))
	if len(segments) == 0 {
		t.Fatal("reflection produced no yaml key segments — walk is broken")
	}

	// fab_version is NOT a config key (260708-j0qm): it lives in the plain-text
	// sibling fab/.fab-version and Config.FabVersion is tagged `yaml:"-"`, so
	// yamlKeySegments skips it and it never appears in `segments` — no exemption is
	// needed here (the positive "not documented" assertion lives in
	// TestConfigReferenceOmitsRelocatedFabVersion).
	//
	// The deprecated pre-2.19.0 spellings session_command/dispatch_command are NOT
	// exempted either (260809-n1he): they are read-time aliases the write surface
	// rejects, but the providers segment NAMES them in its deprecation note, so
	// they earn coverage the same way `agent.tiers` does from the agent segment.
	// Documenting a deprecated spelling is what makes it discoverable to a user
	// reading an unmigrated config — the walker stays exemption-free.
	for seg := range segments {
		if !containsKeyToken(out, seg) {
			t.Errorf("binary-consumed config key %q (from Config yaml tags) is not documented in `fab config explain`", seg)
		}
	}
}

// TestConfigReferenceOmitsRelocatedFabVersion pins the 260708-j0qm relocation:
// fab_version left config.yaml for the plain-text sibling fab/.fab-version, so the
// generated reference (and the registry it walks) must NOT document a fab_version
// key. It is machine-managed and no longer a config-file field.
func TestConfigReferenceOmitsRelocatedFabVersion(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	if containsKeyToken(out, "fab_version") {
		t.Error("fab_version moved to fab/.fab-version and must not appear in the reference")
	}
	keys, err := configref.FieldKeys()
	if err != nil {
		t.Fatalf("FieldKeys returned an error: %v", err)
	}
	for _, k := range keys {
		if k == "fab_version" {
			t.Error("the registry must not carry a fab_version row (it left config.yaml)")
		}
	}
}

// TestConfigInitSeedKeysSubsetOfRegistry is the SKILL-KEY coverage contract's new
// anchor (260708-j0qm): the scaffold config.yaml was deleted — `fab config init
// --project` now generates the initial config.yaml from the registry, so the
// former "reference ⊇ scaffold keys" guard is re-anchored to a registry-internal
// invariant. Every A-class identity key the init generator writes live (InitSeed)
// must be a documented registry field, so a generated project config can never
// carry a key the reference does not describe.
func TestConfigInitSeedKeysSubsetOfRegistry(t *testing.T) {
	seedKeys, err := configref.InitSeedKeys()
	if err != nil {
		t.Fatalf("InitSeedKeys returned an error: %v", err)
	}
	if len(seedKeys) == 0 {
		t.Fatal("no InitSeed keys — the init generator would write no identity fields")
	}
	registryKeys, err := configref.FieldKeys()
	if err != nil {
		t.Fatalf("FieldKeys returned an error: %v", err)
	}
	known := map[string]bool{}
	for _, k := range registryKeys {
		known[k] = true
	}
	for _, k := range seedKeys {
		if !known[k] {
			t.Errorf("init-seeded key %q is not a documented registry field (would generate an undocumented key)", k)
		}
	}

	// The seeded identity set is the A-class fields the design fixes: the project
	// identity, source_paths, and test_paths. Pin it so a future edit that seeds a
	// preference-class field (e.g. agent.profiles — which presence=intent forbids
	// pinning at init) fails here.
	wantSeed := map[string]bool{
		"project.name":        true,
		"project.description": true,
		"source_paths":        true,
		"test_paths":          true,
	}
	got := map[string]bool{}
	for _, k := range seedKeys {
		got[k] = true
	}
	for k := range wantSeed {
		if !got[k] {
			t.Errorf("expected %q to be an init-seed identity field", k)
		}
	}
	for k := range got {
		if !wantSeed[k] {
			t.Errorf("unexpected init-seed field %q (only A-class identity fields are seeded at init)", k)
		}
	}
}

// runConfigInit drives `fab config init <args...>` end to end via the cobra
// command and returns its stdout.
func runConfigInit(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"init"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

// setupInitTempRepo creates a bare repo root with a fab/ directory and chdirs
// into it so resolve.FabRoot finds the temp tree, not the real checkout.
func setupInitTempRepo(t *testing.T) (fabRoot string) {
	t.Helper()
	repoRoot := t.TempDir()
	fabRoot = filepath.Join(repoRoot, "fab")
	if err := os.MkdirAll(fabRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	chdirTestEnv(t, repoRoot, nil)
	return fabRoot
}

// TestConfigInitProjectPrintMatchesWrite pins the --print contract in project
// mode: stdout is byte-for-byte the file the write path produces (same render
// call), and the print run itself writes nothing.
func TestConfigInitProjectPrintMatchesWrite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fabRoot := setupInitTempRepo(t)
	path := filepath.Join(fabRoot, "project", "config.yaml")
	seedArgs := []string{"--project", "--name", "my-app", "--description", "My application",
		"--source-path", "src/", "--test-path", "**/*_test.go"}

	printed, err := runConfigInit(t, append(seedArgs, "--print")...)
	if err != nil {
		t.Fatalf("`config init --print` returned an error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("--print must not write %s", path)
	}

	if _, err := runConfigInit(t, seedArgs...); err != nil {
		t.Fatalf("`config init` write run returned an error: %v", err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(written) {
		t.Errorf("--print stdout != the bytes the write path produced\n--- printed ---\n%s\n--- written ---\n%s", printed, written)
	}
}

// TestConfigInitSystemPrintMatchesWrite pins the same --print contract in
// --system mode against a temp HOME.
func TestConfigInitSystemPrintMatchesWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".fab-kit", "config.yaml")

	printed, err := runConfigInit(t, "--system", "--print")
	if err != nil {
		t.Fatalf("`config init --system --print` returned an error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("--print must not write %s", path)
	}

	if _, err := runConfigInit(t, "--system"); err != nil {
		t.Fatalf("`config init --system` write run returned an error: %v", err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if printed != string(written) {
		t.Errorf("--print stdout != the bytes the write path produced\n--- printed ---\n%s\n--- written ---\n%s", printed, written)
	}
}

// TestConfigInitPrintIgnoresExistingFile: an existing target file does NOT block
// --print (it is a preview, not a write), and the file is left untouched.
func TestConfigInitPrintIgnoresExistingFile(t *testing.T) {
	sentinel := []byte("# user-owned\n")

	t.Run("project", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		fabRoot := setupInitTempRepo(t)
		path := filepath.Join(fabRoot, "project", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, sentinel, 0o644); err != nil {
			t.Fatal(err)
		}
		printed, err := runConfigInit(t, "--project", "--name", "my-app", "--print")
		if err != nil {
			t.Fatalf("--print with an existing file returned an error: %v", err)
		}
		if !strings.Contains(printed, "my-app") {
			t.Errorf("--print output does not look like the seeded project config:\n%s", printed)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(sentinel) {
			t.Error("--print modified the existing project config")
		}
	})

	t.Run("system", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		path := filepath.Join(home, ".fab-kit", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, sentinel, 0o644); err != nil {
			t.Fatal(err)
		}
		printed, err := runConfigInit(t, "--system", "--print")
		if err != nil {
			t.Fatalf("--print with an existing file returned an error: %v", err)
		}
		if printed == "" {
			t.Error("--print produced no output despite the existing file")
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(sentinel) {
			t.Error("--print modified the existing system config")
		}
	})
}

// TestConfigInitForceOverwrites: --force replaces the existing-file refusal with
// an explicit overwrite in both modes; the refusal stays the default without it.
func TestConfigInitForceOverwrites(t *testing.T) {
	sentinel := []byte("# user-owned\n")

	t.Run("project", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		fabRoot := setupInitTempRepo(t)
		path := filepath.Join(fabRoot, "project", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, sentinel, 0o644); err != nil {
			t.Fatal(err)
		}
		seedArgs := []string{"--project", "--name", "my-app"}

		// Default: refusal, verbatim prefix preserved, file untouched.
		if _, err := runConfigInit(t, seedArgs...); err == nil ||
			!strings.Contains(err.Error(), "refusing to overwrite existing project config") {
			t.Fatalf("default run should refuse to overwrite, got: %v", err)
		}
		if after, _ := os.ReadFile(path); string(after) != string(sentinel) {
			t.Fatal("the refused run modified the existing project config")
		}

		if _, err := runConfigInit(t, append(seedArgs, "--force")...); err != nil {
			t.Fatalf("--force run returned an error: %v", err)
		}
		printed, err := runConfigInit(t, append(seedArgs, "--print")...)
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) == string(sentinel) || string(after) != printed {
			t.Error("--force did not overwrite the project config with the rendered content")
		}
	})

	t.Run("system", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		path := filepath.Join(home, ".fab-kit", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, sentinel, 0o644); err != nil {
			t.Fatal(err)
		}

		if _, err := runConfigInit(t, "--system"); err == nil ||
			!strings.Contains(err.Error(), "refusing to overwrite existing system config") {
			t.Fatalf("default run should refuse to overwrite, got: %v", err)
		}
		if after, _ := os.ReadFile(path); string(after) != string(sentinel) {
			t.Fatal("the refused run modified the existing system config")
		}

		if _, err := runConfigInit(t, "--system", "--force"); err != nil {
			t.Fatalf("--force run returned an error: %v", err)
		}
		printed, err := runConfigInit(t, "--system", "--print")
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) == string(sentinel) || string(after) != printed {
			t.Error("--force did not overwrite the system config with the rendered scaffold")
		}
	})
}

// TestConfigInitPrintForceIsPurePreview: --print + --force lets print win — the
// preview prints and writes nothing, even with an existing file in place.
func TestConfigInitPrintForceIsPurePreview(t *testing.T) {
	sentinel := []byte("# user-owned\n")

	t.Run("project", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		fabRoot := setupInitTempRepo(t)
		path := filepath.Join(fabRoot, "project", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, sentinel, 0o644); err != nil {
			t.Fatal(err)
		}
		printed, err := runConfigInit(t, "--project", "--name", "my-app", "--print", "--force")
		if err != nil {
			t.Fatalf("--print --force returned an error: %v", err)
		}
		if printed == "" {
			t.Error("--print --force produced no output")
		}
		if after, _ := os.ReadFile(path); string(after) != string(sentinel) {
			t.Error("--print --force wrote to the project config")
		}
	})

	t.Run("system", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		path := filepath.Join(home, ".fab-kit", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, sentinel, 0o644); err != nil {
			t.Fatal(err)
		}
		printed, err := runConfigInit(t, "--system", "--print", "--force")
		if err != nil {
			t.Fatalf("--print --force returned an error: %v", err)
		}
		if printed == "" {
			t.Error("--print --force produced no output")
		}
		if after, _ := os.ReadFile(path); string(after) != string(sentinel) {
			t.Error("--print --force wrote to the system config")
		}
	})
}

// TestConfigInitSystemScaffoldCarriesScopeAnnotations (260809-wll4 R6/R7): the
// --system scaffold renders the file-bound SHORT form — every advert carries its
// [system|both] scope tag, the `fab config set --system` machine-wide pointer,
// and the `fab config explain <key>` pointer — and none of the long-form essays
// (the old invite-to-uncomment machine-wide phrasing is gone from generated
// files; it stays in `fab config explain`).
func TestConfigInitSystemScaffoldCarriesScopeAnnotations(t *testing.T) {
	scaffold, err := renderSystemScaffold()
	if err != nil {
		t.Fatalf("renderSystemScaffold: %v", err)
	}
	for _, want := range []string{
		"[both]",
		"# Settable machine-wide: fab config set --system ",
		"# Full prose: fab config explain ",
	} {
		if !strings.Contains(scaffold, want) {
			t.Errorf("system scaffold must carry %q.\n--- scaffold ---\n%s", want, scaffold)
		}
	}
	if strings.Contains(scaffold, "outranks the project file") {
		t.Error("system scaffold must not carry the long-form machine-wide essay phrasing (the diet moved it to `fab config explain`)")
	}
	// Project-scoped fields are never system-overridable — no [project] advert.
	if strings.Contains(scaffold, "[project]") {
		t.Error("system scaffold must not advertise project-scoped fields")
	}
	// Byte-stable across renders (the scaffold is a generated file).
	second, err := renderSystemScaffold()
	if err != nil {
		t.Fatalf("renderSystemScaffold (2nd): %v", err)
	}
	if scaffold != second {
		t.Error("system scaffold is not byte-stable across renders")
	}
}

// TestConfigReferenceByteStable: repeated renders are byte-identical (the
// byte-stable stdout contract the docs/website pointer relies on).
func TestConfigReferenceByteStable(t *testing.T) {
	first, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	second, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	if first != second {
		t.Errorf("`fab config explain` output is not byte-stable across renders")
	}
}

// TestConfigReferenceCommandPrintsAndExitsZero drives the cobra command end to
// end via the invisible `reference` alias: it prints the reference to stdout and
// exits 0 with no args, and rejects a second positional. Note the rejection is a
// RUNTIME unknown-key error from RunE (exit 1) — explain is
// cobra.MaximumNArgs(1), so `reference extra` passes arg validation and fails
// key lookup, not a parse-time usage error (exit 2).
func TestConfigReferenceCommandPrintsAndExitsZero(t *testing.T) {
	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reference"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("`config reference` returned an error: %v", err)
	}
	want, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	if out.String() != want {
		t.Error("`config reference` stdout does not match configref.Render()")
	}

	// Extra positional arg is rejected.
	cmd2 := configCmd()
	var errBuf strings.Builder
	cmd2.SetOut(&errBuf)
	cmd2.SetErr(&errBuf)
	cmd2.SetArgs([]string{"reference", "extra"})
	if err := cmd2.Execute(); err == nil {
		t.Error("`config reference extra` should be rejected (unknown-key lookup in RunE)")
	}
}

func TestConfigExplainVisibleAliasAndKeyedSelection(t *testing.T) {
	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"explain", "project.name"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config explain project.name: %v", err)
	}
	want, err := configref.RenderKey("project.name")
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != want || !strings.Contains(out.String(), "project:") {
		t.Fatalf("keyed explain did not render the owning segment\n--- got ---\n%s", out.String())
	}

	jsonCmd := configCmd()
	var jsonOut strings.Builder
	jsonCmd.SetOut(&jsonOut)
	jsonCmd.SetErr(&jsonOut)
	jsonCmd.SetArgs([]string{"explain", "dispatch.column_width", "--json"})
	if err := jsonCmd.Execute(); err != nil {
		t.Fatalf("keyed explain --json: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonOut.String()), &rows); err != nil || len(rows) != 3 {
		t.Fatalf("keyed JSON should return the owning dispatch rows: len=%d err=%v\n%s", len(rows), err, jsonOut.String())
	}

	help := configCmd()
	var helpOut strings.Builder
	help.SetOut(&helpOut)
	help.SetErr(&helpOut)
	help.SetArgs([]string{"--help"})
	if err := help.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(helpOut.String(), "explain") || strings.Contains(helpOut.String(), "\n  reference") {
		t.Fatalf("group help must show explain but hide its reference alias\n%s", helpOut.String())
	}
	for _, section := range []string{"Inspect Commands:", "Modify Commands:", "Lifecycle Commands:"} {
		if !strings.Contains(helpOut.String(), section) {
			t.Errorf("group help is missing %q\n%s", section, helpOut.String())
		}
	}
	visible := make(map[string]string)
	for _, child := range configCmd().Commands() {
		if child.IsAvailableCommand() {
			visible[child.Name()] = child.GroupID
		}
	}
	wantVisible := map[string]string{
		"show": "inspect", "explain": "inspect",
		"set": "modify", "unset": "modify",
		"init": "lifecycle", "upgrade": "lifecycle",
	}
	if !reflect.DeepEqual(visible, wantVisible) {
		t.Errorf("visible config commands/groups = %#v, want %#v", visible, wantVisible)
	}
}

// TestConfigReferenceMentionsCommandPlaceholders guards that the reference's
// providers block documents the optional {model}/{effort} placeholders (the codex
// example command carries them, showing template-substitution mode).
func TestConfigReferenceMentionsCommandPlaceholders(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	for _, placeholder := range []string{"{model}", "{effort}"} {
		if !strings.Contains(out, placeholder) {
			t.Errorf("reference providers comment must document the optional %s placeholder", placeholder)
		}
	}
}

// TestConfigReferenceDocumentsBothSubstitutionSources is the nvad contract: the
// interactive_command comment must name BOTH sources of the {model}/{effort}
// substitution — the resolved role profile (role path) and the --model/--effort
// flags on `fab agent --provider <name>`, which bypasses role resolution. This
// literal renders into every project's config.yaml reference fence, so a
// role-only claim there is a user-facing documentation inaccuracy.
func TestConfigReferenceDocumentsBothSubstitutionSources(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	// The rendered comment is hard-wrapped with "# " prefixes, so assert against
	// the unwrapped form — the contract is the sentence, not where it breaks.
	flat := unwrapComment(out)
	for _, phrase := range []string{
		"substituted from the resolved role profile, or from the",
		"--model/--effort flags on `fab agent --provider <name>`",
		"(which bypasses role resolution)",
	} {
		if !strings.Contains(flat, phrase) {
			t.Errorf("interactive_command comment must document %q (both substitution sources)", phrase)
		}
	}
	// The superseded single-source claim must not survive anywhere in the reference.
	if strings.Contains(out, "substituted from the resolved tier profile (the built-in") {
		t.Error("interactive_command comment still carries the single-source substitution claim")
	}
}

// TestConfigReferenceDocumentsProviders guards that the generated reference
// documents the providers table with both command fields, native capability, and
// the load-bearing no-substitution semantic.
func TestConfigReferenceDocumentsProviders(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	for _, token := range []string{"providers:", "interactive_command", "headless_command", "native"} {
		if !strings.Contains(out, token) {
			t.Errorf("reference must document %q in the providers block", token)
		}
	}
	if !strings.Contains(out, "substitution between command fields") {
		t.Error("reference must document that command fields do not substitute for one another")
	}
}

// TestConfigReferenceDocumentsBuiltInProviders is the j3cm contract (which
// supersedes ho9y's starter-template contract): the providers block documents
// fab-kit's FOUR BUILT-IN providers — claude (the default), codex, agy and kimi —
// with every command string sourced from its canonical agent constant. agy carries
// no {effort} placeholder (its model IDs embed the reasoning level), and kimi
// deliberately ships no fills at all.
func TestConfigReferenceDocumentsBuiltInProviders(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}

	// All four provider names appear as text in the providers block.
	for _, provider := range []string{"claude:", "codex:", "agy:", "kimi:"} {
		if !strings.Contains(out, provider) {
			t.Errorf("providers block must document the %q provider", provider)
		}
	}

	// Every command field a non-claude built-in ships is documented. All three carry
	// both forms. The expectations are DERIVED from the agent command vars (never literal
	// copies), so a grammar change touches only internal/agent. They are compared in
	// their YAML-SCALAR form — the nested-shell dispatch commands contain single
	// quotes, which the renderer must double, so asserting the raw string would
	// demand exactly the invalid YAML the escaping exists to prevent.
	for _, cmd := range []string{
		agent.DefaultCodexInteractiveCommand,
		agent.DefaultCodexHeadlessCommand,
		agent.DefaultAgyInteractiveCommand,
		agent.DefaultAgyHeadlessCommand,
		agent.DefaultKimiInteractiveCommand,
		agent.DefaultKimiHeadlessCommand,
	} {
		quoted := configref.YAMLSingleQuoted(cmd)
		if !strings.Contains(out, quoted) {
			t.Errorf("providers block must document the built-in command %q (as the YAML scalar %s)", cmd, quoted)
		}
	}
	for _, want := range []string{
		"--dangerously-bypass-approvals-and-sandbox",
		"--dangerously-skip-permissions",
		"unattended stage workers cannot answer approval",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("providers block must document the built-in full-auto policy with %q", want)
		}
	}

	// agy carries NO {effort} placeholder — its model IDs embed the reasoning level
	// as an ID suffix, so a separate effort flag would fight the suffix. kimi's
	// approval flag is per FORM: its DISPATCH form carries none (`kimi -p` already
	// auto-approves and errors on --yolo/--auto), while its INTERACTIVE form is
	// exactly where --auto belongs — so the guard here is against an approval flag on
	// the dispatch line, not against --auto appearing anywhere in the document.
	for _, bad := range []string{
		"agy --dangerously-skip-permissions --model {model} --effort",
		"agy --dangerously-skip-permissions --model {model} -c model_reasoning_effort",
		"kimi --yolo",
		"kimi --auto -m {model} -p",
	} {
		if strings.Contains(out, bad) {
			t.Errorf("rendered provider commands must not contain %q", bad)
		}
	}

	// agy's block opens on interactive_command like the other pane-capable built-ins.
	if !strings.Contains(out, "  agy:\n    interactive_command: ") {
		t.Error("the agy block must open on interactive_command — every built-in is pane-capable")
	}
	// kimi's probe closed (260810-ki9v), so its block opens on interactive_command
	// like codex's — the rendered ordering a reader hoisting the block gets.
	if !strings.Contains(out, "  kimi:\n    interactive_command: ") {
		t.Error("the kimi block must open on interactive_command — kimi is pane-capable since its 2026-08-10 probe")
	}
	// The prose carries the corrected capability model and agy's gate behavior.
	for _, phrase := range []string{"Every supported agent", "HEADLESS prompt grammar", "ordinary readiness-gate judgment round", "settings.json"} {
		if !strings.Contains(out, phrase) {
			t.Errorf("providers block must explain pane-by-default capability with %q", phrase)
		}
	}

	if !strings.Contains(out, agent.DefaultHeadlessCommand) || !strings.Contains(out, "native: true") {
		t.Error("providers block must document claude's headless and native capabilities")
	}

	// The superseded ho9y framing must not survive: the non-claude built-ins are no longer
	// "template text only" awaiting an uncomment, and the Go table is no longer
	// claude-only. These literals render into every project's config fence, so a
	// stale claim there is a user-facing documentation inaccuracy.
	for _, retired := range []string{
		"No new built-in providers are added in Go",
		"template text only until you uncomment them",
		"uncomment and adapt a block to add that provider",
		"starter\n# TEMPLATE",
	} {
		if strings.Contains(out, retired) {
			t.Errorf("providers block still carries the retired ho9y claim %q", retired)
		}
	}

	// The built-ins are named as built-in, and the refresh policy that
	// replaced the grammar-only rule (260806-ywkx) is stated: the non-claude fills
	// ARE shipped, refreshed at kit-release cadence, and overridable in one line.
	for _, phrase := range []string{
		"FOUR built-in providers",
		"KIT-RELEASE cadence",
		"providers.<name>.profiles.<role>.model",
		// Only the non-claude providers render a `profiles:` map, so the prose must
		// say why claude's is absent — otherwise the reader concludes claude ships
		// none.
		"rendering choice, not a missing fill",
		// kimi's empty fill map is a deliberate design point, not an omission, so
		// the reference has to say so where a reader would otherwise see a gap.
		"kimi deliberately ships NO fills",
	} {
		if !strings.Contains(out, phrase) {
			t.Errorf("providers block must state %q (the non-claude fills ship, and are refreshed at kit cadence)", phrase)
		}
	}
	// The superseded j3cm framing must not survive either — it renders into
	// `fab config explain` and would now be a user-facing inaccuracy.
	for _, retired := range []string{
		"GRAMMAR ONLY",
		"fab ships no model ID",
		"resolves today with an EMPTY model",
	} {
		if strings.Contains(out, retired) {
			t.Errorf("providers block still carries the retired grammar-only claim %q", retired)
		}
	}
}

// TestConfigReferenceProviderBlocksParse is the promise the providers segment
// makes in its own prose: the four built-in blocks render LIVE and uniformly, and
// what you read is exactly what the built-in table resolves — so a reader who
// hoists a block out of their fence (strip the leading '# ' from every line)
// gets a working override, not a parse error.
//
// It is not a theoretical guard. The agy and kimi dispatch commands nest a shell
// (`sh -c '… -p "$(cat)"'`), so they carry single quotes inside a YAML
// single-quoted scalar; without doubling them the scalar closes early and a user
// who follows the documented instruction gets a parse error instead of a working
// override.
func TestConfigReferenceProviderBlocksParse(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}

	var parsed struct {
		Providers map[string]config.ProviderConfig `yaml:"providers"`
	}
	if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("the rendered reference must parse as valid YAML: %v", err)
	}

	for _, name := range agent.ProviderNames(nil) {
		want, _ := agent.ResolveProvider(nil, name)
		got, ok := parsed.Providers[name]
		if !ok {
			t.Errorf("no live `%s:` block found in the rendered reference", name)
			continue
		}
		if got.InteractiveCommand != want.InteractiveCommand || got.HeadlessCommand != want.HeadlessCommand {
			t.Errorf("rendered %s block = {%q, %q}, want the built-in {%q, %q}",
				name, got.InteractiveCommand, got.HeadlessCommand, want.InteractiveCommand, want.HeadlessCommand)
		}
	}
}

// TestConfigReferenceDocumentsProviderFill is the fill contract, reshaped by
// 260806-j9nh and completed by 260806-ywkx: the providers block documents the
// PER-ROLE `profiles` fill map and the fill precedence, and the registry row's
// Default exposes EVERY built-in's fills — claude's six roles (moved off the agent
// side by j9nh) plus codex's and agy's sparse maps (shipped by ywkx). kimi ships no
// fills at all, so it must project none. What the projection still refuses is the
// DEPRECATED flat pair, on every provider.
func TestConfigReferenceDocumentsProviderFill(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	for _, phrase := range []string{
		"profiles.<role>",
		"cross-role fallback",
		"flag > agent.profiles.<role> field > profiles.<role> >",
	} {
		if !strings.Contains(unwrapComment(out), phrase) {
			t.Errorf("providers block must document %q (the per-role fill map and its precedence)", phrase)
		}
	}
	// The retired flat-fill surface must not be advertised as the fill any more —
	// it survives only as the documented deprecated alias.
	if strings.Contains(out, "DEFAULT FILL") {
		t.Error("providers block still advertises the retired flat providers.<name>.model/.effort fill")
	}

	// The RENDERED reference must carry the shipped codex/agy fills, not just
	// the JSON projection below: those fill lines ARE the user-facing half of
	// R7, and without this assertion they can be dropped from providersSegment with
	// the whole suite staying green. Expectations are DERIVED from ResolveProvider,
	// shaped by the same omitempty rule the renderer applies — so codex's
	// effort-only rows and agy's model-only rows are both pinned, a fill bump in
	// defaults.yaml moves both sides together, and no model ID is written as a
	// literal here.
	for _, name := range []string{"codex", "agy"} {
		prov, ok := agent.ResolveProvider(nil, name)
		if !ok {
			t.Errorf("built-in provider %q does not resolve", name)
			continue
		}
		if len(prov.Profiles) == 0 {
			t.Errorf("built-in provider %q resolves no per-role fills for the reference to render", name)
		}
		for _, role := range agent.RoleNames() {
			fill, ok := prov.Profiles[role]
			if !ok {
				continue // sparse map: an omitted role takes this provider's `default`
			}
			var set []string
			if fill.Model != "" {
				set = append(set, "model: "+fill.Model)
			}
			if fill.Effort != "" {
				set = append(set, "effort: "+fill.Effort)
			}
			want := "      " + role + ": { " + strings.Join(set, ", ") + " }"
			if !strings.Contains(out, want) {
				t.Errorf("providers block must render %s's %s fill as %q", name, role, want)
			}
		}
	}

	// The JSON registry row must advertise all four built-ins and no fill values.
	jsonOut, err := configref.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}
	// Decode leniently (row defaults have per-field shapes) and pick the providers
	// row out of the array.
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &rows); err != nil {
		t.Fatalf("--json output did not decode: %v", err)
	}
	var row map[string]any
	for _, r := range rows {
		if r["key"] == "providers" {
			row = r
			break
		}
	}
	if row == nil {
		t.Fatal("--json output has no `providers` row")
	}
	defaults, ok := row["default"].(map[string]any)
	if !ok {
		t.Fatalf("providers row default = %v, want an object of built-in providers", row["default"])
	}
	for _, name := range []string{"claude", "codex", "agy", "kimi"} {
		entry, ok := defaults[name].(map[string]any)
		if !ok {
			t.Errorf("providers default must advertise the built-in %q, got %v", name, defaults[name])
			continue
		}
		// The flat fill is gone from the shipped surface for EVERY built-in.
		if _, ok := entry["model"]; ok {
			t.Errorf("built-in %q must carry no flat model fill in the registry default", name)
		}
		if _, ok := entry["effort"]; ok {
			t.Errorf("built-in %q must carry no flat effort fill in the registry default", name)
		}
		profiles, hasProfiles := entry["profiles"].(map[string]any)
		if name == "kimi" {
			// kimi ships NO fills, so it must project none — an empty `profiles`
			// object would advertise a fill surface it deliberately does not have.
			if hasProfiles {
				t.Errorf("built-in kimi must project no per-role profiles (it ships none), got %v", profiles)
			}
			continue
		}
		if !hasProfiles {
			t.Errorf("built-in %q must carry its per-role profiles in the registry default, got %v", name, entry)
			continue
		}
		// Every built-in's map must at least carry `default` — the cross-role
		// fallback that makes a SPARSE map well-defined for the roles it omits.
		if _, ok := profiles[agent.RoleDefault]; !ok {
			t.Errorf("built-in %q's registry default has no %q fill (the cross-role fallback), got %v", name, agent.RoleDefault, profiles)
		}
		for role := range profiles {
			if !agent.IsRoleName(role) {
				t.Errorf("built-in %q's registry default carries a fill for %q, which is not a role name", name, role)
			}
		}
		if name != agent.DefaultProviderName {
			continue
		}
		// claude is the one built-in whose map is EXHAUSTIVE. Derived from
		// agent.RoleNames so a role added or renamed surfaces here rather than
		// drifting silently.
		for _, role := range agent.RoleNames() {
			if _, ok := profiles[role]; !ok {
				t.Errorf("built-in claude's registry default is missing the %q role fill", role)
			}
		}
	}
	desc, _ := row["description"].(string)
	for _, phrase := range []string{"per-role fills", "four built-in providers", "kit-release cadence"} {
		if !strings.Contains(desc, phrase) {
			t.Errorf("providers row description must mention %q, got %q", phrase, desc)
		}
	}
}

// TestConfigReferenceProvidersDefaultTracksAgentTable is the DERIVATION contract:
// the providers row's registry default is built by walking agent.ProviderNames(nil)
// and resolving each name through agent.ResolveProvider(nil, …) — it is NOT a
// hand-maintained list of built-in names. Asserting the key set EQUALS the agent
// table (rather than merely contains the three names known today) is what makes a
// built-in added or renamed in defaults.yaml surface here automatically instead of
// silently drifting out of `fab config explain --json`.
//
// It also pins the projection: the two command fields and the per-role fill map
// cross over verbatim from ResolveProvider, while the DEPRECATED flat pair never
// does — the contract that keeps a nonexistent model/effort fill from being
// asserted (see configref.providerDefault).
func TestConfigReferenceProvidersDefaultTracksAgentTable(t *testing.T) {
	jsonOut, err := configref.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &rows); err != nil {
		t.Fatalf("--json output did not decode: %v", err)
	}
	var row map[string]any
	for _, r := range rows {
		if r["key"] == "providers" {
			row = r
			break
		}
	}
	if row == nil {
		t.Fatal("--json output has no `providers` row")
	}
	defaults, ok := row["default"].(map[string]any)
	if !ok {
		t.Fatalf("providers row default = %v, want an object of built-in providers", row["default"])
	}

	// Key set must EQUAL the agent table's built-in set (nil cfg = no project
	// overrides), not merely overlap it.
	want := agent.ProviderNames(nil)
	got := make([]string, 0, len(defaults))
	for name := range defaults {
		got = append(got, name)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("providers default keys = %v, want agent.ProviderNames(nil) = %v", got, want)
	}

	// Each entry's commands and native capability must be ResolveProvider's verbatim.
	for _, name := range want {
		p, ok := agent.ResolveProvider(nil, name)
		if !ok {
			t.Errorf("agent.ProviderNames reported %q but ResolveProvider does not resolve it", name)
			continue
		}
		entry, ok := defaults[name].(map[string]any)
		if !ok {
			t.Errorf("providers default is missing the built-in %q, got %v", name, defaults[name])
			continue
		}
		for field, wantCmd := range map[string]string{
			"interactive_command": p.InteractiveCommand,
			"headless_command":    p.HeadlessCommand,
		} {
			raw, present := entry[field]
			if wantCmd == "" {
				if present {
					t.Errorf("provider %q: %s must be absent when ResolveProvider carries none, got %v", name, field, raw)
				}
				continue
			}
			if !present {
				t.Errorf("provider %q: %s missing, want %q", name, field, wantCmd)
				continue
			}
			if raw != wantCmd {
				t.Errorf("provider %q: %s = %v, want ResolveProvider's %q", name, field, raw, wantCmd)
			}
		}
		if gotNative, _ := entry["native"].(bool); gotNative != p.Native {
			t.Errorf("provider %q: native = %v, want %v", name, gotNative, p.Native)
		}

		// The per-role fill map must cross over VERBATIM too — every shipped role,
		// model AND effort. Comparing the whole object is what makes a dropped role,
		// a rewritten value, or an asserted-but-unshipped field fail here; the
		// command assertions above would pass regardless. `want` is built with the
		// same omitempty rule the JSON shape uses, so agy's model-only rows assert
		// no empty effort, and a provider carrying no fills asserts no `profiles` key.
		var wantProfiles map[string]any
		if len(p.Profiles) > 0 {
			wantProfiles = make(map[string]any, len(p.Profiles))
			for role, fill := range p.Profiles {
				projected := map[string]any{}
				if fill.Model != "" {
					projected["model"] = fill.Model
				}
				if fill.Effort != "" {
					projected["effort"] = fill.Effort
				}
				wantProfiles[role] = projected
			}
		}
		gotProfiles, _ := entry["profiles"].(map[string]any)
		if !reflect.DeepEqual(gotProfiles, wantProfiles) {
			t.Errorf("provider %q: profiles = %v, want ResolveProvider's %v (verbatim, per role and per field)", name, gotProfiles, wantProfiles)
		}
	}
}

// TestConfigReferenceRetiresLegacyKeys guards that the removed keys no longer
// appear in the reference: review_tools (retired to code-review.md § Review Tools),
// agent.spawn_command (relocated to providers.claude.interactive_command), and
// branch_prefix (retired outright — `fab batch switch` names branches by the
// change folder name, the one convention /git-branch and naming.md share).
func TestConfigReferenceRetiresLegacyKeys(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	for _, gone := range []string{"review_tools", "spawn_command", "branch_prefix"} {
		if containsKeyToken(out, gone) {
			t.Errorf("retired key %q must not appear in the reference", gone)
		}
	}
}

// TestConfigReferenceJSONIsValidAndByteStable is the --json VALIDITY + STABILITY
// contract: `fab config explain --json` emits a well-formed JSON array of
// per-field objects, and repeated renders are byte-identical (the same
// byte-stable stdout contract Change 2/3 tooling relies on).
func TestConfigReferenceJSONIsValidAndByteStable(t *testing.T) {
	first, err := configref.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}
	second, err := configref.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}
	if first != second {
		t.Error("`fab config explain --json` output is not byte-stable across renders")
	}

	var arr []map[string]any
	if err := json.Unmarshal([]byte(first), &arr); err != nil {
		t.Fatalf("--json output is not valid JSON: %v", err)
	}
	if len(arr) == 0 {
		t.Fatal("--json output parsed to an empty array")
	}
	for i, obj := range arr {
		for _, required := range []string{"key", "description", "scope", "advertise"} {
			if _, ok := obj[required]; !ok {
				t.Errorf("--json element %d is missing required field %q", i, required)
			}
		}
		// default is present on every element (may be null); renamed_from is
		// omitted (omitempty) on every row EXCEPT the ones in the rename ledger,
		// where it must be emitted so a consumer can carry the old key forward.
		if _, ok := obj["default"]; !ok {
			t.Errorf("--json element %d (%v) is missing the `default` field", i, obj["key"])
		}
		key, _ := obj["key"].(string)
		got, present := obj["renamed_from"]
		if want := wantRenamedFrom[key]; want == "" {
			if present {
				t.Errorf("--json element %d (%s) should omit `renamed_from` (no rename on this row, omitempty)", i, key)
			}
		} else if got != want {
			t.Errorf("--json element %d (%s) renamed_from = %v, want %q", i, key, got, want)
		}
	}
}

// TestConfigReferenceJSONEmptyDefaultConvention pins the uniform empty-default
// convention (T002 / docs/specs/config.md § Default semantics): a field with no
// meaningful built-in default emits JSON `null`, NEVER a typed empty (`[]`, `{}`,
// `""`). This is the single "cascade falls back to absent" signal Change 2's
// resolver consumes; a typed empty would leak a Go-side implementation detail with
// no cascade meaning. Conversely, a non-null `default` must denote a real built-in
// value (the four built-in providers, the six role profiles, and
// dispatch.mode's `native` today). The forbidden shapes are the ones that could
// stand in for "nothing" (`[]`, `{}`, `""`).
func TestConfigReferenceJSONEmptyDefaultConvention(t *testing.T) {
	out, err := configref.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("--json output is not valid JSON: %v", err)
	}

	// The only rows with a real built-in default today. Every other row is
	// "no built-in default" and MUST render as JSON null (not [], {}, or "").
	hasDefault := map[string]bool{
		"providers":             true,
		"agent.profiles":        true,
		"agent.session":         true, // the knob's built-in value IS claude, not "absent"
		"agent.workers":         true,
		"dispatch.mode":         true, // string: the built-in mode IS a real default, not "absent"
		"dispatch.column_width": true, // int: an absent yaml int reads as unset, so the built-in width is real
		"dispatch.reap_done":    true, // bool defaulting TRUE — modeled as *bool so absent ≠ false
		"autopilot.merge_mode":  true, // string: the built-in merge mode IS a real default, not "absent"
	}
	for _, obj := range arr {
		key, _ := obj["key"].(string)
		def, present := obj["default"]
		if !present {
			t.Errorf("field %q is missing the `default` field", key)
			continue
		}
		if hasDefault[key] {
			if def == nil {
				t.Errorf("field %q should carry a real built-in default, got null", key)
			}
			continue
		}
		if def != nil {
			t.Errorf("field %q has no built-in default and must emit JSON null (uniform empty-default convention), got %#v", key, def)
		}
	}
}

// TestConfigReferenceJSONKeysMatchYAML is the NO-DRIFT contract between the two
// renderings: every key the JSON dump advertises must be documented in the
// commented-YAML reference (segment-wise, mirroring the binary-key coverage
// check), so the machine-readable and human-readable views cannot silently
// diverge. Also asserts the JSON key set equals the registry's FieldKeys().
func TestConfigReferenceJSONKeysMatchYAML(t *testing.T) {
	yaml, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	jsonOut, err := configref.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}

	var arr []struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &arr); err != nil {
		t.Fatalf("--json output is not valid JSON: %v", err)
	}

	jsonKeys := make([]string, len(arr))
	for i, e := range arr {
		jsonKeys[i] = e.Key
		// Each dotted key must be documented in the YAML: every segment appears
		// as a key token (the reference documents some keys in dotted-prose form,
		// so a per-segment presence check is the robust parity guard — same
		// technique as TestConfigReferenceCoversBinaryKeys).
		for _, seg := range strings.Split(e.Key, ".") {
			if !containsKeyToken(yaml, seg) {
				t.Errorf("JSON key %q (segment %q) is not documented in the commented-YAML reference (renderings drifted)", e.Key, seg)
			}
		}
	}

	registryKeys, err := configref.FieldKeys()
	if err != nil {
		t.Fatalf("FieldKeys returned an error: %v", err)
	}
	if !reflect.DeepEqual(jsonKeys, registryKeys) {
		t.Errorf("--json key order/set does not match the registry FieldKeys():\n json:     %v\n registry: %v", jsonKeys, registryKeys)
	}
}

// TestConfigReferenceRegistryLint is the FAIL-LOUD registry contract: every
// field row has a non-empty description and a valid scope ∈ {project, system,
// both}. The registry constructor (configref.Fields) runs this lint itself, so a
// row added without metadata fails at construction — this test asserts the
// invariant holds for the shipped table (and that Fields does not error).
func TestConfigReferenceRegistryLint(t *testing.T) {
	fields, err := configref.Fields()
	if err != nil {
		t.Fatalf("Fields returned an error (registry lint or role-profile invariant failed): %v", err)
	}
	if len(fields) == 0 {
		t.Fatal("Fields returned an empty registry")
	}
	validScopes := map[configref.Scope]bool{
		configref.ScopeProject: true,
		configref.ScopeSystem:  true,
		configref.ScopeBoth:    true,
	}
	for _, f := range fields {
		if strings.TrimSpace(f.Description) == "" {
			t.Errorf("field %q has an empty description", f.Key)
		}
		if !validScopes[f.Scope] {
			t.Errorf("field %q has invalid scope %q (want project/system/both)", f.Key, f.Scope)
		}
		// renamed_from is carried only by rows whose key actually moved. Pinning the
		// exact set (rather than merely allowing any value) is what keeps a stray
		// RenamedFrom from being added without a matching migration.
		if want := wantRenamedFrom[f.Key]; f.RenamedFrom != want {
			t.Errorf("field %q RenamedFrom = %q, want %q", f.Key, f.RenamedFrom, want)
		}
	}
}

// wantRenamedFrom is the registry's complete rename ledger. agent.tiers →
// agent.profiles (260806-j9nh) was the first; providers' nested command fields
// (session_command/dispatch_command → interactive_command/headless_command,
// 2.19.0) carry the metadata informationally — the mechanical carry is
// top-level only, so their on-disk rewrite ships as the 2.18.1-to-2.19.0
// migration. Every other row must leave RenamedFrom empty so `renamed_from`
// stays omitted from --json.
var wantRenamedFrom = map[string]string{
	"agent.profiles": "agent.tiers",
	"providers":      "providers.<name>.session_command, providers.<name>.dispatch_command",
}

// TestConfigReferenceScopeAssignments pins the decision-6 scope taxonomy: the
// preference-class fields (agent.profiles, providers) are `both`; the
// semantics-class fields and the one unenumerated field (stage_hooks) are
// `project`. (fab_version left config.yaml in 260708-j0qm and no longer carries a
// scope.) Enforcement landed in Change 2; the assignments are consumed as data, so
// they are pinned.
func TestConfigReferenceScopeAssignments(t *testing.T) {
	fields, err := configref.Fields()
	if err != nil {
		t.Fatalf("Fields returned an error: %v", err)
	}
	got := make(map[string]configref.Scope, len(fields))
	for _, f := range fields {
		got[f.Key] = f.Scope
	}
	want := map[string]configref.Scope{
		"project.name":               configref.ScopeProject,
		"project.description":        configref.ScopeProject,
		"project.linear_workspace":   configref.ScopeProject,
		"source_paths":               configref.ScopeProject,
		"test_paths":                 configref.ScopeProject,
		"true_impact_exclude":        configref.ScopeProject,
		"checklist.extra_categories": configref.ScopeProject,
		"consolidate.detectors":      configref.ScopeProject,
		"providers":                  configref.ScopeBoth,
		"agent.session":              configref.ScopeBoth,
		"agent.workers":              configref.ScopeBoth,
		"agent.profiles":             configref.ScopeBoth,
		"dispatch.mode":              configref.ScopeBoth,
		"dispatch.column_width":      configref.ScopeBoth,
		"dispatch.reap_done":         configref.ScopeBoth,
		"autopilot.merge_mode":       configref.ScopeBoth,
		"stage_hooks":                configref.ScopeProject,
	}
	for key, wantScope := range want {
		gotScope, ok := got[key]
		if !ok {
			t.Errorf("registry is missing expected field %q", key)
			continue
		}
		if gotScope != wantScope {
			t.Errorf("field %q scope = %q, want %q (decision 6)", key, gotScope, wantScope)
		}
	}
}

// TestConfigReferenceCommandJSONFlag drives the cobra command end to end with
// --json: it prints the JSON table and exits 0, matches configref.RenderJSON(),
// rejects an extra positional arg (cobra.NoArgs still applies), and leaves the
// no-flag output contract-identical to configref.Render().
func TestConfigReferenceCommandJSONFlag(t *testing.T) {
	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reference", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("`config reference --json` returned an error: %v", err)
	}
	want, err := configref.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}
	if out.String() != want {
		t.Error("`config reference --json` stdout does not match configref.RenderJSON()")
	}

	// No-flag output is the commented YAML, unchanged.
	cmdYAML := configCmd()
	var yamlOut strings.Builder
	cmdYAML.SetOut(&yamlOut)
	cmdYAML.SetErr(&yamlOut)
	cmdYAML.SetArgs([]string{"reference"})
	if err := cmdYAML.Execute(); err != nil {
		t.Fatalf("`config reference` returned an error: %v", err)
	}
	wantYAML, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	if yamlOut.String() != wantYAML {
		t.Error("`config reference` (no flag) stdout does not match configref.Render()")
	}

	// Extra positional arg is still rejected with --json.
	cmdErr := configCmd()
	var errBuf strings.Builder
	cmdErr.SetOut(&errBuf)
	cmdErr.SetErr(&errBuf)
	cmdErr.SetArgs([]string{"reference", "--json", "extra"})
	if err := cmdErr.Execute(); err == nil {
		t.Error("`config reference --json extra` should be rejected (cobra.NoArgs)")
	}
}

// TestConfigReferenceConsolidateDetectors pins the `consolidate.detectors`
// registry row added for /code-dedupe (260728-4v91, as /fab-dedupe). It is a SKILL-consumed key
// (no Config struct field — markdown reads it), so no reflection-based coverage
// test would catch its loss; this test is the guard. Three properties are pinned:
// the row's metadata (nil default per the empty-default convention, project
// scope, advertised), the commented-out rendering (a live `consolidate:` block
// would opt every project in), and the deliberate ABSENCE of the
// `consolidate.memory_file` key rejected at intake (the memory home is
// hardcoded in the skill prose).
func TestConfigReferenceConsolidateDetectors(t *testing.T) {
	// Isolate HOME so the cascade cannot merge the developer's real system
	// config over the reference (same discipline as TestConfigReferenceRoundTrips).
	t.Setenv("HOME", t.TempDir())

	fields, err := configref.Fields()
	if err != nil {
		t.Fatalf("Fields returned an error: %v", err)
	}
	var row *configref.Field
	for i := range fields {
		switch fields[i].Key {
		case "consolidate.detectors":
			row = &fields[i]
		case "consolidate.memory_file":
			t.Error("consolidate.memory_file was deliberately NOT shipped (the memory home is hardcoded to docs/memory/_shared/utilities.md); remove the row")
		}
	}
	if row == nil {
		t.Fatal("registry is missing the consolidate.detectors row")
	}
	if row.Default != nil {
		t.Errorf("consolidate.detectors Default = %#v, want nil (empty-default convention: no built-in default)", row.Default)
	}
	if row.Scope != configref.ScopeProject {
		t.Errorf("consolidate.detectors Scope = %q, want %q", row.Scope, configref.ScopeProject)
	}
	if !row.Advertise {
		t.Error("consolidate.detectors must be advertised (it is scaffolded into the managed fence)")
	}
	if strings.TrimSpace(row.Segment) == "" {
		t.Error("consolidate.detectors carries no rendered Segment, so `fab config explain` would not document it")
	}

	// The placeholders are substituted SHELL-QUOTED (PR #520 review). A template
	// is run through a shell, so an unquoted {paths}/{out} would word-split — or
	// execute — a scope path containing a space or a shell metacharacter. Both
	// the prose description and the scaffolded comment must say so, since the
	// substitution is performed by the agent reading this text.
	if !strings.Contains(row.Description, "shell-quoted") {
		t.Error("consolidate.detectors Description must state that {paths}/{out} are substituted shell-quoted")
	}
	if !strings.Contains(row.Segment, "shell-quoted") {
		t.Error("consolidate.detectors Segment must state that {paths}/{out} are substituted shell-quoted")
	}

	// The YAML reference documents the key, and it is COMMENTED — a live
	// `consolidate:` block would enable a detector sweep for every project.
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	if !containsKeyToken(out, "detectors") {
		t.Error("consolidate.detectors is not documented in `fab config explain`")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "consolidate:") {
			t.Error("the consolidate block must be commented-out in the reference (parsed as live)")
		}
	}

	// The JSON dump carries the same row with the same metadata.
	jsonOut, err := configref.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}
	var arr []struct {
		Key       string `json:"key"`
		Default   any    `json:"default"`
		Scope     string `json:"scope"`
		Advertise bool   `json:"advertise"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &arr); err != nil {
		t.Fatalf("--json output is not valid JSON: %v", err)
	}
	found := false
	for _, e := range arr {
		if e.Key != "consolidate.detectors" {
			continue
		}
		found = true
		if e.Default != nil {
			t.Errorf("--json consolidate.detectors default = %#v, want null", e.Default)
		}
		if e.Scope != string(configref.ScopeProject) {
			t.Errorf("--json consolidate.detectors scope = %q, want %q", e.Scope, configref.ScopeProject)
		}
		if !e.Advertise {
			t.Error("--json consolidate.detectors advertise = false, want true")
		}
	}
	if !found {
		t.Error("--json dump is missing consolidate.detectors")
	}
}

// yamlKeySegments walks a struct type and returns the set of every yaml key
// segment reachable from it. Descends into nested structs and map value types
// (a map's value type contributes its own struct's segments). Returns segments
// (leaf key names), not full dotted paths, because the reference documents some
// keys in dotted-prose form (`agent.profiles`, `stage_hooks.<stage>.pre`); a
// per-segment presence check catches a new nested field regardless of the
// prose form used.
func yamlKeySegments(t reflect.Type) map[string]struct{} {
	segments := make(map[string]struct{})
	var walk func(rt reflect.Type)
	walk = func(rt reflect.Type) {
		for rt.Kind() == reflect.Pointer {
			rt = rt.Elem()
		}
		switch rt.Kind() {
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				tag := f.Tag.Get("yaml")
				name := strings.Split(tag, ",")[0]
				if name != "" && name != "-" {
					segments[name] = struct{}{}
				}
				walk(f.Type)
			}
		case reflect.Map:
			// The map key is a free-form stage/role name (not a fixed key), so
			// descend only into the value type for its struct fields.
			walk(rt.Elem())
		case reflect.Slice, reflect.Array:
			walk(rt.Elem())
		}
	}
	walk(t)
	return segments
}

// keyTokenBoundary matches a word boundary for a config key token (letters,
// digits, underscore). Used so `test_paths` matches `test_paths:` but a search
// for `paths` would not spuriously match `test_paths`.
// unwrapComment flattens a rendered comment block into one whitespace-normalized
// line: each line's leading `#` marker is stripped and the remainders are joined
// with single spaces. Reference prose is hard-wrapped, so a SEMANTIC pin ("this
// sentence is documented") asserted against the raw render breaks whenever a
// segment is re-wrapped — and a re-wrap is not a contract change. Assert
// wrap-sensitive facts (indentation, column alignment) on the raw output.
func unwrapComment(rendered string) string {
	lines := strings.Split(rendered, "\n")
	stripped := make([]string, 0, len(lines))
	for _, line := range lines {
		stripped = append(stripped, strings.TrimPrefix(strings.TrimSpace(line), "#"))
	}
	return strings.Join(strings.Fields(strings.Join(stripped, " ")), " ")
}

func containsKeyToken(haystack, token string) bool {
	re := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(token) + `([^A-Za-z0-9_]|$)`)
	return re.MatchString(haystack)
}

// TestConfigReferenceDispatchMode: the `dispatch.mode` row (the preference
// ceiling) is present, correctly scoped, advertised, and rendered
// COMMENTED with the semantics a reader needs to decide whether to set it. The row
// is the field's only discoverability surface, so the guard covers both the
// metadata and the rendered prose.
func TestConfigReferenceDispatchMode(t *testing.T) {
	// Isolate HOME so the cascade cannot merge the developer's real system config
	// over the reference (the TestConfigReferenceRoundTrips discipline).
	t.Setenv("HOME", t.TempDir())

	fields, err := configref.Fields()
	if err != nil {
		t.Fatalf("Fields returned an error: %v", err)
	}
	var row *configref.Field
	for i := range fields {
		if fields[i].Key == "dispatch.mode" {
			row = &fields[i]
		}
	}
	if row == nil {
		t.Fatal("registry is missing the dispatch.mode row")
	}
	if row.Default != config.DefaultDispatchMode {
		t.Errorf("dispatch.mode Default = %#v, want %q", row.Default, config.DefaultDispatchMode)
	}
	// Scope `both` is load-bearing: a project-scoped field would be PRUNED out of
	// ~/.fab-kit/config.yaml, defeating the machine-wide opt-in.
	if row.Scope != configref.ScopeBoth {
		t.Errorf("dispatch.mode Scope = %q, want %q (settable machine-wide)", row.Scope, configref.ScopeBoth)
	}
	if !row.Advertise {
		t.Error("dispatch.mode must be advertised (it is scaffolded into the managed fence)")
	}

	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	// Rendered COMMENTED: a live `dispatch:` block would be a no-op today but would
	// register as an override under presence=intent.
	if strings.Contains(out, "\ndispatch:") {
		t.Error("the dispatch block must be rendered COMMENTED (a live block would register as an override)")
	}
	if !strings.Contains(out, "#   mode: "+config.DefaultDispatchMode) {
		t.Errorf("the reference must scaffold `mode: %s` commented.\n--- got ---\n%s", config.DefaultDispatchMode, out)
	}
	for _, want := range []string{"pane", "native", "headless", "interactive_command", "headless_command", "never ascending"} {
		if !strings.Contains(row.Segment, want) {
			t.Errorf("dispatch.mode Segment must mention %q", want)
		}
	}

	// The commented scaffold must parse as an inert config: mode stays native.
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmp, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadPath(tmp)
	if err != nil {
		t.Fatalf("the rendered reference must parse: %v", err)
	}
	if got := cfg.GetDispatchMode(); got != config.DefaultDispatchMode {
		t.Errorf("the reference's dispatch block must be inert, got mode %q", got)
	}
}

// TestConfigReferenceDispatchColumnWidth: the `dispatch.column_width` row (the
// pane-worker column width) is present, correctly scoped, advertised, and carries
// the CANONICAL default rather than a literal copy. Its rendered prose lives inside
// the shared `dispatch:` segment — `dispatch` is one YAML block, so a second
// `# dispatch:` parent would collide into a duplicate key if a reader uncommented
// both — which is why this row's own Segment is deliberately empty (the
// project.name / project.description precedent).
func TestConfigReferenceDispatchColumnWidth(t *testing.T) {
	// Isolate HOME so the cascade cannot merge the developer's real system config
	// over the reference (the TestConfigReferenceRoundTrips discipline).
	t.Setenv("HOME", t.TempDir())

	fields, err := configref.Fields()
	if err != nil {
		t.Fatalf("Fields returned an error: %v", err)
	}
	var row, mode *configref.Field
	for i := range fields {
		switch fields[i].Key {
		case "dispatch.column_width":
			row = &fields[i]
		case "dispatch.mode":
			mode = &fields[i]
		}
	}
	if row == nil {
		t.Fatal("registry is missing the dispatch.column_width row")
	}
	// The canonical built-in default, sourced from the config symbol — a literal
	// here would be the second copy the registry exists to avoid.
	if row.Default != config.DefaultDispatchColumnWidth {
		t.Errorf("dispatch.column_width Default = %#v, want %d (config.DefaultDispatchColumnWidth)",
			row.Default, config.DefaultDispatchColumnWidth)
	}
	// Scope `both` is load-bearing: a project-scoped field would be PRUNED out of
	// ~/.fab-kit/config.yaml, defeating the machine-wide preference.
	if row.Scope != configref.ScopeBoth {
		t.Errorf("dispatch.column_width Scope = %q, want %q (settable machine-wide)", row.Scope, configref.ScopeBoth)
	}
	if !row.Advertise {
		t.Error("dispatch.column_width must be advertised (it is scaffolded into the managed fence)")
	}
	if row.Segment != "" {
		t.Error("dispatch.column_width must carry NO Segment of its own — it is rendered inside the shared dispatch.mode Segment")
	}
	if mode == nil {
		t.Fatal("registry is missing the dispatch.mode row (the shared segment's owner)")
	}
	// The shared segment must document the width: this row's only discoverability
	// surface, so the semantics a reader needs to set it live there.
	for _, want := range []string{"column_width", "-h -l <n>%", "1..99", "pre-3.1"} {
		if !strings.Contains(mode.Segment, want) {
			t.Errorf("the shared dispatch Segment must mention %q (the sizing semantics and the degrade rule)", want)
		}
	}

	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	// Rendered COMMENTED beside mode, under ONE `dispatch:` parent.
	wantScaffold := "#   column_width: " + strconv.Itoa(config.DefaultDispatchColumnWidth)
	if !strings.Contains(out, wantScaffold) {
		t.Errorf("the reference must scaffold %q commented.\n--- got ---\n%s", wantScaffold, out)
	}
	if n := strings.Count(out, "# dispatch:"); n != 1 {
		t.Errorf("the reference renders %d `# dispatch:` parents, want exactly 1 (both keys share one block, or uncommenting both would duplicate the key)", n)
	}

	// The commented scaffold must parse as an inert config: the width stays at its
	// default rather than registering as an override.
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmp, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadPath(tmp)
	if err != nil {
		t.Fatalf("the rendered reference must parse: %v", err)
	}
	if got := cfg.GetDispatchColumnWidth(); got != config.DefaultDispatchColumnWidth {
		t.Errorf("the reference's dispatch block must be inert, got column_width %d", got)
	}
}

// TestConfigReferenceDispatchReapDone: the `dispatch.reap_done` row (done-worker
// pane reaping) is present, correctly scoped, advertised, and carries the CANONICAL
// default rather than a literal copy. Like `dispatch.column_width` it renders inside
// the shared `dispatch:` segment and therefore carries no Segment of its own — but
// unlike both siblings its default is TRUE, so the rendered scaffold must not read
// as a live override and the parsed reference must still resolve to true.
func TestConfigReferenceDispatchReapDone(t *testing.T) {
	// Isolate HOME so the cascade cannot merge the developer's real system config
	// over the reference (the TestConfigReferenceRoundTrips discipline).
	t.Setenv("HOME", t.TempDir())

	fields, err := configref.Fields()
	if err != nil {
		t.Fatalf("Fields returned an error: %v", err)
	}
	var row, mode *configref.Field
	for i := range fields {
		switch fields[i].Key {
		case "dispatch.reap_done":
			row = &fields[i]
		case "dispatch.mode":
			mode = &fields[i]
		}
	}
	if row == nil {
		t.Fatal("registry is missing the dispatch.reap_done row")
	}
	// The canonical built-in default, sourced from the config symbol.
	if row.Default != config.DefaultDispatchReapDone {
		t.Errorf("dispatch.reap_done Default = %#v, want %v (config.DefaultDispatchReapDone)",
			row.Default, config.DefaultDispatchReapDone)
	}
	// Scope `both` is load-bearing: a project-scoped field would be PRUNED out of
	// ~/.fab-kit/config.yaml, defeating the machine-wide opt-out.
	if row.Scope != configref.ScopeBoth {
		t.Errorf("dispatch.reap_done Scope = %q, want %q (settable machine-wide)", row.Scope, configref.ScopeBoth)
	}
	if !row.Advertise {
		t.Error("dispatch.reap_done must be advertised (it is scaffolded into the managed fence)")
	}
	if row.Segment != "" {
		t.Error("dispatch.reap_done must carry NO Segment of its own — it is rendered inside the shared dispatch.mode Segment")
	}
	if mode == nil {
		t.Fatal("registry is missing the dispatch.mode row (the shared segment's owner)")
	}
	// The shared segment is this row's only discoverability surface, so the two
	// semantics a reader must be able to learn there are what reap DOES and — the
	// load-bearing half — what it deliberately does NOT do.
	for _, want := range []string{"reap_done", "done", "headless", ".fab-dispatch/"} {
		if !strings.Contains(mode.Segment, want) {
			t.Errorf("the shared dispatch Segment must mention %q (the reap guard and the no-state-cleanup rule)", want)
		}
	}

	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	// Rendered COMMENTED beside its two siblings, under ONE `dispatch:` parent.
	wantScaffold := "#   reap_done: " + strconv.FormatBool(config.DefaultDispatchReapDone)
	if !strings.Contains(out, wantScaffold) {
		t.Errorf("the reference must scaffold %q commented.\n--- got ---\n%s", wantScaffold, out)
	}
	if n := strings.Count(out, "# dispatch:"); n != 1 {
		t.Errorf("the reference renders %d `# dispatch:` parents, want exactly 1 (all three keys share one block)", n)
	}

	// The commented scaffold must parse as an inert config. For a default-TRUE knob
	// "inert" means the accessor still reports TRUE — which is exactly the case a
	// plain bool would have gotten wrong.
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmp, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadPath(tmp)
	if err != nil {
		t.Fatalf("the rendered reference must parse: %v", err)
	}
	if got := cfg.GetDispatchReapDone(); got != config.DefaultDispatchReapDone {
		t.Errorf("the reference's dispatch block must be inert, got reap_done %v", got)
	}
}

// TestConfigReferenceAutopilotMergeMode: the `autopilot.merge_mode` row
// (standing merge-topology preference) is present, correctly scoped, advertised,
// sources its default from the canonical config symbol, and owns its own
// Segment/ShortSegment — no other row renders an `autopilot:` parent (unlike
// dispatch.column_width, which rides dispatch.mode's segment).
func TestConfigReferenceAutopilotMergeMode(t *testing.T) {
	// Isolate HOME so the cascade cannot merge the developer's real system config
	// over the reference (the TestConfigReferenceRoundTrips discipline).
	t.Setenv("HOME", t.TempDir())

	fields, err := configref.Fields()
	if err != nil {
		t.Fatalf("Fields returned an error: %v", err)
	}
	var row *configref.Field
	for i := range fields {
		if fields[i].Key == "autopilot.merge_mode" {
			row = &fields[i]
		}
	}
	if row == nil {
		t.Fatal("registry is missing the autopilot.merge_mode row")
	}
	// The canonical built-in default, sourced from the config symbol — never a
	// copied literal.
	if row.Default != config.DefaultAutopilotMergeMode {
		t.Errorf("autopilot.merge_mode Default = %#v, want %q (config.DefaultAutopilotMergeMode)",
			row.Default, config.DefaultAutopilotMergeMode)
	}
	// Scope `both` is load-bearing: a project-scoped field would be PRUNED out of
	// ~/.fab-kit/config.yaml, defeating the set-once-machine-wide preference.
	if row.Scope != configref.ScopeBoth {
		t.Errorf("autopilot.merge_mode Scope = %q, want %q (settable machine-wide)", row.Scope, configref.ScopeBoth)
	}
	if !row.Advertise {
		t.Error("autopilot.merge_mode must be advertised (it is scaffolded into the managed fence)")
	}
	if row.Segment == "" || row.ShortSegment == "" {
		t.Error("autopilot.merge_mode must carry its OWN Segment and ShortSegment — no other row owns the `autopilot:` parent")
	}
	// The rendered segment names the resolution ladder and the valid set (the
	// accepted values interpolate the canonical Go list).
	for _, want := range []string{"--mode", "cherry-pick-ladder", "merge-auto", "stacked-prs"} {
		if !strings.Contains(row.Segment, want) {
			t.Errorf("the autopilot Segment must mention %q", want)
		}
	}

	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	// Rendered COMMENTED under its own `autopilot:` parent, exactly once.
	wantScaffold := "#   merge_mode: " + config.DefaultAutopilotMergeMode
	if !strings.Contains(out, wantScaffold) {
		t.Errorf("the reference must scaffold %q commented.\n--- got ---\n%s", wantScaffold, out)
	}
	if n := strings.Count(out, "# autopilot:"); n != 1 {
		t.Errorf("the reference renders %d `# autopilot:` parents, want exactly 1", n)
	}

	// The commented scaffold must parse as an inert config: the accessor still
	// reports the built-in default.
	tmp := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmp, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadPath(tmp)
	if err != nil {
		t.Fatalf("the rendered reference must parse: %v", err)
	}
	if got := cfg.GetAutopilotMergeMode(); got != config.DefaultAutopilotMergeMode {
		t.Errorf("the reference's autopilot block must be inert, got merge_mode %q", got)
	}
}

// runConfigUpgrade drives `fab config upgrade <args...>` end to end via the
// cobra command and returns its stdout and execution error.
func runConfigUpgrade(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"upgrade"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

// TestConfigUpgradeCheckDriftExitsNonZeroWithoutWrite pins the drift-probe
// contract (260809-wll4 R5) across the three drift shapes — a stale fence
// kit-version stamp, an unparked unknown key, and a missing fence: `--check`
// exits non-zero, prints the would-change report to stdout FIRST, and leaves
// the file byte-identical on disk.
func TestConfigUpgradeCheckDriftExitsNonZeroWithoutWrite(t *testing.T) {
	// Build a clean file at a stale stamp for the stale-stamp case (a real
	// upgrade at the old version, then probe with the current one).
	fabRoot := setupInitRepo(t)
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")
	stale := "project:\n    name: t\n"
	if err := os.WriteFile(cfgPath, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runConfigUpgrade(t); err != nil {
		t.Fatalf("seeding upgrade run: %v", err)
	}
	staleBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// Rewind the fence stamp so the current binary sees drift.
	rewound := strings.Replace(string(staleBytes), "(kit "+version+")", "(kit 0.0.1)", 1)
	if rewound == string(staleBytes) {
		t.Fatalf("seeded file carries no (kit %s) fence stamp to rewind", version)
	}

	cases := map[string]string{
		"stale fence stamp":    rewound,
		"unparked unknown key": "project:\n    name: t\n\nlegacy_mode: true\n",
		"missing fence":        "project:\n    name: t\n    description: d\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			out, err := runConfigUpgrade(t, "--check")
			if err == nil {
				t.Errorf("`config upgrade --check` must exit non-zero on %s drift", name)
			}
			if !strings.Contains(out, "drifted") {
				t.Errorf("drift output must name the drift before the error return, got: %q", out)
			}
			after, readErr := os.ReadFile(cfgPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(after) != content {
				t.Errorf("--check must write nothing (file changed on %s drift).\n--- before ---\n%s\n--- after ---\n%s", name, content, string(after))
			}
		})
	}
}

// TestConfigUpgradeCheckUnparkedKeyReportLines: the non-zero drift path prints
// the same advisory report lines an applying run prints, before returning the
// error — the output names WHAT would change.
func TestConfigUpgradeCheckUnparkedKeyReportLines(t *testing.T) {
	fabRoot := setupInitRepo(t)
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")
	content := "project:\n    name: t\n\nlegacy_mode: true\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runConfigUpgrade(t, "--check")
	if err == nil {
		t.Fatal("`config upgrade --check` must exit non-zero on drift")
	}
	if !strings.Contains(out, "  - ") || !strings.Contains(out, "legacy_mode") {
		t.Errorf("the drift path must print the would-change report lines (\"  - …\") naming the unparked key, got: %q", out)
	}
}

// TestConfigUpgradeCheckCleanExitsZeroWithoutWrite: on a file already reconciled
// by a real run, `--check` exits 0 with the "already up to date" message and
// leaves the file byte-identical.
func TestConfigUpgradeCheckCleanExitsZeroWithoutWrite(t *testing.T) {
	fabRoot := setupInitRepo(t)
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("project:\n    name: t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runConfigUpgrade(t); err != nil {
		t.Fatalf("seeding upgrade run: %v", err)
	}
	before, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runConfigUpgrade(t, "--check")
	if err != nil {
		t.Fatalf("`config upgrade --check` on a clean file must exit 0: %v", err)
	}
	if !strings.Contains(out, "already up to date") {
		t.Errorf("clean --check must print the up-to-date message, got: %q", out)
	}
	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("--check on a clean file must write nothing")
	}
}

// TestConfigUpgradeCheckMissingFileExitsNonZeroCreatesNothing: a repo with no
// config.yaml is drift (a real run would create the file), so `--check` exits
// non-zero — and creates nothing.
func TestConfigUpgradeCheckMissingFileExitsNonZeroCreatesNothing(t *testing.T) {
	fabRoot := setupInitRepo(t)
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")

	out, err := runConfigUpgrade(t, "--check")
	if err == nil {
		t.Error("`config upgrade --check` on a missing config.yaml must exit non-zero (a real run would create it)")
	}
	if !strings.Contains(out, "drifted") {
		t.Errorf("the missing-file drift must be reported to stdout, got: %q", out)
	}
	if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
		t.Errorf("--check must not create config.yaml, stat: %v", statErr)
	}
}
