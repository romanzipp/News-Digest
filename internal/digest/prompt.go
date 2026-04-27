package digest

import (
	"fmt"
	"strings"

	"git.romanzipp.net/romanzipp/news/internal/models"
)

func buildSystemPrompt(interests []models.Interest, votes []voteRecord, sections []models.CustomSection, categories []string) string {
	var b strings.Builder

	b.WriteString(`You are a news editor curating a personal daily digest. Your job is to:
1. Select the most relevant articles based on the user's interests
2. Rewrite headlines to be concise and informative. Never mention the source name in the headline — the source is shown separately
3. Write a one-paragraph TL;DR summary for each article
4. Extract 2-3 key bullet points
5. Assign a category from the provided list
6. Assign a priority score (1-10, 10 = highest)
7. Assign importance level (high/medium/low)
8. Estimate read time in minutes
9. Preserve the original language of the article for the summary
10. Order articles by relevance to the user's interests
11. DEDUPLICATE: If multiple articles cover the same story/event, pick the SINGLE best one and discard the rest. Never include multiple articles about the same topic in the main items. Merge key details from duplicates into the selected article's summary and bullets.

`)

	b.WriteString("## Categories\n")
	b.WriteString("Assign one of: " + strings.Join(categories, ", ") + "\n\n")

	if len(interests) > 0 {
		b.WriteString("## User Interests\n")
		for _, i := range interests {
			b.WriteString(fmt.Sprintf("- %s: %s\n", i.Label, i.Value))
		}
		b.WriteString("\n")
	}

	if len(votes) > 0 {
		b.WriteString("## Recent Feedback\nThe user has voted on past articles. Use this to calibrate relevance:\n")
		for _, v := range votes {
			direction := "UPVOTED (wants more like this)"
			if v.Value < 0 {
				direction = "DOWNVOTED (wants less like this)"
			}
			b.WriteString(fmt.Sprintf("- [%s] %s — %s\n", v.Category, v.Headline, direction))
		}
		b.WriteString("\n")
	}

	if len(sections) > 0 {
		b.WriteString("## Custom Sections\nThe user has requested custom sections. For each section, select relevant articles:\n")
		for _, s := range sections {
			b.WriteString(fmt.Sprintf("- Section ID %d (%s): %s\n", s.ID, s.Title, s.Prompt))
		}
		b.WriteString("\n")
	}

	b.WriteString(`## Output
Respond as JSON with keys: "items" (array of articles), "sections" (array of section results), "meta" (object with articles_reviewed, articles_surfaced, estimated_read_minutes).
Each item: article_guid, headline, tldr, bullets[], category, priority (1-10), importance (high/medium/low), read_time, language, source_name, source_url, image_url.
Each section: section_id, items[] with article_guid, headline, severity (high/med/low), indicator, published_at, source_name, source_url, language.

## Instructions
Select at most 20 articles for the main items. Order by priority descending.
Only include categories that naturally fit the available articles. Do NOT force articles into categories just to have coverage — quality over breadth. If a category has relevant articles, aim for at least 3.
Never fabricate or embellish information. If an article's content is short or sparse, keep the summary and bullets short too — only reflect what is actually in the source material.

For section items:
- "headline" is a single short summary line (no separate description)
- "severity" must be one of: "high", "med", "low"
- "indicator" is a short monospaced identifier relevant to the section context (e.g. CVE number, stock ticker, incident ID, country code). Keep it very short. Leave empty if none applies.
- "published_at" is the ISO 8601 timestamp of the original article
`)

	return b.String()
}

func buildArticlePrompt(articles []models.Article) string {
	var b strings.Builder
	b.WriteString("Here are the articles to review:\n\n")

	for i, a := range articles {
		content := a.Content
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}

		b.WriteString(fmt.Sprintf("--- Article #%d ---\n", i+1))
		b.WriteString(fmt.Sprintf("GUID: %s\n", a.GUID))
		b.WriteString(fmt.Sprintf("Title: %s\n", a.Title))
		if a.Author != "" {
			b.WriteString(fmt.Sprintf("Source: %s\n", a.Author))
		}
		b.WriteString(fmt.Sprintf("URL: %s\n", a.URL))
		if a.PublishedAt.Valid {
			b.WriteString(fmt.Sprintf("Published: %s\n", a.PublishedAt.Time.Format("2006-01-02T15:04:05Z")))
		}
		if a.Language != "" {
			b.WriteString(fmt.Sprintf("Language: %s\n", a.Language))
		}
		if a.ImageURL != "" {
			b.WriteString(fmt.Sprintf("Image: %s\n", a.ImageURL))
		}
		if content != "" {
			b.WriteString(fmt.Sprintf("Content: %s\n", content))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func splitIntoBatches(articles []models.Article, maxTokens int, systemPromptTokens int) [][]models.Article {
	available := maxTokens - systemPromptTokens - 1000
	if available <= 0 {
		available = maxTokens / 2
	}

	var batches [][]models.Article
	var current []models.Article
	currentTokens := 0

	for _, a := range articles {
		articleText := a.Title + " " + a.Content
		tokens := estimateTokens(articleText)
		if tokens > 2000 {
			tokens = 2000
		}

		if currentTokens+tokens > available && len(current) > 0 {
			batches = append(batches, current)
			current = nil
			currentTokens = 0
		}

		current = append(current, a)
		currentTokens += tokens
	}

	if len(current) > 0 {
		batches = append(batches, current)
	}

	return batches
}

type voteRecord struct {
	Headline string
	Category string
	Value    int
}
