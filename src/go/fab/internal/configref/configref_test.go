package configref

import (
	"slices"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// TestFieldKeysMatchConfigscopeDottedKeys is the cycle-free registry parity
// lint. internal/config cannot import this package (configref → agent → config),
// so its generic environment walk consumes configscope.DottedKeys instead. This
// guard makes the leaf enumeration equivalent to the canonical registry rather
// than allowing a copied list to drift.
func TestFieldKeysMatchConfigscopeDottedKeys(t *testing.T) {
	want, err := FieldKeys()
	if err != nil {
		t.Fatalf("FieldKeys: %v", err)
	}
	got := configscope.DottedKeys()
	if !slices.Equal(got, want) {
		t.Fatalf("configscope dotted keys do not match configref registry:\n configscope: %v\n configref:   %v", got, want)
	}
}
