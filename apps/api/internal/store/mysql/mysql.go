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
		ID:             uint64(userID),
		Username:       username,
		Balance:        initialBalance,
		LuckLevel:      0,
		ScratchLevel:   1,
		TrashOwned:     false,
		CardSlotsOwned: false,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

func (store *Store) UserByUsername(ctx context.Context, username string) (domain.User, string, error) {
	var user domain.User
	var passwordHash string
	err := store.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, balance, luck_level, scratch_level, trash_owned, card_slots_owned, created_at
		FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &passwordHash, &user.Balance, &user.LuckLevel, &user.ScratchLevel, &user.TrashOwned, &user.CardSlotsOwned, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, "", basestore.ErrNotFound
	}
	return user, passwordHash, err
}

func (store *Store) UserBySession(ctx context.Context, tokenHash [32]byte) (domain.User, error) {
	var user domain.User
	err := store.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.balance, u.luck_level, u.scratch_level, u.trash_owned, u.card_slots_owned, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > UTC_TIMESTAMP(6)`, tokenHash[:],
	).Scan(&user.ID, &user.Username, &user.Balance, &user.LuckLevel, &user.ScratchLevel, &user.TrashOwned, &user.CardSlotsOwned, &user.CreatedAt)
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
			(id, user_id, card_code, card_name, source, purchase_key, price, price_paid, luck_level, prize_tier, reward, symbols)
		VALUES (?, ?, ?, ?, 'purchase', ?, ?, ?, ?, ?, ?, ?)`,
		input.ID, input.UserID, input.CardCode, input.CardName, input.PurchaseKey, input.Price, input.Price,
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
		CreatedAt:   time.Now().UTC(),
	}
	return user, publicTicket(ticket), false, nil
}

func (store *Store) ListTickets(ctx context.Context, userID uint64) ([]domain.Ticket, error) {
	rows, err := store.db.QueryContext(ctx, `
		SELECT id, user_id, card_code, card_name, source, purchase_key, price, price_paid,
			COALESCE(DATE_FORMAT(wheel_date, '%Y-%m-%d'), ''), luck_level,
			prize_tier, reward, symbols, state, location, desk_x, desk_y, rotation, z_index, slot_index,
			created_at, scratched_at, redeemed_at, discarded_at
		FROM tickets
		WHERE user_id = ? AND state IN ('purchased', 'scratched')
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
		`UPDATE tickets SET state = 'redeemed', slot_index = NULL, redeemed_at = ? WHERE id = ?`, now, ticket.ID,
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
	ticket.SlotIndex = nil
	ticket.RedeemedAt = &now
	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.Ticket{}, false, err
	}
	return user, revealTicket(ticket), false, nil
}

func (store *Store) DailyStatus(ctx context.Context, userID uint64, date string) (domain.DailyStatus, error) {
	daily, err := dailyStatusQuery(ctx, store.db, userID, date)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DailyStatus{Date: date, PlateLimit: 5, WheelPool: []domain.WheelPoolItem{}}, nil
	}
	if err != nil {
		return domain.DailyStatus{}, err
	}
	active, err := activePlateQuery(ctx, store.db, userID, date)
	if err == nil {
		daily.ActivePlate = &active
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.DailyStatus{}, err
	}
	return daily, nil
}

func (store *Store) ClaimDailyLogin(ctx context.Context, userID uint64, date string, amount int64) (domain.User, domain.DailyStatus, bool, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	defer tx.Rollback()
	user, err := lockedUser(ctx, tx, userID)
	if err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if err := ensureDailyState(ctx, tx, userID, date); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	daily, err := lockedDailyStatus(ctx, tx, userID, date)
	if err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if daily.LoginClaimed {
		if err := tx.Commit(); err != nil {
			return domain.User{}, domain.DailyStatus{}, false, err
		}
		return user, daily, true, nil
	}
	before := user.Balance
	user.Balance += amount
	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = ? WHERE id = ?`, user.Balance, userID); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE daily_user_state SET login_claimed = 1 WHERE user_id = ? AND local_date = ?`, userID, date); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO coin_ledger
			(user_id, idempotency_key, reason, reference_type, reference_id, delta, balance_before, balance_after)
		VALUES (?, ?, 'daily_login', 'daily', ?, ?, ?, ?)`,
		userID, fmt.Sprintf("daily-login:%d:%s", userID, date), date, amount, before, user.Balance,
	); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	daily.LoginClaimed = true
	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	return user, daily, false, nil
}

