package agent

// SkillPrefix returns the explicit-skill-invocation prefix the receiving
// provider understands: "$" for the exact provider name "codex", "/" for every
// other name — built-in, custom, unknown, or empty. This is the single owner of
// the rule; fab agent -o yaml exposes it as skill_prefix and the skill-bearing
// launchers render through SkillPrompt.
func SkillPrefix(provider string) string {
	if provider == "codex" {
		return "$"
	}
	return "/"
}

// SkillPrompt renders <prefix><skill>[ <arguments>] for the receiving provider.
// Arguments are prompt text, not shell syntax, and are preserved verbatim.
// Callers embedding the result in a shell command must quote it separately.
func SkillPrompt(provider, skill, arguments string) string {
	prompt := SkillPrefix(provider) + skill
	if arguments != "" {
		prompt += " " + arguments
	}
	return prompt
}
