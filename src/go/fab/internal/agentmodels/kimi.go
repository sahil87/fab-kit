package agentmodels

import (
	"fmt"
	"os"
)

func discoverKimi(env environment) ([]Row, string, error) {
	path, err := configPath(env, "", ".kimi-code", "config.toml")
	if err != nil {
		return nil, "", fmt.Errorf("resolve kimi config: %w", err)
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("read kimi config: %w", err)
	}
	doc, err := parseTOML(data)
	if err != nil {
		return nil, "", fmt.Errorf("parse kimi config: %w", err)
	}
	rows := make([]Row, 0, len(doc.tables))
	for _, table := range doc.tables {
		if table.array {
			continue
		}
		id, ok := kimiModelName(table.header)
		if !ok {
			continue
		}
		rows = append(rows, Row{
			ID:      id,
			Label:   table.strings["display_name"],
			Efforts: append([]string{}, table.arrays["support_efforts"]...),
			Source:  SourceTool,
		})
	}
	return rows, doc.top["default_model"], nil
}
