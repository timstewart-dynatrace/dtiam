package template

import (
	"embed"
	"path/filepath"
	"strings"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// ListBuiltin returns info about all built-in templates.
func ListBuiltin() []TemplateInfo {
	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		return nil
	}

	var templates []TemplateInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		content, err := builtinFS.ReadFile("builtin/" + entry.Name())
		if err != nil {
			continue
		}
		templates = append(templates, TemplateInfo{
			Name:   name,
			Source: "builtin",
			Path:   "builtin/" + entry.Name(),
			Vars:   ExtractVariables(string(content)),
		})
	}

	return templates
}

// GetBuiltin returns the content of a built-in template.
func GetBuiltin(name string) ([]byte, error) {
	return builtinFS.ReadFile("builtin/" + name + ".yaml")
}
