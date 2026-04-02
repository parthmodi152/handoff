CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    avatar_url TEXT,
    oauth_provider TEXT NOT NULL,
    oauth_id TEXT NOT NULL,
    device_token TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(oauth_provider, oauth_id)
);

CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    permissions TEXT NOT NULL DEFAULT '["loops:read","loops:write","requests:read","requests:write"]',
    is_active INTEGER NOT NULL DEFAULT 1,
    last_used_at TEXT,
    created_at TEXT NOT NULL,
    expires_at TEXT
);

CREATE TABLE loops (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    icon TEXT NOT NULL DEFAULT 'circle',
    creator_id TEXT NOT NULL REFERENCES users(id),
    invite_code TEXT UNIQUE NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE loop_members (
    loop_id TEXT NOT NULL REFERENCES loops(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id),
    role TEXT NOT NULL DEFAULT 'member',
    status TEXT NOT NULL DEFAULT 'active',
    joined_at TEXT NOT NULL,
    PRIMARY KEY (loop_id, user_id)
);

CREATE TABLE requests (
    id TEXT PRIMARY KEY,
    loop_id TEXT NOT NULL REFERENCES loops(id),
    creator_id TEXT NOT NULL REFERENCES users(id),
    api_key_id TEXT REFERENCES api_keys(id),
    processing_type TEXT NOT NULL,
    content_type TEXT NOT NULL,
    priority TEXT NOT NULL DEFAULT 'medium',
    request_text TEXT NOT NULL,
    image_url TEXT,
    context_json TEXT,
    platform TEXT DEFAULT 'api',
    response_type TEXT NOT NULL,
    response_config_json TEXT NOT NULL,
    default_response_json TEXT,
    timeout_seconds INTEGER,
    timeout_at TEXT,
    callback_url TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    response_data_json TEXT,
    response_by TEXT REFERENCES users(id),
    response_at TEXT,
    response_time_seconds REAL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE webhooks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    url TEXT NOT NULL,
    secret TEXT NOT NULL,
    events TEXT NOT NULL DEFAULT '["request.completed","request.timeout","request.cancelled"]',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL
);

CREATE TABLE webhook_deliveries (
    id TEXT PRIMARY KEY,
    webhook_id TEXT REFERENCES webhooks(id),
    request_id TEXT NOT NULL REFERENCES requests(id),
    url TEXT NOT NULL,
    event TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TEXT,
    next_retry_at TEXT,
    response_status INTEGER,
    response_body TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_requests_loop_id ON requests(loop_id);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_timeout_at ON requests(timeout_at);
CREATE INDEX idx_loop_members_user_id ON loop_members(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(status);
CREATE INDEX idx_webhook_deliveries_next_retry ON webhook_deliveries(next_retry_at);
