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

const (
	TicketPurchased TicketState = "purchased"
	TicketScratched TicketState = "scratched"
	TicketRedeemed  TicketState = "redeemed"
	TicketDiscarded TicketState = "discarded"
)

type Ticket struct {
	ID          string      `json:"id"`
	UserID      uint64      `json:"-"`
	CardCode    string      `json:"cardCode"`
	CardName    string      `json:"cardName"`
	Price       int64       `json:"price"`
	LuckLevel   uint8       `json:"luckLevel"`
	PrizeTier   string      `json:"prizeTier,omitempty"`
	Reward      int64       `json:"reward,omitempty"`
	Symbols     []string    `json:"symbols,omitempty"`
	State       TicketState `json:"state"`
	PurchaseKey string      `json:"-"`
	CreatedAt   time.Time   `json:"createdAt"`
	ScratchedAt *time.Time  `json:"scratchedAt,omitempty"`
	RedeemedAt  *time.Time  `json:"redeemedAt,omitempty"`
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
