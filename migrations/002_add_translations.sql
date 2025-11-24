ALTER TABLE dialogs ADD COLUMN IF NOT EXISTS translations JSONB DEFAULT '{}'::jsonb;

