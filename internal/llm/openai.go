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
	Title        string            `json:"title"`
	Turns        []struct {
		Speaker string `json:"speaker"`
		Text    string `json:"text"`
	} `json:"turns"`
	Translations map[string]string `json:"translations,omitempty"`
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
					"IMPORTANT: You must FIRST translate all provided words/phrases from the input language into the target language, " +
					"then use ONLY the translated versions in the dialog. Never include words from the input language in the dialog. " +
					"Always respond ONLY with JSON matching this exact schema: {\"title\":\"descriptive_title\",\"turns\":[{\"speaker\":\"string\",\"text\":\"string\"}],\"translations\":{\"exact_input_word\":\"translated_word\"}}. " +
					"The \"title\" field is REQUIRED and must be a concise, descriptive title (3-8 words) that expresses the main idea or topic of the dialog in the dialog language. " +
					"The translations object is REQUIRED and must contain an entry for EVERY input word/phrase provided, using the EXACT same spelling and casing as provided. Do not add commentary.",
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

	c.logger.Debug("parsing LLM response",
		slog.Int("content_length", len(content)),
		slog.String("content_preview", truncate([]byte(content), 200)),
	)

	var parsed dialogJSON
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		c.logger.Error("failed to parse LLM JSON response",
			slog.String("error", err.Error()),
			slog.String("content", truncate([]byte(content), 500)),
		)
		return dialogs.Dialog{}, fmt.Errorf("parse dialog json: %w content=%s", err, truncate([]byte(content), 256))
	}

	c.logger.Info("parsed LLM response",
		slog.Int("turns", len(parsed.Turns)),
		slog.Int("translations", len(parsed.Translations)),
		slog.Any("translation_keys", getMapKeys(parsed.Translations)),
	)

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

	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		// Generate a fallback title from the first turn if LLM didn't provide one
		if len(turns) > 0 && len(turns[0].Text) > 0 {
			firstText := strings.TrimSpace(turns[0].Text)
			if len(firstText) > 50 {
				firstText = firstText[:50] + "..."
			}
			title = firstText
		} else {
			title = "Dialog"
		}
	}

	translations := parsed.Translations
	if translations == nil {
		translations = make(map[string]string)
	}

	// Normalize translation keys to match input words (trim whitespace, handle case)
	normalizedTranslations := make(map[string]string)
	for _, inputWord := range params.InputWords {
		trimmed := strings.TrimSpace(inputWord)
		// Try exact match first
		if trans, ok := translations[trimmed]; ok && trans != "" {
			normalizedTranslations[trimmed] = trans
			c.logger.Debug("matched translation", slog.String("input", trimmed), slog.String("translation", trans))
			continue
		}
		// Try case-insensitive match
		found := false
		for key, trans := range translations {
			if strings.EqualFold(strings.TrimSpace(key), trimmed) && trans != "" {
				normalizedTranslations[trimmed] = trans
				c.logger.Debug("matched translation (case-insensitive)", slog.String("input", trimmed), slog.String("key", key), slog.String("translation", trans))
				found = true
				break
			}
		}
		if !found {
			// If no translation found, leave empty (will show placeholder in UI)
			normalizedTranslations[trimmed] = ""
			c.logger.Warn("no translation found for input word", slog.String("input", trimmed), slog.Int("translations_count", len(translations)))
		}
	}

	c.logger.Info("normalized translations",
		slog.Int("input_words", len(params.InputWords)),
		slog.Int("raw_translations", len(translations)),
		slog.Int("normalized_translations", len(normalizedTranslations)),
	)

	return dialogs.Dialog{
		Title:        title,
		Turns:        turns,
		Translations: normalizedTranslations,
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
	sb.WriteString(". You are given these words/phrases in ")
	sb.WriteString(params.InputLanguage)
	sb.WriteString(": ")

	// List each word/phrase separately for clarity
	wordsList := make([]string, len(params.InputWords))
	for i, word := range params.InputWords {
		wordsList[i] = fmt.Sprintf("\"%s\"", strings.TrimSpace(word))
	}
	sb.WriteString(strings.Join(wordsList, ", "))

	sb.WriteString(". FIRST translate each word/phrase into ")
	sb.WriteString(params.DialogLanguage)
	sb.WriteString(", then naturally incorporate the TRANSLATED versions into the dialog. ")
	sb.WriteString("The dialog must contain ONLY ")
	sb.WriteString(params.DialogLanguage)
	sb.WriteString(" - no words from ")
	sb.WriteString(params.InputLanguage)
	sb.WriteString(" should appear. Provide between 6 and 10 turns. ")
	sb.WriteString("CRITICAL: You MUST include a \"translations\" object in your JSON response. ")
	sb.WriteString("The translations object must map EACH input word/phrase (using the EXACT spelling: ")
	sb.WriteString(strings.Join(params.InputWords, ", "))
	sb.WriteString(") to its translation in ")
	sb.WriteString(params.DialogLanguage)
	sb.WriteString(". Example format: {\"title\":\"Shopping at the Market\",\"turns\":[...],\"translations\":{\"")
	if len(params.InputWords) > 0 {
		sb.WriteString(params.InputWords[0])
		sb.WriteString("\":\"translation_here\"")
		if len(params.InputWords) > 1 {
			sb.WriteString(",\"")
			sb.WriteString(params.InputWords[1])
			sb.WriteString("\":\"translation_here\"")
		}
	}
	sb.WriteString("}}")
	return sb.String()
}

func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
