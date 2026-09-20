package game

import "testing"

func TestShopPricesEffectsAndRelocking(t *testing.T) {
	shop := Shop(1600, 0, 1, false, false, 0, false, 0, 0, 0)
	if len(shop.Items) != 9 {
		t.Fatalf("expected nine shop items, got %#v", shop.Items)
	}
	luck := shop.Items[0]
	if luck.Code != LuckItemCode || luck.Level != 0 || luck.NextPrice != 300 || luck.NextEffectPercent != 4 {
		t.Fatalf("unexpected luck item: %#v", luck)
	}
	if len(luck.RelockedCards) != 1 || luck.RelockedCards[0] != "街角杂货铺" {
		t.Fatalf("unexpected relocked cards: %#v", luck.RelockedCards)
	}
	rangeItem := Shop(100000, 10, 10, true, true, 8, true, 8, 6, 8).Items[1]
	if rangeItem.Level != 10 || rangeItem.EffectPercent != 100 || rangeItem.NextPrice != 0 || rangeItem.TotalSpent != 71350 {
		t.Fatalf("unexpected max scratch range: %#v", rangeItem)
	}
	if shop.Items[2].Code != TrashItemCode || shop.Items[2].NextPrice != 100 || shop.Items[2].EffectText != "未购买" {
		t.Fatalf("unexpected trash item: %#v", shop.Items[2])
	}
	if shop.Items[3].Code != CardSlotsItemCode || shop.Items[3].NextPrice != 500 || shop.Items[3].EffectText != "未购买" {
		t.Fatalf("unexpected card slots item: %#v", shop.Items[3])
	}
	if shop.Items[4].Code != FanItemCode || !shop.Items[4].Locked {
		t.Fatalf("fan should require the trash item: %#v", shop.Items[4])
	}
	if shop.Items[5].Code != RobotItemCode || shop.Items[5].NextPrice != 1000 {
		t.Fatalf("unexpected robot item: %#v", shop.Items[5])
	}
	if !shop.Items[6].Locked || !shop.Items[7].Locked || !shop.Items[8].Locked {
		t.Fatalf("robot modules should be locked before robot purchase: %#v", shop.Items[6:])
	}
}

func TestLuckCardImpactsMatchBaseConfiguration(t *testing.T) {
	impacts := LuckCardImpacts(0)
	if len(impacts) != 7 {
		t.Fatalf("expected seven implemented luck cards, got %#v", impacts)
	}
	wantRTP := []int{5250, 7790, 7250, 7300, 7550, 8650, 7820}
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
	maxImpacts := LuckCardImpacts(10)
	for index, impact := range maxImpacts {
		if impact.CurrentRTPBasisPoint <= impacts[index].CurrentRTPBasisPoint {
			t.Fatalf("%s max-luck RTP %d did not improve on base %d", impact.CardName, impact.CurrentRTPBasisPoint, impacts[index].CurrentRTPBasisPoint)
		}
		if impact.CurrentRTPBasisPoint > 10000 {
			t.Fatalf("%s max-luck RTP %d exceeds 100%%", impact.CardName, impact.CurrentRTPBasisPoint)
		}
	}
	if maxImpacts[6].CurrentRTPBasisPoint != 9205 {
		t.Fatalf("max-luck diamond RTP = %d, want 9205", maxImpacts[6].CurrentRTPBasisPoint)
	}
}
