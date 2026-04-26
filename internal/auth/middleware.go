package auth

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/roman-zipp/news/internal/models"
)

type contextKey string

const userContextKey contextKey = "user"

type Middleware struct {
	sessions *scs.SessionManager
	service  *Service
}

func NewMiddleware(sessions *scs.SessionManager, service *Service) *Middleware {
	return &Middleware{sessions: sessions, service: service}
}

func (m *Middleware) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := m.sessions.GetInt64(r.Context(), "user_id")
		if userID > 0 {
			user, err := m.service.GetUser(userID)
			if err == nil {
				ctx := context.WithValue(r.Context(), userContextKey, user)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserFromContext(ctx context.Context) *models.User {
	user, _ := ctx.Value(userContextKey).(*models.User)
	return user
}
