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

type TrendingTopic struct {
	Topic string
	Count int
	Delta string
}

var DefaultCategories = []string{
	"Technology",
	"Economy",
	"Health",
	"Science",
	"World",
	"Sports",
	"Entertainment",
	"Environment",
	"Politics",
	"Business",
	"Finance",
	"AI & Machine Learning",
	"Cybersecurity",
	"Space",
	"Energy",
	"Education",
	"Culture",
	"Travel",
	"Food",
	"Automotive",
	"Real Estate",
	"Cryptocurrency",
	"Gaming",
	"Social Media",
	"Privacy",
	"Climate",
	"Defense",
	"Legal",
	"Startups",
	"Open Source",
	"Infrastructure",
	"Biotech",
	"Robotics",
	"Telecommunications",
	"Supply Chain",
	"Agriculture",
	"Aviation",
	"Maritime",
	"Opinion",
	"Longform",
}

var CategoryColors = map[string]string{
	"Technology":          "#2e4f7a",
	"Economy":             "#7a3d2e",
	"Health":              "#2e7a55",
	"Science":             "#2e6a7a",
	"World":               "#7a6b2e",
	"Sports":              "#7a2e5c",
	"Entertainment":       "#7a5c2e",
	"Environment":         "#3a7a2e",
	"Politics":            "#5a3d7a",
	"Business":            "#6a4a2e",
	"Finance":             "#8a4a2e",
	"AI & Machine Learning": "#2e4f7a",
	"Cybersecurity":       "#7a2e3d",
	"Space":               "#3d2e7a",
	"Energy":              "#5a7a2e",
	"Education":           "#2e5a7a",
	"Culture":             "#7a4a5c",
	"Travel":              "#4a7a6a",
	"Food":                "#7a6a3a",
	"Automotive":          "#4a5a7a",
	"Real Estate":         "#6a5a4a",
	"Cryptocurrency":      "#7a5a2e",
	"Gaming":              "#5a2e7a",
	"Social Media":        "#2e7a7a",
	"Privacy":             "#6a2e5a",
	"Climate":             "#2e7a4a",
	"Defense":             "#5a5a6a",
	"Legal":               "#5a4a5a",
	"Startups":            "#3a6a5a",
	"Open Source":         "#4a6a3a",
	"Infrastructure":      "#5a6a4a",
	"Biotech":             "#3a7a5a",
	"Robotics":            "#4a4a7a",
	"Telecommunications":  "#5a5a4a",
	"Supply Chain":        "#6a5a3a",
	"Agriculture":         "#4a7a3a",
	"Aviation":            "#3a5a7a",
	"Maritime":            "#2e6a6a",
	"Opinion":             "#6a4a6a",
	"Longform":            "#4a4a5a",
}
