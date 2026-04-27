package handlers

import (
	"database/sql"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"git.romanzipp.net/romanzipp/news/internal/auth"
	"git.romanzipp.net/romanzipp/news/internal/templates"
)

type SectionsHandler struct {
	db       *sql.DB
	sessions *scs.SessionManager
	tmpl     *templates.Templates
}

func NewSectionsHandler(db *sql.DB, sessions *scs.SessionManager, tmpl *templates.Templates) *SectionsHandler {
	return &SectionsHandler{db: db, sessions: sessions, tmpl: tmpl}
}

func (h *SectionsHandler) SectionsPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	flash, _ := h.sessions.Pop(r.Context(), "flash").(string)
	sections := h.loadSections(user.ID)

	h.tmpl.Render(w, "sections", map[string]any{
		"Title":    "Custom Sections",
		"User":     user,
		"Sections": sections,
		"Flash":    flash,
	})
}

func (h *SectionsHandler) SectionAdd(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	title := r.FormValue("title")
	prompt := r.FormValue("prompt")

	if title == "" || prompt == "" {
		h.sessions.Put(r.Context(), "flash", "Title and prompt are required.")
		http.Redirect(w, r, "/sections", http.StatusSeeOther)
		return
	}

	var maxPos int
	h.db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM custom_sections WHERE user_id = ?", user.ID).Scan(&maxPos)

	h.db.Exec("INSERT INTO custom_sections (user_id, title, prompt, position) VALUES (?, ?, ?, ?)",
		user.ID, title, prompt, maxPos+1)

	h.sessions.Put(r.Context(), "flash", "Section added.")
	http.Redirect(w, r, "/sections", http.StatusSeeOther)
}

func (h *SectionsHandler) SectionUpdate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id := r.PathValue("id")
	title := r.FormValue("title")
	prompt := r.FormValue("prompt")

	h.db.Exec("UPDATE custom_sections SET title = ?, prompt = ? WHERE id = ? AND user_id = ?", title, prompt, id, user.ID)
	h.sessions.Put(r.Context(), "flash", "Section updated.")
	http.Redirect(w, r, "/sections", http.StatusSeeOther)
}

func (h *SectionsHandler) SectionDelete(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id := r.PathValue("id")

	h.db.Exec("DELETE FROM custom_sections WHERE id = ? AND user_id = ?", id, user.ID)
	h.sessions.Put(r.Context(), "flash", "Section removed.")
	http.Redirect(w, r, "/sections", http.StatusSeeOther)
}

type sectionView struct {
	ID      int64
	Title   string
	Prompt  string
	Enabled bool
}

func (h *SectionsHandler) loadSections(userID int64) []sectionView {
	rows, err := h.db.Query("SELECT id, title, prompt, enabled FROM custom_sections WHERE user_id = ? ORDER BY position", userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var sections []sectionView
	for rows.Next() {
		var s sectionView
		rows.Scan(&s.ID, &s.Title, &s.Prompt, &s.Enabled)
		sections = append(sections, s)
	}
	return sections
}
