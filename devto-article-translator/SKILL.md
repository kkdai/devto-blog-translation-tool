---
name: devto-article-translator
description: Translate and publish draft articles from Dev.to into English using the Gemini API. Use when the user wants to translate and publish their Dev.to draft articles for a wider English-speaking audience.
---

# Devto Article Translator

## Overview

This skill provides an automated workflow to fetch draft articles from a user's Dev.to account, translate them (title and body) from any language to English using Gemini 2.0 Flash Lite, and then update and publish them on Dev.to.

## Requirements

The following environment variables (System Parameters) MUST be set in your shell environment or a `.env` file before using this skill:

- `DEVTO_API_KEY`: Your personal Dev.to API key (can be generated at dev.to/settings/extensions).
- `GEMINI_API_KEY`: Your Google Gemini API key (can be generated at ai.google.dev).

## Workflow

To use this skill, follow these steps:

1.  **Preparation**: Ensure the environment variables `DEVTO_API_KEY` and `GEMINI_API_KEY` are correctly set.
2.  **Execution**: Run the `translate.sh` script located in the `scripts/` directory of the skill.
    ```bash
    bash scripts/translate.sh
    ```
3.  **Process**:
    - The script will connect to Dev.to and list all unpublished (draft) articles.
    - For each draft, it will check if the title is already in English.
    - Non-English titles and contents will be translated to English using the Gemini API.
    - The original draft article on Dev.to will be **UPDATED** with the translated content and **PUBLISHED**.

## Notes

- **Caution**: This skill **updates and publishes** your existing draft articles. It does not create new articles.
- The translation process preserves markdown formatting, code blocks, and tags.
- The `gemini-2.0-flash-lite` model is used by default for cost-effective and high-quality translations.
