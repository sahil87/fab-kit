package main

import (
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
)

// `fab agent [role] [-- <agent-args>...]` forwards everything after `--` to the
// launched agent CLI. The split is cobra's ArgsLenAtDash(), because cobra strips
// the dash and the passthrough is otherwise indistinguishable from a second
// positional. Tokens are shell-quoted on the way in: the composed line is run
// through `sh -c` and printed verbatim by --print, so it survives one more round
// of word splitting.

// TestAgentPassthrough_NoRole: `fab agent -- --resume` forwards with the default
// role — the form a session wrapper uses.
func TestAgentPassthrough_NoRole(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t, "--", "--resume")
	if err != nil {
		t.Fatalf("agent -- --resume: %v", err)
	}
	model, effort := roleFill(t, agent.RoleDefault)
	want := strings.TrimSuffix(builtinClaudeCommand(model, effort), "\n") + " '--resume'\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestAgentPassthrough_WithRole: the arg before `--` is still the role.
func TestAgentPassthrough_WithRole(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t, "operator", "--", "--resume")
	if err != nil {
		t.Fatalf("agent operator -- --resume: %v", err)
	}
	model, effort := roleFill(t, agent.RoleOperator)
	if !strings.Contains(out, model) || !strings.Contains(out, effort) {
		t.Errorf("output = %q, want the operator role's fill (%s/%s)", out, model, effort)
	}
	if !strings.HasSuffix(out, " '--resume'\n") {
		t.Errorf("output = %q, want the passthrough appended", out)
	}
}

// TestAgentPassthrough_QuotesEachToken is the load-bearing one: a token with a
// space must reach the agent CLI as ONE argument. Without per-token quoting the
// composed line word-splits it in `sh -c` and `-p "two words"` becomes two args.
func TestAgentPassthrough_QuotesEachToken(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t, "--", "-p", "two words")
	if err != nil {
		t.Fatalf("agent -- -p 'two words': %v", err)
	}
	if !strings.HasSuffix(out, " '-p' 'two words'\n") {
		t.Errorf("output = %q, want each token separately single-quoted", out)
	}
}

// TestAgentPassthrough_NeutralizesShellMetacharacters: a passthrough token is
// DATA for the agent CLI, never script for the intermediate shell. Command
// substitution, a statement separator and a redirect must all arrive literal.
func TestAgentPassthrough_NeutralizesShellMetacharacters(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t, "--", "-p", "$(touch pwned); rm -rf / > x")
	if err != nil {
		t.Fatalf("agent with metacharacters: %v", err)
	}
	if !strings.HasSuffix(out, " '-p' '$(touch pwned); rm -rf / > x'\n") {
		t.Errorf("output = %q, want the whole token inert inside single quotes", out)
	}
}

// TestAgentPassthrough_EscapesEmbeddedSingleQuote: the close/escape/reopen
// sequence, the one case naive single-quoting gets wrong.
func TestAgentPassthrough_EscapesEmbeddedSingleQuote(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t, "--", "it's")
	if err != nil {
		t.Fatalf("agent -- \"it's\": %v", err)
	}
	if !strings.HasSuffix(out, ` 'it'\''s'`+"\n") {
		t.Errorf("output = %q, want the embedded quote escaped as '\\''", out)
	}
}

// TestAgentPassthrough_ProviderPathToo: passthrough is appended after BOTH
// addressing modes, so the two cannot diverge on it.
func TestAgentPassthrough_ProviderPathToo(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t, "--provider", "claude", "--model", "m", "--effort", "e", "--", "--resume")
	if err != nil {
		t.Fatalf("agent --provider claude -- --resume: %v", err)
	}
	if !strings.HasSuffix(out, " '--resume'\n") {
		t.Errorf("output = %q, want the passthrough appended on the provider path too", out)
	}
}

// TestAgentPassthrough_FlagsAfterDashAreNotFabs: `--print` and `--repo` after the
// dash belong to the agent CLI. If cobra parsed them as fab's own, `--repo
// /bogus` would change resolution (or fail) instead of being forwarded.
func TestAgentPassthrough_FlagsAfterDashAreNotFabs(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t, "--", "--repo", "/bogus")
	if err != nil {
		t.Fatalf("agent -- --repo /bogus: %v", err)
	}
	if !strings.HasSuffix(out, " '--repo' '/bogus'\n") {
		t.Errorf("output = %q, want post-dash flags forwarded verbatim, not consumed by fab", out)
	}
}

// TestAgentPassthrough_AbsentIsByteIdentical: a no-passthrough invocation must be
// unchanged from before this existed, so nothing downstream sees a trailing space.
func TestAgentPassthrough_AbsentIsByteIdentical(t *testing.T) {
	noProjectDir(t)

	out, err := runAgentPrint(t)
	if err != nil {
		t.Fatalf("agent --print: %v", err)
	}
	model, effort := roleFill(t, agent.RoleDefault)
	if want := builtinClaudeCommand(model, effort); out != want {
		t.Errorf("output = %q, want the unchanged composed command %q", out, want)
	}
}

// TestAgentPassthrough_TooManyArgsBeforeDash: arity is validated on the args
// BEFORE the dash. cobra.MaximumNArgs(1) counted the passthrough as extra roles,
// which is exactly why `fab agent -- --resume` used to fail.
func TestAgentPassthrough_TooManyArgsBeforeDash(t *testing.T) {
	noProjectDir(t)

	if _, err := runAgentPrint(t, "a", "b", "--", "--resume"); err == nil {
		t.Fatal("two args before `--` succeeded, want an arity error")
	}
	// And without a dash, the old arity rule still holds — with a hint.
	_, err := runAgentPrint(t, "a", "b")
	if err == nil {
		t.Fatal("two positionals succeeded, want an arity error")
	}
	if !strings.Contains(err.Error(), "after `--`") {
		t.Errorf("error = %v, want it to point at the `--` form", err)
	}
}
