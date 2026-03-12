CREATE TABLE IF NOT EXISTS user_personalization_params (
  user_id TEXT PRIMARY KEY,
  intensity_bias DOUBLE PRECISION NOT NULL DEFAULT 0,
  volume_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1,
  type_preference_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  reason_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  version INT NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_personalization_params_updated_at
  ON user_personalization_params(updated_at DESC);
