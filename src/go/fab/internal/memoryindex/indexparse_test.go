package memoryindex

import "testing"

func TestSplitTableRowEscapedPipes(t *testing.T) {
	for _, tc := range []struct {
		line string
		want []string
	}{
		{`| [A\|B](topic.md) | C\|D | extra |`, []string{`[A\|B](topic.md)`, `C\|D`, "extra"}},
		{`| a\\| b |`, []string{`a\\`, "b"}},
		{`| a\\\|b | |`, []string{`a\\\|b`, ""}},
	} {
		got := splitTableRow(tc.line)
		if len(got) != len(tc.want) {
			t.Fatalf("%q: %#v", tc.line, got)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%q: cell %d = %q, want %q", tc.line, i, got[i], tc.want[i])
			}
		}
	}
}
