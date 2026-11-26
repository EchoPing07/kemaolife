package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AppNavigationTest {

    @Test
    fun `push appends new route and ignores duplicate consecutive taps`() {
        val stack = mutableListOf<AppRoute>(AppRoute.Home)

        assertTrue(AppBackStackOps.push(stack, AppRoute.SettingsChild.Education))
        assertEquals(listOf(AppRoute.Home, AppRoute.SettingsChild.Education), stack)

        assertFalse(AppBackStackOps.push(stack, AppRoute.SettingsChild.Education))
        assertEquals(listOf(AppRoute.Home, AppRoute.SettingsChild.Education), stack)

        // 支持后续多级页面继续压栈，不限制单层深度。
        assertTrue(AppBackStackOps.push(stack, AppRoute.SettingsChild.Logs))
        assertEquals(
            listOf(AppRoute.Home, AppRoute.SettingsChild.Education, AppRoute.SettingsChild.Logs),
            stack,
        )
    }

    @Test
    fun `pop removes top child and never empties root home entry`() {
        val stack = mutableListOf<AppRoute>(AppRoute.Home, AppRoute.SettingsChild.Appearance)

        assertTrue(AppBackStackOps.pop(stack))
        assertEquals(listOf(AppRoute.Home), stack)

        // 连续两次返回（同一帧重复回调）时根页面必须保留，避免 NavDisplay 空栈崩溃。
        assertFalse(AppBackStackOps.pop(stack))
        assertEquals(listOf(AppRoute.Home), stack)
    }

    @Test
    fun `encode and decode round-trip all known routes`() {
        val allRoutes =
            listOf(
                AppRoute.Home,
                AppRoute.SettingsChild.Education,
                AppRoute.SettingsChild.Electricity,
                AppRoute.SettingsChild.Campus,
                AppRoute.SettingsChild.Appearance,
                AppRoute.SettingsChild.About,
                AppRoute.SettingsChild.Logs,
            )
        val encoded = AppBackStackOps.encode(allRoutes)
        assertEquals(allRoutes.size, encoded.toSet().size)
        assertEquals(allRoutes, AppBackStackOps.decode(encoded))
    }

    @Test
    fun `decode recovers safely from empty, unknown, or rootless saved stacks`() {
        assertEquals(listOf(AppRoute.Home), AppBackStackOps.decode(emptyList()))
        assertEquals(listOf(AppRoute.Home), AppBackStackOps.decode(listOf("unknown/route")))
        assertNull(AppRoute.fromId("unknown/route"))

        assertEquals(
            listOf(AppRoute.Home, AppRoute.SettingsChild.About),
            AppBackStackOps.decode(listOf("settings/about", "stale/route")),
        )
        assertEquals(
            listOf(AppRoute.Home, AppRoute.SettingsChild.Campus),
            AppBackStackOps.decode(listOf("settings/campus", "home")),
        )
    }
}
