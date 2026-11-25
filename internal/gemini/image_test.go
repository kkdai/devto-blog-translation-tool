package gemini

import "testing"

func TestParseChineseDetectionJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{
			name: "true json",
			raw:  `{"has_chinese_text": true}`,
			want: true,
		},
		{
			name: "false json",
			raw:  `{"has_chinese_text": false}`,
			want: false,
		},
		{
			name: "wrapped in json markdown fence",
			raw:  "```json\n{\"has_chinese_text\": true}\n```",
			want: true,
		},
		{
			name: "invalid json returns false",
			raw:  `not valid json`,
			want: false,
		},
		{
			name: "empty response returns false",
			raw:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseChineseDetectionJSON(tt.raw); got != tt.want {
				t.Errorf("parseChineseDetectionJSON(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
