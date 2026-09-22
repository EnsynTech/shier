
-- +goose Up
ALTER TABLE `campaign_steps` ADD COLUMN `page_id` bigint(20) DEFAULT 0;
ALTER TABLE `results` ADD COLUMN `last_email_step` bigint(20) DEFAULT 0;

-- +goose Down
ALTER TABLE `results` DROP COLUMN `last_email_step`;
ALTER TABLE `campaign_steps` DROP COLUMN `page_id`;