func (store *Store) StartPlate(ctx context.Context, userID uint64, date string, action domain.PlateAction) (domain.DailyStatus, bool, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.DailyStatus{}, false, err
	}
	defer tx.Rollback()
	if _, err := lockedUser(ctx, tx, userID); err != nil {
		return domain.DailyStatus{}, false, err
	}
	if err := ensureDailyState(ctx, tx, userID, date); err != nil {
		return domain.DailyStatus{}, false, err
	}
	daily, err := lockedDailyStatus(ctx, tx, userID, date)
	if err != nil {
		return domain.DailyStatus{}, false, err
	}
	if daily.PlatesCompleted >= 5 {
		return domain.DailyStatus{}, false, basestore.ErrDailyLimit
	}
	active, err := activePlateQuery(ctx, tx, userID, date)
	if err == nil {
		daily.ActivePlate = &active
		if err := tx.Commit(); err != nil {
			return domain.DailyStatus{}, false, err
		}
		return daily, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.DailyStatus{}, false, err
	}
	action.Sequence = daily.PlatesCompleted + 1
	_, err = tx.ExecContext(ctx, `
		INSERT INTO daily_plate_actions
			(id, user_id, local_date, sequence_no, state, started_at, available_at)
		VALUES (?, ?, ?, ?, 'started', ?, ?)`,
		action.ID, userID, date, action.Sequence, action.StartedAt.UTC(), action.AvailableAt.UTC(),
	)
	if err != nil {
		return domain.DailyStatus{}, false, err
	}
	daily.ActivePlate = &action
	if err := tx.Commit(); err != nil {
		return domain.DailyStatus{}, false, err
	}
	return daily, false, nil
}

func (store *Store) CompletePlate(ctx context.Context, userID uint64, date, actionID string, completedAt time.Time, amount int64) (domain.User, domain.DailyStatus, bool, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	defer tx.Rollback()
	user, err := lockedUser(ctx, tx, userID)
	if err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if err := ensureDailyState(ctx, tx, userID, date); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	daily, err := lockedDailyStatus(ctx, tx, userID, date)
	if err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	plate, err := lockedPlate(ctx, tx, userID, date, actionID)
	if err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if plate.State == "completed" {
		if err := tx.Commit(); err != nil {
			return domain.User{}, domain.DailyStatus{}, false, err
		}
		return user, daily, true, nil
	}
	if completedAt.Before(plate.AvailableAt) {
		return domain.User{}, domain.DailyStatus{}, false, basestore.ErrTooEarly
	}
	if daily.PlatesCompleted >= 5 {
		return domain.User{}, domain.DailyStatus{}, false, basestore.ErrDailyLimit
	}
	before := user.Balance
	user.Balance += amount
	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = ? WHERE id = ?`, user.Balance, userID); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE daily_plate_actions SET state = 'completed', completed_at = ? WHERE id = ?`, completedAt.UTC(), actionID,
	); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE daily_user_state SET plates_completed = plates_completed + 1 WHERE user_id = ? AND local_date = ?`, userID, date,
	); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO coin_ledger
			(user_id, idempotency_key, reason, reference_type, reference_id, delta, balance_before, balance_after)
		VALUES (?, ?, 'plate_reward', 'plate', ?, ?, ?, ?)`,
		userID, "plate:"+actionID, actionID, amount, before, user.Balance,
	); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	daily.PlatesCompleted++
	daily.ActivePlate = nil
	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.DailyStatus{}, false, err
	}
	return user, daily, false, nil
}

