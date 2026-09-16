package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func (store *Store) TopPlayers(ctx context.Context, limit int) ([]domain.RankedUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := store.db.QueryContext(ctx, `
		SELECT id, username, balance
		FROM users
		ORDER BY balance DESC, balance_ranked_at ASC, id ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.RankedUser, 0, limit)
	for rows.Next() {
		var player domain.RankedUser
		if err := rows.Scan(&player.UserID, &player.Username, &player.Balance); err != nil {
			return nil, err
		}
		player.Rank = len(result) + 1
		result = append(result, player)
	}
	return result, rows.Err()
}

func (store *Store) PlayerRank(ctx context.Context, userID uint64) (domain.RankedUser, int, error) {
	var player domain.RankedUser
	var total int
	err := store.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.balance,
			1 + (
				SELECT COUNT(*) FROM users ranked
				WHERE ranked.balance > u.balance
					OR (ranked.balance = u.balance AND ranked.balance_ranked_at < u.balance_ranked_at)
					OR (ranked.balance = u.balance AND ranked.balance_ranked_at = u.balance_ranked_at AND ranked.id < u.id)
			),
			(SELECT COUNT(*) FROM users)
		FROM users u WHERE u.id = ?`, userID,
	).Scan(&player.UserID, &player.Username, &player.Balance, &player.Rank, &total)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RankedUser{}, 0, basestore.ErrNotFound
	}
	return player, total, err
}

func (store *Store) GameHistory(ctx context.Context, userID uint64, limit int) ([]domain.HistoryEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	automated := make(map[string]bool)
	robotRows, err := store.db.QueryContext(ctx, `
		SELECT reference_id FROM coin_ledger
		WHERE user_id = ? AND reason = 'robot_redeem'`, userID)
	if err != nil {
		return nil, err
	}
	for robotRows.Next() {
		var ticketID string
		if err := robotRows.Scan(&ticketID); err != nil {
			robotRows.Close()
			return nil, err
		}
		automated[ticketID] = true
	}
	if err := robotRows.Close(); err != nil {
		return nil, err
	}

	rows, err := store.db.QueryContext(ctx, `
		SELECT id, user_id, card_code, card_name, source, purchase_key, price, price_paid,
			COALESCE(DATE_FORMAT(wheel_date, '%Y-%m-%d'), ''), luck_level,
			prize_tier, reward, symbols, state, location, desk_x, desk_y, rotation, z_index, slot_index,
			created_at, COALESCE(scratch_source, ''), scratched_at, redeemed_at, discarded_at
		FROM tickets
		WHERE user_id = ?
		ORDER BY GREATEST(created_at, COALESCE(scratched_at, created_at), COALESCE(redeemed_at, created_at), COALESCE(discarded_at, created_at)) DESC
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tickets := make([]domain.Ticket, 0, limit)
	for rows.Next() {
		ticket, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return basestore.BuildGameHistory(tickets, automated, limit), nil
}
