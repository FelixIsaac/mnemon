package asr

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestTranscribeQwenLocalFile(t *testing.T) {
	path := t.TempDir() + "/sample.mp3"
	if err := os.WriteFile(path, []byte("fake-audio"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer local-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Fatalf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("model"); got != DefaultQwenModel {
			t.Fatalf("model = %q", got)
		}
		if got := r.FormValue("language"); got != "en" {
			t.Fatalf("language = %q", got)
		}
		if got := r.FormValue("prompt"); got != "names: Mnemon" {
			t.Fatalf("prompt = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		if header.Filename != "sample.mp3" {
			t.Fatalf("filename = %q", header.Filename)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(qwenTranscriptionResponse{
			Text:     "hello mnemon",
			Language: "en",
			Model:    DefaultQwenModel,
			Duration: 2.4,
		})
	}))
	defer server.Close()

	got, err := TranscribeQwen(context.Background(), path, QwenOptions{
		Endpoint:     server.URL,
		APIKey:       "local-token",
		Language:     "en",
		SystemPrompt: "names: Mnemon",
	})
	if err != nil {
		t.Fatalf("TranscribeQwen error: %v", err)
	}
	if got.Text != "hello mnemon" || got.Language != "en" || got.Seconds != 3 {
		t.Fatalf("unexpected transcript: %#v", got)
	}
}

func TestTranscribeQwenRemoteURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("audio_address"); got != "https://example.com/audio.mp3" {
			t.Fatalf("audio_address = %q", got)
		}
		if _, _, err := r.FormFile("file"); err != http.ErrMissingFile {
			t.Fatalf("expected no file upload, got err=%v", err)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("remote transcript"))
	}))
	defer server.Close()

	got, err := TranscribeQwen(context.Background(), "https://example.com/audio.mp3", QwenOptions{Endpoint: server.URL})
	if err != nil {
		t.Fatalf("TranscribeQwen error: %v", err)
	}
	if got.Text != "remote transcript" {
		t.Fatalf("unexpected transcript: %#v", got)
	}
}

func TestBuildTranscriptionRequest(t *testing.T) {
	path := t.TempDir() + "/sample.wav"
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	body, contentType, err := buildTranscriptionRequest(path, QwenOptions{Model: "qwen3-asr-0.6b"})
	if err != nil {
		t.Fatalf("buildTranscriptionRequest error: %v", err)
	}
	reader := multipart.NewReader(body, strings.TrimPrefix(contentType, "multipart/form-data; boundary="))
	form, err := reader.ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}
	if got := form.Value["model"][0]; got != "qwen3-asr-0.6b" {
		t.Fatalf("model = %q", got)
	}
	if len(form.File["file"]) != 1 {
		t.Fatalf("file count = %d", len(form.File["file"]))
	}
}
