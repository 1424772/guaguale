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
	slot := 1
	if _, err := service.PlaceTicket(ctx, authResult.User.ID, purchase.Ticket.ID, store.TicketPlacement{
		Location: domain.TicketInSlot, DeskX: .5, DeskY: .4, ZIndex: 2, SlotIndex: &slot,
	}); err != nil {
		t.Fatal(err)
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
	if result.Ticket.SlotIndex != nil {
		t.Fatal("redeeming a protected ticket did not release its slot")
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

func TestDeskPlacementSlotsAndDiscard(t *testing.T) {
	ctx := context.Background()
	service := New(memory.New())
	service.drawCard = func(string, uint8) (domain.Outcome, error) {
		return domain.Outcome{PrizeTier: "none", Symbols: []string{"狗头金币", "钞票", "碎钻石"}}, nil
	}
	authResult, err := service.Register(ctx, "桌面玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Purchase(ctx, authResult.User, "lingqian-ticket", "desk-purchase-001")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Purchase(ctx, first.User, "lingqian-ticket", "desk-purchase-002")
	if err != nil {
		t.Fatal(err)
	}
	slot := 1
	placed, err := service.PlaceTicket(ctx, authResult.User.ID, first.Ticket.ID, store.TicketPlacement{
		Location: domain.TicketInSlot, DeskX: .3, DeskY: .4, Rotation: -2, ZIndex: 4, SlotIndex: &slot,
	})
	if err != nil || placed.SlotIndex == nil || *placed.SlotIndex != 1 {
		t.Fatalf("place in slot = %#v, %v", placed, err)
	}
	if _, err := service.PlaceTicket(ctx, authResult.User.ID, second.Ticket.ID, store.TicketPlacement{
		Location: domain.TicketInSlot, DeskX: .5, DeskY: .5, ZIndex: 5, SlotIndex: &slot,
	}); !errors.Is(err, store.ErrSlotOccupied) {
		t.Fatalf("expected occupied slot error, got %v", err)
	}
	if _, err := service.DiscardTicket(ctx, authResult.User.ID, first.Ticket.ID); !errors.Is(err, store.ErrProtected) {
		t.Fatalf("expected protected ticket error, got %v", err)
	}
	placed, err = service.PlaceTicket(ctx, authResult.User.ID, first.Ticket.ID, store.TicketPlacement{
		Location: domain.TicketOnDesk, DeskX: .72, DeskY: .28, Rotation: 4, ZIndex: 8,
	})
	if err != nil || placed.Location != domain.TicketOnDesk || placed.SlotIndex != nil {
		t.Fatalf("return to desk = %#v, %v", placed, err)
	}
	if _, err := service.DiscardTicket(ctx, authResult.User.ID, first.Ticket.ID); err != nil {
		t.Fatal(err)
	}
	tickets, err := service.Tickets(ctx, authResult.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 || tickets[0].ID != second.Ticket.ID {
		t.Fatalf("discarded ticket remained visible: %#v", tickets)
	}
}

func TestPermanentItemUpgrades(t *testing.T) {
	ctx := context.Background()
	service := New(memory.New())
	authResult, err := service.Register(ctx, "道具玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	if authResult.User.ScratchLevel != 1 || authResult.User.LuckLevel != 0 {
		t.Fatalf("unexpected default item levels: %#v", authResult.User)
	}
	shop := service.Shop(authResult.User)
	if len(shop.Items) != 2 || shop.Items[0].NextPrice != 300 || shop.Items[1].NextPrice != 100 {
		t.Fatalf("unexpected initial shop: %#v", shop)
	}

	luck, err := service.UpgradeItem(ctx, authResult.User, "luck", "upgrade-luck-001")
	if err != nil {
		t.Fatal(err)
	}
	if luck.User.Balance != 700 || luck.User.LuckLevel != 1 || luck.Idempotent {
		t.Fatalf("unexpected luck upgrade: %#v", luck)
	}
	retry, err := service.UpgradeItem(ctx, luck.User, "luck", "upgrade-luck-001")
	if err != nil || !retry.Idempotent || retry.User.Balance != 700 || retry.User.LuckLevel != 1 {
		t.Fatalf("luck retry was not idempotent: %#v, %v", retry, err)
	}

	scratchUser, err := service.Register(ctx, "刮片玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	scratch, err := service.UpgradeItem(ctx, scratchUser.User, "scratch-range", "upgrade-range-001")
	if err != nil {
		t.Fatal(err)
	}
	if scratch.User.Balance != 900 || scratch.User.ScratchLevel != 2 {
		t.Fatalf("unexpected scratch range upgrade: %#v", scratch)
	}
	var usedLuck uint8
	service.drawCard = func(_ string, level uint8) (domain.Outcome, error) {
		usedLuck = level
		return domain.Outcome{PrizeTier: "none", Symbols: []string{"狗头金币", "钞票", "碎钻石"}}, nil
	}
	if _, err := service.Purchase(ctx, luck.User, "lingqian-ticket", "luck-purchase-001"); err != nil {
		t.Fatal(err)
	}
	if usedLuck != 1 {
		t.Fatalf("purchase used luck level %d, want 1", usedLuck)
	}
}
