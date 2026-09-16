package memory

import (
	"context"
	"fmt"
	"sort"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func (store *Store) BlowFan(_ context.Context, input basestore.BlowFanInput) (domain.User, domain.FanEvent, []domain.RobotQueueItem, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[input.UserID]
	if !exists {
		return domain.User{}, domain.FanEvent{}, nil, basestore.ErrNotFound
	}
	if record.user.FanLevel == 0 {
		return domain.User{}, domain.FanEvent{}, nil, basestore.ErrFanRequired
	}
	if !record.user.TrashOwned {
		return domain.User{}, domain.FanEvent{}, nil, basestore.ErrTrashRequired
	}
	key := fmt.Sprintf("%d:%s", input.UserID, input.EventID)
	if previous, exists := store.fanEvents[key]; exists {
		previous.Idempotent = true
		return record.user, previous, store.robotQueueLocked(input.UserID), nil
	}
	if !record.user.FanRiskAcknowledged && !input.AcknowledgeRisk {
		return domain.User{}, domain.FanEvent{}, nil, basestore.ErrFanRiskRequired
	}
	if input.AcknowledgeRisk && !record.user.FanRiskAcknowledged {
		record.user.FanRiskAcknowledged = true
		store.users[input.UserID] = record
	}

	candidates := make([]domain.Ticket, 0)
	for _, ticket := range store.tickets {
		if ticket.UserID == input.UserID && ticket.Location == domain.TicketOnDesk &&
			(ticket.State == domain.TicketPurchased || ticket.State == domain.TicketScratched) {
			candidates = append(candidates, ticket)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ZIndex == candidates[j].ZIndex {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].ZIndex < candidates[j].ZIndex
	})
	event := domain.FanEvent{ID: input.EventID, Cards: make([]domain.FanCardEvent, 0, len(candidates))}
	queue := store.robotQueues[input.UserID]
	for index, ticket := range candidates {
		wasPurchased := ticket.State == domain.TicketPurchased
		action := "safe"
		if ticket.State == domain.TicketScratched {
			action = "discarded"
		} else {
			intercepted := ticket.CardCode != "all-in" && record.user.RobotOwned && roll(input.Roll, 100) < input.InterceptPercent
			if intercepted && len(queue) < input.RobotCapacity {
				ticket.Location = domain.TicketInRobot
				ticket.SlotIndex = nil
				queue = append(queue, robotJob{ticketID: ticket.ID, remainingMS: input.RobotDurationMS, enqueuedAt: input.Now})
				action = "robot"
			} else if intercepted {
				action = "caught"
			} else if input.MistakePercent > 0 && roll(input.Roll, 100) < input.MistakePercent {
				action = "discarded"
			}
		}
		if action == "discarded" {
			ticket.State = domain.TicketDiscarded
			ticket.SlotIndex = nil
			discardedAt := input.Now
			ticket.DiscardedAt = &discardedAt
		} else if action != "robot" {
			ticket.Location = domain.TicketOnDesk
			ticket.DeskX = .68 + float64(index%3)*.09
			ticket.DeskY = .58 + float64((index/3)%3)*.10
			ticket.Rotation = float64((index%5)-2) * 2
			ticket.ZIndex += len(candidates) + index + 1
		}
		store.tickets[ticket.ID] = ticket
		event.Cards = append(event.Cards, domain.FanCardEvent{Ticket: fanPublicTicket(ticket, wasPurchased), Action: action})
	}
	store.robotQueues[input.UserID] = queue
	store.fanEvents[key] = event
	return record.user, event, store.robotQueueLocked(input.UserID), nil
}

func roll(random basestore.FanRollFunc, maximum int) int {
	if random == nil || maximum <= 0 {
		return 0
	}
	value := random(maximum)
	if value < 0 {
		return 0
	}
	return value % maximum
}

func fanPublicTicket(ticket domain.Ticket, hideOutcome bool) domain.Ticket {
	if hideOutcome {
		ticket.PrizeTier = ""
		ticket.Reward = 0
		ticket.Symbols = nil
		return ticket
	}
	return revealTicket(ticket)
}
