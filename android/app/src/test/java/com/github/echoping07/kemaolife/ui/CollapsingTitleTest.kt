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
}
