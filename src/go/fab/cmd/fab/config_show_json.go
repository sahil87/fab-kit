package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
)

// The JSON forms of `fab config show` are a pure format switch over the text
// forms: the same provenance walk produces the rows (originLine), and these
// renderers encode them. Every document is one JSON value plus a newline, indented
// like `fab config explain --json`.

// derivedFrom names the config key that determined a derived default-tier role
// value, and that key's own winning tier. It rides only on JSON rows.
type derivedFrom struct {
	Key          string `json:"key"`
	Origin       string `json:"origin"`
	OriginDetail string `json:"origin_detail"`
}

// originRowJSON is the stable row schema of `show --origin --json` (both forms).
type originRowJSON struct {
	Key          string       `json:"key"`
	Value        any          `json:"value"`
	Origin       string       `json:"origin"`
	OriginDetail string       `json:"origin_detail"`
	Status       string       `json:"status"`
	DerivedFrom  *derivedFrom `json:"derived_from,omitempty"`
}

// encodeConfigJSON marshals v as one indented document with a trailing newline, matching
// configref's reference JSON (no HTML escaping).
func encodeConfigJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("encoding config JSON: %w", err)
	}
	return buf.String(), nil
}

// jsonSafe rewrites decoded-YAML maps (map[any]any) into map[string]any all the
// way down so encoding/json can marshal them. Scalars and lists pass through.
func jsonSafe(v any) any {
	if m, ok := asGenericMap(v); ok {
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[k] = jsonSafe(val)
		}
		return out
	}
	if list, ok := v.([]any); ok {
		out := make([]any, len(list))
		for i, item := range list {
			out[i] = jsonSafe(item)
		}
		return out
	}
	return v
}

// renderOriginJSON encodes provenance rows as the JSON array. An undefined-key
// sentinel row (kind "") is not a row: it encodes as `[]`.
func renderOriginJSON(lines []originLine) (string, error) {
	rows := make([]originRowJSON, 0, len(lines))
	for _, line := range lines {
		if line.kind == "" {
			continue
		}
		rows = append(rows, originRowJSON{
			Key:          line.key,
			Value:        jsonSafe(line.value),
			Origin:       line.kind,
			OriginDetail: line.detail,
			Status:       line.status,
			DerivedFrom:  line.derived,
		})
	}
	return encodeConfigJSON(rows)
}

// roleLeafPrefix is the key prefix of the derived per-role rows.
const roleLeafPrefix = "agent.profiles."

// annotateDerived sets derived_from on every default-tier `agent.profiles.<role>.
// provider|model|effort` row whose value was derived from user configuration (see
// configref.RoleSources). The source key's winning tier is resolved over the
// ALREADY-LOADED layers; the annotation is omitted when the source is built-in,
// when no tier defines the source key, or when its winner is the default tier too.
func annotateDerived(lines []originLine, layers *config.Layers, defaults map[string]any) {
	var sources map[string]configref.RoleSources
	for i := range lines {
		line := &lines[i]
		role, leaf, ok := roleLeaf(line.key)
		if !ok || line.kind != tierDefault {
			continue
		}
		if sources == nil {
			cfg, err := config.FromMap(layers.Effective)
			if err != nil {
				return // knob-blind projection: nothing derived to attribute
			}
			if sources, err = configref.DerivedRoleSources(cfg); err != nil {
				return
			}
		}
		src, known := sources[role]
		if !known {
			continue
		}
		line.derived = resolveDerivedFrom(sourceKeyFor(src, leaf), layers, defaults)
	}
}

// roleLeaf splits `agent.profiles.<role>.<leaf>` into its role and leaf.
func roleLeaf(key string) (role, leaf string, ok bool) {
	rest, found := strings.CutPrefix(key, roleLeafPrefix)
	if !found {
		return "", "", false
	}
	role, leaf, found = strings.Cut(rest, ".")
	if !found {
		return "", "", false
	}
	switch leaf {
	case "provider", "model", "effort":
		return role, leaf, true
	}
	return "", "", false
}

func sourceKeyFor(src configref.RoleSources, leaf string) string {
	switch leaf {
	case "provider":
		return src.Provider
	case "model":
		return src.Model
	default:
		return src.Effort
	}
}

// resolveDerivedFrom resolves the winning tier of sourceKey over the loaded
// layers. It returns nil when there is nothing user-driven to report.
func resolveDerivedFrom(sourceKey string, layers *config.Layers, defaults map[string]any) *derivedFrom {
	if sourceKey == "" {
		return nil
	}
	descent := descendPath(cascadeTiers(defaults, layers, topLevelKey(sourceKey)), sourceKey)
	var winner cascadeTier
	switch {
	case !descent.found:
		return nil
	case descent.ancestor != nil:
		winner = *descent.ancestor
	default:
		defining := definingTiers(descent.tiers)
		if len(defining) == 0 {
			return nil
		}
		winner = defining[0]
	}
	if winner.kind == tierDefault {
		return nil
	}
	return &derivedFrom{Key: sourceKey, Origin: winner.kind, OriginDetail: winner.detail(sourceKey, layers.EnvOrigins)}
}
