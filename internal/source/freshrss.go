package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.romanzipp.net/romanzipp/news/internal/models"
)

type FreshRSSConfig struct {
	Username    string `json:"username"`
	APIPassword string `json:"api_password"`
}

type FreshRSSProvider struct{}

func (p *FreshRSSProvider) Type() string { return "freshrss" }

func (p *FreshRSSProvider) Validate(cfg json.RawMessage) error {
	var c FreshRSSConfig
	if err := json.Unmarshal(cfg, &c); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if c.Username == "" || c.APIPassword == "" {
		return fmt.Errorf("username and api_password required")
	}
	return nil
}

func (p *FreshRSSProvider) Fetch(ctx context.Context, src models.Source) ([]models.Article, error) {
	var cfg FreshRSSConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, err
	}

	client := &freshRSSClient{
		baseURL:     strings.TrimRight(src.URL, "/"),
		username:    cfg.Username,
		apiPassword: cfg.APIPassword,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}

	if err := client.login(ctx); err != nil {
		return nil, fmt.Errorf("freshrss login: %w", err)
	}

	since := time.Now().Add(-24 * time.Hour)
	if src.LastFetchedAt.Valid {
		since = src.LastFetchedAt.Time
	}

	return client.fetchItems(ctx, since)
}

func TestFreshRSSConnection(baseURL, username, apiPassword string) error {
	client := &freshRSSClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		username:    username,
		apiPassword: apiPassword,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
	return client.login(context.Background())
}

type freshRSSClient struct {
	baseURL     string
	username    string
	apiPassword string
	authToken   string
	httpClient  *http.Client
}

func (c *freshRSSClient) login(ctx context.Context) error {
	data := url.Values{
		"Email":  {c.username},
		"Passwd": {c.apiPassword},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/greader.php/accounts/ClientLogin", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("login failed: %s", resp.Status)
	}

	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "Auth=") {
			c.authToken = strings.TrimPrefix(line, "Auth=")
			return nil
		}
	}
	return fmt.Errorf("no auth token in response")
}

func (c *freshRSSClient) fetchItems(ctx context.Context, since time.Time) ([]models.Article, error) {
	var allArticles []models.Article
	continuation := ""

	for {
		u := fmt.Sprintf("%s/api/greader.php/reader/api/0/stream/contents/user/-/state/com.google/reading-list?ot=%d&n=200&output=json",
			c.baseURL, since.Unix())
		if continuation != "" {
			u += "&c=" + continuation
		}

		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "GoogleLogin auth="+c.authToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		var result struct {
			Items []struct {
				ID      string `json:"id"`
				Title   string `json:"title"`
				Summary struct {
					Content string `json:"content"`
				} `json:"summary"`
				Alternate []struct {
					Href string `json:"href"`
				} `json:"alternate"`
				Author    string `json:"author"`
				Published int64  `json:"published"`
				Origin    struct {
					Title string `json:"title"`
				} `json:"origin"`
			} `json:"items"`
			Continuation string `json:"continuation"`
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse items: %w", err)
		}

		for _, item := range result.Items {
			link := ""
			if len(item.Alternate) > 0 {
				link = item.Alternate[0].Href
			}

			guid := item.ID
			if guid == "" {
				guid = link
			}

			a := models.Article{
				GUID:        guid,
				Title:       item.Title,
				URL:         link,
				Content:     item.Summary.Content,
				Author:      item.Author,
				PublishedAt: sql.NullTime{Time: time.Unix(item.Published, 0), Valid: true},
			}
			allArticles = append(allArticles, a)
		}

		if result.Continuation == "" || len(result.Items) == 0 {
			break
		}
		continuation = result.Continuation
	}

	return allArticles, nil
}
