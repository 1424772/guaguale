package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func (store *Store) ListRobotQueue(ctx context.Context, userID uint64) ([]domain.RobotQueueItem, error) {
	rows, err := store.db.QueryContext(ctx, `
		SELECT q.ticket_id, t.card_code, t.card_name, q.remaining_ms
		FROM robot_queue q JOIN tickets t ON t.id = q.ticket_id
		WHERE q.user_id = ? ORDER BY q.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.RobotQueueItem, 0)
	for rows.Next() {
		var item domain.RobotQueueItem
		if err := rows.Scan(&item.TicketID, &item.CardCode, &item.CardName, &item.RemainingMS); err != nil {
			return nil, err
		}
		item.Position = len(result) + 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (store *Store) EnqueueRobot(ctx context.Context, input basestore.EnqueueRobotInput) (domain.Ticket, []domain.RobotQueueItem, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.Ticket{}, nil, err
	}
	defer tx.Rollback()
	ticket, err := lockedTicket(ctx, tx, input.UserID, input.TicketID)
	if err != nil {
		return domain.Ticket{}, nil, err
	}
	user, err := lockedUser(ctx, tx, input.UserID)
	if err != nil {
		return domain.Ticket{}, nil, err
	}
	if !user.RobotOwned {
		return domain.Ticket{}, nil, basestore.ErrRobotRequired
	}
	if ticket.CardCode == "all-in" {
		return domain.Ticket{}, nil, basestore.ErrRobotUnsupported
	}
	if ticket.State != domain.TicketPurchased || ticket.Location == domain.TicketInSlot || ticket.Location == domain.TicketInRobot {
		return domain.Ticket{}, nil, basestore.ErrInvalidState
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM robot_queue WHERE user_id = ?`, input.UserID).Scan(&count); err != nil {
		return domain.Ticket{}, nil, err
	}
	if count >= input.Capacity {
		return domain.Ticket{}, nil, basestore.ErrRobotQueueFull
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO robot_queue (user_id, ticket_id, remaining_ms, enqueued_at) VALUES (?, ?, ?, ?)`,
		input.UserID, ticket.ID, input.DurationMS, input.EnqueuedAt.UTC(),
	); err != nil {
		return domain.Ticket{}, nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tickets SET location = 'robot', slot_index = NULL WHERE id = ?`, ticket.ID); err != nil {
		return domain.Ticket{}, nil, err
	}
	ticket.Location = domain.TicketInRobot
	ticket.SlotIndex = nil
	if err := tx.Commit(); err != nil {
		return domain.Ticket{}, nil, err
	}
	queue, err := store.ListRobotQueue(ctx, input.UserID)
	return publicTicket(ticket), queue, err
}

