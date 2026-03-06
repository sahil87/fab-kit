package parity

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRuntimeSetIdle(t *testing.T) {
	t.Run("creates file and writes idle_since", func(t *testing.T) {
		tmp := setupTempRepo(t)

		// Ensure .fab-runtime.yaml does not exist yet
		rtPath := filepath.Join(tmp, ".fab-runtime.yaml")
		os.Remove(rtPath)

		res := runGo(t, tmp, "runtime", "set-idle", changeID)
		if res.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", res.ExitCode, res.Stderr)
		}

		// Verify file was created
		data, err := os.ReadFile(rtPath)
		if err != nil {
			t.Fatalf("expected .fab-runtime.yaml to exist: %v", err)
		}

		var m map[string]interface{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("failed to parse runtime file: %v", err)
		}

		folderEntry, ok := m[changeName].(map[string]interface{})
		if !ok {
			t.Fatalf("expected entry for %q, got: %v", changeName, m)
		}

		agentBlock, ok := folderEntry["agent"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected agent block, got: %v", folderEntry)
		}

		if _, ok := agentBlock["idle_since"]; !ok {
			t.Fatalf("expected idle_since in agent block, got: %v", agentBlock)
		}
	})

	t.Run("updates existing file preserving other entries", func(t *testing.T) {
		tmp := setupTempRepo(t)
		rtPath := filepath.Join(tmp, ".fab-runtime.yaml")

		// Pre-populate with another change entry
		initial := map[string]interface{}{
			"other-change": map[string]interface{}{
				"agent": map[string]interface{}{
					"idle_since": 1234567890,
				},
			},
		}
		data, err := yaml.Marshal(initial)
		if err != nil {
			t.Fatalf("failed to marshal initial runtime content: %v", err)
		}
		if err := os.WriteFile(rtPath, data, 0o644); err != nil {
			t.Fatalf("failed to write initial runtime file: %v", err)
		}

		res := runGo(t, tmp, "runtime", "set-idle", changeID)
		if res.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", res.ExitCode, res.Stderr)
		}

		data, err = os.ReadFile(rtPath)
		if err != nil {
			t.Fatalf("reading runtime file: %v", err)
		}

		var m map[string]interface{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("parsing runtime file: %v", err)
		}

		// Both entries should exist
		if _, ok := m["other-change"]; !ok {
			t.Error("expected other-change entry to be preserved")
		}
		if _, ok := m[changeName]; !ok {
			t.Errorf("expected entry for %q", changeName)
		}
	})
}

func TestRuntimeClearIdle(t *testing.T) {
	t.Run("removes agent block from existing file", func(t *testing.T) {
		tmp := setupTempRepo(t)
		rtPath := filepath.Join(tmp, ".fab-runtime.yaml")

		// Pre-populate with the change's agent entry
		initial := map[string]interface{}{
			changeName: map[string]interface{}{
				"agent": map[string]interface{}{
					"idle_since": 1234567890,
				},
			},
			"other-change": map[string]interface{}{
				"agent": map[string]interface{}{
					"idle_since": 9876543210,
				},
			},
		}
		data, err := yaml.Marshal(initial)
		if err != nil {
			t.Fatalf("failed to marshal initial runtime content: %v", err)
		}
		if err := os.WriteFile(rtPath, data, 0o644); err != nil {
			t.Fatalf("failed to write initial runtime file: %v", err)
		}

		res := runGo(t, tmp, "runtime", "clear-idle", changeID)
		if res.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", res.ExitCode, res.Stderr)
		}

		data, err = os.ReadFile(rtPath)
		if err != nil {
			t.Fatalf("reading runtime file: %v", err)
		}

		var m map[string]interface{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("parsing runtime file: %v", err)
		}

		// The test change entry should be gone (empty after agent removed)
		if _, ok := m[changeName]; ok {
			t.Error("expected test change entry to be removed")
		}

		// Other entry should be preserved
		if _, ok := m["other-change"]; !ok {
			t.Error("expected other-change entry to be preserved")
		}
	})

	t.Run("no-op when file does not exist", func(t *testing.T) {
		tmp := setupTempRepo(t)
		rtPath := filepath.Join(tmp, ".fab-runtime.yaml")
		os.Remove(rtPath)

		res := runGo(t, tmp, "runtime", "clear-idle", changeID)
		if res.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", res.ExitCode, res.Stderr)
		}

		// File should still not exist
		if _, err := os.Stat(rtPath); !os.IsNotExist(err) {
			t.Error("expected .fab-runtime.yaml to not exist after no-op clear-idle")
		}
	})
}

func TestRuntimeChangeResolution(t *testing.T) {
	t.Run("resolves 4-char ID for set-idle", func(t *testing.T) {
		tmp := setupTempRepo(t)
		rtPath := filepath.Join(tmp, ".fab-runtime.yaml")
		os.Remove(rtPath)

		res := runGo(t, tmp, "runtime", "set-idle", changeID)
		if res.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", res.ExitCode, res.Stderr)
		}

		data, err := os.ReadFile(rtPath)
		if err != nil {
			t.Fatalf("failed to read runtime file: %v", err)
		}
		var m map[string]interface{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("failed to parse runtime file: %v", err)
		}

		// Should be keyed by full folder name, not the 4-char ID
		if _, ok := m[changeName]; !ok {
			t.Errorf("expected entry keyed by full folder name %q, got keys: %v", changeName, m)
		}
	})

	t.Run("resolves substring for clear-idle", func(t *testing.T) {
		tmp := setupTempRepo(t)
		rtPath := filepath.Join(tmp, ".fab-runtime.yaml")

		initial := map[string]interface{}{
			changeName: map[string]interface{}{
				"agent": map[string]interface{}{
					"idle_since": 1234567890,
				},
			},
		}
		data, err := yaml.Marshal(initial)
		if err != nil {
			t.Fatalf("failed to marshal initial runtime content: %v", err)
		}
		if err := os.WriteFile(rtPath, data, 0o644); err != nil {
			t.Fatalf("failed to write initial runtime file: %v", err)
		}

		res := runGo(t, tmp, "runtime", "clear-idle", "parity-test")
		if res.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", res.ExitCode, res.Stderr)
		}

		data, err = os.ReadFile(rtPath)
		if err != nil {
			t.Fatalf("failed to read runtime file: %v", err)
		}
		var m map[string]interface{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("failed to parse runtime file: %v", err)
		}

		if _, ok := m[changeName]; ok {
			t.Error("expected test change to be cleared via substring resolution")
		}
	})
}
