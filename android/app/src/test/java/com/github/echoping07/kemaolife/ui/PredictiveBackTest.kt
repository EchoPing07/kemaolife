package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class PredictiveBackTest {

    @Test
    fun `predictive back endpoints match settled states`() {
        val start = PredictiveBackMath.predictiveBack(0f, PredictiveBackMath.EDGE_LEFT, 1080f)
        assertEquals(1f, start.childScale, 1e-4f)
        assertEquals(0f, start.childTranslationX, 1e-4f)
        assertEquals(0f, start.childCornerDp, 1e-4f)
        assertEquals(1f, start.childAlpha, 1e-4f)
        assertEquals(0.95f, start.rootScale, 1e-4f)
        assertEquals(0.20f, start.rootScrimAlpha, 1e-4f)

        val end = PredictiveBackMath.predictiveBack(1f, PredictiveBackMath.EDGE_LEFT, 1080f)
        assertEquals(0.88f, end.childScale, 1e-4f)
        assertEquals(28f, end.childCornerDp, 1e-4f)
        assertEquals(0f, end.childAlpha, 1e-4f)
        assertEquals(1f, end.rootScale, 1e-4f)
        assertEquals(0f, end.rootScrimAlpha, 1e-4f)
    }

    @Test
    fun `swipe edge flips horizontal shift direction`() {
        val fromLeft = PredictiveBackMath.predictiveBack(0.5f, PredictiveBackMath.EDGE_LEFT, 1000f)
        val fromRight = PredictiveBackMath.predictiveBack(0.5f, PredictiveBackMath.EDGE_RIGHT, 1000f)
        assertTrue(fromLeft.childTranslationX > 0f)
        assertEquals(-fromLeft.childTranslationX, fromRight.childTranslationX, 1e-4f)
        assertEquals(1f, fromLeft.childAlpha, 1e-4f)
    }

    @Test
    fun `slide transition places child offscreen at zero and full on screen at one`() {
        val atRoot = PredictiveBackMath.slideTransition(0f, 1000f)
        assertEquals(1000f, atRoot.childTranslationX, 1e-4f)
        assertEquals(0f, atRoot.rootTranslationX, 1e-4f)
        assertEquals(1f, atRoot.rootScale, 1e-4f)
        assertEquals(0f, atRoot.rootScrimAlpha, 1e-4f)

        val atChild = PredictiveBackMath.slideTransition(1f, 1000f)
        assertEquals(0f, atChild.childTranslationX, 1e-4f)
        assertEquals(-200f, atChild.rootTranslationX, 1e-4f)
        assertEquals(0.94f, atChild.rootScale, 1e-4f)
        assertEquals(0.16f, atChild.rootScrimAlpha, 1e-4f)
    }

    @Test
    fun `bottom bar reveal stays continuous across push, predictive drag and settle`() {
        assertEquals(1f, PredictiveBackMath.bottomBarReveal(false, 0f, false, 0f), 1e-4f)
        assertEquals(0.4f, PredictiveBackMath.bottomBarReveal(true, 0.6f, false, 0f), 1e-4f)
        assertEquals(0f, PredictiveBackMath.bottomBarReveal(true, 1f, false, 0f), 1e-4f)
        assertEquals(0f, PredictiveBackMath.bottomBarReveal(true, 1f, true, 0f), 1e-4f)
        assertEquals(0.5f, PredictiveBackMath.bottomBarReveal(true, 1f, true, 0.5f), 1e-4f)
        assertEquals(1f, PredictiveBackMath.bottomBarReveal(true, 1f, true, 1f), 1e-4f)
    }

    @Test
    fun `out of range progress values are clamped`() {
        val negative = PredictiveBackMath.predictiveBack(-0.5f, PredictiveBackMath.EDGE_LEFT, -100f)
        assertEquals(1f, negative.childScale, 1e-4f)
        assertEquals(0f, negative.childTranslationX, 1e-4f)

        val overflow = PredictiveBackMath.bottomBarReveal(true, 2f, true, 1.8f)
        assertEquals(1f, overflow, 1e-4f)
    }
}
