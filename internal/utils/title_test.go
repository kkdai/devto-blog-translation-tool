package utils

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateAtWordBoundary(t *testing.T) {
	exact128 := strings.Repeat("a", 128)
	noSpaceOver128 := strings.Repeat("x", 150)
	// 120 chars + " Take Over External Projects" -> cuts before "External" (which would be split at "Exte")
	longTitleWithWords := "[Notes] " + strings.Repeat("word ", 21) + "to Review and Take Over External Projects"

	tests := []struct {
		name   string
		title  string
		maxLen int
		want   string
	}{
		{
			name:   "exact 128 runes unchanged",
			title:  exact128,
			maxLen: 128,
			want:   exact128,
		},
		{
			name:   "shorter than 128 unchanged",
			title:  "Short Title: Keep Trailing,",
			maxLen: 128,
			want:   "Short Title: Keep Trailing,",
		},
		{
			name:   "exceeds 128 with spaces cuts at last word boundary",
			title:  longTitleWithWords,
			maxLen: 128,
			want:   "[Notes] " + strings.Repeat("word ", 21) + "to Review and",
		},
		{
			name:   "boundary falls right before space keeps full word",
			title:  "hello world foo",
			maxLen: 11,
			want:   "hello world",
		},
		{
			name:   "exceeds 128 without spaces hard cuts at 128",
			title:  noSpaceOver128,
			maxLen: 128,
			want:   strings.Repeat("x", 128),
		},
		{
			name:   "only leading spaces before long word falls back to hard cut",
			title:  "   " + strings.Repeat("z", 20),
			maxLen: 10,
			want:   "   zzzzzzz",
		},
		{
			name:   "strips trailing comma",
			title:  "Understanding Go, Concurrency, Channels, and Select Statements",
			maxLen: 35,
			want:   "Understanding Go, Concurrency",
		},
		{
			name:   "strips trailing colon",
			title:  "Go Guide: Building Reliable Microservices",
			maxLen: 15,
			want:   "Go Guide",
		},
		{
			name:   "strips trailing semicolon",
			title:  "First Part; Second Part Is Very Long",
			maxLen: 16,
			want:   "First Part",
		},
		{
			name:   "strips trailing hyphen and space",
			title:  "Kubernetes Networking - Deep Dive Analysis",
			maxLen: 26,
			want:   "Kubernetes Networking",
		},
		{
			name:   "strips trailing en-dash",
			title:  "System Design – Scaling Distributed Queues",
			maxLen: 18,
			want:   "System Design",
		},
		{
			name:   "strips trailing em-dash",
			title:  "System Design — Scaling Distributed Queues",
			maxLen: 18,
			want:   "System Design",
		},
		{
			name:   "mixed Chinese and English rune count with space",
			title:  strings.Repeat("中", 100) + " " + strings.Repeat("a", 40),
			maxLen: 128,
			want:   strings.Repeat("中", 100),
		},
		{
			name:   "mixed Chinese and English rune count without space hard cuts by rune",
			title:  strings.Repeat("中", 100) + strings.Repeat("a", 40),
			maxLen: 128,
			want:   strings.Repeat("中", 100) + strings.Repeat("a", 28),
		},
		{
			name:   "mixed Chinese and English exactly 128 runes unchanged",
			title:  strings.Repeat("中", 64) + strings.Repeat("a", 64),
			maxLen: 128,
			want:   strings.Repeat("中", 64) + strings.Repeat("a", 64),
		},
		{
			name:   "non-positive maxLen returns empty",
			title:  "Hello World",
			maxLen: 0,
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateAtWordBoundary(tc.title, tc.maxLen)
			if got != tc.want {
				t.Errorf("TruncateAtWordBoundary(%q, %d) = %q (len=%d runes), want %q (len=%d runes)",
					tc.title, tc.maxLen, got, utf8.RuneCountInString(got), tc.want, utf8.RuneCountInString(tc.want))
			}
			if tc.maxLen > 0 && utf8.RuneCountInString(got) > tc.maxLen {
				t.Errorf("result length %d runes exceeds maxLen %d", utf8.RuneCountInString(got), tc.maxLen)
			}
		})
	}
}