func (store *Store) SpinDailyWheel(ctx context.Context, userID uint64, date, ticketID string, draw basestore.WheelDrawFunc) (domain.User, domain.Ticket, domain.DailyStatus, bool, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	defer tx.Rollback()
	user, err := lockedUser(ctx, tx, userID)
	if err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	if err := ensureDailyState(ctx, tx, userID, date); err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	daily, err := lockedDailyStatus(ctx, tx, userID, date)
	if err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	if daily.WheelUsed {
		ticket, err := lockedTicket(ctx, tx, userID, daily.WheelTicketID)
		if err != nil {
			return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
		}
		if err := tx.Commit(); err != nil {
			return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
		}
		return user, publicTicket(ticket), daily, true, nil
	}
	selection, err := draw(user.Balance, user.LuckLevel)
	if err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	symbolsJSON, err := json.Marshal(selection.Outcome.Symbols)
	if err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	poolJSON, err := json.Marshal(selection.Pool)
	if err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tickets
			(id, user_id, card_code, card_name, source, purchase_key, price, price_paid, wheel_date, luck_level, prize_tier, reward, symbols)
		VALUES (?, ?, ?, ?, 'daily_wheel', ?, ?, 0, ?, ?, ?, ?, ?)`,
		ticketID, userID, selection.CardCode, selection.CardName, "wheel:"+date,
		selection.NominalPrice, date, user.LuckLevel, selection.Outcome.PrizeTier, selection.Outcome.Reward, symbolsJSON,
	)
	if err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE daily_user_state
		SET wheel_used = 1, wheel_ticket_id = ?, wheel_card_code = ?, wheel_card_name = ?, wheel_pool_snapshot = ?
		WHERE user_id = ? AND local_date = ?`,
		ticketID, selection.CardCode, selection.CardName, poolJSON, userID, date,
	)
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
		LuckLevel: user.LuckLevel,
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
	daily.WheelUsed = true
	daily.WheelTicketID = ticketID
	daily.WheelCardCode = selection.CardCode
	daily.WheelCardName = selection.CardName
	daily.WheelPool = append([]domain.WheelPoolItem(nil), selection.Pool...)
	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.Ticket{}, domain.DailyStatus{}, false, err
	}
	return user, publicTicket(ticket), daily, false, nil
}

func (store *Store) DeleteExpiredSessions(ctx context.Context, cutoff time.Time) error {
	_, err := store.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, cutoff.UTC())
	return err
}

