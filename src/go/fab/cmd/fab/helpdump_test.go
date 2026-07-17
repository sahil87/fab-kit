package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newSyntheticTree builds a small cobra tree exercising every filter case:
// a visible leaf, a hidden command, and fake "completion"/"help" commands —
// plus a child with raw HTML-ish chars in its short text.
func newSyntheticTree() *cobra.Command {
	root := &cobra.Command{Use: "fab", Short: "root"}

	visible := &cobra.Command{Use: "visible", Short: "a visible <leaf> & more"}
	hidden := &cobra.Command{Use: "secret", Short: "hidden", Hidden: true}
	completion := &cobra.Command{Use: "completion", Short: "auto-gen completion"}
	help := &cobra.Command{Use: "help", Short: "auto-gen help"}

	// Add out of alpha order to prove the walker sorts.
	root.AddCommand(visible, hidden, completion, help)
	return root
}

func TestDumpDoc_TopLevelContract(t *testing.T) {
	doc := dumpDoc(newSyntheticTree(), "9.9.9")

	if doc.Tool != "fab" {
		t.Errorf("tool = %q, want %q", doc.Tool, "fab")
	}
	if doc.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", doc.SchemaVersion)
	}
	if doc.Version != "9.9.9" {
		t.Errorf("version = %q, want %q (must reflect passed-in value, not be hardcoded)", doc.Version, "9.9.9")
	}
}

// TestDumpDoc_NoCapturedAt pins the help-dump standard's forbidden field: the
// emitted envelope MUST NOT carry captured_at (the capture timestamp is owned by
// shll.ai's puller, which stamps it after capture — a tool cannot know its own
// capture time). Asserting on the encoded bytes, not just the struct, so a future
// re-introduction of the field (as a struct field or a raw map key) is caught.
func TestDumpDoc_NoCapturedAt(t *testing.T) {
	out := encodeDoc(t, dumpDoc(newSyntheticTree(), "9.9.9"))
	if strings.Contains(out, "captured_at") {
		t.Errorf("help-dump envelope must NOT contain captured_at (owned by shll.ai), got:\n%s", out)
	}
}

// encodeDoc serializes a HelpDoc with the exact encoder settings helpDumpCmd
// uses (2-space indent, HTML escaping off), so key-order assertions reflect the
// real on-the-wire bytes rather than a default Marshal.
func encodeDoc(t *testing.T, doc HelpDoc) string {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.String()
}

// assertKeyOrder fails if the given JSON keys do not appear in the encoded
// output in the listed order. Guards the frozen contract's key ordering against
// accidental struct-field reordering.
func assertKeyOrder(t *testing.T, out string, keys ...string) {
	t.Helper()
	prev := -1
	for _, k := range keys {
		needle := "\"" + k + "\":"
		idx := strings.Index(out[prev+1:], needle)
		if idx < 0 {
			t.Fatalf("key %q not found after position %d in:\n%s", k, prev, out)
		}
		idx += prev + 1
		if idx <= prev {
			t.Errorf("key %q out of order (at %d, previous key ended at %d)", k, idx, prev)
		}
		prev = idx
	}
}

// TestDumpDoc_JSONKeyOrder pins the encoded top-level and node key order to the
// frozen shll.ai contract. The struct field order is what produces this order;
// reordering HelpDoc/Node fields would break the contract and is caught here.
func TestDumpDoc_JSONKeyOrder(t *testing.T) {
	out := encodeDoc(t, dumpDoc(newSyntheticTree(), "9.9.9"))

	// Top-level: tool, version, schema_version, root (no captured_at — the
	// help-dump standard forbids it; the puller stamps the capture timestamp).
	assertKeyOrder(t, out, "tool", "version", "schema_version", "root")

	// Node: name, path, short, usage, text, commands. The synthetic root has a
	// surviving child, so a nested node is present and its order is exercised too.
	assertKeyOrder(t, out, "name", "path", "short", "usage", "text", "commands")
}

func TestBuildNode_FiltersAndSort(t *testing.T) {
	node := buildNode(newSyntheticTree())

	if node.Name != "fab" {
		t.Errorf("root name = %q, want %q", node.Name, "fab")
	}
	if len(node.Commands) != 1 {
		t.Fatalf("expected 1 surviving child (completion/help/hidden filtered), got %d: %+v", len(node.Commands), node.Commands)
	}
	if node.Commands[0].Name != "visible" {
		t.Errorf("surviving child = %q, want %q", node.Commands[0].Name, "visible")
	}

	// None of the filtered names should appear.
	for _, c := range node.Commands {
		if c.Name == "completion" || c.Name == "help" || c.Name == "secret" {
			t.Errorf("filtered command %q leaked into output", c.Name)
		}
	}
}

