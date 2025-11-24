package dialogs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrNotFound signals missing dialog.
	ErrNotFound = errors.New("dialog not found")

	// ErrInvalidInput signals validation errors when creating dialogs.
	ErrInvalidInput = errors.New("invalid dialog input")
)

// Dialog represents a generated dialog with metadata.
type Dialog struct {
	ID             uuid.UUID
	InputLanguage  string
	DialogLanguage string
	CEFRLevel      string
	InputWords     []string
	Turns          []DialogTurn
	CreatedAt      time.Time
}

// DialogTurn is a single utterance inside a dialog.
type DialogTurn struct {
	ID       uuid.UUID
	Speaker  string
	Text     string
	AudioURL string
	Position int
}

// GenerateDialogParams describe the request to the LLM client.
type GenerateDialogParams struct {
	InputLanguage  string
	DialogLanguage string
	CEFRLevel      string
	InputWords     []string
}

// CreateDialogInput collects user input required to create a dialog.
type CreateDialogInput struct {
	InputLanguage  string
	DialogLanguage string
	CEFRLevel      string
	InputWords     []string
}

// DialogFilter is used for search queries.
type DialogFilter struct {
	InputLanguage  *string
	DialogLanguage *string
	CEFRLevel      *string
	Limit          int
	Offset         int
}

// Repository defines the persistence layer contract.
type Repository interface {
	Create(ctx context.Context, dlg Dialog) error
	GetByID(ctx context.Context, id uuid.UUID) (Dialog, error)
	Search(ctx context.Context, filter DialogFilter) ([]Dialog, error)
}

// LLMClient describes the interface to generate dialogs with an LLM.
type LLMClient interface {
	GenerateDialog(ctx context.Context, params GenerateDialogParams) (Dialog, error)
}

// TTSClient describes the interface to synthesize audio URLs for dialog turns.
type TTSClient interface {
	SynthesizeDialog(ctx context.Context, dlg Dialog) (Dialog, error)
}
