package internal

import "testing"

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.44.10", "0.44.10", 0},
		{"0.44.9", "0.44.10", -1},
		{"0.44.10", "0.44.9", 1},
		{"0.45.0", "0.44.10", 1},
		{"0.44.0", "0.45.0", -1},
		{"1.0.0", "0.99.99", 1},
		{"v0.44.10", "0.44.10", 0},
		// Multi-digit components compare numerically, not lexicographically.
		{"10.0.0", "4.0.0", 1},
		{"4.0.0", "10.0.0", -1},
	}
	for _, tt := range tests {
		got := compareSemver(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareSemver(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"0.44.10", [3]int{0, 44, 10}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"0.0.0", [3]int{0, 0, 0}},
	}
	for _, tt := range tests {
		got := parseSemver(tt.input)
		if got != tt.want {
			t.Errorf("parseSemver(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
