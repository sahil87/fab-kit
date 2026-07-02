package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
)

const scaffoldConfigRelPath = "src/kit/scaffold/fab/project/config.yaml"

// TestConfigReferenceRoundTrips is the VALIDITY contract: the emitted reference
// parses cleanly via the same internal/config loader real project configs use.
// A malformed reference (bad indentation, an un-quoted value) would fail here.
func TestConfigReferenceRoundTrips(t *testing.T) {
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
	if cfg.GetSpawnCommand() == "" {
		t.Error("agent.spawn_command should be a live key with a value in the reference")
	}
	if len(cfg.TestPaths) == 0 {
		t.Error("test_paths should be a live key with a value in the reference")
	}
	if len(cfg.TrueImpactExclude) == 0 {
		t.Error("true_impact_exclude should be a live key with a value in the reference")
	}
	// The opt-in override blocks must stay commented-out (uncommenting = opting
	// in) — so they parse to their zero values, not populated maps.
	if _, ok := cfg.GetAgentTier("doing"); ok {
		t.Error("agent.tiers must be commented-out in the reference (parsed as an override)")
	}
	if len(cfg.StageHooks) != 0 {
		t.Error("stage_hooks must be commented-out in the reference (parsed as live)")
	}
	if cfg.GetBranchPrefix() != "" {
		t.Error("branch_prefix must be commented-out in the reference (parsed as live)")
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

	for seg := range segments {
		if !containsKeyToken(out, seg) {
			t.Errorf("binary-consumed config key %q (from Config yaml tags) is not documented in `fab config reference`", seg)
		}
	}
}

// TestConfigReferenceSupersetsScaffoldKeys is the SKILL-KEY coverage contract:
// the reference's key set must be a superset of the scaffold's key set. This
// guards the skill-consumed keys (source_paths, checklist, review_tools,
// project.name/description) that Go reflection over Config cannot see.
func TestConfigReferenceSupersetsScaffoldKeys(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}

	scaffoldPath := findRepoFile(t, scaffoldConfigRelPath)
	scaffoldKeys := scaffoldKeyTokens(t, scaffoldPath)
	if len(scaffoldKeys) == 0 {
		t.Fatal("parsed no keys from the scaffold config — parser is broken")
	}

	for _, key := range scaffoldKeys {
		if !containsKeyToken(out, key) {
			t.Errorf("scaffold key %q is not documented in `fab config reference` (skill-consumed key gap)", key)
		}
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
		t.Errorf("`fab config reference` output is not byte-stable across renders")
	}
}

// TestConfigReferenceCommandPrintsAndExitsZero drives the cobra command end to
// end: it prints the reference to stdout and exits 0 with no args, and rejects
// extra args (cobra.NoArgs).
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
		t.Error("`config reference extra` should be rejected (cobra.NoArgs)")
	}
}

// TestConfigReferenceMentionsSpawnPlaceholders guards the coordination contract
// between `fab config reference` (6nke) and spawn_command template mode (6tmi):
// the reference's spawn_command comment must document the optional
// {model}/{effort} placeholders. (The tiers comment's "{model, effort}" profile
// notation would not satisfy these exact-token checks.)
func TestConfigReferenceMentionsSpawnPlaceholders(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	for _, placeholder := range []string{"{model}", "{effort}"} {
		if !strings.Contains(out, placeholder) {
			t.Errorf("reference spawn_command comment must document the optional %s placeholder", placeholder)
		}
	}
}

// TestConfigReferenceDocumentsTierSpawnCommand guards that the generated
// reference documents the per-tier spawn_command opt-in and its load-bearing
// no-cross-fallback semantic (present → CLI dispatch; absent → native; NO
// fallback to agent.spawn_command). The reflection coverage test only asserts the
// `spawn_command` token appears at all — this asserts the tier-level semantics are
// actually spelled out, not just the whole-session agent.spawn_command comment.
func TestConfigReferenceDocumentsTierSpawnCommand(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	// The tiers block must mention spawn_command in its override example.
	if !strings.Contains(out, "# spawn_command:") {
		t.Error("reference agent.tiers block must show a commented spawn_command override example")
	}
	// The no-cross-fallback semantic must be documented.
	if !strings.Contains(out, "NO fallback from a tier to agent.spawn_command") {
		t.Error("reference must document that a tier spawn_command has NO fallback to agent.spawn_command")
	}
}

// yamlKeySegments walks a struct type and returns the set of every yaml key
// segment reachable from it. Descends into nested structs and map value types
// (a map's value type contributes its own struct's segments). Returns segments
// (leaf key names), not full dotted paths, because the reference documents some
// keys in dotted-prose form (`agent.tiers`, `stage_hooks.<stage>.pre`); a
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
			// The map key is a free-form stage/tier name (not a fixed key), so
			// descend only into the value type for its struct fields.
			walk(rt.Elem())
		case reflect.Slice, reflect.Array:
			walk(rt.Elem())
		}
	}
	walk(t)
	return segments
}

// scaffoldKeyLineRe matches a live (non-commented) `key:` line — top-level or
// nested — capturing the key name. It deliberately ignores comment lines and
// list items, so it collects exactly the scaffold's LIVE keys, which are the
// skill-consumed set the reference must cover.
var scaffoldKeyLineRe = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*):(\s|$)`)

// scaffoldKeyTokens returns the scaffold config.yaml's live key names. The
// scaffold is a TEMPLATE, not valid YAML — it carries `{PLACEHOLDER}` tokens
// (`- {SOURCE_PATHS}`, a bare column-0 `{TEST_PATHS}` slot) that /fab-setup
// substitutes at setup time, so it cannot be YAML-parsed as-is. A line scan for
// `key:` lines (skipping `#` comments) is the robust way to extract the live
// key set without depending on the placeholders being valid YAML.
func scaffoldKeyTokens(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read scaffold: %v", err)
	}
	keys := map[string]struct{}{}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue // commented example — not a live key
		}
		if m := scaffoldKeyLineRe.FindStringSubmatch(line); m != nil {
			keys[m[1]] = struct{}{}
		}
	}
	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	return out
}

// keyTokenBoundary matches a word boundary for a config key token (letters,
// digits, underscore). Used so `test_paths` matches `test_paths:` but a search
// for `paths` would not spuriously match `test_paths`.
func containsKeyToken(haystack, token string) bool {
	re := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(token) + `([^A-Za-z0-9_]|$)`)
	return re.MatchString(haystack)
}

// findRepoFile walks up from the working directory until relPath resolves,
// mirroring internal/agent's findDocFile (same repo-root-relative resolution
// used by the stage-models drift test).
func findRepoFile(t *testing.T, relPath string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate %q by walking up to the filesystem root", relPath)
		}
		dir = parent
	}
}
