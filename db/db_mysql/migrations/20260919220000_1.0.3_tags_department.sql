
-- +goose Up
ALTER TABLE targets ADD COLUMN department varchar(255) DEFAULT '';
ALTER TABLE results ADD COLUMN department varchar(255) DEFAULT '';
ALTER TABLE email_requests ADD COLUMN department varchar(255) DEFAULT '';

ALTER TABLE templates ADD COLUMN scenario_pack_id varchar(255) DEFAULT '';
ALTER TABLE templates ADD COLUMN difficulty integer DEFAULT 0;
ALTER TABLE templates ADD COLUMN industries text;
ALTER TABLE templates ADD COLUMN techniques text;

ALTER TABLE campaigns ADD COLUMN scenario_pack_id varchar(255) DEFAULT '';
ALTER TABLE campaigns ADD COLUMN difficulty integer DEFAULT 0;
ALTER TABLE campaigns ADD COLUMN industries text;
ALTER TABLE campaigns ADD COLUMN techniques text;

-- +goose Down
ALTER TABLE campaigns DROP COLUMN techniques;
ALTER TABLE campaigns DROP COLUMN industries;
ALTER TABLE campaigns DROP COLUMN difficulty;
ALTER TABLE campaigns DROP COLUMN scenario_pack_id;

ALTER TABLE templates DROP COLUMN techniques;
ALTER TABLE templates DROP COLUMN industries;
ALTER TABLE templates DROP COLUMN difficulty;
ALTER TABLE templates DROP COLUMN scenario_pack_id;

ALTER TABLE email_requests DROP COLUMN department;
ALTER TABLE results DROP COLUMN department;
ALTER TABLE targets DROP COLUMN department;
