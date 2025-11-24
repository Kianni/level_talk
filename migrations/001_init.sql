CREATE TABLE IF NOT EXISTS dialogs (
    id UUID PRIMARY KEY,
    input_language TEXT NOT NULL,
    dialog_language TEXT NOT NULL,
    cefr_level TEXT NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2','C1','C2')),
    input_words JSONB NOT NULL,
    dialog_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dialog_turns (
    id UUID PRIMARY KEY,
    dialog_id UUID NOT NULL REFERENCES dialogs(id) ON DELETE CASCADE,
    speaker TEXT NOT NULL,
    text TEXT NOT NULL,
    audio_url TEXT NOT NULL,
    position INT NOT NULL,
    UNIQUE(dialog_id, position)
);

CREATE INDEX IF NOT EXISTS idx_dialogs_input_language ON dialogs(input_language);
CREATE INDEX IF NOT EXISTS idx_dialogs_dialog_language ON dialogs(dialog_language);
CREATE INDEX IF NOT EXISTS idx_dialogs_cefr_level ON dialogs(cefr_level);
CREATE INDEX IF NOT EXISTS idx_dialogs_created_at ON dialogs(created_at DESC);

