package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultQwenEndpoint targets a locally running Qwen3-ASR server that exposes
	// the OpenAI-compatible audio transcriptions endpoint.
	// Example: docker run -p 17003:8000 quantatrisk/qwen3-asr:cpu-latest
	DefaultQwenEndpoint = "http://localhost:17003/v1/audio/transcriptions"
	DefaultQwenModel    = "qwen3-asr-0.6b"
)

// QwenOptions configures a local Qwen ASR transcription request.
type QwenOptions struct {
	Endpoint     string
	APIKey       string
	Model        string
	Language     string
	SystemPrompt string
	HTTPClient   *http.Client
}

// Transcript is the normalized result returned by an ASR provider.
type Transcript struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
	Model    string `json:"model,omitempty"`
	Seconds  int    `json:"seconds,omitempty"`
}

type qwenTranscriptionResponse struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
	Model    string `json:"model,omitempty"`
	Duration int    `json:"duration,omitempty"`
	Seconds  int    `json:"seconds,omitempty"`
	Error    *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// TranscribeQwen transcribes a local audio file or public audio URL with a
// locally hosted Qwen ASR OpenAI-compatible server.
func TranscribeQwen(ctx context.Context, input string, opts QwenOptions) (Transcript, error) {
	if strings.TrimSpace(input) == "" {
		return Transcript{}, errors.New("audio input is required")
	}
	if opts.Endpoint == "" {
		opts.Endpoint = DefaultQwenEndpoint
	}
	if opts.Model == "" {
		opts.Model = DefaultQwenModel
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}

	body, contentType, err := buildTranscriptionRequest(input, opts)
	if err != nil {
		return Transcript{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.Endpoint, body)
	if err != nil {
		return Transcript{}, err
	}
	if opts.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := client.Do(req)
	if err != nil {
		return Transcript{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Transcript{}, fmt.Errorf("read qwen response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var decoded qwenTranscriptionResponse
		if json.Unmarshal(respBody, &decoded) == nil && decoded.Error != nil && decoded.Error.Message != "" {
			return Transcript{}, fmt.Errorf("qwen asr failed (%s): %s", resp.Status, decoded.Error.Message)
		}
		return Transcript{}, fmt.Errorf("qwen asr failed: %s", resp.Status)
	}

	contentType = strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/plain") {
		return Transcript{Text: strings.TrimSpace(string(respBody)), Model: opts.Model}, nil
	}

	var decoded qwenTranscriptionResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return Transcript{}, fmt.Errorf("decode qwen response: %w", err)
	}
	if decoded.Text == "" {
		return Transcript{}, errors.New("qwen asr response did not contain transcript text")
	}
	seconds := decoded.Seconds
	if seconds == 0 {
		seconds = decoded.Duration
	}
	model := decoded.Model
	if model == "" {
		model = opts.Model
	}
	return Transcript{
		Text:     decoded.Text,
		Language: decoded.Language,
		Model:    model,
		Seconds:  seconds,
	}, nil
}

func buildTranscriptionRequest(input string, opts QwenOptions) (io.Reader, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("model", opts.Model); err != nil {
		return nil, "", err
	}
	if err := writer.WriteField("response_format", "verbose_json"); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(opts.Language) != "" {
		if err := writer.WriteField("language", strings.TrimSpace(opts.Language)); err != nil {
			return nil, "", err
		}
	}
	if strings.TrimSpace(opts.SystemPrompt) != "" {
		if err := writer.WriteField("prompt", strings.TrimSpace(opts.SystemPrompt)); err != nil {
			return nil, "", err
		}
	}

	if isRemoteURL(input) {
		if err := writer.WriteField("audio_address", input); err != nil {
			return nil, "", err
		}
	} else {
		file, err := os.Open(input)
		if err != nil {
			return nil, "", fmt.Errorf("open audio file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("file", filepath.Base(input))
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, file); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &body, writer.FormDataContentType(), nil
}

func isRemoteURL(input string) bool {
	parsed, err := url.Parse(input)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}
