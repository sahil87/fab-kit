package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The provider → cli rename (261120-f2ly, kit 2.31.0) keeps every old spelling
// working for one minor release. These tests pin the compatibility surfaces that
// live in cmd/fab: the hidden --provider flag, the dual -o yaml keys, old-spelling
// configs resolving identically, set/unset mapping old keys, and `explain` of the
// deprecated key names.

// runAgentSplit runs `fab agent` capturing stdout and stderr separately.
func runAgentSplit(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := agentCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}

const providerDeprecationLine = "--provider is deprecated, use --cli"

func TestAgentProviderFlagIsHiddenDeprecatedAlias(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")

	canonical, cliErr, err := runAgentSplit(t, "--cli", "codex", "--print")
	if err != nil {
		t.Fatalf("agent --cli codex --print: %v", err)
	}
	if cliErr != "" {
		t.Errorf("--cli must not warn, stderr = %q", cliErr)
	}

	alias, aliasErr, err := runAgentSplit(t, "--provider", "codex", "--print")
	if err != nil {
		t.Fatalf("agent --provider codex --print: %v", err)
	}
	if alias != canonical {
		t.Errorf("--provider stdout = %q, want the same as --cli: %q", alias, canonical)
	}
	if !strings.Contains(aliasErr, providerDeprecationLine) {
		t.Errorf("--provider stderr = %q, want it to contain %q", aliasErr, providerDeprecationLine)
	}

	// Selector form: the alias re-resolves from that CLI's fills exactly like --cli.
	want, _, err := runAgentSplit(t, "review", "--cli", "codex", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := runAgentSplit(t, "review", "--provider", "codex", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("selector form differs between --provider and --cli:\n%s\nvs\n%s", got, want)
	}

	// Hidden: the canonical flag is documented, the alias is not.
	help, _, _ := runAgentSplit(t, "--help")
	if !strings.Contains(help, "--cli") || strings.Contains(help, "--provider") {
		t.Errorf("help must show --cli and hide --provider, got:\n%s", help)
	}
}

func TestAgentCLIAndProviderConflict(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")

	_, _, err := runAgentSplit(t, "review", "--cli", "codex", "--provider", "claude", "-o", "yaml")
	if err == nil {
		t.Fatal("--cli and --provider with different values must be a usage error")
	}
	for _, want := range []string{"--cli", "--provider"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("conflict error %q should name %s", err, want)
		}
	}

	out, errOut, err := runAgentSplit(t, "review", "--cli", "codex", "--provider", "codex", "-o", "yaml")
	if err != nil {
		t.Fatalf("the same value under both spellings is accepted: %v", err)
	}
	if !strings.Contains(out, "cli: codex") || !strings.Contains(errOut, providerDeprecationLine) {
		t.Errorf("same-value run: stdout %q / stderr %q", out, errOut)
	}
}

func TestAgentEmptyProviderFlagKeepsExplicitEmptySemantics(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	// An explicit empty --cli= is a supplied flag, and so is --provider=: both are
	// the lookup failure, never a silent fallback to the depth knob.
	for _, flag := range []string{"--cli=", "--provider="} {
		if _, _, err := runAgentSplit(t, flag, "--print"); err == nil {
			t.Errorf("agent %s --print must fail the CLI lookup", flag)
		}
	}
}

func TestAgentYAMLDualCLIAndProviderKeys(t *testing.T) {
	agentTestRepo(t, "agent:\n  profiles:\n    review: { cli: codex }\n")
	out, errOut, err := runAgentSplit(t, "review", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
	for _, want := range []string{
		"cli: codex\nprovider: codex\nmodel:",
		"source:\n    cli: agent.profiles.review\n    provider: agent.profiles.review\n    model: clis.codex.profiles.review\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("-o yaml missing %q in:\n%s", want, out)
		}
	}

	bare, _, err := runAgentSplit(t, "--cli", "codex", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bare, "kind: cli\n") || strings.Contains(bare, "kind: provider") {
		t.Errorf("bare --cli form must report kind: cli, got:\n%s", bare)
	}
}

