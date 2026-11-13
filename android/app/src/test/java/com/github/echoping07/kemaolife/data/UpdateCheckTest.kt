package com.github.echoping07.kemaolife.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** 版本号解析与三段数值比较；不联网、不依赖 Android framework。 */
class UpdateCheckTest {

    @Test
    fun parsesTagsAndVersionNames() {
        assertEquals(UpdateCheck.Version(0, 5, 2), UpdateCheck.parseVersion("0.5.2"))
        assertEquals(UpdateCheck.Version(0, 5, 2), UpdateCheck.parseVersion("v0.5.2"))
        assertEquals("0.5.2", UpdateCheck.parseVersion("v0.5.2").toString())
    }

    @Test
    fun comparesNumericSegmentsNotStrings() {
        val older = UpdateCheck.parseVersion("v0.9.0")!!
        val newer = UpdateCheck.parseVersion("v0.10.0")!!
        assertTrue(newer > older)
        assertTrue(UpdateCheck.parseVersion("1.0.0")!! > newer)
        assertTrue(UpdateCheck.parseVersion("0.10.1")!! > newer)
        assertTrue(UpdateCheck.parseVersion("0.5.2")!! == UpdateCheck.parseVersion("v0.5.2")!!)
    }

    @Test
    fun rejectsUntrustedShapes() {
        listOf(
            "",
            "v1.2",
            "1.2.3.4",
            "1.2.3-beta",
            "1.2.3; DROP",
            "v1.2.3<script>",
            Int.MAX_VALUE.toString() + ".0.0.0",
            "99999999999999999999.0.0",
        ).forEach { assertNull("不应接受：$it", UpdateCheck.parseVersion(it)) }
    }

    @Test
    fun equalOrOlderRemoteCountsAsUpToDate() {
        val local = UpdateCheck.parseVersion("0.5.2")!!
        listOf("v0.5.2", "v0.5.1", "v0.4.9").forEach { tag ->
            assertTrue("$tag 不应算新版本", UpdateCheck.parseVersion(tag)!! <= local)
        }
        assertTrue(UpdateCheck.parseVersion("v0.5.3")!! > local)
    }
}
