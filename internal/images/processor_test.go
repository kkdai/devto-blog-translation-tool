package images

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/al03034132/devto-blog-translation-tool/internal/gemini"
)

type mockDetector struct {
	fn func(ctx context.Context, imageData []byte, mimeType string) (gemini.Detection, error)
}

func (m *mockDetector) DetectChineseText(ctx context.Context, imageData []byte, mimeType string) (gemini.Detection, error) {
	return m.fn(ctx, imageData, mimeType)
}

type mockTranslator struct {
	fn func(ctx context.Context, imageData []byte, mimeType string) ([]byte, string, error)
}

func (m *mockTranslator) TranslateImage(ctx context.Context, imageData []byte, mimeType string) ([]byte, string, error) {
	return m.fn(ctx, imageData, mimeType)
}

type mockVerifier struct {
	fn func(ctx context.Context, original []byte, originalMIME string, generated []byte, generatedMIME string) (gemini.Verification, error)
}

func (m *mockVerifier) VerifyTranslatedImage(ctx context.Context, original []byte, originalMIME string, generated []byte, generatedMIME string) (gemini.Verification, error) {
	return m.fn(ctx, original, originalMIME, generated, generatedMIME)
}

type mockUploader struct {
	fn func(ctx context.Context, objectName string, data []byte, contentType string) (string, error)
}

func (m *mockUploader) Upload(ctx context.Context, objectName string, data []byte, contentType string) (string, error) {
	return m.fn(ctx, objectName, data, contentType)
}

