CREATE TABLE data_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    encrypted_data BYTEA NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE,
    version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_data_items_user_id ON data_items(user_id);
CREATE INDEX idx_data_items_type ON data_items(type);
CREATE INDEX idx_data_items_updated_at ON data_items(updated_at);
CREATE INDEX idx_data_items_not_deleted ON data_items(user_id) WHERE deleted_at IS NULL;