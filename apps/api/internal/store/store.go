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
	ErrDailyLimit        = errors.New("daily limit reached")
	ErrTooEarly          = errors.New("action completed too early")
	ErrWheelUnavailable  = errors.New("wheel is unavailable")
	ErrSlotOccupied      = errors.New("card slot is occupied")
	ErrProtected         = errors.New("protected ticket cannot be discarded")
	ErrUpgradeConflict   = errors.New("item level changed")
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

type WheelSelection struct {
	CardCode     string
	CardName     string
	NominalPrice int64
	Outcome      domain.Outcome
	Pool         []domain.WheelPoolItem
}

type WheelDrawFunc func(balance int64, luckLevel uint8) (WheelSelection, error)

type TicketPlacement struct {
	Location  domain.TicketLocation
	DeskX     float64
	DeskY     float64
	Rotation  float64
	ZIndex    int
	SlotIndex *int
}

type UpgradeItemInput struct {
	ID                string
	UserID            uint64
	ItemCode          string
	ExpectedFromLevel uint8
	ToLevel           uint8
	MaxLevel          uint8
	Price             int64
	IdempotencyKey    string
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
	UpdateTicketPlacement(context.Context, uint64, string, TicketPlacement) (domain.Ticket, error)
	DiscardTicket(context.Context, uint64, string, time.Time) (domain.Ticket, error)
	UpgradeItem(context.Context, UpgradeItemInput) (domain.User, domain.ItemUpgrade, bool, error)
	DailyStatus(context.Context, uint64, string) (domain.DailyStatus, error)
	ClaimDailyLogin(context.Context, uint64, string, int64) (domain.User, domain.DailyStatus, bool, error)
	StartPlate(context.Context, uint64, string, domain.PlateAction) (domain.DailyStatus, bool, error)
	CompletePlate(context.Context, uint64, string, string, time.Time, int64) (domain.User, domain.DailyStatus, bool, error)
	SpinDailyWheel(context.Context, uint64, string, string, WheelDrawFunc) (domain.User, domain.Ticket, domain.DailyStatus, bool, error)
	DeleteExpiredSessions(context.Context, time.Time) error
}
