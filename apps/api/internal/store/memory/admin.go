package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func (store *Store) AdminUsers(_ context.Context, query string, limit int) ([]basestore.AdminUser, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if limit < 1 || limit > 100 {
		limit = 50
	}
	query = strings.ToLower(strings.TrimSpace(query))
	users := make([]basestore.AdminUser, 0)
	for _, record := range store.users {
		if query != "" && !strings.Contains(strings.ToLower(record.user.Username), query) {
			continue
		}
		users = append(users, adminUser(record, ticketCountForUser(store.tickets, record.user.ID)))
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID > users[j].ID })
	if len(users) > limit {
		users = users[:limit]
	}
	return users, nil
}

func (store *Store) AdminAdjustBalance(_ context.Context, input basestore.AdminBalanceInput) (basestore.AdminUser, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[input.UserID]
	if !exists {
		return basestore.AdminUser{}, false, basestore.ErrNotFound
	}
	if adjustedUserID, exists := store.adminAdjustments[input.IdempotencyKey]; exists {
		adjustedRecord := store.users[adjustedUserID]
		return adminUser(adjustedRecord, ticketCountForUser(store.tickets, adjustedUserID)), true, nil
	}
	delta := input.Amount
	if input.Mode == "subtract" {
		delta = -input.Amount
	} else if input.Mode == "set" {
		delta = input.Amount - record.user.Balance
	}
	if record.user.Balance+delta < 0 {
		return basestore.AdminUser{}, false, basestore.ErrInsufficientFunds
	}
	record.user.Balance += delta
	store.users[input.UserID] = record
	store.balanceRankedAt[input.UserID] = time.Now().UTC()
	store.adminAdjustments[input.IdempotencyKey] = input.UserID
	return adminUser(record, ticketCountForUser(store.tickets, input.UserID)), false, nil
}

func ticketCountForUser(tickets map[string]domain.Ticket, userID uint64) int {
	count := 0
	for _, ticket := range tickets {
		if ticket.UserID == userID {
			count++
		}
	}
	return count
}

func adminUser(record userRecord, ticketCount int) basestore.AdminUser {
	return basestore.AdminUser{
		ID:             record.user.ID,
		Username:       record.user.Username,
		Balance:        record.user.Balance,
		LuckLevel:      record.user.LuckLevel,
		ScratchLevel:   record.user.ScratchLevel,
		TrashOwned:     record.user.TrashOwned,
		CardSlotsOwned: record.user.CardSlotsOwned,
		FanLevel:       record.user.FanLevel,
		RobotOwned:     record.user.RobotOwned,
		TicketCount:    ticketCount,
		CreatedAt:      record.user.CreatedAt,
		UpdatedAt:      storeTime(record.user.CreatedAt),
	}
}

func storeTime(created time.Time) time.Time {
	if created.IsZero() {
		return time.Now().UTC()
	}
	return created
}
