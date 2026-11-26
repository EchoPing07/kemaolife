package com.github.echoping07.kemaolife.data

import com.github.echoping07.kemaolife.ui.AppState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 电费低阈值通知偏好的默认值、模式归一、阈值钳制与输入过滤单测（docs/android.md）。
 */
class ElectricityAlertPrefsTest {

    @Test
    fun defaultsMatchSpec() {
        assertTrue(AppPreferences.DEFAULT_ELECTRICITY_ALERT_ENABLED)
        assertEquals(AppPreferences.ALERT_MODE_BALANCE, AppState().electricityAlertMode)
        assertTrue(AppState().electricityAlertEnabled)
        assertEquals(10.0, AppPreferences.DEFAULT_BALANCE_THRESHOLD, 1e-9)
        assertEquals(10.0, AppState().electricityBalanceThreshold, 1e-9)
        assertEquals(3, AppPreferences.DEFAULT_DAYS_THRESHOLD)
        assertEquals(3, AppState().electricityDaysThreshold)
    }

    @Test
    fun normalizeAlertModeFallsBackToBalance() {
        assertEquals(AppPreferences.ALERT_MODE_BALANCE, AppPreferences.normalizeAlertMode(null))
        assertEquals(AppPreferences.ALERT_MODE_BALANCE, AppPreferences.normalizeAlertMode(""))
        assertEquals(AppPreferences.ALERT_MODE_BALANCE, AppPreferences.normalizeAlertMode("balance"))
        assertEquals(AppPreferences.ALERT_MODE_BALANCE, AppPreferences.normalizeAlertMode("unknown"))
        assertEquals(AppPreferences.ALERT_MODE_DAYS, AppPreferences.normalizeAlertMode("days"))
        assertEquals(AppPreferences.ALERT_MODE_DAYS, AppPreferences.normalizeAlertMode(" DAYS "))
    }

    @Test
    fun clampedBalanceThresholdHandlesBoundsAndPrecision() {
        assertEquals(AppPreferences.MIN_BALANCE_THRESHOLD, AppPreferences.clampedBalanceThreshold(0.0), 1e-9)
        assertEquals(AppPreferences.MIN_BALANCE_THRESHOLD, AppPreferences.clampedBalanceThreshold(-5.0), 1e-9)
        assertEquals(AppPreferences.MAX_BALANCE_THRESHOLD, AppPreferences.clampedBalanceThreshold(99999.0), 1e-9)
        assertEquals(AppPreferences.DEFAULT_BALANCE_THRESHOLD, AppPreferences.clampedBalanceThreshold(Double.NaN), 1e-9)
        assertEquals(12.35, AppPreferences.clampedBalanceThreshold(12.346), 1e-9)
    }

    @Test
    fun clampedDaysThresholdHandlesBounds() {
        assertEquals(AppPreferences.MIN_DAYS_THRESHOLD, AppPreferences.clampedDaysThreshold(0))
        assertEquals(AppPreferences.MIN_DAYS_THRESHOLD, AppPreferences.clampedDaysThreshold(-3))
        assertEquals(5, AppPreferences.clampedDaysThreshold(5))
        assertEquals(AppPreferences.MAX_DAYS_THRESHOLD, AppPreferences.clampedDaysThreshold(999))
    }

    @Test
    fun formatBalanceThresholdDropsTrailingZeros() {
        assertEquals("10", AppPreferences.formatBalanceThreshold(10.0))
        assertEquals("10.5", AppPreferences.formatBalanceThreshold(10.5))
        assertEquals("10.25", AppPreferences.formatBalanceThreshold(10.25))
        assertEquals("0.1", AppPreferences.formatBalanceThreshold(0.1))
    }

    @Test
    fun sanitizeDecimalInputAllowsCleanDecimalsOnly() {
        assertEquals("10", AppPreferences.sanitizeDecimalInput("10"))
        assertEquals("10.5", AppPreferences.sanitizeDecimalInput("10.5"))
        assertEquals("10.25", AppPreferences.sanitizeDecimalInput("10.259"))
        assertEquals("10.58", AppPreferences.sanitizeDecimalInput("1a0.5.8"))
        assertEquals("0.5", AppPreferences.sanitizeDecimalInput(".5"))
        assertEquals("1234", AppPreferences.sanitizeDecimalInput("123456"))
    }
}
