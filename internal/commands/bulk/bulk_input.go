package bulk

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadInputFile(filePath string) ([]map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	switch ext {
	case ".json":
		var data []map[string]string
		// Try as array first
		if err := json.Unmarshal(content, &data); err != nil {
			// Try as single object
			var single map[string]string
			if err := json.Unmarshal(content, &single); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}
			data = []map[string]string{single}
		}
		return data, nil

	case ".yaml", ".yml":
		var data []map[string]string
		// Try as array first
		if err := yaml.Unmarshal(content, &data); err != nil {
			// Try as single object
			var single map[string]string
			if err := yaml.Unmarshal(content, &single); err != nil {
				return nil, fmt.Errorf("failed to parse YAML: %w", err)
			}
			data = []map[string]string{single}
		}
		return data, nil

	case ".csv":
		reader := csv.NewReader(strings.NewReader(string(content)))
		// Skip comments
		reader.Comment = '#'
		records, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV: %w", err)
		}
		if len(records) < 2 {
			return nil, fmt.Errorf("CSV file must have a header row and at least one data row")
		}
		// First row is header
		headers := records[0]
		var data []map[string]string
		for _, row := range records[1:] {
			record := make(map[string]string)
			for i, value := range row {
				if i < len(headers) {
					record[headers[i]] = value
				}
			}
			data = append(data, record)
		}
		return data, nil

	default:
		return nil, fmt.Errorf("unsupported file format: %s (use .json, .yaml, .yml, or .csv)", ext)
	}
}

// loadYAMLFile loads data from a YAML file with nested structure.
func loadYAMLFile(filePath string, key string) ([]map[string]any, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var wrapper map[string][]map[string]any
	if err := yaml.Unmarshal(content, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	data, ok := wrapper[key]
	if !ok {
		// Try without wrapper
		var direct []map[string]any
		if err := yaml.Unmarshal(content, &direct); err != nil {
			return nil, fmt.Errorf("YAML must have '%s' key or be a list", key)
		}
		return direct, nil
	}

	return data, nil
}