func TestAgentOldSpellingConfigResolvesIdentically(t *testing.T) {
	old := "providers:\n  codex:\n    profiles:\n      review: { model: pinned-x }\nagent:\n  profiles:\n    review: { provider: codex }\n"
	cur := "clis:\n  codex:\n    profiles:\n      review: { model: pinned-x }\nagent:\n  profiles:\n    review: { cli: codex }\n"

	agentTestRepo(t, cur)
	want, wantErr, err := runAgentSplit(t, "review", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	agentTestRepo(t, old)
	got, gotErr, err := runAgentSplit(t, "review", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("old-spelling config resolved differently:\n%s\nvs\n%s", got, want)
	}
	if !strings.Contains(got, "cli: codex") || !strings.Contains(got, "model: pinned-x") {
		t.Errorf("old-spelling config did not resolve cli codex / model pinned-x:\n%s", got)
	}
	if gotErr != "" || wantErr != "" {
		t.Errorf("reads of either spelling must be silent, stderr = %q / %q", gotErr, wantErr)
	}
}

func TestResolveAgentProviderFlagAliasKeepsProviderLine(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	cmd := resolveAgentCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"review", "--provider", "codex"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\nprovider=codex\n") {
		t.Errorf("the byte-stable provider= line must stay, got %q", out.String())
	}
	if strings.Contains(out.String(), "cli=") {
		t.Errorf("resolve-agent must not gain a cli= line, got %q", out.String())
	}
	if !strings.Contains(errOut.String(), providerDeprecationLine) {
		t.Errorf("stderr = %q, want %q", errOut.String(), providerDeprecationLine)
	}
}

func TestPaneOpenProviderFlagAlias(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\nclis:\n  myagent:\n    headless_command: \"myagent run\"\n")

	// The alias reaches the same resolution error as --cli, plus the deprecation line.
	_, stderr, err := runPaneCmd(t, "open", "--provider", "myagent")
	if err == nil || !strings.Contains(err.Error(), `CLI "myagent" has no interactive_command`) {
		t.Fatalf("--provider myagent error = %v", err)
	}
	if !strings.Contains(stderr, providerDeprecationLine) {
		t.Errorf("stderr = %q, want %q", stderr, providerDeprecationLine)
	}

	if _, _, err := runPaneCmd(t, "open"); err == nil || !strings.Contains(err.Error(), `"cli"`) {
		t.Errorf("open with neither flag must name the missing cli flag, got %v", err)
	}
	if _, _, err := runPaneCmd(t, "open", "--cli", "a", "--provider", "b"); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Errorf("conflicting flags must be a usage error, got %v", err)
	}
}

// --- fab config set / unset with the deprecated keys ---

func readProjectConfig(t *testing.T, repo string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, "fab", "project", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func liveLines(doc string) string {
	if i := strings.Index(doc, "# >>> fab reference"); i >= 0 {
		doc = doc[:i]
	}
	return doc
}

func TestConfigSetOldKeyWritesNewKeyAndWarns(t *testing.T) {
	t.Run("system scope replaces the old twin", func(t *testing.T) {
		_, home := setupConfigRepo(t, "")
		writeSystemConfig(t, home, "agent:\n  profiles:\n    review:\n      provider: codex\n")

		stdout, stderr, err := runConfigSplit(t, "set", "--system", "agent.profiles.review.provider", "claude")
		if err != nil {
			t.Fatalf("set old key: %v", err)
		}
		if !strings.Contains(stdout, "Set agent.profiles.review.cli ") {
			t.Errorf("stdout = %q, want it to report the mapped key", stdout)
		}
		if !strings.Contains(stderr, "agent.profiles.review.provider is deprecated, wrote agent.profiles.review.cli") {
			t.Errorf("stderr = %q, want the deprecation warning", stderr)
		}
		data, _ := os.ReadFile(filepath.Join(home, ".fab-kit", "config.yaml"))
		live := liveLines(string(data))
		if !strings.Contains(live, "cli: claude") || strings.Contains(live, "provider") {
			t.Errorf("system file must hold the single new spelling, got:\n%s", live)
		}
	})

	t.Run("project scope maps the providers table too", func(t *testing.T) {
		repo, _ := setupConfigRepo(t, "providers:\n  codex:\n    headless_command: old-exec\n")
		_, stderr, err := runConfigSplit(t, "set", "providers.codex.headless_command", "new-exec")
		if err != nil {
			t.Fatalf("set providers.… : %v", err)
		}
		if !strings.Contains(stderr, "providers.codex.headless_command is deprecated, wrote clis.codex.headless_command") {
			t.Errorf("stderr = %q", stderr)
		}
		live := liveLines(readProjectConfig(t, repo))
		if !strings.Contains(live, "clis:") || !strings.Contains(live, "headless_command: new-exec") || strings.Contains(live, "old-exec") {
			t.Errorf("project file after set:\n%s", live)
		}
	})

	t.Run("new key set drops a stale old twin silently", func(t *testing.T) {
		repo, _ := setupConfigRepo(t, "agent:\n  profiles:\n    review:\n      provider: codex\n")
		_, stderr, err := runConfigSplit(t, "set", "agent.profiles.review.cli", "claude")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stderr, "deprecated") {
			t.Errorf("setting the new key must not warn, stderr = %q", stderr)
		}
		live := liveLines(readProjectConfig(t, repo))
		if strings.Contains(live, "provider") || !strings.Contains(live, "cli: claude") {
			t.Errorf("one spelling must remain, got:\n%s", live)
		}
	})

	t.Run("the value gate runs on the mapped key and writes nothing", func(t *testing.T) {
		repo, _ := setupConfigRepo(t, "project:\n  name: x\n")
		before := readProjectConfig(t, repo)
		_, _, err := runConfigSplit(t, "set", "agent.profiles.review.provider", "no-such-cli")
		if err == nil || !strings.Contains(err.Error(), "agent.profiles.review.cli") {
			t.Fatalf("error = %v, want a refusal naming the mapped key", err)
		}
		if after := readProjectConfig(t, repo); after != before {
			t.Errorf("a refused set must not write:\n%s", after)
		}
	})
}

