package gemini

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestParseChineseDetectionJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Detection
	}{
		{
			name: "true json with char count",
			raw:  `{"has_chinese_text": true, "chinese_char_count": 25}`,
			want: Detection{HasChinese: true, ChineseChars: 25},
		},
		{
			name: "false json with zero count",
			raw:  `{"has_chinese_text": false, "chinese_char_count": 0}`,
			want: Detection{HasChinese: false, ChineseChars: 0},
		},
		{
			name: "wrapped in json markdown fence",
			raw:  "```json\n{\"has_chinese_text\": true, \"chinese_char_count\": 120}\n```",
			want: Detection{HasChinese: true, ChineseChars: 120},
		},
		{
			name: "missing char count defaults to zero",
			raw:  `{"has_chinese_text": true}`,
			want: Detection{HasChinese: true, ChineseChars: 0},
		},
		{
			name: "negative char count clamped to zero",
			raw:  `{"has_chinese_text": true, "chinese_char_count": -5}`,
			want: Detection{HasChinese: true, ChineseChars: 0},
		},
		{
			name: "invalid json returns false",
			raw:  `not valid json`,
			want: Detection{HasChinese: false, ChineseChars: 0},
		},
		{
			name: "empty response returns false",
			raw:  "",
			want: Detection{HasChinese: false, ChineseChars: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseChineseDetectionJSON(tt.raw); got != tt.want {
				t.Errorf("parseChineseDetectionJSON(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestClosestAspectRatio(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		want   string
	}{
		{name: "exact square", width: 1000, height: 1000, want: "1:1"},
		{name: "near square diagram 2880x2570", width: 2880, height: 2570, want: "1:1"},
		{name: "exact 4:3", width: 1600, height: 1200, want: "4:3"},
		{name: "exact 3:4", width: 1200, height: 1600, want: "3:4"},
		{name: "exact 3:2", width: 1500, height: 1000, want: "3:2"},
		{name: "exact 2:3", width: 1000, height: 1500, want: "2:3"},
		{name: "exact 16:9", width: 1920, height: 1080, want: "16:9"},
		{name: "exact 9:16", width: 1080, height: 1920, want: "9:16"},
		{name: "exact 21:9", width: 2100, height: 900, want: "21:9"},
		{name: "ultrawide 3440x1440", width: 3440, height: 1440, want: "21:9"},
		{name: "zero width returns empty", width: 0, height: 100, want: ""},
		{name: "negative height returns empty", width: 100, height: -50, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := closestAspectRatio(tt.width, tt.height); got != tt.want {
				t.Errorf("closestAspectRatio(%d, %d) = %q, want %q", tt.width, tt.height, got, tt.want)
			}
		})
	}
}

func TestDetectAspectRatio(t *testing.T) {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, image.NewRGBA(image.Rect(0, 0, 160, 90))); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	if got := detectAspectRatio(pngBuf.Bytes()); got != "16:9" {
		t.Errorf("detectAspectRatio(png 160x90) = %q, want 16:9", got)
	}

	var jpgBuf bytes.Buffer
	if err := jpeg.Encode(&jpgBuf, image.NewRGBA(image.Rect(0, 0, 120, 160)), nil); err != nil {
		t.Fatalf("failed to encode jpeg: %v", err)
	}
	if got := detectAspectRatio(jpgBuf.Bytes()); got != "3:4" {
		t.Errorf("detectAspectRatio(jpeg 120x160) = %q, want 3:4", got)
	}

	// Undecodable bytes (e.g., webp without decoder) should return empty string.
	if got := detectAspectRatio([]byte("RIFF\x00\x00\x00\x00WEBP")); got != "" {
		t.Errorf("detectAspectRatio(webp) = %q, want empty", got)
	}
}

func TestClientSetImageSize(t *testing.T) {
	c := NewClient("test-key")
	if got := c.ImageSize(); got != DefaultImageSize {
		t.Fatalf("default ImageSize() = %q, want %q", got, DefaultImageSize)
	}

	for _, valid := range []string{"1K", "2k", " 4K "} {
		if err := c.SetImageSize(valid); err != nil {
			t.Errorf("SetImageSize(%q) unexpected error: %v", valid, err)
		}
	}
	if got := c.ImageSize(); got != "4K" {
		t.Errorf("ImageSize() after setting 4K = %q, want 4K", got)
	}

	if err := c.SetImageSize("8K"); err == nil {
		t.Error("SetImageSize(\"8K\") expected error, got nil")
	}
	if got := c.ImageSize(); got != "4K" {
		t.Errorf("ImageSize() should remain unchanged after invalid input, got %q", got)
	}
}

func TestParseVerificationJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Verification
	}{
		{
			name: "passing verification",
			raw:  `{"has_chinese_text": false, "preserves_content": true, "reason": "all elements preserved"}`,
			want: Verification{HasChineseText: false, PreservesContent: true, Reason: "all elements preserved"},
		},
		{
			name: "still has Chinese text",
			raw:  `{"has_chinese_text": true, "preserves_content": true, "reason": "Chinese label remains"}`,
			want: Verification{HasChineseText: true, PreservesContent: true, Reason: "Chinese label remains"},
		},
		{
			name: "missing content",
			raw:  `{"has_chinese_text": false, "preserves_content": false, "reason": "right sidebar removed"}`,
			want: Verification{HasChineseText: false, PreservesContent: false, Reason: "right sidebar removed"},
		},
		{
			name: "wrapped in json markdown fence",
			raw:  "```json\n{\"has_chinese_text\": false, \"preserves_content\": true, \"reason\": \"ok\"}\n```",
			want: Verification{HasChineseText: false, PreservesContent: true, Reason: "ok"},
		},
		{
			name: "invalid json fails conservatively",
			raw:  `not valid json`,
			want: Verification{HasChineseText: false, PreservesContent: false, Reason: "invalid verification JSON"},
		},
		{
			name: "empty response fails conservatively",
			raw:  "",
			want: Verification{HasChineseText: false, PreservesContent: false, Reason: "invalid verification JSON"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseVerificationJSON(tt.raw); got != tt.want {
				t.Errorf("parseVerificationJSON(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}
