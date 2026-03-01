package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/al03034132/devto-blog-translation-tool/internal/devto"
	"github.com/al03034132/devto-blog-translation-tool/internal/gemini"
	"github.com/al03034132/devto-blog-translation-tool/internal/translator"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Get API keys from environment variables
	devtoAPIKey := os.Getenv("DEVTO_API_KEY")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	// Validate API keys
	if devtoAPIKey == "" {
		log.Fatal("❌ Error: DEVTO_API_KEY environment variable is not set")
	}

	if geminiAPIKey == "" {
		log.Fatal("❌ Error: GEMINI_API_KEY environment variable is not set")
	}

	// Never log the actual API keys!
	fmt.Println("✅ API keys loaded successfully")
	fmt.Println()

	// Initialize clients
	devtoClient := devto.NewClient(devtoAPIKey)
	geminiClient := gemini.NewClient(geminiAPIKey)

	// Create translation service
	service := translator.NewService(devtoClient, geminiClient)

	// Create context
	ctx := context.Background()

	// Display warning and get confirmation
	fmt.Println("⚠️  WARNING ⚠️")
	fmt.Println("=====================================")
	fmt.Println("This tool will:")
	fmt.Println("1. Fetch ALL your draft articles from Dev.to")
	fmt.Println("2. Translate non-English articles to English")
	fmt.Println("3. UPDATE the existing draft articles with translations")
	fmt.Println("4. PUBLISH the updated articles")
	fmt.Println()
	fmt.Println("Note: This will UPDATE your existing drafts,")
	fmt.Println("      NOT create new articles.")
	fmt.Println("=====================================")
	fmt.Println()
	fmt.Print("Do you want to continue? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("❌ Error reading input: %v", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		fmt.Println("❌ Operation cancelled by user")
		os.Exit(0)
	}

	fmt.Println()
	fmt.Println("✅ Starting translation and publishing process...")
	fmt.Println()

	// Fetch, translate, and publish draft articles
	_, err = service.TranslateDraftArticles(ctx)
	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	fmt.Println("🎉 Process completed!")
}
