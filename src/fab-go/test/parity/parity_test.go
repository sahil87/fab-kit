// Package parity tests that the Go binary produces identical output to the bash scripts.
//
// For each operation, both implementations are run against the same fixtures
// and their stdout, stderr, exit codes, and file mutations are compared.
package parity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// repoRoot returns the absolute path to the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	// test is at src/fab-go/test/parity/ — repo root is 4 levels up
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// fabBinary returns the path to the compiled Go binary.
func fabBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(repoRoot(t), "fab", ".kit", "bin", "fab")
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("fab binary not found at %s — run 'go build -o %s ./cmd/fab' first", bin, bin)
	}
	return bin
}

// checkPrereqs skips the test if yq or jq are not installed (needed by bash scripts).
func checkPrereqs(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"yq", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed — skipping parity test (bash scripts require it)", tool)
		}
	}
}

// fixturesDir returns the path to the test fixtures.
func fixturesDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Join(filepath.Dir(file), "fixtures")
}

// copyDir recursively copies src to dst, preserving file permissions.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// setupTempRepo creates a temp directory with a copy of the fixtures,
// simulating a repo root with fab/ structure. Returns the temp dir path.
func setupTempRepo(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if err := copyDir(fixturesDir(t), tmp); err != nil {
		t.Fatalf("copying fixtures: %v", err)
	}
	// Copy the workflow schema from the real repo
	realSchema := filepath.Join(repoRoot(t), "fab", ".kit", "schemas", "workflow.yaml")
	dstSchema := filepath.Join(tmp, "fab", ".kit", "schemas", "workflow.yaml")
	if err := os.MkdirAll(filepath.Dir(dstSchema), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(realSchema)
	if err != nil {
		t.Fatalf("reading workflow schema: %v", err)
	}
	if err := os.WriteFile(dstSchema, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return tmp
}

// cmdResult captures the output of a command execution.
type cmdResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runBash executes a bash script with args, using tmpDir as the repo root.
func runBash(t *testing.T, tmpDir, script string, args ...string) cmdResult {
	t.Helper()
	realScriptsDir := filepath.Join(repoRoot(t), "fab", ".kit", "scripts", "lib")
	tmpScriptsDir := filepath.Join(tmpDir, "fab", ".kit", "scripts", "lib")
	if err := os.MkdirAll(tmpScriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Copy all lib scripts (they may cross-reference each other)
	entries, err := os.ReadDir(realScriptsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(realScriptsDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		// Strip the shim block so we always run bash, not the Go binary
		data = stripShim(data)
		if err := os.WriteFile(filepath.Join(tmpScriptsDir, e.Name()), data, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Copy the templates directory for changeman new
	realTemplatesDir := filepath.Join(repoRoot(t), "fab", ".kit", "templates")
	tmpTemplatesDir := filepath.Join(tmpDir, "fab", ".kit", "templates")
	if _, err := os.Stat(realTemplatesDir); err == nil {
		if err := os.MkdirAll(tmpTemplatesDir, 0o755); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(realTemplatesDir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(realTemplatesDir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tmpTemplatesDir, e.Name()), data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	scriptPath := filepath.Join(tmpScriptsDir, script)
	allArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command("bash", allArgs...)
	cmd.Dir = tmpDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running bash %s: %v", script, err)
		}
	}

	return cmdResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// stripShim removes the Go binary shim block from a shell script.
func stripShim(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	var result []string
	inShim := false
	for _, line := range lines {
		if strings.Contains(line, "Shim: delegate to Go binary") {
			inShim = true
			continue
		}
		if inShim {
			if line == "fi" {
				inShim = false
				continue
			}
			continue
		}
		result = append(result, line)
	}
	return []byte(strings.Join(result, "\n"))
}

// runGo executes the Go binary with args, using tmpDir as the working directory.
func runGo(t *testing.T, tmpDir string, args ...string) cmdResult {
	t.Helper()
	bin := fabBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = tmpDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running fab %v: %v", args, err)
		}
	}

	return cmdResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// assertParity compares bash and Go results, failing the test on any difference.
// For YAML-like stdout (e.g., preflight output), uses semantic comparison.
func assertParity(t *testing.T, label string, bash, goRes cmdResult) {
	t.Helper()
	if bash.ExitCode != goRes.ExitCode {
		t.Errorf("%s: exit code mismatch — bash=%d, go=%d", label, bash.ExitCode, goRes.ExitCode)
	}
	if bash.Stdout != goRes.Stdout {
		// Try semantic YAML comparison for structured output
		if !compareYAMLSemantics(bash.Stdout, goRes.Stdout) {
			t.Errorf("%s: stdout mismatch\n--- bash ---\n%s\n--- go ---\n%s", label, bash.Stdout, goRes.Stdout)
		}
	}
	bashStderr := normalizeStderr(bash.Stderr)
	goStderr := normalizeStderr(goRes.Stderr)
	if bashStderr != goStderr {
		t.Errorf("%s: stderr mismatch\n--- bash ---\n%s\n--- go ---\n%s", label, bash.Stderr, goRes.Stderr)
	}
}

// compareYAMLSemantics tries to parse both strings as YAML and compare semantically.
func compareYAMLSemantics(a, b string) bool {
	if a == b {
		return true
	}
	var aData, bData interface{}
	if err := yaml.Unmarshal([]byte(a), &aData); err != nil {
		return false
	}
	if err := yaml.Unmarshal([]byte(b), &bData); err != nil {
		return false
	}
	aData = normalizeYAMLTypes(aData)
	bData = normalizeYAMLTypes(bData)
	aNorm, _ := yaml.Marshal(aData)
	bNorm, _ := yaml.Marshal(bData)
	return string(aNorm) == string(bNorm)
}

// normalizeStderr removes temp dir paths and whitespace differences from stderr.
func normalizeStderr(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// readFile returns the contents of a file in the temp repo.
func readFile(t *testing.T, tmpDir, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(tmpDir, relPath))
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatal(err)
	}
	return string(data)
}

// normalizeYAMLTypes converts time.Time → string and strips known-optional
// false boolean fields to handle omitted-vs-false differences.
func normalizeYAMLTypes(v interface{}) interface{} {
	// Fields where Go binary omits false and bash includes it
	optionalFalseFields := map[string]bool{"indicative": true}

	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, v2 := range val {
			// Strip only known-optional false booleans
			if b, ok := v2.(bool); ok && !b && optionalFalseFields[k] {
				continue
			}
			out[k] = normalizeYAMLTypes(v2)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, v2 := range val {
			out[i] = normalizeYAMLTypes(v2)
		}
		return out
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return v
	}
}

// nonEmptyLines splits a string into non-empty lines.
func nonEmptyLines(s string) []string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	var result []string
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

// assertFileParity compares a file's contents between two temp repos.
// YAML: semantic comparison (parse, normalize types, re-serialize).
// JSONL: semantic comparison (parse each line, normalize key order).
func assertFileParity(t *testing.T, label, bashDir, goDir, relPath string) {
	t.Helper()
	bashContent := readFile(t, bashDir, relPath)
	goContent := readFile(t, goDir, relPath)

	if strings.HasSuffix(relPath, ".yaml") || strings.HasSuffix(relPath, ".yml") {
		var bashData, goData interface{}
		if err := yaml.Unmarshal([]byte(bashContent), &bashData); err != nil {
			t.Errorf("%s: failed to parse bash YAML %s: %v", label, relPath, err)
			return
		}
		if err := yaml.Unmarshal([]byte(goContent), &goData); err != nil {
			t.Errorf("%s: failed to parse go YAML %s: %v", label, relPath, err)
			return
		}
		bashData = normalizeYAMLTypes(bashData)
		goData = normalizeYAMLTypes(goData)
		bashNorm, _ := yaml.Marshal(bashData)
		goNorm, _ := yaml.Marshal(goData)
		if string(bashNorm) != string(goNorm) {
			t.Errorf("%s: file %s semantic mismatch\n--- bash (normalized) ---\n%s\n--- go (normalized) ---\n%s", label, relPath, bashNorm, goNorm)
		}
	} else if strings.HasSuffix(relPath, ".jsonl") {
		// Semantic JSONL comparison — parse each line, normalize key order
		bashLines := nonEmptyLines(bashContent)
		goLines := nonEmptyLines(goContent)
		if len(bashLines) != len(goLines) {
			t.Errorf("%s: file %s line count mismatch — bash=%d, go=%d", label, relPath, len(bashLines), len(goLines))
			return
		}
		for i := range bashLines {
			var bashObj, goObj map[string]interface{}
			if err := json.Unmarshal([]byte(bashLines[i]), &bashObj); err != nil {
				t.Errorf("%s: failed to parse bash JSONL line %d: %v", label, i, err)
				continue
			}
			if err := json.Unmarshal([]byte(goLines[i]), &goObj); err != nil {
				t.Errorf("%s: failed to parse go JSONL line %d: %v", label, i, err)
				continue
			}
			bashNorm, _ := json.Marshal(bashObj)
			goNorm, _ := json.Marshal(goObj)
			if string(bashNorm) != string(goNorm) {
				t.Errorf("%s: file %s line %d mismatch\n--- bash ---\n%s\n--- go ---\n%s", label, relPath, i, bashLines[i], goLines[i])
			}
		}
	} else if bashContent != goContent {
		t.Errorf("%s: file %s mismatch\n--- bash ---\n%s\n--- go ---\n%s", label, relPath, bashContent, goContent)
	}
}

// removeAll removes a file or directory from the temp repo.
func removeAll(dir, relPath string) error {
	return os.RemoveAll(filepath.Join(dir, relPath))
}

// writeFileHelper writes content to a file in the temp repo.
func writeFileHelper(dir, relPath, content string) error {
	path := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

const changeName = "260305-t3st-parity-test-change"
const changeID = "t3st"

var statusPath = fmt.Sprintf("fab/changes/%s/.status.yaml", changeName)
var historyPath = fmt.Sprintf("fab/changes/%s/.history.jsonl", changeName)
