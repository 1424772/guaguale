package game

import (
	"fmt"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

const (
	LuckItemCode           = "luck"
	ScratchRangeItemCode   = "scratch-range"
	TrashItemCode          = "trash"
	CardSlotsItemCode      = "card-slots"
	FanItemCode            = "fan"
	RobotItemCode          = "robot"
	RobotSpeedItemCode     = "robot-speed"
	RobotQueueItemCode     = "robot-queue"
	RobotInterceptItemCode = "robot-intercept"
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
	effectTexts  []string
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
	{
		code: TrashItemCode, name: "垃圾桶", category: "safety", defaultLevel: 0,
		effects: []int{0, 100}, prices: []int64{0, 100}, effectTexts: []string{"未购买", "永久开放"},
		description: "购买后可以手动丢弃桌面卡片，容量不限。服务端提交前保留5秒撤销。",
	},
	{
		code: CardSlotsItemCode, name: "固定卡槽", category: "safety", defaultLevel: 0,
		effects: []int{0, 100}, prices: []int64{0, 500}, effectTexts: []string{"未购买", "10个固定卡位"},
		description: "购买后永久开放10个卡位，固定的卡不会被丢弃或参与后续风扇清理。",
	},
	{
		code: FanItemCode, name: "桌面清理风扇", category: "safety", defaultLevel: 0,
		effects:     []int{0, 80, 68, 55, 42, 30, 18, 8, 0},
		prices:      []int64{0, 500, 300, 700, 1500, 3000, 6000, 12000, 24000},
		effectTexts: []string{"未购买", "1.0x风力 · 吹错80%", "1.2x风力 · 吹错68%", "1.5x风力 · 吹错55%", "1.9x风力 · 吹错42%", "2.4x风力 · 吹错30%", "3.0x风力 · 吹错18%", "3.8x风力 · 吹错8%", "5.0x风力 · 吹错0%"},
		description: "按住风扇清理自由桌面。已刮卡直接进垃圾桶，未刮卡先由机器人拦截，再按吹错率判定。",
		notice:      "固定卡槽完全不受影响；首次启动前必须确认风险。",
	},
	{
		code: RobotItemCode, name: "自动刮奖机器人", category: "efficiency", defaultLevel: 0,
		effects: []int{0, 100}, prices: []int64{0, 1000}, effectTexts: []string{"未购买", "速度1 · 队列3 · 拦截45%"},
		description: "购买后可把未刮开的卡交给机器人依次处理；中奖自动兑奖，未中奖退回桌面。",
		notice:      "关闭或切到后台时暂停，不补算离线进度；《放手一博》只能手动刮。",
	},
	{
		code: RobotSpeedItemCode, name: "机器人速度", category: "efficiency", defaultLevel: 1,
		effects:     []int{0, 40, 32, 25, 19, 14, 10, 7, 5},
		prices:      []int64{0, 0, 500, 1000, 2200, 4500, 9000, 18000, 36000},
		effectTexts: []string{"需先购买机器人", "40秒/张", "32秒/张", "25秒/张", "19秒/张", "14秒/张", "10秒/张", "7秒/张", "5秒/张"},
		description: "缩短机器人处理每张卡所需的前台运行时间。",
	},
	{
		code: RobotQueueItemCode, name: "机器人队列", category: "efficiency", defaultLevel: 1,
		effects:     []int{0, 3, 5, 8, 12, 18, 30},
		prices:      []int64{0, 0, 300, 700, 1600, 4000, 10000},
		effectTexts: []string{"需先购买机器人", "3张", "5张", "8张", "12张", "18张", "30张"},
		description: "增加机器人可同时等待处理的卡片数量。",
	},
	{
		code: RobotInterceptItemCode, name: "机器人拦截", category: "safety", defaultLevel: 1,
		effects:     []int{0, 45, 55, 64, 72, 80, 87, 94, 100},
		prices:      []int64{0, 0, 600, 1200, 2500, 5000, 10000, 22000, 45000},
		description: "决定后续风扇吹动未刮卡时，机器人成功拦截并保护卡片的概率。",
	},
}

func Shop(balance int64, luckLevel, scratchLevel uint8, trashOwned, cardSlotsOwned bool, fanLevel uint8, robotOwned bool, robotSpeedLevel, robotQueueLevel, robotInterceptLevel uint8) domain.ShopStatus {
	levels := map[string]uint8{
		LuckItemCode: luckLevel, ScratchRangeItemCode: scratchLevel,
		TrashItemCode: boolLevel(trashOwned), CardSlotsItemCode: boolLevel(cardSlotsOwned),
		FanItemCode:   fanLevel,
		RobotItemCode: boolLevel(robotOwned), RobotSpeedItemCode: robotSpeedLevel,
		RobotQueueItemCode: robotQueueLevel, RobotInterceptItemCode: robotInterceptLevel,
	}
	items := make([]domain.ShopItem, 0, len(itemDefinitions))
	for _, definition := range itemDefinitions {
		level := levels[definition.code]
		if level < definition.defaultLevel && (robotOwned || !isRobotModule(definition.code)) {
			level = definition.defaultLevel
		}
		item := shopItem(definition, level, balance)
		if definition.code == FanItemCode && !trashOwned {
			item.Locked = true
			item.LockedReason = "请先购买垃圾桶"
			item.NextPrice = 0
		}
		if !robotOwned && isRobotModule(definition.code) {
			item.Locked = true
			item.LockedReason = "请先购买自动刮奖机器人"
			item.NextPrice = 0
		}
		items = append(items, item)
	}
	return domain.ShopStatus{Items: items, LuckCards: LuckCardImpacts(luckLevel)}
}

func isRobotModule(code string) bool {
	return code == RobotSpeedItemCode || code == RobotQueueItemCode || code == RobotInterceptItemCode
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
	item.EffectText = effectText(definition, level)
	for target := 1; target <= int(level); target++ {
		item.TotalSpent += definition.prices[target]
	}
	if level < maximum {
		item.NextEffectPercent = definition.effects[level+1]
		item.NextPrice = definition.prices[level+1]
		item.NextEffectText = effectText(definition, level+1)
		remaining := balance - item.NextPrice
		for _, card := range catalog {
			if balance >= card.Price && remaining < card.Price {
				item.RelockedCards = append(item.RelockedCards, card.Name)
			}
		}
	}
	return item
}

func effectText(definition itemDefinition, level uint8) string {
	if len(definition.effectTexts) > int(level) {
		return definition.effectTexts[level]
	}
	return fmt.Sprintf("%d%%", definition.effects[level])
}

func boolLevel(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}
