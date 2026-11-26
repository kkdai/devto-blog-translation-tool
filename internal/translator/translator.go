package translator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/al03034132/devto-blog-translation-tool/internal/devto"
	"github.com/al03034132/devto-blog-translation-tool/internal/gemini"
	"github.com/al03034132/devto-blog-translation-tool/internal/utils"
)

const (
	// Dev.to API enforces a 128-character limit on article titles.
	maxTitleLen             = 128
	maxShortenTitleAttempts = 2
)

type titleFitStatus int

const (
	titleFitUnchanged titleFitStatus = iota
	titleFitShortened
	titleFitTruncated
)

type titleShortenFunc func(ctx context.Context, title string, maxLen int) (string, error)

// fitTitleToLimit ensures title fits within maxLen runes.
// If title exceeds maxLen, it calls shorten up to maxShortenTitleAttempts times.
// If shortening fails or still exceeds maxLen, it falls back to word-boundary truncation.
func fitTitleToLimit(ctx context.Context, title string, maxLen int, shorten titleShortenFunc) (string, titleFitStatus) {
	if utf8.RuneCountInString(title) <= maxLen {
		return title, titleFitUnchanged
	}

	if shorten != nil {
		for attempt := 0; attempt < maxShortenTitleAttempts; attempt++ {
			shortened, err := shorten(ctx, title, maxLen)
			if err != nil {
				break
			}
			shortened = strings.TrimSpace(shortened)
			if shortened != "" && utf8.RuneCountInString(shortened) <= maxLen {
				return shortened, titleFitShortened
			}
		}
	}

	return utils.TruncateAtWordBoundary(title, maxLen), titleFitTruncated
}

// ImageProcessor detects and replaces images containing Chinese text in markdown.
type ImageProcessor interface {
	ProcessMarkdown(ctx context.Context, articleID int, markdown string) string
}

// Service handles the translation of Dev.to articles
type Service struct {
	devtoClient    *devto.Client
	geminiClient   *gemini.Client
	imageProcessor ImageProcessor
}

// TranslatedArticle represents an article with its translation
type TranslatedArticle struct {
	OriginalID      int
	OriginalTitle   string
	TranslatedTitle string
	OriginalBody    string
	TranslatedBody  string
	Tags            []string
	URL             string
}

// NewService creates a new translation service
func NewService(devtoClient *devto.Client, geminiClient *gemini.Client) *Service {
	return &Service{
		devtoClient:  devtoClient,
		geminiClient: geminiClient,
	}
}

// SetImageProcessor configures an optional image processor for translating Chinese text in images.
func (s *Service) SetImageProcessor(p ImageProcessor) {
	s.imageProcessor = p
}

// TranslateDraftArticles fetches draft articles, translates them, and publishes them
func (s *Service) TranslateDraftArticles(ctx context.Context) ([]TranslatedArticle, error) {
	// Fetch draft articles from Dev.to
	fmt.Println("🔍 Fetching draft articles from Dev.to...")
	articles, err := s.devtoClient.GetDraftArticles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch draft articles: %w", err)
	}

	if len(articles) == 0 {
		return nil, errors.New("no draft articles found")
	}

	fmt.Printf("\n✅ Found %d draft article(s)\n\n", len(articles))

	var translatedArticles []TranslatedArticle
	skippedCount := 0
	successCount := 0
	failedCount := 0

	for i, article := range articles {
		fmt.Printf("[%d/%d] Article ID: %d\n", i+1, len(articles), article.ID)
		fmt.Printf("  Original Title: %s\n", article.Title)

		// Check if title is already in English
		if utils.IsEnglish(article.Title) {
			fmt.Println("  ⏭️  Skipping: Title is already in English")
			skippedCount++
			fmt.Println()
			continue
		}

		// Translate title
		fmt.Print("  Translating title... ")
		translatedTitle, err := s.geminiClient.TranslateTitle(ctx, article.Title)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			failedCount++
			fmt.Println()
			continue
		}
		translatedTitle = strings.TrimSpace(translatedTitle)
		var fitStatus titleFitStatus
		translatedTitle, fitStatus = fitTitleToLimit(ctx, translatedTitle, maxTitleLen, s.geminiClient.ShortenTitle)
		fmt.Printf("✅\n")
		fmt.Printf("  Translated Title: %s\n", translatedTitle)
		switch fitStatus {
		case titleFitShortened:
			fmt.Printf("  ✂️  Title shortened to %d characters (Dev.to limit)\n", utf8.RuneCountInString(translatedTitle))
		case titleFitTruncated:
			fmt.Printf("  ⚠️  Title truncated at word boundary to %d characters (Dev.to limit)\n", utf8.RuneCountInString(translatedTitle))
		}

		// Remove frontmatter before translation
		contentToTranslate := utils.RemoveFrontmatter(article.BodyMarkdown)

		// Replace images containing Chinese text with English versions (when enabled)
		if s.imageProcessor != nil {
			contentToTranslate = s.imageProcessor.ProcessMarkdown(ctx, article.ID, contentToTranslate)
		}

		// Translate body
		fmt.Print("  Translating content... ")
		if utils.HasFrontmatter(article.BodyMarkdown) {
			fmt.Print("(removing frontmatter) ")
		}
		translatedBody, err := s.geminiClient.Translate(ctx, contentToTranslate)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			failedCount++
			fmt.Println()
			continue
		}
		translatedBody = strings.TrimSpace(translatedBody)
		fmt.Println("✅")

		// Update and publish the article to Dev.to
		fmt.Printf("  Updating article ID %d and publishing... ", article.ID)
		err = s.devtoClient.PublishArticle(ctx, article.ID, translatedTitle, translatedBody, article.Tags)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			failedCount++
			fmt.Println()
			continue
		}
		fmt.Println("✅ Published!")

		translatedArticles = append(translatedArticles, TranslatedArticle{
			OriginalID:      article.ID,
			OriginalTitle:   article.Title,
			TranslatedTitle: translatedTitle,
			OriginalBody:    article.BodyMarkdown,
			TranslatedBody:  translatedBody,
			Tags:            article.Tags,
			URL:             article.URL,
		})

		successCount++
		fmt.Println()
	}

	// Print summary
	fmt.Println("========================================")
	fmt.Println("SUMMARY")
	fmt.Println("========================================")
	fmt.Printf("✅ Translated & Published: %d\n", successCount)
	fmt.Printf("⏭️  Skipped (English): %d\n", skippedCount)
	fmt.Printf("❌ Failed: %d\n", failedCount)
	fmt.Printf("📊 Total processed: %d\n", len(articles))
	fmt.Println("========================================")
	fmt.Println()

	return translatedArticles, nil
}

// PrintTranslations prints the translated articles
func (s *Service) PrintTranslations(articles []TranslatedArticle) {
	fmt.Println("========================================")
	fmt.Println("TRANSLATION RESULTS")
	fmt.Println("========================================")
	fmt.Println()

	for i, article := range articles {
		fmt.Printf("Article #%d (ID: %d)\n", i+1, article.OriginalID)
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("Original Title: %s\n", article.OriginalTitle)
		fmt.Printf("Translated Title: %s\n", article.TranslatedTitle)
		fmt.Println()
		fmt.Printf("Tags: %s\n", strings.Join(article.Tags, ", "))
		fmt.Printf("URL: %s\n", article.URL)
		fmt.Println()
		fmt.Println("Translated Content:")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println(article.TranslatedBody)
		fmt.Println()
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println()
	}
}
