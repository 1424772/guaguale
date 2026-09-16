package game

var fanForceTexts = []string{"", "1.0x", "1.2x", "1.5x", "1.9x", "2.4x", "3.0x", "3.8x", "5.0x"}
var fanMistakePercents = []int{0, 80, 68, 55, 42, 30, 18, 8, 0}

func FanForceText(level uint8) string {
	if level == 0 {
		return "未购买"
	}
	if int(level) >= len(fanForceTexts) {
		return fanForceTexts[len(fanForceTexts)-1]
	}
	return fanForceTexts[level]
}

func FanMistakePercent(level uint8) int {
	return robotValue(fanMistakePercents, level)
}

func FanDiscardPercent(mistakePercent, interceptPercent int) float64 {
	return float64((100-interceptPercent)*mistakePercent) / 100
}
