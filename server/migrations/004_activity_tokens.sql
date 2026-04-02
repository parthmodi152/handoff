CREATE TABLE IF NOT EXISTS activity_tokens (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    loop_id    TEXT NOT NULL REFERENCES loops(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_activity_tokens_loop ON activity_tokens(loop_id);
CREATE INDEX IF NOT EXISTS idx_activity_tokens_user ON activity_tokens(user_id);
