package main

import (
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/impact"
)

func TestRenderYAML_NoExcluding(t *testing.T) {
	got := renderYAML(impact.Result{Added: 142, Deleted: 38, Net: 104})
	for _, want := range []string{
		"added: 142\n",
		"deleted: 38\n",
		"net: 104\n",
		"computed_at: ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderYAML missing %q\nfull output:\n%s", want, got)
		}
	}
	if strings.Contains(got, "excluding:") {
		t.Errorf("expected no `excluding:` block when Excluding is nil; got:\n%s", got)
	}
}

func TestRenderYAML_WithExcluding(t *testing.T) {
	got := renderYAML(impact.Result{
		Added:   142,
		Deleted: 38,
		Net:     104,
		Excluding: &impact.Pair{
			Added:   87,
			Deleted: 38,
			Net:     49,
		},
	})
	for _, want := range []string{
		"excluding:\n",
		"    added: 87\n",
		"    deleted: 38\n",
		"    net: 49\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderYAML missing %q\nfull output:\n%s", want, got)
		}
	}
}
