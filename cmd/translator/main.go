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
	"github.com/al03034132/devto-blog-translation-tool/internal/images"
	"github.com/al03034132/devto-blog-translation-tool/internal/storage"
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
	gcsBucket := strings.TrimSpace(os.Getenv("GCS_BUCKET"))
	geminiImageModel := strings.TrimSpace(os.Getenv("GEMINI_IMAGE_MODEL"))

	// Validate API keys
	if devtoAPIKey == "" {
		log.Fatal("❌ Error: DEVTO_API_KEY environment variable is not set")
	}

	if geminiAPIKey == "" {
		log.Fatal("❌ Error: GEMINI_API_KEY environment variable is not set")
	}

	// Never log the actual API keys!
	fmt.Println("✅ API keys loaded successfully")

	// Create context
	ctx := context.Background()

	// Initialize clients
	devtoClient := devto.NewClient(devtoAPIKey)
	geminiClient := gemini.NewClient(geminiAPIKey)
	if geminiImageModel != "" {
		geminiClient.SetImageModel(geminiImageModel)
	}

	// Create translation service
	service := translator.NewService(devtoClient, geminiClient)

	// Initialize optional image translation (enabled when GCS_BUCKET is set)
	if gcsBucket != "" {
		uploader, err := storage.NewGCSUploader(ctx, gcsBucket)
		if err != nil {
			log.Fatalf("❌ Error initializing GCS uploader: %v", err)
		}
		defer uploader.Close()

		imgProcessor := images.NewProcessor(geminiClient, geminiClient, uploader)
		service.SetImageProcessor(imgProcessor)
		fmt.Printf("✅ Image translation enabled (GCS bucket: %s, model: %s)\n", gcsBucket, geminiClient.ImageModel())
	} else {
		fmt.Println("ℹ️  GCS_BUCKET not set; Chinese image translation is disabled")
	}
	fmt.Println()

	// Display warning and get confirmation
	fmt.Println("⚠️  WARNING ⚠️")
	fmt.Println("=====================================")
	fmt.Println("This tool will:")
	fmt.Println("1. Fetch ALL your draft articles from Dev.to")
	fmt.Println("2. Translate non-English articles to English")
	if gcsBucket != "" {
		fmt.Println("3. Replace images containing Chinese text with Gemini-generated English versions uploaded to GCS")
		fmt.Println("4. UPDATE the existing draft articles with translations")
		fmt.Println("5. PUBLISH the updated articles")
	} else {
		fmt.Println("3. UPDATE the existing draft articles with translations")
		fmt.Println("4. PUBLISH the updated articles")
	}
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
