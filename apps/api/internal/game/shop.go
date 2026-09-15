package game

import "github.com/1424772/guaguale/apps/api/internal/domain"

const (
	LuckItemCode         = "luck"
	ScratchRangeItemCode = "scratch-range"
)

type itemDefinition struct {
	code         string
	name         string
	category     string
	defaultLevel uint8
	effects      []int
	prices       []int64
	description  string
	notice       string
}

var itemDefinitions = []itemDefinition{
	{
		code: LuckItemCode, name: "好运", category: "luck", defaultLevel: 0,
		effects:     []int{0, 4, 8, 13, 20, 29, 40, 53, 68, 84, 100},
		prices:      []int64{0, 300, 700, 1500, 3000, 6000, 12000, 24000, 45000, 80000, 130000},
		description: "按当前等级在每款卡的基础概率与满级概率之间插值。",
		notice:      "适用于第1–7款卡；《放手一博》不受好运影响。",
	},
	{
		code: ScratchRangeItemCode, name: "刮奖范围", category: "efficiency", defaultLevel: 1,
		effects:     []int{0, 6, 8, 11, 15, 21, 29, 40, 55, 75, 100},
		prices:      []int64{0, 0, 100, 250, 500, 1000, 2200, 4800, 9500, 18000, 35000},
		description: "扩大每次划动清除的刮层范围，只影响操作手感，不改变开奖结果。",
	},
}

func Shop(balance int64, luckLevel, scratchLevel uint8) domain.ShopStatus {
	levels := map[string]uint8{LuckItemCode: luckLevel, ScratchRangeItemCode: scratchLevel}
	items := make([]domain.ShopItem, 0, len(itemDefinitions))
	for _, definition := range itemDefinitions {
		level := levels[definition.code]
		if level < definition.defaultLevel {
			level = definition.defaultLevel
		}
		items = append(items, shopItem(definition, level, balance))
	}
	return domain.ShopStatus{Items: items}
}

func UpgradeDefinition(code string, currentLevel uint8) (domain.ShopItem, bool) {
	for _, definition := range itemDefinitions {
		if definition.code == code {
			return shopItem(definition, currentLevel, 0), true
		}
	}
	return domain.ShopItem{}, false
}

func shopItem(definition itemDefinition, level uint8, balance int64) domain.ShopItem {
	maximum := uint8(len(definition.effects) - 1)
	if level > maximum {
		level = maximum
	}
	item := domain.ShopItem{
		Code: definition.code, Name: definition.name, Category: definition.category,
		Level: level, MaxLevel: maximum, EffectPercent: definition.effects[level],
		Description: definition.description, Notice: definition.notice, RelockedCards: []string{},
	}
	for target := 1; target <= int(level); target++ {
		item.TotalSpent += definition.prices[target]
	}
	if level < maximum {
		item.NextEffectPercent = definition.effects[level+1]
		item.NextPrice = definition.prices[level+1]
		remaining := balance - item.NextPrice
		for _, card := range catalog {
			if balance >= card.Price && remaining < card.Price {
				item.RelockedCards = append(item.RelockedCards, card.Name)
			}
		}
	}
	return item
}
