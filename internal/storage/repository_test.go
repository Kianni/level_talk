package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"leveltalk/internal/dialogs"
)

func TestDialogRepositoryCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDialogRepository(db)
	now := time.Now()
	dlg := dialogs.Dialog{
		ID:             uuid.New(),
		InputLanguage:  "ru",
		DialogLanguage: "es",
		CEFRLevel:      "B1",
		InputWords:     []string{"дом", "улица"},
		CreatedAt:      now,
		Turns: []dialogs.DialogTurn{
			{ID: uuid.New(), Speaker: "Ana", Text: "Hola дом", AudioURL: "/static/audio/placeholder.mp3", Position: 0},
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO dialogs").
		WithArgs(
			dlg.ID,
			dlg.InputLanguage,
			dlg.DialogLanguage,
			dlg.CEFRLevel,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			dlg.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO dialog_turns").
		WithArgs(
			dlg.Turns[0].ID,
			dlg.ID,
			dlg.Turns[0].Speaker,
			dlg.Turns[0].Text,
			dlg.Turns[0].AudioURL,
			dlg.Turns[0].Position,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.Create(context.Background(), dlg)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDialogRepositorySearch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDialogRepository(db)
	wordsJSON, _ := json.Marshal([]string{"casa"})
	turnsJSON, _ := json.Marshal([]dialogs.DialogTurn{
		{Speaker: "Ana", Text: "Hola casa", AudioURL: "/static/audio/placeholder.mp3"},
	})

	rows := sqlmock.NewRows([]string{
		"id", "input_language", "dialog_language", "cefr_level", "input_words", "dialog_json", "created_at",
	}).AddRow(uuid.New(), "ru", "es", "A2", wordsJSON, turnsJSON, time.Now())

	mock.ExpectQuery("SELECT id, input_language").
		WithArgs("ru", "es", "A2", 5).
		WillReturnRows(rows)

	filter := dialogs.DialogFilter{
		InputLanguage:  strPtr("ru"),
		DialogLanguage: strPtr("es"),
		CEFRLevel:      strPtr("A2"),
		Limit:          5,
	}

	result, err := repo.Search(context.Background(), filter)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "ru", result[0].InputLanguage)
	require.Equal(t, "es", result[0].DialogLanguage)
	require.Equal(t, "A2", result[0].CEFRLevel)
	require.NotEmpty(t, result[0].Turns)
	require.NoError(t, mock.ExpectationsWereMet())
}

func strPtr(v string) *string {
	return &v
}
