package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/roman-zipp/news/internal/models"
)

type Provider interface {
	Type() string
	Fetch(ctx context.Context, src models.Source) ([]models.Article, error)
	Validate(cfg json.RawMessage) error
}

type Registry struct {
	providers map[string]Provider
	db        *sql.DB
}

func NewRegistry(db *sql.DB) *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		db:        db,
	}
}

func (r *Registry) Register(p Provider) {
	r.providers[p.Type()] = p
}

func (r *Registry) FetchSource(ctx context.Context, src models.Source) ([]models.Article, error) {
	p, ok := r.providers[src.Type]
	if !ok {
		return nil, fmt.Errorf("unknown source type: %s", src.Type)
	}
	return p.Fetch(ctx, src)
}

func (r *Registry) FetchAllForUser(ctx context.Context, userID int64) (int, error) {
	rows, err := r.db.Query("SELECT id, user_id, type, name, url, config, enabled FROM sources WHERE user_id = ? AND enabled = 1", userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var sources []models.Source
	for rows.Next() {
		var s models.Source
		rows.Scan(&s.ID, &s.UserID, &s.Type, &s.Name, &s.URL, &s.Config, &s.Enabled)
		sources = append(sources, s)
	}

	totalNew := 0
	for _, src := range sources {
		articles, err := r.FetchSource(ctx, src)
		if err != nil {
			log.Printf("fetch source %d (%s): %v", src.ID, src.Name, err)
			continue
		}

		for _, a := range articles {
			a.UserID = userID
			a.SourceID = sql.NullInt64{Int64: src.ID, Valid: true}
			inserted, err := r.upsertArticle(a)
			if err != nil {
				log.Printf("upsert article %s: %v", a.GUID, err)
				continue
			}
			if inserted {
				totalNew++
			}
		}

		r.db.Exec("UPDATE sources SET last_fetched_at = ? WHERE id = ?", time.Now(), src.ID)
	}

	return totalNew, nil
}

func (r *Registry) FetchAllUsers(ctx context.Context) error {
	rows, err := r.db.Query("SELECT DISTINCT user_id FROM sources WHERE enabled = 1")
	if err != nil {
		return err
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		userIDs = append(userIDs, id)
	}

	for _, uid := range userIDs {
		n, err := r.FetchAllForUser(ctx, uid)
		if err != nil {
			log.Printf("fetch user %d: %v", uid, err)
			continue
		}
		log.Printf("fetched %d new articles for user %d", n, uid)
	}
	return nil
}

func (r *Registry) upsertArticle(a models.Article) (bool, error) {
	result, err := r.db.Exec(
		`INSERT INTO articles (user_id, source_id, guid, title, url, content, author, image_url, language, published_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, guid) DO NOTHING`,
		a.UserID, a.SourceID, a.GUID, a.Title, a.URL, a.Content, a.Author, a.ImageURL, a.Language, a.PublishedAt,
	)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}
