package setupcheck

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// stubLookPath returns a LookPathFunc resolving exactly the given executables.
func stubLookPath(found ...string) LookPathFunc {
	set := make(map[string]bool, len(found))
	for _, name := range found {
		set[name] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/local/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
}

func TestLeadingExecutable(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"claude --dangerously-skip-permissions -n \"$(basename \"$(pwd)\")\" --model {model}", "claude"},
		{"codex exec --dangerously-bypass-approvals-and-sandbox -m {model}", "codex"},
		// The nested-shell stdin idioms must resolve the PROVIDER's binary, not sh.
		{`sh -c 'agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'`, "agy"},
		{`sh -c 'kimi -m {model} -p "$(cat)"'`, "kimi"},
		{`bash -c "mytool --flag"`, "mytool"},
		{"env FOO=bar mytool run", "mytool"},
	}
	for _, c := range cases {
		if got := LeadingExecutable(c.in); got != c.want {
			t.Errorf("LeadingExecutable(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestProbeProviders_MissingConfiguredProviderFails(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{Workers: "agy"}, // session stays default claude
	}
	probes, findings := ProbeProviders(cfg, stubLookPath("claude", "kimi"))

	var agy ProviderProbe
	seen := make(map[string]bool)
	for _, p := range probes {
		seen[p.Name] = true
		if p.Name == "agy" {
			agy = p
		}
	}
	for _, name := range []string{"claude", "codex", "agy", "kimi"} {
		if !seen[name] {
			t.Errorf("roster missing built-in provider %q", name)
		}
	}
	if !agy.Configured {
		t.Error("agy probe should be Configured (agent.workers names it)")
	}
	if len(agy.Missing) != 1 || agy.Missing[0] != "agy" {
		t.Errorf("agy Missing = %v, want [agy] — the nested sh -c wrapper must resolve to agy", agy.Missing)
	}

	var failFound, infoFound bool
	for _, f := range findings {
		if f.Check != "providers" {
			continue
		}
		if f.Subject == "agy" && f.Severity == Fail && strings.Contains(f.Detail, "agent role(s)") {
			failFound = true
		}
		// codex is neither configured nor on PATH → informational only.
		if f.Subject == "codex" && f.Severity == Info {
			infoFound = true
		}
		// claude is configured (session knob) and present → OK finding.
		if f.Subject == "codex" && f.Severity == Fail {
			t.Error("unconfigured provider absence must not be failure-severity")
		}
	}
	if !failFound {
		t.Error("expected a Fail finding for configured-but-missing agy")
	}
	if !infoFound {
		t.Error("expected an Info finding for unconfigured-and-missing codex")
	}
}

func TestProbeProviders_UserDefinedProviderInRoster(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"mytool": {HeadlessCommand: "mytool run --batch"},
		},
	}
	probes, _ := ProbeProviders(cfg, stubLookPath("claude", "mytool"))
	var found bool
	for _, p := range probes {
		if p.Name == "mytool" {
			found = true
			if !p.Headless || p.Interactive || p.Native {
				t.Errorf("mytool capabilities = interactive:%v headless:%v native:%v, want headless only",
					p.Interactive, p.Headless, p.Native)
			}
			if !p.Found() {
				t.Errorf("mytool should be found, Missing = %v", p.Missing)
			}
		}
	}
	if !found {
		t.Error("user-defined provider missing from the roster")
	}
}

