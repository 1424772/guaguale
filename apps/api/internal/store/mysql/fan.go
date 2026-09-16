package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func (store *Store) BlowFan(ctx context.Context, input basestore.BlowFanInput) (domain.User, domain.FanEvent, []domain.RobotQueueItem, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	defer tx.Rollback()
	var previousJSON []byte
	err = tx.QueryRowContext(ctx, `SELECT result_json FROM fan_events WHERE id = ? AND user_id = ?`, input.EventID, input.UserID).Scan(&previousJSON)
	if err == nil {
		user, err := lockedUser(ctx, tx, input.UserID)
		if err != nil {
			return domain.User{}, domain.FanEvent{}, nil, err
		}
		var previous domain.FanEvent
		if err := json.Unmarshal(previousJSON, &previous); err != nil {
			return domain.User{}, domain.FanEvent{}, nil, err
		}
		previous.Idempotent = true
		if err := tx.Commit(); err != nil {
			return domain.User{}, domain.FanEvent{}, nil, err
		}
		queue, err := store.ListRobotQueue(ctx, input.UserID)
		return user, previous, queue, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id, user_id, card_code, card_name, source, purchase_key, price, price_paid,
			COALESCE(DATE_FORMAT(wheel_date, '%Y-%m-%d'), ''), luck_level,
			prize_tier, reward, symbols, state, location, desk_x, desk_y, rotation, z_index, slot_index,
			created_at, scratched_at, redeemed_at, discarded_at
		FROM tickets
		WHERE user_id = ? AND location = 'desk' AND state IN ('purchased', 'scratched')
		ORDER BY z_index, id FOR UPDATE`, input.UserID)
	if err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	candidates := make([]domain.Ticket, 0)
	for rows.Next() {
		ticket, scanErr := scanTicket(rows)
		if scanErr != nil {
			rows.Close()
			return domain.User{}, domain.FanEvent{}, nil, scanErr
		}
		candidates = append(candidates, ticket)
	}
	if err := rows.Close(); err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	if err := rows.Err(); err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	user, err := lockedUser(ctx, tx, input.UserID)
	if err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	// A duplicate request may have waited behind the first request's ticket
	// locks. Recheck after acquiring the user lock so it returns the original
	// immutable result rather than processing a second event.
	err = tx.QueryRowContext(ctx, `SELECT result_json FROM fan_events WHERE id = ? AND user_id = ?`, input.EventID, input.UserID).Scan(&previousJSON)
	if err == nil {
		var previous domain.FanEvent
		if err := json.Unmarshal(previousJSON, &previous); err != nil {
			return domain.User{}, domain.FanEvent{}, nil, err
		}
		previous.Idempotent = true
		if err := tx.Commit(); err != nil {
			return domain.User{}, domain.FanEvent{}, nil, err
		}
		queue, err := store.ListRobotQueue(ctx, input.UserID)
		return user, previous, queue, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	if user.FanLevel == 0 {
		return domain.User{}, domain.FanEvent{}, nil, basestore.ErrFanRequired
	}
	if !user.TrashOwned {
		return domain.User{}, domain.FanEvent{}, nil, basestore.ErrTrashRequired
	}
	if !user.FanRiskAcknowledged && !input.AcknowledgeRisk {
		return domain.User{}, domain.FanEvent{}, nil, basestore.ErrFanRiskRequired
	}
	if input.AcknowledgeRisk && !user.FanRiskAcknowledged {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET fan_risk_acknowledged = TRUE WHERE id = ?`, user.ID); err != nil {
			return domain.User{}, domain.FanEvent{}, nil, err
		}
		user.FanRiskAcknowledged = true
	}
	var queueCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM robot_queue WHERE user_id = ?`, user.ID).Scan(&queueCount); err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	event := domain.FanEvent{ID: input.EventID, Cards: make([]domain.FanCardEvent, 0, len(candidates))}
	for index, ticket := range candidates {
		wasPurchased := ticket.State == domain.TicketPurchased
		action := "safe"
		if ticket.State == domain.TicketScratched {
			action = "discarded"
		} else {
			intercepted := ticket.CardCode != "all-in" && user.RobotOwned && fanRoll(input.Roll, 100) < input.InterceptPercent
			if intercepted && queueCount < input.RobotCapacity {
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO robot_queue (user_id, ticket_id, remaining_ms, enqueued_at) VALUES (?, ?, ?, ?)`,
					user.ID, ticket.ID, input.RobotDurationMS, input.Now.UTC()); err != nil {
					return domain.User{}, domain.FanEvent{}, nil, err
				}
				ticket.Location = domain.TicketInRobot
				ticket.SlotIndex = nil
				queueCount++
				action = "robot"
			} else if intercepted {
				action = "caught"
			} else if input.MistakePercent > 0 && fanRoll(input.Roll, 100) < input.MistakePercent {
				action = "discarded"
			}
		}
		if action == "discarded" {
			if _, err := tx.ExecContext(ctx, `
				UPDATE tickets SET state = 'discarded', slot_index = NULL, discarded_at = ? WHERE id = ?`, input.Now.UTC(), ticket.ID); err != nil {
				return domain.User{}, domain.FanEvent{}, nil, err
			}
			ticket.State = domain.TicketDiscarded
			ticket.SlotIndex = nil
			discardedAt := input.Now.UTC()
			ticket.DiscardedAt = &discardedAt
		} else if action == "robot" {
			if _, err := tx.ExecContext(ctx, `UPDATE tickets SET location = 'robot', slot_index = NULL WHERE id = ?`, ticket.ID); err != nil {
				return domain.User{}, domain.FanEvent{}, nil, err
			}
		} else {
			ticket.Location = domain.TicketOnDesk
			ticket.DeskX = .68 + float64(index%3)*.09
			ticket.DeskY = .58 + float64((index/3)%3)*.10
			ticket.Rotation = float64((index%5)-2) * 2
			ticket.ZIndex += len(candidates) + index + 1
			if _, err := tx.ExecContext(ctx, `
				UPDATE tickets SET location = 'desk', desk_x = ?, desk_y = ?, rotation = ?, z_index = ?, slot_index = NULL WHERE id = ?`,
				ticket.DeskX, ticket.DeskY, ticket.Rotation, ticket.ZIndex, ticket.ID); err != nil {
				return domain.User{}, domain.FanEvent{}, nil, err
			}
		}
		event.Cards = append(event.Cards, domain.FanCardEvent{Ticket: fanEventTicket(ticket, wasPurchased), Action: action})
	}
	resultJSON, err := json.Marshal(event)
	if err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO fan_events (id, user_id, result_json) VALUES (?, ?, ?)`, input.EventID, user.ID, resultJSON); err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.FanEvent{}, nil, err
	}
	queue, err := store.ListRobotQueue(ctx, user.ID)
	return user, event, queue, err
}

func fanRoll(random basestore.FanRollFunc, maximum int) int {
	if random == nil || maximum <= 0 {
		return 0
	}
	value := random(maximum)
	if value < 0 {
		return 0
	}
	return value % maximum
}

func fanEventTicket(ticket domain.Ticket, hideOutcome bool) domain.Ticket {
	if hideOutcome {
		ticket.PrizeTier = ""
		ticket.Reward = 0
		ticket.Symbols = nil
	}
	return ticket
}
