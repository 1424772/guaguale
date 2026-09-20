package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/1424772/guaguale/apps/api/internal/auth"
	"github.com/1424772/guaguale/apps/api/internal/domain"
	"github.com/1424772/guaguale/apps/api/internal/game"
	"github.com/1424772/guaguale/apps/api/internal/store"
)

const InitialBalance int64 = 1000

const (
	DailyLoginReward int64 = 100
	PlateReward      int64 = 5
	PlateLimit             = 5
	PlateDuration          = 3 * time.Second
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrCardUnavailable    = errors.New("card is not available")
	ErrItemUnavailable    = errors.New("item is not available")
	usernamePattern       = regexp.MustCompile(`^[\p{Han}A-Za-z0-9_]{3,32}$`)
)

type Service struct {
	store            store.Store
	now              func() time.Time
	drawCard         func(string, uint8) (domain.Outcome, error)
	fanRoll          store.FanRollFunc
	leaderboardCache LeaderboardCache
}

type AuthResult struct {
	User      domain.User
	Token     string
	ExpiresAt time.Time
}

type PurchaseResult struct {
	User       domain.User   `json:"user"`
	Ticket     domain.Ticket `json:"ticket"`
	Idempotent bool          `json:"idempotent"`
}

type RedeemResult struct {
	User       domain.User   `json:"user"`
	Ticket     domain.Ticket `json:"ticket"`
	Idempotent bool          `json:"idempotent"`
}

type DailyResult struct {
	User       domain.User        `json:"user"`
	Daily      domain.DailyStatus `json:"daily"`
	Idempotent bool               `json:"idempotent"`
}

type WheelResult struct {
	User       domain.User        `json:"user"`
	Ticket     domain.Ticket      `json:"ticket"`
	Daily      domain.DailyStatus `json:"daily"`
	Idempotent bool               `json:"idempotent"`
}

type UpgradeResult struct {
	User       domain.User        `json:"user"`
	Shop       domain.ShopStatus  `json:"shop"`
	Upgrade    domain.ItemUpgrade `json:"-"`
	ItemCode   string             `json:"itemCode"`
	Idempotent bool               `json:"idempotent"`
}

type RobotEnqueueResult struct {
	Ticket domain.Ticket      `json:"ticket"`
	Robot  domain.RobotStatus `json:"robot"`
}

type RobotTickResult struct {
	User  domain.User        `json:"user"`
	Robot domain.RobotStatus `json:"robot"`
	Event *domain.RobotEvent `json:"event,omitempty"`
}

type FanResult struct {
	User  domain.User        `json:"user"`
	Fan   domain.FanStatus   `json:"fan"`
	Robot domain.RobotStatus `json:"robot"`
	Event domain.FanEvent    `json:"event"`
}

func New(store store.Store) *Service {
	return &Service{store: store, now: time.Now, drawCard: game.Draw, fanRoll: secureRoll}
}

func NewWithLeaderboardCache(store store.Store, cache LeaderboardCache) *Service {
	service := New(store)
	service.leaderboardCache = cache
	return service
}

func (service *Service) Register(ctx context.Context, username, password string, ageConfirmed bool) (AuthResult, error) {
	username = strings.TrimSpace(username)
	if !ageConfirmed || !validUsername(username) || !validPassword(password) {
		return AuthResult{}, ErrInvalidInput
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return AuthResult{}, err
	}
	user, err := service.store.CreateUser(ctx, username, passwordHash, InitialBalance)
	if err != nil {
		return AuthResult{}, err
	}
	return service.createSession(ctx, user)
}

func (service *Service) Login(ctx context.Context, username, password string) (AuthResult, error) {
	username = strings.TrimSpace(username)
	if !validUsername(username) || !validPassword(password) {
		return AuthResult{}, ErrInvalidCredentials
	}
	user, passwordHash, err := service.store.UserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}
	if !auth.CheckPassword(passwordHash, password) {
		return AuthResult{}, ErrInvalidCredentials
	}
	return service.createSession(ctx, user)
}

func (service *Service) UserByToken(ctx context.Context, token string) (domain.User, error) {
	if len(token) < 32 || len(token) > 128 {
		return domain.User{}, store.ErrNotFound
	}
	return service.store.UserBySession(ctx, auth.HashToken(token))
}

func (service *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return service.store.DeleteSession(ctx, auth.HashToken(token))
}

