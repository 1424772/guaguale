package mysql

import (
	"context"
	"database/sql"
	"errors"

	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func (store *Store) AdminUsers(ctx context.Context, query string, limit int) ([]basestore.AdminUser, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := store.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.balance, u.luck_level, u.scratch_level,
			u.trash_owned, u.card_slots_owned, u.fan_level, u.robot_owned,
			COUNT(t.id), u.created_at, u.updated_at
		FROM users u
		LEFT JOIN tickets t ON t.user_id = u.id
		WHERE (? = '' OR u.username LIKE CONCAT('%', ?, '%'))
		GROUP BY u.id
		ORDER BY u.id DESC
		LIMIT ?`, query, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]basestore.AdminUser, 0)
	for rows.Next() {
		var user basestore.AdminUser
		if err := rows.Scan(
			&user.ID, &user.Username, &user.Balance, &user.LuckLevel, &user.ScratchLevel,
			&user.TrashOwned, &user.CardSlotsOwned, &user.FanLevel, &user.RobotOwned,
			&user.TicketCount, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (store *Store) AdminAdjustBalance(ctx context.Context, input basestore.AdminBalanceInput) (basestore.AdminUser, bool, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return basestore.AdminUser{}, false, err
	}
	defer tx.Rollback()

	var before int64
	if err := tx.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = ? FOR UPDATE`, input.UserID).Scan(&before); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return basestore.AdminUser{}, false, basestore.ErrNotFound
		}
		return basestore.AdminUser{}, false, err
	}

	var existing int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM coin_ledger WHERE idempotency_key = ?`, input.IdempotencyKey).Scan(&existing); err != nil {
		return basestore.AdminUser{}, false, err
	}
	if existing == 0 {
		delta := input.Amount
		if input.Mode == "subtract" {
			delta = -input.Amount
		} else if input.Mode == "set" {
			delta = input.Amount - before
		}
		after := before + delta
		if after < 0 {
			return basestore.AdminUser{}, false, basestore.ErrInsufficientFunds
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = ?, balance_ranked_at = UTC_TIMESTAMP(6) WHERE id = ?`, after, input.UserID); err != nil {
			return basestore.AdminUser{}, false, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO coin_ledger
				(user_id, idempotency_key, reason, reference_type, reference_id, delta, balance_before, balance_after)
			VALUES (?, ?, 'admin_adjustment', 'admin', ?, ?, ?, ?)`,
			input.UserID, input.IdempotencyKey, input.IdempotencyKey, delta, before, after,
		); err != nil {
			return basestore.AdminUser{}, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return basestore.AdminUser{}, false, err
	}

	user, err := store.adminUserByID(ctx, input.UserID)
	return user, existing > 0, err
}

func (store *Store) adminUserByID(ctx context.Context, userID uint64) (basestore.AdminUser, error) {
	var user basestore.AdminUser
	err := store.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.balance, u.luck_level, u.scratch_level,
			u.trash_owned, u.card_slots_owned, u.fan_level, u.robot_owned,
			COUNT(t.id), u.created_at, u.updated_at
		FROM users u
		LEFT JOIN tickets t ON t.user_id = u.id
		WHERE u.id = ?
		GROUP BY u.id`, userID).Scan(
		&user.ID, &user.Username, &user.Balance, &user.LuckLevel, &user.ScratchLevel,
		&user.TrashOwned, &user.CardSlotsOwned, &user.FanLevel, &user.RobotOwned,
		&user.TicketCount, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return basestore.AdminUser{}, basestore.ErrNotFound
	}
	return user, err
}
