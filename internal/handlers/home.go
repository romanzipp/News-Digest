package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/roman-zipp/news/internal/auth"
	"github.com/roman-zipp/news/internal/models"
	"github.com/roman-zipp/news/internal/templates"
)

type HomeHandler struct {
	db       *sql.DB
	sessions *scs.SessionManager
	tmpl     *templates.Templates
}

func NewHomeHandler(db *sql.DB, sessions *scs.SessionManager, tmpl *templates.Templates) *HomeHandler {
	return &HomeHandler{db: db, sessions: sessions, tmpl: tmpl}
}

func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	hasInterests, _ := h.hasInterests(user.ID)
	if !hasInterests {
		http.Redirect(w, r, "/wizard", http.StatusSeeOther)
		return
	}

	today := time.Now().Format("2006-01-02")
	http.Redirect(w, r, "/digest/"+today, http.StatusSeeOther)
}

func (h *HomeHandler) DigestByDate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	date := r.PathValue("date")

	digest, items, err := h.loadDigest(user.ID, date, 0)
	if err != nil {
		h.renderEmpty(w, user, date)
		return
	}

	allDigests := h.listDigestsForDate(user.ID, date)
	h.renderDigest(w, r, user, date, digest, items, allDigests)
}

func (h *HomeHandler) DigestByID(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	date := r.PathValue("date")
	digestID := r.PathValue("id")

	var id int64
	fmt.Sscan(digestID, &id)

	digest, items, err := h.loadDigest(user.ID, date, id)
	if err != nil {
		h.renderEmpty(w, user, date)
		return
	}

	allDigests := h.listDigestsForDate(user.ID, date)
	h.renderDigest(w, r, user, date, digest, items, allDigests)
}

type digestView struct {
	ID          int64
	IsAuto      bool
	GeneratedAt string
}

type itemView struct {
	ID         int64
	Position   int
	Priority   int
	Category   string
	Headline   string
	TLDR       string
	Bullets    []string
	SourceName string
	SourceURL  string
	ImageURL   string
	ReadTime   int
	Language   string
	Importance string
	ItemID     int64
	UserVote   int
	TimeAgo    string
}

func (h *HomeHandler) renderDigest(w http.ResponseWriter, r *http.Request, user *models.User, date string, digest *digestRow, items []itemView, allDigests []digestView) {
	flash, _ := h.sessions.Pop(r.Context(), "flash").(string)

	prevDate := h.adjacentDate(user.ID, date, "prev")
	nextDate := h.adjacentDate(user.ID, date, "next")
	today := time.Now().Format("2006-01-02")
	if nextDate == "" && date != today {
		nextDate = today
	}

	feedNames := h.feedNames(user.ID)
	prefSummary := h.preferenceSummary(user.ID)

	var trending []trendingView
	if digest.RawResponse != "" {
		trending = parseTrending(digest.RawResponse)
	}

	data := map[string]any{
		"Title":             fmt.Sprintf("Digest — %s", date),
		"User":              user,
		"Date":              date,
		"DateFormatted":     formatDateLong(date),
		"PrevDate":          prevDate,
		"NextDate":          nextDate,
		"Digest":            digest,
		"Items":             items,
		"AllDigests":        allDigests,
		"ActiveDigestID":    digest.ID,
		"Flash":             flash,
		"Trending":          trending,
		"FeedNames":         feedNames,
		"PreferenceSummary": prefSummary,
		"DigestMeta": map[string]any{
			"FeedsCount":       digest.FeedsCount,
			"ArticlesReviewed": digest.ArticlesReviewed,
			"ArticlesSurfaced": digest.ArticlesSurfaced,
		},
	}

	top := items
	var rest []itemView
	if len(top) > 3 {
		rest = top[3:]
		top = top[:3]
	}
	data["TopItems"] = top
	data["RestItems"] = rest

	h.tmpl.Render(w, "home", data)
}

func (h *HomeHandler) renderEmpty(w http.ResponseWriter, user *models.User, date string) {
	prevDate := h.adjacentDate(user.ID, date, "prev")
	nextDate := h.adjacentDate(user.ID, date, "next")
	today := time.Now().Format("2006-01-02")
	if nextDate == "" && date != today {
		nextDate = today
	}

	h.tmpl.Render(w, "home", map[string]any{
		"Title":         fmt.Sprintf("Digest — %s", date),
		"User":          user,
		"Date":          date,
		"DateFormatted": formatDateLong(date),
		"PrevDate":      prevDate,
		"NextDate":      nextDate,
		"Empty":         true,
		"DigestMeta":    nil,
		"FeedNames":     h.feedNames(user.ID),
	})
}

