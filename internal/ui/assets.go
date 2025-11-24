package ui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// ParseTemplates builds the template set with common functions.
func ParseTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"formatTime":  formatTime,
		"shortID":     shortID,
		"now":         time.Now,
		"currentYear": currentYear,
	}

	root := template.New("base").Funcs(funcMap)
	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".html") {
			return nil
		}
		bytes, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}
		name := strings.TrimPrefix(path, "templates/")
		if _, err := root.New(name).Parse(string(bytes)); err != nil {
			return fmt.Errorf("parse template %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return root, nil
}

// StaticFiles exposes embedded static assets.
func StaticFiles() http.FileSystem {
	fsys, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(fmt.Sprintf("static assets missing: %v", err))
	}
	return http.FS(fsys)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(time.UTC).Format(time.RFC822)
}

func currentYear() int {
	return time.Now().Year()
}

func shortID(v any) string {
	switch val := v.(type) {
	case fmt.Stringer:
		if len(val.String()) >= 8 {
			return val.String()[:8]
		}
		return val.String()
	case string:
		if len(val) >= 8 {
			return val[:8]
		}
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}
