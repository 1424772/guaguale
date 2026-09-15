CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(32) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 1000,
    luck_level TINYINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_username (username),
    CONSTRAINT chk_users_balance_nonnegative CHECK (balance >= 0),
    CONSTRAINT chk_users_luck_level CHECK (luck_level <= 10)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS sessions (
    token_hash BINARY(32) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (token_hash),
    KEY idx_sessions_user_id (user_id),
    KEY idx_sessions_expires_at (expires_at),
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS tickets (
    id CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    card_code VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    card_name VARCHAR(32) NOT NULL,
    purchase_key VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    price BIGINT NOT NULL,
    luck_level TINYINT UNSIGNED NOT NULL,
    prize_tier VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    reward BIGINT NOT NULL,
    symbols JSON NOT NULL,
    state ENUM('purchased', 'scratched', 'redeemed', 'discarded') NOT NULL DEFAULT 'purchased',
    scratched_at DATETIME(6) NULL,
    redeemed_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_tickets_user_purchase_key (user_id, purchase_key),
    KEY idx_tickets_user_state_created (user_id, state, created_at),
    CONSTRAINT fk_tickets_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT chk_tickets_price_positive CHECK (price > 0),
    CONSTRAINT chk_tickets_reward_nonnegative CHECK (reward >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS coin_ledger (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    idempotency_key VARCHAR(96) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    reason VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    reference_type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    reference_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    delta BIGINT NOT NULL,
    balance_before BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_coin_ledger_idempotency (idempotency_key),
    KEY idx_coin_ledger_user_created (user_id, created_at),
    CONSTRAINT fk_coin_ledger_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT chk_coin_ledger_balance_before CHECK (balance_before >= 0),
    CONSTRAINT chk_coin_ledger_balance_after CHECK (balance_after >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
