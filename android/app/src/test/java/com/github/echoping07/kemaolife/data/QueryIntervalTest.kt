package com.github.echoping07.kemaolife.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 最小查询间隔的判定与钳制（docs/000 §5.6、docs/002 §5.3）。
 *
 * 直接调用生产实现 `AppPreferences.clamped` / `isDueAt`（两者都是 companion
 * 纯函数，不依赖 Android frame），禁止在测试里复刻一份钳制逻辑 —— 那会变成
 * 把生产代码改坏也照样绿的假测试。
 */
class QueryIntervalTest {

    @Test
    fun bothDomainsDefaultToFiveMinutesWithOneMinuteFloor() {
        assertEquals(5, AppPreferences.DEFAULT_TIMETABLE_INTERVAL)
        assertEquals(5, AppPreferences.DEFAULT_ELECTRICITY_INTERVAL)
        assertEquals(1, AppPreferences.MIN_TIMETABLE_INTERVAL)
        assertEquals(1, AppPreferences.MIN_ELECTRICITY_INTERVAL)
        assertEquals(1440, AppPreferences.MAX_INTERVAL)
    }

    @Test
    fun firstQueryIsAlwaysDue() {
        assertTrue(AppPreferences.isDueAt(last = 0L, minutes = 5, now = 1_000_000L))
        assertTrue(AppPreferences.isDueAt(last = -1L, minutes = 5, now = 1_000_000L))
    }

    @Test
    fun withinIntervalIsNotDue() {
        val now = 10 * 60_000L
        // 5 分钟间隔：4 分 59 秒还没到。
        assertFalse(AppPreferences.isDueAt(last = now - 4 * 60_000L - 59_000L, minutes = 5, now = now))
    }

    @Test
    fun exactlyAtIntervalIsDue() {
        val now = 10 * 60_000L
        assertTrue(AppPreferences.isDueAt(last = now - 5 * 60_000L, minutes = 5, now = now))
    }

    @Test
    fun clockRollbackDoesNotSuppressForever() {
        val now = 10 * 60_000L
        // 单调时钟重启归零 / wall-clock 被回拨：last > now 时视为到期。
        assertTrue(AppPreferences.isDueAt(last = now + 60_000L, minutes = 5, now = now))
    }

    @Test
    fun clampedUsesProductionImplementation() {
        // 低于 1 分钟一律钳制到 1 分钟；两个域同口径，没有 600 秒下限。
        assertEquals(1, AppPreferences.clamped(0, AppPreferences.MIN_TIMETABLE_INTERVAL))
        assertEquals(1, AppPreferences.clamped(-5, AppPreferences.MIN_ELECTRICITY_INTERVAL))
        assertEquals(5, AppPreferences.clamped(5))
        assertEquals(AppPreferences.MAX_INTERVAL, AppPreferences.clamped(99_999))
    }

    @Test
    fun clampedValidatesMinimumWithoutThrowing() {
        // 外部传入非法下限（> MAX）时 coerceIn 会抛异常，生产实现必须自己钳住。
        assertEquals(AppPreferences.MAX_INTERVAL, AppPreferences.clamped(3, 999_999))
        // 负下限钳到 1 后，3 本身是合法值，不应被无端改成 1。
        assertEquals(3, AppPreferences.clamped(3, -1))
    }

    @Test
    fun negativeAndHugeIntervalsAreClampedByIsDue() {
        val now = 10 * 60_000L
        val last = now - 2 * 60_000L
        // minutes 为负时按 1 分钟算，2 分钟前应该到期。
        assertTrue(AppPreferences.isDueAt(last, -10, now))
        // minutes 极大时按一天算，2 分钟前不该到期。
        assertFalse(AppPreferences.isDueAt(last, Int.MAX_VALUE, now))
    }
}
