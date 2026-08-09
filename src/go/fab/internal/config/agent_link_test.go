package config_test

// This file's only job is the blank import below: it links internal/agent into
// the config TEST BINARY, so agent's init() runs — pushing defaults.yaml's
// dispatch: block into the three DefaultDispatch* vars — before the
// nil/empty-config accessor tests (config_test.go) read them. The vars carry no
// literal of their own (260809-wll4: defaults.yaml is the single value source),
// so without this import those tests would silently pass against the Go zero
// values and stop meaning anything.
//
// It must be package config_test, NOT config: the internal test files are
// package config, and an in-package import of agent would cycle — agent imports
// config. The external test package sits outside that cycle.
import _ "github.com/sahil87/fab-kit/src/go/fab/internal/agent"
