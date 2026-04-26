package digest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"git.romanzipp.net/romanzipp/news/internal/ai"
	"git.romanzipp.net/romanzipp/news/internal/config"
	"git.romanzipp.net/romanzipp/news/internal/models"
	"git.romanzipp.net/romanzipp/news/internal/source"
)

type Generator struct {
	db       *sql.DB
	cfg      *config.Config
	ai       *ai.Client
	registry *source.Registry
}

func NewGenerator(db *sql.DB, cfg *config.Config, aiClient *ai.Client, registry *source.Registry) *Generator {
	return &Generator{db: db, cfg: cfg, ai: aiClient, registry: registry}
}

func (g *Generator) GenerateForUser(ctx context.Context, userID int64, isAuto bool) (*models.Digest, error) {
	// Fetch fresh articles first
	n, err := g.registry.FetchAllForUser(ctx, userID)
	if err != nil {
		log.Printf("fetch before digest for user %d: %v", userID, err)
	} else {
		log.Printf("fetched %d new articles for user %d before digest", n, userID)
	}

	articles, err := g.loadRecentArticles(userID)
	if err != nil {
		return nil, fmt.Errorf("load articles: %w", err)
	}
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles to digest")
	}

	interests := g.loadInterests(userID)
	votes := g.loadVoteHistory(userID)
	sections := g.loadSections(userID)

	systemPrompt := buildSystemPrompt(interests, votes, sections, models.DefaultCategories)
	systemTokens := estimateTokens(systemPrompt)

	maxContext := g.cfg.AIMaxContext
	if maxContext == 0 {
		maxContext = 128000
	}
	contextBudget := int(float64(maxContext) * 0.8)

	batches := splitIntoBatches(articles, contextBudget, systemTokens)
	log.Printf("digest for user %d: %d articles in %d batches", userID, len(articles), len(batches))

	var responses []*DigestResponse
	for i, batch := range batches {
		articlePrompt := buildArticlePrompt(batch)

		raw, err := g.ai.Complete(ctx, systemPrompt, articlePrompt)
		if err != nil {
			return nil, fmt.Errorf("AI call batch %d: %w", i, err)
		}

		resp, err := parseResponse(raw)
		if err != nil {
			// Retry once
			log.Printf("parse error on batch %d, retrying: %v", i, err)
			raw, err = g.ai.Complete(ctx, systemPrompt+"\n\nIMPORTANT: Respond with valid JSON only.", articlePrompt)
			if err != nil {
				return nil, fmt.Errorf("AI retry batch %d: %w", i, err)
			}
			resp, err = parseResponse(raw)
			if err != nil {
				return nil, fmt.Errorf("parse retry batch %d: %w", i, err)
			}
		}

		responses = append(responses, resp)
	}

	merged := mergeResponses(responses)
	rawJSON, _ := json.Marshal(merged)

	feedCount := g.countFeeds(userID)
	isAutoInt := 0
	if isAuto {
		isAutoInt = 1
	}

	digest, err := g.storeDigest(userID, isAutoInt, merged, feedCount, string(rawJSON))
	if err != nil {
		return nil, fmt.Errorf("store digest: %w", err)
	}

	return digest, nil
}

func (g *Generator) GenerateAllUsers(ctx context.Context) error {
	rows, err := g.db.Query("SELECT DISTINCT user_id FROM sources WHERE enabled = 1")
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
		_, err := g.GenerateForUser(ctx, uid, true)
		if err != nil {
			log.Printf("digest generation for user %d: %v", uid, err)
		}
	}
	return nil
}

