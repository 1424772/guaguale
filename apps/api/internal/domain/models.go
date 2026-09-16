package domain

import "time"

type User struct {
	ID                  uint64    `json:"id"`
	Username            string    `json:"username"`
	Balance             int64     `json:"balance"`
	LuckLevel           uint8     `json:"luckLevel"`
	ScratchLevel        uint8     `json:"scratchLevel"`
	TrashOwned          bool      `json:"trashOwned"`
	CardSlotsOwned      bool      `json:"cardSlotsOwned"`
	RobotOwned          bool      `json:"robotOwned"`
	RobotSpeedLevel     uint8     `json:"robotSpeedLevel"`
	RobotQueueLevel     uint8     `json:"robotQueueLevel"`
	RobotInterceptLevel uint8     `json:"robotInterceptLevel"`
	CreatedAt           time.Time `json:"createdAt"`
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
	TicketInTray  TicketLocation = "tray"
	TicketOnDesk  TicketLocation = "desk"
	TicketInSlot  TicketLocation = "slot"
	TicketInRobot TicketLocation = "robot"
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

type ShopItem struct {
	Code              string   `json:"code"`
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Level             uint8    `json:"level"`
	MaxLevel          uint8    `json:"maxLevel"`
	EffectPercent     int      `json:"effectPercent"`
	NextEffectPercent int      `json:"nextEffectPercent,omitempty"`
	EffectText        string   `json:"effectText"`
	NextEffectText    string   `json:"nextEffectText,omitempty"`
	NextPrice         int64    `json:"nextPrice,omitempty"`
	TotalSpent        int64    `json:"totalSpent"`
	Description       string   `json:"description"`
	Notice            string   `json:"notice,omitempty"`
	RelockedCards     []string `json:"relockedCards"`
	Locked            bool     `json:"locked"`
	LockedReason      string   `json:"lockedReason,omitempty"`
}

type ShopStatus struct {
	Items     []ShopItem       `json:"items"`
	LuckCards []LuckCardImpact `json:"luckCards"`
}

type LuckTierImpact struct {
	Label             string `json:"label"`
	RewardText        string `json:"rewardText"`
	CurrentBasisPoint int    `json:"currentBasisPoint"`
	NextBasisPoint    int    `json:"nextBasisPoint"`
}

type LuckCardImpact struct {
	CardCode             string           `json:"cardCode"`
	CardName             string           `json:"cardName"`
	CurrentLevel         uint8            `json:"currentLevel"`
	NextLevel            uint8            `json:"nextLevel"`
	CurrentRTPBasisPoint int              `json:"currentRtpBasisPoint"`
	NextRTPBasisPoint    int              `json:"nextRtpBasisPoint"`
	Tiers                []LuckTierImpact `json:"tiers"`
}

type ItemUpgrade struct {
	ID             string
	UserID         uint64
	ItemCode       string
	FromLevel      uint8
	ToLevel        uint8
	Price          int64
	IdempotencyKey string
	CreatedAt      time.Time
}

type RobotQueueItem struct {
	TicketID    string `json:"ticketId"`
	CardCode    string `json:"cardCode"`
	CardName    string `json:"cardName"`
	RemainingMS int64  `json:"remainingMs"`
	Position    int    `json:"position"`
}

type RobotStatus struct {
	Owned            bool             `json:"owned"`
	SpeedLevel       uint8            `json:"speedLevel"`
	QueueLevel       uint8            `json:"queueLevel"`
	InterceptLevel   uint8            `json:"interceptLevel"`
	DurationSeconds  int              `json:"durationSeconds"`
	Capacity         int              `json:"capacity"`
	InterceptPercent int              `json:"interceptPercent"`
	Queue            []RobotQueueItem `json:"queue"`
}

type RobotEvent struct {
	Ticket       Ticket `json:"ticket"`
	AutoRedeemed bool   `json:"autoRedeemed"`
}
