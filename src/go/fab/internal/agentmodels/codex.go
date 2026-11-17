package agentmodels

import (
	"encoding/json"
	"fmt"
	"os"
)

type codexModel struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Visibility  string `json:"visibility"`
	Efforts     []struct {
		Effort string `json:"effort"`
	} `json:"supported_reasoning_levels"`
}

func discoverCodex(env environment) ([]Row, string, error) {
	defaultID, defaultErr := codexDefault(env)
	stdout, err := runBounded(env, "codex", "debug", "models")
	if err != nil {
		return nil, defaultID, fmt.Errorf("codex debug models: %w", err)
	}
	var payload struct {
		Models *[]codexModel `json:"models"`
	}
	if err := json.Unmarshal(stdout, &payload); err != nil {
		return nil, defaultID, fmt.Errorf("codex debug models: invalid JSON: %w", err)
	}
	if payload.Models == nil {
		return nil, defaultID, fmt.Errorf("codex debug models: missing models")
	}
	rows := make([]Row, 0, len(*payload.Models))
	for _, model := range *payload.Models {
		if model.Visibility != "list" || model.Slug == "" {
			continue
		}
		efforts := make([]string, 0, len(model.Efforts))
		for _, level := range model.Efforts {
			efforts = append(efforts, level.Effort)
		}
		rows = append(rows, Row{
			ID:      model.Slug,
			Label:   model.DisplayName,
			Efforts: efforts,
			Source:  SourceTool,
		})
	}
	if defaultErr != nil {
		return rows, defaultID, defaultErr
	}
	return rows, defaultID, nil
}

func codexDefault(env environment) (string, error) {
	path, err := configPath(env, "CODEX_HOME", ".codex", "config.toml")
	if err != nil {
		return "", fmt.Errorf("resolve CODEX_HOME: %w", err)
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read codex config: %w", err)
	}
	doc, err := parseTOML(data)
	if err != nil {
		return "", fmt.Errorf("parse codex config: %w", err)
	}
	return doc.top["model"], nil
}
