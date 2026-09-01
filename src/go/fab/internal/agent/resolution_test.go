package agent

import (
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

func TestResolutionLines(t *testing.T) {
	resolution := Resolution{
		Provider:   "oracle",
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
			overrides:  Overrides{Provider: "oracle", ProviderSet: true, Model: "m", ModelSet: true, Effort: "e", EffortSet: true},
			want:       Profile{Provider: "oracle", Model: "m", Effort: "e"},
			wantSource: Source{Provider: "flag", Model: "flag", Effort: "flag"},
		},
		{
			name: "agent profile",
			cfg: &config.Config{Agent: config.AgentConfig{Profiles: map[string]config.RoleProfile{
				RoleDoing: {Provider: "oracle", Model: "m", Effort: "e"},
			}}},
			role:       RoleDoing,
			want:       Profile{Provider: "oracle", Model: "m", Effort: "e"},
			wantSource: Source{Provider: "agent.profiles.doing", Model: "agent.profiles.doing", Effort: "agent.profiles.doing"},
		},
		{
			name: "session knob and provider default fill",
			cfg: &config.Config{
				Agent: config.AgentConfig{Session: "oracle"},
				Providers: map[string]config.ProviderConfig{"oracle": {Profiles: map[string]config.ProviderProfile{
					RoleDefault: {Model: "m", Effort: "e"},
				}}},
			},
			role:       RoleOperator,
			want:       Profile{Provider: "oracle", Model: "m", Effort: "e"},
			wantSource: Source{Provider: "agent.session", Model: "providers.oracle.profiles.default", Effort: "providers.oracle.profiles.default"},
		},
		{
			name: "workers knob and mixed provider fills",
			cfg: &config.Config{
				Agent: config.AgentConfig{Workers: "oracle"},
				Providers: map[string]config.ProviderConfig{"oracle": {Profiles: map[string]config.ProviderProfile{
					RoleDefault: {Effort: "medium"},
					RoleDoing:   {Model: "role-model"},
				}}},
			},
			role:       RoleDoing,
			want:       Profile{Provider: "oracle", Model: "role-model", Effort: "medium"},
			wantSource: Source{Provider: "agent.workers", Model: "providers.oracle.profiles.doing", Effort: "providers.oracle.profiles.default"},
		},
		{
			name:       "built-in provider and fills",
			role:       RoleDoing,
			want:       mustDefaultProfile(t, RoleDoing),
			wantSource: Source{Provider: "built-in", Model: "providers.claude.profiles.doing", Effort: "providers.claude.profiles.doing"},
		},
		{
			name: "empty inherit source",
			cfg: &config.Config{
				Agent:     config.AgentConfig{Workers: "oracle"},
				Providers: map[string]config.ProviderConfig{"oracle": {}},
			},
			role:       RoleDoing,
			want:       Profile{Provider: "oracle"},
			wantSource: Source{Provider: "agent.workers"},
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
