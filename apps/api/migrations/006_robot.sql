ALTER TABLE users
    ADD COLUMN robot_owned BOOLEAN NOT NULL DEFAULT FALSE AFTER card_slots_owned,
    ADD COLUMN robot_speed_level TINYINT UNSIGNED NOT NULL DEFAULT 0 AFTER robot_owned,
    ADD COLUMN robot_queue_level TINYINT UNSIGNED NOT NULL DEFAULT 0 AFTER robot_speed_level,
    ADD COLUMN robot_intercept_level TINYINT UNSIGNED NOT NULL DEFAULT 0 AFTER robot_queue_level,
    ADD CONSTRAINT chk_users_robot_speed CHECK (robot_speed_level <= 8),
    ADD CONSTRAINT chk_users_robot_queue CHECK (robot_queue_level <= 6),
    ADD CONSTRAINT chk_users_robot_intercept CHECK (robot_intercept_level <= 8);

ALTER TABLE tickets
    MODIFY COLUMN location ENUM('tray', 'desk', 'slot', 'robot') NOT NULL DEFAULT 'tray';

CREATE TABLE robot_queue (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    ticket_id CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    remaining_ms BIGINT NOT NULL,
    enqueued_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_robot_queue_ticket (ticket_id),
    KEY idx_robot_queue_user_order (user_id, id),
    CONSTRAINT fk_robot_queue_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_robot_queue_ticket FOREIGN KEY (ticket_id) REFERENCES tickets (id) ON DELETE CASCADE,
    CONSTRAINT chk_robot_queue_remaining CHECK (remaining_ms > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE robot_runtime (
    user_id BIGINT UNSIGNED NOT NULL,
    last_tick_at DATETIME(6) NULL,
    PRIMARY KEY (user_id),
    CONSTRAINT fk_robot_runtime_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
