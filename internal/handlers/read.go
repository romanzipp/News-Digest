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

	var id int64
	fmt.Sscan(itemID, &id)

	h.db.Exec(
		"INSERT INTO read_items (user_id, digest_item_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		user.ID, id,
	)

	var sourceURL string
	h.db.QueryRow("SELECT source_url FROM digest_items WHERE id = ?", id).Scan(&sourceURL)

	if sourceURL == "" {
		sourceURL = "/"
	}

	http.Redirect(w, r, sourceURL, http.StatusFound)
}
