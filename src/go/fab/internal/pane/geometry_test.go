package pane

import (
	"testing"
)

// TestParseGeometry pins the probe's row grammar: exactly four space-separated
// integers (window then pane dimensions), everything else an error. A partial
// or garbled reading must not price the geometry floor on fabricated numbers —
// the caller fails open on ANY parse error, so parseGeometry stays strict.
func TestParseGeometry(t *testing.T) {
	tests := []struct {
		name    string
		out     string
		want    Geometry
		wantErr bool
	}{
		{"well-formed row", "127 16 127 16\n", Geometry{WindowWidth: 127, WindowHeight: 16, PaneWidth: 127, PaneHeight: 16}, false},
		{"carve target: pane smaller than window", "240 60 156 60\n", Geometry{WindowWidth: 240, WindowHeight: 60, PaneWidth: 156, PaneHeight: 60}, false},
		{"extra whitespace tolerated", "  200   50\t100  25\n", Geometry{WindowWidth: 200, WindowHeight: 50, PaneWidth: 100, PaneHeight: 25}, false},
		{"empty output errors", "", Geometry{}, true},
		{"short row errors", "127 16 127\n", Geometry{}, true},
		{"extra fields error", "127 16 127 16 1\n", Geometry{}, true},
		{"non-numeric field errors", "127 abc 127 16\n", Geometry{}, true},
		{"format tokens unresolved error", "#{window_width} 16 127 16\n", Geometry{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGeometry(tt.out)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseGeometry(%q) = %+v, nil; want error", tt.out, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGeometry(%q): %v", tt.out, err)
			}
			if got != tt.want {
				t.Errorf("parseGeometry(%q) = %+v, want %+v", tt.out, got, tt.want)
			}
		})
	}
}
