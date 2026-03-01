package utils

import (
	"regexp"
	"strings"
)

// RemoveFrontmatter removes Jekyll/Hugo frontmatter from markdown content
// Frontmatter is typically enclosed between --- at the beginning of the file
func RemoveFrontmatter(content string) string {
	// Pattern to match frontmatter: starts with ---, ends with ---
	// (?s) makes . match newlines
	frontmatterPattern := regexp.MustCompile(`(?s)^---\s*\n.*?\n---\s*\n`)

	cleaned := frontmatterPattern.ReplaceAllString(content, "")

	// Also handle triple backticks at the start (code block style)
	codeBlockPattern := regexp.MustCompile(`(?s)^\x60\x60\x60\s*\n.*?\n\x60\x60\x60\s*\n`)
	cleaned = codeBlockPattern.ReplaceAllString(cleaned, "")

	return strings.TrimSpace(cleaned)
}

// HasFrontmatter checks if content has frontmatter
func HasFrontmatter(content string) bool {
	frontmatterPattern := regexp.MustCompile(`(?s)^---\s*\n.*?\n---\s*\n`)
	return frontmatterPattern.MatchString(content)
}
