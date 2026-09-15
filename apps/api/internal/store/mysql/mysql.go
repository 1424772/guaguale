package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store { return &Store{db: db} }

func (store *Store) Ping(ctx context.Context) error {
	return store.db.PingContext(ctx)
}

func (store *Store) CreateUser(ctx context.Context, username, passwordHash string, initialBalance int64) (domain.User, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, balance) VALUES (?, ?, ?)`,
		username, passwordHash, initialBalance,
	)
	if err != nil {
		var mysqlErr *mysqldriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.User{}, basestore.ErrUsernameTaken
		}
		return domain.User{}, err
	}
	userID, err := result.LastInsertId()
	if err != nil {
		return domain.User{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO coin_ledger
			(user_id, idempotency_key, reason, reference_type, reference_id, delta, balance_before, balance_after)
		VALUES (?, ?, 'initial_grant', 'user', ?, ?, 0, ?)`,
		userID, fmt.Sprintf("register:%d", userID), userID, initialBalance, initialBalance,
	)
	if err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, err
	}
	return domain.User{
		ID:        uint64(userID),
		Username:  username,
		Balance:   initialBalance,
		LuckLevel: 0,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (store *Store) UserByUsername(ctx context.Context, username string) (domain.User, string, error) {
	var user domain.User
	var passwordHash string
	err := store.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, balance, luck_level, created_at
		FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &passwordHash, &user.Balance, &user.LuckLevel, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, "", basestore.ErrNotFound
	}
	return user, passwordHash, err
}

func (store *Store) UserBySession(ctx context.Context, tokenHash [32]byte) (domain.User, error) {
	var user domain.User
	err := store.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.balance, u.luck_level, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > UTC_TIMESTAMP(6)`, tokenHash[:],
	).Scan(&user.ID, &user.Username, &user.Balance, &user.LuckLevel, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, basestore.ErrNotFound
	}
	return user, err
}

func (store *Store) CreateSession(ctx context.Context, session domain.Session) error {
	_, err := store.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		session.TokenHash[:], session.UserID, session.ExpiresAt.UTC(),
	)
	return err
}

func (store *Store) DeleteSession(ctx context.Context, tokenHash [32]byte) error {
	_, err := store.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash[:])
	return err
}

func (store *Store) PurchaseTicket(ctx context.Context, input basestore.CreateTicketInput) (domain.User, domain.Ticket, bool, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	defer tx.Rollback()

	user, err := lockedUser(ctx, tx, input.UserID)
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	if existing, err := ticketByPurchaseKey(ctx, tx, input.UserID, input.PurchaseKey); err == nil {
		if err := tx.Commit(); err != nil {
			return domain.User{}, domain.Ticket{}, false, err
		}
		return user, publicTicket(existing), true, nil
	} else if !errors.Is(err, basestore.ErrNotFound) {
		return domain.User{}, domain.Ticket{}, false, err
	}
	if user.Balance < input.Price {
		return domain.User{}, domain.Ticket{}, false, basestore.ErrInsufficientFunds
	}

	symbolsJSON, err := json.Marshal(input.Outcome.Symbols)
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	before := user.Balance
	user.Balance -= input.Price
	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = ? WHERE id = ?`, user.Balance, user.ID); err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tickets
			(id, user_id, card_code, card_name, purchase_key, price, luck_level, prize_tier, reward, symbols)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.ID, input.UserID, input.CardCode, input.CardName, input.PurchaseKey, input.Price,
		input.LuckLevel, input.Outcome.PrizeTier, input.Outcome.Reward, symbolsJSON,
	)
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO coin_ledger
			(user_id, idempotency_key, reason, reference_type, reference_id, delta, balance_before, balance_after)
		VALUES (?, ?, 'card_purchase', 'ticket', ?, ?, ?, ?)`,
		input.UserID, "purchase:"+input.ID, input.ID, -input.Price, before, user.Balance,
	)
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
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
		CreatedAt:   time.Now().UTC(),
	}
	return user, publicTicket(ticket), false, nil
}

func (store *Store) ListTickets(ctx context.Context, userID uint64) ([]domain.Ticket, error) {
	rows, err := store.db.QueryContext(ctx, `
		SELECT id, user_id, card_code, card_name, purchase_key, price, luck_level,
			prize_tier, reward, symbols, state, created_at, scratched_at, redeemed_at
		FROM tickets
		WHERE user_id = ? AND state <> 'discarded'
		ORDER BY created_at DESC
		LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Ticket, 0)
	for rows.Next() {
		ticket, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, publicTicket(ticket))
	}
	return result, rows.Err()
}

