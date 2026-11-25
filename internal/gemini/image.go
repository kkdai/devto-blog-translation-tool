package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

const (
	detectChinesePrompt  = `Does this image contain any Chinese characters (Traditional or Simplified Chinese text)? Respond ONLY with a JSON object in the exact format {"has_chinese_text": true} or {"has_chinese_text": false}. Do not include any other text.`
	translateImagePrompt = `Recreate this image exactly, keeping layout, style, colors, icons and non-text content identical, but translate all Chinese text into natural English. Keep code, URLs, and proper nouns as-is.`
)

type chineseDetectionResult struct {
	HasChineseText bool `json:"has_chinese_text"`
}

// DetectChineseText uses the Gemini vision model to determine whether an image contains Chinese text.
// If the model response cannot be parsed as JSON, it returns false without an error.
func (c *Client) DetectChineseText(ctx context.Context, imageData []byte, mimeType string) (bool, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  c.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return false, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromBytes(imageData, mimeType),
			genai.NewPartFromText(detectChinesePrompt),
		}, genai.RoleUser),
	}

	var temp float32 = 0
	config := &genai.GenerateContentConfig{
		Temperature:      &temp,
		ResponseMIMEType: "application/json",
	}

	resp, err := client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return false, fmt.Errorf("failed to detect Chinese text: %w", err)
	}

	return parseChineseDetectionJSON(resp.Text()), nil
}

// parseChineseDetectionJSON extracts has_chinese_text from a JSON string.
// Returns false if parsing fails.
func parseChineseDetectionJSON(raw string) bool {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimPrefix(cleaned, "```JSON")
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	var result chineseDetectionResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return false
	}
	return result.HasChineseText
}

// TranslateImage uses the configured Gemini image model to recreate an image with Chinese text translated to English.
func (c *Client) TranslateImage(ctx context.Context, imageData []byte, mimeType string) ([]byte, string, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  c.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Gemini client: %w", err)
	}

	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromBytes(imageData, mimeType),
			genai.NewPartFromText(translateImagePrompt),
		}, genai.RoleUser),
	}

	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{"IMAGE", "TEXT"},
	}

	resp, err := client.Models.GenerateContent(ctx, c.imageModel, contents, config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate translated image: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, "", errors.New("no image candidate returned by Gemini")
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if part == nil || part.Thought {
			continue
		}
		if part.InlineData != nil && len(part.InlineData.Data) > 0 {
			outMIME := part.InlineData.MIMEType
			if outMIME == "" {
				outMIME = "image/png"
			}
			return part.InlineData.Data, outMIME, nil
		}
	}

	return nil, "", errors.New("no inline image data found in Gemini response")
}
