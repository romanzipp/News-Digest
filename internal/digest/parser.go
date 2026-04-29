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
		// The model sometimes returns array fields as stringified JSON.
		// Try to unwrap and re-parse.
		if fixed, ok := tryUnwrapStringFields(raw); ok {
			if err2 := json.Unmarshal([]byte(fixed), &resp); err2 == nil {
				log.Printf("ai json: recovered from stringified fields")
				return &resp, nil
			}
		}
		tail := raw
		if len(tail) > 200 {
			tail = raw[len(raw)-200:]
		}
		log.Printf("ai json parse failed: len=%d tail=%s", len(raw), tail)
		return nil, fmt.Errorf("parse AI response: %w", err)
	}
	return &resp, nil
}

// tryUnwrapStringFields detects fields that are JSON strings containing
// arrays/objects and unwraps them into their actual JSON types.
func tryUnwrapStringFields(raw string) (string, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &obj) != nil {
		return "", false
	}
	changed := false
	for key, val := range obj {
		var s string
		if json.Unmarshal(val, &s) == nil && len(s) > 0 && (s[0] == '[' || s[0] == '{') {
			if json.Valid([]byte(s)) {
				obj[key] = json.RawMessage(s)
				changed = true
			}
		}
	}
	if !changed {
		return "", false
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return "", false
	}
	return string(out), true
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
