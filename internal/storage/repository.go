package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"leveltalk/internal/dialogs"
)

// DialogRepository persists dialogs in PostgreSQL.
type DialogRepository struct {
	db *sql.DB
}

// NewDialogRepository creates a new repository.
func NewDialogRepository(db *sql.DB) *DialogRepository {
	return &DialogRepository{db: db}
}

// Create inserts a dialog and its turns within a transaction.
func (r *DialogRepository) Create(ctx context.Context, dlg dialogs.Dialog) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	wordsJSON, err := json.Marshal(dlg.InputWords)
	if err != nil {
		return fmt.Errorf("marshal input words: %w", err)
	}

	turnsJSON, err := json.Marshal(dlg.Turns)
	if err != nil {
		return fmt.Errorf("marshal turns: %w", err)
	}

	translationsJSON, err := json.Marshal(dlg.Translations)
	if err != nil {
		return fmt.Errorf("marshal translations: %w", err)
	}

	const insertDialog = `
		INSERT INTO dialogs (
			id, input_language, dialog_language, cefr_level, input_words, dialog_json, translations, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	if _, err := tx.ExecContext(ctx, insertDialog,
		dlg.ID,
		dlg.InputLanguage,
		dlg.DialogLanguage,
		dlg.CEFRLevel,
		wordsJSON,
		turnsJSON,
		translationsJSON,
		dlg.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert dialog: %w", err)
	}

	const insertTurn = `
		INSERT INTO dialog_turns (id, dialog_id, speaker, text, audio_url, position)
		VALUES ($1,$2,$3,$4,$5,$6)
	`
	for _, turn := range dlg.Turns {
		if _, err := tx.ExecContext(ctx, insertTurn,
			turn.ID,
			dlg.ID,
			turn.Speaker,
			turn.Text,
			turn.AudioURL,
			turn.Position,
		); err != nil {
			return fmt.Errorf("insert turn: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// GetByID fetches a dialog with all turns.
func (r *DialogRepository) GetByID(ctx context.Context, id uuid.UUID) (dialogs.Dialog, error) {
	const queryDialog = `
		SELECT id, input_language, dialog_language, cefr_level, input_words, COALESCE(translations, '{}'::jsonb), created_at
		FROM dialogs
		WHERE id = $1
	`
	var dlg dialogs.Dialog
	var inputWordsJSON []byte
	var translationsJSON []byte
	if err := r.db.QueryRowContext(ctx, queryDialog, id).Scan(
		&dlg.ID,
		&dlg.InputLanguage,
		&dlg.DialogLanguage,
		&dlg.CEFRLevel,
		&inputWordsJSON,
		&translationsJSON,
		&dlg.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dialogs.Dialog{}, dialogs.ErrNotFound
		}
		return dialogs.Dialog{}, fmt.Errorf("select dialog: %w", err)
	}
	if err := json.Unmarshal(inputWordsJSON, &dlg.InputWords); err != nil {
		return dialogs.Dialog{}, fmt.Errorf("unmarshal input words: %w", err)
	}
	if err := json.Unmarshal(translationsJSON, &dlg.Translations); err != nil {
		return dialogs.Dialog{}, fmt.Errorf("unmarshal translations: %w", err)
	}
	if dlg.Translations == nil {
		dlg.Translations = make(map[string]string)
	}

	const queryTurns = `
		SELECT id, speaker, text, audio_url, position
		FROM dialog_turns
		WHERE dialog_id = $1
		ORDER BY position ASC
	`
	rows, err := r.db.QueryContext(ctx, queryTurns, id)
	if err != nil {
		return dialogs.Dialog{}, fmt.Errorf("select turns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var turn dialogs.DialogTurn
		if err := rows.Scan(&turn.ID, &turn.Speaker, &turn.Text, &turn.AudioURL, &turn.Position); err != nil {
			return dialogs.Dialog{}, fmt.Errorf("scan turn: %w", err)
		}
		dlg.Turns = append(dlg.Turns, turn)
	}
	if err := rows.Err(); err != nil {
		return dialogs.Dialog{}, fmt.Errorf("rows error: %w", err)
	}
	return dlg, nil
}

// Search returns dialogs filtered by provided criteria.
func (r *DialogRepository) Search(ctx context.Context, filter dialogs.DialogFilter) ([]dialogs.Dialog, error) {
	query := strings.Builder{}
	args := []any{}

	query.WriteString(`
		SELECT id, input_language, dialog_language, cefr_level, input_words, dialog_json, COALESCE(translations, '{}'::jsonb), created_at
		FROM dialogs
		WHERE 1=1
	`)

	if filter.InputLanguage != nil && *filter.InputLanguage != "" {
		args = append(args, *filter.InputLanguage)
		query.WriteString(fmt.Sprintf(" AND input_language = $%d", len(args)))
	}
	if filter.DialogLanguage != nil && *filter.DialogLanguage != "" {
		args = append(args, *filter.DialogLanguage)
		query.WriteString(fmt.Sprintf(" AND dialog_language = $%d", len(args)))
	}
	if filter.CEFRLevel != nil && *filter.CEFRLevel != "" {
		args = append(args, *filter.CEFRLevel)
		query.WriteString(fmt.Sprintf(" AND cefr_level = $%d", len(args)))
	}

	query.WriteString(" ORDER BY created_at DESC")
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query.WriteString(fmt.Sprintf(" LIMIT $%d", len(args)))
	} else {
		args = append(args, 20)
		query.WriteString(fmt.Sprintf(" LIMIT $%d", len(args)))
	}
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query.WriteString(fmt.Sprintf(" OFFSET $%d", len(args)))
	}

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("search dialogs: %w", err)
	}
	defer rows.Close()

	var result []dialogs.Dialog
	for rows.Next() {
		var (
			dlg          dialogs.Dialog
			inputWords   []byte
			dialogTurns  []byte
			translations []byte
		)
		if err := rows.Scan(
			&dlg.ID,
			&dlg.InputLanguage,
			&dlg.DialogLanguage,
			&dlg.CEFRLevel,
			&inputWords,
			&dialogTurns,
			&translations,
			&dlg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dialog: %w", err)
		}
		if err := json.Unmarshal(inputWords, &dlg.InputWords); err != nil {
			return nil, fmt.Errorf("unmarshal input words: %w", err)
		}
		if err := json.Unmarshal(dialogTurns, &dlg.Turns); err != nil {
			return nil, fmt.Errorf("unmarshal turns: %w", err)
		}
		if err := json.Unmarshal(translations, &dlg.Translations); err != nil {
			return nil, fmt.Errorf("unmarshal translations: %w", err)
		}
		if dlg.Translations == nil {
			dlg.Translations = make(map[string]string)
		}
		result = append(result, dlg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return result, nil
}
