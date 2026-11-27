package com.github.echoping07.kemaolife.ui

import android.os.Build
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.platform.LocalResources
import androidx.compose.ui.platform.LocalWindowInfo
import androidx.navigationevent.NavigationEvent
import androidx.navigationevent.NavigationEventDispatcher
import androidx.navigationevent.NavigationEventDispatcherOwner
import androidx.navigationevent.NavigationEventHandler
import androidx.navigationevent.NavigationEventInfo
import androidx.navigationevent.NavigationEventInput
import androidx.navigationevent.compose.LocalNavigationEventDispatcherOwner

/**
 * 小米 / HyperOS 预测性返回进度适配判定（仅基于公开 [Build] 字段，零反射）。
 *
 * HyperOS 魔改了边缘返回手势管线，分发给应用的 `BackEvent.progress` 按较短的固定阈值计算，
 * 侧滑一小段距离就会直接冲到 `1.0f`，导致 Navigation 3 / Compose 预测性返回动画瞬间跳完、松手无过渡；
 * 但 `touchX` 与 `swipeEdge` 仍反映手指真实屏幕坐标。本模块仅在 Android 14+ 小米系设备上启用，
 * 其他系统（ColorOS / 原生 AOSP 等）直接透传零开销。
 */
internal object XiaomiBackProgressCompat {
    private val xiaomiIdentifiers = setOf("xiaomi", "redmi", "poco")

    fun isXiaomiDevice(
        manufacturer: String = Build.MANUFACTURER.orEmpty(),
        brand: String = Build.BRAND.orEmpty(),
    ): Boolean {
        val m = manufacturer.trim().lowercase()
        val b = brand.trim().lowercase()
        return m in xiaomiIdentifiers || b in xiaomiIdentifiers
    }

    fun shouldApplyFix(
        sdkInt: Int = Build.VERSION.SDK_INT,
        manufacturer: String = Build.MANUFACTURER.orEmpty(),
        brand: String = Build.BRAND.orEmpty(),
    ): Boolean = sdkInt >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE && isXiaomiDevice(manufacturer, brand)
}

/**
 * 将 HyperOS 上失真的 `NavigationEvent.progress` 按「相对起点水平位移 / 当前窗口宽度」重算为 `[0, 1]`。
 *
 * - 左边缘手势：以起手 `touchX` 为锚点向右累计位移；手指回拉越过锚点时同步回退锚点，避免再推时出现死区。
 * - 右边缘手势：以起手 `touchX` 为锚点向左累计位移；若起手事件给出无效的 `0f`，延迟到首个有效 `touchX` 再锚定。
 * - 非边缘手势（如按键 / `EDGE_NONE`）、非有限 `touchX` 或无效容器宽度时原样透传原始事件。
 */
internal class XiaomiBackProgressNormalizer {
    private var anchorTouchX: Float = Float.NaN

    fun reset() {
        anchorTouchX = Float.NaN
    }

    fun onStart(
        event: NavigationEvent,
        containerWidthPx: Float,
    ): NavigationEvent {
        anchorTouchX = Float.NaN
        if (!canNormalize(event, containerWidthPx)) return event

        when (event.swipeEdge) {
            NavigationEvent.EDGE_LEFT -> {
                anchorTouchX = event.touchX.coerceAtLeast(0f)
            }

            NavigationEvent.EDGE_RIGHT -> {
                if (event.touchX > 0f) {
                    anchorTouchX = event.touchX.coerceAtMost(containerWidthPx)
                }
            }
        }
        return event.withProgress(0f)
    }

    fun onProgress(
        event: NavigationEvent,
        containerWidthPx: Float,
    ): NavigationEvent {
        if (!canNormalize(event, containerWidthPx)) return event

        val deltaX =
            when (event.swipeEdge) {
                NavigationEvent.EDGE_LEFT -> {
                    val x = event.touchX.coerceAtLeast(0f)
                    if (anchorTouchX.isNaN() || x < anchorTouchX) {
                        anchorTouchX = x
                    }
                    x - anchorTouchX
                }

                NavigationEvent.EDGE_RIGHT -> {
                    if (anchorTouchX.isNaN()) {
                        if (event.touchX <= 0f) return event
                        anchorTouchX = event.touchX.coerceAtMost(containerWidthPx)
                    }
                    val x = event.touchX.coerceAtMost(containerWidthPx)
                    if (x > anchorTouchX) {
                        anchorTouchX = x
                    }
                    anchorTouchX - x
                }

                else -> return event
            }

        val normalizedProgress = (deltaX / containerWidthPx).coerceIn(0f, 1f)
        return event.withProgress(normalizedProgress)
    }

    private fun canNormalize(
        event: NavigationEvent,
        containerWidthPx: Float,
    ): Boolean =
        containerWidthPx > 0f &&
            event.touchX.isFinite() &&
            (event.swipeEdge == NavigationEvent.EDGE_LEFT || event.swipeEdge == NavigationEvent.EDGE_RIGHT)

