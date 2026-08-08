package configvalue

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Kind
		wantErr bool
	}{
		{name: "plain string", input: "codex", want: KindString},
		{name: "quoted string", input: `"42"`, want: KindString},
		{name: "bool", input: "true", want: KindBool},
		{name: "int", input: "42", want: KindInt},
		{name: "float", input: "3.5", want: KindFloat},
		{name: "null", input: "null", want: KindNull},
		{name: "flow sequence", input: "[a, b]", want: KindSequence},
		{name: "flow mapping", input: "{a: b}", want: KindMapping},
		{name: "block sequence", input: "- a\n- b", wantErr: true},
		{name: "block mapping", input: "a: b", wantErr: true},
		{name: "unterminated quote", input: `"codex`, wantErr: true},
		{name: "empty", input: "  ", wantErr: true},
		{name: "two documents", input: "a\n---\nb", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) succeeded: %#v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.input, err)
			}
			if got.Kind != tt.want {
				t.Fatalf("Parse(%q).Kind = %q, want %q", tt.input, got.Kind, tt.want)
			}
			if got.Text != tt.input {
				t.Fatalf("Parse(%q).Text = %q", tt.input, got.Text)
			}
		})
	}
}
