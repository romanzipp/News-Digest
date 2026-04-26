package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"git.romanzipp.net/romanzipp/news/internal/auth"
	"git.romanzipp.net/romanzipp/news/internal/templates"
)

type VotesHandler struct {
	db   *sql.DB
	tmpl *templates.Templates
}

func NewVotesHandler(db *sql.DB, tmpl *templates.Templates) *VotesHandler {
	return &VotesHandler{db: db, tmpl: tmpl}
}

func (h *VotesHandler) Vote(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	itemID := r.FormValue("item_id")
	value := r.FormValue("value")

	var v int
	fmt.Sscan(value, &v)
	if v != 1 && v != -1 {
		v = 0
	}

	var existingValue int
	err := h.db.QueryRow("SELECT value FROM votes WHERE user_id = ? AND digest_item_id = ?", user.ID, itemID).Scan(&existingValue)

	if err == sql.ErrNoRows {
		h.db.Exec("INSERT INTO votes (user_id, digest_item_id, value) VALUES (?, ?, ?)", user.ID, itemID, v)
	} else if existingValue == v {
		h.db.Exec("DELETE FROM votes WHERE user_id = ? AND digest_item_id = ?", user.ID, itemID)
		v = 0
	} else {
		h.db.Exec("UPDATE votes SET value = ? WHERE user_id = ? AND digest_item_id = ?", v, user.ID, itemID)
	}

	// For htmx requests, return just the vote buttons partial
	if r.Header.Get("HX-Request") == "true" {
		h.tmpl.RenderPartial(w, "vote_buttons", map[string]any{
			"ItemID":   itemID,
			"UserVote": v,
		})
		return
	}

	// Fallback: redirect back
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}
