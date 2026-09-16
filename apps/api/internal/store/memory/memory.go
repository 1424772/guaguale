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
	daily             map[string]domain.DailyStatus
	plateActions      map[string]plateRecord
	upgradeByUserKey  map[string]domain.ItemUpgrade
	robotQueues       map[uint64][]robotJob
	robotLastTicks    map[uint64]time.Time
}

type robotJob struct {
	ticketID    string
	remainingMS int64
	enqueuedAt  time.Time
}

type plateRecord struct {
	userID uint64
	date   string
	action domain.PlateAction
}

func New() *Store {
	return &Store{
		nextUserID:        1,
		users:             make(map[uint64]userRecord),
		usernames:         make(map[string]uint64),
		sessions:          make(map[[32]byte]domain.Session),
		tickets:           make(map[string]domain.Ticket),
		purchaseByUserKey: make(map[string]string),
		daily:             make(map[string]domain.DailyStatus),
		plateActions:      make(map[string]plateRecord),
		upgradeByUserKey:  make(map[string]domain.ItemUpgrade),
		robotQueues:       make(map[uint64][]robotJob),
		robotLastTicks:    make(map[uint64]time.Time),
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
		ID:           store.nextUserID,
		Username:     username,
		Balance:      initialBalance,
		LuckLevel:    0,
		ScratchLevel: 1,
		CreatedAt:    time.Now().UTC(),
	}
	store.nextUserID++
	store.users[user.ID] = userRecord{user: user, passwordHash: passwordHash}
	store.usernames[username] = user.ID
	return user, nil
}

func (store *Store) UpgradeItem(_ context.Context, input basestore.UpgradeItemInput) (domain.User, domain.ItemUpgrade, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[input.UserID]
	if !exists {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrNotFound
	}
	key := purchaseKey(input.UserID, input.IdempotencyKey)
	if upgrade, exists := store.upgradeByUserKey[key]; exists {
		return record.user, upgrade, true, nil
	}
	currentLevel := record.user.LuckLevel
	if input.ItemCode == "scratch-range" {
		currentLevel = record.user.ScratchLevel
	} else if input.ItemCode == "trash" {
		currentLevel = boolLevel(record.user.TrashOwned)
	} else if input.ItemCode == "card-slots" {
		currentLevel = boolLevel(record.user.CardSlotsOwned)
	} else if input.ItemCode == "robot" {
		currentLevel = boolLevel(record.user.RobotOwned)
	} else if input.ItemCode == "robot-speed" {
		if !record.user.RobotOwned {
			return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrRobotRequired
		}
		currentLevel = record.user.RobotSpeedLevel
	} else if input.ItemCode == "robot-queue" {
		if !record.user.RobotOwned {
			return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrRobotRequired
		}
		currentLevel = record.user.RobotQueueLevel
	} else if input.ItemCode == "robot-intercept" {
		if !record.user.RobotOwned {
			return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrRobotRequired
		}
		currentLevel = record.user.RobotInterceptLevel
	} else if input.ItemCode != "luck" {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrInvalidState
	}
	if currentLevel != input.ExpectedFromLevel || input.ToLevel != currentLevel+1 {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrUpgradeConflict
	}
	if currentLevel >= input.MaxLevel || input.Price <= 0 {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrInvalidState
	}
	if record.user.Balance < input.Price {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrInsufficientFunds
	}
	record.user.Balance -= input.Price
	if input.ItemCode == "luck" {
		record.user.LuckLevel = input.ToLevel
	} else if input.ItemCode == "scratch-range" {
		record.user.ScratchLevel = input.ToLevel
	} else if input.ItemCode == "trash" {
		record.user.TrashOwned = true
	} else if input.ItemCode == "card-slots" {
		record.user.CardSlotsOwned = true
	} else if input.ItemCode == "robot" {
		record.user.RobotOwned = true
		record.user.RobotSpeedLevel = 1
		record.user.RobotQueueLevel = 1
		record.user.RobotInterceptLevel = 1
	} else if input.ItemCode == "robot-speed" {
		record.user.RobotSpeedLevel = input.ToLevel
	} else if input.ItemCode == "robot-queue" {
		record.user.RobotQueueLevel = input.ToLevel
	} else {
		record.user.RobotInterceptLevel = input.ToLevel
	}
	store.users[input.UserID] = record
	upgrade := domain.ItemUpgrade{
		ID: input.ID, UserID: input.UserID, ItemCode: input.ItemCode,
		FromLevel: currentLevel, ToLevel: input.ToLevel, Price: input.Price,
		IdempotencyKey: input.IdempotencyKey, CreatedAt: time.Now().UTC(),
	}
	store.upgradeByUserKey[key] = upgrade
	return record.user, upgrade, false, nil
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
		PricePaid:   input.Price,
		Source:      "purchase",
		LuckLevel:   input.LuckLevel,
		PrizeTier:   input.Outcome.PrizeTier,
		Reward:      input.Outcome.Reward,
		Symbols:     append([]string(nil), input.Outcome.Symbols...),
		State:       domain.TicketPurchased,
		Location:    domain.TicketInTray,
		DeskX:       .5,
		DeskY:       .35,
		ZIndex:      1,
		PurchaseKey: input.PurchaseKey,
		CreatedAt:   now,
	}
	store.tickets[ticket.ID] = ticket
	store.purchaseByUserKey[key] = ticket.ID
	return record.user, publicTicket(ticket), false, nil
}

