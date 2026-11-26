package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class FloatingNavIndicatorTest {

    @Test
    fun `resting pill aligns exactly to each equal tab slot`() {
        // 3 个 100px 槽位 + 2 个 10px 缝隙 = 320px 总宽。
        val total = 320f
        val gap = 10f

        val tab0 = FloatingNavIndicatorMath.bounds(0f, 0f, itemCount = 3, totalWidthPx = total, gapPx = gap)
        assertEquals(0f, tab0.leftPx, 1e-4f)
        assertEquals(100f, tab0.rightPx, 1e-4f)
        assertEquals(100f, tab0.widthPx, 1e-4f)

        val tab1 = FloatingNavIndicatorMath.bounds(1f, 1f, itemCount = 3, totalWidthPx = total, gapPx = gap)
        assertEquals(110f, tab1.leftPx, 1e-4f)
        assertEquals(210f, tab1.rightPx, 1e-4f)

        val tab2 = FloatingNavIndicatorMath.bounds(2f, 2f, itemCount = 3, totalWidthPx = total, gapPx = gap)
        assertEquals(220f, tab2.leftPx, 1e-4f)
        assertEquals(320f, tab2.rightPx, 1e-4f)
    }

    @Test
    fun `mid-flight stretch expands smoothly across intermediate tabs`() {
        val stretched = FloatingNavIndicatorMath.bounds(
            leftSlot = 0.4f,
            rightSlot = 1.6f,
            itemCount = 3,
            totalWidthPx = 320f,
            gapPx = 10f,
        )
        assertEquals(44f, stretched.leftPx, 1e-4f)
        assertEquals(276f, stretched.rightPx, 1e-4f)
        assertTrue(stretched.widthPx > 100f)
    }

    @Test
    fun `reversed slot ordering during rapid retarget never inverts or collapses pill`() {
        val normal = FloatingNavIndicatorMath.bounds(0.3f, 1.4f, itemCount = 3, totalWidthPx = 320f, gapPx = 10f)
        val reversed = FloatingNavIndicatorMath.bounds(1.4f, 0.3f, itemCount = 3, totalWidthPx = 320f, gapPx = 10f)
        assertEquals(normal.leftPx, reversed.leftPx, 1e-4f)
        assertEquals(normal.rightPx, reversed.rightPx, 1e-4f)
        assertTrue(reversed.widthPx >= 100f)
    }

    @Test
    fun `overshoot is bounded within container padding limit`() {
        val leftOver = FloatingNavIndicatorMath.bounds(
            leftSlot = -0.5f,
            rightSlot = 0f,
            itemCount = 3,
            totalWidthPx = 320f,
            gapPx = 10f,
            overshootLimitPx = 6f,
        )
        assertEquals(-6f, leftOver.leftPx, 1e-4f)
        assertEquals(100f, leftOver.rightPx, 1e-4f)

        val rightOver = FloatingNavIndicatorMath.bounds(
            leftSlot = 2f,
            rightSlot = 2.5f,
            itemCount = 3,
            totalWidthPx = 320f,
            gapPx = 10f,
            overshootLimitPx = 6f,
        )
        assertEquals(220f, rightOver.leftPx, 1e-4f)
        assertEquals(326f, rightOver.rightPx, 1e-4f)
    }

    @Test
    fun `rtl mirrors pill coordinates symmetrically`() {
        val ltr = FloatingNavIndicatorMath.bounds(0f, 0f, itemCount = 3, totalWidthPx = 320f, gapPx = 10f, isRtl = false)
        val rtl = FloatingNavIndicatorMath.bounds(0f, 0f, itemCount = 3, totalWidthPx = 320f, gapPx = 10f, isRtl = true)
        assertEquals(220f, rtl.leftPx, 1e-4f)
        assertEquals(320f, rtl.rightPx, 1e-4f)
        assertEquals(ltr.widthPx, rtl.widthPx, 1e-4f)
    }

    @Test
    fun `empty or zero-width container returns empty bounds`() {
        assertEquals(PillBounds(0f, 0f), FloatingNavIndicatorMath.bounds(0f, 0f, itemCount = 0, totalWidthPx = 320f, gapPx = 10f))
        assertEquals(PillBounds(0f, 0f), FloatingNavIndicatorMath.bounds(0f, 0f, itemCount = 3, totalWidthPx = 0f, gapPx = 10f))
    }
}
