package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"leveltalk/internal/dialogs"
)

// StubClient implements dialogs.LLMClient with deterministic output for development.
type StubClient struct {
	logger *slog.Logger
}

// NewStubClient returns a stubbed LLM client.
func NewStubClient(logger *slog.Logger) *StubClient {
	return &StubClient{logger: logger}
}

// GenerateDialog creates a deterministic dialog that includes all input words.
func (s *StubClient) GenerateDialog(ctx context.Context, params dialogs.GenerateDialogParams) (dialogs.Dialog, error) {
	// In production, this is where we'd craft a prompt like:
	// "You are a language tutor. Create a CEFR {CEFRLevel} level dialog in {DialogLanguage}
	// that naturally uses the following vocabulary: {InputWords}. Respond using JSON with speaker turns."

	if len(params.InputWords) == 0 {
		return dialogs.Dialog{}, fmt.Errorf("input words required")
	}

	turnCount := max(4, len(params.InputWords))
	speakers := []string{"Ana", "Luis"}

	turns := make([]dialogs.DialogTurn, 0, turnCount)
	for i := 0; i < turnCount; i++ {
		word := params.InputWords[i%len(params.InputWords)]
		sentence := buildSentence(params.DialogLanguage, params.CEFRLevel, word, i)
		turns = append(turns, dialogs.DialogTurn{
			Speaker: speakers[i%len(speakers)],
			Text:    sentence,
		})
	}

	s.logger.Debug("stub LLM generated dialog",
		slog.String("input_language", params.InputLanguage),
		slog.String("dialog_language", params.DialogLanguage),
		slog.String("cefr", params.CEFRLevel),
		slog.String("words", strings.Join(params.InputWords, ",")),
	)

	return dialogs.Dialog{
		Turns: turns,
	}, nil
}

func buildSentence(language, level, word string, idx int) string {
	// Generate sentences entirely in the dialog language (monolingual)
	prefix := map[string]string{
		"es": "Hablemos sobre",
		"en": "Let's talk about",
		"ru": "Давайте поговорим о",
		"fi": "Puhutaan",
		"de": "Lass uns über",
		"fr": "Parlons de",
	}
	base := prefix[strings.ToLower(language)]
	if base == "" {
		base = "Let's discuss"
	}
	return fmt.Sprintf("%s %s (CEFR %s, turn %d).", base, word, level, idx+1)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
