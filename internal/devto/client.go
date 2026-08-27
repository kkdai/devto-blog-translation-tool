package devto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://dev.to/api"
)

// Client represents a Dev.to API client
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// Article represents a Dev.to article
type Article struct {
	ID           int        `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	BodyMarkdown string     `json:"body_markdown"`
	Published    bool       `json:"published"`
	Slug         string     `json:"slug"`
	URL          string     `json:"url"`
	CreatedAt    time.Time  `json:"created_at"`
	PublishedAt  *time.Time `json:"published_at"`
	Tags         []string   `json:"tags"`
}

// NewClient creates a new Dev.to API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetDraftArticles fetches all unpublished (draft) articles with pagination
func (c *Client) GetDraftArticles(ctx context.Context) ([]Article, error) {
	var allArticles []Article
	page := 1
	perPage := 30 // Dev.to API default/max per page

	for {
		url := fmt.Sprintf("%s/articles/me/unpublished?page=%d&per_page=%d", baseURL, page, perPage)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set required headers
		req.Header.Set("api-key", c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var articles []Article
		if err := json.Unmarshal(body, &articles); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// No more articles, break the loop
		if len(articles) == 0 {
			break
		}

		allArticles = append(allArticles, articles...)

		fmt.Printf("  Fetched page %d (%d articles, total: %d)\n", page, len(articles), len(allArticles))

		// If we got fewer articles than perPage, we've reached the end
		if len(articles) < perPage {
			break
		}

		page++

		// Add a small delay to respect rate limits
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return allArticles, nil
}

// GetAllArticles fetches all articles (published and unpublished)
func (c *Client) GetAllArticles(ctx context.Context) ([]Article, error) {
	url := fmt.Sprintf("%s/articles/me/all", baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var articles []Article
	if err := json.Unmarshal(body, &articles); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return articles, nil
}

// UpdateArticlePayload represents the payload for updating an article
type UpdateArticlePayload struct {
	Article UpdateArticleFields `json:"article"`
}

// UpdateArticleFields represents the fields that can be updated
type UpdateArticleFields struct {
	Title        string   `json:"title,omitempty"`
	BodyMarkdown string   `json:"body_markdown,omitempty"`
	Published    *bool    `json:"published,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

// PublishArticle updates an existing draft article with translated content and publishes it
// This uses PUT /articles/:id to UPDATE the existing article, not create a new one
func (c *Client) PublishArticle(ctx context.Context, articleID int, title, bodyMarkdown string, tags []string) error {
	url := fmt.Sprintf("%s/articles/%d", baseURL, articleID)

	published := true
	payload := UpdateArticlePayload{
		Article: UpdateArticleFields{
			Title:        title,
			BodyMarkdown: bodyMarkdown,
			Published:    &published,
			Tags:         tags,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
