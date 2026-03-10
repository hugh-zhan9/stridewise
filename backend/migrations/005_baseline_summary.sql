CREATE TABLE IF NOT EXISTS baseline_current (
  user_id TEXT PRIMARY KEY,
  computed_at TIMESTAMPTZ NOT NULL,
  data_sessions_7d INT NOT NULL,
  acute_load_srpe DOUBLE PRECISION,
  chronic_load_srpe DOUBLE PRECISION,
  acwr_srpe DOUBLE PRECISION,
  acute_load_distance DOUBLE PRECISION,
  chronic_load_distance DOUBLE PRECISION,
  acwr_distance DOUBLE PRECISION,
  monotony DOUBLE PRECISION,
  strain DOUBLE PRECISION,
  pace_avg_sec_per_km INT,
  pace_low_sec_per_km INT,
  pace_high_sec_per_km INT,
  status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS baseline_history (
  baseline_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  computed_at TIMESTAMPTZ NOT NULL,
  trigger_type TEXT NOT NULL,
  trigger_ref TEXT NOT NULL,
  data_sessions_7d INT NOT NULL,
  acute_load_srpe DOUBLE PRECISION,
  chronic_load_srpe DOUBLE PRECISION,
  acwr_srpe DOUBLE PRECISION,
  acute_load_distance DOUBLE PRECISION,
  chronic_load_distance DOUBLE PRECISION,
  acwr_distance DOUBLE PRECISION,
  monotony DOUBLE PRECISION,
  strain DOUBLE PRECISION,
  pace_avg_sec_per_km INT,
  pace_low_sec_per_km INT,
  pace_high_sec_per_km INT,
  status TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_baseline_history_user_time ON baseline_history(user_id, computed_at DESC);

CREATE TABLE IF NOT EXISTS training_summaries (
  summary_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  log_id TEXT NOT NULL UNIQUE,
  completion_rate TEXT NOT NULL,
  intensity_match TEXT NOT NULL,
  recovery_advice TEXT NOT NULL,
  anomaly_notes TEXT NOT NULL,
  performance_notes TEXT NOT NULL,
  next_suggestion TEXT NOT NULL,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS training_feedbacks (
  feedback_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  log_id TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
