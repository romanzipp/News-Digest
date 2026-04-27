package templates

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"git.romanzipp.net/romanzipp/news/internal/models"
)

type Templates struct {
	dir   string
	funcs template.FuncMap
}

func New(dir string) *Templates {
	t := &Templates{dir: dir}
	t.funcs = template.FuncMap{
		"formatDate": func(s string) string {
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return s
			}
			return t.Format("Monday, January 2, 2006")
		},
		"formatTime": func(t time.Time) string {
			return t.Format("15:04")
		},
		"proxyURL": func(rawURL string) string {
			if rawURL == "" {
				return ""
			}
			return "/proxy/image?url=" + url.QueryEscape(rawURL)
		},
		"categoryHex": func(cat string) string {
			if c, ok := models.CategoryColors[cat]; ok {
				return c
			}
			return "#6b5f52"
		},
		"jsonBullets": func(s string) []string {
			var bullets []string
			if err := json.Unmarshal([]byte(s), &bullets); err != nil {
				return nil
			}
			return bullets
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		"lower":    strings.ToLower,
		"contains": strings.Contains,
		"dict": func(pairs ...any) map[string]any {
			m := make(map[string]any)
			for i := 0; i+1 < len(pairs); i += 2 {
				key, _ := pairs[i].(string)
				m[key] = pairs[i+1]
			}
			return m
		},
	}
	return t
}

func (t *Templates) Render(w http.ResponseWriter, name string, data any) error {
	tmpl, err := t.parse(name)
	if err != nil {
		return fmt.Errorf("parse template %s: %w", name, err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, "layout", data)
}

func (t *Templates) RenderPartial(w http.ResponseWriter, name string, data any) error {
	partials, _ := filepath.Glob(filepath.Join(t.dir, "partials", "*.html"))
	tmpl, err := template.New("").Funcs(t.funcs).ParseFiles(partials...)
	if err != nil {
		return fmt.Errorf("parse partial %s: %w", name, err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, name, data)
}

func (t *Templates) parse(name string) (*template.Template, error) {
	partials, _ := filepath.Glob(filepath.Join(t.dir, "partials", "*.html"))
	files := append([]string{
		filepath.Join(t.dir, "layout.html"),
		filepath.Join(t.dir, name+".html"),
	}, partials...)

	return template.New("").Funcs(t.funcs).ParseFiles(files...)
}
