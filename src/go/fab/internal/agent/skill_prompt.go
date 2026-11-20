package agent

// SkillPrefix returns the explicit-skill-invocation prefix the receiving
// CLI understands: "$" for the exact CLI name "codex", "/" for every
// other name — built-in, custom, unknown, or empty. This is the single owner of
// the rule; fab agent -o yaml exposes it as skill_prefix and the skill-bearing
// launchers render through SkillPrompt.
func SkillPrefix(cli string) string {
	if cli == "codex" {
		return "$"
	}
	return "/"
}

// SkillPrompt renders <prefix><skill>[ <arguments>] for the receiving CLI.
// Arguments are prompt text, not shell syntax, and are preserved verbatim.
// Callers embedding the result in a shell command must quote it separately.
func SkillPrompt(cli, skill, arguments string) string {
	prompt := SkillPrefix(cli) + skill
	if arguments != "" {
		prompt += " " + arguments
	}
	return prompt
}
