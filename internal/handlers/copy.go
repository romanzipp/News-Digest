package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.romanzipp.net/romanzipp/news/internal/templates"
)

type CopyHandler struct {
	db   *sql.DB
	tmpl *templates.Templates
}

func NewCopyHandler(db *sql.DB, tmpl *templates.Templates) *CopyHandler {
	return &CopyHandler{db: db, tmpl: tmpl}
}

func (h *CopyHandler) CopyMarkdown(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("id")

	var headline, sourceName, sourceURL string
	var publishedAt sql.NullString
	h.db.QueryRow(
		`SELECT di.headline, di.source_name, di.source_url, a.published_at
		 FROM digest_items di
		 LEFT JOIN articles a ON a.id = di.article_id
		 WHERE di.id = ?`, itemID,
	).Scan(&headline, &sourceName, &sourceURL, &publishedAt)

	date := time.Now().Format("02.01.2006")
	if publishedAt.Valid {
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, publishedAt.String); err == nil {
				date = t.Format("02.01.2006")
				break
			}
		}
	}

	markdown := fmt.Sprintf("**%s**: %s ([%s](%s))", date, headline, sourceName, sourceURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<span class="copy-toast">
		<input type="text" readonly value="%s" class="copy-toast-input" onfocus="this.select()" autofocus>
		<button type="button" class="vote-btn" onclick="this.parentElement.remove()">&#10005;</button>
	</span>`, escapeAttr(markdown))
}

func escapeAttr(s string) string {
	r := strings.NewReplacer(`"`, "&quot;", "&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
