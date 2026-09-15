package game

import "testing"

func TestLingqianOddsStayNormalized(t *testing.T) {
	for level := uint8(0); level <= 10; level++ {
		odds := LingqianOdds(level)
		total := 0
		for _, value := range odds {
			total += value
		}
		if total != 10000 {
			t.Fatalf("luck level %d odds sum to %d", level, total)
		}
		if odds["none"] < 2000 {
			t.Fatalf("luck level %d miss odds below 20%%", level)
		}
		if odds["jackpot"] > 2000 {
			t.Fatalf("luck level %d jackpot odds above 20%%", level)
		}
	}
}

func TestDrawLingqianProducesValidShape(t *testing.T) {
	for index := 0; index < 100; index++ {
		outcome, err := DrawLingqian(10)
		if err != nil {
			t.Fatal(err)
		}
		if len(outcome.Symbols) != 3 {
			t.Fatalf("expected 3 symbols, got %d", len(outcome.Symbols))
		}
		counts := map[string]int{}
		for _, symbol := range outcome.Symbols {
			counts[symbol]++
		}
		if outcome.Reward == 0 && len(counts) != 3 {
			t.Fatalf("losing outcome contains a match: %#v", outcome)
		}
		if outcome.Reward > 0 && len(counts) == 3 {
			t.Fatalf("winning outcome has no match: %#v", outcome)
		}
	}
}

func TestWheelPoolUsesConfirmedEligibilityAndWeights(t *testing.T) {
	cases := []struct {
		balance int64
		count   int
	}{
		{balance: 49, count: 0},
		{balance: 50, count: 1},
		{balance: 1499, count: 1},
		{balance: 1500, count: 2},
		{balance: 80000, count: 6},
		{balance: 3000000, count: 6},
	}
	for _, test := range cases {
		pool := WheelPool(test.balance)
		if len(pool) != test.count {
			t.Fatalf("balance %d: expected %d cards, got %d", test.balance, test.count, len(pool))
		}
		total := 0
		for _, item := range pool {
			total += item.BasisPoint
			if item.CardCode == "eternal-color-diamond" || item.CardCode == "all-in" {
				t.Fatalf("excluded card appeared in wheel: %#v", item)
			}
		}
		if len(pool) > 0 && total != 10000 {
			t.Fatalf("balance %d: wheel odds total %d", test.balance, total)
		}
	}
}

func TestAllWheelCardsProduceValidSymbolCounts(t *testing.T) {
	expectedCounts := map[string]int{
		"lingqian-ticket":  3,
		"street-store":     7,
		"arcade-challenge": 9,
		"gold-mine":        7,
		"rocket-launch":    4,
		"deep-sea-salvage": 8,
	}
	for code, count := range expectedCounts {
		for index := 0; index < 40; index++ {
			outcome, err := Draw(code, uint8(index%11))
			if err != nil {
				t.Fatalf("%s: %v", code, err)
			}
			if len(outcome.Symbols) != count {
				t.Fatalf("%s: expected %d symbols, got %#v", code, count, outcome.Symbols)
			}
		}
	}
}