func TestProcessor_ProcessMarkdown(t *testing.T) {
	fakePNG := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 1, 2, 3}
	fakeGIF := []byte("GIF89a")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chinese.png", "/english.png", "/fail-gen.png", "/dense.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(fakePNG)
		case "/anim.gif":
			w.Header().Set("Content-Type", "image/gif")
			_, _ = w.Write(fakeGIF)
		case "/not-image.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("hello"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Run("replaces image when verification passes", func(t *testing.T) {
		var uploadedObject, uploadedContentType string
		detectCalls := 0
		translateCalls := 0
		verifyCalls := 0

		proc := NewProcessor(
			&mockDetector{fn: func(_ context.Context, _ []byte, _ string) (gemini.Detection, error) {
				detectCalls++
				return gemini.Detection{HasChinese: true, ChineseChars: 20}, nil
			}},
			&mockTranslator{fn: func(_ context.Context, _ []byte, _ string) ([]byte, string, error) {
				translateCalls++
				return []byte("english-image-bytes"), "image/png", nil
			}},
			&mockVerifier{fn: func(_ context.Context, orig []byte, origMIME string, gen []byte, genMIME string) (gemini.Verification, error) {
				verifyCalls++
				if len(orig) == 0 || origMIME != "image/png" || string(gen) != "english-image-bytes" || genMIME != "image/png" {
					t.Errorf("unexpected VerifyTranslatedImage args: origLen=%d origMIME=%q gen=%q genMIME=%q", len(orig), origMIME, string(gen), genMIME)
				}
				return gemini.Verification{HasChineseText: false, PreservesContent: true, Reason: "ok"}, nil
			}},
			&mockUploader{fn: func(_ context.Context, objectName string, _ []byte, contentType string) (string, error) {
				uploadedObject = objectName
				uploadedContentType = contentType
				return "https://storage.googleapis.com/test-bucket/" + objectName, nil
			}},
		).WithHTTPClient(server.Client())

		imgURL := server.URL + "/chinese.png"
		md := "Here is an image:\n![arch](" + imgURL + ")\nAnd again: <img src=\"" + imgURL + "\">"
		got := proc.ProcessMarkdown(context.Background(), 12345, md)

		if detectCalls != 1 {
			t.Errorf("expected 1 detect call for duplicate URL, got %d", detectCalls)
		}
		if translateCalls != 1 {
			t.Errorf("expected 1 translate call, got %d", translateCalls)
		}
		if verifyCalls != 1 {
			t.Errorf("expected 1 verify call, got %d", verifyCalls)
		}
		if !strings.HasPrefix(uploadedObject, "devto-images/12345/") || !strings.HasSuffix(uploadedObject, "-en.png") {
			t.Errorf("unexpected object name: %q", uploadedObject)
		}
		if uploadedContentType != "image/png" {
			t.Errorf("unexpected content type: %q", uploadedContentType)
		}
		expectedURL := "https://storage.googleapis.com/test-bucket/" + uploadedObject
		want := "Here is an image:\n![arch](" + expectedURL + ")\nAnd again: <img src=\"" + expectedURL + "\">"
		if got != want {
			t.Errorf("ProcessMarkdown() =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("keeps original image and skips upload when generated image still has Chinese text", func(t *testing.T) {
		detectCalls := 0
		translateCalls := 0
		verifyCalls := 0
		uploadCalled := false

		proc := NewProcessor(
			&mockDetector{fn: func(_ context.Context, _ []byte, _ string) (gemini.Detection, error) {
				detectCalls++
				return gemini.Detection{HasChinese: true, ChineseChars: 15}, nil
			}},
			&mockTranslator{fn: func(_ context.Context, _ []byte, _ string) ([]byte, string, error) {
				translateCalls++
				return []byte("garbled-image-bytes"), "image/png", nil
			}},
			&mockVerifier{fn: func(_ context.Context, _ []byte, _ string, _ []byte, _ string) (gemini.Verification, error) {
				verifyCalls++
				return gemini.Verification{HasChineseText: true, PreservesContent: true, Reason: "Chinese text remains"}, nil
			}},
			&mockUploader{fn: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
				uploadCalled = true
				return "", nil
			}},
		).WithHTTPClient(server.Client())

		md := "![arch](" + server.URL + "/chinese.png)"
		got := proc.ProcessMarkdown(context.Background(), 42, md)

		if got != md {
			t.Errorf("expected original markdown preserved, got %q", got)
		}
		if detectCalls != 1 {
			t.Errorf("expected 1 detect call, got %d", detectCalls)
		}
		if translateCalls != 1 {
			t.Errorf("expected 1 translate call, got %d", translateCalls)
		}
		if verifyCalls != 1 {
			t.Errorf("expected 1 verify call, got %d", verifyCalls)
		}
		if uploadCalled {
			t.Error("expected Upload not to be called when verification detects Chinese text")
		}
	})

	t.Run("keeps original image and skips upload when generated image loses content", func(t *testing.T) {
		verifyCalls := 0
		uploadCalled := false

		proc := NewProcessor(
			&mockDetector{fn: func(_ context.Context, _ []byte, _ string) (gemini.Detection, error) {
				return gemini.Detection{HasChinese: true, ChineseChars: 12}, nil
			}},
			&mockTranslator{fn: func(_ context.Context, _ []byte, _ string) ([]byte, string, error) {
				return []byte("cropped-image-bytes"), "image/png", nil
			}},
			&mockVerifier{fn: func(_ context.Context, _ []byte, _ string, _ []byte, _ string) (gemini.Verification, error) {
				verifyCalls++
				return gemini.Verification{HasChineseText: false, PreservesContent: false, Reason: "missing dialog panel"}, nil
			}},
			&mockUploader{fn: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
				uploadCalled = true
				return "", nil
			}},
		).WithHTTPClient(server.Client())

		md := "![arch](" + server.URL + "/chinese.png)"
		got := proc.ProcessMarkdown(context.Background(), 42, md)

		if got != md {
			t.Errorf("expected original markdown preserved when content is lost, got %q", got)
		}
		if verifyCalls != 1 {
			t.Errorf("expected 1 verify call, got %d", verifyCalls)
		}
		if uploadCalled {
			t.Error("expected Upload not to be called when generated image loses content")
		}
	})

	t.Run("keeps original image when post-generation verification errors", func(t *testing.T) {
		uploadCalled := false

		proc := NewProcessor(
			&mockDetector{fn: func(_ context.Context, _ []byte, _ string) (gemini.Detection, error) {
				return gemini.Detection{HasChinese: true, ChineseChars: 10}, nil
			}},
			&mockTranslator{fn: func(_ context.Context, _ []byte, _ string) ([]byte, string, error) {
				return []byte("english-image-bytes"), "image/png", nil
			}},
			&mockVerifier{fn: func(_ context.Context, _ []byte, _ string, _ []byte, _ string) (gemini.Verification, error) {
				return gemini.Verification{}, errors.New("verify timeout")
			}},
			&mockUploader{fn: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
				uploadCalled = true
				return "", nil
			}},
		).WithHTTPClient(server.Client())

		md := "![arch](" + server.URL + "/chinese.png)"
		got := proc.ProcessMarkdown(context.Background(), 42, md)

		if got != md {
			t.Errorf("expected original markdown preserved on verification error, got %q", got)
		}
		if uploadCalled {
			t.Error("expected Upload not to be called when verification errors")
		}
	})

	t.Run("skips image when Chinese character count exceeds threshold", func(t *testing.T) {
		translateCalled := false
		uploadCalled := false

		proc := NewProcessor(
			&mockDetector{fn: func(_ context.Context, _ []byte, _ string) (gemini.Detection, error) {
				return gemini.Detection{HasChinese: true, ChineseChars: 80}, nil
			}},
			&mockTranslator{fn: func(_ context.Context, _ []byte, _ string) ([]byte, string, error) {
				translateCalled = true
				return nil, "", nil
			}},
			&mockVerifier{fn: func(_ context.Context, _ []byte, _ string, _ []byte, _ string) (gemini.Verification, error) {
				return gemini.Verification{HasChineseText: false, PreservesContent: true}, nil
			}},
			&mockUploader{fn: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
				uploadCalled = true
				return "", nil
			}},
		).WithHTTPClient(server.Client()).WithMaxChineseChars(50)

		md := "![dense](" + server.URL + "/dense.png)"
		got := proc.ProcessMarkdown(context.Background(), 42, md)

		if got != md {
			t.Errorf("expected markdown unchanged for text-dense image, got %q", got)
		}
		if translateCalled {
			t.Error("expected TranslateImage not to be called for text-dense image")
		}
		if uploadCalled {
			t.Error("expected Upload not to be called for text-dense image")
		}
	})

	t.Run("keeps image unchanged when no Chinese text", func(t *testing.T) {
		translateCalled := false
		uploadCalled := false

		proc := NewProcessor(
			&mockDetector{fn: func(_ context.Context, _ []byte, _ string) (gemini.Detection, error) {
				return gemini.Detection{HasChinese: false, ChineseChars: 0}, nil
			}},
			&mockTranslator{fn: func(_ context.Context, _ []byte, _ string) ([]byte, string, error) {
				translateCalled = true
				return nil, "", nil
			}},
			&mockVerifier{fn: func(_ context.Context, _ []byte, _ string, _ []byte, _ string) (gemini.Verification, error) {
				return gemini.Verification{HasChineseText: false, PreservesContent: true}, nil
			}},
			&mockUploader{fn: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
				uploadCalled = true
				return "", nil
			}},
		).WithHTTPClient(server.Client())

		md := "![eng](" + server.URL + "/english.png)"
		got := proc.ProcessMarkdown(context.Background(), 42, md)

		if got != md {
			t.Errorf("expected markdown unchanged, got %q", got)
		}
		if translateCalled {
			t.Error("expected TranslateImage not to be called")
		}
		if uploadCalled {
			t.Error("expected Upload not to be called")
		}
	})

	t.Run("keeps original image when generation fails", func(t *testing.T) {
		uploadCalled := false

		proc := NewProcessor(
			&mockDetector{fn: func(_ context.Context, _ []byte, _ string) (gemini.Detection, error) {
				return gemini.Detection{HasChinese: true, ChineseChars: 10}, nil
			}},
			&mockTranslator{fn: func(_ context.Context, _ []byte, _ string) ([]byte, string, error) {
				return nil, "", errors.New("quota exceeded")
			}},
			&mockVerifier{fn: func(_ context.Context, _ []byte, _ string, _ []byte, _ string) (gemini.Verification, error) {
				return gemini.Verification{HasChineseText: false, PreservesContent: true}, nil
			}},
			&mockUploader{fn: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
				uploadCalled = true
				return "", nil
			}},
		).WithHTTPClient(server.Client())

		md := "![fail](" + server.URL + "/fail-gen.png)"
		got := proc.ProcessMarkdown(context.Background(), 42, md)

		if got != md {
			t.Errorf("expected original markdown preserved on failure, got %q", got)
		}
		if uploadCalled {
			t.Error("expected Upload not to be called when generation fails")
		}
	})

	t.Run("skips GIF and non-image responses without failing", func(t *testing.T) {
		detectCalled := false
		proc := NewProcessor(
			&mockDetector{fn: func(_ context.Context, _ []byte, _ string) (gemini.Detection, error) {
				detectCalled = true
				return gemini.Detection{HasChinese: true, ChineseChars: 5}, nil
			}},
			&mockTranslator{fn: func(_ context.Context, _ []byte, _ string) ([]byte, string, error) {
				return nil, "", nil
			}},
			&mockVerifier{fn: func(_ context.Context, _ []byte, _ string, _ []byte, _ string) (gemini.Verification, error) {
				return gemini.Verification{HasChineseText: false, PreservesContent: true}, nil
			}},
			&mockUploader{fn: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
				return "", nil
			}},
		).WithHTTPClient(server.Client())

		md := "![g](" + server.URL + "/anim.gif) ![t](" + server.URL + "/not-image.txt)"
		got := proc.ProcessMarkdown(context.Background(), 42, md)

		if got != md {
			t.Errorf("expected markdown unchanged, got %q", got)
		}
		if detectCalled {
			t.Error("expected detector not to be called for GIF or text/plain")
		}
	})
}

func TestBuildObjectName(t *testing.T) {
	got := BuildObjectName(999, "https://example.com/pic.png", "image/jpeg")
	if !strings.HasPrefix(got, "devto-images/999/") || !strings.HasSuffix(got, "-en.jpg") {
		t.Errorf("BuildObjectName() = %q", got)
	}
	// Verify hash segment is 16 hex chars: "devto-images/999/" + 16 + "-en.jpg"
	trimmed := strings.TrimPrefix(got, "devto-images/999/")
	trimmed = strings.TrimSuffix(trimmed, "-en.jpg")
	if len(trimmed) != 16 {
		t.Errorf("expected 16-char hash, got %q (len %d)", trimmed, len(trimmed))
	}
}
