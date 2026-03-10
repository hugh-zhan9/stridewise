CREATE TABLE IF NOT EXISTS training_logs (
  log_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  source TEXT NOT NULL,
  training_type TEXT NOT NULL,
  training_type_custom TEXT NOT NULL DEFAULT '',
  start_time TIMESTAMP NOT NULL,
  duration_sec INT NOT NULL,
  distance_km NUMERIC NOT NULL,
  pace_str TEXT NOT NULL,
  pace_sec_per_km INT NOT NULL,
  rpe INT NOT NULL,
  discomfort BOOLEAN NOT NULL,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS async_jobs (
  job_id TEXT PRIMARY KEY,
  job_type TEXT NOT NULL,
  user_id TEXT NOT NULL,
  payload_json JSONB NOT NULL,
  status TEXT NOT NULL,
  retry_count INT NOT NULL,
  error_message TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
