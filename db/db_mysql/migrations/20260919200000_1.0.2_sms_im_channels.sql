
-- +goose Up
ALTER TABLE targets ADD COLUMN phone varchar(255) DEFAULT '';
ALTER TABLE results ADD COLUMN phone varchar(255) DEFAULT '';
ALTER TABLE email_requests ADD COLUMN phone varchar(255) DEFAULT '';
ALTER TABLE campaigns ADD COLUMN channel varchar(16) DEFAULT 'email';
ALTER TABLE campaigns ADD COLUMN sending_channel_id bigint DEFAULT 0;

CREATE TABLE IF NOT EXISTS sending_channels (
  id bigint auto_increment primary key,
  user_id bigint,
  name varchar(255),
  channel varchar(16),
  endpoint_url varchar(1024),
  http_method varchar(16) DEFAULT 'POST',
  auth_header_name varchar(255) DEFAULT '',
  auth_header_value varchar(1024) DEFAULT '',
  content_type varchar(255) DEFAULT 'application/json',
  body_template mediumtext,
  success_status_codes varchar(255) DEFAULT '200,201,204',
  ignore_cert_errors boolean DEFAULT 0,
  modified_date datetime
);

CREATE TABLE IF NOT EXISTS delivery_logs (
  id bigint auto_increment primary key,
  user_id bigint,
  campaign_id bigint,
  r_id varchar(255),
  send_date datetime,
  send_attempt integer DEFAULT 0,
  processing boolean DEFAULT 0,
  sent boolean DEFAULT 0,
  channel varchar(16),
  step_index integer DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS delivery_logs;
DROP TABLE IF EXISTS sending_channels;
ALTER TABLE campaigns DROP COLUMN sending_channel_id;
ALTER TABLE campaigns DROP COLUMN channel;
ALTER TABLE email_requests DROP COLUMN phone;
ALTER TABLE results DROP COLUMN phone;
ALTER TABLE targets DROP COLUMN phone;
