package agent

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Fill modes name the two composition paths owned by spawn.WithProfile.
const (
	FillModeTemplate = "template"
	FillModeAppend   = "append"
)

// Source records the precedence rung that supplied each profile field. An empty
// model or effort source is the inherit signal: no rung supplied a value.
type Source struct {
	CLI string `yaml:"cli"`
	// Provider duplicates CLI in the -o yaml document for the deprecation window
	// of the 261120-f2ly rename (provider → cli); YAML() fills it, nothing else
	// reads or writes it. Removed with the other pre-2.31 spellings.
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Effort   string `yaml:"effort"`
}

// DispatchResolution is the non-native dispatch projection. Native resolutions
// carry a nil dispatch pointer so yaml's omitempty rule omits dispatch entirely.
type DispatchResolution struct {
	Rung    string `yaml:"rung"`
	Command string `yaml:"command"`
	Reason  string `yaml:"reason,omitempty"`
}

// Resolution is the complete result shared by fab agent and fab resolve-agent.
// Change 1's seven YAML keys stay first and in their shipped relative order; the
// full-schema keys append after them.
type Resolution struct {
	Selector string `yaml:"selector"`
	Kind     string `yaml:"kind"`
	Role     string `yaml:"role"`
	CLI      string `yaml:"cli"`
	// Provider is the deprecated duplicate of CLI (same value) kept in the slot
	// right after it for the rename's deprecation window, so consumers of the
	// pre-2.31 `provider:` key keep working; YAML() fills it. fab resolve-agent's
	// byte-stable `provider=` line is a separate, unchanged surface (Lines).
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Effort   string `yaml:"effort"`
	Command  string `yaml:"command"`

	ModelAlias string              `yaml:"model_alias"`
	Template   string              `yaml:"template"`
	FillMode   string              `yaml:"fill_mode"`
	Source     Source              `yaml:"source"`
	Dispatch   *DispatchResolution `yaml:"dispatch,omitempty"`
	// SkillPrefix is SkillPrefix(CLI) — the receiving harness's explicit
	// skill-invocation prefix. Always present and always last (dispatch is
	// omitempty) so every consumer finds it at the same place; the Lines
	// projection deliberately never carries it.
	SkillPrefix string `yaml:"skill_prefix"`
}

// Lines projects a resolution onto fab resolve-agent's byte-stable line
// protocol. Alias changes only model=: recognized Claude IDs use ModelAlias;
// non-Claude and empty models retain Model verbatim. Dispatch always carries the
// already-composed command with the resolved model verbatim.
func (r Resolution) Lines(alias bool) string {
	model := r.Model
	if alias && r.ModelAlias != "" {
		model = r.ModelAlias
	}

	var out strings.Builder
	fmt.Fprintf(&out, "model=%s\n", model)
	if r.Effort != "" {
		fmt.Fprintf(&out, "effort=%s\n", r.Effort)
	}
	if r.CLI != "" {
		fmt.Fprintf(&out, "provider=%s\n", r.CLI)
	}
	if r.Dispatch != nil {
		fmt.Fprintf(&out, "dispatch=%s\n", r.Dispatch.Command)
	}
	return out.String()
}

// YAML projects the complete resolution as the ordered structured document used
// by fab agent -o yaml.
//
// During the provider → cli deprecation window the document carries `cli:` and a
// duplicate `provider:` with the same value, in Resolution and in Source.
func (r Resolution) YAML() ([]byte, error) {
	r.Provider = r.CLI
	r.Source.Provider = r.Source.CLI
	return yaml.Marshal(r)
}
