package agent

// SkillPrompt renders an explicit skill invocation for the receiving provider.
// Arguments are prompt text, not shell syntax, and are preserved verbatim.
// Callers embedding the result in a shell command must quote it separately.
func SkillPrompt(provider, skill, arguments string) string {
	prefix := "/"
	if provider == "codex" {
		prefix = "$"
	}
	prompt := prefix + skill
	if arguments != "" {
		prompt += " " + arguments
	}
	return prompt
}
