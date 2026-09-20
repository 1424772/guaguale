package game

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

type prizeDefinition struct {
	tier      string
	symbol    string
	baseOdds  int
	maxOdds   int
	basePrize int64
}

var lingqianPrizes = []prizeDefinition{
	{tier: "cash_stack", symbol: "钞票堆", baseOdds: 200, maxOdds: 300, basePrize: 75},
	{tier: "first", symbol: "碎钻石", baseOdds: 1000, maxOdds: 1250, basePrize: 60},
	{tier: "second", symbol: "钞票", baseOdds: 2000, maxOdds: 2400, basePrize: 50},
	{tier: "third", symbol: "狗头金币", baseOdds: 3000, maxOdds: 3700, basePrize: 25},
}

var luckCoefficients = [...]int{0, 400, 800, 1300, 2000, 2900, 4000, 5300, 6800, 8400, 10000}

func LingqianOdds(luckLevel uint8) map[string]int {
	if luckLevel > 10 {
		luckLevel = 10
	}
	coefficient := luckCoefficients[luckLevel]
	odds := make(map[string]int, len(lingqianPrizes)+1)
	winningTotal := 0
	for _, prize := range lingqianPrizes {
		value := prize.baseOdds + ((prize.maxOdds-prize.baseOdds)*coefficient+5000)/10000
		odds[prize.tier] = value
		winningTotal += value
	}
	odds["none"] = 10000 - winningTotal
	return odds
}

func DrawLingqian(luckLevel uint8) (domain.Outcome, error) {
	odds := LingqianOdds(luckLevel)
	roll, err := randomInt(10000)
	if err != nil {
		return domain.Outcome{}, fmt.Errorf("draw prize tier: %w", err)
	}

	threshold := 0
	for _, prize := range lingqianPrizes {
		threshold += odds[prize.tier]
		if roll < threshold {
			return winningOutcome(prize)
		}
	}
	return losingOutcome()
}

func winningOutcome(prize prizeDefinition) (domain.Outcome, error) {
	tripleRoll, err := randomInt(10000)
	if err != nil {
		return domain.Outcome{}, fmt.Errorf("draw match shape: %w", err)
	}
	triple := tripleRoll < 500
	symbols := []string{prize.symbol, prize.symbol, prize.symbol}
	if !triple {
		otherSymbols := symbolsExcept(prize.symbol)
		otherIndex, err := randomInt(len(otherSymbols))
		if err != nil {
			return domain.Outcome{}, fmt.Errorf("draw secondary symbol: %w", err)
		}
		symbols[2] = otherSymbols[otherIndex]
	}
	if err := shuffle(symbols); err != nil {
		return domain.Outcome{}, err
	}
	reward := prize.basePrize
	if triple {
		reward *= 2
	}
	tier := prize.tier
	if triple && prize.tier == "cash_stack" {
		tier = "jackpot"
	}
	return domain.Outcome{PrizeTier: tier, Reward: reward, Symbols: symbols}, nil
}

func losingOutcome() (domain.Outcome, error) {
	symbols := allSymbols()
	if err := shuffle(symbols); err != nil {
		return domain.Outcome{}, err
	}
	return domain.Outcome{PrizeTier: "none", Reward: 0, Symbols: symbols[:3]}, nil
}

func allSymbols() []string {
	result := make([]string, 0, len(lingqianPrizes))
	for _, prize := range lingqianPrizes {
		result = append(result, prize.symbol)
	}
	return result
}

func symbolsExcept(excluded string) []string {
	result := make([]string, 0, len(lingqianPrizes)-1)
	for _, symbol := range allSymbols() {
		if symbol != excluded {
			result = append(result, symbol)
		}
	}
	return result
}

func shuffle(values []string) error {
	for index := len(values) - 1; index > 0; index-- {
		other, err := randomInt(index + 1)
		if err != nil {
			return fmt.Errorf("shuffle symbols: %w", err)
		}
		values[index], values[other] = values[other], values[index]
	}
	return nil
}

func randomInt(max int) (int, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}
