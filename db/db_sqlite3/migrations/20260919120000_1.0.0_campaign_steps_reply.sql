
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS "campaign_steps" (
    "id" integer primary key autoincrement,
    "campaign_id" integer,
    "step_index" integer,
    "template_id" integer,
    "delay_minutes" integer default 0,
    "trigger" varchar(64) default 'after_sent'
);

ALTER TABLE mail_logs ADD COLUMN step_index integer default 0;
ALTER TABLE mail_logs ADD COLUMN message_id varchar(255) default '';
ALTER TABLE mail_logs ADD COLUMN sent boolean default 0;

ALTER TABLE results ADD COLUMN replied boolean default 0;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE results DROP COLUMN replied;
ALTER TABLE mail_logs DROP COLUMN sent;
ALTER TABLE mail_logs DROP COLUMN message_id;
ALTER TABLE mail_logs DROP COLUMN step_index;
DROP TABLE IF EXISTS "campaign_steps";
