package utils

import (
	"strings"
	"unicode"
)

// TruncateAtWordBoundary truncates title to at most maxLen runes at the last
// whitespace boundary within maxLen, stripping trailing whitespace and
// punctuation/connector characters (, : ; - – —). If there is no whitespace
// within the first maxLen runes (or trimming leaves an empty string), it falls
// back to a hard cut at maxLen runes.
func TruncateAtWordBoundary(title string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(title)
	if len(runes) <= maxLen {
		return title
	}

	cutIdx := -1
	if unicode.IsSpace(runes[maxLen]) {
		cutIdx = maxLen
	} else {
		for i := maxLen - 1; i >= 0; i-- {
			if unicode.IsSpace(runes[i]) {
				cutIdx = i
				break
			}
		}
	}

	if cutIdx <= 0 {
		return string(runes[:maxLen])
	}

	truncated := strings.TrimRightFunc(string(runes[:cutIdx]), isTrailingCutRune)
	if truncated == "" {
		return string(runes[:maxLen])
	}
	return truncated
}

func isTrailingCutRune(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case ',', ':', ';', '-', '–', '—':
		return true
	default:
		return false
	}
}
