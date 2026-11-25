package images

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultMaxImageBytes is the maximum allowed image download size (20MB).
	DefaultMaxImageBytes int64 = 20 * 1024 * 1024
	// DefaultHTTPTimeout is the timeout for downloading an image.
	DefaultHTTPTimeout = 30 * time.Second
)

var errSkipGIF = errors.New("gif images are skipped")

// ChineseDetector checks whether an image contains Chinese text.
type ChineseDetector interface {
	DetectChineseText(ctx context.Context, imageData []byte, mimeType string) (bool, error)
}

// ImageTranslator generates an English version of an image containing Chinese text.
type ImageTranslator interface {
	TranslateImage(ctx context.Context, imageData []byte, mimeType string) ([]byte, string, error)
}

// Uploader uploads generated image bytes and returns a public URL.
type Uploader interface {
	Upload(ctx context.Context, objectName string, data []byte, contentType string) (string, error)
}

// Processor scans markdown for images, translates those containing Chinese text, and uploads them.
type Processor struct {
	httpClient *http.Client
	maxBytes   int64
	detector   ChineseDetector
	translator ImageTranslator
	uploader   Uploader
}

// NewProcessor creates a new image Processor with default HTTP timeout and size limit.
func NewProcessor(detector ChineseDetector, translator ImageTranslator, uploader Uploader) *Processor {
	return &Processor{
		httpClient: &http.Client{Timeout: DefaultHTTPTimeout},
		maxBytes:   DefaultMaxImageBytes,
		detector:   detector,
		translator: translator,
		uploader:   uploader,
	}
}

// WithHTTPClient overrides the HTTP client used to download images.
func (p *Processor) WithHTTPClient(client *http.Client) *Processor {
	if client != nil {
		p.httpClient = client
	}
	return p
}

// WithMaxBytes overrides the maximum allowed image download size in bytes.
func (p *Processor) WithMaxBytes(maxBytes int64) *Processor {
	if maxBytes > 0 {
		p.maxBytes = maxBytes
	}
	return p
}

// ProcessMarkdown scans markdown for images, detects Chinese text, generates English versions,
// uploads them via the configured Uploader, and replaces their URLs in the returned markdown.
// Errors on individual images are logged as warnings and leave the original URL intact.
func (p *Processor) ProcessMarkdown(ctx context.Context, articleID int, markdown string) string {
	urls := ExtractImageURLs(markdown)
	if len(urls) == 0 {
		return markdown
	}

	replacements := make(map[string]string)

	for i, imgURL := range urls {
		fmt.Printf("  🖼️  Image %d/%d: %s ... ", i+1, len(urls), imgURL)

		data, mimeType, err := p.downloadImage(ctx, imgURL)
		if err != nil {
			if errors.Is(err, errSkipGIF) {
				fmt.Println("⏭️  skipped (GIF)")
			} else {
				fmt.Printf("⚠️  download failed (%v), keeping original\n", err)
			}
			continue
		}

		hasChinese, err := p.detector.DetectChineseText(ctx, data, mimeType)
		if err != nil {
			fmt.Printf("⚠️  detection failed (%v), keeping original\n", err)
			continue
		}
		if !hasChinese {
			fmt.Println("⏭️  no Chinese text")
			continue
		}

		translatedData, outMIME, err := p.translator.TranslateImage(ctx, data, mimeType)
		if err != nil {
			fmt.Printf("⚠️  image generation failed (%v), keeping original\n", err)
			continue
		}

		objectName := BuildObjectName(articleID, imgURL, outMIME)
		newURL, err := p.uploader.Upload(ctx, objectName, translatedData, outMIME)
		if err != nil {
			fmt.Printf("⚠️  upload failed (%v), keeping original\n", err)
			continue
		}

		replacements[imgURL] = newURL
		fmt.Println("✅ replaced")
	}

	return ReplaceImageURLs(markdown, replacements)
}

func (p *Processor) downloadImage(ctx context.Context, imgURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("invalid image URL: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, "", fmt.Errorf("invalid Content-Type %q", resp.Header.Get("Content-Type"))
	}
	mediaType = strings.ToLower(mediaType)
	if !strings.HasPrefix(mediaType, "image/") {
		return nil, "", fmt.Errorf("non-image Content-Type %q", mediaType)
	}
	if mediaType == "image/jpg" {
		mediaType = "image/jpeg"
	}
	if mediaType == "image/gif" {
		return nil, "", errSkipGIF
	}
	if mediaType != "image/png" && mediaType != "image/jpeg" && mediaType != "image/webp" {
		return nil, "", fmt.Errorf("unsupported image type %q", mediaType)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, p.maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > p.maxBytes {
		return nil, "", fmt.Errorf("image exceeds %d bytes limit", p.maxBytes)
	}
	if len(data) == 0 {
		return nil, "", errors.New("empty image response")
	}

	return data, mediaType, nil
}

// BuildObjectName builds the GCS object path:
// devto-images/<articleID>/<first 16 hex chars of sha256(originalURL)>-en.<ext>
func BuildObjectName(articleID int, originalURL, mimeType string) string {
	sum := sha256.Sum256([]byte(originalURL))
	hash16 := hex.EncodeToString(sum[:])[:16]
	ext := extensionForMIME(mimeType)
	return fmt.Sprintf("devto-images/%d/%s-en.%s", articleID, hash16, ext)
}

func extensionForMIME(mimeType string) string {
	mediaType, _, err := mime.ParseMediaType(mimeType)
	if err == nil {
		mimeType = mediaType
	}
	switch strings.ToLower(mimeType) {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/png":
		return "png"
	default:
		return "png"
	}
}
