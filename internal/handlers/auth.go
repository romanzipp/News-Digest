package handlers

import (
	"errors"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"git.romanzipp.net/romanzipp/news/internal/auth"
	"git.romanzipp.net/romanzipp/news/internal/config"
	"git.romanzipp.net/romanzipp/news/internal/templates"
)

type AuthHandler struct {
	cfg      *config.Config
	service  *auth.Service
	sessions *scs.SessionManager
	tmpl     *templates.Templates
}

func NewAuthHandler(cfg *config.Config, service *auth.Service, sessions *scs.SessionManager, tmpl *templates.Templates) *AuthHandler {
	return &AuthHandler{cfg: cfg, service: service, sessions: sessions, tmpl: tmpl}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.tmpl.Render(w, "login", map[string]any{
		"RegistrationEnabled": h.cfg.RegistrationEnabled,
	})
}

func (h *AuthHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := h.service.Login(email, password)
	if err != nil {
		h.tmpl.Render(w, "login", map[string]any{
			"Error":               "Invalid email or password.",
			"Email":               email,
			"RegistrationEnabled": h.cfg.RegistrationEnabled,
		})
		return
	}

	h.sessions.Put(r.Context(), "user_id", user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.RegistrationEnabled {
		http.NotFound(w, r)
		return
	}
	h.tmpl.Render(w, "register", map[string]any{})
}

func (h *AuthHandler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.RegistrationEnabled {
		http.NotFound(w, r)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	if password != confirm {
		h.tmpl.Render(w, "register", map[string]any{
			"Error":    "Passwords do not match.",
			"Username": username,
			"Email":    email,
		})
		return
	}

	user, err := h.service.Register(email, username, password)
	if err != nil {
		msg := "Registration failed."
		if errors.Is(err, auth.ErrUserExists) {
			msg = "An account with that email or username already exists."
		} else if errors.Is(err, auth.ErrPasswordTooShort) {
			msg = "Password must be at least 8 characters."
		} else if errors.Is(err, auth.ErrEmailRequired) || errors.Is(err, auth.ErrUsernameRequired) {
			msg = "All fields are required."
		}
		h.tmpl.Render(w, "register", map[string]any{
			"Error":    msg,
			"Username": username,
			"Email":    email,
		})
		return
	}

	h.sessions.Put(r.Context(), "user_id", user.ID)
	http.Redirect(w, r, "/wizard", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessions.Destroy(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
