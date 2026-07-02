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
	prov, ok := cfg.GetProvider("claude")
	if !ok || prov.SessionCommand == "" {
		t.Error("providers.claude.session_command should be a live key with a value in the reference")
	}
	// claude's dispatch_command ships COMMENTED (uncommenting flips claude from
	// native Agent-tool dispatch to headless CLI dispatch), so the commented
	// template text must parse as absent — a live dispatch_command here would
	// mean the opt-in leaked into the shipped default.
	if prov.DispatchCommand != "" {
		t.Errorf("providers.claude.dispatch_command must parse as absent (commented template), got %q", prov.DispatchCommand)
	}
	// codex and gemini are commented starter-template blocks only — never Go
	// defaults and never live in the reference. They must parse as absent so the
	// three-provider template text can never accidentally register a provider.
	if _, ok := cfg.GetProvider("codex"); ok {
		t.Error("providers.codex must be commented-out in the reference (parsed as live)")
	}
	if _, ok := cfg.GetProvider("gemini"); ok {
		t.Error("providers.gemini must be commented-out in the reference (parsed as live)")
	}
	if len(cfg.TestPaths) == 0 {
		t.Error("test_paths should be a live key with a value in the reference")
	}
	if len(cfg.TrueImpactExclude) == 0 {
		t.Error("true_impact_exclude should be a live key with a value in the reference")
	}
	// The five agent.tiers are shown LIVE with explicit providers (documented
	// style — provider written on every line). They must parse to a populated map.
	if _, ok := cfg.GetAgentTier("doing"); !ok {
		t.Error("agent.tiers must be live in the reference (five role tiers with explicit providers)")
	}
	// The opt-in override blocks must stay commented-out (uncommenting = opting in).
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
// guards the skill-consumed keys (source_paths, checklist,
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

// TestConfigReferenceDocumentsProviders guards that the generated reference
// documents the providers table with both command fields and the load-bearing
// no-cross-fallback semantic (absent dispatch_command → native dispatch; NO
// fallback from dispatch_command to session_command).
func TestConfigReferenceDocumentsProviders(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	for _, token := range []string{"providers:", "session_command", "dispatch_command"} {
		if !strings.Contains(out, token) {
			t.Errorf("reference must document %q in the providers block", token)
		}
	}
	// The no-cross-fallback semantic must be documented (the "NO" precedes on the
	// prior comment line; assert on the stable tail phrase).
	if !strings.Contains(out, "fallback from dispatch_command to session_command") {
		t.Error("reference must document that dispatch_command has NO fallback to session_command")
	}
}

// TestConfigReferenceDocumentsThreeProviderTemplate is the ho9y contract: the
// providers block ships as a three-provider starter template — claude (built-in
// default), codex, and gemini — each with both command fields present as text,
// so a user adding a non-claude provider copies and adapts rather than composing
// grammar from scratch. Gemini carries no {effort} placeholder (the gemini CLI
// has no reasoning-effort flag).
func TestConfigReferenceDocumentsThreeProviderTemplate(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}

	// All three provider names appear as text in the providers template.
	for _, provider := range []string{"claude:", "codex:", "gemini:"} {
		if !strings.Contains(out, provider) {
			t.Errorf("providers template must document the %q provider block", provider)
		}
	}

	// Both command fields are documented for the non-claude template providers.
	// codex and gemini each carry a session_command AND a dispatch_command line
	// (present as commented text). Assert on the distinctive command bodies so a
	// single generic session_command/dispatch_command elsewhere can't satisfy this.
	for _, cmd := range []string{
		"codex -m {model} -c model_reasoning_effort={effort}",      // codex session_command
		"codex exec -m {model} -c model_reasoning_effort={effort}", // codex dispatch_command
		"gemini -m {model}", // gemini session + dispatch
	} {
		if !strings.Contains(out, cmd) {
			t.Errorf("providers template must document the command %q", cmd)
		}
	}

	// Gemini carries NO {effort} placeholder (the gemini CLI has no
	// reasoning-effort flag) and NO -p on its command (fab dispatch pipes the
	// prompt to stdin, which gemini reads in non-TTY mode; -p takes prompt text
	// appended after stdin). Guard that no gemini command string smuggles these in.
	for _, badGemini := range []string{
		"gemini -m {model} -c model_reasoning_effort",
		"gemini -m {model} --effort",
		"gemini -m {model} {effort}",
		"gemini -m {model} -p",
	} {
		if strings.Contains(out, badGemini) {
			t.Errorf("gemini command must not contain %q (no {effort} flag; no -p for stdin dispatch)", badGemini)
		}
	}

	// claude's dispatch_command ships commented (uncommenting flips native→CLI
	// dispatch), so it must be present as text but parse as absent from Config.
	if !strings.Contains(out, "claude -p --dangerously-skip-permissions --model {model} --effort {effort}") {
		t.Error("providers template must document claude's (commented) dispatch_command")
	}
}

// TestConfigReferenceRetiresLegacyKeys guards that the removed keys no longer
// appear in the reference: review_tools (retired to code-review.md § Review Tools)
// and agent.spawn_command (relocated to providers.claude.session_command).
func TestConfigReferenceRetiresLegacyKeys(t *testing.T) {
	out, err := configref.Render()
	if err != nil {
		t.Fatalf("Render returned an error: %v", err)
	}
	for _, gone := range []string{"review_tools", "spawn_command"} {
		if containsKeyToken(out, gone) {
			t.Errorf("retired key %q must not appear in the reference", gone)
		}
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
