# 道具商店概念图 v1 生图提示词

> 生成方式：Codex 内置 ImageGen。参考图仅用于继承主桌面的胡桃木、深绿色皮革、黄铜和暖金灯光风格。概念图用于确认布局与视觉方向，图中局部文字、等级及价格不能替代 `docs/product-framework-v0.1.md` 和服务端配置。

## PC 端

```text
Use case: ui-mockup
Asset type: PC H5 game item shop concept
Primary request: create a high-fidelity, practical in-game item shop opened as a large overlay above the existing scratch-card desk. Use Image 1 only as the visual style reference for the warm walnut desk, dark green leather, brass trim, warm gold lighting, and premium tactile game UI.
Style/medium: realistic shippable game UI mockup, not concept art
Composition/framing: 16:9 desktop layout. A centered wide shop panel uses about 80% of the screen. Background desk softly blurred. Top bar: title “道具商店”; three tabs “好运”, “效率”, “整理与安全”; balance “金币 128,560”; close button. Main left area has a three-column item card grid. Right detail panel shows selected item “好运”, “7级 → 8级”, “53% → 68%”, price “45,000”, current balance and after-purchase balance. Primary button “升级”. Include item cards for 好运, 刮奖范围, 机器人, 风扇, 固定卡槽, 垃圾桶.
Visual hierarchy: clear readable Chinese, generous spacing, large buttons, real product UI alignment. Warm gold price numbers, subdued warning color only where needed.
Constraints: render the listed Chinese labels verbatim where possible; virtual coins only; no recharge, no RMB, no discount badge, no countdown, no loot box, no casino neon, no watermark. Keep interface practical and uncluttered.
```

## 移动端

```text
Use case: ui-mockup
Asset type: mobile H5 game item shop concept
Primary request: create a high-fidelity, practical mobile item shop matching the existing scratch-card desk game. Use Image 1 only as the visual style reference for warm walnut, dark green leather, brass trim, warm gold lighting, and premium tactile game UI.
Style/medium: realistic shippable mobile game UI mockup, not concept art
Composition/framing: portrait phone screen. Top bar with back arrow, title “道具商店”, balance “金币 12,500”. Sticky tabs “好运”, “效率”, “整理与安全”. A single-column scrollable list of large item cards for 好运, 刮奖范围, 机器人, 风扇, 固定卡槽, 垃圾桶. Selected 好运 card shows “7级 → 8级”, “53% → 68%”, price “45,000”. A bottom sheet is open with current effect, next effect, purchase price, after-purchase balance, and a disabled “金币不足” button.
Visual hierarchy: readable Chinese, 44px minimum touch targets, clear cards, generous margins, modern mobile information hierarchy.
Constraints: render listed Chinese labels verbatim where possible; virtual coins only; no recharge, no RMB, no discounts, no countdown, no loot box, no casino neon, no watermark. Practical uncluttered UI.
```
