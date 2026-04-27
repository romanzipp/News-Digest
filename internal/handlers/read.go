package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"git.romanzipp.net/romanzipp/news/internal/auth"
)

type ReadHandler struct {
	db *sql.DB
}

func NewReadHandler(db *sql.DB) *ReadHandler {
	return &ReadHandler{db: db}
}

func (h *ReadHandler) MarkAndRedirect(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	itemID := r.PathValue("id")
	itemType := r.URL.Query().Get("type")

	var id int64
	fmt.Sscan(itemID, &id)

	var articleID sql.NullInt64
	var sourceURL string

	if itemType == "section" {
		h.db.QueryRow("SELECT article_id, source_url FROM section_items WHERE id = ?", id).Scan(&articleID, &sourceURL)
	} else {
		h.db.QueryRow("SELECT article_id, source_url FROM digest_items WHERE id = ?", id).Scan(&articleID, &sourceURL)
	}

	if articleID.Valid {
		h.db.Exec(
			"INSERT INTO read_items (user_id, article_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			user.ID, articleID.Int64,
		)
	}

	if sourceURL == "" {
		sourceURL = "/"
	}

	http.Redirect(w, r, sourceURL, http.StatusFound)
}