func (store *Store) UpgradeItem(ctx context.Context, input basestore.UpgradeItemInput) (domain.User, domain.ItemUpgrade, bool, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.User{}, domain.ItemUpgrade{}, false, err
	}
	defer tx.Rollback()
	user, err := lockedUser(ctx, tx, input.UserID)
	if err != nil {
		return domain.User{}, domain.ItemUpgrade{}, false, err
	}
	if existing, err := itemUpgradeByKey(ctx, tx, input.UserID, input.IdempotencyKey); err == nil {
		if err := tx.Commit(); err != nil {
			return domain.User{}, domain.ItemUpgrade{}, false, err
		}
		return user, existing, true, nil
	} else if !errors.Is(err, basestore.ErrNotFound) {
		return domain.User{}, domain.ItemUpgrade{}, false, err
	}
	currentLevel := user.LuckLevel
	updateQuery := `UPDATE users SET balance = ?, luck_level = ? WHERE id = ?`
	if input.ItemCode == "scratch-range" {
		currentLevel = user.ScratchLevel
		updateQuery = `UPDATE users SET balance = ?, scratch_level = ? WHERE id = ?`
	} else if input.ItemCode == "trash" {
		currentLevel = boolLevel(user.TrashOwned)
		updateQuery = `UPDATE users SET balance = ?, trash_owned = ? WHERE id = ?`
	} else if input.ItemCode == "card-slots" {
		currentLevel = boolLevel(user.CardSlotsOwned)
		updateQuery = `UPDATE users SET balance = ?, card_slots_owned = ? WHERE id = ?`
	} else if input.ItemCode != "luck" {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrInvalidState
	}
	if currentLevel != input.ExpectedFromLevel || input.ToLevel != currentLevel+1 {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrUpgradeConflict
	}
	if currentLevel >= input.MaxLevel || input.Price <= 0 {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrInvalidState
	}
	if user.Balance < input.Price {
		return domain.User{}, domain.ItemUpgrade{}, false, basestore.ErrInsufficientFunds
	}
	before := user.Balance
	user.Balance -= input.Price
	if _, err := tx.ExecContext(ctx, updateQuery, user.Balance, input.ToLevel, user.ID); err != nil {
		return domain.User{}, domain.ItemUpgrade{}, false, err
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO item_upgrades
			(id, user_id, item_code, from_level, to_level, price, idempotency_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.ID, user.ID, input.ItemCode, currentLevel, input.ToLevel, input.Price, input.IdempotencyKey, now,
	)
	if err != nil {
		return domain.User{}, domain.ItemUpgrade{}, false, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO coin_ledger
			(user_id, idempotency_key, reason, reference_type, reference_id, delta, balance_before, balance_after)
		VALUES (?, ?, 'item_upgrade', 'item_upgrade', ?, ?, ?, ?)`,
		user.ID, "upgrade:"+input.ID, input.ID, -input.Price, before, user.Balance,
	)
	if err != nil {
		return domain.User{}, domain.ItemUpgrade{}, false, err
	}
	if input.ItemCode == "luck" {
		user.LuckLevel = input.ToLevel
	} else if input.ItemCode == "scratch-range" {
		user.ScratchLevel = input.ToLevel
	} else if input.ItemCode == "trash" {
		user.TrashOwned = true
	} else {
		user.CardSlotsOwned = true
	}
	upgrade := domain.ItemUpgrade{
		ID: input.ID, UserID: user.ID, ItemCode: input.ItemCode, FromLevel: currentLevel,
		ToLevel: input.ToLevel, Price: input.Price, IdempotencyKey: input.IdempotencyKey, CreatedAt: now,
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.ItemUpgrade{}, false, err
	}
	return user, upgrade, false, nil
}

func (store *Store) UpdateTicketPlacement(ctx context.Context, userID uint64, ticketID string, placement basestore.TicketPlacement) (domain.Ticket, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.Ticket{}, err
	}
	defer tx.Rollback()
	ticket, err := lockedTicket(ctx, tx, userID, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if ticket.State != domain.TicketPurchased && ticket.State != domain.TicketScratched {
		return domain.Ticket{}, basestore.ErrInvalidState
	}
	if placement.Location == domain.TicketInSlot {
		user, err := lockedUser(ctx, tx, userID)
		if err != nil {
			return domain.Ticket{}, err
		}
		if !user.CardSlotsOwned {
			return domain.Ticket{}, basestore.ErrCardSlotsRequired
		}
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE tickets
		SET location = ?, desk_x = ?, desk_y = ?, rotation = ?, z_index = ?, slot_index = ?
		WHERE id = ?`,
		placement.Location, placement.DeskX, placement.DeskY, placement.Rotation, placement.ZIndex, placement.SlotIndex, ticketID,
	)
	if err != nil {
		var mysqlErr *mysqldriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.Ticket{}, basestore.ErrSlotOccupied
		}
		return domain.Ticket{}, err
	}
	ticket.Location = placement.Location
	ticket.DeskX = placement.DeskX
	ticket.DeskY = placement.DeskY
	ticket.Rotation = placement.Rotation
	ticket.ZIndex = placement.ZIndex
	ticket.SlotIndex = placement.SlotIndex
	if err := tx.Commit(); err != nil {
		return domain.Ticket{}, err
	}
	return publicTicket(ticket), nil
}

