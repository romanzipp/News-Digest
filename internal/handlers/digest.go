package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"git.romanzipp.net/romanzipp/news/internal/auth"
	"git.romanzipp.net/romanzipp/news/internal/digest"
	"git.romanzipp.net/romanzipp/news/internal/templates"
)

type DigestHandler struct {
	sessions  *scs.SessionManager
	tmpl      *templates.Templates
	generator *digest.Generator
}

func NewDigestHandler(sessions *scs.SessionManager, tmpl *templates.Templates, generator *digest.Generator) *DigestHandler {
	return &DigestHandler{sessions: sessions, tmpl: tmpl, generator: generator}
}

func (h *DigestHandler) Generate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	_, err := h.generator.GenerateForUser(ctx, user.ID, false)
	if err != nil {
		h.sessions.Put(r.Context(), "flash", "Generation failed: "+err.Error())
	} else {
		h.sessions.Put(r.Context(), "flash", "Digest generated successfully.")
	}

	today := time.Now().Format("2006-01-02")
	http.Redirect(w, r, "/digest/"+today, http.StatusSeeOther)
}
