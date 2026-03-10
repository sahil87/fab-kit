package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/fab-go/internal/resolve"
	"gopkg.in/yaml.v3"
)

func runtimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtime",
		Short: "Manage runtime state (.fab-runtime.yaml)",
	}

	cmd.AddCommand(
		runtimeSetIdleCmd(),
		runtimeClearIdleCmd(),
		runtimeIsIdleCmd(),
	)

	return cmd
}

// runtimeFilePath returns the absolute path to .fab-runtime.yaml at the repo root.
func runtimeFilePath(fabRoot string) string {
	repoRoot := filepath.Dir(fabRoot)
	return filepath.Join(repoRoot, ".fab-runtime.yaml")
}

// loadRuntimeFile reads and parses .fab-runtime.yaml, returning an empty map if the file doesn't exist.
func loadRuntimeFile(path string) (map[string]interface{}, error) {
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

// saveRuntimeFile marshals the map and writes it atomically via temp+rename.
func saveRuntimeFile(path string, m map[string]interface{}) error {
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

func runtimeSetIdleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-idle <change>",
		Short: "Record agent idle timestamp for a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			folder, err := resolve.ToFolder(fabRoot, args[0])
			if err != nil {
				return err
			}

			rtPath := runtimeFilePath(fabRoot)
			m, err := loadRuntimeFile(rtPath)
			if err != nil {
				return err
			}

			// Ensure the folder entry exists as a map
			folderEntry, ok := m[folder].(map[string]interface{})
			if !ok {
				folderEntry = make(map[string]interface{})
			}

			// Set or update agent.idle_since without discarding other agent fields
			agentEntry, ok := folderEntry["agent"].(map[string]interface{})
			if !ok || agentEntry == nil {
				agentEntry = make(map[string]interface{})
			}
			agentEntry["idle_since"] = time.Now().Unix()
			folderEntry["agent"] = agentEntry
			m[folder] = folderEntry

			return saveRuntimeFile(rtPath, m)
		},
	}
}

func runtimeClearIdleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear-idle <change>",
		Short: "Clear agent idle state for a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			folder, err := resolve.ToFolder(fabRoot, args[0])
			if err != nil {
				return err
			}

			rtPath := runtimeFilePath(fabRoot)
			if _, err := os.Stat(rtPath); os.IsNotExist(err) {
				return nil // no-op if file doesn't exist
			}

			m, err := loadRuntimeFile(rtPath)
			if err != nil {
				return err
			}

			// Delete the agent block for this folder
			if folderEntry, ok := m[folder].(map[string]interface{}); ok {
				delete(folderEntry, "agent")
				if len(folderEntry) == 0 {
					delete(m, folder)
				} else {
					m[folder] = folderEntry
				}
			}

			return saveRuntimeFile(rtPath, m)
		},
	}
}

func runtimeIsIdleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "is-idle <change>",
		Short: "Check if an agent is idle for a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			folder, err := resolve.ToFolder(fabRoot, args[0])
			if err != nil {
				return err
			}

			rtPath := runtimeFilePath(fabRoot)
			if _, err := os.Stat(rtPath); os.IsNotExist(err) {
				fmt.Fprintln(cmd.OutOrStdout(), "unknown")
				return nil
			}

			m, err := loadRuntimeFile(rtPath)
			if err != nil {
				return err
			}

			folderEntry, ok := m[folder].(map[string]interface{})
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "active")
				return nil
			}

			agentBlock, ok := folderEntry["agent"].(map[string]interface{})
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "active")
				return nil
			}

			idleSince, ok := agentBlock["idle_since"]
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "active")
				return nil
			}

			// Parse idle_since as Unix timestamp
			var ts int64
			switch v := idleSince.(type) {
			case int:
				ts = int64(v)
			case int64:
				ts = v
			case float64:
				ts = int64(v)
			default:
				fmt.Fprintln(cmd.OutOrStdout(), "active")
				return nil
			}

			elapsed := time.Now().Unix() - ts
			if elapsed < 0 {
				elapsed = 0
			}

			fmt.Fprintf(cmd.OutOrStdout(), "idle %s\n", formatIdleDuration(elapsed))
			return nil
		},
	}
}
