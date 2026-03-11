CREATE TABLE IF NOT EXISTS nightly_baseline_runs (
  run_date DATE PRIMARY KEY,
  status TEXT NOT NULL,
  error_message TEXT NOT NULL DEFAULT '',
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
