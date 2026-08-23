// Package fab is the module root's one-file package: it exists solely to embed
// defaults.yaml, which sits here — rather than beside its consumer,
// internal/agent — so the most-referred-to data file in the module is visible at
// the top of the tree. go:embed cannot reach above the embedding package's
// directory, so hosting the file at the module root requires a package at the
// module root; this is that package. All parsing, validation, and consumption
// stay in internal/agent — nothing else belongs here.
package fab

import _ "embed"

// DefaultsYAML is the raw bytes of defaults.yaml — fab-kit's built-in agent,
// provider, dispatch, and autopilot defaults, the bottom tier of the config
// cascade. It is deliberately EMBEDDED rather than read from the kit cache at
// runtime: kit and binary release atomically, so an on-disk read would gain
// nothing and add a binary↔kit version-skew failure mode to a resolution path
// that cannot fail today. internal/agent parses it once at package init.
//
//go:embed defaults.yaml
var DefaultsYAML []byte
