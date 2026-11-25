package images

import (
	"strings"
)

type urlSpan struct {
	start int
	end   int
	url   string
}

// ExtractImageURLs extracts unique http/https image URLs from markdown and HTML <img> tags,
// skipping fenced code blocks and inline code spans, preserving first-seen order.
func ExtractImageURLs(markdown string) []string {
	spans := findImageURLSpans(markdown)
	if len(spans) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(spans))
	urls := make([]string, 0, len(spans))
	for _, s := range spans {
		if _, ok := seen[s.url]; ok {
			continue
		}
		seen[s.url] = struct{}{}
		urls = append(urls, s.url)
	}
	return urls
}

// ReplaceImageURLs replaces image URLs found in markdown and HTML <img> tags outside code blocks.
func ReplaceImageURLs(markdown string, replacements map[string]string) string {
	if len(replacements) == 0 {
		return markdown
	}

	spans := findImageURLSpans(markdown)
	if len(spans) == 0 {
		return markdown
	}

	var b strings.Builder
	b.Grow(len(markdown))
	last := 0
	for _, s := range spans {
		newURL, ok := replacements[s.url]
		if !ok {
			continue
		}
		b.WriteString(markdown[last:s.start])
		b.WriteString(newURL)
		last = s.end
	}
	if last == 0 {
		return markdown
	}
	b.WriteString(markdown[last:])
	return b.String()
}

func isHTTPURL(u string) bool {
	lower := strings.ToLower(u)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// findImageURLSpans locates all http/https image URLs in markdown outside fenced code blocks and inline code.
func findImageURLSpans(s string) []urlSpan {
	var spans []urlSpan
	n := len(s)
	i := 0
	atLineStart := true
	inFence := false
	var fenceChar byte
	fenceLen := 0

	for i < n {
		if atLineStart {
			// Check for fenced code block opener/closer (up to 3 leading spaces).
			j := i
			spaces := 0
			for j < n && s[j] == ' ' && spaces < 3 {
				j++
				spaces++
			}
			if j < n && (s[j] == '`' || s[j] == '~') {
				ch := s[j]
				k := j
				for k < n && s[k] == ch {
					k++
				}
				runLen := k - j
				if !inFence && runLen >= 3 {
					// Opening fence: for backtick fences, info string must not contain backticks.
					lineEnd := findLineEnd(s, k)
					validOpen := true
					if ch == '`' && strings.ContainsRune(s[k:lineEnd], '`') {
						validOpen = false
					}
					if validOpen {
						inFence = true
						fenceChar = ch
						fenceLen = runLen
						i = advancePastLine(s, lineEnd)
						atLineStart = true
						continue
					}
				} else if inFence && ch == fenceChar && runLen >= fenceLen {
					lineEnd := findLineEnd(s, k)
					if strings.TrimSpace(s[k:lineEnd]) == "" {
						inFence = false
						i = advancePastLine(s, lineEnd)
						atLineStart = true
						continue
					}
				}
			}

			if inFence {
				lineEnd := findLineEnd(s, i)
				i = advancePastLine(s, lineEnd)
				atLineStart = true
				continue
			}
		}

		ch := s[i]
		if ch == '\n' {
			i++
			atLineStart = true
			continue
		}
		atLineStart = false

		// Skip inline code spans (`...`).
		if ch == '`' {
			tickEnd := i
			for tickEnd < n && s[tickEnd] == '`' {
				tickEnd++
			}
			tickLen := tickEnd - i
			closePos := findClosingBackticks(s, tickEnd, tickLen)
			if closePos >= 0 {
				i = closePos
				continue
			}
			i = tickEnd
			continue
		}

		// Markdown image: ![alt](url ...)
		if ch == '!' && i+1 < n && s[i+1] == '[' {
			if span, nextIdx, ok := parseMarkdownImage(s, i); ok {
				if isHTTPURL(span.url) {
					spans = append(spans, span)
				}
				i = nextIdx
				continue
			}
		}

		// HTML <img ... src="..." ...>
		if ch == '<' && isHTMLImgStart(s, i) {
			if span, nextIdx, ok := parseHTMLImg(s, i); ok {
				if isHTTPURL(span.url) {
					spans = append(spans, span)
				}
				i = nextIdx
				continue
			}
		}

		i++
	}

	return spans
}

func findLineEnd(s string, start int) int {
	idx := strings.IndexByte(s[start:], '\n')
	if idx < 0 {
		return len(s)
	}
	return start + idx
}

func advancePastLine(s string, lineEnd int) int {
	if lineEnd < len(s) && s[lineEnd] == '\n' {
		return lineEnd + 1
	}
	return lineEnd
}

func findClosingBackticks(s string, start, tickLen int) int {
	n := len(s)
	i := start
	for i < n {
		if s[i] == '`' {
			j := i
			for j < n && s[j] == '`' {
				j++
			}
			if j-i == tickLen {
				return j
			}
			i = j
			continue
		}
		i++
	}
	return -1
}

// parseMarkdownImage parses ![alt](url) or ![alt](url "title") starting at index start ('!').
func parseMarkdownImage(s string, start int) (urlSpan, int, bool) {
	n := len(s)
	// Skip "!["
	i := start + 2
	bracketDepth := 1
	for i < n && bracketDepth > 0 {
		if s[i] == '\\' && i+1 < n {
			i += 2
			continue
		}
		if s[i] == '[' {
			bracketDepth++
		} else if s[i] == ']' {
			bracketDepth--
		}
		i++
	}
	if bracketDepth != 0 || i >= n || s[i] != '(' {
		return urlSpan{}, start, false
	}
	// Skip '('
	i++
	for i < n && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= n {
		return urlSpan{}, start, false
	}

	var urlStart, urlEnd int
	if s[i] == '<' {
		urlStart = i + 1
		j := urlStart
		for j < n && s[j] != '>' && s[j] != '\n' {
			j++
		}
		if j >= n || s[j] != '>' {
			return urlSpan{}, start, false
		}
		urlEnd = j
		i = j + 1
	} else {
		urlStart = i
		parenDepth := 0
		for i < n {
			c := s[i]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				break
			}
			if c == '(' {
				parenDepth++
			} else if c == ')' {
				if parenDepth == 0 {
					break
				}
				parenDepth--
			}
			i++
		}
		urlEnd = i
	}

	if urlEnd <= urlStart {
		return urlSpan{}, start, false
	}

	// Skip optional whitespace and optional title before closing ')'.
	for i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	if i < n && (s[i] == '"' || s[i] == '\'' || s[i] == '(') {
		closeQuote := s[i]
		if closeQuote == '(' {
			closeQuote = ')'
		}
		i++
		for i < n && s[i] != closeQuote {
			if s[i] == '\\' && i+1 < n {
				i += 2
				continue
			}
			i++
		}
		if i >= n {
			return urlSpan{}, start, false
		}
		i++ // skip closing quote
		for i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
	}

	if i >= n || s[i] != ')' {
		return urlSpan{}, start, false
	}

	return urlSpan{
		start: urlStart,
		end:   urlEnd,
		url:   s[urlStart:urlEnd],
	}, i + 1, true
}

