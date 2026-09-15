package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	"github.com/1424772/guaguale/apps/api/internal/store"
	"github.com/1424772/guaguale/apps/api/internal/store/memory"
)

func TestAccountAndTicketLifecycle(t *testing.T) {
	ctx := context.Background()
	memoryStore := memory.New()
	service := New(memoryStore)

	authResult, err := service.Register(ctx, "测试玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	if authResult.User.Balance != InitialBalance {
		t.Fatalf("expected initial balance %d, got %d", InitialBalance, authResult.User.Balance)
	}

	service.drawCard = func(string, uint8) (domain.Outcome, error) {
		return domain.Outcome{PrizeTier: "first", Reward: 100, Symbols: []string{"碎钻石", "钞票", "碎钻石"}}, nil
	}
	purchase, err := service.Purchase(ctx, authResult.User, "lingqian-ticket", "purchase-test-001")
	if err != nil {
		t.Fatal(err)
	}
	if purchase.User.Balance != 950 {
		t.Fatalf("expected balance 950, got %d", purchase.User.Balance)
	}
	if len(purchase.Ticket.Symbols) != 0 {
		t.Fatal("purchase response exposed hidden symbols")
	}

	retry, err := service.Purchase(ctx, purchase.User, "lingqian-ticket", "purchase-test-001")
	if err != nil {
		t.Fatal(err)
	}
	if !retry.Idempotent || retry.User.Balance != 950 || retry.Ticket.ID != purchase.Ticket.ID {
		t.Fatalf("purchase retry was not idempotent: %#v", retry)
	}

	scratched, err := service.Scratch(ctx, authResult.User.ID, purchase.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(scratched.Symbols) != 3 {
		t.Fatalf("expected revealed symbols, got %#v", scratched.Symbols)
	}

	result, err := service.Redeem(ctx, authResult.User.ID, purchase.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Balance != 950+scratched.Reward {
		t.Fatalf("unexpected redeemed balance %d", result.User.Balance)
	}

	retryRedeem, err := service.Redeem(ctx, authResult.User.ID, purchase.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !retryRedeem.Idempotent || retryRedeem.User.Balance != result.User.Balance {
		t.Fatal("redeem retry was not idempotent")
	}
}

func TestDailyRecoveryLoop(t *testing.T) {
	ctx := context.Background()
	service := New(memory.New())
	clock := time.Date(2026, 9, 15, 0, 30, 0, 0, time.UTC)
	service.now = func() time.Time { return clock }
	service.drawCard = func(string, uint8) (domain.Outcome, error) {
		return domain.Outcome{PrizeTier: "third", Reward: 20, Symbols: []string{"狗头金币", "钞票", "狗头金币"}}, nil
	}
	authResult, err := service.Register(ctx, "每日玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}

	status, err := service.DailyStatus(ctx, authResult.User)
	if err != nil {
		t.Fatal(err)
	}
	if status.Date != "2026-09-15" || len(status.WheelPool) != 1 || status.WheelPool[0].BasisPoint != 10000 {
		t.Fatalf("unexpected initial daily status: %#v", status)
	}

	claim, err := service.ClaimDailyLogin(ctx, authResult.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if claim.User.Balance != 1100 || claim.Idempotent || !claim.Daily.LoginClaimed {
		t.Fatalf("unexpected login claim: %#v", claim)
	}
	retryClaim, err := service.ClaimDailyLogin(ctx, authResult.User.ID)
	if err != nil || !retryClaim.Idempotent || retryClaim.User.Balance != 1100 {
		t.Fatalf("login claim retry was not idempotent: %#v, %v", retryClaim, err)
	}

	started, idempotent, err := service.StartPlate(ctx, claim.User)
	if err != nil || idempotent || started.ActivePlate == nil {
		t.Fatalf("unexpected plate start: %#v, %v", started, err)
	}
	if len(started.WheelPool) == 0 {
		t.Fatal("start plate response omitted eligible wheel pool")
	}
	retryStart, idempotent, err := service.StartPlate(ctx, claim.User)
	if err != nil || !idempotent || retryStart.ActivePlate.ID != started.ActivePlate.ID {
		t.Fatalf("plate start retry was not idempotent: %#v, %v", retryStart, err)
	}
	if _, err := service.CompletePlate(ctx, authResult.User.ID, started.ActivePlate.ID); !errors.Is(err, store.ErrTooEarly) {
		t.Fatalf("expected too-early error, got %v", err)
	}
	clock = clock.Add(PlateDuration)
	completed, err := service.CompletePlate(ctx, authResult.User.ID, started.ActivePlate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.User.Balance != 1105 || completed.Daily.PlatesCompleted != 1 {
		t.Fatalf("unexpected plate completion: %#v", completed)
	}

	wheel, err := service.SpinDailyWheel(ctx, authResult.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if wheel.Ticket.CardCode != "lingqian-ticket" || wheel.Ticket.Source != "daily_wheel" || wheel.Ticket.PricePaid != 0 || !wheel.Daily.WheelUsed {
		t.Fatalf("unexpected wheel result: %#v", wheel)
	}
	retryWheel, err := service.SpinDailyWheel(ctx, authResult.User.ID)
	if err != nil || !retryWheel.Idempotent || retryWheel.Ticket.ID != wheel.Ticket.ID {
		t.Fatalf("wheel retry was not idempotent: %#v, %v", retryWheel, err)
	}
}

func TestRegistrationValidation(t *testing.T) {
	service := New(memory.New())
	if _, err := service.Register(context.Background(), "x", "short", true); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if _, err := service.Register(context.Background(), "测试玩家", "correct-horse-42", false); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected age confirmation to be required, got %v", err)
	}
}
