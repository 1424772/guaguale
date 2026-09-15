package domain

import "time"

type User struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Balance   int64     `json:"balance"`
	LuckLevel uint8     `json:"luckLevel"`
	CreatedAt time.Time `json:"createdAt"`
}

type Session struct {
	TokenHash [32]byte
	UserID    uint64
	ExpiresAt time.Time
}

type TicketState string
type TicketLocation string

const (
	TicketPurchased TicketState = "purchased"
	TicketScratched TicketState = "scratched"
	TicketRedeemed  TicketState = "redeemed"
	TicketDiscarded TicketState = "discarded"
)

const (
	TicketInTray TicketLocation = "tray"
	TicketOnDesk TicketLocation = "desk"
	TicketInSlot TicketLocation = "slot"
)

type Ticket struct {
	ID          string         `json:"id"`
	UserID      uint64         `json:"-"`
	CardCode    string         `json:"cardCode"`
	CardName    string         `json:"cardName"`
	Price       int64          `json:"nominalPrice"`
	PricePaid   int64          `json:"pricePaid"`
	Source      string         `json:"source"`
	WheelDate   string         `json:"wheelDate,omitempty"`
	LuckLevel   uint8          `json:"luckLevel"`
	PrizeTier   string         `json:"prizeTier,omitempty"`
	Reward      int64          `json:"reward,omitempty"`
	Symbols     []string       `json:"symbols,omitempty"`
	State       TicketState    `json:"state"`
	Location    TicketLocation `json:"location"`
	DeskX       float64        `json:"deskX"`
	DeskY       float64        `json:"deskY"`
	Rotation    float64        `json:"rotation"`
	ZIndex      int            `json:"zIndex"`
	SlotIndex   *int           `json:"slotIndex,omitempty"`
	PurchaseKey string         `json:"-"`
	CreatedAt   time.Time      `json:"createdAt"`
	ScratchedAt *time.Time     `json:"scratchedAt,omitempty"`
	RedeemedAt  *time.Time     `json:"redeemedAt,omitempty"`
	DiscardedAt *time.Time     `json:"discardedAt,omitempty"`
}

type WheelPoolItem struct {
	CardCode   string `json:"cardCode"`
	CardName   string `json:"cardName"`
	Price      int64  `json:"price"`
	Weight     int    `json:"weight"`
	BasisPoint int    `json:"basisPoint"`
}

type PlateAction struct {
	ID          string     `json:"id"`
	Sequence    int        `json:"sequence"`
	State       string     `json:"state"`
	StartedAt   time.Time  `json:"startedAt"`
	AvailableAt time.Time  `json:"availableAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type DailyStatus struct {
	Date            string          `json:"date"`
	LoginClaimed    bool            `json:"loginClaimed"`
	PlatesCompleted int             `json:"platesCompleted"`
	PlateLimit      int             `json:"plateLimit"`
	ActivePlate     *PlateAction    `json:"activePlate,omitempty"`
	WheelUsed       bool            `json:"wheelUsed"`
	WheelTicketID   string          `json:"wheelTicketId,omitempty"`
	WheelCardCode   string          `json:"wheelCardCode,omitempty"`
	WheelCardName   string          `json:"wheelCardName,omitempty"`
	WheelPool       []WheelPoolItem `json:"wheelPool"`
}

type Card struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Price       int64  `json:"price"`
	Implemented bool   `json:"implemented"`
	Unlocked    bool   `json:"unlocked"`
}

type Outcome struct {
	PrizeTier string
	Reward    int64
	Symbols   []string
}
