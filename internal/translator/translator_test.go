package translator

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFitTitleToLimit(t *testing.T) {
	longTitle := "[Notes] How to Build a Comprehensive Automated Workflow to Review and Take Over External Open Source Projects with AI Agents Every Day"
	if utf8.RuneCountInString(longTitle) <= maxTitleLen {
		t.Fatalf("test setup error: longTitle length %d must exceed %d", utf8.RuneCountInString(longTitle), maxTitleLen)
	}

	t.Run("within limit returns unchanged without calling shortener", func(t *testing.T) {
		shortTitle := "[Notes] Concise English Title"
		calls := 0
		got, status := fitTitleToLimit(context.Background(), shortTitle, maxTitleLen, func(ctx context.Context, title string, maxLen int) (string, error) {
			calls++
			return "should not be called", nil
		})
		if status != titleFitUnchanged {
			t.Errorf("status = %v, want titleFitUnchanged", status)
		}
		if got != shortTitle {
			t.Errorf("got = %q, want %q", got, shortTitle)
		}
		if calls != 0 {
			t.Errorf("calls = %d, want 0", calls)
		}
	})

	t.Run("shorten succeeds on first attempt", func(t *testing.T) {
		calls := 0
		want := "[Notes] Automating External OSS Project Reviews with AI Agents"
		got, status := fitTitleToLimit(context.Background(), longTitle, maxTitleLen, func(ctx context.Context, title string, maxLen int) (string, error) {
			calls++
			return "  " + want + "\n", nil
		})
		if status != titleFitShortened {
			t.Errorf("status = %v, want titleFitShortened", status)
		}
		if got != want {
			t.Errorf("got = %q, want %q", got, want)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
	})

	t.Run("shorten too long on first attempt succeeds on second attempt", func(t *testing.T) {
		calls := 0
		want := "[Notes] Automated AI Workflow to Review External OSS Projects"
		got, status := fitTitleToLimit(context.Background(), longTitle, maxTitleLen, func(ctx context.Context, title string, maxLen int) (string, error) {
			calls++
			if calls == 1 {
				return strings.Repeat("word ", 30), nil
			}
			return want, nil
		})
		if status != titleFitShortened {
			t.Errorf("status = %v, want titleFitShortened", status)
		}
		if got != want {
			t.Errorf("got = %q, want %q", got, want)
		}
		if calls != 2 {
			t.Errorf("calls = %d, want 2", calls)
		}
	})

	t.Run("shorten still too long after 2 attempts falls back to word boundary", func(t *testing.T) {
		calls := 0
		got, status := fitTitleToLimit(context.Background(), longTitle, maxTitleLen, func(ctx context.Context, title string, maxLen int) (string, error) {
			calls++
			return strings.Repeat("still too long ", 15), nil
		})
		if status != titleFitTruncated {
			t.Errorf("status = %v, want titleFitTruncated", status)
		}
		if calls != maxShortenTitleAttempts {
			t.Errorf("calls = %d, want %d", calls, maxShortenTitleAttempts)
		}
		want := "[Notes] How to Build a Comprehensive Automated Workflow to Review and Take Over External Open Source Projects with AI Agents"
		if got != want {
			t.Errorf("got = %q (len=%d), want %q (len=%d)", got, utf8.RuneCountInString(got), want, utf8.RuneCountInString(want))
		}
		if utf8.RuneCountInString(got) > maxTitleLen {
			t.Errorf("result length %d exceeds %d", utf8.RuneCountInString(got), maxTitleLen)
		}
	})

	t.Run("shorten error falls back to word boundary", func(t *testing.T) {
		calls := 0
		got, status := fitTitleToLimit(context.Background(), longTitle, maxTitleLen, func(ctx context.Context, title string, maxLen int) (string, error) {
			calls++
			return "", errors.New("api error")
		})
		if status != titleFitTruncated {
			t.Errorf("status = %v, want titleFitTruncated", status)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
		want := "[Notes] How to Build a Comprehensive Automated Workflow to Review and Take Over External Open Source Projects with AI Agents"
		if got != want {
			t.Errorf("got = %q, want %q", got, want)
		}
	})

	t.Run("shorten returns blank string falls back to word boundary", func(t *testing.T) {
		calls := 0
		got, status := fitTitleToLimit(context.Background(), longTitle, maxTitleLen, func(ctx context.Context, title string, maxLen int) (string, error) {
			calls++
			return "   \t\n ", nil
		})
		if status != titleFitTruncated {
			t.Errorf("status = %v, want titleFitTruncated", status)
		}
		if calls != maxShortenTitleAttempts {
			t.Errorf("calls = %d, want %d", calls, maxShortenTitleAttempts)
		}
		if utf8.RuneCountInString(got) > maxTitleLen || got == "" {
			t.Errorf("unexpected fallback result: %q", got)
		}
	})
}
