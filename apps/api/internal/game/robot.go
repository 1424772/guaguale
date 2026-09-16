package game

var robotSpeedSeconds = []int{0, 40, 32, 25, 19, 14, 10, 7, 5}
var robotQueueCapacities = []int{0, 3, 5, 8, 12, 18, 30}
var robotInterceptPercents = []int{0, 45, 55, 64, 72, 80, 87, 94, 100}

func RobotDurationSeconds(level uint8) int {
	return robotValue(robotSpeedSeconds, level)
}

func RobotQueueCapacity(level uint8) int {
	return robotValue(robotQueueCapacities, level)
}

func RobotInterceptPercent(level uint8) int {
	return robotValue(robotInterceptPercents, level)
}

func robotValue(values []int, level uint8) int {
	if level == 0 {
		return 0
	}
	if int(level) >= len(values) {
		return values[len(values)-1]
	}
	return values[level]
}
