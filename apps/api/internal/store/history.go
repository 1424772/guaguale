package store

import (
	"fmt"
	"sort"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

func BuildGameHistory(tickets []domain.Ticket, automated map[string]bool, limit int) []domain.HistoryEvent {
	events := make([]domain.HistoryEvent, 0, len(tickets)*3)
	for _, ticket := range tickets {
		purchaseTitle := "购买刮刮卡"
		purchaseDetail := fmt.Sprintf("花费 %d 金币购买《%s》", ticket.PricePaid, ticket.CardName)
		purchaseDelta := -ticket.PricePaid
		if ticket.Source == "daily_wheel" {
			purchaseTitle = "每日转盘获得"
			purchaseDetail = fmt.Sprintf("免费获得《%s》", ticket.CardName)
			purchaseDelta = 0
		}
		events = append(events, domain.HistoryEvent{
			ID: ticket.ID + ":purchase", Type: "purchase", Title: purchaseTitle, Detail: purchaseDetail,
			CardCode: ticket.CardCode, CardName: ticket.CardName, Delta: purchaseDelta, CreatedAt: ticket.CreatedAt,
		})

		isAutomated := ticket.ScratchSource == "robot" || automated[ticket.ID]
		if ticket.ScratchedAt != nil {
			if isAutomated {
				title := "机器人自动刮奖"
				detail := fmt.Sprintf("《%s》未中奖，卡片已返回桌面", ticket.CardName)
				delta := int64(0)
				if ticket.RedeemedAt != nil && ticket.Reward > 0 {
					title = "机器人自动刮奖并兑奖"
					detail = fmt.Sprintf("《%s》获得 %d 金币", ticket.CardName, ticket.Reward)
					delta = ticket.Reward
				}
				events = append(events, domain.HistoryEvent{
					ID: ticket.ID + ":robot", Type: "robot", Title: title, Detail: detail,
					CardCode: ticket.CardCode, CardName: ticket.CardName, Delta: delta, Automated: true, CreatedAt: *ticket.ScratchedAt,
				})
			} else {
				detail := fmt.Sprintf("《%s》未中奖", ticket.CardName)
				if ticket.Reward > 0 {
					detail = fmt.Sprintf("《%s》刮出 %d 金币", ticket.CardName, ticket.Reward)
				}
				events = append(events, domain.HistoryEvent{
					ID: ticket.ID + ":scratch", Type: "scratch", Title: "手动刮开卡片", Detail: detail,
					CardCode: ticket.CardCode, CardName: ticket.CardName, CreatedAt: *ticket.ScratchedAt,
				})
			}
		}

		if ticket.RedeemedAt != nil && !isAutomated {
			events = append(events, domain.HistoryEvent{
				ID: ticket.ID + ":redeem", Type: "redeem", Title: "兑奖完成",
				Detail:   fmt.Sprintf("《%s》兑奖获得 %d 金币", ticket.CardName, ticket.Reward),
				CardCode: ticket.CardCode, CardName: ticket.CardName, Delta: ticket.Reward, CreatedAt: *ticket.RedeemedAt,
			})
		}

		if ticket.DiscardedAt != nil {
			detail := fmt.Sprintf("《%s》未刮开即被丢弃", ticket.CardName)
			if ticket.ScratchedAt != nil && ticket.Reward > 0 {
				detail = fmt.Sprintf("《%s》中奖 %d 金币但未兑奖，已被丢弃", ticket.CardName, ticket.Reward)
			} else if ticket.ScratchedAt != nil {
				detail = fmt.Sprintf("《%s》未中奖卡已被清理", ticket.CardName)
			}
			events = append(events, domain.HistoryEvent{
				ID: ticket.ID + ":discard", Type: "discard", Title: "卡片进入垃圾桶", Detail: detail,
				CardCode: ticket.CardCode, CardName: ticket.CardName, CreatedAt: *ticket.DiscardedAt,
			})
		}
	}
	sort.SliceStable(events, func(left, right int) bool { return events[left].CreatedAt.After(events[right].CreatedAt) })
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}
	return events
}