func (store *Store) ScratchTicket(ctx context.Context, userID uint64, ticketID string) (domain.Ticket, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.Ticket{}, err
	}
	defer tx.Rollback()
	ticket, err := lockedTicket(ctx, tx, userID, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if ticket.State == domain.TicketPurchased {
		now := time.Now().UTC()
		if _, err := tx.ExecContext(ctx,
			`UPDATE tickets SET state = 'scratched', scratched_at = ? WHERE id = ?`, now, ticket.ID,
		); err != nil {
			return domain.Ticket{}, err
		}
		ticket.State = domain.TicketScratched
		ticket.ScratchedAt = &now
	}
	if ticket.State != domain.TicketScratched && ticket.State != domain.TicketRedeemed {
		return domain.Ticket{}, basestore.ErrInvalidState
	}
	if err := tx.Commit(); err != nil {
		return domain.Ticket{}, err
	}
	return revealTicket(ticket), nil
}

func (store *Store) RedeemTicket(ctx context.Context, userID uint64, ticketID string) (domain.User, domain.Ticket, bool, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	defer tx.Rollback()
	ticket, err := lockedTicket(ctx, tx, userID, ticketID)
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	user, err := lockedUser(ctx, tx, userID)
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	if ticket.State == domain.TicketRedeemed {
		if err := tx.Commit(); err != nil {
			return domain.User{}, domain.Ticket{}, false, err
		}
		return user, revealTicket(ticket), true, nil
	}
	if ticket.State != domain.TicketScratched {
		return domain.User{}, domain.Ticket{}, false, basestore.ErrInvalidState
	}
	if ticket.Reward <= 0 {
		return domain.User{}, domain.Ticket{}, false, basestore.ErrNotWinner
	}

	before := user.Balance
	user.Balance += ticket.Reward
	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = ? WHERE id = ?`, user.Balance, user.ID); err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx,
		`UPDATE tickets SET state = 'redeemed', redeemed_at = ? WHERE id = ?`, now, ticket.ID,
	); err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO coin_ledger
			(user_id, idempotency_key, reason, reference_type, reference_id, delta, balance_before, balance_after)
		VALUES (?, ?, 'ticket_redeem', 'ticket', ?, ?, ?, ?)`,
		userID, "redeem:"+ticket.ID, ticket.ID, ticket.Reward, before, user.Balance,
	)
	if err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	ticket.State = domain.TicketRedeemed
	ticket.RedeemedAt = &now
	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	return user, revealTicket(ticket), false, nil
}

func (store *Store) DeleteExpiredSessions(ctx context.Context, cutoff time.Time) error {
	_, err := store.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, cutoff.UTC())
	return err
}

type scanner interface {
	Scan(...any) error
}

func scanTicket(row scanner) (domain.Ticket, error) {
	var ticket domain.Ticket
	var symbolsJSON []byte
	var state string
	err := row.Scan(
		&ticket.ID, &ticket.UserID, &ticket.CardCode, &ticket.CardName, &ticket.PurchaseKey,
		&ticket.Price, &ticket.LuckLevel, &ticket.PrizeTier, &ticket.Reward, &symbolsJSON,
		&state, &ticket.CreatedAt, &ticket.ScratchedAt, &ticket.RedeemedAt,
	)
	if err != nil {
		return domain.Ticket{}, err
	}
	if err := json.Unmarshal(symbolsJSON, &ticket.Symbols); err != nil {
		return domain.Ticket{}, err
	}
	ticket.State = domain.TicketState(state)
	return ticket, nil
}

func lockedUser(ctx context.Context, tx *sql.Tx, userID uint64) (domain.User, error) {
	var user domain.User
	err := tx.QueryRowContext(ctx, `
		SELECT id, username, balance, luck_level, created_at
		FROM users WHERE id = ? FOR UPDATE`, userID,
	).Scan(&user.ID, &user.Username, &user.Balance, &user.LuckLevel, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, basestore.ErrNotFound
	}
	return user, err
}

func ticketByPurchaseKey(ctx context.Context, tx *sql.Tx, userID uint64, key string) (domain.Ticket, error) {
	ticket, err := scanTicket(tx.QueryRowContext(ctx, `
		SELECT id, user_id, card_code, card_name, purchase_key, price, luck_level,
			prize_tier, reward, symbols, state, created_at, scratched_at, redeemed_at
		FROM tickets WHERE user_id = ? AND purchase_key = ?`, userID, key))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Ticket{}, basestore.ErrNotFound
	}
	return ticket, err
}

func lockedTicket(ctx context.Context, tx *sql.Tx, userID uint64, ticketID string) (domain.Ticket, error) {
	ticket, err := scanTicket(tx.QueryRowContext(ctx, `
		SELECT id, user_id, card_code, card_name, purchase_key, price, luck_level,
			prize_tier, reward, symbols, state, created_at, scratched_at, redeemed_at
		FROM tickets WHERE id = ? AND user_id = ? FOR UPDATE`, ticketID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Ticket{}, basestore.ErrNotFound
	}
	return ticket, err
}

func publicTicket(ticket domain.Ticket) domain.Ticket {
	if ticket.State == domain.TicketPurchased {
		ticket.PrizeTier = ""
		ticket.Reward = 0
		ticket.Symbols = nil
	}
	return ticket
}

func revealTicket(ticket domain.Ticket) domain.Ticket { return ticket }