func (store *Store) DiscardTicket(ctx context.Context, userID uint64, ticketID string, discardedAt time.Time) (domain.Ticket, error) {
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.Ticket{}, err
	}
	defer tx.Rollback()
	ticket, err := lockedTicket(ctx, tx, userID, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if ticket.Location == domain.TicketInSlot {
		return domain.Ticket{}, basestore.ErrProtected
	}
	user, err := lockedUser(ctx, tx, userID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !user.TrashOwned {
		return domain.Ticket{}, basestore.ErrTrashRequired
	}
	if ticket.State != domain.TicketPurchased && ticket.State != domain.TicketScratched {
		return domain.Ticket{}, basestore.ErrInvalidState
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tickets SET state = 'discarded', slot_index = NULL, discarded_at = ? WHERE id = ?`, discardedAt.UTC(), ticketID,
	); err != nil {
		return domain.Ticket{}, err
	}
	ticket.State = domain.TicketDiscarded
	ticket.SlotIndex = nil
	ticket.DiscardedAt = &discardedAt
	if err := tx.Commit(); err != nil {
		return domain.Ticket{}, err
	}
	return revealTicket(ticket), nil
}

type scanner interface {
	Scan(...any) error
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func scanTicket(row scanner) (domain.Ticket, error) {
	var ticket domain.Ticket
	var symbolsJSON []byte
	var state string
	var location string
	var slotIndex sql.NullInt64
	err := row.Scan(
		&ticket.ID, &ticket.UserID, &ticket.CardCode, &ticket.CardName, &ticket.Source, &ticket.PurchaseKey,
		&ticket.Price, &ticket.PricePaid, &ticket.WheelDate, &ticket.LuckLevel, &ticket.PrizeTier, &ticket.Reward, &symbolsJSON,
		&state, &location, &ticket.DeskX, &ticket.DeskY, &ticket.Rotation, &ticket.ZIndex, &slotIndex,
		&ticket.CreatedAt, &ticket.ScratchedAt, &ticket.RedeemedAt, &ticket.DiscardedAt,
	)
	if err != nil {
		return domain.Ticket{}, err
	}
	if err := json.Unmarshal(symbolsJSON, &ticket.Symbols); err != nil {
		return domain.Ticket{}, err
	}
	ticket.State = domain.TicketState(state)
	ticket.Location = domain.TicketLocation(location)
	if slotIndex.Valid {
		value := int(slotIndex.Int64)
		ticket.SlotIndex = &value
	}
	return ticket, nil
}

func lockedUser(ctx context.Context, tx *sql.Tx, userID uint64) (domain.User, error) {
	var user domain.User
	err := tx.QueryRowContext(ctx, `
		SELECT id, username, balance, luck_level, scratch_level, trash_owned, card_slots_owned, created_at
		FROM users WHERE id = ? FOR UPDATE`, userID,
	).Scan(&user.ID, &user.Username, &user.Balance, &user.LuckLevel, &user.ScratchLevel, &user.TrashOwned, &user.CardSlotsOwned, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, basestore.ErrNotFound
	}
	return user, err
}

func boolLevel(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

func itemUpgradeByKey(ctx context.Context, tx *sql.Tx, userID uint64, key string) (domain.ItemUpgrade, error) {
	var upgrade domain.ItemUpgrade
	err := tx.QueryRowContext(ctx, `
		SELECT id, user_id, item_code, from_level, to_level, price, idempotency_key, created_at
		FROM item_upgrades WHERE user_id = ? AND idempotency_key = ?`, userID, key,
	).Scan(
		&upgrade.ID, &upgrade.UserID, &upgrade.ItemCode, &upgrade.FromLevel, &upgrade.ToLevel,
		&upgrade.Price, &upgrade.IdempotencyKey, &upgrade.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ItemUpgrade{}, basestore.ErrNotFound
	}
	return upgrade, err
}

func ticketByPurchaseKey(ctx context.Context, tx *sql.Tx, userID uint64, key string) (domain.Ticket, error) {
	ticket, err := scanTicket(tx.QueryRowContext(ctx, `
		SELECT id, user_id, card_code, card_name, source, purchase_key, price, price_paid,
			COALESCE(DATE_FORMAT(wheel_date, '%Y-%m-%d'), ''), luck_level,
			prize_tier, reward, symbols, state, location, desk_x, desk_y, rotation, z_index, slot_index,
			created_at, scratched_at, redeemed_at, discarded_at
		FROM tickets WHERE user_id = ? AND purchase_key = ?`, userID, key))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Ticket{}, basestore.ErrNotFound
	}
	return ticket, err
}

func lockedTicket(ctx context.Context, tx *sql.Tx, userID uint64, ticketID string) (domain.Ticket, error) {
	ticket, err := scanTicket(tx.QueryRowContext(ctx, `
		SELECT id, user_id, card_code, card_name, source, purchase_key, price, price_paid,
			COALESCE(DATE_FORMAT(wheel_date, '%Y-%m-%d'), ''), luck_level,
			prize_tier, reward, symbols, state, location, desk_x, desk_y, rotation, z_index, slot_index,
			created_at, scratched_at, redeemed_at, discarded_at
		FROM tickets WHERE id = ? AND user_id = ? FOR UPDATE`, ticketID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Ticket{}, basestore.ErrNotFound
	}
	return ticket, err
}

func ensureDailyState(ctx context.Context, tx *sql.Tx, userID uint64, date string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT IGNORE INTO daily_user_state (user_id, local_date) VALUES (?, ?)`, userID, date,
	)
	return err
}

func dailyStatusQuery(ctx context.Context, source queryer, userID uint64, date string) (domain.DailyStatus, error) {
	return scanDailyStatus(source.QueryRowContext(ctx, `
		SELECT DATE_FORMAT(local_date, '%Y-%m-%d'), login_claimed, plates_completed, wheel_used,
			COALESCE(wheel_ticket_id, ''), COALESCE(wheel_card_code, ''), COALESCE(wheel_card_name, ''),
			COALESCE(wheel_pool_snapshot, JSON_ARRAY())
		FROM daily_user_state WHERE user_id = ? AND local_date = ?`, userID, date))
}

func lockedDailyStatus(ctx context.Context, tx *sql.Tx, userID uint64, date string) (domain.DailyStatus, error) {
	return scanDailyStatus(tx.QueryRowContext(ctx, `
		SELECT DATE_FORMAT(local_date, '%Y-%m-%d'), login_claimed, plates_completed, wheel_used,
			COALESCE(wheel_ticket_id, ''), COALESCE(wheel_card_code, ''), COALESCE(wheel_card_name, ''),
			COALESCE(wheel_pool_snapshot, JSON_ARRAY())
		FROM daily_user_state WHERE user_id = ? AND local_date = ? FOR UPDATE`, userID, date))
}

func scanDailyStatus(row scanner) (domain.DailyStatus, error) {
	var daily domain.DailyStatus
	var poolJSON []byte
	if err := row.Scan(
		&daily.Date, &daily.LoginClaimed, &daily.PlatesCompleted, &daily.WheelUsed,
		&daily.WheelTicketID, &daily.WheelCardCode, &daily.WheelCardName, &poolJSON,
	); err != nil {
		return domain.DailyStatus{}, err
	}
	daily.PlateLimit = 5
	if err := json.Unmarshal(poolJSON, &daily.WheelPool); err != nil {
		return domain.DailyStatus{}, err
	}
	if daily.WheelPool == nil {
		daily.WheelPool = []domain.WheelPoolItem{}
	}
	return daily, nil
}

func activePlateQuery(ctx context.Context, source queryer, userID uint64, date string) (domain.PlateAction, error) {
	return scanPlate(source.QueryRowContext(ctx, `
		SELECT id, sequence_no, state, started_at, available_at, completed_at
		FROM daily_plate_actions
		WHERE user_id = ? AND local_date = ? AND state = 'started'
		ORDER BY sequence_no LIMIT 1`, userID, date))
}

func lockedPlate(ctx context.Context, tx *sql.Tx, userID uint64, date, actionID string) (domain.PlateAction, error) {
	plate, err := scanPlate(tx.QueryRowContext(ctx, `
		SELECT id, sequence_no, state, started_at, available_at, completed_at
		FROM daily_plate_actions
		WHERE id = ? AND user_id = ? AND local_date = ? FOR UPDATE`, actionID, userID, date))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PlateAction{}, basestore.ErrNotFound
	}
	return plate, err
}

func scanPlate(row scanner) (domain.PlateAction, error) {
	var plate domain.PlateAction
	err := row.Scan(&plate.ID, &plate.Sequence, &plate.State, &plate.StartedAt, &plate.AvailableAt, &plate.CompletedAt)
	return plate, err
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
