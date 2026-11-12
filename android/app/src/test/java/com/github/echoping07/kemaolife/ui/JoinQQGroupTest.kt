package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertTrue
import org.junit.Test

class JoinQQGroupTest {
    @Test
    fun `group card uri uses only the public group number`() {
        val uri = qqGroupCardUri(QQ_GROUP_NUMBER)
        assertTrue(uri.startsWith("mqqapi://card/show_pslcard?"))
        assertTrue(uri.contains("card_type=group"))
        assertTrue(uri.contains("uin=1126474633"))
        // 卡片 scheme 只带公开群号，不应拼入任何 authKey / token。
        assertTrue(!uri.contains("authKey"))
        assertTrue(!uri.contains("token"))
    }

    @Test
    fun `fallback share url points at the same public group`() {
        assertTrue(QQ_GROUP_SHARE_URL.startsWith("https://qm.qq.com/q/"))
        assertTrue(QQ_GROUP_NUMBER == "1126474633")
    }
}
