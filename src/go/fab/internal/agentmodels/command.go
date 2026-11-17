package agentmodels

import (
	"encoding/json"
	"fmt"
)

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
			filtered = append(filtered, row)
			if row.Default && defaultID == "" {
				defaultID = row.ID
			}
		}
		return filtered, defaultID, nil
	}
}
