package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class JoinQQGroupTest {
    @Test
    fun `join uri keeps the official inner url, encoded once`() {
        val uri = qqGroupJoinUri("TESTKEY")
        assertEquals(
            "mqqopensdkapi://bizAgent/qm/qr?url=" +
                "http%3A%2F%2Fqm.qq.com%2Fcgi-bin%2Fqm%2Fqr%3Ffrom%3Dapp%26p%3Dandroid" +
                "%26jump_from%3Dwebapi%26k%3DTESTKEY",
            uri,
        )
        assertTrue(uri.startsWith("mqqopensdkapi://bizAgent/qm/qr?url="))
    }

    @Test
    fun `join uri keeps the key untouched and carries no local data`() {
        val uri = qqGroupJoinUri(QQ_GROUP_JOIN_KEY)
        // key 由腾讯生成，含 `_`，不应被二次编码或改写。
        assertTrue(uri.endsWith("k%3D$QQ_GROUP_JOIN_KEY"))
        // 只有公开群号与官方 key，不拼学号 / 快照 / 日志。
        assertTrue(!uri.contains("studentId"))
        assertTrue(!uri.contains("roomName"))
    }

    @Test
    fun `fallback share url points at the same public group`() {
        assertTrue(QQ_GROUP_SHARE_URL.startsWith("https://qm.qq.com/q/"))
    }
}
