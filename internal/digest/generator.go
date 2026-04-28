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
	ai       ai.Provider
	registry *source.Registry
}

func NewGenerator(db *sql.DB, cfg *config.Config, aiClient ai.Provider, registry *source.Registry) *Generator {
	return &Generator{db: db, cfg: cfg, ai: aiClient, registry: registry}
}

func (g *Generator) CleanupStaleJobs() {
	res, _ := g.db.Exec(
		"UPDATE digest_jobs SET status = 'failed', step = 'Interrupted — server was restarted' WHERE status IN ('pending', 'fetching', 'generating')",
	)
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("cleaned up %d stale digest jobs", n)
	}
}

func (g *Generator) CreateJob(userID int64) (int64, error) {
	result, err := g.db.Exec(
		"INSERT INTO digest_jobs (user_id, status, step) VALUES (?, 'pending', 'Starting...')",
		userID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (g *Generator) updateJob(jobID int64, status, step string, currentBatch, totalBatches int) {
	g.db.Exec(
		"UPDATE digest_jobs SET status = ?, step = ?, current_batch = ?, total_batches = ?, updated_at = ? WHERE id = ?",
		status, step, currentBatch, totalBatches, time.Now(), jobID,
	)
}

func (g *Generator) finishJob(jobID int64, digestID int64) {
	g.db.Exec(
		"UPDATE digest_jobs SET status = 'complete', step = 'Done', digest_id = ?, updated_at = ? WHERE id = ?",
		digestID, time.Now(), jobID,
	)
}

func (g *Generator) failJob(jobID int64, err error) {
	g.db.Exec(
		"UPDATE digest_jobs SET status = 'failed', step = ?, updated_at = ? WHERE id = ?",
		err.Error(), time.Now(), jobID,
	)
}

type JobStatus struct {
	ID           int64
	UserID       int64
	Status       string
	Step         string
	CurrentBatch int
	TotalBatches int
	DigestID     sql.NullInt64
	Error        string
}

func (g *Generator) GetJob(jobID int64) (*JobStatus, error) {
	var j JobStatus
	err := g.db.QueryRow(
		"SELECT id, user_id, status, step, current_batch, total_batches, digest_id, error FROM digest_jobs WHERE id = ?",
		jobID,
	).Scan(&j.ID, &j.UserID, &j.Status, &j.Step, &j.CurrentBatch, &j.TotalBatches, &j.DigestID, &j.Error)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (g *Generator) ActiveJobForUser(userID int64) (*JobStatus, error) {
	var j JobStatus
	err := g.db.QueryRow(
		"SELECT id, user_id, status, step, current_batch, total_batches, digest_id, error FROM digest_jobs WHERE user_id = ? AND status IN ('pending', 'fetching', 'generating') ORDER BY id DESC LIMIT 1",
		userID,
	).Scan(&j.ID, &j.UserID, &j.Status, &j.Step, &j.CurrentBatch, &j.TotalBatches, &j.DigestID, &j.Error)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (g *Generator) GenerateAsync(jobID, userID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	g.updateJob(jobID, "fetching", "Fetching articles from feeds...", 0, 0)

	n, errCount, err := g.registry.FetchAllForUser(ctx, userID)
	if err != nil {
		log.Printf("fetch before digest for user %d: %v", userID, err)
	} else if errCount > 0 {
		log.Printf("fetched %d new articles for user %d (%d sources failed)", n, userID, errCount)
	} else {
		log.Printf("fetched %d new articles for user %d before digest", n, userID)
	}

	g.updateJob(jobID, "generating", "Loading articles...", 0, 0)

	articles, err := g.loadRecentArticles(userID)
	if err != nil {
		g.failJob(jobID, fmt.Errorf("load articles: %w", err))
		return
	}
	if len(articles) == 0 {
		g.failJob(jobID, fmt.Errorf("no articles to digest"))
		return
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
	totalBatches := len(batches)
	log.Printf("digest for user %d: %d articles in %d batches", userID, len(articles), totalBatches)

	g.updateJob(jobID, "generating", fmt.Sprintf("Processing batch 1/%d...", totalBatches), 0, totalBatches)

	var responses []*DigestResponse
	for i, batch := range batches {
		g.updateJob(jobID, "generating", fmt.Sprintf("Processing batch %d/%d...", i+1, totalBatches), i, totalBatches)

		articlePrompt := buildArticlePrompt(batch)

		raw, err := g.ai.Complete(ctx, systemPrompt, articlePrompt)
		if err != nil {
			g.failJob(jobID, fmt.Errorf("AI call batch %d: %w", i, err))
			return
		}

		resp, err := parseResponse(raw)
		if err != nil {
			g.failJob(jobID, fmt.Errorf("parse batch %d: %w", i, err))
			return
		}

		responses = append(responses, resp)
	}

	g.updateJob(jobID, "generating", "Saving digest...", totalBatches, totalBatches)

	merged := mergeResponses(responses)
	rawJSON, _ := json.Marshal(merged)

	feedCount := g.countFeeds(userID)

	digest, err := g.storeDigest(userID, 0, merged, feedCount, string(rawJSON))
	if err != nil {
		g.failJob(jobID, fmt.Errorf("store digest: %w", err))
		return
	}

	g.finishJob(jobID, digest.ID)
	log.Printf("digest job %d complete for user %d: digest %d", jobID, userID, digest.ID)
}

// GenerateForUser is used by the cron scheduler (synchronous)
func (g *Generator) GenerateForUser(ctx context.Context, userID int64, isAuto bool) (*models.Digest, error) {
	n, errCount, err := g.registry.FetchAllForUser(ctx, userID)
	if err != nil {
		log.Printf("fetch before digest for user %d: %v", userID, err)
	} else if errCount > 0 {
		log.Printf("fetched %d new articles for user %d (%d sources failed)", n, userID, errCount)
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
			return nil, fmt.Errorf("parse batch %d: %w", i, err)
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

			tx.Exec(
				`INSERT INTO section_items (digest_id, section_id, article_id, position, headline, source_name, source_url, language, severity, indicator, published_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				digestID, sec.SectionID, articleID, j, item.Headline,
				item.SourceName, item.SourceURL, item.Language, item.Severity, item.Indicator, item.PublishedAt,
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
	since := time.Now().Add(-24 * time.Hour)
	rows, err := g.db.Query(
		`SELECT a.id, a.user_id, a.source_id, a.guid, a.title, a.url, a.content, a.author, a.image_url, a.language, a.published_at
		 FROM articles a
		 LEFT JOIN read_items ri ON ri.article_id = a.id AND ri.user_id = ?
		 WHERE a.user_id = ? AND COALESCE(a.published_at, a.fetched_at) > ? AND ri.id IS NULL
		 ORDER BY COALESCE(a.published_at, a.fetched_at) DESC`,
		userID, userID, since,
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
