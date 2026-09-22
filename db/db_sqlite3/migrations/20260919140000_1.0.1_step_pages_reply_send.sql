
-- +goose Up
ALTER TABLE campaign_steps ADD COLUMN page_id integer default 0;
ALTER TABLE results ADD COLUMN last_email_step integer default 0;

-- +goose Down
ALTER TABLE results DROP COLUMN last_email_step;
ALTER TABLE campaign_steps DROP COLUMN page_id;
