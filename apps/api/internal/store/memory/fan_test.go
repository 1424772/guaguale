package memory

import (
	"context"
	"testing"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	basestore "github.com/1424772/guaguale/apps/api/internal/store"
)

func TestAllInCardBypassesRobotInterception(t *testing.T) {
	memory := New()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	memory.users[1] = userRecord{user: domain.User{
		ID: 1, FanLevel: 1, TrashOwned: true, RobotOwned: true, FanRiskAcknowledged: true,
	}}
	memory.tickets["all-in-1"] = domain.Ticket{
		ID: "all-in-1", UserID: 1, CardCode: "all-in", CardName: "放手一博",
		State: domain.TicketPurchased, Location: domain.TicketOnDesk,
	}

	_, event, queue, err := memory.BlowFan(context.Background(), basestore.BlowFanInput{
		UserID: 1, EventID: "all-in-fan", MistakePercent: 100, InterceptPercent: 100,
		RobotCapacity: 3, RobotDurationMS: 1000, Now: now, Roll: func(int) int { return 0 },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(queue) != 0 || len(event.Cards) != 1 || event.Cards[0].Action != "discarded" {
		t.Fatalf("all-in card was intercepted by robot: event=%#v queue=%#v", event, queue)
	}
}
