ALTER TABLE training_summaries
  ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'log';

ALTER TABLE training_summaries
  ADD COLUMN IF NOT EXISTS source_id TEXT;

UPDATE training_summaries
SET source_id = log_id
WHERE source_id IS NULL;

ALTER TABLE training_summaries
  ALTER COLUMN source_id SET NOT NULL;

ALTER TABLE training_summaries
  ALTER COLUMN log_id DROP NOT NULL;

ALTER TABLE training_summaries
  DROP CONSTRAINT IF EXISTS training_summaries_log_id_key;

ALTER TABLE training_summaries
  ADD CONSTRAINT training_summaries_source_unique UNIQUE (user_id, source_type, source_id);

ALTER TABLE training_feedbacks
  ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'log';

ALTER TABLE training_feedbacks
  ADD COLUMN IF NOT EXISTS source_id TEXT;

UPDATE training_feedbacks
SET source_id = log_id
WHERE source_id IS NULL;

ALTER TABLE training_feedbacks
  ALTER COLUMN source_id SET NOT NULL;

ALTER TABLE training_feedbacks
  ALTER COLUMN log_id DROP NOT NULL;

ALTER TABLE training_feedbacks
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE training_feedbacks
  ADD CONSTRAINT training_feedbacks_source_unique UNIQUE (user_id, source_type, source_id);
