package com.github.echoping07.kemaolife.ui.theme

import androidx.compose.foundation.shape.CornerBasedShape
import androidx.compose.foundation.shape.CornerSize
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Shapes
import androidx.compose.runtime.Composable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.data.DEFAULT_CONTINUOUS_CORNERS

/**
 * 圆角曲率的主题级开关。默认与偏好缺省值一致：标准圆弧。
 *
 * 为什么用 CompositionLocal 而不是逐层传参：形状散落在十几个 Composable 里，
 * 逐个加参数会把「显示偏好」渗进业务无关的组件签名；形状只在组合期读取，
 * 静态 Local 的读取点在开关变化时重组，不整棵子树失效。
 */
internal val LocalCornerStyle = staticCompositionLocalOf { DEFAULT_CONTINUOUS_CORNERS }

/**
 * 全 App 唯一的形状入口：半径在两种模式下完全一致，只换曲率算法。
 *
 * 关闭时走 Compose 的 `RoundedCornerShape`（标准圆弧），开启时走
 * [SmoothCornerShape]（超椭圆，接缝处曲率收敛到零）。`CircleShape` 不经过这里，
 * 因为圆形在两种模式下没有差异。
 */
@Composable
internal fun appShape(radius: Dp): CornerBasedShape =
    shapeFor(radius, radius, radius, radius)

/** 上下不同圆角（分组卡片的首行 / 末行）。 */
@Composable
internal fun appShape(top: Dp, bottom: Dp): CornerBasedShape =
    shapeFor(top, top, bottom, bottom)

/** 四角独立圆角（连体钮的首段 / 末段：外侧最大、内侧收窄）。 */
@Composable
internal fun appShape(
    topStart: Dp,
    topEnd: Dp,
    bottomEnd: Dp,
    bottomStart: Dp,
): CornerBasedShape = shapeFor(topStart, topEnd, bottomEnd, bottomStart)

@Composable
private fun shapeFor(
    topStart: Dp,
    topEnd: Dp,
    bottomEnd: Dp,
    bottomStart: Dp,
): CornerBasedShape {
    val corners = arrayOf(
        CornerSize(topStart),
        CornerSize(topEnd),
        CornerSize(bottomEnd),
        CornerSize(bottomStart),
    )
    return if (LocalCornerStyle.current) {
        SmoothCornerShape(corners[0], corners[1], corners[2], corners[3])
    } else {
        RoundedCornerShape(corners[0], corners[1], corners[2], corners[3])
    }
}

/**
 * `MaterialTheme.shapes` 的五个槽位是 `CornerBasedShape` 常量，不能在组合期读
 * [LocalCornerStyle]，所以由 [KemaolifeTheme] 按开关整体选择一套。
 *
 * 两套半径逐项相同：切换只影响曲率，不影响布局与裁切范围（docs/002 §5.3 圆角层级）。
 */
internal val SmoothShapes = Shapes(
    extraSmall = SmoothCornerShape(8.dp),
    small = SmoothCornerShape(12.dp),
    medium = SmoothCornerShape(16.dp),
    large = SmoothCornerShape(24.dp),
    extraLarge = SmoothCornerShape(28.dp),
)

internal val RoundedShapes = Shapes(
    extraSmall = RoundedCornerShape(8.dp),
    small = RoundedCornerShape(12.dp),
    medium = RoundedCornerShape(16.dp),
    large = RoundedCornerShape(24.dp),
    extraLarge = RoundedCornerShape(28.dp),
)
