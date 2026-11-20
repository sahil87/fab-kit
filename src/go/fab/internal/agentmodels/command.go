package agentmodels

import (
	"encoding/json"
	"fmt"
)

// commandAdapter runs a provider's models_command and normalizes its JSON rows.
func commandAdapter(command string) adapter {
	return func(env environment) ([]Row, string, error) {
		stdout, err := runBounded(env, "sh", "-c", command)
		if err != nil {
			return nil, "", fmt.Errorf("models_command: %w", err)
		}
		var rows []Row
		if err := json.Unmarshal(stdout, &rows); err != nil {
			return nil, "", fmt.Errorf("models_command: invalid JSON: %w", err)
		}
		if rows == nil {
			return nil, "", fmt.Errorf("models_command: expected a JSON array")
		}
		defaultID := ""
		filtered := rows[:0]
		for _, row := range rows {
			if row.ID == "" {
				continue
			}
			row.Source = SourceTool
			if row.Default {
				if defaultID == "" {
					defaultID = row.ID
				} else {
					row.Default = false
				}
			}
			filtered = append(filtered, row)
		}
		return filtered, defaultID, nil
	}
}
