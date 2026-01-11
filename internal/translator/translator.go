package translator

import (
	"context"
	"fmt"
	"strings"

	"github.com/al03034132/devto-blog-translation-tool/internal/devto"
	"github.com/al03034132/devto-blog-translation-tool/internal/gemini"
	"github.com/al03034132/devto-blog-translation-tool/internal/utils"
)

// Service handles the translation of Dev.to articles
type Service struct {
	devtoClient  *devto.Client
	geminiClient *gemini.Client
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

// TranslateDraftArticles fetches draft articles, translates them, and publishes them
func (s *Service) TranslateDraftArticles(ctx context.Context) ([]TranslatedArticle, error) {
	// Fetch draft articles from Dev.to
	fmt.Println("🔍 Fetching draft articles from Dev.to...")
	articles, err := s.devtoClient.GetDraftArticles()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch draft articles: %w", err)
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("no draft articles found")
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
		fmt.Printf("✅\n")
		fmt.Printf("  Translated Title: %s\n", translatedTitle)

		// Remove frontmatter before translation
		contentToTranslate := utils.RemoveFrontmatter(article.BodyMarkdown)

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
		err = s.devtoClient.PublishArticle(article.ID, translatedTitle, translatedBody, article.Tags)
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
