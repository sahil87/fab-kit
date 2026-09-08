package agent

import "testing"

func TestSkillPrompt(t *testing.T) {
	for _, tc := range []struct{ provider, prefix string }{
		{"codex", "$"}, {"claude", "/"}, {"agy", "/"}, {"kimi", "/"},
		{"custom", "/"}, {"", "/"}, {"codex-wrapper", "/"},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			for _, arguments := range []string{"", "ab12", "  $HOME /fab-new 'quoted'\nnext\n"} {
				want := tc.prefix + "fab-fff"
				if arguments != "" {
					want += " " + arguments
				}
				if got := SkillPrompt(tc.provider, "fab-fff", arguments); got != want {
					t.Errorf("SkillPrompt(%q, %q) = %q, want %q", tc.provider, arguments, got, want)
				}
			}
		})
	}
}
