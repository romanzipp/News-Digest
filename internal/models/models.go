package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type User struct {
	ID           int64
	Email        string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type Source struct {
	ID            int64
	UserID        int64
	Type          string
	Name          string
	URL           string
	Config        json.RawMessage
	Enabled       bool
	LastFetchedAt sql.NullTime
	CreatedAt     time.Time
}

type Article struct {
	ID          int64
	UserID      int64
	SourceID    sql.NullInt64
	GUID        string
	Title       string
	URL         string
	Content     string
	Author      string
	ImageURL    string
	Language    string
	PublishedAt sql.NullTime
	FetchedAt   time.Time
}

type Digest struct {
	ID               int64
	UserID           int64
	Date             string
	IsAuto           bool
	ArticlesReviewed int
	ArticlesSurfaced int
	FeedsCount       int
	GenerationModel  string
	GeneratedAt      time.Time
	RawResponse      string
}

type DigestItem struct {
	ID         int64
	DigestID   int64
	ArticleID  sql.NullInt64
	Position   int
	Priority   int
	Category   string
	Headline   string
	TLDR       string
	Bullets    []string
	SourceName string
	SourceURL  string
	ImageURL   string
	ReadTime   int
	Language   string
	Importance string
}

type CustomSection struct {
	ID        int64
	UserID    int64
	Title     string
	Prompt    string
	Position  int
	Enabled   bool
	CreatedAt time.Time
}

type SectionItem struct {
	ID         int64
	DigestID   int64
	SectionID  int64
	ArticleID  sql.NullInt64
	Position   int
	Headline   string
	TLDR       string
	Bullets    []string
	SourceName string
	SourceURL  string
	Language   string
}

type Interest struct {
	ID       int64
	UserID   int64
	Label    string
	Value    string
	Position int
}

type Vote struct {
	ID           int64
	UserID       int64
	DigestItemID int64
	Value        int
	CreatedAt    time.Time
}
