package handlers

import (
	"database/sql"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"git.romanzipp.net/romanzipp/news/internal/auth"
	"git.romanzipp.net/romanzipp/news/internal/templates"
)

type InterestsHandler struct {
	db       *sql.DB
	sessions *scs.SessionManager
	tmpl     *templates.Templates
}

func NewInterestsHandler(db *sql.DB, sessions *scs.SessionManager, tmpl *templates.Templates) *InterestsHandler {
	return &InterestsHandler{db: db, sessions: sessions, tmpl: tmpl}
}

func (h *InterestsHandler) WizardPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	interests := h.loadInterests(user.ID)

	h.tmpl.Render(w, "wizard", map[string]any{
		"User":          user,
		"InterestsDeep": findInterest(interests, "Topics I care deeply about"),
		"InterestsLess": findInterest(interests, "Topics I want less of"),
		"Languages":     findInterest(interests, "Languages I read"),
		"InterestsOther": findInterest(interests, "Other notes"),
	})
}

func (h *InterestsHandler) WizardSave(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	fields := []struct {
		label string
		name  string
	}{
		{"Topics I care deeply about", "interests_deep"},
		{"Topics I want less of", "interests_less"},
		{"Languages I read", "languages"},
		{"Other notes", "interests_other"},
	}

	h.db.Exec("DELETE FROM interests WHERE user_id = ?", user.ID)
	for i, f := range fields {
		val := r.FormValue(f.name)
		if val != "" {
			h.db.Exec("INSERT INTO interests (user_id, label, value, position) VALUES (?, ?, ?, ?)",
				user.ID, f.label, val, i)
		}
	}

	http.Redirect(w, r, "/feeds", http.StatusSeeOther)
}

func (h *InterestsHandler) InterestsPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	interests := h.loadInterests(user.ID)

	h.tmpl.Render(w, "interests", map[string]any{
		"Title":         "Interests",
		"User":          user,
		"InterestsDeep": findInterest(interests, "Topics I care deeply about"),
		"InterestsLess": findInterest(interests, "Topics I want less of"),
		"Languages":     findInterest(interests, "Languages I read"),
		"InterestsOther": findInterest(interests, "Other notes"),
	})
}

func (h *InterestsHandler) InterestsSave(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	fields := []struct {
		label string
		name  string
	}{
		{"Topics I care deeply about", "interests_deep"},
		{"Topics I want less of", "interests_less"},
		{"Languages I read", "languages"},
		{"Other notes", "interests_other"},
	}

	h.db.Exec("DELETE FROM interests WHERE user_id = ?", user.ID)
	for i, f := range fields {
		val := r.FormValue(f.name)
		if val != "" {
			h.db.Exec("INSERT INTO interests (user_id, label, value, position) VALUES (?, ?, ?, ?)",
				user.ID, f.label, val, i)
		}
	}

	h.sessions.Put(r.Context(), "flash", "Interests updated.")
	http.Redirect(w, r, "/interests", http.StatusSeeOther)
}

type interestRow struct {
	Label string
	Value string
}

func (h *InterestsHandler) loadInterests(userID int64) []interestRow {
	rows, err := h.db.Query("SELECT label, value FROM interests WHERE user_id = ? ORDER BY position", userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var interests []interestRow
	for rows.Next() {
		var i interestRow
		rows.Scan(&i.Label, &i.Value)
		interests = append(interests, i)
	}
	return interests
}

func findInterest(interests []interestRow, label string) string {
	for _, i := range interests {
		if i.Label == label {
			return i.Value
		}
	}
	return ""
}
