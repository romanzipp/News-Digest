package digest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/roman-zipp/news/internal/models"
)

func buildSystemPrompt(interests []models.Interest, votes []voteRecord, sections []models.CustomSection, categories []string) string {
	var b strings.Builder

	b.WriteString(`You are a news editor curating a personal daily digest. Your job is to:
1. Select the most relevant articles based on the user's interests
2. Rewrite headlines to be concise and informative
3. Write a one-paragraph TL;DR summary for each article
4. Extract 2-3 key bullet points
5. Assign a category from the provided list
6. Assign a priority score (1-10, 10 = highest)
7. Assign importance level (high/medium/low)
8. Estimate read time in minutes
9. Preserve the original language of the article for the summary
10. Order articles by relevance to the user's interests

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

	b.WriteString(`## Output Format
Respond with valid JSON in this exact structure:
{
  "items": [
    {
      "article_guid": "the guid from the input",
      "headline": "rewritten headline",
      "tldr": "one paragraph summary in the article's original language",
      "bullets": ["key point 1", "key point 2", "key point 3"],
      "category": "one of the categories above",
      "priority": 9,
      "importance": "high",
      "read_time": 3,
      "language": "en",
      "source_name": "Source Name",
      "source_url": "https://...",
      "image_url": "https://... or empty string"
    }
  ],
  "sections": [
    {
      "section_id": 1,
      "items": [
        {
          "article_guid": "...",
          "headline": "...",
          "tldr": "...",
          "bullets": ["..."],
          "source_name": "...",
          "source_url": "...",
          "language": "..."
        }
      ]
    }
  ],
  "trending": [
    {"topic": "topic name", "count": 14, "delta": "+112%"}
  ],
  "meta": {
    "articles_reviewed": 98,
    "articles_surfaced": 12,
    "estimated_read_minutes": 14
  }
}

Select at most 20 articles for the main items. Order by priority descending.
For trending, identify 5-8 topics that appear across multiple articles.
`)

	return b.String()
}

type articleInput struct {
	ID        int    `json:"id"`
	GUID      string `json:"guid"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	URL       string `json:"url"`
	Published string `json:"published"`
	Language  string `json:"language"`
	ImageURL  string `json:"image_url"`
}

func buildArticlePrompt(articles []models.Article) string {
	inputs := make([]articleInput, len(articles))
	for i, a := range articles {
		published := ""
		if a.PublishedAt.Valid {
			published = a.PublishedAt.Time.Format("2006-01-02T15:04:05Z")
		}
		content := a.Content
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}
		inputs[i] = articleInput{
			ID:        i + 1,
			GUID:      a.GUID,
			Title:     a.Title,
			Content:   content,
			Source:     a.Author,
			URL:       a.URL,
			Published: published,
			Language:  a.Language,
			ImageURL:  a.ImageURL,
		}
	}

	data, _ := json.Marshal(inputs)
	return fmt.Sprintf("Here are the articles to review:\n\n%s", string(data))
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func splitIntoBatches(articles []models.Article, maxTokens int, systemPromptTokens int) [][]models.Article {
	available := maxTokens - systemPromptTokens - 1000 // reserve for output
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
