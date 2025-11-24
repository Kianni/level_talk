package ui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"leveltalk/internal/i18n"
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
		"safeURL": func(u string) template.URL {
			return template.URL(u)
		},
		"dialogName": dialogName,
		"t": func(lang, key string) template.HTML {
			return template.HTML(i18n.Get(lang, key))
		},
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

// dialogName generates a concise name for a dialog.
// Uses the stored title if available, otherwise falls back to metadata-based name.
func dialogName(title, inputLang, dialogLang, cefr string, inputWords []string) string {
	// If title is provided and not empty, use it
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	// Fallback to metadata-based name
	var name strings.Builder
	name.WriteString(strings.ToUpper(inputLang))
	name.WriteString("→")
	name.WriteString(strings.ToUpper(dialogLang))
	name.WriteString(" ")
	name.WriteString(cefr)
	if len(inputWords) > 0 {
		firstWord := strings.TrimSpace(inputWords[0])
		if len(firstWord) > 15 {
			firstWord = firstWord[:15] + "..."
		}
		name.WriteString(" - ")
		name.WriteString(firstWord)
	}
	return name.String()
}
