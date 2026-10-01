package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

// TemplateInfo describes a template.
type TemplateInfo struct {
	Name   string
	Source string // "builtin" or "custom"
	Path   string
	Vars   []string
}

// Store manages custom templates on the filesystem.
type Store struct {
	dir string
}

// NewStore creates a new template store at the XDG data directory.
func NewStore() (*Store, error) {
	dir := filepath.Join(xdg.DataHome, "dtiam", "templates")
	return &Store{dir: dir}, nil
}

// Path returns the template storage directory.
func (s *Store) Path() string {
	return s.dir
}

// List returns all custom templates.
func (s *Store) List() ([]TemplateInfo, error) {
	if _, err := os.Stat(s.dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []TemplateInfo
	for _, entry := range entries {
		if entry.IsDir() || !isTemplateFile(entry.Name()) {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		content, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		templates = append(templates, TemplateInfo{
			Name:   name,
			Source: "custom",
			Path:   filepath.Join(s.dir, entry.Name()),
			Vars:   ExtractVariables(string(content)),
		})
	}

	return templates, nil
}

// Get returns the content of a custom template.
func (s *Store) Get(name string) ([]byte, error) {
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		path := filepath.Join(s.dir, name+ext)
		if content, err := os.ReadFile(path); err == nil {
			return content, nil
		}
	}
	return nil, fmt.Errorf("custom template %q not found", name)
}

// Save saves a custom template.
func (s *Store) Save(name string, content []byte) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}
	path := filepath.Join(s.dir, name+".yaml")
	return os.WriteFile(path, content, 0644)
}

// Delete removes a custom template.
func (s *Store) Delete(name string) error {
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		path := filepath.Join(s.dir, name+ext)
		if _, err := os.Stat(path); err == nil {
			return os.Remove(path)
		}
	}
	return fmt.Errorf("custom template %q not found", name)
}

func isTemplateFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml" || ext == ".json"
}
