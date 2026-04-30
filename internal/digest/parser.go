package digest

import (
	"encoding/json"
	"fmt"
	"log"
)

type DigestResponse struct {
	Articles []DigestItemResponse `json:"articles"`
	Sections []SectionResponse    `json:"sections"`
	Meta     MetaResponse         `json:"meta"`
}

// UnmarshalJSON handles the case where the AI model returns array/object fields
// as stringified JSON (e.g. "articles": "[{...}]" instead of "articles": [{...}]).
func (r *DigestResponse) UnmarshalJSON(data []byte) error {
	// Try standard decode first.
	type plain DigestResponse
	if err := json.Unmarshal(data, (*plain)(r)); err == nil {
		return nil
	}

	// Fallback: decode into raw fields and unwrap any stringified values.
	var raw struct {
		Articles json.RawMessage `json:"articles"`
		Sections json.RawMessage `json:"sections"`
		Meta     json.RawMessage `json:"meta"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if err := unmarshalStringOrJSON(raw.Articles, &r.Articles); err != nil {
		return fmt.Errorf("articles: %w", err)
	}
	if err := unmarshalStringOrJSON(raw.Sections, &r.Sections); err != nil {
		return fmt.Errorf("sections: %w", err)
	}
	if err := unmarshalStringOrJSON(raw.Meta, &r.Meta); err != nil {
		return fmt.Errorf("meta: %w", err)
	}
	return nil
}

// unmarshalStringOrJSON tries to unmarshal raw JSON into dst. If the raw value
// is a JSON string containing valid JSON, it unwraps the string first.
func unmarshalStringOrJSON(raw json.RawMessage, dst any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	// Try direct decode.
	if err := json.Unmarshal(raw, dst); err == nil {
		return nil
	}
	// If it's a string, unwrap and retry.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return json.Unmarshal([]byte(s), dst)
	}
	// Return the original error.
	return json.Unmarshal(raw, dst)
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
	SectionID int                   `json:"section_id"`
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

type MetaResponse struct {
	ArticlesReviewed     int `json:"articles_reviewed"`
	ArticlesSurfaced     int `json:"articles_surfaced"`
	EstimatedReadMinutes int `json:"estimated_read_minutes"`
}

func parseResponse(raw string) (*DigestResponse, error) {
	var resp DigestResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		tail := raw
		if len(tail) > 200 {
			tail = raw[len(raw)-200:]
		}
		log.Printf("ai json parse failed: len=%d tail=%s", len(raw), tail)
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

	for _, r := range responses {
		for _, item := range r.Articles {
			if !seenGUIDs[item.ArticleGUID] {
				merged.Articles = append(merged.Articles, item)
				seenGUIDs[item.ArticleGUID] = true
			}
		}
		for _, s := range r.Sections {
			merged.Sections = append(merged.Sections, s)
		}
		merged.Meta.ArticlesReviewed += r.Meta.ArticlesReviewed
		merged.Meta.ArticlesSurfaced += r.Meta.ArticlesSurfaced
	}

	for i := 0; i < len(merged.Articles); i++ {
		for j := i + 1; j < len(merged.Articles); j++ {
			if merged.Articles[j].Priority > merged.Articles[i].Priority {
				merged.Articles[i], merged.Articles[j] = merged.Articles[j], merged.Articles[i]
			}
		}
	}

	if len(merged.Articles) > 20 {
		merged.Articles = merged.Articles[:20]
	}

	merged.Meta.ArticlesSurfaced = len(merged.Articles)
	totalRead := 0
	for _, item := range merged.Articles {
		totalRead += item.ReadTime
	}
	merged.Meta.EstimatedReadMinutes = totalRead

	return merged
}