func TestProbeEnvironment(t *testing.T) {
	findings := ProbeEnvironment(stubLookPath("gh"), "")

	bySubject := make(map[string]Finding)
	for _, f := range findings {
		bySubject[f.Subject] = f
	}
	if f := bySubject["tmux"]; f.Severity != Info || !strings.Contains(f.Detail, "no tmux") {
		t.Errorf("tmux finding = %+v, want Info naming the absent classification", f)
	}
	if f := bySubject["gh"]; f.Severity != OK {
		t.Errorf("gh finding = %+v, want OK (stubbed present)", f)
	}
	if f := bySubject["yq"]; f.Severity != Warn {
		t.Errorf("yq finding = %+v, want Warn", f)
	}
	if f := bySubject["rk"]; f.Severity != Info {
		t.Errorf("rk finding = %+v, want Info (fail-silent optional tooling)", f)
	}

	live := ProbeEnvironment(stubLookPath("gh", "yq", "rk"), "/tmp/tmux-1000/default,1,0")
	for _, f := range live {
		if f.Severity != OK {
			t.Errorf("with everything present and $TMUX set, want all OK; got %+v", f)
		}
	}
}

func TestProbeVersions(t *testing.T) {
	kitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(kitDir, "VERSION"), []byte("2.19.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("aligned triplet is quiet", func(t *testing.T) {
		for _, f := range ProbeVersions("2.19.4", kitDir, "2.19.4") {
			if f.Severity > OK {
				t.Errorf("aligned triplet produced %+v, want OK-only", f)
			}
		}
	})

	t.Run("kit skew warns", func(t *testing.T) {
		staleKit := t.TempDir()
		if err := os.WriteFile(filepath.Join(staleKit, "VERSION"), []byte("2.18.0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var warned bool
		for _, f := range ProbeVersions("2.19.4", staleKit, "") {
			if f.Severity == Warn && strings.Contains(f.Detail, "kit cache") {
				warned = true
			}
		}
		if !warned {
			t.Error("binary/kit mismatch should warn")
		}
	})

	t.Run("pin mismatch warns", func(t *testing.T) {
		var warned bool
		for _, f := range ProbeVersions("2.19.4", kitDir, "2.18.0") {
			if f.Severity == Warn && strings.Contains(f.Detail, "project pin") {
				warned = true
			}
		}
		if !warned {
			t.Error("binary/pin mismatch should warn")
		}
	})

	t.Run("dev binary never compares", func(t *testing.T) {
		for _, f := range ProbeVersions("dev", kitDir, "9.9.9") {
			if f.Severity == Warn {
				t.Errorf("dev build must not be compared, got %+v", f)
			}
		}
	})

	t.Run("unresolvable kit dir degrades to info", func(t *testing.T) {
		var info bool
		for _, f := range ProbeVersions("2.19.4", "", "2.19.4") {
			if f.Severity > Warn {
				t.Errorf("missing kit dir must not fail, got %+v", f)
			}
			if f.Severity == Info && f.Subject == "kit cache" {
				info = true
			}
		}
		if !info {
			t.Error("expected an Info finding for the unresolvable kit cache")
		}
	})
}

func TestProbeOverrideMasking(t *testing.T) {
	parse := func(yamlText string) map[string]any {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte(yamlText), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.LoadPath(path)
		if err != nil {
			t.Fatalf("fixture config parse: %v", err)
		}
		// Rebuild the raw layer shape via the typed config (the probe consumes
		// layer maps; FromMap round-trips them, so marshal back for the test).
		_ = cfg
		layers, err := config.LoadLayers(path)
		if err != nil {
			t.Fatalf("fixture layers: %v", err)
		}
		return layers.Project
	}

	t.Run("override on a key the embedded defaults lack is load-bearing", func(t *testing.T) {
		// The #573 shape: agy's embedded defaults historically lacked
		// interactive_command; a system-tier override supplying one is
		// load-bearing against such a binary. kimi currently SHIPS an
		// interactive_command, so use a provider+key the embedded table leaves
		// undefined regardless of fixture drift: agy.native.
		system := parse("providers:\n  agy:\n    native: true\n")
		findings := ProbeOverrideMasking(system, nil)
		var flagged bool
		for _, f := range findings {
			if f.Severity == Warn && f.Subject == "providers.agy.native" &&
				strings.Contains(f.Detail, "load-bearing") && strings.Contains(f.Detail, "system") {
				flagged = true
			}
		}
		if !flagged {
			t.Errorf("expected a load-bearing warning for providers.agy.native, got %+v", findings)
		}
	})

	t.Run("override matching the embedded default is not flagged", func(t *testing.T) {
		// claude's embedded defaults define interactive_command — overriding
		// its VALUE is ordinary configuration, not a mask.
		system := parse("providers:\n  claude:\n    interactive_command: 'claude --custom'\n")
		for _, f := range ProbeOverrideMasking(system, nil) {
			t.Errorf("unexpected finding for an embedded-defined key: %+v", f)
		}
	})

	t.Run("user-defined provider is a definition, not a mask", func(t *testing.T) {
		system := parse("providers:\n  mytool:\n    interactive_command: 'mytool -i'\n")
		for _, f := range ProbeOverrideMasking(system, nil) {
			t.Errorf("user-defined provider must be skipped, got %+v", f)
		}
	})
}

func TestProbeDispatchMode(t *testing.T) {
	t.Run("viable native default is OK", func(t *testing.T) {
		findings := ProbeDispatchMode(&config.Config{}, "")
		if len(findings) != 1 || findings[0].Severity != OK {
			t.Errorf("default native mode with claude capabilities should be OK, got %+v", findings)
		}
	})

	t.Run("pane without tmux warns with the ladder's own reason", func(t *testing.T) {
		cfg := &config.Config{Dispatch: config.DispatchConfig{Mode: "pane"}}
		findings := ProbeDispatchMode(cfg, "")
		if len(findings) != 1 || findings[0].Severity != Warn {
			t.Fatalf("pane mode without tmux should warn, got %+v", findings)
		}
		if !strings.Contains(findings[0].Detail, "pane unavailable: no tmux") {
			t.Errorf("descent reason must reuse SelectMode's exact string, got %q", findings[0].Detail)
		}
	})

	t.Run("no reachable rung fails", func(t *testing.T) {
		cfg := &config.Config{
			Dispatch: config.DispatchConfig{Mode: "headless"},
			Agent:    config.AgentConfig{Workers: "mytool"},
			Providers: map[string]config.ProviderConfig{
				"mytool": {InteractiveCommand: "mytool -i"}, // no headless_command
			},
		}
		findings := ProbeDispatchMode(cfg, "")
		if len(findings) != 1 || findings[0].Severity != Fail {
			t.Errorf("headless preference with a headless-less provider should fail, got %+v", findings)
		}
	})

	t.Run("undefined knob provider fails", func(t *testing.T) {
		cfg := &config.Config{Agent: config.AgentConfig{Workers: "ghost"}}
		findings := ProbeDispatchMode(cfg, "")
		if len(findings) != 1 || findings[0].Severity != Fail || findings[0].Subject != "ghost" {
			t.Errorf("undefined workers provider should be a Fail finding naming it, got %+v", findings)
		}
	})
}

func TestRun_AggregatesAndDegradesOutsideRepo(t *testing.T) {
	// Isolate from the real ~/.fab-kit/config.yaml — Run honors the system
	// tier by design, and the developer machine's own config must not leak
	// into the fixture (config.homeDir honors $HOME on unix).
	t.Setenv("HOME", t.TempDir())
	report := Run(Input{
		BinaryVersion: "dev",
		TmuxEnv:       "",
		LookPath:      stubLookPath("claude", "gh", "yq"),
		KitDir:        t.TempDir(), // no VERSION file → Info, not failure
	})
	if report == nil || len(report.Providers) == 0 {
		t.Fatal("Run outside a repo must still produce the provider roster")
	}
	if report.HasFailures() {
		var fails []string
		for _, f := range report.Findings {
			if f.Severity == Fail {
				fails = append(fails, f.Subject+": "+f.Detail)
			}
		}
		t.Errorf("a clean default environment must not fail, got %v", fails)
	}
}
