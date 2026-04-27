package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"git.romanzipp.net/romanzipp/news/internal/auth"
	"git.romanzipp.net/romanzipp/news/internal/source"
	"git.romanzipp.net/romanzipp/news/internal/templates"
)

type FeedsHandler struct {
	db       *sql.DB
	sessions *scs.SessionManager
	tmpl     *templates.Templates
	registry *source.Registry
}

func NewFeedsHandler(db *sql.DB, sessions *scs.SessionManager, tmpl *templates.Templates, registry *source.Registry) *FeedsHandler {
	return &FeedsHandler{db: db, sessions: sessions, tmpl: tmpl, registry: registry}
}

func (h *FeedsHandler) FeedsPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	flash, _ := h.sessions.Pop(r.Context(), "flash").(string)

	feeds := h.loadFeeds(user.ID)
	freshrss := h.loadFreshRSS(user.ID)

	h.tmpl.Render(w, "feeds", map[string]any{
		"Title":    "Feeds",
		"User":     user,
		"Feeds":    feeds,
		"FreshRSS": freshrss,
		"Flash":    flash,
	})
}

func (h *FeedsHandler) FeedAdd(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	feedURL := r.FormValue("url")
	name := r.FormValue("name")

	if feedURL == "" {
		h.sessions.Put(r.Context(), "flash", "URL is required.")
		http.Redirect(w, r, "/feeds", http.StatusSeeOther)
		return
	}

	_, err := h.db.Exec(
		"INSERT INTO sources (user_id, type, name, url) VALUES (?, 'rss', ?, ?)",
		user.ID, name, feedURL,
	)
	if err != nil {
		h.sessions.Put(r.Context(), "flash", "Feed already exists or could not be added.")
		http.Redirect(w, r, "/feeds", http.StatusSeeOther)
		return
	}

	h.sessions.Put(r.Context(), "flash", "Feed added.")
	http.Redirect(w, r, "/feeds", http.StatusSeeOther)
}

func (h *FeedsHandler) FeedDelete(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id := r.PathValue("id")

	h.db.Exec("DELETE FROM sources WHERE id = ? AND user_id = ?", id, user.ID)
	h.sessions.Put(r.Context(), "flash", "Feed removed.")
	http.Redirect(w, r, "/feeds", http.StatusSeeOther)
}

func (h *FeedsHandler) FeedToggle(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id := r.PathValue("id")

	h.db.Exec("UPDATE sources SET enabled = CASE WHEN enabled = 1 THEN 0 ELSE 1 END WHERE id = ? AND user_id = ?", id, user.ID)
	http.Redirect(w, r, "/feeds", http.StatusSeeOther)
}

func (h *FeedsHandler) FreshRSSPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	flash, _ := h.sessions.Pop(r.Context(), "flash").(string)
	freshrss := h.loadFreshRSS(user.ID)

	h.tmpl.Render(w, "freshrss", map[string]any{
		"Title":    "FreshRSS",
		"User":     user,
		"FreshRSS": freshrss,
		"Flash":    flash,
	})
}

func (h *FeedsHandler) FreshRSSSave(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	baseURL := r.FormValue("url")
	username := r.FormValue("username")
	apiPassword := r.FormValue("api_password")

	cfg, _ := json.Marshal(source.FreshRSSConfig{
		Username:    username,
		APIPassword: apiPassword,
	})

	existing := h.loadFreshRSS(user.ID)
	if existing != nil {
		h.db.Exec("UPDATE sources SET url = ?, config = ?, name = 'FreshRSS' WHERE id = ?", baseURL, string(cfg), existing.ID)
	} else {
		h.db.Exec("INSERT INTO sources (user_id, type, name, url, config) VALUES (?, 'freshrss', 'FreshRSS', ?, ?)",
			user.ID, baseURL, string(cfg))
	}

	h.sessions.Put(r.Context(), "flash", "FreshRSS configuration saved.")
	http.Redirect(w, r, "/feeds", http.StatusSeeOther)
}

func (h *FeedsHandler) FreshRSSTest(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	baseURL := r.FormValue("url")
	username := r.FormValue("username")
	apiPassword := r.FormValue("api_password")

	var flash string
	err := source.TestFreshRSSConnection(baseURL, username, apiPassword)
	if err != nil {
		flash = fmt.Sprintf("Connection failed: %v", err)
	} else {
		flash = "Connection successful!"
	}

	h.tmpl.Render(w, "freshrss", map[string]any{
		"Title": "FreshRSS",
		"User":  user,
		"Flash": flash,
		"FreshRSS": &freshRSSView{
			URL:         baseURL,
			Username:    username,
			APIPassword: apiPassword,
		},
	})
}

func (h *FeedsHandler) FetchNow(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	n, errCount, err := h.registry.FetchAllForUser(context.Background(), user.ID)
	if err != nil {
		h.sessions.Put(r.Context(), "flash", fmt.Sprintf("Fetch error: %v", err))
	} else if errCount > 0 {
		h.sessions.Put(r.Context(), "flash", fmt.Sprintf("Fetched %d new articles. %d source(s) failed — check server logs.", n, errCount))
	} else {
		h.sessions.Put(r.Context(), "flash", fmt.Sprintf("Fetched %d new articles.", n))
	}
	http.Redirect(w, r, "/feeds", http.StatusSeeOther)
}

type feedView struct {
	ID            int64
	Name          string
	URL           string
	Type          string
	Enabled       bool
	LastFetchedAt string
}

func (h *FeedsHandler) loadFeeds(userID int64) []feedView {
	rows, err := h.db.Query("SELECT id, name, url, type, enabled, last_fetched_at FROM sources WHERE user_id = ? ORDER BY created_at", userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var feeds []feedView
	for rows.Next() {
		var f feedView
		var lastFetched sql.NullTime
		rows.Scan(&f.ID, &f.Name, &f.URL, &f.Type, &f.Enabled, &lastFetched)
		if lastFetched.Valid {
			f.LastFetchedAt = lastFetched.Time.Format("2006-01-02 15:04")
		}
		feeds = append(feeds, f)
	}
	return feeds
}

type freshRSSView struct {
	ID          int64
	URL         string
	Username    string
	APIPassword string
}

func (h *FeedsHandler) loadFreshRSS(userID int64) *freshRSSView {
	var v freshRSSView
	var cfgStr string
	err := h.db.QueryRow("SELECT id, url, config FROM sources WHERE user_id = ? AND type = 'freshrss'", userID).
		Scan(&v.ID, &v.URL, &cfgStr)
	if err != nil {
		return nil
	}
	var cfg source.FreshRSSConfig
	json.Unmarshal([]byte(cfgStr), &cfg)
	v.Username = cfg.Username
	v.APIPassword = cfg.APIPassword
	return &v
}
