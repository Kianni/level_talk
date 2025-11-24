package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"leveltalk/internal/dialogs"
)

const (
	defaultOpenAIEndpoint = "https://api.openai.com/v1/chat/completions"
	defaultTemperature    = 0.6
	defaultMaxTokens      = 800
)

// OpenAIOptions allows overriding HTTP behavior.
type OpenAIOptions struct {
	BaseURL     string
	HTTPClient  *http.Client
	Temperature float64
	MaxTokens   int
}

// OpenAIClient implements LLMClient against OpenAI's Chat Completions API.
type OpenAIClient struct {
	logger      *slog.Logger
	apiKey      string
	model       string
	endpoint    string
	httpClient  *http.Client
	temperature float64
	maxTokens   int
}

// NewOpenAIClient constructs a new OpenAIClient.
func NewOpenAIClient(logger *slog.Logger, apiKey, model string, opts *OpenAIOptions) *OpenAIClient {
	if opts == nil {
		opts = &OpenAIOptions{}
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 45 * time.Second,
		}
	}

	endpoint := opts.BaseURL
	if endpoint == "" {
		endpoint = defaultOpenAIEndpoint
	}

	temperature := opts.Temperature
	if temperature == 0 {
		temperature = defaultTemperature
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}

	return &OpenAIClient{
		logger:      logger,
		apiKey:      apiKey,
		model:       model,
		endpoint:    endpoint,
		httpClient:  httpClient,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type completionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

type dialogJSON struct {
	Turns []struct {
		Speaker string `json:"speaker"`
		Text    string `json:"text"`
	} `json:"turns"`
}

// GenerateDialog sends a prompt to OpenAI and parses the JSON payload into Dialogs.
func (c *OpenAIClient) GenerateDialog(ctx context.Context, params dialogs.GenerateDialogParams) (dialogs.Dialog, error) {
	reqPayload := completionRequest{
		Model:       c.model,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "You are an expert language tutor. Produce monolingual dialogs entirely in the target language in valid JSON. " +
					"Both speakers must speak ONLY in the dialog language. " +
					"Always respond ONLY with JSON matching the schema {\"turns\":[{\"speaker\":\"string\",\"text\":\"string\"}]} " +
					"and naturally incorporate the provided vocabulary from the input language into the target language context. Do not add commentary.",
			},
			{
				Role:    "user",
				Content: buildUserPrompt(params),
			},
		},
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return dialogs.Dialog{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return dialogs.Dialog{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return dialogs.Dialog{}, fmt.Errorf("call openai: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return dialogs.Dialog{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return dialogs.Dialog{}, fmt.Errorf("openai error: status=%d body=%s", resp.StatusCode, truncate(respBody, 512))
	}

	var completion completionResponse
	if err := json.Unmarshal(respBody, &completion); err != nil {
		return dialogs.Dialog{}, fmt.Errorf("decode response: %w body=%s", err, truncate(respBody, 256))
	}

	if completion.Error != nil {
		return dialogs.Dialog{}, fmt.Errorf("openai error: %s (%s)", completion.Error.Message, completion.Error.Type)
	}

	if len(completion.Choices) == 0 {
		return dialogs.Dialog{}, fmt.Errorf("openai returned no choices")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	content = stripCodeFence(content)

	var parsed dialogJSON
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return dialogs.Dialog{}, fmt.Errorf("parse dialog json: %w content=%s", err, truncate([]byte(content), 256))
	}

	if len(parsed.Turns) == 0 {
		return dialogs.Dialog{}, fmt.Errorf("openai returned no dialog turns")
	}

	turns := make([]dialogs.DialogTurn, 0, len(parsed.Turns))
	for i, turn := range parsed.Turns {
		speaker := strings.TrimSpace(turn.Speaker)
		text := strings.TrimSpace(turn.Text)
		if speaker == "" || text == "" {
			continue
		}
		turns = append(turns, dialogs.DialogTurn{
			Speaker:  speaker,
			Text:     text,
			Position: i,
		})
	}

	if len(turns) == 0 {
		return dialogs.Dialog{}, fmt.Errorf("openai returned empty turns after validation")
	}

	return dialogs.Dialog{
		Turns: turns,
	}, nil
}

func buildUserPrompt(params dialogs.GenerateDialogParams) string {
	var sb strings.Builder
	sb.WriteString("Generate a CEFR ")
	sb.WriteString(params.CEFRLevel)
	sb.WriteString(" level dialog entirely in ")
	sb.WriteString(params.DialogLanguage)
	sb.WriteString(". Both speakers must speak only in ")
	sb.WriteString(params.DialogLanguage)
	sb.WriteString(". The learner's native language is ")
	sb.WriteString(params.InputLanguage)
	sb.WriteString(", and you should naturally incorporate these words/phrases from ")
	sb.WriteString(params.InputLanguage)
	sb.WriteString(" into the ")
	sb.WriteString(params.DialogLanguage)
	sb.WriteString(" dialog context: ")
	sb.WriteString(strings.Join(params.InputWords, ", "))
	sb.WriteString(". Provide between 6 and 10 turns. The entire dialog must be in ")
	sb.WriteString(params.DialogLanguage)
	sb.WriteString(" only.")
	return sb.String()
}

func stripCodeFence(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "```") {
		v = strings.TrimPrefix(v, "```")
		if idx := strings.Index(v, "\n"); idx != -1 {
			v = v[idx+1:]
		}
		v = strings.TrimSuffix(v, "```")
	}
	return strings.TrimSpace(v)
}

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "…"
}
