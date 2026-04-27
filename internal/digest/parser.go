package digest

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type DigestResponse struct {
	Items    []DigestItemResponse `json:"items"`
	Sections []SectionResponse    `json:"sections"`
	Trending []TrendingResponse   `json:"trending"`
	Meta     MetaResponse         `json:"meta"`
}

type DigestItemResponse struct {
	ArticleGUID string   `json:"article_guid"`
	Headline    string   `json:"headline"`
	TLDR        string   `json:"tldr"`
	Bullets     []string `json:"bullets"`
	Category    string   `json:"category"`
	Priority    int      `json:"priority"`
	Importance  string   `json:"importance"`
	ReadTime    int      `json:"read_time"`
	Language    string   `json:"language"`
	SourceName  string   `json:"source_name"`
	SourceURL   string   `json:"source_url"`
	ImageURL    string   `json:"image_url"`
}

type SectionResponse struct {
	SectionID int                  `json:"section_id"`
	Items     []SectionItemResponse `json:"items"`
}

type SectionItemResponse struct {
	ArticleGUID string `json:"article_guid"`
	Headline    string `json:"headline"`
	SourceName  string `json:"source_name"`
	SourceURL   string `json:"source_url"`
	Language    string `json:"language"`
	Severity    string `json:"severity"`
	Indicator   string `json:"indicator"`
	PublishedAt string `json:"published_at"`
}

type TrendingResponse struct {
	Topic string `json:"topic"`
	Count int    `json:"count"`
	Delta string `json:"delta"`
}

type MetaResponse struct {
	ArticlesReviewed    int `json:"articles_reviewed"`
	ArticlesSurfaced    int `json:"articles_surfaced"`
	EstimatedReadMinutes int `json:"estimated_read_minutes"`
}

func parseResponse(raw string) (*DigestResponse, error) {
	cleaned := extractJSON(raw)
	var resp DigestResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		tail := cleaned
		if len(tail) > 200 {
			tail = cleaned[len(cleaned)-200:]
		}
		log.Printf("ai json parse failed: len=%d tail=%s", len(cleaned), tail)
		return nil, fmt.Errorf("parse AI response: %w", err)
	}
	return &resp, nil
}

func extractJSON(s string) string {
	// Strip markdown code fences if present
	if strings.Contains(s, "```") {
		start := strings.Index(s, "```")
		// Skip the opening fence line
		afterFence := s[start+3:]
		if nl := strings.Index(afterFence, "\n"); nl >= 0 {
			afterFence = afterFence[nl+1:]
		}
		if end := strings.Index(afterFence, "```"); end >= 0 {
			return strings.TrimSpace(afterFence[:end])
		}
	}
	// Find first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func mergeResponses(responses []*DigestResponse) *DigestResponse {
	if len(responses) == 1 {
		return responses[0]
	}

	merged := &DigestResponse{}
	seenGUIDs := make(map[string]bool)
	seenTopics := make(map[string]bool)

	for _, r := range responses {
		for _, item := range r.Items {
			if !seenGUIDs[item.ArticleGUID] {
				merged.Items = append(merged.Items, item)
				seenGUIDs[item.ArticleGUID] = true
			}
		}
		for _, s := range r.Sections {
			merged.Sections = append(merged.Sections, s)
		}
		for _, t := range r.Trending {
			if !seenTopics[t.Topic] {
				merged.Trending = append(merged.Trending, t)
				seenTopics[t.Topic] = true
			}
		}
		merged.Meta.ArticlesReviewed += r.Meta.ArticlesReviewed
		merged.Meta.ArticlesSurfaced += r.Meta.ArticlesSurfaced
	}

	// Sort by priority descending
	for i := 0; i < len(merged.Items); i++ {
		for j := i + 1; j < len(merged.Items); j++ {
			if merged.Items[j].Priority > merged.Items[i].Priority {
				merged.Items[i], merged.Items[j] = merged.Items[j], merged.Items[i]
			}
		}
	}

	if len(merged.Items) > 20 {
		merged.Items = merged.Items[:20]
	}

	merged.Meta.ArticlesSurfaced = len(merged.Items)
	totalRead := 0
	for _, item := range merged.Items {
		totalRead += item.ReadTime
	}
	merged.Meta.EstimatedReadMinutes = totalRead

	return merged
}
