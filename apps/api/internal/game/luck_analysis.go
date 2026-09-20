package game

import (
	"fmt"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

func LuckCardImpacts(currentLevel uint8) []domain.LuckCardImpact {
	if currentLevel > 10 {
		currentLevel = 10
	}
	nextLevel := currentLevel
	if nextLevel < 10 {
		nextLevel++
	}
	impacts := make([]domain.LuckCardImpact, 0, 7)
	impacts = append(impacts, lingqianImpact(currentLevel, nextLevel))
	definitions := []struct {
		card    domain.Card
		results []weightedResult
	}{
		{catalog[1], streetResults},
		{catalog[2], arcadeResults},
		{catalog[3], goldMineResults},
		{catalog[4], rocketResults},
		{catalog[5], deepSeaResults},
		{catalog[6], diamondResults},
	}
	for _, definition := range definitions {
		currentWeights := resultWeights(definition.results, currentLevel)
		nextWeights := resultWeights(definition.results, nextLevel)
		tiers := make([]domain.LuckTierImpact, 0, len(definition.results))
		for index, result := range definition.results {
			tiers = append(tiers, domain.LuckTierImpact{
				Label:             impactLabel(definition.card.Code, result),
				RewardText:        fmt.Sprintf("%d金币", result.reward),
				CurrentBasisPoint: currentWeights[index], NextBasisPoint: nextWeights[index],
			})
		}
		impacts = append(impacts, domain.LuckCardImpact{
			CardCode: definition.card.Code, CardName: definition.card.Name,
			CurrentLevel: currentLevel, NextLevel: nextLevel,
			CurrentRTPBasisPoint: resultRTP(definition.results, currentWeights, definition.card.Price),
			NextRTPBasisPoint:    resultRTP(definition.results, nextWeights, definition.card.Price),
			Tiers:                tiers,
		})
	}
	return impacts
}

func lingqianImpact(currentLevel, nextLevel uint8) domain.LuckCardImpact {
	currentOdds := LingqianOdds(currentLevel)
	nextOdds := LingqianOdds(nextLevel)
	tiers := make([]domain.LuckTierImpact, 0, len(lingqianPrizes)+1)
	currentExpected, nextExpected := int64(0), int64(0)
	for _, prize := range lingqianPrizes {
		label := prize.symbol
		if prize.tier == "cash_stack" {
			label += "（三同图为头奖）"
		}
		tiers = append(tiers, domain.LuckTierImpact{
			Label: label, RewardText: fmt.Sprintf("%d金币；三同%d金币", prize.basePrize, prize.basePrize*2),
			CurrentBasisPoint: currentOdds[prize.tier], NextBasisPoint: nextOdds[prize.tier],
		})
		currentExpected += int64(currentOdds[prize.tier]) * prize.basePrize * 105
		nextExpected += int64(nextOdds[prize.tier]) * prize.basePrize * 105
	}
	tiers = append(tiers, domain.LuckTierImpact{Label: "未中奖", RewardText: "0金币", CurrentBasisPoint: currentOdds["none"], NextBasisPoint: nextOdds["none"]})
	denominator := int64(100) * catalog[0].Price
	return domain.LuckCardImpact{
		CardCode: catalog[0].Code, CardName: catalog[0].Name,
		CurrentLevel: currentLevel, NextLevel: nextLevel,
		CurrentRTPBasisPoint: int((currentExpected + denominator/2) / denominator),
		NextRTPBasisPoint:    int((nextExpected + denominator/2) / denominator),
		Tiers:                tiers,
	}
}

func resultWeights(results []weightedResult, luckLevel uint8) []int {
	coefficient := luckCoefficients[luckLevel]
	weights := make([]int, len(results))
	remaining := 10000
	for index, result := range results {
		weight := remaining
		if index < len(results)-1 {
			weight = interpolateWeight(result.base, result.maximum, coefficient)
			remaining -= weight
		}
		weights[index] = weight
	}
	return weights
}

func resultRTP(results []weightedResult, weights []int, price int64) int {
	total := int64(0)
	for index, result := range results {
		total += int64(weights[index]) * result.reward
	}
	return int((total + price/2) / price)
}

func impactLabel(cardCode string, result weightedResult) string {
	if result.tier == "none" {
		return "未中奖"
	}
	if result.tier == "jackpot" {
		return "头奖 · " + result.symbol
	}
	switch cardCode {
	case "gold-mine":
		return result.symbol + "个目标矿石"
	case "rocket-launch":
		return "燃料合计" + result.symbol
	default:
		return result.symbol
	}
}