func TestConfigUnsetRemovesBothSpellings(t *testing.T) {
	for _, key := range []string{"agent.profiles.review.cli", "agent.profiles.review.provider"} {
		t.Run(key, func(t *testing.T) {
			repo, _ := setupConfigRepo(t, "agent:\n  profiles:\n    review:\n      provider: codex\n      model: m\n")
			stdout, stderr, err := runConfigSplit(t, "unset", key)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout, "Unset agent.profiles.review.cli") {
				t.Errorf("stdout = %q, want the unset reported", stdout)
			}
			if strings.HasSuffix(key, ".provider") != strings.Contains(stderr, key+" is deprecated") {
				t.Errorf("stderr = %q: the warning belongs to the old key only", stderr)
			}
			live := liveLines(readProjectConfig(t, repo))
			if strings.Contains(live, "provider") || strings.Contains(live, "cli:") || !strings.Contains(live, "model: m") {
				t.Errorf("both spellings must go and the sibling stay, got:\n%s", live)
			}
		})
	}

	t.Run("new and old live together", func(t *testing.T) {
		repo, _ := setupConfigRepo(t, "agent:\n  profiles:\n    review:\n      cli: claude\n      provider: codex\n")
		if _, _, err := runConfigSplit(t, "unset", "agent.profiles.review.cli"); err != nil {
			t.Fatal(err)
		}
		if live := liveLines(readProjectConfig(t, repo)); strings.Contains(live, "claude") || strings.Contains(live, "codex") {
			t.Errorf("both leaves must be removed, got:\n%s", live)
		}
	})

	t.Run("neither live is a no-op notice", func(t *testing.T) {
		setupConfigRepo(t, "project:\n  name: x\n")
		stdout, _, err := runConfigSplit(t, "unset", "agent.profiles.review.provider")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stdout, "not set") {
			t.Errorf("stdout = %q, want the not-set notice", stdout)
		}
	})
}

func TestConfigExplainAnswersDeprecatedKeys(t *testing.T) {
	setupConfigRepo(t, "")
	stdout, stderr, err := runConfigSplit(t, "explain", "providers")
	if err != nil {
		t.Fatalf("explain providers: %v", err)
	}
	canonical, _, err := runConfigSplit(t, "explain", "clis")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "providers is the deprecated spelling of clis") || !strings.HasSuffix(stdout, canonical) {
		t.Errorf("explain providers must be the clis explanation plus the deprecation note, got:\n%s", stdout[:min(len(stdout), 300)])
	}
	if stderr != "" {
		t.Errorf("stderr = %q", stderr)
	}

	jsonOut, jsonErr, err := runConfigSplit(t, "explain", "providers", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jsonOut, `"key": "clis"`) || strings.Contains(jsonOut, "deprecated spelling") {
		t.Errorf("--json stdout must stay a clean clis document, got:\n%s", jsonOut)
	}
	if !strings.Contains(jsonErr, "providers is the deprecated spelling of clis") {
		t.Errorf("--json note belongs on stderr, got %q", jsonErr)
	}
}
