# 每日免费卡转盘概念图提示词

> 生成方式：内置图像生成工具

## PC 横屏

```text
Use case: precise-object-edit
Asset type: high-fidelity desktop H5 game UI modal concept
Input images: Image 1 is the existing desktop game UI and must remain the background and style reference.
Primary request: Show the “daily free scratch-card wheel” opened from 今日任务. Add a large centered modal over Image 1 while preserving the complete desk behind it under a restrained 45% dark translucent overlay.
Modal style: practical shippable game UI, dark green leather panel, brushed brass frame, subtle walnut accents, matching the existing collector’s desk. No casino styling.
Composition: wide desktop screen. On the left two-thirds of the modal place one large readable mechanical prize wheel with a fixed pointer at top. The active wheel contains six visibly unequal colored sectors proportional in descending size to weights 30, 22, 16, 12, 8, 6. Each active sector has a simple distinct ticket icon and short Chinese card label. On the right place today status, current eligible-pool summary, a large primary button, and a smaller rules link. Along the bottom of the modal show two separate dark disabled ticket plaques for cards 7 and 8, visible but clearly outside the active wheel.
Exact active labels: "零钱小票", "街角杂货铺", "街机挑战券", "黄金矿洞", "火箭发射", "深海打捞"
Exact disabled labels: "永恒彩钻  不参与", "放手一博  不参与"
Other text (verbatim): "每日免费转盘", "今日 1/1", "免费转动", "转盘规则", "仅抽当前余额可购买的卡", "关闭"
UI state: existing balance is 128,560, therefore all first six cards are eligible and colorful; cards 7 and 8 remain permanently excluded. Do not show scratch-card internal win probabilities. A small rules panel may show the six selection weights as 30 / 22 / 16 / 12 / 8 / 6, not percentages.
Invariants: preserve the exact canvas, top bar, upper-left fan, redemption slot, purchase tray, card drawer, scratch cards, robot, lower-right trash can, warm lighting and camera angle behind the modal.
Constraints: no real money, casino chips, roulette aesthetics, slot machine, cash rain, jackpot slogans, false near-win indicators, extra prize categories, people, brand logos, trademarks, watermark, or dense illegible text.
```

## 手机竖屏

```text
Use case: precise-object-edit
Asset type: high-fidelity portrait mobile H5 game UI modal concept
Input images: Image 1 is the existing portrait mobile game UI and must remain the background and style reference.
Primary request: Show the daily free scratch-card wheel opened from 今日任务. Add a practical full-height mobile modal sheet over Image 1 while preserving the desk behind it under a restrained dark translucent overlay.
Modal style: dark green leather, brushed brass frame, subtle walnut trim, matching the existing collector’s desk; shippable touch UI, not casino styling.
Composition: vertical 9:16 app screen with safe margins. Header has title and close button. Upper-middle contains one large readable mechanical wheel with a fixed pointer. The active wheel contains six visibly unequal colored sectors proportional in descending size to weights 30, 22, 16, 12, 8, 6, each with a simple distinct ticket icon. Below the wheel show a concise eligible-pool row, today status, one large thumb-friendly primary button, and a rules link. At the bottom show two compact disabled ticket plaques for cards 7 and 8, visible but outside the active wheel.
Active card labels, keep short and readable: "零钱小票", "街角杂货铺", "街机挑战券", "黄金矿洞", "火箭发射", "深海打捞"
Disabled labels (verbatim): "永恒彩钻  不参与", "放手一博  不参与"
Other text (verbatim): "每日免费转盘", "今日 1/1", "免费转动", "转盘规则", "只抽当前余额可购买的卡"
UI state: existing balance is 1250, so only 零钱小票 is currently eligible; cards 2–6 remain visible on the wheel but display small lock symbols and their purchase thresholds, while cards 7–8 are permanently disabled with “不参与”. The pointer must be able to land only on the eligible card. Do not show scratch-card internal win probabilities.
Invariants: preserve the exact portrait canvas, top status bar, upper-left fan, redemption slot, purchase tray, card-slot tab, scratch cards, robot, lower-right trash can, bottom action bar, warm lighting, materials and camera angle behind the modal.
Constraints: no real money, casino chips, roulette styling, slot machine, cash rain, jackpot slogans, false near-win cues, extra prizes, people, logos, trademarks, watermark, or tiny dense text. Ensure all primary controls are at least visually 44px touch targets.
```

## 手机门槛文字校正

```text
Use case: text-localization
Asset type: portrait mobile H5 daily wheel UI correction
Primary request: Correct only the five locked purchase-threshold numbers printed inside the prize wheel.
Exact replacements: 街角杂货铺 1500; 街机挑战券 5000; 黄金矿洞 15000; 火箭发射 40000; 深海打捞 80000.
Constraints: change only these five numeric labels; preserve all other layout, text, icons, sectors and lighting; no probability text; no watermark.
```
