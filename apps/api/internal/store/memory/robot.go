package memory

import (
	"context"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func (store *Store) ListRobotQueue(_ context.Context, userID uint64) ([]domain.RobotQueueItem, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.users[userID]; !exists {
		return nil, basestore.ErrNotFound
	}
	return store.robotQueueLocked(userID), nil
}

func (store *Store) EnqueueRobot(_ context.Context, input basestore.EnqueueRobotInput) (domain.Ticket, []domain.RobotQueueItem, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[input.UserID]
	if !exists {
		return domain.Ticket{}, nil, basestore.ErrNotFound
	}
	if !record.user.RobotOwned {
		return domain.Ticket{}, nil, basestore.ErrRobotRequired
	}
	ticket, exists := store.tickets[input.TicketID]
	if !exists || ticket.UserID != input.UserID {
		return domain.Ticket{}, nil, basestore.ErrNotFound
	}
	if ticket.CardCode == "all-in" {
		return domain.Ticket{}, nil, basestore.ErrRobotUnsupported
	}
	if ticket.State != domain.TicketPurchased || ticket.Location == domain.TicketInSlot || ticket.Location == domain.TicketInRobot {
		return domain.Ticket{}, nil, basestore.ErrInvalidState
	}
	queue := store.robotQueues[input.UserID]
	if len(queue) >= input.Capacity {
		return domain.Ticket{}, nil, basestore.ErrRobotQueueFull
	}
	queue = append(queue, robotJob{ticketID: ticket.ID, remainingMS: input.DurationMS, enqueuedAt: input.EnqueuedAt})
	store.robotQueues[input.UserID] = queue
	ticket.Location = domain.TicketInRobot
	ticket.SlotIndex = nil
	store.tickets[ticket.ID] = ticket
	return publicTicket(ticket), store.robotQueueLocked(input.UserID), nil
}

func (store *Store) TickRobot(_ context.Context, userID uint64, now time.Time, durationMS int64) (domain.User, *domain.RobotEvent, []domain.RobotQueueItem, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[userID]
	if !exists {
		return domain.User{}, nil, nil, basestore.ErrNotFound
	}
	if !record.user.RobotOwned {
		return domain.User{}, nil, nil, basestore.ErrRobotRequired
	}
	queue := store.robotQueues[userID]
	lastTick, hadTick := store.robotLastTicks[userID]
	store.robotLastTicks[userID] = now
	if len(queue) == 0 {
		return record.user, nil, []domain.RobotQueueItem{}, nil
	}
	if durationMS > 0 && queue[0].remainingMS > durationMS {
		queue[0].remainingMS = durationMS
	}
	progressMS := int64(0)
	if hadTick {
		delta := now.Sub(lastTick)
		if delta > 0 && delta <= 3*time.Second {
			progressMS = delta.Milliseconds()
			if progressMS > 2000 {
				progressMS = 2000
			}
		}
	}
	queue[0].remainingMS -= progressMS
	if queue[0].remainingMS > 0 {
		store.robotQueues[userID] = queue
		return record.user, nil, store.robotQueueLocked(userID), nil
	}
	job := queue[0]
	ticket, exists := store.tickets[job.ticketID]
	if !exists || ticket.UserID != userID || ticket.State != domain.TicketPurchased || ticket.Location != domain.TicketInRobot {
		return domain.User{}, nil, nil, basestore.ErrInvalidState
	}
	ticket.State = domain.TicketScratched
	ticket.ScratchedAt = &now
	event := &domain.RobotEvent{}
	if ticket.Reward > 0 {
		record.user.Balance += ticket.Reward
		ticket.State = domain.TicketRedeemed
		ticket.RedeemedAt = &now
		event.AutoRedeemed = true
	} else {
		ticket.Location = domain.TicketOnDesk
		ticket.DeskX = .78
		ticket.DeskY = .72
		ticket.Rotation = 3
		ticket.ZIndex++
	}
	store.users[userID] = record
	store.tickets[ticket.ID] = ticket
	store.robotQueues[userID] = append([]robotJob(nil), queue[1:]...)
	event.Ticket = revealTicket(ticket)
	return record.user, event, store.robotQueueLocked(userID), nil
}

func (store *Store) robotQueueLocked(userID uint64) []domain.RobotQueueItem {
	queue := store.robotQueues[userID]
	result := make([]domain.RobotQueueItem, 0, len(queue))
	for index, job := range queue {
		ticket, exists := store.tickets[job.ticketID]
		if !exists {
			continue
		}
		result = append(result, domain.RobotQueueItem{
			TicketID: ticket.ID, CardCode: ticket.CardCode, CardName: ticket.CardName,
			RemainingMS: job.remainingMS, Position: index + 1,
		})
	}
	return result
}
