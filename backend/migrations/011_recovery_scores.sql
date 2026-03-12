CREATE TABLE IF NOT EXISTS recovery_scores (
  score_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  computed_at TIMESTAMPTZ NOT NULL,
  overall_score DOUBLE PRECISION NOT NULL,
  fatigue_score DOUBLE PRECISION NOT NULL,
  recovery_score DOUBLE PRECISION NOT NULL,
  acwr_component DOUBLE PRECISION NOT NULL,
  monotony_component DOUBLE PRECISION NOT NULL,
  strain_component DOUBLE PRECISION NOT NULL,
  discomfort_penalty DOUBLE PRECISION NOT NULL DEFAULT 0,
  resting_hr_penalty DOUBLE PRECISION NOT NULL DEFAULT 0,
  recovery_status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recovery_scores_user_time
  ON recovery_scores(user_id, computed_at DESC);
