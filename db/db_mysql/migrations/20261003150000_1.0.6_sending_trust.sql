
-- +goose Up
ALTER TABLE campaigns ADD COLUMN sending_trust varchar(32) DEFAULT '';

-- +goose Down
ALTER TABLE campaigns DROP COLUMN sending_trust;
