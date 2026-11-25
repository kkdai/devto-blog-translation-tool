package images

import (
	"reflect"
	"testing"
)

func TestExtractImageURLs(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []string
	}{
		{
			name:     "basic markdown image",
			markdown: "Hello ![alt text](https://example.com/a.png) world",
			want:     []string{"https://example.com/a.png"},
		},
		{
			name:     "markdown image with double-quoted title",
			markdown: `![diagram](https://example.com/diagram.jpg "Architecture Diagram")`,
			want:     []string{"https://example.com/diagram.jpg"},
		},
		{
			name:     "markdown image with single-quoted title and angle brackets",
			markdown: `![diagram](<https://example.com/diagram.webp> 'My Title')`,
			want:     []string{"https://example.com/diagram.webp"},
		},
		{
			name:     "markdown image with parentheses in URL",
			markdown: `![wiki](https://example.com/image_(1).png)`,
			want:     []string{"https://example.com/image_(1).png"},
		},
		{
			name:     "HTML img tags with double and single quotes",
			markdown: `<img alt="foo" src="https://example.com/1.png" width="500"> and <IMG SRC='http://example.com/2.jpg' />`,
			want:     []string{"https://example.com/1.png", "http://example.com/2.jpg"},
		},
		{
			name:     "duplicate URLs across markdown and HTML are deduplicated in order",
			markdown: "![a](https://example.com/dup.png)\n![b](https://example.com/other.png)\n<img src=\"https://example.com/dup.png\">",
			want:     []string{"https://example.com/dup.png", "https://example.com/other.png"},
		},
		{
			name:     "non-http URLs are ignored",
			markdown: "![rel](./local.png) ![abs](/assets/img.png) ![data](data:image/png;base64,abc) <img src=\"ftp://example.com/a.png\">",
			want:     nil,
		},
		{
			name: "images inside fenced code blocks and inline code are skipped",
			markdown: "Before ![real](https://example.com/real.png)\n" +
				"```markdown\n![fenced](https://example.com/fenced.png)\n<img src=\"https://example.com/fenced-html.png\">\n```\n" +
				"Inline `![inline](https://example.com/inline.png)`\n~~~\n![tilde](https://example.com/tilde.png)\n~~~\n" +
				"After ![real2](https://example.com/real2.jpg)",
			want: []string{"https://example.com/real.png", "https://example.com/real2.jpg"},
		},
		{
			name:     "unclosed syntax does not panic",
			markdown: "![unclosed(https://example.com/a.png <img src=\"https://example.com/b.png `unclosed",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractImageURLs(tt.markdown)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractImageURLs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestReplaceImageURLs(t *testing.T) {
	tests := []struct {
		name         string
		markdown     string
		replacements map[string]string
		want         string
	}{
		{
			name:     "replaces markdown, titled markdown, and HTML img occurrences",
			markdown: "![a](https://example.com/a.png)\n![b](https://example.com/a.png \"title\")\n<img src=\"https://example.com/a.png\" width=\"100\">",
			replacements: map[string]string{
				"https://example.com/a.png": "https://storage.googleapis.com/my-bucket/devto-images/1/abc-en.png",
			},
			want: "![a](https://storage.googleapis.com/my-bucket/devto-images/1/abc-en.png)\n![b](https://storage.googleapis.com/my-bucket/devto-images/1/abc-en.png \"title\")\n<img src=\"https://storage.googleapis.com/my-bucket/devto-images/1/abc-en.png\" width=\"100\">",
		},
		{
			name:     "preserves URLs inside fenced code blocks",
			markdown: "![a](https://example.com/a.png)\n```\n![a](https://example.com/a.png)\n```",
			replacements: map[string]string{
				"https://example.com/a.png": "https://storage.googleapis.com/b/new.png",
			},
			want: "![a](https://storage.googleapis.com/b/new.png)\n```\n![a](https://example.com/a.png)\n```",
		},
		{
			name:     "does not corrupt prefix-matching URLs",
			markdown: "![short](https://example.com/img.png) ![long](https://example.com/img.png?size=large)",
			replacements: map[string]string{
				"https://example.com/img.png": "https://storage.googleapis.com/b/new.png",
			},
			want: "![short](https://storage.googleapis.com/b/new.png) ![long](https://example.com/img.png?size=large)",
		},
		{
			name:         "empty replacements returns original unchanged",
			markdown:     "![a](https://example.com/a.png)",
			replacements: map[string]string{},
			want:         "![a](https://example.com/a.png)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplaceImageURLs(tt.markdown, tt.replacements)
			if got != tt.want {
				t.Errorf("ReplaceImageURLs() =\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
