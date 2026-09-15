ALTER TABLE users
    ADD COLUMN scratch_level TINYINT UNSIGNED NOT NULL DEFAULT 1 AFTER luck_level,
    ADD CONSTRAINT chk_users_scratch_level CHECK (scratch_level BETWEEN 1 AND 10);

CREATE TABLE item_upgrades (
    id CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    item_code VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    from_level TINYINT UNSIGNED NOT NULL,
    to_level TINYINT UNSIGNED NOT NULL,
    price BIGINT NOT NULL,
    idempotency_key VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_item_upgrades_user_key (user_id, idempotency_key),
    KEY idx_item_upgrades_user_created (user_id, created_at),
    CONSTRAINT fk_item_upgrades_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT chk_item_upgrades_level CHECK (to_level = from_level + 1),
    CONSTRAINT chk_item_upgrades_price CHECK (price > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
