package game

import "testing"

func TestShopPricesEffectsAndRelocking(t *testing.T) {
	shop := Shop(1600, 0, 1)
	if len(shop.Items) != 2 {
		t.Fatalf("expected two shop items, got %#v", shop.Items)
	}
	luck := shop.Items[0]
	if luck.Code != LuckItemCode || luck.Level != 0 || luck.NextPrice != 300 || luck.NextEffectPercent != 4 {
		t.Fatalf("unexpected luck item: %#v", luck)
	}
	if len(luck.RelockedCards) != 1 || luck.RelockedCards[0] != "街角杂货铺" {
		t.Fatalf("unexpected relocked cards: %#v", luck.RelockedCards)
	}
	rangeItem := Shop(100000, 10, 10).Items[1]
	if rangeItem.Level != 10 || rangeItem.EffectPercent != 100 || rangeItem.NextPrice != 0 || rangeItem.TotalSpent != 71350 {
		t.Fatalf("unexpected max scratch range: %#v", rangeItem)
	}
}
