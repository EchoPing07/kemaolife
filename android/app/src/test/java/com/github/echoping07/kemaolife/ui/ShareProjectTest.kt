package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class ShareProjectTest {
    @Test
    fun `share text is exactly the public project card`() {
        assertEquals(
            "KeMao Life · 科贸生活\n官网：https://echoping07.github.io/kemaolife/\n项目地址：https://github.com/EchoPing07/kemaolife",
            shareProjectText("科贸生活"),
        )
    }

    @Test
    fun `share text has only the three public lines`() {
        val lines = shareProjectText("应用名").lines()
        assertEquals(3, lines.size)
        assertEquals("KeMao Life · 应用名", lines[0])
        assertEquals("官网：https://echoping07.github.io/kemaolife/", lines[1])
        assertEquals("项目地址：https://github.com/EchoPing07/kemaolife", lines[2])
    }
}