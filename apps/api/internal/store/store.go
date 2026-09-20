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
	ErrTrashRequired     = errors.New("trash item is required")
	ErrCardSlotsRequired = errors.New("card slots item is required")
	ErrRobotRequired     = errors.New("robot item is required")
	ErrRobotQueueFull    = errors.New("robot queue is full")
	ErrRobotManaged      = errors.New("ticket is managed by robot")
	ErrRobotUnsupported  = errors.New("ticket does not support robot")
	ErrFanRequired       = errors.New("fan item is required")
	ErrFanRiskRequired   = errors.New("fan risk acknowledgement is required")
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

type EnqueueRobotInput struct {
	UserID     uint64
	TicketID   string
	DurationMS int64
	Capacity   int
	EnqueuedAt time.Time
}

type FanRollFunc func(max int) int

type BlowFanInput struct {
	UserID           uint64
	EventID          string
	AcknowledgeRisk  bool
	MistakePercent   int
	InterceptPercent int
	RobotCapacity    int
	RobotDurationMS  int64
	Now              time.Time
	Roll             FanRollFunc
}

type AdminUser struct {
	ID             uint64    `json:"id"`
	Username       string    `json:"username"`
	Balance        int64     `json:"balance"`
	LuckLevel      uint8     `json:"luckLevel"`
	ScratchLevel   uint8     `json:"scratchLevel"`
	TrashOwned     bool      `json:"trashOwned"`
	CardSlotsOwned bool      `json:"cardSlotsOwned"`
	FanLevel       uint8     `json:"fanLevel"`
	RobotOwned     bool      `json:"robotOwned"`
	TicketCount    int       `json:"ticketCount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AdminBalanceInput struct {
	UserID         uint64
	Mode           string
	Amount         int64
	IdempotencyKey string
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
	RevealTicket(context.Context, uint64, string) (domain.Ticket, error)
	ScratchTicket(context.Context, uint64, string) (domain.Ticket, error)
	RedeemTicket(context.Context, uint64, string) (domain.User, domain.Ticket, bool, error)
	UpdateTicketPlacement(context.Context, uint64, string, TicketPlacement) (domain.Ticket, error)
	DiscardTicket(context.Context, uint64, string, time.Time) (domain.Ticket, error)
	RestoreDiscardedTicket(context.Context, uint64, string, time.Time) (domain.Ticket, error)
	UpgradeItem(context.Context, UpgradeItemInput) (domain.User, domain.ItemUpgrade, bool, error)
	ListRobotQueue(context.Context, uint64) ([]domain.RobotQueueItem, error)
	EnqueueRobot(context.Context, EnqueueRobotInput) (domain.Ticket, []domain.RobotQueueItem, error)
	TickRobot(context.Context, uint64, time.Time, int64) (domain.User, *domain.RobotEvent, []domain.RobotQueueItem, error)
	BlowFan(context.Context, BlowFanInput) (domain.User, domain.FanEvent, []domain.RobotQueueItem, error)
	DailyStatus(context.Context, uint64, string) (domain.DailyStatus, error)
	ClaimDailyLogin(context.Context, uint64, string, int64) (domain.User, domain.DailyStatus, bool, error)
	StartPlate(context.Context, uint64, string, domain.PlateAction) (domain.DailyStatus, bool, error)
	CompletePlate(context.Context, uint64, string, string, time.Time, int64) (domain.User, domain.DailyStatus, bool, error)
	SpinDailyWheel(context.Context, uint64, string, string, WheelDrawFunc) (domain.User, domain.Ticket, domain.DailyStatus, bool, error)
	DeleteExpiredSessions(context.Context, time.Time) error
	TopPlayers(context.Context, int) ([]domain.RankedUser, error)
	PlayerRank(context.Context, uint64) (domain.RankedUser, int, error)
	GameHistory(context.Context, uint64, int) ([]domain.HistoryEvent, error)
	AdminUsers(context.Context, string, int) ([]AdminUser, error)
	AdminAdjustBalance(context.Context, AdminBalanceInput) (AdminUser, bool, error)
}
