package store

import (
	"strings"
	"testing"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

func TestGameHistoryHidesUnscratchedOutcomeAndLabelsRobot(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	discardedAt := now.Add(time.Minute)
	scratchedAt := now.Add(2 * time.Minute)
	tickets := []domain.Ticket{
		{
			ID: "hidden", CardCode: "all-in", CardName: "放手一博", Price: 300000, PricePaid: 300000,
			Source: "purchase", Reward: 3000000, State: domain.TicketDiscarded, CreatedAt: now, DiscardedAt: &discardedAt,
		},
		{
			ID: "robot-loser", CardCode: "lingqian-ticket", CardName: "零钱小票", Price: 50, PricePaid: 50,
			Source: "purchase", State: domain.TicketScratched, ScratchSource: "robot", CreatedAt: now, ScratchedAt: &scratchedAt,
		},
	}
	events := BuildGameHistory(tickets, nil, 100)
	if len(events) != 4 || events[0].Type != "robot" || !events[0].Automated {
		t.Fatalf("robot event was not preserved: %#v", events)
	}
	for _, event := range events {
		if event.ID == "hidden:discard" && (strings.Contains(event.Detail, "3000000") || strings.Contains(event.Detail, "天使")) {
			t.Fatalf("unscratched outcome leaked into history: %#v", event)
		}
	}
}
