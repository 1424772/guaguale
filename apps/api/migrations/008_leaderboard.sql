ALTER TABLE users
    ADD COLUMN balance_ranked_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) AFTER balance,
    ADD KEY idx_users_balance_rank (balance DESC, balance_ranked_at ASC, id ASC);

UPDATE users SET balance_ranked_at = created_at;

ALTER TABLE tickets
    ADD COLUMN scratch_source ENUM('manual', 'robot') NULL AFTER scratched_at;
