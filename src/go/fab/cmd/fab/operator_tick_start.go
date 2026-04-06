package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/runtime"
	"gopkg.in/yaml.v3"
	"os"
)

// operatorRepoRootOverride is used in tests to redirect .fab-operator.yaml I/O
// to a temp directory instead of the real git repo root.
var operatorRepoRootOverride string

func operatorTickStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tick-start",
		Short: "Increment tick_count and record last_tick_at in .fab-operator.yaml",
		Args:  cobra.NoArgs,
		RunE:  runOperatorTickStart,
	}
}

func runOperatorTickStart(cmd *cobra.Command, args []string) error {
	var repoRoot string
	if operatorRepoRootOverride != "" {
		repoRoot = operatorRepoRootOverride
	} else {
		var err error
		repoRoot, err = gitRepoRoot()
		if err != nil {
			return fmt.Errorf("cannot determine repo root: %w", err)
		}
	}

	yamlPath := filepath.Join(repoRoot, ".fab-operator.yaml")

	// Read existing file, or start with empty map if missing
	data := make(map[string]interface{})
	raw, err := os.ReadFile(yamlPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read %s: %w", yamlPath, err)
	}
	if err == nil && len(raw) > 0 {
		if parseErr := yaml.Unmarshal(raw, &data); parseErr != nil {
			return fmt.Errorf("cannot parse %s: %w", yamlPath, parseErr)
		}
	}

	// Increment tick_count
	tickCount := 0
	if v, ok := data["tick_count"]; ok {
		switch n := v.(type) {
		case int:
			tickCount = n
		case int64:
			tickCount = int(n)
		case float64:
			tickCount = int(n)
		}
	}
	tickCount++

	// Capture time once so last_tick_at and stdout are consistent
	now := time.Now()

	data["tick_count"] = tickCount
	data["last_tick_at"] = now.UTC().Format(time.RFC3339)

	// Write back atomically via temp+rename
	if err := runtime.SaveFile(yamlPath, data); err != nil {
		return fmt.Errorf("cannot write %s: %w", yamlPath, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "tick: %d\nnow: %s\n", tickCount, now.Format("15:04"))
	return nil
}
