package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"leveltalk/internal/dialogs"
)

const (
	defaultElevenLabsEndpoint = "https://api.elevenlabs.io/v1/text-to-speech/"
	defaultElevenLabsModel    = "eleven_multilingual_v2"
)

// ElevenLabsOptions configures optional client behavior.
type ElevenLabsOptions struct {
	BaseURL    string
	ModelID    string
	HTTPClient *http.Client
}

// ElevenLabsClient implements TTSClient using ElevenLabs' API.
type ElevenLabsClient struct {
	logger     *slog.Logger
	apiKey     string
	voiceID    string
	modelID    string
	httpClient *http.Client
	endpoint   string
}

// NewElevenLabsClient creates a new ElevenLabs TTS client.
func NewElevenLabsClient(logger *slog.Logger, apiKey, voiceID string, opts *ElevenLabsOptions) *ElevenLabsClient {
	if opts == nil {
		opts = &ElevenLabsOptions{}
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	modelID := opts.ModelID
	if modelID == "" {
		modelID = defaultElevenLabsModel
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultElevenLabsEndpoint
	}

	return &ElevenLabsClient{
		logger:     logger,
		apiKey:     apiKey,
		voiceID:    voiceID,
		modelID:    modelID,
		httpClient: httpClient,
		endpoint:   strings.TrimRight(baseURL, "/") + "/" + voiceID,
	}
}

type elevenLabsRequest struct {
	Text          string `json:"text"`
	ModelID       string `json:"model_id"`
	VoiceSettings struct {
		Stability       float64 `json:"stability"`
		SimilarityBoost float64 `json:"similarity_boost"`
	} `json:"voice_settings"`
}

// SynthesizeDialog converts each dialog turn into an audio data URL.
func (c *ElevenLabsClient) SynthesizeDialog(ctx context.Context, dlg dialogs.Dialog) (dialogs.Dialog, error) {
	c.logger.Info("starting ElevenLabs synthesis", slog.Int("turns", len(dlg.Turns)))

	for i := range dlg.Turns {
		if dlg.Turns[i].ID == uuid.Nil {
			dlg.Turns[i].ID = uuid.New()
		}

		c.logger.Debug("synthesizing turn",
			slog.Int("turn", i),
			slog.String("speaker", dlg.Turns[i].Speaker),
			slog.Int("text_length", len(dlg.Turns[i].Text)),
		)

		audio, err := c.synthesizeText(ctx, dlg.Turns[i].Text)
		if err != nil {
			c.logger.Error("elevenlabs synthesis failed",
				slog.Int("turn", i),
				slog.String("speaker", dlg.Turns[i].Speaker),
				slog.String("text", dlg.Turns[i].Text),
				slog.String("error", err.Error()),
			)
			return dialogs.Dialog{}, fmt.Errorf("elevenlabs synthesize turn %d: %w", i, err)
		}

		if len(audio) == 0 {
			c.logger.Warn("elevenlabs returned empty audio",
				slog.Int("turn", i),
				slog.String("speaker", dlg.Turns[i].Speaker),
			)
			dlg.Turns[i].AudioURL = fmt.Sprintf("/static/audio/placeholder.mp3?turn=%d", i)
			continue
		}

		c.logger.Debug("elevenlabs synthesis succeeded",
			slog.Int("turn", i),
			slog.Int("audio_bytes", len(audio)),
		)

		encoded := base64.StdEncoding.EncodeToString(audio)
		dlg.Turns[i].AudioURL = "data:audio/mpeg;base64," + encoded
		
		c.logger.Debug("audio data URL created",
			slog.Int("turn", i),
			slog.Int("url_length", len(dlg.Turns[i].AudioURL)),
		)
	}

	c.logger.Info("completed ElevenLabs synthesis", slog.Int("turns", len(dlg.Turns)))
	return dlg, nil
}

func (c *ElevenLabsClient) synthesizeText(ctx context.Context, text string) ([]byte, error) {
	reqBody := elevenLabsRequest{
		Text:    text,
		ModelID: c.modelID,
	}
	reqBody.VoiceSettings.Stability = 0.5
	reqBody.VoiceSettings.SimilarityBoost = 0.75

	payload, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.Error("failed to marshal ElevenLabs request", slog.String("error", err.Error()))
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		c.logger.Error("failed to create ElevenLabs request", slog.String("error", err.Error()))
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	c.logger.Debug("calling ElevenLabs API",
		slog.String("endpoint", c.endpoint),
		slog.String("voice_id", c.voiceID),
		slog.String("model_id", c.modelID),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("ElevenLabs HTTP request failed",
			slog.String("endpoint", c.endpoint),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("call elevenlabs: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("ElevenLabs response received",
		slog.Int("status_code", resp.StatusCode),
		slog.String("content_type", resp.Header.Get("Content-Type")),
		slog.Int64("content_length", resp.ContentLength),
	)

	if resp.StatusCode >= 400 {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
		bodyStr := string(body)
		if readErr != nil {
			bodyStr = fmt.Sprintf("(failed to read body: %v)", readErr)
		}

		c.logger.Error("ElevenLabs API error",
			slog.Int("status_code", resp.StatusCode),
			slog.String("response_body", bodyStr),
			slog.String("endpoint", c.endpoint),
		)
		return nil, fmt.Errorf("elevenlabs error: status=%d body=%s", resp.StatusCode, bodyStr)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read ElevenLabs audio response",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("read audio: %w", err)
	}

	if len(audio) == 0 {
		c.logger.Warn("ElevenLabs returned empty audio response")
		return nil, fmt.Errorf("elevenlabs returned empty audio")
	}

	c.logger.Debug("successfully received audio from ElevenLabs",
		slog.Int("audio_bytes", len(audio)),
	)

	return audio, nil
}
