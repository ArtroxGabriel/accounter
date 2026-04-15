package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
)

//go:embed templates
var TemplatesFS embed.FS

// LoadTemplates parses all HTML templates under the templates directory.
func LoadTemplates(fsys fs.FS) (*template.Template, error) {
	templateFiles := make([]string, 0)

	if err := fs.WalkDir(fsys, "templates", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}

		templateFiles = append(templateFiles, path)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walking templates directory: %w", err)
	}

	if len(templateFiles) == 0 {
		return nil, fmt.Errorf("no template files found")
	}

	tmpl, err := template.ParseFS(fsys, templateFiles...)
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	return tmpl, nil
}
