package gemini

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	// DefaultTextModel is the default Gemini model used for text translation and verification.
	DefaultTextModel = "gemini-4-neon"
	// DefaultImageModel is the default Gemini model used for translating images.
	DefaultImageModel = "gemini-3-pro-image"
	// DefaultImageSize is the default output resolution for translated images.
	DefaultImageSize = "2K"
)

// Client represents a Gemini API client
type Client struct {
	apiKey     string
	model      string
	imageModel string
	imageSize  string
}

// NewClient creates a new Gemini API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      DefaultTextModel,
		imageModel: DefaultImageModel,
		imageSize:  DefaultImageSize,
	}
}

// SetTextModel overrides the Gemini model used for text translation and image verification.
func (c *Client) SetTextModel(model string) {
	if model != "" {
		c.model = model
	}
}

// TextModel returns the configured Gemini model used for text translation and image verification.
func (c *Client) TextModel() string {
	return c.model
}

// SetImageModel overrides the Gemini model used for image generation.
func (c *Client) SetImageModel(model string) {
	if model != "" {
		c.imageModel = model
	}
}

// ImageModel returns the configured Gemini model used for image generation.
func (c *Client) ImageModel() string {
	return c.imageModel
}

// SetImageSize overrides the output image size ("1K", "2K", or "4K").
// Returns an error and leaves the current size unchanged if the value is invalid.
func (c *Client) SetImageSize(size string) error {
	normalized := strings.ToUpper(strings.TrimSpace(size))
	switch normalized {
	case "1K", "2K", "4K":
		c.imageSize = normalized
		return nil
	default:
		return fmt.Errorf("unsupported image size %q (allowed: 1K, 2K, 4K)", size)
	}
}

// ImageSize returns the configured output image size.
func (c *Client) ImageSize() string {
	if c.imageSize == "" {
		return DefaultImageSize
	}
	return c.imageSize
}

// retryOnceOnMaxTokens runs attempt and, if it fails with retryable == true
// (e.g., hit MaxOutputTokens or returned empty content), retries it once.
func retryOnceOnMaxTokens(attempt func() (result string, retryable bool, err error)) (string, error) {
	result, retryable, err := attempt()
	if err == nil {
		return result, nil
	}
	if !retryable {
		return "", err
	}
	fmt.Println("⚠️ hit MaxOutputTokens, retrying once...")
	result, _, err = attempt()
	return result, err
}

func generateTranslationOnce(ctx context.Context, model *genai.GenerativeModel, prompt, maxTokensErrMsg string) (string, bool, error) {
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", false, fmt.Errorf("failed to generate translation: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", true, errors.New("no translation generated")
	}

	cand := resp.Candidates[0]
	if cand.FinishReason == genai.FinishReasonMaxTokens {
		return "", true, errors.New(maxTokensErrMsg)
	}

	if cand.Content == nil || len(cand.Content.Parts) == 0 {
		return "", true, errors.New("no translation generated")
	}

	translatedText := fmt.Sprintf("%v", cand.Content.Parts[0])
	if strings.TrimSpace(translatedText) == "" {
		return "", true, errors.New("no translation generated")
	}

	return translatedText, false, nil
}

// Translate translates text to English using Gemini
func (c *Client) Translate(ctx context.Context, text string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(c.model)

	// Configure the model for translation
	model.SetTemperature(0.3) // Lower temperature for more consistent translations
	model.SetTopP(0.95)
	model.SetTopK(40)
	// gemini-4-neon is a thinking model: internal reasoning tokens
	// count against MaxOutputTokens. Long articles easily exhaust a small
	// budget, so allow generous headroom. Billing is on tokens actually used,
	// so a high ceiling has no cost downside.
	model.SetMaxOutputTokens(65536)

	prompt := fmt.Sprintf(`Translate the following text to English.
If the text is already in English, return it as-is.
Preserve all markdown formatting, code blocks, links, and technical terms.
Only translate the readable text content, not code or technical identifiers.

Text to translate:
%s

Translated text:`, text)

	return retryOnceOnMaxTokens(func() (string, bool, error) {
		return generateTranslationOnce(ctx, model, prompt, "translation truncated: hit MaxOutputTokens (increase the limit or split the content)")
	})
}

// TranslateTitle translates a title to English
func (c *Client) TranslateTitle(ctx context.Context, title string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(c.model)

	model.SetTemperature(0.3)
	model.SetTopP(0.95)
	model.SetTopK(40)
	// gemini-4-neon is a thinking model: internal reasoning tokens count
	// against MaxOutputTokens. A tight limit (e.g. 256) gets consumed by
	// thinking and truncates the title mid-output, so allow generous headroom.
	// Billing is on tokens actually used, so a high ceiling has no cost downside.
	model.SetMaxOutputTokens(4096)

	prompt := fmt.Sprintf(`Translate this article title to English.
If it's already in English, return it as-is.
Keep it concise and natural.
Only return the translated title, nothing else.

Title: %s

Translated title:`, title)

	return retryOnceOnMaxTokens(func() (string, bool, error) {
		return generateTranslationOnce(ctx, model, prompt, "title translation truncated: hit MaxOutputTokens (thinking model consumed the token budget)")
	})
}
