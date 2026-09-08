package main

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/shellquote"
	"github.com/spf13/cobra"
)

var skillPromptName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func skillPromptCmd() *cobra.Command {
	var provider, repo string
	var quoted, jsonFlag bool
	cmd := &cobra.Command{
		Use:   "skill-prompt <skill> [arguments]",
		Short: "Print a skill invocation for the receiving agent without sending it",
		Long: "Render a bare skill name and optional argument string as a prompt.\n" +
			"Codex uses $skill; every other provider (including unknown or omitted) uses\n" +
			"/skill. --provider names the RECEIVING agent; no config is read by default.\n" +
			"For a fresh default-role launch, --repo resolves that repo's session provider\n" +
			"without checking stage-dispatch capabilities. Do not use --repo for an\n" +
			"existing pane: pass its live receiver identity with --provider instead.\n\n" +
			"Pass arguments as one quoted string, after -- if they begin with a dash.\n" +
			"Argument text is preserved literally. --shell-quote prints one POSIX shell\n" +
			"token for embedding in a launch command; --json prints provider, skill, and\n" +
			"prompt. Neither mode launches an agent or sends keys.\n\n" +
			"Exit codes: 0 rendered; 1 config/read or output failure; 2 invalid usage.",
		Example: `  fab skill-prompt --provider codex fab-fff ab12
  fab skill-prompt --provider custom --shell-quote fab-new 'Fix $HOME handling'
  fab skill-prompt --json --provider claude fab-ff -- '--light ab12'`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(1, 2)(cmd, args); err != nil {
				return err
			}
			if !skillPromptName.MatchString(args[0]) {
				return fmt.Errorf("invalid skill name %q: use a bare name starting with a letter or underscore, followed by letters, digits, hyphens or underscores (no / or $ prefix)", args[0])
			}
			if cmd.Flags().Changed("repo") && repo == "" {
				return fmt.Errorf("--repo requires a nonempty target repository path")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			receiver := provider
			if cmd.Flags().Changed("repo") {
				cfg, err := loadRepoConfig(repo)
				if err != nil {
					return err
				}
				profile, err := agent.ResolveRole(cfg, agent.RoleDefault)
				if err != nil {
					return err
				}
				receiver = profile.Provider
			}
			arguments := ""
			if len(args) == 2 {
				arguments = args[1]
			}
			prompt := agent.SkillPrompt(receiver, args[0], arguments)
			if jsonFlag {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
					Provider string `json:"provider"`
					Skill    string `json:"skill"`
					Prompt   string `json:"prompt"`
				}{receiver, args[0], prompt})
			}
			if quoted {
				prompt = shellquote.Single(prompt)
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), prompt)
			return err
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "receiving provider name (codex uses $; all other or omitted names use /)")
	cmd.Flags().StringVar(&repo, "repo", "", "resolve the target repo's default-role provider for a fresh launch (mutually exclusive with --provider)")
	cmd.Flags().BoolVar(&quoted, "shell-quote", false, "print the prompt quoted as one POSIX shell token")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "print provider, skill, and prompt as JSON")
	cmd.MarkFlagsMutuallyExclusive("shell-quote", "json")
	cmd.MarkFlagsMutuallyExclusive("provider", "repo")
	return cmd
}
