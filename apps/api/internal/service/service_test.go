package service

import (
	"context"
	"errors"
	"testing"

	"github.com/1424772/guaguale/apps/api/internal/domain"
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

	service.drawLingqian = func(uint8) (domain.Outcome, error) {
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

func TestRegistrationValidation(t *testing.T) {
	service := New(memory.New())
	if _, err := service.Register(context.Background(), "x", "short", true); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if _, err := service.Register(context.Background(), "测试玩家", "correct-horse-42", false); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected age confirmation to be required, got %v", err)
	}
}
