"""刮刮乐首版金币经济复算脚本。

数据来源：docs/game-01..07-draft.md、docs/daily-wheel-v0.1.md。
用途：检查每日免费产出、升级总价和满级好运的理论回收速度。
不模拟用户行为、方差或留存；这里只做理论均值边界。
"""

CARD_NAMES = [
    "零钱小票",
    "街角杂货铺",
    "街机挑战券",
    "黄金矿洞",
    "火箭发射",
    "深海打捞",
    "永恒彩钻",
]
CARD_PRICES = [50, 1500, 5000, 15000, 40000, 80000, 150000]
BASE_EV = [37.8, 1032, 3630, 10545, 28800, 55040, 106950]
MAX_LUCK_EV = [47.145, 1386, 4548, 13530, 36440, 73600, 137625]
WHEEL_WEIGHTS = [30, 22, 16, 12, 8, 6]
DAILY_FIXED_COINS = 125

PROPOSED_PRICES = {
    "好运 1-10": [300, 700, 1500, 3000, 6000, 12000, 24000, 45000, 80000, 130000],
    "刮奖范围 2-10": [100, 250, 500, 1000, 2200, 4800, 9500, 18000, 35000],
    "机器人本体": [1000],
    "机器人速度 2-8": [500, 1000, 2200, 4500, 9000, 18000, 36000],
    "机器人队列 2-6": [300, 700, 1600, 4000, 10000],
    "机器人拦截 2-8": [600, 1200, 2500, 5000, 10000, 22000, 45000],
    "风扇 1-8": [500, 300, 700, 1500, 3000, 6000, 12000, 24000],
    "垃圾桶": [100],
    "固定卡槽": [500],
}


def weighted_average(values, weights):
    return sum(value * weight for value, weight in zip(values, weights)) / sum(weights)


def daily_output_by_band(values):
    rows = []
    for count in range(1, 7):
        wheel_ev = weighted_average(values[:count], WHEEL_WEIGHTS[:count])
        rows.append((count, CARD_PRICES[count - 1], wheel_ev, DAILY_FIXED_COINS + wheel_ev))
    return rows


def main():
    print("每日免费产出（基础好运）")
    for count, threshold, wheel_ev, total in daily_output_by_band(BASE_EV):
        print(f"解锁前 {count} 款 / 余额门槛 {threshold:>6}: 转盘EV={wheel_ev:>8.2f}, 每日总EV={total:>8.2f}")

    print("\n每日免费产出（满级好运）")
    for count, threshold, wheel_ev, total in daily_output_by_band(MAX_LUCK_EV):
        print(f"解锁前 {count} 款 / 余额门槛 {threshold:>6}: 转盘EV={wheel_ev:>8.2f}, 每日总EV={total:>8.2f}")

    print("\n各系统建议总价")
    grand_total = 0
    for name, prices in PROPOSED_PRICES.items():
        subtotal = sum(prices)
        grand_total += subtotal
        print(f"{name:<18} {subtotal:>8}")
    print(f"全部永久道具合计       {grand_total:>8}")

    luck_total = sum(PROPOSED_PRICES["好运 1-10"])
    print("\n满级好运理论回收速度（未考虑资金门槛和结果方差）")
    for name, base_ev, max_ev in zip(CARD_NAMES, BASE_EV, MAX_LUCK_EV):
        gain = max_ev - base_ev
        purchases = luck_total / gain
        print(f"{name:<8}: 单张EV提升={gain:>9.3f}, 理论回收={purchases:>9.2f} 张")


if __name__ == "__main__":
    main()
