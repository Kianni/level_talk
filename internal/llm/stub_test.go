package llm

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"leveltalk/internal/dialogs"
)

func TestStubClientIncludesAllWords(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewStubClient(logger)

	params := dialogs.GenerateDialogParams{
		InputLanguage:  "ru",
		DialogLanguage: "es",
		CEFRLevel:      "B1",
		InputWords:     []string{"casa", "perro", "biblioteca"},
	}

	dlg, err := client.GenerateDialog(context.Background(), params)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(dlg.Turns), len(params.InputWords))

	for _, word := range params.InputWords {
		found := false
		for _, turn := range dlg.Turns {
			if strings.Contains(turn.Text, word) {
				found = true
				break
			}
		}
		require.Truef(t, found, "word %q missing in turns", word)
	}
}
