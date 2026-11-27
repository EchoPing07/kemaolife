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
import kotlin.math.min

/**
 * 超椭圆圆角：曲率在与直边相接处平缓过渡，避免普通圆弧的接缝感。
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

    constructor(radius: Dp) : this(CornerSize(radius))

    private constructor(corner: CornerSize) : this(corner, corner, corner, corner)

    override fun createOutline(
        size: Size,
        topStart: Float,
        topEnd: Float,
        bottomEnd: Float,
        bottomStart: Float,
        layoutDirection: LayoutDirection,
    ): Outline {
        val w = size.width
        val h = size.height
        if (w <= 0f || h <= 0f) {
            return Outline.Rectangle(Rect(0f, 0f, w, h))
        }
        val maxR = min(w, h) / 2f
        val isLtr = layoutDirection == LayoutDirection.Ltr
        val tl = (if (isLtr) topStart else topEnd).coerceIn(0f, maxR)
        val tr = (if (isLtr) topEnd else topStart).coerceIn(0f, maxR)
        val br = (if (isLtr) bottomEnd else bottomStart).coerceIn(0f, maxR)
        val bl = (if (isLtr) bottomStart else bottomEnd).coerceIn(0f, maxR)
        if (tl == 0f && tr == 0f && br == 0f && bl == 0f) {
            return Outline.Rectangle(Rect(0f, 0f, w, h))
        }
        // 单段三次贝塞尔逼近 n=4 超椭圆（无三角 / 幂运算）：
        // 控制点距角顶点 0.091 * r，对角线中点精确落在 (1 - 2^-0.25) * r，
        // 与直边接缝处曲率较标准圆弧降低一个数量级，且每角严格位于 [0, r] 内不重叠。
        val k = SQUIRCLE_CONTROL_RATIO
        val path = Path().apply {
            moveTo(0f, tl)
            if (tl > 0f) {
                val c = tl * k
                cubicTo(0f, c, c, 0f, tl, 0f)
            }
            lineTo(w - tr, 0f)
            if (tr > 0f) {
                val c = tr * k
                cubicTo(w - c, 0f, w, c, w, tr)
            }
            lineTo(w, h - br)
            if (br > 0f) {
                val c = br * k
                cubicTo(w, h - c, w - c, h, w - br, h)
            }
            lineTo(bl, h)
            if (bl > 0f) {
                val c = bl * k
                cubicTo(c, h, 0f, h - c, 0f, h - bl)
            }
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

    private companion object {
        // n=4 超椭圆三次贝塞尔控制点比例：(8 * (1 - 2^-0.25) - 1) / 3 ≈ 0.090943
        const val SQUIRCLE_CONTROL_RATIO = 0.091f
    }
}
