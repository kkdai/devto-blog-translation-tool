package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"

	"google.golang.org/genai"
)

const (
	detectChinesePrompt  = `Does this image contain any Chinese characters (Traditional or Simplified Chinese text)? Estimate the total number of Chinese characters visible in the image. Respond ONLY with a JSON object in the exact format {"has_chinese_text": true, "chinese_char_count": 12} or {"has_chinese_text": false, "chinese_char_count": 0}. Do not include any other text.`
	translateImagePrompt = `Recreate this image exactly, keeping layout, style, colors, icons and non-text content identical, but translate all Chinese text into natural English. Keep code, URLs, and proper nouns as-is.`
	verifyImagePrompt    = `The first image is the original image, and the second image is its English-translated version. Compare both images carefully and check two things:
1. Does the second image still contain any Chinese characters (Traditional or Simplified Chinese text)?
2. Does the second image preserve all major UI elements, panels, dialogs, chart/diagram sections, and information from the first image (with only the text language changed to English)? If any major element is missing, removed, cropped, or has noticeably different content, set preserves_content to false.
Respond ONLY with a JSON object in the exact format {"has_chinese_text": false, "preserves_content": true, "reason": "short string"}. Do not include any other text.`
)

// Detection holds the result of checking an image for Chinese text.
type Detection struct {
	HasChinese   bool
	ChineseChars int
}

// Verification holds the result of comparing an original image with its translated version.
type Verification struct {
	HasChineseText   bool
	PreservesContent bool
	Reason           string
}

type chineseDetectionResult struct {
	HasChineseText   bool `json:"has_chinese_text"`
	ChineseCharCount int  `json:"chinese_char_count"`
}

type verificationResult struct {
	HasChineseText   bool   `json:"has_chinese_text"`
	PreservesContent bool   `json:"preserves_content"`
	Reason           string `json:"reason"`
}

type aspectRatioOption struct {
	label string
	ratio float64
}

var supportedAspectRatios = []aspectRatioOption{
	{label: "1:1", ratio: 1.0 / 1.0},
	{label: "2:3", ratio: 2.0 / 3.0},
	{label: "3:2", ratio: 3.0 / 2.0},
	{label: "3:4", ratio: 3.0 / 4.0},
	{label: "4:3", ratio: 4.0 / 3.0},
	{label: "9:16", ratio: 9.0 / 16.0},
	{label: "16:9", ratio: 16.0 / 9.0},
	{label: "21:9", ratio: 21.0 / 9.0},
}

// closestAspectRatio returns the supported Gemini aspect ratio closest to width/height
// by minimizing |log(w/h) - log(option)|. Returns "" if width or height is non-positive.
func closestAspectRatio(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	target := math.Log(float64(width) / float64(height))
	bestLabel := supportedAspectRatios[0].label
	bestDiff := math.Inf(1)
	for _, opt := range supportedAspectRatios {
		diff := math.Abs(target - math.Log(opt.ratio))
		if diff < bestDiff {
			bestDiff = diff
			bestLabel = opt.label
		}
	}
	return bestLabel
}

func detectAspectRatio(imageData []byte) string {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return ""
	}
	return closestAspectRatio(cfg.Width, cfg.Height)
}

func (c *Client) newGenAIClient(ctx context.Context) (*genai.Client, error) {
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:      c.apiKey,
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{APIVersion: "v1beta"},
	})
}

// DetectChineseText uses the Gemini vision model to determine whether an image contains Chinese text
// and estimate how many Chinese characters appear in it.
// If the model response cannot be parsed as JSON, it returns Detection{HasChinese: false} without an error.
func (c *Client) DetectChineseText(ctx context.Context, imageData []byte, mimeType string) (Detection, error) {
	client, err := c.newGenAIClient(ctx)
	if err != nil {
		return Detection{}, fmt.Errorf("failed to create Gemini client: %w", err)
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
		return Detection{}, fmt.Errorf("failed to detect Chinese text: %w", err)
	}

	return parseChineseDetectionJSON(resp.Text()), nil
}

// parseChineseDetectionJSON extracts detection fields from a JSON string.
// Returns Detection{HasChinese: false} if parsing fails.
func parseChineseDetectionJSON(raw string) Detection {
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
		return Detection{}
	}
	if result.ChineseCharCount < 0 {
		result.ChineseCharCount = 0
	}
	return Detection{
		HasChinese:   result.HasChineseText,
		ChineseChars: result.ChineseCharCount,
	}
}

// VerifyTranslatedImage compares the original image and the generated English image
// using the configured Gemini text model to verify no Chinese text remains and all
// major visual elements and information are preserved.
// If the model response cannot be parsed as JSON, it conservatively returns
// Verification{PreservesContent: false}.
func (c *Client) VerifyTranslatedImage(ctx context.Context, original []byte, originalMIME string, generated []byte, generatedMIME string) (Verification, error) {
	client, err := c.newGenAIClient(ctx)
	if err != nil {
		return Verification{}, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromBytes(original, originalMIME),
			genai.NewPartFromBytes(generated, generatedMIME),
			genai.NewPartFromText(verifyImagePrompt),
		}, genai.RoleUser),
	}

	var temp float32 = 0
	config := &genai.GenerateContentConfig{
		Temperature:      &temp,
		ResponseMIMEType: "application/json",
	}

	resp, err := client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return Verification{}, fmt.Errorf("failed to verify translated image: %w", err)
	}

	return parseVerificationJSON(resp.Text()), nil
}

// parseVerificationJSON extracts verification fields from a JSON string.
// Conservatively returns PreservesContent: false if parsing fails.
func parseVerificationJSON(raw string) Verification {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimPrefix(cleaned, "```JSON")
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	var result verificationResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return Verification{
			HasChineseText:   false,
			PreservesContent: false,
			Reason:           "invalid verification JSON",
		}
	}
	return Verification{
		HasChineseText:   result.HasChineseText,
		PreservesContent: result.PreservesContent,
		Reason:           strings.TrimSpace(result.Reason),
	}
}

// TranslateImage uses the configured Gemini image model to recreate an image with Chinese text translated to English.
func (c *Client) TranslateImage(ctx context.Context, imageData []byte, mimeType string) ([]byte, string, error) {
	client, err := c.newGenAIClient(ctx)
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
		ImageConfig: &genai.ImageConfig{
			ImageSize:   c.ImageSize(),
			AspectRatio: detectAspectRatio(imageData),
		},
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
