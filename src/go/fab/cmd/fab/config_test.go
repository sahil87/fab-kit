package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

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
	// The six agent.tiers are shown LIVE with explicit providers (documented
	// style — provider written on every line). They must parse to a populated map.
	if _, ok := cfg.GetAgentTier("doing"); !ok {
		t.Error("agent.tiers must be live in the reference (six role tiers with explicit providers)")
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

	// fab_version is NOT a config key (260708-j0qm): it lives in the plain-text
	// sibling fab/.fab-version and Config.FabVersion is tagged `yaml:"-"`, so
	// yamlKeySegments skips it and it never appears in `segments` — no exemption is
	// needed here (the positive "not documented" assertion lives in
	// TestConfigReferenceOmitsRelocatedFabVersion).
	for seg := range segments {
		if !containsKeyToken(out, seg) {
			t.Errorf("binary-consumed config key %q (from Config yaml tags) is not documented in `fab config reference`", seg)
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
	// preference-class field (e.g. agent.tiers — which presence=intent forbids
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

// TestConfigReferenceJSONIsValidAndByteStable is the --json VALIDITY + STABILITY
// contract: `fab config reference --json` emits a well-formed JSON array of
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
		t.Error("`fab config reference --json` output is not byte-stable across renders")
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
		// omitted when empty (omitempty), which is every row today.
		if _, ok := obj["default"]; !ok {
			t.Errorf("--json element %d (%v) is missing the `default` field", i, obj["key"])
		}
		if _, ok := obj["renamed_from"]; ok {
			t.Errorf("--json element %d (%v) should omit `renamed_from` (empty on every row today, omitempty)", i, obj["key"])
		}
	}
}

// TestConfigReferenceJSONEmptyDefaultConvention pins the uniform empty-default
// convention (T002 / docs/specs/config.md § Default semantics): a field with no
// meaningful built-in default emits JSON `null`, NEVER a typed empty (`[]`, `{}`,
// `""`). This is the single "cascade falls back to absent" signal Change 2's
// resolver consumes; a typed empty would leak a Go-side implementation detail with
// no cascade meaning. Conversely, a non-null `default` must denote a real built-in
// value (the claude provider and the six tier profiles today).
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
		"providers":   true,
		"agent.tiers": true,
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
		t.Fatalf("Fields returned an error (registry lint or tier invariant failed): %v", err)
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
		// renamed_from is empty on every row today (future field renames only).
		if f.RenamedFrom != "" {
			t.Errorf("field %q has a non-empty RenamedFrom %q; no historical rename is backfilled in this change", f.Key, f.RenamedFrom)
		}
	}
}

// TestConfigReferenceScopeAssignments pins the decision-6 scope taxonomy: the
// preference-class fields (agent.tiers, providers) are `both`; the
// semantics-class fields and the two unenumerated fields (stage_hooks,
// branch_prefix) are `project`. (fab_version left config.yaml in 260708-j0qm and
// no longer carries a scope.) Enforcement landed in Change 2; the assignments are
// consumed as data, so they are pinned.
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
		"providers":                  configref.ScopeBoth,
		"agent.tiers":                configref.ScopeBoth,
		"stage_hooks":                configref.ScopeProject,
		"branch_prefix":              configref.ScopeProject,
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

// keyTokenBoundary matches a word boundary for a config key token (letters,
// digits, underscore). Used so `test_paths` matches `test_paths:` but a search
// for `paths` would not spuriously match `test_paths`.
func containsKeyToken(haystack, token string) bool {
	re := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(token) + `([^A-Za-z0-9_]|$)`)
	return re.MatchString(haystack)
}
