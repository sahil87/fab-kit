package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// validateSetValue is the value-DOMAIN gate of `fab config set` (and the `fab setup`
// wizard's writes): it refuses a value outside the key's accepted set or range
// before anything is written. layers is the read model the caller loaded (nil when
// it could not be). It runs on top of configupgrade's KIND check, never
// instead of it — a value that does not parse as the key's kind (`column_width abc`)
// passes through here so the writer's own, better kind error is the one reported.
//
// Validation is write-time only. The read path stays fail-open and provider-neutral
// (a hand-edited bad value degrades with a warning; a malformed personal preference
// must not brick every repo), and `unset` — the repair path — never inspects a value.
func validateSetValue(key, rawValue string, layers *config.Layers) error {
	value, err := config.ParseYAMLValue(rawValue)
	if err != nil {
		return nil // the writer's parse error is the better message
	}
	switch {
	case key == "dispatch.mode":
		return validateEnum(key, value, config.ValidDispatchModes)
	case key == "autopilot.merge_mode":
		return validateEnum(key, value, config.ValidAutopilotMergeModes)
	case key == "dispatch.column_width":
		return validateIntRange(key, value, config.MinDispatchColumnWidth, config.MaxDispatchColumnWidth)
	case key == "dispatch.min_cols", key == "dispatch.min_rows":
		return validateIntRange(key, value, config.MinDispatchFloor, 0)
	case isProviderKey(key):
		name, ok := value.(string)
		if !ok {
			return nil // kind mismatch: left to the writer
		}
		return validateProviderName(key, name, layers)
	}
	return nil
}

func validateEnum(key string, value any, accepted []string) error {
	s, ok := value.(string)
	if !ok {
		return nil // kind mismatch: left to the writer
	}
	if !slices.Contains(accepted, s) {
		return fmt.Errorf("invalid value %q for %s — must be one of: %s", s, key, strings.Join(accepted, ", "))
	}
	return nil
}

// validateIntRange refuses an int outside lo..hi; hi <= 0 means no upper bound.
// yaml.v3 decodes an integer too large for int as int64/uint64 — still a YAML
// integer to the writer, but one the typed config cannot load, so it is refused.
func validateIntRange(key string, value any, lo, hi int) error {
	switch n := value.(type) {
	case int:
		if n >= lo && (hi <= 0 || n <= hi) {
			return nil
		}
	case int64, uint64:
	default:
		return nil // kind mismatch: left to the writer
	}
	if hi <= 0 {
		return fmt.Errorf("invalid value %v for %s — must be an integer of at least %d", value, key, lo)
	}
	return fmt.Errorf("invalid value %v for %s — must be an integer from %d to %d", value, key, lo, hi)
}

// isProviderKey reports whether key names a provider: the two depth knobs and any
// role's provider override.
func isProviderKey(key string) bool {
	if key == "agent.session" || key == "agent.workers" {
		return true
	}
	parts := strings.Split(key, ".")
	return len(parts) == 4 && parts[0] == "agent" && parts[1] == "profiles" && parts[3] == "provider"
}

// validateProviderName refuses a name that is not in the effective providers table:
// fab's built-in providers plus the `providers:` keys of the config visible from the
// cwd (project ⊕ system ⊕ env inside a project, system ⊕ env outside one).
func validateProviderName(key, name string, layers *config.Layers) error {
	available := agent.ProviderNames(effectiveProvidersConfig(layers))
	if slices.Contains(available, name) {
		return nil
	}
	return fmt.Errorf("invalid value %q for %s — not a provider in the effective providers table (available: %s)",
		name, key, strings.Join(available, ", "))
}

// effectiveProvidersConfig is the read model's effective config for the provider
// table, from the layers the caller loaded. When they are absent or will not decode
// (a malformed project file) it falls back to config.LoadNoProject's fail-open
// cascade — which drops a schema-incompatible system file but keeps the env
// providers — rather than failing the write for that reason alone; the built-in
// providers are always acceptable.
func effectiveProvidersConfig(layers *config.Layers) *config.Config {
	if layers != nil {
		if cfg, err := config.FromMap(layers.Effective); err == nil {
			return cfg
		}
	}
	return config.LoadNoProject()
}
