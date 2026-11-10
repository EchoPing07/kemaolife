package com.github.echoping07.kemaolife.ui.theme

import androidx.compose.foundation.shape.CornerBasedShape
import androidx.compose.foundation.shape.CornerSize
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Outline
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.LayoutDirection
import kotlin.math.abs
import kotlin.math.cos
import kotlin.math.min
import kotlin.math.pow
import kotlin.math.sign
import kotlin.math.sin

/**
 * 超椭圆圆角：曲率在与直边相接处收敛到零，避免普通圆弧的接缝感。
 * 路径仅负责视觉形状，不需要 Android 12+ 的 RoundedCorner 平台 API。
 *
 * 必须继承 [CornerBasedShape] 而非直接实现 [Shape] —— `MaterialTheme.shapes` 的
 * 五个槽位声明为 `CornerBasedShape`，只实现 `Shape` 无法赋给 `Shapes(...)`。
 */
internal class SmoothCornerShape(
    topStart: CornerSize,
    topEnd: CornerSize,
    bottomEnd: CornerSize,
    bottomStart: CornerSize,
) : CornerBasedShape(topStart, topEnd, bottomEnd, bottomStart) {

    constructor(radius: Dp) : this(
        CornerSize(radius), CornerSize(radius), CornerSize(radius), CornerSize(radius),
    )

    override fun createOutline(
        size: Size,
        topStart: Float,
        topEnd: Float,
        bottomEnd: Float,
        bottomStart: Float,
        layoutDirection: LayoutDirection,
    ): Outline {
        val maxR = min(size.width, size.height) / 2f
        val r1 = topStart.coerceIn(0f, maxR)
        val r2 = topEnd.coerceIn(0f, maxR)
        val r3 = bottomEnd.coerceIn(0f, maxR)
        val r4 = bottomStart.coerceIn(0f, maxR)
        if (r1 == 0f && r2 == 0f && r3 == 0f && r4 == 0f) {
            return Outline.Rectangle(Rect(0f, 0f, size.width, size.height))
        }
        // 象限路径逐点生成，避免 arc/corner 的相对坐标在小尺寸下退化。
        val path = Path().apply {
            corner(r1, r1, 180f, 270f, r1)
            lineTo(size.width - r2, 0f)
            corner(size.width - r2, r2, 270f, 360f, r2)
            lineTo(size.width, size.height - r3)
            corner(size.width - r3, size.height - r3, 0f, 90f, r3)
            lineTo(r4, size.height)
            corner(r4, size.height - r4, 90f, 180f, r4)
            close()
        }
        return Outline.Generic(path)
    }

    override fun copy(
        topStart: CornerSize,
        topEnd: CornerSize,
        bottomEnd: CornerSize,
        bottomStart: CornerSize,
    ): CornerBasedShape = SmoothCornerShape(topStart, topEnd, bottomEnd, bottomStart)

    private fun Path.corner(cx: Float, cy: Float, start: Float, end: Float, radius: Float) {
        if (radius == 0f) {
            lineTo(cx, cy)
            return
        }
        // n=4 的 squircle；分段足够密，填充轮廓在屏幕上连续平滑。
        val steps = 16
        for (step in 0..steps) {
            val angle = Math.toRadians((start + (end - start) * step / steps).toDouble())
            val c = cos(angle)
            val s = sin(angle)
            val x = cx + radius * sign(c) * abs(c).pow(0.5)
            val y = cy + radius * sign(s) * abs(s).pow(0.5)
            lineTo(x.toFloat(), y.toFloat())
        }
    }

    override fun equals(other: Any?): Boolean =
        other is SmoothCornerShape &&
            topStart == other.topStart && topEnd == other.topEnd &&
            bottomEnd == other.bottomEnd && bottomStart == other.bottomStart

    override fun hashCode(): Int {
        var result = topStart.hashCode()
        result = 31 * result + topEnd.hashCode()
        result = 31 * result + bottomEnd.hashCode()
        result = 31 * result + bottomStart.hashCode()
        return result
    }
}

internal fun smoothShape(radius: Dp): CornerBasedShape = SmoothCornerShape(radius)

/** 上下不同圆角（分组卡片的首行 / 末行）。 */
internal fun smoothShape(top: Dp, bottom: Dp): CornerBasedShape = SmoothCornerShape(
    CornerSize(top),
    CornerSize(top),
    CornerSize(bottom),
    CornerSize(bottom),
)
