# 刮奖桌面概念图 v2 编辑提示词

> 生成方式：内置图像生成工具
> 用例分类：`precise-object-edit`

## PC 横屏 v2

```text
Use case: precise-object-edit
Asset type: desktop H5 game UI concept revision
Input images: Image 1 is the edit target, the clean PC desk UI. Image 2 is a markup reference only: its green rectangle marks the new fan location and its red arrow explains the required airflow direction.
Primary request: Move the old-brass electric fan from the lower-right area of Image 1 into the upper-left green-marked empty area, between the purchase tray and the brass redemption slot. Scale the fan down naturally to fit that space. Rotate and aim the fan diagonally from upper-left toward lower-right, so it physically blows cards across the desk toward the robot and the trash can.
Required changes: completely remove the original lower-right fan and its base, restoring continuous realistic walnut wood grain and lighting where it stood. Place exactly one fan in the new upper-left location. Keep a compact readable brass hold control associated with the fan, preferably labeled "按住风扇". A few extremely subtle translucent airflow streaks may point diagonally toward the lower-right, but do not draw a large arrow.
Invariants: preserve the exact canvas size, camera angle, top navigation, purchase tray, redemption slot, fixed card-slot drawer, all scratch cards and their positions, selected-card toolbar, robot, desk lamp, books, desk edge, chair, trash can, lighting, materials, and overall UI style from Image 1. Change only the fan position/orientation and repair its old location.
Constraints: Image 2's green box, red box, red arrow and all markup must not appear in the final image. No extra fan, no probability text, no watermark, no new objects, no redesign of the rest of the interface.
```

## 手机竖屏 v2

```text
Use case: precise-object-edit
Asset type: portrait mobile H5 game UI concept revision
Input images: Image 1 is the edit target.
Primary request: Move the old-brass fan and its "按住风扇" control from the lower-right of the desk to the upper-left portion of the desk, immediately below the top status bar and beside the purchase tray, aimed diagonally from upper-left toward lower-right. It must visibly blow cards toward the lower-right robot and trash can. Scale the fan compactly for mobile while preserving a usable hold control.
Required changes: completely remove the original lower-right fan and base, restoring continuous walnut wood grain where it stood. Place exactly one compact fan in the upper-left functional area. Add only a few subtle translucent airflow streaks traveling diagonally down-right; no large arrow.
Invariants: keep the exact portrait canvas size, camera angle, top bar, purchase tray, central redemption area, card-slot tab, all scratch cards and their positions, selected-card outline, robot, trash can, bottom action bar, lighting, materials and overall UI style. Do not move the robot or trash can. Make only the minimum spacing adjustment needed around the upper-left fan.
Constraints: no second fan, no probability text, no watermark, no new objects, no redesign of the rest of the interface.
```
