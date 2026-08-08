// Package configvalue parses the scalar and flow-collection values accepted by
// `fab config set`. It is dependency-free apart from yaml.v3 so the same
// semantics can be reused by other config input surfaces.
package configvalue

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Kind is the YAML value shape used by the config registry for validation.
type Kind string

const (
	KindString   Kind = "string"
	KindBool     Kind = "bool"
	KindInt      Kind = "int"
	KindFloat    Kind = "float"
	KindNull     Kind = "null"
	KindSequence Kind = "sequence"
	KindMapping  Kind = "mapping"
)

// Valid reports whether k is one of the supported registry value kinds.
func Valid(k Kind) bool {
	switch k {
	case KindString, KindBool, KindInt, KindFloat, KindNull, KindSequence, KindMapping:
		return true
	default:
		return false
	}
}

// Parsed is one validated command-line value. Text is the trimmed original
// spelling so a surgical writer can preserve the caller's quoting rather than
// re-marshalling the value through YAML.
type Parsed struct {
	Text string
	Kind Kind
	Node *yaml.Node
}

// Parse accepts exactly one YAML scalar or a flow-style sequence/mapping. Block
// collections are refused: the CLI contract uses flow syntax (`[a, b]`,
// `{a: b}`), which remains a single shell argument and a single config line.
func Parse(input string) (Parsed, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return Parsed{}, fmt.Errorf("config value is empty")
	}

	dec := yaml.NewDecoder(strings.NewReader(text))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return Parsed{}, fmt.Errorf("parsing YAML value: %w", err)
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return Parsed{}, fmt.Errorf("config value must contain exactly one YAML document")
		}
		return Parsed{}, fmt.Errorf("parsing trailing YAML value: %w", err)
	}
	if len(doc.Content) != 1 {
		return Parsed{}, fmt.Errorf("config value must contain exactly one YAML value")
	}
	node := doc.Content[0]
	kind, err := nodeKind(node)
	if err != nil {
		return Parsed{}, err
	}
	if (node.Kind == yaml.SequenceNode || node.Kind == yaml.MappingNode) && node.Style&yaml.FlowStyle == 0 {
		return Parsed{}, fmt.Errorf("%s value must use YAML flow syntax", kind)
	}
	return Parsed{Text: text, Kind: kind, Node: node}, nil
}

func nodeKind(node *yaml.Node) (Kind, error) {
	switch node.Kind {
	case yaml.SequenceNode:
		return KindSequence, nil
	case yaml.MappingNode:
		return KindMapping, nil
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str":
			return KindString, nil
		case "!!bool":
			return KindBool, nil
		case "!!int":
			return KindInt, nil
		case "!!float":
			return KindFloat, nil
		case "!!null":
			return KindNull, nil
		default:
			return "", fmt.Errorf("unsupported YAML scalar tag %q", node.Tag)
		}
	case yaml.AliasNode:
		return "", fmt.Errorf("YAML aliases are not supported as config values")
	default:
		return "", fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
}
