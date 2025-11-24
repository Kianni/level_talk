package tts

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"leveltalk/internal/dialogs"
)

// StubClient simulates ElevenLabs synthesis for development.
type StubClient struct{}

// NewStubClient constructs StubClient.
func NewStubClient() *StubClient {
	return &StubClient{}
}

// SynthesizeDialog assigns placeholder audio URLs.
func (s *StubClient) SynthesizeDialog(ctx context.Context, dlg dialogs.Dialog) (dialogs.Dialog, error) {
	for i := range dlg.Turns {
		if dlg.Turns[i].ID == uuid.Nil {
			dlg.Turns[i].ID = uuid.New()
		}
		dlg.Turns[i].AudioURL = fmt.Sprintf("/static/audio/placeholder.mp3?turn=%d", dlg.Turns[i].Position)
	}

	// TODO: Replace with ElevenLabs API call.
	// 1. Send text, language, and desired voice per turn.
	// 2. Store resulting audio files (e.g., S3 or local disk).
	// 3. Persist accessible URLs and serve securely.

	return dlg, nil
}
