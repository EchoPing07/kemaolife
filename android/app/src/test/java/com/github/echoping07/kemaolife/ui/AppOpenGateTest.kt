package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 冷启动时 Lifecycle 会向新 observer 补发 ON_RESUME；把它当成「打开 App」
 * 会在门面 Init 前触发一轮自动查询并弹「桥接尚未初始化」。
 */
class AppOpenGateTest {

    @Test
    fun replayedResumeAtRegistrationIsNotAnOpen() {
        val gate = AppOpenGate()

        assertFalse(gate.onResume())
    }

    @Test
    fun onlyFirstResumeAfterStopCountsAsOpen() {
        val gate = AppOpenGate()

        gate.onStop()
        assertTrue(gate.onResume())
        assertFalse(gate.onResume())
    }

    @Test
    fun everyReturnFromBackgroundCountsAsOpen() {
        val gate = AppOpenGate()
        assertFalse(gate.onResume())

        repeat(3) {
            gate.onStop()
            assertTrue(gate.onResume())
        }
    }

    @Test
    fun repeatedStopBeforeResumeStillCountsAsOneOpen() {
        val gate = AppOpenGate()

        gate.onStop()
        gate.onStop()
        assertTrue(gate.onResume())
        assertFalse(gate.onResume())
    }
}
