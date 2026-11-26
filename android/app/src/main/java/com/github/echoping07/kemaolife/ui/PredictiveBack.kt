package com.github.echoping07.kemaolife.ui

import kotlin.math.abs

/**
 * 预测性返回与二级页推入 / 退出的纯几何换算。
 *
 * 抽成纯函数（不依赖 Android Framework），以便在 JVM 单元测试中直接覆盖端点、
 * 单调性与左右边缘方向，避免手势拖动过程中出现跳变或裁切错位。
 */
internal data class PageLayerTransform(
    val rootTranslationX: Float,
    val rootScale: Float,
    val rootScrimAlpha: Float,
    val childTranslationX: Float,
    val childScale: Float,
    val childAlpha: Float,
    val childCornerDp: Float,
)

internal object PredictiveBackMath {
    const val EDGE_LEFT = 0
    const val EDGE_RIGHT = 1

    /**
     * 预测性返回手势（`progress` ∈ [0, 1]）：
     * - 顶层子页按 Material 3 规范轻微缩小（1.0 → 0.88）、顺手势边缘偏移并裁出大圆角；
     * - 底层总览从 0.95 放大至 1.0，遮罩同步淡出；
     * - 松手确认后进入收尾段（progress > 0.85），顶层平滑淡出避免硬消失。
     */
    fun predictiveBack(
        progress: Float,
        swipeEdge: Int,
        viewportWidthPx: Float,
    ): PageLayerTransform {
        val p = progress.coerceIn(0f, 1f)
        val width = viewportWidthPx.coerceAtLeast(0f)
        val direction = if (swipeEdge == EDGE_RIGHT) -1f else 1f
        val fadeOutAlpha =
            if (p <= 0.85f) {
                1f
            } else {
                ((1f - p) / 0.15f).coerceIn(0f, 1f)
            }
        return PageLayerTransform(
            rootTranslationX = 0f,
            rootScale = lerp(0.95f, 1f, p),
            rootScrimAlpha = 0.20f * (1f - p),
            childTranslationX = width * 0.08f * p * direction,
            childScale = lerp(1f, 0.88f, p),
            childAlpha = fadeOutAlpha,
            childCornerDp = 28f * p,
        )
    }

    /**
     * 非手势转场（点击进子页 / 点击左上角返回 / 无手势进度的系统返回键）：
     * `childProgress` = 0 表示完全在总览，1 表示完全在子页。
     */
    fun slideTransition(
        childProgress: Float,
        viewportWidthPx: Float,
    ): PageLayerTransform {
        val t = childProgress.coerceIn(0f, 1f)
        val width = viewportWidthPx.coerceAtLeast(0f)
        return PageLayerTransform(
            rootTranslationX = -width * 0.20f * t,
            rootScale = lerp(1f, 0.94f, t),
            rootScrimAlpha = 0.16f * t,
            childTranslationX = width * (1f - t),
            childScale = 1f,
            childAlpha = 1f,
            childCornerDp = 0f,
        )
    }

    /**
     * 设置页驱动悬浮底栏显隐进度（0 = 完全隐藏，1 = 完全显示）。
     *
     * 进入二级页时随 `transitionProgress` 同步隐藏；在二级页拖拽预测性返回时，
     * 随手势进度 `predictiveBackProgress` 实时渐现，保证确认或取消时均无跳变。
     */
    fun bottomBarReveal(
        hasChild: Boolean,
        transitionProgress: Float,
        predictiveBackActive: Boolean,
        predictiveBackProgress: Float,
    ): Float {
        if (!hasChild) return 1f
        val raw =
            if (predictiveBackActive) {
                predictiveBackProgress
            } else {
                1f - transitionProgress
            }
        return raw.coerceIn(0f, 1f)
    }

    /** 避免浮点尾差导致判断失效。 */
    internal fun isNearlyEqual(a: Float, b: Float, eps: Float = 1e-4f): Boolean = abs(a - b) <= eps

    private fun lerp(start: Float, stop: Float, fraction: Float): Float =
        start + (stop - start) * fraction
}
