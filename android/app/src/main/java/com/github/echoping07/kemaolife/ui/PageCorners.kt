package com.github.echoping07.kemaolife.ui

import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * 更多页与设置页共用的圆角层级：外层卡片同半径，卡内嵌套块按层级递减。
 *
 * 统一放在这里而不是每个 Composable 各写一个数字，避免两页分别调圆角后逐渐漂移；
 * 课表页仍用 `MaterialTheme.shapes`，本次不跟随放大。
 */
internal object PageCorners {
    /** 主卡片、独立分组卡片、弹窗与底部弹层。 */
    val Card: Dp = 36.dp

    /** 窄缝分组首项 / 末项的外角，与 [Card] 对齐。 */
    val GroupOuter: Dp = 36.dp

    /** 窄缝分组的中间项与相邻内角：保留可辨的独立条目轮廓。 */
    val GroupInner: Dp = 12.dp

    /** 卡片内的指标块、展开区与详情块。 */
    val Nested: Dp = 28.dp

    /** 输入框、开关行与 44dp 高的按钮（恰好成连续曲率胶囊）。 */
    val Field: Dp = 22.dp

    /** 关于页 104dp 方形图标。 */
    val Logo: Dp = 40.dp
}