func (store *Store) DailyStatus(_ context.Context, userID uint64, date string) (domain.DailyStatus, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.users[userID]; !exists {
		return domain.DailyStatus{}, basestore.ErrNotFound
	}
	return cloneDaily(store.dailyStatus(userID, date)), nil
}

func (store *Store) ClaimDailyLogin(_ context.Context, userID uint64, date string, amount int64) (domain.User, domain.DailyStatus, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[userID]
	if !exists {
		return domain.User{}, domain.DailyStatus{}, false, basestore.ErrNotFound
	}
	daily := store.dailyStatus(userID, date)
	if daily.LoginClaimed {
		return record.user, cloneDaily(daily), true, nil
	}
	record.user.Balance += amount
	store.users[userID] = record
	daily.LoginClaimed = true
	store.daily[dailyKey(userID, date)] = daily
	return record.user, cloneDaily(daily), false, nil
}

func (store *Store) StartPlate(_ context.Context, userID uint64, date string, action domain.PlateAction) (domain.DailyStatus, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.users[userID]; !exists {
		return domain.DailyStatus{}, false, basestore.ErrNotFound
	}
	daily := store.dailyStatus(userID, date)
	if daily.PlatesCompleted >= 5 {
		return domain.DailyStatus{}, false, basestore.ErrDailyLimit
	}
	if daily.ActivePlate != nil && daily.ActivePlate.State == "started" {
		return cloneDaily(daily), true, nil
	}
	action.Sequence = daily.PlatesCompleted + 1
	actionCopy := action
	daily.ActivePlate = &actionCopy
	store.daily[dailyKey(userID, date)] = daily
	store.plateActions[action.ID] = plateRecord{userID: userID, date: date, action: action}
	return cloneDaily(daily), false, nil
}

func (store *Store) CompletePlate(_ context.Context, userID uint64, date, actionID string, completedAt time.Time, amount int64) (domain.User, domain.DailyStatus, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[userID]
	if !exists {
		return domain.User{}, domain.DailyStatus{}, false, basestore.ErrNotFound
	}
	plate, exists := store.plateActions[actionID]
	if !exists || plate.userID != userID || plate.date != date {
		return domain.User{}, domain.DailyStatus{}, false, basestore.ErrNotFound
	}
	daily := store.dailyStatus(userID, date)
	if plate.action.State == "completed" {
		return record.user, cloneDaily(daily), true, nil
	}
	if completedAt.Before(plate.action.AvailableAt) {
		return domain.User{}, domain.DailyStatus{}, false, basestore.ErrTooEarly
	}
	if daily.PlatesCompleted >= 5 {
		return domain.User{}, domain.DailyStatus{}, false, basestore.ErrDailyLimit
	}
	record.user.Balance += amount
	store.users[userID] = record
	plate.action.State = "completed"
	plate.action.CompletedAt = &completedAt
	store.plateActions[actionID] = plate
	daily.PlatesCompleted++
	daily.ActivePlate = nil
	store.daily[dailyKey(userID, date)] = daily
	return record.user, cloneDaily(daily), false, nil
}

func (store *Store) SpinDailyWheel(_ context.Context, userID uint64, date, ticketID string, draw basestore.WheelDrawFunc) (domain.User, domain.Ticket, domain.DailyStatus, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, exists := store.users[userID]
	if !exists {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, basestore.ErrNotFound
	}
	daily := store.dailyStatus(userID, date)
	if daily.WheelUsed {
		ticket, exists := store.tickets[daily.WheelTicketID]
		if !exists {
			return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, basestore.ErrNotFound
		}
		return record.user, publicTicket(ticket), cloneDaily(daily), true, nil
	}
	selection, err := draw(record.user.Balance, record.user.LuckLevel)
	if err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	now := time.Now().UTC()
	ticket := domain.Ticket{
		ID:        ticketID,
		UserID:    userID,
		CardCode:  selection.CardCode,
		CardName:  selection.CardName,
		Price:     selection.NominalPrice,
		PricePaid: 0,
		Source:    "daily_wheel",
		WheelDate: date,
		LuckLevel: record.user.LuckLevel,
		PrizeTier: selection.Outcome.PrizeTier,
		Reward:    selection.Outcome.Reward,
		Symbols:   append([]string(nil), selection.Outcome.Symbols...),
		State:     domain.TicketPurchased,
		Location:  domain.TicketInTray,
		DeskX:     .5,
		DeskY:     .35,
		ZIndex:    1,
		CreatedAt: now,
	}
	store.tickets[ticket.ID] = ticket
	daily.WheelUsed = true
	daily.WheelTicketID = ticket.ID
	daily.WheelCardCode = ticket.CardCode
	daily.WheelCardName = ticket.CardName
	daily.WheelPool = append([]domain.WheelPoolItem(nil), selection.Pool...)
	store.daily[dailyKey(userID, date)] = daily
	return record.user, publicTicket(ticket), cloneDaily(daily), false, nil
}

