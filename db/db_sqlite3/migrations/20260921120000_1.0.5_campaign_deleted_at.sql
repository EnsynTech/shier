
-- +goose Up
ALTER TABLE campaigns ADD COLUMN deleted_at DATETIME;
CREATE INDEX IF NOT EXISTS idx_campaigns_user_deleted ON campaigns (user_id, deleted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_campaigns_user_deleted;
ALTER TABLE campaigns DROP COLUMN deleted_at;
