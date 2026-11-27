package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AppNavigationTest {

    @Test
    fun `push appends settings page on home and ignores duplicate consecutive taps`() {
        val stack = mutableListOf<AppDestination>(AppDestination.Home)
        val education = AppDestination.Settings(SettingsPage.EDUCATION)
        val logs = AppDestination.Settings(SettingsPage.LOGS)

        assertTrue(AppBackStackOps.push(stack, education))
        assertEquals(listOf(AppDestination.Home, education), stack)

        assertFalse(AppBackStackOps.push(stack, education))
        assertEquals(listOf(AppDestination.Home, education), stack)

        assertTrue(AppBackStackOps.push(stack, logs))
        assertEquals(listOf(AppDestination.Home, education, logs), stack)
    }

    @Test
    fun `pop removes top child and never empties root home entry`() {
        val appearance = AppDestination.Settings(SettingsPage.APPEARANCE)
        val stack = mutableListOf<AppDestination>(AppDestination.Home, appearance)

        assertTrue(AppBackStackOps.pop(stack))
        assertEquals(listOf(AppDestination.Home), stack)

        // 连续两次返回（同一帧重复回调）时根页面必须保留，避免 NavDisplay 空栈崩溃。
        assertFalse(AppBackStackOps.pop(stack))
        assertEquals(listOf(AppDestination.Home), stack)
    }

    @Test
    fun `encode and decode round-trip home and every settings page`() {
        val allDestinations =
            listOf(AppDestination.Home) + SettingsPage.entries.map(AppDestination::Settings)
        val encoded = AppBackStackOps.encode(allDestinations)
        assertEquals(allDestinations.size, encoded.toSet().size)
        assertEquals(allDestinations, AppBackStackOps.decode(encoded))
    }

    @Test
    fun `decode recovers safely from empty, unknown, or rootless saved stacks`() {
        assertEquals(listOf(AppDestination.Home), AppBackStackOps.decode(emptyList()))
        assertEquals(listOf(AppDestination.Home), AppBackStackOps.decode(listOf("unknown/route")))
        assertNull(AppDestination.fromId("unknown/route"))

        assertEquals(
            listOf(AppDestination.Home, AppDestination.Settings(SettingsPage.ABOUT)),
            AppBackStackOps.decode(listOf("settings/about", "stale/route")),
        )
        assertEquals(
            listOf(AppDestination.Home, AppDestination.Settings(SettingsPage.CAMPUS)),
            AppBackStackOps.decode(listOf("settings/campus", "home")),
        )
    }

    @Test
    fun `transitions and predictive pop respect motionEnabled flag`() {
        val disabledPush = AppNavTransitions.push(motionEnabled = false)
        val disabledPop = AppNavTransitions.pop(motionEnabled = false)
        assertEquals(EnterTransition.None, disabledPush.targetContentEnter)
        assertEquals(ExitTransition.None, disabledPush.initialContentExit)
        assertEquals(EnterTransition.None, disabledPop.targetContentEnter)
        assertEquals(ExitTransition.None, disabledPop.initialContentExit)

        val enabledPush = AppNavTransitions.push(motionEnabled = true)
        val enabledPop = AppNavTransitions.pop(motionEnabled = true)
        assertNotEquals(EnterTransition.None, enabledPush.targetContentEnter)
        assertNotEquals(ExitTransition.None, enabledPush.initialContentExit)
        assertNotEquals(EnterTransition.None, enabledPop.targetContentEnter)
        assertNotEquals(ExitTransition.None, enabledPop.initialContentExit)
    }
}
