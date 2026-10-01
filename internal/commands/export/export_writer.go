package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func writeData(data []map[string]any, path, format string) error {
	var content []byte
	var err error

	switch format {
	case "json":
		content, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	case "yaml":
		content, err = yaml.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
	case "csv":
		if len(data) == 0 {
			return os.WriteFile(path, []byte(""), 0644)
		}

		// Flatten nested dicts for CSV
		var flatData []map[string]string
		var fields []string

		// Collect all fields
		fieldSet := make(map[string]bool)
		for _, item := range data {
			for k := range item {
				fieldSet[k] = true
			}
		}
		for f := range fieldSet {
			fields = append(fields, f)
		}

		// Flatten data
		for _, item := range data {
			flatItem := make(map[string]string)
			for k, v := range item {
				switch val := v.(type) {
				case []any, map[string]any:
					jsonBytes, _ := json.Marshal(val)
					flatItem[k] = string(jsonBytes)
				default:
					flatItem[k] = fmt.Sprintf("%v", val)
				}
			}
			flatData = append(flatData, flatItem)
		}

		// Write CSV
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Header
		if err := writer.Write(fields); err != nil {
			return err
		}

		// Rows
		for _, item := range flatData {
			var row []string
			for _, f := range fields {
				row = append(row, item[f])
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}

		return nil
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	return os.WriteFile(path, content, 0644)
}
