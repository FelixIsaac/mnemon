package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mnemon-dev/mnemon/internal/asr"
	"github.com/spf13/cobra"
)

var transcribeOpts struct {
	provider     string
	endpoint     string
	model        string
	apiKey       string
	language     string
	systemPrompt string
	jsonOutput   bool
	timeout      time.Duration
}

var transcribeCmd = &cobra.Command{
	Use:   "transcribe <audio-file-or-url>",
	Short: "Transcribe audio with locally hosted Qwen ASR",
	Long: `Transcribe a local audio file or public audio URL with a locally hosted
Qwen ASR OpenAI-compatible server.

Start a local server first, for example:

  docker run -p 17003:8000 quantatrisk/qwen3-asr:cpu-latest

Then run:

  mnemon transcribe ./voice-note.mp3`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if transcribeOpts.provider != "qwen" {
			return fmt.Errorf("unsupported ASR provider %q (supported: qwen)", transcribeOpts.provider)
		}

		ctx := cmd.Context()
		if transcribeOpts.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, transcribeOpts.timeout)
			defer cancel()
		}

		result, err := asr.TranscribeQwen(ctx, args[0], asr.QwenOptions{
			Endpoint:     transcribeOpts.endpoint,
			APIKey:       transcribeOpts.apiKey,
			Model:        transcribeOpts.model,
			Language:     transcribeOpts.language,
			SystemPrompt: transcribeOpts.systemPrompt,
		})
		if err != nil {
			return err
		}

		if transcribeOpts.jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}
		fmt.Fprintln(os.Stdout, result.Text)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(transcribeCmd)
	transcribeCmd.Flags().StringVar(&transcribeOpts.provider, "provider", "qwen", "ASR provider")
	transcribeCmd.Flags().StringVar(&transcribeOpts.endpoint, "endpoint", asr.DefaultQwenEndpoint, "local Qwen ASR OpenAI-compatible transcription endpoint")
	transcribeCmd.Flags().StringVar(&transcribeOpts.model, "model", asr.DefaultQwenModel, "Qwen ASR model")
	transcribeCmd.Flags().StringVar(&transcribeOpts.apiKey, "api-key", "", "optional bearer token for a protected local endpoint")
	transcribeCmd.Flags().StringVar(&transcribeOpts.language, "language", "", "optional source language hint, for example English or Chinese")
	transcribeCmd.Flags().StringVar(&transcribeOpts.systemPrompt, "system-prompt", "", "optional context text to bias recognition")
	transcribeCmd.Flags().BoolVar(&transcribeOpts.jsonOutput, "json", false, "print full transcript metadata as JSON")
	transcribeCmd.Flags().DurationVar(&transcribeOpts.timeout, "timeout", 5*time.Minute, "request timeout")
}
