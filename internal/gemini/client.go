package gemini

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Client represents a Gemini API client
type Client struct {
	apiKey string
	model  string
}

// NewClient creates a new Gemini API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  "gemini-3-flash-preview", // Using the latest available model
	}
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
	// gemini-3-flash-preview is a thinking model: internal reasoning tokens
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

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate translation: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("no translation generated")
	}

	if resp.Candidates[0].FinishReason == genai.FinishReasonMaxTokens {
		return "", errors.New("translation truncated: hit MaxOutputTokens (increase the limit or split the content)")
	}

	// Extract the translated text from the response
	translatedText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	return translatedText, nil
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
	// gemini-2.5-flash is a thinking model: internal reasoning tokens count
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

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate translation: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("no translation generated")
	}

	if resp.Candidates[0].FinishReason == genai.FinishReasonMaxTokens {
		return "", errors.New("title translation truncated: hit MaxOutputTokens (thinking model consumed the token budget)")
	}

	translatedTitle := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	return translatedTitle, nil
}
