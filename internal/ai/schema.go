package ai

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema"
)

type DigestSchema struct {
	Articles []DigestItemSchema `json:"articles" jsonschema_description:"Selected articles for the digest, ordered by priority descending"`
	Sections []SectionSchema    `json:"sections" jsonschema_description:"Custom section results"`
	Meta     MetaSchema         `json:"meta" jsonschema_description:"Metadata about the digest generation"`
}

type DigestItemSchema struct {
	ArticleGUID string   `json:"article_guid" jsonschema_description:"The GUID from the input article"`
	Headline    string   `json:"headline" jsonschema_description:"Rewritten concise headline"`
	TLDR        string   `json:"tldr" jsonschema_description:"One paragraph summary in the article original language"`
	Bullets     []string `json:"bullets" jsonschema_description:"2-3 key bullet points"`
	Category    string   `json:"category" jsonschema_description:"Category from the provided list"`
	Priority    int      `json:"priority" jsonschema_description:"Priority score 1-10, 10 highest"`
	Importance  string   `json:"importance" jsonschema_description:"high, medium, or low"`
	ReadTime    int      `json:"read_time" jsonschema_description:"Estimated read time in minutes"`
	Language    string   `json:"language" jsonschema_description:"ISO language code of the article"`
	SourceName  string   `json:"source_name" jsonschema_description:"Name of the source"`
	SourceURL   string   `json:"source_url" jsonschema_description:"URL to the original article"`
	ImageURL    string   `json:"image_url" jsonschema_description:"Image URL or empty string"`
}

type SectionSchema struct {
	SectionID int                 `json:"section_id" jsonschema_description:"The section ID from the input"`
	Items     []SectionItemSchema `json:"items" jsonschema_description:"Items for this section"`
}

type SectionItemSchema struct {
	ArticleGUID string `json:"article_guid" jsonschema_description:"The GUID from the input article"`
	Headline    string `json:"headline" jsonschema_description:"Short single-line summary"`
	SourceName  string `json:"source_name" jsonschema_description:"Name of the source"`
	SourceURL   string `json:"source_url" jsonschema_description:"URL to the original article"`
	Language    string `json:"language" jsonschema_description:"ISO language code"`
	Severity    string `json:"severity" jsonschema_description:"high, med, or low"`
	Indicator   string `json:"indicator" jsonschema_description:"Short identifier like CVE number or stock ticker"`
	PublishedAt string `json:"published_at" jsonschema_description:"ISO 8601 timestamp"`
}

type MetaSchema struct {
	ArticlesReviewed     int `json:"articles_reviewed" jsonschema_description:"Total articles reviewed"`
	ArticlesSurfaced     int `json:"articles_surfaced" jsonschema_description:"Articles selected for digest"`
	EstimatedReadMinutes int `json:"estimated_read_minutes" jsonschema_description:"Total estimated read time"`
}

func generateDigestSchema() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	schema := reflector.Reflect(DigestSchema{})
	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}
}