func (store *Store) TickRobot(ctx context.Context, userID uint64, now time.Time, durationMS int64) (domain.User, *domain.RobotEvent, []domain.RobotQueueItem, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, nil, nil, err
	}
	defer tx.Rollback()
	var jobID uint64
	var ticketID string
	var remainingMS int64
	err = tx.QueryRowContext(ctx, `SELECT id, ticket_id, remaining_ms FROM robot_queue WHERE user_id = ? ORDER BY id LIMIT 1`, userID).Scan(&jobID, &ticketID, &remainingMS)
	if errors.Is(err, sql.ErrNoRows) {
		user, userErr := lockedUser(ctx, tx, userID)
		if userErr != nil {
			return domain.User{}, nil, nil, userErr
		}
		if !user.RobotOwned {
			return domain.User{}, nil, nil, basestore.ErrRobotRequired
		}
		if err := touchRobotRuntime(ctx, tx, userID, now); err != nil {
			return domain.User{}, nil, nil, err
		}
		if err := tx.Commit(); err != nil {
			return domain.User{}, nil, nil, err
		}
		return user, nil, []domain.RobotQueueItem{}, nil
	}
	if err != nil {
		return domain.User{}, nil, nil, err
	}
	ticket, err := lockedTicket(ctx, tx, userID, ticketID)
	if err != nil {
		return domain.User{}, nil, nil, err
	}
	user, err := lockedUser(ctx, tx, userID)
	if err != nil {
		return domain.User{}, nil, nil, err
	}
	if !user.RobotOwned {
		return domain.User{}, nil, nil, basestore.ErrRobotRequired
	}
	var lockedTicketID string
	err = tx.QueryRowContext(ctx, `SELECT ticket_id, remaining_ms FROM robot_queue WHERE id = ? FOR UPDATE`, jobID).Scan(&lockedTicketID, &remainingMS)
	if errors.Is(err, sql.ErrNoRows) {
		// A concurrent heartbeat may have finished this job while this
		// transaction waited for the ticket lock. Treat that tick as an
		// idempotent refresh instead of surfacing a transient server error.
		if err := tx.Commit(); err != nil {
			return domain.User{}, nil, nil, err
		}
		queue, err := store.ListRobotQueue(ctx, userID)
		return user, nil, queue, err
	}
	if err != nil {
		return domain.User{}, nil, nil, err
	}
	if lockedTicketID != ticket.ID || ticket.State != domain.TicketPurchased || ticket.Location != domain.TicketInRobot {
		return domain.User{}, nil, nil, basestore.ErrInvalidState
	}
	if durationMS > 0 && remainingMS > durationMS {
		remainingMS = durationMS
	}
	progressMS, err := robotProgress(ctx, tx, userID, now)
	if err != nil {
		return domain.User{}, nil, nil, err
	}
	remainingMS -= progressMS
	if remainingMS > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE robot_queue SET remaining_ms = ? WHERE id = ?`, remainingMS, jobID); err != nil {
			return domain.User{}, nil, nil, err
		}
		if err := tx.Commit(); err != nil {
			return domain.User{}, nil, nil, err
		}
		queue, err := store.ListRobotQueue(ctx, userID)
		return user, nil, queue, err
	}
	now = now.UTC()
	ticket.State = domain.TicketScratched
	ticket.ScratchedAt = &now
	event := &domain.RobotEvent{}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tickets SET state = 'scratched', location = 'desk', desk_x = 0.805000, desk_y = 0.665000,
			rotation = 3, z_index = z_index + 1, slot_index = NULL, scratch_source = 'robot',
			scratched_at = ?, redeemed_at = NULL WHERE id = ?`, now, ticket.ID,
	); err != nil {
		return domain.User{}, nil, nil, err
	}
	ticket.Location = domain.TicketOnDesk
	ticket.DeskX = .805
	ticket.DeskY = .665
	ticket.Rotation = 3
	ticket.ZIndex++
	ticket.SlotIndex = nil
	ticket.ScratchSource = "robot"
	ticket.RedeemedAt = nil
	if _, err := tx.ExecContext(ctx, `DELETE FROM robot_queue WHERE id = ?`, jobID); err != nil {
		return domain.User{}, nil, nil, err
	}
	event.Ticket = revealTicket(ticket)
	if err := tx.Commit(); err != nil {
		return domain.User{}, nil, nil, err
	}
	queue, err := store.ListRobotQueue(ctx, userID)
	return user, event, queue, err
}

func robotProgress(ctx context.Context, tx *sql.Tx, userID uint64, now time.Time) (int64, error) {
	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO robot_runtime (user_id) VALUES (?)`, userID); err != nil {
		return 0, err
	}
	var lastTick sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT last_tick_at FROM robot_runtime WHERE user_id = ? FOR UPDATE`, userID).Scan(&lastTick); err != nil {
		return 0, err
	}
	progressMS := int64(0)
	if lastTick.Valid {
		delta := now.Sub(lastTick.Time)
		if delta > 0 && delta <= 3*time.Second {
			progressMS = delta.Milliseconds()
			if progressMS > 2000 {
				progressMS = 2000
			}
		}
	}
	_, err := tx.ExecContext(ctx, `UPDATE robot_runtime SET last_tick_at = ? WHERE user_id = ?`, now.UTC(), userID)
	return progressMS, err
}

func touchRobotRuntime(ctx context.Context, tx *sql.Tx, userID uint64, now time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO robot_runtime (user_id, last_tick_at) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE last_tick_at = VALUES(last_tick_at)`, userID, now.UTC())
	return err
}
