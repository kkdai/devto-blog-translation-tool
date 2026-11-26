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
	baseURL        = "https://dev.to/api"
	userAgent      = "devto-blog-translation-tool/1.0"
	defaultTimeout = 90 * time.Second
	maxRetries     = 3
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
			Timeout: defaultTimeout,
		},
	}
}

// doWithRetry executes an HTTP request builder with exponential backoff retries on transient errors or timeouts
func (c *Client) doWithRetry(ctx context.Context, buildReq func() (*http.Request, error)) (*http.Response, []byte, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := buildReq()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("api-key", c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/vnd.forem.api-v1+json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			if attempt < maxRetries {
				backoff := time.Duration(attempt*2) * time.Second
				fmt.Printf("  ⚠️ Request failed (attempt %d/%d): %v. Retrying in %v...\n", attempt, maxRetries, err, backoff)
				select {
				case <-ctx.Done():
					return nil, nil, ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}
			return nil, nil, lastErr
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", readErr)
			if attempt < maxRetries {
				backoff := time.Duration(attempt*2) * time.Second
				fmt.Printf("  ⚠️ Reading body failed (attempt %d/%d): %v. Retrying in %v...\n", attempt, maxRetries, readErr, backoff)
				select {
				case <-ctx.Done():
					return nil, nil, ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}
			return nil, nil, lastErr
		}

		// Retry on 429 Too Many Requests or 5xx server errors
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			if attempt < maxRetries {
				backoff := time.Duration(attempt*3) * time.Second
				fmt.Printf("  ⚠️ Server returned status %d (attempt %d/%d). Retrying in %v...\n", resp.StatusCode, attempt, maxRetries, backoff)
				select {
				case <-ctx.Done():
					return nil, nil, ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}
			return resp, body, lastErr
		}

		if resp.StatusCode != http.StatusOK {
			return resp, body, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		return resp, body, nil
	}

	return nil, nil, lastErr
}

// GetDraftArticles fetches all unpublished (draft) articles with pagination
func (c *Client) GetDraftArticles(ctx context.Context) ([]Article, error) {
	var allArticles []Article
	page := 1
	perPage := 30 // Dev.to API default/max per page

	for {
		url := fmt.Sprintf("%s/articles/me/unpublished?page=%d&per_page=%d", baseURL, page, perPage)

		_, body, err := c.doWithRetry(ctx, func() (*http.Request, error) {
			return http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		})
		if err != nil {
			return nil, err
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

		// Add a delay to respect rate limits
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}

	return allArticles, nil
}

// GetAllArticles fetches all articles (published and unpublished)
func (c *Client) GetAllArticles(ctx context.Context) ([]Article, error) {
	url := fmt.Sprintf("%s/articles/me/all", baseURL)

	_, body, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	})
	if err != nil {
		return nil, err
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

	_, _, err = c.doWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(jsonData))
	})
	if err != nil {
		return err
	}

	return nil
}
