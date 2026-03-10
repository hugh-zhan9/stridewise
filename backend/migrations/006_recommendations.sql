CREATE TABLE IF NOT EXISTS recommendations (
  rec_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  recommendation_date DATE NOT NULL,
  input_json JSONB NOT NULL,
  output_json JSONB NOT NULL,
  risk_level TEXT NOT NULL,
  override_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  is_fallback BOOLEAN NOT NULL DEFAULT FALSE,
  ai_provider TEXT NOT NULL,
  ai_model TEXT NOT NULL,
  prompt_version TEXT NOT NULL,
  engine_version TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recommendations_user_time ON recommendations(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS recommendation_feedbacks (
  feedback_id TEXT PRIMARY KEY,
  rec_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  useful TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rec_id, user_id)
);
