package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

type userRecord struct {
	user         domain.User
	passwordHash string
}

type Store struct {
	mu                sync.Mutex
	nextUserID        uint64
	users             map[uint64]userRecord
	usernames         map[string]uint64
	sessions          map[[32]byte]domain.Session
	tickets           map[string]domain.Ticket
	purchaseByUserKey map[string]string
}

func New() *Store {
	return &Store{
		nextUserID:        1,
		users:             make(map[uint64]userRecord),
		usernames:         make(map[string]uint64),
		sessions:          make(map[[32]byte]domain.Session),
		tickets:           make(map[string]domain.Ticket),
		purchaseByUserKey: make(map[string]string),
	}
}

func (store *Store) Ping(context.Context) error { return nil }

func (store *Store) CreateUser(_ context.Context, username, passwordHash string, initialBalance int64) (domain.User, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.usernames[username]; exists {
		return domain.User{}, basestore.ErrUsernameTaken
	}
	user := domain.User{
		ID:        store.nextUserID,
		Username:  username,
		Balance:   initialBalance,
		LuckLevel: 0,
		CreatedAt: time.Now().UTC(),
	}
	store.nextUserID++
	store.users[user.ID] = userRecord{user: user, passwordHash: passwordHash}
	store.usernames[username] = user.ID
	return user, nil
}

func (store *Store) UserByUsername(_ context.Context, username string) (domain.User, string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	userID, exists := store.usernames[username]
	if !exists {
		return domain.User{}, "", basestore.ErrNotFound
	}
	record := store.users[userID]
	return record.user, record.passwordHash, nil
}

func (store *Store) UserBySession(_ context.Context, tokenHash [32]byte) (domain.User, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	session, exists := store.sessions[tokenHash]
	if !exists || !session.ExpiresAt.After(time.Now()) {
		return domain.User{}, basestore.ErrNotFound
	}
	record, exists := store.users[session.UserID]
	if !exists {
		return domain.User{}, basestore.ErrNotFound
	}
	return record.user, nil
}

func (store *Store) CreateSession(_ context.Context, session domain.Session) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.users[session.UserID]; !exists {
		return basestore.ErrNotFound
	}
	store.sessions[session.TokenHash] = session
	return nil
}

func (store *Store) DeleteSession(_ context.Context, tokenHash [32]byte) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.sessions, tokenHash)
	return nil
}

func (store *Store) PurchaseTicket(_ context.Context, input basestore.CreateTicketInput) (domain.User, domain.Ticket, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[input.UserID]
	if !exists {
		return domain.User{}, domain.Ticket{}, false, basestore.ErrNotFound
	}
	key := purchaseKey(input.UserID, input.PurchaseKey)
	if ticketID, exists := store.purchaseByUserKey[key]; exists {
		return record.user, publicTicket(store.tickets[ticketID]), true, nil
	}
	if record.user.Balance < input.Price {
		return domain.User{}, domain.Ticket{}, false, basestore.ErrInsufficientFunds
	}
	record.user.Balance -= input.Price
	store.users[input.UserID] = record
	now := time.Now().UTC()
	ticket := domain.Ticket{
		ID:          input.ID,
		UserID:      input.UserID,
		CardCode:    input.CardCode,
		CardName:    input.CardName,
		Price:       input.Price,
		LuckLevel:   input.LuckLevel,
		PrizeTier:   input.Outcome.PrizeTier,
		Reward:      input.Outcome.Reward,
		Symbols:     append([]string(nil), input.Outcome.Symbols...),
		State:       domain.TicketPurchased,
		PurchaseKey: input.PurchaseKey,
		CreatedAt:   now,
	}
	store.tickets[ticket.ID] = ticket
	store.purchaseByUserKey[key] = ticket.ID
	return record.user, publicTicket(ticket), false, nil
}

func (store *Store) ListTickets(_ context.Context, userID uint64) ([]domain.Ticket, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.users[userID]; !exists {
		return nil, basestore.ErrNotFound
	}
	result := make([]domain.Ticket, 0)
	for _, ticket := range store.tickets {
		if ticket.UserID == userID && ticket.State != domain.TicketDiscarded {
			result = append(result, publicTicket(ticket))
		}
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].CreatedAt.After(result[right].CreatedAt)
	})
	return result, nil
}

func (store *Store) ScratchTicket(_ context.Context, userID uint64, ticketID string) (domain.Ticket, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	ticket, exists := store.tickets[ticketID]
	if !exists || ticket.UserID != userID {
		return domain.Ticket{}, basestore.ErrNotFound
	}
	if ticket.State == domain.TicketPurchased {
		now := time.Now().UTC()
		ticket.State = domain.TicketScratched
		ticket.ScratchedAt = &now
		store.tickets[ticketID] = ticket
	}
	if ticket.State != domain.TicketScratched && ticket.State != domain.TicketRedeemed {
		return domain.Ticket{}, basestore.ErrInvalidState
	}
	return revealTicket(ticket), nil
}

func (store *Store) RedeemTicket(_ context.Context, userID uint64, ticketID string) (domain.User, domain.Ticket, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	ticket, exists := store.tickets[ticketID]
	if !exists || ticket.UserID != userID {
		return domain.User{}, domain.Ticket{}, false, basestore.ErrNotFound
	}
	record := store.users[userID]
	if ticket.State == domain.TicketRedeemed {
		return record.user, revealTicket(ticket), true, nil
	}
	if ticket.State != domain.TicketScratched {
		return domain.User{}, domain.Ticket{}, false, basestore.ErrInvalidState
	}
	if ticket.Reward <= 0 {
		return domain.User{}, domain.Ticket{}, false, basestore.ErrNotWinner
	}
	record.user.Balance += ticket.Reward
	store.users[userID] = record
	now := time.Now().UTC()
	ticket.State = domain.TicketRedeemed
	ticket.RedeemedAt = &now
	store.tickets[ticketID] = ticket
	return record.user, revealTicket(ticket), false, nil
}

func (store *Store) DeleteExpiredSessions(_ context.Context, cutoff time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	for tokenHash, session := range store.sessions {
		if !session.ExpiresAt.After(cutoff) {
			delete(store.sessions, tokenHash)
		}
	}
	return nil
}

func purchaseKey(userID uint64, key string) string {
	return fmt.Sprintf("%d:%s", userID, key)
}

func publicTicket(ticket domain.Ticket) domain.Ticket {
	if ticket.State == domain.TicketPurchased {
		ticket.PrizeTier = ""
		ticket.Reward = 0
		ticket.Symbols = nil
	} else {
		ticket.Symbols = append([]string(nil), ticket.Symbols...)
	}
	return ticket
}

func revealTicket(ticket domain.Ticket) domain.Ticket {
	ticket.Symbols = append([]string(nil), ticket.Symbols...)
	return ticket
}