    private fun NavigationEvent.withProgress(progress: Float): NavigationEvent =
        NavigationEvent(
            swipeEdge = swipeEdge,
            progress = progress,
            touchX = touchX,
            touchY = touchY,
            frameTimeMillis = frameTimeMillis,
        )
}

/**
 * 基于公开 [NavigationEventDispatcher] / [NavigationEventInput] / [NavigationEventHandler] 的代理分发桥。
 *
 * 子树通过 [LocalNavigationEventDispatcherOwner] 注册到独立 [navigationEventDispatcher]：
 * - 子树有无启用中的 handler 时，经 [NavigationEventInput.onHasEnabledHandlersChanged] 精确同步到父级
 *   [parentHandler]，保证位于根页面时系统级预测性返回桌面完全不受干扰；
 * - 侧滑过程中拦截上游 `NavigationEvent`，经 [XiaomiBackProgressNormalizer] 重算 `progress` 后转发给子树，
 *   无需改动任何 `NavDisplay`、`BackHandler` 或业务页面代码。
 */
internal class XiaomiPredictiveBackBridge(
    private val parentDispatcher: NavigationEventDispatcher,
    private val containerWidthPx: () -> Float,
    private val normalizer: XiaomiBackProgressNormalizer = XiaomiBackProgressNormalizer(),
) : NavigationEventDispatcherOwner {
    override val navigationEventDispatcher: NavigationEventDispatcher = NavigationEventDispatcher()

    private class ProxyInput(
        private val onHasEnabledChanged: (Boolean) -> Unit,
    ) : NavigationEventInput() {
        override fun onHasEnabledHandlersChanged(hasEnabledHandlers: Boolean) {
            onHasEnabledChanged(hasEnabledHandlers)
        }

        fun start(event: NavigationEvent) = dispatchOnBackStarted(event)

        fun progress(event: NavigationEvent) = dispatchOnBackProgressed(event)

        fun cancel() = dispatchOnBackCancelled()

        fun complete() = dispatchOnBackCompleted()
    }

    private val proxyInput = ProxyInput { hasEnabledHandlers ->
        parentHandler.isBackEnabled = hasEnabledHandlers
    }

    private val parentHandler: NavigationEventHandler<NavigationEventInfo> =
        object : NavigationEventHandler<NavigationEventInfo>(
            initialInfo = NavigationEventInfo.None,
            isBackEnabled = false,
        ) {
            override fun onBackStarted(event: NavigationEvent) {
                proxyInput.start(normalizer.onStart(event, containerWidthPx()))
            }

            override fun onBackProgressed(event: NavigationEvent) {
                proxyInput.progress(normalizer.onProgress(event, containerWidthPx()))
            }

            override fun onBackCancelled() {
                normalizer.reset()
                proxyInput.cancel()
            }

            override fun onBackCompleted() {
                normalizer.reset()
                proxyInput.complete()
            }
        }

    private var attached = false
    private var disposed = false

    init {
        navigationEventDispatcher.addInput(proxyInput)
    }

    fun attach() {
        if (attached || disposed) return
        attached = true
        parentDispatcher.addHandler(parentHandler)
    }

    fun dispose() {
        if (disposed) return
        disposed = true
        if (attached) {
            attached = false
            parentHandler.remove()
        }
        navigationEventDispatcher.dispose()
    }
}

/**
 * 全局小米 HyperOS 预测性返回进度修正包装器。
 *
 * 在根组合通过 [LocalNavigationEventDispatcherOwner] 注入代理分发器，自动覆盖子树内所有
 * Navigation 3 `NavDisplay`、`NavigationBackHandler`、`BackHandler` 与 `PredictiveBackHandler`；
 * 非小米设备或 Android 14 以下直接透传 [content]。
 */
@Composable
internal fun XiaomiPredictiveBackFix(content: @Composable () -> Unit) {
    val parentOwner = LocalNavigationEventDispatcherOwner.current
    if (!XiaomiBackProgressCompat.shouldApplyFix() || parentOwner == null) {
        content()
        return
    }

    val windowWidthPx = LocalWindowInfo.current.containerSize.width
    val fallbackWidthPx = LocalResources.current.displayMetrics.widthPixels
    val widthPx = (if (windowWidthPx > 0) windowWidthPx else fallbackWidthPx).toFloat()
    val latestWidthPx by rememberUpdatedState(widthPx)

    val parentDispatcher = parentOwner.navigationEventDispatcher
    val bridge =
        remember(parentDispatcher) {
            XiaomiPredictiveBackBridge(
                parentDispatcher = parentDispatcher,
                containerWidthPx = { latestWidthPx },
            )
        }

    DisposableEffect(bridge) {
        bridge.attach()
        onDispose { bridge.dispose() }
    }

    CompositionLocalProvider(LocalNavigationEventDispatcherOwner provides bridge) {
        content()
    }
}
