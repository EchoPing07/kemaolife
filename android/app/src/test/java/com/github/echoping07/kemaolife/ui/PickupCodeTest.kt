package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PickupCodeTest {
    @Test
    fun `attempts follow the documented fallback order`() {
        for (destination in PickupDestination.entries) {
            val attempts = pickupAttempts(destination)
            assertEquals(destination.appUris.size + 2, attempts.size)
            // 私有 scheme 候选按序钉住目标包名。
            destination.appUris.forEachIndexed { index, uri ->
                assertEquals(PickupAttempt(uri, destination.packageName), attempts[index])
            }
            // 官方 H5 先钉包名（App Links），最后不钉包名交系统浏览器。
            assertEquals(
                PickupAttempt(destination.webUri, destination.packageName),
                attempts[destination.appUris.size],
            )
            assertEquals(PickupAttempt(destination.webUri, null), attempts.last())
            assertNull(attempts.last().packageName)
        }
    }

    @Test
    fun `pinduoduo candidates degrade from full params to bare route`() {
        val uris = PickupDestination.PINDUODUO.appUris
        assertEquals(3, uris.size)
        assertTrue(uris[0].contains("tab=ID_CODE"))
        assertTrue(uris[0].contains("refer_page_sn=10169"))
        assertTrue(uris[1].contains("tab=ID_CODE"))
        assertFalse(uris[1].contains("refer_page_name"))
        assertEquals("pinduoduo://com.xunmeng.pinduoduo/mdkd/package", uris[2])
        assertTrue(uris.all { it.startsWith("pinduoduo://com.xunmeng.pinduoduo/") })
    }

    @Test
    fun `links carry only public facts, no local data`() {
        for (destination in PickupDestination.entries) {
            val all = destination.appUris + destination.webUri
            assertTrue(all.all { it.startsWith("https://") || it.contains("://") })
            assertTrue(destination.webUri.startsWith("https://"))
            // 不拼学号 / 快照 / 日志等本机数据（P1 / P2）。
            assertTrue(all.none { it.contains("studentId") || it.contains("roomName") || it.contains("cookie") })
        }
    }
}