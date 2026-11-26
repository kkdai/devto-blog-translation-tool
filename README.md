# Dev.to Blog Translation Tool

[English](README.md) | [繁體中文](README-zhtw.md)

A command-line tool that automatically translates your Dev.to draft articles from any language to English using Google's Gemini AI.

## Features

- 🔍 Automatically fetches **all** draft articles from your Dev.to account (supports pagination for 300+ articles)
- 🌐 Translates article titles and content to English using Gemini 4 Neon (`gemini-4-neon`, overridable via `GEMINI_TEXT_MODEL`)
- 🖼️ Optional image translation — detects images containing Chinese text, recreates English versions with Gemini, verifies them against the originals, uploads them to Google Cloud Storage, and replaces the image URLs
- 🧠 Smart language detection — automatically skips articles that already have English titles
- 📝 Preserves markdown formatting, code blocks, and technical terms
- 🧹 Automatically removes Jekyll/Hugo frontmatter metadata before translation
- ✂️ Fits translated titles to Dev.to's 128-character limit (shortens with Gemini, falling back to word-boundary truncation)
- 🚀 Automatically publishes translated articles to Dev.to
- 🔒 Secure API key management using environment variables
- ⚡ Rate-limited, cancellable request handling
- 📊 Detailed progress reporting and statistics

## Prerequisites

- Go 1.26 or higher (the version is declared in `go.mod`)
- Dev.to account with API access
- Google Gemini API key
- *(Optional, for image translation)* Google Cloud SDK (`gcloud` CLI) and a GCP project with a Cloud Storage bucket

## Installation

1. Clone the repository:
```bash
git clone https://github.com/kkdai/devto-blog-translation-tool.git
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

## Configuration

Variables can be placed in a `.env` file in the project root or exported directly as shell environment variables. `godotenv` loads `.env` without overriding environment variables that are already set in your shell.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DEVTO_API_KEY` | Yes | — | Dev.to API key for fetching drafts and publishing updates |
| `GEMINI_API_KEY` | Yes | — | Google Gemini API key |
| `GEMINI_TEXT_MODEL` | No | `gemini-4-neon` | Gemini model used for text translation, title shortening, Chinese detection, and image verification |
| `GCS_BUCKET` | No | *(disabled)* | GCS bucket name for translated images; when unset, image translation is disabled |
| `GEMINI_IMAGE_MODEL` | No | `gemini-3-pro-image` | Gemini model used to generate English versions of images |
| `GEMINI_IMAGE_SIZE` | No | `2K` | Output resolution for translated images (`1K`, `2K`, or `4K`) |
| `IMAGE_MAX_CHINESE_CHARS` | No | `60` | Skip redrawing images whose estimated Chinese character count exceeds this threshold |

## Getting API Keys

### Dev.to API Key

