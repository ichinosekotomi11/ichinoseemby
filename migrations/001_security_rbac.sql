CREATE TABLE users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  email TEXT,
  level TEXT NOT NULL CHECK (level IN ('A', 'B', 'C')) DEFAULT 'C',
  password_hash TEXT NOT NULL,
  telegram_id_enc TEXT,
  emby_username_enc TEXT,
  emby_password_enc TEXT,
  emby_user_id TEXT,
  coins BIGINT NOT NULL DEFAULT 0,
  last_charge_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  emby_disabled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_level ON users(level);
CREATE INDEX idx_users_emby_user_id ON users(emby_user_id);

CREATE TABLE system_configs (
  key TEXT PRIMARY KEY,
  value_json JSONB NOT NULL,
  encrypted BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sensitive_configs (
  key TEXT PRIMARY KEY,
  value_enc TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE coin_transactions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  delta BIGINT NOT NULL,
  balance BIGINT NOT NULL,
  reason TEXT NOT NULL,
  operator TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_coin_transactions_user_id_created_at ON coin_transactions(user_id, created_at DESC);

CREATE TABLE security_events (
  id TEXT PRIMARY KEY,
  user_id TEXT REFERENCES users(id),
  type TEXT NOT NULL,
  encrypted_note TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_security_events_user_id_created_at ON security_events(user_id, created_at DESC);

INSERT INTO system_configs(key, value_json, encrypted)
VALUES (
  'feature_policies',
  '{
    "moviepilot.request": {
      "enabled": true,
      "allowed_levels": ["A", "B"],
      "coin_cost": 10,
      "level_cost_multiplier": {"A": 0, "B": 1, "C": 2}
    },
    "lottery.advanced": {
      "enabled": true,
      "allowed_levels": ["A"],
      "coin_cost": 20,
      "level_win_probability": {"A": 0.15, "B": 0.05, "C": 0.01}
    },
    "registration.apply": {
      "enabled": true,
      "allowed_levels": ["C"]
    },
    "checkin.daily": {
      "enabled": true,
      "allowed_levels": ["A", "B", "C"],
      "level_reward": {"A": 10, "B": 5, "C": 2}
    }
  }'::jsonb,
  FALSE
);
