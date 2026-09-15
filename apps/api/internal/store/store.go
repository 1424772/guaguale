package store

import (
	"context"
	"errors"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrUsernameTaken     = errors.New("username already exists")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidState      = errors.New("invalid state")
	ErrNotWinner         = errors.New("ticket is not a winner")
)

type CreateTicketInput struct {
	ID          string
	UserID      uint64
	CardCode    string
	CardName    string
	PurchaseKey string
	Price       int64
	LuckLevel   uint8
	Outcome     domain.Outcome
}

type Store interface {
	Ping(context.Context) error
	CreateUser(context.Context, string, string, int64) (domain.User, error)
	UserByUsername(context.Context, string) (domain.User, string, error)
	UserBySession(context.Context, [32]byte) (domain.User, error)
	CreateSession(context.Context, domain.Session) error
	DeleteSession(context.Context, [32]byte) error
	PurchaseTicket(context.Context, CreateTicketInput) (domain.User, domain.Ticket, bool, error)
	ListTickets(context.Context, uint64) ([]domain.Ticket, error)
	ScratchTicket(context.Context, uint64, string) (domain.Ticket, error)
	RedeemTicket(context.Context, uint64, string) (domain.User, domain.Ticket, bool, error)
	DeleteExpiredSessions(context.Context, time.Time) error
}
