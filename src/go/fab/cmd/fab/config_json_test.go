package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// setupProjectless chdirs into a bare temp dir (no fab/ above it — the daemon-cwd
// shape of a $HOME-launched consumer) with an isolated, empty HOME and every FAB_*
// override scrubbed. It returns the fake home. Nothing here can reach the real
// ~/.fab-kit/config.yaml.
func setupProjectless(t *testing.T) (home string) {
	t.Helper()
	home = t.TempDir()
	env := map[string]string{"HOME": home}
	for _, key := range configscope.DottedKeys() {
		env["FAB_"+strings.ToUpper(strings.ReplaceAll(key, ".", "_"))] = ""
	}
	chdirTestEnv(t, t.TempDir(), env)
	return home
}

// runConfigSplit runs a `fab config` invocation capturing stdout and stderr
// separately, so JSON-on-stdout tests are not polluted by warnings.
func runConfigSplit(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := configCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errBuf.String(), err
}

// jsonRows decodes a `--origin --json` document into raw-field rows, so a test can
// assert on the exact encoded spelling of a value (`50`, not `50.0`).
func jsonRows(t *testing.T, doc string) []map[string]json.RawMessage {
	t.Helper()
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(doc), &rows); err != nil {
		t.Fatalf("not a JSON array of rows: %v\n%s", err, doc)
	}
	return rows
}

func rowString(t *testing.T, row map[string]json.RawMessage, field string) string {
	t.Helper()
	var s string
	if err := json.Unmarshal(row[field], &s); err != nil {
		t.Fatalf("field %q is not a string: %s", field, row[field])
	}
	return s
}

// rowFor returns the first row with the given key.
func rowFor(t *testing.T, rows []map[string]json.RawMessage, key string) map[string]json.RawMessage {
	t.Helper()
	for _, r := range rows {
		if rowString(t, r, "key") == key {
			return r
		}
	}
	t.Fatalf("no row for key %q", key)
	return nil
}

