CREATE TABLE IF NOT EXISTS sync_jobs (
  id BIGSERIAL PRIMARY KEY,
  job_id VARCHAR(128) NOT NULL UNIQUE,
  user_id VARCHAR(128) NOT NULL,
  source VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL,
  retry_count INTEGER NOT NULL DEFAULT 0,
  error_message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_jobs_user_source ON sync_jobs(user_id, source);

CREATE TABLE IF NOT EXISTS raw_activities (
  id BIGSERIAL PRIMARY KEY,
  job_id VARCHAR(128) NOT NULL,
  user_id VARCHAR(128) NOT NULL,
  source VARCHAR(32) NOT NULL,
  source_activity_id VARCHAR(128) NOT NULL,
  payload_json JSONB NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, source, source_activity_id)
);

CREATE TABLE IF NOT EXISTS activities (
  id BIGSERIAL PRIMARY KEY,
  user_id VARCHAR(128) NOT NULL,
  source VARCHAR(32) NOT NULL,
  source_activity_id VARCHAR(128) NOT NULL,
  name TEXT,
  distance_m DOUBLE PRECISION NOT NULL DEFAULT 0,
  moving_time_sec INTEGER NOT NULL DEFAULT 0,
  start_time_utc TIMESTAMPTZ NOT NULL,
  start_time_local TIMESTAMPTZ NOT NULL,
  timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
  summary_polyline TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, source, source_activity_id)
);

CREATE INDEX IF NOT EXISTS idx_activities_user_start ON activities(user_id, start_time_local DESC);
