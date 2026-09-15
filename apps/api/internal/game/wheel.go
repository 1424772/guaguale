package game

import (
	"fmt"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

var wheelWeights = [...]int{30, 22, 16, 12, 8, 6}

func WheelPool(balance int64) []domain.WheelPoolItem {
	cards := WheelCatalog(balance)
	if len(cards) == 0 {
		return []domain.WheelPoolItem{}
	}
	totalWeight := 0
	for index := range cards {
		totalWeight += wheelWeights[index]
	}
	result := make([]domain.WheelPoolItem, len(cards))
	remainingBasisPoints := 10000
	for index, card := range cards {
		basisPoints := 0
		if index == len(cards)-1 {
			basisPoints = remainingBasisPoints
		} else {
			basisPoints = (wheelWeights[index]*10000 + totalWeight/2) / totalWeight
			remainingBasisPoints -= basisPoints
		}
		result[index] = domain.WheelPoolItem{
			CardCode:   card.Code,
			CardName:   card.Name,
			Price:      card.Price,
			Weight:     wheelWeights[index],
			BasisPoint: basisPoints,
		}
	}
	return result
}

func DrawWheelCard(balance int64) (domain.Card, []domain.WheelPoolItem, error) {
	pool := WheelPool(balance)
	if len(pool) == 0 {
		return domain.Card{}, pool, fmt.Errorf("no eligible wheel cards")
	}
	roll, err := randomInt(10000)
	if err != nil {
		return domain.Card{}, nil, err
	}
	threshold := 0
	for _, item := range pool {
		threshold += item.BasisPoint
		if roll < threshold {
			card, _ := CardByCode(item.CardCode)
			return card, pool, nil
		}
	}
	card, _ := CardByCode(pool[len(pool)-1].CardCode)
	return card, pool, nil
}
