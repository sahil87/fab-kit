package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
)

// agentTestRepo creates a temp repo with fab/project/config.yaml holding the
// given config body and chdirs into the repo root (cwd restored on cleanup).
func agentTestRepo(t *testing.T, configBody string) string {
	t.Helper()
	root := t.TempDir()
	projectDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configBody), 0o644); err != nil {
		t.Fatal(err)
	}
	chdirTestEnv(t, root, map[string]string{"TMUX": ""})
	return root
}

// runAgentPrint executes `fab agent --print` with the given extra args.
func runAgentPrint(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := agentCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(append([]string{"--print"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func TestAgentWorkersOverride(t *testing.T) {
	agentTestRepo(t, `providers:
  claude:
    session_command: "claude"
`)

	t.Run("exec environment receives the supplied value", func(t *testing.T) {
		originalExec := execAgent
		t.Cleanup(func() { execAgent = originalExec })

		var gotPath string
		var gotArgv, gotEnv []string
		execAgent = func(path string, argv, env []string) error {
			gotPath = path
			gotArgv = append([]string(nil), argv...)
			gotEnv = append([]string(nil), env...)
			return nil
		}

		cmd := agentCmd()
		cmd.SetArgs([]string{"--workers", "kimi3"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("agent --workers: %v", err)
		}
		if gotPath != "/bin/sh" {
			t.Errorf("exec path = %q, want /bin/sh", gotPath)
		}
		if len(gotArgv) != 3 || gotArgv[1] != "-c" || strings.Contains(gotArgv[2], agentWorkersEnv) {
			t.Errorf("exec argv = %#v; workers override must remain out of the resolved command", gotArgv)
		}
		want := agentWorkersEnv + "=kimi3"
		found := false
		for _, entry := range gotEnv {
			if entry == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("exec environment missing %q", want)
		}
	})

	t.Run("print output remains the resolved command only", func(t *testing.T) {
		out, err := runAgentPrint(t, "--workers", "unregistered-provider")
		if err != nil {
			t.Fatalf("agent --print --workers: %v", err)
		}
		if strings.Contains(out, agentWorkersEnv) || strings.Contains(out, "unregistered-provider") {
			t.Errorf("--workers leaked into --print output: %q", out)
		}
	})
}

// TestAgentPrintDefaultRole: `fab agent --print` with no role arg resolves the
// `default` role (claude/claude-fable-5/high) and appends the profile to the
// non-templated claude session command.
func TestAgentPrintDefaultRole(t *testing.T) {
	agentTestRepo(t, `providers:
  claude:
    session_command: "claude --dangerously-skip-permissions"
`)
	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	want := "claude --dangerously-skip-permissions --model claude-fable-5 --effort high\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentPrintOperatorRole: `fab agent operator --print` resolves the `operator`
// role (claude-sonnet-5/medium).
func TestAgentPrintOperatorRole(t *testing.T) {
	agentTestRepo(t, `providers:
  claude:
    session_command: "claude"
`)
	out, err := runAgentPrint(t, "operator")
	if err != nil {
		t.Fatalf("agent operator --print: %v", err)
	}
	want := "claude --model claude-sonnet-5 --effort medium\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentPrintTemplatedSessionCommand: a templated session_command has the
// resolved profile substituted (not appended); no literal braces survive. The role
// profile pins the model/effort because `codex` supplies no per-role fills of its
// own (260806-j9nh) — model/effort come from the role profile, that provider's
// fills, or a flag.
func TestAgentPrintTemplatedSessionCommand(t *testing.T) {
	agentTestRepo(t, `providers:
  codex:
    session_command: "codex -m {model} -c model_reasoning_effort={effort}"
agent:
  profiles:
    default: { provider: codex, model: gpt-5.3-codex, effort: high }
`)
	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	want := "codex -m gpt-5.3-codex -c model_reasoning_effort=high\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentPrintProviderFillSuppliesTemplate: a role profile that pins no
// model/effort takes them from the named provider's DEFAULT FILL (the deprecated
// flat `providers.<name>.model`/`.effort` spelling of `profiles.default`) — the
// "extra set of config that fills the templates". This is the fill-precedence rung
// between an explicit `agent.profiles.<role>` field and empty.
func TestAgentPrintProviderFillSuppliesTemplate(t *testing.T) {
	agentTestRepo(t, `providers:
  codex:
    model: gpt-5.3-codex
    effort: high
agent:
  profiles:
    default: { provider: codex }
`)
	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	// The command grammar comes from the codex BUILT-IN (no session_command
	// configured), the fill from the provider entry.
	want := "codex --dangerously-bypass-approvals-and-sandbox -m gpt-5.3-codex -c model_reasoning_effort=high\n"
	if out != want {
		t.Errorf("output = %q, want the built-in codex grammar filled from providers.codex %q", out, want)
	}
}

// TestAgentPrintBuiltinCodexTakesItsOwnFill: pointing the `default` role at the
// codex BUILT-IN with no providers: config composes codex's own shipped fill
// (260806-ywkx) — and crucially does NOT leak claude's `default`-role model across
// the provider switch: model/effort come only from the RESOLVED provider's own
// fills (260806-j9nh).
func TestAgentPrintBuiltinCodexTakesItsOwnFill(t *testing.T) {
	agentTestRepo(t, `agent:
  profiles:
    default: { provider: codex }
`)
	codex, _ := agent.ResolveProvider(nil, "codex")
	fill := codex.Profiles[agent.RoleDefault]

	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	want := "codex --dangerously-bypass-approvals-and-sandbox -m " + fill.Model + " -c model_reasoning_effort=" + fill.Effort + "\n"
	if out != want {
		t.Errorf("output = %q, want codex's own default-role fill %q", out, want)
	}
	if strings.Contains(out, "claude") {
		t.Errorf("output = %q — no claude model may be inherited across the provider switch", out)
	}
}

// TestAgentPrintBuiltinFallback: with no providers config, the built-in claude
// provider supplies the templated default session command, into which the
// `default`-role profile is substituted (the constant is a {model}/{effort}
// template, so this path resolves via substitution, not append).
func TestAgentPrintBuiltinFallback(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	// Pin the full resolved command: the built-in templated default with the
	// `default` role's {claude-fable-5, high} substituted into its placeholders.
	want := "claude --dangerously-skip-permissions -n \"$(basename \"$(pwd)\")\" --model claude-fable-5 --effort high\n"
	if out != want {
		t.Errorf("output = %q, want the `default`-role profile substituted into the templated built-in command %q", out, want)
	}
}

// TestAgentPrintUnknownRoleErrors: an unknown role name exits non-zero.
func TestAgentPrintUnknownRoleErrors(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentPrint(t, "bogus")
	if err == nil {
		t.Fatal("expected an error for an unknown role")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should name the unknown role, got: %v", err)
	}
}

// TestAgentPrintNoSessionCommandErrors: a resolved provider with no
// session_command errors with a config-key hint. The fixture uses a project-only
// provider (`myagent`) — every BUILT-IN provider now carries a session_command
// (claude, codex, gemini — 260805-j3cm), so only a user-defined dispatch-only
// provider can reach this error.
func TestAgentPrintNoSessionCommandErrors(t *testing.T) {
	agentTestRepo(t, `providers:
  myagent:
    dispatch_command: "myagent run"
agent:
  profiles:
    default: { provider: myagent }
`)
	_, err := runAgentPrint(t)
	if err == nil {
		t.Fatal("expected an error when the resolved provider has no session_command")
	}
	if !strings.Contains(err.Error(), "providers.myagent.session_command") {
		t.Errorf("error = %q, want the config-key hint", err.Error())
	}
}

// --- provider-addressed form (`fab agent --provider <name>`) ---

// TestAgentPrintProviderExplicitProfile: --provider bypasses role resolution and
// substitutes the explicitly supplied --model/--effort into the provider's
// templated session_command.
func TestAgentPrintProviderExplicitProfile(t *testing.T) {
	agentTestRepo(t, `providers:
  codex:
    session_command: "codex -m {model} -c model_reasoning_effort={effort}"
`)
	out, err := runAgentPrint(t, "--provider", "codex", "--model", "gpt-5.3-codex", "--effort", "high")
	if err != nil {
		t.Fatalf("agent --provider codex --print: %v", err)
	}
	want := "codex -m gpt-5.3-codex -c model_reasoning_effort=high\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentPrintProviderEmptyProfileDropsTokens: with --model/--effort omitted,
// the empty-value token-drop rule strips each placeholder token AND its preceding
// `-`-flag, so a bare `codex` invocation is composed and the CLI's own default
// model applies. This is the documented reason the provider form is usable without
// knowing the installed CLI's model IDs.
//
// The provider form BYPASSES role resolution, so codex's shipped per-role fills
// (260806-ywkx) are deliberately not consulted here — only the explicit flags are.
// The role path is covered by TestAgentPrintBuiltinCodexTakesItsOwnFill.
func TestAgentPrintProviderEmptyProfileDropsTokens(t *testing.T) {
	agentTestRepo(t, `providers:
  codex:
    session_command: "codex -m {model} -c model_reasoning_effort={effort}"
`)
	out, err := runAgentPrint(t, "--provider", "codex")
	if err != nil {
		t.Fatalf("agent --provider codex --print: %v", err)
	}
	if out != "codex\n" {
		t.Errorf("output = %q, want %q", out, "codex\n")
	}
}

// TestAgentPrintProviderEmptyProfileAppendsNothing: on a NON-templated
// session_command the append-mode empty-value rule omits both flags, so the
// command passes through unchanged (the append-mode counterpart of the test above).
func TestAgentPrintProviderEmptyProfileAppendsNothing(t *testing.T) {
	agentTestRepo(t, `providers:
  gemini:
    session_command: "gemini"
`)
	out, err := runAgentPrint(t, "--provider", "gemini")
	if err != nil {
		t.Fatalf("agent --provider gemini --print: %v", err)
	}
	if out != "gemini\n" {
		t.Errorf("output = %q, want %q (no --model/--effort appended)", out, "gemini\n")
	}
}

// TestAgentPrintProviderBypassesRole: the provider form ignores role resolution
// entirely — the `default` role's model/effort must NOT leak into the composed
// command (the role path would have substituted claude-fable-5/high).
func TestAgentPrintProviderBypassesRole(t *testing.T) {
	agentTestRepo(t, `providers:
  claude:
    session_command: "claude --model {model} --effort {effort}"
`)
	out, err := runAgentPrint(t, "--provider", "claude")
	if err != nil {
		t.Fatalf("agent --provider claude --print: %v", err)
	}
	if out != "claude\n" {
		t.Errorf("output = %q, want %q — the `default` role's profile must not leak into the provider path", out, "claude\n")
	}
}

// TestAgentPrintProviderBuiltinClaude: the built-in claude provider resolves on
// the provider path with no providers: block configured at all.
func TestAgentPrintProviderBuiltinClaude(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "--provider", "claude", "--model", "claude-opus-5", "--effort", "xhigh")
	if err != nil {
		t.Fatalf("agent --provider claude --print: %v", err)
	}
	want := "claude --dangerously-skip-permissions -n \"$(basename \"$(pwd)\")\" --model claude-opus-5 --effort xhigh\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentRoleAndProviderMutuallyExclusive: supplying both the [role] positional
// and --provider is a usage error — no command is printed or exec'd.
func TestAgentRoleAndProviderMutuallyExclusive(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "doing", "--provider", "claude")
	if err == nil {
		t.Fatal("expected an error when both [role] and --provider are given")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should state the mutual exclusion, got: %v", err)
	}
	// A usage error must not compose a command. (The standalone command under test
	// still writes cobra's usage block to its out buffer — the real root sets
	// SilenceUsage — so assert on the absence of a composed command, not emptiness.)
	if strings.Contains(out, "--dangerously-skip-permissions") {
		t.Errorf("no session command should be composed on a usage error, got %q", out)
	}
}

// TestAgentModelEffortRequireProvider: --model or --effort without --provider is a
// usage error (the role path's profile comes from the role, so a bare --model has
// no coherent semantics).
func TestAgentModelEffortRequireProvider(t *testing.T) {
	for _, flag := range []string{"--model", "--effort"} {
		t.Run(flag, func(t *testing.T) {
			agentTestRepo(t, "project:\n  name: test\n")
			out, err := runAgentPrint(t, flag, "somevalue")
			if err == nil {
				t.Fatalf("expected an error for %s without --provider", flag)
			}
			if !strings.Contains(err.Error(), "require --provider") {
				t.Errorf("error should say the flags require --provider, got: %v", err)
			}
			// See the note in TestAgentRoleAndProviderMutuallyExclusive: assert no
			// command was composed rather than an empty buffer (cobra's usage block).
			if strings.Contains(out, "--dangerously-skip-permissions") {
				t.Errorf("no session command should be composed on a usage error, got %q", out)
			}
		})
	}
}

// TestAgentEmptyProviderStillMutuallyExclusive: an EXPLICITLY EMPTY `--provider=`
// is a supplied flag, so the mutual exclusion with the [role] positional still
// trips. Pins the guard on cobra's Flag.Changed rather than on value emptiness —
// an emptiness test would let this invocation fall through to the role path and
// print a command.
func TestAgentEmptyProviderStillMutuallyExclusive(t *testing.T) {
	agentTestRepo(t, `providers:
  claude:
    session_command: "claude --dangerously-skip-permissions"
`)
	out, err := runAgentPrint(t, "doing", "--provider=")
	if err == nil {
		t.Fatal("expected an error for [role] plus an explicitly-empty --provider=")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should state the mutual exclusion, got: %v", err)
	}
	// See the note in TestAgentRoleAndProviderMutuallyExclusive: assert no command
	// was composed rather than an empty buffer (cobra's usage block).
	if strings.Contains(out, "--dangerously-skip-permissions") {
		t.Errorf("no session command should be composed on a usage error, got %q", out)
	}
}

// TestAgentEmptyProviderAloneIsLookupFailure: a bare `--provider=` (no role) takes
// the PROVIDER path — the empty name resolves to nothing, so it is the
// unknown-provider lookup failure, never a silent fallback to the `default` role.
// The error's config-key hint substitutes a `<name>` placeholder for the empty
// name, so it never suggests the malformed `providers.` path.
func TestAgentEmptyProviderAloneIsLookupFailure(t *testing.T) {
	agentTestRepo(t, `providers:
  claude:
    session_command: "claude --dangerously-skip-permissions"
`)
	out, err := runAgentPrint(t, "--provider=")
	if err == nil {
		t.Fatal("expected a lookup error for an explicitly-empty --provider=")
	}
	msg := err.Error()
	if !strings.Contains(msg, "unknown provider") {
		t.Errorf("error should be the unknown-provider lookup failure, got: %v", err)
	}
	if !strings.Contains(msg, "providers.<name>") {
		t.Errorf("hint should name the placeholder key path providers.<name>, got: %v", err)
	}
	if strings.Contains(msg, "providers. ") {
		t.Errorf("hint must not suggest the malformed path %q, got: %v", "providers.", err)
	}
	if strings.Contains(out, "--dangerously-skip-permissions") {
		t.Errorf("the role path must not run for --provider=, got %q", out)
	}
}

// TestAgentEmptyModelEffortStillRequireProvider: an explicitly empty `--model=` /
// `--effort=` without --provider is still the requires-provider usage error — the
// same Flag.Changed guard, on the flag-scoping side.
func TestAgentEmptyModelEffortStillRequireProvider(t *testing.T) {
	for _, arg := range []string{"--model=", "--effort="} {
		t.Run(arg, func(t *testing.T) {
			agentTestRepo(t, `providers:
  claude:
    session_command: "claude --dangerously-skip-permissions"
`)
			out, err := runAgentPrint(t, arg)
			if err == nil {
				t.Fatalf("expected an error for %s without --provider", arg)
			}
			if !strings.Contains(err.Error(), "require --provider") {
				t.Errorf("error should say the flags require --provider, got: %v", err)
			}
			if strings.Contains(out, "--dangerously-skip-permissions") {
				t.Errorf("no session command should be composed on a usage error, got %q", out)
			}
		})
	}
}

// TestAgentUnknownProviderNamesAvailable: an unknown provider name exits non-zero
// and the error lists the resolvable providers (project block ∪ built-in table).
func TestAgentUnknownProviderNamesAvailable(t *testing.T) {
	agentTestRepo(t, `providers:
  codex:
    session_command: "codex"
`)
	_, err := runAgentPrint(t, "--provider", "bogus")
	if err == nil {
		t.Fatal("expected an error for an unknown provider name")
	}
	msg := err.Error()
	if !strings.Contains(msg, "bogus") {
		t.Errorf("error should name the unknown provider, got: %v", err)
	}
	// Both the project-configured codex and the built-in claude are resolvable.
	for _, want := range []string{"claude", "codex"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error should list available provider %q, got: %v", want, err)
		}
	}
}

// TestAgentProviderNoSessionCommandErrors: a provider that resolves but carries no
// session_command errors with the config-key hint (the provider-path counterpart of
// TestAgentPrintNoSessionCommandErrors). Uses a project-only dispatch-only provider
// for the same reason: all three built-ins now ship a session_command.
func TestAgentProviderNoSessionCommandErrors(t *testing.T) {
	agentTestRepo(t, `providers:
  myagent:
    dispatch_command: "myagent run"
`)
	_, err := runAgentPrint(t, "--provider", "myagent")
	if err == nil {
		t.Fatal("expected an error when the named provider has no session_command")
	}
	if !strings.Contains(err.Error(), "providers.myagent.session_command") {
		t.Errorf("error = %q, want the config-key hint", err.Error())
	}
}

// TestAgentProviderBuiltinCodexNoConfig (260805-j3cm): `fab agent --provider codex`
// works with NO providers: block at all — codex is a built-in. With no --model the
// composition drops both placeholder tokens while preserving the fixed bypass
// flag, so the installed CLI's own default model applies. The provider form bypasses role
// resolution, so codex's shipped per-role fills (260806-ywkx) do not apply on this
// path — see TestAgentPrintBuiltinCodexTakesItsOwnFill for the role path.
func TestAgentProviderBuiltinCodexNoConfig(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")

	out, err := runAgentPrint(t, "--provider", "codex")
	if err != nil {
		t.Fatalf("agent --provider codex --print: %v", err)
	}
	if out != "codex --dangerously-bypass-approvals-and-sandbox\n" {
		t.Errorf("output = %q, want the model-free built-in codex grammar with its bypass flag", out)
	}

	out, err = runAgentPrint(t, "--provider", "gemini", "--model", "gemini-2.5-pro")
	if err != nil {
		t.Fatalf("agent --provider gemini --print: %v", err)
	}
	if out != "gemini --approval-mode=yolo -m gemini-2.5-pro\n" {
		t.Errorf("output = %q, want the built-in gemini grammar with the model substituted", out)
	}
}

// TestAgentPrintProviderWithRepoFlag: --provider composes with --repo, reading the
// TARGET repo's providers: table (not the current repo's).
func TestAgentPrintProviderWithRepoFlag(t *testing.T) {
	agentTestRepo(t, `providers:
  codex:
    session_command: "current-codex"
`)

	target := t.TempDir()
	projectDir := filepath.Join(target, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"),
		[]byte("providers:\n  codex:\n    session_command: \"target-codex -m {model}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runAgentPrint(t, "--provider", "codex", "--model", "some-model", "--repo", target)
	if err != nil {
		t.Fatalf("agent --provider codex --print --repo: %v", err)
	}
	want := "target-codex -m some-model\n"
	if out != want {
		t.Errorf("output = %q, want the target repo's session command %q", out, want)
	}
}

// TestAgentPrintRepoFlag: --repo reads a different repo's config.
func TestAgentPrintRepoFlag(t *testing.T) {
	// The current repo has no providers; the target repo does.
	agentTestRepo(t, "project:\n  name: current\n")

	target := t.TempDir()
	projectDir := filepath.Join(target, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"),
		[]byte("providers:\n  claude:\n    session_command: \"target-claude\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runAgentPrint(t, "--repo", target)
	if err != nil {
		t.Fatalf("agent --print --repo: %v", err)
	}
	if !strings.Contains(out, "target-claude") {
		t.Errorf("output = %q, want the target repo's session command", out)
	}
}
