package gemini

import (
	"context"
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
		model:  "gemini-2.0-flash-lite", // Using the latest available model
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
	model.SetMaxOutputTokens(8192)

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
		return "", fmt.Errorf("no translation generated")
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
	model.SetMaxOutputTokens(256)

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
		return "", fmt.Errorf("no translation generated")
	}

	translatedTitle := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	return translatedTitle, nil
}