func TestConfigShowJSON_PlainAndKeyedValues(t *testing.T) {
	setupConfigRepo(t, "")

	out, _, err := runConfigSplit(t, "show", "dispatch.column_width", "--json")
	if err != nil {
		t.Fatalf("show key --json: %v", err)
	}
	if out != "45\n" {
		t.Errorf("an int must encode as a bare number, got %q", out)
	}

	out, _, err = runConfigSplit(t, "show", "dispatch.reap_done", "--json")
	if err != nil || out != "true\n" {
		t.Errorf("a bool must encode as true, got %q (err %v)", out, err)
	}

	out, _, err = runConfigSplit(t, "show", "dispatch.mode", "--json")
	if err != nil || out != "\"native\"\n" {
		t.Errorf("a string must encode quoted, got %q (err %v)", out, err)
	}

	out, _, err = runConfigSplit(t, "show", "--json")
	if err != nil {
		t.Fatalf("show --json: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("show --json is not one JSON object: %v\n%s", err, out)
	}
	dispatch, _ := doc["dispatch"].(map[string]any)
	if dispatch["reap_done"] != true || dispatch["column_width"] != float64(45) {
		t.Errorf("composed config lost typed dispatch values: %v", dispatch)
	}
	if !strings.HasSuffix(out, "}\n") || !strings.Contains(out, "\n  \"") {
		t.Errorf("document must be indented like `config explain --json` and end with a newline:\n%s", out)
	}
}

func TestConfigShowJSON_MapKeyAndListValue(t *testing.T) {
	setupConfigRepo(t, "source_paths:\n  - src/\n  - scripts/\n")

	out, _, err := runConfigSplit(t, "show", "source_paths", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var list []string
	if err := json.Unmarshal([]byte(out), &list); err != nil || len(list) != 2 || list[0] != "src/" {
		t.Errorf("a list key must encode as a JSON array, got %q (err %v)", out, err)
	}

	out, _, err = runConfigSplit(t, "show", "agent.profiles.doing", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(out), &obj); err != nil || obj["provider"] == nil {
		t.Errorf("a map key must encode as a JSON object, got %q (err %v)", out, err)
	}
}

func TestConfigShowJSON_UnknownKeyErrorsWithNoStdout(t *testing.T) {
	setupConfigRepo(t, "")
	for _, args := range [][]string{
		{"show", "no.such.key", "--json"},
		{"show", "no.such.key", "--origin", "--json"},
	} {
		out, _, err := runConfigSplit(t, args...)
		if err == nil {
			t.Fatalf("%v: unknown key must error", args)
		}
		if json.Valid([]byte(out)) {
			t.Errorf("%v: no JSON document may reach stdout on error, got %q", args, out)
		}
	}
}

// TestConfigShowOriginJSON_ProjectlessTierDetails: the hexokit shape — cwd outside
// any fab project, an env override, a system file, and built-in defaults — with
// every row's origin and origin_detail pinned.
func TestConfigShowOriginJSON_ProjectlessTierDetails(t *testing.T) {
	home := setupProjectless(t)
	writeSystemConfig(t, home, "agent:\n  workers: agy\n")
	t.Setenv("FAB_AGENT_SESSION", "codex")
	systemPath := filepath.Join(home, ".fab-kit", "config.yaml")

	out, _, err := runConfigSplit(t, "show", "--origin", "--json")
	if err != nil {
		t.Fatalf("show --origin --json: %v", err)
	}
	rows := jsonRows(t, out)

	for _, r := range rows {
		if got := rowString(t, r, "status"); got != "effective" {
			t.Errorf("winner-only rows are all effective, %s is %q", rowString(t, r, "key"), got)
		}
		for _, field := range []string{"key", "value", "origin", "origin_detail", "status"} {
			if _, ok := r[field]; !ok {
				t.Errorf("row %s lacks field %q", rowString(t, r, "key"), field)
			}
		}
	}

	workers := rowFor(t, rows, "agent.workers")
	want := `{"key":"agent.workers","value":"agy","origin":"system","origin_detail":` + jsonQuote(systemPath) + `,"status":"effective"}`
	if got := compactJSON(t, workers); got != want {
		t.Errorf("agent.workers row:\n got %s\nwant %s", got, want)
	}

	session := rowFor(t, rows, "agent.session")
	if rowString(t, session, "origin") != "env" || rowString(t, session, "origin_detail") != "FAB_AGENT_SESSION" {
		t.Errorf("agent.session must carry env + the variable name without `$`: %s", compactJSON(t, session))
	}

	minCols := rowFor(t, rows, "dispatch.min_cols")
	if string(minCols["value"]) != "50" || rowString(t, minCols, "origin") != "default" || rowString(t, minCols, "origin_detail") != "" {
		t.Errorf("a default int must encode as 50 with an empty detail: %s", compactJSON(t, minCols))
	}
}

func TestConfigShowOriginJSON_MapLeafUnderEnvUsesRegistryVariable(t *testing.T) {
	setupProjectless(t)
	t.Setenv("FAB_AGENT_PROFILES", "{review: {provider: codex}}")

	out, _, err := runConfigSplit(t, "show", "--origin", "--json")
	if err != nil {
		t.Fatal(err)
	}
	row := rowFor(t, jsonRows(t, out), "agent.profiles.review.provider")
	if rowString(t, row, "origin") != "env" || rowString(t, row, "origin_detail") != "FAB_AGENT_PROFILES" {
		t.Errorf("a leaf beneath a map-valued env row names the registry-row variable: %s", compactJSON(t, row))
	}
}

func TestConfigShowOriginJSON_KeyedStack(t *testing.T) {
	home := setupProjectless(t)
	writeSystemConfig(t, home, "agent:\n  workers: agy\n")
	t.Setenv("FAB_AGENT_WORKERS", "codex")
	systemPath := filepath.Join(home, ".fab-kit", "config.yaml")

	out, _, err := runConfigSplit(t, "show", "agent.workers", "--origin", "--json")
	if err != nil {
		t.Fatal(err)
	}
	rows := jsonRows(t, out)
	type want struct{ value, origin, detail, status string }
	wants := []want{
		{"codex", "env", "FAB_AGENT_WORKERS", "effective"},
		{"agy", "system", systemPath, "shadowed"},
		{"claude", "default", "", "shadowed"},
	}
	if len(rows) != len(wants) {
		t.Fatalf("want %d stack rows, got %d:\n%s", len(wants), len(rows), out)
	}
	for i, w := range wants {
		r := rows[i]
		got := want{rowString(t, r, "value"), rowString(t, r, "origin"), rowString(t, r, "origin_detail"), rowString(t, r, "status")}
		if got != w {
			t.Errorf("row %d: got %+v, want %+v", i, got, w)
		}
	}
}

func TestConfigShowOriginJSON_UndefinedKeyIsEmptyArray(t *testing.T) {
	setupConfigRepo(t, "source_paths: []\n")

	out, _, err := runConfigSplit(t, "show", "source_paths", "--origin", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("a key no tier defines must print [], got %q", out)
	}
}

func TestConfigShowOriginJSON_AncestorReplacedDescendantIsOneNullRow(t *testing.T) {
	repo, _ := setupConfigRepo(t, "providers: oops\n")
	projectPath := filepath.Join(repo, "fab", "project", "config.yaml")

	out, _, err := runConfigSplit(t, "show", "providers.claude.interactive_command", "--origin", "--json")
	if err != nil {
		t.Fatal(err)
	}
	rows := jsonRows(t, out)
	if len(rows) != 1 {
		t.Fatalf("want exactly one row, got:\n%s", out)
	}
	want := `{"key":"providers.claude.interactive_command","value":null,"origin":"project","origin_detail":` + jsonQuote(projectPath) + `,"status":"effective"}`
	if got := compactJSON(t, rows[0]); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestConfigShowOriginJSON_ListValuesStayTyped(t *testing.T) {
	setupConfigRepo(t, "source_paths:\n  - src/\n")

	out, _, err := runConfigSplit(t, "show", "source_paths", "--origin", "--json")
	if err != nil {
		t.Fatal(err)
	}
	row := rowFor(t, jsonRows(t, out), "source_paths")
	if got := compactRaw(t, row["value"]); got != `["src/"]` {
		t.Errorf("a list leaf must encode as an array, got %s", got)
	}
}

func TestConfigShowOriginJSON_DerivedFrom(t *testing.T) {
	t.Run("system knob drives default-tier role rows", func(t *testing.T) {
		home := setupProjectless(t)
		writeSystemConfig(t, home, "agent:\n  workers: agy\n")
		systemPath := filepath.Join(home, ".fab-kit", "config.yaml")

		out, _, err := runConfigSplit(t, "show", "--origin", "--json")
		if err != nil {
			t.Fatal(err)
		}
		row := rowFor(t, jsonRows(t, out), "agent.profiles.doing.provider")
		if rowString(t, row, "value") != "agy" || rowString(t, row, "origin") != "default" {
			t.Fatalf("setup: doing.provider must be a derived default row: %s", compactJSON(t, row))
		}
		want := `{"key":"agent.workers","origin":"system","origin_detail":` + jsonQuote(systemPath) + `}`
		if got := compactRaw(t, row["derived_from"]); got != want {
			t.Errorf("derived_from:\n got %s\nwant %s", got, want)
		}
	})

	t.Run("no config anywhere means no derived_from", func(t *testing.T) {
		setupProjectless(t)
		out, _, err := runConfigSplit(t, "show", "--origin", "--json")
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range jsonRows(t, out) {
			if _, ok := r["derived_from"]; ok {
				t.Errorf("row %s must not carry derived_from with no user config", rowString(t, r, "key"))
			}
		}
	})

	t.Run("a role's own provider override is a literal, not derived", func(t *testing.T) {
		setupConfigRepo(t, "agent:\n  workers: agy\n  profiles:\n    review:\n      provider: codex\n")
		out, _, err := runConfigSplit(t, "show", "--origin", "--json")
		if err != nil {
			t.Fatal(err)
		}
		row := rowFor(t, jsonRows(t, out), "agent.profiles.review.provider")
		if rowString(t, row, "origin") != "project" {
			t.Fatalf("setup: review.provider must be won by the project tier: %s", compactJSON(t, row))
		}
		if _, ok := row["derived_from"]; ok {
			t.Errorf("a row won by a user tier carries no derived_from: %s", compactJSON(t, row))
		}
	})

	t.Run("keyed stack annotates the default row", func(t *testing.T) {
		home := setupProjectless(t)
		writeSystemConfig(t, home, "agent:\n  workers: agy\n")

		out, _, err := runConfigSplit(t, "show", "agent.profiles.doing.provider", "--origin", "--json")
		if err != nil {
			t.Fatal(err)
		}
		rows := jsonRows(t, out)
		if len(rows) != 1 || rowString(t, rows[0], "origin") != "default" {
			t.Fatalf("want the single default row, got:\n%s", out)
		}
		if !strings.Contains(compactRaw(t, rows[0]["derived_from"]), `"key":"agent.workers"`) {
			t.Errorf("keyed default row must carry derived_from: %s", compactJSON(t, rows[0]))
		}
	})

	t.Run("a user-set provider fill drives model derivation", func(t *testing.T) {
		home := setupProjectless(t)
		writeSystemConfig(t, home, "agent:\n  workers: agy\nproviders:\n  agy:\n    profiles:\n      doing:\n        model: my-model\n")
		systemPath := filepath.Join(home, ".fab-kit", "config.yaml")

		out, _, err := runConfigSplit(t, "show", "agent.profiles.doing.model", "--origin", "--json")
		if err != nil {
			t.Fatal(err)
		}
		rows := jsonRows(t, out)
		if len(rows) != 1 || rowString(t, rows[0], "value") != "my-model" || rowString(t, rows[0], "origin") != "default" {
			t.Fatalf("setup: want the single derived default row, got:\n%s", out)
		}
		want := `{"key":"providers.agy.profiles.doing.model","origin":"system","origin_detail":` + jsonQuote(systemPath) + `}`
		if got := compactRaw(t, rows[0]["derived_from"]); got != want {
			t.Errorf("derived_from:\n got %s\nwant %s", got, want)
		}
	})
}

// TestConfigShowText_DerivedFromNeverAppears: the text form is unchanged.
func TestConfigShowText_DerivedFromNeverAppears(t *testing.T) {
	home := setupProjectless(t)
	writeSystemConfig(t, home, "agent:\n  workers: agy\n")
	out, err := runConfig(t, "show", "--origin")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "derived_from") || strings.Contains(out, "via ") {
		t.Errorf("text --origin must not change:\n%s", out)
	}
	if !strings.Contains(out, "agent.profiles.doing.provider = agy") {
		t.Errorf("setup: expected the derived row in text output:\n%s", out)
	}
}

// --- set value-domain validation ---------------------------------------------

func TestConfigSet_RefusesOutOfDomainValuesWithoutWriting(t *testing.T) {
	cases := []struct {
		key, value string
		wantInErr  []string
	}{
		{"dispatch.mode", "weird", []string{"dispatch.mode", "weird", "pane, native, headless"}},
		{"dispatch.column_width", "500", []string{"dispatch.column_width", "500", "1 to 99"}},
		{"dispatch.column_width", "0", []string{"dispatch.column_width", "1 to 99"}},
		{"dispatch.min_cols", "0", []string{"dispatch.min_cols", "at least 1"}},
		{"dispatch.min_rows", "0", []string{"dispatch.min_rows", "at least 1"}},
		{"autopilot.merge_mode", "chaos", []string{"autopilot.merge_mode", "chaos", "cherry-pick-ladder"}},
		{"agent.workers", "bogus", []string{"agent.workers", "bogus", "agy, claude, codex, kimi"}},
		{"agent.session", "bogus", []string{"agent.session", "bogus", "available:"}},
		{"agent.profiles.review.provider", "bogus", []string{"agent.profiles.review.provider", "bogus", "available:"}},
	}
	for _, c := range cases {
		for _, system := range []bool{true, false} {
			name := c.key + "=" + c.value
			if system {
				name += " --system"
			}
			t.Run(name, func(t *testing.T) {
				home := setupProjectless(t)
				args := []string{"set", c.key, c.value}
				if system {
					args = append(args, "--system")
				} else {
					repo, _ := setupConfigRepo(t, "")
					_ = repo
				}
				_, _, err := runConfigSplit(t, args...)
				if err == nil {
					t.Fatal("an out-of-domain value must exit non-zero")
				}
				for _, w := range c.wantInErr {
					if !strings.Contains(err.Error(), w) {
						t.Errorf("error must mention %q: %v", w, err)
					}
				}
				if system {
					if _, statErr := os.Stat(filepath.Join(home, ".fab-kit", "config.yaml")); !os.IsNotExist(statErr) {
						t.Errorf("a refused set must not create the system file (stat err %v)", statErr)
					}
				}
			})
		}
	}
}

func TestConfigSet_RefusedProjectWriteLeavesFileUntouched(t *testing.T) {
	repo, _ := setupConfigRepo(t, "agent:\n    workers: claude\n")
	path := filepath.Join(repo, "fab", "project", "config.yaml")
	before, _ := os.ReadFile(path)

	if _, _, err := runConfigSplit(t, "set", "dispatch.mode", "weird"); err == nil {
		t.Fatal("expected refusal")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("project file changed on a refused set:\n%s", after)
	}
}

func TestConfigSet_AcceptsBoundaryAndValidValues(t *testing.T) {
	for _, c := range [][2]string{
		{"dispatch.mode", "headless"},
		{"dispatch.column_width", "99"},
		{"dispatch.column_width", "1"},
		{"dispatch.min_cols", "1"},
		{"dispatch.min_rows", "9999"},
		{"dispatch.reap_done", "false"},
		{"autopilot.merge_mode", "merge-auto"},
		{"agent.workers", "codex"},
		{"agent.session", "kimi"},
		{"agent.profiles.review.provider", "agy"},
	} {
		t.Run(c[0]+"="+c[1], func(t *testing.T) {
			home := setupProjectless(t)
			if _, _, err := runConfigSplit(t, "set", "--system", c[0], c[1]); err != nil {
				t.Fatalf("set %s %s: %v", c[0], c[1], err)
			}
			if _, err := os.Stat(filepath.Join(home, ".fab-kit", "config.yaml")); err != nil {
				t.Errorf("accepted value must be written: %v", err)
			}
		})
	}
}

// TestConfigSet_KindMismatchKeepsTheKindError: the domain gate runs on top of the
// kind check, never instead of it.
func TestConfigSet_KindMismatchKeepsTheKindError(t *testing.T) {
	for _, c := range [][2]string{
		{"dispatch.column_width", "abc"},
		{"dispatch.reap_done", "bogus"},
		{"dispatch.mode", "5"},
	} {
		t.Run(c[0]+"="+c[1], func(t *testing.T) {
			setupProjectless(t)
			_, _, err := runConfigSplit(t, "set", "--system", c[0], c[1])
			if err == nil {
				t.Fatal("expected a refusal")
			}
			if !strings.Contains(err.Error(), "expects a YAML") {
				t.Errorf("a kind mismatch must report the existing kind error, got: %v", err)
			}
		})
	}
}

func TestConfigSet_ProviderTableIncludesProjectProviders(t *testing.T) {
	setupConfigRepo(t, "providers:\n  mytool:\n    headless_command: mytool run\n")

	if _, _, err := runConfigSplit(t, "set", "agent.profiles.review.provider", "mytool"); err != nil {
		t.Fatalf("a provider defined in the project providers table must be accepted: %v", err)
	}
	_, _, err := runConfigSplit(t, "set", "agent.workers", "othertool")
	if err == nil || !strings.Contains(err.Error(), "mytool") {
		t.Errorf("the refusal must list the project-defined provider among the available names: %v", err)
	}
}

func TestConfigSet_ProviderTableIncludesSystemProvidersOutsideProject(t *testing.T) {
	home := setupProjectless(t)
	writeSystemConfig(t, home, "providers:\n  mytool:\n    headless_command: mytool run\n")

	if _, _, err := runConfigSplit(t, "set", "--system", "agent.workers", "mytool"); err != nil {
		t.Fatalf("a system-defined provider must be accepted outside a project: %v", err)
	}
}

// TestConfigSet_MalformedProjectStillAcceptsBuiltinProvider: when the read model
// cannot be loaded the provider table falls back to the project-free cascade, so a
// built-in name is never refused for that reason alone.
func TestConfigSet_MalformedProjectStillAcceptsBuiltinProvider(t *testing.T) {
	setupConfigRepo(t, "this: is: not: valid: yaml: [[[\n")

	if _, _, err := runConfigSplit(t, "set", "--system", "agent.workers", "codex"); err != nil {
		t.Fatalf("a built-in provider must be accepted when the project file is malformed: %v", err)
	}
	if _, _, err := runConfigSplit(t, "set", "--system", "agent.workers", "bogus"); err == nil {
		t.Error("an unknown provider must still be refused under the fallback table")
	}
}

// TestSetupWizard_WriteRefusesInvalidValue: the wizard's own prompts re-ask on
// most bad answers, so the shared domain gate is the backstop — exercised here at
// the write seam (writeOne), where it must refuse before anything is written.
func TestSetupWizard_WriteRefusesInvalidValue(t *testing.T) {
	home := setupProjectless(t)
	path := filepath.Join(home, ".fab-kit", "config.yaml")
	w := &setupWizard{cmd: setupCmd(), path: path}

	err := w.writeOne(wizardAnswer{key: "dispatch.mode", answer: "weird"})
	if err == nil || !strings.Contains(err.Error(), "dispatch.mode") || !strings.Contains(err.Error(), "pane, native, headless") {
		t.Fatalf("writeOne must refuse an out-of-domain value naming key and accepted set, got: %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("a refused wizard write must create no file, stat err = %v", statErr)
	}
	if err := w.writeOne(wizardAnswer{key: "dispatch.mode", answer: "headless"}); err != nil {
		t.Fatalf("a valid answer must still write: %v", err)
	}
}

// --- shadow / live-tier notices outside a project -------------------------------

func TestConfigSet_ShadowWarningFiresOutsideAProject(t *testing.T) {
	setupProjectless(t)
	t.Setenv("FAB_AGENT_WORKERS", "codex")

	_, stderr, err := runConfigSplit(t, "set", "--system", "agent.workers", "agy")
	if err != nil {
		t.Fatalf("set --system: %v", err)
	}
	want := "fab: warning: agent.workers is shadowed by env $FAB_AGENT_WORKERS — the written value is not in effect"
	if !strings.Contains(stderr, want) {
		t.Errorf("missing shadow warning outside a project:\n got %q\nwant it to contain %q", stderr, want)
	}
}

func TestConfigUnset_LiveTierNoticeFiresOutsideAProject(t *testing.T) {
	setupProjectless(t)
	t.Setenv("FAB_AGENT_SESSION", "codex")

	stdout, stderr, err := runConfigSplit(t, "unset", "--system", "agent.session")
	if err != nil {
		t.Fatalf("unset --system: %v", err)
	}
	if got := stdout + stderr; !strings.Contains(got, "env $FAB_AGENT_SESSION") {
		t.Errorf("unset must name the live env tier outside a project, got:\n%s", got)
	}
}

// --- helpers --------------------------------------------------------------------

func jsonQuote(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

// compactRaw strips the indentation from one encoded JSON value ("" for absent).
func compactRaw(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	if len(raw) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// compactJSON re-encodes a row in schema field order without indentation.
func compactJSON(t *testing.T, row map[string]json.RawMessage) string {
	t.Helper()
	var order = []string{"key", "value", "origin", "origin_detail", "status", "derived_from"}
	var parts []string
	for _, f := range order {
		raw, ok := row[f]
		if !ok {
			continue
		}
		parts = append(parts, jsonQuote(f)+":"+compactRaw(t, raw))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
