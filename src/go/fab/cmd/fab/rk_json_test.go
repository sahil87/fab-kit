package main

import (
	"strings"
	"testing"
)

func TestUnwrapRkJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string // expected payload when wantErr is false
		wantErr string // substring the error must carry; "" → no error
	}{
		{"bare array passes through", `[{"id":"a"}]`, `[{"id":"a"}]`, ""},
		{"bare array with leading whitespace", "\n  [1,2]\n", `[1,2]`, ""},
		{"bare object without ok passes through", `{"pane":"%1","lines":2}`, `{"pane":"%1","lines":2}`, ""},
		{"ok:true with array result", `{"ok":true,"result":[{"id":"a"}]}`, `[{"id":"a"}]`, ""},
		{"ok:true with object result", `{"ok":true,"result":{"pane":"%1"}}`, `{"pane":"%1"}`, ""},
		{"ok:true with result absent", `{"ok":true}`, "", "without a result"},
		{"ok:true with null result", `{"ok":true,"result":null}`, "", "without a result"},
		{"ok:false carries code and message", `{"ok":false,"error":{"code":"operational","message":"list sessions: exit status 1"}}`, "", "operational: list sessions: exit status 1"},
		{"malformed json", `{"ok":true,"result":[`, "", "rk json:"},
		{"malformed bare array is rejected, not passed through", `[`, "", "malformed bare array"},
		{"bare array with trailing garbage is rejected", `[] trailing`, "", "malformed bare array"},
		{"bare null is rejected — it is not an established document", `null`, "", "null document"},
		{"empty input", ``, "", "empty output"},
		{"whitespace-only input", " \n\t", "", "empty output"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := unwrapRkJSON([]byte(tc.in))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("unwrapRkJSON(%q) = %q, want error containing %q", tc.in, got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unwrapRkJSON(%q): %v", tc.in, err)
			}
			if string(got) != tc.want {
				t.Fatalf("payload = %q, want %q", got, tc.want)
			}
		})
	}
}
