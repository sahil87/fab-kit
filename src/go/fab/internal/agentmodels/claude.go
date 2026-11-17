package agentmodels

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	claudeAliases = []string{"opus", "sonnet", "haiku", "fable"}
	claudeEfforts = []string{"low", "medium", "high", "xhigh", "max"}
)

type claudeSettings struct {
	Model       *string            `json:"model"`
	ModelPicker *claudeModelPicker `json:"modelPicker"`
}

type claudeModelPicker struct {
	ReplaceBuiltInOptions bool `json:"replaceBuiltInOptions"`
	Options               []struct {
		Model string `json:"model"`
		Label string `json:"label"`
	} `json:"options"`
}

// discoverClaude combines built-in aliases with the user's configured model-picker options.
func discoverClaude(env environment) ([]Row, string, error) {
	directory, err := configPath(env, "CLAUDE_CONFIG_DIR", ".claude", "")
	if err != nil {
		return nil, "", fmt.Errorf("resolve CLAUDE_CONFIG_DIR: %w", err)
	}
	base, baseErr := readClaudeSettings(filepath.Join(directory, "settings.json"))
	local, localErr := readClaudeSettings(filepath.Join(directory, "settings.local.json"))
	if local.Model != nil {
		base.Model = local.Model
	}
	if local.ModelPicker != nil {
		base.ModelPicker = local.ModelPicker
	}

	rows := make([]Row, 0, len(claudeAliases))
	if base.ModelPicker == nil || !base.ModelPicker.ReplaceBuiltInOptions {
		for _, alias := range claudeAliases {
			rows = append(rows, claudeRow(alias, alias, SourceAlias))
		}
	}
	if base.ModelPicker != nil {
		for _, option := range base.ModelPicker.Options {
			rows = append(rows, claudeRow(option.Model, option.Label, SourceTool))
		}
	}
	defaultID := ""
	if base.Model != nil {
		defaultID = *base.Model
	}
	return rows, defaultID, joinDiscoveryErrors(baseErr, localErr)
}

func joinDiscoveryErrors(errs ...error) error {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			messages = append(messages, strings.ReplaceAll(err.Error(), "\n", "; "))
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return errors.New(strings.Join(messages, "; "))
}

func claudeRow(id, label, source string) Row {
	return Row{ID: id, Label: label, Efforts: append([]string(nil), claudeEfforts...), Source: source}
}

func readClaudeSettings(path string) (claudeSettings, error) {
	var settings claudeSettings
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return settings, fmt.Errorf("read claude %s: %w", filepath.Base(path), err)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return claudeSettings{}, fmt.Errorf("parse claude %s: %w", filepath.Base(path), err)
	}
	return settings, nil
}
