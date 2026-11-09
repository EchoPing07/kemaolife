package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 启动时序（docs/002 §5.3「每次打开」）：
 *  - 生命周期补发的初始 ON_RESUME 不算一次打开；
 *  - Init 未就绪时到来的打开要挂起，Init 完成后只补跑一次；
 *  - Init 失败时保留挂起，不能永久丢掉那一轮无条件联网。
 */
class StartupTimingTest {

    @Test
    fun `Init 未就绪时的打开挂起，就绪后只消费一次`() {
        var runs = 0
        var pending = false

        assertTrue(StartupTiming.shouldDefer(ready = false))
        pending = true
        assertFalse(StartupTiming.consumePending(ready = false, pending = pending))
        assertEquals(0, runs)

        // 就绪后消费一次；后续重复 Init（设置页保存）不再多打一轮。
        if (StartupTiming.consumePending(ready = true, pending = pending)) {
            pending = false
            runs++
        }
        assertEquals(1, runs)
        assertFalse(pending)
        assertFalse(StartupTiming.consumePending(ready = true, pending = pending))
        assertEquals(1, runs)
    }

    @Test
    fun `Init 失败时挂起保留到下次成功`() {
        var runs = 0
        var pending = true

        assertFalse(StartupTiming.consumePending(ready = false, pending = pending))
        assertEquals(0, runs)
        assertTrue(pending)

        if (StartupTiming.consumePending(ready = true, pending = pending)) {
            pending = false
            runs++
        }
        assertEquals(1, runs)
        assertFalse(pending)
    }

    @Test
    fun `Init 已就绪时打开直接发起且不留挂起`() {
        assertFalse(StartupTiming.shouldDefer(ready = true))
        assertFalse(StartupTiming.consumePending(ready = true, pending = false))
    }

    @Test
    fun `多个后台返回在 Init 前只合并为一个挂起`() {
        var pending = false
        repeat(3) { pending = true }
        assertTrue(pending)
        assertTrue(StartupTiming.consumePending(ready = true, pending = pending))
    }
}
