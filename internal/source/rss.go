package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/mmcdole/gofeed"
	"git.romanzipp.net/romanzipp/news/internal/models"
)

type RSSProvider struct{}

func (p *RSSProvider) Type() string { return "rss" }

func (p *RSSProvider) Validate(_ json.RawMessage) error { return nil }

func (p *RSSProvider) Fetch(ctx context.Context, src models.Source) ([]models.Article, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(src.URL, ctx)
	if err != nil {
		return nil, err
	}

	var articles []models.Article
	for _, item := range feed.Items {
		a := models.Article{
			GUID:    itemGUID(item),
			Title:   item.Title,
			URL:     item.Link,
			Content: itemContent(item),
			Author:  itemAuthor(item),
		}

		if item.PublishedParsed != nil {
			a.PublishedAt = sql.NullTime{Time: *item.PublishedParsed, Valid: true}
		} else if item.UpdatedParsed != nil {
			a.PublishedAt = sql.NullTime{Time: *item.UpdatedParsed, Valid: true}
		} else {
			a.PublishedAt = sql.NullTime{Time: time.Now(), Valid: true}
		}

		a.ImageURL = itemImage(item)
		a.Language = feed.Language

		articles = append(articles, a)
	}

	return articles, nil
}

func itemGUID(item *gofeed.Item) string {
	if item.GUID != "" {
		return item.GUID
	}
	return item.Link
}

func itemContent(item *gofeed.Item) string {
	if item.Content != "" {
		return item.Content
	}
	return item.Description
}

func itemAuthor(item *gofeed.Item) string {
	if item.Author != nil {
		return item.Author.Name
	}
	return ""
}

func itemImage(item *gofeed.Item) string {
	if item.Image != nil && item.Image.URL != "" {
		return item.Image.URL
	}
	for _, enc := range item.Enclosures {
		if len(enc.Type) >= 5 && enc.Type[:5] == "image" {
			return enc.URL
		}
	}
	return ""
}
