package dialogs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service orchestrates dialog generation, synthesis, and persistence.
type Service struct {
	repo Repository
	llm  LLMClient
	tts  TTSClient
}

// NewService constructs a Service.
func NewService(repo Repository, llm LLMClient, tts TTSClient) *Service {
	return &Service{
		repo: repo,
		llm:  llm,
		tts:  tts,
	}
}

// CreateDialog validates input, generates dialog content, synthesizes audio, and persists the result.
func (s *Service) CreateDialog(ctx context.Context, input CreateDialogInput) (Dialog, error) {
	if err := validateCreateInput(input); err != nil {
		return Dialog{}, fmt.Errorf("validate input: %w", err)
	}

	generated, err := s.llm.GenerateDialog(ctx, GenerateDialogParams{
		InputLanguage:  input.InputLanguage,
		DialogLanguage: input.DialogLanguage,
		CEFRLevel:      input.CEFRLevel,
		InputWords:     input.InputWords,
	})
	if err != nil {
		return Dialog{}, fmt.Errorf("generate dialog: %w", err)
	}

	now := time.Now().UTC()
	dlg := Dialog{
		ID:             uuid.New(),
		Title:          generated.Title,
		InputLanguage:  input.InputLanguage,
		DialogLanguage: input.DialogLanguage,
		CEFRLevel:      input.CEFRLevel,
		InputWords:     input.InputWords,
		Translations:   generated.Translations,
		Turns:          generated.Turns,
		CreatedAt:      now,
	}
	if dlg.Translations == nil {
		dlg.Translations = make(map[string]string)
	}

	for i := range dlg.Turns {
		if dlg.Turns[i].ID == uuid.Nil {
			dlg.Turns[i].ID = uuid.New()
		}
		dlg.Turns[i].Position = i
	}

	withAudio, err := s.tts.SynthesizeDialog(ctx, dlg)
	if err != nil {
		// Log error but don't expose internal details to user
		return Dialog{}, fmt.Errorf("tts synthesize: %w", err)
	}

	if err := s.repo.Create(ctx, withAudio); err != nil {
		return Dialog{}, fmt.Errorf("persist dialog: %w", err)
	}

	return withAudio, nil
}

// GetDialog fetches a single dialog by id.
func (s *Service) GetDialog(ctx context.Context, id uuid.UUID) (Dialog, error) {
	dlg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Dialog{}, err
	}
	return dlg, nil
}

// SearchDialogs queries dialogs using filter criteria.
func (s *Service) SearchDialogs(ctx context.Context, filter DialogFilter) ([]Dialog, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	return s.repo.Search(ctx, filter)
}

func validateCreateInput(input CreateDialogInput) error {
	if input.InputLanguage == "" || input.DialogLanguage == "" || input.CEFRLevel == "" {
		return ErrInvalidInput
	}
	if len(input.InputWords) == 0 {
		return fmt.Errorf("%w: at least one input word is required", ErrInvalidInput)
	}
	for _, word := range input.InputWords {
		if strings.TrimSpace(word) == "" {
			return fmt.Errorf("%w: empty word provided", ErrInvalidInput)
		}
	}
	return nil
}
