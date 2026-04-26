package digest

import (
	"encoding/json"
	"fmt"
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
	ArticleGUID string   `json:"article_guid"`
	Headline    string   `json:"headline"`
	TLDR        string   `json:"tldr"`
	Bullets     []string `json:"bullets"`
	SourceName  string   `json:"source_name"`
	SourceURL   string   `json:"source_url"`
	Language    string   `json:"language"`
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
	var resp DigestResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("parse AI response: %w", err)
	}
	return &resp, nil
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