func (g *Generator) storeDigest(userID int64, isAuto int, resp *DigestResponse, feedCount int, rawJSON string) (*models.Digest, error) {
	tx, err := g.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	date := time.Now().Format("2006-01-02")

	result, err := tx.Exec(
		`INSERT INTO digests (user_id, date, is_auto, articles_reviewed, articles_surfaced, feeds_count, generation_model, raw_response)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, date, isAuto, resp.Meta.ArticlesReviewed, resp.Meta.ArticlesSurfaced, feedCount, g.cfg.AIModel, rawJSON,
	)
	if err != nil {
		return nil, err
	}
	digestID, _ := result.LastInsertId()

	for i, item := range resp.Items {
		articleID := g.findArticleID(userID, item.ArticleGUID)
		bullets, _ := json.Marshal(item.Bullets)

		tx.Exec(
			`INSERT INTO digest_items (digest_id, article_id, position, priority, category, headline, tldr, bullets, source_name, source_url, image_url, read_time, language, importance)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			digestID, articleID, i, item.Priority, item.Category, item.Headline, item.TLDR, string(bullets),
			item.SourceName, item.SourceURL, item.ImageURL, item.ReadTime, item.Language, item.Importance,
		)
	}

	for _, sec := range resp.Sections {
		for j, item := range sec.Items {
			articleID := g.findArticleID(userID, item.ArticleGUID)
			bullets, _ := json.Marshal(item.Bullets)

			tx.Exec(
				`INSERT INTO section_items (digest_id, section_id, article_id, position, headline, tldr, bullets, source_name, source_url, language)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				digestID, sec.SectionID, articleID, j, item.Headline, item.TLDR, string(bullets),
				item.SourceName, item.SourceURL, item.Language,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.Digest{
		ID:               digestID,
		UserID:           userID,
		Date:             date,
		IsAuto:           isAuto == 1,
		ArticlesReviewed: resp.Meta.ArticlesReviewed,
		ArticlesSurfaced: resp.Meta.ArticlesSurfaced,
		FeedsCount:       feedCount,
	}, nil
}

func (g *Generator) loadRecentArticles(userID int64) ([]models.Article, error) {
	since := time.Now().Add(-48 * time.Hour)
	rows, err := g.db.Query(
		"SELECT id, user_id, source_id, guid, title, url, content, author, image_url, language, published_at FROM articles WHERE user_id = ? AND fetched_at > ? ORDER BY published_at DESC",
		userID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var a models.Article
		rows.Scan(&a.ID, &a.UserID, &a.SourceID, &a.GUID, &a.Title, &a.URL, &a.Content, &a.Author, &a.ImageURL, &a.Language, &a.PublishedAt)
		articles = append(articles, a)
	}
	return articles, nil
}

func (g *Generator) loadInterests(userID int64) []models.Interest {
	rows, err := g.db.Query("SELECT id, user_id, label, value, position FROM interests WHERE user_id = ? ORDER BY position", userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var interests []models.Interest
	for rows.Next() {
		var i models.Interest
		rows.Scan(&i.ID, &i.UserID, &i.Label, &i.Value, &i.Position)
		interests = append(interests, i)
	}
	return interests
}

func (g *Generator) loadVoteHistory(userID int64) []voteRecord {
	rows, err := g.db.Query(
		`SELECT di.headline, di.category, v.value
		 FROM votes v JOIN digest_items di ON di.id = v.digest_item_id
		 WHERE v.user_id = ? ORDER BY v.created_at DESC LIMIT 50`,
		userID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var votes []voteRecord
	for rows.Next() {
		var v voteRecord
		rows.Scan(&v.Headline, &v.Category, &v.Value)
		votes = append(votes, v)
	}
	return votes
}

func (g *Generator) loadSections(userID int64) []models.CustomSection {
	rows, err := g.db.Query("SELECT id, user_id, title, prompt, position FROM custom_sections WHERE user_id = ? AND enabled = 1 ORDER BY position", userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var sections []models.CustomSection
	for rows.Next() {
		var s models.CustomSection
		rows.Scan(&s.ID, &s.UserID, &s.Title, &s.Prompt, &s.Position)
		sections = append(sections, s)
	}
	return sections
}

func (g *Generator) findArticleID(userID int64, guid string) sql.NullInt64 {
	var id int64
	err := g.db.QueryRow("SELECT id FROM articles WHERE user_id = ? AND guid = ?", userID, guid).Scan(&id)
	if err != nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: id, Valid: true}
}

func (g *Generator) countFeeds(userID int64) int {
	var count int
	g.db.QueryRow("SELECT COUNT(*) FROM sources WHERE user_id = ? AND enabled = 1", userID).Scan(&count)
	return count
}