func isHTMLImgStart(s string, i int) bool {
	if i+4 >= len(s) {
		return false
	}
	if !strings.EqualFold(s[i:i+4], "<img") {
		return false
	}
	next := s[i+4]
	return next == ' ' || next == '\t' || next == '\n' || next == '\r' || next == '/' || next == '>'
}

// parseHTMLImg parses an HTML <img ... src="..." ...> tag starting at index start ('<').
func parseHTMLImg(s string, start int) (urlSpan, int, bool) {
	n := len(s)
	i := start + 4 // after "<img"
	var found urlSpan
	hasSrc := false

	for i < n {
		// Skip whitespace
		for i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
		if i >= n {
			return urlSpan{}, start, false
		}
		if s[i] == '>' {
			return found, i + 1, hasSrc
		}
		if s[i] == '/' && i+1 < n && s[i+1] == '>' {
			return found, i + 2, hasSrc
		}

		// Read attribute name
		nameStart := i
		for i < n && s[i] != '=' && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' && s[i] != '>' && s[i] != '/' {
			i++
		}
		attrName := s[nameStart:i]

		// Skip whitespace before '='
		for i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
		if i >= n || s[i] != '=' {
			// Boolean attribute without value
			continue
		}
		i++ // skip '='

		// Skip whitespace after '='
		for i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
		if i >= n {
			return urlSpan{}, start, false
		}

		var valStart, valEnd int
		if s[i] == '"' || s[i] == '\'' {
			quote := s[i]
			i++
			valStart = i
			for i < n && s[i] != quote {
				i++
			}
			if i >= n {
				return urlSpan{}, start, false
			}
			valEnd = i
			i++ // skip closing quote
		} else {
			valStart = i
			for i < n && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' && s[i] != '>' {
				i++
			}
			valEnd = i
		}

		if strings.EqualFold(attrName, "src") && !hasSrc {
			// Trim leading/trailing spaces inside quoted src attribute while keeping accurate span.
			for valStart < valEnd && (s[valStart] == ' ' || s[valStart] == '\t') {
				valStart++
			}
			for valEnd > valStart && (s[valEnd-1] == ' ' || s[valEnd-1] == '\t') {
				valEnd--
			}
			if valEnd > valStart {
				found = urlSpan{
					start: valStart,
					end:   valEnd,
					url:   s[valStart:valEnd],
				}
				hasSrc = true
			}
		}
	}

	return urlSpan{}, start, false
}