1. Go to [Dev.to Settings > Extensions](https://dev.to/settings/extensions)
2. Generate a new API key
3. Copy the key to your `.env` file

### Google Gemini API Key

1. Visit [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Create a new API key
3. Copy the key to your `.env` file

### Google Cloud Storage (Optional — Chinese Image Translation)

If `GCS_BUCKET` is not set, image translation is disabled and the tool runs as usual. To enable it:

1. Authenticate locally with Application Default Credentials:
   ```bash
   gcloud auth application-default login
   ```
2. Create a GCS bucket and make it publicly readable (works with uniform bucket-level access — no per-object ACLs are set):
   ```bash
   gcloud storage buckets create gs://YOUR_BUCKET_NAME --uniform-bucket-level-access
   gcloud storage buckets add-iam-policy-binding gs://YOUR_BUCKET_NAME \
     --member=allUsers --role=roles/storage.objectViewer
   ```
3. Add `GCS_BUCKET` (and any optional image settings) to `.env`:
   ```env
   GCS_BUCKET=YOUR_BUCKET_NAME
   # Optional overrides:
   GEMINI_TEXT_MODEL=gemini-4-neon
   GEMINI_IMAGE_MODEL=gemini-3-pro-image
   GEMINI_IMAGE_SIZE=2K
   IMAGE_MAX_CHINESE_CHARS=60
   ```

See [Image Translation](#image-translation) below for the full processing pipeline and tuning tips.

## Usage

Run the translation tool:

```bash
go run ./cmd/translator
```

The tool will:
1. Load your API keys and configuration from `.env` / environment variables
2. Display a confirmation prompt explaining what will happen
3. Fetch **all** draft articles from your Dev.to account (with pagination support)
4. For each draft article:
   - Display the article ID and original title
   - Check if the title is already in English (skip if yes)
   - Translate the title to English using Gemini (shortening it if it exceeds 128 characters)
   - Remove Jekyll/Hugo frontmatter (if present)
   - *(If `GCS_BUCKET` is set)* Scan and translate images containing Chinese text, upload them to GCS, and replace their URLs
   - Translate the article body to English using Gemini
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
7. **All-or-Nothing Run**: There is currently no dry-run mode or per-article filter. The tool processes every non-English draft in one pass
8. **Public GCS Uploads**: When `GCS_BUCKET` is set, translated images are uploaded to a publicly readable GCS bucket so Dev.to can display them

### Example Output

```
✅ API keys loaded successfully
✅ Gemini text model: gemini-4-neon
ℹ️  GCS_BUCKET not set; Chinese image translation is disabled

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
├── .github/
│   └── workflows/
│       └── go.yml                   # CI: build and test on push/PR
├── cmd/
│   └── translator/
│       └── main.go                  # Main application entry point
├── internal/
│   ├── devto/
│   │   └── client.go                # Dev.to API client (pagination & publish)
│   ├── gemini/
│   │   ├── client.go                # Gemini text translation & title shortening client
│   │   ├── client_test.go           # Unit tests for Gemini text client
│   │   ├── image.go                 # Gemini image detection, generation & verification
│   │   └── image_test.go            # Unit tests for Gemini image helpers
│   ├── images/
│   │   ├── markdown.go              # Markdown / HTML <img> URL extraction & replacement
│   │   ├── markdown_test.go         # Unit tests for image URL parsing
│   │   ├── processor.go             # End-to-end per-image translation pipeline
│   │   └── processor_test.go        # Unit tests for image processor
│   ├── storage/
│   │   └── gcs.go                   # Google Cloud Storage uploader
│   ├── translator/
│   │   ├── translator.go            # Translation service orchestration
│   │   └── translator_test.go       # Unit tests for translation service
│   └── utils/
│       ├── language.go              # Language detection utilities
│       ├── markdown.go              # Markdown/frontmatter processing
│       ├── title.go                 # Word-boundary title truncation
│       └── title_test.go            # Unit tests for title truncation
├── devto-article-translator/        # Packaged skill distribution (separate Go module)
├── .env                             # Your API keys (DO NOT COMMIT!)
├── .env.example                     # Template for environment variables
├── .gitignore                       # Git ignore file
├── go.mod                           # Go module file
├── test_security.sh                 # Security verification script
├── README.md                        # English documentation
└── README-zhtw.md                   # Traditional Chinese documentation
```

> **Note:** `devto-article-translator/` contains a packaged copy of the same
> program with its own `go.mod`, so it is not built by `go build ./...` from the
> repository root. Consolidating the two copies is tracked on the roadmap.

## Translation Behavior

### Language Detection

`utils.IsEnglish` decides whether an article needs translating. An article is treated as **not** English if its title contains any CJK character (Han, Hiragana, Katakana, or Hangul). Otherwise the title counts as English when at least 80% of its letters and digits are ASCII Latin.

Only the **title** is checked. If the title is English, the article is skipped entirely and its body is never translated.

### Title Length

Dev.to rejects titles longer than 128 characters. When a translated title exceeds 128 runes, the tool first asks Gemini (`ShortenTitle`) to rewrite it within the limit — preserving leading bracketed tags (such as `[Notes]` or `[Tech]`) and key meaning — for up to **2 attempts**.

If shortening fails or still exceeds 128 runes, the tool falls back to `utils.TruncateAtWordBoundary`, which cuts at the last whitespace boundary within 128 runes and trims trailing whitespace or punctuation (`,`, `:`, `;`, `-`, `–`, `—`), only hard-cutting at 128 runes if there is no whitespace in range. The console logs whether the title was shortened (`✂️`) or truncated (`⚠️`).

### Token Budget

Gemini 4 Neon (`gemini-4-neon`) is a thinking model: its internal reasoning tokens count against `MaxOutputTokens`. The limits are therefore set generously — **65,536** for article bodies and **16,384** for title translation and title shortening — because a tight budget gets consumed by reasoning and truncates the output mid-sentence. Billing is based on tokens actually used, so a high ceiling costs nothing extra.

If a response still stops at `FinishReasonMaxTokens` (or returns empty content), the tool automatically retries once before reporting a truncation error rather than publishing a half-translated article.

## Image Translation

When `GCS_BUCKET` is configured, each non-English article goes through an image translation pipeline **before** its body text is translated. Each unique image URL is processed in order:

```
Extract URLs ─▶ Download ─▶ Detect Chinese ─▶ Generate English ─▶ Verify (orig vs new) ─▶ Upload GCS & Replace URL
```

1. **Extract image URLs**: Scans markdown `![alt](url)` and HTML `<img src="url">` tags, skipping fenced code blocks, inline code spans, non-`http(s)` URLs, and duplicate URLs.
2. **Download image**: Downloads with a **30-second timeout** and a **20 MB** size limit. Supports `image/png`, `image/jpeg`, and `image/webp`; `image/gif` images are skipped.
3. **Detect Chinese text**: Sends the image to the configured text model (`GEMINI_TEXT_MODEL`) to check for Chinese characters and estimate the character count:
   - No Chinese text → skipped, keeps original.
   - Estimated Chinese characters exceed `IMAGE_MAX_CHINESE_CHARS` (default `60`) → skipped, keeps original. Redrawing text-dense infographics or diagrams often produces garbled characters.
4. **Generate English version**: Sends the image to `GEMINI_IMAGE_MODEL` (default `gemini-3-pro-image`) at the configured `GEMINI_IMAGE_SIZE` (`1K`, `2K`, or `4K`), automatically selecting the supported aspect ratio closest to the original image (`1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, or `21:9`).
5. **Verify against original**: Sends both the original and generated images to `GEMINI_TEXT_MODEL` to verify that no Chinese text remains and all major UI elements, panels, diagrams, and information are preserved. If Chinese text remains, content is lost, or verification fails → keeps original.
6. **Upload & replace**: Uploads the verified image to GCS at `devto-images/<articleID>/<first-16-hex-of-sha256>-en.<ext>`, replaces all occurrences of the original URL in the markdown with the public GCS URL (`https://storage.googleapis.com/<bucket>/...`), and then proceeds to translate the article body.

Failure on any individual image only prints a warning and keeps the original image URL — it never aborts translation or publishing of the article.

### Example Console Output

```
  🖼️  Image 1/4: https://example.com/diagram-en.png ... ⏭️  no Chinese text
  🖼️  Image 2/4: https://example.com/ui-screenshot.png ... ✅ replaced
  🖼️  Image 3/4: https://example.com/infographic.png ... ⏭️  too much text (~390 Chinese chars), keeping original
  🖼️  Image 4/4: https://example.com/settings.png ... ⚠️  generated image lost content (sidebar panel missing), keeping original
```

### Tuning & Cost Notes

- **Threshold tuning (`IMAGE_MAX_CHINESE_CHARS`)**: The default of `60` is conservative to avoid garbled text. For UI screenshots with moderate labels, raising it to `100`–`150` works well, though occasional minor typos may appear.
- **API cost and latency**: Every image that contains Chinese text (and is within the threshold) makes **at least 3 Gemini API calls** — detection, image generation, and verification — which adds both latency and API usage per image.

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

Run the bundled check to verify no keys have leaked into tracked files:

```bash
./test_security.sh
```

## Building

To build a standalone executable:

```bash
go build -o translator ./cmd/translator
```

Then run:
```bash
./translator
```

## Development

```bash
go build ./...   # compile
go vet ./...     # static checks
gofmt -l .       # formatting (should print nothing)
go test ./...    # run unit tests
```

The unit tests cover `internal/gemini`, `internal/images`, `internal/translator`, and `internal/utils` using mocks and local fixtures — they do not call external APIs and require no API keys or GCS credentials.

## Troubleshooting

### "DEVTO_API_KEY environment variable is not set"

Make sure you have created a `.env` file in the project root with your API keys (or exported them in your shell).

### "failed to fetch draft articles: API request failed with status 401"

Your Dev.to API key may be invalid or expired. Generate a new key from Dev.to settings.

### "failed to generate translation"

Check that your Gemini API key is valid and you haven't exceeded the rate limits.

### "models/... is not found for API version v1"

Image detection, generation, and verification already target the Gemini `v1beta` API (`google.golang.org/genai`). If you override `GEMINI_TEXT_MODEL` or `GEMINI_IMAGE_MODEL` with a custom model name, verify that the model is available on `v1beta`.

### "translation truncated: hit MaxOutputTokens"

The tool automatically retries once when a response hits `MaxOutputTokens`. If both attempts still exhaust the token budget, raise `SetMaxOutputTokens` in `internal/gemini/client.go` or split the article before translating.

### GCS: "could not find default credentials"

Run `gcloud auth application-default login` to configure Application Default Credentials locally before running the tool with `GCS_BUCKET` set.

### GCS: Image uploaded successfully but fails to load

Your GCS bucket is not publicly readable. Grant `allUsers` the `roles/storage.objectViewer` role:

```bash
gcloud storage buckets add-iam-policy-binding gs://YOUR_BUCKET_NAME \
  --member=allUsers --role=roles/storage.objectViewer
```

### All images are skipped (`too much text`)

Check `IMAGE_MAX_CHINESE_CHARS` (default `60`). If your posts contain UI screenshots with more labels, try raising it to `100`–`150`.

### "API request failed with status 422"

Dev.to rejected the update payload. The most common causes are a title over 128 characters or an invalid tag. Check the error body printed alongside the status code.

## Dependencies

- [godotenv](https://github.com/joho/godotenv) - Environment variable management
- [generative-ai-go](https://github.com/google/generative-ai-go) - Google Gemini AI SDK (text translation and title shortening)
- [google.golang.org/genai](https://pkg.go.dev/google.golang.org/genai) - Google Gen AI SDK (image Chinese detection, generation, and verification)
- [cloud.google.com/go/storage](https://pkg.go.dev/cloud.google.com/go/storage) - Google Cloud Storage client (uploading translated images)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

Created for translating Dev.to blog posts efficiently.
