package game

import (
	"fmt"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

type weightedResult struct {
	tier    string
	base    int
	maximum int
	reward  int64
	symbol  string
}

var streetResults = []weightedResult{
	{tier: "jackpot", base: 300, maximum: 400, reward: 4500, symbol: "游戏机"},
	{tier: "television", base: 500, maximum: 800, reward: 3000, symbol: "小电视"},
	{tier: "toy_car", base: 600, maximum: 1000, reward: 2250, symbol: "玩具车"},
	{tier: "ice_cream", base: 900, maximum: 1300, reward: 1800, symbol: "雪糕"},
	{tier: "ice_pop", base: 1200, maximum: 1700, reward: 1500, symbol: "冰棍"},
	{tier: "cola", base: 1500, maximum: 1400, reward: 1200, symbol: "可乐"},
	{tier: "latiao", base: 1500, maximum: 1400, reward: 600, symbol: "辣条"},
	{tier: "none", base: 3500, maximum: 2000},
}

var arcadeResults = []weightedResult{
	{tier: "jackpot", base: 500, maximum: 700, reward: 15000, symbol: "街机皇冠"},
	{tier: "plane", base: 1000, maximum: 1400, reward: 8000, symbol: "飞机"},
	{tier: "race_car", base: 1500, maximum: 2400, reward: 5200, symbol: "赛车"},
	{tier: "boxing_glove", base: 2000, maximum: 1700, reward: 4000, symbol: "拳套"},
	{tier: "marble", base: 2000, maximum: 1800, reward: 2500, symbol: "弹珠"},
	{tier: "none", base: 3000, maximum: 2000},
}

var goldMineResults = []weightedResult{
	{tier: "match_6", base: 500, maximum: 700, reward: 45000, symbol: "6"},
	{tier: "match_5", base: 900, maximum: 1400, reward: 24000, symbol: "5"},
	{tier: "match_4", base: 1500, maximum: 2500, reward: 15000, symbol: "4"},
	{tier: "match_3", base: 1800, maximum: 1600, reward: 12000, symbol: "3"},
	{tier: "match_2", base: 2300, maximum: 1800, reward: 7500, symbol: "2"},
	{tier: "none", base: 3000, maximum: 2000, symbol: "0"},
}

var rocketResults = []weightedResult{
	{tier: "fuel_11_12", base: 500, maximum: 700, reward: 120000, symbol: "11-12"},
	{tier: "fuel_9_10", base: 1000, maximum: 1500, reward: 64000, symbol: "9-10"},
	{tier: "fuel_8", base: 1500, maximum: 2400, reward: 40000, symbol: "8"},
	{tier: "fuel_6_7", base: 2000, maximum: 1700, reward: 32000, symbol: "6-7"},
	{tier: "fuel_4_5", base: 2000, maximum: 1700, reward: 20000, symbol: "4-5"},
	{tier: "none", base: 3000, maximum: 2000, symbol: "0-3"},
}

var deepSeaResults = []weightedResult{
	{tier: "crown", base: 300, maximum: 600, reward: 240000, symbol: "海神王冠"},
	{tier: "chest", base: 800, maximum: 1800, reward: 120000, symbol: "黄金宝箱"},
	{tier: "pearl", base: 2000, maximum: 2600, reward: 72000, symbol: "珍珠贝"},
	{tier: "anchor", base: 2400, maximum: 2200, reward: 56000, symbol: "生锈船锚"},
	{tier: "bottle", base: 2000, maximum: 1300, reward: 32000, symbol: "漂流瓶"},
	{tier: "boot", base: 1500, maximum: 900, reward: 20000, symbol: "破皮靴"},
	{tier: "seaweed", base: 1000, maximum: 600, reward: 10000, symbol: "海草团"},
}

var diamondResults = []weightedResult{
	{tier: "eternal", base: 150, maximum: 300, reward: 450000, symbol: "永恒级"},
	{tier: "royal", base: 350, maximum: 600, reward: 270000, symbol: "皇室级"},
	{tier: "collection", base: 600, maximum: 1100, reward: 180000, symbol: "典藏级"},
	{tier: "selected", base: 1000, maximum: 1500, reward: 157500, symbol: "精选级"},
	{tier: "rare", base: 1200, maximum: 1800, reward: 150000, symbol: "稀有级"},
	{tier: "jewelry", base: 1800, maximum: 1800, reward: 120000, symbol: "珠宝级"},
	{tier: "industrial", base: 2200, maximum: 1600, reward: 75000, symbol: "工业级"},
	{tier: "cracked", base: 2700, maximum: 1300, reward: 30000, symbol: "裂纹级"},
}

var diamondGrades = []string{"裂纹级", "工业级", "珠宝级", "稀有级", "精选级", "典藏级", "皇室级", "永恒级"}

const (
	allInAngelWeight  = 1
	allInScytheWeight = 100
	allInSkullWeight  = 99899
)

func Draw(cardCode string, luckLevel uint8) (domain.Outcome, error) {
	switch cardCode {
	case FirstCardCode:
		return DrawLingqian(luckLevel)
	case "street-store":
		return drawStreetStore(luckLevel)
	case "arcade-challenge":
		return drawArcade(luckLevel)
	case "gold-mine":
		return drawGoldMine(luckLevel)
	case "rocket-launch":
		return drawRocket(luckLevel)
	case "deep-sea-salvage":
		return drawDeepSea(luckLevel)
	case "eternal-color-diamond":
		return drawEternalColorDiamond(luckLevel)
	case "all-in":
		return drawAllIn()
	default:
		return domain.Outcome{}, fmt.Errorf("unsupported card code %q", cardCode)
	}
}

func drawEternalColorDiamond(luckLevel uint8) (domain.Outcome, error) {
	selected, err := chooseWeighted(diamondResults, luckLevel)
	if err != nil {
		return domain.Outcome{}, err
	}
	minimumIndex := 0
	for index, grade := range diamondGrades {
		if grade == selected.symbol {
			minimumIndex = index
			break
		}
	}
	symbols := []string{selected.symbol}
	for len(symbols) < 5 {
		index, err := randomInt(len(diamondGrades) - minimumIndex)
		if err != nil {
			return domain.Outcome{}, err
		}
		symbols = append(symbols, diamondGrades[minimumIndex+index])
	}
	if err := shuffle(symbols); err != nil {
		return domain.Outcome{}, err
	}
	return domain.Outcome{PrizeTier: selected.tier, Reward: selected.reward, Symbols: symbols}, nil
}

// AllInOdds deliberately ignores luckLevel: this card's odds never change.
func AllInOdds(_ uint8) map[string]int {
	return map[string]int{"angel": allInAngelWeight, "scythe": allInScytheWeight, "skull": allInSkullWeight}
}

func drawAllIn() (domain.Outcome, error) {
	roll, err := randomInt(100000)
	if err != nil {
		return domain.Outcome{}, err
	}
	if roll < allInAngelWeight {
		return domain.Outcome{PrizeTier: "angel", Reward: 3000000, Symbols: []string{"天使"}}, nil
	}
	if roll < allInAngelWeight+allInScytheWeight {
		return domain.Outcome{PrizeTier: "scythe", Reward: 300000, Symbols: []string{"镰刀"}}, nil
	}
	return domain.Outcome{PrizeTier: "skull", Symbols: []string{"骷髅头"}}, nil
}

func drawStreetStore(luckLevel uint8) (domain.Outcome, error) {
	selected, err := chooseWeighted(streetResults, luckLevel)
	if err != nil {
		return domain.Outcome{}, err
	}
	all := []string{"辣条", "可乐", "冰棍", "雪糕", "玩具车", "小电视", "游戏机"}
	if selected.tier == "none" {
		pool := duplicatePool(all, 2)
		if err := shuffle(pool); err != nil {
			return domain.Outcome{}, err
		}
		return domain.Outcome{PrizeTier: "none", Symbols: pool[:7]}, nil
	}
	fillers := duplicatePool(without(all, selected.symbol), 2)
	if err := shuffle(fillers); err != nil {
		return domain.Outcome{}, err
	}
	symbols := append([]string{selected.symbol, selected.symbol, selected.symbol}, fillers[:4]...)
	if err := shuffle(symbols); err != nil {
		return domain.Outcome{}, err
	}
	return domain.Outcome{PrizeTier: selected.tier, Reward: selected.reward, Symbols: symbols}, nil
}

func drawArcade(luckLevel uint8) (domain.Outcome, error) {
	selected, err := chooseWeighted(arcadeResults, luckLevel)
	if err != nil {
		return domain.Outcome{}, err
	}
	all := []string{"弹珠", "拳套", "赛车", "飞机", "街机皇冠"}
	rows := make([][]string, 3)
	for index := range rows {
		row, err := nonMatchingRow(all)
		if err != nil {
			return domain.Outcome{}, err
		}
		rows[index] = row
	}
	if selected.tier != "none" {
		winningRow, err := randomInt(3)
		if err != nil {
			return domain.Outcome{}, err
		}
		rows[winningRow] = []string{selected.symbol, selected.symbol, selected.symbol}
	}
	symbols := append(append(rows[0], rows[1]...), rows[2]...)
	return domain.Outcome{PrizeTier: selected.tier, Reward: selected.reward, Symbols: symbols}, nil
}

func drawGoldMine(luckLevel uint8) (domain.Outcome, error) {
	selected, err := chooseWeighted(goldMineResults, luckLevel)
	if err != nil {
		return domain.Outcome{}, err
	}
	matchCount := 0
	if selected.tier == "none" {
		matchCount, err = randomInt(2)
		if err != nil {
			return domain.Outcome{}, err
		}
	} else {
		_, err = fmt.Sscanf(selected.symbol, "%d", &matchCount)
		if err != nil {
			return domain.Outcome{}, err
		}
	}
	mines := []string{"青晶簇", "红晶簇", "紫晶簇", "金色矿石"}
	targetIndex, err := randomInt(len(mines))
	if err != nil {
		return domain.Outcome{}, err
	}
	target := mines[targetIndex]
	points := make([]string, 0, 6)
	for index := 0; index < matchCount; index++ {
		points = append(points, target)
	}
	other := without(mines, target)
	for len(points) < 6 {
		index, err := randomInt(len(other))
		if err != nil {
			return domain.Outcome{}, err
		}
		points = append(points, other[index])
	}
	if err := shuffle(points); err != nil {
		return domain.Outcome{}, err
	}
	symbols := append([]string{"目标·" + target}, points...)
	return domain.Outcome{PrizeTier: selected.tier, Reward: selected.reward, Symbols: symbols}, nil
}

func drawRocket(luckLevel uint8) (domain.Outcome, error) {
	selected, err := chooseWeighted(rocketResults, luckLevel)
	if err != nil {
		return domain.Outcome{}, err
	}
	minimum, maximum, err := parseRange(selected.symbol)
	if err != nil {
		return domain.Outcome{}, err
	}
	total := minimum
	if maximum > minimum {
		offset, err := randomInt(maximum - minimum + 1)
		if err != nil {
			return domain.Outcome{}, err
		}
		total += offset
	}
	combinations := make([][4]int, 0)
	for a := 0; a <= 3; a++ {
		for b := 0; b <= 3; b++ {
			for c := 0; c <= 3; c++ {
				for d := 0; d <= 3; d++ {
					if a+b+c+d == total {
						combinations = append(combinations, [4]int{a, b, c, d})
					}
				}
			}
		}
	}
	choice, err := randomInt(len(combinations))
	if err != nil {
		return domain.Outcome{}, err
	}
	values := combinations[choice]
	symbols := []string{fmt.Sprintf("燃料 %d", values[0]), fmt.Sprintf("燃料 %d", values[1]), fmt.Sprintf("燃料 %d", values[2]), fmt.Sprintf("燃料 %d", values[3])}
	return domain.Outcome{PrizeTier: selected.tier, Reward: selected.reward, Symbols: symbols}, nil
}

func drawDeepSea(luckLevel uint8) (domain.Outcome, error) {
	selected, err := chooseWeighted(deepSeaResults, luckLevel)
	if err != nil {
		return domain.Outcome{}, err
	}
	ordered := []string{"海神王冠", "黄金宝箱", "珍珠贝", "生锈船锚", "漂流瓶", "破皮靴", "海草团", "空网"}
	selectedIndex := 0
	for index, symbol := range ordered {
		if symbol == selected.symbol {
			selectedIndex = index
			break
		}
	}
	symbols := []string{selected.symbol}
	fillers := ordered[selectedIndex+1:]
	for len(symbols) < 8 {
		index, err := randomInt(len(fillers))
		if err != nil {
			return domain.Outcome{}, err
		}
		symbols = append(symbols, fillers[index])
	}
	if err := shuffle(symbols); err != nil {
		return domain.Outcome{}, err
	}
	return domain.Outcome{PrizeTier: selected.tier, Reward: selected.reward, Symbols: symbols}, nil
}

func chooseWeighted(results []weightedResult, luckLevel uint8) (weightedResult, error) {
	if luckLevel > 10 {
		luckLevel = 10
	}
	coefficient := luckCoefficients[luckLevel]
	roll, err := randomInt(10000)
	if err != nil {
		return weightedResult{}, err
	}
	threshold := 0
	remaining := 10000
	for index, result := range results {
		weight := remaining
		if index < len(results)-1 {
			weight = interpolateWeight(result.base, result.maximum, coefficient)
			remaining -= weight
		}
		threshold += weight
		if roll < threshold {
			return result, nil
		}
	}
	return results[len(results)-1], nil
}

func interpolateWeight(base, maximum, coefficient int) int {
	scaled := (maximum - base) * coefficient
	if scaled >= 0 {
		scaled += 5000
	} else {
		scaled -= 5000
	}
	return base + scaled/10000
}

func duplicatePool(values []string, copies int) []string {
	result := make([]string, 0, len(values)*copies)
	for _, value := range values {
		for count := 0; count < copies; count++ {
			result = append(result, value)
		}
	}
	return result
}

func without(values []string, excluded string) []string {
	result := make([]string, 0, len(values)-1)
	for _, value := range values {
		if value != excluded {
			result = append(result, value)
		}
	}
	return result
}

func nonMatchingRow(symbols []string) ([]string, error) {
	row := make([]string, 3)
	for index := range row {
		choice, err := randomInt(len(symbols))
		if err != nil {
			return nil, err
		}
		row[index] = symbols[choice]
	}
	if row[0] == row[1] && row[1] == row[2] {
		others := without(symbols, row[0])
		choice, err := randomInt(len(others))
		if err != nil {
			return nil, err
		}
		row[2] = others[choice]
	}
	return row, nil
}

func parseRange(value string) (int, int, error) {
	var minimum, maximum int
	if _, err := fmt.Sscanf(value, "%d-%d", &minimum, &maximum); err == nil {
		return minimum, maximum, nil
	}
	if _, err := fmt.Sscanf(value, "%d", &minimum); err != nil {
		return 0, 0, err
	}
	return minimum, minimum, nil
}
