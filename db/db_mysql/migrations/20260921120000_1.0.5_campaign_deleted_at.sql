
-- +goose Up
ALTER TABLE `campaigns` ADD COLUMN `deleted_at` DATETIME NULL;
CREATE INDEX idx_campaigns_user_deleted ON campaigns (user_id, deleted_at);

-- +goose Down
DROP INDEX idx_campaigns_user_deleted ON campaigns;
ALTER TABLE `campaigns` DROP COLUMN `deleted_at`;