func (store *Store) ListTickets(_ context.Context, userID uint64) ([]domain.Ticket, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.users[userID]; !exists {
		return nil, basestore.ErrNotFound
	}
	result := make([]domain.Ticket, 0)
	for _, ticket := range store.tickets {
		if ticket.UserID == userID && (ticket.State == domain.TicketPurchased || ticket.State == domain.TicketScratched) {
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
		if ticket.Location == domain.TicketInRobot {
			return domain.Ticket{}, basestore.ErrRobotManaged
		}
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
	ticket.SlotIndex = nil
	ticket.RedeemedAt = &now
	store.tickets[ticketID] = ticket
	return record.user, revealTicket(ticket), false, nil
}

func (store *Store) UpdateTicketPlacement(_ context.Context, userID uint64, ticketID string, placement basestore.TicketPlacement) (domain.Ticket, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	ticket, exists := store.tickets[ticketID]
	if !exists || ticket.UserID != userID {
		return domain.Ticket{}, basestore.ErrNotFound
	}
	if ticket.State != domain.TicketPurchased && ticket.State != domain.TicketScratched {
		return domain.Ticket{}, basestore.ErrInvalidState
	}
	if ticket.Location == domain.TicketInRobot {
		return domain.Ticket{}, basestore.ErrRobotManaged
	}
	if placement.Location == domain.TicketInSlot && !store.users[userID].user.CardSlotsOwned {
		return domain.Ticket{}, basestore.ErrCardSlotsRequired
	}
	if placement.Location == domain.TicketInSlot {
		for id, candidate := range store.tickets {
			if id != ticketID && candidate.UserID == userID && candidate.SlotIndex != nil && placement.SlotIndex != nil && *candidate.SlotIndex == *placement.SlotIndex && (candidate.State == domain.TicketPurchased || candidate.State == domain.TicketScratched) {
				return domain.Ticket{}, basestore.ErrSlotOccupied
			}
		}
	}
	ticket.Location = placement.Location
	ticket.DeskX = placement.DeskX
	ticket.DeskY = placement.DeskY
	ticket.Rotation = placement.Rotation
	ticket.ZIndex = placement.ZIndex
	ticket.SlotIndex = placement.SlotIndex
	store.tickets[ticketID] = ticket
	return publicTicket(ticket), nil
}

func (store *Store) DiscardTicket(_ context.Context, userID uint64, ticketID string, discardedAt time.Time) (domain.Ticket, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	ticket, exists := store.tickets[ticketID]
	if !exists || ticket.UserID != userID {
		return domain.Ticket{}, basestore.ErrNotFound
	}
	if ticket.Location == domain.TicketInSlot {
		return domain.Ticket{}, basestore.ErrProtected
	}
	if ticket.Location == domain.TicketInRobot {
		return domain.Ticket{}, basestore.ErrRobotManaged
	}
	if !store.users[userID].user.TrashOwned {
		return domain.Ticket{}, basestore.ErrTrashRequired
	}
	if ticket.State != domain.TicketPurchased && ticket.State != domain.TicketScratched {
		return domain.Ticket{}, basestore.ErrInvalidState
	}
	ticket.State = domain.TicketDiscarded
	ticket.SlotIndex = nil
	ticket.DiscardedAt = &discardedAt
	store.tickets[ticketID] = ticket
	return revealTicket(ticket), nil
}

func boolLevel(value bool) uint8 {
	if value {
		return 1
	}
	return 0
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

func dailyKey(userID uint64, date string) string {
	return fmt.Sprintf("%d:%s", userID, date)
}

func (store *Store) dailyStatus(userID uint64, date string) domain.DailyStatus {
	if daily, exists := store.daily[dailyKey(userID, date)]; exists {
		return daily
	}
	return domain.DailyStatus{Date: date, PlateLimit: 5, WheelPool: []domain.WheelPoolItem{}}
}

func cloneDaily(daily domain.DailyStatus) domain.DailyStatus {
	daily.WheelPool = append([]domain.WheelPoolItem(nil), daily.WheelPool...)
	if daily.ActivePlate != nil {
		copy := *daily.ActivePlate
		daily.ActivePlate = &copy
	}
	return daily
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
