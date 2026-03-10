package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// FilePath returns the absolute path to .fab-runtime.yaml at the repo root.
func FilePath(fabRoot string) string {
	repoRoot := filepath.Dir(fabRoot)
	return filepath.Join(repoRoot, ".fab-runtime.yaml")
}

// LoadFile reads and parses .fab-runtime.yaml, returning an empty map if the file doesn't exist.
func LoadFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if m == nil {
		m = make(map[string]interface{})
	}
	return m, nil
}

// SaveFile marshals the map and writes it atomically via temp+rename.
func SaveFile(path string, m map[string]interface{}) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".fab-runtime-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	// Ensure temporary file is cleaned up on error.
	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	success = true
	return nil
}

// SetIdle writes an agent.idle_since Unix timestamp for the given change folder.
func SetIdle(fabRoot, folder string) error {
	rtPath := FilePath(fabRoot)
	m, err := LoadFile(rtPath)
	if err != nil {
		return err
	}

	folderEntry, ok := m[folder].(map[string]interface{})
	if !ok {
		folderEntry = make(map[string]interface{})
	}

	agentEntry, ok := folderEntry["agent"].(map[string]interface{})
	if !ok || agentEntry == nil {
		agentEntry = make(map[string]interface{})
	}
	agentEntry["idle_since"] = time.Now().Unix()
	folderEntry["agent"] = agentEntry
	m[folder] = folderEntry

	return SaveFile(rtPath, m)
}

// ClearIdle deletes the agent block for a change folder.
// Returns nil if the runtime file doesn't exist.
func ClearIdle(fabRoot, folder string) error {
	rtPath := FilePath(fabRoot)
	if _, err := os.Stat(rtPath); os.IsNotExist(err) {
		return nil // no-op if file doesn't exist
	}

	m, err := LoadFile(rtPath)
	if err != nil {
		return err
	}

	if folderEntry, ok := m[folder].(map[string]interface{}); ok {
		delete(folderEntry, "agent")
		if len(folderEntry) == 0 {
			delete(m, folder)
		} else {
			m[folder] = folderEntry
		}
	}

	return SaveFile(rtPath, m)
}
