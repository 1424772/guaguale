package memory

import (
	"context"
	"sort"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func (store *Store) TopPlayers(_ context.Context, limit int) ([]domain.RankedUser, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	ranked := store.rankedUsersLocked()
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return append([]domain.RankedUser(nil), ranked...), nil
}

func (store *Store) PlayerRank(_ context.Context, userID uint64) (domain.RankedUser, int, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.users[userID]; !exists {
		return domain.RankedUser{}, 0, basestore.ErrNotFound
	}
	ranked := store.rankedUsersLocked()
	for _, player := range ranked {
		if player.UserID == userID {
			return player, len(ranked), nil
		}
	}
	return domain.RankedUser{}, len(ranked), basestore.ErrNotFound
}

func (store *Store) GameHistory(_ context.Context, userID uint64, limit int) ([]domain.HistoryEvent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.users[userID]; !exists {
		return nil, basestore.ErrNotFound
	}
	tickets := make([]domain.Ticket, 0)
	for _, ticket := range store.tickets {
		if ticket.UserID == userID {
			tickets = append(tickets, ticket)
		}
	}
	automated := make(map[string]bool)
	for ticketID, processed := range store.robotProcessed {
		if processed {
			automated[ticketID] = true
		}
	}
	return basestore.BuildGameHistory(tickets, automated, limit), nil
}

func (store *Store) rankedUsersLocked() []domain.RankedUser {
	type candidate struct {
		user domain.User
	}
	candidates := make([]candidate, 0, len(store.users))
	for _, record := range store.users {
		candidates = append(candidates, candidate{user: record.user})
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].user.Balance != candidates[right].user.Balance {
			return candidates[left].user.Balance > candidates[right].user.Balance
		}
		leftAt := store.balanceRankedAt[candidates[left].user.ID]
		rightAt := store.balanceRankedAt[candidates[right].user.ID]
		if !leftAt.Equal(rightAt) {
			return leftAt.Before(rightAt)
		}
		return candidates[left].user.ID < candidates[right].user.ID
	})
	result := make([]domain.RankedUser, len(candidates))
	for index, item := range candidates {
		result[index] = domain.RankedUser{
			Rank: index + 1, UserID: item.user.ID, Username: item.user.Username, Balance: item.user.Balance,
		}
	}
	return result
}
