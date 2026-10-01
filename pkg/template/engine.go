// Package template provides a template engine for rendering IAM resource templates.
package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// RenderTemplate renders a Go text/template string with the given variables.
func RenderTemplate(content string, vars map[string]string) (string, error) {
	funcMap := template.FuncMap{
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}

	tmpl, err := template.New("resource").Funcs(funcMap).Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// ParseSetFlags parses --set key=value flags into a map.
func ParseSetFlags(flags []string) (map[string]string, error) {
	vars := make(map[string]string)
	for _, f := range flags {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --set format %q: expected key=value", f)
		}
		vars[parts[0]] = parts[1]
	}
	return vars, nil
}

// ExtractVariables returns the variable names referenced in a Go template string.
func ExtractVariables(content string) []string {
	var vars []string
	seen := make(map[string]bool)

	// Simple extraction: find {{.varname}} patterns
	i := 0
	for i < len(content) {
		start := strings.Index(content[i:], "{{")
		if start == -1 {
			break
		}
		start += i
		end := strings.Index(content[start:], "}}")
		if end == -1 {
			break
		}
		end += start

		expr := strings.TrimSpace(content[start+2 : end])

		// Extract variable name from expressions like .name, .name | default "x"
		if strings.HasPrefix(expr, ".") {
			varName := expr[1:]
			if pipeIdx := strings.Index(varName, "|"); pipeIdx != -1 {
				varName = strings.TrimSpace(varName[:pipeIdx])
			}
			if varName != "" && !seen[varName] {
				vars = append(vars, varName)
				seen[varName] = true
			}
		}

		i = end + 2
	}

	return vars
}
