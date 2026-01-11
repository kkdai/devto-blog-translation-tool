# Dev.to Blog Translation Tool

A command-line tool that automatically translates your Dev.to draft articles from any language to English using Google's Gemini AI.

## Features

- 🔍 Automatically fetches **all** draft articles from your Dev.to account (supports pagination for 300+ articles)
- 🌐 Translates article titles and content to English using Gemini 2.0 Flash Lite
- 🧠 Smart language detection - automatically skips articles with English titles
- 📝 Preserves markdown formatting, code blocks, and technical terms
- 🧹 Automatically removes Jekyll/Hugo frontmatter metadata before translation
- 🚀 Automatically publishes translated articles to Dev.to
- 🔒 Secure API key management using environment variables
- ⚡ Fast and efficient translation processing with rate limiting
- 📊 Detailed progress reporting and statistics

## Prerequisites

- Go 1.21 or higher
- Dev.to account with API access
- Google Gemini API key

## Installation

1. Clone the repository:
```bash
git clone https://github.com/al03034132/devto-blog-translation-tool.git
cd devto-blog-translation-tool
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
```

4. Edit `.env` and add your API keys:
```env
DEVTO_API_KEY=your_devto_api_key_here
GEMINI_API_KEY=your_gemini_api_key_here
```

## Getting API Keys

### Dev.to API Key

1. Go to [Dev.to Settings > Extensions](https://dev.to/settings/extensions)
2. Generate a new API key
3. Copy the key to your `.env` file

### Google Gemini API Key

1. Visit [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Create a new API key
3. Copy the key to your `.env` file

## Usage

Run the translation tool:

```bash
go run cmd/translator/main.go
```

The tool will:
1. Display a confirmation prompt explaining what will happen
2. Load your API keys from the `.env` file
3. Fetch **all** draft articles from your Dev.to account (with pagination support)
4. For each draft article:
   - Display the article ID and original title
   - Check if the title is already in English (skip if yes)
   - Translate title and content to English using Gemini AI
   - **UPDATE the existing draft article** with translated content
   - Publish the updated article
5. Display a summary with statistics:
   - Number of articles translated and published
   - Number of articles skipped (already in English)
   - Number of failed translations

### ⚠️ IMPORTANT NOTES

1. **Updates Existing Drafts**: This tool **UPDATES your existing draft articles**, it does NOT create new articles
2. **Uses PUT Request**: The tool uses `PUT /articles/:id` to update the same article with translated content
3. **Publishes Automatically**: After translation, the article will be automatically published
4. **Confirmation Required**: The tool will ask for confirmation before starting
5. **Shows Article IDs**: Each processed article will display its ID so you can verify it's updating the correct article
6. **Frontmatter Removal**: Automatically removes Jekyll/Hugo frontmatter (metadata between `---`) before translation to avoid translating metadata

### Example Output

```
⚠️  WARNING ⚠️
=====================================
This tool will:
1. Fetch ALL your draft articles from Dev.to
2. Translate non-English articles to English
3. UPDATE the existing draft articles with translations
4. PUBLISH the updated articles

Note: This will UPDATE your existing drafts,
      NOT create new articles.
=====================================

Do you want to continue? (yes/no): yes

✅ Starting translation and publishing process...

🔍 Fetching draft articles from Dev.to...
  Fetched page 1 (30 articles, total: 30)
  Fetched page 2 (30 articles, total: 60)
  ...

✅ Found 300 draft article(s)

[1/300] Article ID: 2457408
  Original Title: 使用 Gemini 3.0 Pro Image API 打造 PDF 文字優化工具
  Translating title... ✅
  Translated Title: Building a PDF Text Optimization Tool with Gemini 3.0 Pro Image API
  Translating content... (removing frontmatter) ✅
  Updating article ID 2457408 and publishing... ✅ Published!

[2/300] Article ID: 2457409
  Original Title: Introduction to Machine Learning
  ⏭️  Skipping: Title is already in English

...

========================================
SUMMARY
========================================
✅ Translated & Published: 250
⏭️  Skipped (English): 45
❌ Failed: 5
📊 Total processed: 300
========================================

🎉 Process completed!
```

## Project Structure

```
devto-blog-translation-tool/
├── cmd/
│   └── translator/
│       └── main.go              # Main application entry point
├── internal/
│   ├── devto/
│   │   └── client.go            # Dev.to API client (with pagination & publish)
│   ├── gemini/
│   │   └── client.go            # Gemini API client for translation
│   ├── translator/
│   │   └── translator.go        # Translation service logic
│   └── utils/
│       ├── language.go          # Language detection utilities
│       └── markdown.go          # Markdown/frontmatter processing
├── .env                         # Your API keys (DO NOT COMMIT!)
├── .env.example                 # Template for environment variables
├── .gitignore                   # Git ignore file
├── go.mod                       # Go module file
├── test_security.sh             # Security verification script
└── README.md                    # This file
```

## Frontmatter Handling

The tool automatically detects and removes Jekyll/Hugo frontmatter before translation. This prevents metadata from being translated.

**Example:**

Original content with frontmatter:
```markdown
---
title: [VS Code][Colab] Google Officially Releases Colab VS Code Plugin
published: false
date: 2025-11-14 00:00:00 UTC
tags:
canonical_url: https://www.evanlin.com/colab-vscode-plugin/
---

# Introduction

This is the actual article content...
```

The tool will:
1. Detect the frontmatter (content between `---`)
2. Remove it before translation
3. Only translate the actual article content:
```markdown
# Introduction

This is the actual article content...
```

**Supported formats:**
- Standard frontmatter: `--- ... ---`
- Code block style: ` ``` ... ``` `

## Security

🔒 **IMPORTANT**: This project uses environment variables to store sensitive API keys.

- **NEVER** commit your `.env` file to Git
- The `.env` file is already listed in `.gitignore`
- Use `.env.example` as a template for other developers
- API keys are never logged or displayed in the output

## Building

To build a standalone executable:

```bash
go build -o translator cmd/translator/main.go
```

Then run:
```bash
./translator
```

## Troubleshooting

### "DEVTO_API_KEY environment variable is not set"

Make sure you have created a `.env` file in the project root with your API keys.

### "failed to fetch draft articles: API request failed with status 401"

Your Dev.to API key may be invalid or expired. Generate a new key from Dev.to settings.

### "failed to generate translation"

Check that your Gemini API key is valid and you haven't exceeded the rate limits.

## Dependencies

- [godotenv](https://github.com/joho/godotenv) - Environment variable management
- [generative-ai-go](https://github.com/google/generative-ai-go) - Google Gemini AI SDK

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

Created for translating Dev.to blog posts efficiently.
