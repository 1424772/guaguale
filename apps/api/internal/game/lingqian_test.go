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
