ALTER TABLE tickets
    ADD COLUMN location ENUM('tray', 'desk', 'slot') NOT NULL DEFAULT 'tray' AFTER state,
    ADD COLUMN desk_x DECIMAL(7,6) NOT NULL DEFAULT 0.500000 AFTER location,
    ADD COLUMN desk_y DECIMAL(7,6) NOT NULL DEFAULT 0.350000 AFTER desk_x,
    ADD COLUMN rotation DECIMAL(5,2) NOT NULL DEFAULT 0.00 AFTER desk_y,
    ADD COLUMN z_index INT UNSIGNED NOT NULL DEFAULT 1 AFTER rotation,
    ADD COLUMN slot_index TINYINT UNSIGNED NULL AFTER z_index,
    ADD COLUMN discarded_at DATETIME(6) NULL AFTER redeemed_at;

ALTER TABLE tickets
    ADD UNIQUE KEY uk_tickets_user_slot (user_id, slot_index),
    ADD CONSTRAINT chk_tickets_desk_x CHECK (desk_x BETWEEN 0 AND 1),
    ADD CONSTRAINT chk_tickets_desk_y CHECK (desk_y BETWEEN 0 AND 1),
    ADD CONSTRAINT chk_tickets_rotation CHECK (rotation BETWEEN -15 AND 15),
    ADD CONSTRAINT chk_tickets_slot_index CHECK (slot_index IS NULL OR slot_index BETWEEN 1 AND 10);
