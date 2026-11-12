package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class CollapsingTitleTest {
    @Test
    fun `title has two stable endpoints`() {
        assertEquals(0f, collapsingTitleProgress(0, 200), 0f)
        assertEquals(1f, collapsingTitleProgress(200, 200), 0f)
        assertEquals(0.5f, collapsingTitleProgress(100, 200), 0f)
    }

    @Test
    fun `scroll never moves title beyond either endpoint`() {
        assertEquals(0f, collapsingTitleProgress(-20, 200), 0f)
        assertEquals(1f, collapsingTitleProgress(400, 200), 0f)
        assertEquals(1f, collapsingTitleProgress(0, 0), 0f)
    }

    @Test
    fun `short page gets exactly one collapse distance of scroll room`() {
        // 视口 1000、收起距离 64 → 内容最小高 1064，可滚距离恰好等于收起距离。
        val minHeight = collapsingContentMinHeightPx(viewportPx = 1000, collapseDistancePx = 64)
        assertEquals(1064, minHeight)
        assertEquals(1f, collapsingTitleProgress(minHeight - 1000, 64), 0f)
    }

    @Test
    fun `unknown viewport or zero distance adds no minimum`() {
        // 无界约束 / 退化收起距离时不补空白，否则会凭空拉出一整屏空白。
        assertEquals(0, collapsingContentMinHeightPx(0, 64))
        assertEquals(0, collapsingContentMinHeightPx(-1, 64))
        assertEquals(0, collapsingContentMinHeightPx(1000, 0))
    }
}
