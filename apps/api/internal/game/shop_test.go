package game

import "testing"

func TestShopPricesEffectsAndRelocking(t *testing.T) {
	shop := Shop(1600, 0, 1, false, false, false, 0, 0, 0)
	if len(shop.Items) != 8 {
		t.Fatalf("expected eight shop items, got %#v", shop.Items)
	}
	luck := shop.Items[0]
	if luck.Code != LuckItemCode || luck.Level != 0 || luck.NextPrice != 300 || luck.NextEffectPercent != 4 {
		t.Fatalf("unexpected luck item: %#v", luck)
	}
	if len(luck.RelockedCards) != 1 || luck.RelockedCards[0] != "街角杂货铺" {
		t.Fatalf("unexpected relocked cards: %#v", luck.RelockedCards)
	}
	rangeItem := Shop(100000, 10, 10, true, true, true, 8, 6, 8).Items[1]
	if rangeItem.Level != 10 || rangeItem.EffectPercent != 100 || rangeItem.NextPrice != 0 || rangeItem.TotalSpent != 71350 {
		t.Fatalf("unexpected max scratch range: %#v", rangeItem)
	}
	if shop.Items[2].Code != TrashItemCode || shop.Items[2].NextPrice != 100 || shop.Items[2].EffectText != "未购买" {
		t.Fatalf("unexpected trash item: %#v", shop.Items[2])
	}
	if shop.Items[3].Code != CardSlotsItemCode || shop.Items[3].NextPrice != 500 || shop.Items[3].EffectText != "未购买" {
		t.Fatalf("unexpected card slots item: %#v", shop.Items[3])
	}
	if shop.Items[4].Code != RobotItemCode || shop.Items[4].NextPrice != 1000 {
		t.Fatalf("unexpected robot item: %#v", shop.Items[4])
	}
	if !shop.Items[5].Locked || !shop.Items[6].Locked || !shop.Items[7].Locked {
		t.Fatalf("robot modules should be locked before robot purchase: %#v", shop.Items[5:])
	}
}

func TestLuckCardImpactsMatchBaseConfiguration(t *testing.T) {
	impacts := LuckCardImpacts(0)
	if len(impacts) != 6 {
		t.Fatalf("expected six implemented luck cards, got %#v", impacts)
	}
	wantRTP := []int{7560, 6880, 7260, 7030, 7200, 6880}
	for index, impact := range impacts {
		if impact.CurrentRTPBasisPoint != wantRTP[index] {
			t.Fatalf("%s RTP = %d, want %d", impact.CardName, impact.CurrentRTPBasisPoint, wantRTP[index])
		}
		currentTotal, nextTotal := 0, 0
		for _, tier := range impact.Tiers {
			currentTotal += tier.CurrentBasisPoint
			nextTotal += tier.NextBasisPoint
		}
		if currentTotal != 10000 || nextTotal != 10000 {
			t.Fatalf("%s probabilities do not sum to 100%%: %d/%d", impact.CardName, currentTotal, nextTotal)
		}
	}
}
