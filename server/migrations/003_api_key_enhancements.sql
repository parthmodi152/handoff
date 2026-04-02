-- Add agent_name to API keys and requests
ALTER TABLE api_keys ADD COLUMN agent_name TEXT;
ALTER TABLE requests ADD COLUMN agent_name TEXT;

-- Junction table for loop-scoped API key access
CREATE TABLE IF NOT EXISTS api_key_loops (
    api_key_id TEXT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    loop_id    TEXT NOT NULL REFERENCES loops(id) ON DELETE CASCADE,
    PRIMARY KEY (api_key_id, loop_id)
);
CREATE INDEX IF NOT EXISTS idx_api_key_loops_key ON api_key_loops(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_loops_loop ON api_key_loops(loop_id);
