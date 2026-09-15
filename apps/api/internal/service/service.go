package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrCardUnavailable    = errors.New("card is not available")
	usernamePattern       = regexp.MustCompile(`^[\p{Han}A-Za-z0-9_]{3,32}$`)
)

type Service struct {
	store        store.Store
	now          func() time.Time
	drawLingqian func(uint8) (domain.Outcome, error)
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

func New(store store.Store) *Service {
	return &Service{store: store, now: time.Now, drawLingqian: game.DrawLingqian}
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

func (service *Service) Purchase(ctx context.Context, user domain.User, cardCode, idempotencyKey string) (PurchaseResult, error) {
	if !validIdempotencyKey(idempotencyKey) {
		return PurchaseResult{}, ErrInvalidInput
	}
	card, exists := game.CardByCode(cardCode)
	if !exists || !card.Implemented {
		return PurchaseResult{}, ErrCardUnavailable
	}
	outcome, err := service.drawLingqian(user.LuckLevel)
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

func (service *Service) Tickets(ctx context.Context, userID uint64) ([]domain.Ticket, error) {
	return service.store.ListTickets(ctx, userID)
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

func randomID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
