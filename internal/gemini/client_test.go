package gemini

import (
	"errors"
	"testing"
)

func TestClientModels(t *testing.T) {
	c := NewClient("test-key")
	if got := c.TextModel(); got != DefaultTextModel {
		t.Errorf("default TextModel() = %q, want %q", got, DefaultTextModel)
	}
	if DefaultTextModel != "gemini-4-neon" {
		t.Errorf("DefaultTextModel = %q, want %q", DefaultTextModel, "gemini-4-neon")
	}
	if got := c.ImageModel(); got != DefaultImageModel {
		t.Errorf("default ImageModel() = %q, want %q", got, DefaultImageModel)
	}
	if DefaultImageModel != "gemini-3-pro-image" {
		t.Errorf("DefaultImageModel = %q, want %q", DefaultImageModel, "gemini-3-pro-image")
	}

	c.SetTextModel("custom-text-model")
	if got := c.TextModel(); got != "custom-text-model" {
		t.Errorf("TextModel() after SetTextModel = %q, want %q", got, "custom-text-model")
	}

	c.SetTextModel("")
	if got := c.TextModel(); got != "custom-text-model" {
		t.Errorf("TextModel() after empty SetTextModel = %q, want %q", got, "custom-text-model")
	}
}

func TestRetryOnceOnMaxTokens(t *testing.T) {
	t.Run("first attempt succeeds", func(t *testing.T) {
		calls := 0
		got, err := retryOnceOnMaxTokens(func() (string, bool, error) {
			calls++
			return "translated text", false, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "translated text" {
			t.Errorf("got %q, want %q", got, "translated text")
		}
		if calls != 1 {
			t.Errorf("expected 1 call, got %d", calls)
		}
	})

	t.Run("first attempt fails retryable, second succeeds", func(t *testing.T) {
		calls := 0
		got, err := retryOnceOnMaxTokens(func() (string, bool, error) {
			calls++
			if calls == 1 {
				return "", true, errors.New("translation truncated: hit MaxOutputTokens")
			}
			return "translated on retry", false, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "translated on retry" {
			t.Errorf("got %q, want %q", got, "translated on retry")
		}
		if calls != 2 {
			t.Errorf("expected 2 calls, got %d", calls)
		}
	})

	t.Run("both attempts fail retryable", func(t *testing.T) {
		calls := 0
		wantErr := errors.New("translation truncated: hit MaxOutputTokens")
		_, err := retryOnceOnMaxTokens(func() (string, bool, error) {
			calls++
			return "", true, wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
		if calls != 2 {
			t.Errorf("expected 2 calls, got %d", calls)
		}
	})

	t.Run("non-retryable error does not retry", func(t *testing.T) {
		calls := 0
		wantErr := errors.New("invalid api key")
		_, err := retryOnceOnMaxTokens(func() (string, bool, error) {
			calls++
			return "", false, wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
		if calls != 1 {
			t.Errorf("expected 1 call, got %d", calls)
		}
	})
}