func (service *Service) Cards(balance int64) []domain.Card {
	return game.Catalog(balance)
}

func (service *Service) Shop(user domain.User) domain.ShopStatus {
	return game.Shop(user.Balance, user.LuckLevel, user.ScratchLevel, user.TrashOwned, user.CardSlotsOwned,
		user.FanLevel, user.RobotOwned, user.RobotSpeedLevel, user.RobotQueueLevel, user.RobotInterceptLevel)
}

func (service *Service) UpgradeItem(ctx context.Context, user domain.User, itemCode, idempotencyKey string) (UpgradeResult, error) {
	if !validIdempotencyKey(idempotencyKey) {
		return UpgradeResult{}, ErrInvalidInput
	}
	currentLevel := user.LuckLevel
	if itemCode == game.ScratchRangeItemCode {
		currentLevel = user.ScratchLevel
	} else if itemCode == game.TrashItemCode {
		if user.TrashOwned {
			currentLevel = 1
		} else {
			currentLevel = 0
		}
	} else if itemCode == game.CardSlotsItemCode {
		if user.CardSlotsOwned {
			currentLevel = 1
		} else {
			currentLevel = 0
		}
	} else if itemCode == game.FanItemCode {
		if !user.TrashOwned {
			return UpgradeResult{}, store.ErrTrashRequired
		}
		currentLevel = user.FanLevel
	} else if itemCode == game.RobotItemCode {
		currentLevel = boolLevel(user.RobotOwned)
	} else if itemCode == game.RobotSpeedItemCode {
		if !user.RobotOwned {
			return UpgradeResult{}, store.ErrRobotRequired
		}
		currentLevel = user.RobotSpeedLevel
	} else if itemCode == game.RobotQueueItemCode {
		if !user.RobotOwned {
			return UpgradeResult{}, store.ErrRobotRequired
		}
		currentLevel = user.RobotQueueLevel
	} else if itemCode == game.RobotInterceptItemCode {
		if !user.RobotOwned {
			return UpgradeResult{}, store.ErrRobotRequired
		}
		currentLevel = user.RobotInterceptLevel
	} else if itemCode != game.LuckItemCode {
		return UpgradeResult{}, ErrItemUnavailable
	}
	item, available := game.UpgradeDefinition(itemCode, currentLevel)
	if !available {
		return UpgradeResult{}, ErrItemUnavailable
	}
	upgradeID, err := randomID()
	if err != nil {
		return UpgradeResult{}, err
	}
	updatedUser, upgrade, idempotent, err := service.store.UpgradeItem(ctx, store.UpgradeItemInput{
		ID: upgradeID, UserID: user.ID, ItemCode: itemCode, ExpectedFromLevel: currentLevel,
		ToLevel: currentLevel + 1, MaxLevel: item.MaxLevel, Price: item.NextPrice, IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if errors.Is(err, store.ErrInvalidState) && currentLevel >= item.MaxLevel {
			return UpgradeResult{}, ErrItemUnavailable
		}
		return UpgradeResult{}, err
	}
	return UpgradeResult{
		User: updatedUser, Shop: service.Shop(updatedUser), Upgrade: upgrade,
		ItemCode: upgrade.ItemCode, Idempotent: idempotent,
	}, nil
}

func boolLevel(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

func (service *Service) RobotStatus(ctx context.Context, user domain.User) (domain.RobotStatus, error) {
	queue, err := service.store.ListRobotQueue(ctx, user.ID)
	if err != nil {
		return domain.RobotStatus{}, err
	}
	return robotStatus(user, queue), nil
}

func (service *Service) EnqueueRobot(ctx context.Context, user domain.User, ticketID string) (RobotEnqueueResult, error) {
	if !validTicketID(ticketID) {
		return RobotEnqueueResult{}, ErrInvalidInput
	}
	if !user.RobotOwned {
		return RobotEnqueueResult{}, store.ErrRobotRequired
	}
	durationMS := int64(game.RobotDurationSeconds(user.RobotSpeedLevel)) * 1000
	ticket, queue, err := service.store.EnqueueRobot(ctx, store.EnqueueRobotInput{
		UserID: user.ID, TicketID: ticketID, DurationMS: durationMS,
		Capacity: game.RobotQueueCapacity(user.RobotQueueLevel), EnqueuedAt: service.now().UTC(),
	})
	if err != nil {
		return RobotEnqueueResult{}, err
	}
	return RobotEnqueueResult{Ticket: ticket, Robot: robotStatus(user, queue)}, nil
}

func (service *Service) TickRobot(ctx context.Context, user domain.User) (RobotTickResult, error) {
	if !user.RobotOwned {
		return RobotTickResult{}, store.ErrRobotRequired
	}
	durationMS := int64(game.RobotDurationSeconds(user.RobotSpeedLevel)) * 1000
	updatedUser, event, queue, err := service.store.TickRobot(ctx, user.ID, service.now().UTC(), durationMS)
	if err != nil {
		return RobotTickResult{}, err
	}
	return RobotTickResult{User: updatedUser, Robot: robotStatus(updatedUser, queue), Event: event}, nil
}

func robotStatus(user domain.User, queue []domain.RobotQueueItem) domain.RobotStatus {
	if queue == nil {
		queue = []domain.RobotQueueItem{}
	}
	return domain.RobotStatus{
		Owned: user.RobotOwned, SpeedLevel: user.RobotSpeedLevel, QueueLevel: user.RobotQueueLevel,
		InterceptLevel: user.RobotInterceptLevel, DurationSeconds: game.RobotDurationSeconds(user.RobotSpeedLevel),
		Capacity: game.RobotQueueCapacity(user.RobotQueueLevel), InterceptPercent: game.RobotInterceptPercent(user.RobotInterceptLevel),
		Queue: queue,
	}
}

func (service *Service) FanStatus(user domain.User) domain.FanStatus {
	interceptPercent := 0
	if user.RobotOwned {
		interceptPercent = game.RobotInterceptPercent(user.RobotInterceptLevel)
	}
	mistakePercent := game.FanMistakePercent(user.FanLevel)
	return domain.FanStatus{
		Owned: user.FanLevel > 0, Level: user.FanLevel, ForceText: game.FanForceText(user.FanLevel),
		MistakePercent:            mistakePercent,
		UnprotectedDiscardPercent: game.FanDiscardPercent(mistakePercent, interceptPercent),
		RiskAcknowledged:          user.FanRiskAcknowledged,
	}
}

func (service *Service) BlowFan(ctx context.Context, user domain.User, eventID string, acknowledgeRisk bool) (FanResult, error) {
	if !validIdempotencyKey(eventID) {
		return FanResult{}, ErrInvalidInput
	}
	if user.FanLevel == 0 {
		return FanResult{}, store.ErrFanRequired
	}
	if !user.TrashOwned {
		return FanResult{}, store.ErrTrashRequired
	}
	interceptPercent := 0
	if user.RobotOwned {
		interceptPercent = game.RobotInterceptPercent(user.RobotInterceptLevel)
	}
	updatedUser, event, queue, err := service.store.BlowFan(ctx, store.BlowFanInput{
		UserID: user.ID, EventID: eventID, AcknowledgeRisk: acknowledgeRisk,
		MistakePercent: game.FanMistakePercent(user.FanLevel), InterceptPercent: interceptPercent,
		RobotCapacity:   game.RobotQueueCapacity(user.RobotQueueLevel),
		RobotDurationMS: int64(game.RobotDurationSeconds(user.RobotSpeedLevel)) * 1000,
		Now:             service.now().UTC(), Roll: service.fanRoll,
	})
	if err != nil {
		return FanResult{}, err
	}
	return FanResult{User: updatedUser, Fan: service.FanStatus(updatedUser), Robot: robotStatus(updatedUser, queue), Event: event}, nil
}

func secureRoll(maximum int) int {
	if maximum <= 1 {
		return 0
	}
	value, err := rand.Int(rand.Reader, big.NewInt(int64(maximum)))
	if err != nil {
		return maximum - 1
	}
	return int(value.Int64())
}

func (service *Service) Purchase(ctx context.Context, user domain.User, cardCode, idempotencyKey string) (PurchaseResult, error) {
	if !validIdempotencyKey(idempotencyKey) {
		return PurchaseResult{}, ErrInvalidInput
	}
	card, exists := game.CardByCode(cardCode)
	if !exists || !card.Implemented {
		return PurchaseResult{}, ErrCardUnavailable
	}
	outcome, err := service.drawCard(card.Code, user.LuckLevel)
	if err != nil {
		return PurchaseResult{}, err
	}
	ticketID, err := randomID()
	if err != nil {
		return PurchaseResult{}, err
	}
	updatedUser, ticket, idempotent, err := service.store.PurchaseTicket(ctx, store.CreateTicketInput{
		ID:          ticketID,
		UserID:      user.ID,
		CardCode:    card.Code,
		CardName:    card.Name,
		PurchaseKey: idempotencyKey,
		Price:       card.Price,
		LuckLevel:   user.LuckLevel,
		Outcome:     outcome,
	})
	if err != nil {
		return PurchaseResult{}, err
	}
	return PurchaseResult{User: updatedUser, Ticket: ticket, Idempotent: idempotent}, nil
}

func (service *Service) DailyStatus(ctx context.Context, user domain.User) (domain.DailyStatus, error) {
	date := shanghaiDate(service.now())
	status, err := service.store.DailyStatus(ctx, user.ID, date)
	if err != nil {
		return domain.DailyStatus{}, err
	}
	status.Date = date
	status.PlateLimit = PlateLimit
	if !status.WheelUsed {
		status.WheelPool = game.WheelPool(user.Balance)
	}
	return status, nil
}

func (service *Service) ClaimDailyLogin(ctx context.Context, userID uint64) (DailyResult, error) {
	date := shanghaiDate(service.now())
	user, daily, idempotent, err := service.store.ClaimDailyLogin(ctx, userID, date, DailyLoginReward)
	if err != nil {
		return DailyResult{}, err
	}
	daily.PlateLimit = PlateLimit
	if !daily.WheelUsed {
		daily.WheelPool = game.WheelPool(user.Balance)
	}
	return DailyResult{User: user, Daily: daily, Idempotent: idempotent}, nil
}

func (service *Service) StartPlate(ctx context.Context, user domain.User) (domain.DailyStatus, bool, error) {
	now := service.now().UTC()
	actionID, err := randomID()
	if err != nil {
		return domain.DailyStatus{}, false, err
	}
	date := shanghaiDate(now)
	status, idempotent, err := service.store.StartPlate(ctx, user.ID, date, domain.PlateAction{
		ID:          actionID,
		State:       "started",
		StartedAt:   now,
		AvailableAt: now.Add(PlateDuration),
	})
	if err != nil {
		return domain.DailyStatus{}, false, err
	}
	status.PlateLimit = PlateLimit
	if !status.WheelUsed {
		status.WheelPool = game.WheelPool(user.Balance)
	}
	return status, idempotent, nil
}

func (service *Service) CompletePlate(ctx context.Context, userID uint64, actionID string) (DailyResult, error) {
	if !validTicketID(actionID) {
		return DailyResult{}, ErrInvalidInput
	}
	now := service.now().UTC()
	date := shanghaiDate(now)
	user, daily, idempotent, err := service.store.CompletePlate(ctx, userID, date, actionID, now, PlateReward)
	if err != nil {
		return DailyResult{}, err
	}
	daily.PlateLimit = PlateLimit
	if !daily.WheelUsed {
		daily.WheelPool = game.WheelPool(user.Balance)
	}
	return DailyResult{User: user, Daily: daily, Idempotent: idempotent}, nil
}

func (service *Service) SpinDailyWheel(ctx context.Context, userID uint64) (WheelResult, error) {
	ticketID, err := randomID()
	if err != nil {
		return WheelResult{}, err
	}
	date := shanghaiDate(service.now())
	user, ticket, daily, idempotent, err := service.store.SpinDailyWheel(ctx, userID, date, ticketID, service.drawWheel)
	if err != nil {
		return WheelResult{}, err
	}
	daily.PlateLimit = PlateLimit
	return WheelResult{User: user, Ticket: ticket, Daily: daily, Idempotent: idempotent}, nil
}

func (service *Service) drawWheel(balance int64, luckLevel uint8) (store.WheelSelection, error) {
	card, pool, err := game.DrawWheelCard(balance)
	if err != nil {
		return store.WheelSelection{}, store.ErrWheelUnavailable
	}
	outcome, err := service.drawCard(card.Code, luckLevel)
	if err != nil {
		return store.WheelSelection{}, err
	}
	return store.WheelSelection{
		CardCode:     card.Code,
		CardName:     card.Name,
		NominalPrice: card.Price,
		Outcome:      outcome,
		Pool:         pool,
	}, nil
}

func (service *Service) Tickets(ctx context.Context, userID uint64) ([]domain.Ticket, error) {
	return service.store.ListTickets(ctx, userID)
}

func (service *Service) Reveal(ctx context.Context, userID uint64, ticketID string) (domain.Ticket, error) {
	if !validTicketID(ticketID) {
		return domain.Ticket{}, ErrInvalidInput
	}
	return service.store.RevealTicket(ctx, userID, ticketID)
}

func (service *Service) Scratch(ctx context.Context, userID uint64, ticketID string) (domain.Ticket, error) {
	if !validTicketID(ticketID) {
		return domain.Ticket{}, ErrInvalidInput
	}
	return service.store.ScratchTicket(ctx, userID, ticketID)
}

func (service *Service) Redeem(ctx context.Context, userID uint64, ticketID string) (RedeemResult, error) {
	if !validTicketID(ticketID) {
		return RedeemResult{}, ErrInvalidInput
	}
	user, ticket, idempotent, err := service.store.RedeemTicket(ctx, userID, ticketID)
	if err != nil {
		return RedeemResult{}, err
	}
	return RedeemResult{User: user, Ticket: ticket, Idempotent: idempotent}, nil
}

func (service *Service) PlaceTicket(ctx context.Context, userID uint64, ticketID string, placement store.TicketPlacement) (domain.Ticket, error) {
	if !validTicketID(ticketID) || !validPlacement(placement) {
		return domain.Ticket{}, ErrInvalidInput
	}
	return service.store.UpdateTicketPlacement(ctx, userID, ticketID, placement)
}

func (service *Service) DiscardTicket(ctx context.Context, userID uint64, ticketID string) (domain.Ticket, error) {
	if !validTicketID(ticketID) {
		return domain.Ticket{}, ErrInvalidInput
	}
	return service.store.DiscardTicket(ctx, userID, ticketID, service.now().UTC())
}

func (service *Service) RestoreDiscardedTicket(ctx context.Context, userID uint64, ticketID string) (domain.Ticket, error) {
	if !validTicketID(ticketID) {
		return domain.Ticket{}, ErrInvalidInput
	}
	return service.store.RestoreDiscardedTicket(ctx, userID, ticketID, service.now().UTC())
}

func (service *Service) createSession(ctx context.Context, user domain.User) (AuthResult, error) {
	now := service.now().UTC()
	token, session, err := auth.NewSession(user.ID, now)
	if err != nil {
		return AuthResult{}, err
	}
	if err := service.store.CreateSession(ctx, session); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, Token: token, ExpiresAt: session.ExpiresAt}, nil
}

func validUsername(username string) bool {
	return utf8.ValidString(username) && usernamePattern.MatchString(username)
}

func validPassword(password string) bool {
	length := len([]byte(password))
	return utf8.ValidString(password) && length >= 8 && length <= 72
}

func validIdempotencyKey(key string) bool {
	if len(key) < 8 || len(key) > 64 {
		return false
	}
	for _, character := range key {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			strings.ContainsRune("-_:", character) {
			continue
		}
		return false
	}
	return true
}

func validTicketID(ticketID string) bool {
	if len(ticketID) != 32 {
		return false
	}
	_, err := hex.DecodeString(ticketID)
	return err == nil
}

func validPlacement(placement store.TicketPlacement) bool {
	if placement.Location != domain.TicketOnDesk && placement.Location != domain.TicketInSlot {
		return false
	}
	if math.IsNaN(placement.DeskX) || math.IsInf(placement.DeskX, 0) || placement.DeskX < 0 || placement.DeskX > 1 ||
		math.IsNaN(placement.DeskY) || math.IsInf(placement.DeskY, 0) || placement.DeskY < 0 || placement.DeskY > 1 ||
		math.IsNaN(placement.Rotation) || math.IsInf(placement.Rotation, 0) || placement.Rotation < -15 || placement.Rotation > 15 ||
		placement.ZIndex < 1 || placement.ZIndex > 1_000_000_000 {
		return false
	}
	if placement.Location == domain.TicketInSlot {
		return placement.SlotIndex != nil && *placement.SlotIndex >= 1 && *placement.SlotIndex <= 10
	}
	return placement.SlotIndex == nil
}

func randomID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

var shanghaiLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func shanghaiDate(value time.Time) string {
	return value.In(shanghaiLocation).Format("2006-01-02")
}