type digestRow struct {
	ID               int64
	IsAuto           bool
	ArticlesReviewed int
	ArticlesSurfaced int
	FeedsCount       int
	GeneratedAt      time.Time
	RawResponse      string
}

func (h *HomeHandler) loadDigest(userID int64, date string, specificID int64) (*digestRow, []itemView, error) {
	var d digestRow
	var err error

	if specificID > 0 {
		err = h.db.QueryRow(
			"SELECT id, is_auto, articles_reviewed, articles_surfaced, feeds_count, generated_at, raw_response FROM digests WHERE id = ? AND user_id = ? AND date = ?",
			specificID, userID, date,
		).Scan(&d.ID, &d.IsAuto, &d.ArticlesReviewed, &d.ArticlesSurfaced, &d.FeedsCount, &d.GeneratedAt, &d.RawResponse)
	} else {
		err = h.db.QueryRow(
			"SELECT id, is_auto, articles_reviewed, articles_surfaced, feeds_count, generated_at, raw_response FROM digests WHERE user_id = ? AND date = ? ORDER BY generated_at DESC LIMIT 1",
			userID, date,
		).Scan(&d.ID, &d.IsAuto, &d.ArticlesReviewed, &d.ArticlesSurfaced, &d.FeedsCount, &d.GeneratedAt, &d.RawResponse)
	}
	if err != nil {
		return nil, nil, err
	}

	rows, err := h.db.Query(
		`SELECT di.id, di.position, di.priority, di.category, di.headline, di.tldr, di.bullets,
		        di.source_name, di.source_url, di.image_url, di.read_time, di.language, di.importance,
		        COALESCE(v.value, 0)
		 FROM digest_items di
		 LEFT JOIN votes v ON v.digest_item_id = di.id AND v.user_id = ?
		 WHERE di.digest_id = ?
		 ORDER BY di.position`,
		userID, d.ID,
	)
	if err != nil {
		return &d, nil, nil
	}
	defer rows.Close()

	var items []itemView
	for rows.Next() {
		var it itemView
		var bulletsJSON string
		rows.Scan(&it.ID, &it.Position, &it.Priority, &it.Category, &it.Headline, &it.TLDR, &bulletsJSON,
			&it.SourceName, &it.SourceURL, &it.ImageURL, &it.ReadTime, &it.Language, &it.Importance,
			&it.UserVote)
		json.Unmarshal([]byte(bulletsJSON), &it.Bullets)
		it.ItemID = it.ID
		it.TimeAgo = timeAgo(d.GeneratedAt)
		items = append(items, it)
	}

	return &d, items, nil
}

func (h *HomeHandler) listDigestsForDate(userID int64, date string) []digestView {
	rows, err := h.db.Query(
		"SELECT id, is_auto, generated_at FROM digests WHERE user_id = ? AND date = ? ORDER BY generated_at",
		userID, date,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var digests []digestView
	for rows.Next() {
		var d digestView
		var t time.Time
		rows.Scan(&d.ID, &d.IsAuto, &t)
		d.GeneratedAt = t.Format("15:04")
		digests = append(digests, d)
	}
	return digests
}

func (h *HomeHandler) adjacentDate(userID int64, date, direction string) string {
	var d string
	var op, order string
	if direction == "prev" {
		op, order = "<", "DESC"
	} else {
		op, order = ">", "ASC"
	}
	h.db.QueryRow(
		fmt.Sprintf("SELECT date FROM digests WHERE user_id = ? AND date %s ? ORDER BY date %s LIMIT 1", op, order),
		userID, date,
	).Scan(&d)
	return d
}

func (h *HomeHandler) hasInterests(userID int64) (bool, error) {
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM interests WHERE user_id = ?", userID).Scan(&count)
	return count > 0, err
}

func (h *HomeHandler) feedNames(userID int64) string {
	rows, err := h.db.Query("SELECT name FROM sources WHERE user_id = ? AND enabled = 1", userID)
	if err != nil {
		return ""
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, " · ")
}

func (h *HomeHandler) preferenceSummary(userID int64) string {
	var val string
	h.db.QueryRow("SELECT value FROM interests WHERE user_id = ? AND label = 'Topics I care deeply about'", userID).Scan(&val)
	return val
}

type trendingView struct {
	Topic string
	Count int
	Delta string
}

func parseTrending(raw string) []trendingView {
	var resp struct {
		Trending []trendingView `json:"trending"`
	}
	json.Unmarshal([]byte(raw), &resp)
	return resp.Trending
}

func formatDateLong(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return t.Format("Monday, January 2, 2006")
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
