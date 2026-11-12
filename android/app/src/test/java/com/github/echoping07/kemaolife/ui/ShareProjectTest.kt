package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class ShareProjectTest {
    @Test
    fun `share text is exactly the public project card`() {
        assertEquals(
            "KeMao Life · 科贸生活\n项目地址：https://github.com/EchoPing07/kemaolife",
            shareProjectText("科贸生活"),
        )
    }

    @Test
    fun `share text has only the two public lines`() {
        val lines = shareProjectText("应用名").lines()
        assertEquals(2, lines.size)
        assertEquals("KeMao Life · 应用名", lines[0])
        assertEquals("项目地址：https://github.com/EchoPing07/kemaolife", lines[1])
    }
}
