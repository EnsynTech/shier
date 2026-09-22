
-- +goose Up
CREATE TABLE IF NOT EXISTS ai_plans (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    idempotency_key VARCHAR(255) DEFAULT '',
    client_label VARCHAR(255) DEFAULT '',
    project_label VARCHAR(255) DEFAULT '',
    status VARCHAR(64) NOT NULL DEFAULT 'draft',
    group_id BIGINT DEFAULT 0,
    template_id BIGINT DEFAULT 0,
    page_id BIGINT DEFAULT 0,
    campaign_id BIGINT DEFAULT 0,
    smtp_id BIGINT DEFAULT 0,
    sending_channel_id BIGINT DEFAULT 0,
    scenario_pack_id VARCHAR(255) DEFAULT '',
    warnings TEXT,
    created_at DATETIME
);

CREATE INDEX idx_ai_plans_user_idem ON ai_plans (user_id, idempotency_key);

-- +goose Down
DROP TABLE IF EXISTS ai_plans;
