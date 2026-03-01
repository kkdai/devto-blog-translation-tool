package utils

import (
	"regexp"
	"unicode"
)

// IsEnglish checks if the text is primarily in English
// Returns true only if the text contains NO CJK characters (Chinese, Japanese, Korean)
// and mostly Latin alphabet characters
func IsEnglish(text string) bool {
	if text == "" {
		return false
	}

	// Check for CJK characters (Chinese, Japanese, Korean)
	// If any CJK character is found, it's not purely English
	for _, r := range text {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			return false
		}
	}

	// Remove common punctuation and whitespace
	cleanText := regexp.MustCompile(`[^\p{L}\p{N}]`).ReplaceAllString(text, "")

	if len(cleanText) == 0 {
		return false
	}

	englishChars := 0
	totalChars := 0

	for _, r := range cleanText {
		totalChars++
		// Check if character is in Latin/ASCII range (English letters and numbers)
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || unicode.IsDigit(r) {
			englishChars++
		}
	}

	// Consider text as English if more than 80% of characters are Latin/ASCII
	threshold := 0.8
	ratio := float64(englishChars) / float64(totalChars)

	return ratio >= threshold
}
