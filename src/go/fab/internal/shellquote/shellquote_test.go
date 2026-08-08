package shellquote

import "testing"

func TestSingle(t *testing.T) {
	tests := map[string]string{
		"":                 "''",
		"plain":            "'plain'",
		"co'dex; $(touch)": "'co'\\''dex; $(touch)'",
	}
	for input, want := range tests {
		if got := Single(input); got != want {
			t.Errorf("Single(%q) = %q, want %q", input, got, want)
		}
	}
}
