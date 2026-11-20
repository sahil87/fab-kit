package agent

import "testing"

func TestSkillPrefix(t *testing.T) {
	for _, tc := range []struct{ cli, want string }{
		{"codex", "$"}, {"claude", "/"}, {"agy", "/"}, {"kimi", "/"},
		{"custom", "/"}, {"", "/"}, {"codex-wrapper", "/"}, {"Codex", "/"},
	} {
		if got := SkillPrefix(tc.cli); got != tc.want {
			t.Errorf("SkillPrefix(%q) = %q, want %q", tc.cli, got, tc.want)
		}
	}
}

func TestSkillPrompt(t *testing.T) {
	for _, tc := range []struct{ cli, prefix string }{
		{"codex", "$"}, {"claude", "/"}, {"agy", "/"}, {"kimi", "/"},
		{"custom", "/"}, {"", "/"}, {"codex-wrapper", "/"},
	} {
		t.Run(tc.cli, func(t *testing.T) {
			for _, arguments := range []string{"", "ab12", "  $HOME /fab-new 'quoted'\nnext\n"} {
				want := tc.prefix + "fab-fff"
				if arguments != "" {
					want += " " + arguments
				}
				if got := SkillPrompt(tc.cli, "fab-fff", arguments); got != want {
					t.Errorf("SkillPrompt(%q, %q) = %q, want %q", tc.cli, arguments, got, want)
				}
			}
		})
	}
}
