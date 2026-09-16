package game

import "github.com/1424772/guaguale/apps/api/internal/domain"

const FirstCardCode = "lingqian-ticket"

var catalog = []domain.Card{
	{Code: FirstCardCode, Name: "零钱小票", Price: 50, Implemented: true},
	{Code: "street-store", Name: "街角杂货铺", Price: 1500, Implemented: true},
	{Code: "arcade-challenge", Name: "街机挑战券", Price: 5000, Implemented: true},
	{Code: "gold-mine", Name: "黄金矿洞", Price: 15000, Implemented: true},
	{Code: "rocket-launch", Name: "火箭发射", Price: 40000, Implemented: true},
	{Code: "deep-sea-salvage", Name: "深海打捞", Price: 80000, Implemented: true},
	{Code: "eternal-color-diamond", Name: "永恒彩钻", Price: 150000, Implemented: true},
	{Code: "all-in", Name: "放手一博", Price: 300000, Implemented: true},
}

func WheelCatalog(balance int64) []domain.Card {
	result := make([]domain.Card, 0, 6)
	for _, card := range catalog[:6] {
		if balance >= card.Price {
			card.Unlocked = true
			result = append(result, card)
		}
	}
	return result
}

func Catalog(balance int64) []domain.Card {
	result := make([]domain.Card, len(catalog))
	copy(result, catalog)
	for index := range result {
		result[index].Unlocked = balance >= result[index].Price
	}
	return result
}

func CardByCode(code string) (domain.Card, bool) {
	for _, card := range catalog {
		if card.Code == code {
			return card, true
		}
	}
	return domain.Card{}, false
}