func TestBuildNode_SortsChildren(t *testing.T) {
	root := &cobra.Command{Use: "fab"}
	root.AddCommand(
		&cobra.Command{Use: "zeta"},
		&cobra.Command{Use: "alpha"},
		&cobra.Command{Use: "mike"},
	)
	node := buildNode(root)
	got := []string{node.Commands[0].Name, node.Commands[1].Name, node.Commands[2].Name}
	want := []string{"alpha", "mike", "zeta"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("children not sorted: got %v, want %v", got, want)
			break
		}
	}
}

func TestBuildNode_CapturesFields(t *testing.T) {
	root := &cobra.Command{Use: "fab"}
	child := &cobra.Command{
		Use:   "build [target]",
		Short: "build a target",
	}
	root.AddCommand(child)

	node := buildNode(root)
	if len(node.Commands) != 1 {
		t.Fatalf("expected 1 child, got %d", len(node.Commands))
	}
	c := node.Commands[0]
	if c.Path != "fab build" {
		t.Errorf("path = %q, want %q", c.Path, "fab build")
	}
	if c.Short != "build a target" {
		t.Errorf("short = %q, want %q", c.Short, "build a target")
	}
	if !strings.Contains(c.Usage, "build [target]") {
		t.Errorf("usage = %q, want it to contain %q", c.Usage, "build [target]")
	}
	if c.Text == "" {
		t.Errorf("text (UsageString) is empty, want the raw -h body")
	}
}

func TestBuildNode_LeafEmitsEmptyArrayNotNull(t *testing.T) {
	leaf := buildNode(&cobra.Command{Use: "leaf"})
	if leaf.Commands == nil {
		t.Fatalf("leaf.Commands is nil; want non-nil empty slice")
	}
	b, err := json.Marshal(leaf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"commands":[]`) {
		t.Errorf("leaf must serialize commands as [], got: %s", b)
	}
	if strings.Contains(string(b), `"commands":null`) {
		t.Errorf("leaf must NOT serialize commands as null, got: %s", b)
	}
}

func TestDumpDoc_EncoderDoesNotEscapeHTML(t *testing.T) {
	doc := dumpDoc(newSyntheticTree(), "1.0.0")

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()

	// The visible child's short contains "<leaf> & more" — with SetEscapeHTML(false)
	// it must be byte-preserved (literal angle brackets / ampersand).
	if !strings.Contains(out, "<leaf> & more") {
		t.Errorf("HTML chars should be preserved verbatim, got: %s", out)
	}

	// Contrast: the same doc encoded with HTML escaping ON must differ and must
	// NOT contain the literal substring (the encoder would escape <, >, &). This
	// proves SetEscapeHTML(false) is the load-bearing setting.
	var escapedBuf bytes.Buffer
	escEnc := json.NewEncoder(&escapedBuf)
	escEnc.SetIndent("", "  ")
	escEnc.SetEscapeHTML(true)
	if err := escEnc.Encode(doc); err != nil {
		t.Fatalf("encode (escaped): %v", err)
	}
	escaped := escapedBuf.String()
	if escaped == out {
		t.Errorf("escaped and non-escaped output are identical; SetEscapeHTML had no effect")
	}
	if strings.Contains(escaped, "<leaf> & more") {
		t.Errorf("with escaping ON the literal substring should be escaped away, but it was present")
	}

	// 2-space indent sanity check.
	if !strings.Contains(out, "\n  \"tool\": \"fab\"") {
		t.Errorf("expected 2-space indented top-level keys, got: %s", out)
	}
}

func TestHelpDumpCmd_IsHiddenNoArgs(t *testing.T) {
	cmd := helpDumpCmd()
	if !cmd.Hidden {
		t.Errorf("help-dump command must be Hidden")
	}
	if cmd.Use != "help-dump" {
		t.Errorf("Use = %q, want %q", cmd.Use, "help-dump")
	}
	if err := cmd.Args(cmd, []string{"extra"}); err == nil {
		t.Errorf("help-dump must reject positional args (cobra.NoArgs)")
	}
}

// TestHelpDumpCmd_MinimalConformance is the standard's minimal contract-pinning
// test, run end-to-end against the assembled root command (not just dumpDoc): the
// command exits 0, writes valid JSON to stdout and nothing to stderr, and the
// decoded envelope carries the expected tool and schema_version with no
// captured_at. This is the surface the shll.ai puller invokes.
func TestHelpDumpCmd_MinimalConformance(t *testing.T) {
	root := newRootCmd()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"help-dump"})

	if err := root.Execute(); err != nil {
		t.Fatalf("help-dump returned a non-nil error (want exit 0): %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("help-dump wrote to stderr (want empty): %q", stderr.String())
	}

	var doc HelpDoc
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("help-dump stdout is not valid JSON: %v\n%s", err, stdout.String())
	}
	if doc.Tool != "fab" {
		t.Errorf("tool = %q, want %q", doc.Tool, "fab")
	}
	if doc.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", doc.SchemaVersion)
	}
	if strings.Contains(stdout.String(), "captured_at") {
		t.Errorf("help-dump envelope must NOT contain captured_at, got:\n%s", stdout.String())
	}
}
