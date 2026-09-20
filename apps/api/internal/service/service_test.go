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
	slots, err := service.UpgradeItem(ctx, authResult.User, "card-slots", "lifecycle-slots-001")
	if err != nil {
		t.Fatal(err)
	}
	purchase, err := service.Purchase(ctx, slots.User, "lingqian-ticket", "purchase-test-001")
	if err != nil {
		t.Fatal(err)
	}
	if purchase.User.Balance != 450 {
		t.Fatalf("expected balance 450, got %d", purchase.User.Balance)
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
	if !retry.Idempotent || retry.User.Balance != 450 || retry.Ticket.ID != purchase.Ticket.ID {
		t.Fatalf("purchase retry was not idempotent: %#v", retry)
	}

	revealed, err := service.Reveal(ctx, authResult.User.ID, purchase.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if revealed.State != domain.TicketPurchased || len(revealed.Symbols) != 3 {
		t.Fatalf("preview should reveal symbols without scratching ticket: %#v", revealed)
	}
	listed, err := service.Tickets(ctx, authResult.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].State != domain.TicketPurchased || len(listed[0].Symbols) != 0 {
		t.Fatalf("preview changed public ticket state or leaked symbols: %#v", listed)
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
	if result.User.Balance != 450+scratched.Reward {
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

func TestLeaderboardAndGameHistory(t *testing.T) {
	ctx := context.Background()
	service := New(memory.New())
	first, err := service.Register(ctx, "排行甲玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Register(ctx, "排行乙玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	third, err := service.Register(ctx, "排行丙玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	service.drawCard = func(string, uint8) (domain.Outcome, error) {
		return domain.Outcome{PrizeTier: "first", Reward: 100, Symbols: []string{"碎钻石", "钞票", "碎钻石"}}, nil
	}
	purchase, err := service.Purchase(ctx, first.User, "lingqian-ticket", "ranking-history-card")
	if err != nil {
		t.Fatal(err)
	}
	leaderboard, err := service.Leaderboard(ctx, purchase.User)
	if err != nil {
		t.Fatal(err)
	}
	if leaderboard.TotalUsers != 3 || leaderboard.CurrentUser.Rank != 3 || leaderboard.CurrentUser.Username != "排***家" {
		t.Fatalf("unexpected leaderboard: %#v", leaderboard)
	}
	if leaderboard.Entries[0].UserID != second.User.ID || leaderboard.Entries[1].UserID != third.User.ID {
		t.Fatalf("tie-break order is unstable: %#v", leaderboard.Entries)
	}

	if _, err := service.Scratch(ctx, first.User.ID, purchase.Ticket.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Redeem(ctx, first.User.ID, purchase.Ticket.ID); err != nil {
		t.Fatal(err)
	}
	events, err := service.GameHistory(ctx, first.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].Type != "redeem" || events[1].Type != "scratch" || events[2].Type != "purchase" {
		t.Fatalf("unexpected lifecycle history: %#v", events)
	}
	if events[0].Delta != 100 || events[2].Delta != -50 {
		t.Fatalf("unexpected history deltas: %#v", events)
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
	if _, err := service.PlaceTicket(ctx, authResult.User.ID, first.Ticket.ID, store.TicketPlacement{
		Location: domain.TicketInSlot, DeskX: .3, DeskY: .4, Rotation: -2, ZIndex: 4, SlotIndex: &slot,
	}); !errors.Is(err, store.ErrCardSlotsRequired) {
		t.Fatalf("expected card slots purchase requirement, got %v", err)
	}
	if _, err := service.DiscardTicket(ctx, authResult.User.ID, first.Ticket.ID); !errors.Is(err, store.ErrTrashRequired) {
		t.Fatalf("expected trash purchase requirement, got %v", err)
	}
	trash, err := service.UpgradeItem(ctx, second.User, "trash", "buy-trash-001")
	if err != nil {
		t.Fatal(err)
	}
	if !trash.User.TrashOwned {
		t.Fatalf("trash purchase did not update user: %#v", trash.User)
	}
	slots, err := service.UpgradeItem(ctx, trash.User, "card-slots", "buy-slots-001")
	if err != nil {
		t.Fatal(err)
	}
	if !slots.User.CardSlotsOwned {
		t.Fatalf("card slots purchase did not update user: %#v", slots.User)
	}
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
	if _, err := service.Scratch(ctx, authResult.User.ID, first.Ticket.ID); err != nil {
		t.Fatalf("scratch before discard: %v", err)
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
	restored, err := service.RestoreDiscardedTicket(ctx, authResult.User.ID, first.Ticket.ID)
	if err != nil || restored.State != domain.TicketScratched {
		t.Fatalf("restore discarded ticket = %#v, %v", restored, err)
	}
	tickets, err = service.Tickets(ctx, authResult.User.ID)
	if err != nil || len(tickets) != 2 {
		t.Fatalf("restored ticket did not return to active list: %#v, %v", tickets, err)
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
	if len(shop.Items) != 9 || shop.Items[0].NextPrice != 300 || shop.Items[1].NextPrice != 100 || shop.Items[2].NextPrice != 100 || shop.Items[3].NextPrice != 500 || shop.Items[5].NextPrice != 1000 {
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

	trash, err := service.UpgradeItem(ctx, luck.User, "trash", "upgrade-trash-001")
	if err != nil || !trash.User.TrashOwned || trash.User.Balance != 550 {
		t.Fatalf("unexpected trash purchase: %#v, %v", trash, err)
	}
	slots, err := service.UpgradeItem(ctx, trash.User, "card-slots", "upgrade-slots-001")
	if err != nil || !slots.User.CardSlotsOwned || slots.User.Balance != 50 {
		t.Fatalf("unexpected card slots purchase: %#v, %v", slots, err)
	}
}

func TestRobotQueuePausesOfflineAndProcessesInOrder(t *testing.T) {
	ctx := context.Background()
	service := New(memory.New())
	clock := time.Date(2026, 9, 16, 2, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return clock }
	user, err := service.Register(ctx, "机器人玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	daily, err := service.ClaimDailyLogin(ctx, user.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	robot, err := service.UpgradeItem(ctx, daily.User, "robot", "buy-robot-001")
	if err != nil {
		t.Fatal(err)
	}
	if !robot.User.RobotOwned || robot.User.RobotSpeedLevel != 1 || robot.User.RobotQueueLevel != 1 || robot.User.RobotInterceptLevel != 1 || robot.User.Balance != 100 {
		t.Fatalf("unexpected robot purchase: %#v", robot.User)
	}
	robotShop := service.Shop(robot.User)
	if robotShop.Items[6].Locked || robotShop.Items[6].Level != 1 || robotShop.Items[6].NextPrice != 500 || robotShop.Items[7].NextPrice != 300 || robotShop.Items[8].NextPrice != 600 {
		t.Fatalf("unexpected unlocked robot modules: %#v", robotShop.Items[6:])
	}
	draws := 0
	service.drawCard = func(string, uint8) (domain.Outcome, error) {
		draws++
		if draws == 1 {
			return domain.Outcome{PrizeTier: "first", Reward: 100, Symbols: []string{"碎钻石", "钞票", "碎钻石"}}, nil
		}
		return domain.Outcome{PrizeTier: "none", Symbols: []string{"狗头金币", "钞票", "碎钻石"}}, nil
	}
	first, err := service.Purchase(ctx, robot.User, "lingqian-ticket", "robot-card-001")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Purchase(ctx, first.User, "lingqian-ticket", "robot-card-002")
	if err != nil {
		t.Fatal(err)
	}
	queued, err := service.EnqueueRobot(ctx, second.User, first.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	queued, err = service.EnqueueRobot(ctx, second.User, second.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued.Robot.Queue) != 2 || queued.Robot.Capacity != 3 || queued.Robot.DurationSeconds != 40 {
		t.Fatalf("unexpected robot queue: %#v", queued.Robot)
	}
	initial, err := service.TickRobot(ctx, second.User)
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(time.Minute)
	paused, err := service.TickRobot(ctx, second.User)
	if err != nil {
		t.Fatal(err)
	}
	if paused.Robot.Queue[0].RemainingMS != initial.Robot.Queue[0].RemainingMS {
		t.Fatalf("offline time advanced robot: before=%d after=%d", initial.Robot.Queue[0].RemainingMS, paused.Robot.Queue[0].RemainingMS)
	}
	var winner RobotTickResult
	for index := 0; index < 20; index++ {
		clock = clock.Add(2 * time.Second)
		winner, err = service.TickRobot(ctx, second.User)
		if err != nil {
			t.Fatal(err)
		}
	}
	if winner.Event == nil || winner.Event.AutoRedeemed || winner.Event.Ticket.ID != first.Ticket.ID || winner.Event.Ticket.State != domain.TicketScratched || winner.Event.Ticket.Location != domain.TicketOnDesk || winner.User.Balance != 0 {
		t.Fatalf("unexpected winner event: %#v", winner)
	}
	var loser RobotTickResult
	for index := 0; index < 20; index++ {
		clock = clock.Add(2 * time.Second)
		loser, err = service.TickRobot(ctx, winner.User)
		if err != nil {
			t.Fatal(err)
		}
	}
	if loser.Event == nil || loser.Event.AutoRedeemed || loser.Event.Ticket.ID != second.Ticket.ID || loser.Event.Ticket.State != domain.TicketScratched || loser.Event.Ticket.Location != domain.TicketOnDesk || len(loser.Robot.Queue) != 0 {
		t.Fatalf("unexpected loser event: %#v", loser)
	}
}

func TestFanRequiresRiskAcknowledgementAndProcessesDeskOnce(t *testing.T) {
	ctx := context.Background()
	service := New(memory.New())
	clock := time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return clock }
	service.fanRoll = func(int) int { return 99 }
	account, err := service.Register(ctx, "风扇玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	trash, err := service.UpgradeItem(ctx, account.User, "trash", "fan-trash-001")
	if err != nil {
		t.Fatal(err)
	}
	fan, err := service.UpgradeItem(ctx, trash.User, "fan", "buy-fan-001")
	if err != nil {
		t.Fatal(err)
	}
	if fan.User.FanLevel != 1 || fan.User.Balance != 400 {
		t.Fatalf("unexpected fan purchase: %#v", fan.User)
	}
	service.drawCard = func(string, uint8) (domain.Outcome, error) {
		return domain.Outcome{PrizeTier: "none", Symbols: []string{"狗头金币", "钞票", "碎钻石"}}, nil
	}
	first, err := service.Purchase(ctx, fan.User, "lingqian-ticket", "fan-card-001")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Purchase(ctx, first.User, "lingqian-ticket", "fan-card-002")
	if err != nil {
		t.Fatal(err)
	}
	for index, ticket := range []domain.Ticket{first.Ticket, second.Ticket} {
		if _, err := service.PlaceTicket(ctx, account.User.ID, ticket.ID, store.TicketPlacement{
			Location: domain.TicketOnDesk, DeskX: .3 + float64(index)*.2, DeskY: .4, ZIndex: index + 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.Scratch(ctx, account.User.ID, first.Ticket.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BlowFan(ctx, second.User, "fan-event-001", false); !errors.Is(err, store.ErrFanRiskRequired) {
		t.Fatalf("expected risk acknowledgement error, got %v", err)
	}
	result, err := service.BlowFan(ctx, second.User, "fan-event-001", true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.User.FanRiskAcknowledged || len(result.Event.Cards) != 2 || result.Fan.MistakePercent != 80 {
		t.Fatalf("unexpected fan result: %#v", result)
	}
	actions := map[string]string{}
	for _, card := range result.Event.Cards {
		actions[card.Ticket.ID] = card.Action
	}
	if actions[first.Ticket.ID] != "discarded" || actions[second.Ticket.ID] != "safe" {
		t.Fatalf("unexpected fan actions: %#v", actions)
	}
	retry, err := service.BlowFan(ctx, result.User, "fan-event-001", true)
	if err != nil || !retry.Event.Idempotent || len(retry.Event.Cards) != 2 {
		t.Fatalf("fan retry was not idempotent: %#v, %v", retry, err)
	}
	active, err := service.Tickets(ctx, account.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].ID != second.Ticket.ID || active[0].Location != domain.TicketOnDesk {
		t.Fatalf("unexpected tickets after fan: %#v", active)
	}
}

func TestFanRobotInterceptionKeepsCardSafeWhenQueueIsFull(t *testing.T) {
	ctx := context.Background()
	service := New(memory.New())
	service.fanRoll = func(int) int { return 0 }
	account, err := service.Register(ctx, "联动玩家", "correct-horse-42", true)
	if err != nil {
		t.Fatal(err)
	}
	service.drawCard = func(string, uint8) (domain.Outcome, error) {
		return domain.Outcome{PrizeTier: "top", Reward: 2000, Symbols: []string{"钞票堆", "钞票堆", "狗头金币"}}, nil
	}
	seed, err := service.Purchase(ctx, account.User, "lingqian-ticket", "fan-seed-card")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scratch(ctx, account.User.ID, seed.Ticket.ID); err != nil {
		t.Fatal(err)
	}
	funded, err := service.Redeem(ctx, account.User.ID, seed.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	trash, err := service.UpgradeItem(ctx, funded.User, "trash", "fan-link-trash")
	if err != nil {
		t.Fatal(err)
	}
	fan, err := service.UpgradeItem(ctx, trash.User, "fan", "fan-link-buy")
	if err != nil {
		t.Fatal(err)
	}
	robot, err := service.UpgradeItem(ctx, fan.User, "robot", "fan-link-robot")
	if err != nil {
		t.Fatal(err)
	}
	service.drawCard = func(string, uint8) (domain.Outcome, error) {
		return domain.Outcome{PrizeTier: "none", Symbols: []string{"狗头金币", "钞票", "碎钻石"}}, nil
	}
	cards := make([]domain.Ticket, 0, 4)
	current := robot.User
	for index := 0; index < 4; index++ {
		purchase, err := service.Purchase(ctx, current, "lingqian-ticket", "fan-link-card-00"+string(rune('1'+index)))
		if err != nil {
			t.Fatal(err)
		}
		current = purchase.User
		cards = append(cards, purchase.Ticket)
	}
	for _, ticket := range cards[:3] {
		if _, err := service.EnqueueRobot(ctx, current, ticket.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.PlaceTicket(ctx, account.User.ID, cards[3].ID, store.TicketPlacement{
		Location: domain.TicketOnDesk, DeskX: .4, DeskY: .4, ZIndex: 4,
	}); err != nil {
		t.Fatal(err)
	}
	result, err := service.BlowFan(ctx, current, "fan-link-event", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Event.Cards) != 1 || result.Event.Cards[0].Action != "caught" || len(result.Robot.Queue) != 3 {
		t.Fatalf("full robot queue did not catch card safely: %#v", result)
	}
	active, err := service.Tickets(ctx, account.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ticket := range active {
		if ticket.ID == cards[3].ID {
			found = ticket.Location == domain.TicketOnDesk
		}
	}
	if !found {
		t.Fatal("intercepted card was not returned safely to the desk")
	}
}
