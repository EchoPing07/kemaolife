package com.github.echoping07.kemaolife.ui

import android.os.Build
import androidx.navigationevent.DirectNavigationEventInput
import androidx.navigationevent.NavigationEvent
import androidx.navigationevent.NavigationEventDispatcher
import androidx.navigationevent.NavigationEventHandler
import androidx.navigationevent.NavigationEventInfo
import androidx.navigationevent.NavigationEventInput
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class XiaomiPredictiveBackFixTest {

    @Test
    fun `device detection matches xiaomi redmi and poco only on api 34+`() {
        assertTrue(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "Xiaomi", brand = "Xiaomi"))
        assertTrue(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "XIAOMI", brand = "Redmi"))
        assertTrue(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "Xiaomi", brand = "POCO"))
        assertTrue(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "Other", brand = "redmi"))

        assertFalse(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "OPPO", brand = "OPPO"))
        assertFalse(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "OnePlus", brand = "OnePlus"))
        assertFalse(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "Google", brand = "google"))
        assertFalse(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "samsung", brand = "samsung"))
        assertFalse(XiaomiBackProgressCompat.isXiaomiDevice(manufacturer = "", brand = ""))

        assertTrue(
            XiaomiBackProgressCompat.shouldApplyFix(
                sdkInt = Build.VERSION_CODES.UPSIDE_DOWN_CAKE,
                manufacturer = "Xiaomi",
                brand = "Xiaomi",
            ),
        )
        assertFalse(
            XiaomiBackProgressCompat.shouldApplyFix(
                sdkInt = Build.VERSION_CODES.TIRAMISU,
                manufacturer = "Xiaomi",
                brand = "Xiaomi",
            ),
        )
        assertFalse(
            XiaomiBackProgressCompat.shouldApplyFix(
                sdkInt = Build.VERSION_CODES.UPSIDE_DOWN_CAKE,
                manufacturer = "OPPO",
                brand = "OPPO",
            ),
        )
    }

    @Test
    fun `left edge normalizes hyperos saturated progress by container width`() {
        val normalizer = XiaomiBackProgressNormalizer()
        val width = 1080f

        val start =
            normalizer.onStart(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_LEFT,
                    progress = 0.25f,
                    touchX = 40f,
                    touchY = 500f,
                    frameTimeMillis = 100L,
                ),
                width,
            )
        assertEquals(0f, start.progress, 1e-6f)
        assertEquals(40f, start.touchX, 1e-6f)
        assertEquals(500f, start.touchY, 1e-6f)
        assertEquals(NavigationEvent.EDGE_LEFT, start.swipeEdge)
        assertEquals(100L, start.frameTimeMillis)

        // HyperOS 侧滑短距离就上报 progress = 1.0f，修正后应按位移占比平滑递增。
        val step1 =
            normalizer.onProgress(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_LEFT,
                    progress = 1.0f,
                    touchX = 148f,
                    touchY = 505f,
                    frameTimeMillis = 116L,
                ),
                width,
            )
        assertEquals(0.1f, step1.progress, 1e-5f)
        assertEquals(116L, step1.frameTimeMillis)

        val step2 =
            normalizer.onProgress(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_LEFT,
                    progress = 1.0f,
                    touchX = 580f,
                    touchY = 510f,
                ),
                width,
            )
        assertEquals(0.5f, step2.progress, 1e-5f)

        val clamped =
            normalizer.onProgress(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_LEFT,
                    progress = 1.0f,
                    touchX = 1300f,
                    touchY = 510f,
                ),
                width,
            )
        assertEquals(1f, clamped.progress, 1e-6f)
    }

    @Test
    fun `right edge normalizes leftward drag and handles uninitialized start x`() {
        val normalizer = XiaomiBackProgressNormalizer()
        val width = 1080f

        val start =
            normalizer.onStart(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_RIGHT,
                    progress = 0.3f,
                    touchX = 1040f,
                    touchY = 600f,
                ),
                width,
            )
        assertEquals(0f, start.progress, 1e-6f)

        val mid =
            normalizer.onProgress(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_RIGHT,
                    progress = 1.0f,
                    touchX = 500f,
                    touchY = 600f,
                ),
                width,
            )
        assertEquals(0.5f, mid.progress, 1e-5f)

        // 若起手事件误给 touchX = 0f，应延迟到首个有效坐标再锚定，不直接跳满。
        normalizer.reset()
        val zeroStart =
            normalizer.onStart(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_RIGHT,
                    progress = 0.2f,
                    touchX = 0f,
                    touchY = 600f,
                ),
                width,
            )
        assertEquals(0f, zeroStart.progress, 1e-6f)

        val firstMove =
            normalizer.onProgress(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_RIGHT,
                    progress = 0.9f,
                    touchX = 1020f,
                    touchY = 600f,
                ),
                width,
            )
        assertEquals(0f, firstMove.progress, 1e-6f)

        val secondMove =
            normalizer.onProgress(
                NavigationEvent(
                    swipeEdge = NavigationEvent.EDGE_RIGHT,
                    progress = 1.0f,
                    touchX = 912f,
                    touchY = 600f,
                ),
                width,
            )
        assertEquals(0.1f, secondMove.progress, 1e-5f)
    }

    @Test
    fun `pulling back past start anchor shifts anchor without dead zone`() {
        val normalizer = XiaomiBackProgressNormalizer()
        val width = 1000f

        // 左边缘回拉
        normalizer.onStart(NavigationEvent(NavigationEvent.EDGE_LEFT, 0f, 80f, 300f), width)
        val pulledBackLeft =
            normalizer.onProgress(NavigationEvent(NavigationEvent.EDGE_LEFT, 0.5f, 30f, 300f), width)
        assertEquals(0f, pulledBackLeft.progress, 1e-6f)
        val resumedLeft =
            normalizer.onProgress(NavigationEvent(NavigationEvent.EDGE_LEFT, 1f, 80f, 300f), width)
        assertEquals(0.05f, resumedLeft.progress, 1e-6f)

        // 右边缘回拉
        normalizer.reset()
        normalizer.onStart(NavigationEvent(NavigationEvent.EDGE_RIGHT, 0f, 920f, 300f), width)
        val pulledBackRight =
            normalizer.onProgress(NavigationEvent(NavigationEvent.EDGE_RIGHT, 0.5f, 970f, 300f), width)
        assertEquals(0f, pulledBackRight.progress, 1e-6f)
        val resumedRight =
            normalizer.onProgress(NavigationEvent(NavigationEvent.EDGE_RIGHT, 1f, 920f, 300f), width)
        assertEquals(0.05f, resumedRight.progress, 1e-6f)
    }

    @Test
    fun `non-edge, non-finite touchX, or invalid width falls back to original event`() {
        val normalizer = XiaomiBackProgressNormalizer()

        val edgeNone = NavigationEvent(NavigationEvent.EDGE_NONE, 0.42f, 100f, 200f)
        assertEquals(0.42f, normalizer.onStart(edgeNone, 1080f).progress, 1e-6f)
        assertEquals(0.42f, normalizer.onProgress(edgeNone, 1080f).progress, 1e-6f)

        val nanTouch = NavigationEvent(NavigationEvent.EDGE_LEFT, 0.35f, Float.NaN, 200f)
        assertEquals(0.35f, normalizer.onStart(nanTouch, 1080f).progress, 1e-6f)
        assertEquals(0.35f, normalizer.onProgress(nanTouch, 1080f).progress, 1e-6f)

        val validEvent = NavigationEvent(NavigationEvent.EDGE_LEFT, 0.6f, 200f, 200f)
        assertEquals(0.6f, normalizer.onProgress(validEvent, 0f).progress, 1e-6f)
    }

    @Test
    fun `bridge syncs enabled state and forwards normalized predictive and non-predictive events`() {
        val parentDispatcher = NavigationEventDispatcher()
        val systemInput = DirectNavigationEventInput()
        var parentHasEnabled = false
        val parentObserverInput =
            object : NavigationEventInput() {
                override fun onHasEnabledHandlersChanged(hasEnabledHandlers: Boolean) {
                    parentHasEnabled = hasEnabledHandlers
                }
            }
        parentDispatcher.addInput(systemInput)
        parentDispatcher.addInput(parentObserverInput)

        val bridge =
            XiaomiPredictiveBackBridge(
                parentDispatcher = parentDispatcher,
                containerWidthPx = { 1000f },
            )

        val startedProgress = mutableListOf<Float>()
        val progressedValues = mutableListOf<Float>()
        var cancelledCount = 0
        var completedCount = 0

        val childHandler =
            object : NavigationEventHandler<NavigationEventInfo>(
                initialInfo = NavigationEventInfo.None,
                isBackEnabled = false,
            ) {
                override fun onBackStarted(event: NavigationEvent) {
                    startedProgress.add(event.progress)
                }

                override fun onBackProgressed(event: NavigationEvent) {
                    progressedValues.add(event.progress)
                }

                override fun onBackCancelled() {
                    cancelledCount++
                }

                override fun onBackCompleted() {
                    completedCount++
                }
            }

        // 模拟 Compose：子树先注册 handler，再挂载桥接。
        bridge.navigationEventDispatcher.addHandler(childHandler)
        bridge.attach()
        assertFalse(parentHasEnabled)

        // 压入二级页后启用，父级同步变为启用。
        childHandler.isBackEnabled = true
        assertTrue(parentHasEnabled)

        // 预测性返回：HyperOS 上报 progress = 1.0f，子树收到的是按屏幕宽度归一化后的值。
        systemInput.backStarted(NavigationEvent(NavigationEvent.EDGE_LEFT, 0.2f, 50f, 400f))
        systemInput.backProgressed(NavigationEvent(NavigationEvent.EDGE_LEFT, 1.0f, 250f, 400f))
        systemInput.backCancelled()

        assertEquals(listOf(0f), startedProgress)
        assertEquals(listOf(0.2f), progressedValues)
        assertEquals(1, cancelledCount)
        assertEquals(0, completedCount)

        // 取消后再次手势应重新锚定并正常完成。
        systemInput.backStarted(NavigationEvent(NavigationEvent.EDGE_RIGHT, 0.4f, 950f, 400f))
        systemInput.backProgressed(NavigationEvent(NavigationEvent.EDGE_RIGHT, 1.0f, 650f, 400f))
        systemInput.backCompleted()

        assertEquals(listOf(0f, 0f), startedProgress)
        assertEquals(listOf(0.2f, 0.3f), progressedValues)
        assertEquals(1, completedCount)

        // 非预测性返回（如三键返回）不应触发 onBackStarted / onBackProgressed。
        systemInput.backCompleted()
        assertEquals(2, startedProgress.size)
        assertEquals(2, progressedValues.size)
        assertEquals(2, completedCount)

        // 退回根页面后禁用，父级同步关闭以便系统返回桌面。
        childHandler.isBackEnabled = false
        assertFalse(parentHasEnabled)

        // 销毁后从父级解绑。
        childHandler.isBackEnabled = true
        assertTrue(parentHasEnabled)
        bridge.dispose()
        assertFalse(parentHasEnabled)
    }
}
