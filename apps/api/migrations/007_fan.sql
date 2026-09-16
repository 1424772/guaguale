ALTER TABLE users
    ADD COLUMN fan_level TINYINT UNSIGNED NOT NULL DEFAULT 0 AFTER card_slots_owned,
    ADD COLUMN fan_risk_acknowledged BOOLEAN NOT NULL DEFAULT FALSE AFTER fan_level,
    ADD CONSTRAINT chk_users_fan_level CHECK (fan_level <= 8);

CREATE TABLE fan_events (
    id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    result_json JSON NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_fan_events_user_created (user_id, created_at),
    CONSTRAINT fk_fan_events_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
