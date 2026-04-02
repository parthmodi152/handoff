-- Support multiple devices per user for push notifications
CREATE TABLE IF NOT EXISTS device_tokens (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL,
    platform    TEXT NOT NULL DEFAULT 'ios',
    app_version TEXT,
    last_used   TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    UNIQUE(token)
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user ON device_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_device_tokens_token ON device_tokens(token);

-- Add composite index for badge count query performance
CREATE INDEX IF NOT EXISTS idx_requests_loop_status ON requests(loop_id, status);
