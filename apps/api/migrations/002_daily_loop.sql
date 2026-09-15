ALTER TABLE tickets
    ADD COLUMN source ENUM('purchase', 'daily_wheel') NOT NULL DEFAULT 'purchase' AFTER card_name,
    ADD COLUMN price_paid BIGINT NOT NULL DEFAULT 0 AFTER price,
    ADD COLUMN wheel_date DATE NULL AFTER price_paid;

UPDATE tickets SET price_paid = price WHERE source = 'purchase';

ALTER TABLE tickets
    ADD KEY idx_tickets_user_source_created (user_id, source, created_at),
    ADD CONSTRAINT chk_tickets_price_paid_nonnegative CHECK (price_paid >= 0);

CREATE TABLE IF NOT EXISTS daily_user_state (
    user_id BIGINT UNSIGNED NOT NULL,
    local_date DATE NOT NULL,
    login_claimed TINYINT(1) NOT NULL DEFAULT 0,
    plates_completed TINYINT UNSIGNED NOT NULL DEFAULT 0,
    wheel_used TINYINT(1) NOT NULL DEFAULT 0,
    wheel_ticket_id CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NULL,
    wheel_card_code VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NULL,
    wheel_card_name VARCHAR(32) NULL,
    wheel_pool_snapshot JSON NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id, local_date),
    UNIQUE KEY uk_daily_wheel_ticket (wheel_ticket_id),
    CONSTRAINT fk_daily_state_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_daily_state_wheel_ticket FOREIGN KEY (wheel_ticket_id) REFERENCES tickets (id) ON DELETE SET NULL,
    CONSTRAINT chk_daily_plates_completed CHECK (plates_completed <= 5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS daily_plate_actions (
    id CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    local_date DATE NOT NULL,
    sequence_no TINYINT UNSIGNED NOT NULL,
    state ENUM('started', 'completed') NOT NULL DEFAULT 'started',
    started_at DATETIME(6) NOT NULL,
    available_at DATETIME(6) NOT NULL,
    completed_at DATETIME(6) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_plate_user_date_sequence (user_id, local_date, sequence_no),
    KEY idx_plate_user_date_state (user_id, local_date, state),
    CONSTRAINT fk_plate_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT chk_plate_sequence CHECK (sequence_no BETWEEN 1 AND 5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
