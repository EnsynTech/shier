
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS `campaign_steps` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `campaign_id` bigint(20),
    `step_index` bigint(20),
    `template_id` bigint(20),
    `delay_minutes` bigint(20) DEFAULT 0,
    `trigger` varchar(64) DEFAULT 'after_sent',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8;

ALTER TABLE `mail_logs` ADD COLUMN `step_index` bigint(20) DEFAULT 0;
ALTER TABLE `mail_logs` ADD COLUMN `message_id` varchar(255) DEFAULT '';
ALTER TABLE `mail_logs` ADD COLUMN `sent` tinyint(1) DEFAULT 0;

ALTER TABLE `results` ADD COLUMN `replied` tinyint(1) DEFAULT 0;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `results` DROP COLUMN `replied`;
ALTER TABLE `mail_logs` DROP COLUMN `sent`;
ALTER TABLE `mail_logs` DROP COLUMN `message_id`;
ALTER TABLE `mail_logs` DROP COLUMN `step_index`;
DROP TABLE IF EXISTS `campaign_steps`;
