package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.ContentTransform
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
        val disabledPredictivePop = AppNavTransitions.predictivePop(motionEnabled = false)
        assertEquals(EnterTransition.None, disabledPush.targetContentEnter)
        assertEquals(ExitTransition.None, disabledPush.initialContentExit)
        assertEquals(EnterTransition.None, disabledPop.targetContentEnter)
        assertEquals(ExitTransition.None, disabledPop.initialContentExit)
        assertEquals(EnterTransition.None, disabledPredictivePop.targetContentEnter)
        assertEquals(ExitTransition.None, disabledPredictivePop.initialContentExit)

        val enabledPush = AppNavTransitions.push(motionEnabled = true)
        val enabledPop = AppNavTransitions.pop(motionEnabled = true)
        val enabledPredictivePop = AppNavTransitions.predictivePop(motionEnabled = true)
        assertNotEquals(EnterTransition.None, enabledPush.targetContentEnter)
        assertNotEquals(ExitTransition.None, enabledPush.initialContentExit)
        assertNotEquals(EnterTransition.None, enabledPop.targetContentEnter)
        assertNotEquals(ExitTransition.None, enabledPop.initialContentExit)
        assertNotEquals(EnterTransition.None, enabledPredictivePop.targetContentEnter)
        assertNotEquals(ExitTransition.None, enabledPredictivePop.initialContentExit)
    }

    @Test
    fun `nav settle easing is monotonic, bounded, and front-loaded`() {
        val easing = NavSettleEasing()
        assertEquals(0f, easing.transform(0f), 1e-6f)
        assertEquals(1f, easing.transform(1f), 1e-6f)

        var prev = 0f
        for (step in 1..100) {
            val value = easing.transform(step / 100f)
            assertTrue("expected monotonic increase at step $step: $value >= $prev", value >= prev)
            assertTrue("expected bounded value at step $step: $value <= 1f", value <= 1f)
            prev = value
        }

        assertTrue(easing.transform(0.2f) > 0.4f)
        assertTrue(easing.transform(0.5f) > 0.85f)
    }

    @Test
    fun `predictive pop easing tracks near-linearly at start and decelerates smoothly`() {
        val easing = AppNavTransitions.PredictivePopEasing
        assertEquals(0f, easing.transform(0f), 1e-6f)
        assertEquals(1f, easing.transform(1f), 1e-6f)

        var prev = 0f
        for (step in 1..100) {
            val value = easing.transform(step / 100f)
            assertTrue("expected monotonic increase at step $step", value >= prev)
            assertTrue("expected bounded value at step $step", value <= 1f)
            prev = value
        }

        // 末端明显减速，避免 LinearEasing 松手硬停。
        val earlyDelta = easing.transform(0.2f) - easing.transform(0f)
        val lateDelta = easing.transform(1f) - easing.transform(0.8f)
        assertTrue("expected deceleration near end: $lateDelta < $earlyDelta", lateDelta < earlyDelta)
    }

    @Test
    fun `forward transition corrects direction when target is home`() {
        val home = AppDestination.Home.routeId
        val appearance = SettingsPage.APPEARANCE.routeId
        val about = SettingsPage.ABOUT.routeId
        val push = AppNavTransitions.push(motionEnabled = true)
        val pop = AppNavTransitions.pop(motionEnabled = true)
        val predictivePop = AppNavTransitions.predictivePop(motionEnabled = true)

        // 先确认三种基础规格互不相等，确保下方等价断言具有区分力。
        assertNotEquals(push.targetContentEnter, pop.targetContentEnter)
        assertNotEquals(push.initialContentExit, pop.initialContentExit)
        assertNotEquals(pop.targetContentEnter, predictivePop.targetContentEnter)
        assertNotEquals(pop.initialContentExit, predictivePop.initialContentExit)

        // 正常进入子页（Home → Settings，以及子页间推入）走 push。
        assertSameTransform(push, AppNavTransitions.forward(home, appearance, motionEnabled = true))
        assertSameTransform(push, AppNavTransitions.forward(appearance, about, motionEnabled = true))

        // 进场未完被返回打断（Settings → Home）及启动首帧缓存（Home → Home）走 predictivePop。
        assertSameTransform(predictivePop, AppNavTransitions.forward(appearance, home, motionEnabled = true))
        assertSameTransform(predictivePop, AppNavTransitions.forward(home, home, motionEnabled = true))
    }

    @Test
    fun `pop transition corrects direction when re-entering child from home`() {
        val home = AppDestination.Home.routeId
        val appearance = SettingsPage.APPEARANCE.routeId
        val about = SettingsPage.ABOUT.routeId
        val push = AppNavTransitions.push(motionEnabled = true)
        val pop = AppNavTransitions.pop(motionEnabled = true)

        assertSameTransform(pop, AppNavTransitions.pop(appearance, home, motionEnabled = true))
        assertSameTransform(pop, AppNavTransitions.pop(appearance, about, motionEnabled = true))
        assertSameTransform(pop, AppNavTransitions.pop(home, home, motionEnabled = true))

        // 退场未完立即再次进入子页，自校正为 push。
        assertSameTransform(push, AppNavTransitions.pop(home, appearance, motionEnabled = true))
    }

    @Test
    fun `predictive pop transition corrects direction when re-entering child from home`() {
        val home = AppDestination.Home.routeId
        val appearance = SettingsPage.APPEARANCE.routeId
        val about = SettingsPage.ABOUT.routeId
        val push = AppNavTransitions.push(motionEnabled = true)
        val predictivePop = AppNavTransitions.predictivePop(motionEnabled = true)

        assertSameTransform(predictivePop, AppNavTransitions.predictivePop(appearance, home, motionEnabled = true))
        assertSameTransform(predictivePop, AppNavTransitions.predictivePop(appearance, about, motionEnabled = true))
        assertSameTransform(predictivePop, AppNavTransitions.predictivePop(home, home, motionEnabled = true))

        // 预测性返回未完立即再次进入子页，自校正为 push。
        assertSameTransform(push, AppNavTransitions.predictivePop(home, appearance, motionEnabled = true))
    }

    @Test
    fun `route-aware transitions return None for all pairs when motion is disabled`() {
        val home = AppDestination.Home.routeId
        val appearance = SettingsPage.APPEARANCE.routeId
        val about = SettingsPage.ABOUT.routeId
        val pairs =
            listOf(
                home to appearance,
                appearance to home,
                home to home,
                appearance to about,
            )

        for ((initialKey, targetKey) in pairs) {
            val transforms =
                listOf(
                    AppNavTransitions.forward(initialKey, targetKey, motionEnabled = false),
                    AppNavTransitions.pop(initialKey, targetKey, motionEnabled = false),
                    AppNavTransitions.predictivePop(initialKey, targetKey, motionEnabled = false),
                )
            for (transform in transforms) {
                assertEquals(EnterTransition.None, transform.targetContentEnter)
                assertEquals(ExitTransition.None, transform.initialContentExit)
            }
        }
    }

    private fun assertSameTransform(
        expected: ContentTransform,
        actual: ContentTransform,
    ) {
        assertEquals(expected.targetContentEnter, actual.targetContentEnter)
        assertEquals(expected.initialContentExit, actual.initialContentExit)
    }
}
