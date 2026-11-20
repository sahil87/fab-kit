package agent

import (
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

func TestResolutionLines(t *testing.T) {
	resolution := Resolution{
		CLI:        "oracle",
		Model:      "claude-haiku-4-5-20251001",
		ModelAlias: "haiku",
		Effort:     "high",
		Dispatch: &DispatchResolution{
			Rung:    "headless",
			Command: "oracle -m claude-haiku-4-5-20251001",
		},
	}
	if got, want := resolution.Lines(false), "model=claude-haiku-4-5-20251001\neffort=high\nprovider=oracle\ndispatch=oracle -m claude-haiku-4-5-20251001\n"; got != want {
		t.Errorf("full projection = %q, want %q", got, want)
	}
	if got, want := resolution.Lines(true), "model=haiku\neffort=high\nprovider=oracle\ndispatch=oracle -m claude-haiku-4-5-20251001\n"; got != want {
		t.Errorf("alias projection = %q, want %q", got, want)
	}

	if got, want := (Resolution{}).Lines(true), "model=\n"; got != want {
		t.Errorf("empty projection = %q, want %q", got, want)
	}
	if got, want := (Resolution{Model: "gpt-5"}).Lines(true), "model=gpt-5\n"; got != want {
		t.Errorf("non-Claude alias projection = %q, want %q", got, want)
	}
	// skill_prefix rides the YAML projection only; the deprecated-compat line
	// protocol is byte-stable and must never grow a prefix line.
	if got, want := (Resolution{CLI: "codex", SkillPrefix: "$"}).Lines(false), "model=\nprovider=codex\n"; got != want {
		t.Errorf("skill_prefix leaked into Lines: %q, want %q", got, want)
	}
}

func TestResolveRoleWithSource(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		role       string
		overrides  Overrides
		want       Profile
		wantSource Source
	}{
		{
			name:       "flags",
			role:       RoleDoing,
			overrides:  Overrides{CLI: "oracle", CLISet: true, Model: "m", ModelSet: true, Effort: "e", EffortSet: true},
			want:       Profile{CLI: "oracle", Model: "m", Effort: "e"},
			wantSource: Source{CLI: "flag", Model: "flag", Effort: "flag"},
		},
		{
			name: "agent profile",
			cfg: &config.Config{Agent: config.AgentConfig{Profiles: map[string]config.RoleProfile{
				RoleDoing: {CLI: "oracle", Model: "m", Effort: "e"},
			}}},
			role:       RoleDoing,
			want:       Profile{CLI: "oracle", Model: "m", Effort: "e"},
			wantSource: Source{CLI: "agent.profiles.doing", Model: "agent.profiles.doing", Effort: "agent.profiles.doing"},
		},
		{
			name: "session knob and CLI default fill",
			cfg: &config.Config{
				Agent: config.AgentConfig{Session: "oracle"},
				CLIs: map[string]config.CLIConfig{"oracle": {Profiles: map[string]config.CLIProfile{
					RoleDefault: {Model: "m", Effort: "e"},
				}}},
			},
			role:       RoleOperator,
			want:       Profile{CLI: "oracle", Model: "m", Effort: "e"},
			wantSource: Source{CLI: "agent.session", Model: "clis.oracle.profiles.default", Effort: "clis.oracle.profiles.default"},
		},
		{
			name: "workers knob and mixed CLI fills",
			cfg: &config.Config{
				Agent: config.AgentConfig{Workers: "oracle"},
				CLIs: map[string]config.CLIConfig{"oracle": {Profiles: map[string]config.CLIProfile{
					RoleDefault: {Effort: "medium"},
					RoleDoing:   {Model: "role-model"},
				}}},
			},
			role:       RoleDoing,
			want:       Profile{CLI: "oracle", Model: "role-model", Effort: "medium"},
			wantSource: Source{CLI: "agent.workers", Model: "clis.oracle.profiles.doing", Effort: "clis.oracle.profiles.default"},
		},
		{
			name:       "built-in CLI and fills",
			role:       RoleDoing,
			want:       mustDefaultProfile(t, RoleDoing),
			wantSource: Source{CLI: "built-in", Model: "clis.claude.profiles.doing", Effort: "clis.claude.profiles.doing"},
		},
		{
			name: "empty inherit source",
			cfg: &config.Config{
				Agent: config.AgentConfig{Workers: "oracle"},
				CLIs:  map[string]config.CLIConfig{"oracle": {}},
			},
			role:       RoleDoing,
			want:       Profile{CLI: "oracle"},
			wantSource: Source{CLI: "agent.workers"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, source, err := ResolveRoleWithSource(tc.cfg, tc.role, tc.overrides)
			if err != nil {
				t.Fatalf("ResolveRoleWithSource: %v", err)
			}
			if got != tc.want {
				t.Errorf("profile = %+v, want %+v", got, tc.want)
			}
			if source != tc.wantSource {
				t.Errorf("source = %+v, want %+v", source, tc.wantSource)
			}
		})
	}
}

func mustDefaultProfile(t *testing.T, role string) Profile {
	t.Helper()
	p, ok := DefaultProfile(role)
	if !ok {
		t.Fatalf("DefaultProfile(%q) is unknown", role)
	}
	return p
}

// TestResolutionYAMLDualCLIAndProviderKeys pins the -o yaml schema through the
// provider → cli deprecation window (261120-f2ly): `cli:` takes the slot
// `provider:` occupied, followed immediately by a duplicate `provider:`, and the
// same for `source`; every pre-existing key keeps its relative order.
func TestResolutionYAMLDualCLIAndProviderKeys(t *testing.T) {
	data, err := (Resolution{
		Selector: "review", Kind: "role", Role: "review",
		CLI: "codex", Model: "m", Effort: "high", Command: "codex -m m",
		Source:      Source{CLI: "agent.profiles.review", Model: "clis.codex.profiles.review", Effort: "clis.codex.profiles.review"},
		SkillPrefix: "$",
	}).YAML()
	if err != nil {
		t.Fatal(err)
	}
	want := "selector: review\nkind: role\nrole: review\ncli: codex\nprovider: codex\nmodel: m\neffort: high\ncommand: codex -m m\n" +
		"model_alias: \"\"\ntemplate: \"\"\nfill_mode: \"\"\n" +
		"source:\n    cli: agent.profiles.review\n    provider: agent.profiles.review\n    model: clis.codex.profiles.review\n    effort: clis.codex.profiles.review\n" +
		"skill_prefix: $\n"
	if string(data) != want {
		t.Errorf("YAML =\n%s\nwant\n%s", data, want)
	}

	// The duplicate is filled by YAML() alone; the Go value never carries it.
	r := Resolution{CLI: "codex"}
	if _, err := r.YAML(); err != nil {
		t.Fatal(err)
	}
	if r.Provider != "" || r.Source.Provider != "" {
		t.Errorf("YAML() must not mutate the receiver: %+v", r)
	}
}
